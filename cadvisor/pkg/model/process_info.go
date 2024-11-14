package model

import "time"

type ProcessInfo struct {
	Name                       string
	Command                    string
	Username                   string
	NodeName                   string
	Host                       string
	Namespace                  string
	PodName                    string
	PodUid                     string
	ContainerName              string
	ContainerID                string
	PID                        int
	PPID                       int
	NSPID                      int
	State                      string
	TotalCPUPercent            float64
	RSSMemory                  int64
	Threads                    int
	OpenFiles                  int
	CurrentWorkingDirectory    string
	StartedAgo                 time.Duration
	VoluntaryContextSwitches   int
	InvoluntaryContextSwitches int
	ReadsPerSecond             float64
	WritesPerSecond            float64
	ReadBytesPerSecond         float64
	WriteBytesPerSecond        float64
}
