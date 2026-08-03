package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// La plantilla viaja dentro del binario: el deploy es una sola llamada a la API de
// CloudFormation, sin Node, sin npx y sin bootstrap stack.
//
//go:embed template.yml
var cloudFormationTemplate string

const stackEventsPollInterval = 4 * time.Second

// Despliega la infraestructura completa: crea el stack si no existe, o lo actualiza.
func DeployCloudFormation(params DeployParams) {
	stackName := params.APP_NAME + "-stack"

	awsConfig, err := MakeAwsConfig(params.AWS_PROFILE, params.AWS_REGION)
	if err != nil {
		panic("Error al cargar la configuración de AWS: " + err.Error())
	}
	client := cloudformation.NewFromConfig(awsConfig)
	ctx := context.TODO()

	templateParameters := []cfnTypes.Parameter{
		makeStackParameter("NamePrefix", params.APP_NAME),
		makeStackParameter("FrontendBucketName", params.FRONTEND_BUCKET),
		makeStackParameter("DeploymentBucket", params.DEPLOYMENT_BUCKET),
		makeStackParameter("CompiledS3Key", params.S3_COMPILED_PATH),
		makeStackParameter("LambdaIamRole", params.LAMBDA_IAM_ROLE),
		makeStackParameter("AppCode", appCodeEnvValue),
	}

	currentStatus, stackExists := DescribeStackStatus(ctx, client, stackName)

	// Un create fallido deja el stack en ROLLBACK_COMPLETE: en ese estado solo se puede
	// borrar, cualquier update es rechazado. Se avisa en vez de fallar con un error opaco.
	if stackExists && currentStatus == cfnTypes.StackStatusRollbackComplete {
		panic(fmt.Sprintf(
			"El stack %v está en ROLLBACK_COMPLETE (creación fallida previa). Bórralo antes de reintentar.",
			stackName))
	}

	// Solo se imprimen los eventos posteriores al arranque; los de deploys anteriores sobran.
	deployStartedAt := time.Now().Add(-stackEventsPollInterval)

	if stackExists {
		fmt.Printf("Actualizando stack %v (estado actual: %v)...\n", stackName, currentStatus)
		_, err = client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
			StackName:    &stackName,
			TemplateBody: &cloudFormationTemplate,
			Parameters:   templateParameters,
		})
		// CloudFormation responde con un error cuando la plantilla no cambió nada. No es un fallo.
		if err != nil && strings.Contains(err.Error(), "No updates are to be performed") {
			fmt.Println("Sin cambios en la infraestructura.")
			PrintStackOutputsAndSyncCredentials(ctx, client, stackName)
			return
		}
	} else {
		fmt.Printf("Creando stack %v...\n", stackName)
		_, err = client.CreateStack(ctx, &cloudformation.CreateStackInput{
			StackName:    &stackName,
			TemplateBody: &cloudFormationTemplate,
			Parameters:   templateParameters,
			// Sin esto un create fallido queda en ROLLBACK_COMPLETE y hay que borrarlo a mano.
			OnFailure: cfnTypes.OnFailureDelete,
		})
	}

	if err != nil {
		exitWithDeployError("Error al enviar la plantilla a CloudFormation: " + err.Error())
	}

	finalStatus, rootFailureReason := WaitForStackAndPrintEvents(ctx, client, stackName, deployStartedAt)

	// Un fallo real arrastra a los demás recursos con "Resource creation cancelled", así que se
	// repite la primera causa: es la única línea que dice de verdad qué hay que arreglar.
	if !isSuccessfulStackStatus(finalStatus) {
		message := fmt.Sprintf("El despliegue del stack %v FALLÓ (estado %v).", stackName, finalStatus)
		if len(rootFailureReason) > 0 {
			message += "\nCausa raíz: " + rootFailureReason
		}
		exitWithDeployError(message)
	}

	fmt.Printf("\nStack %v desplegado (%v)\n", stackName, finalStatus)
	PrintStackOutputsAndSyncCredentials(ctx, client, stackName)
}

