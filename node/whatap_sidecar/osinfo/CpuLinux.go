//go:build linux
// +build linux

package osinfo

/*
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <stdlib.h>
*/
import "C"

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
	sidecarpack "whatap.io/k8s/sidecar/lang/pack"
)

var (
	cpuCoreCount int
)

// GetCpuNum GetCpuNum
func GetCPUNum() int {
	if cpuCoreCount < 1 {
		numcpu := runtime.NumCPU()
		numcpu = 0
		if numcpu < 1 {
			f, err := os.Open("/proc/stat")
			if err != nil {
				return 1
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "cpu") {
					numcpu += 1
				}
			}
			numcpu -= 1
		}
		if numcpu > 0 {
			cpuCoreCount = numcpu
		}
	}

	return cpuCoreCount
}

// cpuStat CPUStat
type cpuStatRaw struct {
	TotalCpuTime  int64
	Device        string
	User          int64
	Nice          int64
	System        int64
	Idle          int64
	Iowait        int64
	Irq           int64
	Softirq       int64
	Steal         int64
	Load1         int32
	Load5         int32
	Load15        int32
	Ctxt          int64
	Timestamp     int64
	NewProcForked int64
}

func parseCPUStat() ([]LinuxCPUStat, error) {

	loads := make([]C.double, 3)
	ret := int32(C.int(C.getloadavg((*C.double)(unsafe.Pointer(&loads[0])), 3)))
	var load1, load5, load15 float32
	if ret == 3 {
		load1 = float32(loads[0])
		load5 = float32(loads[1])
		load15 = float32(loads[2])
	} else {
		load1 = 0
		load5 = 0
		load15 = 0
	}

	filepath := "/proc/stat"
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	cpuCoreCount := GetCPUNum()
	rawCPUStats := make([]cpuStatRaw, cpuCoreCount+1)

	scanner := bufio.NewScanner(file)
	i := 0
	timestamp := dateutil.SysNow()
	var ctxt, processes, procsRunning, procsBlocked int64
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if strings.HasPrefix(line, "cpu") && i < (cpuCoreCount+1) {
			if len(words) > 8 {
				rawCPUStats[i].Device = words[0]
				rawCPUStats[i].User, _ = strconv.ParseInt(words[1], 10, 64)
				rawCPUStats[i].Nice, _ = strconv.ParseInt(words[2], 10, 64)
				rawCPUStats[i].System, _ = strconv.ParseInt(words[3], 10, 64)
				rawCPUStats[i].Idle, _ = strconv.ParseInt(words[4], 10, 64)
				rawCPUStats[i].Iowait, _ = strconv.ParseInt(words[5], 10, 64)
				rawCPUStats[i].Irq, _ = strconv.ParseInt(words[6], 10, 64)
				rawCPUStats[i].Softirq, _ = strconv.ParseInt(words[7], 10, 64)
				rawCPUStats[i].Steal, _ = strconv.ParseInt(words[8], 10, 64)
				rawCPUStats[i].TotalCpuTime = rawCPUStats[i].User + rawCPUStats[i].Nice + rawCPUStats[i].System +
					rawCPUStats[i].Idle + rawCPUStats[i].Iowait + rawCPUStats[i].Irq + rawCPUStats[i].Softirq + rawCPUStats[i].Steal
				rawCPUStats[i].Timestamp = timestamp
				i++
			} else {
				return nil, fmt.Errorf("malformed /proc/stats")
			}
		} else if strings.HasPrefix(line, "ctxt") && len(words) > 1 {
			ctxt, _ = strconv.ParseInt(words[1], 10, 64)
		} else if strings.HasPrefix(line, "processes") && len(words) > 1 {
			processes, _ = strconv.ParseInt(words[1], 10, 64)
		} else if strings.HasPrefix(line, "procs_running") && len(words) > 1 {
			procsRunning, _ = strconv.ParseInt(words[1], 10, 64)
		} else if strings.HasPrefix(line, "procs_blocked") && len(words) > 1 {
			procsBlocked, _ = strconv.ParseInt(words[1], 10, 64)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for i, rawCPUStat := range rawCPUStats {
		if rawCPUStat.Device == "cpu" {
			rawCPUStatModify := &rawCPUStats[i]
			rawCPUStatModify.Ctxt = ctxt
			rawCPUStatModify.NewProcForked = processes
		}
	}

	scClkTck := float32(100)

	cpuStats := make([]LinuxCPUStat, cpuCoreCount+1)
	var rawCPUStats2 []cpuStatRaw
	loadedCache := LoadCache(filepath)
	if loadedCache != nil {
		rawCPUStats2 = loadedCache.([]cpuStatRaw)
	}

	if rawCPUStats2 != nil {
		for i, rawCPUStat := range rawCPUStats {
			timediff := float32(rawCPUStat.TotalCpuTime - rawCPUStats2[i].TotalCpuTime)
			cpuStats[i].Device = rawCPUStat.Device
			cpuStats[i].User = 100.0 * float32(rawCPUStat.User-rawCPUStats2[i].User) / float32(timediff)
			cpuStats[i].Nice = 100.0 * float32(rawCPUStat.Nice-rawCPUStats2[i].Nice) / float32(timediff)
			cpuStats[i].System = 100.0 * float32(rawCPUStat.System-rawCPUStats2[i].System) / float32(timediff)
			cpuStats[i].Idle = 100.0 * float32(rawCPUStat.Idle-rawCPUStats2[i].Idle) / float32(timediff)
			cpuStats[i].Iowait = 100.0 * float32(rawCPUStat.Iowait-rawCPUStats2[i].Iowait) / float32(timediff)
			cpuStats[i].Irq = 100.0 * float32(rawCPUStat.Irq-rawCPUStats2[i].Irq) / scClkTck / float32(timediff)
			cpuStats[i].Softirq = 100.0 * float32(rawCPUStat.Softirq-rawCPUStats2[i].Softirq) / float32(timediff)
			cpuStats[i].Steal = 100.0 * float32(rawCPUStat.Steal-rawCPUStats2[i].Steal) / float32(timediff)
			cpuStats[i].Load1 = load1
			cpuStats[i].Load5 = load5
			cpuStats[i].Load15 = load15
			cpuStats[i].Ctxt = float32(rawCPUStat.Ctxt-rawCPUStats2[i].Ctxt) / float32(rawCPUStat.Timestamp-rawCPUStats2[i].Timestamp) * float32(1000)
			cpuStats[i].LoadPerCore = load1 / float32(cpuCoreCount)
			cpuStats[i].ProcsRunning = float32(procsRunning)
			cpuStats[i].ProcsBlocked = float32(procsBlocked)
			cpuStats[i].NewProcForked = float32(rawCPUStat.NewProcForked - rawCPUStats2[i].NewProcForked)
		}
	}

	SaveCache(filepath, rawCPUStats)

	return cpuStats, nil
}

// GetCPUUtil GetCPUUtil
func GetCPUUtil() (sidecarpack.Cpu, []sidecarpack.Cpu) {
	cpuStats, e := parseCPUStat()
	for e != nil || len(cpuStats[0].Device) < 1 {
		time.Sleep(1 * time.Second)
		cpuStats, e = parseCPUStat()
	}
	sz := len(cpuStats)
	var tot sidecarpack.Cpu
	var corecpu = make([]sidecarpack.Cpu, sz)
	j := 0
	for i := 0; i < len(cpuStats); i++ {
		c := new(sidecarpack.CpuLinux)
		c.User = cpuStats[i].User
		c.System = cpuStats[i].System
		c.Idle = cpuStats[i].Idle
		c.Nice = cpuStats[i].Nice
		c.Irq = cpuStats[i].Irq
		c.Softirq = cpuStats[i].Softirq
		c.Steal = cpuStats[i].Steal
		c.Iowait = cpuStats[i].Iowait
		c.Load1 = cpuStats[i].Load1
		c.Load5 = cpuStats[i].Load5
		c.Load15 = cpuStats[i].Load15
		c.Ctxt = cpuStats[i].Ctxt
		c.LoadPerCore = cpuStats[i].LoadPerCore
		c.ProcsRunning = cpuStats[i].ProcsRunning
		c.ProcsBlocked = cpuStats[i].ProcsBlocked
		c.NewProcForked = cpuStats[i].NewProcForked

		if cpuStats[i].Device == "cpu" {
			tot = c
		} else {
			corecpu[j] = c
			j++
		}
	}

	return tot, corecpu[:j]
}
