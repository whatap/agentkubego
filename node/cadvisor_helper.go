package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/gorilla/mux"
	discovery_kube "github.com/whatap/kube/node/discovery/kube"
	"github.com/whatap/kube/node/src/whatap/client"
	whatap_config "github.com/whatap/kube/node/src/whatap/config"
	whatap_model "github.com/whatap/kube/node/src/whatap/lang/model"
	whatap_osinfo "github.com/whatap/kube/node/src/whatap/osinfo"
	whatap_cgroup "github.com/whatap/kube/node/src/whatap/util/cgroup"
	whatap_containerd "github.com/whatap/kube/node/src/whatap/util/containerd"
	whatap_crio "github.com/whatap/kube/node/src/whatap/util/crio"
	whatap_docker "github.com/whatap/kube/node/src/whatap/util/docker"
	whatap_iputil "github.com/whatap/kube/node/src/whatap/util/iputil"
	"github.com/whatap/kube/node/src/whatap/util/logutil"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8srest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"math"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	//CONN_EXPIRE CONN_EXPIRE
	CONN_EXPIRE = int64(30 * 60)
	//INIT_POOL_SIZE INIT_POOL_SIZE
	INIT_POOL_SIZE = 5
	//MAX_POOL_SIZE MAX_POOL_SIZE
	MAX_POOL_SIZE      = 5
	stdWriterPrefixLen = 8
	stdWriterSizeIndex = 4
)

var (
	HOSTPATH_PREFIX string
	PATH_SYS_BLOCK  string
	mu              = sync.Mutex{}
	DEBUG           = os.Getenv("DEBUG") == "true"
)

type h0 func()

func dumpGoroutine() {

	f, _ := os.OpenFile("/whatap_conf/logs/cadvisor_helper.golang.dump.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	for {
		// if connectionPool != nil{
		// f.Write([]byte("\n"))
		// f.Write([]byte(time.Now().Format("2006-01-02 15:04:05")))
		// f.Write([]byte(fmt.Sprint("PoolSize:", connectionPool.Len(), "/",MAX_POOL_SIZE )))
		// }
		f.Write([]byte("\n"))
		f.Write([]byte(time.Now().Format("2006-01-02 15:04:05")))
		p := pprof.Lookup("goroutine")
		p.WriteTo(f, 1)
		time.Sleep(1 * time.Second)
	}
}

func printContainerEnv(containerID string) error {
	container, ctx, err := whatap_containerd.LoadContainerD(containerID)
	if err != nil {
		return err
	}

	spec, err := container.Spec(ctx)
	if err != nil {
		return err
	}

	processSpec := spec.Process
	fmt.Printf("Environment Variables for Container %s:\n", containerID)
	for _, env := range processSpec.Env {
		fmt.Println(env)
	}
	return nil
}

func runTestContainerInspect(containerId string) {
	printEnvErr := printContainerEnv(containerId)
	logutil.Infof("runTestContainerInspect", "env=%v", printEnvErr)

	resp, ctx, err := whatap_containerd.LoadContainerD(containerId)
	if err != nil {
		logutil.Infof("runTestContainerInspect", "loadErr=%v", err)
		return
	}
	spec, err := resp.Spec(ctx)
	if err != nil {
		logutil.Infof("runTestContainerInspect", "specErr=%v", err)
		return
	}
	specJSON, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		logutil.Infof("runTestContainerInspect", "jsonMarshalErr=%v", err)
		return
	}
	logutil.Infof("runTestContainerInspect", "spec=%s", string(specJSON))
	pid, _ := whatap_osinfo.GetContainerPid(containerId)
	logutil.Infof("runTestContainerInspect", "pid=%v", pid)
	//task, err := resp.Task(ctx, nil)
	if err != nil {
		logutil.Infof("runTestContainerInspect", "TaskErr=%v", err)
	}
	//name := ""
	//restartCount := 0

	//memoryLimit := int64(0)

	//cgroupParent := spec.Linux.CgroupsPath
	//var statserr error
	//content, statserr := whatap_cgroup.GetContainerStatsEx(HOSTPATH_PREFIX, containerId, name, cgroupParent,
	//	restartCount, pid, memoryLimit)
	//if statserr != nil {
	//	logutil.Infof("runTestContainerInspect", "statsErr=%v", statserr)
	//}
	//logutil.Infof("runTestContainerInspect", "content=%v", content)
}

var whatapConfig *whatap_config.Config

