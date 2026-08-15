package main

import (
	"app/agent"
	"app/business"
	"app/config"
	"app/core"
	"app/exec"
	"app/finance"
	"app/logistics"
	"app/sales"
	"app/security"
	"app/webpage"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"time"
)

// Agrupa los handlers de los módulos en uns solo map
var appHandlersModules = []core.AppRouterType{
	agent.ModuleHandlers,
	exec.ModuleHandlers,
	sales.ModuleHandlers,
	config.ModuleHandlers,
	webpage.ModuleHandlers,
	finance.ModuleHandlers,
	logistics.ModuleHandlers,
	business.ModuleHandlers,
	security.ModuleHandlers,
}

var appHandlers = core.AppRouterType{}

// Obtiene los handlers
func makeAppHandlers() *core.AppRouterType {
	if len(appHandlers) == 0 {
		for _, moduleHandlers := range appHandlersModules {
			for path, handlerFunc := range moduleHandlers {
				appHandlers[path] = handlerFunc
			}
		}
	}
	return &appHandlers
}

// Handler principal (para lambda y para local)
var apiNames = []string{"api", "go1", "go2", "go3", "go4", "go5"}

// Keep the YAML embedded in the main package because the source file remains in backend/.
//
//go:embed access_list.yml
var accessListYamlContent []byte

// The helper lives in core, but the main package owns the embedded bytes and injects them once.
var accessHelper = func() *core.AccessHelper {
	return core.LoadEmbeddedAccessList(accessListYamlContent)
}()

// saasCompanyID identifica a la company dueña de la plataforma: la única que opera el módulo SYSTEM.
const saasCompanyID = 1

// Rutas del módulo SYSTEM (Empresas, Server Panel, Acciones Cron). Administran la plataforma
// entera —no un tenant— así que se restringen a la company dueña del SaaS. Quedan fuera a
// propósito: "company-parametros" (es "Mi Empresa", la usa cada tenant), "p-company-names-by-ids"
// (pública, previa al login) y "system-parameters" (particionada por company, la usa el POS).
var saasOnlyRoutes = map[string]bool{
	"GET.empresas":               true,
	"POST.company":               true,
	"GET.system-metrics-stream":  true,
	"GET.system-memory-packages": true,
	"GET.cron-actions-scheduled": true,
}

// Rutas POST que exigen sesión pero ningún acceso del catálogo: el usuario opera sobre sí mismo,
// así que un perfil restringido igual debe poder usarlas (editar su perfil, cambiar su password).
// El handler es el responsable de forzar el ámbito propio —PostUsuarios ignora el ID del body
// cuando la ruta es "user-self"—; esta lista sólo salta la comprobación de accesos.
var selfServiceRoutes = map[string]bool{
	"POST.user-self": true,
}

