package aws

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaExecOutput struct {
	Body       string
	StatusCode int32
	Metadata   map[any]any
	Logs       string
	Error      string
	Response   core.FuncResponse
}

type ExecLambdaInput struct {
	ExecArgs core.ExecArgs `json:"fn_exec"`
}

func ExecLambda(args core.ExecArgs) LambdaExecOutput {

	core.Log("*Ejecutando función ASYNC:: " /*args.Param6, " | ",*/, args.FuncToExec)
	core.LogsSaved = []string{}

	input := ExecLambdaInput{ExecArgs: args}
	body := core.ToJsonNoErr(input)
	lambdaAPIResponse := []byte{}

	// core.Log("*Body a enviar a Lambda", body)
	output := LambdaExecOutput{Response: core.FuncResponse{}}
	nowTime := time.Now().Unix()

	// Si es local, entonces envía hacia el backend local
	if core.Env.IS_LOCAL {
		bodyBuff := bytes.NewBuffer([]byte(body))
		req, _ := http.NewRequest("POST", core.Env.API_ROUTE, bodyBuff)
		req.Header.Set("Content-Type", "plain/text")
		// Debe ser asíncrono para que no bloquee la ejecución en curso
		core.Log("Ivocando Lambda paralelta en local:", core.Env.API_ROUTE)
		if args.InvokeAsEvent {
			reponseBody := map[string]any{} //TODO: COMPLETE HERE
			go core.SendHttpRequestNoTLS(req, &reponseBody)
			return LambdaExecOutput{}
		} else {
			lambdaAPIResponse, _ = core.SendHttpRequestT(req, false)
		}
	} else {
		payloadBytes, _ := json.Marshal(map[string]string{"body": body})

		invokeInput := lambda.InvokeInput{
			FunctionName: &args.LambdaName,
			LogType:      "None",
			Payload:      payloadBytes,
		}

		if args.InvokeAsEvent {
			invokeInput.InvocationType = "Event"
		}

		client := lambda.NewFromConfig(core.GetAwsConfig())
		response, err := client.Invoke(context.TODO(), &invokeInput)

		if err != nil {
			output.Error = err.Error()
			core.Log("*Función Ejecutada (ERROR): ", args.FuncToExec, "|", err.Error())
		} else {
			output.StatusCode = response.StatusCode

			lambdaAPIResponse = response.Payload
			core.Log("*Lambda Reponse:", core.StrCut(string(lambdaAPIResponse), 400))
			if response.LogResult != nil {
				output.Logs = *response.LogResult
			}
			if response.FunctionError != nil {
				output.Error = *response.FunctionError
			}
			elapsedTime := int(time.Now().Unix() - nowTime)
			core.Log("*Función Ejecutada (END): ", args.FuncToExec, "|", core.Concats(elapsedTime, "s"))
		}
	}

	response := events.APIGatewayV2HTTPResponse{}

	err := json.Unmarshal(lambdaAPIResponse, &response)
	if err != nil {
		output.Error = fmt.Sprintf("Error al parsear el Lambda Response: %v", err)
	} else {
		if args.ParseResponse && !args.InvokeAsEvent {
			err := json.Unmarshal([]byte(response.Body), &output.Response)
			if err != nil {
				output.Error = fmt.Sprintf("Error al parsear el body: %v", err)
			}
			if len(output.Response.Error) > 0 {
				output.Error = output.Response.Error
			}
		} else {
			output.Body = response.Body
		}
	}
	return output
}

type LambdaEnviromentArgs struct {
	Account    uint8
	LambdaName string
	Variables  map[string]string
	ClearAll   bool
}

func UpdateLambdaEnvVariables(args LambdaEnviromentArgs) error {

	inputGet := lambda.GetFunctionConfigurationInput{
		FunctionName: &args.LambdaName,
	}

	client := lambda.NewFromConfig(core.GetAwsConfig())
	response1, err := client.GetFunctionConfiguration(context.TODO(), &inputGet)

	if err != nil {
		return core.Err("Error al obtener los parámetros de la Lambda.", err)
	}

	core.Print(response1.Environment.Variables)

	Variables := response1.Environment.Variables
	for key, v := range args.Variables {
		Variables[key] = v
	}

	inputPost := lambda.UpdateFunctionConfigurationInput{
		FunctionName: &args.LambdaName,
		Environment: &lambdaTypes.Environment{
			Variables: Variables,
		},
	}

	response2, err := client.UpdateFunctionConfiguration(context.TODO(), &inputPost)
	if err != nil {
		return core.Err("Error al actualizar los parámetros de la Lambda.", err)
	}

	core.Log("Variables guardadas::")
	core.Print(response2.Environment.Variables)

	return nil
}

func GetLambdaEnvVariables(args LambdaEnviromentArgs) (map[string]string, error) {

	inputGet := lambda.GetFunctionConfigurationInput{
		FunctionName: &args.LambdaName,
	}

	client := lambda.NewFromConfig(core.GetAwsConfig())
	response1, err := client.GetFunctionConfiguration(context.TODO(), &inputGet)

	if err != nil {
		return nil, core.Err("Error al obtener los parámetros de la Lambda.", err)
	}

	return response1.Environment.Variables, nil
}

type DescribeECSTasksArgs struct {
	Account uint8
	Cluster string
	Tasks   []string
}
