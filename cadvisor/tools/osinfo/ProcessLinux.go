//go:build linux
// +build linux

package osinfo

import "C"
import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/whatap/golib/io"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	"github.com/whatap/kube/cadvisor/tools/util/parseutil"
	"github.com/whatap/kube/tools/util/fileutil"
	"github.com/whatap/kube/tools/util/iputil"
	"github.com/whatap/kube/tools/util/logutil"
	"github.com/whatap/kube/tools/util/stringutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

/*
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <stdlib.h>
*/
import "C"

func load(path string) *map[string]*ProcessInfo {
	loadedCache := LoadCache(path)
	if loadedCache != nil {
		castedloadedCache := loadedCache.(map[string]*ProcessInfo)
		return &castedloadedCache
	}

	return nil
}

func save(path string, fileList *map[string]*ProcessInfo) {
	SaveCache(path, *fileList)
}

type ProcessInfo struct {
	PPid          int
	Pid           int
	Cpu           float64
	ChildCpu      float64
	MemoryBytes   int64
	MemoryPercent float32
	Timestamp     int
	Cmd1          string
	Cmd2          string
	User          string
	State         string
	CreateTime    int
	TotalCpuTime  int64
	SharedMemory  int64
	Vsz           uint64
	Rss           int64
	Thcount       int
	ReadOctets    int64
	WriteOctets   int64
	ReadBytes     int64
	WriteBytes    int64
	Rchar         int64
	Wchar         int64
	Syscr         int64
	Syscw         int64
	Excluded      bool
}

// String 메서드 추가
func (p *ProcessPerfInfo) String() string {
	return fmt.Sprintf(
		"PPid: %d, Pid: %d, Cpu: %.2f, MemoryBytes: %d, MemoryPercent: %.2f,Cmd1: %s, Cmd2: %s, User: %s, State: %s, CreateTime: %d, MemoryShared: %d, OpenFileDescriptors: %d, Rss: %d, Vsz: %d, Thcount: %d",
		p.PPid, p.Pid, p.Cpu, p.MemoryBytes, p.MemoryPercent, p.Cmd1, p.Cmd2, p.User, p.State, p.CreateTime, p.MemoryShared, p.OpenFileDescriptors, p.Rss, p.Vsz, p.Thcount)
}

func GetProcessPerfList() *[]*ProcessPerfInfo {
	filepathWps := "/tmp/whatap_agent_wps"
	oldProcessInfoMap := load(filepathWps)
	newProcessInfoMap := measureProcessPerformance()
	if whatap_config.GetConfig().Debug {
		logutil.Infof("GPP", "Process performance info collected(newProcessInfoMap): %d processes", len(*newProcessInfoMap))
	}
	save(filepathWps, newProcessInfoMap)
	processPerfs := getProcessPerfList(oldProcessInfoMap, newProcessInfoMap)
	if whatap_config.GetConfig().Debug {
		logutil.Infof("GPPAAA", "Process performance info collected: %d processes", len(*processPerfs))
	}
	if processPerfs == nil || len(*processPerfs) < 1 {
		time.Sleep(1 * time.Second)
		newProcessInfoMap2 := measureProcessPerformance()
		processPerfs = getProcessPerfList(newProcessInfoMap, newProcessInfoMap2)
	}
	if whatap_config.GetConfig().CollectProcessFD {
		populateNetworkStatus(processPerfs)
	}

	return processPerfs
}

