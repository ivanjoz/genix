package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"p2p_bridge/signal"
)

const Version = "2.7.1" // Added comprehensive debug logging

// Helper function to log with context
func logInfo(message string) {
	fmt.Printf("[INFO] %s\n", message)
}

func logError(message string, err error) {
	fmt.Printf("[ERROR] %s: %v\n", message, err)
}

func logDebug(message string) {
	fmt.Printf("[DEBUG] %s\n", message)
}

func logTrace(message string) {
	fmt.Printf("[TRACE] %s\n", message)
}

func handler(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Log entire event structure for debugging
	eventJSON, _ := json.MarshalIndent(event, "", "  ")
	logTrace(fmt.Sprintf("Full Event:\n%s", string(eventJSON)))

	connectionID := event.RequestContext.ConnectionID
	routeKey := event.RequestContext.RouteKey

	// Log entry point
	logInfo("===== Lambda Handler Started =====")
	logInfo(fmt.Sprintf("Lambda Version: %s", Version))
	logDebug(fmt.Sprintf("ConnectionID: %s", connectionID))
	logDebug(fmt.Sprintf("RouteKey: %s", routeKey))
	logDebug(fmt.Sprintf("EventType: %s", event.RequestContext.EventType))
	logDebug(fmt.Sprintf("DomainName: %s", event.RequestContext.DomainName))
	logDebug(fmt.Sprintf("Stage: %s", event.RequestContext.Stage))
	logDebug(fmt.Sprintf("API ID: %s", event.RequestContext.APIID))
	logDebug(fmt.Sprintf("Request ID: %s", event.RequestContext.RequestID))

	// Log the full body for sendSignal events
	if routeKey == "sendSignal" {
		logInfo("===== sendSignal Route Detected =====")
		logDebug(fmt.Sprintf("Raw Body: %s", event.Body))
		logDebug(fmt.Sprintf("Body Length: %d bytes", len(event.Body)))
	} else {
		logInfo(fmt.Sprintf("Processing route: %s", routeKey))
	}

	// Load AWS config
	logInfo("Loading AWS config...")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logError("Failed to load AWS config", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("failed to load config: %w", err)
	}
	logInfo("AWS config loaded successfully")

	// Create API Gateway client
	callbackURL := fmt.Sprintf("https://%s/%s", event.RequestContext.DomainName, event.RequestContext.Stage)
	logDebug(fmt.Sprintf("Callback URL: %s", callbackURL))

	logInfo("Creating API Gateway Management API client...")
	apigw := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(callbackURL)
	})
	logInfo("API Gateway client created successfully")

	// Route handling
	logInfo(fmt.Sprintf("Processing route: %s", routeKey))

	switch routeKey {
	case "$connect":
		logInfo("===== Processing $connect Route =====")

		// Check LAPTOP_ID environment variable
		laptopID := os.Getenv("LAPTOP_ID")
		if laptopID != "" {
			logDebug(fmt.Sprintf("LAPTOP_ID environment variable is set: %s", laptopID))

			// Verify LAPTOP_ID connection is still alive
			logInfo("Verifying LAPTOP_ID connection is alive...")
			pingMsg := map[string]interface{}{
				"action": "ping",
				"from":   connectionID,
			}
			pingPayload, err := json.Marshal(pingMsg)
			if err != nil {
				logError("Failed to marshal ping message", err)
			} else {
				_, pingErr := apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
					ConnectionId: aws.String(laptopID),
					Data:         pingPayload,
				})
				if pingErr != nil {
					logError(fmt.Sprintf("LAPTOP_ID connection %s is not available (homelab_server not connected)", laptopID), pingErr)
					// Log but allow connection - frontend will handle failure when sending signals
				} else {
					logInfo("LAPTOP_ID connection verified as alive")
				}
			}
		} else {
			logDebug("LAPTOP_ID environment variable is NOT set (this is normal on initial connection)")
		}

		logInfo("Connection allowed, returning 200")
		logDebug("===== $connect Route Complete =====")
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil

	case "$disconnect":
		logInfo("===== Processing $disconnect Route =====")
		logDebug(fmt.Sprintf("Client disconnecting: %s", connectionID))
		logDebug("===== $disconnect Route Complete =====")
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil

	case "sendSignal":
		logInfo("===== Processing sendSignal Route =====")

		// Parse incoming message
		logInfo("Parsing incoming signal message")
		var msg signal.Msg
		if err := json.Unmarshal([]byte(event.Body), &msg); err != nil {
			logError("Failed to unmarshal message body", err)
			logDebug(fmt.Sprintf("Body that failed to parse: %s", event.Body))
			logDebug("===== sendSignal Route Failed (Parse Error) =====")
			return events.APIGatewayProxyResponse{StatusCode: 400}, fmt.Errorf("invalid message body: %w", err)
		}

		// Set the sender
		msg.From = connectionID
		logDebug(fmt.Sprintf("Message parsed - Action: %s", msg.Action))
		logDebug(fmt.Sprintf("Message parsed - From: %s", msg.From))
		logDebug(fmt.Sprintf("Message parsed - To: %s", msg.To))

		// Handle target resolution
		targetID := msg.To
		logDebug(fmt.Sprintf("Target ID from message: %s", targetID))

		if targetID == "laptop" {
			logInfo("Resolving 'laptop' target to LAPTOP_ID environment variable")
			targetID = os.Getenv("LAPTOP_ID")
			logDebug(fmt.Sprintf("Resolved target ID: %s", targetID))
		} else if targetID == "me" {
			logInfo("Resolving 'me' target to sender's connection ID")
			targetID = connectionID
			// Also change action to "connected" if it's an identification request
			// to keep compatibility with what homelab_server expects
			if msg.Data == "identify" {
				msg.Action = "connected"
				msg.Data = connectionID
			}
			logDebug(fmt.Sprintf("Resolved target ID: %s", targetID))
		}

		if targetID == "" {
			logError("Target ID is empty - LAPTOP_ID not set or target not specified", nil)
			logDebug("===== sendSignal Route Failed (Empty Target) =====")
			return events.APIGatewayProxyResponse{StatusCode: 404}, fmt.Errorf("target not found")
		}

		// Prepare payload
		logInfo(fmt.Sprintf("Sending signal to target connection: %s", targetID))
		payload, err := json.Marshal(msg)
		if err != nil {
			logError("Failed to marshal signal message", err)
			logDebug("===== sendSignal Route Failed (Marshal Error) =====")
			return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("failed to marshal signal: %w", err)
		}
		logDebug(fmt.Sprintf("Signal payload: %s", string(payload)))

		// Send to target
		logInfo("Calling PostToConnection...")
		result, err := apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(targetID),
			Data:         payload,
		})
		if err != nil {
			// Check if the error is due to connection not found (GoneException) or invalid connectionId
			errStr := err.Error()
			logTrace(fmt.Sprintf("PostToConnection error: %s", errStr))

			if strings.Contains(errStr, "GoneException") || strings.Contains(errStr, "Connection does not exist") || strings.Contains(errStr, "BadRequestException") || strings.Contains(errStr, "Invalid connectionId") || strings.Contains(errStr, "NotFoundException") {
				logError(fmt.Sprintf("Target connection %s is invalid or not available (homelab_server not connected)", targetID), err)

				// Send an error message back to the sender
				errorMsg := map[string]interface{}{
					"error":   "backend_unreachable",
					"message": "The target (homelab_server) is not currently connected. Please start the homelab_server to establish a WebRTC connection.",
					"target":  targetID,
				}
				errorPayload, _ := json.Marshal(errorMsg)
				logTrace(fmt.Sprintf("Sending error message to sender: %s", string(errorPayload)))

				_, sendErr := apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
					ConnectionId: aws.String(connectionID),
					Data:         errorPayload,
				})
				if sendErr != nil {
					logError("Failed to send error message back to sender", sendErr)
				} else {
					logInfo("Sent error notification to sender")
				}

				logDebug("===== sendSignal Route Failed (Target Unavailable) =====")
				return events.APIGatewayProxyResponse{StatusCode: 200}, nil
			}

			logError("Failed to post signal to target", err)
			logDebug(fmt.Sprintf("PostToConnection result: %+v", result))
			logDebug("===== sendSignal Route Failed (Unexpected Error) =====")
			return events.APIGatewayProxyResponse{StatusCode: 200}, nil
		}

		logInfo("Signal sent successfully")
		logDebug(fmt.Sprintf("PostToConnection result: %+v", result))
		logDebug("===== sendSignal Route Complete =====")
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil

	default:
		logError(fmt.Sprintf("Unknown route key: %s", routeKey), nil)
		logDebug("===== Lambda Handler Failed (Unknown Route) =====")
		return events.APIGatewayProxyResponse{StatusCode: 400}, fmt.Errorf("unknown route: %s", routeKey)
	}
}

func main() {
	logInfo("Lambda function starting...")
	lambda.Start(handler)
}
