package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pelletier/go-toml/v2"
)

// Mismos valores que las constantes de cloud/main.go: el backend deduce IS_PROD con
// Contains(APP_CODE, "_prd") y LAMBDA_RESPONSE_STREAMING debe coincidir con el InvokeMode
// declarado en template.yml.
const (
	lambdaAppCode           = "gerp_prd"
	lambdaResponseStreaming = "1"
)

// Mismo default que cloud/webpage-renderer.go.
const defaultRendererZipURL = "https://genix-dev.un.pe/webpage-renderer.zip"

// El CONFIG viaja zstd + base64 url-safe: es exactamente lo que MakeB64UrlDecode +
// DecompressZstd deshacen en backend/core/security.go, donde el payload descomprimido
// se parsea como TOML.
var urlSafeBase64Replacer = strings.NewReplacer("/", "_", "+", "-", "=", "~")

type lambdaConfig struct {
	AppName string `toml:"app_name"`
	AWS     struct {
		Profile string `toml:"profile"`
		Region  string `toml:"region"`
	} `toml:"aws"`
	Frontend struct {
		CDNURL             string `toml:"cdn_url"`
		WebpageRendererURL string `toml:"webpage_renderer_url"`
	} `toml:"frontend"`
	Cloudflare struct {
		Account string `toml:"account"`
		Token   string `toml:"token"`
		Bucket  string `toml:"bucket"`
	} `toml:"cloudflare"`
}

// updateLambdaEnvironmentVariables es el equivalente a 'cloud accion=2' pero con AWS CLI, sin
// compilar. UpdateFunctionConfiguration REEMPLAZA el entorno completo (no lo fusiona), así que
// cada Lambda recibe aquí TODAS sus variables o las que falten se borran de la función.
func updateLambdaEnvironmentVariables(context deployContext) error {
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("falta la herramienta requerida: aws")
	}

	configContent, err := os.ReadFile(context.configFile)
	if err != nil {
		return err
	}

	var config lambdaConfig
	if err := toml.Unmarshal(configContent, &config); err != nil {
		return fmt.Errorf("no se pudo parsear %s: %w", filepath.Base(context.configFile), err)
	}

	if config.AppName == "" || config.AWS.Profile == "" || config.AWS.Region == "" {
		return fmt.Errorf("app_name, aws.profile y aws.region son requeridos en %s",
			filepath.Base(context.configFile))
	}

	fmt.Printf("Perfil: %s | Región: %s | App: %s\n",
		config.AWS.Profile, config.AWS.Region, config.AppName)

	configValue, err := compressToUrlSafeBase64(configContent)
	if err != nil {
		return err
	}

	backendEnvironment := map[string]map[string]string{"Variables": {
		"APP_CODE":                  lambdaAppCode,
		"CONFIG":                    configValue,
		"LAMBDA_RESPONSE_STREAMING": lambdaResponseStreaming,
	}}

	someLambdaFailed := false
	for _, lambdaName := range []string{config.AppName + "-backend", config.AppName + "-backend_2"} {
		if err := applyLambdaEnvironment(config, lambdaName, backendEnvironment); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			someLambdaFailed = true
		}
	}

	// La Lambda de render recibe las variables sueltas (no el CONFIG comprimido) para no
	// obligar al handler de Node a descomprimir zstd.
	if config.Frontend.CDNURL == "" {
		fmt.Println("⚠️  frontend.cdn_url vacío: se omite la Lambda de render.")
	} else {
		cloudflareBucket := config.Cloudflare.Bucket
		if cloudflareBucket == "" {
			cloudflareBucket = config.AppName + "-files"
		}
		rendererZipURL := config.Frontend.WebpageRendererURL
		if rendererZipURL == "" {
			rendererZipURL = defaultRendererZipURL
		}

		rendererEnvironment := map[string]map[string]string{"Variables": {
			"RENDERER_ZIP_URL":   rendererZipURL,
			"FRONTEND_CDN":       config.Frontend.CDNURL,
			"CLOUDFLARE_ACCOUNT": config.Cloudflare.Account,
			"CLOUDFLARE_TOKEN":   config.Cloudflare.Token,
			"CLOUDFLARE_BUCKET":  cloudflareBucket,
		}}

		if err := applyLambdaEnvironment(config, config.AppName+"-webpage-renderer", rendererEnvironment); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			someLambdaFailed = true
		}
	}

	if someLambdaFailed {
		return fmt.Errorf("una o más Lambdas no se pudieron actualizar")
	}
	fmt.Println("✅ Variables de entorno actualizadas!")
	return nil
}

func compressToUrlSafeBase64(content []byte) (string, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return "", err
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(content, nil)
	if len(compressed) == 0 {
		return "", fmt.Errorf("no se pudo comprimir el archivo de configuración")
	}
	return urlSafeBase64Replacer.Replace(base64.StdEncoding.EncodeToString(compressed)), nil
}

// applyLambdaEnvironment salta las Lambdas inexistentes (el stack puede no tener la _2 ni el
// renderer) en vez de abortar el resto de la actualización.
func applyLambdaEnvironment(config lambdaConfig, lambdaName string, environment any) error {
	awsBaseArgs := []string{"--profile", config.AWS.Profile, "--region", config.AWS.Region, "lambda"}
	awsCommand := func(args ...string) *exec.Cmd {
		return exec.Command("aws", append(append([]string{}, awsBaseArgs...), args...)...)
	}

	// Sin stdout/stderr conectados: sondear una Lambda ausente no debe ensuciar la salida.
	if err := awsCommand("get-function-configuration", "--function-name", lambdaName).Run(); err != nil {
		fmt.Printf("⚠️  Lambda no encontrada, se omite: %s\n", lambdaName)
		return nil
	}

	environmentJSON, err := json.Marshal(environment)
	if err != nil {
		return err
	}

	fmt.Printf("--- Actualizando variables de %s ---\n", lambdaName)
	update := awsCommand("update-function-configuration",
		"--function-name", lambdaName,
		"--environment", string(environmentJSON),
		"--query", "LastModified", "--output", "text")
	update.Stdout, update.Stderr = os.Stdout, os.Stderr

	if err := update.Run(); err != nil {
		return fmt.Errorf("error al actualizar las variables de %s: %w", lambdaName, err)
	}
	return nil
}
