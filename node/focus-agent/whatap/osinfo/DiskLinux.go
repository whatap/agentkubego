// +build linux

package osinfo

/*
#include <unistd.h>
#include <stdlib.h>
*/
import "C"

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/stringutil"
)

const (
	PathSysBlock = "/sys/block"
)

var (
	filesystems = map[string]bool{"btrfs": true, "ext2": true, "ext3": true, "ext4": true, "reiser": true, "xfs": true,
		"ffs": true, "ufs": true, "jfs": true, "jfs2": true, "vxfs": true, "hfs": true, "ntfs": true, "fat32": true,
		"zfs": true, "refs": true, "nfs": true, "nfs2": true, "nfs3": true, "nfs4": true, "cifs": true, "ocfs2": true,
		"fuse": true, "fuse.glusterfs": true}
	MaxDisks int32 = 30
)

//DiskIORaw DiskIORaw
type DiskIORaw struct {
	DeviceID         string
	MajorMinor       string
	ReadIoCount      int64
	ReadIoByteCount  int64
	WriteIoCount     int64
	WriteIoByteCount int64
	Timestamp        int64
	IoMillis         int64
	AvgQSize         int64
	Guestjiff        int64
}

func getMappedDeviceId(deviceId string) string {
	searchDir := "/dev/mapper"
	filesInfo, err := ioutil.ReadDir(searchDir)
	for i := 0; i < len(filesInfo) && err == nil; i++ {
		fileinfo := filesInfo[i]
		linkpath := strings.Join([]string{searchDir, fileinfo.Name()}, "/")
		link, err := os.Readlink(linkpath)
		if err == nil {
			linkpath := strings.Join([]string{searchDir, link}, "/")
			link, err = filepath.Abs(linkpath)
			if err == nil {
				if deviceId == link[5:] {
					return fileinfo.Name()
				}
			}

		}
	}
	return ""
}

func DiskSectorSizeBytes(logicalDisk string) int64 {
	var disk string
	filesInfo, err := ioutil.ReadDir(PathSysBlock)
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
	path := filepath.Join(PathSysBlock, disk, "queue", "hw_sector_size")
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return 512
	}
	i, err := strconv.Atoi(strings.TrimSpace(string(contents)))
	if err != nil {
		return 512
	}
	return int64(i)
}

func ParseDiskIO(diskQuotas []DiskPerf) ([]DiskPerf, error) {
	var nativeDiskQuotas []DiskPerf
	for _, diskQuota := range diskQuotas {
		nativeDiskQuotas = append(nativeDiskQuotas, diskQuota)
	}
	nativeDiskQuotas, err := parseNativeDiskIO(nativeDiskQuotas)
	if err != nil {
		return diskQuotas, err
	}

	return nativeDiskQuotas, nil
}

