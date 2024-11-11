//+build !windows

package panicutil

import (
	// "fmt"
	// "log/syslog"

	"github.com/natefinch/lumberjack"
)

var (
	//MaxErrorLogSize MaxErrorLogSize
	MaxErrorLogSize = 10
	//MaxErrorLogBackup MaxErrorLogBackup
	MaxErrorLogBackup = 2
	//MaxErrorLogAge MaxErrorLogAge
	MaxErrorLogAge = 37
)

func getErrorLogger() *lumberjack.Logger {
	if errorLogger == nil {
		errorLogger = &lumberjack.Logger{
			Filename:   "/var/log/whatap_infrad.log",
			MaxSize:    MaxErrorLogSize,   // megabytes after which new file is created
			MaxBackups: MaxErrorLogBackup, // number of backups
			MaxAge:     MaxErrorLogAge,    //days
		}
	}

	return errorLogger
}

