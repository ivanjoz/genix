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

// cloudflareAPIError carries the HTTP status plus Cloudflare's own error payload so a
// caller can branch on the specific failure instead of matching an error string: a 404
// on an R2 path means that bucket does not exist, not that the endpoint is wrong.
type cloudflareAPIError struct {
	StatusCode int
	Method     string
	Path       string
	Detail     string
}

func (apiError *cloudflareAPIError) Error() string {
	message := fmt.Sprintf("Cloudflare API HTTP %d en %s %s", apiError.StatusCode, apiError.Method, apiError.Path)
	if apiError.Detail != "" {
		message += ": " + apiError.Detail
	}
	return message
}

// cloudflareErrorDetail flattens the `errors: [{code, message}]` array Cloudflare returns
// on failure into "[code] message" pairs, falling back to the raw body when the payload
// is not the documented shape (HTML from an edge error, for example).
func cloudflareErrorDetail(responseBytes []byte) string {
	var payload struct {
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(responseBytes, &payload) == nil && len(payload.Errors) > 0 {
		details := make([]string, 0, len(payload.Errors))
		for _, apiError := range payload.Errors {
			details = append(details, fmt.Sprintf("[%d] %s", apiError.Code, apiError.Message))
		}
		return strings.Join(details, "; ")
	}

	rawBody := strings.TrimSpace(string(responseBytes))
	if len(rawBody) > 500 {
		rawBody = rawBody[:500] + "..."
	}
	return rawBody
}

// maskCloudflareAccount keeps the account ID out of error text and logs; the path is only
// useful to identify which endpoint failed.
func maskCloudflareAccount(requestPath string) string {
	account := strings.TrimSpace(core.Env.CLOUDFLARE_ACCOUNT)
	if account == "" {
		return requestPath
	}
	return strings.ReplaceAll(requestPath, url.PathEscape(account), "<account>")
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
	if deployError := DeployCloudflareWorker(); deployError != nil {
		return core.FuncResponse{Error: deployError.Error()}
	}

	return core.FuncResponse{Message: "Cloudflare Worker de tiendas desplegado"}
}

// DeployCloudflareWorker publica el Worker que sirve las tiendas. Es independiente del
// contenido: no sube HTML de ningún tenant (eso lo hace la Lambda de render, por company),
// así que basta con desplegarlo una vez y cada vez que cambie serve-worker.js.
func DeployCloudflareWorker() error {
	projectRoot, rootError := findGenixProjectRoot()
	if rootError != nil {
		return rootError
	}

	if strings.TrimSpace(core.Env.CLOUDFLARE_ACCOUNT) == "" ||
		strings.TrimSpace(core.Env.CLOUDFLARE_TOKEN) == "" {
		return errors.New("CLOUDFLARE_ACCOUNT y CLOUDFLARE_TOKEN son requeridos")
	}
	// Sin bucket el Worker se desplegaría con un binding vacío y devolvería 404 a todo.
	if strings.TrimSpace(core.Env.CLOUDFLARE_BUCKET) == "" {
		return errors.New("CLOUDFLARE_BUCKET es requerido para bindear el Worker al HTML en R2")
	}

	workerDirectory := filepath.Join(projectRoot, "frontend", "webpage", "cloudflare")
	fmt.Printf("[cloudflare-worker] dir=%s\n", workerDirectory)

	if deployError := deployStorefrontWorker(workerDirectory); deployError != nil {
		return fmt.Errorf("error desplegando Cloudflare Worker: %w", deployError)
	}

	fmt.Println("[cloudflare-worker] deployment completed")
	return nil
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

// findGenixProjectRoot forwards to core.FindProjectRoot, which owns the lookup so `cloud` can use it
// too without importing this package (that import would be a cycle).
func findGenixProjectRoot() (string, error) {
	return core.FindProjectRoot()
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
		return &cloudflareAPIError{
			StatusCode: response.StatusCode,
			Method:     method,
			Path:       maskCloudflareAccount(path),
			Detail:     cloudflareErrorDetail(responseBytes),
		}
	}
	if unmarshalError := json.Unmarshal(responseBytes, target); unmarshalError != nil {
		return unmarshalError
	}

	return nil
}
