package crio

import (
	// "fmt"
	"bufio"
	"encoding/json"
	"fmt"
	whatap_cgroup "github.com/whatap/kube/cadvisor/pkg/cgroup"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	whatap_model "github.com/whatap/kube/cadvisor/pkg/model"
	"github.com/whatap/kube/tools/util/stringutil"
)

var (
	HOSTPATH_PREFIX = whatap_config.GetConfig().HostPathPrefix
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

func GetContainerStats(containerId string) (statsjson string, statserr error) {
	restartCount, _, err := whatap_cgroup.GetContainerRestartCount(containerId)
	if err != nil {
		return "", err
	}

	err = GetContainerParams(HOSTPATH_PREFIX, containerId,
		func(name string, cgroupParent string, pid int) error {
			statsjson, statserr = whatap_cgroup.GetContainerStatsEx(HOSTPATH_PREFIX, containerId, name, cgroupParent,
				restartCount, pid, 0)

			return statserr
		})

	if err != nil {
		return "", err
	}

	return
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
	return GetContainerInspectWithPid(prefix, containerId, 0)
}

func GetContainerInspectWithPid(prefix string, containerId string, unifiedPid int) (string, error) {
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

	// Use unified PID if provided, otherwise fall back to CRI-O's original PID
	if unifiedPid > 0 {
		cinfo.State.Pid = unifiedPid
	} else {
		cinfo.State.Pid = ostate.Pid
	}

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
