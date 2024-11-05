package counter

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"whatap.io/aws/ecs/cache"
)

func (taskEcs *TaskECS) init() {
	region, regionErr := getEcsRegion()
	if regionErr != nil {
		log.Println("TaskECS error:", regionErr.Error())
		return
	}
	sess, sessionErr := session.NewSession(&aws.Config{Region: &region})
	if sessionErr == nil {
		taskEcs.sess = sess
	} else {
		log.Println("TaskECS error:", sessionErr.Error())
	}
}

func (taskEcs *TaskECS) interval() int {

	return 5
}

func (taskEcs *TaskECS) process(now int64) (err error) {
	if taskEcs.sess == nil {
		return fmt.Errorf("session not ready")
	}

	listContErr := taskEcs.listContainerInstances(now)
	if listContErr != nil {
		err = listContErr
		return
	}
	listServErr := taskEcs.listServices(now)
	if listServErr != nil {
		err = listServErr
		return
	}

	listTaskErr := taskEcs.listTasks(now)
	if listTaskErr != nil {
		err = listTaskErr
		return
	}

	return

}

func (taskEcs *TaskECS) listContainerInstances(now int64) (err error) {
	isRoot, ecsRootErr := getEcsRoot()
	if !isRoot || ecsRootErr != nil {
		err = ecsRootErr
		return
	}
	cluster, getClusterErr := getECSCluster()
	if getClusterErr != nil {
		err = getClusterErr
		return
	}

	containerInsts, getECSContainerErr := taskEcs.getECSContainerInstAll()
	if getECSContainerErr != nil {
		err = getECSContainerErr
		return
	}

	input := &ecs.DescribeContainerInstancesInput{Cluster: &cluster, ContainerInstances: containerInsts}

	svc := ecs.New(taskEcs.sess)
	result, descContainerInstancesErr := svc.DescribeContainerInstances(input)

	if descContainerInstancesErr != nil {
		err = descContainerInstancesErr
		return
	}

	for _, ci := range result.ContainerInstances {
		tags := map[string]interface{}{}
		fields := map[string]interface{}{}
		populateAttribute(ci.Attributes, func(k string, v string) {
			tags[k] = v
		})
		setKeyStringValue(&tags, "Ec2InstanceId", ci.Ec2InstanceId)
		setKeyStringValue(&tags, "ContainerInstanceArn", ci.ContainerInstanceArn)
		setKeyStringValue(&tags, "DockerVersion", ci.VersionInfo.DockerVersion)
		setKeyStringValue(&tags, "AgentVersion", ci.VersionInfo.AgentVersion)
		setKeyInt64Value(&fields, "PendingTasksCount", ci.PendingTasksCount)
		setKeyInt64Value(&fields, "RunningTasksCount", ci.RunningTasksCount)
		populateTag(ci.Tags, func(k string, v string) {
			tags[k] = v
		})
		setKeyInt64Value(&tags, "Version", ci.Version)

		setKeyStringValue(&fields, "Status", ci.Status)
		setKeyStringValue(&fields, "StatusReason", ci.StatusReason)
		setKeyTimeValue(&fields, "RegisteredAt", ci.RegisteredAt)
		sendTagFieldPack("ecs_node", tags, fields, now)
	}

	return
}

func (taskEcs *TaskECS) getECSContainerInstAll() (contInstances []*string, err error) {
	cluster, getClusterErr := getECSCluster()
	if getClusterErr != nil {
		err = getClusterErr
		return
	}

	svc := ecs.New(taskEcs.sess)
	var nextToken *string
	for {
		input := &ecs.ListContainerInstancesInput{Cluster: &cluster, NextToken: nextToken}
		result, listContInstancesErr := svc.ListContainerInstances(input)

		if listContInstancesErr != nil {
			err = listContInstancesErr
			return
		}

		if result.ContainerInstanceArns != nil {
			contInstances = append(contInstances, result.ContainerInstanceArns...)
		}

		if result.NextToken == nil || len(*result.NextToken) < 1 {
			break
		}
		nextToken = result.NextToken
	}

	return
}

func (taskEcs *TaskECS) getAllServices() (services []*string, err error) {
	svc := ecs.New(taskEcs.sess)
	cluster, getClusterErr := getECSCluster()
	if getClusterErr != nil {
		err = getClusterErr
		return
	}

	var nextToken *string
	for {
		input := &ecs.ListServicesInput{Cluster: &cluster, NextToken: nextToken}
		result, listServicesErr := svc.ListServices(input)

		if listServicesErr != nil {
			err = listServicesErr
			return
		}

		if result.ServiceArns != nil {
			services = append(services, result.ServiceArns...)
		}

		if result.NextToken == nil || len(*result.NextToken) < 1 {
			break
		}
		nextToken = result.NextToken
	}

	return
}

