# Config Package

This package provides centralized configuration management for the P2P Bridge application. It reads configuration from a `config.toml` file and supports environment variable overrides.

## Configuration File

Set these values in the project root `config.toml`:

```toml
app_name = "p2p-bridge"

[aws]
profile = "default"
region  = "us-east-1"

[signaling]
app_name    = ""
stack_name  = ""
aws_account = ""
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `aws.profile` | string | No | AWS profile name from `~/.aws/credentials`. Case-insensitive. Uses default AWS profile if not set. |
| `app_name` | string | **Yes** | Base application identifier used to generate other names. |
| `signaling.app_name` | string | No | Specific name for the signaling app. Defaults to `app_name + "-signaling"` if empty. |
| `signaling.stack_name` | string | No | CDK stack name for CloudFormation. Defaults to `app_name + "-signaling"` if empty. |
| `aws.region` | string | No | AWS region to deploy to. Empty uses default from AWS profile. |
| `signaling.aws_account` | string | No | AWS account ID. Empty uses default from AWS profile. Not required if `aws.profile` is set. |

### Derived Values

The following values are automatically computed and not stored in the config file:

- **`lambda_function_name`**: Always set to `app_name + "-signaling"`

### Environment Variable Overrides

You can override configuration values using environment variables (all case-insensitive):

- `AWS_PROFILE` → overrides `aws.profile`
- `AWS_REGION` → overrides `aws.region`
- `AWS_ACCOUNT` → overrides `signaling.aws_account`

## Usage

### Loading Configuration

```go
import "p2p_bridge/config"

// Load configuration with error handling
cfg, err := config.Load()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}
```

### Loading with Environment Overrides

```go
import "p2p_bridge/config"

// Load with environment variable overrides
cfg, err := config.LoadWithEnv()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}

// Values can be overridden by AWS_PROFILE, AWS_REGION, AWS_ACCOUNT env vars
profile := cfg.AWSProfile
region := cfg.AWSRegion
```

### Quick Loading (Panics on Error)

For simple applications where you want to fail fast if configuration is invalid:

```go
import "p2p_bridge/config"

cfg := config.GetDefaultConfig()

// Access configuration values
appName := cfg.AppName
```

### Accessing Derived Values

```go
cfg := config.GetDefaultConfig()

// Get signaling app name (defaults to app_name + "-signaling")
signalingAppName := cfg.GetSignalingAppName()

// Get stack name (defaults to app_name + "-signaling")
stackName := cfg.GetStackName()

// Get lambda function name (always app_name + "-signaling")
lambdaName := cfg.GetLambdaFunctionName()
```

## Examples

### Using with AWS CDK

See `deploy/deploy.go` for a complete example:

```go
func main() {
    defer jsii.Close()
    
    // Load configuration with environment overrides
    cfg := config.GetDefaultConfig()
    
    // Set AWS_PROFILE environment variable for AWS SDK and CDK
    if cfg.AWSProfile != "" {
        os.Setenv("AWS_PROFILE", cfg.AWSProfile)
    }
    
    app := awscdk.NewApp(nil)
    
    // Use configuration values
    NewDeployStack(app, cfg.GetStackName(), &DeployStackProps{
        awscdk.StackProps{
            Env: env(cfg),
        },
    }, cfg)
    
    app.Synth(nil)
}
```

### Using with Home Lab Server

```go
func updateLambdaConfig(cfg *config.Config, connectionID string) {
    ctx := context.TODO()
    awsCfg, _ := awssdkconfig.LoadDefaultConfig(ctx, func(o *awssdkconfig.LoadOptions) {
        if cfg.AWSRegion != "" {
            o.Region = cfg.AWSRegion
        }
        if cfg.AWSProfile != "" {
            o.SharedConfigProfile = cfg.AWSProfile
        }
    })
    svc := lambda.NewFromConfig(awsCfg)
    
    // Use derived lambda function name
    lambdaName := cfg.GetLambdaFunctionName()
    res, _ := svc.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
        FunctionName: &lambdaName,
    })
}
```

## Security

**Important:** Never commit `config.toml` to version control. Use `config.example.toml` as a template and add `config.toml` to your `.gitignore` file.

## Auto-Discovery

The config loader automatically searches for `config.toml` in:

1. Current working directory
2. Parent directories (up to the root)

This means you can run commands from subdirectories and still find the config file.

## Validation

The configuration loader validates that:
- `app_name` is present and non-empty
- All derived values can be computed

If validation fails, an error is returned describing which field is missing or invalid.

## Case Sensitivity

- `aws.profile`: Case-insensitive (normalized to lowercase)
- Other string fields: Case-sensitive as written

## AWS Profile Behavior

When `aws.profile` is set:
- The package sets the `AWS_PROFILE` environment variable
- AWS SDK and CDK automatically use this profile
- `signaling.aws_account` is optional (derived from profile)
- `aws.region` is optional (uses profile default if not set)

When `aws.profile` is not set:
- Uses default AWS profile from `~/.aws/config`
- You may need to specify `signaling.aws_account` and `aws.region` explicitly