// Termina con un mensaje legible en vez de un panic: un stack trace de Go no aporta nada
// cuando el error viene de AWS, y además tapa la causa raíz impresa arriba.
func exitWithDeployError(message string) {
	fmt.Println("\n" + message)
	os.Exit(1)
}

func makeStackParameter(key, value string) cfnTypes.Parameter {
	return cfnTypes.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value)}
}

// Devuelve el estado actual del stack y si existe. DescribeStacks falla cuando no existe.
func DescribeStackStatus(ctx context.Context, client *cloudformation.Client, stackName string) (cfnTypes.StackStatus, bool) {
	result, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{StackName: &stackName})
	if err != nil || len(result.Stacks) == 0 {
		return "", false
	}
	return result.Stacks[0].StackStatus, true
}

// Sondea el stack hasta que llega a un estado terminal, imprimiendo cada evento nuevo.
// Reemplaza el progreso que antes mostraba la CLI de CDK: cuando algo falla, el evento
// nombra el recurso exacto y la razón.
// Devuelve además la primera causa real de fallo, para poder repetirla al final.
func WaitForStackAndPrintEvents(ctx context.Context, client *cloudformation.Client,
	stackName string, since time.Time) (cfnTypes.StackStatus, string) {

	printedEventIDs := map[string]bool{}
	rootFailureReason := ""

	recordFailure := func(reason string) {
		if len(rootFailureReason) == 0 {
			rootFailureReason = reason
		}
	}

	for {
		recordFailure(PrintNewStackEvents(ctx, client, stackName, since, printedEventIDs))

		status, exists := DescribeStackStatus(ctx, client, stackName)
		// OnFailureDelete borra el stack tras un create fallido: deja de existir en pleno sondeo.
		if !exists {
			return "DELETED_AFTER_FAILURE", rootFailureReason
		}
		if isTerminalStackStatus(status) {
			// Última pasada: los eventos finales suelen llegar después del cambio de estado.
			time.Sleep(time.Second)
			recordFailure(PrintNewStackEvents(ctx, client, stackName, since, printedEventIDs))
			return status, rootFailureReason
		}

		time.Sleep(stackEventsPollInterval)
	}
}

// Imprime los eventos posteriores a "since" que aún no se han mostrado, del más antiguo
// al más reciente (la API los devuelve al revés). Devuelve la primera causa real de fallo
// del lote, si la hay.
func PrintNewStackEvents(ctx context.Context, client *cloudformation.Client,
	stackName string, since time.Time, printedEventIDs map[string]bool) string {

	result, err := client.DescribeStackEvents(ctx, &cloudformation.DescribeStackEventsInput{StackName: &stackName})
	if err != nil {
		return ""
	}

	pendingEvents := []cfnTypes.StackEvent{}
	for _, event := range result.StackEvents {
		if event.Timestamp == nil || event.Timestamp.Before(since) {
			continue
		}
		if event.EventId == nil || printedEventIDs[*event.EventId] {
			continue
		}
		printedEventIDs[*event.EventId] = true
		pendingEvents = append(pendingEvents, event)
	}

	sort.Slice(pendingEvents, func(a, b int) bool {
		return pendingEvents[a].Timestamp.Before(*pendingEvents[b].Timestamp)
	})

	rootFailureReason := ""
	for _, event := range pendingEvents {
		logicalID := aws.ToString(event.LogicalResourceId)
		reason := aws.ToString(event.ResourceStatusReason)

		line := fmt.Sprintf("  %v  %-34v %v",
			event.Timestamp.Local().Format("15:04:05"), logicalID, event.ResourceStatus)
		if len(reason) > 0 {
			line += " | " + reason
		}
		fmt.Println(line)

		// "Resource creation cancelled" es el efecto dominó del primer fallo, no una causa.
		if strings.HasSuffix(string(event.ResourceStatus), "_FAILED") && len(rootFailureReason) == 0 &&
			!strings.Contains(reason, "cancelled") {
			rootFailureReason = logicalID + ": " + reason
		}
	}

	return rootFailureReason
}

