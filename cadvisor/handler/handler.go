package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	whatap_cgroup "github.com/whatap/kube/cadvisor/pkg/cgroup"
	"github.com/whatap/kube/cadvisor/pkg/client"
	whatap_client "github.com/whatap/kube/cadvisor/pkg/client"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	whatap_containerd "github.com/whatap/kube/cadvisor/pkg/containerd"
	whatap_crio "github.com/whatap/kube/cadvisor/pkg/crio"
	whatap_docker "github.com/whatap/kube/cadvisor/pkg/docker"
	whatap_model "github.com/whatap/kube/cadvisor/pkg/model"
	whatap_osinfo "github.com/whatap/kube/cadvisor/tools/osinfo"
	"github.com/whatap/kube/cadvisor/tools/util/runtimeutil"
	"github.com/whatap/kube/tools/util/logutil"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	re = regexp.MustCompile("([0-9\\.]+s)$")
)

var (
	HOSTPATH_PREFIX = whatap_config.GetConfig().HostPathPrefix
)

func HealthHandler(w http.ResponseWriter, req *http.Request) {
	contentType := "applicatin/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte("{\"OK\": 1}"))
}

func DebugGoroutineHandler(w http.ResponseWriter, req *http.Request) {
	contentType := "text/plain"
	w.Header().Add("Content-Type", contentType)
	p := pprof.Lookup("goroutine")
	p.WriteTo(w, 1)
}

func GetAllContainerHandler(w http.ResponseWriter, req *http.Request) {
	content, err := getAllContainers()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Header().Add("X-CONTAINER-RUNTIME", "containerd")
	w.Write([]byte(content))
}

func GetContainerInspectHandler(w http.ResponseWriter, req *http.Request) {
	containerRuntime := whatap_config.GetConfig().Runtime
	vars := mux.Vars(req)
	containerId := vars["containerid"]
	if len(containerId) < 1 {
		w.WriteHeader(404)
		w.Write([]byte("containerid missing"))
		return
	}
	var content string
	var err error
	if containerRuntime == "docker" {
		content, err = whatap_docker.GetContainerInspect(containerId)
	} else if containerRuntime == "containerd" {
		content, err = whatap_containerd.GetContainerInspect(containerId)
	} else if containerRuntime == "crio" {
		content, err = whatap_crio.GetContainerInspect(HOSTPATH_PREFIX, containerId)
	}

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	contentType := "applicatin/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(content))
}
func GetContainerStatsHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	containerId := vars["containerid"]

	// Validate containerId
	if len(containerId) < 1 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("containerid missing"))
		return
	}

	// Get runtime configuration
	containerRuntime := whatap_config.GetConfig().Runtime
	var content string
	var err error

	// Runtime-specific logic
	switch containerRuntime {
	case "docker":
		content, err = whatap_docker.GetContainerStats(containerId)
	case "containerd":
		content, err = whatap_containerd.GetContainerStats(containerId)
	case "crio":
		content, err = whatap_crio.GetContainerStats(containerId)
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("unsupported container runtime"))
		return
	}

	// Error handling for runtime-specific logic
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// Return the stats as JSON
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func GetContainerLogHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	containerId := vars["containerid"]
	if len(containerId) < 1 {
		w.WriteHeader(404)
		w.Write([]byte("containerid missing"))
		return
	}
	tail := vars["tail"]

	contentType := "plain/text; charset=utf8"
	w.Header().Add("Content-Type", contentType)

	err := getContainerLogsContainerd(containerId, tail,
		func(buf []byte) {
			w.Write(buf)
		})
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

}

type mountInfo struct {
	Source    string
	Target    string
	MountType string `json:",omitempty"`
	Driver    string `json:",omitempty"`
}
type volumeinfo map[string]mountInfo

