package panicutil

import (
	"os"
	"time"
	"runtime/pprof"
	"github.com/shirou/gopsutil/process"
)

var SelfCpuUsage float32
var SelfCpuThresholdEnabled = false
var SelfCpuThreshold = float32(50)
var SelfSleepInterval = int(10)

var SelfCpuPerfBufLen = uint64(50)
var SelfCpuPerfs []float32 = make([]float32, SelfCpuPerfBufLen)
var SelfCpuMeasureInterval = 100
var i uint64 = uint64(0)

var IsDumpGoroutine = false
func DumpGoroutine() {
	for {
		if IsDumpGoroutine{
			f, _ := os.OpenFile("/var/log/whatap_infrad.goroutine.dump.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			f.Write([]byte("\n"))
			f.Write([]byte(time.Now().Format("2006-01-02 15:04:05")))
			p := pprof.Lookup("goroutine")
			p.WriteTo(f, 1)
		}
		time.Sleep(1 * time.Second)
	}
}

//StartSelfMonitoring
func StartSelfMonitoring() {

	for {
		if IsDebug{
			SelfCpuUsage, _ = GetSelfCPUUsage()
			SelfCpuPerfs[i%SelfCpuPerfBufLen] = SelfCpuUsage
			// fmt.Println("StartSelfMonitoring ", SelfCpuUsage, err)
			i += 1
		}


		
		time.Sleep(time.Duration(SelfCpuMeasureInterval) * time.Millisecond)
	}
}

//SelfSleep SelfSleep
func SelfSleep() {
	if SelfCpuThresholdEnabled && SelfCpuUsage > SelfCpuThreshold {
		time.Sleep(time.Duration(SelfSleepInterval) * time.Millisecond)
	}
}

func GetSelfCpuPerf() (uint64, []float32) {
	return i % SelfCpuPerfBufLen, SelfCpuPerfs
}

var thisProcess *process.Process

func GetSelfMemoryUsage() (int64, error) {
	if thisProcess == nil {
		p, err := process.NewProcess(int32(os.Getpid()))
		if err != nil {
			return 0, err
		}
		thisProcess = p
	}

	processMemory, err := thisProcess.MemoryInfo()
	if err != nil {
		return 0, err
	}
	used := processMemory.RSS

	return int64(used), err
}