func mainHandler(args *core.HandlerArgs) (response core.MainResponse) {
	requestStartedAt := time.Now().UnixMilli()
	setResponseMetadata := func(handlerResponse *core.HandlerResponse) {
		if handlerResponse == nil {
			return
		}
		if handlerResponse.Headers == nil {
			handlerResponse.Headers = map[string]string{}
		}

		// Persist both timings so the transport layer can append the final total after encoding.
		handlerResponse.RequestStart = requestStartedAt
		handlerResponse.PreSerializeMs = time.Now().UnixMilli() - requestStartedAt
	}

	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Internal Server Error (Panic): %v", r)
			core.Logx(5, errStr)
			core.Log(string(debug.Stack()))

			handlerResponse := core.HandlerResponse{
				Error:      errStr,
				StatusCode: 500,
			}
			setResponseMetadata(&handlerResponse)
			response = prepareResponse(args, &handlerResponse)
		}
	}()

	// coloca algunas variables de entorno que pueden ser utilizadas por otros handlers
	args.Authorization = core.MapGetKeys(args.Headers, "Authorization", "authorization")
	args.Encoding = core.MapGetKeys(args.Headers, "Accept-Encoding", "accept-encoding")

	// Minted before anything else can fail, so even a request rejected at the token check carries
	// an identity into its user_logs row. Lives on args, which is exact in both runtimes; the Env
	// mirror that core.Log prints from is set further down, once the identity is known.
	args.RequestID = core.MakeRequestID()
	core.ResetRequestErrors()

	// NOTE: The Lambda runtime processes one invocation at a time per execution environment,
	// but local/VPS HTTP mode is concurrent. Avoid mutating global per-request state when local.
	if core.Env.IS_SERVERLESS {
		core.StartTime = (time.Now()).UnixMilli()
		core.Env.REQ_PARAMS = core.MapGetKeys(args.Headers, "x-api-key", "X-Api-Key")
		core.Env.REQ_USER_AGENT = core.MapGetKeys(args.Headers, "User-Agent", "user-agent")
		core.Env.REQ_ID = core.IntToBase64(time.Now().UnixMilli())
	}

	core.Log(args.Route)

	if args.Route[0] == '/' {
		args.Route = args.Route[1:]
	}

	pathSegments := strings.Split(args.Route, "/")
	if slices.Contains(apiNames, pathSegments[0]) {
		args.Route = strings.Join(pathSegments[1:], "/")
	}

	core.Log("Route:", args.Route)
	handlerResponse := core.HandlerResponse{Encoding: args.Encoding}

	// Los es públicos comienzan con "p-" y no necesitan validacion del user Tocken
	isPublicPath := len(args.Route) > 2 && args.Route[0:2] == "p-"
	funcPath := args.Method + "." + args.Route
	args.RouteID = core.APIRouteID(funcPath)
	core.SetLogRequest(args.RequestID, args.RouteID)

	if !isPublicPath {
		args.User = core.CheckUser(args, 0)
		core.SetLogUser(args.User.CompanyID, args.User.ID)

		// Si no es público, valida el user
		if len(args.User.Error) > 0 {
			core.Log("User Error::", args.User.Error)
			handlerResponse.Error = args.User.Error
			setResponseMetadata(&handlerResponse)
			return prepareResponse(args, &handlerResponse)
		}

		// Se valida antes del catálogo de accesos porque un perfil no puede otorgar esto: el
		// usuario 1 de cualquier company salta la comprobación de accesos, pero no esta.
		if saasOnlyRoutes[funcPath] && args.User.CompanyID != saasCompanyID {
			core.Log("Ruta SaaS rechazada::", funcPath, "company::", args.User.CompanyID)
			handlerResponse.Error = "Esta operación sólo está disponible para la company administradora de la plataforma."
			setResponseMetadata(&handlerResponse)
			return prepareResponse(args, &handlerResponse)
		}

		// Las rutas privadas siempre se limitan a la company del token. Se descarta cualquier "cmp"
		// enviado por el cliente para que un query param no pueda apuntar a la partición de otra
		// company; los handlers caen entonces a args.User.CompanyID. Las rutas públicas ("p-") sí
		// necesitan leer "cmp", por eso esto vive solo en esta rama.
		delete(args.Query, "cmp")

		accessInfos, _ := accessHelper.GetAccesosByRoute(funcPath)
		// Un GET sin acceso mapeado es libre para cualquier sesión; POST/PUT siempre exigen uno. Pero
		// un GET que SÍ figura en el catálogo se comporta como el resto: hay que tener el acceso. Así
		// se cierra lectura por lectura, sin bloquear de golpe los GET que todavía no están mapeados.
		hasAllowedAccess := (args.Method == "GET" && len(accessInfos) == 0) || selfServiceRoutes[funcPath]

		nivel := uint8(1)
		if args.Method == "POST" || args.Method == "PUT" {
			nivel = 2
			if len(accessInfos) == 0 && !selfServiceRoutes[funcPath] {
				core.Log(fmt.Sprintf("Warning: La ruta \"%v\" está desprotegida.", funcPath))
			}
		}

		for _, accessInfo := range accessInfos {
			if args.HasAccesoNivel(accessInfo.ID, nivel) {
				core.Log("Acceso:", funcPath, "con", accessInfo.ID, ":", nivel)
				hasAllowedAccess = true
				break
			}
		}

		if !hasAllowedAccess && args.User.ID != 1 {
			accessNames := []string{}
			for _, accessInfo := range accessInfos {
				accessNames = append(accessNames, accessInfo.Name)
			}

			handlerResponse.Error = fmt.Sprintf("El user no posee alguno de los accesos: %s", strings.Join(accessNames, ", "))
			setResponseMetadata(&handlerResponse)
			return prepareResponse(args, &handlerResponse)
		}
	} else {
		args.User = &core.UsuarioToken{}
	}

	// Request header log is only meaningful in Lambda mode (it uses REQ_ID / REQ_LAMBDA_ID).
	//
	// It carries the same three index tokens every following line of this request carries, so a
	// filter on c7 or r118 returns the header alongside the body of the request rather than
	// stranding one without the other. The route names replace the FnvHashString64 list that used
	// to live here: with a number on every line, the header is where the number is spelled out.
	if core.Env.IS_SERVERLESS {
		logHeader := core.Concat("|", "$Req", core.Env.REQ_ID, args.RequestID, core.Env.REQ_LAMBDA_ID,
			"c"+core.Concats(args.User.CompanyID),
			"u"+core.Concats(args.User.CompanyID)+"_"+core.Concats(args.User.ID),
			"r"+core.Concats(args.RouteID),
			args.User.User, strings.Join(core.REQ_PATHS, "&"))
		fmt.Println(logHeader)
	}

	handlerFunc, ok := appHandlers[funcPath]
	if !ok {
		core.Log("no hay una lambda para el path solicitado::", funcPath)
		handlerResponse.Error = "no hay una lambda para el path solicitado: " + funcPath
	} else {
		if !isPublicPath && args.Method == "POST" {
			requestBodyBytes := 0
			if args.Body != nil {
				requestBodyBytes = len(*args.Body)
			}
			if rateLimitResponse := enforceAPICreditLimit(args, requestBodyBytes); rateLimitResponse != nil {
				handlerResponse = *rateLimitResponse
				setResponseMetadata(&handlerResponse)
				return prepareResponse(args, &handlerResponse)
			}
		}

		core.Log("Ejecutando Handler::", funcPath)
		handlerResponse = handlerFunc(args)
		respLen := 0
		if handlerResponse.Body != nil {
			respLen = len(*handlerResponse.Body)
		}
		core.Log("Finalizado Handler::", funcPath, " | Len: ", respLen)
		if !isPublicPath && args.Method == "GET" && handlerResponse.Error == "" && !handlerResponse.StreamHandled {
			if rateLimitResponse := enforceAPICreditLimit(args, respLen); rateLimitResponse != nil {
				handlerResponse = *rateLimitResponse
			}
		}

		if !core.Env.IS_SERVERLESS && !core.Env.IS_LOCAL {
			registerLocalRequestUsage(args, &handlerResponse, requestStartedAt)
		}
	}

	setResponseMetadata(&handlerResponse)
	return prepareResponse(args, &handlerResponse)
}

