package exec

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	storefrontWorkerName = "genix-storefront"
	cloudflareAPIBaseURL = "https://api.cloudflare.com/client/v4"
)

var hostnameLabelPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type cloudflareResponse[T any] struct {
	Result   T     `json:"result"`
	Success  bool  `json:"success"`
	Errors   []any `json:"errors"`
	Messages []any `json:"messages"`
}

type cloudflareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cloudflareWorkerDomain struct {
	ID          string `json:"id"`
	Environment string `json:"environment"`
	Hostname    string `json:"hostname"`
	Service     string `json:"service"`
	ZoneID      string `json:"zone_id"`
	ZoneName    string `json:"zone_name"`
}

func DeployCloudflareWorkerHandler(_ *core.ExecArgs) core.FuncResponse {
	tenantCount, deployError := DeployCloudflareWorker()
	if deployError != nil {
		return core.FuncResponse{Error: deployError.Error()}
	}

	return core.FuncResponse{
		Message: fmt.Sprintf("Cloudflare Worker desplegado con %d sitio(s)", tenantCount),
	}
}

func DeployCloudflareWorker() (int, error) {
	projectRoot, rootError := findGenixProjectRoot()
	if rootError != nil {
		return 0, rootError
	}

	workerDirectory := filepath.Join(projectRoot, "frontend", "webpage", "cloudflare")
	webpagesDirectory := filepath.Join(workerDirectory, "webpages")
	tenantCount, validationError := validateWebpageArtifacts(webpagesDirectory)
	if validationError != nil {
		return 0, validationError
	}

	if strings.TrimSpace(core.Env.CLOUDFLARE_ACCOUNT) == "" ||
		strings.TrimSpace(core.Env.CLOUDFLARE_TOKEN) == "" {
		return 0, errors.New("CLOUDFLARE_ACCOUNT y CLOUDFLARE_TOKEN son requeridos")
	}

	fmt.Printf("[cloudflare-worker] dir=%s tenants=%d\n", workerDirectory, tenantCount)
	fmt.Println("[cloudflare-worker] deploying Worker and Static Assets")

	if deployError := deployStorefrontWorker(workerDirectory); deployError != nil {
		return 0, fmt.Errorf("error desplegando Cloudflare Worker: %w", deployError)
	}

	fmt.Println("[cloudflare-worker] deployment completed")
	return tenantCount, nil
}

func validateWebpageArtifacts(webpagesDirectory string) (int, error) {
	if createError := os.MkdirAll(webpagesDirectory, 0o755); createError != nil {
		return 0, fmt.Errorf("error creando directorio webpages: %w", createError)
	}

	entries, readError := os.ReadDir(webpagesDirectory)
	if readError != nil {
		return 0, fmt.Errorf("error leyendo directorio webpages: %w", readError)
	}

	normalizedHostnames := map[string]bool{}
	tenantCount := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		entryInfo, infoError := entry.Info()
		if infoError != nil {
			return 0, fmt.Errorf("error leyendo %s: %w", entry.Name(), infoError)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 || !entry.IsDir() {
			return 0, fmt.Errorf("webpages solo admite directorios de dominio: %s", entry.Name())
		}

		hostname, hostnameError := normalizeAndValidateHostname(entry.Name())
		if hostnameError != nil || hostname != entry.Name() {
			return 0, fmt.Errorf("directorio de dominio inválido %q", entry.Name())
		}
		if normalizedHostnames[hostname] {
			return 0, fmt.Errorf("dominio duplicado en webpages: %s", hostname)
		}
		normalizedHostnames[hostname] = true

		indexPath := filepath.Join(webpagesDirectory, hostname, "index.html")
		indexInfo, indexError := os.Stat(indexPath)
		if indexError != nil || indexInfo.IsDir() {
			return 0, fmt.Errorf("falta index.html para %s", hostname)
		}
		tenantCount++
	}

	return tenantCount, nil
}

func normalizeAndValidateHostname(rawHostname string) (string, error) {
	hostname := strings.ToLower(strings.TrimSpace(rawHostname))
	hostname = strings.TrimPrefix(hostname, "https://")
	hostname = strings.TrimPrefix(hostname, "http://")
	hostname = strings.TrimSuffix(strings.Split(hostname, "/")[0], ".")

	if len(hostname) < 4 || len(hostname) > 253 || !strings.Contains(hostname, ".") {
		return "", fmt.Errorf("dominio inválido: %s", rawHostname)
	}

	for _, label := range strings.Split(hostname, ".") {
		if !hostnameLabelPattern.MatchString(label) {
			return "", fmt.Errorf("dominio inválido: %s", rawHostname)
		}
	}

	return hostname, nil
}

func findGenixProjectRoot() (string, error) {
	currentDirectory, workingDirectoryError := os.Getwd()
	if workingDirectoryError != nil {
		return "", workingDirectoryError
	}

	for {
		deployScript := filepath.Join(currentDirectory, "deploy.sh")
		prerenderScript := filepath.Join(currentDirectory, "scripts", "prerender.mjs")
		if fileExists(deployScript) && fileExists(prerenderScript) {
			return currentDirectory, nil
		}

		parentDirectory := filepath.Dir(currentDirectory)
		if parentDirectory == currentDirectory {
			return "", errors.New("no se encontró la raíz del proyecto Genix")
		}
		currentDirectory = parentDirectory
	}
}

