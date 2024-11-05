package cgroup

import (
	"encoding/json"
	"fmt"
	whatap_config "github.com/whatap/kube/node/src/whatap/config"
	whatap_model "github.com/whatap/kube/node/src/whatap/lang/model"
	"github.com/whatap/kube/node/src/whatap/util/logutil"
)

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

	cgroupMode := GetMode()
	if whatap_config.GetConfig().CgroupVersion == "" {
		whatap_config.GetConfig().CgroupVersion = cgroupMode
	}
	if whatap_config.GetConfig().Debug {
		logutil.Infof("CgroupCheck", "mode=%v", cgroupMode)
	}
	switch cgroupMode {
	case "legacy":
		return GetContainerStatsCgroupV1(prefix, containerId, name, cgroupParent,
			restartCount, pid, memoryLimit)
	case "hybrid":
		return GetContainerStatsCgroupV1(prefix, containerId, name, cgroupParent,
			restartCount, pid, memoryLimit)
	case "unified":
		return GetContainerStatsCgroupV2(prefix, containerId, name, cgroupParent,
			restartCount, pid, memoryLimit)
	default:
		return whatap_model.ContainerStat{}, fmt.Errorf("cgroup mode not supported %s", cgroupMode)
	}
}