func (taskEcs *TaskECS) listServices(now int64) (err error) {
	isRoot, ecsRootErr := getEcsRoot()
	if !isRoot || ecsRootErr != nil {
		err = ecsRootErr
		return
	}

	svc := ecs.New(taskEcs.sess)
	cluster, getClusterErr := getECSCluster()
	if getClusterErr != nil {
		err = getClusterErr
		return
	}
	services, getServicesErr := taskEcs.getAllServices()
	if getServicesErr != nil {
		err = getServicesErr
		return
	}

	input := &ecs.DescribeServicesInput{Cluster: &cluster, Services: services}

	result, descServicesErr := svc.DescribeServices(input)

	if descServicesErr != nil {
		err = descServicesErr
		return
	}

	for _, f := range result.Failures {
		taskEcs.sendFailure(f)
	}

	for _, s := range result.Services {
		tags := map[string]interface{}{}
		fields := map[string]interface{}{}
		setKeyStringValue(&tags, "ClusterArn", s.ClusterArn)
		setKeyStringValue(&tags, "LaunchType", s.LaunchType)
		setKeyStringValue(&tags, "PlatformVersion", s.PlatformVersion)
		setKeyStringValue(&tags, "PropagateTags", s.PropagateTags)
		setKeyStringValue(&tags, "RoleArn", s.RoleArn)
		setKeyStringValue(&tags, "CreatedBy", s.CreatedBy)
		setKeyStringValue(&tags, "SchedulingStrategy", s.SchedulingStrategy)
		setKeyStringValue(&tags, "ServiceArn", s.ServiceArn)
		setKeyStringValue(&tags, "ServiceName", s.ServiceName)
		setKeyStringValue(&tags, "TaskDefinition", s.TaskDefinition)
		if s.DeploymentController != nil {
			setKeyStringValue(&tags, "DeploymentControllerType", s.DeploymentController.Type)
		}

		setKeyInt64Value(&tags, "HealthCheckGracePeriodSeconds", s.HealthCheckGracePeriodSeconds)
		populateTag(s.Tags, func(k string, v string) {
			tags[k] = v
		})

		setKeyStringValue(&fields, "Status", s.Status)
		setKeyTimeValue(&fields, "CreatedAt", s.CreatedAt)
		setKeyInt64Value(&fields, "DesiredCount", s.DesiredCount)
		setKeyInt64Value(&fields, "PendingCount", s.PendingCount)
		setKeyInt64Value(&fields, "RunningCount", s.RunningCount)

		sendTagFieldPack("ecs_service", tags, fields, now)

		for _, d := range s.Deployments {
			taskEcs.sendDeployment(*s.ServiceArn, *s.ServiceName, d, now)
		}
	}

	return
}

func (taskEcs *TaskECS) sendFailure(f *ecs.Failure) {

}

func (taskEcs *TaskECS) sendDeployment(serviceArn string, serviceName string, d *ecs.Deployment, now int64) {
	tags := map[string]interface{}{}
	fields := map[string]interface{}{}

	tags["ServiceArn"] = serviceArn
	tags["ServiceName"] = serviceName
	setKeyStringValue(&tags, "Id", d.Id)
	setKeyStringValue(&tags, "LaunchType", d.LaunchType)
	setKeyStringValue(&tags, "PlatformVersion", d.PlatformVersion)
	setKeyStringValue(&tags, "TaskDefinition", d.TaskDefinition)

	setKeyStringValue(&fields, "Status", d.Status)
	setKeyTimeValue(&fields, "CreatedAt", d.CreatedAt)
	setKeyInt64Value(&fields, "DesiredCount", d.DesiredCount)
	setKeyInt64Value(&fields, "PendingCount", d.PendingCount)
	setKeyInt64Value(&fields, "RunningCount", d.RunningCount)
	setKeyTimeValue(&fields, "UpdatedAt", d.UpdatedAt)

	sendTagFieldPack("ecs_deployment", tags, fields, now)
}

