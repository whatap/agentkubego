//go:build linux && cgo

package osinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
)

// Uses the real collector with an isolated proc tree, never the host's proc.
func TestMeasureProcessPerformanceStatusPrefilter(t *testing.T) {
	root := t.TempDir()
	proc := filepath.Join(root, "proc")
	for _, dir := range []string{proc, filepath.Join(root, "etc")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeProcessStatusFixture(t, filepath.Join(proc, "uptime"), "1000.00 1000.00\n")
	writeProcessStatusFixture(t, filepath.Join(proc, "stat"), "cpu 10 20 30 40 50 60 70 80\n")
	writeProcessStatusFixture(t, filepath.Join(root, "etc", "passwd"), "fixture:x:1234:1234::/:/bin/false\n")
	cfg := whatap_config.GetConfig()
	oldRoot, oldTargets := cfg.HostPathPrefix, cfg.CollectKubeNodeProcessMetricTargetList
	oldPss, oldIO, oldDebug := cfg.CollectProcessPssEnabled, cfg.CollectProcessIO, cfg.Debug
	t.Cleanup(func() {
		cfg.HostPathPrefix, cfg.CollectKubeNodeProcessMetricTargetList = oldRoot, oldTargets
		cfg.CollectProcessPssEnabled, cfg.CollectProcessIO, cfg.Debug = oldPss, oldIO, oldDebug
	})
	cfg.HostPathPrefix, cfg.CollectKubeNodeProcessMetricTargetList = root, []string{"kubelet"}
	cfg.CollectProcessPssEnabled, cfg.CollectProcessIO, cfg.Debug = false, false, false
	add := func(pid, comm, name string, ppid int) {
		dir := filepath.Join(proc, pid)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if comm != "" {
			writeProcessStatusFixture(t, filepath.Join(dir, "comm"), comm+"\n")
		}
		writeProcessStatusFixture(t, filepath.Join(dir, "status"), "Name:\t"+name+"\nState:\tS (sleeping)\nPPid:\t"+strconv.Itoa(ppid)+"\nUid:\t1234\t1234\t1234\t1234\nVmSize:\t100 kB\nThreads:\t3\n")
		fields := make([]string, 22)
		for i := range fields {
			fields[i] = "0"
		}
		fields[0], fields[1], fields[2] = pid, "("+name+")", "S"
		fields[13], fields[14], fields[15], fields[16], fields[21] = "11", "12", "3", "4", "100"
		writeProcessStatusFixture(t, filepath.Join(dir, "stat"), strings.Join(fields, " ")+"\n")
		writeProcessStatusFixture(t, filepath.Join(dir, "cmdline"), name+"\x00--fixture\x00")
		writeProcessStatusFixture(t, filepath.Join(dir, "statm"), "100 7 2\n")
	}
	add("100", "kubelet", "kubelet", 1)
	add("101", "unrelated", "unrelated", 1)
	add("102", "", "kubelet", 1)        // comm failure must retain status fallback.
	add("103", "kubelet", "renamed", 1) // Real status must be revalidated.
	add("104", "kubelet", "kubelet", 2)
	add("105", "kubelet", "kubelet", 1)
	add("106", "containerd", "containerd", 1)
	add("2", "kubelet", "kubelet", 1)
	if err := os.Remove(filepath.Join(proc, "105", "status")); err != nil {
		t.Fatal(err)
	}
	got := measureProcessPerformance()
	if got == nil || len(*got) != 2 || (*got)["100"] == nil || (*got)["102"] == nil {
		t.Fatalf("first scan: got %v; want PIDs 100 and 102", got)
	}
	p := (*got)["100"]
	if p.Cmd1 != "kubelet" || p.Cmd2 != "kubelet --fixture " || p.User != "fixture" || p.PPid != 1 || p.Thcount != 3 || p.Cpu != 23 || p.ChildCpu != 7 || p.TotalCpuTime != 360 || p.MemoryBytes != int64(7*os.Getpagesize()) || p.SharedMemory != int64(2*os.Getpagesize()) {
		t.Fatalf("source fields changed: %+v", p)
	}
	// Reusing a PID and changing targets must be visible on the next request.
	cfg.CollectKubeNodeProcessMetricTargetList = []string{"containerd"}
	writeProcessStatusFixture(t, filepath.Join(proc, "100", "comm"), "containerd\n")
	writeProcessStatusFixture(t, filepath.Join(proc, "100", "status"), "Name:\tcontainerd\nPPid:\t1\n")
	writeProcessStatusFixture(t, filepath.Join(proc, "100", "statm"), "100 9 4\n")
	writeProcessStatusFixture(t, filepath.Join(proc, "stat"), "cpu 20 20 30 40 50 60 70 80\n")
	if err := os.RemoveAll(filepath.Join(proc, "102")); err != nil {
		t.Fatal(err)
	}
	got = measureProcessPerformance()
	if got == nil || len(*got) != 2 || (*got)["100"] == nil || (*got)["106"] == nil {
		t.Fatalf("second scan: got %v; want PIDs 100 and 106", got)
	}
	if p := (*got)["100"]; p.Cmd1 != "containerd" || p.TotalCpuTime != 370 || p.MemoryBytes != int64(9*os.Getpagesize()) {
		t.Fatalf("stale source fields: %+v", p)
	}
	cfg.CollectKubeNodeProcessMetricTargetList = nil
	if got = measureProcessPerformance(); got == nil || len(*got) != 0 {
		t.Fatalf("empty targets retained processes: %v", got)
	}
}
