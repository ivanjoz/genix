package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type DeployParams struct {
	APP_NAME          string `json:"APP_NAME"`
	AWS_PROFILE       string `json:"AWS_PROFILE"`
	AWS_REGION        string `json:"AWS_REGION"`
	STACK_NAME        string `json:"STACK_NAME"`
	DEPLOYMENT_BUCKET string
	FRONTEND_BUCKET   string
	LAMBDA_IAM_ROLE   string
	S3_COMPILED_PATH  string
}

const s3CompiledPath = "gerp-artifacts/lambda-compiled.zip"
const compilePath = "/cloud/main-compiled"

func GetBaseWD() string {
	wd, _ := os.Getwd()
	dirname := strings.Split(wd, "/")
	return strings.Join(dirname[:(len(dirname)-1)], "/")
}

func main() {
	w1 := ""

	for _, arg := range os.Args {
		if len(arg) > 7 && arg[:7] == "accion=" {
			w1 = strings.Split(arg, "=")[1]
			break
		}
	}

	if len(w1) == 0 {
		fmt.Println("Selecciona acción: [1] Publicar Código [2] Actualizar Variables [3] Deplegar Infraestructura")

		_, err := fmt.Scanln(&w1)
		if err != nil {
			panic(err)
		}
	}

	baseWD := GetBaseWD()
	credentialsJson, err := ReadFile(baseWD + "/credentials.json")
	if err != nil {
		panic(err)
	}

	params := DeployParams{S3_COMPILED_PATH: s3CompiledPath}
	err = json.Unmarshal(credentialsJson, &params)

	if err != nil {
		panic("Error parsing credentials.json:" + err.Error())
	}

	if w1 == "cdk" {
		StartCDK(params)
		return
	}

	missingParams := []string{params.STACK_NAME, params.DEPLOYMENT_BUCKET,
		params.AWS_PROFILE, params.AWS_REGION, params.LAMBDA_IAM_ROLE}

	for _, e := range missingParams {
		if len(e) == 0 {
			panic("Los parámetros STACK_NAME, DEPLOYMENT_BUCKET, AWS_REGION, AWS_PROFILE y LAMBDA_IAM_ROLE son requeridos.")
		}
	}

	if len(params.FRONTEND_BUCKET) == 0 {
		params.FRONTEND_BUCKET = params.STACK_NAME + "-frontend"
	}

	if w1 == "1" {
		CompileBackendToS3(params, false)
		DeployLambda(params, 0)
		CompileBackendToS3(params, false)
		DeployLambda(params, 2)
	} else if w1 == "2" {
		UpdateEnviromentVariables(params, 0)
		UpdateEnviromentVariables(params, 2)
	} else if w1 == "3" {
		DeployIfraestructure(params)
	} else {
		fmt.Println("No se reconoció la opción seleccionada.")
	}

	fmt.Println("Presione cualquier tecla para cerrar.")
	fmt.Scanln()
}

// Compila el código Backend y lo envía a S3
func CompileBackendToS3(params DeployParams, sendToS3 bool) {

	compiledPath := GetBaseWD() + compilePath
	command := `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags '-s -w' -o %v`

	command = fmt.Sprintf(command, compiledPath)
	fmt.Println("Compilando con:: ", command)
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = GetBaseWD() + "/backend/"

	stdout, err := cmd.Output()
	if err != nil {
		panic("Error generar el compilado: " + err.Error())
	}
	fmt.Println(stdout)

	compiledZipPath, err := CompressExeAndArgs(compiledPath)

	if err != nil {
		panic("Error comprimir el compilado en .zip: " + err.Error())
	}

	if sendToS3 {
		awsConfig, _ := MakeAwsConfig(params.AWS_PROFILE, params.AWS_REGION)
		s3Client := s3.NewFromConfig(awsConfig)

		s3Args := FileToS3Args{
			Bucket:        params.DEPLOYMENT_BUCKET,
			LocalFilePath: compiledZipPath,
			FilePath:      params.S3_COMPILED_PATH,
		}

		fmt.Println("Enviando compilado a S3: ", s3Args.Bucket+" | "+s3Args.FilePath)
		SendFileToS3Client(s3Args, s3Client)
	}
}

// Despliega el código compilado de la Lambda
func DeployLambda(params DeployParams, lambdaNro uint8) {

	lambdaName := params.STACK_NAME + "-backend"
	if lambdaNro == 2 {
		lambdaName += "_2"
	}

	zipFile, err := ReadFile(GetBaseWD() + compilePath + ".zip")

	if err != nil {
		panic("Error al leer el compilado.zip: " + err.Error())
	}

	fmt.Println("Enviando .zip con el compilado del backend...")
	zipLen := int(float64(len(zipFile)) / 1024)
	fmt.Printf("Tamaño del compilado: %v kb\n", zipLen)

	deployInput := lambda.UpdateFunctionCodeInput{
		FunctionName: &lambdaName,
		ZipFile:      zipFile,
	}

	awsConfig, _ := MakeAwsConfig(params.AWS_PROFILE, params.AWS_REGION)
	client := lambda.NewFromConfig(awsConfig)

	_, err = client.UpdateFunctionCode(context.TODO(), &deployInput)

	if err != nil {
		panic("Error al actualizar el código de la Lambda: " + err.Error())
	}
	fmt.Println("Código de la Lambda actualizado!")
}

// Actualiza las variables de entorno de la Lambda
func UpdateEnviromentVariables(params DeployParams, lambdaNro uint8) {

	lambdaName := params.STACK_NAME + "-backend"
	if lambdaNro == 2 {
		lambdaName += "_2"
	}

	fmt.Println("Actulizando variables de entorno...")
	enviromentVars := map[string]any{}

	jsonFileBytes, err := ReadFile(GetBaseWD() + "/credentials.json")
	if err != nil {
		panic("No se pudo leer el archivo credentials.json: " + err.Error())
	}

	err = json.Unmarshal(jsonFileBytes, &enviromentVars)
	if err != nil {
		panic("EL archivo credentials.json posee un formato erróneo: " + err.Error())
	}

	fmt.Println("Leyendo y comprimiendo credentials.json...")

	jsonString := string(jsonFileBytes)
	jsonBase64 := BytesToBase64(CompressZstd(&jsonString), true)

	variables := map[string]string{
		"APP_CODE": "gerp_prd", "CONFIG": jsonBase64,
	}

	configInput := lambda.UpdateFunctionConfigurationInput{
		FunctionName: &lambdaName,
		Environment: &lambdaTypes.Environment{
			Variables: variables,
		},
	}

	awsConfig, _ := MakeAwsConfig(params.AWS_PROFILE, params.AWS_REGION)
	client := lambda.NewFromConfig(awsConfig)

	fmt.Println("Enviando actualización a AWS Lambda...")

	_, err = client.UpdateFunctionConfiguration(context.TODO(), &configInput)
	if err != nil {
		fmt.Println("Error al actualizar los parámetros de la Lambda. ", err)
		return
	}

	fmt.Println("Variables actualizadas!")
}

// Despliega la infraestructura
func DeployIfraestructure(params DeployParams) {
	CompileBackendToS3(params, true)

	fmt.Println("Desplegando infraestructura con CDK...")

	// Ejecuta CDK deploy
	// Usamos cdk.json para definir el comando --app
	command := fmt.Sprintf("npx --yes aws-cdk@latest deploy %v-stack --require-approval never --profile %v", params.STACK_NAME, params.AWS_PROFILE)

	fmt.Println("Comando:: ", command)
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = GetBaseWD() + "/cloud/"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		panic("Error al ejecutar CDK deploy: " + err.Error())
	}
}
