package counter

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	gonet "net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/pack"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/iputil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/stringutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"whatap.io/k8s/sidecar/config"
	"whatap.io/k8s/sidecar/kube"
	"whatap.io/k8s/sidecar/session"
)

var (
	selfContainerId string
)

func getSelfContainerId() string {
	if len(selfContainerId) < 1 {
		selfpid := os.Getpid()
		selfContainerId, _ = getContainerIdByPid(selfpid)
	}

	return selfContainerId
}

var (
	gcpu float32
	gmem float32
)

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

func getContainerIdByPid(pid int) (string, error) {
	cgroupPath := filepath.Join("/proc", fmt.Sprint(pid), "cgroup")
	if _, err := os.Stat(cgroupPath); err != nil {
		return "", err
	}

	f, err := os.Open(cgroupPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := stringutil.Split(line, "/")
		if len(words) > 0 {
			containerid := words[len(words)-1]
			return containerid, nil
		}
	}

	return "", fmt.Errorf("container id not found")
}

func findMainContainerRootPrefix() (string, int, string, error) {
	pid := os.Getpid()
	selfContainerId, err := getContainerIdByPid(pid)
	if err != nil {
		return "", 0, "", err
	}
	fmt.Println("selfContainerId:", selfContainerId)

	searchDir := "/proc"
	filesInfo, err := ioutil.ReadDir(searchDir)
	// fmt.Println("step -1 ", err)
	for i := 0; i < len(filesInfo) && err == nil; i++ {
		fileInfo := filesInfo[i]
		// fmt.Println("step -2 ", fileInfo)
		if !fileInfo.IsDir() {
			continue
		}

		pid := fileInfo.Name()
		// fmt.Println("step -3 ", pid)
		ipid, err := strconv.Atoi(pid)
		if err == nil {
			if ipid == 1 {
				continue
			}
			cmdline_bytes, _ := ioutil.ReadFile(strings.Join([]string{searchDir, pid, "cmdline"}, "/"))
			status_bytes, err := ioutil.ReadFile(strings.Join([]string{searchDir, pid, "status"}, "/"))
			if err == nil {
				status_content := string(status_bytes)
				for _, line := range strings.Split(status_content, "\n") {

					if strings.HasPrefix(line, "PPid") {
						ppid, err := strconv.Atoi(strings.Split(line, "\t")[1])
						// fmt.Println("step -4 ", pid, ppid, err)
						if err == nil && ppid == 0 {

							containerId, err := getContainerIdByPid(ipid)
							// fmt.Println("step -5 ", containerId)
							cmdline := string(cmdline_bytes)
							fmt.Println("step -5 ", cmdline, selfContainerId, containerId)
							if selfContainerId == containerId && !strings.Contains(cmdline, "/bin/bash") && err == nil {
								return containerId, ipid, filepath.Join("proc", pid, "cwd"), nil
							}
						}
					}
				}

			}
		}
	}

	return "", 0, "", fmt.Errorf("main container not found")
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

func populateCgroupKeyValue(prefix string, device string, filename string, callback func(key string, v int64)) (reterr error) {
	// fmt.Println("populateCgroupKeyValue step -1 ", prefix, device, cgroupsPath, filename)
	// /rootfs/sys/fs/cgroup/cpu/kubepods/besteffort/podb889ec29-d166-4ba6-b98c-bc56d99f2d69/crio-a695a8eef261faf66a26a766883e2751f9e96b79b252cd9451e8b037c3024465
	// fmt.Println("populateCgroupKeyValue step -3 ",cgroup_realpath)
	calculated_path := filepath.Join(prefix, "/sys/fs/cgroup", device, filename)
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

func populateCgroupValues(prefix string, device string, filename string, callback func(tokens []string)) (reterr error) {
	calculated_path := filepath.Join(prefix, "/sys/fs/cgroup", device, filename)
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

func getContainerStats(prefix string, containerId string, name string,
	restartCount int, pid int) (*ContainerStat, error) {

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
	// fmt.Println("GetContainerStats step -1")
	populateCgroupKeyValue(prefix, "cpu", "cpu.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "nr_periods" {
			containerStat.CPUStats.ThrottlingData.Periods = v
		} else if key == "throttled_time" {
			containerStat.CPUStats.ThrottlingData.ThrottledTime = v
		} else if key == "nr_throttled" {
			containerStat.CPUStats.ThrottlingData.ThrottledPeriods = v
		}
	})

	// fmt.Println("GetContainerStats step -2")
	populateCgroupKeyValue(prefix, "cpu", "cpuacct.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "user" {
			containerStat.CPUStats.CPUUsage.UsageInUsermode = v
		} else if key == "system" {
			containerStat.CPUStats.CPUUsage.UsageInKernelmode = v
		}
	})
	populateCgroupKeyValue(prefix, "cpuacct", "cpuacct.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "user" {
			containerStat.CPUStats.CPUUsage.UsageInUsermode = v
		} else if key == "system" {
			containerStat.CPUStats.CPUUsage.UsageInKernelmode = v
		}
	})

	populateCgroupKeyValue(prefix, "cpu", "cpuacct.shares", func(key string, v int64) {
		containerStat.CPUStats.OnlineCpus = int(v)
	})

	populateCgroupKeyValue(prefix, "cpu", "cpu.shares", func(key string, v int64) {
		containerStat.CPUStats.OnlineCpus = int(v)
	})

	containerStat.CPUStats.CPUUsage.TotalUsage = containerStat.CPUStats.CPUUsage.UsageInUsermode + containerStat.CPUStats.CPUUsage.UsageInKernelmode

	// fmt.Println("GetContainerStats step -3")
	err = populateCgroupKeyValue(prefix, "memory", "memory.max_usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.MaxUsage = v
	})
	if err != nil {
		return nil, err
	}

	err = populateCgroupKeyValue(prefix, "memory", "memory.usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.Usage = v
	})
	if err != nil {
		return nil, err
	}

	err = populateCgroupKeyValue(prefix, "memory", "memory.failcnt", func(key string, v int64) {
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

	err = populateCgroupKeyValue(prefix, "memory", "memory.stat", func(key string, v int64) {
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
	populateCgroupValues(prefix, "blkio", "blkio.io_service_bytes_recursive", callbackIoServiceBytesRecursive)

	populateCgroupValues(prefix, "blkio", "blkio.throttle.io_service_bytes", callbackIoServiceBytesRecursive)

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
	populateCgroupValues(prefix, "blkio", "blkio.io_serviced_recursive", callbackIoServicedRecursive)

	populateCgroupValues(prefix, "blkio", "blkio.throttle.io_serviced", callbackIoServicedRecursive)

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
	var containers []*FGContainerInfo
	containerLookup := map[string]*FGContainerInfo{}
	containerLookupEx := map[string]*FGContainerInfo{}
	var pod, namespace string

	podCallback := func(k string, v string) {
		switch k {
		case "pod":
			pod = v
		case "namespace":
			namespace = v
			setNodeNamespace(v)
		}
	}

	statusCallback := func(c v1.ContainerStatus) {

		container := FGContainerInfo{containerId: parseContainerId(c.ContainerID),
			name: c.Name, restartCount: c.RestartCount}
		containers = append(containers, &container)
		containerLookup[c.Name] = &container
		containerId := parseContainerId(c.ContainerID)
		// fmt.Println("findContainers containerId:", c.ContainerID, idTokens)
		if len(containerId) > 0 {
			containerLookupEx[containerId] = &container
		}

		if c.State.Waiting != nil {
			container.state = WAITING
			container.status = c.State.Waiting.Reason
		} else if c.State.Running != nil {
			container.state = RUNNING
			container.status = fmt.Sprint("Up ", time.Since(c.State.Running.StartedAt.Time).String())
			container.created = int32(c.State.Running.StartedAt.Time.UnixNano() / 1000000)
		} else if c.State.Terminated != nil {
			container.state = DEAD
			container.status = c.State.Terminated.Reason
		}
		if c.Ready {
			container.ready = 1
		}
		container.pod = pod
		container.namespace = namespace
		container.onodeName = conf.ONODE
		container.onode = conf.OID

		container.imageId = c.ImageID
	}

	specCallback := func(c v1.Container) {
		var cpuLimit, memoryLimit, cpuRequest, memoryRequest int64

		if c.Resources.Limits != nil {
			if c.Resources.Limits.Cpu() != nil {
				cpuLimit = c.Resources.Limits.Cpu().MilliValue()
				//fmt.Println("specCallback cpu limit ", c.Resources.Limits.Cpu().Value(), c.Resources.Limits.Cpu().MilliValue())
			}
			if c.Resources.Limits.Memory() != nil {
				memoryLimit = c.Resources.Limits.Memory().Value()
				// fmt.Println("specCallback memory limit ", c.Resources.Limits.Memory().Value(), c.Resources.Limits.Memory().MilliValue())
			}
		}

		if c.Resources.Requests != nil {
			if c.Resources.Requests.Cpu() != nil {
				cpuRequest = c.Resources.Requests.Cpu().MilliValue()
				// fmt.Println("specCallback cpu req ", c.Resources.Requests.Cpu().Value(), c.Resources.Requests.Cpu().MilliValue())
			}
			if c.Resources.Requests.Memory() != nil {
				memoryRequest = c.Resources.Requests.Memory().Value()
				// fmt.Println("specCallback memory req ", c.Resources.Requests.Memory().Value(), c.Resources.Requests.Memory().MilliValue())
			}
		}

		container := containerLookup[c.Name]
		container.cpuLimit = int32(cpuLimit)
		container.memoryLimit = int32(memoryLimit)
		container.cpuRequest = int32(cpuRequest)
		container.memoryRequest = int32(memoryRequest)
		container.command = strings.Join(c.Command, " ")
		container.image = c.Image
	}

	selfpid := os.Getpid()
	selfContainerId, _ := getContainerIdByPid(selfpid)

	err := findAllContainersOnNode(podCallback, statusCallback, specCallback)
	if err != nil {
		return nil, err
	}

	for _, cinfo := range containerLookup {
		cinfo.pod = pod
		cinfo.namespace = namespace
	}

	searchDir := "/proc"
	filesInfo, err := ioutil.ReadDir(searchDir)
	// fmt.Println("step -1 ", err)
	// log.Println("findContainers step -1")
	for i := 0; i < len(filesInfo) && err == nil; i++ {
		fileInfo := filesInfo[i]
		// log.Println("findContainers step -2", fileInfo)
		// fmt.Println("step -2 ", fileInfo)
		if !fileInfo.IsDir() {
			continue
		}
		// log.Println("findContainers step -3")
		pid := fileInfo.Name()
		// fmt.Println("step -3 ", pid)
		// log.Println("findContainers step -4 pid:", pid)
		ipid, err := strconv.Atoi(pid)
		if err == nil {
			if ipid == 1 {
				continue
			}
			// cmdline_bytes, _ := ioutil.ReadFile(strings.Join([]string{searchDir, pid, "cmdline"}, "/"))
			status_bytes, err := ioutil.ReadFile(strings.Join([]string{searchDir, pid, "status"}, "/"))
			// log.Println("findContainers step -5 err:", err)

			if err == nil {
				status_content := string(status_bytes)
				// log.Println("findContainers step -5 status_bytes:", status_content)
				for _, line := range strings.Split(status_content, "\n") {

					if strings.HasPrefix(line, "PPid") {
						ppid, err := strconv.Atoi(strings.Split(line, "\t")[1])
						// fmt.Println("step -4 ", pid, ppid, err)
						// log.Println("findContainers step -5.1 err:", err, " ppid:", ppid)
						if err == nil && ppid == 0 {

							containerId, err := getContainerIdByPid(ipid)
							if err != nil {
								return nil, err
							}
							// log.Println("findContainers step -5.2 containerId:", containerId)
							if _, ok := containerLookupEx[containerId]; !ok {
								// fmt.Println("findContainers not found: ", containerId, "pid:", ipid)
								continue

							}
							// log.Println("findContainers step -5.3 ipid:", ipid)
							containerLookupEx[containerId].pid = int32(ipid)
							// log.Println("findContainers step -5.4 containerId:", containerId, selfContainerId)
							if containerId != selfContainerId {
								containerLookupEx[containerId].prefix = strings.Join([]string{searchDir, pid, "root"}, "/")
							} else {
								containerLookupEx[containerId].prefix = "/"
							}

						}
					}
				}
				// log.Println("findContainers step -6")
			}
		}
	}

	return containers, nil
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

var (
	conf *config.Config = config.GetConfig()
)

func createPack() *pack.TagCountPack {
	now := dateutil.Now()

	p := pack.NewTagCountPack()
	p.Pcode = conf.PCODE
	p.Oid = conf.OID
	p.Time = now
	p.Category = "container"

	return p
}

func sendOneway(pcode int64, licenseHash64 int64, p pack.Pack) bool {
	session.SendOneway(pcode, licenseHash64, p)
	return false
}

func send(p pack.Pack) bool {

	return session.SendHide(p)
}

func sendHide(p pack.Pack) bool {

	return session.SendHide(p)
}

func sendEncrypted(p pack.Pack) bool {

	return session.SendEncrypted(p)
}

func findAllContainersOnNode(podCallback func(string, string), statusCallback func(v1.ContainerStatus), specCallback func(v1.Container)) error {
	cli, err := kube.GetKubeClient()
	if err != nil {
		return err
	}
	nodename := os.Getenv("NODE_NAME")
	listOptions := metav1.ListOptions{FieldSelector: fmt.Sprint("spec.nodeName=", nodename)}
	pods, err := cli.CoreV1().Pods("").List(context.Background(), listOptions)
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		podCallback("pod", pod.Name)
		podCallback("namespace", pod.Namespace)

		if pod.OwnerReferences != nil {
			for _, r := range pod.OwnerReferences {
				podCallback("replicaSeName", r.Name)
			}
		}
		for _, c := range pod.Status.ContainerStatuses {
			statusCallback(c)
			onContainerDetected(parseContainerId(c.ContainerID), pod.Namespace, pod.Name, c.Name)
		}
		for _, c := range pod.Spec.Containers {
			specCallback(c)
		}
	}
	return nil
}