func init() {
	printWhatap := fmt.Sprint("\n" +
		" _      ____       ______WHATAP-KUBER-AGENT\n" +
		"| | /| / / /  ___ /_  __/__ ____\n" +
		"| |/ |/ / _ \\/ _ `// / / _ `/ _ \\\n" +
		"|__/|__/_//_/\\_,_//_/  \\_,_/ .__/\n" +
		"                          /_/\n" +
		"Just Tap, Always Monitoring\n")
	fmt.Print(printWhatap)
	whatapConfig = whatap_config.GetConfig()
	HOSTPATH_PREFIX = whatapConfig.HostPathPrefix
	PATH_SYS_BLOCK = whatapConfig.PathSysBlock

	// 구성 정보 출력
	fmt.Printf("-DEBUG: %v\n", whatapConfig.Debug)
	fmt.Printf("-HostPathPrefix: %v\n", whatapConfig.HostPathPrefix)
	fmt.Printf("-KubeConfigPath: %v\n", whatapConfig.KubeConfigPath)
	fmt.Printf("-KubeMasterUrl: %v\n", whatapConfig.KubeMasterUrl)
	fmt.Printf("-MasterAgentHost: %v\n", whatapConfig.MasterAgentHost)
	fmt.Printf("-MasterAgentPort: %v\n", whatapConfig.MasterAgentPort)
	fmt.Printf("-ConfBaseAgentHost: %v\n", whatapConfig.ConfBaseAgentHost)
	fmt.Printf("-ConfBaseAgentPort: %v\n", whatapConfig.ConfBaseAgentPort)
	fmt.Printf("-VERSION: %v\n", whatapConfig.Version)
	fmt.Printf("-PORT: %v\n", whatapConfig.Port)
	fmt.Printf("-CYCLE: %vs\n", whatapConfig.Cycle)
	fmt.Printf("-ConfFilePath: %v\n", whatap_config.GetConfFilePath())
	fmt.Printf("-Test: %v\n", whatapConfig.Test)
	fmt.Printf("-TestContainerId: %v\n", whatapConfig.TestContainerId)
	fmt.Printf("-LogSysOut: %v\n", whatapConfig.LogSysOut)
	fmt.Printf("-InjectContainerIdToApmAgentEnabled: %v\n", whatapConfig.InjectContainerIdToApmAgentEnabled)
	fmt.Printf("-UseCachedMountPointEnabled: %v\n", whatapConfig.UseCachedMountPointEnabled)
	fmt.Printf("-CollectVolumeDetailEnabled: %v\n", whatapConfig.CollectVolumeDetailEnabled)
	fmt.Printf("-CollectNfsDiskEnabled: %v\n", whatapConfig.CollectNfsDiskEnabled)
	fmt.Printf("-CollectProcessIO: %v\n", whatapConfig.CollectProcessIO)
	fmt.Printf("-CollectProcessFD: %v\n", whatapConfig.CollectProcessFD)
	fmt.Printf("-CollectKubeNodeProcessMetricEnabled: %v\n", whatapConfig.CollectKubeNodeProcessMetricEnabled)
	fmt.Printf("-CollectKubeNodeProcessMetricTargetList: %v\n", whatapConfig.CollectKubeNodeProcessMetricTargetList)
	fmt.Printf("-InspectWhatapAgentPathFromProc: %v\n", whatapConfig.InspectWhatapAgentPathFromProc)
	fmt.Printf("-DefaultJavaAgentPath: %v\n", whatapConfig.WhatapJavaAgentPath)
	fmt.Println("===========================================================================================")
}

