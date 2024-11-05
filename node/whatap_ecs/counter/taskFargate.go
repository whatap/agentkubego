package counter

import (
	"fmt"
	"runtime"

	"github.com/whatap/go-api/common/lang/value"
	"github.com/whatap/go-api/common/util/hash"
	"whatap.io/aws/ecs/config"
)

func (taskFargate *TaskFargate) init() {
	taskFargate.lastStatCache = map[string]*ContainerStat{}
	taskFargate.numcpu = runtime.NumCPU()
}

func (taskFargate *TaskFargate) interval() int {

	return 10
}

type ECSContainerInfo struct {
	id     string
	tags   map[string]string
	fields map[string]string
	cinfo  ECSContainer
	stats  ContainerStat
}

func newEcsContainerInfo(task ECSTaskResp,
	cinfo ECSContainer,
	stats ContainerStat) ECSContainerInfo {
	ret := ECSContainerInfo{
		id:    cinfo.DockerID,
		cinfo: cinfo,
		stats: stats,
	}
	ret.tags = map[string]string{}
	ret.fields = map[string]string{}

	ret.tags["TaskFamily"] = task.Family
	ret.tags["AvailabilityZone"] = task.AvailabilityZone
	ret.tags["Cluster"] = task.Cluster
	ret.fields["TaskDesiredStatus"] = task.DesiredStatus
	ret.fields["TaskKnownStatus"] = task.KnownStatus
	ret.tags["TaskRevision"] = task.Revision
	ret.tags["TaskARN"] = task.TaskARN

	return ret
}

func (taskFargate *TaskFargate) findContainersECSEndPoint() (ecscontainers []ECSContainerInfo,
	err error) {
	task, taskerr := EcsMetaV4Task()
	if taskerr != nil {
		err = taskerr
		return
	}
	stats, statserr := EcsMetaV4Stats()
	if taskerr != nil {
		err = statserr
		return
	}
	for _, c := range task.Containers {
		stat := stats[c.DockerID]
		c.Limits.CPU = int(task.Limits.CPU * 1000)
		c.Limits.Memory = int64(task.Limits.Memory * 1024 * 1024)
		ecsContInfo := newEcsContainerInfo(task, c, stat)
		ecscontainers = append(ecscontainers, ecsContInfo)
	}

	return
}

