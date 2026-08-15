package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// openRouterImagesEndpoint es el endpoint de generación de imágenes, distinto de
// /chat/completions: los modelos vectoriales no aceptan parámetros de chat (su lista de
// supported_parameters viene vacía) y devuelven el archivo, no un mensaje.
const openRouterImagesEndpoint = "https://openrouter.ai/api/v1/images"

// vectorImageConfig es la vista parcial de config.toml que necesita este script: la key del
// proveedor y la tabla [[image_models]]. Deliberadamente NO lee [[models]]: ese array es el
// registro de modelos de chat del agente y su primer elemento decide el default del selector.
type vectorImageConfig struct {
	Agent struct {
		OpenRouterKey string `toml:"openrouter_key"`
	} `toml:"agent"`
	ImageModels []struct {
		ID       string `toml:"id"`
		Provider string `toml:"provider"`
	} `toml:"image_models"`
}

// imagesResponse refleja la respuesta real del endpoint, verificada contra una llamada en vivo:
// cada elemento trae el archivo en base64 y su media_type ("image/svg+xml" en los modelos
// vectoriales), y usage.cost el gasto real en USD de esa llamada.
type imagesResponse struct {
	Data []struct {
		B64JSON   string `json:"b64_json"`
		MediaType string `json:"media_type"`
	} `json:"data"`
	Usage struct {
		Cost float64 `json:"cost"`
	} `json:"usage"`
}

// GenerateVectorImage genera SVG con los modelos de [[image_models]] y los escribe en disco.
// El destino por defecto es tmp/ (ignorado por git) porque un asset generado se revisa antes
// de entrar al repositorio: el script produce candidatos, no los publica.
func GenerateVectorImage(args []string) {
	flagSet := flag.NewFlagSet("generate_vector_image", flag.ExitOnError)
	promptText := flagSet.String("prompt", "", "descripción de la imagen a generar (requerido)")
	modelSelector := flagSet.String("model", "", "id del modelo o parte de él; vacío = primer [[image_models]]")
	outputPath := flagSet.String("out", "", "ruta del archivo destino; vacío = tmp/vector/<slug del prompt>.svg")
	aspectRatio := flagSet.String("aspect", "1:1", "relación de aspecto, p.ej. 1:1, 16:9, 3:4")
	imageCount := flagSet.Int("n", 1, "cuántas variantes generar en una sola llamada")
	if err := flagSet.Parse(args); err != nil {
		exitWithVectorImageError(err)
	}
	if strings.TrimSpace(*promptText) == "" {
		exitWithVectorImageError(fmt.Errorf("-prompt es requerido"))
	}

	configPath := resolveConfigPath()
	config, err := loadVectorImageConfig(configPath)
	if err != nil {
		exitWithVectorImageError(err)
	}
	modelID, err := selectImageModel(config, *modelSelector)
	if err != nil {
		exitWithVectorImageError(err)
	}

	fmt.Printf("Config: %s\nModelo: %s | Aspecto: %s | Variantes: %d\n",
		configPath, modelID, *aspectRatio, *imageCount)
	fmt.Println("Generando; una imagen vectorial puede tardar más de un minuto...")

	response, err := requestVectorImages(config.Agent.OpenRouterKey, modelID, *promptText, *aspectRatio, *imageCount)
	if err != nil {
		exitWithVectorImageError(err)
	}
	if len(response.Data) == 0 {
		exitWithVectorImageError(fmt.Errorf("el modelo no devolvió ninguna imagen"))
	}

	writtenPaths, err := writeVectorImages(response, *outputPath, *promptText)
	if err != nil {
		exitWithVectorImageError(err)
	}
	for _, path := range writtenPaths {
		fileInfo, statErr := os.Stat(path)
		if statErr != nil {
			fmt.Printf("Generado: %s\n", path)
			continue
		}
		fmt.Printf("Generado: %s (%.1f KB)\n", path, float64(fileInfo.Size())/1024)
	}
	// El costo se imprime siempre: estos modelos cobran por token de imagen y una sola llamada
	// cuesta bastante más que una de texto, así que el gasto tiene que ser visible en el momento.
	fmt.Printf("Costo de la llamada: $%.4f USD\n", response.Usage.Cost)
}

func loadVectorImageConfig(configPath string) (vectorImageConfig, error) {
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		return vectorImageConfig{}, fmt.Errorf("no se pudo leer %s: %w", configPath, err)
	}
	var config vectorImageConfig
	if err := toml.Unmarshal(configContent, &config); err != nil {
		return vectorImageConfig{}, fmt.Errorf("no se pudo parsear %s: %w", configPath, err)
	}
	config.Agent.OpenRouterKey = strings.TrimSpace(config.Agent.OpenRouterKey)
	if config.Agent.OpenRouterKey == "" {
		return vectorImageConfig{}, fmt.Errorf("agent.openrouter_key es requerido en %s", configPath)
	}
	if len(config.ImageModels) == 0 {
		return vectorImageConfig{}, fmt.Errorf("no hay entradas [[image_models]] en %s", configPath)
	}
	return config, nil
}

