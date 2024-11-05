package osinfo

//WinCPUStat WinCPUStat
type WinCPUStat struct {
	Device string
	User   float32
	System float32
	Idle   float32
	C1 float32
	C2 float32
	C3 float32
	DPC float32
	Interrupt float32
}

//LinuxCPUStat LinuxCPUStat
type LinuxCPUStat struct {
	Device  string
	User    float32
	Nice    float32
	System  float32
	Idle    float32
	Iowait  float32
	Irq     float32
	Softirq float32
	Steal   float32
	Load1   float32
	Load5   float32
	Load15  float32
	Ctxt	float32
	LoadPerCore float32
	ProcsBlocked float32
	ProcsRunning float32
	NewProcForked float32
}

//WinMemoryStat WinMemoryStat
type WinMemoryStat struct {
	Total             uint64
	Free              uint64
	Cached            uint64
	Used              uint64
	Pused             float32
	Available         uint64
	Pavailable        float32
	SwapTotal         uint64
	SwapFree          uint64
	SwapUsed          uint64
	SwapPused         float32
	PageFaults        float32
	PoolPagedBytes    int64
	PoolNonpagedBytes int64
}

//LinuxMemoryStat LinuxMemoryStat
type LinuxMemoryStat struct {
	Total        uint64
	Free         uint64
	Buffers      uint64
	Cached       uint64
	Used         uint64
	Pused        float32
	Available    uint64
	Pavailable   float32
	Shared       uint64
	SwapTotal    uint64
	SwapFree     uint64
	SwapUsed     uint64
	SwapPused    float32
	PageFaults   float32
	Slab         uint64
	SReclaimable uint64
	SUnreclaim   uint64
}

//DiskPerf DiskPerf
type DiskPerf struct {
	DeviceID         string
	MajorMinor       string
	MountPoint       string
	FileSystem       string
	FreeSpace        uint64
	UsedSpace        uint64
	FreePercent      float32
	UsedPercent      float32
	TotalSpace       uint64
	Blksize          int32
	ReadIops         float64
	WriteIops        float64
	ReadBps          float64
	WriteBps         float64
	IOPercent        float32
	QueueLength      float32
	InodeTotal       int64
	InodeUsed        int64
	InodeUsedPercent float32
	MountOption      string
}

//NetworkPerformance NetworkPerformance
type NetworkPerformance struct {
	Desc       string
	IP         []uint32
	HwAddr     string
	TrafficIn  float64
	TrafficOut float64
	PacketIn   float64
	PacketOut  float64
	ErrorOut   float64
	ErrorIn    float64
	DroppedOut float64
	DroppedIn  float64
	TrafficInAcct  int64
	TrafficOutAcct int64	
}

type ProcNetPerf struct {
	LocalIP   int32
	LocalPort int16
	Conn      int32
}

type FileInfo struct {
	Name string
	Size int64
}
type ProcessPerfInfo struct {
	PPid                int32
	Pid                 int32
	Cpu                 float32
	MemoryBytes         int64
	MemoryPercent       float32
	ReadBps             float32
	WriteBps            float32
	Cmd1                string
	Cmd2                string
	ReadIops            float32
	WriteIops           float32
	User                string
	State               string
	CreateTime          int64
	Net                 []ProcNetPerf
	File                []*FileInfo
	MemoryShared        int64
	OpenFileDescriptors int64
}

type TcpPerfInfo struct {
	Port    int32
	IsAlive bool
}

type LogEvent struct {
	FilePath   *string
	LogContent *string
	Severity   int
	TriggerId  *string
	Target     *string
}

type IpPort struct {
	localip     int32
	localport   int16
	ilocaladdr  int64
	iremoteaddr int64
	conn        int32
	inode       int32
}
