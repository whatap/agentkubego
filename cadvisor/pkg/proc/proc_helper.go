package proc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/containerd/containerd/api/services/tasks/v1"
	whatap_client "github.com/whatap/kube/cadvisor/pkg/client"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	whatap_crio "github.com/whatap/kube/cadvisor/pkg/crio"
)

const (
	// 캐시된 PID 재사용 기간(초)
	pidCacheTTLSeconds = 60
	// stale 항목 일괄 정리 주기(초)
	pidCacheSweepIntervalSeconds = 600
	// TTL 만료 후 이 기간 동안 갱신되지 못한 항목은 죽은 컨테이너로 간주
	pidCacheMaxAgeSeconds = 600
)

var (
	containerPidLookup      = map[string][]int64{}
	containerPidLookupMutex = sync.RWMutex{}
	lastPidLookupSweepAt    int64
)

// RemoveContainerPid는 캐시에서 컨테이너 항목을 제거한다.
// 파드/컨테이너 삭제 시점에 호출해 죽은 컨테이너 ID가 누적되지 않게 한다.
func RemoveContainerPid(containerId string) {
	containerPidLookupMutex.Lock()
	defer containerPidLookupMutex.Unlock()
	delete(containerPidLookup, containerId)
}

// sweepStaleContainerPids는 갱신이 끊긴 지 maxAge를 넘긴 항목을 제거한다.
// 삭제 이벤트를 받지 못해 RemoveContainerPid가 호출되지 않은 항목의 안전망.
func sweepStaleContainerPids(now int64) {
	containerPidLookupMutex.Lock()
	defer containerPidLookupMutex.Unlock()
	if now-lastPidLookupSweepAt < pidCacheSweepIntervalSeconds {
		return
	}
	lastPidLookupSweepAt = now
	for id, vals := range containerPidLookup {
		if len(vals) < 2 || (now-vals[1]) > pidCacheMaxAgeSeconds {
			delete(containerPidLookup, id)
		}
	}
}

func GetContainerPid(containerId string) (int, error) {
	now := time.Now().Unix()
	containerPidLookupMutex.RLock()
	vals, ok := containerPidLookup[containerId]
	sweepDue := now-lastPidLookupSweepAt >= pidCacheSweepIntervalSeconds
	containerPidLookupMutex.RUnlock()
	if sweepDue {
		sweepStaleContainerPids(now)
	}
	if ok {
		pid := vals[0]
		timestamp := vals[1]
		if (now - timestamp) <= pidCacheTTLSeconds {
			return int(pid), nil
		}
	}

	runtime := whatap_config.GetConfig().Runtime
	if runtime == "docker" {
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
		// 조회 실패한 컨테이너의 만료 항목은 캐시에 남겨둘 이유가 없다
		RemoveContainerPid(containerId)
		return 0, err
	} else if runtime == "containerd" {
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
			// 조회 실패한 컨테이너의 만료 항목은 캐시에 남겨둘 이유가 없다
			RemoveContainerPid(containerId)
			return 0, err
		}

		containerPidLookupMutex.Lock()
		containerPidLookup[containerId] = []int64{int64(resp.Process.Pid), now}
		defer containerPidLookupMutex.Unlock()
		return int(containerPidLookup[containerId][0]), nil
	} else if runtime == "crio" {
		pid, err := whatap_crio.GetContainerPid(whatap_config.GetConfig().HostPathPrefix, containerId)
		if err != nil {
			return 0, err
		}
		return pid, err
	}
	return 0, fmt.Errorf("any container api not enabled")
}
