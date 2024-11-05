package crio

import (
	// "fmt"
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	whatap_model "github.com/whatap/kube/node/src/whatap/lang/model"
	"github.com/whatap/kube/node/src/whatap/util/stringutil"
)

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

func populateCgroupKeyValue(prefix string, device string, cgroupsPath string, filename string, callback func(key string, v int64)) {
	// fmt.Println("populateCgroupKeyValue step -1 ", prefix, device, cgroupsPath, filename)
	// /rootfs/sys/fs/cgroup/cpu/kubepods/besteffort/podb889ec29-d166-4ba6-b98c-bc56d99f2d69/crio-a695a8eef261faf66a26a766883e2751f9e96b79b252cd9451e8b037c3024465
	cgroup_realpath := parseRealPath(cgroupsPath)
	// fmt.Println("populateCgroupKeyValue step -3 ",cgroup_realpath)
	calculated_path := filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)
	// fmt.Println("populateCgroupKeyValue step -4 ", calculated_path)
	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
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
}

func populateCgroupValues(prefix string, device string, cgroupsPath string, filename string, callback func(tokens []string)) {
	cgroup_realpath := parseRealPath(cgroupsPath)

	calculated_path := filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)
	// fmt.Println("populateCgroupValues calculated_path:",calculated_path)
	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
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
}

func populateFileKeyValue(prefix string, filename string, callback func(key string, v []int64)) {
	calculated_path := filepath.Join(prefix, filename)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
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

}

func populateFileValues(prefix string, filename string, callback func(tokens []string)) {
	calculated_path := filepath.Join(prefix, filename)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
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
}

func GetContainerParams(prefix string, containerId string, h3 func(string, string, int) error) error {
	oconfig, err := getOverlayConfig(prefix, containerId)
	if err != nil {
		return err
	}

	ostate, err := getOverlayState(prefix, containerId)
	if err != nil {
		return err
	}

	return h3(oconfig.Annotations.IoKubernetesContainerName, oconfig.Linux.CgroupsPath, ostate.Pid)
}

