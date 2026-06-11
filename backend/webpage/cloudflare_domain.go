package webpage

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
	"strings"
	"time"
)

const (
	cloudflareAPIBaseURL = "https://api.cloudflare.com/client/v4"
	storefrontWorkerName = "genix-storefront"
)

type cloudflareResponse[T any] struct {
	Result  T     `json:"result"`
	Success bool  `json:"success"`
	Errors  []any `json:"errors"`
}

type cloudflareZone struct {
	ID string `json:"id"`
}

type cloudflareDNSRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type cloudflareWorkerDomain struct {
	Hostname string `json:"hostname"`
	Service  string `json:"service"`
}

// provisionStorefrontDomain rejects occupied names and attaches available names to the Worker.
func provisionStorefrontDomain(hostname string, allowExistingStorefrontDomain bool) error {
	if strings.TrimSpace(core.Env.CLOUDFLARE_ACCOUNT) == "" ||
		strings.TrimSpace(core.Env.CLOUDFLARE_TOKEN) == "" {
		return errors.New("CLOUDFLARE_ACCOUNT y CLOUDFLARE_TOKEN son requeridos")
	}

	zoneName := strings.TrimSpace(core.Env.ZONE_NAME)
	if zoneName == "" {
		return errors.New("ZONE_NAME es requerido")
	}

	zone, zoneError := findCloudflareZone(zoneName)
	if zoneError != nil {
		return zoneError
	}

	existingWorkerDomain, workerDomainError := findCloudflareWorkerDomain(hostname)
	if workerDomainError != nil {
		return workerDomainError
	}
	if existingWorkerDomain != nil {
		if allowExistingStorefrontDomain && existingWorkerDomain.Service == storefrontWorkerName {
			core.Log("Dominio Cloudflare ya registrado::", hostname)
			return nil
		}
		return fmt.Errorf("el dominio %s ya está registrado", hostname)
	}

	dnsRecords, dnsError := findConflictingDNSRecords(zone.ID, hostname)
	if dnsError != nil {
		return dnsError
	}
	if len(dnsRecords) > 0 {
		return fmt.Errorf("el dominio %s ya tiene un registro DNS %s", hostname, dnsRecords[0].Type)
	}

	// Worker Custom Domains creates the proxied DNS record and TLS certificate atomically.
	payload := map[string]string{
		"environment": "production",
		"hostname":    hostname,
		"service":     storefrontWorkerName,
		"zone_id":     zone.ID,
	}
	var createResponse cloudflareResponse[cloudflareWorkerDomain]
	requestPath := "/accounts/" + url.PathEscape(core.Env.CLOUDFLARE_ACCOUNT) + "/workers/domains"
	if requestError := cloudflareRequest(http.MethodPut, requestPath, nil, payload, &createResponse); requestError != nil {
		return fmt.Errorf("error creando Worker Custom Domain: %w", requestError)
	}
	if !createResponse.Success {
		return errors.New("Cloudflare rechazó la creación del dominio")
	}

	core.Log("Dominio Cloudflare registrado::", hostname)
	return nil
}

// findCloudflareZone resolves the configured zone within the configured account.
func findCloudflareZone(zoneName string) (*cloudflareZone, error) {
	query := url.Values{"name": {zoneName}, "account.id": {core.Env.CLOUDFLARE_ACCOUNT}}
	var response cloudflareResponse[[]cloudflareZone]
	if requestError := cloudflareRequest(http.MethodGet, "/zones", query, nil, &response); requestError != nil {
		return nil, fmt.Errorf("error consultando la zona Cloudflare: %w", requestError)
	}
	if !response.Success || len(response.Result) != 1 {
		return nil, fmt.Errorf("no se encontró la zona Cloudflare %s", zoneName)
	}
	return &response.Result[0], nil
}

// findCloudflareWorkerDomain checks reservations created through Worker Custom Domains.
func findCloudflareWorkerDomain(hostname string) (*cloudflareWorkerDomain, error) {
	query := url.Values{"hostname": {hostname}}
	var response cloudflareResponse[[]cloudflareWorkerDomain]
	requestPath := "/accounts/" + url.PathEscape(core.Env.CLOUDFLARE_ACCOUNT) + "/workers/domains"
	if requestError := cloudflareRequest(http.MethodGet, requestPath, query, nil, &response); requestError != nil {
		return nil, fmt.Errorf("error consultando Worker Custom Domains: %w", requestError)
	}
	if !response.Success {
		return nil, errors.New("Cloudflare rechazó la consulta de Worker Custom Domains")
	}
	for index := range response.Result {
		if strings.EqualFold(response.Result[index].Hostname, hostname) {
			return &response.Result[index], nil
		}
	}
	return nil, nil
}

// findConflictingDNSRecords checks the record types that can occupy a storefront hostname.
func findConflictingDNSRecords(zoneID string, hostname string) ([]cloudflareDNSRecord, error) {
	conflictingRecords := []cloudflareDNSRecord{}
	for _, recordType := range []string{"A", "CNAME"} {
		query := url.Values{"name": {hostname}, "type": {recordType}}
		var response cloudflareResponse[[]cloudflareDNSRecord]
		requestPath := "/zones/" + url.PathEscape(zoneID) + "/dns_records"
		if requestError := cloudflareRequest(http.MethodGet, requestPath, query, nil, &response); requestError != nil {
			return nil, fmt.Errorf("error consultando registros DNS %s: %w", recordType, requestError)
		}
		if !response.Success {
			return nil, fmt.Errorf("Cloudflare rechazó la consulta de registros DNS %s", recordType)
		}
		conflictingRecords = append(conflictingRecords, response.Result...)
	}
	return conflictingRecords, nil
}

// cloudflareRequest executes a bounded authenticated API request without logging credentials.
func cloudflareRequest(method string, path string, query url.Values, body any, target any) error {
	requestContext, cancelRequest := context.WithTimeout(context.Background(), 30*time.Second)
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
	core.Log("Cloudflare API::", method, path)
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
		return fmt.Errorf("Cloudflare API HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBytes)))
	}
	if unmarshalError := json.Unmarshal(responseBytes, target); unmarshalError != nil {
		return unmarshalError
	}
	return nil
}
