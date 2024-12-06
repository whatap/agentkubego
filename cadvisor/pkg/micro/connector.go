package micro

import (
	"fmt"
	"github.com/whatap/kube/cadvisor/pkg/containerd"
	"github.com/whatap/kube/cadvisor/pkg/crio"
	"github.com/whatap/kube/cadvisor/pkg/docker"
	"github.com/whatap/kube/cadvisor/tools/util/runtimeutil"
	"github.com/whatap/kube/tools/util/logutil"
	"regexp"
	"strconv"
	"strings"
)

var (
	IsContainerD = runtimeutil.CheckContainerdEnabled()
	IsDocker     = runtimeutil.CheckDockerEnabled()
	IsCrio       = runtimeutil.CheckCrioEnabled()
)

func InspectWhatapAgentPath(containerID string) (string, error) {
	if IsContainerD {
		return containerd.InspectContainerdImageForWhatapPath(containerID)
	}
	if IsDocker {
		return docker.InspectDockerImageForWhatapPath(containerID)
	}
	if IsCrio {
		return crio.InspectCrioImageForWhatapPath(containerID)
	}
	return "", fmt.Errorf("no container runtime detected")
}

func IsValidAgentPath(path string) bool {
	if strings.Contains(path, "whatap.agent.kube.jar") {
		return true
	}
	re := regexp.MustCompile(`whatap\.agent-(\d+\.\d+\.\d+)\.jar`)
	match := re.FindStringSubmatch(path)
	if match == nil {
		return false
	}
	version := match[1]
	return isVersionGreaterOrEqual(version, "2.2.33")
}

func isVersionGreaterOrEqual(v1, v2 string) bool {
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		v1Int, err1 := strconv.Atoi(v1Parts[i])
		v2Int, err2 := strconv.Atoi(v2Parts[i])
		if err1 != nil {
			logutil.Infoln("versionParseError err1=%v", err1)
			return false
		}
		if err2 != nil {
			logutil.Infoln("versionParseError err2=%v", err2)
			return false
		}
		if v1Int > v2Int {
			return true
		} else if v1Int < v2Int {
			return false
		}
	}
	return len(v1Parts) >= len(v2Parts)
}
