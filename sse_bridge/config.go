package main

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
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

// credentialsSubset is the slice of credentials.json the bridge cares about.
//
// Two names for one value: a bridge host holds a minimal credentials.json with
// only the keys this process needs, where the secret is SSE_BRIDGE_APIKEY. A
// developer machine has the backend's full file instead, where the same value is
// SECRET_PHRASE. They must be identical or every token is rejected.
type credentialsSubset struct {
	ApiKey       string `json:"SSE_BRIDGE_APIKEY"`
	SecretPhrase string `json:"SECRET_PHRASE"`
}

// LoadBridgeConfig resolves the runtime configuration. credentials.json is the
// primary source (same file and same lookup order as the backend's
// PopulateVariables), and environment variables override it so a systemd unit or
// a container can run without shipping the credentials file.
func LoadBridgeConfig() (BridgeConfig, error) {
	config := BridgeConfig{
		ListenPort:  defaultListenPort,
		VerboseLogs: strings.TrimSpace(os.Getenv("SSE_BRIDGE_VERBOSE")) == "1",
	}

	credentialsPath, credentials := readCredentialsFile()
	if len(credentialsPath) > 0 {
		logInfo("credentials loaded from", credentialsPath)
	}
	config.ApiKey = credentials.ApiKey
	if len(config.ApiKey) == 0 {
		config.ApiKey = credentials.SecretPhrase
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
		return config, errors.New("no se encontró SSE_BRIDGE_APIKEY: agréguelo a credentials.json (GENIX_CREDENTIALS_FILE) o expórtelo como variable de entorno")
	}
	return config, nil
}

// readCredentialsFile walks the same candidate paths the backend uses, returning
// the first readable credentials.json. A missing file is not an error here — the
// environment variables may carry everything the bridge needs.
func readCredentialsFile() (string, credentialsSubset) {
	candidatePaths := []string{}
	if configuredPath := strings.TrimSpace(os.Getenv("GENIX_CREDENTIALS_FILE")); len(configuredPath) > 0 {
		candidatePaths = append(candidatePaths, configuredPath)
	}
	candidatePaths = append(candidatePaths, "../credentials.json", "credentials.json")

	for _, candidatePath := range candidatePaths {
		fileContent, readError := os.ReadFile(candidatePath)
		if readError != nil {
			continue
		}
		credentials := credentialsSubset{}
		if parseError := json.Unmarshal(fileContent, &credentials); parseError != nil {
			logWarn("credentials.json ilegible en", candidatePath, "::", parseError)
			continue
		}
		return candidatePath, credentials
	}
	return "", credentialsSubset{}
}
