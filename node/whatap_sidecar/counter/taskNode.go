package counter

import (
	"runtime"
	"time"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/hash"
	sidecarpack "whatap.io/k8s/sidecar/lang/pack"
	"whatap.io/k8s/sidecar/osinfo"
	"whatap.io/k8s/sidecar/text"
)

func (this *TaskKubeNode) init() {
	parseMemoryStat(func(k string, v int64) {
		if k == "memtotal" {
			this.totalMemory = v
		}
	})
	this.numcpu = int32(runtime.NumCPU())
	this.containerKey = hash.HashStr(getSelfContainerId())
}

func (this *TaskKubeNode) interval() int {

	return 10
}

func (this *TaskKubeNode) process(now int64) error {
	nodepack := sidecarpack.NewKubeNodePack()
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

	findNamespace(nodeNamespace, func(pcode int64, licenseHash64 int64) {
		if conf.PCODE != pcode {
			nodepack.Pcode = pcode
			sendOneway(pcode, licenseHash64, nodepack)
		}
	})

	return nil
}

func (this *TaskNode) interval() int {

	return 5
}

func (this *TaskNode) process(now int64) error {
	var nsPcode, nsLicenseHash64 int64
	findNamespace(nodeNamespace, func(pcode int64, licenseHash64 int64) {
		if conf.PCODE != pcode {
			nsPcode = pcode
			nsLicenseHash64 = licenseHash64
		}
	})

	basepack := sidecarpack.NewSMBasePack()
	basepack.Pcode = conf.PCODE
	basepack.Oid = conf.OID
	basepack.Time = now
	basepack.OS = sidecarpack.OS_KUBE_NODE

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
	diskpack.OS = sidecarpack.OS_KUBE_NODE
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
			text.SendHashText(sidecarpack.TEXT_SYS_MOUNT_OPTION, n[i].MountOption, disks[i].MountOption)

			if nsPcode != 0 {
				text.SendHashTextOneway(nsPcode, nsLicenseHash64, sidecarpack.TEXT_SYS_MOUNT_OPTION, n[i].MountOption, disks[i].MountOption)
			}
		}
	}
	diskpack.Disk = n

	textCallback := func(div int32, h int32, src string) {
		text.SendHashText(byte(div), h, src)

		if nsPcode != 0 {
			text.SendHashTextOneway(nsPcode, nsLicenseHash64, byte(div), h, src)
		}
	}

	nicpack := sidecarpack.NewSMNetPerfPack()
	nicpack.Pcode = conf.PCODE
	nicpack.Oid = conf.OID
	nicpack.Time = now
	nicpack.OS = sidecarpack.OS_KUBE_NODE
	nicpack.Net = osinfo.GetNicUtil(textCallback)

	send(basepack)
	send(diskpack)
	send(nicpack)

	if nsPcode != 0 {
		basepack.Pcode = nsPcode
		diskpack.Pcode = nsPcode
		nicpack.Pcode = nsPcode

		sendOneway(nsPcode, nsLicenseHash64, basepack)
		sendOneway(nsPcode, nsLicenseHash64, diskpack)
		sendOneway(nsPcode, nsLicenseHash64, nicpack)
	}

	return nil
}
