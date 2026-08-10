package main

import (
	"reflect"
	"testing"
)

// El comando debe coincidir con BackendLogGroup y usar las credenciales del entorno elegido.
func TestCloudWatchLogsArguments(t *testing.T) {
	var config cloudWatchLogsConfig
	config.AppName = "genix-test"
	config.AWS.Profile = "developer"
	config.AWS.Region = "us-west-2"

	got := cloudWatchLogsArguments(config)
	want := []string{
		"--profile", "developer",
		"--region", "us-west-2",
		"logs", "tail", "/aws/lambda/genix-test-backend",
		"--follow",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argumentos AWS incorrectos:\n got: %v\nwant: %v", got, want)
	}
}
