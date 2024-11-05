package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"whatap.io/k8s/sidecar/confbase"
	"whatap.io/k8s/sidecar/config"
	"whatap.io/k8s/sidecar/control"
	"whatap.io/k8s/sidecar/counter"
	"whatap.io/k8s/sidecar/logsink"
	"whatap.io/k8s/sidecar/micro"
	"whatap.io/k8s/sidecar/session"
	"whatap.io/k8s/sidecar/text"
)

const (
	AGENT_VERSION = "0.0.1"
	BUILDNO       = "001"
)

func main() {
	// fmt.Println("hello world")
	// logutil.Println("WA000", "Hello World")

	session.InitWhatapNet()
	// configLog()
	conf := config.GetConfig()
	configObserver()

	log.Println("WhaTap Version", AGENT_VERSION, "pid=", os.Getpid(), " pcode=", conf.PCODE, " oid=", conf.OID)

	session.InitSecureSession()
	text.InitializeTextSender()
	micro.StartMicroManager()
	addControlObserver()
	control.StartControlManager()
	counter.StartCounterManager()
	logsink.StartPolling()
	addCounterObserver()

	serveForever()
}

func configObserver() {
	config.AddObserver(func(conf *config.Config) {
		confbase.Host = conf.ConfbaseHost
		confbase.Port = conf.ConfbasePort

		logsink.Pcode = conf.PCODE
		logsink.Oid = conf.OID
		logsink.OnodeName = conf.ONODE
	})
}

func addCounterObserver() {
	confbase.StartPolling()
	confbase.AddObserver(counter.OnNamespaceProjectChange)
}

func addControlObserver() {
	session.AddReadObserver(control.PackObserver)
	counter.AddNSObserver(control.OnNamespaceDetected)
	counter.AddObserver(control.OnContainerDetected)
	counter.AddObserver(logsink.OnContainerDetected)
}

func configLog() {
	logfilepath := filepath.Join(config.GetWhatapHome(), "logs", "whatap.node.log")
	logStruct := &lumberjack.Logger{
		Filename:   logfilepath,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     28}

	log.SetOutput(logStruct)

}

func serveForever() {
	for {
		time.Sleep(100 * time.Millisecond)
	}
}
