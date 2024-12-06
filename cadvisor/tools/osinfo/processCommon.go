package osinfo

import (
	"fmt"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//var (
//	containerPidLookup      = map[string][]int64{}
//	containerPidLookupMutex = sync.RWMutex{}
//)
//
//func GetContainerPid(containerId string) (int, error) {
//	now := time.Now().Unix()
//	containerPidLookupMutex.RLock()
//	vals, ok := containerPidLookup[containerId]
//	containerPidLookupMutex.RUnlock()
//	if ok {
//		pid := vals[0]
//		timestamp := vals[1]
//		if (now - timestamp) <= 60 {
//			return int(pid), nil
//		}
//	}
//
//	if whatap_docker.CheckDockerEnabled() {
//		cli, err := whatap_docker.GetDockerClient()
//		if err != nil {
//			return 0, err
//		}
//		// defer pc.Release()
//		// cli := pc.Conn
//		contInfo, err := cli.ContainerInspect(context.Background(), containerId)
//		if err == nil {
//			containerPidLookupMutex.Lock()
//			containerPidLookup[containerId] = []int64{int64(contInfo.State.Pid), now}
//			defer containerPidLookupMutex.Unlock()
//
//			return int(containerPidLookup[containerId][0]), nil
//		}
//		return 0, err
//	} else if whatap_containerd.CheckContainerdEnabled() {
//		cli, err := whatap_containerd.GetContainerdClient()
//		if err != nil {
//			return 0, err
//		}
//		_, ctx, err := whatap_containerd.LoadContainerD(containerId)
//		if err != nil {
//			return 0, err
//		}
//
//		resp, err := cli.TaskService().Get(ctx, &tasks.GetRequest{
//			ContainerID: containerId,
//		})
//		if err != nil {
//			return 0, err
//		}
//
//		containerPidLookupMutex.Lock()
//		containerPidLookup[containerId] = []int64{int64(resp.Process.Pid), now}
//		defer containerPidLookupMutex.Unlock()
//		return int(containerPidLookup[containerId][0]), nil
//	} else if whatap_crio.CheckCrioEnabled() {
//		pid, err := whatap_crio.GetContainerPid(whatap_config.GetConfig().HostPathPrefix, containerId)
//		if err != nil {
//			return 0, err
//		}
//		return pid, err
//	}
//	return 0, fmt.Errorf("any container api not enabled")
//}

func FindChildPIDs(parentPID int) ([]int, error) {
	childPIDs := []int{}
	hostPrefix := whatap_config.GetConfig().HostPathPrefix
	procPath := filepath.Join(hostPrefix, "proc")

	// Read the /proc directory.
	files, err := os.ReadDir(procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %v directory: err=%w", procPath, err)
	}

	// Iterate over each entry in /proc.
	for _, file := range files {
		// Check if the entry is a directory and the name is a PID.
		if file.IsDir() && isNumeric(file.Name()) {
			pid := file.Name()
			statusPath := filepath.Join(procPath, pid, "status")

			// Read the status file.
			statusContent, err := os.ReadFile(statusPath)
			if err != nil {
				// Skip files that cannot be read
				continue
			}

			// Parse the status file to find the PPID.
			ppid, err := parsePPID(string(statusContent))
			if err != nil {
				// Skip files that cannot be parsed
				continue
			}

			// If the PPID matches the given parent PID, add the PID to the child list.
			if ppid == parentPID {
				childPID, err := strconv.Atoi(pid)
				if err != nil {
					return nil, fmt.Errorf("failed to convert PID to int: %w", err)
				}
				childPIDs = append(childPIDs, childPID)
			}
		}
	}
	return childPIDs, nil
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
func parsePPID(statusContent string) (int, error) {
	lines := strings.Split(statusContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Pid:") {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				ppid, err := strconv.Atoi(fields[1])
				if err != nil {
					return 0, fmt.Errorf("failed to convert PPID to int: %w", err)
				}
				return ppid, nil
			}
		}
	}
	return 0, fmt.Errorf("PPID not found in status content")
}

//func FindAllPIDs() ([]int, error) {
//	childPIDs := []int{}
//	hostPrefix := whatap_config.GetConfig().HostPathPrefix
//	procPath := filepath.Join(hostPrefix, "proc")
//
//	// Read the /proc directory.
//	files, err := os.ReadDir(procPath)
//	if err != nil {
//		return nil, fmt.Errorf("failed to read %v directory: err=%w", procPath, err)
//	}
//
//	// Iterate over each entry in /proc.
//	for _, file := range files {
//		// Check if the entry is a directory and the name is a PID.
//		if file.IsDir() && isNumeric(file.Name()) {
//			pid := file.Name()
//			statusPath := filepath.Join(procPath, pid, "status")
//
//			// Read the status file.
//			statusContent, err := os.ReadFile(statusPath)
//			if err != nil {
//				// Skip files that cannot be read
//				continue
//			}
//
//			// Parse the status file to find the PPID.
//			ppid, err := parsePPID(string(statusContent))
//			if err != nil {
//				// Skip files that cannot be parsed
//				continue
//			}
//
//			// If the PPID matches the given parent PID, add the PID to the child list.
//			if ppid == parentPID {
//				childPID, err := strconv.Atoi(pid)
//				if err != nil {
//					return nil, fmt.Errorf("failed to convert PID to int: %w", err)
//				}
//				childPIDs = append(childPIDs, childPID)
//			}
//		}
//	}
//	return childPIDs, nil
//}
