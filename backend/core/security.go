package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/pelletier/go-toml/v2"
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
	IS_PROD       bool
	IS_LOCAL      bool
	IS_SERVERLESS bool
	// LAMBDA_RESPONSE_STREAMING must mirror the deployed Function URL's InvokeMode. When true the
	// handler returns a RESPONSE_STREAM body (raw bytes, no base64, 20 MB ceiling); when false it
	// returns the BUFFERED shape. A mismatch with the deployed InvokeMode breaks every request,
	// so both ends are pinned together in cloud/template.yml.
	LAMBDA_RESPONSE_STREAMING bool
	// APP_NAME prefixes every deployed resource name: DynamoDB table, Lambdas, R2 bucket.
	APP_NAME    string
	APP_CODE    string
	ENVIROMENT  string
	DB_NAME     string
	DB_USER     string
	DB_HOST     string
	DB_PASSWORD string
	TMP_DIR     string
	REQ_IP      string
	REQ_ID      string
	// REQUEST_ID is the numeric identity of the request being served, the one user_logs stores.
	// It is a mirror of HandlerArgs.RequestID and is only set under IS_SERVERLESS, where the
	// runtime serves one invocation at a time and a package global is therefore safe; the free
	// core.Log function reads it to build its prefix. In server mode it stays zero and the prefix
	// is not printed at all.
	REQUEST_ID     int64
	REQ_PARAMS     string
	REQ_USER_AGENT string
	// HANDLER_PARH   string
	REQ_PATH           string
	AWS_PROFILE        string
	AWS_REGION         string
	S3_BUCKET          string
	DYNAMO_TABLE       string
	REQ_LAMBDA_ID      string
	API_ROUTE          string
	LAMBDA_NAME        string
	LOGS_FULL          bool
	LOGS_DEBUG         bool
	LOGS_ONLY_SAVE     bool
	// LOG_ALL_REQUESTS widens user_logs from the failures it keeps by default to every finished
	// request. Off, a request that produced no error leaves no row — which is what keeps the
	// table small enough to scan a fifteen-minute window. Turn it on to measure traffic, and
	// expect a row per request for as long as it stays on.
	LOG_ALL_REQUESTS bool
	DB_DISABLE_SSL     bool
	DB_PORT            int32
	MAX_CLUSTERING_KEY int32 // Node's max_clustering_key_restrictions_per_query; 0 uses the ORM default of 100
	SERVER_PORT        int32 // Listen port of the standalone HTTP server; must match the port in NGINX_PROCESS, 0 uses 3589
	USUARIO_ID         int32
	ADMIN_PASSWORD     string
	// SECRET_PHRASE signs what users hold: session tokens (usuario-accesos.go) and password
	// hashes. It never authenticates one Genix process to another — that is INTERNAL_APIKEY.
	SECRET_PHRASE string
	// INTERNAL_APIKEY authenticates service-to-service calls: the credit rate limiter's TCP
	// frames and the SSE bridge's X-Bridge-Auth header, both handled by server_utils/. Split
	// from SECRET_PHRASE so the inter-service key can be rotated without invalidating every
	// live session token.
	INTERNAL_APIKEY string
	SMTP_HOST       string
	SMTP_EMAIL      string
	SMTP_USER       string
	SMTP_PASSWORD   string
	SMTP_PORT       int32
	// BACKEND_PROVIDER selects the cloud data mirror: DynamoDB (aws) or D1 (cloudflare).
	BACKEND_PROVIDER string
	// CDN_PROVIDER selects the object storage and public asset origin: S3 (aws) or R2 (cloudflare).
	CDN_PROVIDER           string
	CLOUDFLARE_ACCOUNT     string
	CLOUDFLARE_TOKEN       string
	CLOUDFLARE_BUCKET      string // R2 bucket for files and images; defaults to "<APP_NAME>-files", set it to pin an existing bucket
	CLOUDFLARE_DATABASE_ID string
	FRONTEND_CDN           string
	ZONE_NAME              string
	// APP_URL is the public origin of the Genix web app ("https://app.example.com", no trailing
	// slash). Only the sign-up email needs it, to build the verification link. It is read from
	// config and never from the request, so a caller cannot make our SMTP mail out a link that
	// points at their own domain.
	APP_URL string
	// WEBPAGE_RENDERER_URL is the storefront renderer artifact (webpage-renderer.zip: SSR
	// bundle + assets). Outside Lambda the backend runs the renderer locally and has to hand
	// this to the Node process, so the value has to reach the backend and not only the deploy
	// CLI. Optional: defaults to the CI-published URL, same as cloud/webpage-renderer.go.
	WEBPAGE_RENDERER_URL string
	// SSE_BRIDGE_URL is the SSE relay (server_utils/, bridge half) that keeps the browser's
	// stream open on behalf of this backend. Lambda cannot hold a stream for a
	// whole agent turn nor receive the browser's reply inside the same
	// invocation, so in serverless mode every server→browser message and every
	// browser RPC goes through it. Empty (or outside Lambda) = the backend serves
	// its own /agent/stream and the bridge is not used.
	SSE_BRIDGE_URL string
	// LLM fallback provider for an unregistered/default model. Registered picker
	// entries use their own ModelEntry.Provider and corresponding API key.
	MODEL_PROVIDER string
	META_KEY       string
	OPENROUTER_KEY string
	// DEFAULT_MODEL overrides the model used when a request carries no explicit
	// one. It must name a registered model. Blank = the first MODELS entry.
	DEFAULT_MODEL string
	// CLASSIFIER_MODEL_ID is independent from the model selected in the chat UI.
	// Blank reuses DEFAULT_MODEL, which keeps existing deployments valid.
	CLASSIFIER_MODEL_ID string
	// CLASSIFIER_PROVIDER stays fixed even if the user-facing agent provider changes.
	CLASSIFIER_PROVIDER string
	// MODELS is the agent's model registry, straight from the [[models]] array table.
	// File order is meaningful: it is the order of the model picker and its first entry
	// is the default when DEFAULT_MODEL is blank. Consumed by
	// backend/agent/llm, which turns each entry into its own request knobs.
	MODELS []ModelEntry
	// GenixSearch — lexical search backend reached over TCP. The
	// daemon is installed by scripts/configure/configure_db.py, which writes both
	// search.url and search.password into config.toml. GENIXSEARCH_URL
	// is the full endpoint (e.g. "https://host:14446" or "host:14446")
	// — only the host and port are used; the scheme is ignored. Falls
	// back to 127.0.0.1:14446 when empty.
	GENIXSEARCH_URL      string
	GENIXSEARCH_PASSWORD string
	// Qdrant stores user-documentation dense and lexical vectors. The same
	// INTERNAL_APIKEY used by deployment protects its gRPC endpoint.
	QDRANT_HOST       string
	QDRANT_HTTP_PORT  int
	QDRANT_GRPC_PORT  int
	QDRANT_PUBLIC     bool
	QDRANT_USE_TLS    bool
	QDRANT_COLLECTION string
	// Embeddings are independent from the chat model provider. The first RAG
	// collection uses Qwen through OpenRouter at its full 4096 dimensions.
	EMBEDDING_PROVIDER   string
	EMBEDDING_MODEL_ID   string
	EMBEDDING_DIMENSIONS int
	// SERVER_UTILS_ADDRESS is the Rust raw-TCP endpoint: the credit limiter and the lock service
	// share it, routed by the frame's opcode. One address for every operation the daemon serves.
	SERVER_UTILS_ADDRESS             string
	RATE_LIMIT_COMPANY_CPU_24H       uint64
	RATE_LIMIT_COMPANY_INFERENCE_24H uint64
	RATE_LIMIT_USER_CPU_24H          uint64
	RATE_LIMIT_USER_INFERENCE_24H    uint64
	// Public registration is unauthenticated and skips the credit limiter, so the only platform
	// brake is how many distinct emails one client IP may register within a window.
	SIGNUP_MAX_EMAILS_PER_IP int32
	SIGNUP_WINDOW_MINUTES    int32
	// CONTACT_EMAIL is the inbox the public contact form delivers to. Empty disables the endpoint
	// outright: with nowhere to send a message, accepting one would only be a way to fill a table.
	// It is read from config and never from the request, for the same reason APP_URL is — a caller
	// must not be able to choose who our SMTP credentials mail.
	CONTACT_EMAIL string
	// The contact form is unauthenticated too and skips the credit limiter as well, so it carries
	// its own per-IP window. Messages are counted whole here, not distinct addresses like sign-up:
	// the abuse is the volume of mail, and the sender picks the address on every submission.
	CONTACT_MAX_MESSAGES_PER_IP int32
	CONTACT_WINDOW_MINUTES      int32
}

