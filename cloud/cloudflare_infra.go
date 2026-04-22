package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go/v2"
	"github.com/cloudflare/cloudflare-go/v2/option"
	"github.com/cloudflare/cloudflare-go/v2/r2"
	"github.com/tidwall/sjson"
)

type cfManagedDomainResult struct {
	Domain  string `json:"domain"`
	Enabled bool   `json:"enabled"`
}

type cfManagedDomainResponse struct {
	Result  cfManagedDomainResult `json:"result"`
	Success bool                  `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func DeployCloudflareInfra(params DeployParams) {
	bucketName := params.APP_NAME + "-files"
	ctx := context.Background()

	fmt.Println("=== DESPLEGANDO INFRAESTRUCTURA ===")
	fmt.Println("Cloud Provider: Cloudflare")
	fmt.Printf("Account ID:     %s\n", params.CLOUDFLARE_ACCOUNT)
	fmt.Printf("Bucket R2:      %s\n\n", bucketName)

	fmt.Println("Conectando con la API de Cloudflare...")
	client := cloudflare.NewClient(
		option.WithAPIToken(params.CLOUDFLARE_TOKEN),
	)

	fmt.Printf("Revisando si el bucket '%s' existe en R2...\n", bucketName)
	_, err := client.R2.Buckets.New(ctx, r2.BucketNewParams{
		AccountID: cloudflare.F(params.CLOUDFLARE_ACCOUNT),
		Name:      cloudflare.F(bucketName),
	})

	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			fmt.Printf("El bucket '%s' ya existe, omitiendo creación.\n", bucketName)
		} else {
			panic("Error al crear el bucket R2: " + err.Error())
		}
	} else {
		fmt.Printf("Bucket '%s' creado exitosamente en R2!\n", bucketName)
	}

	fmt.Printf("\nHabilitando acceso público (r2.dev) para '%s'...\n", bucketName)
	publicURL := enableR2PublicAccess(params.CLOUDFLARE_ACCOUNT, params.CLOUDFLARE_TOKEN, bucketName)
	fmt.Printf("Acceso público habilitado!\n")
	fmt.Printf("URL pública: %s\n\n", publicURL)

	fmt.Println("Actualizando FRONTEND_CDN en credentials.json...")
	updateCredentialsCDN(GetBaseWD(), publicURL)
	fmt.Println("credentials.json actualizado!")
}

func enableR2PublicAccess(accountID, token, bucketName string) string {
	apiURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/r2/buckets/%s/domains/managed", accountID, bucketName)
	fmt.Printf("  -> PUT %s\n", apiURL)

	body := []byte(`{"enabled":true}`)
	req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(body))
	if err != nil {
		panic("Error al crear request para Cloudflare: " + err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic("Error al habilitar acceso público en R2: " + err.Error())
	}
	defer resp.Body.Close()

	fmt.Printf("  -> HTTP %d\n", resp.StatusCode)
	respBytes, _ := io.ReadAll(resp.Body)

	var result cfManagedDomainResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		panic("Error al parsear respuesta de Cloudflare: " + err.Error())
	}

	if !result.Success || result.Result.Domain == "" {
		panic("Error al habilitar acceso público en R2: " + string(respBytes))
	}

	return "https://" + result.Result.Domain
}

func updateCredentialsCDN(baseWD, publicURL string) {
	filePath := baseWD + "/credentials.json"
	fmt.Printf("  -> Leyendo %s\n", filePath)
	content, err := ReadFile(filePath)
	if err != nil {
		panic("Error leyendo credentials.json: " + err.Error())
	}

	updated, err := sjson.Set(string(content), "FRONTEND_CDN", publicURL)
	if err != nil {
		panic("Error actualizando FRONTEND_CDN: " + err.Error())
	}

	if err := os.WriteFile(filePath, []byte(updated), 0644); err != nil {
		panic("Error guardando credentials.json: " + err.Error())
	}

	fmt.Printf("  -> FRONTEND_CDN = %s\n", publicURL)
}
