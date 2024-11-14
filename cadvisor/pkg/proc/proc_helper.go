package proc

import (
	"context"
	"fmt"
	"github.com/containerd/containerd/api/services/tasks/v1"
	whatap_client "github.com/whatap/kube/cadvisor/pkg/client"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	whatap_crio "github.com/whatap/kube/cadvisor/pkg/crio"
	"github.com/whatap/kube/cadvisor/tools/util/runtimeutil"
	"sync"
	"time"
)

var (
	containerPidLookup      = map[string][]int64{}
	containerPidLookupMutex = sync.RWMutex{}
)

func GetContainerPid(containerId string) (int, error) {
	now := time.Now().Unix()
	containerPidLookupMutex.RLock()
	vals, ok := containerPidLookup[containerId]
	containerPidLookupMutex.RUnlock()
	if ok {
		pid := vals[0]
		timestamp := vals[1]
		if (now - timestamp) <= 60 {
			return int(pid), nil
		}
	}

	if runtimeutil.CheckDockerEnabled() {
		cli, err := whatap_client.GetDockerClient()
		if err != nil {
			return 0, err
		}
		// defer pc.Release()
		// cli := pc.Conn
		contInfo, err := cli.ContainerInspect(context.Background(), containerId)
		if err == nil {
			containerPidLookupMutex.Lock()
			containerPidLookup[containerId] = []int64{int64(contInfo.State.Pid), now}
			defer containerPidLookupMutex.Unlock()

			return int(containerPidLookup[containerId][0]), nil
		}
		return 0, err
	} else if runtimeutil.CheckContainerdEnabled() {
		cli, err := whatap_client.GetContainerdClient()
		if err != nil {
			return 0, err
		}
		_, ctx, err := whatap_client.LoadContainerD(containerId)
		if err != nil {
			return 0, err
		}

		resp, err := cli.TaskService().Get(ctx, &tasks.GetRequest{
			ContainerID: containerId,
		})
		if err != nil {
			return 0, err
		}

		containerPidLookupMutex.Lock()
		containerPidLookup[containerId] = []int64{int64(resp.Process.Pid), now}
		defer containerPidLookupMutex.Unlock()
		return int(containerPidLookup[containerId][0]), nil
	} else if runtimeutil.CheckCrioEnabled() {
		pid, err := whatap_crio.GetContainerPid(whatap_config.GetConfig().HostPathPrefix, containerId)
		if err != nil {
			return 0, err
		}
		return pid, err
	}
	return 0, fmt.Errorf("any container api not enabled")
}
