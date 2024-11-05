package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/control"
	"whatap.io/aws/ecs/counter"
	"whatap.io/aws/ecs/logsink"
	"whatap.io/aws/ecs/session"
	"whatap.io/aws/ecs/text"
	"whatap.io/aws/ecs/whataplog"
)

const (
	AGENT_VERSION = "0.0.1"
	BUILDNO       = "001"
)

func main() {

	launchType, err := counter.GetEcsLaunchType()
	log.Println("main ecsLaunchType:", launchType, " err:", err)
	if launchType != "FARGATE" {
		whataplog.Init()
	} else {
		config.IsFARGATE = true
	}

	if config.GetConfig().DEBUG {
		log.Println("main ecsLaunchType:", launchType)
	}
	session.InitWhatapNet()

	if config.GetConfig().DEBUG {
		log.Println("main whatapNet")
	}
	// configLog()
	conf := config.GetConfig()
	configObserver()

	session.InitSecureSession()
	if config.GetConfig().DEBUG {
		log.Println("main InitSecureSession")
	}
	text.InitializeTextSender()
	if config.GetConfig().DEBUG {
		log.Println("main InitializeTextSender")
	}
	addControlObserver()
	control.StartControlManager()
	if config.GetConfig().DEBUG {
		log.Println("main StartControlManager")
	}
	counter.StartCounterManager()
	if config.GetConfig().DEBUG {
		log.Println("main StartPolling")
	}
	if launchType != "FARGATE" {
		logsink.StartPolling()
	}
	log.Println("WhaTap Version", AGENT_VERSION, "pid=", os.Getpid(), " pcode=", conf.PCODE, " oid=", conf.OID)
	serveForever()
}

func configObserver() {
	config.AddObserver(func(conf *config.Config) {

		logsink.Pcode = conf.PCODE
		logsink.Oid = conf.OID
		logsink.OnodeName = conf.ONODE
	})
}

func addControlObserver() {
	session.AddReadObserver(control.PackObserver)
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
