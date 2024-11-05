package counter

import (
	"fmt"
	"log"
	"runtime"

	"github.com/whatap/go-api/common/lang/value"
	"github.com/whatap/go-api/common/util/hash"
	"whatap.io/aws/ecs/cache"
	"whatap.io/aws/ecs/config"
)

var (
	NodeVolPrefix string = "/rootfs"
)

func (this *TaskContainer) init() {
	this.lastStatCache = map[string]*ContainerStat{}
	parseMemoryStat(func(k string, v int64) {
		if k == "memtotal" {
			this.totalMemory = v
		}
	})
	this.numcpu = runtime.NumCPU()

	// containers, err := findContainers()
	// if err != nil {
	// 	panic(err)
	// }
	// this.containers = containers

	return
}

func (this *TaskContainer) interval() int {

	return 5
}

func (this *TaskContainer) process(now int64) error {
	conf := config.GetConfig()
	containers, err := findContainers()
	if err != nil {
		return err
	}
	for _, c := range containers {
		//log.Println("taskContainer c.pid:", c.pid)

		containerstats, err := GetContainerStats(NodeVolPrefix, c.containerId, c.name,
			c.cgroupParent, int(c.restartCount), int(c.pid))
		if err == nil {

			if lastContainerstats, ok := this.lastStatCache[c.containerId]; ok {
				//fmt.Println("step -5 ")
				totalCpu, userCpu, sysCpu := calcCpuUsage(containerstats, lastContainerstats)
				sysCpuMillis := sysCpu * float32(this.numcpu) * float32(10)
				totalCpuMillis := totalCpu * float32(this.numcpu) * float32(10)
				userCpuMillis := userCpu * float32(this.numcpu) * float32(10)
				agentOid := conf.OID
				agentPcode := conf.PCODE
				//fmt.Println(c.containerId, c.pid, totalCpu, userCpu, sysCpu)
				p := createPack()

				p.Tags.PutString("containerId", c.containerId)
				p.Tags.PutString("containerName", c.name)
				p.Tags.Put("agentOid", value.NewDecimalValue(int64(agentOid)))
				p.Tags.Put("agentPcode", value.NewLongValue(agentPcode))
				p.Tags.PutString("command", c.command)
				p.Tags.Put("containerKey", value.NewDecimalValue(int64(hash.HashStr(c.containerId))))
				p.Tags.Put("created", value.NewDecimalValue(int64(c.created)))
				p.Tags.PutString("image", c.image)
				p.Tags.Put("imageHash", value.NewDecimalValue(int64(hash.HashStr(c.image))))
				p.Tags.PutString("imageId", c.imageId)
				p.Tags.Put("onode", value.NewDecimalValue(int64(c.onode)))
				p.Tags.PutString("onodeName", c.onodeName)

				isFound, m := cache.GetContainerCache(c.containerId)

				p.Tags.PutString("dimension", fmt.Sprint("TaskARN=", m.Tags["TaskArn"], ";containerId=", c.containerId))

				if isFound {
					populateAll(p.Tags, m.Tags)
					populateAll(p.Data, m.Fields)
				}

				p.Put("cpu_user", value.NewFloatValue(userCpu))
				p.Put("cpu_user_millis", value.NewFloatValue(userCpuMillis))
				p.Put("cpu_sys", value.NewFloatValue(sysCpu))
				p.Put("cpu_sys_millis", value.NewFloatValue(sysCpuMillis))
				p.Put("cpu_total", value.NewFloatValue(totalCpu))
				p.Put("cpu_total_millis", value.NewFloatValue(totalCpuMillis))
				p.Put("mem_usage", value.NewLongValue(containerstats.MemoryStats.Usage))
				p.Put("mem_totalrss", value.NewLongValue(containerstats.MemoryStats.Stats.TotalRss))
				blkioRbps, blkioRiops, blkioWbps, blkioWiops := calcBlkioUsage(containerstats, lastContainerstats)
				p.Put("blkio_rbps", value.NewFloatValue(blkioRbps))
				p.Put("blkio_riops", value.NewFloatValue(blkioRiops))
				p.Put("blkio_wbps", value.NewFloatValue(blkioWbps))
				p.Put("blkio_wiops", value.NewFloatValue(blkioWiops))
				workingSet := containerstats.MemoryStats.Usage - containerstats.MemoryStats.Stats.InactiveFile
				cpuPerQuota := parseCpuPer(this.numcpu, totalCpu, c.cpuLimit)
				cpuPerRequest := parseCpuPer(this.numcpu, totalCpu, c.cpuRequest)
				p.Put("cpu_per_quota", value.NewFloatValue(cpuPerQuota))
				p.Put("cpu_per_request", value.NewFloatValue(cpuPerRequest))
				var memPerQuota float32
				if c.memoryLimit > 0 {
					memPerQuota = float32(workingSet) / float32(c.memoryLimit) * float32(100)
				} else {
					memPerQuota = float32(workingSet) / float32(this.totalMemory) * float32(100)
				}

				cpuQuotaPercent := float32(100) * float32(c.cpuLimit) / float32(this.numcpu*1000)
				memPerRequest := float32(workingSet) / float32(c.memoryRequest) * float32(100)
				netRbps, netRiops, netWbps, netWiops := calcNetUsage(containerstats, lastContainerstats)
				nodeCpu, nodeMem := getNodePerf()

				p.Put("mem_percent", value.NewFloatValue(memPerQuota))
				p.Put("cpu_quota", value.NewDecimalValue(int64(c.cpuLimit)))
				p.Put("cpu_quota_percent", value.NewFloatValue(cpuQuotaPercent))
				p.Put("mem_limit", value.NewDecimalValue(int64(c.memoryLimit)))
				p.Put("cpu_request", value.NewDecimalValue(int64(c.cpuRequest)))
				p.Put("mem_request", value.NewDecimalValue(int64(c.memoryRequest)))
				p.Put("cpu_throttledperiods", value.NewLongValue(containerstats.CPUStats.ThrottlingData.ThrottledPeriods))
				p.Put("cpu_throttledtime", value.NewLongValue(containerstats.CPUStats.ThrottlingData.ThrottledTime))
				p.Put("mem_failcnt", value.NewDecimalValue(int64(containerstats.MemoryStats.FailCnt)))
				p.Put("mem_maxusage", value.NewLongValue(containerstats.MemoryStats.MaxUsage))
				p.Put("mem_per_request", value.NewFloatValue(memPerRequest))
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
				p.Put("restart_count", value.NewDecimalValue(int64(c.restartCount)))
				p.Put("state", value.NewDecimalValue(int64(c.state)))
				p.Put("status", value.NewTextValue(c.status))

				cache.SetPerfCache(c.containerId, p.Data)
				send(p)

			}
			this.lastStatCache[c.containerId] = containerstats
			//fmt.Println("step -6 ")
		} else {
			log.Println("container task err:", err)
		}
		//fmt.Println("step -7 ")
	}
	return nil
}
