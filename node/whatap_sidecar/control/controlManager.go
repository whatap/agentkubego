package control

import (
	"bytes"
	"context"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/value"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/loggingutil"
	corev1 "k8s.io/api/core/v1"
	"whatap.io/k8s/sidecar/config"
	"whatap.io/k8s/sidecar/kube"
	"whatap.io/k8s/sidecar/session"
)

var (
	controlManagerStarted bool = false
	RecvBuffer            chan pack.Pack
	conf                  = config.GetConfig()
	RecvBufferSize        = 100
	podns                 string
	containerLookup       = map[string][]string{}
)

func OnNamespaceDetected(ns string) {
	podns = ns
}

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
	switch p.Id {
	case net.GET_ENV:
		p = p.ToResponse()
		p.Pcode = conf.PCODE
		p.Oid = conf.OID

		m := value.NewMapValue()
		for _, e := range os.Environ() {
			pair := strings.Split(e, "=")
			m.PutString(pair[0], pair[1])
		}
		p.Put("env", m)

		session.SendEncrypted(p)

		break
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
		}
		p = p.ToResponse()
		p.Pcode = conf.PCODE
		p.Oid = conf.OID

		session.SendEncrypted(p)

		break
	case net.CONFIGURE_GET:
		p = p.ToResponse()
		p.Pcode = conf.PCODE
		p.Oid = conf.OID

		m := config.GetAllPropertiesMapValue()
		keyEnum := m.Keys()
		for keyEnum.HasMoreElements() {
			k := keyEnum.NextString()
			p.Put(k, m.Get(k))
		}

		sendEncrypted(p)
		break

	case net.AGENT_LOG_LIST:
		p = p.ToResponse()
		p.Pcode = conf.PCODE
		p.Oid = conf.OID
		m := value.NewMapValue()

		logfileprefix := filepath.Join(config.GetWhatapHome(), "logs")
		filepath.Walk(logfileprefix, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				m.Put(path, value.NewDecimalValue(info.Size()))
			}
			return nil
		})

		p.Put("files", m)
		sendEncrypted(p)
	case net.AGENT_LOG_READ:
		p = p.ToResponse()
		p.Pcode = conf.PCODE
		p.Oid = conf.OID
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

		sendEncrypted(p)
	case net.KUBERNETES:
		p = p.ToResponse()
		p.Pcode = conf.PCODE
		p.Oid = conf.OID

		cmd := p.GetString("cmd")
		log.Println("kubernetes cmd:" + cmd)
		switch cmd {
		case "getLastLog":
			containerid := p.GetString("containerid")
			if _, ok := containerLookup[containerid]; !ok {
				log.Println("kubernetes containerid:" + containerid + " not found")
				break
			}
			containerName := containerLookup[containerid][0]
			podname := containerLookup[containerid][1]

			taillines := p.GetLong("taillines")

			cli, err := kube.GetKubeClient()
			if err != nil {
				log.Println(err)
				break
			}

			podLogOpts := corev1.PodLogOptions{Follow: false, Container: containerName, TailLines: &taillines}
			req := cli.CoreV1().Pods(podns).GetLogs(podname, &podLogOpts)
			podLogs, err := req.Stream(context.Background())
			if err != nil {
				log.Println("GetLastLog podLogOpts:", podLogOpts)
				log.Println("GetLastLog podns:", podns)
				log.Println("GetLastLog podname:", podname)
				log.Println("GetLastLog", err)
				break
			}
			defer podLogs.Close()

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				log.Println("GetLastLog", err)
				break
			}
			lastlog := buf.String()

			p.PutString("log", lastlog)
		}
		sendEncrypted(p)
	default:
		break
	}
}