func main() {
	// staticConfig
	//if whatapConfig.Test == true {
	//	go func() {
	//		for {
	//			// whatapConfig.WhatapLogger 는 conf 에 등록되어있는 debug 가 true 경우에만 파일에 출력된다. 동적으로 debug 설정 변경시 conf apply 완료전까지 출력될 수 있음.
	//			logutil.Infoln("TEST", "INFO")
	//			logutil.Debugln("TEST", "DEBUG")
	//			if whatapConfig.TestContainerId != "" {
	//				runTestContainerInspect()
	//			}
	//			time.Sleep(time.Duration(5) * time.Second)
	//		}
	//	}()
	//}
	portFlag := flag.Int("port", 6801, "string, whatap-node-helper port")
	testTypeFlag := flag.String("t", "", "string , image - get agent path, spec - spec")
	containerID := flag.String("id", "", "string , id - agent container id")
	flag.Parse()
	if *testTypeFlag == "image" {
		path, err := discovery_kube.InspectWhatapAgentPath(*containerID)
		logutil.Infof("imageFlag", "agentPath=%v\n", path)
		if err != nil {
			logutil.Infof("imageFlag", "agentPath=%v,err=%w\n", path, err)
		}
		os.Exit(0)
	} else if *testTypeFlag == "spec" {
		runTestContainerInspect(*containerID)
		os.Exit(0)
	}

	logutil.Infof("run cadvisor", "port=%v\n", *portFlag)
	discovery_kube.RunPodInformer()
	//go scraper_kube.FindAllK8sProcesses()
	//go dumpGoroutine()
	r := mux.NewRouter()
	r.HandleFunc("/health", healthHandler)
	r.HandleFunc("/debug/goroutine", debugGoroutineHandler)
	if whatap_docker.CheckDockerEnabled() {
		r.HandleFunc("/container/{containerid}", containerInspectHandler)

		r.HandleFunc("/container/{containerid}/stats", containerStatsHandler)

		r.HandleFunc("/container_events", eventHandler).
			Queries("since", "{since}").
			Queries("until", "{until}").
			Queries("containerid", "{containerid}")
		r.HandleFunc("/host", hostHandler)
	} else if whatap_containerd.CheckContainerdEnabled() {
		r.HandleFunc("/container/{containerid}/stats", containerStatsContainerdHandler)
		r.HandleFunc("/container/{containerid}", containerInspectContainerdHandler)
	} else if whatap_crio.CheckCrioEnabled() {
		r.HandleFunc("/container/{containerid}", containerInspectCrioHandler)
		r.HandleFunc("/container/{containerid}/stats", containerStatsCrioHandler)
	}
	r.HandleFunc("/container", allContainerHandler)

	r.HandleFunc("/container/{containerid}/logs", containerLogHandler).
		Queries("stdout", "{stdout}").
		Queries("stderr", "{stderr}").
		Queries("since", "{since}").
		Queries("until", "{until}").
		Queries("timestamps", "{timestamps}").
		Queries("tail", "{tail}")

	r.HandleFunc("/container/{containerid}/volumes", containerVolumeHandler)
	r.HandleFunc("/container/{containerid}/netstat", containerNetstatHandler)

	r.HandleFunc("/host/disks", hostDiskHandler)
	r.HandleFunc("/host/processes", hostProcessHandler)

	// fmt.Println(time.Now(), "trying to listen", fmt.Sprintf(":%d", *port))
	// loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, r)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", whatapConfig.Port),
		Handler: r,
	}
	err := srv.ListenAndServe()
	if err != nil {
		logutil.Errorln("ListenAndServe", err)
		return
	}
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	contentType := "applicatin/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte("{\"OK\": 1}"))
}

func debugGoroutineHandler(w http.ResponseWriter, req *http.Request) {
	contentType := "text/plain"
	w.Header().Add("Content-Type", contentType)
	p := pprof.Lookup("goroutine")
	p.WriteTo(w, 1)
}

type containerNetworkStats struct {
	RxBytes   int64 `json:"rxBytes"`
	RxDropped int64 `json:"rxDropped"`
	RxErrors  int64 `json:"rxErrors"`
	RxPackets int64 `json:"rxPackets"`
	TxBytes   int64 `json:"txBytes"`
	TxDropped int64 `json:"txDropped"`
	TxErrors  int64 `json:"txErrors"`
	TxPackets int64 `json:"txPackets"`
}

var (
	containerRestartLookup      = map[string]int{}
	containerNameLookup         = map[string]string{}
	containerRestartLookupMutex = sync.RWMutex{}
	restartCacheLock            = sync.Mutex{}
	lastCacheUpdate             int64
)

func getContainerRestartCount(containerId string) (int, string, error) {
	restartCacheLock.Lock()
	defer restartCacheLock.Unlock()
	now := time.Now().Unix()
	containerRestartLookupMutex.RLock()
	restartCnt, ok := containerRestartLookup[containerId]
	containerRestartLookupMutex.RUnlock()
	if ok && now-lastCacheUpdate < 60 {
		return restartCnt, containerNameLookup[containerId], nil
	}
	cli, err := client.GetKubernetesClient()
	if err != nil {
		return 0, "", err
	}

	nodename := os.Getenv("NODE_NAME")
	listOptions := metav1.ListOptions{FieldSelector: fmt.Sprint("spec.nodeName=", nodename)}
	pods, err := cli.CoreV1().Pods("").List(context.Background(), listOptions)
	if err != nil {
		return 0, "", err
	}
	for _, pod := range pods.Items {
		for _, c := range pod.Status.ContainerStatuses {
			cid := c.ContainerID
			if strings.Contains(cid, "//") {
				cid = strings.Split(cid, "//")[1]
			}
			containerRestartLookupMutex.Lock()
			containerRestartLookup[cid] = int(c.RestartCount)
			containerNameLookup[cid] = c.Name
			containerRestartLookupMutex.Unlock()
		}
	}
	lastCacheUpdate = now
	containerRestartLookupMutex.RLock()
	restartCnt, ok = containerRestartLookup[containerId]
	containerRestartLookupMutex.RUnlock()
	if ok {
		return restartCnt, containerNameLookup[containerId], nil
	} else {
		return 0, "", fmt.Errorf("container ", containerId, " not found")
	}
}

