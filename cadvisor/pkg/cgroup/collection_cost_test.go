package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	whatap_model "github.com/whatap/kube/cadvisor/pkg/model"
)

const collectionTestCPU = "cpu  100 200 300 400 500 600 700 800 900 1000\n"

var collectionStatsSink whatap_model.ContainerStat

func writeCollectionFile(t testing.TB, prefix, name, content string) {
	t.Helper()
	path := filepath.Join(prefix, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupCollectionFixture(t testing.TB, version string) string {
	t.Helper()
	prefix := t.TempDir()
	files := map[string]string{
		"proc/stat": collectionTestCPU,
		"proc/42/net/dev": "Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\n" +
			"eth0: 1000 10 1 2 0 0 0 0 2000 20 3 4 0 0 0 0\n",
	}
	if version == "v1" {
		for name, value := range map[string]string{
			"cpu/cpu.stat":                           "nr_periods 100\nnr_throttled 2\nthrottled_time 300\n",
			"cpu/cpuacct.stat":                       "user 60\nsystem 40\n",
			"memory/memory.max_usage_in_bytes":       "2048\n",
			"memory/memory.usage_in_bytes":           "1024\n",
			"memory/memory.failcnt":                  "0\n",
			"memory/memory.stat":                     "rss 100\ncache 200\n",
			"blkio/blkio.io_service_bytes_recursive": "8:0 Read 100\n",
			"blkio/blkio.throttle.io_service_bytes":  "8:0 Write 200\n",
			"blkio/blkio.io_serviced_recursive":      "8:0 Read 10\n",
			"blkio/blkio.throttle.io_serviced":       "8:0 Write 20\n",
		} {
			parts := strings.SplitN(name, "/", 2)
			files[filepath.Join("sys/fs/cgroup", parts[0], testCgroupParent, parts[1])] = value
		}
	} else {
		for name, value := range map[string]string{
			"cpu.stat":       "user_usec 600000\nsystem_usec 400000\nnr_periods 100\n",
			"memory.current": "1024\n",
			"memory.events":  "oom 0\n",
			"memory.stat":    "anon 100\nfile 200\n",
			"io.stat":        "8:0 rbytes=100 wbytes=200 rios=10 wios=20\n",
		} {
			files[filepath.Join("sys/fs/cgroup", testCgroupParent, name)] = value
		}
	}
	for name, value := range files {
		writeCollectionFile(t, prefix, name, value)
	}
	return prefix
}

func collectFixtureStats(prefix, version string) (whatap_model.ContainerStat, error) {
	if version == "v1" {
		return GetContainerStatsCgroupV1(prefix, "id", "name", testCgroupParent, 2, 42, 4096)
	}
	return GetContainerStatsCgroupV2(prefix, "id", "name", testCgroupParent, 2, 42, 4096)
}

func wideHostCPUStat() string {
	var text strings.Builder
	text.WriteString(collectionTestCPU)
	for cpu := 0; cpu < 256; cpu++ {
		fmt.Fprintf(&text, "cpu%d 100 200 300 400 500 600 700 800 900 1000\n", cpu)
	}
	text.WriteString("intr 1000 " + strings.Repeat("1 ", 6824) + "\n")
	return text.String()
}

func TestSystemCPUParsingAllocationsDoNotScaleWithHostCPUCount(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			prefix := setupCollectionFixture(t, version)
			measure := func() float64 {
				return testing.AllocsPerRun(10, func() {
					stat, err := collectFixtureStats(prefix, version)
					if err != nil {
						t.Fatal(err)
					}
					if stat.CPUStats.SystemCPUUsage != 3600 {
						t.Fatalf("system CPU = %d, want first eight counters only", stat.CPUStats.SystemCPUUsage)
					}
					collectionStatsSink = stat
				})
			}
			small := measure()
			writeCollectionFile(t, prefix, "proc/stat", wideHostCPUStat())
			large := measure()
			const allocationTolerance = 16
			if large > small+allocationTolerance {
				t.Fatalf("unconsumed CPU/interrupt rows added allocations: small=%.0f, large=%.0f", small, large)
			}
		})
	}
}

func TestSystemCPUIsReadFreshOnEveryCollection(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			prefix := setupCollectionFixture(t, version)
			first, err := collectFixtureStats(prefix, version)
			if err != nil {
				t.Fatal(err)
			}
			writeCollectionFile(t, prefix, "proc/stat", "cpu  200 200 300 400 500 600 700 800 900 1000\n")
			second, err := collectFixtureStats(prefix, version)
			if err != nil {
				t.Fatal(err)
			}
			if second.CPUStats.SystemCPUUsage != first.CPUStats.SystemCPUUsage+100 {
				t.Fatalf("CPU source was not refreshed: first=%d second=%d", first.CPUStats.SystemCPUUsage, second.CPUStats.SystemCPUUsage)
			}
		})
	}
}

func BenchmarkContainerStatsWideHost(b *testing.B) {
	for _, version := range []string{"v1", "v2"} {
		b.Run(version, func(b *testing.B) {
			prefix := setupCollectionFixture(b, version)
			writeCollectionFile(b, prefix, "proc/stat", wideHostCPUStat())
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				stat, err := collectFixtureStats(prefix, version)
				if err != nil {
					b.Fatal(err)
				}
				collectionStatsSink = stat
			}
		})
	}
}