func isTerminalStackStatus(status cfnTypes.StackStatus) bool {
	text := string(status)
	return !strings.HasSuffix(text, "_IN_PROGRESS")
}

func isSuccessfulStackStatus(status cfnTypes.StackStatus) bool {
	return status == cfnTypes.StackStatusCreateComplete || status == cfnTypes.StackStatusUpdateComplete
}

// Muestra los outputs del stack y sincroniza LAMBDA_URL en credentials.json.
func PrintStackOutputsAndSyncCredentials(ctx context.Context, client *cloudformation.Client, stackName string) {
	result, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{StackName: &stackName})
	if err != nil || len(result.Stacks) == 0 {
		fmt.Println("No se pudieron leer los outputs del stack: ", err)
		return
	}

	outputs := map[string]string{}
	fmt.Println("\nOutputs del stack:")
	for _, output := range result.Stacks[0].Outputs {
		key, value := aws.ToString(output.OutputKey), aws.ToString(output.OutputValue)
		outputs[key] = value
		fmt.Printf("  %-28v %v\n", key, value)
	}

	backendUrl := outputs["BackendUrl"]
	if len(backendUrl) == 0 {
		fmt.Println("\nEl stack no devolvió BackendUrl; LAMBDA_URL no se modificó.")
		return
	}
	SyncLambdaUrlInCredentials(backendUrl)
}

var lambdaUrlInJsonPattern = regexp.MustCompile(`("LAMBDA_URL"\s*:\s*")([^"]*)(")`)

// Escribe la Function URL recién desplegada en credentials.json. Se hace por reemplazo de
// texto y no re-serializando el JSON para no perder el orden de las claves ni el formato
// del archivo, que se mantiene a mano.
func SyncLambdaUrlInCredentials(deployedLambdaUrl string) {
	credentialsPath := GetBaseWD() + "/credentials.json"

	credentialsBytes, err := ReadFile(credentialsPath)
	if err != nil {
		fmt.Println("\nNo se pudo leer credentials.json para actualizar LAMBDA_URL: ", err)
		return
	}

	credentialsText := string(credentialsBytes)
	currentMatch := lambdaUrlInJsonPattern.FindStringSubmatch(credentialsText)
	if currentMatch == nil {
		fmt.Println("\nNo se encontró la clave LAMBDA_URL en credentials.json; no se modificó nada.")
		fmt.Println("URL del backend desplegado: " + deployedLambdaUrl)
		return
	}

	previousLambdaUrl := currentMatch[2]
	if previousLambdaUrl == deployedLambdaUrl {
		fmt.Println("\nLAMBDA_URL ya apuntaba al backend desplegado: " + deployedLambdaUrl)
		return
	}

	updatedText := lambdaUrlInJsonPattern.ReplaceAllString(
		credentialsText, "${1}"+deployedLambdaUrl+"${3}")

	if err := os.WriteFile(credentialsPath, []byte(updatedText), 0644); err != nil {
		fmt.Println("\nNo se pudo escribir credentials.json: ", err)
		return
	}

	fmt.Println("\nLAMBDA_URL actualizado en credentials.json:")
	fmt.Println("  anterior: " + previousLambdaUrl)
	fmt.Println("  nuevo:    " + deployedLambdaUrl)

	// La URL cruda de Lambda no pasa por el dominio propio: si el valor anterior no era una
	// URL de Function URL, se acaba de perder un dominio personalizado y hay que reapuntarlo.
	if !strings.Contains(previousLambdaUrl, ".lambda-url.") {
		fmt.Println("  AVISO: el valor anterior era un dominio propio, no una Function URL.")
		fmt.Println("         Reapunta ese dominio a la nueva URL, o restaura el valor anterior.")
	}
}