func (taskEcs *TaskECS) getAllTasks() (tasks []*string, err error) {
	svc := ecs.New(taskEcs.sess)
	cluster, getClusterErr := getECSCluster()
	if getClusterErr != nil {
		err = getClusterErr
		return
	}

	contInst, getContInstErr := getECSContainerInst()
	if getContInstErr != nil {
		err = getContInstErr
		return
	}

	var nextToken *string
	for {
		input := &ecs.ListTasksInput{Cluster: &cluster, ContainerInstance: &contInst, NextToken: nextToken}
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

func (taskEcs *TaskECS) listTasks(now int64) (err error) {
	svc := ecs.New(taskEcs.sess)

	cluster, getClusterErr := getECSCluster()
	if getClusterErr != nil {
		err = getClusterErr
		return
	}

	tasks, getTasksErr := taskEcs.getAllTasks()
	if getTasksErr != nil {
		err = getTasksErr
		return
	}

	input := &ecs.DescribeTasksInput{Cluster: &cluster, Tasks: tasks}

	result, descTasksErr := svc.DescribeTasks(input)

	if descTasksErr != nil {
		err = descTasksErr
		return
	}

	for _, f := range result.Failures {
		taskEcs.sendFailure(f)
	}

	for _, t := range result.Tasks {
		taskEcs.parseTask(t)
	}

	return
}

func (taskEcs *TaskECS) parseTask(t *ecs.Task) (err error) {
	tags := map[string]interface{}{}
	populateTag(t.Tags, func(k string, v string) {
		tags[k] = v
	})
	populateAttribute(t.Attributes, func(k string, v string) {
		tags[k] = v
	})

	setKeyStringValue(&tags, "AvailabilityZone", t.AvailabilityZone)
	setKeyStringValue(&tags, "Group", t.Group)
	setKeyStringValue(&tags, "ClusterArn", t.ClusterArn)
	setKeyStringValue(&tags, "Connectivity", t.Connectivity)
	setKeyStringValue(&tags, "DesiredStatus", t.DesiredStatus)
	setKeyStringValue(&tags, "CapacityProviderName", t.CapacityProviderName)
	setKeyStringValue(&tags, "ContainerInstanceArn", t.ContainerInstanceArn)
	setKeyStringValue(&tags, "TaskArn", t.TaskArn)
	setKeyStringValue(&tags, "TaskDefinitionArn", t.TaskDefinitionArn)

	fields := map[string]interface{}{}
	setKeyStringValue(&fields, "HealthStatus", t.HealthStatus)
	setKeyStringValue(&fields, "LastStatus", t.LastStatus)
	if t.Overrides != nil {
		setKeyStringValue(&fields, "OverridesCpu", t.Overrides.Cpu)
		setKeyStringValue(&fields, "OverridesMemory", t.Overrides.Memory)
	}

	setKeyStringValue(&fields, "Cpu", t.Cpu)
	setKeyStringValue(&fields, "Memory", t.Memory)

	setKeyStringValue(&fields, "DesiredStatus", t.DesiredStatus)
	setKeyStringValue(&fields, "LastStatus", t.LastStatus)
	setKeyStringValue(&fields, "LaunchType", t.LaunchType)
	setKeyStringValue(&fields, "StartedBy", t.StartedBy)

	setKeyTimeValue(&fields, "PullStartedAt", t.PullStartedAt)
	setKeyTimeValue(&fields, "PullStoppedAt", t.PullStoppedAt)
	setKeyTimeValue(&fields, "CreatedAt", t.CreatedAt)
	setKeyTimeValue(&fields, "StartedAt", t.StartedAt)
	setKeyTimeValue(&fields, "StoppedAt", t.StoppedAt)
	setKeyTimeValue(&fields, "StoppingAt", t.StoppingAt)

	for _, c := range t.Containers {
		cache.SetContainerCache(*c.RuntimeId, tags, fields)
	}

	return
}

func setKeyStringValue(tags *map[string]interface{}, key string, val *string) {
	if val != nil {
		(*tags)[key] = *val
	}
}

func setKeyInt64Value(tags *map[string]interface{}, key string, val *int64) {
	if val != nil {
		(*tags)[key] = *val
	}
}

func setKeyTimeValue(tags *map[string]interface{}, key string, val *time.Time) {
	if val != nil {
		(*tags)[key] = *val
	}
}

func populateTag(ecstags []*ecs.Tag, h2 func(string, string)) {
	if ecstags == nil {
		return
	}
	for _, ecstag := range ecstags {
		key := ecstag.Key
		val := ecstag.Value
		if key != nil && val != nil {
			h2(*key, *val)
		}
	}
}

func populateAttribute(attrs []*ecs.Attribute, h2 func(string, string)) {
	if attrs == nil {
		return
	}

	for _, a := range attrs {
		key := a.Name
		val := a.Value

		if key != nil && val != nil {
			h2(*key, *val)
		}
	}
}
