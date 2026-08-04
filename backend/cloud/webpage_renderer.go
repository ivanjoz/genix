package cloud

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// WebpageRenderPage es una página a renderizar: su PageID (el que identifica el contenido
// en el snapshot del CDN) y la ruta pública bajo la que se publica el HTML.
type WebpageRenderPage struct {
	ID   int16  `json:"id"`
	Path string `json:"path"`
}

type WebpageRenderRequest struct {
	CompanyID int32               `json:"companyID"`
	Hostname  string              `json:"hostname"`
	Pages     []WebpageRenderPage `json:"pages"`
	// ForceAssets vuelve a subir los js/css aunque la company ya tenga publicada esa
	// versión del renderer (útil si se borraron del CDN a mano).
	ForceAssets bool `json:"forceAssets,omitempty"`
}

type WebpageRenderResult struct {
	BuildID string `json:"buildId"`
	Pages   int    `json:"pages"`
	Assets  int    `json:"assets"`
	Bytes   int    `json:"bytes"`
}

// InvokeWebpageRenderer renderiza y publica las webpages de una company. El renderer es el
// mismo código de Node en los dos caminos (webpage-renderer/handler.mjs): en serverless se
// invoca la Lambda, y fuera de ella se ejecuta el archivo desde el repo. Los llamadores no
// distinguen los dos casos.
//
// No reusa ExecLambda: aquel habla el protocolo `fn_exec` de Go y, fuera de Lambda, redirige a
// un HTTP contra el backend local, que no es donde vive el renderer.
func InvokeWebpageRenderer(request WebpageRenderRequest) (WebpageRenderResult, error) {
	if core.Env.IS_SERVERLESS {
		return invokeWebpageRendererLambda(request)
	}
	return runWebpageRendererLocally(request)
}

// invokeWebpageRendererLambda llama a la Lambda de Node desplegada en AWS.
func invokeWebpageRendererLambda(request WebpageRenderRequest) (WebpageRenderResult, error) {
	result := WebpageRenderResult{}

	functionName := core.Env.APP_NAME + "-webpage-renderer"
	payload, marshalError := json.Marshal(request)
	if marshalError != nil {
		return result, marshalError
	}

	core.Log("Invocando renderer::", functionName, "| company:", request.CompanyID,
		"| host:", request.Hostname, "| páginas:", len(request.Pages))

	startedAt := time.Now()
	client := lambda.NewFromConfig(core.GetAwsConfig())
	response, invokeError := client.Invoke(context.TODO(), &lambda.InvokeInput{
		FunctionName: &functionName,
		Payload:      payload,
	})
	if invokeError != nil {
		return result, fmt.Errorf("error invocando %s: %w", functionName, invokeError)
	}

	// Un fallo dentro del handler llega como HTTP 200 con FunctionError: hay que mirarlo
	// explícitamente o un render roto pasaría por bueno.
	if response.FunctionError != nil {
		return result, fmt.Errorf("el renderer falló (%s): %s",
			*response.FunctionError, core.StrCut(string(response.Payload), 800))
	}
	if unmarshalError := json.Unmarshal(response.Payload, &result); unmarshalError != nil {
		return result, fmt.Errorf("respuesta ilegible del renderer: %s",
			core.StrCut(string(response.Payload), 800))
	}

	core.Log("Renderer OK::", "build:", result.BuildID, "| páginas:", result.Pages,
		"| assets:", result.Assets, "| bytes:", result.Bytes,
		"|", fmt.Sprintf("%.1fs", time.Since(startedAt).Seconds()))
	return result, nil
}