func getContainerNetworkStats(containerId string) (containerNetworkStats, error) {
	var totalNetStats = containerNetworkStats{}
	pid, err := whatap_osinfo.GetContainerPid(containerId)
	if err != nil {
		return totalNetStats, err
	}

	netdev := filepath.Join(HOSTPATH_PREFIX, "proc", fmt.Sprint(pid), "net", "dev")
	f, err := os.Open(netdev)
	if err != nil {
		return totalNetStats, err
	}
	j := 0
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		j++
		if j < 3 {
			continue
		}
		words := strings.Fields(strings.Replace(line, ":", " ", -1))
		deviceId := words[0]

		if deviceId == "lo" {
			continue
		}
		readByteCount, _ := strconv.ParseInt(words[1], 10, 64)
		readCount, _ := strconv.ParseInt(words[2], 10, 64)
		readErrorCount, _ := strconv.ParseInt(words[3], 10, 64)
		readDroppedCount, _ := strconv.ParseInt(words[4], 10, 64)

		writeByteCount, _ := strconv.ParseInt(words[9], 10, 64)
		writeCount, _ := strconv.ParseInt(words[10], 10, 64)
		writeErrorCount, _ := strconv.ParseInt(words[11], 10, 64)
		writeDroppedCount, _ := strconv.ParseInt(words[12], 10, 64)

		totalNetStats.RxBytes += readByteCount
		totalNetStats.RxPackets += readCount
		totalNetStats.RxErrors += readErrorCount
		totalNetStats.RxDropped += readDroppedCount

		totalNetStats.TxBytes += writeByteCount
		totalNetStats.TxPackets += writeCount
		totalNetStats.TxErrors += writeErrorCount
		totalNetStats.TxDropped += writeDroppedCount

	}
	return totalNetStats, nil
}

func getContainerStats(containerId string) (string, error) {
	cli, err := whatap_docker.GetDockerClient()
	if err != nil {
		return "", err
	}
	//defer pc.Release()
	// cli := pc.Conn
	statsresp, err := cli.ContainerStats(context.Background(), containerId, false)
	if err == nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(statsresp.Body)
		statsjson := buf.String()

		netstats, err := getContainerNetworkStats(containerId)
		if err == nil {
			netstatJson, err := json.Marshal(netstats)
			if err == nil {
				restartCount, _, err := getContainerRestartCount(containerId)
				if err != nil {
					return "", err
				}

				statsjson = fmt.Sprint(strings.TrimSuffix(strings.TrimSpace(statsjson), "}"), ", \"network_stats\":", string(netstatJson), ", \"restart_count\" : ", restartCount, " }")
			}
		}

		return statsjson, nil
	}

	return "", err
}