// selectImageModel resuelve el selector contra la tabla: vacío toma la primera entrada (el
// orden del archivo decide el default, igual que en [[models]]) y cualquier otro valor hace
// coincidencia por subcadena, para poder pedir "pro" en vez del id completo.
func selectImageModel(config vectorImageConfig, modelSelector string) (string, error) {
	availableIDs := make([]string, 0, len(config.ImageModels))
	for _, model := range config.ImageModels {
		availableIDs = append(availableIDs, model.ID)
	}

	selector := strings.TrimSpace(modelSelector)
	for _, model := range config.ImageModels {
		if selector != "" && !strings.Contains(model.ID, selector) {
			continue
		}
		// Sólo OpenRouter tiene este endpoint; una entrada con otro proveedor es un error de
		// configuración que conviene nombrar aquí y no como un 404 del upstream.
		if provider := strings.TrimSpace(model.Provider); provider != "" && provider != "openrouter" {
			return "", fmt.Errorf("el modelo %s declara provider %q; sólo openrouter genera imágenes", model.ID, provider)
		}
		return model.ID, nil
	}
	return "", fmt.Errorf("ningún [[image_models]] coincide con %q; disponibles: %s",
		selector, strings.Join(availableIDs, ", "))
}

func requestVectorImages(openRouterKey, modelID, promptText, aspectRatio string, imageCount int) (imagesResponse, error) {
	requestBody, err := json.Marshal(map[string]any{
		"model":        modelID,
		"prompt":       promptText,
		"aspect_ratio": aspectRatio,
		"n":            imageCount,
	})
	if err != nil {
		return imagesResponse{}, fmt.Errorf("no se pudo serializar la petición: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, openRouterImagesEndpoint, bytes.NewReader(requestBody))
	if err != nil {
		return imagesResponse{}, fmt.Errorf("no se pudo crear la petición: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+openRouterKey)
	request.Header.Set("Content-Type", "application/json")

	// Timeout largo a propósito: generar un SVG tarda del orden de un minuto, muy por encima
	// de lo que responde un modelo de texto.
	httpClient := &http.Client{Timeout: 5 * time.Minute}
	httpResponse, err := httpClient.Do(request)
	if err != nil {
		return imagesResponse{}, fmt.Errorf("la llamada a OpenRouter falló: %w", err)
	}
	defer httpResponse.Body.Close()

	var response imagesResponse
	responseBody := new(bytes.Buffer)
	if _, err := responseBody.ReadFrom(httpResponse.Body); err != nil {
		return imagesResponse{}, fmt.Errorf("no se pudo leer la respuesta: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return imagesResponse{}, fmt.Errorf("OpenRouter respondió %d: %s",
			httpResponse.StatusCode, strings.TrimSpace(responseBody.String()))
	}
	if err := json.Unmarshal(responseBody.Bytes(), &response); err != nil {
		return imagesResponse{}, fmt.Errorf("no se pudo parsear la respuesta: %w", err)
	}
	return response, nil
}

// writeVectorImages decide la ruta de cada variante y la escribe. Con -out se respeta la ruta
// pedida; sin él se deriva del prompt para que el archivo sea reconocible sin abrirlo.
func writeVectorImages(response imagesResponse, requestedPath, promptText string) ([]string, error) {
	basePath := strings.TrimSpace(requestedPath)
	if basePath == "" {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return nil, err
		}
		basePath = filepath.Join(repoRoot, "tmp", "vector", slugFromPrompt(promptText)+".svg")
	}

	writtenPaths := make([]string, 0, len(response.Data))
	for index, image := range response.Data {
		fileContent, err := base64.StdEncoding.DecodeString(image.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("no se pudo decodificar la imagen %d: %w", index+1, err)
		}

		targetPath := basePath
		// Varias variantes en una llamada no pueden compartir archivo: se numeran desde la
		// segunda para que la primera conserve el nombre pedido.
		if index > 0 {
			extension := filepath.Ext(basePath)
			targetPath = strings.TrimSuffix(basePath, extension) + fmt.Sprintf("_%d", index+1) + extension
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("no se pudo crear la carpeta destino: %w", err)
		}
		if err := os.WriteFile(targetPath, fileContent, 0644); err != nil {
			return nil, fmt.Errorf("no se pudo escribir %s: %w", targetPath, err)
		}
		if image.MediaType != "" && image.MediaType != "image/svg+xml" {
			fmt.Printf("Aviso: el modelo devolvió %s, no SVG\n", image.MediaType)
		}
		writtenPaths = append(writtenPaths, targetPath)
	}
	return writtenPaths, nil
}

var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

func slugFromPrompt(promptText string) string {
	slug := nonSlugCharacters.ReplaceAllString(strings.ToLower(promptText), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 60 {
		slug = strings.Trim(slug[:60], "-")
	}
	if slug == "" {
		return "vector"
	}
	return slug
}

func exitWithVectorImageError(err error) {
	fmt.Fprintf(os.Stderr, "generate_vector_image: %v\n", err)
	os.Exit(1)
}
