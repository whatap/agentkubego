package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	whatap_config "github.com/whatap/kube/node/src/whatap/config"
	"github.com/whatap/kube/node/src/whatap/util/logutil"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	whatap_model "github.com/whatap/kube/node/src/whatap/lang/model"
	"github.com/whatap/kube/node/src/whatap/util/stringutil"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var dockerClient *client.Client
var mu = sync.Mutex{}

func GetDockerClient() (*client.Client, error) {
	mu.Lock()
	defer mu.Unlock()
	if dockerClient == nil {
		dockerClientThisTime, err := client.NewClientWithOpts(client.WithVersion("1.40"))

		if err != nil {
			logutil.Errorf("execDebug", "GetDockerClientErr=%v", err)
			return nil, err
		}
		dockerClient = dockerClientThisTime
	}
	return dockerClient, nil
}

func getDockerHostConfig(prefix string, containerId string) (oconfig whatap_model.DockerHostConfig, err error) {
	overlay_config_path := filepath.Join(prefix, "/var/lib/docker/containers", containerId, "hostconfig.json")
	if _, err = os.Stat(overlay_config_path); os.IsNotExist(err) {
		return
	}

	oc_bytes, err := ioutil.ReadFile(overlay_config_path)
	if err != nil {
		return
	}
	err = json.Unmarshal(oc_bytes, &oconfig)

	return

}

func getDockerConfigV2(prefix string, containerId string) (ostate whatap_model.DockerConfigV2, err error) {
	overlay_state_path := filepath.Join(prefix, "/var/lib/docker/containers", containerId, "config.v2.json")

	os_bytes, err := ioutil.ReadFile(overlay_state_path)
	if err != nil {
		return
	}
	err = json.Unmarshal(os_bytes, &ostate)
	if err != nil {
		return
	}

	return

}

func GetContainerInspect(prefix string, containerId string) (string, error) {
	hconfig, err := getDockerHostConfig(prefix, containerId)
	if err != nil {
		return "", err
	}

	dconfig, err := getDockerConfigV2(prefix, containerId)
	if err != nil {
		return "", err
	}

	var cinfo whatap_model.ContainerInfo
	cinfo.ID = containerId
	cinfo.Created = dconfig.Created
	cinfo.Path = dconfig.Path
	cinfo.HostConfig.CPUPeriod = hconfig.CPUPeriod
	cinfo.HostConfig.CPUQuota = hconfig.CPUQuota
	cinfo.HostConfig.CPUShares = hconfig.CPUShares
	cinfo.State.Status = dconfig.ParseState()
	cinfo.State.StartedAt = dconfig.State.StartedAt
	cinfo.State.FinishedAt = dconfig.State.FinishedAt
	cinfo.State.Running = dconfig.State.Running
	cinfo.State.Pid = dconfig.State.Pid
	cinfo.State.Paused = dconfig.State.Paused
	cinfo.State.Restarting = dconfig.State.Restarting
	cinfo.State.OOMKilled = dconfig.State.OOMKilled
	cinfo.State.Dead = dconfig.State.Dead
	cinfo.State.ExitCode = dconfig.State.ExitCode
	cinfo.State.Error = dconfig.State.Error
	cinfo.Config.Env = dconfig.Config.Env

	cinfojson, err := json.Marshal(cinfo)
	if err == nil {
		return string(cinfojson), nil
	}

	return "", err
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

func populateCgroupKeyValue(prefix string, device string, cgroup_realpath string, filename string, callback func(key string, v int64)) (reterr error) {
	// fmt.Println("populateCgroupKeyValue step -1 ", prefix, device, cgroupsPath, filename)
	// /rootfs/sys/fs/cgroup/cpu/kubepods/besteffort/podb889ec29-d166-4ba6-b98c-bc56d99f2d69/crio-a695a8eef261faf66a26a766883e2751f9e96b79b252cd9451e8b037c3024465
	// fmt.Println("populateCgroupKeyValue step -3 ",cgroup_realpath)
	calculated_path := filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)
	// fmt.Println("populateCgroupKeyValue step -4 ", calculated_path)
	f, err := os.Open(calculated_path)
	if err != nil {
		//logutil.Infof("DHPCK", "OPEN %v is failed, ERROR=%v", calculated_path, err)
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

func populateCgroupValues(prefix string, device string, cgroup_realpath string, filename string, callback func(tokens []string)) (reterr error) {
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
		ret = fmt.Sprint(ret, "/docker-", containerId, ".scope")
	} else {
		ret = fmt.Sprint(ret, "/", containerId)
	}

	return
}

func GetContainerParams(prefix string, containerId string, callback hcontparam) (err error) {
	hconfig, err := getDockerHostConfig(prefix, containerId)
	if err == nil {
		dconfig, dconfigerr := getDockerConfigV2(prefix, containerId)
		if dconfigerr == nil {
			if whatap_config.GetConfig().Debug {
				logutil.Infof("GCP", "hconfig.CgroupParent=%v", hconfig.CgroupParent)
			}
			readCgropuParent := getRealCgroupParent(hconfig.CgroupParent, containerId)

			err = callback(dconfig.Name, readCgropuParent, dconfig.RestartCount, dconfig.State.Pid, hconfig.Memory)
			return
		}
	}

	containerJson, err := getContainerInspectDockerAPI(containerId)
	if err == nil {
		readCgropuParent := getRealCgroupParent(containerJson.HostConfig.CgroupParent, containerId)
		err = callback(containerJson.Name, readCgropuParent,
			containerJson.RestartCount, containerJson.State.Pid, containerJson.HostConfig.Memory)
		return
	}
	err = fmt.Errorf("Cannot find container ", containerId)
	return
}

func getContainerInspectDockerAPI(containerId string) (*dockertypes.ContainerJSON, error) {
	cli, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	containerJson, err := cli.ContainerInspect(context.Background(), containerId)

	return &containerJson, err
}

func GetContainerStatsEx(prefix string, containerId string, name string, cgroupParent string,
	restartCount int, pid int, memoryLimit int64) (string, error) {
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
		return "", err
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
		return "", err
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
		return "", err
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
		return "", err
	}

	err = populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.Usage = v
	})
	if err != nil {
		return "", err
	}

	err = populateCgroupKeyValue(prefix, "memory", cgroupParent, "memory.failcnt", func(key string, v int64) {
		containerStat.MemoryStats.FailCnt = int(v)
	})
	if err != nil {
		return "", err
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
			return "", err
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
		return "", err
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
			return "", err
		}
	}
	err = populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.throttle.io_service_bytes", callbackIoServiceBytesRecursive)
	if err != nil {
		return "", err
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
			return "", err
		}
	}
	err = populateCgroupValues(prefix, "blkio", cgroupParent, "blkio.throttle.io_serviced", callbackIoServicedRecursive)
	if err != nil {
		return "", err
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
	if err != nil {
		return "", err
	}
	// fmt.Println("GetContainerStats step -8")
	containerStatJson, err := json.Marshal(containerStat)
	if err == nil {
		return string(containerStatJson), nil
	}
	// fmt.Println("GetContainerStats step -9", err)
	return "", err
}