func GetContainerStats(prefix string, containerId string) (whatap_model.ContainerStat, error) {
	var containerStat whatap_model.ContainerStat
	oconfig, err := getOverlayConfig(prefix, containerId)
	if err != nil {
		return containerStat, err
	}

	containerStat.Name = oconfig.Annotations.IoKubernetesContainerName
	containerStat.ID = containerId
	restartCount, e := strconv.ParseInt(oconfig.Annotations.IoKubernetesContainerRestartCount, 10, 64)
	if e == nil {
		containerStat.RestartCount = int(restartCount)
	}

	populateFileKeyValue(prefix, "/proc/stat", func(key string, v []int64) {
		if key == "cpu" {
			for i, ev := range v {
				if i < 8 {
					containerStat.CPUStats.SystemCPUUsage += ev
				}
			}
		}
	})
	// fmt.Println("GetContainerStats step -1")
	populateCgroupKeyValue(prefix, "cpu", oconfig.Linux.CgroupsPath, "cpu.stat", func(key string, v int64) {
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
	populateCgroupKeyValue(prefix, "cpu", oconfig.Linux.CgroupsPath, "cpuacct.stat", func(key string, v int64) {
		// fmt.Println("populateCgroupKeyValue: ",key, v)
		if key == "user" {
			containerStat.CPUStats.CPUUsage.UsageInUsermode = v
		} else if key == "system" {
			containerStat.CPUStats.CPUUsage.UsageInKernelmode = v
		}
	})

	populateCgroupKeyValue(prefix, "cpu", oconfig.Linux.CgroupsPath, "cpuacct.shares", func(key string, v int64) {
		containerStat.CPUStats.OnlineCpus = int(v)
	})

	containerStat.CPUStats.CPUUsage.TotalUsage = containerStat.CPUStats.CPUUsage.UsageInUsermode + containerStat.CPUStats.CPUUsage.UsageInKernelmode

	// fmt.Println("GetContainerStats step -3")
	populateCgroupKeyValue(prefix, "memory", oconfig.Linux.CgroupsPath, "memory.max_usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.MaxUsage = v
	})

	populateCgroupKeyValue(prefix, "memory", oconfig.Linux.CgroupsPath, "memory.usage_in_bytes", func(key string, v int64) {
		containerStat.MemoryStats.Usage = v
	})

	populateCgroupKeyValue(prefix, "memory", oconfig.Linux.CgroupsPath, "memory.failcnt", func(key string, v int64) {
		containerStat.MemoryStats.FailCnt = int(v)
	})

	containerStat.MemoryStats.Limit = oconfig.Linux.Resources.Memory.Limit
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

	populateCgroupKeyValue(prefix, "memory", oconfig.Linux.CgroupsPath, "memory.stat", func(key string, v int64) {
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
	populateCgroupValues(prefix, "blkio", oconfig.Linux.CgroupsPath, "blkio.io_service_bytes_recursive", callbackIoServiceBytesRecursive)
	populateCgroupValues(prefix, "blkio", oconfig.Linux.CgroupsPath, "blkio.throttle.io_service_bytes", callbackIoServiceBytesRecursive)
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
	populateCgroupValues(prefix, "blkio", oconfig.Linux.CgroupsPath, "blkio.io_serviced_recursive", callbackIoServicedRecursive)
	populateCgroupValues(prefix, "blkio", oconfig.Linux.CgroupsPath, "blkio.throttle.io_serviced", callbackIoServicedRecursive)

	ostate, err := getOverlayState(prefix, containerId)
	if err != nil {
		// fmt.Println("GetContainerStats step -6", err)
		return containerStat, err
	}
	// fmt.Println("GetContainerStats step -7")

	populateFileValues(prefix, filepath.Join("proc", fmt.Sprint(ostate.Pid), "net/dev"), func(tokens []string) {
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

	return containerStat, nil
	// fmt.Println("GetContainerStats step -8")
	//containerStatJson, err := json.Marshal(containerStat)
	//if err == nil {
	//	return string(containerStatJson), nil
	//}
	// fmt.Println("GetContainerStats step -9", err)
	//return "", err
}

func GetOverlayConfig(prefix string, containerId string, h1 func(whatap_model.OverlayConfig)) (err error) {
	oc_config, err := getOverlayConfig(prefix, containerId)
	if h1 != nil && err == nil {
		h1(oc_config)
	}
	return

}

func getOverlayConfig(prefix string, containerId string) (oconfig whatap_model.OverlayConfig, err error) {
	// 기본 경로 설정
	overlay_config_path := filepath.Join(prefix, "var/lib/containers/storage/overlay-containers", containerId, "userdata", "config.json")
	// 파일 존재 여부 확인
	if _, err = os.Stat(overlay_config_path); os.IsNotExist(err) {
		// 환경 변수에서 대체 경로 가져오기
		envPath := os.Getenv("overlay_config_path")
		if envPath == "" {
			// 환경 변수가 설정되어 있지 않으면 에러 반환
			return
		}
		// 환경 변수에서 가져온 경로로 전체 파일 경로 재설정
		overlay_config_path = filepath.Join(envPath, containerId, "userdata", "config.json")

		// 대체 경로에도 파일이 없으면 반환
		if _, err = os.Stat(overlay_config_path); os.IsNotExist(err) {
			return
		}
	}

	// 파일 읽기
	oc_bytes, err := ioutil.ReadFile(overlay_config_path)
	if err != nil {
		return
	}

	// JSON 데이터 언마샬링
	err = json.Unmarshal(oc_bytes, &oconfig)

	return
}

func getOverlayState(prefix string, containerId string) (ostate whatap_model.OverlayState, err error) {
	// 기본 경로 설정
	overlay_state_path := filepath.Join(prefix, "/var/lib/containers/storage/overlay-containers", containerId, "userdata", "state.json")

	// 파일 읽기 시도
	os_bytes, err := ioutil.ReadFile(overlay_state_path)
	if err != nil {
		if os.IsNotExist(err) {
			// 환경 변수에서 대체 경로 가져오기
			overlay_state_path = os.Getenv("overlay_state_path")
			if overlay_state_path == "" {
				// 환경 변수가 설정되어 있지 않으면 에러 반환
				return
			}

			// 환경 변수에서 가져온 경로로 전체 파일 경로 재설정
			overlay_state_path = filepath.Join(overlay_state_path, containerId, "userdata", "state.json")

			// 대체 경로에서 파일 읽기 시도
			os_bytes, err = ioutil.ReadFile(overlay_state_path)
			if err != nil {
				// 여전히 파일을 읽을 수 없으면 에러 반환
				return
			}
		} else {
			// 다른 종류의 에러가 발생하면 바로 에러 반환
			return
		}
	}

	// JSON 데이터 언마샬링
	err = json.Unmarshal(os_bytes, &ostate)
	if err != nil {
		return
	}

	return
}

func GetContainerPid(prefix string, containerId string) (int, error) {
	ostate, err := getOverlayState(prefix, containerId)
	if err != nil {
		return 0, err
	}
	pid := ostate.Pid
	return pid, nil
}

func GetContainerInspect(prefix string, containerId string) (string, error) {
	oconfig, err := getOverlayConfig(prefix, containerId)
	if err != nil {
		return "", err
	}

	ostate, err := getOverlayState(prefix, containerId)
	if err != nil {
		return "", err
	}

	var cinfo whatap_model.ContainerInfo
	cinfo.ID = containerId
	cinfo.Created = ostate.Created
	cinfo.Path = oconfig.Root.Path
	cinfo.HostConfig.CPUPeriod = oconfig.Linux.Resources.CPU.Period
	cinfo.HostConfig.CPUQuota = oconfig.Linux.Resources.CPU.Quota
	cinfo.HostConfig.CPUShares = oconfig.Linux.Resources.CPU.Shares
	cinfo.State.Status = ostate.Status
	cinfo.State.StartedAt = ostate.Started
	cinfo.State.FinishedAt = ostate.Finished
	switch ostate.Status {
	case "running":
		cinfo.State.Running = true
	case "exited":
		cinfo.State.Dead = true
	case "paused":
		cinfo.State.Paused = true
	default:

	}
	cinfo.State.Pid = ostate.Pid
	cinfo.LogPath = oconfig.Annotations.IoKubernetesCriOLogPath

	for _, m := range oconfig.Mounts {
		cinfo.Mounts = append(cinfo.Mounts, whatap_model.Mount{
			Source:      m.Source,
			Destination: m.Destination})
	}

	cinfojson, err := json.Marshal(cinfo)
	if err == nil {
		return string(cinfojson), nil
	}
	return "", err
}

func CheckCrioEnabled() bool {
	fi, err := os.Stat("/var/run/crio/crio.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}
func InspectCrioImageForWhatapPath(containerID string) (string, error) {
	oconfig, err := getOverlayConfig("/rootfs", containerID)
	if err != nil {
		return "", err
	}
	for _, arg := range oconfig.Process.Args {
		if strings.Contains(arg, "javaagent:") {
			// "javaagent:" 문자열로 시작하는 부분을 찾아냅니다.
			javaAgentPath := strings.SplitN(arg, "javaagent:", 2)[1]
			// javaagent 경로 추출 후 공백이나 다른 옵션으로 분리된 경우를 고려하여 첫 번째 부분만 반환합니다.
			javaAgentPath = strings.SplitN(javaAgentPath, " ", 2)[0]
			return javaAgentPath, nil
		}
	}
	return "", fmt.Errorf("javaagent path not found")
}
