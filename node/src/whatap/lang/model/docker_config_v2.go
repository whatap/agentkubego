package model

import (
	"github.com/hako/durafmt"
	"time"
	"fmt"
)

type DockerConfigV2 struct {
	StreamConfig struct {
	} `json:"StreamConfig"`
	State struct {
		Running           bool        `json:"Running"`
		Paused            bool        `json:"Paused"`
		Restarting        bool        `json:"Restarting"`
		OOMKilled         bool        `json:"OOMKilled"`
		RemovalInProgress bool        `json:"RemovalInProgress"`
		Dead              bool        `json:"Dead"`
		Pid               int         `json:"Pid"`
		ExitCode          int         `json:"ExitCode"`
		Error             string      `json:"Error"`
		StartedAt         time.Time   `json:"StartedAt"`
		FinishedAt        time.Time   `json:"FinishedAt"`
		Health            interface{} `json:"Health"`
	} `json:"State"`
	ID      string        `json:"ID"`
	Created time.Time     `json:"Created"`
	Managed bool          `json:"Managed"`
	Path    string        `json:"Path"`
	Args    []interface{} `json:"Args"`
	Config  struct {
		Hostname     string      `json:"Hostname"`
		Domainname   string      `json:"Domainname"`
		User         string      `json:"User"`
		AttachStdin  bool        `json:"AttachStdin"`
		AttachStdout bool        `json:"AttachStdout"`
		AttachStderr bool        `json:"AttachStderr"`
		Tty          bool        `json:"Tty"`
		OpenStdin    bool        `json:"OpenStdin"`
		StdinOnce    bool        `json:"StdinOnce"`
		Env          []string    `json:"Env"`
		Cmd          interface{} `json:"Cmd"`
		Image        string      `json:"Image"`
		Volumes      interface{} `json:"Volumes"`
		WorkingDir   string      `json:"WorkingDir"`
		Entrypoint   []string    `json:"Entrypoint"`
		OnBuild      interface{} `json:"OnBuild"`
		Labels       struct {
			AnnotationKubernetesIoConfigSeen                time.Time `json:"annotation.kubernetes.io/config.seen"`
			AnnotationKubernetesIoConfigSource              string    `json:"annotation.kubernetes.io/config.source"`
			AnnotationSchedulerAlphaKubernetesIoCriticalPod string    `json:"annotation.scheduler.alpha.kubernetes.io/critical-pod"`
			ControllerRevisionHash                          string    `json:"controller-revision-hash"`
			IoKubernetesContainerName                       string    `json:"io.kubernetes.container.name"`
			IoKubernetesDockerType                          string    `json:"io.kubernetes.docker.type"`
			IoKubernetesPodName                             string    `json:"io.kubernetes.pod.name"`
			IoKubernetesPodNamespace                        string    `json:"io.kubernetes.pod.namespace"`
			IoKubernetesPodUID                              string    `json:"io.kubernetes.pod.uid"`
			K8SApp                                          string    `json:"k8s-app"`
			PodTemplateGeneration                           string    `json:"pod-template-generation"`
		} `json:"Labels"`
	} `json:"Config"`
	Image           string `json:"Image"`
	NetworkSettings struct {
		Bridge                 string `json:"Bridge"`
		SandboxID              string `json:"SandboxID"`
		HairpinMode            bool   `json:"HairpinMode"`
		LinkLocalIPv6Address   string `json:"LinkLocalIPv6Address"`
		LinkLocalIPv6PrefixLen int    `json:"LinkLocalIPv6PrefixLen"`
		Networks               struct {
			Host struct {
				IPAMConfig          interface{} `json:"IPAMConfig"`
				Links               interface{} `json:"Links"`
				Aliases             interface{} `json:"Aliases"`
				NetworkID           string      `json:"NetworkID"`
				EndpointID          string      `json:"EndpointID"`
				Gateway             string      `json:"Gateway"`
				IPAddress           string      `json:"IPAddress"`
				IPPrefixLen         int         `json:"IPPrefixLen"`
				IPv6Gateway         string      `json:"IPv6Gateway"`
				GlobalIPv6Address   string      `json:"GlobalIPv6Address"`
				GlobalIPv6PrefixLen int         `json:"GlobalIPv6PrefixLen"`
				MacAddress          string      `json:"MacAddress"`
				DriverOpts          interface{} `json:"DriverOpts"`
				IPAMOperational     bool        `json:"IPAMOperational"`
			} `json:"host"`
		} `json:"Networks"`
		Service interface{} `json:"Service"`
		Ports   struct {
		} `json:"Ports"`
		SandboxKey             string      `json:"SandboxKey"`
		SecondaryIPAddresses   interface{} `json:"SecondaryIPAddresses"`
		SecondaryIPv6Addresses interface{} `json:"SecondaryIPv6Addresses"`
		IsAnonymousEndpoint    bool        `json:"IsAnonymousEndpoint"`
		HasSwarmEndpoint       bool        `json:"HasSwarmEndpoint"`
	} `json:"NetworkSettings"`
	LogPath                string `json:"LogPath"`
	Name                   string `json:"Name"`
	Driver                 string `json:"Driver"`
	OS                     string `json:"OS"`
	MountLabel             string `json:"MountLabel"`
	ProcessLabel           string `json:"ProcessLabel"`
	RestartCount           int    `json:"RestartCount"`
	HasBeenStartedBefore   bool   `json:"HasBeenStartedBefore"`
	HasBeenManuallyStopped bool   `json:"HasBeenManuallyStopped"`
	MountPoints            struct {
	} `json:"MountPoints"`
	SecretReferences interface{} `json:"SecretReferences"`
	ConfigReferences interface{} `json:"ConfigReferences"`
	AppArmorProfile  string      `json:"AppArmorProfile"`
	HostnamePath     string      `json:"HostnamePath"`
	HostsPath        string      `json:"HostsPath"`
	ShmPath          string      `json:"ShmPath"`
	ResolvConfPath   string      `json:"ResolvConfPath"`
	SeccompProfile   string      `json:"SeccompProfile"`
	NoNewPrivileges  bool        `json:"NoNewPrivileges"`
}

func (self *DockerConfigV2) ParseState()(string){
	var age string 
	if self.State.StartedAt.Unix() > 0{
		age = durafmt.ParseShort(time.Now().Sub(self.State.StartedAt)).String()
	}

	if self.State.Running {
		return fmt.Sprint("Up", age)
	}else if self.State.Paused{
		return fmt.Sprint("Paused")
	}else if self.State.Restarting{
		return fmt.Sprint("Restarting")
	}else if self.State.OOMKilled{
		return fmt.Sprint("OOMKilled")
	}else if self.State.RemovalInProgress{
		return fmt.Sprint("RemovalInProgress")
	}else if self.State.Dead{
		return fmt.Sprint("Dead")
	}

	return "N/A"


}