// runWebpageRendererLocally ejecuta el renderer con `node webpage-renderer/cli.mjs`, para el VPS
// y el desarrollo local, donde no hay ninguna Lambda que invocar.
//
// El contrato con cli.mjs es por descriptores estándar: el evento entra por stdin, el resultado
// JSON sale por stdout y los logs por stderr. Esa separación es la que permite deserializar la
// respuesta sin filtrar líneas, porque el handler loguea una por página.
func runWebpageRendererLocally(request WebpageRenderRequest) (WebpageRenderResult, error) {
	result := WebpageRenderResult{}

	projectRoot, rootError := core.FindProjectRoot()
	if rootError != nil {
		return result, fmt.Errorf("no se pudo ubicar el renderer en disco: %w", rootError)
	}
	rendererPath := filepath.Join(projectRoot, "webpage-renderer", "cli.mjs")
	if _, statError := os.Stat(rendererPath); statError != nil {
		return result, fmt.Errorf("no se encontró el renderer en %s: %w", rendererPath, statError)
	}

	payload, marshalError := json.Marshal(request)
	if marshalError != nil {
		return result, marshalError
	}

	for variableName, variableValue := range map[string]string{
		"FRONTEND_CDN":       core.Env.FRONTEND_CDN,
		"CLOUDFLARE_ACCOUNT": core.Env.CLOUDFLARE_ACCOUNT,
		"CLOUDFLARE_TOKEN":   core.Env.CLOUDFLARE_TOKEN,
		"CLOUDFLARE_BUCKET":  core.Env.CLOUDFLARE_BUCKET,
	} {
		// El handler valida lo mismo y aborta, pero desde aquí el mensaje dice qué falta en
		// credentials.json en vez de aparecer como un fallo del proceso de Node.
		if strings.TrimSpace(variableValue) == "" {
			return result, fmt.Errorf("%s es requerido para ejecutar el renderer en local", variableName)
		}
	}

	core.Log("Ejecutando renderer local::", rendererPath, "| company:", request.CompanyID,
		"| host:", request.Hostname, "| páginas:", len(request.Pages))

	startedAt := time.Now()
	// El timeout es el de la Lambda en cloud/template.yml: un render colgado no debe dejar
	// bloqueado el proceso que lo lanzó.
	commandContext, cancelCommand := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancelCommand()

	command := exec.CommandContext(commandContext, "node", rendererPath)
	command.Stdin = bytes.NewReader(payload)
	// Se hereda el entorno y se añaden las variables encima: node necesita PATH y HOME, y una
	// lista blanca los dejaría fuera.
	command.Env = append(os.Environ(),
		"RENDERER_ZIP_URL="+core.Env.WEBPAGE_RENDERER_URL,
		"FRONTEND_CDN="+core.Env.FRONTEND_CDN,
		"CLOUDFLARE_ACCOUNT="+core.Env.CLOUDFLARE_ACCOUNT,
		"CLOUDFLARE_TOKEN="+core.Env.CLOUDFLARE_TOKEN,
		"CLOUDFLARE_BUCKET="+core.Env.CLOUDFLARE_BUCKET,
	)

	standardOutput := bytes.Buffer{}
	standardError := bytes.Buffer{}
	command.Stdout = &standardOutput
	command.Stderr = &standardError

	runError := command.Run()
	// Los logs del renderer salen siempre, tanto si funcionó como si no: son el único rastro de
	// lo que hizo, y en un fallo son el mensaje de error.
	if rendererLogs := strings.TrimSpace(standardError.String()); rendererLogs != "" {
		core.Log("Renderer local::\n" + rendererLogs)
	}
	if runError != nil {
		if commandContext.Err() != nil {
			return result, fmt.Errorf("el renderer local excedió el tiempo límite: %w", commandContext.Err())
		}
		return result, fmt.Errorf("el renderer local falló: %w", runError)
	}

	if unmarshalError := json.Unmarshal(standardOutput.Bytes(), &result); unmarshalError != nil {
		return result, fmt.Errorf("respuesta ilegible del renderer local: %s",
			core.StrCut(standardOutput.String(), 800))
	}

	core.Log("Renderer local OK::", "build:", result.BuildID, "| páginas:", result.Pages,
		"| assets:", result.Assets, "| bytes:", result.Bytes,
		"|", fmt.Sprintf("%.1fs", time.Since(startedAt).Seconds()))
	return result, nil
}