func getProcessPerfList(oldPinfoMap *map[string]*ProcessInfo, newPinfoMap *map[string]*ProcessInfo) *[]*ProcessPerfInfo {
	if whatap_config.GetConfig().Debug {
		logutil.Infof("GPL", "start")
	}
	var processPerfInfos []*ProcessPerfInfo
	if oldPinfoMap != nil {
		for pid := range *newPinfoMap {
			newPinfo := (*newPinfoMap)[pid]
			oldPinfo := (*oldPinfoMap)[pid]
			if whatap_config.GetConfig().Debug {
				logutil.Infof("GPL", "newPinfo.User=%v, newPinfo.Cmd1=%v, newPinfo.Cmd2=%v", newPinfo.User, newPinfo.Cmd1, newPinfo.Cmd2)
			}
			//verify same process by pid, createtime, cmd1, cmd2
			if oldPinfo != nil && ((oldPinfo.CreateTime == newPinfo.CreateTime) || (oldPinfo.Cmd1 == newPinfo.Cmd1 && oldPinfo.Cmd2 == newPinfo.Cmd2)) {
				//timeelapsed := float64(newPinfo.Timestamp - oldPinfo.Timestamp)
				totalCpuDiff := float64(newPinfo.TotalCpuTime - oldPinfo.TotalCpuTime)
				cpuClocks := newPinfo.Cpu - oldPinfo.Cpu
				pcpu := 100.0 * float32(cpuClocks/totalCpuDiff)
				//readbps := float32(newPinfo.ReadBytes-oldPinfo.ReadBytes) / float32(timeelapsed)
				//writebps := float32(newPinfo.WriteBytes-oldPinfo.WriteBytes) / float32(timeelapsed)
				//rchar := float32(newPinfo.Rchar-oldPinfo.Rchar) / float32(timeelapsed)
				//wchar := float32(newPinfo.Wchar-oldPinfo.Wchar) / float32(timeelapsed)
				//syscr := float32(newPinfo.Syscr-oldPinfo.Syscr) / float32(timeelapsed)
				//syscw := float32(newPinfo.Syscw-oldPinfo.Syscw) / float32(timeelapsed)
				processPerfInfos = append(processPerfInfos, &ProcessPerfInfo{User: newPinfo.User, PPid: int32(newPinfo.PPid), Pid: int32(newPinfo.Pid), State: newPinfo.State,
					CreateTime: int64(newPinfo.CreateTime), Cpu: pcpu,
					MemoryBytes: int64(newPinfo.MemoryBytes), MemoryPercent: newPinfo.MemoryPercent,
					//WriteBps: writebps, ReadBps: readbps,
					//Cmd1:         excludeProcessExe(newPinfo.User, newPinfo.Cmd1, newPinfo.Cmd2),
					//Cmd2:         excludeProcessCmdline(newPinfo.User, newPinfo.Cmd1, newPinfo.Cmd2),
					Cmd1:         newPinfo.Cmd1,
					Cmd2:         newPinfo.Cmd2,
					MemoryShared: newPinfo.SharedMemory,
					Rss:          uint64(newPinfo.MemoryBytes), Vsz: uint64(int64(newPinfo.Vsz)),
					Thcount: newPinfo.Thcount,
					//Rchar:   rchar, Wchar: wchar, Syscr: syscr, Syscw: syscw,
				})
			}
		}
	}
	return &processPerfInfos
}

