package containerd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	whatap_model "github.com/whatap/kube/node/src/whatap/lang/model"
	"github.com/whatap/kube/node/src/whatap/util/fileutil"
	"github.com/whatap/kube/node/src/whatap/util/stringutil"
)

var (
	mu = sync.Mutex{}
)
var containerdClient *containerd.Client
var containerdNamespaces []string

func GetContainerdClient() (*containerd.Client, error) {
	mu.Lock()
	defer mu.Unlock()
	if containerdClient == nil {
		newContainerdClient, err := containerd.New("/run/containerd/containerd.sock")
		if err != nil {
			return nil, err
		}

		containerdClient = newContainerdClient

		if nss, err := containerdClient.NamespaceService().List(context.Background()); err == nil {
			containerdNamespaces = nss

		}
	}
	return containerdClient, nil
}
func LoadContainerD(containerid string) (containerd.Container, context.Context, error) {
	cli, err := GetContainerdClient()
	if err != nil {
		return nil, nil, err
	}

	for _, containerdNamespace := range containerdNamespaces {
		ctx := namespaces.WithNamespace(context.Background(), containerdNamespace)

		resp, err := cli.LoadContainer(ctx, containerid)
		if err == nil {
			return resp, ctx, err
		}
	}
	return nil, nil, fmt.Errorf("container ", containerid, " not found")
}

func parseRealPath(cgroupsPath string) (ret string) {
	if strings.Contains(cgroupsPath, "slice") {
		cgroup_realpath := "kubepods.slice"
		tokens := stringutil.Tokenizer(cgroupsPath, ":")
		if len(tokens) < 3 {
			return
		}
		// fmt.Println("populateCgroupKeyValue step -2")
		if strings.Contains(cgroupsPath, "besteffort") {
			cgroup_realpath = filepath.Join(cgroup_realpath, "kubepods-besteffort.slice", tokens[0], fmt.Sprint(tokens[1], "-", tokens[2], ".scope"))
		} else if strings.Contains(cgroupsPath, "burstable") {
			cgroup_realpath = filepath.Join(cgroup_realpath, "kubepods-burstable.slice", tokens[0], fmt.Sprint(tokens[1], "-", tokens[2], ".scope"))
		} else {
			cgroup_realpath = filepath.Join(cgroup_realpath, tokens[0], fmt.Sprint(tokens[1], "-", tokens[2], ".scope"))
		}
		ret = cgroup_realpath
	} else {
		ret = cgroupsPath
	}
	return
}

func parseRealPathEx(cgroupsPath string) (ret string) {
	if strings.Contains(cgroupsPath, "slice") {
		cgroup_realpath := "kubepods.slice"
		tokens := stringutil.Tokenizer(cgroupsPath, ":")
		if len(tokens) < 3 {
			return
		}
		// fmt.Println("populateCgroupKeyValue step -2")
		if strings.Contains(cgroupsPath, "besteffort") {
			cgroup_realpath = filepath.Join(cgroup_realpath, "kubepods-besteffort.slice", tokens[0])
		} else if strings.Contains(cgroupsPath, "burstable") {
			cgroup_realpath = filepath.Join(cgroup_realpath, "kubepods-burstable.slice", tokens[0])
		} else {
			cgroup_realpath = filepath.Join(cgroup_realpath, tokens[0])
		}
		ret = cgroup_realpath
	} else {
		ret = cgroupsPath
	}
	return
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

func populateCgroupKeyValue(prefix string, device string, cgroupsPath string, filename string, callback func(key string, v int64)) (reterr error) {
	// fmt.Println("populateCgroupKeyValue step -1 ", prefix, device, cgroupsPath, filename)
	// /rootfs/sys/fs/cgroup/cpu/kubepods/besteffort/podb889ec29-d166-4ba6-b98c-bc56d99f2d69/crio-a695a8eef261faf66a26a766883e2751f9e96b79b252cd9451e8b037c3024465
	// fmt.Println("populateCgroupKeyValue step -3 ",cgroup_realpath)

	var calculated_path string
	cgroup_realpath := parseRealPath(cgroupsPath)

	if !fileutil.IsExists(filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)) {
		cgroup_realpath = parseRealPathEx(cgroupsPath)
	}

	calculated_path = filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)

	// fmt.Println("populateCgroupKeyValue cgroup: ", cgroupsPath)
	// fmt.Println("populateCgroupKeyValue realPath: ", cgroup_realpath)
	// fmt.Println("populateCgroupKeyValue calculated_path: ", calculated_path)

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

