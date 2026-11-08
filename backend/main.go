//go:build !js

package main

import (
	"app/business"
	"app/core"
	"app/db"
	"app/exec"
	"context"
	"fmt"
	fareward "github.com/ivanjoz/fareward/go"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/kr/pretty"
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

// runLambdaRequest is the request path shared by both invoke modes. It stops at the finished
// core.HandlerResponse, so the only difference between the buffered and streaming entrypoints is
// which AWS shape they render it into.
func runLambdaRequest(request *events.APIGatewayV2HTTPRequest) *core.HandlerResponse {
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
		return &core.HandlerResponse{Body: &bodyBytes, Headers: map[string]string{}}
	}

	// Tick programado de EventBridge. No hay ruta HTTP detrás, así que no puede seguir al
	// mainHandler: sin Authorization moriría en CheckUser.
	if strings.HasPrefix(request.Body, scheduledCronTickPrefix) {
		bodyBytes := []byte(core.ToJsonNoErr(map[string]int{"executed": runScheduledCronTick(request.Body)}))
		return &core.HandlerResponse{Body: &bodyBytes, Headers: map[string]string{}}
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

	return core.MakeLambdaResponse(runLambdaRequest(request)), nil
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

	return core.MakeLambdaStreamingResponse(runLambdaRequest(request)), nil
}

// serializedAPIRequests is the queue behind config.toml's disable_api_concurrency_local. The lock
// is taken for the whole request, so the log of one API is never interleaved with another's.
var serializedAPIRequests sync.Mutex

// shouldSerializeRequest keeps the queue away from the shape that would turn it into a hang.
//
// A `-stream` route (SSE) holds its connection until the client goes away: it would own the lock
// for minutes while every other request piles up behind it. It is also the shape whose logs are
// already self-contained, which is what the flag is for. Everything else — the delta syncs a page
// load fires in parallel — queues.
func shouldSerializeRequest(routePath string) bool {
	if !core.Env.DISABLE_API_CONCURRENCY_LOCAL {
		return false
	}
	return !strings.HasSuffix(routePath, "-stream")
}

func LocalHandler(w http.ResponseWriter, request *http.Request) {
	if shouldSerializeRequest(request.URL.Path) {
		serializedAPIRequests.Lock()
		defer serializedAPIRequests.Unlock()
	}

	// A request slot before any work. An SSE route holds its connection for minutes and would pin
	// a slot the whole time, so it is the one exception.
	if !strings.HasSuffix(request.URL.Path, "-stream") {
		releaseSlot, isAdmitted := core.AdmitRequest(request.Context())
		if !isAdmitted {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"El servidor está ocupado. Intente nuevamente."}`))
			return
		}
		defer releaseSlot()
	}

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

	// Per request, on args only: this server handles many requests at once, so the Env.REQ_IP
	// global the Lambda path sets would be a data race here.
	clientIP := core.ClientIPFromRequest(request)
	// Counted so a stream handler, which writes here directly, still has a size to be charged by.
	var responseWriter http.ResponseWriter = core.NewCountingResponseWriter(w)

	args := core.HandlerArgs{
		Body:           &body,
		Method:         strings.ToUpper(request.Method),
		Route:          request.URL.Path,
		ClientIP:       clientIP,
		ResponseWriter: &responseWriter,
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
	// cannot import core (cycle), the same as the text search backends.
	fareward.SetLogger(core.Log)
	// Past the daemon's per-connection cap a frame is refused, not queued; the client queues at it.
	fareward.SetMaxInflightPerConnection(int(core.Env.FAREWARD_MAX_INFLIGHT))
	// The client dials on first use: the SQL builds own their sequences, so under [fareward].bypass
	// they never dial it for one.
	if err := core.ConfigureFareward(core.Env.FAREWARD_ADDRESS, core.Env.INTERNAL_APIKEY); err != nil {
		panic("invalid fareward configuration: " + err.Error())
	}
	if core.Env.FAREWARD_BYPASS {
		core.Log("WARNING: [fareward].bypass is on: no credit charges, no access checks, no request log (load tests only)")
	}
	serverPort := resolveServerPort()
	// Install the text search backend before any DB write that might touch a table with TextSearch.
	core.ConfigureTextSearch()
	// The delta logs and the by-IDs slot files, written by every write to a cached table.
	core.ConfigureFileStore()
	// Print deployment path early so systemd logs show which cloned repo the
	// binary will scan for route markdown and generated menu descriptions.
	fmt.Println("GENIX_REPOSITORY_ROOT=", os.Getenv("GENIX_REPOSITORY_ROOT"))
	makeAppHandlers()
	// The edge enforces this process's own gate, so it gets the table this binary resolves. A failure
	// leaves the daemon on the previous table, which may not match this binary: loud, not fatal.
	if core.Env.FAREWARD_ROUTES_FILE != "" && !core.Env.IS_SERVERLESS {
		if err := writeFarewardRouteFile(core.Env.FAREWARD_ROUTES_FILE, appHandlers); err != nil {
			core.Log("ERROR: the fareward route table could not be written::", core.Env.FAREWARD_ROUTES_FILE, "::", err)
		} else {
			core.Log("fareward route table written::", core.Env.FAREWARD_ROUTES_FILE)
		}
	}
	fmt.Println("Setting full logs...")
	// os.Setenv("LOGS_FULL", "1")

	if core.Env.IS_SERVERLESS { // Controla los panic error
		defer func() {
			if r := recover(); r != nil {
				OnPanic(r)
			}
		}()
	}

	fmt.Printf("Starting DB connection. MariaDB %v:%v ..."+"\n", core.Env.MARIADB_HOST, core.Env.MARIADB_PORT)

	db.Connect(db.ConnectParams{
		Keyspace: core.Env.DB_NAME,
		OceanBase: db.OceanBaseParams{
			Host: core.Env.OCEANBASE_HOST, Port: core.Env.OCEANBASE_PORT,
			User: core.Env.OCEANBASE_USER, Password: core.Env.OCEANBASE_PASSWORD,
			Partitions: core.Env.OCEANBASE_PARTITIONS, MaxConns: core.Env.OCEANBASE_MAX_CONNS,
			Engine: core.Env.OCEANBASE_ENGINE,
		},
		Postgres: db.PostgresParams{
			Host: core.Env.POSTGRES_HOST, Port: core.Env.POSTGRES_PORT,
			User: core.Env.POSTGRES_USER, Password: core.Env.POSTGRES_PASSWORD,
			Database: core.Env.POSTGRES_DATABASE, MaxConns: core.Env.POSTGRES_MAX_CONNS,
		},
		MariaDB: db.MariaDBParams{
			Host: core.Env.MARIADB_HOST, Port: core.Env.MARIADB_PORT,
			User: core.Env.MARIADB_USER, Password: core.Env.MARIADB_PASSWORD, MaxConns: core.Env.MARIADB_MAX_CONNS,
			Socket: core.Env.MARIADB_SOCKET,
		},
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

	// GENIX_DB_LOG turns the ORM's own logging on for a debugging session without touching code:
	// 1 prints every statement with its values inlined — which is what makes a delta's watermark
	// predicate visible — and 2 adds the verbose internal traces. Unset means silent, because at
	// level 1 a page load prints one line per query.
	dbLogLevel, _ := strconv.Atoi(os.Getenv("GENIX_DB_LOG"))
	if dbLogLevel > 0 {
		fmt.Println("ORM logging habilitado. GENIX_DB_LOG =", dbLogLevel)
	}
	db.SetDebugLogging(dbLogLevel)

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
			// The command's result is its output: printed whatever the log level.
			pretty.Println(funcResponse.Content)
		}
		return
	}

	// Si se está desarrollando en local
	if !core.Env.IS_SERVERLESS {
		// The VPS server is one long-lived process: it may reserve table keys in blocks.
		db.UseTableKeyBlocks()
		// GENIX_CPU_PROFILE records the CPU profile the release build is optimized with (PGO):
		// run a load test against this process, stop it, and save the file as backend/default.pgo.
		if cpuProfilePath := strings.TrimSpace(os.Getenv("GENIX_CPU_PROFILE")); cpuProfilePath != "" {
			startCPUProfileUntilSignal(cpuProfilePath)
		}
		core.ConfigureServerAdmission()
		exec.StartUsageLogFlushWorker()
		if !core.Env.IS_DEV_ARG {
			// Off the main goroutine on purpose: ScheduleCronAction panics when its query fails,
			// and on a VPS (IS_SERVERLESS=false) no recover is installed, so a database that is
			// briefly unreachable at boot would otherwise stop the HTTP listener from ever
			// starting. The listener must come up whether or not the cron seed succeeds.
			go bootstrapCronSchedulers()
		}

		// "*" forces the line out: outside dev mode core.Log prints only failures.
		core.Log("*Ejecutando en local. http://localhost" + serverPort)

		corsMiddleware := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{http.MethodPost, http.MethodPut, http.MethodGet},
			AllowedHeaders:   []string{"*"},
			ExposedHeaders:   []string{"X-Metadata", "X-Rate-Limit-Code"},
			AllowCredentials: false,
		})

		// The agent's stream and routes (/agent/*, p-agent-turn) are not served here: they live in
		// their own process, genix/agent.
		mux := http.NewServeMux()
		mux.Handle("/", corsMiddleware.Handler(http.HandlerFunc(LocalHandler)))

		// Inicia el servidor con timeouts (previene slowloris y mejora resiliencia).
		srv := &http.Server{
			Addr:              serverPort,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			// Keep disabled to allow long-lived SSE streams (metrics and future real-time endpoints).
			WriteTimeout: 0,
			// Must outlive nginx's upstream keepalive_timeout (100s), so nginx is always the side
			// that closes an idle pooled connection and never reuses one Go already dropped.
			IdleTimeout:    120 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1 MiB
			// Track active connections for operational metrics and SSE dashboards.
			ConnState: func(connection net.Conn, currentState http.ConnState) {
				core.UpdateHTTPConnectionState(connection, currentState)
			},
		}

		// On the VPS, nginx shares the host and reaches the backend through a Unix socket: no TCP
		// handshake or port allocation per upstream connection, and no port open on the public IP.
		// systemd sets SERVER_SOCKET and creates its directory (RuntimeDirectory=genix).
		if serverSocketPath := strings.TrimSpace(os.Getenv("SERVER_SOCKET")); serverSocketPath != "" && !core.Env.IS_DEV_ARG {
			// A socket file left by a crashed run makes Listen fail with "address already in use".
			os.Remove(serverSocketPath)
			socketListener, err := net.Listen("unix", serverSocketPath)
			if err != nil {
				core.Log("HTTP server error: no se pudo abrir el socket", serverSocketPath, "::", err)
				return
			}
			// nginx workers run as another user and need write permission to connect. The socket
			// is local-only, so world-writable exposes nothing beyond this host.
			if err := os.Chmod(serverSocketPath, 0o666); err != nil {
				core.Log("HTTP server error: no se pudo dar permisos al socket", serverSocketPath, "::", err)
				return
			}
			core.Log("*Escuchando en unix:" + serverSocketPath)
			if err := srv.Serve(socketListener); err != nil && err != http.ErrServerClosed {
				core.Log("HTTP server error:", err)
			}
			return
		}

		// Without a socket (a split-host install) the binary binds every interface: nginx reaches it
		// by a different address, and narrowing that is a deploy decision, not this one. Only a dev
		// launch restricts itself. See resolveDevListenAddresses.
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
			core.Log("*Escuchando en", listenAddress)
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

// startCPUProfileUntilSignal profiles the whole process until SIGINT or SIGTERM, then writes the
// profile and exits. Only for profiling runs (GENIX_CPU_PROFILE): a profiled server does not shut
// down gracefully.
func startCPUProfileUntilSignal(cpuProfilePath string) {
	profileFile, err := os.Create(cpuProfilePath)
	if err != nil {
		core.Log("CPU profile: no se pudo crear", cpuProfilePath, "::", err)
		return
	}
	if err := pprof.StartCPUProfile(profileFile); err != nil {
		core.Log("CPU profile: no se pudo iniciar ::", err)
		return
	}
	core.Log("*CPU profile activo:", cpuProfilePath)
	stopSignals := make(chan os.Signal, 1)
	signal.Notify(stopSignals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopSignals
		pprof.StopCPUProfile()
		profileFile.Close()
		os.Exit(0)
	}()
}
