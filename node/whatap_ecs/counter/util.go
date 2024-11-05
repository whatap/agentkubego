package counter

import (
	"bufio"
	"context"
	"fmt"
	"log"

	//"log"
	gonet "net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/lang/pack"
	"github.com/whatap/go-api/common/lang/value"
	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/common/util/iputil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/stringutil"
	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/session"
	"whatap.io/aws/ecs/text"

	docker_types "github.com/docker/docker/api/types"

	whatap_docker "github.com/whatap/kube/node/src/whatap/util/docker"
)

var (
	gcpu         float32
	gmem         float32
	cgroupPrefix string = "/cgroup"
	cgroupMode   string = CGROUP_LEGACY_HYBRID
)

const (
	METERINGTYPE         = "mtype"
	METERINGTYPE_ECS     = "ecs"
	CGROUP_LEGACY_HYBRID = "legacyhybrid"
	CGROUP_UNIFIED       = "unified"
)

func init() {
	cgroupMount, err := GetCgroupMountsPath()
	fmt.Println("util.go cgroupMount:", cgroupMount)
	if err == nil {
		if prefix, ok := cgroupMount["cgroup"]; ok {
			cgroupPrefix = prefix
			cgroupMode = CGROUP_LEGACY_HYBRID
		}

		if prefix, ok := cgroupMount["cgroup2"]; ok {
			cgroupPrefix = prefix
			cgroupMode = CGROUP_UNIFIED
		}
	}
}

func setNodePerf(cpu float32, mem float32) {
	gcpu = cpu
	gmem = mem
}

func getNodePerf() (float32, float32) {
	return gcpu, gmem
}

var (
	myaddr    int32
	myaddrerr error
)

func getMyAddr() int32 {
	if myaddr == 0 && myaddrerr == nil {

		addrs, err := gonet.InterfaceAddrs()
		if err != nil {
			myaddrerr = err
		}

		for _, a := range addrs {
			if ipnet, ok := a.(*gonet.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					io.ToInt(iputil.ToBytes(ipnet.IP.String()), 0)
				}
			}
		}
		myaddrerr = fmt.Errorf("addr not found")
	}

	return myaddr
}

func calcCpuUsage(containerstats *ContainerStat, lastContainerstats *ContainerStat) (float32, float32, float32) {

	systemcpudiff := containerstats.CPUStats.SystemCPUUsage - lastContainerstats.CPUStats.SystemCPUUsage
	//fmt.Println("calcCpuUsage", containerstats.CPUStats.SystemCPUUsage, lastContainerstats.CPUStats.SystemCPUUsage)
	totalCpu := float32(containerstats.CPUStats.CPUUsage.TotalUsage-lastContainerstats.CPUStats.CPUUsage.TotalUsage) / float32(systemcpudiff) * float32(100)
	userCpu := float32(containerstats.CPUStats.CPUUsage.UsageInUsermode-lastContainerstats.CPUStats.CPUUsage.UsageInUsermode) / float32(systemcpudiff) * float32(100)
	sysCpu := float32(containerstats.CPUStats.CPUUsage.UsageInKernelmode-lastContainerstats.CPUStats.CPUUsage.UsageInKernelmode) / float32(systemcpudiff) * float32(100)

	return totalCpu, userCpu, sysCpu
}

func sumIoServiceRecursive(containerstats *ContainerStat, readCallback func(int64), writeCallback func(write int64)) {
	for _, x := range containerstats.BlkioStats.IoServicedRecursive {
		if x.Op == "Read" {
			readCallback(x.Value)
		} else if x.Op == "Write" {
			writeCallback(x.Value)
		}
	}
}
func sumIoServiceBytesRecursive(containerstats *ContainerStat, readCallback func(int64), writeCallback func(write int64)) {

	for _, x := range containerstats.BlkioStats.IoServiceBytesRecursive {
		if x.Op == "Read" {
			readCallback(x.Value)
		} else if x.Op == "Write" {
			writeCallback(x.Value)
		}
	}
}