func measureProcessPerformance() *map[string]*ProcessInfo {
	if whatap_config.GetConfig().Debug {
		logutil.Infof("MPP", "Start")
	}

	clktck := C.sysconf(C._SC_CLK_TCK)
	pagesize := C.sysconf(C._SC_PAGE_SIZE)
	memorysize := C.sysconf(C._SC_PHYS_PAGES) * C.sysconf(C._SC_PAGE_SIZE)
	targetList := whatap_config.GetConfig().CollectKubeNodeProcessMetricTargetList
	if whatap_config.GetConfig().Debug {
		logutil.Infof("MPP", "targetList=%v", targetList)
	}
	hostPathPrefix := whatap_config.GetConfig().HostPathPrefix
	procPath := filepath.Join(hostPathPrefix, "proc")
	procUptimePath := filepath.Join(procPath, "uptime")
	uptimeBytes, err := os.ReadFile(procUptimePath)
	if err != nil {
		if whatap_config.GetConfig().Debug {
			logutil.Errorf("MPP", "Failed to read uptime file: %v", err)
		}
		return nil
	}
	uptimeContent := string(uptimeBytes)
	uptime, err := strconv.ParseFloat(strings.Split(uptimeContent, " ")[0], 10)
	if err != nil {
		if whatap_config.GetConfig().Debug {
			logutil.Errorf("MPP", "Failed to parse uptime: %v", err)
		}
		return nil
	}

	if whatap_config.GetConfig().Debug {
		logutil.Infof("MPP", "System uptime: %f", uptime)
	}

	uidmap := map[string]string{}
	passwdPath := filepath.Join(whatap_config.GetConfig().HostPathPrefix, "etc", "passwd")
	passwdBytes, err := os.ReadFile(passwdPath)
	if err == nil {
		passwdContent := string(passwdBytes)
		for _, line2 := range strings.Split(passwdContent, "\n") {
			userTokens := strings.Split(line2, ":")
			if len(userTokens) > 2 {
				uidmap[userTokens[2]] = userTokens[0]
			}
		}
	} else {
		if whatap_config.GetConfig().Debug {
			logutil.Errorf("MPP", "Failed to read passwd file: %v", err)
		}
	}

	searchDir := procPath

	fileList := make(map[string]*ProcessInfo)
	totalCpuTime, err := getTotalCpuTime()
	if err != nil {
		if whatap_config.GetConfig().Debug {
			logutil.Errorf("MPP", "Failed to get total CPU time: %v", err)
		}
		return &fileList
	}

	if whatap_config.GetConfig().Debug {
		logutil.Infof("MPP", "Total CPU time: %f", totalCpuTime)
	}

	filesInfo, err := os.ReadDir(searchDir)
	if err != nil {
		if whatap_config.GetConfig().Debug {
			logutil.Errorf("MPP", "Failed to read /proc directory: %v", err)
		}
		return &fileList
	}

	for i := 0; i < len(filesInfo) && err == nil; i++ {
		fileInfo := filesInfo[i]
		if !fileInfo.IsDir() {
			continue
		}

		pid := fileInfo.Name()
		ipid, pidConvertErr := strconv.Atoi(pid)
		if pidConvertErr != nil {
			if whatap_config.GetConfig().Debug {
				logutil.Infof("MPP", "Failed to convert PID: %s", pid)
			}
			continue
		}
		if ipid == 2 {
			if whatap_config.GetConfig().Debug {
				logutil.Infof("MPP", "Skipping process with PID 2")
			}
			continue
		}

		pinfo := new(ProcessInfo)
		pinfo.Pid = ipid
		pinfo.Timestamp = int(time.Now().Unix())
		pinfo.TotalCpuTime = totalCpuTime

		if whatap_config.GetConfig().Debug {
			logutil.Infof("MPP", "Processing PID: %d", pinfo.Pid)
		}

		statusBytes, readStatusErr := os.ReadFile(strings.Join([]string{searchDir, pid, "status"}, "/"))
		if readStatusErr == nil {
			statusContent := string(statusBytes)
			for _, line := range strings.Split(statusContent, "\n") {
				if strings.HasPrefix(line, "Uid:") {
					uid := strings.Split(line, "\t")[1]
					pinfo.User = uidmap[uid]
				} else if strings.HasPrefix(line, "Name:") {
					pinfo.Cmd1 = strings.Split(line, "\t")[1]
					if pinfo.Cmd1 == "" || !stringutil.StringInSlice(pinfo.Cmd1, whatap_config.GetConfig().CollectKubeNodeProcessMetricTargetList) {
						pinfo.Excluded = true
						if whatap_config.GetConfig().Debug {
							logutil.Infof("MPP", "Skipping process with PID %d and Name '%s': Not in PorcessTarget list or Name is empty", pinfo.Pid, pinfo.Cmd1)
						}
						break
					} else {
						if whatap_config.GetConfig().Debug {
							logutil.Infof("MPP", "Hanlding process with PID %d and Name '%s': in PorcessTarget list", pinfo.Pid, pinfo.Cmd1)
						}
					}
				} else if strings.HasPrefix(line, "State:") {
					pinfo.State = strings.Split(line, "\t")[1]
				} else if strings.HasPrefix(line, "PPid") {
					pinfo.PPid, _ = strconv.Atoi(strings.Split(line, "\t")[1])
					if pinfo.PPid == 2 {
						pinfo.Excluded = true
						if whatap_config.GetConfig().Debug {
							logutil.Infof("MPP", "Skipping process with PPID 2: %d", pinfo.Pid)
						}
						break
					}
				} else if strings.HasPrefix(line, "VmSize") {
					vmSize := strings.Split(line, "\t")[1]
					vmSize = strings.ReplaceAll(vmSize, " ", "")
					pinfo.Vsz, _ = humanize.ParseBytes(vmSize)
				} else if strings.HasPrefix(line, "Threads") {
					pinfo.Thcount, _ = strconv.Atoi(strings.Split(line, "\t")[1])
				}
			}
		} else {
			pinfo.Excluded = true
			if whatap_config.GetConfig().Debug {
				logutil.Errorf("MPP", "Failed to read status for PID: %d", pinfo.Pid)
			}
			continue
		}

		if pinfo != nil && pinfo.Excluded {
			continue
		}

		statBytes, readStatErr := os.ReadFile(strings.Join([]string{searchDir, pid, "stat"}, "/"))
		if readStatErr == nil {
			statContent := string(statBytes)
			statContents := strings.Split(statContent, " ")
			pcpu1, _ := strconv.ParseInt(statContents[13], 10, 64)
			pcpu2, _ := strconv.ParseInt(statContents[14], 10, 64)
			pinfo.Cpu = float64(pcpu1 + pcpu2)

			pcpu3, _ := strconv.ParseInt(statContents[15], 10, 64)
			pcpu4, _ := strconv.ParseInt(statContents[16], 10, 64)
			pinfo.ChildCpu = float64(pcpu3 + pcpu4)

			createElapsedTime, _ := strconv.ParseInt(statContents[21], 10, 32)
			createElapsedTime = createElapsedTime / int64(clktck)
			pinfo.CreateTime = pinfo.Timestamp - int(uptime) + int(createElapsedTime)
		} else {
			if whatap_config.GetConfig().Debug {
				logutil.Errorf("MPP", "Failed to read stat for PID: %d", pinfo.Pid)
			}
		}

		cmdContent, readCmdLineErr := os.ReadFile(strings.Join([]string{searchDir, pid, "cmdline"}, "/"))
		if readCmdLineErr == nil {
			if len(cmdContent) > 0 {
				buf := new(bytes.Buffer)

				for c := range cmdContent {
					if cmdContent[c] > 0 {
						buf.WriteByte(cmdContent[c])
					} else {
						buf.WriteByte(' ')
					}
				}
				pinfo.Cmd2 = string(buf.Bytes())
			}
		} else {
			if whatap_config.GetConfig().Debug {
				logutil.Errorf("MPP", "Failed to read cmdline for PID: %d", pinfo.Pid)
			}
		}

		if whatap_config.GetConfig().CollectProcessPssEnabled && stringutil.StringInSlice(pinfo.Cmd1, whatap_config.GetConfig().CollectProcessPssTargetList) {
			parseutil.ParseKeyValue(pid, "smaps", func(key string, val int64) {
				if key == "Pss" {
					pinfo.MemoryBytes += val
					pinfo.Rss = pinfo.MemoryBytes
				}
			})
			pinfo.MemoryPercent = float32(float64(pinfo.MemoryBytes) / float64(memorysize) * 100.0)
		}

		memoryBytes, readStatmErr := os.ReadFile(strings.Join([]string{searchDir, pid, "statm"}, "/"))
		if readStatmErr == nil {
			memoryContent := string(memoryBytes)
			tokens := strings.Fields(memoryContent)
			if len(tokens) > 2 {
				rssPages, parseTokenErr := strconv.ParseInt(tokens[1], 10, 64)
				if parseTokenErr == nil {
					pinfo.MemoryBytes = int64(rssPages) * int64(pagesize)
					pinfo.MemoryPercent = float32(float64(pinfo.MemoryBytes) / float64(memorysize) * 100.0)
					pinfo.Rss = pinfo.MemoryBytes
				} else {
					if whatap_config.GetConfig().Debug {
						logutil.Errorf("MPP", "Failed to parse RSS pages for PID: %d", pinfo.Pid)
					}
				}
				sharedPages, parseIntErr := strconv.ParseInt(tokens[2], 10, 64)
				if parseIntErr == nil {
					pinfo.SharedMemory = int64(sharedPages) * int64(pagesize)
				} else {
					if whatap_config.GetConfig().Debug {
						logutil.Errorf("MPP", "Failed to parse shared memory pages for PID: %d", pinfo.Pid)
					}
				}
			}
		} else {
			if whatap_config.GetConfig().Debug {
				logutil.Errorf("MPP", "Failed to read statm for PID: %d", pinfo.Pid)
			}
		}

		if whatap_config.GetConfig().CollectProcessIO {
			ioBytes, readIoFileErr := os.ReadFile(strings.Join([]string{searchDir, pid, "io"}, "/"))
			if readIoFileErr == nil {
				ioContent := string(ioBytes)
				for _, line := range strings.Split(ioContent, "\n") {
					if strings.HasPrefix(line, "readBytes") {
						readBytes, err := strconv.ParseInt(strings.Split(line, " ")[1], 10, 64)
						if err == nil {
							pinfo.ReadBytes = readBytes
						}
					} else if strings.HasPrefix(line, "writeBytes") {
						writeBytes, err := strconv.ParseInt(strings.Split(line, " ")[1], 10, 64)
						if err == nil {
							pinfo.WriteBytes = writeBytes
						}
					} else if strings.HasPrefix(line, "rchar") {
						rchar, err := strconv.ParseInt(strings.Split(line, " ")[1], 10, 64)
						if err == nil {
							pinfo.Rchar = rchar
						}
					} else if strings.HasPrefix(line, "wchar") {
						wchar, err := strconv.ParseInt(strings.Split(line, " ")[1], 10, 64)
						if err == nil {
							pinfo.Wchar = wchar
						}
					} else if strings.HasPrefix(line, "syscr") {
						syscr, err := strconv.ParseInt(strings.Split(line, " ")[1], 10, 64)
						if err == nil {
							pinfo.Syscr = syscr
						}
					} else if strings.HasPrefix(line, "syscw") {
						syscw, err := strconv.ParseInt(strings.Split(line, " ")[1], 10, 64)
						if err == nil {
							pinfo.Syscw = syscw
						}
					}
				}
			} else {
				if whatap_config.GetConfig().Debug {
					logutil.Errorf("MPP", "Failed to read io for PID: %d", pinfo.Pid)
				}
			}
		}
		if whatap_config.GetConfig().Debug {
			logutil.Infof("MPP", "fileList[%v]=%v", pinfo.Pid, pinfo)
		}
		fileList[pid] = pinfo
	}

	return &fileList
}

