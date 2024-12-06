package pods

import (
	"fmt"
	"github.com/whatap/kube/cadvisor/pkg/client"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	"github.com/whatap/kube/cadvisor/pkg/kubeapi"
	whatap_micro "github.com/whatap/kube/cadvisor/pkg/micro"
	whatap_model "github.com/whatap/kube/cadvisor/pkg/model"
	"github.com/whatap/kube/cadvisor/pkg/proc"
	whatap_osinfo "github.com/whatap/kube/cadvisor/tools/osinfo"
	"github.com/whatap/kube/tools/util/logutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	nodename            = os.Getenv("NODE_NAME")
	processedContainers sync.Map // sync.Map을 사용하여 동시성 문제를 해결
)

type SimpleContainerInfo struct {
	ContainerName       string
	ContainerId         string
	IsApm               bool
	ExecProcessed       bool
	whatapJavaAgentPath string
	ContainerState      string
	ProcessPids         []int
	ProcessInfo         map[string]whatap_model.ProcessInfo
}

type SimplePodInfo struct {
	Name             string
	Namespace        string
	NodeName         string
	Uid              string
	ContainerInfos   []SimpleContainerInfo
	ContainsApmAgent bool
	Labels           map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
}

func RunPodInformer() {
	logutil.Infof("whatap-node-helper", "start nodeName=%v\n", nodename)
	kubeClient, err := client.GetKubernetesClient()
	if kubeClient == nil || err != nil {
		logutil.Errorf("whatap-node-helper", "InitializeInformer error getting client: %v\n", err)
		return
	}

	// 인포머 팩토리 생성
	factory := informers.NewSharedInformerFactory(kubeClient, 0)
	podsInformer := factory.Core().V1().Pods().Informer()

	ch := make(chan struct{})
	// Workqueue 구성 및 생성
	//rateLimiter := workqueue.DefaultControllerRateLimiter()
	//queueConfig := workqueue.RateLimitingQueueConfig{
	//	Name: "Endpoints"}
	//workQueue := workqueue.NewRateLimitingQueueWithConfig(rateLimiter, queueConfig)

	//Pod 이벤트 핸들러 등록
	_, addEventHandlerErr := podsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			podEventHandler(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			podEventHandler(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			//endpointsMapLock.Lock()
			//defer endpointsMapLock.Unlock()
			//endPointsMap = make(map[string]SimplePodInfo) // Reset on delete
		},
	})
	if addEventHandlerErr != nil {
		return
	}
	if err != nil {
		return
	}
	go podsInformer.Run(ch)
	cache.WaitForCacheSync(ch, podsInformer.HasSynced)
}

func podEventHandler(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok || pod.Spec.NodeName != nodename || pod.Spec.NodeName == "" {
		logutil.Debugf("whatap-node-helper", "pod.Spec.NodeName=%v, nodename(env)=%v", pod.Spec.NodeName, nodename)
		return
	}
	logutil.Debugln("whatap-node-helper", "podEventHandler Start")
	processPodsMap(pod)
}
func processPodsMap(pod *corev1.Pod) {
	podsMapLock.Lock()
	defer podsMapLock.Unlock()
	podInfo := createSimplePodInfo(pod)
	process(&podInfo)
	if podInfo.ContainsApmAgent && whatap_config.GetConfig().InjectContainerIdToApmAgentEnabled {
		logutil.Debugf("whatap-node-helper", "podInfo.ContainsApmAgent=%v, InjectContainerIdToApmAgentEnabled=%v", podInfo.ContainsApmAgent, whatap_config.GetConfig().InjectContainerIdToApmAgentEnabled)
		executeInjection(&podInfo)
	}
	podsMap[podInfo.Uid] = podInfo
}

func createSimplePodInfo(pod *corev1.Pod) SimplePodInfo {
	if podsMap == nil {
		podsMap = make(map[string]SimplePodInfo)
	}

	podInfo := SimplePodInfo{
		Name:      pod.GetName(),
		Namespace: pod.GetNamespace(),
		NodeName:  pod.Spec.NodeName,
		Uid:       string(pod.GetUID()),
		Labels:    pod.Labels,
	}

	for _, containerSpec := range pod.Spec.Containers {
		// container spec 1개씩 까보기
		containerInfo := SimpleContainerInfo{
			ContainerName:       containerSpec.Name,
			ContainerId:         getContainerId(pod.Status.ContainerStatuses, containerSpec.Name),
			IsApm:               checkIfApmAgent(containerSpec.Env),
			whatapJavaAgentPath: getWhatapJavaAgentPath(containerSpec.Env),
			ContainerState:      getContainerState(pod.Status.ContainerStatuses, containerSpec.Name)}

		if containerInfo.IsApm {
			podInfo.ContainsApmAgent = true
			if containerInfo.ContainerId != "" {
				containerInfo.ExecProcessed = isContainerProcessed(containerInfo.ContainerId)
			}
		}

		podInfo.ContainerInfos = append(podInfo.ContainerInfos, containerInfo)
	}
	return podInfo
}
func getContainerId(statuses []corev1.ContainerStatus, containerName string) string {
	for _, cs := range statuses {
		if cs.Name == containerName {
			containerId := strings.SplitN(cs.ContainerID, "://", 2)
			if len(containerId) == 2 {
				return containerId[1]
			} else {
				logutil.Debugf("whatap-node-helper", "containerId not exist=%v", containerId)
				return ""
			}
		}
	}
	return ""
}
func checkIfApmAgent(envs []corev1.EnvVar) bool {
	for _, env := range envs {
		if env.Name == "POD_NAME" {
			return true
		}
	}
	return false
}
func getWhatapJavaAgentPath(envs []corev1.EnvVar) string {
	for _, env := range envs {
		if env.Name == "WHATAP_JAVA_AGENT_PATH" {
			return env.Value
		}
	}
	return ""
}
func getContainerState(ContainerStatuses []corev1.ContainerStatus, containerName string) string {
	for _, containerStatus := range ContainerStatuses {
		// containerStatus 1개씩 까서 spec과 이름 동일한거 찾기
		if containerStatus.Name == containerName {
			if containerStatus.State.Running != nil {
				return "Running"
			} else if containerStatus.State.Waiting != nil {
				return "Waiting"
			} else if containerStatus.State.Terminated != nil {
				return "Terminated"
			}
		}
	}
	return "Unknown"
}