func (taskFargate *TaskFargate) process(now int64) (err error) {
	conf := config.GetConfig()
	containers, err := taskFargate.findContainersECSEndPoint()
	if err != nil {
		return err
	}

	for _, container := range containers {
		c := container.cinfo
		containerstats := container.stats
		if lastContainerstats, ok := taskFargate.lastStatCache[c.DockerID]; ok {

			totalCpu, userCpu, sysCpu := calcCpuUsage(&containerstats, lastContainerstats)

			sysCpuMillis := sysCpu * float32(taskFargate.numcpu) * float32(10)
			totalCpuMillis := totalCpu * float32(taskFargate.numcpu) * float32(10)
			userCpuMillis := userCpu * float32(taskFargate.numcpu) * float32(10)
			agentOid := conf.OID
			agentPcode := conf.PCODE

			p := createPack()
			p.Tags.PutString("dimension", fmt.Sprint("TaskARN=", container.tags["TaskARN"], ";containerId=", c.DockerID))
			p.Tags.PutString("containerId", c.DockerID)
			p.Tags.PutString("name", c.Name)
			p.Tags.Put("agentOid", value.NewDecimalValue(int64(agentOid)))
			p.Tags.Put("agentPcode", value.NewLongValue(agentPcode))
			p.Tags.PutString("KnownStatus", c.KnownStatus)
			p.Tags.Put("containerKey", value.NewDecimalValue(int64(hash.HashStr(c.DockerID))))
			p.Tags.Put("created", value.NewDecimalValue(int64(c.CreatedAt.UnixNano()/1000000)))
			p.Tags.PutString("image", c.Image)
			p.Tags.Put("imageHash", value.NewDecimalValue(int64(hash.HashStr(c.Image))))
			p.Tags.PutString("imageId", c.ImageID)

			p.Put("cpu_user", value.NewFloatValue(userCpu))
			p.Put("cpu_user_millis", value.NewFloatValue(userCpuMillis))
			p.Put("cpu_sys", value.NewFloatValue(sysCpu))
			p.Put("cpu_sys_millis", value.NewFloatValue(sysCpuMillis))
			p.Put("cpu_total", value.NewFloatValue(totalCpu))
			p.Put("cpu_total_millis", value.NewFloatValue(totalCpuMillis))
			p.Put("mem_usage", value.NewLongValue(containerstats.MemoryStats.Usage))
			p.Put("mem_totalrss", value.NewLongValue(containerstats.MemoryStats.Stats.TotalRss))
			blkioRbps, blkioRiops, blkioWbps, blkioWiops := calcBlkioUsage(&containerstats, lastContainerstats)
			p.Put("blkio_rbps", value.NewFloatValue(blkioRbps))
			p.Put("blkio_riops", value.NewFloatValue(blkioRiops))
			p.Put("blkio_wbps", value.NewFloatValue(blkioWbps))
			p.Put("blkio_wiops", value.NewFloatValue(blkioWiops))
			workingSet := containerstats.MemoryStats.Usage - containerstats.MemoryStats.Stats.InactiveFile
			cpuPerQuota := parseCpuPer(taskFargate.numcpu, totalCpu, int32(c.Limits.CPU))
			p.Put("cpu_per_quota", value.NewFloatValue(cpuPerQuota))
			var memPerQuota float32
			if c.Limits.Memory > 0 {
				memPerQuota = float32(workingSet) / float32(c.Limits.Memory) * float32(100)
			}

			cpuQuotaPercent := float32(100) * float32(c.Limits.CPU) / float32(taskFargate.numcpu*1000)
			netRbps, netRiops, netWbps, netWiops := calcNetUsage(&containerstats, lastContainerstats)
			nodeCpu, nodeMem := getNodePerf()

			p.Put("mem_percent", value.NewFloatValue(memPerQuota))
			p.Put("cpu_quota", value.NewDecimalValue(int64(c.Limits.CPU)))
			p.Put("cpu_quota_percent", value.NewFloatValue(cpuQuotaPercent))
			p.Put("mem_limit", value.NewDecimalValue(int64(c.Limits.Memory)))
			p.Put("cpu_throttledperiods", value.NewLongValue(containerstats.CPUStats.ThrottlingData.ThrottledPeriods))
			p.Put("cpu_throttledtime", value.NewLongValue(containerstats.CPUStats.ThrottlingData.ThrottledTime))
			p.Put("mem_failcnt", value.NewDecimalValue(int64(containerstats.MemoryStats.FailCnt)))
			p.Put("mem_maxusage", value.NewLongValue(containerstats.MemoryStats.MaxUsage))
			p.Put("mem_totalcache", value.NewLongValue(containerstats.MemoryStats.Stats.TotalCache))
			p.Put("mem_totalpgfault", value.NewLongValue(containerstats.MemoryStats.Stats.TotalPgfault))
			p.Put("mem_totalrss_percent", value.NewFloatValue(memPerQuota))
			p.Put("mem_totalunevictable", value.NewLongValue(containerstats.MemoryStats.Stats.TotalUnevictable))
			p.Put("mem_workingset", value.NewLongValue(workingSet))
			p.Put("mem_inactivefile", value.NewLongValue(containerstats.MemoryStats.Stats.InactiveFile))

			p.Put("network_rbps", value.NewFloatValue(netRbps))
			p.Put("network_rdropped", value.NewLongValue(containerstats.NetworkStats.RxDropped))
			p.Put("network_rerror", value.NewLongValue(containerstats.NetworkStats.RxErrors))
			p.Put("network_riops", value.NewFloatValue(netRiops))
			p.Put("network_wbps", value.NewFloatValue(netWbps))
			p.Put("network_wdropped", value.NewLongValue(containerstats.NetworkStats.TxDropped))
			p.Put("network_werror", value.NewLongValue(containerstats.NetworkStats.TxErrors))
			p.Put("network_wiops", value.NewFloatValue(netWiops))
			p.Put("node_cpu", value.NewFloatValue(nodeCpu))
			p.Put("node_mem", value.NewFloatValue(nodeMem))

			for k, v := range container.tags {
				p.Tags.PutString(k, v)
			}

			for k, v := range container.fields {
				p.Data.PutString(k, v)
			}

			//fmt.Println(dateutil.DateTime(p.Time), p.Category, p.Tags.ToString(), p.Data.ToString())

			send(p)
		}
		taskFargate.lastStatCache[c.DockerID] = &containerstats
	}
	return
}
