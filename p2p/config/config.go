package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all configuration parameters for the P2P Bridge application, in the flat
// shape p2p/deploy/ and p2p/homelab_server/ consume. Load() populates it by parsing
// config.toml through fileConfig below.
type Config struct {
	AWSProfile               string
	AppName                  string
	SignalingAppName         string
	StackName                string
	SignalingEndpoint        string
	SignalingSocket          string
	ApiKey                   string
	LambdaFunctionName       string // Not in the file, derived from app_name
	LambdaFunctionNameActual string // Actual Lambda function name from CDK output
	AWSRegion                string
	AWSAccount               string
}

// fileConfig reflects config.toml's section layout (PLAN_CONFIG_TOML.md §2.7). It exists
// only for parsing: Load() copies its fields onto the flat Config above so TOML fixes the
// keys and the per-field lookup-by-multiple-names it replaced is no longer needed.
type fileConfig struct {
	AppName string `toml:"app_name"`
	AWS     struct {
		Profile string `toml:"profile"`
		Region  string `toml:"region"`
	} `toml:"aws"`
	Signaling struct {
		Socket             string `toml:"socket"`
		Endpoint           string `toml:"endpoint"`
		ApiKey             string `toml:"api_key"`
		AppName            string `toml:"app_name"`
		StackName          string `toml:"stack_name"`
		LambdaFunctionName string `toml:"lambda_function_name"`
		AWSAccount         string `toml:"aws_account"`
	} `toml:"signaling"`
}

// GetSignalingAppName returns the signaling app name, defaulting to app_name + "-signaling"
func (c *Config) GetSignalingAppName() string {
	if c.SignalingAppName != "" {
		return c.SignalingAppName
	}
	return c.AppName + "-signaling"
}

// GetStackName returns the stack name, defaulting to app_name + "-signaling"
func (c *Config) GetStackName() string {
	if c.StackName != "" {
		return c.StackName
	}
	return c.AppName + "-signaling"
}

// GetLambdaFunctionName returns the lambda function name (app_name + "-signaling")
func (c *Config) GetLambdaFunctionName() string {
	// If actual Lambda function name is set in config, use it
	if c.LambdaFunctionNameActual != "" {
		return c.LambdaFunctionNameActual
	}
	return c.AppName + "-signaling"
}

// configPath returns the path to config.toml relative to the project root
func configPath() (string, error) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Look for config.toml in current directory or parent directories
	for {
		configFile := filepath.Join(cwd, "config.toml")
		if _, err := os.Stat(configFile); err == nil {
			return configFile, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// Reached the root directory without finding config.toml
			return "", fmt.Errorf("config.toml not found in current directory or parent directories")
		}
		cwd = parent
	}
}

// Load reads and parses the config.toml file
func Load() (*Config, error) {
	configFile, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.toml: %w", err)
	}

	var parsed fileConfig
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse config.toml: %w", err)
	}

	cfg := &Config{
		AppName:                  parsed.AppName,
		AWSProfile:               strings.ToLower(strings.TrimSpace(parsed.AWS.Profile)),
		AWSRegion:                parsed.AWS.Region,
		AWSAccount:               parsed.Signaling.AWSAccount,
		SignalingAppName:         parsed.Signaling.AppName,
		StackName:                parsed.Signaling.StackName,
		SignalingEndpoint:        parsed.Signaling.Endpoint,
		SignalingSocket:          parsed.Signaling.Socket,
		ApiKey:                   parsed.Signaling.ApiKey,
		LambdaFunctionNameActual: parsed.Signaling.LambdaFunctionName,
	}

	// Validate required fields
	if cfg.AppName == "" {
		return nil, fmt.Errorf("app_name is required in config.toml")
	}

	// Set derived fields
	cfg.LambdaFunctionName = cfg.GetLambdaFunctionName()

	return cfg, nil
}

// LoadWithEnv reads and parses the config.toml file and overrides with environment variables
// Environment variable overrides:
//
//	AWS_PROFILE -> aws_profile
//	AWS_REGION -> aws_region
//	AWS_ACCOUNT -> aws_account
//	SIGNALING_ENDPOINT -> signaling_endpoint
//	LAMBDA_FUNCTION_NAME -> lambda_function_name
func LoadWithEnv() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	// Override with environment variables if present
	if awsProfile := os.Getenv("AWS_PROFILE"); awsProfile != "" {
		cfg.AWSProfile = strings.ToLower(strings.TrimSpace(awsProfile))
	}
	if awsRegion := os.Getenv("AWS_REGION"); awsRegion != "" {
		cfg.AWSRegion = strings.TrimSpace(awsRegion)
	}
	if awsAccount := os.Getenv("AWS_ACCOUNT"); awsAccount != "" {
		cfg.AWSAccount = strings.TrimSpace(awsAccount)
	}
	if signalingEndpoint := os.Getenv("SIGNALING_ENDPOINT"); signalingEndpoint != "" {
		cfg.SignalingEndpoint = strings.TrimSpace(signalingEndpoint)
	}
	if signalingSocket := os.Getenv("SIGNALING_SOCKET"); signalingSocket != "" {
		cfg.SignalingSocket = strings.TrimSpace(signalingSocket)
	}
	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		cfg.ApiKey = strings.TrimSpace(apiKey)
	}
	if lambdaName := os.Getenv("LAMBDA_FUNCTION_NAME"); lambdaName != "" {
		cfg.LambdaFunctionNameActual = strings.TrimSpace(lambdaName)
	}

	return cfg, nil
}

// GetDefaultConfig returns the loaded configuration with environment overrides or panics if it fails
// This is a convenience function for simple applications
func GetDefaultConfig() *Config {
	cfg, err := LoadWithEnv()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}
	return cfg
}