func calcPs(a int64, b int64, timediff int64) float32 {
	// log.Println("calcPS a:", a, " b:", b, " timediff:", timediff, " ps:", float32(a-b)/float32(timediff)*float32(1000))
	return float32(a-b) / float32(timediff) * float32(1000)
}

func calcBlkioUsage(containerstats *ContainerStat, lastContainerstats *ContainerStat) (float32, float32, float32, float32) {

	timediff := int64(containerstats.Read.Sub(lastContainerstats.Read).Nanoseconds() / 1000000)
	//containerstats.BlkioStats.IoServicedRecursive
	var blkioRbps, blkioRiops, blkioWbps, blkioWiops int64
	sumIoServiceRecursive(containerstats, func(read int64) {
		blkioRiops += read
	}, func(write int64) {
		blkioWiops += write
	})
	sumIoServiceBytesRecursive(containerstats, func(read int64) {
		blkioRbps += read
	}, func(write int64) {
		blkioWbps += write
	})

	var blkioRbpsOld, blkioRiopsOld, blkioWbpsOld, blkioWiopsOld int64
	sumIoServiceRecursive(lastContainerstats, func(read int64) {
		blkioRiopsOld += read
	}, func(write int64) {
		blkioWiopsOld += write
	})
	sumIoServiceBytesRecursive(lastContainerstats, func(read int64) {
		blkioRbpsOld += read
	}, func(write int64) {
		blkioWbpsOld += write
	})
	return calcPs(blkioRbps, blkioRbpsOld, timediff),
		calcPs(blkioRiops, blkioRiopsOld, timediff),
		calcPs(blkioWbps, blkioWbpsOld, timediff),
		calcPs(blkioWiops, blkioWiopsOld, timediff)
}

func calcNetUsage(containerstats *ContainerStat, lastContainerstats *ContainerStat) (float32, float32, float32, float32) {
	timediff := int64(containerstats.Read.Sub(lastContainerstats.Read).Nanoseconds() / 1000000)

	//netRbps, netRiops, netWbps, netWiops
	return calcPs(containerstats.NetworkStats.RxBytes, lastContainerstats.NetworkStats.RxBytes, timediff),
		calcPs(containerstats.NetworkStats.RxPackets, lastContainerstats.NetworkStats.RxPackets, timediff),
		calcPs(containerstats.NetworkStats.TxBytes, lastContainerstats.NetworkStats.TxBytes, timediff),
		calcPs(containerstats.NetworkStats.TxPackets, lastContainerstats.NetworkStats.TxPackets, timediff)
}

func populateFileKeyValue(prefix string, filename string, callback func(key string, v []int64)) (reterr error) {
	calculated_path := filepath.Join(prefix, filename)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 1 {
			var vals []int64
			for _, word := range words[1:] {
				vals = append(vals, stringutil.ToInt64(word))
			}
			callback(words[0], vals)
		}
	}

	return
}

