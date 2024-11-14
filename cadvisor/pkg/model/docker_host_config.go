package model

type DockerHostConfig struct {
	Binds           interface{} `json:"Binds"`
	ContainerIDFile string      `json:"ContainerIDFile"`
	LogConfig       struct {
		Type   string `json:"Type"`
		Config struct {
		} `json:"Config"`
	} `json:"LogConfig"`
	NetworkMode  string `json:"NetworkMode"`
	PortBindings struct {
	} `json:"PortBindings"`
	RestartPolicy struct {
		Name              string `json:"Name"`
		MaximumRetryCount int    `json:"MaximumRetryCount"`
	} `json:"RestartPolicy"`
	AutoRemove           bool        `json:"AutoRemove"`
	VolumeDriver         string      `json:"VolumeDriver"`
	VolumesFrom          interface{} `json:"VolumesFrom"`
	CapAdd               interface{} `json:"CapAdd"`
	CapDrop              interface{} `json:"CapDrop"`
	DNS                  interface{} `json:"Dns"`
	DNSOptions           interface{} `json:"DnsOptions"`
	DNSSearch            interface{} `json:"DnsSearch"`
	ExtraHosts           interface{} `json:"ExtraHosts"`
	GroupAdd             interface{} `json:"GroupAdd"`
	IpcMode              string      `json:"IpcMode"`
	Cgroup               string      `json:"Cgroup"`
	Links                interface{} `json:"Links"`
	OomScoreAdj          int         `json:"OomScoreAdj"`
	PidMode              string      `json:"PidMode"`
	Privileged           bool        `json:"Privileged"`
	PublishAllPorts      bool        `json:"PublishAllPorts"`
	ReadonlyRootfs       bool        `json:"ReadonlyRootfs"`
	SecurityOpt          []string    `json:"SecurityOpt"`
	UTSMode              string      `json:"UTSMode"`
	UsernsMode           string      `json:"UsernsMode"`
	ShmSize              int         `json:"ShmSize"`
	Runtime              string      `json:"Runtime"`
	ConsoleSize          []int       `json:"ConsoleSize"`
	Isolation            string      `json:"Isolation"`
	CPUShares            int         `json:"CpuShares"`
	Memory               int64       `json:"Memory"`
	NanoCpus             int         `json:"NanoCpus"`
	CgroupParent         string      `json:"CgroupParent"`
	BlkioWeight          int         `json:"BlkioWeight"`
	BlkioWeightDevice    interface{} `json:"BlkioWeightDevice"`
	BlkioDeviceReadBps   interface{} `json:"BlkioDeviceReadBps"`
	BlkioDeviceWriteBps  interface{} `json:"BlkioDeviceWriteBps"`
	BlkioDeviceReadIOps  interface{} `json:"BlkioDeviceReadIOps"`
	BlkioDeviceWriteIOps interface{} `json:"BlkioDeviceWriteIOps"`
	CPUPeriod            int         `json:"CpuPeriod"`
	CPUQuota             int         `json:"CpuQuota"`
	CPURealtimePeriod    int         `json:"CpuRealtimePeriod"`
	CPURealtimeRuntime   int         `json:"CpuRealtimeRuntime"`
	CpusetCpus           string      `json:"CpusetCpus"`
	CpusetMems           string      `json:"CpusetMems"`
	Devices              interface{} `json:"Devices"`
	DeviceCgroupRules    interface{} `json:"DeviceCgroupRules"`
	DiskQuota            int         `json:"DiskQuota"`
	KernelMemory         int64       `json:"KernelMemory"`
	MemoryReservation    int64       `json:"MemoryReservation"`
	MemorySwap           int64       `json:"MemorySwap"`
	MemorySwappiness     interface{} `json:"MemorySwappiness"`
	OomKillDisable       bool        `json:"OomKillDisable"`
	PidsLimit            int         `json:"PidsLimit"`
	Ulimits              interface{} `json:"Ulimits"`
	CPUCount             int         `json:"CpuCount"`
	CPUPercent           int         `json:"CpuPercent"`
	IOMaximumIOps        int         `json:"IOMaximumIOps"`
	IOMaximumBandwidth   int         `json:"IOMaximumBandwidth"`
	MaskedPaths          []string    `json:"MaskedPaths"`
	ReadonlyPaths        []string    `json:"ReadonlyPaths"`
}