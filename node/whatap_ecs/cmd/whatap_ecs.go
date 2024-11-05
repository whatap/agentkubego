package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Region struct {
	Regions []struct {
		RegionName string `json:"RegionName"`
	} `json:"Regions"`
}

type Cluster struct {
	ClusterArns []string `json:"clusterArns"`
}

type Subnet struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

type SecurityGroup struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

type Role struct {
	Role struct {
		Arn string `json:"Arn"`
	} `json:"Role"`
}

type RoleExistenceCheck struct {
	Role struct {
		RoleName string `json:"RoleName"`
	} `json:"Role"`
}

type TaskDefinition struct {
	TaskDefinitionArns []string `json:"taskDefinitionArns"`
}

type ContainerDefinition struct {
	Name        string `json:"name"`
	Image       string `json:"image"`
	Essential   bool   `json:"essential"`
	Environment []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"environment"`
}

type TaskDefinitionDetail struct {
	Family                  string                `json:"family"`
	RequiresCompatibilities []string              `json:"requiresCompatibilities"`
	NetworkMode             string                `json:"networkMode"`
	Cpu                     string                `json:"cpu"`
	Memory                  string                `json:"memory"`
	ExecutionRoleArn        string                `json:"executionRoleArn"`
	TaskRoleArn             string                `json:"taskRoleArn"`
	ContainerDefinitions    []ContainerDefinition `json:"containerDefinitions"`
}

func getInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func runCommand(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	return string(output), err
}

func selectOption(options []string, prompt string) string {
	for i, option := range options {
		fmt.Printf("%d. %s\n", i+1, option)
	}
	choice := getInput(prompt)
	choiceIndex := 0
	fmt.Sscanf(choice, "%d", &choiceIndex)
	if choiceIndex < 1 || choiceIndex > len(options) {
		fmt.Println("Invalid choice")
		os.Exit(1)
	}
	return options[choiceIndex-1]
}

func checkRoleExists(roleName string) (bool, string) {
	output, err := runCommand("aws", []string{"iam", "get-role", "--role-name", roleName})
	if err != nil {
		return false, ""
	}

	var role RoleExistenceCheck
	if err := json.Unmarshal([]byte(output), &role); err != nil {
		return false, ""
	}
	return true, role.Role.RoleName
}

func main() {
	// Collect user input
	whatapAccessKey := getInput("Enter Whatap ACCESSKEY: ")
	whatapHost := getInput("Enter Whatap HOST: ")
	ecsTaskExecutionRoleName := getInput("Enter ECS Task Execution Role Name for Whatap-Single (default: whatap-ecs-single-execute): ")
	if ecsTaskExecutionRoleName == "" {
		ecsTaskExecutionRoleName = "whatap-ecs-single-execute"
	}
	appRoleName := getInput("Enter Monitoring Role Name for Whatap-Single (default: whatap-ecs-single-app): ")
	if appRoleName == "" {
		appRoleName = "whatap-ecs-single-app"
	}

	fmt.Println("Fetching available AWS regions...")
	regionOutput, err := runCommand("aws", []string{"ec2", "describe-regions", "--query", "Regions[*].RegionName", "--output", "json"})
	if err != nil || regionOutput == "" {
		fmt.Println("Failed to fetch regions")
		return
	}

	var regions []string
	if err := json.Unmarshal([]byte(regionOutput), &regions); err != nil {
		fmt.Println("Error parsing regions:", err)
		return
	}

	region := selectOption(regions, "Select REGION: ")

	fmt.Println("Fetching available ECS clusters...")
	clusterOutput, err := runCommand("aws", []string{"ecs", "list-clusters", "--region", region, "--query", "clusterArns", "--output", "json"})
	if err != nil || clusterOutput == "" {
		fmt.Println("Failed to fetch ECS clusters")
		return
	}

	var clusterArns []string
	if err := json.Unmarshal([]byte(clusterOutput), &clusterArns); err != nil {
		fmt.Println("Error parsing clusters:", err)
		return
	}

	cluster := selectOption(clusterArns, "Select CLUSTER: ")

	fmt.Println("Fetching available subnets...")
	subnetOutput, err := runCommand("aws", []string{"ec2", "describe-subnets", "--region", region, "--query", "Subnets[*].{ID:SubnetId,Name:Tags[?Key=='Name']|[0].Value}", "--output", "json"})
	if err != nil || subnetOutput == "" {
		fmt.Println("Failed to fetch subnets")
		return
	}

	var subnets []Subnet
	if err := json.Unmarshal([]byte(subnetOutput), &subnets); err != nil {
		fmt.Println("Error parsing subnets:", err)
		return
	}

	subnetOptions := make([]string, len(subnets))
	for i, subnet := range subnets {
		subnetName := subnet.ID
		if subnet.Name != "" {
			subnetName = fmt.Sprintf("%s (%s)", subnet.ID, subnet.Name)
		}
		subnetOptions[i] = subnetName
	}
	subnet1 := selectOption(subnetOptions, "Select SUBNET_1: ")
	subnet2 := selectOption(subnetOptions, "Select SUBNET_2: ")

	fmt.Println("Fetching available security groups...")
	sgOutput, err := runCommand("aws", []string{"ec2", "describe-security-groups", "--region", region, "--query", "SecurityGroups[*].{ID:GroupId,Name:GroupName}", "--output", "json"})
	if err != nil || sgOutput == "" {
		fmt.Println("Failed to fetch security groups")
		return
	}

	var securityGroups []SecurityGroup
	if err := json.Unmarshal([]byte(sgOutput), &securityGroups); err != nil {
		fmt.Println("Error parsing security groups:", err)
		return
	}

	sgOptions := make([]string, len(securityGroups))
	for i, sg := range securityGroups {
		sgName := sg.ID
		if sg.Name != "" {
			sgName = fmt.Sprintf("%s (%s)", sg.ID, sg.Name)
		}
		sgOptions[i] = sgName
	}
	sggrp := selectOption(sgOptions, "Select SGGRP: ")

	sggrpID := strings.Split(sggrp, " ")[0]

	// Create IAM roles if they don't exist
	fmt.Println("Checking and creating IAM roles if necessary...")

	executionRoleArn := ""
	appRoleArn := ""

	roleExists, _ := checkRoleExists(ecsTaskExecutionRoleName)
	if roleExists {
		fmt.Println("ECS Task Execution Role already exists")
		executionRoleOutput, _ := runCommand("aws", []string{"iam", "get-role", "--role-name", ecsTaskExecutionRoleName})
		var executionRole Role
		if err := json.Unmarshal([]byte(executionRoleOutput), &executionRole); err != nil {
			fmt.Println("Error parsing ECS Task Execution Role response:", err)
			return
		}
		executionRoleArn = executionRole.Role.Arn
	} else {
		fmt.Println("Creating ECS Task Execution Role...")
		executionRoleOutput, err := runCommand("aws", []string{"iam", "create-role", "--role-name", ecsTaskExecutionRoleName, "--assume-role-policy-document", `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Principal": {"Service": "ecs-tasks.amazonaws.com"}, "Action": "sts:AssumeRole"}]}`})
		if err != nil || executionRoleOutput == "" {
			fmt.Println("Failed to create ECS Task Execution Role")
			return
		}
		var executionRole Role
		if err := json.Unmarshal([]byte(executionRoleOutput), &executionRole); err != nil {
			fmt.Println("Error parsing ECS Task Execution Role creation response:", err)
			return
		}
		executionRoleArn = executionRole.Role.Arn
		runCommand("aws", []string{"iam", "attach-role-policy", "--role-name", ecsTaskExecutionRoleName, "--policy-arn", "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"})
	}

	roleExists, _ = checkRoleExists(appRoleName)
	if roleExists {
		fmt.Println("Application Role already exists")
		appRoleOutput, _ := runCommand("aws", []string{"iam", "get-role", "--role-name", appRoleName})
		var appRole Role
		if err := json.Unmarshal([]byte(appRoleOutput), &appRole); err != nil {
			fmt.Println("Error parsing Application Role response:", err)
			return
		}
		appRoleArn = appRole.Role.Arn
	} else {
		fmt.Println("Creating Application Role...")
		appRoleOutput, err := runCommand("aws", []string{"iam", "create-role", "--role-name", appRoleName, "--assume-role-policy-document", `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Principal": {"Service": "ecs-tasks.amazonaws.com"}, "Action": "sts:AssumeRole"}]}`})
		if err != nil || appRoleOutput == "" {
			fmt.Println("Failed to create Application Role")
			return
		}
		var appRole Role
		if err := json.Unmarshal([]byte(appRoleOutput), &appRole); err != nil {
			fmt.Println("Error parsing Application Role creation response:", err)
			return
		}
		appRoleArn = appRole.Role.Arn
		runCommand("aws", []string{"iam", "put-role-policy", "--role-name", appRoleName, "--policy-name", "app_policy", "--policy-document", `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["ecs:Describe*", "ecs:List*"], "Resource": "*"}]}`})
	}

	// Generate the AWS CLI commands for creating the ECS service
	taskDefinition := fmt.Sprintf(`
cat > whatap_ecs_task_definition.json <<EOL
{
	"family": "whatap-single",
	"requiresCompatibilities": ["FARGATE"],
	"networkMode": "awsvpc",
	"cpu": "256",
	"memory": "512",
	"executionRoleArn": "%s",
	"taskRoleArn": "%s",
	"containerDefinitions": [
		{
			"name": "whatap-single",
			"image": "whatap/ecs_mon",
			"essential": true,
			"environment": [
				{
					"name": "ACCESSKEY",
					"value": "%s"
				},
				{
					"name": "WHATAP_HOST",
					"value": "%s"
				},
				{
					"name": "FARGATE_HELPER",
					"value": "true"
				}
			]
		}
	]
}
EOL

aws ecs register-task-definition \
	--cli-input-json file://$(pwd)/whatap_ecs_task_definition.json \
	--region %s
`, executionRoleArn, appRoleArn, whatapAccessKey, whatapHost, region)

	serviceCreation := fmt.Sprintf(`
aws ecs create-service \
	--cluster %s \
	--service-name whatap-single \
	--task-definition whatap-single \
	--launch-type FARGATE \
	--desired-count 1 \
	--network-configuration "awsvpcConfiguration={subnets=[%s,%s],securityGroups=[%s],assignPublicIp=ENABLED}" \
	--region %s
`, cluster, subnet1, subnet2, sggrpID, region)

	fmt.Println("Generating AWS CLI commands...")
	fmt.Println(taskDefinition)
	fmt.Println(serviceCreation)

	// Optionally write the scripts to files
	err = os.WriteFile("01_whatap_single_task_definition.sh", []byte(taskDefinition), 0755)
	if err != nil {
		fmt.Println("Error writing task definition script:", err)
		return
	}
	err = os.WriteFile("02_whatap_single_service.sh", []byte(serviceCreation), 0755)
	if err != nil {
		fmt.Println("Error writing service creation script:", err)
		return
	}

	fmt.Println("Scripts have been generated and saved to '01_whatap_single_task_definition.sh' and '02_whatap_single_service.sh'.")
	fmt.Println("Run 'bash 01_whatap_single_task_definition.sh' to create the task definition.")
	fmt.Println("Then run 'bash 02_whatap_single_service.sh' to create the ECS service.")

	// Add sidecar container to an existing task definition
	fmt.Println("Fetching available task definitions...")
	taskDefOutput, err := runCommand("aws", []string{"ecs", "list-task-definitions", "--region", region, "--query", "taskDefinitionArns", "--output", "json"})
	if err != nil || taskDefOutput == "" {
		fmt.Println("Failed to fetch task definitions")
		return
	}

	var taskDefinitions []string
	if err := json.Unmarshal([]byte(taskDefOutput), &taskDefinitions); err != nil {
		fmt.Println("Error parsing task definitions:", err)
		return
	}

	taskDefinitionArn := selectOption(taskDefinitions, "Select TASK DEFINITION to add sidecar: ")

	fmt.Println("Fetching selected task definition details...")
	taskDefDetailOutput, err := runCommand("aws", []string{"ecs", "describe-task-definition", "--task-definition", taskDefinitionArn, "--region", region, "--query", "taskDefinition", "--output", "json"})
	if err != nil || taskDefDetailOutput == "" {
		fmt.Println("Failed to fetch task definition details")
		return
	}

	var taskDefinitionDetail TaskDefinitionDetail
	if err := json.Unmarshal([]byte(taskDefDetailOutput), &taskDefinitionDetail); err != nil {
		fmt.Println("Error parsing task definition details:", err)
		return
	}

	sidecarContainer := ContainerDefinition{
		Name:      "whatap-sidecar",
		Image:     "whatap/ecs_mon:latest",
		Essential: true,
		Environment: []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}{
			{"ACCESSKEY", whatapAccessKey},
			{"WHATAP_HOST", whatapHost},
		},
	}

	taskDefinitionDetail.ContainerDefinitions = append(taskDefinitionDetail.ContainerDefinitions, sidecarContainer)

	updatedTaskDefinition := TaskDefinitionDetail{
		Family:                  taskDefinitionDetail.Family,
		RequiresCompatibilities: taskDefinitionDetail.RequiresCompatibilities,
		NetworkMode:             taskDefinitionDetail.NetworkMode,
		Cpu:                     taskDefinitionDetail.Cpu,
		Memory:                  taskDefinitionDetail.Memory,
		ExecutionRoleArn:        taskDefinitionDetail.ExecutionRoleArn,
		TaskRoleArn:             appRoleArn, // Apply the whatap single app role
		ContainerDefinitions:    taskDefinitionDetail.ContainerDefinitions,
	}

	updatedTaskDefJson, err := json.MarshalIndent(updatedTaskDefinition, "", "  ")
	if err != nil {
		fmt.Println("Error generating updated task definition JSON:", err)
		return
	}

	updatedTaskDefScript := fmt.Sprintf(`
cat > updated_task_definition.json <<EOL
%s
EOL

aws ecs register-task-definition \
	--cli-input-json file://$(pwd)/updated_task_definition.json \
	--region %s
`, string(updatedTaskDefJson), region)

	err = os.WriteFile("03_user_task_definition_update.sh", []byte(updatedTaskDefScript), 0755)
	if err != nil {
		fmt.Println("Error writing update task definition script:", err)
		return
	}

	fmt.Println("Script to update task definition with sidecar has been generated and saved to '03_user_task_definition_update.sh'.")
	fmt.Println("Run 'bash 03_user_task_definition_update.sh' to update the task definition with the Whatap sidecar.")
}
