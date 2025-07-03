package proc

import (
	"context"
	"fmt"
	"github.com/containerd/containerd/api/services/tasks/v1"
	whatap_client "github.com/whatap/kube/cadvisor/pkg/client"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	whatap_crio "github.com/whatap/kube/cadvisor/pkg/crio"
	"github.com/whatap/kube/cadvisor/tools/util/runtimeutil"
	"github.com/whatap/kube/tools/util/logutil"
	"sync"
	"time"
)

var (
	containerPidLookup      = map[string][]int64{}
	containerPidLookupMutex = sync.RWMutex{}
)

func GetContainerPid(containerId string) (int, error) {
	logutil.Infof("GetContainerPid", "Starting PID lookup for container: %s", containerId)

	now := time.Now().Unix()
	containerPidLookupMutex.RLock()
	vals, ok := containerPidLookup[containerId]
	containerPidLookupMutex.RUnlock()
	if ok {
		pid := vals[0]
		timestamp := vals[1]
		if (now - timestamp) <= 60 {
			logutil.Infof("GetContainerPid", "Cache hit for container %s, PID: %d", containerId, int(pid))
			return int(pid), nil
		} else {
			logutil.Infof("GetContainerPid", "Cache expired for container %s (age: %d seconds)", containerId, now-timestamp)
		}
	} else {
		logutil.Infof("GetContainerPid", "No cache entry found for container: %s", containerId)
	}

	if runtimeutil.CheckDockerEnabled() {
		logutil.Infof("GetContainerPid", "Using Docker runtime for container: %s", containerId)
		cli, err := whatap_client.GetDockerClient()
		if err != nil {
			logutil.Errorf("GetContainerPid", "Failed to get Docker client: %v", err)
			return 0, err
		}
		// defer pc.Release()
		// cli := pc.Conn
		contInfo, err := cli.ContainerInspect(context.Background(), containerId)
		if err == nil {
			pid := contInfo.State.Pid
			logutil.Infof("GetContainerPid", "Docker: Found PID %d for container %s", pid, containerId)
			containerPidLookupMutex.Lock()
			containerPidLookup[containerId] = []int64{int64(pid), now}
			defer containerPidLookupMutex.Unlock()

			return int(containerPidLookup[containerId][0]), nil
		}
		logutil.Errorf("GetContainerPid", "Docker: Failed to inspect container %s: %v", containerId, err)
		return 0, err
	} else if runtimeutil.CheckContainerdEnabled() {
		logutil.Infof("GetContainerPid", "Using containerd runtime for container: %s", containerId)
		cli, err := whatap_client.GetContainerdClient()
		if err != nil {
			logutil.Errorf("GetContainerPid", "Failed to get containerd client: %v", err)
			return 0, err
		}
		_, ctx, err := whatap_client.LoadContainerD(containerId)
		if err != nil {
			logutil.Errorf("GetContainerPid", "Failed to load container %s from containerd: %v", containerId, err)
			return 0, err
		}

		logutil.Infof("GetContainerPid", "Containerd: Getting task info for container: %s", containerId)
		resp, err := cli.TaskService().Get(ctx, &tasks.GetRequest{
			ContainerID: containerId,
		})
		if err != nil {
			logutil.Errorf("GetContainerPid", "Containerd: Failed to get task for container %s: %v", containerId, err)
			return 0, err
		}

		pid := resp.Process.Pid
		logutil.Infof("GetContainerPid", "Containerd: Found PID %d for container %s", pid, containerId)
		containerPidLookupMutex.Lock()
		containerPidLookup[containerId] = []int64{int64(pid), now}
		defer containerPidLookupMutex.Unlock()
		return int(containerPidLookup[containerId][0]), nil
	} else if runtimeutil.CheckCrioEnabled() {
		logutil.Debugf("GetContainerPid", "Using CRI-O runtime for container: %s", containerId)
		pid, err := whatap_crio.GetContainerPid(whatap_config.GetConfig().HostPathPrefix, containerId)
		if err != nil {
			logutil.Errorf("GetContainerPid", "CRI-O: Failed to get PID for container %s: %v", containerId, err)
			return 0, err
		}
		logutil.Debugf("GetContainerPid", "CRI-O: Found PID %d for container %s", pid, containerId)
		return pid, err
	}
	logutil.Errorf("GetContainerPid", "No container runtime enabled for container: %s", containerId)
	return 0, fmt.Errorf("any container api not enabled")
}
