package conf

import (
	//"log"
	//"fmt"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/logutil"
)

type Runnable interface {
	Run()
}

var observer map[string]Runnable = make(map[string]Runnable)

func AddConfObserver(cls string, run Runnable) {
	//fmt.Println("Add=", cls)
	observer[cls] = run
}
func RunConfObserver() {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("RunConfObserver", " Recover", r)
		}
	}()

	for _, v := range observer {
		v.Run()
	}
}
