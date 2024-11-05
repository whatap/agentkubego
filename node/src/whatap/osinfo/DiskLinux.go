package osinfo

import (
	"bufio"
	"fmt"
	whatap_config "github.com/whatap/kube/node/src/whatap/config"
	whatap_model "github.com/whatap/kube/node/src/whatap/lang/model"
	"github.com/whatap/kube/node/src/whatap/util/logutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func ParseNativeDiskIO(diskQuotas []whatap_model.NodeDiskPerf) ([]whatap_model.NodeDiskPerf, error) {
	diskIOMap := make(map[string]whatap_model.NodeDiskIORaw)
	diskPerfMap := make(map[string][]whatap_model.NodeDiskPerf)

	for _, diskQuota := range diskQuotas {
		//panicutil.Debug("ParseDiskIO", diskQuota.MajorMinor, diskQuota.DeviceID)
		if _, ok := diskPerfMap[diskQuota.DeviceID]; !ok {
			diskPerfMap[diskQuota.DeviceID] = []whatap_model.NodeDiskPerf{diskQuota}
		} else {
			diskPerfMap[diskQuota.DeviceID] = append(diskPerfMap[diskQuota.DeviceID], diskQuota)
		}
	}

	diskStatsFilepath := fmt.Sprint(whatap_config.GetConfig().HostPathPrefix, "/proc/diskstats")
	file, err := os.Open(diskStatsFilepath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logutil.Debugf("ParseNativeDiskIO-/proc/diskstats", "error=%v\n", err)
		}
	}(file)

	var diskIoMap2 map[string]whatap_model.NodeDiskIORaw
	loadedCache := LoadCache(diskStatsFilepath)
	if loadedCache != nil {
		diskIoMap2 = loadedCache.(map[string]whatap_model.NodeDiskIORaw)
	}

	var diskPerfs []whatap_model.NodeDiskPerf
	guestjiff, _ := getCpuGuestJiff()
	timestamp := time.Now().UnixNano()
	scanner := bufio.NewScanner(file)
	// i := 0
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		deviceID := words[2]
		sectorSize := DiskSectorSizeBytes(deviceID)

		major := words[0]
		minor := words[1]
		majorminor := fmt.Sprint(major, ":", minor)
		dqs, ok := diskPerfMap[majorminor]

		//panicutil.Debug("DiskLinux ParseDiskIO ", deviceId, majorminor, ok, diskPerfMapEx)
		if ok {
			readIoCount, _ := strconv.ParseInt(words[3], 10, 64)
			readIoByteCount, _ := strconv.ParseInt(words[5], 10, 64)
			readIoByteCount = readIoByteCount * sectorSize
			writeIoCount, _ := strconv.ParseInt(words[7], 10, 64)
			writeIoByteCount, _ := strconv.ParseInt(words[9], 10, 64)
			writeIoByteCount = writeIoByteCount * sectorSize
			ioMillis, _ := strconv.ParseInt(words[12], 10, 64)
			avgQSize, _ := strconv.ParseInt(words[13], 10, 64)

			diskIORaw := whatap_model.NodeDiskIORaw{DeviceID: deviceID, ReadIoCount: readIoCount, ReadIoByteCount: readIoByteCount,
				WriteIoCount: writeIoCount, WriteIoByteCount: writeIoByteCount, Timestamp: timestamp,
				IoMillis: ioMillis, AvgQSize: avgQSize, Guestjiff: guestjiff}
			diskIOMap[deviceID] = diskIORaw

			if diskIoMap2 != nil {
				if diskIORaw2, ok := diskIoMap2[deviceID]; ok {
					for _, val := range dqs {
						val.ReadIops = float64(diskIORaw.ReadIoCount-diskIORaw2.ReadIoCount) / float64(diskIORaw.Timestamp-diskIORaw2.Timestamp) * 1000000000.0
						val.WriteIops = float64(diskIORaw.WriteIoCount-diskIORaw2.WriteIoCount) / float64(diskIORaw.Timestamp-diskIORaw2.Timestamp) * 1000000000.0
						val.ReadBps = float64(diskIORaw.ReadIoByteCount-diskIORaw2.ReadIoByteCount) / float64(diskIORaw.Timestamp-diskIORaw2.Timestamp) * 1000000000.0
						val.WriteBps = float64(diskIORaw.WriteIoByteCount-diskIORaw2.WriteIoByteCount) / float64(diskIORaw.Timestamp-diskIORaw2.Timestamp) * 1000000000.0
						logutil.Debugf("WriteBps", "diskIORaw.WriteIoByteCount=%v, diskIORaw2.WriteIoByteCount=%v, diskIORaw.Timestamp=%v, diskIORaw2.Timestamp=%v", float64(diskIORaw.WriteIoByteCount), float64(diskIORaw2.WriteIoByteCount), float64(diskIORaw.Timestamp), float64(diskIORaw2.Timestamp))
						val.IOPercent = float32(float64(diskIORaw.IoMillis-diskIORaw2.IoMillis) / float64(diskIORaw.Timestamp-diskIORaw2.Timestamp) * 1000000.0 * 100.0)
						val.QueueLength = float32(float64(diskIORaw.AvgQSize-diskIORaw2.AvgQSize) / float64(diskIORaw.Guestjiff-diskIORaw2.Guestjiff) * 100.0 / 1000.0)
						diskPerfs = append(diskPerfs, val)
						//fmt.Println(deviceID, "write iops", val.WriteIops, "read iops", val.ReadIops,"read bps", val.ReadBps,"write bps", val.WriteBps, "%", val.IOPercent)
						//fmt.Println("QueueLength",deviceID,  fmt.Sprintf("%.2f",val.QueueLength),diskIORaw.Guestjiff-diskIORaw2.Guestjiff )
					}
				}
			}
			// ioPopulatedDiskPerfs[i] = val
			// i++
			delete(diskPerfMap, majorminor)
		}
	}
	SaveCache(diskStatsFilepath, diskIOMap)

	for _, vals := range diskPerfMap {
		for _, val := range vals {
			diskPerfs = append(diskPerfs, val)
		}
	}
	return diskPerfs, nil
}
func DiskSectorSizeBytes(logicalDisk string) int64 {
	var disk string
	filesInfo, err := os.ReadDir(whatap_config.GetConfig().PathSysBlock)
	for i := 0; i < len(filesInfo) && err == nil; i++ {
		fileInfo := filesInfo[i]

		physicalDisk := fileInfo.Name()
		if strings.HasPrefix(logicalDisk, physicalDisk) {
			disk = physicalDisk
			break
		}
	}
	if len(disk) < 1 {
		return 512
	}
	path := filepath.Join(whatap_config.GetConfig().PathSysBlock, disk, "queue", "hw_sector_size")
	contents, err := os.ReadFile(path)
	if err != nil {
		return 512
	}
	i, err := strconv.Atoi(strings.TrimSpace(string(contents)))
	if err != nil {
		return 512
	}
	return int64(i)
}

func getCpuGuestJiff() (int64, error) {
	uptimeFilepath := fmt.Sprint(whatap_config.GetConfig().HostPathPrefix, "/proc/uptime")
	file, err := os.Open(uptimeFilepath)
	if err != nil {
		return 0, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logutil.Debugf("getCpuGuestJiff-open /proc/uptime", "error=%v\n", err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		numbers := strings.Split(words[0], ".")
		seconds, _ := strconv.ParseInt(numbers[0], 10, 64)
		millis, _ := strconv.ParseInt(numbers[1], 10, 64)

		return seconds*100 + millis, nil
	}

	return 0, nil
}
