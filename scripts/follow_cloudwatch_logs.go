package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type cloudWatchLogsConfig struct {
	AppName string `toml:"app_name"`
	AWS     struct {
		Profile string `toml:"profile"`
		Region  string `toml:"region"`
	} `toml:"aws"`
}

// FollowCloudWatchLogs sigue el grupo que BackendLogGroup declara en cloud/template.yml.
func FollowCloudWatchLogs() {
	configPath := resolveConfigPath()
	config, err := loadCloudWatchLogsConfig(configPath)
	if err != nil {
		exitWithCloudWatchLogsError(err)
	}
	if _, err := exec.LookPath("aws"); err != nil {
		exitWithCloudWatchLogsError(fmt.Errorf("falta la herramienta requerida: aws"))
	}

	logGroup := "/aws/lambda/" + config.AppName + "-backend"
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("Perfil: %s | Región: %s | Log group: %s\n", config.AWS.Profile, config.AWS.Region, logGroup)
	fmt.Println("Siguiendo CloudWatch Logs; presione Ctrl+C para salir.")

	command := exec.Command("aws", cloudWatchLogsArguments(config)...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		exitWithCloudWatchLogsError(fmt.Errorf("aws logs tail terminó con error: %w", err))
	}
}

// GENIX_CONFIG_FILE lo define deploy.sh; el fallback conserva el uso directo desde scripts/.
func resolveConfigPath() string {
	configuredPath := strings.TrimSpace(os.Getenv("GENIX_CONFIG_FILE"))
	if configuredPath != "" {
		if filepath.IsAbs(configuredPath) || pathExists(configuredPath) {
			return configuredPath
		}
		// deploy.sh se inicia en la raíz, pero ejecuta este dispatcher dentro de scripts/.
		repositoryRelativePath := filepath.Join("..", configuredPath)
		if pathExists(repositoryRelativePath) {
			return repositoryRelativePath
		}
		return configuredPath
	}
	return filepath.Join("..", "config.toml")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadCloudWatchLogsConfig(configPath string) (cloudWatchLogsConfig, error) {
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		return cloudWatchLogsConfig{}, fmt.Errorf("no se pudo leer %s: %w", configPath, err)
	}

	var config cloudWatchLogsConfig
	if err := toml.Unmarshal(configContent, &config); err != nil {
		return cloudWatchLogsConfig{}, fmt.Errorf("no se pudo parsear %s: %w", configPath, err)
	}
	config.AppName = strings.TrimSpace(config.AppName)
	config.AWS.Profile = strings.TrimSpace(config.AWS.Profile)
	config.AWS.Region = strings.TrimSpace(config.AWS.Region)
	if config.AppName == "" || config.AWS.Profile == "" || config.AWS.Region == "" {
		return cloudWatchLogsConfig{}, fmt.Errorf("app_name, aws.profile y aws.region son requeridos en %s", configPath)
	}
	return config, nil
}

func cloudWatchLogsArguments(config cloudWatchLogsConfig) []string {
	return []string{
		"--profile", config.AWS.Profile,
		"--region", config.AWS.Region,
		"logs", "tail", "/aws/lambda/" + config.AppName + "-backend",
		"--follow",
	}
}

func exitWithCloudWatchLogsError(err error) {
	fmt.Fprintf(os.Stderr, "follow_cloudwatch_logs failed: %v\n", err)
	os.Exit(1)
}
