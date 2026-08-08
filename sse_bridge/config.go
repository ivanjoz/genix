package main

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// defaultListenPort keeps the bridge next to the other Genix services running on
// the same hosts (14008 ScyllaDB, 14010 backend, 14446 GenixSearch).
const defaultListenPort = 14012

// BridgeConfig is everything this process needs. ApiKey is the only value that
// MUST match the backend: both the browser's session token and the service-auth
// header are HMACs keyed with it. It is the backend's SECRET_PHRASE under the
// name this process is deployed with.
type BridgeConfig struct {
	ListenPort  int
	ApiKey      string
	VerboseLogs bool
}

// configSubset is the slice of config.toml the bridge cares about.
//
// Two names for one value: a bridge host holds a minimal config.toml with
// only the keys this process needs, where the secret is sse_bridge.apikey. A
// developer machine has the backend's full file instead, where the same value is
// secret_phrase. They must be identical or every token is rejected.
type configSubset struct {
	SecretPhrase string `toml:"secret_phrase"`
	SSEBridge    struct {
		ApiKey string `toml:"apikey"`
	} `toml:"sse_bridge"`
}

// LoadBridgeConfig resolves the runtime configuration. config.toml is the
// primary source (same file and same lookup order as the backend's
// PopulateVariables), and environment variables override it so a systemd unit or
// a container can run without shipping the config file.
func LoadBridgeConfig() (BridgeConfig, error) {
	config := BridgeConfig{
		ListenPort:  defaultListenPort,
		VerboseLogs: strings.TrimSpace(os.Getenv("SSE_BRIDGE_VERBOSE")) == "1",
	}

	configPath, fileConfig := readConfigFile()
	if len(configPath) > 0 {
		logInfo("config loaded from", configPath)
	}
	config.ApiKey = fileConfig.SSEBridge.ApiKey
	if len(config.ApiKey) == 0 {
		config.ApiKey = fileConfig.SecretPhrase
	}

	if apiKeyFromEnvironment := strings.TrimSpace(os.Getenv("SSE_BRIDGE_APIKEY")); len(apiKeyFromEnvironment) > 0 {
		config.ApiKey = apiKeyFromEnvironment
	}
	if portFromEnvironment := strings.TrimSpace(os.Getenv("SSE_BRIDGE_PORT")); len(portFromEnvironment) > 0 {
		parsedPort, parseError := strconv.Atoi(portFromEnvironment)
		if parseError != nil || parsedPort <= 0 || parsedPort > 65535 {
			return config, errors.New("SSE_BRIDGE_PORT inválido: " + portFromEnvironment)
		}
		config.ListenPort = parsedPort
	}

	if len(config.ApiKey) == 0 {
		return config, errors.New("no se encontró apikey: agréguelo como apikey en la sección [sse_bridge] de config.toml (GENIX_CONFIG_FILE) o expórtelo como variable de entorno")
	}
	return config, nil
}

// readConfigFile walks the same candidate paths the backend uses, returning
// the first readable config.toml. A missing file is not an error here — the
// environment variables may carry everything the bridge needs.
func readConfigFile() (string, configSubset) {
	candidatePaths := []string{}
	if configuredPath := strings.TrimSpace(os.Getenv("GENIX_CONFIG_FILE")); len(configuredPath) > 0 {
		candidatePaths = append(candidatePaths, configuredPath)
	}
	candidatePaths = append(candidatePaths, "../config.toml", "config.toml")

	for _, candidatePath := range candidatePaths {
		fileContent, readError := os.ReadFile(candidatePath)
		if readError != nil {
			continue
		}
		parsedConfig := configSubset{}
		if parseError := toml.Unmarshal(fileContent, &parsedConfig); parseError != nil {
			logWarn("config.toml ilegible en", candidatePath, "::", parseError)
			continue
		}
		return candidatePath, parsedConfig
	}
	return "", configSubset{}
}
