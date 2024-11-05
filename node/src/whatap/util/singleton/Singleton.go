package singleton

import (
	"github.com/whatap/golib/util/panicutil"
	"sync"
	"time"
)

type Singleton struct {
	mutex    *sync.Mutex
	started  bool
	runnable func()
	interval func() int32
}

func NewSingleton(runnable func()) *Singleton {
	proc := &Singleton{}
	proc.mutex = &sync.Mutex{}
	proc.started = false
	proc.runnable = runnable
	return proc
}
func NewSingletonTimer(runnable func(), interval func() int32) *Singleton {
	proc := &Singleton{}
	proc.mutex = &sync.Mutex{}
	proc.started = false
	proc.runnable = runnable
	proc.interval = interval
	return proc
}
func (this *Singleton) Start() bool {
	if this.started {
		return false
	}
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.started {
		return false
	}
	this.started = true
	if this.interval == nil {
		go this.process()
	} else {
		go func() {
			for {

				this.process()

				tm := this.interval()
				if tm <= 0 {
					tm = 1000
				}
				time.Sleep(time.Duration(tm) * time.Millisecond)
			}
		}()
	}

	return true
}

func (this *Singleton) process() {
	defer func() {
		if r := recover(); r != nil {
			_, ok := r.(error)
			if ok {
				panicutil.Error("Singleton: ", (r.(error)).Error())
			}
		}
	}()

	this.runnable()
}