func populateCgroupKeyValue(prefix string, device string, cgroup_realpath string, filename string, callback func(key string, v int64)) (reterr error) {
	// fmt.Println("populateCgroupKeyValue step -1 ", prefix, device, cgroupsPath, filename)
	// /rootfs/sys/fs/cgroup/cpu/kubepods/besteffort/podb889ec29-d166-4ba6-b98c-bc56d99f2d69/crio-a695a8eef261faf66a26a766883e2751f9e96b79b252cd9451e8b037c3024465
	// fmt.Println("populateCgroupKeyValue step -3 ",cgroup_realpath)
	calculated_path := filepath.Join(prefix, cgroupPrefix, device, cgroup_realpath, filename)
	// fmt.Println("populateCgroupKeyValue step -4 ", calculated_path)
	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
		return
	}
	// fmt.Println("populateCgroupKeyValue step -5")
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		// fmt.Println("populateCgroupKeyValue step -6 ", line)
		words := strings.Fields(line)
		// fmt.Println("populateCgroupKeyValue step -6.1 ", words)
		switch len(words) {
		case 1:
			callback("", stringutil.ToInt64(words[0]))
		case 2:
			callback(words[0], stringutil.ToInt64(words[1]))
		default:
			// fmt.Println("invalid cgroup file:", calculated_path, " content:", line)
		}
	}
	// fmt.Println("populateCgroupKeyValue step -7")
	return
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func populateCgroup2KeyValue(prefix string, device string, cgroup_realpath string, filename string, callback func(key string, v int64)) (reterr error) {
	// fmt.Println("populateCgroupKeyValue step -1 ", prefix, device, cgroupsPath, filename)
	// /rootfs/sys/fs/cgroup/cpu/kubepods/besteffort/podb889ec29-d166-4ba6-b98c-bc56d99f2d69/crio-a695a8eef261faf66a26a766883e2751f9e96b79b252cd9451e8b037c3024465
	// fmt.Println("populateCgroupKeyValue step -3 ",cgroup_realpath)
	for _, cgroupslice := range []string{"ecstasks.slice", "user.slice", "system.slice"} {
		calculated_path := filepath.Join(prefix, cgroupPrefix, cgroupslice, device, cgroup_realpath, filename)
		if !fileExists(calculated_path) {
			continue
		}
		// fmt.Println("populateCgroupKeyValue step -4 ", calculated_path)
		f, err := os.Open(calculated_path)
		if err != nil {
			// fmt.Println(err)
			reterr = err
			return
		}
		// fmt.Println("populateCgroupKeyValue step -5")
		defer f.Close()
		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			line := scanner.Text()
			// fmt.Println("populateCgroupKeyValue step -6 ", line)
			words := strings.Fields(line)
			// fmt.Println("populateCgroupKeyValue step -6.1 ", words)
			switch len(words) {
			case 1:
				callback("", stringutil.ToInt64(words[0]))
			case 2:
				callback(words[0], stringutil.ToInt64(words[1]))
			default:
				// fmt.Println("invalid cgroup file:", calculated_path, " content:", line)
			}
		}
	}
	// fmt.Println("populateCgroupKeyValue step -7")
	return
}

func populateCgroupValues(prefix string, device string, cgroup_realpath string, filename string, callback func(tokens []string)) (reterr error) {
	calculated_path := filepath.Join(prefix, cgroupPrefix, device, cgroup_realpath, filename)
	//fmt.Println("populateCgroupValues calculated_path:",calculated_path)
	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
		return
	}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 0 {
			callback(words)
		}
	}
	return
}

func populateFileValues(prefix string, filename string, callback func(tokens []string)) (reterr error) {
	calculated_path := filepath.Join(prefix, filename)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 0 {
			callback(words)
		}
	}

	return
}

func GetContainerStats(prefix string, containerId string, name string,
	cgroupParent string, restartCount int, pid int) (*ContainerStat, error) {
	if config.GetConfig().DEBUG {
		log.Println("")
		log.Println("GetContainerStats cgroupMode: ", cgroupMode)
		log.Println("GetContainerStats prefix: ", prefix)
		log.Println("GetContainerStats containerId: ", containerId)
		log.Println("GetContainerStats name: ", name)
		log.Println("GetContainerStats cgroupParent: ", cgroupParent)
		log.Println("GetContainerStats restartCount: ", restartCount)
		log.Println("GetContainerStats pid: ", pid)
	}

	switch cgroupMode {
	case CGROUP_LEGACY_HYBRID:
		return getContainerStats(prefix, containerId, name,
			cgroupParent, restartCount, pid)
	case CGROUP_UNIFIED:
		var memoryLimit int64 = 0
		return getContainerStatsCgroupV2(prefix, containerId, name, cgroupParent,
			restartCount, pid, memoryLimit)
	}

	return nil, fmt.Errorf("cgroup mode error %s ", cgroupMode)
}