func populateCgroupValues(prefix string, device string, cgroupsPath string, filename string, callback func(tokens []string)) (reterr error) {
	cgroup_realpath := parseRealPath(cgroupsPath)

	if !fileutil.IsExists(filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)) {
		cgroup_realpath = parseRealPathEx(cgroupsPath)
	}

	calculated_path := filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)
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

// func GetContainerStats(prefix string, containerId string) (string, error) {
// 	hconfig, err := getDockerHostConfig(prefix, containerId)
// 	if err != nil {
// 		return "", err
// 	}

// 	dconfig, err := getDockerConfigV2(prefix, containerId)
// 	if err != nil {
// 		return "", err
// 	}

// 	return GetContainerStatsEx(prefix, containerId, dconfig.Name, hconfig.CgroupParent)
// }

type hcontparam func(name string, cgroupParent string,
	restartCount int, pid int, memoryLimit int64) error

func getParams(regEx, src string) (paramsMap map[string]string) {
	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(src)
	paramsMap = make(map[string]string)

	for i, name := range compRegEx.SubexpNames() {
		if len(match) > 0 && i > 0 {
			paramsMap[name] = match[i]
		}
	}
	return
}

func getRealCgroupParent(cgroupParent string, containerId string) (ret string) {
	ret = cgroupParent
	m := getParams("(?P<prefix1>[a-zA-Z]+)\\-(?P<prefix2>[a-zA-Z]+)\\-(?P<prefix3>[a-zA-Z0-9_]+)\\.slice", cgroupParent)
	if len(m) == 3 {
		ret = filepath.Join("kubepods.slice", fmt.Sprint(m["prefix1"], "-", m["prefix2"], ".slice"), cgroupParent)
		ret = fmt.Sprint(ret, "/containerd-", containerId, ".scope")
	} else {
		ret = fmt.Sprint(ret, "/", containerId)
	}

	return
}

func GetContainerStatsEx(prefix string, containerId string, name string, cgroupParent string,
	restartCount int, pid int, memoryLimit int64) (string, error) {
	// fmt.Println("GetContainerStatsEx", prefix, containerId, name, cgroupParent,
	// 	restartCount, pid, memoryLimit)
	containerStat, err := GetContainerStatsExRaw(prefix, containerId, name, cgroupParent,
		restartCount, pid, memoryLimit)

	if err != nil {
		return "", err
	}

	containerStatJson, err := json.Marshal(containerStat)
	if err != nil {
		return "", err
	}
	return string(containerStatJson), nil

}

func GetContainerStatsExRaw(prefix string, containerId string, name string, cgroupParent string,
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
func InspectContainerdImageForWhatapPath(containerID string) (string, error) {
	container, ctx, err := LoadContainerD(containerID)
	if err != nil {
		return "", fmt.Errorf("error=%w", err)
	}
	spec, err := container.Spec(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get container spec for %s: %w", containerID, err)
	}

	process := spec.Process
	processArgs := process.Args

	for _, processArg := range processArgs {
		if strings.Contains(processArg, "javaagent:") {
			javaAgentPath := strings.SplitN(processArg, "javaagent:", 2)[1]

			// javaagent 경로에서 첫 번째 공백 또는 옵션까지만 추출
			javaAgentPath = strings.SplitN(javaAgentPath, " ", 2)[0]
			return javaAgentPath, nil
		}
	}
	return "", fmt.Errorf("javaAgentPathNotFound-contaier:%s", containerID)
}

func CheckContainerdEnabled() bool {
	fi, err := os.Stat("/run/containerd/containerd.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}
