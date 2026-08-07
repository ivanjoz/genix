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
	// APP_NAME prefija todo nombre físico: stack, Lambdas, tabla DynamoDB, buckets.
	APP_NAME          string `json:"APP_NAME"`
	AWS_PROFILE       string `json:"AWS_PROFILE"`
	AWS_REGION        string `json:"AWS_REGION"`
	DEPLOYMENT_BUCKET string
	FRONTEND_BUCKET   string
	LAMBDA_IAM_ROLE   string
	S3_COMPILED_PATH  string
	// BACKEND_PROVIDER selects the cloud data mirror and AWS backend infrastructure.
	BACKEND_PROVIDER string `json:"BACKEND_PROVIDER"`
	// CDN_PROVIDER selects the object storage and public frontend origin.
	CDN_PROVIDER       string `json:"CDN_PROVIDER"`
	CLOUDFLARE_ACCOUNT string `json:"CLOUDFLARE_ACCOUNT"`
	CLOUDFLARE_TOKEN   string `json:"CLOUDFLARE_TOKEN"`
	// CLOUDFLARE_BUCKET fija el bucket R2 cuando su nombre no sigue el patrón "<APP_NAME>-files":
	// renombrar la app no debe apuntar el deploy a un bucket vacío. Vacío = nombre autogenerado.
	CLOUDFLARE_BUCKET string `json:"CLOUDFLARE_BUCKET"`
	// Origen público del CDN. La Lambda de render lo necesita para construir la base de
	// assets de cada company (<FRONTEND_CDN>/websites/<companyID>).
	FRONTEND_CDN string `json:"FRONTEND_CDN"`
	// URL del artefacto webpage-renderer.zip que publica CI. Vacío = el valor por defecto
	// de cloud/webpage-renderer.go.
	WEBPAGE_RENDERER_URL string `json:"WEBPAGE_RENDERER_URL"`
}

const s3CompiledPath = "gerp-artifacts/lambda-compiled.zip"
const compilePath = "/cloud/main-compiled"

// El backend deduce IS_PROD con strings.Contains(APP_CODE, "_prd"), con guion bajo
// (backend/core/security.go). Un guion normal apagaría el modo producción en silencio, así
// que el valor vive aquí una sola vez y lo usan tanto la plantilla como la acción 2.
const appCodeEnvValue = "gerp_prd"

// Debe coincidir con el InvokeMode de las Function URL en template.yml (RESPONSE_STREAM).
// El handler de Go elige la forma de su respuesta con esta variable y un desajuste rompe
// todas las peticiones, así que las dos se cambian juntas.
const lambdaResponseStreamingFlag = "1"

func GetBaseWD() string {
	wd, _ := os.Getwd()
	dirname := strings.Split(wd, "/")
	return strings.Join(dirname[:(len(dirname)-1)], "/")
}

// GetCredentialsPath keeps every cloud read and write on the environment selected by deploy.sh.
func GetCredentialsPath() string {
	configuredCredentialsPath := strings.TrimSpace(os.Getenv("GENIX_CREDENTIALS_FILE"))
	if configuredCredentialsPath != "" {
		return configuredCredentialsPath
	}
	return GetBaseWD() + "/credentials.json"
}

