package cgroup

import (
	"fmt"
	"path/filepath"
	"strings"

	whatap_model "github.com/whatap/kube/cadvisor/pkg/model"
	"github.com/whatap/kube/tools/util/stringutil"
)

func GetContainerStatsCgroupV2(prefix string, containerId string, name string, cgroupParent string,
	restartCount int, pid int, memoryLimit int64) (whatap_model.ContainerStat, error) {
	var containerStat whatap_model.ContainerStat
	containerStat.Name = name
	containerStat.ID = containerId
	containerStat.RestartCount = restartCount

	err := populateFileKeyValue(prefix, "/proc/stat", func(key string, v []int64) {
		if key == "cpu" {
			for i, ev := range v {
				if i < 8 {
					containerStat.CPUStats.SystemCPUUsage += ev
				}
			}
		}
	})
	if err != nil {
		return containerStat, err
	}
	// fmt.Println("GetContainerStats step -1")
	err = populateCgroupKeyValue(prefix, "", cgroupParent, "cpu.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "nr_periods" {
			containerStat.CPUStats.ThrottlingData.Periods = v
		} else if key == "throttled_usec" {
			// v1 cpu.stat throttled_time은 ns, v2 throttled_usec은 μs — 서버 계약(ns)에 맞춰 정규화
			containerStat.CPUStats.ThrottlingData.ThrottledTime = v * 1000
		} else if key == "nr_throttled" {
			containerStat.CPUStats.ThrottlingData.ThrottledPeriods = v
		} else if key == "user_usec" {
			containerStat.CPUStats.CPUUsage.UsageInUsermode = v / 10000
		} else if key == "system_usec" {
			containerStat.CPUStats.CPUUsage.UsageInKernelmode = v / 10000
		}

	})
	if err != nil {
		return containerStat, err
	}

	containerStat.CPUStats.CPUUsage.TotalUsage = containerStat.CPUStats.CPUUsage.UsageInUsermode + containerStat.CPUStats.CPUUsage.UsageInKernelmode

	err = populateCgroupKeyValue(prefix, "", cgroupParent, "memory.current", func(key string, v int64) {
		containerStat.MemoryStats.Usage = v
	})
	if err != nil {
		return containerStat, err
	}

	// memory.peak는 kernel 5.19+에만 존재 — 없으면 MaxUsage 0 유지
	populateCgroupKeyValue(prefix, "", cgroupParent, "memory.peak", func(key string, v int64) {
		containerStat.MemoryStats.MaxUsage = v
	})

	// memory.swap.current — swap 사용량(bytes). swap 미구성/파일 부재는 미수집으로 무시
	populateCgroupKeyValue(prefix, "", cgroupParent, "memory.swap.current", func(key string, v int64) {
		containerStat.MemoryStats.SwapUsage = v
	})

	err = populateCgroupKeyValue(prefix, "", cgroupParent, "memory.events", func(key string, v int64) {
		if key == "oom" {
			containerStat.MemoryStats.FailCnt = int(v)
		}
	})
	if err != nil {
		return containerStat, err
	}

	containerStat.MemoryStats.Limit = memoryLimit

	if containerStat.MemoryStats.Limit == 0 {
		err = populateFileValues(prefix, "/proc/meminfo", func(tokens []string) {
			if len(tokens) < 2 {
				return
			}

			if tokens[0] == "MemTotal:" {
				memtotal := stringutil.ToInt64(tokens[1])
				switch tokens[2] {
				case "kB":
					memtotal *= 1000
				case "gB":
					memtotal *= 1000 * 1000
				case "tB":
					memtotal *= 1000 * 1000 * 1000
				case "KB":
					memtotal *= 1024
				case "GB":
					memtotal *= 1024 * 1024
				case "TB":
					memtotal *= 1024 * 1024 * 1024
				case "KiB":
					memtotal *= 1024
				case "GiB":
					memtotal *= 1024 * 1024
				case "TiB":
					memtotal *= 1024 * 1024 * 1024
				}
				containerStat.MemoryStats.Limit = memtotal
			}
		})
		if err != nil {
			return containerStat, err
		}
	}

	err = populateCgroupKeyValue(prefix, "", cgroupParent, "memory.stat", func(key string, v int64) {
		switch key {
		case "cache":
			containerStat.MemoryStats.Stats.Cache = v
		case "rss":
			containerStat.MemoryStats.Stats.Rss = v
		case "rss_huge":
			containerStat.MemoryStats.Stats.RssHuge = v
		case "mapped_file":
			containerStat.MemoryStats.Stats.MappedFile = v
		case "dirty":
			containerStat.MemoryStats.Stats.Dirty = v
		case "writeback":
			containerStat.MemoryStats.Stats.Writeback = v
		case "pgpgin":
			containerStat.MemoryStats.Stats.Pgpgin = v
		case "pgpgout":
			containerStat.MemoryStats.Stats.Pgpgout = v
		case "pgfault":
			containerStat.MemoryStats.Stats.Pgfault = v
		case "pgmajfault":
			containerStat.MemoryStats.Stats.Pgmajfault = v
		case "inactive_anon":
			containerStat.MemoryStats.Stats.InactiveAnon = v
		case "active_anon":
			containerStat.MemoryStats.Stats.ActiveAnon = v
		case "inactive_file":
			containerStat.MemoryStats.Stats.InactiveFile = v
		case "active_file":
			containerStat.MemoryStats.Stats.ActiveFile = v
		case "unevictable":
			containerStat.MemoryStats.Stats.Unevictable = v
		case "hierarchical_memory_limit":
			containerStat.MemoryStats.Stats.HierarchicalMemoryLimit = v
		case "total_cache":
			containerStat.MemoryStats.Stats.TotalCache = v
		//cgroupV2 대응
		case "file":
			containerStat.MemoryStats.Stats.TotalCache = v
		case "total_rss":
			containerStat.MemoryStats.Stats.TotalRss = v
		//cgroupV2 대응
		case "anon":
			containerStat.MemoryStats.Stats.TotalRss = v
		case "total_rss_huge":
			containerStat.MemoryStats.Stats.TotalRssHuge = v
		case "total_mapped_file":
			containerStat.MemoryStats.Stats.TotalMappedFile = v
		//cgroupV2 대응
		case "file_mapped":
			containerStat.MemoryStats.Stats.TotalMappedFile = v
		case "total_dirty":
			containerStat.MemoryStats.Stats.TotalDirty = v
		case "total_writeback":
			containerStat.MemoryStats.Stats.TotalWriteback = v
		case "total_pgpgin":
			containerStat.MemoryStats.Stats.TotalPgpgin = v
		case "total_pgpgout":
			containerStat.MemoryStats.Stats.TotalPgpgout = v
		case "total_pgfault":
			containerStat.MemoryStats.Stats.TotalPgfault = v
		case "total_pgmajfault":
			containerStat.MemoryStats.Stats.TotalPgmajfault = v
		case "total_inactive_anon":
			containerStat.MemoryStats.Stats.TotalInactiveAnon = v
		case "total_active_anon":
			containerStat.MemoryStats.Stats.TotalActiveAnon = v
		case "total_inactive_file":
			containerStat.MemoryStats.Stats.TotalInactiveFile = v
		case "total_active_file":
			containerStat.MemoryStats.Stats.TotalActiveFile = v
		case "total_unevictable":
			containerStat.MemoryStats.Stats.TotalUnevictable = v
		}
	})
	if err != nil {
		return containerStat, err
	}
	// io.stat 포맷: "MAJ:MIN rbytes=N wbytes=N rios=N wios=N ..." — 디바이스별로 분해 수집
	err = populateCgroupValues(prefix, "", cgroupParent, "io.stat", func(words []string) {
		if len(words) < 2 {
			return
		}
		strmajor, strminor := stringutil.Split2(words[0], ":")
		if strmajor == "" && strminor == "" {
			return
		}
		major := int(stringutil.ToInt64(strmajor))
		minor := int(stringutil.ToInt64(strminor))
		for _, word := range words[1:] {
			key, val := stringutil.Split2(word, "=")
			v := stringutil.ToInt64(val)
			switch key {
			case "rbytes":
				containerStat.BlkioStats.IoServiceBytesRecursive = append(containerStat.BlkioStats.IoServiceBytesRecursive, whatap_model.BlkDeviceValue{Major: major, Minor: minor, Op: "Read", Value: v})
			case "wbytes":
				containerStat.BlkioStats.IoServiceBytesRecursive = append(containerStat.BlkioStats.IoServiceBytesRecursive, whatap_model.BlkDeviceValue{Major: major, Minor: minor, Op: "Write", Value: v})
			case "rios":
				containerStat.BlkioStats.IoServicedRecursive = append(containerStat.BlkioStats.IoServicedRecursive, whatap_model.BlkDeviceValue{Major: major, Minor: minor, Op: "Read", Value: v})
			case "wios":
				containerStat.BlkioStats.IoServicedRecursive = append(containerStat.BlkioStats.IoServicedRecursive, whatap_model.BlkDeviceValue{Major: major, Minor: minor, Op: "Write", Value: v})
			}
		}
	})
	if err != nil {
		return containerStat, err
	}

	// PSI(Pressure Stall Information) — 컨테이너 cgroup과 부모 파드 슬라이스에서 각각 직접 읽는다.
	// PSI는 가산 불가 지표라 파드 수준을 컨테이너 합산으로 만들 수 없다(KAZAA-3031).
	// 미가용(파일 부재·PSI 비활성)은 오류가 아닌 nil로 남겨 필드 자체를 생략한다.
	containerStat.Pressure = readCgroupPressure(prefix, cgroupParent, false)
	containerStat.PodPressure = readCgroupPressure(prefix, cgroupParent, true)

	err = populateFileValues(prefix, filepath.Join("proc", fmt.Sprint(pid), "net/dev"), func(tokens []string) {
		// fmt.Println("GetContainerStats step -7.1 ", tokens)
		if len(tokens) < 13 {
			return
		}

		deviceId := tokens[0]

		if !strings.Contains(deviceId, ":") || deviceId == "lo:" {
			return
		}
		containerStat.NetworkStats.RxBytes += stringutil.ToInt64(tokens[1])
		containerStat.NetworkStats.RxPackets += stringutil.ToInt64(tokens[2])
		containerStat.NetworkStats.RxErrors += stringutil.ToInt64(tokens[3])
		containerStat.NetworkStats.RxDropped += stringutil.ToInt64(tokens[4])

		containerStat.NetworkStats.TxBytes += stringutil.ToInt64(tokens[9])
		containerStat.NetworkStats.TxPackets += stringutil.ToInt64(tokens[10])
		containerStat.NetworkStats.TxErrors += stringutil.ToInt64(tokens[11])
		containerStat.NetworkStats.TxDropped += stringutil.ToInt64(tokens[12])
	})

	return containerStat, err
}

