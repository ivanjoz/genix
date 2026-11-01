//go:build !js

package core

import (
	"context"
	"os"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// GetAwsConfig is server-only: aws-sdk-go-v2/config links the whole credential chain (sts, sso,
// signin), which the browser build has no use for.
func GetAwsConfig() aws_sdk.Config {
	var cfg aws_sdk.Config
	var err error

	setConfig := func(lo *config.LoadOptions) error {
		lo.Region = Env.AWS_REGION
		return nil
	}

	accessKeyEnv := os.Getenv("AWS_ACCESS_KEY_ID")
	if len(accessKeyEnv) > 0 {
		Log("Generando AWS Config con ACCESS_KEY en Región:", Env.AWS_REGION)
		cfg, err = config.LoadDefaultConfig(context.TODO(), setConfig)
	} else {
		Log("Generando AWS Config con profile:", Env.AWS_PROFILE, "|", Env.AWS_REGION)
		cfg, err = config.LoadDefaultConfig(
			context.TODO(), config.WithSharedConfigProfile(Env.AWS_PROFILE), setConfig)
	}
	if err != nil {
		panic(Concat(" ", "No se pudo obtener la configuración de AWS.", err))
	}
	return cfg
}