// ModelEntry es una entrada de la tabla de array [[models]]: un modelo que el agente puede
// usar y los parámetros con los que se le llama. Vive aquí, y no en agent/llm, porque el
// parseo del archivo es de este paquete y llm importa a core (nunca al revés). Los nombres
// son los del contrato del upstream para que la traducción en llm sea campo a campo.
type ModelEntry struct {
	ID       string `toml:"id"`
	Provider string `toml:"provider"` // "meta" | "openrouter"; vacío = openrouter
	// Reasoning nulo = el modelo no razona: llm omite el parámetro en la petición, porque
	// enviarlo a un modelo sin razonamiento es un campo desconocido para el upstream.
	Reasoning *ModelReasoning `toml:"reasoning"`
	// Routing nulo = OpenRouter elige el upstream. Sin efecto en Meta, que sirve sus propios
	// modelos y no tiene el concepto.
	Routing *ModelRouting `toml:"routing"`
}

// ModelReasoning es el presupuesto de razonamiento por modelo. En Meta sólo sobreviven
// Effort y Enabled (ver llm.metaReasoningEffort); en OpenRouter se envía tal cual.
type ModelReasoning struct {
	Effort    string `toml:"effort"`     // "minimal"|"low"|"medium"|"high"|"xhigh"
	MaxTokens int    `toml:"max_tokens"` // tope duro alternativo a Effort
	Exclude   bool   `toml:"exclude"`    // oculta la traza para que no infle el prompt siguiente
	Enabled   *bool  `toml:"enabled"`    // false = no razonar
}

