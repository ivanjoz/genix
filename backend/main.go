package main

import (
	"app/agent"
	"app/business"
	"app/core"
	server_utils "app/core/server_utils"
	"app/db"
	"app/exec"
	"context"
	"fmt"
	"github.com/ivanjoz/genix-orm/scylla"
	"github.com/ivanjoz/genix-orm/scylla/text_search"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/cors"
)

// La regla de EventBridge de cloud/template.yml manda {"exec":<minutos>} como body cada 10
// minutos. El número es solo la cadencia declarada, para el log; lo que dispara el trabajo es
// el prefijo.
const scheduledCronTickPrefix = `{"exec":`

// runScheduledCronTick ejecuta las acciones pendientes de cron_actions y devuelve cuántas
// corrieron. Es el equivalente en Lambda del StartCronWatcher que solo vive en el VPS.
func runScheduledCronTick(tickBody string) int {
	// clearEnvVariables acaba de apagar este flag, y en serverless core.Log descarta toda línea
	// que no empiece con "*" ni contenga "error"/"warn": sin esto el tick no deja rastro en
	// CloudWatch, que es el único sitio donde se puede observar.
	core.Env.LOGS_FULL = true
	core.Log("*Cron tick programado:: ", core.StrCut(tickBody, 60))

	// Siembra la cadena de 30 min del rebuild de productos. La continuidad ya no depende de esto:
	// la fila guarda su propia cadencia y el executor encola el frame siguiente pase lo que pase.
	// Sigue haciendo falta como semilla inicial, porque en Lambda no hay arranque de VPS que la
	// cree. Es idempotente: ScheduleCronAction deduplica contra la fila pendiente del mismo frame.
	// Va con recover porque hace panic ante un error de DB, y ese fallo no debe impedir que se
	// ejecute el resto de la cola.
	func() {
		defer func() {
			if recoveredValue := recover(); recoveredValue != nil {
				core.Log("*Cron tick error resembrando el rebuild de productos:: ", recoveredValue)
			}
		}()
		business.ScheduleProductsDbRebuildCron()
	}()

	executedActionsCount := core.RunPendingCronActions()
	core.Log("*Cron tick finalizado:: acciones ejecutadas:", executedActionsCount)
	return executedActionsCount
}

// runLambdaRequest is the request path shared by both invoke modes. It stops at
// core.MainResponse, which carries whichever of the two AWS response shapes prepareResponse
// built, so the only difference between the buffered and streaming entrypoints is which field
// they unwrap.
func runLambdaRequest(request *events.APIGatewayV2HTTPRequest) core.MainResponse {
	clearEnvVariables()

	core.Env.REQ_IP = request.RequestContext.HTTP.SourceIP
	if len(request.Body) > 0 {
		core.Log("*body enviado: ", core.StrCut(request.Body, 400))
	}

	// Revisa si lo que se está pidiendo es ejecutar una funcion
	if len(request.Body) > 11 && request.Body[0:11] == `{"fn_exec":` {
		funcResponse := ExecFuncHandler(request.Body)
		bodyBytes := []byte(core.ToJsonNoErr(funcResponse))
		core.Log("*Body response::" + core.StrCut(string(bodyBytes), 400))
		handlerResponse := core.HandlerResponse{Body: &bodyBytes, Headers: map[string]string{}}

		if core.Env.LAMBDA_RESPONSE_STREAMING {
			return core.MainResponse{LambdaStreamingResponse: core.MakeStreamingResponseFinal(&handlerResponse)}
		}
		return core.MainResponse{LambdaResponse: core.MakeResponseFinal(&handlerResponse)}
	}

	// Tick programado de EventBridge. No hay ruta HTTP detrás, así que no puede seguir al
	// mainHandler: sin Authorization moriría en CheckUser.
	if strings.HasPrefix(request.Body, scheduledCronTickPrefix) {
		bodyBytes := []byte(core.ToJsonNoErr(map[string]int{"executed": runScheduledCronTick(request.Body)}))
		handlerResponse := core.HandlerResponse{Body: &bodyBytes, Headers: map[string]string{}}

		if core.Env.LAMBDA_RESPONSE_STREAMING {
			return core.MainResponse{LambdaStreamingResponse: core.MakeStreamingResponseFinal(&handlerResponse)}
		}
		return core.MainResponse{LambdaResponse: core.MakeResponseFinal(&handlerResponse)}
	}

	route := request.RequestContext.HTTP.Path
	if len(route) == 0 {
		route = request.RawPath
	}
	if len(route) == 0 {
		core.Log("No custom path given, but AWS routed this request to this Lambda anyways.")
		route = "MISSING"
	}

	args := core.HandlerArgs{
		Body:     &request.Body,
		Query:    request.QueryStringParameters,
		Headers:  request.Headers,
		Route:    route,
		Method:   request.RequestContext.HTTP.Method,
		ClientIP: request.RequestContext.HTTP.SourceIP,
	}
	return mainHandler(&args)
}

