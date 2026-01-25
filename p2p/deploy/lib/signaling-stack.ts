import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigw2 from 'aws-cdk-lib/aws-apigatewayv2';
import { WebSocketLambdaIntegration } from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as path from 'path';
import { Construct } from 'constructs';

export class SignalingStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Lambda for signaling
    const signalingLambda = new lambda.Function(this, 'SignalingRelay', {
      runtime: lambda.Runtime.PROVIDED_AL2023,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../signaling_lambda'), {
        bundling: {
          image: lambda.Runtime.GO_1_X.bundlingImage,
          command: [
            'bash', '-c',
            'GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /asset-output/bootstrap main.go'
          ],
          user: 'root',
        },
      }),
      environment: {
        LAPTOP_ID: 'init',
      },
    });

    // WebSocket API
    const webSocketApi = new apigw2.WebSocketApi(this, 'SignalingApi', {
      routeSelectionExpression: '$request.body.action',
    });

    const integration = new WebSocketLambdaIntegration('SignalingIntegration', signalingLambda);

    // Add routes
    webSocketApi.addRoute('$connect', { integration });
    webSocketApi.addRoute('$disconnect', { integration });
    webSocketApi.addRoute('sendSignal', { integration });

    const stage = new apigw2.WebSocketStage(this, 'ProdStage', {
      webSocketApi,
      stageName: 'prod',
      autoDeploy: true,
    });

    // Permissions for Lambda to post messages back to connections
    signalingLambda.addToRolePolicy(new cdk.aws_iam.PolicyStatement({
      actions: ['execute-api:ManageConnections'],
      resources: [this.formatArn({
        service: 'execute-api',
        resource: webSocketApi.apiId,
        resourceName: `${stage.stageName}/POST/@connections/*`,
      })],
    }));

    new cdk.CfnOutput(this, 'WebSocketURL', {
      value: stage.callbackUrl,
    });
  }
}
