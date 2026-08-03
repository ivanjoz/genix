package cloud

import (
	"app/core"
	"context"
	"encoding/json"
	"fmt"
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

// InvokeWebpageRenderer ejecuta la Lambda de Node que renderiza y publica las webpages de
// una company.
//
// No reusa ExecLambda: aquel habla el protocolo `fn_exec` de Go y, fuera de Lambda,
// redirige a un HTTP contra el backend local. Aquí siempre hay que llamar a la Lambda real
// de AWS —el bundle SSR de SvelteKit solo existe allí—, tanto desde el CLI de deploy como
// desde la propia Lambda de Go.
func InvokeWebpageRenderer(request WebpageRenderRequest) (WebpageRenderResult, error) {
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
