package main

import (
	"app/agent"
	"app/business"
	"app/core"
	"app/db"
	"app/exec"
	"context"
	"fmt"
	fareward "github.com/ivanjoz/fareward/go"
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
// scripts/configure/configure_server.py), then the server.port config value itself, then the 3589 default.
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

// tailnetCGNATBlock is the range Tailscale assigns every node (100.64.0.0/10). Matching on the
// address rather than on an interface name keeps this working whether the device is tailscale0,
// utun on macOS or a userspace one.
var tailnetCGNATBlock = net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// resolveDevListenAddresses returns the addresses a dev backend binds, instead of every interface.
//
// A dev run holds the production database credentials and answers on a port with no nginx and no
// firewall rule in front of it, so "all interfaces" hands it to the LAN and to anything forwarded
// to this host — which is how VPN-appliance scanner traffic ends up in the request log. Loopback
// serves the machine itself, and the tailnet address is what serve_tailscale needs for the browser
// on another node; nothing else has a reason to reach it.
//
// An empty result means no tailnet address was found, and the caller falls back to the port alone.
func resolveDevListenAddresses(serverPort string) []string {
	addresses := []string{"127.0.0.1" + serverPort}

	interfaceAddresses, err := net.InterfaceAddrs()
	if err != nil {
		core.Log("No se pudieron enumerar las interfaces, se escucha sólo en loopback:", err)
		return addresses
	}
	for _, interfaceAddress := range interfaceAddresses {
		addressCIDR, ok := interfaceAddress.(*net.IPNet)
		if !ok {
			continue
		}
		if ipv4 := addressCIDR.IP.To4(); ipv4 != nil && tailnetCGNATBlock.Contains(ipv4) {
			addresses = append(addresses, ipv4.String()+serverPort)
		}
	}
	return addresses
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
	// A frozen clock backdates every record the process writes, so it can never be silent.
	core.LogHistoricalClockIfActive()
	// One address, one secret, one connection: the credit limiter and the lock service share it
	// and are told apart by the frame's opcode. The logger is pushed in because that package
	// cannot import core (cycle), the same as text_search.
	fareward.SetLogger(core.Log)
	if err := core.ConfigureFareward(core.Env.FAREWARD_ADDRESS, core.Env.INTERNAL_APIKEY); err != nil {
		panic("invalid fareward configuration: " + err.Error())
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

	if core.Env.IS_DEV_ARG {
		core.Env.LOGS_FULL = true
	}

	// Mirror runtime logging flags into db so query debug logs follow the
	// resolved environment: LOGS_FULL → level 2 (verbose), IS_DEV_ARG → level
	// 1 (basic), otherwise silent.
	dbLogLevel := 0
	/*
		if core.Env.IS_DEV_ARG {
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
		if !core.Env.IS_DEV_ARG {
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
		// SSE bridge (fareward/) instead of this process.
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

		// A deployed binary keeps binding every interface: nginx, the health check and the Function
		// URL all reach it by a different address, and narrowing that is a deploy decision, not this
		// one. Only a dev launch restricts itself. See resolveDevListenAddresses.
		if !core.Env.IS_DEV_ARG {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				core.Log("HTTP server error:", err)
			}
			return
		}

		// One http.Server, one listener per address: Serve can be called concurrently on the same
		// server, and sharing it keeps the timeouts and the ConnState metrics identical on both.
		devListeners := []net.Listener{}
		for _, listenAddress := range resolveDevListenAddresses(serverPort) {
			listener, err := net.Listen("tcp", listenAddress)
			if err != nil {
				// Not fatal on its own: losing the tailnet address still leaves a usable loopback
				// backend, and the address that failed is named so the cause is visible.
				core.Log("No se pudo escuchar en", listenAddress, "::", err)
				continue
			}
			core.Log("Escuchando en", listenAddress)
			devListeners = append(devListeners, listener)
		}
		if len(devListeners) == 0 {
			core.Log("HTTP server error: ninguna dirección de desarrollo pudo abrirse")
			return
		}

		serveErrors := make(chan error, len(devListeners))
		for _, listener := range devListeners {
			go func(listener net.Listener) { serveErrors <- srv.Serve(listener) }(listener)
		}
		if err := <-serveErrors; err != nil && err != http.ErrServerClosed {
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