func getContainerStats(prefix string, containerId string, name string,
	cgroupParent string, restartCount int, pid int) (*ContainerStat, error) {

	var containerStat ContainerStat
	containerStat.Name = name
	containerStat.ID = containerId
	containerStat.RestartCount = restartCount
	containerStat.Read = time.Now()

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
		return nil, err
	}

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
		return nil, err
	}
	// fmt.Println("GetContainerStats step -2")
	populateCgroupKeyValue(prefix, "cpu", cgroupParent, "cpuacct.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "user" {
			containerStat.CPUStats.CPUUsage.UsageInUsermode = v
		} else if key == "system" {
			containerStat.CPUStats.CPUUsage.UsageInKernelmode = v
		}
	})
	populateCgroupKeyValue(prefix, "cpuacct", cgroupParent, "cpuacct.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "user" {
			containerStat.CPUStats.CPUUsage.UsageInUsermode = v
		} else if key == "system" {
			containerStat.CPUStats.CPUUsage.UsageInKernelmode = v
		}
	})

	populateCgroupKeyValue(prefix, "cpu", cgroupParent, "cpuacct.shares", func(key string, v int64) {
		containerStat.CPUStats.OnlineCpus = int(v)
	})
	populateCgroupKeyValue(prefix, "cpu", cgroupParent, "cpu.shares", func(key string, v int64) {
		containerStat.CPUStats.OnlineCpus = int(v)
	})

	containerStat.CPUStats.CPUUsage.TotalUsage = containerStat.CPUStats.CPUUsage.UsageInUsermode + containerStat.CPUStats.CPUUsage.UsageInKernelmode

	// fmt.Println("GetContainerStats step -3")
	populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.max_usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.MaxUsage = v
	})
	if err != nil {
		return nil, err
	}

	err = populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.Usage = v
	})
	if err != nil {
		return nil, err
	}

	err = populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.failcnt", func(key string, v int64) {
		containerStat.MemoryStats.FailCnt = int(v)
	})
	if err != nil {
		return nil, err
	}

	//containerStat.MemoryStats.Limit = memoryLimit

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
			return nil, err
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
		return nil, err
	}
	callbackIoServiceBytesRecursive := func(words []string) {
		// fmt.Println("callbackIoServiceBytesRecursive ", words)
		if len(words) == 3 {
			strmajor, strminor := stringutil.Split2(words[0], ":")
			major := int(stringutil.ToInt64(strmajor))
			minor := int(stringutil.ToInt64(strminor))
			op := words[1]
			v := stringutil.ToInt64(words[2])

			ioServiceBytesRecursive := BlkDeviceValue{major, minor, op, v}
			containerStat.BlkioStats.IoServiceBytesRecursive = append(containerStat.BlkioStats.IoServiceBytesRecursive, ioServiceBytesRecursive)
		}
	}
	// fmt.Println("GetContainerStats step -4")
	populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.io_service_bytes_recursive", callbackIoServiceBytesRecursive)
	populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.throttle.io_service_bytes", callbackIoServiceBytesRecursive)
	callbackIoServicedRecursive := func(words []string) {
		// fmt.Println("callbackIoServicedRecursive ", words)
		if len(words) == 3 {
			strmajor, strminor := stringutil.Split2(words[0], ":")
			major := int(stringutil.ToInt64(strmajor))
			minor := int(stringutil.ToInt64(strminor))
			op := words[1]
			v := stringutil.ToInt64(words[2])

			ioServicedRecursive := BlkDeviceValue{major, minor, op, v}
			containerStat.BlkioStats.IoServicedRecursive = append(containerStat.BlkioStats.IoServicedRecursive, ioServicedRecursive)
		}
	}
	// fmt.Println("GetContainerStats step -5")
	populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.io_serviced_recursive", callbackIoServicedRecursive)
	populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.throttle.io_serviced", callbackIoServicedRecursive)
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
	if err != nil {
		return nil, err
	}

	return &containerStat, nil
}