func GetContainerVolumeHandler(w http.ResponseWriter, req *http.Request) {
	if !whatap_config.GetConfig().CollectVolumeDetailEnabled {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf("CollectVolumeDetailEnabled=%v", whatap_config.GetConfig().CollectVolumeDetailEnabled)))
		return
	}
	vars := mux.Vars(req)
	containerid := vars["containerid"]
	var err error
	var volumeLookup *volumeinfo
	if runtimeutil.CheckDockerEnabled() {
		volumeLookup, err = getVolumeInfo(containerid)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
	} else if runtimeutil.CheckContainerdEnabled() {
		volumeLookup, err = getVolumeInfoEx(containerid)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
	} else {
		volumeLookup, err = getVolumeInfoCrio(containerid)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
	}
	var volumes []whatap_model.VolumeUsage

	err = whatap_cgroup.ExecuteCommandEx(containerid, []string{"df"}, true, func(line string) {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			return
		}
		mountpath := fields[5]

		if strings.HasSuffix(mountpath, "/shm") ||
			strings.HasSuffix(mountpath, "/merged") ||
			strings.HasPrefix(mountpath, "Mounted") {
			return
		}
		filesystem := fields[0]
		if filesystem == "tmpfs" || filesystem == "shm" || filesystem == "Filesystem" {
			return
		}
		var source string
		var driver string
		var mountType string
		//fmt.Println("containerVolumeHandler step -1 ", mountpath)
		if volumeinfo, ok := (*volumeLookup)[mountpath]; ok {
			//fmt.Println("containerVolumeHandler step -1.1 ", volumeinfo.Source)
			source = volumeinfo.Source
			driver = volumeinfo.Driver
			mountType = volumeinfo.MountType
		}
		total, _ := strconv.ParseInt(fields[1], 10, 64)
		total *= 1024
		pused, _ := strconv.ParseFloat(strings.Split(fields[4], "%")[0], 64)
		used := int64(float64(total) * pused / float64(100))
		volume := whatap_model.VolumeUsage{
			Source:      source,
			Driver:      driver,
			Mount:       mountpath,
			MountType:   mountType,
			Total:       total,
			Used:        used,
			UsedPercent: pused}

		volumes = append(volumes, volume)
	})
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contJson, err := json.Marshal(volumes)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(contJson))
}

type Netstat struct {
	Listen []string
	Outer  []string
}

func GetContainerNetstatHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var netstat Netstat
	containerid := vars["containerid"]

	if runtimeutil.CheckDockerEnabled() {
		err := whatap_docker.ExecuteCommand(containerid, []string{"ss", "-at"}, true,
			func(line string) {
				fields := strings.Fields(line)
				if len(fields) != 5 {
					return
				}
				state := fields[0]
				addr := fields[3]
				if state == "LISTEN" {
					netstat.Listen = append(netstat.Listen, addr)
				} else if state == "ESTAB" {
					netstat.Outer = append(netstat.Outer, addr)
				}
			})
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
	} else if runtimeutil.CheckContainerdEnabled() {
		err := whatap_cgroup.ExecuteCommandEx(containerid, []string{"ss", "-at"}, true,
			func(line string) {
				fields := strings.Fields(line)
				if len(fields) != 5 {
					return
				}
				state := fields[0]
				addr := fields[3]
				if state == "LISTEN" {
					netstat.Listen = append(netstat.Listen, addr)
				} else if state == "ESTAB" {
					netstat.Outer = append(netstat.Outer, addr)
				}
			})
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
	}

	contJson, err := json.Marshal(netstat)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(contJson))
}

func GetHostDiskHandler(w http.ResponseWriter, req *http.Request) {
	content, err := getDiskUsage()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(content))
}
func GetHostProcessHandler(w http.ResponseWriter, req *http.Request) {
	processPerfs := whatap_osinfo.GetProcessPerfList()
	contentType := "application/json"
	if whatap_config.GetConfig().Debug {
		for _, perf := range *processPerfs {
			logutil.Infof("PHTEST", "%v", perf)
		}
	}
	processPerfsJson, processHandlerErr := json.Marshal(processPerfs)
	if processHandlerErr != nil {
		logutil.Errorf("processPerfsJson-jsonMarshal", "error=%v\n", processHandlerErr)
		w.WriteHeader(500)
		_, err := w.Write([]byte(processHandlerErr.Error()))
		if err != nil {
			logutil.Errorf("processPerfsJson-jsonMarshal", "error-write-Error=%v\n", err.Error())
			return
		}
		return
	}
	w.Header().Add("Content-Type", contentType)
	_, err := w.Write(processPerfsJson)
	if err != nil {
		logutil.Errorf("processPerfsJson-jsonMarshal", "processPerfsJson-write-Error=%v\n", err.Error())
		return
	}
}

