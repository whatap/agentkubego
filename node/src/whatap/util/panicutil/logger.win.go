//+build windows

package panicutil

import (
	"fmt"
	"os"
	"path/filepath"

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
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath := filepath.Dir(ex)

		errorLogger = &lumberjack.Logger{
			Filename:   fmt.Sprintf("%s/error.log", exPath),
			MaxSize:    MaxErrorLogSize,   // megabytes after which new file is created
			MaxBackups: MaxErrorLogBackup, // number of backups
			MaxAge:     MaxErrorLogAge,    //days
		}
	}

	return errorLogger
}