const (
	RUNNING    = 'r'
	PAUSED     = 'p'
	RESTARTING = 'e'
	OOMKILLED  = 'o'
	DEAD       = 'd'
	WAITING    = 'w'
)

func parseContainerId(containerId string) string {
	idTokens := stringutil.Split(containerId, "://")
	if len(idTokens) > 1 {
		containerId := idTokens[1]
		return containerId
	}
	return containerId
}

func findContainers() ([]*FGContainerInfo, error) {
	conf := config.GetConfig()
	var containers []*FGContainerInfo
	containerLookup := map[string]*FGContainerInfo{}
	containerLookupEx := map[string]*FGContainerInfo{}

	cpuCoreCount := runtime.NumCPU()

	callback := func(c docker_types.ContainerJSON) {

		container := FGContainerInfo{containerId: parseContainerId(c.ID),
			name: c.Name, restartCount: int32(c.RestartCount),
			cgroupParent: c.HostConfig.Resources.CgroupParent}
		containers = append(containers, &container)
		containerLookup[c.Name] = &container
		containerId := parseContainerId(c.ID)
		// fmt.Println("findContainers containerId:", c.ContainerID, idTokens)
		if len(containerId) > 0 {
			containerLookupEx[containerId] = &container
		}

		container.status = c.State.Status
		if c.State.Dead {
			container.state = DEAD
		} else if c.State.Running {
			container.state = RUNNING
		} else if c.State.Paused {
			container.state = PAUSED
		} else if c.State.Restarting {
			container.state = RESTARTING
		} else if c.State.OOMKilled {
			container.state = OOMKILLED
		}

		container.onodeName = conf.ONODE
		container.onode = conf.OID

		container.imageId = c.Image

		fullCpuMillis := int64(cpuCoreCount) * int64(1000)

		memoryLimit := c.HostConfig.Resources.Memory
		cpuLimit := fullCpuMillis
		cpuQuotaPercent := float32(100)

		if c.HostConfig.CPUQuota > 0 && c.HostConfig.CPUPeriod > 0 {
			cpuQuotaPercent *= float32(c.HostConfig.CPUQuota) / float32(c.HostConfig.CPUPeriod) / float32(cpuCoreCount)
			cpuLimit = int64(float32(c.HostConfig.CPUQuota) / float32(c.HostConfig.CPUPeriod) * float32(1000))
		} else if c.HostConfig.CPUPeriod == 0 && c.HostConfig.CPUQuota > 0 && c.HostConfig.CPUQuota < fullCpuMillis {
			cpuQuotaPercent *= float32(c.HostConfig.CPUQuota) / float32(fullCpuMillis)
			cpuLimit = int64(c.HostConfig.CPUQuota)
		}

		container.cpuLimit = int32(cpuLimit)
		container.memoryLimit = int32(memoryLimit)
		container.command = strings.Join(c.ExecIDs, " ")
		container.image = c.Image
		container.pid = int32(c.State.Pid)
	}

	err := findAllContainersOnNode(callback)

	return containers, err
}

// 2021/11/04 07:35:14 container task err: open proc/22/net/dev: no such file or directory
// 2021/11/04 07:35:14 container task err: open /proc/9/cwd/proc/stat: no such file or directory
func parseCpuPer(coreCount int, totalCpu float32, cpuLimit int32) float32 {
	if cpuLimit > 0 {
		return float32(100) * float32(totalCpu) / (float32(100) * float32(cpuLimit) / (float32(coreCount) * float32(1000)))
	} else {
		return totalCpu
	}
}

