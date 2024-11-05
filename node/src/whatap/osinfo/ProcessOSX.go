//go:build darwin
// +build darwin

package osinfo

import (
	"fmt"
	"runtime"
	"time"
)

var (
	ProcessCmdPatternEnabled = true
)

type ProcessInfo struct {
	PPid          int
	Pid           int
	Cpu           float64
	MemoryBytes   int64
	MemoryPercent float32
	Timestamp     int
	ReadBytes     int64
	WriteBytes    int64
	Cmd1          string
	Cmd2          string
	ReadOctets    int64
	WriteOctets   int64
	User          string
	State         string
	CreateTime    int
}

func load(path string) *map[string]*ProcessInfo {
	loadedCache := LoadCache(path)
	if loadedCache != nil {
		castedloadedCache := loadedCache.(map[string]*ProcessInfo)
		return &castedloadedCache
	}

	return nil
}

func save(path string, fileList *map[string]*ProcessInfo) {
	SaveCache(path, *fileList)
}

func measureProcessPerformance() *map[string]*ProcessInfo {
	processMap := make(map[string]*ProcessInfo)

	parser := NewPosixProcessParser()
	parser.Populate()
	for _, p := range parser.ProcessList {
		pid := fmt.Sprint(p.Pid)
		processMap[pid] = p
	}

	return &processMap
}

func getProcessPerfList(old_pinfo_map *map[string]*ProcessInfo, new_pinfo_map *map[string]*ProcessInfo) *[]*ProcessPerfInfo {
	corecount := runtime.NumCPU()
	var processPerfInfos []*ProcessPerfInfo
	if old_pinfo_map != nil {
		for pid := range *new_pinfo_map {
			new_pinfo := (*new_pinfo_map)[pid]
			old_pinfo := (*old_pinfo_map)[pid]
			if old_pinfo != nil && old_pinfo.Cmd1 == new_pinfo.Cmd1 && old_pinfo.Cmd2 == new_pinfo.Cmd2 {
				timeelapsed := float64(new_pinfo.Timestamp - old_pinfo.Timestamp)
				pcpu := float32((new_pinfo.Cpu - old_pinfo.Cpu) / timeelapsed * 100.0 / float64(corecount))
				readbps := float32(new_pinfo.ReadBytes-old_pinfo.ReadBytes) / float32(timeelapsed)
				writebps := float32(new_pinfo.WriteBytes-old_pinfo.WriteBytes) / float32(timeelapsed)
				processPerfInfos = append(processPerfInfos, &ProcessPerfInfo{User: new_pinfo.User, PPid: int32(new_pinfo.PPid), Pid: int32(new_pinfo.Pid), State: new_pinfo.State,
					CreateTime: int64(new_pinfo.CreateTime), Cpu: pcpu,
					MemoryBytes: int64(new_pinfo.MemoryBytes), MemoryPercent: new_pinfo.MemoryPercent,
					WriteBps: writebps, ReadBps: readbps,
					Cmd1: new_pinfo.Cmd1,
					Cmd2: new_pinfo.Cmd2})
			}
		}
	}
	return &processPerfInfos
}

// GetProcessPerfList GetProcessPerfList
func GetProcessPerfList() (ret *[]*ProcessPerfInfo) {
	filepath := "/tmp/whatap_agent_wps"
	oldProcessInfoMap := load(filepath)
	newProcessInfoMap := measureProcessPerformance()

	save(filepath, newProcessInfoMap)
	processPerfs := getProcessPerfList(oldProcessInfoMap, newProcessInfoMap)
	if processPerfs == nil || len(*processPerfs) < 1 {
		time.Sleep(1 * time.Second)
		newProcessInfoMap2 := measureProcessPerformance()
		processPerfs = getProcessPerfList(newProcessInfoMap, newProcessInfoMap2)
	}

	return processPerfs
}
