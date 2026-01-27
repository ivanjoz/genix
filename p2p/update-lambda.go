package main

import (
	"archive/zip"
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	appconfig "p2p_bridge/config"
)

func main() {
	log.Println("=========================================")
	log.Println("  Lambda Code Update Script")
	log.Println("=========================================")
	log.Println("")

	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := appconfig.LoadWithEnv()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}
	log.Println("✅ Configuration loaded")
	log.Printf("   AWS Profile: %s\n", cfg.AWSProfile)
	log.Printf("   AWS Region: %s\n", cfg.AWSRegion)
	log.Printf("   Lambda Function: %s\n", cfg.GetLambdaFunctionName())
	log.Println("")

	// Get the function name
	functionName := cfg.GetLambdaFunctionName()
	if functionName == "" {
		log.Fatal("❌ Lambda function name is empty. Check your configuration.")
	}

	// Build the Lambda binary first
	log.Println("Building Lambda binary...")
	buildCmd := exec.Command("sh", "build-lambda.sh")
	buildCmd.Dir = "."
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		log.Fatalf("❌ Failed to build Lambda binary: %v\nOutput: %s", err, string(buildOutput))
	}
	log.Println("✅ Lambda binary built successfully")
	log.Println("")

	// Read the Lambda binary that was just built
	binaryData, err := os.ReadFile("signaling_lambda/bootstrap")
	if err != nil {
		log.Fatalf("❌ Failed to read Lambda binary: %v", err)
	}
	log.Printf("✅ Lambda binary loaded (%d bytes)\n", len(binaryData))
	log.Println("")

	// Create a zip buffer
	log.Println("Creating zip file...")
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	// Add the binary to the zip file as "bootstrap"
	zipEntry, err := zipWriter.Create("bootstrap")
	if err != nil {
		log.Fatalf("❌ Failed to create zip entry: %v", err)
	}

	// Write the binary data to the zip entry
	_, err = zipEntry.Write(binaryData)
	if err != nil {
		log.Fatalf("❌ Failed to write binary to zip: %v", err)
	}

	// Close the zip writer to flush all data
	if err := zipWriter.Close(); err != nil {
		log.Fatalf("❌ Failed to close zip writer: %v", err)
	}

	log.Printf("✅ Zip file created (%d bytes)\n", zipBuffer.Len())
	log.Println("")

	// Load AWS configuration
	log.Println("Loading AWS SDK configuration...")
	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("❌ Failed to load AWS configuration: %v", err)
	}
	log.Println("✅ AWS SDK configuration loaded")
	log.Println("")

	// Create Lambda client
	log.Println("Creating Lambda client...")
	lambdaClient := lambda.NewFromConfig(awsCfg)
	log.Println("✅ Lambda client created")
	log.Println("")

	// Update function code
	log.Printf("Updating Lambda function: %s\n", functionName)
	log.Println("This may take a moment...")

	updateInput := &lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(functionName),
		ZipFile:      zipBuffer.Bytes(),
	}

	result, err := lambdaClient.UpdateFunctionCode(context.TODO(), updateInput)
	if err != nil {
		log.Fatalf("❌ Failed to update Lambda function code: %v", err)
	}

	log.Println("")
	log.Println("=========================================")
	log.Println("✅ Lambda function updated successfully!")
	log.Println("=========================================")
	log.Printf("Function Name: %s\n", *result.FunctionName)
	log.Printf("Function ARN: %s\n", *result.FunctionArn)
	log.Printf("Last Modified: %s\n", *result.LastModified)
	log.Printf("Code Size: %d bytes\n", result.CodeSize)
	log.Printf("Version: %s\n", *result.Version)
	log.Printf("State: %s\n", result.State)
	if result.StateReason != nil {
		log.Printf("State Reason: %s\n", *result.StateReason)
	}
	log.Println("")
	log.Println("💡 Note: It may take a few seconds for the update to complete.")
	log.Println("   You can check the status with:")
	log.Printf("   aws lambda get-function --function-name %s\n", functionName)
	log.Println("   Or check CloudWatch logs for debug information.")
	log.Println("")

	// Optionally wait for the update to complete
	if len(os.Args) > 1 && os.Args[1] == "--wait" {
		log.Println("Waiting for update to complete...")
		waiter := lambda.NewFunctionUpdatedV2Waiter(lambdaClient)
		waitInput := &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		}

		err := waiter.Wait(context.TODO(), waitInput, 5*60) // Wait up to 5 minutes
		if err != nil {
			log.Printf("⚠️  Warning: Error waiting for update to complete: %v\n", err)
		} else {
			log.Println("✅ Update completed!")
		}
	}
}