func toBytes(unit string) int64 {
	var ret int64
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

func parseMemoryStat(callback func(key string, val int64)) error {
	filepath := "/proc/meminfo"
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		intValue, _ := strconv.ParseInt(words[1], 10, 64)
		value := int64(intValue)

		switch strings.ToLower(words[0]) {
		case "memtotal:":
			callback("memtotal", value*toBytes(words[2]))
		default:
		}
	}

	return nil
}

func createPack() *pack.TagCountPack {
	conf := config.GetConfig()
	now := dateutil.Now()

	p := pack.NewTagCountPack()
	p.Pcode = conf.PCODE
	p.Oid = conf.OID
	p.Time = now
	p.Category = "ecs_task"
	p.Tags.PutString(METERINGTYPE, METERINGTYPE_ECS)

	return p
}

func send(p pack.Pack) bool {

	return session.Send(p)
}

func sendHide(p pack.Pack) bool {

	return session.SendHide(p)
}

func sendEncrypted(p pack.Pack) bool {

	return session.SendEncrypted(p)
}

func findAllContainersOnNode(onContainerDetected func(docker_types.ContainerJSON)) error {
	cli, err := whatap_docker.GetDockerClient()
	if err != nil {
		return err
	}
	containers, err := cli.ContainerList(context.Background(), docker_types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, c := range containers {

		inspectContainer, err := cli.ContainerInspect(context.Background(), c.ID)

		if err != nil {
			return err
		}
		onContainerDetected(inspectContainer)
		text.SendText(pack.CONTAINER, inspectContainer.ID)
	}
	return nil
}

func sendTagFieldPack(category string, tags map[string]interface{}, fields map[string]interface{}, now int64) {
	conf := config.GetConfig()
	p := pack.NewTagCountPack()
	p.Pcode = conf.PCODE
	p.Oid = conf.OID
	p.Time = now
	p.Category = category

	populateAll(p.Tags, tags)
	populateAll(p.Data, fields)

	send(p)
}
func populateAll(m *value.MapValue, s map[string]interface{}) {
	for k, v := range s {
		populate(m, k, v)
	}
}

func populate(m *value.MapValue, name string, v interface{}) {
	switch v.(type) {
	case value.Value:
		m.Put(name, v.(value.Value))
	case int:
		m.Put(name, value.NewDecimalValue(int64(v.(int))))
	case int16:
		m.Put(name, value.NewDecimalValue(int64(v.(int16))))
	case int32:
		m.Put(name, value.NewDecimalValue(int64(v.(int32))))
	case int64:
		m.Put(name, value.NewDecimalValue(v.(int64)))
	case uint:
		m.Put(name, value.NewDecimalValue(int64(v.(uint))))
	case uint32:
		m.Put(name, value.NewDecimalValue(int64(v.(uint32))))
	case uint64:
		m.Put(name, value.NewDecimalValue(int64(v.(uint64))))
	case float32:
		m.Put(name, value.NewFloatValue(v.(float32)))
	case float64:
		m.Put(name, value.NewDoubleValue(v.(float64)))
	case string:
		m.Put(name, value.NewTextValue(v.(string)))
	default:

	}
}

// GetCgroupMountsPath는 cgroup이 마운트된 경로를 반환하는 함수입니다.
func GetCgroupMountsPath() (map[string]string, error) {
	// /proc/mounts 파일을 엽니다.
	file, err := os.Open(filepath.Join(NodeVolPrefix, "/proc/mounts"))
	if err != nil {
		return nil, fmt.Errorf("error opening /proc/mounts: %v", err)
	}
	defer file.Close()

	// cgroup 마운트 경로를 저장할 맵을 만듭니다.
	cgroupMounts := make(map[string]string)

	// 파일을 한 줄씩 읽습니다.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 각 줄에서 cgroup 마운트 정보만 찾습니다.
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			mountType := fields[2]
			mountPath := fields[1]
			if strings.HasPrefix(mountPath, filepath.Join(NodeVolPrefix, "/var/lib/docker")) {
				continue
			}
			if strings.HasPrefix(mountPath, NodeVolPrefix) {
				switch mountType {
				case "cgroup":
					cgroupMounts[mountType] = strings.TrimPrefix(filepath.Dir(mountPath), NodeVolPrefix)
					break
				case "cgroup2":
					cgroupMounts[mountType] = strings.TrimPrefix(mountPath, NodeVolPrefix)
					break
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading /proc/mounts: %v", err)
	}

	return cgroupMounts, nil
}

func getContainerStatsCgroupV2(prefix string, containerId string, name string, cgroupParent string,
	restartCount int, pid int, memoryLimit int64) (*ContainerStat, error) {

	var containerStat ContainerStat
	containerStat.Name = name
	containerStat.ID = containerId
	containerStat.RestartCount = restartCount
	containerStat.Read = time.Now()

	populateFileKeyValue(prefix, "/proc/stat", func(key string, v []int64) {
		if key == "cpu" {
			for i, ev := range v {
				if i < 8 {
					containerStat.CPUStats.SystemCPUUsage += ev
				}
			}
		}
	})

	populateCgroup2KeyValue(prefix, "", cgroupParent, "cpu.stat", func(key string, v int64) {
		if key == "nr_periods" {
			containerStat.CPUStats.ThrottlingData.Periods = v
		} else if key == "throttled_usec" {
			containerStat.CPUStats.ThrottlingData.ThrottledTime = v
		} else if key == "nr_throttled" {
			containerStat.CPUStats.ThrottlingData.ThrottledPeriods = v
		} else if key == "usage_usec" {
			containerStat.CPUStats.CPUUsage.UsageInUsermode = v / 10000
		} else if key == "user_usec" {
			containerStat.CPUStats.CPUUsage.UsageInKernelmode = v / 10000
		}

	})

	containerStat.CPUStats.CPUUsage.TotalUsage = containerStat.CPUStats.CPUUsage.UsageInUsermode + containerStat.CPUStats.CPUUsage.UsageInKernelmode

	populateCgroup2KeyValue(prefix, "", cgroupParent, "memory.current", func(key string, v int64) {
		containerStat.MemoryStats.Usage = v
	})

	populateCgroup2KeyValue(prefix, "", cgroupParent, "memory.events", func(key string, v int64) {
		if key == "oom" {
			containerStat.MemoryStats.FailCnt = int(v)
		}
	})

	containerStat.MemoryStats.Limit = memoryLimit

	if containerStat.MemoryStats.Limit == 0 {
		populateFileValues(prefix, "/proc/meminfo", func(tokens []string) {
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

	}

	populateCgroup2KeyValue(prefix, "", cgroupParent, "memory.stat", func(key string, v int64) {
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

	callbackIoServiceBytesRecursive := func(op string, v int64) {
		major := 1
		minor := 1

		bdv := BlkDeviceValue{major, minor, op, v}
		containerStat.BlkioStats.IoServiceBytesRecursive = append(containerStat.BlkioStats.IoServiceBytesRecursive, bdv)
	}

	callbackIoServicedRecursive := func(op string, v int64) {
		major := 1
		minor := 1

		bdv := BlkDeviceValue{major, minor, op, v}
		containerStat.BlkioStats.IoServicedRecursive = append(containerStat.BlkioStats.IoServicedRecursive, bdv)
	}

	populateCgroup2KeyValue(prefix, "", cgroupParent, "io.stat", func(key string, v int64) {
		switch key {
		case "rbytes":
			callbackIoServiceBytesRecursive("Read", v)
		case "wbytes":
			callbackIoServiceBytesRecursive("Write", v)
		case "rios":
			callbackIoServicedRecursive("Read", v)
		case "wios":
			callbackIoServicedRecursive("Write", v)
		default:
		}
	})

	populateFileValues(prefix, filepath.Join("proc", fmt.Sprint(pid), "net/dev"), func(tokens []string) {
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

	return &containerStat, nil
}
