package counter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var (
	ecsCluster       string
	ecsRegion        string
	ecsTaskArn       string
	ecsContainerInst string
	ecsGroup         string
	ecsLaunchType    string
	ecsIsRoot        bool
	ecsStartedAt     time.Time

	lastRootVoteTime time.Time
)

func GetEcsLaunchType() (launchType string, err error) {
	if len(ecsLaunchType) < 1 {
		err = populateEcsEnv()
		if err != nil {
			return
		}
	}

	launchType = ecsLaunchType

	return
}

func getEcsRoot() (ret bool, err error) {
	if len(ecsRegion) < 1 {
		err = populateEcsEnv()
		if err != nil {
			return
		}
	}

	now := time.Now()
	if now.Sub(lastRootVoteTime) > time.Minute*1 {
		isRoot := true
		findWhatapContainer(func(startedAt time.Time) {
			isRoot = isRoot && startedAt.Sub(ecsStartedAt) >= 0
		})
		ecsIsRoot = isRoot
		lastRootVoteTime = now
	}
	ret = ecsIsRoot

	return
}

func getAllTasks(svc *ecs.ECS) (tasks []*string, err error) {
	cluster, getClusterErr := getECSCluster()
	if getClusterErr != nil {
		err = getClusterErr
		return
	}

	var nextToken *string
	for {
		input := &ecs.ListTasksInput{Cluster: &cluster, NextToken: nextToken}
		result, listTasksErr := svc.ListTasks(input)

		if listTasksErr != nil {
			err = listTasksErr
			return
		}

		if result.TaskArns != nil {
			tasks = append(tasks, result.TaskArns...)
		}

		if result.NextToken == nil || len(*result.NextToken) < 1 {
			break
		}
		nextToken = result.NextToken
	}

	return
}

func findWhatapContainer(h1 func(time.Time)) (err error) {
	sess, sessionErr := session.NewSession(&aws.Config{Region: &ecsRegion})
	if sessionErr != nil {
		err = sessionErr
		return
	}

	svc := ecs.New(sess)

	alltasks, getAllTasksErr := getAllTasks(svc)
	if getAllTasksErr != nil {
		err = getAllTasksErr
		return
	}

	input := &ecs.DescribeTasksInput{Cluster: &ecsCluster, Tasks: alltasks}

	result, descTasksErr := svc.DescribeTasks(input)

	if descTasksErr != nil {
		err = descTasksErr
		return
	}

	for _, t := range result.Tasks {

		if ecsGroup == *t.Group && ecsTaskArn != *t.TaskArn {
			h1(*t.StartedAt)
		}
	}

	return
}

func getEcsRegion() (ret string, err error) {
	if len(ecsRegion) < 1 {
		err = populateEcsEnv()
		if err != nil {
			return
		}
	}
	ret = ecsRegion

	return
}

func getECSCluster() (cluster string, err error) {
	if len(ecsCluster) < 1 {
		err = populateEcsEnv()
		if err != nil {
			return
		}
	}

	cluster = ecsCluster

	return
}

func getECSTaskArn() (taskArn string, err error) {
	if len(ecsTaskArn) < 1 {
		err = populateEcsEnv()
		if err != nil {
			return
		}
	}

	taskArn = ecsTaskArn

	return
}

func getECSContainerInst() (containerInst string, err error) {
	if len(ecsContainerInst) < 1 {
		err = populateEcsEnv()
		if err != nil {
			return
		}
	}
	containerInst = ecsContainerInst

	return
}

func populateEcsEnv() (err error) {
	metauri := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	if len(metauri) < 1 {
		err = fmt.Errorf("error env ECS_CONTAINER_METADATA_URI_V4 not found")
		return
	}

	reqerr := request(fmt.Sprintf("%s/task", metauri), func(k string, v string) {
		if k == "TaskARN" {
			//arn:aws:ecs:us-east-2:
			words := strings.Split(v, ":")
			if len(words) > 4 {
				ecsRegion = words[3]
			}
			ecsTaskArn = v
		} else if k == "Cluster" {
			ecsCluster = v
		} else if k == "LaunchType" {
			ecsLaunchType = v
		}
	})

	if reqerr != nil {
		err = reqerr
		return
	}

	popTaskErr := populateTask()
	if popTaskErr != nil {
		err = popTaskErr
		return
	}

	return
}

func populateTask() (err error) {
	sess, sessionErr := session.NewSession(&aws.Config{Region: &ecsRegion})
	if sessionErr != nil {
		err = sessionErr
		return
	}

	svc := ecs.New(sess)
	input := &ecs.DescribeTasksInput{Cluster: &ecsCluster, Tasks: []*string{&ecsTaskArn}}

	result, descTasksErr := svc.DescribeTasks(input)

	if descTasksErr != nil {
		err = descTasksErr
		return
	}

	for _, t := range result.Tasks {
		if t.ContainerInstanceArn != nil {
			ecsContainerInst = *t.ContainerInstanceArn
		}

		if t.Group != nil {
			ecsGroup = *t.Group
		}
		if t.StartedAt != nil {
			ecsStartedAt = *t.StartedAt
			//always ends here
			return
		}
	}

	err = fmt.Errorf("containerInst not found")
	return
}

func request(url string, h2 func(string, string)) (err error) {

	client := http.DefaultClient

	resp, clientErr := client.Get(url)
	if clientErr != nil {
		err = clientErr
		return
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		err = readErr
		return
	}

	r := ECSTaskResp{}
	jsonErr := json.Unmarshal(body, &r)
	if jsonErr != nil {
		err = jsonErr
		return
	}

	h2("Cluster", r.Cluster)
	h2("DesiredStatus", r.DesiredStatus)
	h2("Family", r.Family)
	h2("KnownStatus", r.KnownStatus)
	h2("TaskARN", r.TaskARN)
	h2("LaunchType", r.LaunchType)

	return
}

func EcsMetaV4Task() (task ECSTaskResp, err error) {
	metauri := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	if len(metauri) < 1 {
		err = fmt.Errorf("error env ECS_CONTAINER_METADATA_URI_V4 not found")
		return
	}

	url := fmt.Sprintf("%s/task", metauri)

	client := http.DefaultClient

	resp, clientErr := client.Get(url)
	if clientErr != nil {
		err = clientErr
		return
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		err = readErr
		return
	}

	jsonErr := json.Unmarshal(body, &task)
	if jsonErr != nil {
		err = jsonErr
		return
	}

	return
}

func EcsMetaV4Stats() (stats map[string]ContainerStat, err error) {
	metauri := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	if len(metauri) < 1 {
		err = fmt.Errorf("error env ECS_CONTAINER_METADATA_URI_V4 not found")
		return
	}

	url := fmt.Sprintf("%s/task/stats", metauri)

	client := http.DefaultClient

	resp, clientErr := client.Get(url)
	if clientErr != nil {
		err = clientErr
		return
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		err = readErr
		return
	}

	jsonErr := json.Unmarshal(body, &stats)
	if jsonErr != nil {
		err = jsonErr
		return
	}

	return
}