// ModelRouting elige el upstream de OpenRouter que sirve el modelo. Sort es la vía
// declarativa ("throughput" pide el endpoint más rápido de la lista del modelo); Order con
// AllowFallbacks=false lo fija a uno concreto, para cuando importa una cuantización o región.
type ModelRouting struct {
	Order          []string `toml:"order"`
	Sort           string   `toml:"sort"` // "throughput" | "price" | "latency"
	AllowFallbacks *bool    `toml:"allow_fallbacks"`
}

// fileConfig refleja la forma por secciones de config.toml. Sólo existe para el parseo:
// el resto del backend sigue leyendo core.Env, que es plano, y así los 213 puntos de uso
// de core.Env.X no cambian. Todo campo nuevo del archivo se añade aquí Y en el bloque de
// asignación de PopulateVariables, o queda silenciosamente en su cero-valor.
type fileConfig struct {
	AppName        string `toml:"app_name"`
	IsLocal        bool   `toml:"is_local"`
	Environment    string `toml:"environment"`
	AdminPassword  string `toml:"admin_password"`
	SecretPhrase   string `toml:"secret_phrase"`
	InternalApikey string `toml:"internal_apikey"`
	// Its own section, not a key under [rate_limit]: one raw-TCP endpoint serves every
	// server-utils operation, and the opcode picks which. Nesting it under one of its consumers
	// would read as if the lock service had an address of its own.
	//
	// Host is where a client dials; Public is what the daemon binds (0.0.0.0 vs loopback). They
	// are separate fields because behind NAT they cannot be the same value: the public IP a
	// Lambda dials is never an address the VM's own interface holds.
	ServerUtils struct {
		Host   string `toml:"host"`
		Port   int    `toml:"port"`
		Public bool   `toml:"public"`
	} `toml:"server_utils"`

	Providers struct {
		Backend string `toml:"backend"`
		CDN     string `toml:"cdn"`
		Model   string `toml:"model"`
	} `toml:"providers"`

	DB struct {
		Host             string `toml:"host"`
		Port             int32  `toml:"port"`
		Name             string `toml:"name"`
		User             string `toml:"user"`
		Password         string `toml:"password"`
		DisableSSL       bool   `toml:"disable_ssl"`
		MaxClusteringKey int32  `toml:"max_clustering_key"`
	} `toml:"db"`

	Server struct {
		Port int32 `toml:"port"`
	} `toml:"server"`

	AWS struct {
		Region   string `toml:"region"`
		Profile  string `toml:"profile"`
		S3Bucket string `toml:"s3_bucket"`
	} `toml:"aws"`

	Cloudflare struct {
		Account    string `toml:"account"`
		Token      string `toml:"token"`
		Bucket     string `toml:"bucket"`
		DatabaseID string `toml:"database_id"`
	} `toml:"cloudflare"`

	Frontend struct {
		CDNURL             string `toml:"cdn_url"`
		AppURL             string `toml:"app_url"`
		ZoneName           string `toml:"zone_name"`
		WebpageRendererURL string `toml:"webpage_renderer_url"`
	} `toml:"frontend"`

	SMTP struct {
		Host     string `toml:"host"`
		Port     int32  `toml:"port"`
		Email    string `toml:"email"`
		User     string `toml:"user"`
		Password string `toml:"password"`
	} `toml:"smtp"`

	Agent struct {
		DefaultModel       string `toml:"default_model"`
		ClassifierModel    string `toml:"classifier_model"`
		ClassifierProvider string `toml:"classifier_provider"`
		MetaKey            string `toml:"meta_key"`
		OpenRouterKey      string `toml:"openrouter_key"`
	} `toml:"agent"`

	RateLimit struct {
		CompanyCPU24h       uint64 `toml:"company_cpu_24h"`
		CompanyInference24h uint64 `toml:"company_inference_24h"`
		UserCPU24h          uint64 `toml:"user_cpu_24h"`
		UserInference24h    uint64 `toml:"user_inference_24h"`
	} `toml:"rate_limit"`

	SignUp struct {
		MaxEmailsPerIP int32 `toml:"max_emails_per_ip"`
		WindowMinutes  int32 `toml:"window_minutes"`
	} `toml:"sign_up"`

	Contact struct {
		Email            string `toml:"email"`
		MaxMessagesPerIP int32  `toml:"max_messages_per_ip"`
		WindowMinutes    int32  `toml:"window_minutes"`
	} `toml:"contact"`

	Search struct {
		URL      string `toml:"url"`
		Password string `toml:"password"`
	} `toml:"search"`

	Qdrant struct {
		Host       string `toml:"host"`
		HTTPPort   int    `toml:"http_port"`
		GRPCPort   int    `toml:"grpc_port"`
		Public     bool   `toml:"public"`
		UseTLS     bool   `toml:"use_tls"`
		Collection string `toml:"collection"`
	} `toml:"qdrant"`

	EmbeddingModel struct {
		Provider   string `toml:"provider"`
		ID         string `toml:"id"`
		Dimensions int    `toml:"dimensions"`
	} `toml:"embedding_model"`

	SSEBridge struct {
		URL string `toml:"url"`
	} `toml:"sse_bridge"`

	Logs struct {
		Full     bool `toml:"full"`
		Debug    bool `toml:"debug"`
		OnlySave bool `toml:"only_save"`
	} `toml:"logs"`

	// Shares the [request_log] section with the daemon, which reads every other key in it
	// (server_utils/src/config.rs). This one is the backend's alone: the daemon's `enabled`
	// decides whether a record that arrives is written, this decides whether one is sent at all.
	RequestLog struct {
		LogAllRequests bool `toml:"log_all_requests"`
	} `toml:"request_log"`

	// Tabla de array: en el archivo va al final, con los demás [[...]].
	Models []ModelEntry `toml:"models"`
}

