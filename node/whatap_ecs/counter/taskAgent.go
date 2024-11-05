package counter

import (
	"fmt"
	"os"
	"runtime"

	"github.com/whatap/go-api/common/lang/pack"
	"github.com/whatap/go-api/common/lang/value"

	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/common/util/hash"
	"github.com/whatap/go-api/common/util/iputil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/osinfo"
	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/text"
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

	return
}

func (this *TaskAgentInfo) interval() int {

	return 86400
}

func (this *TaskAgentInfo) process(now int64) error {
	conf := config.GetConfig()
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

	// this.sendNames()

	return nil
}

func (this *TaskAgentInfo) sendNames() {
	conf := config.GetConfig()
	text.SendHashText(pack.ONODE_NAME, hash.HashStr(conf.ONODE), conf.ONODE)
	text.SendHashText(pack.TEXT_ONAME, conf.OID, conf.ONAME)
}