func fileExists(filePath string) bool {
	fileInfo, fileError := os.Stat(filePath)
	return fileError == nil && !fileInfo.IsDir()
}

func provisionStorefrontDomain(hostname string) error {
	zoneName := strings.ToLower(strings.TrimSpace(core.Env.ZONE_NAME))
	if zoneName == "" {
		zoneName = "un.pe"
	}
	if hostname == zoneName || !strings.HasSuffix(hostname, "."+zoneName) {
		return fmt.Errorf("el dominio %s no pertenece a la zona %s", hostname, zoneName)
	}

	zone, zoneError := findCloudflareZone(zoneName)
	if zoneError != nil {
		return zoneError
	}

	existingDomain, domainError := findCloudflareWorkerDomain(hostname)
	if domainError != nil {
		return domainError
	}
	if existingDomain != nil {
		if existingDomain.Service != storefrontWorkerName {
			return fmt.Errorf(
				"el dominio %s ya está asignado al Worker %s",
				hostname,
				existingDomain.Service,
			)
		}
		fmt.Printf("[company-webpage] domain already provisioned: %s\n", hostname)
		return nil
	}

	payload := map[string]string{
		"environment": "production",
		"hostname":    hostname,
		"service":     storefrontWorkerName,
		"zone_id":     zone.ID,
	}
	var createdDomain cloudflareResponse[cloudflareWorkerDomain]
	if requestError := cloudflareRequest(
		context.Background(),
		http.MethodPut,
		"/accounts/"+url.PathEscape(core.Env.CLOUDFLARE_ACCOUNT)+"/workers/domains",
		nil,
		payload,
		&createdDomain,
	); requestError != nil {
		return fmt.Errorf("error creando Worker Custom Domain: %w", requestError)
	}
	if !createdDomain.Success {
		return errors.New("Cloudflare rechazó la creación del Worker Custom Domain")
	}

	fmt.Printf("[company-webpage] domain association created: %s\n", hostname)
	for attempt := 1; attempt <= 10; attempt++ {
		associatedDomain, pollError := findCloudflareWorkerDomain(hostname)
		if pollError == nil && associatedDomain != nil && associatedDomain.Service == storefrontWorkerName {
			fmt.Printf("[company-webpage] domain active: %s\n", hostname)
			return nil
		}
		if attempt < 10 {
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("timeout esperando la activación del dominio %s", hostname)
}

func findCloudflareZone(zoneName string) (*cloudflareZone, error) {
	query := url.Values{}
	query.Set("name", zoneName)
	query.Set("account.id", core.Env.CLOUDFLARE_ACCOUNT)

	var zonesResponse cloudflareResponse[[]cloudflareZone]
	if requestError := cloudflareRequest(
		context.Background(),
		http.MethodGet,
		"/zones",
		query,
		nil,
		&zonesResponse,
	); requestError != nil {
		return nil, fmt.Errorf("error consultando zona Cloudflare: %w", requestError)
	}
	if !zonesResponse.Success {
		return nil, errors.New("Cloudflare rechazó la consulta de zona")
	}
	if len(zonesResponse.Result) != 1 {
		return nil, fmt.Errorf("se esperaba una zona Cloudflare para %s", zoneName)
	}

	return &zonesResponse.Result[0], nil
}

func findCloudflareWorkerDomain(hostname string) (*cloudflareWorkerDomain, error) {
	query := url.Values{}
	query.Set("hostname", hostname)

	var domainsResponse cloudflareResponse[[]cloudflareWorkerDomain]
	if requestError := cloudflareRequest(
		context.Background(),
		http.MethodGet,
		"/accounts/"+url.PathEscape(core.Env.CLOUDFLARE_ACCOUNT)+"/workers/domains",
		query,
		nil,
		&domainsResponse,
	); requestError != nil {
		return nil, fmt.Errorf("error consultando Worker Custom Domain: %w", requestError)
	}
	if !domainsResponse.Success {
		return nil, errors.New("Cloudflare rechazó la consulta de Worker Custom Domains")
	}

	for index := range domainsResponse.Result {
		if strings.EqualFold(domainsResponse.Result[index].Hostname, hostname) {
			return &domainsResponse.Result[index], nil
		}
	}
	return nil, nil
}

func cloudflareRequest(
	parentContext context.Context,
	method string,
	path string,
	query url.Values,
	body any,
	target any,
) error {
	requestContext, cancelRequest := context.WithTimeout(parentContext, 30*time.Second)
	defer cancelRequest()

	var requestBody io.Reader
	if body != nil {
		bodyBytes, marshalError := json.Marshal(body)
		if marshalError != nil {
			return marshalError
		}
		requestBody = bytes.NewReader(bodyBytes)
	}

	requestURL := cloudflareAPIBaseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}
	request, requestError := http.NewRequestWithContext(requestContext, method, requestURL, requestBody)
	if requestError != nil {
		return requestError
	}
	request.Header.Set("Authorization", "Bearer "+core.Env.CLOUDFLARE_TOKEN)
	request.Header.Set("Content-Type", "application/json")

	response, responseError := http.DefaultClient.Do(request)
	if responseError != nil {
		return responseError
	}
	defer response.Body.Close()

	responseBytes, readError := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if readError != nil {
		return readError
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Cloudflare API HTTP %d", response.StatusCode)
	}
	if unmarshalError := json.Unmarshal(responseBytes, target); unmarshalError != nil {
		return unmarshalError
	}

	return nil
}