// applyToEnv vuelca el archivo por secciones sobre la Env plana. Es el único punto donde
// las dos formas se tocan.
// Same default the daemon falls back to (server_utils/src/config.rs), so an omitted port keeps
// both halves pointing at the same socket.
const defaultServerUtilsPort = 14013

const (
	defaultQdrantHTTPPort      = 14014
	defaultQdrantGRPCPort      = 14015
	defaultQdrantCollection    = "genix_user_documentation_v1"
	defaultEmbeddingProvider   = "openrouter"
	defaultEmbeddingDimensions = 4096
)

// makeServerUtilsAddress turns the [server_utils] section into the one address this process dials.
//
// A private daemon is only reachable on loopback no matter what host is written, so the host is
// ignored rather than trusted: a value left over from a public deployment would otherwise turn
// every lock call into a connection to a machine that cannot answer. A public daemon with no host
// is left empty on purpose, so ConfigureServerUtils refuses it at startup instead of the first
// lock failing at request time.
func makeServerUtilsAddress(host string, port int, public bool) string {
	if port <= 0 {
		port = defaultServerUtilsPort
	}
	if !public {
		return fmt.Sprintf("127.0.0.1:%d", port)
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%v:%d", host, port)
}

func (file *fileConfig) applyToEnv(env *EnvStruct) {
	env.APP_NAME = file.AppName
	env.IS_LOCAL = file.IsLocal
	env.ENVIROMENT = file.Environment
	env.ADMIN_PASSWORD = file.AdminPassword
	env.SECRET_PHRASE = file.SecretPhrase
	env.INTERNAL_APIKEY = file.InternalApikey

	env.BACKEND_PROVIDER = file.Providers.Backend
	env.CDN_PROVIDER = file.Providers.CDN
	env.MODEL_PROVIDER = file.Providers.Model

	env.DB_HOST = file.DB.Host
	env.DB_PORT = file.DB.Port
	env.DB_NAME = file.DB.Name
	env.DB_USER = file.DB.User
	env.DB_PASSWORD = file.DB.Password
	env.DB_DISABLE_SSL = file.DB.DisableSSL
	env.MAX_CLUSTERING_KEY = file.DB.MaxClusteringKey

	env.SERVER_PORT = file.Server.Port

	env.AWS_REGION = file.AWS.Region
	env.AWS_PROFILE = file.AWS.Profile
	env.S3_BUCKET = file.AWS.S3Bucket

	env.CLOUDFLARE_ACCOUNT = file.Cloudflare.Account
	env.CLOUDFLARE_TOKEN = file.Cloudflare.Token
	env.CLOUDFLARE_BUCKET = file.Cloudflare.Bucket
	env.CLOUDFLARE_DATABASE_ID = file.Cloudflare.DatabaseID

	env.FRONTEND_CDN = file.Frontend.CDNURL
	env.APP_URL = strings.TrimRight(file.Frontend.AppURL, "/")
	env.ZONE_NAME = file.Frontend.ZoneName
	env.WEBPAGE_RENDERER_URL = file.Frontend.WebpageRendererURL

	env.SMTP_HOST = file.SMTP.Host
	env.SMTP_PORT = file.SMTP.Port
	env.SMTP_EMAIL = file.SMTP.Email
	env.SMTP_USER = file.SMTP.User
	env.SMTP_PASSWORD = file.SMTP.Password

	env.DEFAULT_MODEL = file.Agent.DefaultModel
	env.CLASSIFIER_MODEL_ID = strings.TrimSpace(file.Agent.ClassifierModel)
	if env.CLASSIFIER_MODEL_ID == "" {
		env.CLASSIFIER_MODEL_ID = env.DEFAULT_MODEL
	}
	env.CLASSIFIER_PROVIDER = strings.ToLower(strings.TrimSpace(file.Agent.ClassifierProvider))
	if env.CLASSIFIER_PROVIDER == "" {
		env.CLASSIFIER_PROVIDER = env.MODEL_PROVIDER
	}
	env.META_KEY = file.Agent.MetaKey
	env.OPENROUTER_KEY = file.Agent.OpenRouterKey
	env.MODELS = file.Models
	env.SERVER_UTILS_ADDRESS = makeServerUtilsAddress(
		file.ServerUtils.Host, file.ServerUtils.Port, file.ServerUtils.Public)
	env.RATE_LIMIT_COMPANY_CPU_24H = file.RateLimit.CompanyCPU24h
	env.RATE_LIMIT_COMPANY_INFERENCE_24H = file.RateLimit.CompanyInference24h
	env.RATE_LIMIT_USER_CPU_24H = file.RateLimit.UserCPU24h
	env.RATE_LIMIT_USER_INFERENCE_24H = file.RateLimit.UserInference24h
	env.SIGNUP_MAX_EMAILS_PER_IP = file.SignUp.MaxEmailsPerIP
	env.SIGNUP_WINDOW_MINUTES = file.SignUp.WindowMinutes
	env.CONTACT_EMAIL = strings.ToLower(strings.TrimSpace(file.Contact.Email))
	env.CONTACT_MAX_MESSAGES_PER_IP = file.Contact.MaxMessagesPerIP
	env.CONTACT_WINDOW_MINUTES = file.Contact.WindowMinutes

	env.GENIXSEARCH_URL = file.Search.URL
	env.GENIXSEARCH_PASSWORD = file.Search.Password

	env.QDRANT_HOST = strings.TrimSpace(file.Qdrant.Host)
	if env.QDRANT_HOST == "" && !file.Qdrant.Public {
		env.QDRANT_HOST = "127.0.0.1"
	}
	env.QDRANT_HTTP_PORT = file.Qdrant.HTTPPort
	if env.QDRANT_HTTP_PORT <= 0 {
		env.QDRANT_HTTP_PORT = defaultQdrantHTTPPort
	}
	env.QDRANT_GRPC_PORT = file.Qdrant.GRPCPort
	if env.QDRANT_GRPC_PORT <= 0 {
		env.QDRANT_GRPC_PORT = defaultQdrantGRPCPort
	}
	env.QDRANT_PUBLIC = file.Qdrant.Public
	env.QDRANT_USE_TLS = file.Qdrant.UseTLS
	env.QDRANT_COLLECTION = strings.TrimSpace(file.Qdrant.Collection)
	if env.QDRANT_COLLECTION == "" {
		env.QDRANT_COLLECTION = defaultQdrantCollection
	}

	env.EMBEDDING_PROVIDER = strings.ToLower(strings.TrimSpace(file.EmbeddingModel.Provider))
	if env.EMBEDDING_PROVIDER == "" {
		env.EMBEDDING_PROVIDER = defaultEmbeddingProvider
	}
	env.EMBEDDING_MODEL_ID = strings.TrimSpace(file.EmbeddingModel.ID)
	env.EMBEDDING_DIMENSIONS = file.EmbeddingModel.Dimensions
	if env.EMBEDDING_DIMENSIONS <= 0 {
		env.EMBEDDING_DIMENSIONS = defaultEmbeddingDimensions
	}

	env.SSE_BRIDGE_URL = file.SSEBridge.URL

	env.LOGS_FULL = file.Logs.Full
	env.LOGS_DEBUG = file.Logs.Debug
	env.LOGS_ONLY_SAVE = file.Logs.OnlySave
	env.LOG_ALL_REQUESTS = file.RequestLog.LogAllRequests
}

// DefaultWebpageRendererURL is the CI-published storefront renderer artifact. It is duplicated in
// cloud/webpage-renderer.go (defaultRendererZipUrl) and the two must stay equal — `cloud` is a
// separate Go module and cannot import this package, so the compiler cannot enforce it.
const DefaultWebpageRendererURL = "https://genix-dev.un.pe/webpage-renderer.zip"

var Env *EnvStruct
var BuildDate string

func PopulateVariables() {
	log.Printf("[core.config] populate_variables")

	APP_CODE := os.Getenv("APP_CODE")
	isServerlessRuntime := IsRunningInLambda()
	useCredentialsFile := len(APP_CODE) == 0
	configuredConfigPath := strings.TrimSpace(os.Getenv("GENIX_CONFIG_FILE"))

	wd, _ := os.Getwd()

	var variablesBytes []byte

	if useCredentialsFile {
		APP_CODE = "genix"
		dirname := strings.Split(wd, "/")
		parentPath := strings.Join(dirname[0:len(dirname)-1], "/")
		var fileError error

		configSearchPaths := []string{parentPath + "/config.toml", wd + "/config.toml"}
		if len(configuredConfigPath) > 0 {
			// An explicit path selects the environment even when the default file also exists.
			configSearchPaths = append([]string{configuredConfigPath}, configSearchPaths...)
		}

		for _, candidateConfigPath := range configSearchPaths {
			file, err := os.Open(candidateConfigPath)
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
				log.Printf("[core.config] config_file_selected path=%s", candidateConfigPath)
				break
			}
		}

		if len(variablesBytes) == 0 {
			log.Printf("[core.config] config_file_missing error=%v", fileError)
			panic("Archivo config.toml no encontrado. Configure GENIX_CONFIG_FILE o suba el archivo al directorio esperado.")
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

	// Env se aloja aquí porque hasta ahora la creaba el propio json.Unmarshal sobre el puntero.
	Env = &EnvStruct{}
	parsedFile := fileConfig{}
	if err := toml.Unmarshal(variablesBytes, &parsedFile); err != nil {
		// Volver con Env vacío hacía que el fallo apareciera mucho después, como un panic por
		// campo faltante (p. ej. server_utils) que no menciona la configuración. Un CONFIG que
		// aún trae el JSON anterior a la migración a TOML es exactamente ese caso: el error real
		// es este, así que se aborta aquí y con el origen del contenido a la vista.
		source := "config.toml"
		if !useCredentialsFile {
			source = "la variable de entorno CONFIG"
		}
		panic(fmt.Sprintf("no se pudo parsear %s como TOML: %v", source, err))
	}
	parsedFile.applyToEnv(Env)

	log.Printf("[core.config] config_parsed is_local=%t", Env.IS_LOCAL)

	if len(Env.DYNAMO_TABLE) == 0 {
		Env.DYNAMO_TABLE = Env.APP_NAME + "-db"
	}
	if len(Env.API_ROUTE) == 0 {
		Env.API_ROUTE = "http://localhost:3589"
	}
	// Same rule as cloud/main.go: an explicit CLOUDFLARE_BUCKET wins, so the runtime writes to the
	// bucket the infra deploy actually created instead of one derived from the current APP_NAME.
	Env.CLOUDFLARE_BUCKET = strings.TrimSpace(Env.CLOUDFLARE_BUCKET)
	if len(Env.CLOUDFLARE_BUCKET) == 0 {
		Env.CLOUDFLARE_BUCKET = Env.APP_NAME + "-files"
	}
	// Keep this default identical to defaultRendererZipUrl in cloud/webpage-renderer.go: the same
	// artifact has to be used whether the renderer runs in Lambda or locally.
	Env.WEBPAGE_RENDERER_URL = strings.TrimSpace(Env.WEBPAGE_RENDERER_URL)
	if len(Env.WEBPAGE_RENDERER_URL) == 0 {
		Env.WEBPAGE_RENDERER_URL = DefaultWebpageRendererURL
	}
	// The environment override lets systemd/Lambda point at a private daemon without rewriting TOML.
	if serverUtilsAddress := strings.TrimSpace(os.Getenv("SERVER_UTILS_ADDRESS")); serverUtilsAddress != "" {
		Env.SERVER_UTILS_ADDRESS = serverUtilsAddress
	}
	Env.SERVER_UTILS_ADDRESS = strings.TrimSpace(Env.SERVER_UTILS_ADDRESS)
	if Env.SERVER_UTILS_ADDRESS == "" {
		Env.SERVER_UTILS_ADDRESS = "127.0.0.1:14013"
	}
	// Public registration must never be unlimited by omission, so an absent or nonsensical
	// setting falls back to the documented defaults instead of to zero.
	if signUpMax := strings.TrimSpace(os.Getenv("SIGNUP_MAX_EMAILS_PER_IP")); signUpMax != "" {
		if parsed, err := strconv.Atoi(signUpMax); err == nil {
			Env.SIGNUP_MAX_EMAILS_PER_IP = int32(parsed)
		}
	}
	if signUpWindow := strings.TrimSpace(os.Getenv("SIGNUP_WINDOW_MINUTES")); signUpWindow != "" {
		if parsed, err := strconv.Atoi(signUpWindow); err == nil {
			Env.SIGNUP_WINDOW_MINUTES = int32(parsed)
		}
	}
	if Env.SIGNUP_MAX_EMAILS_PER_IP <= 0 {
		Env.SIGNUP_MAX_EMAILS_PER_IP = 5
	}
	if Env.SIGNUP_WINDOW_MINUTES <= 0 {
		Env.SIGNUP_WINDOW_MINUTES = 20
	}
	// Same rule for the contact form, and the same reason: an omitted setting must land on the
	// documented limit rather than on zero, which the handler would read as "no messages allowed"
	// or, worse for a maximum, as no limit at all.
	if contactEmail := strings.TrimSpace(os.Getenv("CONTACT_EMAIL")); contactEmail != "" {
		Env.CONTACT_EMAIL = strings.ToLower(contactEmail)
	}
	if contactMax := strings.TrimSpace(os.Getenv("CONTACT_MAX_MESSAGES_PER_IP")); contactMax != "" {
		if parsed, err := strconv.Atoi(contactMax); err == nil {
			Env.CONTACT_MAX_MESSAGES_PER_IP = int32(parsed)
		}
	}
	if contactWindow := strings.TrimSpace(os.Getenv("CONTACT_WINDOW_MINUTES")); contactWindow != "" {
		if parsed, err := strconv.Atoi(contactWindow); err == nil {
			Env.CONTACT_WINDOW_MINUTES = int32(parsed)
		}
	}
	if Env.CONTACT_MAX_MESSAGES_PER_IP <= 0 {
		Env.CONTACT_MAX_MESSAGES_PER_IP = 3
	}
	if Env.CONTACT_WINDOW_MINUTES <= 0 {
		Env.CONTACT_WINDOW_MINUTES = 2
	}
	// Mirror the limiter's environment precedence so displayed quotas cannot drift from enforcement.
	applyRateLimitUint32Override("RATE_LIMIT_COMPANY_CPU_24H", &Env.RATE_LIMIT_COMPANY_CPU_24H)
	applyRateLimitUint32Override("RATE_LIMIT_COMPANY_INFERENCE_24H", &Env.RATE_LIMIT_COMPANY_INFERENCE_24H)
	applyRateLimitUint32Override("RATE_LIMIT_USER_CPU_24H", &Env.RATE_LIMIT_USER_CPU_24H)
	applyRateLimitUint32Override("RATE_LIMIT_USER_INFERENCE_24H", &Env.RATE_LIMIT_USER_INFERENCE_24H)
	validateRateLimitDailyMaximum("rate_limit.company_cpu_24h", Env.RATE_LIMIT_COMPANY_CPU_24H)
	validateRateLimitDailyMaximum("rate_limit.company_inference_24h", Env.RATE_LIMIT_COMPANY_INFERENCE_24H)
	validateRateLimitDailyMaximum("rate_limit.user_cpu_24h", Env.RATE_LIMIT_USER_CPU_24H)
	validateRateLimitDailyMaximum("rate_limit.user_inference_24h", Env.RATE_LIMIT_USER_INFERENCE_24H)

	Env.LAMBDA_NAME = Env.APP_NAME + "-backend"
	Env.APP_CODE = APP_CODE
	Env.IS_SERVERLESS = isServerlessRuntime
	Env.TMP_DIR = If(Env.IS_SERVERLESS, "/tmp/", wd+"/tmp/")

	// Response streaming is owned by the deployment, not by config.toml: the Function URL's
	// InvokeMode and this flag are both declared in cloud/template.yml, because a handler that
	// disagrees with the deployed invoke mode fails every request. Reading the same variable
	// here is what keeps the two ends in step.
	Env.LAMBDA_RESPONSE_STREAMING = strings.TrimSpace(os.Getenv("LAMBDA_RESPONSE_STREAMING")) == "1"

	Env.IS_PROD = strings.Contains(APP_CODE, "_prd")
	for _, value := range os.Args {
		if value == "prod" {
			Env.IS_PROD = true
			break
		}
	}
}

func applyRateLimitUint32Override(environmentName string, destination *uint64) {
	rawValue := strings.TrimSpace(os.Getenv(environmentName))
	if rawValue == "" {
		return
	}
	parsedValue, err := strconv.ParseUint(rawValue, 10, 32)
	if err != nil || parsedValue == 0 {
		panic(fmt.Sprintf("%s must be a positive uint32", environmentName))
	}
	*destination = parsedValue
}

func validateRateLimitDailyMaximum(configName string, configuredValue uint64) {
	// Persisted usage is uint32-wide, matching the Rust limiter's startup validation.
	if configuredValue == 0 || configuredValue > math.MaxUint32 {
		panic(fmt.Sprintf("%s must be a positive uint32", configName))
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
