package main

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type GenixStackProps struct {
	awscdk.StackProps
	Params DeployParams
}

func NewGenixStack(scope constructs.Construct, id string, props *GenixStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// S3 Bucket for Frontend
	frontendBucket := awss3.NewBucket(stack, jsii.String("FrontendBucket"), &awss3.BucketProps{
		BucketName:    jsii.String(props.Params.FRONTEND_BUCKET),
		RemovalPolicy: awscdk.RemovalPolicy_RETAIN,
		Cors: &[]*awss3.CorsRule{
			{
				AllowedOrigins: jsii.Strings("*"),
				AllowedHeaders: jsii.Strings("*"),
				AllowedMethods: &[]awss3.HttpMethods{
					awss3.HttpMethods_GET,
					awss3.HttpMethods_PUT,
					awss3.HttpMethods_POST,
					awss3.HttpMethods_DELETE,
					awss3.HttpMethods_HEAD,
				},
				MaxAge: jsii.Number(3000),
			},
		},
	})

	// CloudFront Origin Access Identity
	oai := awscloudfront.NewOriginAccessIdentity(stack, jsii.String("OAI"), &awscloudfront.OriginAccessIdentityProps{
		Comment: jsii.String("Genix Identity"),
	})

	frontendBucket.GrantRead(oai, jsii.String("*"))

	// CloudFront Distribution
	s3Origin := awscloudfrontorigins.NewS3Origin(frontendBucket, &awscloudfrontorigins.S3OriginProps{
		OriginAccessIdentity: oai,
	})

	distribution := awscloudfront.NewDistribution(stack, jsii.String("CloudfrontFrontend"), &awscloudfront.DistributionProps{
		Comment:            jsii.String("Frontend Genix"),
		DefaultRootObject:  jsii.String("index.html"),
		HttpVersion:        awscloudfront.HttpVersion_HTTP2_AND_3,
		EnableIpv6:         jsii.Bool(true),
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin:               s3Origin,
			ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			CachePolicy:          awscloudfront.CachePolicy_CACHING_OPTIMIZED(),
			Compress:             jsii.Bool(true),
		},
		ErrorResponses: &[]*awscloudfront.ErrorResponse{
			{
				HttpStatus:          jsii.Number(403),
				ResponsePagePath:   jsii.String("/index.html"),
				ResponseHttpStatus: jsii.Number(200),
				Ttl:                awscdk.Duration_Seconds(jsii.Number(10)),
			},
			{
				HttpStatus:          jsii.Number(404),
				ResponsePagePath:   jsii.String("/index.html"),
				ResponseHttpStatus: jsii.Number(200),
				Ttl:                awscdk.Duration_Seconds(jsii.Number(10)),
			},
		},
	})

	// Add specific behaviors
	noCachePolicy := awscloudfront.CachePolicy_CACHING_DISABLED()

	distribution.AddBehavior(jsii.String("*.html"), s3Origin, &awscloudfront.AddBehaviorOptions{
		AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_GET_HEAD(),
		ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
		CachePolicy:          noCachePolicy,
		Compress:             jsii.Bool(false),
	})

	behaviorWebManifest := &awscloudfront.AddBehaviorOptions{
		AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_GET_HEAD_OPTIONS(),
		ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_HTTPS_ONLY,
		CachePolicy:          noCachePolicy,
		Compress:             jsii.Bool(false),
	}

	distribution.AddBehavior(jsii.String("*.webmanifest"), s3Origin, behaviorWebManifest)
	distribution.AddBehavior(jsii.String("sw.js"), s3Origin, behaviorWebManifest)
	distribution.AddBehavior(jsii.String("registerSW.js"), s3Origin, behaviorWebManifest)

	// Lambda Role
	lambdaRole := awsiam.Role_FromRoleArn(stack, jsii.String("LambdaRole"), jsii.String(props.Params.LAMBDA_IAM_ROLE), nil)

	deploymentBucket := awss3.Bucket_FromBucketName(stack, jsii.String("DeploymentBucket"), jsii.String(props.Params.DEPLOYMENT_BUCKET))

	lambdaName := props.Params.STACK_NAME + "-backend"
	s3CompiledPath := props.Params.S3_COMPILED_PATH

	// Lambda Function 1
	lambdaGO := awslambda.NewFunction(stack, jsii.String("LambdaGO"), &awslambda.FunctionProps{
		FunctionName: jsii.String(lambdaName),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Handler:      jsii.String("index.handler"),
		Code:         awslambda.Code_FromBucket(deploymentBucket, jsii.String(s3CompiledPath), nil),
		MemorySize:   jsii.Number(192),
		Timeout:      awscdk.Duration_Minutes(jsii.Number(5)),
		Architecture: awslambda.Architecture_ARM_64(),
		Role:         lambdaRole,
		Environment: &map[string]*string{
			"APP_CODE": jsii.String("gerp-prd"),
		},
	})

	// Log Group for Lambda 1
	awslogs.NewLogGroup(stack, jsii.String("LambdaGOLogGroup"), &awslogs.LogGroupProps{
		LogGroupName: jsii.String(fmt.Sprintf("/aws/lambda/%s", lambdaName)),
		Retention:    awslogs.RetentionDays_ONE_MONTH,
	})

	// Function URL 1
	lambdaGO.AddFunctionUrl(&awslambda.FunctionUrlOptions{
		AuthType: awslambda.FunctionUrlAuthType_NONE,
		Cors: &awslambda.FunctionUrlCorsOptions{
			AllowedOrigins: jsii.Strings("*"),
			AllowedHeaders: jsii.Strings("*"),
			AllowedMethods: &[]awslambda.HttpMethod{
				awslambda.HttpMethod_GET,
				awslambda.HttpMethod_POST,
				awslambda.HttpMethod_HEAD,
				awslambda.HttpMethod_PUT,
			},
		},
	})

	// EventBridge Rule
	awsevents.NewRule(stack, jsii.String("EventBridgeRule"), &awsevents.RuleProps{
		Description: jsii.String("SmartBerry GO | Cron Job Event 10 min"),
		Schedule:    awsevents.Schedule_Cron(&awsevents.CronOptions{Minute: jsii.String("*/10")}),
		RuleName:    jsii.String(fmt.Sprintf("%s_cron_lambda_every_10_min_go", lambdaName)),
		Targets: &[]awsevents.IRuleTarget{
			awseventstargets.NewLambdaFunction(lambdaGO, &awseventstargets.LambdaFunctionProps{
				Event: awsevents.RuleTargetInput_FromObject(map[string]interface{}{
					"body": "exec:cron",
				}),
			}),
		},
	})

	// DynamoDB Table
	tableName := props.Params.STACK_NAME + "-db"
	mainTable := awsdynamodb.NewTable(stack, jsii.String("MainTable"), &awsdynamodb.TableProps{
		TableName:           jsii.String(tableName),
		PartitionKey:        &awsdynamodb.Attribute{Name: jsii.String("pk"), Type: awsdynamodb.AttributeType_STRING},
		SortKey:             &awsdynamodb.Attribute{Name: jsii.String("sk"), Type: awsdynamodb.AttributeType_STRING},
		BillingMode:         awsdynamodb.BillingMode_PAY_PER_REQUEST,
		TimeToLiveAttribute: jsii.String("ttl"),
		RemovalPolicy:       awscdk.RemovalPolicy_RETAIN,
	})

	// GSIs
	addGsi := func(name string, type_ awsdynamodb.AttributeType) {
		mainTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
			IndexName:      jsii.String(name),
			PartitionKey:   &awsdynamodb.Attribute{Name: jsii.String("pk"), Type: awsdynamodb.AttributeType_STRING},
			SortKey:        &awsdynamodb.Attribute{Name: jsii.String(name), Type: type_},
			ProjectionType: awsdynamodb.ProjectionType_ALL,
		})
	}

	addGsi("ix1", awsdynamodb.AttributeType_STRING)
	addGsi("ix2", awsdynamodb.AttributeType_STRING)
	addGsi("ix3", awsdynamodb.AttributeType_STRING)
	addGsi("ix4", awsdynamodb.AttributeType_STRING)
	addGsi("in5", awsdynamodb.AttributeType_NUMBER)

	// Lambda Function 2 (Test)
	lambdaName2 := lambdaName + "_2"
	lambdaGOn2 := awslambda.NewFunction(stack, jsii.String("LambdaGOn2"), &awslambda.FunctionProps{
		FunctionName: jsii.String(lambdaName2),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Handler:      jsii.String("index.handler"),
		Code:         awslambda.Code_FromBucket(deploymentBucket, jsii.String(s3CompiledPath), nil),
		MemorySize:   jsii.Number(2048),
		Timeout:      awscdk.Duration_Minutes(jsii.Number(5)),
		Architecture: awslambda.Architecture_ARM_64(),
		Role:         lambdaRole,
		Environment: &map[string]*string{
			"APP_CODE": jsii.String("gerp-prd"),
		},
	})

	awslogs.NewLogGroup(stack, jsii.String("LambdaGOn2LogGroup"), &awslogs.LogGroupProps{
		LogGroupName: jsii.String(fmt.Sprintf("/aws/lambda/%s", lambdaName2)),
		Retention:    awslogs.RetentionDays_ONE_MONTH,
	})

	lambdaGOn2.AddFunctionUrl(&awslambda.FunctionUrlOptions{
		AuthType: awslambda.FunctionUrlAuthType_NONE,
		Cors: &awslambda.FunctionUrlCorsOptions{
			AllowedOrigins: jsii.Strings("*"),
			AllowedHeaders: jsii.Strings("*"),
			AllowedMethods: &[]awslambda.HttpMethod{
				awslambda.HttpMethod_GET,
				awslambda.HttpMethod_POST,
				awslambda.HttpMethod_HEAD,
				awslambda.HttpMethod_PUT,
			},
		},
	})

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("WebsiteURL"), &awscdk.CfnOutputProps{
		Value: frontendBucket.BucketWebsiteUrl(),
	})

	return stack
}

func StartCDK(params DeployParams) {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewGenixStack(app, params.STACK_NAME+"-stack", &GenixStackProps{
		awscdk.StackProps{
			Env: &awscdk.Environment{
				Region: jsii.String(params.AWS_REGION),
			},
		},
		params,
	})

	app.Synth(nil)
}
