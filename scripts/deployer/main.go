package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Misma versión de Node que exportaba deploy.sh, para que 'bun' y 'serve' resuelvan igual.
const nodeBinaryPath = ".nvm/versions/node/v20.16.0/bin"

func main() {
	repositoryRoot, err := findRepositoryRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	prependNodeToPath()

	defaultConfig := filepath.Join(repositoryRoot, "config.toml")
	alternateConfig := filepath.Join(repositoryRoot, "config.1.toml")

	arguments := os.Args[1:]
	selection := parseArguments(arguments)

	var configFile string

	// Haber pasado argumentos es lo que decide el modo: si ninguno era válido hay que fallar,
	// no abrir la interfaz, o un typo en un script quedaría esperando input.
	if len(arguments) > 0 {
		if len(selection.actionIDs) == 0 && len(selection.scriptKeys) == 0 {
			fmt.Fprintf(os.Stderr, "❌ Ninguna acción ni script válido en: %s\n", strings.Join(arguments, " "))
			os.Exit(1)
		}
		// Modo no interactivo: sin TUI, el entorno sale del env o del archivo por defecto.
		configFile = strings.TrimSpace(os.Getenv("GENIX_CONFIG_FILE"))
		if configFile == "" {
			configFile = defaultConfig
		}
	} else {
		environmentPaths := []string{defaultConfig}
		if fileExists(alternateConfig) {
			environmentPaths = append(environmentPaths, alternateConfig)
		}

		var confirmed bool
		configFile, selection.actionIDs, selection.scriptKeys, confirmed = runDeployTUI(environmentPaths, readBackendProvider)
		if !confirmed {
			fmt.Println("Cancelado.")
			return
		}
	}

	if !fileExists(configFile) {
		fmt.Fprintf(os.Stderr, "❌ No se encontró el archivo de configuración: %s\n", configFile)
		os.Exit(1)
	}
	if len(selection.actionIDs) == 0 && len(selection.scriptKeys) == 0 {
		fmt.Println("No se seleccionó ninguna acción ni script.")
		return
	}

	// Todo proceso hijo, incluidos los tres pasos de la acción 6, usa exactamente este archivo.
	os.Setenv("GENIX_CONFIG_FILE", configFile)
	fmt.Printf("✅ Environment seleccionado: %s\n", filepath.Base(configFile))

	context := deployContext{
		repositoryRoot:  repositoryRoot,
		configFile:      configFile,
		goBinary:        resolveGoBinary(),
		companyID:       selection.companyID,
		scriptArguments: selection.scriptArguments,
	}

	if err := runSelectedActions(context, selection.actionIDs); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ %v\n", err)
		os.Exit(1)
	}
	if err := runSelectedScripts(context, selection.scriptKeys); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\nFinalizado!")
}

// commandLineSelection es lo que se pudo deducir de los argumentos.
type commandLineSelection struct {
	actionIDs  []int
	scriptKeys []string
	// Sólo tiene sentido cuando se pidió un único script que declara argumentsHint.
	scriptArguments []string
	companyID       string
}

// parseArguments acepta IDs de acción y claves de script separados por espacios o comas. Los
// tokens que no son ninguna de las dos cosas son los argumentos del script pedido cuando ese
// script los necesita (así "create <ruta> <tabla> ..." funciona igual que en app.sh) y, si no,
// el CompanyID de la acción 11. De esa forma "6 9" siguen siendo dos acciones.
func parseArguments(arguments []string) commandLineSelection {
	tokens := strings.FieldsFunc(strings.Join(arguments, " "), func(character rune) bool {
		return character == ' ' || character == ','
	})

	var selection commandLineSelection
	var unrecognized []string

	for _, token := range tokens {
		if parsedID, err := strconv.Atoi(token); err == nil && findAction(parsedID) != nil {
			selection.actionIDs = append(selection.actionIDs, parsedID)
			continue
		}
		if findScript(token) != nil {
			selection.scriptKeys = append(selection.scriptKeys, token)
			continue
		}
		unrecognized = append(unrecognized, token)
	}

	if len(selection.scriptKeys) == 1 && findScript(selection.scriptKeys[0]).argumentsHint != "" {
		selection.scriptArguments = unrecognized
	} else if len(unrecognized) > 0 {
		selection.companyID = unrecognized[0]
	}

	return selection
}

// findRepositoryRoot sube desde el directorio actual buscando la raíz del repo, para que el
// deployer funcione lo mismo si se lo invoca desde scripts/ que desde la raíz.
func findRepositoryRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if fileExists(filepath.Join(directory, "backend")) && fileExists(filepath.Join(directory, "scripts")) {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("no se encontró la raíz del repositorio desde el directorio actual")
		}
		directory = parent
	}
}

func readBackendProvider(configPath string) string {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var parsed struct {
		Providers struct {
			Backend string `toml:"backend"`
		} `toml:"providers"`
	}
	if err := toml.Unmarshal(content, &parsed); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parsed.Providers.Backend))
}

func resolveGoBinary() string {
	const systemGoBinary = "/usr/local/go/bin/go"
	if info, err := os.Stat(systemGoBinary); err == nil && info.Mode()&0o111 != 0 {
		return systemGoBinary
	}
	return "go"
}

func prependNodeToPath() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	os.Setenv("PATH", filepath.Join(home, nodeBinaryPath)+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
