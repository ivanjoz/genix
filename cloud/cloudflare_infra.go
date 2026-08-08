package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go/v2"
	"github.com/cloudflare/cloudflare-go/v2/option"
	"github.com/cloudflare/cloudflare-go/v2/r2"
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

// DeployCloudflareInfra provisions R2 and returns its public URL for the Lambda renderer.
func DeployCloudflareInfra(params DeployParams) string {
	// main() ya resolvió el nombre: cloudflare.bucket del environment seleccionado si está seteado,
	// "<app_name>-files" en caso contrario.
	bucketName := params.Cloudflare.Bucket
	bucketSource := "environment seleccionado (cloudflare.bucket)"
	if bucketName == params.AppName+"-files" {
		bucketSource = "autogenerado desde app_name"
	}
	ctx := context.Background()

	fmt.Println("=== DESPLEGANDO INFRAESTRUCTURA ===")
	fmt.Println("Cloud Provider: Cloudflare")
	fmt.Printf("Account ID:     %s\n", params.Cloudflare.Account)
	fmt.Printf("Bucket R2:      %s (%s)\n\n", bucketName, bucketSource)

	fmt.Println("Conectando con la API de Cloudflare...")
	client := cloudflare.NewClient(
		option.WithAPIToken(params.Cloudflare.Token),
	)

	fmt.Printf("Revisando si el bucket '%s' existe en R2...\n", bucketName)
	_, err := client.R2.Buckets.New(ctx, r2.BucketNewParams{
		AccountID: cloudflare.F(params.Cloudflare.Account),
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
	publicURL := enableR2PublicAccess(params.Cloudflare.Account, params.Cloudflare.Token, bucketName)
	fmt.Printf("Acceso público habilitado!\n")
	fmt.Printf("URL pública: %s\n\n", publicURL)

	fmt.Println("Actualizando frontend.cdn_url en el environment seleccionado...")
	updateConfigCDN(publicURL)
	fmt.Println("Archivo de configuración actualizado!")
	return publicURL
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

// cdnUrlInTomlPattern ancla en la clave de la sección [frontend]; en el archivo sólo
// existe un cdn_url. Reemplazo de texto y no re-serialización por la misma razón que
// lambdaUrlInTomlPattern: el archivo se mantiene a mano y sus comentarios son la razón
// de ser del formato TOML.
var cdnUrlInTomlPattern = regexp.MustCompile(`(?m)^(\s*cdn_url\s*=\s*")([^"]*)(")`)

func updateConfigCDN(publicURL string) {
	filePath := GetConfigPath()
	fmt.Printf("  -> Leyendo %s\n", filePath)
	content, err := ReadFile(filePath)
	if err != nil {
		panic("Error leyendo " + filePath + ": " + err.Error())
	}

	if !cdnUrlInTomlPattern.Match(content) {
		panic("No se encontró la clave cdn_url en " + filePath)
	}
	updated := cdnUrlInTomlPattern.ReplaceAllString(string(content), "${1}"+publicURL+"${3}")

	if err := os.WriteFile(filePath, []byte(updated), 0644); err != nil {
		panic("Error guardando " + filePath + ": " + err.Error())
	}

	fmt.Printf("  -> frontend.cdn_url = %s\n", publicURL)
}