func parseNativeDiskIO(diskQuotas []DiskPerf) ([]DiskPerf, error) {
	diskIOMap := make(map[string]DiskIORaw)
	diskPerfMap := make(map[string][]DiskPerf)
	diskPerfMapEx := make(map[string][]DiskPerf)
	for _, diskQuota := range diskQuotas {
		//panicutil.Debug("ParseDiskIO", diskQuota.MajorMinor, diskQuota.DeviceID)
		if _, ok := diskPerfMap[diskQuota.MajorMinor]; !ok {
			diskPerfMap[diskQuota.MajorMinor] = []DiskPerf{diskQuota}
		} else {
			diskPerfMap[diskQuota.MajorMinor] = append(diskPerfMap[diskQuota.MajorMinor], diskQuota)
		}

		if _, ok := diskPerfMapEx[diskQuota.DeviceID]; !ok {
			diskPerfMapEx[diskQuota.DeviceID] = []DiskPerf{diskQuota}
		} else {
			diskPerfMapEx[diskQuota.DeviceID] = append(diskPerfMapEx[diskQuota.DeviceID], diskQuota)
		}

	}

	filepath := "/proc/diskstats"

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var diskIoMap2 map[string]DiskIORaw
	loadedCache := LoadCache(filepath)
	if loadedCache != nil {
		diskIoMap2 = loadedCache.(map[string]DiskIORaw)
	}

	var diskPerfs []DiskPerf
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
		deviceId := words[2]
		dqs, ok := diskPerfMap[majorminor]
		if !ok {
			deviceId = fmt.Sprint("/dev/", deviceId)
			dqs, ok = diskPerfMapEx[deviceId]
		}

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

			diskIORaw := DiskIORaw{DeviceID: deviceID, ReadIoCount: readIoCount, ReadIoByteCount: readIoByteCount,
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
	SaveCache(filepath, diskIOMap)

	for _, vals := range diskPerfMap {
		for _, val := range vals {
			diskPerfs = append(diskPerfs, val)
		}

	}

	return diskPerfs, nil
}

func getCpuGuestJiff() (int64, error) {
	filepath := "/proc/uptime"
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

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

//GetDisk GetDisk
func GetDisk() ([]DiskPerf, error) {
	diskPerfs, err := getDisk()
	if len(diskPerfs) < 1 || err != nil {
		time.Sleep(1 * time.Second)
		diskPerfs, err = getDisk()
	}
	return diskPerfs, err
}

func parseDeviceIdByUUID(deviceID string) string {
	f, err := os.Lstat(deviceID)
	if err == nil {
		if f.Mode()&os.ModeSymlink == os.ModeSymlink {
			link, err := os.Readlink(deviceID)
			if err == nil {
				deviceID = link
			}
		}
	}

	return deviceID
}

func getDisk() ([]DiskPerf, error) {
	filepath := "/proc/self/mountinfo"
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	diskQuotas := make([]DiskPerf, MaxDisks)

	scanner := bufio.NewScanner(file)
	i := int32(0)
	deviceFilter := map[string]string{}
	for scanner.Scan() && i < MaxDisks {
		line := scanner.Text()
		words := strings.Fields(line)
		var filesystem string
		var mountSource string
		if len(words) == 10 {
			filesystem = words[7]
			mountSource = words[8]
		} else {
			filesystem = words[8]
			mountSource = words[9]
		}

		mountId := words[0]
		parentMountId := words[1]
		majorminor := words[2]
		mountPoint := words[4]
		mountOption := words[5]

		// log.Println("getDisk step -1", mountId, parentMountId, filesystem, mountSource, mountPoint)
		if _, ok := filesystems[filesystem]; ok {
			// log.Println("getDisk step -2")
			//Skip monitoring if parent child mount point is equal
			deviceFilter[mountId] = mountPoint
			if _, ok := deviceFilter[parentMountId]; ok {
				// log.Println("getDisk step -3")
				if deviceFilter[parentMountId] == mountPoint {
					// log.Println("getDisk step -4")
					continue
				}
			}
			//skip zfs child mount
			if filesystem == "zfs" || filesystem == "vzfs" {
				if parentMountpoint, ok := deviceFilter[parentMountId]; ok {
					// log.Println("getDisk step -3")
					parentIsZfsMount := false
					for j := int32(0); j < i; j += 1 {
						if parentMountpoint == diskQuotas[j].MountPoint && (diskQuotas[j].FileSystem == "zfs" ||
							diskQuotas[j].FileSystem == "vzfs") {
							parentIsZfsMount = true
							break
						}
					}
					if parentIsZfsMount {
						continue
					}
				}
			}

			deviceID := parseDeviceIdByUUID(mountSource)

			diskQuotas[i].MountPoint = stringutil.EscapeSpace(mountPoint)
			diskQuotas[i].DeviceID = deviceID
			diskQuotas[i].FileSystem = filesystem
			diskQuotas[i].MajorMinor = majorminor
			var stat syscall.Statfs_t
			syscall.Statfs(diskQuotas[i].MountPoint, &stat)

			diskQuotas[i].TotalSpace = uint64(stat.Bsize) * uint64(stat.Blocks)
			diskQuotas[i].UsedSpace = uint64(stat.Bsize) * (stat.Blocks - stat.Bfree)

			diskQuotas[i].UsedPercent = float32(100.0) - float32(100.0*float32(stat.Bavail)/float32(stat.Blocks-stat.Bfree+stat.Bavail))

			diskQuotas[i].FreeSpace = uint64(stat.Bsize) * uint64(stat.Bavail)
			diskQuotas[i].FreePercent = float32(100.0 * float32(stat.Bavail) / float32(stat.Blocks-stat.Bfree+stat.Bavail))
			diskQuotas[i].Blksize = int32(stat.Bsize)
			diskQuotas[i].InodeTotal = int64(stat.Files)
			diskQuotas[i].InodeUsed = int64(stat.Files - stat.Ffree)
			diskQuotas[i].InodeUsedPercent = float32(100.0) * float32(diskQuotas[i].InodeUsed) / float32(diskQuotas[i].InodeTotal)

			diskQuotas[i].MountOption = mountOption
			i++
		}
	}

	return ParseDiskIO(diskQuotas[:i])
}
