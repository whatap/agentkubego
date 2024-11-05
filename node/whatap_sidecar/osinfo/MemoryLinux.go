//go:build linux
// +build linux

package osinfo

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	sidecarpack "whatap.io/k8s/sidecar/lang/pack"
)

// cpuStat CPUStat
type vmstatRaw struct {
	Timestamp  int64
	PageFaults int64
}

func toBytes(unit string) uint64 {
	var ret uint64
	switch strings.ToLower(unit) {
	case "tb":
		ret = 0x10000000000
	case "gb":
		ret = 0x40000000
	case "mb":
		ret = 0x100000
	case "kb":
		ret = 0x400
	default:
		ret = 1
	}
	return ret
}

// ParseMemoryStat ParseMemoryStat
func ParseMemoryStat() (LinuxMemoryStat, error) {
	var memoryStat LinuxMemoryStat
	filepath := "/proc/meminfo"
	file, err := os.Open(filepath)
	if err != nil {
		return memoryStat, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	isMemAvailableFound := false
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		intValue, _ := strconv.ParseInt(words[1], 10, 64)
		value := uint64(intValue)

		switch strings.ToLower(words[0]) {
		case "memtotal:":
			memoryStat.Total = value * toBytes(words[2])
		case "memfree:":
			memoryStat.Free = value * toBytes(words[2])
		case "memavailable:":
			isMemAvailableFound = true
			memoryStat.Available = value * toBytes(words[2])
		case "buffers:":
			memoryStat.Buffers = value * toBytes(words[2])
		case "cached:":
			memoryStat.Cached = value * toBytes(words[2])
		case "shmem:":
			memoryStat.Shared = value * toBytes(words[2])
		case "swaptotal:":
			memoryStat.SwapTotal = value * toBytes(words[2])
		case "swapfree:":
			memoryStat.SwapFree = value * toBytes(words[2])
		case "slab:":
			memoryStat.Slab = value * toBytes(words[2])
		case "sreclaimable:":
			memoryStat.SReclaimable = value * toBytes(words[2])
		case "sunreclaim:":
			memoryStat.SUnreclaim = value * toBytes(words[2])
		}
	}

	if !isMemAvailableFound {
		memoryStat.Used = memoryStat.Total - memoryStat.Free - memoryStat.Cached - memoryStat.Buffers
	} else {
		memoryStat.Used = memoryStat.Total - memoryStat.Available
	}
	memoryStat.Pused = 100.0 * float32(memoryStat.Used) / float32(memoryStat.Total)
	memoryStat.Pavailable = 100.0 * float32(memoryStat.Available) / float32(memoryStat.Total)
	memoryStat.SwapUsed = uint64(memoryStat.SwapTotal - memoryStat.SwapFree)
	if memoryStat.SwapTotal > 0 {
		memoryStat.SwapPused = float32(float64(memoryStat.SwapUsed) / float64(memoryStat.SwapTotal) * float64(100.0))
	} else {
		memoryStat.SwapPused = 0
	}

	return memoryStat, nil
}

func ParseVmStat(memoryStat *LinuxMemoryStat) (*LinuxMemoryStat, error) {
	filepath := "/proc/vmstat"
	file, err := os.Open(filepath)
	if err != nil {
		return memoryStat, err
	}
	defer file.Close()

	var lastVmstatRaw vmstatRaw
	timestamp := time.Now().UnixNano()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		intValue, _ := strconv.ParseInt(words[1], 10, 64)
		switch strings.ToLower(words[0]) {
		case "pgfault":
			loadedCache := LoadCache(filepath)
			if loadedCache != nil {
				lastVmstatRaw = loadedCache.(vmstatRaw)
				timediff := float32(timestamp - lastVmstatRaw.Timestamp)
				memoryStat.PageFaults = float32(float64(intValue-lastVmstatRaw.PageFaults) / float64(timediff) * float64(nanotimes))
			} else {
				memoryStat.PageFaults = 0
			}
			lastVmstatRaw.Timestamp = timestamp
			lastVmstatRaw.PageFaults = intValue
			SaveCache(filepath, lastVmstatRaw)
		}
	}

	return memoryStat, nil
}

func GetMemoryUtil() sidecarpack.Memory {
	stat, _ := ParseMemoryStat()
	_, _ = ParseVmStat(&stat)

	p := new(sidecarpack.MemoryLinux)
	p.Total = int64(stat.Total)
	p.Free = int64(stat.Free)
	p.Cached = int64(stat.Cached)
	p.Used = int64(stat.Used)
	p.Pused = stat.Pused
	p.Available = int64(stat.Available)
	p.Pavailable = stat.Pavailable

	p.Buffers = int64(stat.Buffers)
	p.Shared = int64(stat.Shared)

	p.SwapUsed = int64(stat.SwapUsed)
	p.SwapPused = float32(stat.SwapPused)
	p.SwapTotal = int64(stat.SwapTotal)
	p.PageFault = stat.PageFaults
	p.Slab = int64(stat.Slab)
	p.SReclaimable = int64(stat.SReclaimable)
	p.SUnreclaim = int64(stat.SUnreclaim)

	return p
}
