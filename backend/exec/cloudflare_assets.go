package exec

import (
	"app/core"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Native-Go replacement for `wrangler deploy` and `wrangler r2 bucket cors set`.
// Talking to the Cloudflare REST API directly removes the bun/wrangler/node
// dependency from the deploy path (and sidesteps wrangler's own token handling).
// Uploading Workers Static Assets is the documented three-step flow: register a
// content-hashed manifest, upload the buckets Cloudflare is missing, then PUT the
// Worker script with the returned assets completion token. The Worker code itself
// never changes — it is the fixed serve-worker.js shipped in the repo — so each
// run effectively just swaps the asset bundle.

const (
	storefrontCompatibilityDate = "2026-06-10"
	storefrontWorkerMainModule  = "serve-worker.js"
)

type assetUploadFile struct {
	hash        string
	localPath   string
	contentType string
}

type assetManifestEntry struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type assetUploadSession struct {
	JWT     string     `json:"jwt"`
	Buckets [][]string `json:"buckets"`
}

// deployStorefrontWorker uploads every tenant's static assets and PUTs the Worker
// script, replacing `wrangler deploy`. workerDirectory is <root>/frontend/webpage/cloudflare.
func deployStorefrontWorker(workerDirectory string) error {
	webpagesDirectory := filepath.Join(workerDirectory, "webpages")
	workerSourcePath := filepath.Join(workerDirectory, storefrontWorkerMainModule)

	assets, manifest, manifestError := buildStorefrontAssetManifest(webpagesDirectory)
	if manifestError != nil {
		return manifestError
	}
	if len(assets) == 0 {
		return fmt.Errorf("no hay assets en %s para desplegar", webpagesDirectory)
	}
	fmt.Printf("[cloudflare-worker] manifest files=%d\n", len(assets))

	completionToken, uploadError := uploadStorefrontAssets(assets, manifest)
	if uploadError != nil {
		return uploadError
	}

	workerSource, readError := os.ReadFile(workerSourcePath)
	if readError != nil {
		return fmt.Errorf("error leyendo el Worker %s: %w", workerSourcePath, readError)
	}

	if putError := putStorefrontWorker(completionToken, workerSource); putError != nil {
		return putError
	}
	fmt.Printf("[cloudflare-worker] uploaded %d asset(s) and deployed %q\n", len(assets), storefrontWorkerName)
	return nil
}

// buildStorefrontAssetManifest walks the webpages tree honoring .assetsignore (the
// same gitignore-style rules wrangler applies) and returns per-file metadata plus
// the manifest posted to the upload-session endpoint.
func buildStorefrontAssetManifest(webpagesDirectory string) ([]assetUploadFile, map[string]assetManifestEntry, error) {
	ignorePatterns, ignoreError := readStorefrontAssetsIgnore(filepath.Join(webpagesDirectory, ".assetsignore"))
	if ignoreError != nil {
		return nil, nil, ignoreError
	}

	assets := []assetUploadFile{}
	manifest := map[string]assetManifestEntry{}
	walkError := filepath.WalkDir(webpagesDirectory, func(currentPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		relativePath, relError := filepath.Rel(webpagesDirectory, currentPath)
		if relError != nil {
			return relError
		}
		relativePath = filepath.ToSlash(relativePath)
		if isStorefrontAssetIgnored(relativePath, ignorePatterns) {
			return nil
		}

		content, readError := os.ReadFile(currentPath)
		if readError != nil {
			return fmt.Errorf("error leyendo asset %s: %w", relativePath, readError)
		}

		extension := strings.TrimPrefix(filepath.Ext(currentPath), ".")
		contentType := mime.TypeByExtension(filepath.Ext(currentPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		hash := storefrontAssetHash(content, extension)
		manifest["/"+relativePath] = assetManifestEntry{Hash: hash, Size: int64(len(content))}
		assets = append(assets, assetUploadFile{
			hash:        hash,
			localPath:   currentPath,
			contentType: contentType,
		})
		return nil
	})
	if walkError != nil {
		return nil, nil, fmt.Errorf("error recorriendo %s: %w", webpagesDirectory, walkError)
	}
	return assets, manifest, nil
}

// storefrontAssetHash matches wrangler: first 32 hex chars of
// SHA-256(base64(content) + extensionWithoutDot).
func storefrontAssetHash(content []byte, extension string) string {
	encoded := base64.StdEncoding.EncodeToString(content)
	digest := sha256.Sum256([]byte(encoded + extension))
	return hex.EncodeToString(digest[:])[:32]
}

func readStorefrontAssetsIgnore(ignorePath string) ([]string, error) {
	// wrangler always skips the .assetsignore file itself.
	patterns := []string{".assetsignore"}
	content, readError := os.ReadFile(ignorePath)
	if readError != nil {
		if os.IsNotExist(readError) {
			return patterns, nil
		}
		return nil, fmt.Errorf("error leyendo .assetsignore: %w", readError)
	}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		patterns = append(patterns, trimmed)
	}
	return patterns, nil
}

func isStorefrontAssetIgnored(relativePath string, patterns []string) bool {
	base := path.Base(relativePath)
	for _, pattern := range patterns {
		pattern = strings.TrimPrefix(pattern, "/")
		if pattern == relativePath || pattern == base {
			return true
		}
		if matched, _ := path.Match(pattern, relativePath); matched {
			return true
		}
		if matched, _ := path.Match(pattern, base); matched {
			return true
		}
	}
	return false
}

// uploadStorefrontAssets registers the manifest, uploads each requested bucket, and
// returns the assets completion token used by the Worker deploy.
func uploadStorefrontAssets(assets []assetUploadFile, manifest map[string]assetManifestEntry) (string, error) {
	manifestBody, marshalError := json.Marshal(map[string]any{"manifest": manifest})
	if marshalError != nil {
		return "", marshalError
	}

	var session cloudflareResponse[assetUploadSession]
	if requestError := cloudflareAssetRequest(
		http.MethodPost,
		"/accounts/"+url.PathEscape(core.Env.CLOUDFLARE_ACCOUNT)+"/workers/scripts/"+storefrontWorkerName+"/assets-upload-session",
		core.Env.CLOUDFLARE_TOKEN,
		"application/json",
		manifestBody,
		&session,
	); requestError != nil {
		return "", fmt.Errorf("error iniciando upload-session de assets: %w", requestError)
	}
	if !session.Success {
		return "", errors.New("Cloudflare rechazó el upload-session de assets")
	}

	// No buckets means Cloudflare already has every file; the session JWT doubles
	// as the deploy completion token.
	if len(session.Result.Buckets) == 0 {
		fmt.Println("[cloudflare-worker] all assets already present (no upload needed)")
		return session.Result.JWT, nil
	}

	assetsByHash := make(map[string]assetUploadFile, len(assets))
	for _, asset := range assets {
		assetsByHash[asset.hash] = asset
	}

	completionToken := ""
	for bucketIndex, bucket := range session.Result.Buckets {
		token, uploadError := uploadStorefrontBucket(session.Result.JWT, bucket, assetsByHash)
		if uploadError != nil {
			return "", fmt.Errorf("error subiendo bucket %d: %w", bucketIndex+1, uploadError)
		}
		if token != "" {
			completionToken = token
		}
		fmt.Printf("[cloudflare-worker] uploaded bucket %d/%d (%d file(s))\n",
			bucketIndex+1, len(session.Result.Buckets), len(bucket))
	}
	if completionToken == "" {
		return "", errors.New("Cloudflare no devolvió el token de finalización de assets")
	}
	return completionToken, nil
}

func uploadStorefrontBucket(sessionJWT string, bucket []string, assetsByHash map[string]assetUploadFile) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, hash := range bucket {
		asset, exists := assetsByHash[hash]
		if !exists {
			return "", fmt.Errorf("Cloudflare pidió un hash desconocido: %s", hash)
		}
		content, readError := os.ReadFile(asset.localPath)
		if readError != nil {
			return "", fmt.Errorf("error leyendo %s: %w", asset.localPath, readError)
		}

		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q`, hash))
		header.Set("Content-Type", asset.contentType)
		part, partError := writer.CreatePart(header)
		if partError != nil {
			return "", partError
		}
		if _, writeError := part.Write([]byte(base64.StdEncoding.EncodeToString(content))); writeError != nil {
			return "", writeError
		}
	}
	if closeError := writer.Close(); closeError != nil {
		return "", closeError
	}

	var uploadResponse struct {
		Result struct {
			JWT string `json:"jwt"`
		} `json:"result"`
		JWT     string `json:"jwt"`
		Success bool   `json:"success"`
	}
	// The upload is authenticated with the session JWT, not the account token.
	if requestError := cloudflareAssetRequest(
		http.MethodPost,
		"/accounts/"+url.PathEscape(core.Env.CLOUDFLARE_ACCOUNT)+"/workers/assets/upload?base64=true",
		sessionJWT,
		writer.FormDataContentType(),
		body.Bytes(),
		&uploadResponse,
	); requestError != nil {
		return "", requestError
	}
	if uploadResponse.Result.JWT != "" {
		return uploadResponse.Result.JWT, nil
	}
	return uploadResponse.JWT, nil
}

func putStorefrontWorker(assetsToken string, workerSource []byte) error {
	metadata := map[string]any{
		"main_module":        storefrontWorkerMainModule,
		"compatibility_date": storefrontCompatibilityDate,
		"assets": map[string]any{
			"jwt": assetsToken,
			"config": map[string]any{
				"html_handling":      "none",
				"not_found_handling": "none",
				"run_worker_first":   true,
			},
		},
		"observability": map[string]any{"enabled": true},
		"bindings": []map[string]any{
			{"name": "ASSETS", "type": "assets"},
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
	return nil
}

// ensureCompanyWebpageAssetCORS sets the R2 bucket CORS rules via the REST API,
// replacing `wrangler r2 bucket cors set`. Browser ES modules need CORS because
// the HTML and JS are served from different hostnames.
func ensureCompanyWebpageAssetCORS(projectRoot string) error {
	bucketName := strings.TrimSpace(core.Env.STACK_NAME) + "-files"
	if bucketName == "-files" {
		return fmt.Errorf("STACK_NAME es requerido para configurar CORS de R2")
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
		return fmt.Errorf("error configurando CORS de R2: %w", requestError)
	}
	if !corsResponse.Success {
		return errors.New("Cloudflare rechazó la configuración de CORS de R2")
	}
	return nil
}

// cloudflareAssetRequest performs a Cloudflare API call with an explicit bearer
// token and content type (multipart uploads use the session JWT, not the account
// token), unlike cloudflareRequest which always sends JSON with the account token.
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
		return fmt.Errorf("Cloudflare API HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBytes)))
	}
	if target != nil {
		if unmarshalError := json.Unmarshal(responseBytes, target); unmarshalError != nil {
			return fmt.Errorf("error parseando respuesta de Cloudflare: %w", unmarshalError)
		}
	}
	return nil
}