func getContainerStatsCrio(containerId string) (statsjson string, statserr error) {
	restartCount, _, err := getContainerRestartCount(containerId)
	if err != nil {
		return "", err
	}

	err = whatap_crio.GetContainerParams(HOSTPATH_PREFIX, containerId,
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

func getContainerInspect(containerId string) (string, error) {
	cli, err := whatap_docker.GetDockerClient()
	if err != nil {
		return "", err
	}
	// defer pc.Release()
	// cli := pc.Conn
	resp, err := cli.ContainerInspect(context.Background(), containerId)
	if err == nil {
		contJson, err := json.Marshal(resp)
		if err == nil {
			return string(contJson), nil
		}
	}

	return "", err
}

func executeCommand(containerId string, cmds []string, omitFirstLine bool, h1 func(string)) error {
	cli, err := whatap_docker.GetDockerClient()
	if err != nil {
		return err
	}
	// defer pc.Release()
	// cli := pc.Conn
	options := types.ExecConfig{AttachStdout: true, Detach: false, Tty: false}

	for _, cmd := range cmds {
		options.Cmd = append(options.Cmd, cmd)
	}
	ctx := context.Background()
	respid, err := cli.ContainerExecCreate(ctx, containerId, options)
	if err != nil {
		return err
	}
	execconfig := types.ExecStartCheck{Tty: true}
	hijackresp, err := cli.ContainerExecAttach(ctx, respid.ID, execconfig)
	if err == nil {
		err = cli.ContainerExecStart(ctx, respid.ID, execconfig)
		if err == nil {
			for {
				inspect, err := cli.ContainerExecInspect(ctx, respid.ID)
				if err != nil || !inspect.Running {
					break
				}
			}
			scanner := bufio.NewScanner(hijackresp.Reader)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				line := scanner.Text()
				h1(line)
				//lines = append(lines, &line)
			}
			return nil
		}
	}

	return err
}

type mountInfo struct {
	Source    string
	Target    string
	MountType string `json:",omitempty"`
	Driver    string `json:",omitempty"`
}
type volumeinfo map[string]mountInfo

func getVolumeInfo(containerid string) (*volumeinfo, error) {
	cli, err := whatap_docker.GetDockerClient()
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

func getVolumeInfoEx(containerid string) (*volumeinfo, error) {
	resp, ctx, err := whatap_containerd.LoadContainerD(containerid)
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

/*
config=<string> config name or ID
container=<string> container name or ID
daemon=<string> daemon name or ID
event=<string> event type
image=<string> image name or ID
label=<string> image or container label
network=<string> network name or ID
node=<string> node ID
plugin= plugin name or ID
scope= local or swarm
secret=<string> secret name or ID
service=<string> service name or ID
type=<string> object to filter by, one of container, image, volume, network, daemon, plugin, node, service, secret or config
volume=<string> volume name
*/
func getEvents(containerid string, since string, until string) (string, error) {
	cli, err := whatap_docker.GetDockerClient()
	if err != nil {
		return "", err
	}
	// defer pc.Release()
	// cli := pc.Conn
	options := types.EventsOptions{}

	if len(since) > 0 {
		options.Since = since
	}

	if len(until) > 0 {
		options.Until = until
	} else {
		options.Until = "0s"
	}

	if len(containerid) > 0 {
		options.Filters = filters.NewArgs(filters.KeyValuePair{Key: "container", Value: containerid})
	}

	var eventList []events.Message
	resultCh, errorCh := cli.Events(context.Background(), options)
	for {
		select {
		case resp := <-resultCh:
			eventList = append(eventList, resp)
		case err = <-errorCh:
			contJson, err := json.Marshal(eventList)
			if err == nil {
				return string(contJson), nil
			} else {
				return "", err
			}
		}
	}
}

func getSystemInfo() (string, error) {
	cli, err := whatap_docker.GetDockerClient()
	if err != nil {
		return "", err
	}
	// defer pc.Release()
	// cli := pc.Conn
	resp, err := cli.Info(context.Background())
	if err == nil {
		contJson, err := json.Marshal(resp)
		if err == nil {
			return string(contJson), nil
		}
	}

	return "", err
}

func containerInspectHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	containerId := vars["containerid"]
	if len(containerId) < 1 {
		w.WriteHeader(404)
		w.Write([]byte("containerid missing"))
		return
	}
	var content string
	var err error
	if whatap_docker.CheckDockerEnabled() {
		content, err = getContainerInspect(containerId)
	} else {
		content, err = whatap_docker.GetContainerInspect(HOSTPATH_PREFIX, containerId)
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

func containerInspectCrioHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	containerId := vars["containerid"]
	if len(containerId) < 1 {
		w.WriteHeader(404)
		w.Write([]byte("containerid missing"))
		return
	}
	content, err := whatap_crio.GetContainerInspect(HOSTPATH_PREFIX, containerId)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(content))
}

func containerStatsCrioHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	containerId := vars["containerid"]
	if len(containerId) < 1 {
		w.WriteHeader(404)
		w.Write([]byte("containerid missing"))
		return
	}
	// fmt.Println("containerStatsCrioHandler step -1")

	content, err := getContainerStatsCrio(containerId)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contentType := "applicatin/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(content))
}

func containerStatsHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	containerId := vars["containerid"]
	if len(containerId) < 1 {
		w.WriteHeader(404)
		w.Write([]byte("containerid missing"))
		return
	}
	var content string
	err := whatap_docker.GetContainerParams(HOSTPATH_PREFIX, containerId, func(name string, cgroupParent string,
		restartCount int, pid int, memoryLimit int64) error {

		restartCount, _, _ = getContainerRestartCount(containerId)

		var statserr error
		content, statserr = whatap_docker.GetContainerStatsEx(HOSTPATH_PREFIX, containerId, name, cgroupParent,
			restartCount, pid, memoryLimit)
		return statserr
	})

	if err != nil {
		content, err = getContainerStats(containerId)
	}

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(content))
}

func containerLogHandler(w http.ResponseWriter, req *http.Request) {
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

func eventHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	since := vars["since"]
	until := vars["until"]
	containerid := vars["containerid"]
	content, err := getEvents(containerid, since, until)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(content))
}

type VolumeUsage struct {
	Source      string
	Driver      string `json:",omitempty"`
	Mount       string
	MountType   string
	Total       int64
	Used        int64
	UsedPercent float64
}

