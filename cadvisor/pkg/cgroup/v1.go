package cgroup

import (
	"fmt"
	"path/filepath"
	"strings"

	whatap_model "github.com/whatap/kube/cadvisor/pkg/model"
	"github.com/whatap/kube/tools/util/stringutil"
)

func GetContainerStatsCgroupV1(prefix string, containerId string, name string, cgroupParent string,
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
	err = populateCgroupKeyValue(prefix, "cpu", cgroupParent, "cpu.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "nr_periods" {
			containerStat.CPUStats.ThrottlingData.Periods = v
		} else if key == "throttled_time" {
			containerStat.CPUStats.ThrottlingData.ThrottledTime = v
		} else if key == "nr_throttled" {
			containerStat.CPUStats.ThrottlingData.ThrottledPeriods = v
		}
	})
	if err != nil {
		return containerStat, err
	}
	// fmt.Println("GetContainerStats step -2")
	err = populateCgroupKeyValue(prefix, "cpu", cgroupParent, "cpuacct.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "user" {
			containerStat.CPUStats.CPUUsage.UsageInUsermode = v
		} else if key == "system" {
			containerStat.CPUStats.CPUUsage.UsageInKernelmode = v
		}
	})
	if err != nil {
		return containerStat, err
	}

	populateCgroupKeyValue(prefix, "cpu", cgroupParent, "cpuacct.shares", func(key string, v int64) {
		containerStat.CPUStats.OnlineCpus = int(v)
	})

	containerStat.CPUStats.CPUUsage.TotalUsage = containerStat.CPUStats.CPUUsage.UsageInUsermode + containerStat.CPUStats.CPUUsage.UsageInKernelmode

	// fmt.Println("GetContainerStats step -3")
	err = populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.max_usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.MaxUsage = v
	})
	if err != nil {
		return containerStat, err
	}

	err = populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.Usage = v
	})
	if err != nil {
		return containerStat, err
	}

	err = populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.failcnt", func(key string, v int64) {
		containerStat.MemoryStats.FailCnt = int(v)
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

	err = populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.stat", func(key string, v int64) {
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
		case "total_rss":
			containerStat.MemoryStats.Stats.TotalRss = v
		case "total_rss_huge":
			containerStat.MemoryStats.Stats.TotalRssHuge = v
		case "total_mapped_file":
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
	callbackIoServiceBytesRecursive := func(words []string) {
		// fmt.Println("callbackIoServiceBytesRecursive ", words)
		if len(words) == 3 {
			strmajor, strminor := stringutil.Split2(words[0], ":")
			major := int(stringutil.ToInt64(strmajor))
			minor := int(stringutil.ToInt64(strminor))
			op := words[1]
			v := stringutil.ToInt64(words[2])

			ioServiceBytesRecursive := whatap_model.BlkDeviceValue{major, minor, op, v}
			containerStat.BlkioStats.IoServiceBytesRecursive = append(containerStat.BlkioStats.IoServiceBytesRecursive, ioServiceBytesRecursive)
		}
	}
	// fmt.Println("GetContainerStats step -4")
	err = populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.io_service_bytes_recursive", callbackIoServiceBytesRecursive)
	if err != nil {

		err = populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.throttle.io_service_bytes_recursive", callbackIoServiceBytesRecursive)
		if err != nil {
			return containerStat, err
		}
	}
	err = populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.throttle.io_service_bytes", callbackIoServiceBytesRecursive)
	if err != nil {
		return containerStat, err
	}
	callbackIoServicedRecursive := func(words []string) {
		// fmt.Println("callbackIoServicedRecursive ", words)
		if len(words) == 3 {
			strmajor, strminor := stringutil.Split2(words[0], ":")
			major := int(stringutil.ToInt64(strmajor))
			minor := int(stringutil.ToInt64(strminor))
			op := words[1]
			v := stringutil.ToInt64(words[2])

			ioServicedRecursive := whatap_model.BlkDeviceValue{major, minor, op, v}
			containerStat.BlkioStats.IoServicedRecursive = append(containerStat.BlkioStats.IoServicedRecursive, ioServicedRecursive)
		}
	}
	// fmt.Println("GetContainerStats step -5")
	err = populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.io_serviced_recursive", callbackIoServicedRecursive)
	if err != nil {
		err = populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.throttle.io_serviced_recursive", callbackIoServicedRecursive)
		if err != nil {
			return containerStat, err
		}
	}
	err = populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.throttle.io_serviced", callbackIoServicedRecursive)
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
