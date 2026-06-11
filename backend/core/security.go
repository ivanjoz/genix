package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// ParseGenixSearchURL accepts the GENIXSEARCH_URL credential in any of:
//   - "host:port"
//   - "scheme://host:port"     (scheme is ignored)
//   - "scheme://host:port/path" (path is ignored)
//   - ""                       → 127.0.0.1:14446
//
// The scheme is informational only; the text_search client always opens
// a raw TCP socket. A bare host with no port keeps the default port.
func ParseGenixSearchURL(raw string) (string, int) {
	const (
		defaultHost = "127.0.0.1"
		defaultPort = 14446
	)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultHost, defaultPort
	}
	if i := strings.Index(raw, "://"); i >= 0 {
		raw = raw[i+3:]
	}
	if i := strings.IndexAny(raw, "/?"); i >= 0 {
		raw = raw[:i]
	}
	host, portStr, splitErr := net.SplitHostPort(raw)
	if splitErr != nil {
		// No port component — keep the raw token as host.
		return raw, defaultPort
	}
	port := defaultPort
	if n, err := strconv.Atoi(portStr); err == nil && n > 0 {
		port = n
	}
	if host == "" {
		host = defaultHost
	}
	return host, port
}

type EnvStruct struct {
	IS_PROD        bool
	IS_LOCAL       bool
	IS_SERVERLESS  bool
	STACK_NAME     string
	APP_CODE       string
	ENVIROMENT     string
	DB_NAME        string
	DB_USER        string
	DB_HOST        string
	DB_PASSWORD    string
	TMP_DIR        string
	REQ_IP         string
	REQ_ID         string
	REQ_PARAMS     string
	REQ_USER_AGENT string
	// HANDLER_PARH   string
	REQ_PATH               string
	AWS_PROFILE            string
	AWS_REGION             string
	S3_BUCKET              string
	DYNAMO_TABLE           string
	REQ_LAMBDA_ID          string
	API_ROUTE              string
	LAMBDA_NAME            string
	LOGS_FULL              bool
	LOGS_DEBUG             bool
	LOGS_ONLY_SAVE         bool
	DB_DISABLE_SSL         bool
	DB_PORT                int32
	USUARIO_ID             int32
	ADMIN_PASSWORD         string
	SECRET_PHRASE          string
	SMTP_HOST              string
	SMTP_EMAIL             string
	SMTP_USER              string
	SMTP_PASSWORD          string
	SMTP_PORT              int32
	CLOUD_PROVIDER         string
	CLOUDFLARE_ACCOUNT     string
	CLOUDFLARE_TOKEN       string
	CLOUDFLARE_DATABASE_ID string
	FRONTEND_CDN           string
	ZONE_NAME              string
	// OpenRouter — used by the in-app agent (backend/agent/llm). Model is
	// optional; the llm package defaults to tencent/hy3-preview when blank.
	OPENROUTER_KEY   string
	OPENROUTER_MODEL string
	// GenixSearch — lexical search backend reached over TCP. The
	// daemon is installed by cloud/text-searh/install_search.py, which
	// writes GENIXSEARCH_PASSWORD into credentials.json. GENIXSEARCH_URL
	// is the full endpoint (e.g. "https://host:14446" or "host:14446")
	// — only the host and port are used; the scheme is ignored. Falls
	// back to 127.0.0.1:14446 when empty.
	GENIXSEARCH_URL      string
	GENIXSEARCH_PASSWORD string
}

var Env *EnvStruct
var BuildDate string

