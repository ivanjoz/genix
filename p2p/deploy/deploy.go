package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsappsync"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"p2p_bridge/config"
)

type DeployStackProps struct {
	awscdk.StackProps
}

func NewDeployStack(scope constructs.Construct, id string, props *DeployStackProps, cfg *config.Config) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// AppSync Events API for signaling
	api := awsappsync.NewEventApi(stack, jsii.String("SignalingEventApi"), &awsappsync.EventApiProps{
		ApiName: jsii.String(cfg.GetStackName() + "-events-api"),
		AuthorizationConfig: &awsappsync.EventApiAuthConfig{
			AuthProviders: &[]*awsappsync.AppSyncAuthProvider{
				&awsappsync.AppSyncAuthProvider{
					AuthorizationType: awsappsync.AppSyncAuthorizationType_API_KEY,
				},
			},
			ConnectionAuthModeTypes: &[]awsappsync.AppSyncAuthorizationType{
				awsappsync.AppSyncAuthorizationType_API_KEY,
			},
			DefaultPublishAuthModeTypes: &[]awsappsync.AppSyncAuthorizationType{
				awsappsync.AppSyncAuthorizationType_API_KEY,
			},
			DefaultSubscribeAuthModeTypes: &[]awsappsync.AppSyncAuthorizationType{
				awsappsync.AppSyncAuthorizationType_API_KEY,
			},
		},
	})

	// Add channel namespace for P2P signaling
	api.AddChannelNamespace(jsii.String("genix-bridge"), nil)

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("WebSocketUrl"), &awscdk.CfnOutputProps{
		Value: jsii.String("wss://" + *api.RealtimeDns()),
	})

	awscdk.NewCfnOutput(stack, jsii.String("ApiKey"), &awscdk.CfnOutputProps{
		Value: (*api.ApiKeys())["Default"].AttrApiKey(),
	})

	return stack
}

func main() {
	defer jsii.Close()

	// Load configuration from credentials.json
	cfg := config.GetDefaultConfig()

	// Set AWS_PROFILE environment variable for AWS SDK and CDK
	if cfg.AWSProfile != "" {
		os.Setenv("AWS_PROFILE", cfg.AWSProfile)
	}

	app := awscdk.NewApp(nil)

	NewDeployStack(app, cfg.GetStackName(), &DeployStackProps{
		awscdk.StackProps{
			Env: env(cfg),
		},
	}, cfg)

	app.Synth(nil)
}

func env(cfg *config.Config) *awscdk.Environment {
	account := cfg.AWSAccount
	if account == "" {
		account = os.Getenv("CDK_DEFAULT_ACCOUNT")
	}
	region := cfg.AWSRegion
	if region == "" {
		region = os.Getenv("CDK_DEFAULT_REGION")
	}

	// If account or region is not specified, return nil to let CDK auto-discover
	if account == "" || region == "" {
		return nil
	}

	return &awscdk.Environment{
		Account: jsii.String(account),
		Region:  jsii.String(region),
	}
}
