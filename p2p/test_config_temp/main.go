package main

import (
    "fmt"
    "p2p_bridge/config"
)

func main() {
    fmt.Println("=== Configuration Test ===")
    fmt.Println()
    
    cfg, err := config.Load()
    if err != nil {
        fmt.Printf("❌ Failed to load configuration: %v\n", err)
        return
    }
    
    fmt.Println("Raw Configuration:")
    fmt.Printf("  APP_NAME:              %s\n", cfg.AppName)
    fmt.Printf("  SIGNALING_ENDPOINT:    %s\n", cfg.SignalingEndpoint)
    fmt.Printf("  LAMBDA_FUNCTION_NAME: %s\n", cfg.LambdaFunctionNameActual)
    fmt.Printf("  AWS_PROFILE:           %s\n", cfg.AWSProfile)
    fmt.Printf("  AWS_REGION:            %s\n", cfg.AWSRegion)
    fmt.Println()
    
    fmt.Println("Derived Values:")
    fmt.Printf("  Stack Name:            %s\n", cfg.GetStackName())
    fmt.Printf("  Lambda Function:        %s\n", cfg.GetLambdaFunctionName())
    fmt.Println()
    
    fmt.Println("Validation:")
    allValid := true
    
    if cfg.AppName == "" {
        fmt.Println("❌ APP_NAME is empty (required)")
        allValid = false
    } else {
        fmt.Printf("✅ APP_NAME: %s\n", cfg.AppName)
    }
    
    if cfg.SignalingEndpoint == "" {
        fmt.Println("⚠️  SIGNALING_ENDPOINT not set (needed for home lab server)")
    } else {
        fmt.Printf("✅ SIGNALING_ENDPOINT: %s\n", cfg.SignalingEndpoint)
    }
    
    if cfg.GetLambdaFunctionName() == "" {
        fmt.Println("❌ Lambda function name is empty")
        allValid = false
    } else {
        fmt.Printf("✅ Lambda Function: %s\n", cfg.GetLambdaFunctionName())
    }
    
    if cfg.AWSProfile == "" {
        fmt.Println("⚠️  AWS_PROFILE not set (will use default)")
    } else {
        fmt.Printf("✅ AWS_PROFILE: %s\n", cfg.AWSProfile)
    }
    
    fmt.Println()
    
    if allValid {
        fmt.Println("✅ Configuration is valid!")
    } else {
        fmt.Println("❌ Configuration has errors")
    }
}
