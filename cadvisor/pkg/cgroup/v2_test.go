package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	whatap_model "github.com/whatap/kube/cadvisor/pkg/model"
)

const testCgroupParent = "kubepods/pod-test/testcontainer"

func writeTestFile(t *testing.T, prefix string, relpath string, content string) {
	t.Helper()
	fullpath := filepath.Join(prefix, relpath)
	if err := os.MkdirAll(filepath.Dir(fullpath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullpath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupCgroupV2Fixture(t *testing.T, withMemoryPeak bool) string {
	prefix := t.TempDir()
	writeTestFile(t, prefix, "proc/stat", "cpu  100 200 300 400 500 600 700 800 900\n")
	writeTestFile(t, prefix, "proc/42/net/dev",
		"Inter-|   Receive                                                |  Transmit\n"+
			" face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n"+
			"  eth0: 1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0\n")

	cgroupDir := filepath.Join("sys/fs/cgroup", testCgroupParent)
	writeTestFile(t, prefix, filepath.Join(cgroupDir, "cpu.stat"),
		"usage_usec 1000000\nuser_usec 600000\nsystem_usec 400000\nnr_periods 1000\nnr_throttled 3\nthrottled_usec 250000\n")
	writeTestFile(t, prefix, filepath.Join(cgroupDir, "memory.current"), "104857600\n")
	writeTestFile(t, prefix, filepath.Join(cgroupDir, "memory.events"), "low 0\nhigh 0\nmax 0\noom 2\noom_kill 1\n")
	writeTestFile(t, prefix, filepath.Join(cgroupDir, "memory.stat"), "anon 1000\nfile 2000\nfile_mapped 300\n")
	writeTestFile(t, prefix, filepath.Join(cgroupDir, "io.stat"),
		"8:0 rbytes=1000 wbytes=2000 rios=10 wios=20 dbytes=0 dios=0\n259:1 rbytes=500 wbytes=600 rios=5 wios=6\n")
	if withMemoryPeak {
		writeTestFile(t, prefix, filepath.Join(cgroupDir, "memory.peak"), "209715200\n")
	}
	return prefix
}

func TestGetContainerStatsCgroupV2(t *testing.T) {
	prefix := setupCgroupV2Fixture(t, true)

	stat, err := GetContainerStatsCgroupV2(prefix, "testcontainer", "test", testCgroupParent, 0, 42, 1<<30)
	if err != nil {
		t.Fatal(err)
	}

	// throttled_usec(μs)은 v1 throttled_time(ns) 계약에 맞춰 ×1000 정규화되어야 한다
	if stat.CPUStats.ThrottlingData.ThrottledTime != 250000*1000 {
		t.Errorf("ThrottledTime=%d, want %d", stat.CPUStats.ThrottlingData.ThrottledTime, 250000*1000)
	}
	if stat.CPUStats.ThrottlingData.Periods != 1000 {
		t.Errorf("Periods=%d, want 1000", stat.CPUStats.ThrottlingData.Periods)
	}
	if stat.CPUStats.ThrottlingData.ThrottledPeriods != 3 {
		t.Errorf("ThrottledPeriods=%d, want 3", stat.CPUStats.ThrottlingData.ThrottledPeriods)
	}

	if stat.MemoryStats.Usage != 104857600 {
		t.Errorf("Usage=%d, want 104857600", stat.MemoryStats.Usage)
	}
	if stat.MemoryStats.MaxUsage != 209715200 {
		t.Errorf("MaxUsage=%d, want 209715200", stat.MemoryStats.MaxUsage)
	}
	if stat.MemoryStats.FailCnt != 2 {
		t.Errorf("FailCnt=%d, want 2", stat.MemoryStats.FailCnt)
	}

	wantBytes := []whatap_model.BlkDeviceValue{
		{Major: 8, Minor: 0, Op: "Read", Value: 1000},
		{Major: 8, Minor: 0, Op: "Write", Value: 2000},
		{Major: 259, Minor: 1, Op: "Read", Value: 500},
		{Major: 259, Minor: 1, Op: "Write", Value: 600},
	}
	if len(stat.BlkioStats.IoServiceBytesRecursive) != len(wantBytes) {
		t.Fatalf("IoServiceBytesRecursive len=%d, want %d: %v",
			len(stat.BlkioStats.IoServiceBytesRecursive), len(wantBytes), stat.BlkioStats.IoServiceBytesRecursive)
	}
	for i, want := range wantBytes {
		if stat.BlkioStats.IoServiceBytesRecursive[i] != want {
			t.Errorf("IoServiceBytesRecursive[%d]=%v, want %v", i, stat.BlkioStats.IoServiceBytesRecursive[i], want)
		}
	}

	wantOps := []whatap_model.BlkDeviceValue{
		{Major: 8, Minor: 0, Op: "Read", Value: 10},
		{Major: 8, Minor: 0, Op: "Write", Value: 20},
		{Major: 259, Minor: 1, Op: "Read", Value: 5},
		{Major: 259, Minor: 1, Op: "Write", Value: 6},
	}
	if len(stat.BlkioStats.IoServicedRecursive) != len(wantOps) {
		t.Fatalf("IoServicedRecursive len=%d, want %d: %v",
			len(stat.BlkioStats.IoServicedRecursive), len(wantOps), stat.BlkioStats.IoServicedRecursive)
	}
	for i, want := range wantOps {
		if stat.BlkioStats.IoServicedRecursive[i] != want {
			t.Errorf("IoServicedRecursive[%d]=%v, want %v", i, stat.BlkioStats.IoServicedRecursive[i], want)
		}
	}
}

func TestGetContainerStatsCgroupV2WithoutMemoryPeak(t *testing.T) {
	// kernel 5.19 미만: memory.peak 부재 시 MaxUsage는 0 유지, 에러 없이 수집 계속
	prefix := setupCgroupV2Fixture(t, false)

	stat, err := GetContainerStatsCgroupV2(prefix, "testcontainer", "test", testCgroupParent, 0, 42, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	if stat.MemoryStats.MaxUsage != 0 {
		t.Errorf("MaxUsage=%d, want 0 (memory.peak absent)", stat.MemoryStats.MaxUsage)
	}
	if stat.MemoryStats.Usage != 104857600 {
		t.Errorf("Usage=%d, want 104857600", stat.MemoryStats.Usage)
	}
}

func writePressureFiles(t *testing.T, prefix string, dir string, base int64) {
	t.Helper()
	writeTestFile(t, prefix, filepath.Join(dir, "cpu.pressure"),
		fmt.Sprintf("some avg10=0.12 avg60=0.10 avg300=0.05 total=%d\nfull avg10=0.01 avg60=0.00 avg300=0.00 total=%d\n", base+1, base+2))
	writeTestFile(t, prefix, filepath.Join(dir, "memory.pressure"),
		fmt.Sprintf("some avg10=0.00 avg60=0.00 avg300=0.00 total=%d\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=%d\n", base+3, base+4))
	writeTestFile(t, prefix, filepath.Join(dir, "io.pressure"),
		fmt.Sprintf("some avg10=0.00 avg60=0.00 avg300=0.00 total=%d\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=%d\n", base+5, base+6))
}

func TestGetContainerStatsCgroupV2PressureAndSwap(t *testing.T) {
	prefix := setupCgroupV2Fixture(t, false)
	containerDir := filepath.Join("sys/fs/cgroup", testCgroupParent)
	podDir := filepath.Dir(containerDir)
	writePressureFiles(t, prefix, containerDir, 100)
	writePressureFiles(t, prefix, podDir, 1000)
	writeTestFile(t, prefix, filepath.Join(containerDir, "memory.swap.current"), "12345\n")

	stat, err := GetContainerStatsCgroupV2(prefix, "testcontainer", "test", testCgroupParent, 0, 42, 1<<30)
	if err != nil {
		t.Fatal(err)
	}

	if stat.Pressure == nil {
		t.Fatal("Pressure=nil, want values")
	}
	wantContainer := whatap_model.PressureStats{
		CPU:    whatap_model.PressureValues{SomeTotal: 101, FullTotal: 102},
		Memory: whatap_model.PressureValues{SomeTotal: 103, FullTotal: 104},
		IO:     whatap_model.PressureValues{SomeTotal: 105, FullTotal: 106},
	}
	if *stat.Pressure != wantContainer {
		t.Errorf("Pressure=%+v, want %+v", *stat.Pressure, wantContainer)
	}

	// 파드 수준은 컨테이너 합산이 아닌 파드 슬라이스 파일의 값 그대로여야 한다
	if stat.PodPressure == nil {
		t.Fatal("PodPressure=nil, want values")
	}
	wantPod := whatap_model.PressureStats{
		CPU:    whatap_model.PressureValues{SomeTotal: 1001, FullTotal: 1002},
		Memory: whatap_model.PressureValues{SomeTotal: 1003, FullTotal: 1004},
		IO:     whatap_model.PressureValues{SomeTotal: 1005, FullTotal: 1006},
	}
	if *stat.PodPressure != wantPod {
		t.Errorf("PodPressure=%+v, want %+v", *stat.PodPressure, wantPod)
	}

	if stat.MemoryStats.SwapUsage != 12345 {
		t.Errorf("SwapUsage=%d, want 12345", stat.MemoryStats.SwapUsage)
	}
}

func TestGetContainerStatsCgroupV2PressureAbsent(t *testing.T) {
	// pressure 파일 부재(커널 4.20 미만 등): 에러 없이 nil로 남겨 미가용으로 구분
	prefix := setupCgroupV2Fixture(t, false)

	stat, err := GetContainerStatsCgroupV2(prefix, "testcontainer", "test", testCgroupParent, 0, 42, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Pressure != nil {
		t.Errorf("Pressure=%+v, want nil (files absent)", *stat.Pressure)
	}
	if stat.PodPressure != nil {
		t.Errorf("PodPressure=%+v, want nil (files absent)", *stat.PodPressure)
	}
	if stat.MemoryStats.SwapUsage != 0 {
		t.Errorf("SwapUsage=%d, want 0 (file absent)", stat.MemoryStats.SwapUsage)
	}
}

func TestGetContainerStatsCgroupV2PressureZeroAndEmpty(t *testing.T) {
	// 전부-0이어도 파싱 가능한 라인이 있으면 가용(0 값 노출), 내용이 비면(PSI 비활성 읽기 실패 상당) nil
	prefix := setupCgroupV2Fixture(t, false)
	containerDir := filepath.Join("sys/fs/cgroup", testCgroupParent)
	writePressureFiles(t, prefix, containerDir, -1) // total=0부터 시작
	podDir := filepath.Dir(containerDir)
	writeTestFile(t, prefix, filepath.Join(podDir, "cpu.pressure"), "")
	writeTestFile(t, prefix, filepath.Join(podDir, "memory.pressure"), "")
	writeTestFile(t, prefix, filepath.Join(podDir, "io.pressure"), "")

	stat, err := GetContainerStatsCgroupV2(prefix, "testcontainer", "test", testCgroupParent, 0, 42, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Pressure == nil {
		t.Fatal("Pressure=nil, want zero values (파일이 읽히면 0도 유효)")
	}
	if stat.Pressure.CPU.SomeTotal != 0 || stat.Pressure.IO.FullTotal != 5 {
		t.Errorf("Pressure=%+v, want base -1 values", *stat.Pressure)
	}
	if stat.PodPressure != nil {
		t.Errorf("PodPressure=%+v, want nil (빈 파일 = 파싱 0줄)", *stat.PodPressure)
	}
}

func TestReadCgroupPressureSystemdPodSlice(t *testing.T) {
	// systemd 드라이버 경로: 컨테이너 scope의 부모 = kubepods-*-pod<uid>.slice(파드 슬라이스)
	prefix := t.TempDir()
	cgroupsPath := "kubepods-burstable-podabc123.slice:cri-containerd:deadbeef"
	podDir := "sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podabc123.slice"
	writeTestFile(t, prefix, filepath.Join(podDir, "cpu.pressure"),
		"some avg10=0.00 avg60=0.00 avg300=0.00 total=777\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=88\n")

	ps := readCgroupPressure(prefix, cgroupsPath, true)
	if ps == nil {
		t.Fatal("pod pressure=nil, want values")
	}
	if ps.CPU.SomeTotal != 777 || ps.CPU.FullTotal != 88 {
		t.Errorf("pod CPU pressure=%+v, want some=777 full=88", ps.CPU)
	}

	// 컨테이너 scope 디렉토리에는 pressure가 없으므로 컨테이너 수준은 미가용
	if got := readCgroupPressure(prefix, cgroupsPath, false); got != nil {
		t.Errorf("container pressure=%+v, want nil", *got)
	}
}