func populateNetworkStatus(processPerfs *[]*ProcessPerfInfo) {
	var localips []uint32
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, i := range ifaces {
			if addrs, err := i.Addrs(); err == nil {
				for _, addr := range addrs {
					var ip net.IP
					switch v := addr.(type) {
					case *net.IPNet:
						ip = v.IP
					case *net.IPAddr:
						ip = v.IP
					}
					if ip.To4() != nil {
						localips = append(localips, binary.BigEndian.Uint32(ip.To4()))
					}
				}
			}
		}
	}

	tcpmaxlength := int64(150 * 7000)
	inodeportlookup := map[int64]*IpPort{}
	ipinodelookup := map[int64]*IpPort{}
	var remoteconns []*IpPort
	netTcpPath := filepath.Join(whatap_config.GetConfig().HostPathPrefix, "/proc/net/tcp")
	netTcpSixPath := filepath.Join(whatap_config.GetConfig().HostPathPrefix, "/proc/net/tcp6")
	for _, tcpfilepath := range []string{netTcpPath, netTcpSixPath} {
		tcpBytes, nbytethistime, err := fileutil.ReadFile(tcpfilepath, int64(tcpmaxlength))
		// log.Println("ProcessLinux step -1 ", err,nbytethistime,  nbytethistime < tcpmaxlength, tcpfilepath)
		if err == nil && nbytethistime < tcpmaxlength {
			tcpContent := string(tcpBytes)
			for i, line := range strings.Split(tcpContent, "\n") {
				if i == 0 || len(line) < 1 {
					continue
				}
				words := strings.Fields(line)
				if len(words) < 17 {
					// log.Println("skipping", line )
					continue
				}
				inode, _ := strconv.ParseInt(words[9], 10, 32)
				localIPBytes, _ := iputil.ParseHexString(words[1])
				localIP := io.ToInt(localIPBytes, 0)
				localPort := io.ToShort(localIPBytes, 4)
				localIPPortInt := io.ToLong6(localIPBytes, 0)
				remoteIPBytes, _ := iputil.ParseHexString(words[2])
				remoteIPInt := io.ToInt(remoteIPBytes, 0)
				remotePortInt := io.ToShort(remoteIPBytes, 4)
				ipport := IpPort{localip: localIP, localport: localPort, conn: 0, ilocaladdr: localIPPortInt, inode: int32(inode)}

				// log.Println(i, "remote ", iputil.ToStringFrInt(remoteIPInt) , remotePortInt)
				if remoteIPInt == 0 && remotePortInt == 0 {
					// log.Println("listen port", localPort)
					inodeportlookup[inode] = &ipport
					if localIPPortInt > 0 {
						ipinodelookup[localIPPortInt] = &ipport
						for _, ip := range localips {
							ipbytes := io.ToBytesInt(int32(ip))
							deviceIPPortBytes := make([]byte, 6)
							deviceIPPortBytes[0] = ipbytes[0]
							deviceIPPortBytes[1] = ipbytes[1]
							deviceIPPortBytes[2] = ipbytes[2]
							deviceIPPortBytes[3] = ipbytes[3]
							deviceIPPortBytes[4] = localIPBytes[4]
							deviceIPPortBytes[5] = localIPBytes[5]
							deviceIPPortInt := io.ToLong6(deviceIPPortBytes, 0)
							ipinodelookup[deviceIPPortInt] = &ipport
							//fmt.Println(iputil.ToStringFrInt(int32(ip)), io.ToShort(deviceIPPortBytes, 4))
						}
					}

				} else {
					// log.Println("remote conn to local ",ipport.localport)
					remoteconns = append(remoteconns, &ipport)
				}
			}
		}
	}

	for _, pipport := range remoteconns {
		ipport := ipinodelookup[pipport.ilocaladdr]
		if ipport != nil {
			ipport.conn++
		}
	}

	totalFileDescriptors := int32(0)
	for _, processPerf := range *processPerfs {
		var ports []ProcNetPerf
		var filesinfo []*FileInfo
		//panicutil.Debug("Checking Process FD pid:", processPerf.Pid)
		searchDirPath := filepath.Join(whatap_config.GetConfig().HostPathPrefix, "proc")
		searchDir := strings.Join([]string{searchDirPath, strconv.FormatInt(int64(processPerf.Pid), 10), "fd"}, "/")
		_ = filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
			link, _ := os.Readlink(path)

			totalFileDescriptors += 1
			if strings.HasPrefix(link, "socket:[") {
				inode, _ := strconv.ParseInt(link[8:len(link)-1], 10, 64)
				ipport := inodeportlookup[inode]
				if ipport != nil {
					ports = append(ports, ProcNetPerf{LocalIP: ipport.localip, LocalPort: ipport.localport, Conn: ipport.conn})
				}
			} else if !strings.HasPrefix(link, "/dev") && len(link) > 0 && link[0] == '/' {
				filestat, err := os.Stat(link)
				if err == nil && strings.HasSuffix(link, ".log") {
					fileinfo := FileInfo{Name: link, Size: filestat.Size()}
					filesinfo = append(filesinfo, &fileinfo)
				}
			}
			if len(link) > 0 {
				processPerf.OpenFileDescriptors += 1
			}

			return nil
		})
		processPerf.Net = ports
		processPerf.File = filesinfo

		PutIntValue("TotalFileDescriptors", totalFileDescriptors)
	}

	return
}
func getTotalCpuTime() (int64, error) {
	hostPathPrefix := whatap_config.GetConfig().HostPathPrefix
	procStatPath := filepath.Join(hostPathPrefix, "proc", "stat")
	file, err := os.Open(procStatPath)
	if err != nil {
		return 0, err
	}
	defer func(file *os.File) {
		closeErr := file.Close()
		if closeErr != nil {
			logutil.Errorf("WHA-PL-ERR-001", "closeErr=%v", closeErr)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu") {
			words := strings.Fields(line)

			device := words[0]
			if device == "cpu" {
				user, _ := strconv.ParseInt(words[1], 10, 64)
				nice, _ := strconv.ParseInt(words[2], 10, 64)
				system, _ := strconv.ParseInt(words[3], 10, 64)
				idle, _ := strconv.ParseInt(words[4], 10, 64)
				iowait, _ := strconv.ParseInt(words[5], 10, 64)
				irq, _ := strconv.ParseInt(words[6], 10, 64)
				softirq, _ := strconv.ParseInt(words[7], 10, 64)
				steal, _ := strconv.ParseInt(words[8], 10, 64)
				totalCpuTime := user + nice + system + idle + iowait + irq + softirq + steal

				return totalCpuTime, nil
			}

		}
	}
	return 1, fmt.Errorf("cannot parse %v/proc/stat for total cpu time ", hostPathPrefix)
}

var (
	ProcessCmdPatternEnabled = true
	patternInit              = false
	excludePatterns          = []*CmdPattern{&CmdPattern{user: "oracle", cmd1: "(.+)_.+_.+", cmd2: ".*"},
		&CmdPattern{user: ".+", cmd1: "oracle", cmd2: ".+_.+_.+"}}
)

func excludeProcessExe(user string, exe string, cmdline string) (ret string) {
	ret = exe
	if !ProcessCmdPatternEnabled {
		return
	}

	for _, pattern := range excludePatterns {
		match := pattern.matchExe(user, exe, cmdline, func(cmdlinePrefix string) {
			ret = cmdlinePrefix
		})
		if match {
			//fmt.Println("match found ret:", ret)
			return
		}
	}

	return
}

func excludeProcessCmdline(user string, exe string, cmdline string) (ret string) {
	ret = cmdline
	if !ProcessCmdPatternEnabled {
		return
	}

	for _, pattern := range excludePatterns {
		match := pattern.matchCmdline(user, exe, cmdline)
		if match {
			//fmt.Println("match found ret:", ret)
			return ""
		}
	}

	return
}
