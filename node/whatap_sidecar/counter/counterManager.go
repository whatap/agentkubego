package counter

import (
	"log"
	"time"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
)

var (
	AGENT_VERSION = "0.0.1"
	BUILDNO       = "001"
	READ_MAX      = 8 * 1024 * 1024
	NET_TIMEOUT   = 30 * time.Second
)

func StartCounterManager() {
	go runCounter()
}

func runCounter() {
	taskContainer := &TaskContainer{}
	taskContainer.init()
	taskNode := &TaskNode{}
	taskAgentInfo := &TaskAgentInfo{}
	taskAgentInfo.init()
	taskKubeNode := &TaskKubeNode{}
	taskKubeNode.init()

	tasks := []*TaskAction{
		&TaskAction{name: "agentInfo", task: taskAgentInfo},
		&TaskAction{name: "container", task: taskContainer},
		&TaskAction{name: "workerNode", task: taskNode},
		&TaskAction{name: "kubeNode", task: taskKubeNode}}
	for {
		now := dateutil.Now()
		for _, ta := range tasks {
			//logutil.Println("DEBUG", "tasks ", now, ta.lastActTime, now-ta.lastActTime)
			if now-ta.lastActTime > int64(ta.task.interval()*1000) {
				ta.lastActTime = now
				err := ta.task.process(now)
				if err != nil {
					log.Println("Error", "Task", err)
				}
			}
		}

		time.Sleep(time.Millisecond * 100)

	}
}