func enforceAPICreditLimit(args *core.HandlerArgs, payloadBytes int) *core.HandlerResponse {
	requestContext := context.Background()
	if args.ReqContext != nil {
		requestContext = args.ReqContext.Context()
	}
	err := core.ChargeAPIUsage(
		requestContext, args.User.CompanyID, args.User.ID, args.Method, payloadBytes,
	)
	if err != nil {
		core.Log("credit rate limiter rejected::", " method::", args.Method, " company::", args.User.CompanyID,
			" user::", args.User.ID, " bytes::", payloadBytes, " err::", err)
		response := args.MakeCreditRateLimitResponse(err)
		return &response
	}
	apiGroup, _ := core.APIGroup(args.Method, payloadBytes)
	cpuCredits, _ := core.APICPUCredits(args.Method, payloadBytes)
	core.Log("credit rate limiter accepted::", " method::", args.Method, " company::", args.User.CompanyID,
		" user::", args.User.ID, " bytes::", payloadBytes, " api_group::", apiGroup, " cpu_credits::", cpuCredits)
	return nil
}

func registerLocalRequestUsage(args *core.HandlerArgs, handlerResponse *core.HandlerResponse, requestStartedAt int64) {
	companyID, userID := int32(0), int32(0)

	if args.User != nil {
		companyID, userID = args.User.CompanyID, args.User.ID
	}
	if args.User.CompanyID <= 0 {
		return
	}

	requestType := core.If(args.Method == "GET", int8(1), 2)
	bandwidthTotalBytes := 0

	if args.Body != nil {
		bandwidthTotalBytes += len(*args.Body)
	}

	if handlerResponse.Body != nil {
		bandwidthTotalBytes = len(*handlerResponse.Body)
	}

	bandwidthUnits := int32((bandwidthTotalBytes + 4095) / 4096)
	elapsedMilliseconds := time.Now().UnixMilli() - requestStartedAt
	usageTimeUnits := int32((elapsedMilliseconds + 3) / 4)

	core.AddRequestUsage(companyID, userID, bandwidthUnits, usageTimeUnits, requestType)
}

func clearEnvVariables() {
	core.User = core.UsuarioToken{}
	core.LogsSaved = []string{}
	core.REQ_PATHS = []string{}
	core.Env.USUARIO_ID = 0
	core.Env.LOGS_ONLY_SAVE = false
	core.Env.LOGS_FULL = false
}

type ExecLambdaInput struct {
	ExecArgs core.ExecArgs `json:"fn_exec"`
}