// parsePSILine은 "some avg10=0.00 avg60=0.00 avg300=0.00 total=12345" 형식에서 total(μs)만 수집한다.
// total을 찾아 반영했으면 true를 반환한다.
func parsePSILine(words []string, pv *whatap_model.PressureValues) bool {
	if len(words) < 2 {
		return false
	}
	kind := words[0]
	if kind != "some" && kind != "full" {
		return false
	}
	for _, word := range words[1:] {
		key, val := stringutil.Split2(word, "=")
		if key == "total" {
			if kind == "some" {
				pv.SomeTotal = stringutil.ToInt64(val)
			} else {
				pv.FullTotal = stringutil.ToInt64(val)
			}
			return true
		}
	}
	return false
}

// readCgroupPressure는 cgroup 디렉토리(pod=true면 부모 파드 슬라이스)의 {cpu,memory,io}.pressure를 읽는다.
// 파일 부재(커널 4.20 미만·cgroup v1)나 PSI 비활성(RHEL 계열: 읽기 실패로 파싱 0줄)은 nil을 반환해
// 오류가 아닌 미가용으로 구분한다. cpu.pressure의 full 라인은 커널 5.13+에만 존재 — 없으면 0 유지.
func readCgroupPressure(prefix string, cgroupsPath string, pod bool) *whatap_model.PressureStats {
	populate := populateCgroupValues
	if pod {
		populate = populatePodCgroupValues
	}
	ps := &whatap_model.PressureStats{}
	targets := []struct {
		filename string
		pv       *whatap_model.PressureValues
	}{
		{"cpu.pressure", &ps.CPU},
		{"memory.pressure", &ps.Memory},
		{"io.pressure", &ps.IO},
	}
	found := false
	for _, target := range targets {
		pv := target.pv
		populate(prefix, "", cgroupsPath, target.filename, func(words []string) {
			if parsePSILine(words, pv) {
				found = true
			}
		})
	}
	if !found {
		return nil
	}
	return ps
}
