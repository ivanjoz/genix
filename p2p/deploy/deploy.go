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

	// AppSync GraphQL API for signaling
	api := awsappsync.NewGraphqlApi(stack, jsii.String("SignalingApi"), &awsappsync.GraphqlApiProps{
		Name: jsii.String(cfg.GetStackName() + "-api"),
		Definition: awsappsync.Definition_FromSchema(awsappsync.SchemaFile_FromAsset(jsii.String("schema.graphql"))),
		AuthorizationConfig: &awsappsync.AuthorizationConfig{
			DefaultAuthorization: &awsappsync.AuthorizationMode{
				AuthorizationType: awsappsync.AuthorizationType_API_KEY,
			},
		},
	})

	// Local Data Source (None) for pub/sub signaling
	dataSource := api.AddNoneDataSource(jsii.String("NoneDataSource"), &awsappsync.DataSourceOptions{
		Name: jsii.String("NoneDataSource"),
	})

	// Mutation resolver for sendSignal
	dataSource.CreateResolver(jsii.String("SendSignalResolver"), &awsappsync.BaseResolverProps{
		TypeName: jsii.String("Mutation"),
		FieldName: jsii.String("sendSignal"),
		RequestMappingTemplate: awsappsync.MappingTemplate_FromString(jsii.String(`{
			"version": "2018-05-29",
			"payload": {
				"from": "$context.arguments.from",
				"to": "$context.arguments.to",
				"action": "$context.arguments.action",
				"data": "$context.arguments.data"
			}
		}`)),
		ResponseMappingTemplate: awsappsync.MappingTemplate_FromString(jsii.String(`$util.toJson($context.result)`)),
	})

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("GraphQLUrl"), &awscdk.CfnOutputProps{
		Value: api.GraphqlUrl(),
	})

	awscdk.NewCfnOutput(stack, jsii.String("ApiKey"), &awscdk.CfnOutputProps{
		Value: api.ApiKey(),
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
	return &awscdk.Environment{
		Account: jsii.String(account),
		Region:  jsii.String(region),
	}
}