package control

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/whatap/go-api/common/lang/pack"
	"github.com/whatap/go-api/common/lang/value"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/loggingutil"
	"whatap.io/aws/ecs/config"
)

var (
	controlManagerStarted bool = false
	RecvBuffer            chan pack.Pack

	RecvBufferSize  = 100
	containerLookup = map[string][]string{}
)

func OnContainerDetected(cid string, ns string, podName string, cName string) {
	containerLookup[cid] = []string{cName, podName}
}

func StartControlManager() {

	if !controlManagerStarted {
		controlManagerStarted = true
		RecvBuffer = make(chan pack.Pack, RecvBufferSize)
		go runControl()
	}
}

func PackObserver(p pack.Pack) {
	if RecvBuffer != nil && len(RecvBuffer) < RecvBufferSize {
		RecvBuffer <- p
	}
}

func runControl() {
	for {
		p := <-RecvBuffer

		switch p.GetPackType() {
		case pack.PACK_PARAMETER:
			process(p.(*pack.ParamPack))

		default:

		}
	}
}

func process(p *pack.ParamPack) {
	conf := config.GetConfig()
	p = p.ToResponse()
	p.Pcode = conf.PCODE
	p.Oid = conf.OID

	// log.Println("controlManager.process:", p.ToString())

	switch p.Id {
	case net.GET_ENV:
		m := value.NewMapValue()
		for _, e := range os.Environ() {
			pair := strings.Split(e, "=")
			m.PutString(pair[0], pair[1])
		}
		p.Put("env", m)
	case net.SET_CONFIG:
		configmap := p.GetMap("config")
		if configmap != nil {
			keyValues := map[string]string{}
			keyEnumer := configmap.Keys()
			for keyEnumer.HasMoreElements() {
				key := keyEnumer.NextString()
				value := configmap.GetString(key)
				keyValues[key] = value
			}
			config.SetValues(&keyValues)
			config.Update()
		}
	case net.CONFIGURE_GET:
		m := config.GetAllPropertiesMapValue()

		keyEnum := m.Keys()
		for keyEnum.HasMoreElements() {
			k := keyEnum.NextString()
			p.Put(k, m.Get(k))
		}
	case net.AGENT_LOG_LIST:
		m := value.NewMapValue()

		logfileprefix := filepath.Join(config.GetWhatapHome(), "logs")
		filepath.Walk(logfileprefix, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() {
				m.Put(path, value.NewDecimalValue(info.Size()))
			}
			return nil
		})

		p.Put("files", m)
	case net.AGENT_LOG_READ:
		filename := p.GetString("file")
		endpos := p.GetLong("pos")
		length := p.GetLong("length")
		length = int64(math.Min(float64(length), 8000))

		before, next, logText, e := loggingutil.ReadLog(filename, endpos, length)
		if e == nil {
			p.Put("before", value.NewDecimalValue(before))
			p.Put("next", value.NewDecimalValue(next))
			p.PutString("text", logText)
		} else {
			p.Put("before", value.NewDecimalValue(0))
			p.Put("next", value.NewDecimalValue(-1))
			p.PutString("text", e.Error())
		}
	case net.KUBERNETES:
		cmd := p.GetString("cmd")
		if cmd == "getLastLog" && p.Get("containerid") != value.NULL_VALUE && p.Get("taillines") != value.NULL_VALUE {
			containerid := p.GetString("containerid")
			taillines := p.GetLong("taillines")

			var lines bytes.Buffer
			err := getLastLog(containerid, taillines, func(line []byte) {
				lines.Write(line)
				lines.WriteString("\n")
			})
			if err == nil {
				p.PutString("log", lines.String())
			} else {
				p.PutString("log", fmt.Sprint(err.Error(), " id:", containerid))
			}
		}

	default:

	}
	sendEncrypted(p)
}
