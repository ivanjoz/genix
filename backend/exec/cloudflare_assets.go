package exec

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Native-Go replacement for `wrangler deploy` and `wrangler r2 bucket cors set`.
// Talking to the Cloudflare REST API directly removes the bun/wrangler/node
// dependency from the deploy path (and sidesteps wrangler's own token handling).
// The deploy is a single PUT of the Worker script: the storefront HTML lives in R2
// and is published per company by the render Lambda, so this no longer ships any
// tenant content. (It used to upload Workers Static Assets, whose manifest replaces
// the whole namespace — republishing one company needed every other tenant's HTML
// on disk.)

const (
	storefrontCompatibilityDate = "2026-06-10"
	storefrontWorkerMainModule  = "serve-worker.js"
	// Binding del bucket R2 con el HTML publicado. Debe coincidir con env.SITE_HTML
	// en frontend/webpage/cloudflare/serve-worker.js.
	storefrontBucketBinding = "SITE_HTML"
)

// deployStorefrontWorker PUTs the Worker script bound to the R2 bucket that holds every
// tenant's HTML. workerDirectory is <root>/frontend/webpage/cloudflare.
func deployStorefrontWorker(workerDirectory string) error {
	workerSourcePath := filepath.Join(workerDirectory, storefrontWorkerMainModule)
	workerSource, readError := os.ReadFile(workerSourcePath)
	if readError != nil {
		return fmt.Errorf("error leyendo el Worker %s: %w", workerSourcePath, readError)
	}

	metadata := map[string]any{
		"main_module":        storefrontWorkerMainModule,
		"compatibility_date": storefrontCompatibilityDate,
		"observability":      map[string]any{"enabled": true},
		"bindings": []map[string]any{
			{
				"name":        storefrontBucketBinding,
				"type":        "r2_bucket",
				"bucket_name": core.Env.CLOUDFLARE_BUCKET,
			},
		},
	}
	metadataBytes, marshalError := json.Marshal(metadata)
	if marshalError != nil {
		return marshalError
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	metadataHeader := textproto.MIMEHeader{}
	metadataHeader.Set("Content-Disposition", `form-data; name="metadata"`)
	metadataHeader.Set("Content-Type", "application/json")
	metadataPart, metadataError := writer.CreatePart(metadataHeader)
	if metadataError != nil {
		return metadataError
	}
	if _, writeError := metadataPart.Write(metadataBytes); writeError != nil {
		return writeError
	}

	moduleHeader := textproto.MIMEHeader{}
	moduleHeader.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name=%q; filename=%q`, storefrontWorkerMainModule, storefrontWorkerMainModule))
	moduleHeader.Set("Content-Type", "application/javascript+module")
	modulePart, moduleError := writer.CreatePart(moduleHeader)
	if moduleError != nil {
		return moduleError
	}
	if _, writeError := modulePart.Write(workerSource); writeError != nil {
		return writeError
	}
	if closeError := writer.Close(); closeError != nil {
		return closeError
	}

	var deployResponse cloudflareResponse[json.RawMessage]
	if requestError := cloudflareAssetRequest(
		http.MethodPut,
		"/accounts/"+url.PathEscape(core.Env.CLOUDFLARE_ACCOUNT)+"/workers/scripts/"+storefrontWorkerName,
		core.Env.CLOUDFLARE_TOKEN,
		writer.FormDataContentType(),
		body.Bytes(),
		&deployResponse,
	); requestError != nil {
		return fmt.Errorf("error desplegando el Worker: %w", requestError)
	}
	if !deployResponse.Success {
		return errors.New("Cloudflare rechazó el deploy del Worker")
	}

	fmt.Printf("[cloudflare-worker] deployed %q (bucket R2 %s)\n", storefrontWorkerName, core.Env.CLOUDFLARE_BUCKET)
	return nil
}

// ensureCompanyWebpageAssetCORS sets the R2 bucket CORS rules via the REST API,
// replacing `wrangler r2 bucket cors set`. Browser ES modules need CORS because
// the HTML and JS are served from different hostnames.
func ensureCompanyWebpageAssetCORS(projectRoot string) error {
	bucketName := core.Env.CLOUDFLARE_BUCKET
	if bucketName == "" || bucketName == "-files" {
		return fmt.Errorf("CLOUDFLARE_BUCKET o APP_NAME es requerido para configurar CORS de R2")
	}

	corsFile := filepath.Join(projectRoot, "frontend", "webpage", "cloudflare", "r2-cors.json")
	corsBytes, readError := os.ReadFile(corsFile)
	if readError != nil {
		return fmt.Errorf("error leyendo %s: %w", corsFile, readError)
	}

	fmt.Printf("[company-webpage] configuring R2 CORS bucket=%s\n", bucketName)
	var corsResponse cloudflareResponse[json.RawMessage]
	if requestError := cloudflareRequest(
		context.Background(),
		http.MethodPut,
		"/accounts/"+url.PathEscape(core.Env.CLOUDFLARE_ACCOUNT)+"/r2/buckets/"+url.PathEscape(bucketName)+"/cors",
		nil,
		json.RawMessage(corsBytes),
		&corsResponse,
	); requestError != nil {
		// A 404 here is almost always a missing bucket rather than a bad endpoint: when
		// CLOUDFLARE_BUCKET is blank the name comes from APP_NAME, so renaming the app points the
		// deploy at a bucket nobody created.
		var apiError *cloudflareAPIError
		if errors.As(requestError, &apiError) && apiError.StatusCode == http.StatusNotFound {
			nameSource := fmt.Sprintf("cloudflare.bucket=%q en config.toml", bucketName)
			if strings.TrimSpace(core.Env.CLOUDFLARE_BUCKET) == strings.TrimSpace(core.Env.APP_NAME)+"-files" {
				nameSource = fmt.Sprintf("derivado de app_name=%q; fíjelo con cloudflare.bucket en config.toml", strings.TrimSpace(core.Env.APP_NAME))
			}
			return fmt.Errorf(
				"el bucket R2 %q no existe (%s).\n"+
					"  Créelo antes de desplegar:  wrangler r2 bucket create %s\n"+
					"  o en el dashboard: Cloudflare > R2 > Create bucket.\n"+
					"  Si el bucket sí existe, revise que CLOUDFLARE_ACCOUNT sea la cuenta correcta y que CLOUDFLARE_TOKEN tenga permiso de edición sobre R2.\n"+
					"  Detalle: %w",
				bucketName, nameSource, bucketName, requestError)
		}
		return fmt.Errorf("error configurando CORS de R2 (bucket %s): %w", bucketName, requestError)
	}
	if !corsResponse.Success {
		return fmt.Errorf("Cloudflare rechazó la configuración de CORS de R2 en el bucket %s", bucketName)
	}
	return nil
}

// cloudflareAssetRequest performs a Cloudflare API call with an explicit content type,
// unlike cloudflareRequest which always sends JSON: the Worker script is uploaded as a
// multipart body (metadata part + module part).
func cloudflareAssetRequest(method, requestPath, bearer, contentType string, body []byte, target any) error {
	requestContext, cancelRequest := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancelRequest()

	request, requestError := http.NewRequestWithContext(requestContext, method, cloudflareAPIBaseURL+requestPath, bytes.NewReader(body))
	if requestError != nil {
		return requestError
	}
	request.Header.Set("Authorization", "Bearer "+bearer)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	response, responseError := http.DefaultClient.Do(request)
	if responseError != nil {
		return responseError
	}
	defer response.Body.Close()

	responseBytes, readError := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if readError != nil {
		return readError
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &cloudflareAPIError{
			StatusCode: response.StatusCode,
			Method:     method,
			Path:       maskCloudflareAccount(requestPath),
			Detail:     cloudflareErrorDetail(responseBytes),
		}
	}
	if target != nil {
		if unmarshalError := json.Unmarshal(responseBytes, target); unmarshalError != nil {
			return fmt.Errorf("error parseando respuesta de Cloudflare: %w", unmarshalError)
		}
	}
	return nil
}