func main() {
	w1 := ""

	// First check for standalone valid arguments
	validArgs := map[string]bool{
		"1": true,
		"2": true,
		"3": true,
	}
	for _, arg := range os.Args {
		if validArgs[arg] {
			w1 = arg
			break
		}
	}

	// If no standalone argument found, check for accion= prefix
	if len(w1) == 0 {
		for _, arg := range os.Args {
			if len(arg) > 7 && arg[:7] == "accion=" {
				w1 = strings.Split(arg, "=")[1]
				break
			}
		}
	}

	// If still no valid argument found, prompt for interactive input
	if len(w1) == 0 {
		fmt.Println("Selecciona acción: [1] Publicar Código [2] Actualizar Variables [3] Deplegar Infraestructura")

		_, err := fmt.Scanln(&w1)
		if err != nil {
			panic(err)
		}
	}

	credentialsPath := GetCredentialsPath()
	fmt.Println("Leyendo credenciales desde:", credentialsPath)
	credentialsJson, err := ReadFile(credentialsPath)
	if err != nil {
		panic(err)
	}

	params := DeployParams{S3_COMPILED_PATH: s3CompiledPath}
	err = json.Unmarshal(credentialsJson, &params)

	if err != nil {
		panic(fmt.Sprintf("Error parsing credentials file %s: %v", credentialsPath, err))
	}

	if params.BACKEND_PROVIDER != "aws" && params.BACKEND_PROVIDER != "cloudflare" && params.BACKEND_PROVIDER != "none" {
		panic("BACKEND_PROVIDER debe ser 'aws', 'cloudflare' o 'none'.")
	}
	if params.CDN_PROVIDER != "aws" && params.CDN_PROVIDER != "cloudflare" {
		panic("CDN_PROVIDER debe ser 'aws' o 'cloudflare'.")
	}
	if w1 != "3" || params.BACKEND_PROVIDER == "aws" {
		missingParams := []string{params.APP_NAME, params.DEPLOYMENT_BUCKET,
			params.AWS_PROFILE, params.AWS_REGION, params.LAMBDA_IAM_ROLE}

		for _, e := range missingParams {
			if len(e) == 0 {
				panic("Los parámetros APP_NAME, DEPLOYMENT_BUCKET, AWS_REGION, AWS_PROFILE y LAMBDA_IAM_ROLE son requeridos.")
			}
		}
	}
	if params.CDN_PROVIDER == "cloudflare" && (strings.TrimSpace(params.CLOUDFLARE_ACCOUNT) == "" || strings.TrimSpace(params.CLOUDFLARE_TOKEN) == "") {
		panic("CLOUDFLARE_ACCOUNT y CLOUDFLARE_TOKEN son requeridos cuando CDN_PROVIDER es 'cloudflare'.")
	}

	if len(params.FRONTEND_BUCKET) == 0 {
		params.FRONTEND_BUCKET = params.APP_NAME + "-frontend"
	}

	params.CLOUDFLARE_BUCKET = strings.TrimSpace(params.CLOUDFLARE_BUCKET)
	if len(params.CLOUDFLARE_BUCKET) == 0 {
		params.CLOUDFLARE_BUCKET = params.APP_NAME + "-files"
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

	lambdaName := params.APP_NAME + "-backend"
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

	lambdaName := params.APP_NAME + "-backend"
	if lambdaNro == 2 {
		lambdaName += "_2"
	}

	fmt.Println("Actulizando variables de entorno...")
	enviromentVars := map[string]any{}

	credentialsPath := GetCredentialsPath()
	fmt.Println("Leyendo variables de Lambda desde:", credentialsPath)
	jsonFileBytes, err := ReadFile(credentialsPath)
	if err != nil {
		panic("No se pudo leer el archivo de credenciales " + credentialsPath + ": " + err.Error())
	}

	err = json.Unmarshal(jsonFileBytes, &enviromentVars)
	if err != nil {
		panic("El archivo de credenciales " + credentialsPath + " posee un formato erróneo: " + err.Error())
	}

	fmt.Println("Leyendo y comprimiendo el archivo de credenciales seleccionado...")

	jsonString := string(jsonFileBytes)
	jsonBase64 := BytesToBase64(CompressZstd(&jsonString), true)

	// UpdateFunctionConfiguration reemplaza el entorno completo, no lo fusiona: toda variable
	// que define la plantilla debe repetirse aquí o se borra de la Lambda desplegada.
	variables := map[string]string{
		"APP_CODE":                  appCodeEnvValue,
		"CONFIG":                    jsonBase64,
		"LAMBDA_RESPONSE_STREAMING": lambdaResponseStreamingFlag,
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
	if params.CDN_PROVIDER == "cloudflare" {
		// R2 must be ready before the renderer receives its public asset URL.
		params.FRONTEND_CDN = DeployCloudflareInfra(params)
	}

	if params.BACKEND_PROVIDER != "aws" {
		return
	}

	CompileBackendToS3(params, true)
	CompileRendererToS3(params)

	fmt.Println("Desplegando infraestructura con CloudFormation...")
	DeployCloudFormation(params)

	// La plantilla declara el bloque Environment completo y CloudFormation lo reemplaza en vez
	// de fusionarlo, así que cada despliegue de infraestructura borra CONFIG y la Lambda entra
	// en panic al arrancar (PopulateVariables lo exige cuando APP_CODE está seteado).
	// Reinyectarlo aquí deja los secretos fuera de la plantilla, donde serían visibles en la
	// consola de CloudFormation.
	UpdateEnviromentVariables(params, 0)
	UpdateEnviromentVariables(params, 2)
	UpdateRendererEnviromentVariables(params)
}
