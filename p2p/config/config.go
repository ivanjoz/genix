package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all configuration parameters for the P2P Bridge application
// It supports both uppercase and lowercase field names for backward compatibility
type Config struct {
	AWSProfile       string `json:"AWS_PROFILE,omitempty"`
	AppName          string `json:"APP_NAME,omitempty"`
	SignalingAppName string `json:"SIGNALING_APP_NAME,omitempty"`
	StackName        string `json:"SIGNALING_STACK_NAME,omitempty"`
	WebSocketURL     string `json:"WEBSOCKET_URL,omitempty"`
	LambdaFunctionName string `json:"-"` // Not in JSON, derived from app_name
	LambdaFunctionNameActual string `json:"LAMBDA_FUNCTION_NAME,omitempty"` // Actual Lambda function name from CDK output
	AWSRegion        string `json:"AWS_REGION,omitempty"`
	AWSAccount       string `json:"AWS_ACCOUNT,omitempty"`
}

// rawConfig is used for unmarshaling to support both uppercase and lowercase field names
type rawConfig map[string]interface{}

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

// normalizeKey converts field names to a consistent format for lookup
func normalizeKey(key string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
}

// getValueFromRaw extracts a value from rawConfig trying multiple key formats
func getValueFromRaw(raw rawConfig, keys []string) (string, bool) {
	for _, key := range keys {
		normalizedKey := normalizeKey(key)
		for k, v := range raw {
			if normalizeKey(k) == normalizedKey {
				if str, ok := v.(string); ok {
					return strings.TrimSpace(str), true
				}
			}
		}
	}
	return "", false
}

// configPath returns the path to credentials.json relative to the project root
func configPath() (string, error) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Look for credentials.json in current directory or parent directories
	for {
		configFile := filepath.Join(cwd, "credentials.json")
		if _, err := os.Stat(configFile); err == nil {
			return configFile, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// Reached the root directory without finding credentials.json
			return "", fmt.Errorf("credentials.json not found in current directory or parent directories")
		}
		cwd = parent
	}
}

// Load reads and parses the credentials.json file
func Load() (*Config, error) {
	configFile, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials.json: %w", err)
	}

	// First try to unmarshal as a generic map to handle both uppercase and lowercase keys
	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse credentials.json: %w", err)
	}

	cfg := &Config{}

	// Try to get AWS_PROFILE from multiple possible key names
	if val, found := getValueFromRaw(raw, []string{"AWS_PROFILE", "aws_profile", "profile"}); found {
		cfg.AWSProfile = strings.ToLower(val)
	}

	// Try to get APP_NAME from multiple possible key names
	if val, found := getValueFromRaw(raw, []string{"APP_NAME", "app_name"}); found {
		cfg.AppName = val
	}

	// Try to get SIGNALING_APP_NAME from multiple possible key names
	if val, found := getValueFromRaw(raw, []string{"SIGNALING_APP_NAME", "signaling_app_name"}); found {
		cfg.SignalingAppName = val
	}

	// Try to get SIGNALING_STACK_NAME from multiple possible key names
	if val, found := getValueFromRaw(raw, []string{"SIGNALING_STACK_NAME", "stack_name"}); found {
		cfg.StackName = val
	}

	// Try to get AWS_REGION from multiple possible key names
	if val, found := getValueFromRaw(raw, []string{"AWS_REGION", "aws_region", "region"}); found {
		cfg.AWSRegion = val
	}

	// Try to get AWS_ACCOUNT from multiple possible key names
	if val, found := getValueFromRaw(raw, []string{"AWS_ACCOUNT", "aws_account", "account"}); found {
		cfg.AWSAccount = val
	}

	// Try to get WEBSOCKET_URL from multiple possible key names
	if val, found := getValueFromRaw(raw, []string{"WEBSOCKET_URL", "websocket_url", "ws_url"}); found {
		cfg.WebSocketURL = val
	}

	// Try to get LAMBDA_FUNCTION_NAME from multiple possible key names
	if val, found := getValueFromRaw(raw, []string{"LAMBDA_FUNCTION_NAME", "lambda_function_name", "lambda_name"}); found {
		cfg.LambdaFunctionNameActual = val
	}

	// Validate required fields
	if cfg.AppName == "" {
		return nil, fmt.Errorf("APP_NAME is required in credentials.json")
	}

	// Set derived fields
	cfg.LambdaFunctionName = cfg.GetLambdaFunctionName()

	return cfg, nil
}

// LoadWithEnv reads and parses the credentials.json file and overrides with environment variables
// Environment variable overrides:
//   AWS_PROFILE -> aws_profile
//   AWS_REGION -> aws_region
//   AWS_ACCOUNT -> aws_account
//   WEBSOCKET_URL -> websocket_url
//   LAMBDA_FUNCTION_NAME -> lambda_function_name
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
	if wsURL := os.Getenv("WEBSOCKET_URL"); wsURL != "" {
		cfg.WebSocketURL = strings.TrimSpace(wsURL)
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
