package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/aws-cdk-go/awscdkapigatewayv2alpha/v2"
	"github.com/aws/aws-cdk-go/awscdkapigatewayv2integrationsalpha/v2"
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

	// Lambda for signaling
	currDir, _ := os.Getwd()
	// Since we are in 'deploy' directory, signaling_lambda is one level up
	lambdaSourcePath := filepath.Join(currDir, "..", "signaling_lambda")

	signalingLambda := awslambda.NewFunction(stack, jsii.String(cfg.GetLambdaFunctionName()), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2023(),
		Handler: jsii.String("bootstrap"),
		Code:     awslambda.Code_FromAsset(jsii.String(lambdaSourcePath), &awss3assets.AssetOptions{}),
		Environment: &map[string]*string{
			"LAPTOP_ID": jsii.String(""),
		},
	})

	// WebSocket API
	webSocketApi := awscdkapigatewayv2alpha.NewWebSocketApi(stack, jsii.String("SignalingApi"), &awscdkapigatewayv2alpha.WebSocketApiProps{
		RouteSelectionExpression: jsii.String("$request.body.action"),
	})

	integration := awscdkapigatewayv2integrationsalpha.NewWebSocketLambdaIntegration(jsii.String("SignalingIntegration"), signalingLambda)

	// Add routes
	webSocketApi.AddRoute(jsii.String("$connect"), &awscdkapigatewayv2alpha.WebSocketRouteOptions{
		Integration: integration,
	})
	webSocketApi.AddRoute(jsii.String("$disconnect"), &awscdkapigatewayv2alpha.WebSocketRouteOptions{
		Integration: integration,
	})
	webSocketApi.AddRoute(jsii.String("$default"), &awscdkapigatewayv2alpha.WebSocketRouteOptions{
		Integration: integration,
	})
	webSocketApi.AddRoute(jsii.String("sendSignal"), &awscdkapigatewayv2alpha.WebSocketRouteOptions{
		Integration: integration,
	})

	stage := awscdkapigatewayv2alpha.NewWebSocketStage(stack, jsii.String("ProdStage"), &awscdkapigatewayv2alpha.WebSocketStageProps{
		WebSocketApi: webSocketApi,
		StageName:    jsii.String("prod"),
		AutoDeploy:   jsii.Bool(true),
	})

	// Enable logging for the stage
	cfnStage := stage.Node().DefaultChild().(awscdkapigatewayv2alpha.CfnStage)
	cfnStage.SetDefaultRouteSettings(&awscdkapigatewayv2alpha.CfnStage_RouteSettingsProperty{
		DataTraceEnabled:       jsii.Bool(true),
		DetailedMetricsEnabled: jsii.Bool(true),
		LoggingLevel:           jsii.String("INFO"),
	})

	// Explicitly grant API Gateway permission to invoke the Lambda for all routes
	signalingLambda.AddPermission(jsii.String("AllowApiGatewayInvoke"), &awslambda.Permission{
		Principal: awsiam.NewServicePrincipal(jsii.String("apigateway.amazonaws.com"), nil),
		Action:    jsii.String("lambda:InvokeFunction"),
		SourceArn: jsii.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*", *stack.Region(), *stack.Account(), *webSocketApi.ApiId())),
	})

	// Permissions for Lambda to post messages back to connections
	resourceArn := stack.FormatArn(&awscdk.ArnComponents{
		Service:      jsii.String("execute-api"),
		Resource:     webSocketApi.ApiId(),
		ResourceName: jsii.String(fmt.Sprintf("%s/POST/@connections/*", *stage.StageName())),
	})

	signalingLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("execute-api:ManageConnections"),
		Resources: jsii.Strings(*resourceArn),
	}))

	awscdk.NewCfnOutput(stack, jsii.String("WebSocketConnectURL"), &awscdk.CfnOutputProps{
		Value: jsii.String(*webSocketApi.ApiEndpoint() + "/" + *stage.StageName()),
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
