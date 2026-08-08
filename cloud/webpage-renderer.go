package main

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Código de la Lambda de render (Node). Es un único archivo sin dependencias npm, así que
// el .zip que sube a S3 lleva solo eso: el bundle SSR y los assets de la tienda los
// descarga la función en caliente desde RENDERER_ZIP_URL.
//
// La carpeta webpage-renderer/ del repo tiene además cli.mjs (la entrada que usa el backend
// fuera de Lambda) y sus tests, que NO viajan en el zip: aquí se empaqueta un solo archivo.
//
// Ojo con los nombres: este es el CÓDIGO de la función; webpage-renderer.zip (sin el
// sufijo -lambda) es el ARTEFACTO que publica CI con el servidor SSR y los assets.
const rendererHandlerPath = "/webpage-renderer/handler.mjs"
const rendererS3Path = "gerp-artifacts/webpage-renderer-lambda.zip"
const rendererLocalZipPath = "/cloud/webpage-renderer-lambda.zip"

// URL por defecto del artefacto de CI (GitHub Pages, siempre la última versión). Se puede
// sobrescribir con frontend.webpage_renderer_url en config.toml.
//
// Duplicada en backend/core/security.go (DefaultWebpageRendererURL) porque el backend también
// necesita el artefacto cuando ejecuta el renderer en local, y este directorio es otro módulo
// Go: el compilador no puede vigilar que las dos sigan iguales.
const defaultRendererZipUrl = "https://genix-dev.un.pe/webpage-renderer.zip"

func rendererZipUrl(params DeployParams) string {
	if url := strings.TrimSpace(params.Frontend.WebpageRendererURL); url != "" {
		return url
	}
	return defaultRendererZipUrl
}

// Empaqueta el handler de Node y lo sube al bucket de despliegues, desde donde
// CloudFormation lo toma como código de la función.
func CompileRendererToS3(params DeployParams) {
	handlerBytes, err := ReadFile(GetBaseWD() + rendererHandlerPath)
	if err != nil {
		panic("Error al leer el handler del renderer: " + err.Error())
	}

	zipPath := GetBaseWD() + rendererLocalZipPath
	zipFile, err := os.Create(zipPath)
	if err != nil {
		panic("Error al crear el zip del renderer: " + err.Error())
	}

	zipWriter := zip.NewWriter(zipFile)
	// El nombre dentro del zip fija el Handler de la plantilla: "handler.render".
	entry, err := zipWriter.Create("handler.mjs")
	if err == nil {
		_, err = entry.Write(handlerBytes)
	}
	if err != nil {
		panic("Error al comprimir el handler del renderer: " + err.Error())
	}
	if err := zipWriter.Close(); err != nil {
		panic("Error al cerrar el zip del renderer: " + err.Error())
	}
	if err := zipFile.Close(); err != nil {
		panic("Error al cerrar el archivo zip del renderer: " + err.Error())
	}

	awsConfig, _ := MakeAwsConfig(params.AWS.Profile, params.AWS.Region)
	fmt.Println("Enviando handler del renderer a S3: ", params.AWS.DeploymentBucket+" | "+rendererS3Path)
	SendFileToS3Client(FileToS3Args{
		Bucket:        params.AWS.DeploymentBucket,
		LocalFilePath: zipPath,
		FilePath:      rendererS3Path,
	}, s3.NewFromConfig(awsConfig))
}

// Reinyecta las variables de la Lambda de render tras el despliegue del stack: la
// plantilla declara el bloque Environment completo y CloudFormation lo reemplaza en vez
// de fusionarlo, así que las credenciales de Cloudflare tienen que escribirse aquí para
// no quedar expuestas en la consola de CloudFormation.
//
// Se pasan sueltas (no el CONFIG zstd+base64 del backend) para no obligar al handler de
// Node a descomprimir zstd.
func UpdateRendererEnviromentVariables(params DeployParams) {
	lambdaName := params.AppName + "-webpage-renderer"

	if strings.TrimSpace(params.Frontend.CDNURL) == "" {
		panic("frontend.cdn_url es requerido en config.toml para la Lambda de render")
	}

	variables := map[string]string{
		"RENDERER_ZIP_URL":   rendererZipUrl(params),
		"FRONTEND_CDN":       params.Frontend.CDNURL,
		"CLOUDFLARE_ACCOUNT": params.Cloudflare.Account,
		"CLOUDFLARE_TOKEN":   params.Cloudflare.Token,
		"CLOUDFLARE_BUCKET":  params.Cloudflare.Bucket,
	}

	awsConfig, _ := MakeAwsConfig(params.AWS.Profile, params.AWS.Region)
	client := lambda.NewFromConfig(awsConfig)

	fmt.Println("Actualizando variables de la Lambda de render...")
	_, err := client.UpdateFunctionConfiguration(context.TODO(), &lambda.UpdateFunctionConfigurationInput{
		FunctionName: &lambdaName,
		Environment:  &lambdaTypes.Environment{Variables: variables},
	})
	if err != nil {
		fmt.Println("Error al actualizar las variables de la Lambda de render. ", err)
		return
	}
	fmt.Println("Variables de la Lambda de render actualizadas!")
}
