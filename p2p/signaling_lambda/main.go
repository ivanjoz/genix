package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"p2p_bridge/signal"
)

func handler(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := event.RequestContext.ConnectionID
	routeKey := event.RequestContext.RouteKey

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	callbackURL := fmt.Sprintf("https://%s/%s", event.RequestContext.DomainName, event.RequestContext.Stage)
	apigw := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(callbackURL)
	})

	switch routeKey {
	case "$connect":
		// Inform the client of their connection ID
		msg := signal.Msg{
			Action: "connected",
			Data:   connectionID,
		}
		payload, _ := json.Marshal(msg)
		apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(connectionID),
			Data:         payload,
		})
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil

	case "$disconnect":
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil

	case "sendSignal":
		var msg signal.Msg
		if err := json.Unmarshal([]byte(event.Body), &msg); err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 400}, nil
		}
		msg.From = connectionID

		targetID := msg.To
		if targetID == "laptop" {
			targetID = os.Getenv("LAPTOP_ID")
		}

		if targetID == "" {
			return events.APIGatewayProxyResponse{StatusCode: 404}, nil
		}

		payload, _ := json.Marshal(msg)
		_, err = apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(targetID),
			Data:         payload,
		})
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, nil
		}
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil

	default:
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}
}

func main() {
	lambda.Start(handler)
}