func process(podInfo *SimplePodInfo) {
	for containerIdx, containerInfo := range podInfo.ContainerInfos {
		containerID := containerInfo.ContainerId
		if containerID == "" {
			continue
		}
		if containerInfo.ContainerState != "Running" {
			continue
		}

		pid, err := proc.GetContainerPid(containerID)
		if err != nil {
			logutil.Errorf("process", "GetContainerPidErr=%v", err)
			continue
		}

		pids, err := whatap_osinfo.FindChildPIDs(pid)
		if err != nil {
			logutil.Errorf("process", "GetContainerFindChildPidErr=%v", err)
		}
		pids = append(pids, pid)
		containerInfo.ProcessPids = pids
		podInfo.ContainerInfos[containerIdx] = containerInfo
	}
}

func executeInjection(podInfo *SimplePodInfo) {
	// Pod 내의 모든 컨테이너에 대해 반복
	for i, containerInfo := range podInfo.ContainerInfos {
		if containerInfo.ContainerId == "" {
			continue
		}
		if !containerInfo.IsApm {
			continue
		}

		// 이미 처리된 컨테이너는 건너뜀
		if containerInfo.ExecProcessed {
			continue
		}

		// 실행할 명령이 있는 경우
		// 1. env
		if checkAndExecuteInjection(i, podInfo, containerInfo, containerInfo.whatapJavaAgentPath, "ApmEnv") {
			podsMap[podInfo.Uid] = *podInfo
			continue
		}

		// 2. label
		whatapJavaAgentPathFromLabel := podInfo.Labels["WHATAP_JAVA_AGENT_PATH"]
		if checkAndExecuteInjection(i, podInfo, containerInfo, whatapJavaAgentPathFromLabel, "Label") {
			podsMap[podInfo.Uid] = *podInfo
			continue
		}

		// 3. WHATAP_NODE_HELPER 환경변수에 WHATAP_JAVA_AGENT_PATH 를 넣는 경우
		whatapJavaAgentPathFromNodeHelperEnv := whatap_config.GetConfig().WhatapJavaAgentPath
		if checkAndExecuteInjection(i, podInfo, containerInfo, whatapJavaAgentPathFromNodeHelperEnv, "NodeHelperEnv") {
			podsMap[podInfo.Uid] = *podInfo
			continue
		}

		//proc 정보를 통해 조회
		if whatap_config.GetConfig().InspectWhatapAgentPathFromProc {
			whatapJavaAgentPathFromProc, err := getWhatapJavaAgentPathFromProc(containerInfo.ProcessPids)
			if err != nil {
				logutil.Infof("whatapJavaAgentPathFromProc", "podName=%v, containerName=%v, id=%v, err=%v", podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, err)
			} else {
				if checkAndExecuteInjection(i, podInfo, containerInfo, whatapJavaAgentPathFromProc, "whatapJavaAgentPathFromProc") {
					podsMap[podInfo.Uid] = *podInfo
					continue
				}
			}
		}

		// 위 모든 경우에서 PATH 를 찾지 못하면 런타임 API 를 사용
		whataJavaAgentPathFromRuntimeAPI, err := whatap_micro.InspectWhatapAgentPath(containerInfo.ContainerId)
		if err != nil {
			logutil.Infof("whataJavaAgentPathFromRuntimeAPI", "podName=%v, containerName=%v, id=%v, InspectErr=%v", podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, err)
			podsMap[podInfo.Uid] = *podInfo
			continue
		}

		logutil.Infof("whataJavaAgentPathFromRuntimeAPI", "podName=%v, containerName=%v, id=%v, agentPath=%v", podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, whataJavaAgentPathFromRuntimeAPI)
		if whataJavaAgentPathFromRuntimeAPI != "" {
			if !whatap_micro.IsValidAgentPath(whataJavaAgentPathFromRuntimeAPI) {
				logutil.Infof("whataJavaAgentPathFromRuntimeAPI", "agentPath=%v is not valid", whataJavaAgentPathFromRuntimeAPI)
				continue
			}
			ok, execErr := execCall(i, whataJavaAgentPathFromRuntimeAPI, podInfo, containerInfo)
			if execErr != nil {
				logutil.Infof("whataJavaAgentPathFromRuntimeAPI", "runtimeErr=%v podName=%v, containerName=%v, id=%v, whatapJavaAgentPathFromRuntimeAPI=%v", execErr, podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, whataJavaAgentPathFromRuntimeAPI)
				continue
			}
			if ok {
				logutil.Infof("whataJavaAgentPathFromRuntimeAPI", "runtimeOK podName=%v, containerName=%v, id=%v, whatapJavaAgentPathFromRuntimeAPI=%v", podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, whataJavaAgentPathFromRuntimeAPI)
				podsMap[podInfo.Uid] = *podInfo
			}
		}
	}
	// podsMap 내 해당 Pod 정보 업데이트
	podsMap[podInfo.Uid] = *podInfo
}
func checkAndExecuteInjection(i int, podInfo *SimplePodInfo, containerInfo SimpleContainerInfo, agentPath, source string) bool {
	if agentPath != "" {
		agentPathIsValid := whatap_micro.IsValidAgentPath(agentPath)
		logutil.Infof("executeInjection", "podName=%v, containerName=%v, id=%v, %s=%v, valid=%v", podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, source, agentPath, agentPathIsValid)
		if agentPathIsValid {
			ok, _ := execCall(i, agentPath, podInfo, containerInfo)
			return ok
		}
	} else {
		logutil.Infof("executeInjection", "podName=%v, containerName=%v, id=%v, %s not found", podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, source)
	}
	return false
}
func execCall(i int, agentPath string, podInfo *SimplePodInfo, containerInfo SimpleContainerInfo) (bool, error) {
	_, execErr := executeWhatapAgentCommand(podInfo.Namespace, podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, agentPath)
	processedContainers.Store(containerInfo.ContainerId, true) // 처리된 컨테이너 ID를 저장
	podInfo.ContainerInfos[i].ExecProcessed = true
	podName := podInfo.Name
	containerName := containerInfo.ContainerName
	containerId := containerInfo.ContainerId
	containerState := containerInfo.ContainerState
	if execErr == nil {
		// 명령 실행 성공 시, ExecProcessed를 true로 설정
		logutil.Infof("execCall", "Successfully executed Whatap agent command for pod: %v // container: %v:%v[%v]", podName, containerName, containerId, containerState)
		return true, nil
	} else {
		logutil.Infof("execCall", "Failed to execute Whatap agent command for pod: %v, container: %v:%v[%v], err: %v", podInfo.Name, containerInfo.ContainerName, containerInfo.ContainerId, containerInfo.ContainerState, execErr)
		return false, execErr
	}
}
func isContainerProcessed(containerID string) bool {
	_, processed := processedContainers.Load(containerID)
	return processed
}
func getWhatapJavaAgentPathFromProc(pids []int) (string, error) {
	for _, chpid := range pids {
		line, cmdLineErr := getWhatapJavaAgentPathFromCmdLine(chpid)
		if cmdLineErr != nil {
			return "", cmdLineErr
		}
		return line, nil
	}
	return "", fmt.Errorf("agentPathNotFound")
}
func getWhatapJavaAgentPathFromCmdLine(pid int) (string, error) {
	// Convert PID to string
	pidStr := fmt.Sprintf("%d", pid)
	cmdline := filepath.Join(whatap_config.GetConfig().HostPathPrefix, "proc", pidStr, "cmdline")
	logutil.Infof("cmdLine", "path=%v", cmdline)
	//Read the cmdline file content
	content, err := os.ReadFile(cmdline)
	if err != nil {
		return "", err
	}

	//Split the cmdline content by null character (\x00)
	args := strings.Split(string(content), "\x00")

	//Iterate over the arguments to find the -javaagent argument
	for _, arg := range args {
		if strings.HasPrefix(arg, "-javaagent:") {
			// Extract the JAR file path
			javaAgentPath := strings.TrimPrefix(arg, "-javaagent:")
			return javaAgentPath, nil
		}
	}
	return "", nil
}

func executeWhatapAgentCommand(podNamespace, podName, containerName, containerID, agentPath string) (bool, error) {
	cmds := []string{"java", "-cp", agentPath, "whatap.agent.ContainerConf", containerID}
	err := kubeapi.ExecCommandForInjectContainerIdToWhatapAgent(podNamespace, podName, containerName, containerID, cmds)
	if err != nil {
		logutil.Debugf("whatap-node-helper", "Failed to execute command for container %s: %v", containerName, err)
		return false, err
	} else {
		logutil.Debugf("whatap-node-helper", "Successfully executed Whatap agent command for pod: %s / container: %s[%v]", podName, containerName, containerID)
		return true, nil
	}
}