// LambdaHandler serves a Function URL deployed with the default BUFFERED invoke mode.
func LambdaHandler(_ context.Context, request *events.APIGatewayV2HTTPRequest) (resp *events.APIGatewayV2HTTPResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Internal Server Error (Panic in LambdaHandler): %v", r)
			core.Logx(5, errStr)
			core.Log(string(debug.Stack()))
			resp = core.MakeErrRespFinal(500, errStr)
			err = nil // return nil error to Lambda runtime so it sends our response
		}
	}()

	return runLambdaRequest(request).LambdaResponse, nil
}

// LambdaStreamingHandler serves a Function URL deployed with InvokeMode RESPONSE_STREAM. The
// body goes out as raw bytes after a JSON prelude, so there is no base64 expansion and the
// response ceiling is 20 MB instead of 6 MB.
func LambdaStreamingHandler(_ context.Context, request *events.APIGatewayV2HTTPRequest) (resp *events.LambdaFunctionURLStreamingResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Internal Server Error (Panic in LambdaStreamingHandler): %v", r)
			core.Logx(5, errStr)
			core.Log(string(debug.Stack()))
			resp = core.MakeErrStreamingFinal(500, errStr)
			err = nil // return nil error to Lambda runtime so it sends our response
		}
	}()

	return runLambdaRequest(request).LambdaStreamingResponse, nil
}

func LocalHandler(w http.ResponseWriter, request *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Internal Server Error (Panic in LocalHandler): %v", r)
			core.Logx(5, errStr)
			core.Log(string(debug.Stack()))

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			errorMap := map[string]string{
				"error": errStr,
			}
			errorJson := core.ToJsonNoErr(errorMap)
			w.Write([]byte(errorJson))
		}
	}()

	const maxBodyBytes = int64(10 << 20) // 10 MiB
	bodyReader := http.MaxBytesReader(w, request.Body, maxBodyBytes)
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	body := string(bodyBytes)

	clientIP := core.ClientIPFromRequest(request)
	// Also fed to the log record, which reads it from the package global.
	core.Env.REQ_IP = clientIP

	args := core.HandlerArgs{
		Body:           &body,
		Method:         strings.ToUpper(request.Method),
		Route:          request.URL.Path,
		ClientIP:       clientIP,
		ResponseWriter: &w,
		ReqContext:     request,
		StartTime:      time.Now().UnixMilli(),
	}

	// Revisa si lo que se está pidiendo es ejecutar una funcion
	if len(body) > 11 && body[0:11] == `{"fn_exec":` {
		core.Log("Ejecutando funcion...")
		funcResponse := ExecFuncHandler(body)
		lambdaResponse := map[string]any{
			"statusCode": 200,
			"body":       core.ToJsonNoErr(funcResponse),
			"headers": map[string]string{
				"Content-Type": "application/json; charset=utf-8",
			},
		}
		body := core.ToJsonNoErr(lambdaResponse)
		bodyBytes := []byte(body)
		response := core.HandlerResponse{Body: &bodyBytes}
		core.SendLocalResponse(args, response)
		return
	}

	// Mismo tick programado que en Lambda, para poder dispararlo en local con un POST plano.
	if strings.HasPrefix(body, scheduledCronTickPrefix) {
		bodyBytes := []byte(core.ToJsonNoErr(map[string]int{"executed": runScheduledCronTick(body)}))
		core.SendLocalResponse(args, core.HandlerResponse{Body: &bodyBytes})
		return
	}

	// Convierte los query params en un map[string]: stirng
	queryString := request.URL.Query()
	args.Query = make(map[string]string)

	for key, values := range queryString {
		value := strings.Join(values[:], ",")
		args.Query[key] = value
	}

	// Convierte los headers en un map[string]: string
	args.Headers = make(map[string]string)

	for key, values := range request.Header {
		value := strings.Join(values[:], ",")
		args.Headers[key] = value
	}

	mainHandler(&args)
}

func OnPanic(panicMessage interface{}) {
	core.Logx(5, "Error 500 (Panic): ", panicMessage)
	core.Log(string(debug.Stack()))
}

// configureTextSearchGenixSearch resolves the GenixSearch endpoint
// from credentials (falling back to 127.0.0.1:14446) and pushes it
// into the text_search package. GENIXSEARCH_PASSWORD must be set in
// prod or writes will fail at handshake; we log a warning when it's
// missing.
func configureTextSearchGenixSearch() {
	host, port := core.ParseGenixSearchURL(core.Env.GENIXSEARCH_URL)
	password := strings.TrimSpace(core.Env.GENIXSEARCH_PASSWORD)
	if password == "" && core.Env.IS_PROD {
		core.Log("text_search: GENIXSEARCH_PASSWORD empty in prod; writes will fail at handshake")
	}
	text_search.Configure(host, port, password)
}