func getContainerLogsContainerd(containerId, tail string, h1 func([]byte)) error {
	cli, err := client.GetKubernetesClient()
	if err != nil {
		return err
	}
	nodename := os.Getenv("NODE_NAME")
	listOptions := metav1.ListOptions{FieldSelector: fmt.Sprint("spec.nodeName=", nodename)}
	pods, err := cli.CoreV1().Pods("").List(context.Background(), listOptions)
	if err != nil {
		return err
	}
	var podns, podname, containerName string
	for _, pod := range pods.Items {
		for _, c := range pod.Status.ContainerStatuses {
			containerid := c.ContainerID
			if strings.Contains(containerid, "//") {
				containerid = strings.Split(containerid, "//")[1]
			}
			if containerid == containerId {
				podname = pod.Name
				podns = pod.Namespace
				containerName = c.Name
			}
		}
	}

	if len(podname) < 1 {
		return fmt.Errorf("container not found")
	}

	taillines, err := strconv.ParseInt(tail, 0, 64)
	if err != nil {
		return err
	}

	podLogOpts := v1.PodLogOptions{Follow: false, Container: containerName, TailLines: &taillines}
	req := cli.CoreV1().Pods(podns).GetLogs(podname, &podLogOpts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {

		return err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return err
	}
	h1(buf.Bytes())

	return nil
}
func getAllContainers() (string, error) {
	cli, err := client.GetKubernetesClient()
	if err != nil {
		return "", err
	}
	nodename := os.Getenv("NODE_NAME")

	nodelistOptions := metav1.ListOptions{FieldSelector: fmt.Sprint("metadata.name=", nodename)}
	nodesResp, nodeserr := cli.CoreV1().Nodes().List(context.Background(), nodelistOptions)
	if nodeserr != nil {
		return "", nodeserr
	}

	var nodeCpu resource.Quantity
	var nodeMemory resource.Quantity
	for _, node := range nodesResp.Items {
		if q, ok := node.Status.Capacity["cpu"]; ok {
			nodeCpu = q
		}

		if q, ok := node.Status.Capacity["memory"]; ok {
			nodeMemory = q
		}
	}

	listOptions := metav1.ListOptions{FieldSelector: fmt.Sprint("spec.nodeName=", nodename)}
	pods, err := cli.CoreV1().Pods("").List(context.Background(), listOptions)
	if err != nil {
		return "", err
	}

	var csums []whatap_model.ContainerSummary
	for _, pod := range pods.Items {
		for _, c := range pod.Status.ContainerStatuses {
			var state, status string

			if c.State.Waiting != nil {
				state = "waiting"
				status = c.State.Waiting.Reason
			} else if c.State.Running != nil {
				state = "running"
				status = fmt.Sprint("Up ", time.Since(c.State.Running.StartedAt.Time).String())

				status = re.ReplaceAllString(status, "")
			} else if c.State.Terminated != nil {
				state = "terminated"
				status = c.State.Terminated.Reason
			}
			var command string
			var names []string
			names = append(names, c.Name)
			var cpuLimit, memoryLimit, cpuRequest, memoryRequest interface{}
			for _, cinfo := range pod.Spec.Containers {
				if c.Name == cinfo.Name {
					command = strings.Join(cinfo.Command, " ")
					if cinfo.Resources.Limits.Cpu() == nil || cinfo.Resources.Limits.Cpu().Value() == 0 {
						cpuLimit = nodeCpu
					} else {
						cpuLimit = cinfo.Resources.Limits.Cpu()
					}
					if cinfo.Resources.Limits.Memory() == nil || cinfo.Resources.Limits.Memory().Value() == 0 {
						memoryLimit = nodeMemory
					} else {
						memoryLimit = cinfo.Resources.Limits.Memory()
					}

					cpuRequest = cinfo.Resources.Requests.Cpu()
					memoryRequest = cinfo.Resources.Requests.Memory()
				}
			}

			containerId := c.ContainerID
			if strings.Index(c.ContainerID, "//") >= 0 {
				containerId = strings.Split(c.ContainerID, "//")[1]
			}
			var ready int32 = 0
			if c.Ready {
				ready = 1
			}

			csums = append(csums, whatap_model.ContainerSummary{
				Image:         c.Image,
				ImageID:       c.ImageID,
				RestartCount:  c.RestartCount,
				Status:        status,
				State:         state,
				Names:         names,
				Id:            containerId,
				Podname:       pod.Name,
				Command:       command,
				Created:       pod.Status.StartTime.Unix() * 1000,
				Labels:        pod.Labels,
				CpuLimit:      cpuLimit,
				MemoryLimit:   memoryLimit,
				Namespace:     pod.Namespace,
				CpuRequest:    cpuRequest,
				MemoryRequest: memoryRequest,
				Ready:         ready})
		}
	}

	statsJson, err := json.Marshal(csums)
	if err == nil {
		return string(statsJson), nil
	}

	return "", err
}
func getVolumeInfoEx(containerid string) (*volumeinfo, error) {
	resp, ctx, err := whatap_client.LoadContainerD(containerid)
	if err != nil {
		return nil, err
	}
	spec, err := resp.Spec(ctx)
	if err != nil {
		return nil, err
	}
	v := volumeinfo{}
	for _, s := range spec.Mounts {
		v[s.Destination] = mountInfo{Source: s.Source,
			Target:    s.Destination,
			MountType: s.Type}
	}

	return &v, nil
}
func getVolumeInfo(containerid string) (*volumeinfo, error) {
	cli, err := whatap_client.GetDockerClient()
	if err != nil {
		return nil, err
	}
	// defer pc.Release()
	// cli := pc.Conn
	resp, err := cli.ContainerInspect(context.Background(), containerid)
	if err != nil {
		return nil, err
	}
	v := volumeinfo{}

	for _, mount := range resp.Mounts {
		if !mount.RW {
			continue
		}

		v[mount.Destination] = mountInfo{Source: mount.Source,
			Target:    mount.Destination,
			MountType: string(mount.Type),
			Driver:    mount.Driver}
	}

	return &v, nil
}
func getVolumeInfoCrio(containerid string) (*volumeinfo, error) {
	v := volumeinfo{}

	err := whatap_crio.GetOverlayConfig(HOSTPATH_PREFIX, containerid, func(occonfig whatap_model.OverlayConfig) {
		for _, m := range occonfig.Mounts {
			v[m.Destination] = mountInfo{Source: m.Source,
				Target:    m.Destination,
				MountType: m.Type}
		}
	})
	if err != nil {
		return nil, err
	}

	return &v, nil
}
func getDiskUsage() (ret []byte, err error) {
	statsFile := "/proc/1/mountinfo"
	var diskPerfList []whatap_model.NodeDiskPerf

	deviceFilter := map[string]string{}
	errthistime := populateFileValues(HOSTPATH_PREFIX, statsFile, func(words []string) {

		var filesystem string
		var device string
		if len(words) == 10 {
			filesystem = words[7]
			device = words[8]
		} else {
			filesystem = words[8]
			device = words[9]
		}
		if !whatap_config.GetConfig().CollectNfsDiskEnabled {
			if whatap_config.GetConfig().Debug {
				logutil.Debugf("NfsDisk", "enabled=%v\n", whatap_config.GetConfig().CollectNfsDiskEnabled)
			}
			if strings.Contains(filesystem, "nfs") {
				if whatap_config.GetConfig().Debug {
					logutil.Debugf("NfsDisk", "filesystem=%v\n", filesystem)
				}
				return
			}
		}
		deviceID := words[2]
		majorminor := strings.Split(words[2], ":")
		mountPoint := words[4]

		mountId := words[0]
		parentMountId := words[1]

		deviceFilter[mountId] = mountPoint
		if _, ok := deviceFilter[parentMountId]; ok {
			if deviceFilter[parentMountId] == mountPoint {
				return
			}
		}

		mountPointReal := filepath.Join(HOSTPATH_PREFIX, mountPoint)

		var stat syscall.Statfs_t
		syscallError := syscall.Statfs(mountPointReal, &stat)
		if syscallError != nil {
			logutil.Debugf("syscall", "error=%v\n", syscallError)
			return
		}
		if stat.Blocks < 1 {
			logutil.Debugf("BlockSize", "mountPoint=%v, stat.Blocks=%v\n", mountPointReal, stat.Blocks)
			return
		}

		totalSpace := uint64(stat.Bsize) * uint64(stat.Blocks)
		freeSpace := totalSpace - uint64(stat.Bsize)*uint64(stat.Bavail)
		freePercent := float64(100.0 * float64(stat.Bfree) / float64(stat.Blocks-stat.Bfree+stat.Bavail))
		availableSpace := uint64(stat.Bsize) * uint64(stat.Bavail)
		availablePercent := float64(100.0 * float64(stat.Bavail) / float64(stat.Blocks-stat.Bfree+stat.Bavail))
		usedSpace := uint64(stat.Bsize) * (stat.Blocks - stat.Bfree)
		usedPercent := float64(100.0) - float64(100.0*float64(stat.Bavail)/float64(stat.Blocks-stat.Bfree+stat.Bavail))
		blksize := int32(stat.Bsize)
		inodeTotal := int64(stat.Files)
		inodeUsed := int64(stat.Files - stat.Ffree)
		inodeUsedPercent := float32(100.0) * float32(inodeUsed) / float32(inodeTotal)
		major, _ := strconv.ParseInt(majorminor[0], 10, 32)
		minor, _ := strconv.ParseInt(majorminor[1], 10, 32)
		diskPerf := whatap_model.NodeDiskPerf{
			Device:           device,
			DeviceID:         deviceID,
			MountPoint:       mountPoint,
			FileSystem:       filesystem,
			Major:            int(major),
			Minor:            int(minor),
			Type:             filesystem,
			Capacity:         int64(totalSpace),
			Free:             int64(freeSpace),
			Available:        int64(availableSpace),
			AvailablePercent: float32(availablePercent),
			UsedSpace:        int64(usedSpace),
			FreePercent:      float32(freePercent),
			UsedPercent:      float32(usedPercent),
			Blksize:          blksize,
			InodeTotal:       inodeTotal,
			InodeUsed:        inodeUsed,
			InodeUsedPercent: inodeUsedPercent,
		}
		diskPerfList = append(diskPerfList, diskPerf)
	})

	if errthistime != nil {
		err = errthistime
		logutil.Debugf("parseDisk", "err=%v\n", err)
		return
	}
	logutil.Debugln("parseDisk", "start")
	nativeDiskPerfs, err := parseDiskIO(diskPerfList)

	// NaN 및 Inf 값 정리
	for i := range nativeDiskPerfs {
		sanitizeNodeDiskPerf(&nativeDiskPerfs[i])
	}
	if err == nil {
		diskPerfsJson, errthistime := json.Marshal(nativeDiskPerfs)
		if errthistime != nil {
			err = errthistime
			logutil.Debugf("ParseDiskIO-jsonMarshal", "error=%v\n", err)
			return nil, err
		}
		logutil.Debugln("parseDisk", "ok")
		ret = diskPerfsJson
		return ret, nil
	}
	logutil.Debugf("parseDisk", "error=%v\n", err)
	return nil, err
}
func parseDiskIO(diskPerfs []whatap_model.NodeDiskPerf) ([]whatap_model.NodeDiskPerf, error) {
	nativeDiskPerfs, err := whatap_osinfo.ParseNativeDiskIO(diskPerfs)
	if err != nil {
		return diskPerfs, err
	}
	return nativeDiskPerfs, nil
}

var initialMountPointsCache [][]string

func populateFileValues(prefix string, filename string, callback func(tokens []string)) (reterr error) {

	//UseCachedMountPointEnabled 사용하면서 nil 이 아닌경우 mountPoint 값을 캐시에서 가져온다
	if whatap_config.GetConfig().UseCachedMountPointEnabled && len(initialMountPointsCache) > 0 {
		for _, mountinfoCache := range initialMountPointsCache {
			callback(mountinfoCache)
		}
		return
	}

	//cached 사용하지 않을 경우 or flag 자체가 false 일 경우 파일에서 값을 가져온다
	calculated_path := filepath.Join(prefix, filename)
	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
		return
	}
	defer func(f *os.File) {
		errPop := f.Close()
		if errPop != nil {
			logutil.Infoln("getDisUsageHandler", "err=%v\n", errPop)
		}
	}(f)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 0 {
			callback(words)
			if whatap_config.GetConfig().UseCachedMountPointEnabled {
				initialMountPointsCache = append(initialMountPointsCache, words)
			}
		}
	}
	return
}
func sanitizeNodeDiskPerf(ndp *whatap_model.NodeDiskPerf) {
	ndp.AvailablePercent = sanitizeFloat32(ndp.AvailablePercent, "AvailablePercent")
	ndp.FreePercent = sanitizeFloat32(ndp.FreePercent, "FreePercent")
	ndp.UsedPercent = sanitizeFloat32(ndp.UsedPercent, "UsedPercent")
	ndp.IOPercent = sanitizeFloat32(ndp.IOPercent, "IOPercent")
	ndp.QueueLength = sanitizeFloat32(ndp.QueueLength, "QueueLength")
	ndp.InodeUsedPercent = sanitizeFloat32(ndp.InodeUsedPercent, "InodeUsedPercent")
	ndp.ReadBps = sanitizeFloat64(ndp.ReadBps, "ReadBps")
	ndp.ReadIops = sanitizeFloat64(ndp.ReadIops, "ReadIops")
	ndp.WriteBps = sanitizeFloat64(ndp.WriteBps, "WriteBps")
	ndp.WriteIops = sanitizeFloat64(ndp.WriteIops, "WriteIops")
}
func sanitizeFloat32(value float32, name string) float32 {
	if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		if whatap_config.GetConfig().Debug {
			logutil.Debugf("SF32", "%v=%v\n", name, value)
		}
		return 0
	}
	return value
}
func sanitizeFloat64(value float64, name string) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		if whatap_config.GetConfig().Debug {
			logutil.Debugf("SF64", "%v=%v\n", name, value)
		}
		return 0
	}
	return value
}
