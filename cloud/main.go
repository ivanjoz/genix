package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pelletier/go-toml/v2"
)

// DeployParams refleja config.toml por secciones; el resto de cloud/ consume los campos
// anidados directamente (params.AWS.Region, params.Cloudflare.Bucket, …), sin la capa
// plana que sí conserva backend/core/security.go.
type DeployParams struct {
	// AppName prefija todo nombre físico: stack, Lambdas, tabla DynamoDB, buckets.
	AppName string `toml:"app_name"`
	// S3CompiledPath y FrontendBucket son derivados en main(): el primero es una ruta fija
	// del binario, el segundo cae a "<app_name>-frontend" cuando aws.frontend_bucket viene
	// vacío. Ninguno de los dos lleva tag de archivo.
	S3CompiledPath string
	FrontendBucket string

	Providers struct {
		// Backend selecciona el espejo de datos y la infraestructura AWS del backend.
		Backend string `toml:"backend"`
		// CDN selecciona el object storage y el origen público del frontend.
		CDN string `toml:"cdn"`
	} `toml:"providers"`

	AWS struct {
		Profile          string `toml:"profile"`
		Region           string `toml:"region"`
		DeploymentBucket string `toml:"deployment_bucket"`
		LambdaIAMRole    string `toml:"lambda_iam_role"`
		FrontendBucket   string `toml:"frontend_bucket"`
	} `toml:"aws"`

	Cloudflare struct {
		Account string `toml:"account"`
		Token   string `toml:"token"`
		// Bucket fija el bucket R2 cuando su nombre no sigue el patrón "<app_name>-files":
		// renombrar la app no debe apuntar el deploy a un bucket vacío. Vacío = nombre autogenerado.
		Bucket string `toml:"bucket"`
	} `toml:"cloudflare"`

	Frontend struct {
		// CDNURL es el origen público del CDN. La Lambda de render lo necesita para construir
		// la base de assets de cada company (<CDNURL>/websites/<companyID>).
		CDNURL string `toml:"cdn_url"`
		// WebpageRendererURL es la URL del artefacto webpage-renderer.zip que publica CI.
		// Vacío = el valor por defecto de cloud/webpage-renderer.go.
		WebpageRendererURL string `toml:"webpage_renderer_url"`
	} `toml:"frontend"`
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

// GetConfigPath keeps every cloud read and write on the environment selected by deploy.sh.
func GetConfigPath() string {
	configuredConfigPath := strings.TrimSpace(os.Getenv("GENIX_CONFIG_FILE"))
	if configuredConfigPath != "" {
		return configuredConfigPath
	}
	return GetBaseWD() + "/config.toml"
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

	configPath := GetConfigPath()
	fmt.Println("Leyendo configuración desde:", configPath)
	configToml, err := ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	params := DeployParams{S3CompiledPath: s3CompiledPath}
	err = toml.Unmarshal(configToml, &params)

	if err != nil {
		panic(fmt.Sprintf("Error parsing config file %s: %v", configPath, err))
	}

	if params.Providers.Backend != "aws" && params.Providers.Backend != "cloudflare" && params.Providers.Backend != "none" {
		panic("providers.backend debe ser 'aws', 'cloudflare' o 'none'.")
	}
	if params.Providers.CDN != "aws" && params.Providers.CDN != "cloudflare" {
		panic("providers.cdn debe ser 'aws' o 'cloudflare'.")
	}
	if w1 != "3" || params.Providers.Backend == "aws" {
		missingParams := []string{params.AppName, params.AWS.DeploymentBucket,
			params.AWS.Profile, params.AWS.Region, params.AWS.LambdaIAMRole}

		for _, e := range missingParams {
			if len(e) == 0 {
				panic("Los parámetros app_name, aws.deployment_bucket, aws.region, aws.profile y aws.lambda_iam_role son requeridos.")
			}
		}
	}
	if params.Providers.CDN == "cloudflare" && (strings.TrimSpace(params.Cloudflare.Account) == "" || strings.TrimSpace(params.Cloudflare.Token) == "") {
		panic("cloudflare.account y cloudflare.token son requeridos cuando providers.cdn es 'cloudflare'.")
	}

	params.FrontendBucket = strings.TrimSpace(params.AWS.FrontendBucket)
	if len(params.FrontendBucket) == 0 {
		params.FrontendBucket = params.AppName + "-frontend"
	}

	params.Cloudflare.Bucket = strings.TrimSpace(params.Cloudflare.Bucket)
	if len(params.Cloudflare.Bucket) == 0 {
		params.Cloudflare.Bucket = params.AppName + "-files"
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
		awsConfig, _ := MakeAwsConfig(params.AWS.Profile, params.AWS.Region)
		s3Client := s3.NewFromConfig(awsConfig)

		s3Args := FileToS3Args{
			Bucket:        params.AWS.DeploymentBucket,
			LocalFilePath: compiledZipPath,
			FilePath:      params.S3CompiledPath,
		}

		fmt.Println("Enviando compilado a S3: ", s3Args.Bucket+" | "+s3Args.FilePath)
		SendFileToS3Client(s3Args, s3Client)
	}
}

// Despliega el código compilado de la Lambda
func DeployLambda(params DeployParams, lambdaNro uint8) {

	lambdaName := params.AppName + "-backend"
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

	awsConfig, _ := MakeAwsConfig(params.AWS.Profile, params.AWS.Region)
	client := lambda.NewFromConfig(awsConfig)

	_, err = client.UpdateFunctionCode(context.TODO(), &deployInput)

	if err != nil {
		panic("Error al actualizar el código de la Lambda: " + err.Error())
	}
	fmt.Println("Código de la Lambda actualizado!")
}

// Actualiza las variables de entorno de la Lambda
func UpdateEnviromentVariables(params DeployParams, lambdaNro uint8) {

	lambdaName := params.AppName + "-backend"
	if lambdaNro == 2 {
		lambdaName += "_2"
	}

	fmt.Println("Actulizando variables de entorno...")
	configValues := map[string]any{}

	configPath := GetConfigPath()
	fmt.Println("Leyendo variables de Lambda desde:", configPath)
	configFileBytes, err := ReadFile(configPath)
	if err != nil {
		panic("No se pudo leer el archivo de configuración " + configPath + ": " + err.Error())
	}

	err = toml.Unmarshal(configFileBytes, &configValues)
	if err != nil {
		panic("El archivo de configuración " + configPath + " posee un formato erróneo: " + err.Error())
	}

	fmt.Println("Leyendo y comprimiendo el archivo de configuración seleccionado...")

	configText := string(configFileBytes)
	configBase64 := BytesToBase64(CompressZstd(&configText), true)

	// UpdateFunctionConfiguration reemplaza el entorno completo, no lo fusiona: toda variable
	// que define la plantilla debe repetirse aquí o se borra de la Lambda desplegada.
	variables := map[string]string{
		"APP_CODE":                  appCodeEnvValue,
		"CONFIG":                    configBase64,
		"LAMBDA_RESPONSE_STREAMING": lambdaResponseStreamingFlag,
	}

	configInput := lambda.UpdateFunctionConfigurationInput{
		FunctionName: &lambdaName,
		Environment: &lambdaTypes.Environment{
			Variables: variables,
		},
	}

	awsConfig, _ := MakeAwsConfig(params.AWS.Profile, params.AWS.Region)
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
	if params.Providers.CDN == "cloudflare" {
		// R2 must be ready before the renderer receives its public asset URL.
		params.Frontend.CDNURL = DeployCloudflareInfra(params)
	}

	if params.Providers.Backend != "aws" {
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