// resolveServerPort picks the listen address for the standalone HTTP server. The SERVER_PORT
// environment variable wins (systemd sets it from config.toml via
// scripts/configure_server.py), then the server.port config value itself, then the 3589 default.
// Nginx proxies to the port half of server.nginx_process, so the two must agree.
func resolveServerPort() string {
	if envPort := strings.TrimSpace(os.Getenv("SERVER_PORT")); envPort != "" {
		if parsedPort, err := strconv.Atoi(envPort); err == nil && parsedPort > 0 && parsedPort < 65536 {
			return fmt.Sprintf(":%d", parsedPort)
		}
		core.Log("SERVER_PORT env var is not a valid port, ignoring it:", envPort)
	}
	if core.Env != nil && core.Env.SERVER_PORT > 0 {
		return fmt.Sprintf(":%d", core.Env.SERVER_PORT)
	}
	return ":3589"
}

// bootstrapCronSchedulers starts the VPS cron watcher and seeds the recurring products rebuild.
// It recovers on its own because a panic in a goroutine takes the whole process down, and a cron
// seed that cannot reach the database is not a reason to kill a working HTTP server.
func bootstrapCronSchedulers() {
	defer func() {
		if recovered := recover(); recovered != nil {
			core.Log("cron bootstrap error:", recovered)
		}
	}()

	core.StartCronWatcher()
	// Seed the recurring 30-min products .db rebuild tick (self-reschedules thereafter).
	business.ScheduleProductsDbRebuildCron()
}

