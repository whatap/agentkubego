package counter

import (
	"log"
	"time"

	"github.com/whatap/go-api/common/util/dateutil"
	"whatap.io/aws/ecs/config"
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
	err := populateEcsEnv()
	if err != nil {
		log.Println("populateEcsEnv error:", err)
	}
	if config.GetConfig().DEBUG {
		log.Println("counterManager ecsLaunchType:", ecsLaunchType, " IsFargateHelper:", config.GetConfig().IsFargateHelper)
	}
	tasks := []*TaskAction{}
	if ecsLaunchType == "FARGATE" {
		if config.GetConfig().IsFargateHelper {
			taskEcsSingletion := &TaskECSSingleton{}
			taskEcsSingletion.init()
			tasks = append(tasks, &TaskAction{name: "ecssingle", task: taskEcsSingletion})
		} else {
			taskFargate := &TaskFargate{}
			taskFargate.init()
			tasks = append(tasks, &TaskAction{name: "fargate", task: taskFargate})
		}

	} else {
		taskAgentInfo := &TaskAgentInfo{}
		taskAgentInfo.init()
		tasks = append(tasks, &TaskAction{name: "agentInfo", task: taskAgentInfo})

		taskEcs := &TaskECS{}
		taskEcs.init()
		tasks = append(tasks, &TaskAction{name: "ecs", task: taskEcs})

		taskNode := &TaskNode{}
		tasks = append(tasks, &TaskAction{name: "workerNode", task: taskNode})
		taskContainer := &TaskContainer{}
		taskContainer.init()
		tasks = append(tasks, &TaskAction{name: "container", task: taskContainer})
	}

	config.AddObserver(func(conf *config.Config) {
		NodeVolPrefix = conf.NodeVolPrefix
	})

	for {
		now := dateutil.Now()
		for _, ta := range tasks {
			if now-ta.lastActTime > int64(ta.task.interval()*1000) {
				ta.lastActTime = now
				// log.Println("running ", time.Now(), " task name:", ta.name)
				err := ta.task.process(now)
				if err != nil {
					log.Println("Error", "Task", err)
				}
			}
		}

		time.Sleep(time.Millisecond * 100)
	}
}
