package model

import(
	"time"
)

type OverlayState struct {
	OciVersion  string `json:"ociVersion"`
	ID          string `json:"id"`
	Status      string `json:"status"`
	Pid         int    `json:"pid"`
	Bundle      string `json:"bundle"`
	Annotations struct {
		IoKubernetesContainerHash                     string    `json:"io.kubernetes.container.hash"`
		IoKubernetesContainerName                     string    `json:"io.kubernetes.container.name"`
		IoKubernetesContainerRestartCount             string    `json:"io.kubernetes.container.restartCount"`
		IoKubernetesContainerTerminationMessagePath   string    `json:"io.kubernetes.container.terminationMessagePath"`
		IoKubernetesContainerTerminationMessagePolicy string    `json:"io.kubernetes.container.terminationMessagePolicy"`
		IoKubernetesCriOAnnotations                   string    `json:"io.kubernetes.cri-o.Annotations"`
		IoKubernetesCriOContainerID                   string    `json:"io.kubernetes.cri-o.ContainerID"`
		IoKubernetesCriOContainerType                 string    `json:"io.kubernetes.cri-o.ContainerType"`
		IoKubernetesCriOCreated                       time.Time `json:"io.kubernetes.cri-o.Created"`
		IoKubernetesCriOIP                            string    `json:"io.kubernetes.cri-o.IP"`
		IoKubernetesCriOImage                         string    `json:"io.kubernetes.cri-o.Image"`
		IoKubernetesCriOImageName                     string    `json:"io.kubernetes.cri-o.ImageName"`
		IoKubernetesCriOImageRef                      string    `json:"io.kubernetes.cri-o.ImageRef"`
		IoKubernetesCriOLabels                        string    `json:"io.kubernetes.cri-o.Labels"`
		IoKubernetesCriOLogPath                       string    `json:"io.kubernetes.cri-o.LogPath"`
		IoKubernetesCriOMetadata                      string    `json:"io.kubernetes.cri-o.Metadata"`
		IoKubernetesCriOMountPoint                    string    `json:"io.kubernetes.cri-o.MountPoint"`
		IoKubernetesCriOName                          string    `json:"io.kubernetes.cri-o.Name"`
		IoKubernetesCriOResolvPath                    string    `json:"io.kubernetes.cri-o.ResolvPath"`
		IoKubernetesCriOSandboxID                     string    `json:"io.kubernetes.cri-o.SandboxID"`
		IoKubernetesCriOSandboxName                   string    `json:"io.kubernetes.cri-o.SandboxName"`
		IoKubernetesCriOSeccompProfilePath            string    `json:"io.kubernetes.cri-o.SeccompProfilePath"`
		IoKubernetesCriOStdin                         string    `json:"io.kubernetes.cri-o.Stdin"`
		IoKubernetesCriOStdinOnce                     string    `json:"io.kubernetes.cri-o.StdinOnce"`
		IoKubernetesCriOTTY                           string    `json:"io.kubernetes.cri-o.TTY"`
		IoKubernetesCriOVolumes                       string    `json:"io.kubernetes.cri-o.Volumes"`
		IoKubernetesPodName                           string    `json:"io.kubernetes.pod.name"`
		IoKubernetesPodNamespace                      string    `json:"io.kubernetes.pod.namespace"`
		IoKubernetesPodTerminationGracePeriod         string    `json:"io.kubernetes.pod.terminationGracePeriod"`
		IoKubernetesPodUID                            string    `json:"io.kubernetes.pod.uid"`
	} `json:"annotations"`
	Created  time.Time `json:"created"`
	Started  time.Time `json:"started"`
	Finished time.Time `json:"finished"`
}