func main() {
	core.PopulateVariables()
	// One address, one secret, one connection: the credit limiter and the lock service share it
	// and are told apart by the frame's opcode. The logger is pushed in because that package
	// cannot import core (cycle), the same as text_search.
	server_utils.SetLogger(core.Log)
	if err := core.ConfigureServerUtils(core.Env.SERVER_UTILS_ADDRESS, core.Env.INTERNAL_APIKEY); err != nil {
		panic("invalid server-utils configuration: " + err.Error())
	}
	serverPort := resolveServerPort()
	// Wire the GenixSearch endpoint before any DB write that might
	// touch a TextSearchColumn-backed table. The text_search package
	// can't import core (cycle: core -> core/types -> db ->
	// text_search), so the resolved config is pushed in from here.
	configureTextSearchGenixSearch()
	// Print deployment path early so systemd logs show which cloned repo the
	// binary will scan for route markdown and generated menu descriptions.
	fmt.Println("GENIX_REPOSITORY_ROOT=", os.Getenv("GENIX_REPOSITORY_ROOT"))
	makeAppHandlers()
	fmt.Println("Setting full logs...")
	// os.Setenv("LOGS_FULL", "1")

	if core.Env.IS_SERVERLESS { // Controla los panic error
		defer func() {
			if r := recover(); r != nil {
				OnPanic(r)
			}
		}()
	}

	fmt.Printf("Starting DB connection. HOST %v:%v ..."+"\n", core.Env.DB_HOST, core.Env.DB_PORT)

	scylla.SetScyllaConnection(scylla.ConnParams{
		Host:             core.Env.DB_HOST,
		Port:             int(core.Env.DB_PORT),
		User:             core.Env.DB_USER,
		Password:         core.Env.DB_PASSWORD,
		Keyspace:         core.Env.DB_NAME,
		MaxClusteringKey: int(core.Env.MAX_CLUSTERING_KEY),
	})

	fmt.Println("DB connection started!")

	invokeFun := ""
	invokeFunIndex := -1
	for argumentIndex, value := range os.Args {
		if len(value) >= 2 && value[0:2] == "fn" {
			core.Env.LOGS_FULL = true
			invokeFun = value
			invokeFunIndex = argumentIndex
			break
		}
	}

	if core.Env.IS_LOCAL {
		core.Env.LOGS_FULL = true
	}

	// Mirror runtime logging flags into db so query debug logs follow the
	// resolved environment: LOGS_FULL → level 2 (verbose), IS_LOCAL → level
	// 1 (basic), otherwise silent.
	dbLogLevel := 0
	/*
		if core.Env.IS_LOCAL {
			dbLogLevel = 1
		}
		if core.Env.LOGS_FULL {
			dbLogLevel = 2
		}
	*/
	db.SetDebugLogging(dbLogLevel)

	// Create project-local tmp/promps once so per-call prompt writes can skip
	// the parent-dir check. Local-only; no-op in serverless/prod.
	agent.InitPromptLog()

	// Revisa si lo que se requiere es ejecutar una función
	if len(invokeFun) != 0 {
		fmt.Println("Invocando función...")
		funcToInvoke, ok := exec.ExecHandlers[invokeFun]
		if !ok {
			funcToInvoke, ok = exec.ExecHandlersTesting[invokeFun]
		}
		if !ok {
			core.Log("No se encontró la función a ejecutar:: ", invokeFun)
			os.Exit(1)
		}
		execMessage := ""
		if invokeFunIndex >= 0 && invokeFunIndex+1 < len(os.Args) {
			execMessage = strings.Join(os.Args[invokeFunIndex+1:], " ")
		}
		args := core.ExecArgs{Message: execMessage}
		funcResponse := funcToInvoke(&args)
		if len(funcResponse.Error) > 0 {
			core.Log("Exec function error::", funcResponse.Error)
			os.Exit(1)
		}
		if len(funcResponse.Message) > 0 {
			core.Log("Exec function message::", funcResponse.Message)
		}
		if len(funcResponse.Content) > 0 {
			core.Print(funcResponse.Content)
		}
		return
	}

	// Si se está desarrollando en local
	if !core.Env.IS_SERVERLESS {
		exec.StartUsageLogFlushWorker()
		if !core.Env.IS_LOCAL {
			// Off the main goroutine on purpose: ScheduleCronAction panics when its query fails,
			// and on a VPS (IS_SERVERLESS=false) no recover is installed, so a database that is
			// briefly unreachable at boot would otherwise stop the HTTP listener from ever
			// starting. The listener must come up whether or not the cron seed succeeds.
			go bootstrapCronSchedulers()
		}

		core.Log("Ejecutando en local. http://localhost" + serverPort)

		corsMiddleware := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{http.MethodPost, http.MethodPut, http.MethodGet},
			AllowedHeaders:   []string{"*"},
			ExposedHeaders:   []string{"X-Metadata", "X-Rate-Limit-Code"},
			AllowCredentials: false,
		})

		// SSE+POST channel so the backend can drive the browser as an agent.
		// /agent/stream is the tab's permanent event stream (chat events AND page
		// commands); /agent/in carries every browser→backend message (command
		// replies and unsolicited events). The turn itself is not here: it is a
		// plain API route (POST p-agent-turn, agent/turn.go) so that the exact
		// same client code works against Lambda, where the stream lives on the
		// SSE bridge (server_utils/) instead of this process.
		mux := http.NewServeMux()
		// The browser connects to these cross-origin (app served from the dev
		// proxy, backend on another port), so unlike the old WS upgrade they need
		// CORS headers. No method prefix on the patterns: that lets the CORS
		// middleware answer the JSON POST's preflight OPTIONS on /agent/in too.
		mux.Handle("/agent/stream", corsMiddleware.Handler(http.HandlerFunc(agent.HandleStream)))
		mux.Handle("/agent/in", corsMiddleware.Handler(http.HandlerFunc(agent.HandleIn)))
		// HTTP entrypoint for external LLM agents (Claude Code / Gemini): batch
		// actions in, post-action page snapshot out. Requires the tab's
		// /agent/stream to be open.
		mux.HandleFunc("POST /agent", agent.HandleAgentHTTP)
		// GET /agent serves read-only side-channel queries (currently `?get=menu`).
		mux.HandleFunc("GET /agent", agent.HandleAgentGet)
		mux.Handle("/", corsMiddleware.Handler(http.HandlerFunc(LocalHandler)))

		// Inicia el servidor con timeouts (previene slowloris y mejora resiliencia).
		srv := &http.Server{
			Addr:              serverPort,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			// Keep disabled to allow long-lived SSE streams (metrics and future real-time endpoints).
			WriteTimeout:   0,
			IdleTimeout:    120 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1 MiB
			// Track active connections for operational metrics and SSE dashboards.
			ConnState: func(connection net.Conn, currentState http.ConnState) {
				core.UpdateHTTPConnectionState(connection, currentState)
			},
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			core.Log("HTTP server error:", err)
		}
	} else {
		// Si se está en Lamnda
		logger := log.New(os.Stdout, "", log.LstdFlags|log.Llongfile)
		logger.Println("Lambda has started.")
		// The main goroutine in a Lambda might never run its deferred statements.
		// This is because of how the Lambda is shutdown.
		defer logger.Println("Lambda has stopped.")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// The handler shape has to match the deployed Function URL InvokeMode; returning the
		// wrong one breaks every request, so it is driven by explicit config rather than guessed.
		if core.Env.LAMBDA_RESPONSE_STREAMING {
			logger.Println("Invoke mode: RESPONSE_STREAM")
			lambda.StartWithOptions(LambdaStreamingHandler, lambda.WithContext(ctx))
		} else {
			logger.Println("Invoke mode: BUFFERED")
			lambda.StartWithOptions(LambdaHandler, lambda.WithContext(ctx))
		}
	}
}
