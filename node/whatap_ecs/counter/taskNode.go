package counter

import (
	"runtime"
	"time"

	"github.com/whatap/go-api/common/util/hash"
	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/lang/pack"
	"whatap.io/aws/ecs/osinfo"
	"whatap.io/aws/ecs/text"
)

func (this *TaskKubeNode) init() {
	parseMemoryStat(func(k string, v int64) {
		if k == "memtotal" {
			this.totalMemory = v
		}
	})
	this.numcpu = int32(runtime.NumCPU())
}

func (this *TaskKubeNode) interval() int {

	return 10
}

func (this *TaskKubeNode) process(now int64) error {
	conf := config.GetConfig()
	nodepack := pack.NewKubeNodePack()
	nodepack.Pcode = conf.PCODE
	nodepack.Oid = conf.OID
	nodepack.Onode = hash.HashStr(conf.ONODE)
	nodepack.Time = now

	nodepack.HostIp = getMyAddr()
	nodepack.Starttime = this.starttime
	nodepack.SysCores = this.numcpu
	nodepack.SysMem = this.totalMemory
	nodepack.ContainerKey = this.containerKey
	sendHide(nodepack)

	return nil
}

func (this *TaskNode) interval() int {

	return 5
}

func (this *TaskNode) process(now int64) error {
	conf := config.GetConfig()
	basepack := pack.NewSMBasePack()
	basepack.Pcode = conf.PCODE
	basepack.Oid = conf.OID
	basepack.Time = now
	basepack.OS = pack.OS_KUBE_NODE

	basepack.Cpu, basepack.CpuCore = osinfo.GetCPUUtil()
	basepack.Memory = osinfo.GetMemoryUtil()
	basepack.EpochTime = time.Now().Unix()
	basepack.IP = getMyAddr()

	setNodePerf(basepack.Cpu.Percent(), basepack.Memory.Percent())

	osinfo.GetDisk()

	diskpack := pack.NewSMDiskPerfPack()
	diskpack.Pcode = conf.PCODE
	diskpack.Oid = conf.OID
	diskpack.Time = now
	diskpack.OS = pack.OS_KUBE_NODE
	disks, _ := osinfo.GetDisk()
	n := make([]pack.DiskPerf, len(disks))
	for i := 0; i < len(n); i++ {
		n[i].Blksize = disks[i].Blksize
		n[i].DeviceID = hash.HashStr(disks[i].DeviceID)
		text.SendHashText(pack.TEXT_SYS_DEVICE_ID, n[i].DeviceID, disks[i].DeviceID)
		n[i].FileSystem = hash.HashStr(disks[i].FileSystem)
		text.SendHashText(pack.TEXT_SYS_FILE_SYSTEM, n[i].FileSystem, disks[i].FileSystem)
		n[i].FreePercent = disks[i].FreePercent
		n[i].FreeSpace = int64(disks[i].FreeSpace)
		n[i].MountPoint = hash.HashStr(disks[i].MountPoint)
		text.SendHashText(pack.TEXT_SYS_MOUNT_POINT, n[i].MountPoint, disks[i].MountPoint)
		n[i].ReadBps = disks[i].ReadBps
		n[i].ReadIops = disks[i].ReadIops
		n[i].TotalSpace = int64(disks[i].TotalSpace)
		n[i].UsedPercent = disks[i].UsedPercent
		n[i].UsedSpace = int64(disks[i].UsedSpace)
		n[i].WriteBps = disks[i].WriteBps
		n[i].WriteIops = disks[i].WriteIops
		n[i].IOPercent = disks[i].IOPercent
		n[i].QueueLength = disks[i].QueueLength
		n[i].InodeTotal = disks[i].InodeTotal
		n[i].InodeUsed = disks[i].InodeUsed
		n[i].InodeUsedPercent = disks[i].InodeUsedPercent
		if len(disks[i].MountOption) > 0 {
			n[i].MountOption = hash.HashStr(disks[i].MountOption)
			text.SendHashText(pack.TEXT_SYS_MOUNT_OPTION, n[i].MountOption, disks[i].MountOption)
		}
	}
	diskpack.Disk = n

	textCallback := func(div int32, h int32, src string) {
		text.SendHashText(byte(div), h, src)

	}

	nicpack := pack.NewSMNetPerfPack()
	nicpack.Pcode = conf.PCODE
	nicpack.Oid = conf.OID
	nicpack.Time = now
	nicpack.OS = pack.OS_KUBE_NODE
	nicpack.Net = osinfo.GetNicUtil(textCallback)

	send(basepack)
	send(diskpack)
	send(nicpack)

	return nil
}
