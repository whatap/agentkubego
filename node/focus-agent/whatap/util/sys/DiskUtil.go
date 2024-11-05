package sys

import (
	//"log"

	"github.com/shirou/gopsutil/disk"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/logutil"
)

//type UsageStat struct {
//    Path              string  `json:"path"`
//    Fstype            string  `json:"fstype"`
//    Total             uint64  `json:"total"`
//    Free              uint64  `json:"free"`
//    Used              uint64  `json:"used"`
//    UsedPercent       float64 `json:"usedPercent"`
//    InodesTotal       uint64  `json:"inodesTotal"`
//    InodesUsed        uint64  `json:"inodesUsed"`
//    InodesFree        uint64  `json:"inodesFree"`
//    InodesUsedPercent float64 `json:"inodesUsedPercent"`
//}

func GetSysDiskUsedPercent(path string) float64 {

	stat, err := disk.Usage(path)

	if err != nil {
		logutil.Println("WA851", " Usage Error path=", path, stat.UsedPercent, ",Error=", err)
		return 0
	}

	return stat.UsedPercent
}
func GetSysDisk(path string) (*disk.UsageStat, error) {
	return disk.Usage(path)
}
