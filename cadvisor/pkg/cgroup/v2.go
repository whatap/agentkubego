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
