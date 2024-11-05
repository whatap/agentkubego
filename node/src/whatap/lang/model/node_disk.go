package model

type NodeDiskPerf struct {
	Device     string `json:"Device"`
	DeviceID   string `json:"DeviceID"`
	MountPoint string `json:"MountPoint"`
	FileSystem string `json:"FileSystem"`
	Major      int    `json:"Major"`
	Minor      int    `json:"Minor"`
	Type       string `json:"Type"`
	Inodes     int64  `json:"Inodes"`
	InodesFree int64  `json:"InodesFree"`

	Capacity         int64   `json:"Capacity"`
	Free             int64   `json:"Free"`
	Available        int64   `json:"Available"`
	AvailablePercent float32 `json:"AvailablePercent"`
	UsedSpace        int64   `json:"UsedSpace"`
	FreePercent      float32 `json:"FreePercent"`
	UsedPercent      float32 `json:"UsedPercent"`
	Blksize          int32   `json:"Blksize"`
	IOPercent        float32 `json:"IOPercent"`
	QueueLength      float32 `json:"QueueLength"`
	ReadBps          float64 `json:"ReadBps"`
	ReadIops         float64 `json:"ReadIops"`
	InodeTotal       int64   `json:"InodeTotal"`
	InodeUsed        int64   `json:"InodeUsed"`
	InodeUsedPercent float32 `json:"InodeUsedPercent"`
	WriteBps         float64 `json:"WriteBps"`
	WriteIops        float64 `json:"WriteIops"`
}

type NodeDiskIORaw struct {
	DeviceID         string `json:"DeviceID"`
	MajorMinor       string `json:"MajorMinor"`
	ReadIoCount      int64  `json:"ReadIoCount"`
	ReadIoByteCount  int64  `json:"ReadIoByteCount"`
	WriteIoCount     int64  `json:"WriteIoCount"`
	WriteIoByteCount int64  `json:"WriteIoByteCount"`
	Timestamp        int64  `json:"Timestamp"`
	IoMillis         int64  `json:"IoMillis"`
	AvgQSize         int64  `json:"AvgQSize"`
	Guestjiff        int64  `json:"Guestjiff"`
}
