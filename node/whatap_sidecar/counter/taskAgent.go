package counter

import (
	"fmt"
	"os"
	"runtime"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/hash"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/osinfo"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/iputil"
	"whatap.io/k8s/sidecar/text"
)

type TaskAgentInfo struct {
	totalMemory int64
	numcpu      int
}

func (this *TaskAgentInfo) init() {
	parseMemoryStat(func(k string, v int64) {
		if k == "memtotal" {
			this.totalMemory = v
		}
	})
	this.numcpu = runtime.NumCPU()

	AddNSObserver(func(ns string) {
		onNamespaceDetected(ns, this.numcpu, this.totalMemory)
	})

	return
}

func onNamespaceDetected(ns string, numcpu int, totalMemory int64) {
	findNamespace(nodeNamespace, func(pcode int64, licenseHash64 int64) {
		if conf.PCODE != pcode {
			text.SendHashTextOneway(pcode, licenseHash64, pack.ONODE_NAME, hash.HashStr(conf.ONODE), conf.ONODE)
			text.SendHashTextOneway(pcode, licenseHash64, pack.TEXT_ONAME, conf.OID, conf.ONAME)
			text.SendHashTextOneway(pcode, licenseHash64, pack.CONTAINER, hash.HashStr(getSelfContainerId()), getSelfContainerId())

			p := pack.NewParamPack()
			p.Pcode = pcode
			p.Oid = conf.OID
			p.Time = dateutil.Now()
			p.Id = net.AGENT_BOOT_ENV
			p.PutString("whatap.version", AGENT_VERSION)
			p.PutString("whatap.build", BUILDNO)
			p.PutString("whatap.home", os.Getenv("WHATAP_HOME"))
			p.PutString("whatap.name", conf.ONAME)
			p.PutString("whatap.ip", iputil.ToStringInt(getMyAddr()))
			p.PutString("whatap.pid", fmt.Sprint(os.Getpid()))
			p.PutLong("java.start", dateutil.Now())

			p.PutString("os.arch", runtime.GOARCH)
			p.PutString("os.name", runtime.GOOS)
			p.PutString("os.cpucore", fmt.Sprint(numcpu))

			p.PutString("os.memory", fmt.Sprint(totalMemory))
			p.PutString("os.release", osinfo.GetOsRelease())
			p.Put("sms.starttime", value.NewDecimalValue(dateutil.Now()))

		}
	})

}

func (this *TaskAgentInfo) interval() int {

	return 86400
}

func (this *TaskAgentInfo) process(now int64) error {

	this.sendNames()
	p := pack.NewParamPack()
	p.Pcode = conf.PCODE
	p.Oid = conf.OID
	p.Time = dateutil.Now()
	p.Id = net.AGENT_BOOT_ENV
	p.PutString("whatap.version", AGENT_VERSION)
	p.PutString("whatap.build", BUILDNO)
	p.PutString("whatap.home", os.Getenv("WHATAP_HOME"))
	p.PutString("whatap.name", conf.ONAME)
	p.PutString("whatap.ip", iputil.ToStringInt(getMyAddr()))
	p.PutString("whatap.pid", fmt.Sprint(os.Getpid()))
	p.PutLong("java.start", dateutil.Now())

	p.PutString("os.arch", runtime.GOARCH)
	p.PutString("os.name", runtime.GOOS)
	p.PutString("os.cpucore", fmt.Sprint(this.numcpu))

	p.PutString("os.memory", fmt.Sprint(this.totalMemory))
	p.PutString("os.release", osinfo.GetOsRelease())
	p.Put("sms.starttime", value.NewDecimalValue(dateutil.Now()))

	sendEncrypted(p)

	findNamespace(nodeNamespace, func(pcode int64, licenseHash64 int64) {
		if conf.PCODE != pcode {
			p.Pcode = pcode
			sendOneway(pcode, licenseHash64, p)
		}
	})

	return nil
}

func (this *TaskAgentInfo) sendNames() {
	text.SendHashText(pack.ONODE_NAME, hash.HashStr(conf.ONODE), conf.ONODE)
	text.SendHashText(pack.TEXT_ONAME, conf.OID, conf.ONAME)
	text.SendHashText(pack.CONTAINER, hash.HashStr(getSelfContainerId()), getSelfContainerId())

}
