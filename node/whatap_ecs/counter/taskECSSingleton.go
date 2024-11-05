package counter

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func (taskEcsSingleton *TaskECSSingleton) init() {
	region, regionErr := getEcsRegion()
	if regionErr != nil {
		log.Println("TaskECSSingleton error:", regionErr.Error())
		return
	}
	sess, sessionErr := session.NewSession(&aws.Config{Region: &region})
	if sessionErr == nil {
		taskEcsSingleton.sess = sess
	} else {
		log.Println("TaskECSSingleton error:", sessionErr.Error())
	}
}

func (taskEcsSingleton *TaskECSSingleton) interval() int {

	return 10
}

func (taskEcsSingleton *TaskECSSingleton) process(now int64) (err error) {
	if taskEcsSingleton.sess == nil {
		return fmt.Errorf("session not ready")
	}

	listServErr := taskEcsSingleton.listServices(now)
	if listServErr != nil {
		err = listServErr
		return
	}

	return

}

func (taskEcsSingleton *TaskECSSingleton) getAllServices() (services []*string, err error) {
	svc := ecs.New(taskEcsSingleton.sess)
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

func (taskEcsSingleton *TaskECSSingleton) getAllServicesEx(onFound func([]*string)) (err error) {
	svc := ecs.New(taskEcsSingleton.sess)
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
			onFound(result.ServiceArns)
		}

		if result.NextToken == nil || len(*result.NextToken) < 1 {
			break
		}
		nextToken = result.NextToken
	}

	return
}

func (taskEcsSingleton *TaskECSSingleton) listServices(now int64) (err error) {
	svc := ecs.New(taskEcsSingleton.sess)
	cluster, getClusterErr := getECSCluster()
	if getClusterErr != nil {
		err = getClusterErr
		return
	}

	onServicesFound := func(services []*string) {
		input := &ecs.DescribeServicesInput{Cluster: &cluster, Services: services}

		result, descServicesErr := svc.DescribeServices(input)

		if descServicesErr != nil {
			err = descServicesErr
			return
		}

		for _, f := range result.Failures {
			taskEcsSingleton.sendFailure(f)
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
				taskEcsSingleton.sendDeployment(*s.ServiceArn, *s.ServiceName, d, now)
			}
		}
	}
	getServicesErr := taskEcsSingleton.getAllServicesEx(onServicesFound)
	if getServicesErr != nil {
		err = getServicesErr
		return
	}

	return
}

func (taskEcsSingleton *TaskECSSingleton) sendFailure(f *ecs.Failure) {

}

func (taskEcsSingleton *TaskECSSingleton) sendDeployment(serviceArn string, serviceName string, d *ecs.Deployment, now int64) {
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
