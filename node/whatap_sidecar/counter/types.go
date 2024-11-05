package counter

import "time"

type TaskAction struct {
	task        Task
	lastActTime int64
	name        string
}

type Task interface {
	interval() int
	process(int64) error
}

type TaskContainer struct {
	totalMemory   int64
	numcpu        int
	lastStatCache map[string]*ContainerStat
	// containers    []*FGContainerInfo
}

type TaskNode struct {
}

type TaskKubeNode struct {
	numcpu       int32
	starttime    int64
	totalMemory  int64
	containerKey int32
}

type FGContainerInfo struct {
	prefix        string
	containerId   string
	name          string
	restartCount  int32
	pid           int32
	cpuLimit      int32
	memoryLimit   int32
	cpuRequest    int32
	memoryRequest int32
	state         int32
	status        string
	ready         int32

	command        string
	created        int32
	image          string
	imageId        string
	microOid       int32
	namespace      string
	onode          int32
	onodeName      string
	pod            string
	replicaSetName string
}

type BlkDeviceValue struct {
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Op    string `json:"op"`
	Value int64  `json:"value"`
}

type ContainerStat struct {
	Read      time.Time `json:"read"`
	Preread   time.Time `json:"preread"`
	PidsStats struct {
		Current int `json:"current"`
	} `json:"pids_stats"`
	BlkioStats struct {
		IoServiceBytesRecursive []BlkDeviceValue `json:"io_service_bytes_recursive"`
		IoServicedRecursive     []BlkDeviceValue `json:"io_serviced_recursive"`
		IoQueueRecursive        []BlkDeviceValue `json:"io_queue_recursive"`
		IoServiceTimeRecursive  []BlkDeviceValue `json:"io_service_time_recursive"`
		IoWaitTimeRecursive     []BlkDeviceValue `json:"io_wait_time_recursive"`
		IoMergedRecursive       []BlkDeviceValue `json:"io_merged_recursive"`
		IoTimeRecursive         []BlkDeviceValue `json:"io_time_recursive"`
		SectorsRecursive        []BlkDeviceValue `json:"sectors_recursive"`
	} `json:"blkio_stats"`
	NumProcs     int `json:"num_procs"`
	StorageStats struct {
	} `json:"storage_stats"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage        int64   `json:"total_usage"`
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		OnlineCpus     int   `json:"online_cpus"`
		ThrottlingData struct {
			Periods          int64 `json:"periods"`
			ThrottledPeriods int64 `json:"throttled_periods"`
			ThrottledTime    int64 `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"cpu_stats"`
	PrecpuStats struct {
		CPUUsage struct {
			TotalUsage        int64   `json:"total_usage"`
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		OnlineCpus     int   `json:"online_cpus"`
		ThrottlingData struct {
			Periods          int64 `json:"periods"`
			ThrottledPeriods int64 `json:"throttled_periods"`
			ThrottledTime    int64 `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage    int64 `json:"usage"`
		MaxUsage int64 `json:"max_usage"`
		Stats    struct {
			ActiveAnon              int64 `json:"active_anon"`
			ActiveFile              int64 `json:"active_file"`
			Cache                   int64 `json:"cache"`
			Dirty                   int64 `json:"dirty"`
			HierarchicalMemoryLimit int64 `json:"hierarchical_memory_limit"`
			HierarchicalMemswLimit  int64 `json:"hierarchical_memsw_limit"`
			InactiveAnon            int64 `json:"inactive_anon"`
			InactiveFile            int64 `json:"inactive_file"`
			MappedFile              int64 `json:"mapped_file"`
			Pgfault                 int64 `json:"pgfault"`
			Pgmajfault              int64 `json:"pgmajfault"`
			Pgpgin                  int64 `json:"pgpgin"`
			Pgpgout                 int64 `json:"pgpgout"`
			Rss                     int64 `json:"rss"`
			RssHuge                 int64 `json:"rss_huge"`
			TotalActiveAnon         int64 `json:"total_active_anon"`
			TotalActiveFile         int64 `json:"total_active_file"`
			TotalCache              int64 `json:"total_cache"`
			TotalDirty              int64 `json:"total_dirty"`
			TotalInactiveAnon       int64 `json:"total_inactive_anon"`
			TotalInactiveFile       int64 `json:"total_inactive_file"`
			TotalMappedFile         int64 `json:"total_mapped_file"`
			TotalPgfault            int64 `json:"total_pgfault"`
			TotalPgmajfault         int64 `json:"total_pgmajfault"`
			TotalPgpgin             int64 `json:"total_pgpgin"`
			TotalPgpgout            int64 `json:"total_pgpgout"`
			TotalRss                int64 `json:"total_rss"`
			TotalRssHuge            int64 `json:"total_rss_huge"`
			TotalUnevictable        int64 `json:"total_unevictable"`
			TotalWriteback          int64 `json:"total_writeback"`
			Unevictable             int64 `json:"unevictable"`
			Writeback               int64 `json:"writeback"`
		} `json:"stats"`
		Limit   int64 `json:"limit"`
		FailCnt int   `json:"failcnt"`
	} `json:"memory_stats"`
	Name         string `json:"name"`
	ID           string `json:"id"`
	NetworkStats struct {
		RxBytes   int64 `json:"rxBytes"`
		RxDropped int64 `json:"rxDropped"`
		RxErrors  int64 `json:"rxErrors"`
		RxPackets int64 `json:"rxPackets"`
		TxBytes   int64 `json:"txBytes"`
		TxDropped int64 `json:"txDropped"`
		TxErrors  int64 `json:"txErrors"`
		TxPackets int64 `json:"txPackets"`
	} `json:"network_stats"`
	RestartCount int `json:"restart_count"`
}