func CheckDockerEnabled() bool {
	fi, err := os.Stat("/var/run/docker.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}

// InspectDockerImageForWhatapPath는 주어진 Docker 컨테이너 ID로부터 javaagent 경로를 추출
func InspectDockerImageForWhatapPath(containerID string) (string, error) {
	logutil.Debugln("execDebug", "InspectDockerImageForWhatapPath start")
	cli, err := GetDockerClient()
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}

	containerJSON, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		logutil.Debugf("execDebug", "containerJSON ERR=%v", err)
		return "", fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	logutil.Debugf("execDebug", "containerJSON=%v", containerJSON)
	// Cmd 필드에서 javaagent 경로를 추출합니다.
	javaAgentPath, err := extractJavaAgentPath(containerJSON.Config.Cmd)
	if err == nil {
		return javaAgentPath, nil
	}

	// Entrypoint 필드에서 javaagent 경로를 추출합니다.
	javaAgentPath, err = extractJavaAgentPath(containerJSON.Config.Entrypoint)
	if err == nil {
		return javaAgentPath, nil
	}

	return "", fmt.Errorf("Java agent path not found in container %s", containerID)
}

// extractJavaAgentPath는 주어진 커맨드 배열에서 javaagent 경로를 추출
func extractJavaAgentPath(commands []string) (string, error) {
	for _, command := range commands {
		if strings.Contains(command, "javaagent:") {
			parts := strings.SplitN(command, "javaagent:", 2)
			if len(parts) > 1 {
				// javaagent 경로 추출 후 공백이나 다른 옵션으로 분리된 경우를 고려하여 첫 번째 부분만 반환합니다.
				javaAgentPath := strings.SplitN(parts[1], " ", 2)[0]
				return javaAgentPath, nil
			}
		}
	}
	return "", fmt.Errorf("javaagent path not found")
}
