package whataplog

import (
	"log"
	"os"
	"path"

	"github.com/natefinch/lumberjack"
	"whatap.io/aws/ecs/config"
)

func Init() {
	whatapHome := config.GetWhatapHome()
	logdir := path.Join(whatapHome, "logs")
	if _, err := os.Stat(logdir); os.IsNotExist(err) {
		os.MkdirAll(logdir, 0750)
	}

	logfullpath := path.Join(logdir, "whatap_ecs.log")

	log.SetOutput(&lumberjack.Logger{
		Filename:   logfullpath,
		MaxSize:    10,
		MaxBackups: 2,
		MaxAge:     28,
	})
}
