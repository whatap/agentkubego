package taghelper

import (
	"os"
	"path/filepath"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/config"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/iputil"
)

var (
	executionFilePath string
	ip                string
	version           string
)

func FocusTraceInfoIter(tagcallback func(string, string), fieldcallback func(string, string)) {
	if len(executionFilePath) < 1 {
		executionFilePath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	}

	if len(ip) < 1 {
		ip = iputil.GetIPsToString()
	}

	if tagcallback != nil {
		tagcallback("installDir", executionFilePath)
		tagcallback("version", config.GetConfig().VERSION)
	}

	if fieldcallback != nil {
		fieldcallback("whatapfocus_ip", ip)
	}

}