func containerVolumeHandler(w http.ResponseWriter, req *http.Request) {
	if !whatapConfig.CollectVolumeDetailEnabled {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf("CollectVolumeDetailEnabled=%v", whatapConfig.CollectVolumeDetailEnabled)))
		return
	}
	vars := mux.Vars(req)
	containerid := vars["containerid"]
	var err error
	var volumeLookup *volumeinfo
	if whatap_docker.CheckDockerEnabled() {
		volumeLookup, err = getVolumeInfo(containerid)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
	} else if whatap_containerd.CheckContainerdEnabled() {
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
	var volumes []VolumeUsage

	err = executeCommandEx(containerid, []string{"df"}, true, func(line string) {
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
		volume := VolumeUsage{
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

func containerNetstatHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var netstat Netstat
	containerid := vars["containerid"]

	if whatap_docker.CheckDockerEnabled() {
		err := executeCommand(containerid, []string{"ss", "-at"}, true,
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
	} else if whatap_containerd.CheckContainerdEnabled() {
		err := executeCommandEx(containerid, []string{"ss", "-at"}, true,
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
	contentType := "applicaton/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(contJson))
}

func hostHandler(w http.ResponseWriter, req *http.Request) {
	sysJson, err := getSystemInfo()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(sysJson))
}

func allContainerHandler(w http.ResponseWriter, req *http.Request) {
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

type ContainerSummary struct {
	Image        interface{}
	ImageID      string
	RestartCount int32
	Status       string
	State        string
	Names        []string
	Id           string
	Podname      string
	//not implemented
	Command       string
	Created       int64
	Labels        map[string]string
	CpuLimit      interface{}
	MemoryLimit   interface{}
	CpuRequest    interface{}
	MemoryRequest interface{}
	Namespace     string
	Ready         int32
}

var re = regexp.MustCompile("([0-9\\.]+s)$")

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

	var csums []ContainerSummary
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

			csums = append(csums, ContainerSummary{
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

func containerStatsContainerdHandler(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)
	containerId := vars["containerid"]
	if len(containerId) < 1 {
		w.WriteHeader(404)
		w.Write([]byte("containerid missing"))
		return
	}
	content, err := getContainerStatsContainerd(containerId)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(content))
}

type ContainerdSandboxMetadata struct {
	Version  string `json:"Version"`
	Metadata struct {
		ID        string `json:"ID"`
		Name      string `json:"Name"`
		SandboxID string `json:"SandboxID"`
		Config    struct {
			Metadata map[string]interface{} `json:"metadata"`
			Image    map[string]string      `json:"image"`
			Command  []string               `json:"command"`
			Envs     []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"envs"`
			Mounts []struct {
				ContainerPath string `json:"container_path"`
				HostPath      string `json:"host_path"`
				Readonly      bool   `json:"readonly,omitempty"`
			} `json:"mounts"`
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			LogPath     string            `json:"log_path"`
			Linux       struct {
				Resources struct {
					CPUPeriod          int `json:"cpu_period"`
					CPUQuota           int `json:"cpu_quota"`
					CPUShares          int `json:"cpu_shares"`
					MemoryLimitInBytes int `json:"memory_limit_in_bytes"`
					OomScoreAdj        int `json:"oom_score_adj"`
				} `json:"resources"`
				SecurityContext struct {
					NamespaceOptions struct {
						Pid int `json:"pid"`
					} `json:"namespace_options"`
					RunAsUser struct {
					} `json:"run_as_user"`
					MaskedPaths   []string `json:"masked_paths"`
					ReadonlyPaths []string `json:"readonly_paths"`
				} `json:"security_context"`
			} `json:"linux"`
		} `json:"Config"`
		ImageRef   string      `json:"ImageRef"`
		LogPath    string      `json:"LogPath"`
		StopSignal interface{} `json:"StopSignal"`
	} `json:"Metadata"`
}

func getContainerStatusContainerD(c containerd.Container, ctx context.Context, h1 func(containerd.Status)) {
	task, errtask := c.Task(ctx, nil)
	if errtask != nil {
		return
	}
	status, errstatus := task.Status(ctx)
	if errstatus != nil {
		return
	}

	h1(status)

	return
}

func getContainerInspectContainerd(containerId string) (cinfo whatap_model.ContainerInfo, err error) {
	cinfo.ID = containerId
	restartCount, _, errrst := getContainerRestartCount(containerId)
	if errrst != nil {
		err = errrst
		return
	}
	cinfo.RestartCount = restartCount

	container, ctx, errld := whatap_containerd.LoadContainerD(containerId)
	if errld != nil {
		err = errld
		return
	}

	getContainerStatusContainerD(container, ctx, func(status containerd.Status) {
		switch status.Status {
		case containerd.Running:
			cinfo.State.Status = "running"
			cinfo.State.Running = true
		case containerd.Paused:
			cinfo.State.Status = "paused"
			cinfo.State.Paused = true
		case containerd.Pausing:
			cinfo.State.Status = "pausing"
			cinfo.State.Paused = true
		case containerd.Stopped:
			cinfo.State.Status = "stopped"
			cinfo.State.Dead = true
		case containerd.Created:
			cinfo.State.Status = "created"
			cinfo.State.Running = true
		default:
			cinfo.State.Status = "unknown"
			cinfo.State.Dead = true
		}
	})

	cexts, errext := container.Extensions(ctx)
	if errext != nil {
		err = errext
		return
	}
	for _, v := range cexts {
		if "github.com/containerd/cri/pkg/store/container/Metadata" == v.GetTypeUrl() {

			csm := ContainerdSandboxMetadata{}
			errmarshal := json.Unmarshal(v.GetValue(), &csm)
			if errmarshal != nil {
				if whatap_config.GetConfig().Debug {
					fmt.Println("getContainerInspectContainerd error:%s json:%s", errmarshal.Error(), string(v.GetValue()))
				}
				return
			}

			if whatap_config.GetConfig().Debug {
				fmt.Println("getContainerInspectContainerd json:%s", string(v.GetValue()))
			}

			cinfo.LogPath = csm.Metadata.LogPath
			if len(cinfo.LogPath) < 1 {
				err = fmt.Errorf("Container %s Extension not found cexts:%s", containerId, string(v.GetValue()))
			}
			for _, m := range csm.Metadata.Config.Mounts {
				cinfo.Mounts = append(cinfo.Mounts, whatap_model.Mount{
					Source:      m.HostPath,
					Destination: m.ContainerPath,
					RW:          !m.Readonly,
				})
			}

		}
	}

	return
}

func containerInspectContainerdHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	containerId := vars["containerid"]
	if len(containerId) < 1 {
		w.WriteHeader(404)
		w.Write([]byte("containerid missing"))
		return
	}

	cinfo, err := getContainerInspectContainerd(containerId)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	cinfojson, err := json.Marshal(cinfo)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	contentType := "application/json"
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(cinfojson))
}

type ThrottlingData struct {
	Periods          uint64 `json:"periods"`
	ThrottledPeriods uint64 `json:"throttled_periods"`
	ThrottledTime    uint64 `json:"throttled_time"`
}
type CpuUsage struct {
	TotalUsage      uint64   `json:"total_usage"`
	PercpuUsage     []uint64 `json:"percpu_usage"`
	UsageKernelmode uint64   `json:"usage_in_kernelmode"`
	UsageUsermode   uint64   `json:"usage_in_usermode"`
}
type CpuStats struct {
	CpuUsage       CpuUsage       `json:"cpu_usage"`
	SystemCpuUsage uint64         `json:"system_cpu_usage"`
	ThrottlingData ThrottlingData `json:"throttling_data"`
}

type Stats struct {
	ActiveAnon        int64 `json:"active_anon"`
	ActiveFile        int64 `json:"active_file"`
	Cache             int64 `json:"cache"`
	InactiveAnon      int64 `json:"inactive_anon"`
	InactiveFile      int64 `json:"inactive_file"`
	MappedFile        int64 `json:"mapped_file"`
	Pgfault           int64 `json:"pgfault"`
	Pgmajfault        int64 `json:"pgmajfault"`
	Pgpgin            int64 `json:"pgpgin"`
	Pgpgout           int64 `json:"pgpgout"`
	Rss               int64 `json:"rss"`
	TotalActiveAnon   int64 `json:"total_active_anon"`
	TotalActiveFile   int64 `json:"total_active_file"`
	TotalCache        int64 `json:"total_cache"`
	TotalInactiveAnon int64 `json:"total_inactive_anon"`
	TotalInactiveFile int64 `json:"total_inactive_file"`
	TotalMappedFile   int64 `json:"total_mapped_file"`
	TotalPgfault      int64 `json:"total_pgfault"`
	TotalPgmajfault   int64 `json:"total_pgmajfault"`
	TotalPgpgin       int64 `json:"total_pgpgin"`
	TotalPgpgout      int64 `json:"total_pgpgout"`
	TotalRss          int64 `json:"total_rss"`
	TotalUnevictable  int64 `json:"total_unevictable"`
	TotalWriteback    int64 `json:"total_writeback"`
	Unevictable       int64 `json:"unevictable"`
	Writeback         int64 `json:"writeback"`
}

type MemoryStats struct {
	Usage    uint64 `json:"usage"`
	MaxUsage uint64 `json:"max_usage"`
	Stats    Stats  `json:"stats"`
	Limit    uint64 `json:"limit"`
	FailCnt  uint64 `json:"failcnt"`
}

type Dockermetric struct {
	BlkioStats  interface{} `json:"blkio_stats"`
	CpuStats    CpuStats    `json:"cpu_stats"`
	PrecpuStats CpuStats    `json:"precpu_stats"`
	MemoryStats MemoryStats `json:"memory_stats"`
}

const nanoSecondsPerSecond = 1e9

func systemHostCpuUsage() (uint64, error) {
	filepath := filepath.Join(HOSTPATH_PREFIX, "proc", "stat")
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scClkTck := 100
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if words[0] == "cpu" {
			user, _ := strconv.ParseInt(words[1], 10, 64)
			nice, _ := strconv.ParseInt(words[2], 10, 64)
			system, _ := strconv.ParseInt(words[3], 10, 64)
			idle, _ := strconv.ParseInt(words[4], 10, 64)
			iowait, _ := strconv.ParseInt(words[5], 10, 64)
			irq, _ := strconv.ParseInt(words[6], 10, 64)
			softirq, _ := strconv.ParseInt(words[7], 10, 64)
			steal, _ := strconv.ParseInt(words[8], 10, 64)

			return uint64(nanoSecondsPerSecond *
				(user + nice + system + idle + iowait + irq + softirq + steal) /
				int64(scClkTck)), nil
		}
	}

	return 0, fmt.Errorf("cpu not found")
}

func getContainerStatsContainerd(containerId string) (string, error) {
	resp, ctx, err := whatap_containerd.LoadContainerD(containerId)
	if err != nil {
		return "", err
	}

	spec, err := resp.Spec(ctx)
	if err != nil {
		return "", err
	}

	restartCount, name, _ := getContainerRestartCount(containerId)
	pid, _ := whatap_osinfo.GetContainerPid(containerId)
	memoryLimit := int64(0)

	cgroupParent := spec.Linux.CgroupsPath

	var statserr error
	statsjson, statserr := whatap_cgroup.GetContainerStatsEx(HOSTPATH_PREFIX, containerId, name, cgroupParent,
		restartCount, pid, memoryLimit)

	if statserr == nil {

		netstats, err := getContainerNetworkStats(containerId)
		if err == nil {
			netstatJson, err := json.Marshal(netstats)
			if err == nil {

				statsjson = fmt.Sprint(strings.TrimSuffix(strings.TrimSpace(statsjson), "}"), ", \"network_stats\":", string(netstatJson), " }")
			}
		}

		return statsjson, nil
	}

	return "", statserr
}

func executeCommandEx(containerId string, cmds []string, omitFirstLine bool, h1 func(string)) error {
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
	var namespace, podname, fullcontainerid string
	for _, pod := range pods.Items {
		for _, c := range pod.Status.ContainerStatuses {
			containerid := c.ContainerID
			if strings.Contains(containerid, "//") {
				containerid = strings.Split(containerid, "//")[1]
			}
			if containerid == containerId {
				podname = pod.Name
				namespace = pod.Namespace
				fullcontainerid = c.Name
			}
		}
	}

	if len(podname) < 1 {
		return fmt.Errorf("container not found")
	}

	apihost := os.Getenv("KUBERNETES_SERVICE_HOST")
	apiport := os.Getenv("KUBERNETES_SERVICE_PORT")
	if whatap_iputil.IsIPv6(apihost) {
		apihost = fmt.Sprintf("[%s]", apihost)
	}
	urlencoded_cmd := url.QueryEscape(strings.Join(cmds, " "))
	fullurl := fmt.Sprint("https://", apihost, ":", apiport, "/api/v1/namespaces/",
		namespace, "/pods/", podname, "/exec?container=", url.QueryEscape(fullcontainerid),
		"&stdin=0&stdout=1&stderr=1&tty=1&command=", urlencoded_cmd)

	url, err := url.ParseRequestURI(fullurl)
	if err != nil {
		return err
	}
	conf, err := k8srest.InClusterConfig()
	if err != nil {
		return err
	}
	e, err := remotecommand.NewSPDYExecutor(conf, "POST", url)
	if err != nil {
		return err
	}
	localOut := &bytes.Buffer{}
	localErr := &bytes.Buffer{}
	err = e.Stream(remotecommand.StreamOptions{Stdin: nil, Stdout: localOut, Stderr: localErr, Tty: true})

	if err != nil {
		return err
	}
	for {
		l, e := localOut.ReadString(byte('\n'))
		if e == nil {
			h1(l)
		} else {
			break
		}
	}
	for {
		l, e := localErr.ReadString(byte('\n'))
		if e == nil {
			h1(l)
		} else {
			break
		}
	}

	return nil
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

func hostDiskHandler(w http.ResponseWriter, req *http.Request) {
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

func hostProcessHandler(w http.ResponseWriter, req *http.Request) {
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
	nativeDiskPerfs, err := ParseDiskIO(diskPerfList)

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
func ParseDiskIO(diskPerfs []whatap_model.NodeDiskPerf) ([]whatap_model.NodeDiskPerf, error) {
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
