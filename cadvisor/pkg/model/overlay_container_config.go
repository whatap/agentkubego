package model

import(
	"time"
)

type OverlayConfig struct {
	OciVersion string `json:"ociVersion"`
	Process    struct {
		User struct {
			UID            int   `json:"uid"`
			Gid            int   `json:"gid"`
			AdditionalGids []int `json:"additionalGids"`
		} `json:"user"`
		Args         []string `json:"args"`
		Env          []string `json:"env"`
		Cwd          string   `json:"cwd"`
		Capabilities struct {
			Bounding    []string `json:"bounding"`
			Effective   []string `json:"effective"`
			Inheritable []string `json:"inheritable"`
			Permitted   []string `json:"permitted"`
		} `json:"capabilities"`
		OomScoreAdj int `json:"oomScoreAdj"`
	} `json:"process"`
	Root struct {
		Path string `json:"path"`
	} `json:"root"`
	Hostname string `json:"hostname"`
	Mounts   []struct {
		Destination string   `json:"destination"`
		Type        string   `json:"type"`
		Source      string   `json:"source"`
		Options     []string `json:"options"`
	} `json:"mounts"`
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
	Linux struct {
		Resources struct {
			Devices []struct {
				Allow  bool   `json:"allow"`
				Access string `json:"access"`
			} `json:"devices"`
			Memory struct {
				Limit int64 `json:"limit"`
			} `json:"memory"`
			CPU struct {
				Shares int `json:"shares"`
				Quota  int `json:"quota"`
				Period int `json:"period"`
			} `json:"cpu"`
			Pids struct {
				Limit int `json:"limit"`
			} `json:"pids"`
		} `json:"resources"`
		CgroupsPath string `json:"cgroupsPath"`
		Namespaces  []struct {
			Type string `json:"type"`
			Path string `json:"path,omitempty"`
		} `json:"namespaces"`
		MaskedPaths   []string `json:"maskedPaths"`
		ReadonlyPaths []string `json:"readonlyPaths"`
	} `json:"linux"`
}