func PopulateVariables() {
	fmt.Println("Populando Variables:: ")

	APP_CODE := os.Getenv("APP_CODE")
	isServerlessRuntime := IsRunningInLambda()
	useCredentialsFile := len(APP_CODE) == 0
	configuredCredentialsPath := strings.TrimSpace(os.Getenv("GENIX_CREDENTIALS_FILE"))

	wd, _ := os.Getwd()

	var variablesBytes []byte

	if useCredentialsFile {
		APP_CODE = "genix"
		dirname := strings.Split(wd, "/")
		parentPath := strings.Join(dirname[0:len(dirname)-1], "/")
		var fileError error

		credentialsSearchPaths := []string{parentPath + "/credentials.json", wd + "/credentials.json"}
		if len(configuredCredentialsPath) > 0 {
			// Allow systemd and other runtimes to point the backend to a fixed credentials location.
			credentialsSearchPaths = append(credentialsSearchPaths, configuredCredentialsPath)
		}

		for _, candidateCredentialsPath := range credentialsSearchPaths {
			file, err := os.Open(candidateCredentialsPath)
			if err != nil {
				fileError = err
				continue
			}
			defer file.Close()

			// Read the content of the file
			variablesBytes, err = io.ReadAll(file)
			if err != nil {
				fileError = err
				continue
			} else {
				fmt.Println("Seteando credentials.json desde:", candidateCredentialsPath)
				break
			}
		}

		if len(variablesBytes) == 0 {
			fmt.Println(fileError)
			panic("Archivo credentials.json no encontrado. Configure GENIX_CREDENTIALS_FILE o suba el archivo al directorio esperado.")
		}

	} else {
		configJsonBase64 := os.Getenv("CONFIG")
		if len(configJsonBase64) == 0 {
			panic("No se encontraron las variables de entorno.")
		}
		configJsonBase64 = MakeB64UrlDecode(configJsonBase64)
		baseBytes := Base64ToBytes(configJsonBase64)
		variablesBytes = []byte(DecompressZstd(&baseBytes))
	}

	err := json.Unmarshal(variablesBytes, &Env)
	if err != nil {
		fmt.Println("Error parsing credentials.json:", err)
		return
	}

	fmt.Println("Credenciales .json Parseadas:: ", "| Is Local:", Env.IS_LOCAL)

	if len(Env.DYNAMO_TABLE) == 0 {
		Env.DYNAMO_TABLE = Env.STACK_NAME + "-db"
	}
	if len(Env.API_ROUTE) == 0 {
		Env.API_ROUTE = "http://localhost:3589"
	}

	Env.LAMBDA_NAME = Env.STACK_NAME + "-backend"
	Env.APP_CODE = APP_CODE
	Env.IS_SERVERLESS = isServerlessRuntime
	Env.TMP_DIR = If(Env.IS_SERVERLESS, "/tmp/", wd+"/tmp/")

	Env.IS_PROD = strings.Contains(APP_CODE, "_prd")
	for _, value := range os.Args {
		if value == "prod" {
			Env.IS_PROD = true
			break
		}
	}
}

func IsRunningInLambda() bool {
	// AWS Lambda sets one or more of these runtime variables.
	return len(strings.TrimSpace(os.Getenv("AWS_LAMBDA_FUNCTION_NAME"))) > 0 ||
		len(strings.TrimSpace(os.Getenv("AWS_EXECUTION_ENV"))) > 0 ||
		len(strings.TrimSpace(os.Getenv("LAMBDA_TASK_ROOT"))) > 0
}

var REQ_PATHS = []string{}

func GetAwsConfig() aws_sdk.Config {
	var cfg aws_sdk.Config
	var err error

	setConfig := func(lo *config.LoadOptions) error {
		lo.Region = Env.AWS_REGION
		return nil
	}

	accessKeyEnv := os.Getenv("AWS_ACCESS_KEY_ID")
	if len(accessKeyEnv) > 0 {
		Log("Generando AWS Config con ACCESS_KEY en Región:", Env.AWS_REGION)
		cfg, err = config.LoadDefaultConfig(context.TODO(), setConfig)
	} else {
		Log("Generando AWS Config con profile:", Env.AWS_PROFILE, "|", Env.AWS_REGION)
		cfg, err = config.LoadDefaultConfig(
			context.TODO(), config.WithSharedConfigProfile(Env.AWS_PROFILE), setConfig)
	}
	if err != nil {
		panic(Concat(" ", "No se pudo obtener la configuración de AWS.", err))
	}
	return cfg
}