func ExecFuncHandler(lambdaInput string) (response core.FuncResponse) {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Internal Server Error (Panic in ExecFuncHandler): %v", r)
			core.Logx(5, errStr)
			core.Log(string(debug.Stack()))
			response = core.FuncResponse{Error: errStr}
		}
	}()
	core.Env.LOGS_ONLY_SAVE = true

	input := ExecLambdaInput{}
	err := json.Unmarshal([]byte(lambdaInput), &input)
	if err != nil {
		return core.FuncResponse{
			Error: "no se pudieron interpretar los argumentos recibidos: " + core.StrCut(lambdaInput, 200),
		}
	}

	args := input.ExecArgs
	core.Log("func to exec:: ", core.StrCut(lambdaInput, 200))

	// Función a ejecutarse con nombre específico
	if len(args.FuncToExec) > 0 {
		for key := range exec.ExecHandlers {
			if args.FuncToExec == key {
				core.Log("invocando funcion:: ", args.FuncToExec)
				return exec.ExecHandlers[key](&args)
			}
		}
		core.Log("No se encontró la función e ejecutar::", args.FuncToExec)
	}
	return core.FuncResponse{}
}

func prepareResponse(args *core.HandlerArgs, handlerResponse *core.HandlerResponse) core.MainResponse {
	response := core.MainResponse{}

	// Emitted here and nowhere else: this is the one funnel every response passes through, panics
	// and token rejections included, and it runs in both runtimes. It is deliberately ahead of the
	// branches below, two of which return early — a streamed response is still a request that
	// happened, and a request that failed at the token check is one of the more interesting rows
	// in the table.
	elapsedMs := int64(0)
	if handlerResponse.RequestStart > 0 {
		elapsedMs = time.Now().UnixMilli() - handlerResponse.RequestStart
	}
	core.EmitRequestLog(args, elapsedMs)

	if !core.Env.IS_SERVERLESS {
		// Stream handlers (SSE) write directly to ResponseWriter and must bypass normal compression flow.
		if handlerResponse.StreamHandled {
			return response
		}
		// core.Print(handlerResponse)
		core.SendLocalResponse(*args, *handlerResponse)
		return response
	}

	hasError := len(handlerResponse.Error) > 0
	statusCode := int32(400)
	if handlerResponse.StatusCode != 0 {
		statusCode = int32(handlerResponse.StatusCode)
	}

	// Under RESPONSE_STREAM the body leaves as raw bytes behind a JSON prelude, so it is a
	// different response type entirely rather than a variation of the buffered one.
	if core.Env.LAMBDA_RESPONSE_STREAMING {
		if hasError {
			response.LambdaStreamingResponse = core.MakeErrStreamingFinal(statusCode, handlerResponse.Error)
			for headerName, headerValue := range handlerResponse.Headers {
				response.LambdaStreamingResponse.Headers[headerName] = headerValue
			}
			response.LambdaStreamingResponse.Headers["X-Metadata"] = fmt.Sprintf("%d,%d",
				handlerResponse.PreSerializeMs,
				time.Now().UnixMilli()-handlerResponse.RequestStart,
			)
		} else {
			response.LambdaStreamingResponse = core.MakeStreamingResponseFinal(handlerResponse)
		}
		return response
	}

	if hasError {
		response.LambdaResponse = core.MakeErrRespFinal(statusCode, handlerResponse.Error)
		if response.LambdaResponse.Headers == nil {
			response.LambdaResponse.Headers = map[string]string{}
		}
		for headerName, headerValue := range handlerResponse.Headers {
			response.LambdaResponse.Headers[headerName] = headerValue
		}
		response.LambdaResponse.Headers["Access-Control-Expose-Headers"] = "X-Metadata, X-Rate-Limit-Code"
		response.LambdaResponse.Headers["X-Metadata"] = fmt.Sprintf("%d,%d",
			handlerResponse.PreSerializeMs,
			time.Now().UnixMilli()-handlerResponse.RequestStart,
		)
	} else {
		response.LambdaResponse = core.MakeResponseFinal(handlerResponse)
	}
	return response
}

func ExecFunc(
	funcToInvoke func(*core.ExecArgs) core.FuncResponse,
	args *core.ExecArgs,
	secondsTimeout time.Duration,
) core.FuncResponse {

	nowTime := time.Now().Unix()
	chanResult := make(chan core.FuncResponse, 1)
	funcResponse := core.FuncResponse{}

	go func() {
		defer func() {
			// recover from panic
			if r := recover(); r != nil {
				funcResponse.Error = fmt.Sprintf("Error (PANIC): %v", r)
			}
		}()
		chanResult <- funcToInvoke(args)
	}()

	hasTimeout := false

	select {
	case <-time.After(secondsTimeout * time.Second):
		hasTimeout = true
	case funcResponse = <-chanResult:
	}

	funcResponse.ElapsedTime = int(time.Now().Unix() - nowTime)

	if hasTimeout {
		funcResponse.Error = fmt.Sprintf("Error (TIMEOUT): %vs", funcResponse.ElapsedTime)
	}

	return funcResponse
}
