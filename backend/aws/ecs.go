package aws

import (
	"app/core"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type ExecContainerOutput struct {
	Failures       []types.Failure
	Tasks          []types.Task
	ResultMetadata map[string]any
	TaskID         string
	Error          string
	Failure        string
}

type ExecContainerArgs struct {
	Account             uint8
	Cluster             string
	ContainerName       string
	TaskDefinition      string
	ResultMeTaskRoleArn string
	CapacityProvider    string
	Subnets             []string
	SecurityGroups      []string
	Command             []string
	Commands            []string
	EnviromentVariables [][]string
	AssignPublicIp      bool
	WaitUntilStart      int8
	MaxExecTimeSeconds  int32
}

func ExecContainer(args ExecContainerArgs) ExecContainerOutput {

	client := ecs.NewFromConfig(core.GetAwsConfig(args.Account))

	var publicIp types.AssignPublicIp = "ENABLED"
	if !args.AssignPublicIp {
		publicIp = "DISABLED"
	}

	Environment := []types.KeyValuePair{}
	for _, e := range args.EnviromentVariables {
		Environment = append(Environment, types.KeyValuePair{Name: &e[0], Value: &e[1]})
	}

	count := int32(1)

	commands := []string{}
	if len(args.Commands) > 0 {
		commandsJoined := strings.Join(args.Commands, " && ")
		if args.MaxExecTimeSeconds > 0 {
			commandsJoined = fmt.Sprintf(`(sleep %v && echo 'Killing container process...' && ps -eo pid | grep -v -w '1\|PID' | xargs -I {} kill {}) & (%v)`, args.MaxExecTimeSeconds, commandsJoined)
		}

		commands = append(commands, fmt.Sprintf(`/bin/sh -c "%s"`, commandsJoined))
	} else {
		commands = args.Command
	}

	input := ecs.RunTaskInput{
		Cluster:              &args.Cluster,
		Count:                &count,
		TaskDefinition:       &args.TaskDefinition,
		EnableExecuteCommand: true,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        args.Subnets,
				SecurityGroups: args.SecurityGroups,
				AssignPublicIp: publicIp,
			},
		},
		CapacityProviderStrategy: []types.CapacityProviderStrategyItem{{
			CapacityProvider: &args.CapacityProvider,
			Weight:           1,
		}},
		Overrides: &types.TaskOverride{
			ContainerOverrides: []types.ContainerOverride{
				{
					Name:        &args.ContainerName,
					Command:     commands,
					Environment: Environment,
				},
			},
			TaskRoleArn: &args.ResultMeTaskRoleArn,
		},
	}

	// core.Print(input)

	awsOutput, err := client.RunTask(context.TODO(), &input)

	output := ExecContainerOutput{}
	if err != nil {
		core.Print(err.Error())
		output.Error = err.Error()
	}

	if awsOutput != nil {
		core.Print(awsOutput.Failures)
		core.Print(awsOutput.Tasks)

		output.Failures = awsOutput.Failures
		output.Tasks = awsOutput.Tasks
		if len(output.Tasks) > 0 {
			task := output.Tasks[0]
			taskSlice := strings.Split((*task.TaskArn), "/")
			output.TaskID = taskSlice[len(taskSlice)-1]
		}
		if len(output.Failures) > 0 {
			if output.Failures[0].Reason != nil {
				output.Failure = *output.Failures[0].Reason
			}
			if output.Failures[0].Detail != nil {
				output.Error = *output.Failures[0].Detail
			}
		}
	}

	statusPending := []string{"PROVISIONING", "PENDING", "ACTIVATING"}

	if args.WaitUntilStart > 0 && len(output.TaskID) > 0 {
		provisionTime := time.Now().UnixMilli()

		input := ecs.DescribeTasksInput{
			Tasks:   []string{output.TaskID},
			Cluster: &args.Cluster,
		}

		maxAttemps := 60
		currentAttemps := 0
		for currentAttemps < maxAttemps {
			var sleepTime time.Duration = 2
			if currentAttemps > 10 {
				sleepTime = 1
			}
			time.Sleep(sleepTime * time.Second)
			currentAttemps++

			statusOutput, err := client.DescribeTasks(context.TODO(), &input)

			if err != nil {
				core.Print(err.Error())
				output.Error = err.Error()
				break
			}

			if len(statusOutput.Failures) > 0 {
				output.Failure = *statusOutput.Failures[0].Reason
				output.Error = *statusOutput.Failures[0].Detail
				break
			}

			if len(statusOutput.Tasks) == 0 {
				output.Error = "El Task no se encontró."
				break
			}

			elapsedMs := time.Now().UnixMilli() - provisionTime
			elapsed := int32(math.Round(float64(elapsedMs) / 1000))
			statusName := *statusOutput.Tasks[0].LastStatus
			core.Log("Status:: ", statusName, " | ", elapsed, "s")

			if !core.Contains(statusPending, statusName) {
				if statusName != "RUNNING" && len(output.Error) == 0 {
					output.Error = "Estado del contenedor no reconocido: " + statusName
				}
				break
			}
		}
		if maxAttemps == currentAttemps {
			output.Error = "El contenedor demasiado mucho en aprovisionarse."
		}
	}

	cmd := `aws --profile hortifrut_prod ecs execute-command --cluster smartberry --interactive --task %s --command "/bin/bash"`

	if len(output.TaskID) > 0 {
		cmd = fmt.Sprintf(cmd, output.TaskID)
		core.Log("Puede ingresar al contendor:: ")
		core.Log(cmd)
	}

	return output
}
