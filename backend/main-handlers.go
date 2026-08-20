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
	"GET.empresas":                    true,
	"POST.company":                    true,
	"GET.system-metrics-stream":       true,
	"GET.server-metrics":              true,
	"GET.observability":               true,
	"GET.company-credit-usage-report": true,
	"GET.company-credit-usage-detail": true,
	"GET.company-credit-usage-users":  true,
	"GET.company-credit-budget":       true,
	"POST.company-credit-budget":      true,
	"GET.request-errors-by-ids":       true,
	"GET.system-memory-packages":      true,
	"GET.cron-actions-scheduled":      true,
}

// Rutas exentas del cobro y del límite de créditos: son las que se usan para *ver o corregir* el
// consumo, así que cobrarlas las volvería inalcanzables justo cuando hacen falta. "credit-usage" es
// la lectura propia de cada tenant (panel Config. → Créditos): sin la exención, un usuario con el
// límite agotado no puede ni averiguar por qué. Las otras tres son el control de presupuesto del
// SaaS más las dos lecturas necesarias para llegar a él. Los reportes de detalle y las escrituras
// de company siguen cobrándose.
var creditControlRoutes = map[string]bool{
	"GET.credit-usage":                true,
	"GET.empresas":                    true,
	"GET.company-credit-usage-report": true,
	"GET.company-credit-budget":       true,
	"POST.company-credit-budget":      true,
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

		// Las dos preguntas que deciden la request, en un solo viaje a server_utils: si el user
		// tiene el acceso que la ruta exige y si a su company le quedan créditos.
		//
		// Corre antes de buscar el handler, así que una ruta que no existe también se cobra: un GET
		// sin mapear a un path inexistente paga sus dos créditos base y después recibe "no hay una
		// lambda". Es a propósito —escanear rutas inexistentes deja de ser gratis, y el daemon lo
		// imputa a la ruta 0, que el protocolo cobra deliberadamente— pero antes era gratis, porque
		// el cargo vivía dentro de la rama en que el handler sí existe.
		if limitResponse := enforceAccessAndCredits(args, funcPath); limitResponse != nil {
			handlerResponse = *limitResponse
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
		core.Log("Ejecutando Handler::", funcPath)
		handlerResponse = handlerFunc(args)
		respLen := 0
		if handlerResponse.Body != nil {
			respLen = len(*handlerResponse.Body)
		}
		core.Log("Finalizado Handler::", funcPath, " | Len: ", respLen)
		// Liquidación del GET: sólo lo que excede el primer bloque, que para un GET significa sólo
		// cuando la respuesta pasó de 8 KB. La mayoría no llega, así que la mayoría no manda un
		// segundo frame. Un error o un stream no liquidan: esos bytes no se enviaron.
		//
		// La condición se lee de chargedMethodFor y no se reconstruye: es la misma función que
		// decidió cobrar la base, así que las dos mitades del cargo no pueden discrepar.
		if !isPublicPath && chargedMethodFor(args.Method, funcPath) == "GET" &&
			handlerResponse.Error == "" && !handlerResponse.StreamHandled {
			if topUpResponse := chargeGetResponseTopUp(args, respLen); topUpResponse != nil {
				handlerResponse = *topUpResponse
			}
		}

		if !core.Env.IS_SERVERLESS && !core.Env.IS_LOCAL {
			registerLocalRequestUsage(args, &handlerResponse, requestStartedAt)
		}
	}

	setResponseMetadata(&handlerResponse)
	return prepareResponse(args, &handlerResponse)
}

// routeAccessDecision es lo que el gate concluyó sobre la autorización de una request: los accesos
// empaquetados que server_utils debe verificar, o un rechazo que no necesita preguntarle nada.
type routeAccessDecision struct {
	requiredAccess []uint16
	// accessNames acompaña a requiredAccess sólo para el mensaje de error: el daemon no conoce
	// access_list.yml, así que los nombres salen de este lado.
	accessNames []string
	denyMessage string
	denyCode    int32
}

// resolveRouteAccess traduce el catálogo a lo que viaja en el frame. Es una función pura sobre
// (método, ruta, user, accesos mapeados) justamente porque es la decisión de seguridad del router:
// toda la política vive aquí, en el proceso que embebe access_list.yml, y el daemon sólo responde
// "este user tiene alguno de estos accesos".
//
// Devolver una lista vacía significa "no preguntes", así que cada camino que no exige acceso tiene
// que ser deliberado. Son tres: un GET sin mapear, una ruta self-service, y el user 1.
func resolveRouteAccess(
	method, funcPath string, userID int32, accessInfos []core.AccessInfo,
) routeAccessDecision {
	// Un GET sin acceso mapeado es libre para cualquier sesión; POST/PUT siempre exigen uno. Pero un
	// GET que SÍ figura en el catálogo se comporta como el resto: hay que tener el acceso. Así se
	// cierra lectura por lectura, sin bloquear de golpe los GET que todavía no están mapeados.
	if (method == "GET" && len(accessInfos) == 0) || selfServiceRoutes[funcPath] {
		return routeAccessDecision{}
	}
	// El user 1 salta la comprobación, y esto NO es una comodidad: login.go le sintetiza la lista
	// completa de accesos sólo en la respuesta de login y nunca la persiste, así que su blob en
	// ScyllaDB está vacío y el daemon lo negaría. No se le puede preguntar.
	if userID == 1 {
		return routeAccessDecision{}
	}
	// Sin accesos mapeados no hay nada que preguntarle al daemon, y el catálogo niega por defecto lo
	// que no figura: un POST/PUT sin mapear se rechaza aquí mismo. Delegarlo al frame lo habría
	// convertido en una ruta abierta, porque un frame sin accesos requeridos es exactamente un frame
	// que no pide autorización.
	if len(accessInfos) == 0 {
		return routeAccessDecision{
			denyMessage: fmt.Sprintf(
				"La ruta \"%v\" no declara accesos, así que nadie puede usarla.", funcPath),
			denyCode: 403,
		}
	}
	// Una ruta mapeada a más accesos de los que caben en un frame es un error de configuración, no
	// del cliente: truncar la lista autorizaría contra menos accesos de los declarados, así que
	// negar es lo único seguro. TestEveryRouteFitsTheRequiredAccessSlots lo detecta antes.
	if len(accessInfos) > core.MaxRequiredAccess {
		return routeAccessDecision{
			denyMessage: "La ruta declara más accesos de los que el limitador puede verificar.",
			denyCode:    500,
		}
	}

	nivel := uint8(1)
	if isWriteMethod(method) {
		nivel = 2
	}
	decision := routeAccessDecision{
		requiredAccess: make([]uint16, 0, len(accessInfos)),
		accessNames:    make([]string, 0, len(accessInfos)),
	}
	for _, accessInfo := range accessInfos {
		decision.requiredAccess = append(
			decision.requiredAccess, core.MakeAccesoNivelPacked(accessInfo.ID, nivel))
		decision.accessNames = append(decision.accessNames, accessInfo.Name)
	}
	return decision
}

// isWriteMethod: POST y PUT son la misma cosa en todo lo que este router decide —exigen nivel 2 y
// pagan la misma tarifa sobre el body—, así que la pareja se declara una vez y se usa en todos lados.
// Tenerla escrita dos veces es exactamente lo que dejó a PUT autorizado pero sin cobrar.
func isWriteMethod(method string) bool {
	return method == "POST" || method == "PUT"
}

// chargedMethodFor decide qué se cobra en el frame; vacío significa "sólo autoriza".
//
// Sólo un caso no se cobra, y es deliberado: las creditControlRoutes son el panel de créditos
// leyéndose a sí mismo, y cobrarlas las volvería inalcanzables justo cuando hacen falta. Igual
// mandan frame, porque la exención salta el cobro y nunca la autorización.
//
// Cualquier otro método —uno que no lea ni escriba— no tiene tarifa y sólo se autoriza, en vez de
// convertirse en un 503: APICPUCredits devuelve error para un método que no conoce, y el router
// falla cerrado ante un error.
func chargedMethodFor(method, funcPath string) string {
	if creditControlRoutes[funcPath] {
		return ""
	}
	if method != "GET" && !isWriteMethod(method) {
		return ""
	}
	return method
}

// enforceAccessAndCredits es la única puerta de una request autenticada: resuelve el acceso y los
// créditos en un solo viaje a server_utils. Devuelve la respuesta de error, o nil para continuar.
//
// Una sola función y un solo frame porque son la misma decisión tomada con los mismos datos: el
// daemon cachea los accesos del user y su cuota en el mismo shard, bajo la misma clave
// (company, user) y detrás del mismo mutex, así que preguntar las dos cosas juntas cuesta un
// bloqueo en vez de dos —y en Lambda, cero lecturas a ScyllaDB en vez de una por entorno nuevo.
//
// Qué se cobra depende del método:
//   - POST y PUT: el cargo completo, porque el body ya se conoce. No hay distinción entre los dos,
//     ni en la tarifa ni en el nivel de acceso que exigen.
//   - GET: sólo la base; el excedente lo liquida chargeGetResponseTopUp cuando ya existe el tamaño
//     de la respuesta, que es la única cosa que no se sabe todavía aquí.
//   - creditControlRoutes: nada. Se manda un frame de sólo autorización, porque la exención salta
//     el COBRO y nunca el frame: tres de esas rutas están mapeadas en access_list.yml y dos son
//     sólo-SaaS, así que saltar el frame las dejaría abiertas a cualquier sesión.
func enforceAccessAndCredits(args *core.HandlerArgs, funcPath string) *core.HandlerResponse {
	// El catálogo decide qué accesos exigir. Toda la política vive de este lado, en el proceso que
	// embebe access_list.yml; el daemon sólo responde "este user tiene alguno de estos accesos".
	accessInfos, _ := accessHelper.GetAccesosByRoute(funcPath)
	decision := resolveRouteAccess(args.Method, funcPath, args.User.ID, accessInfos)
	if decision.denyMessage != "" {
		core.Log("Gate de accesos rechazó::", funcPath, " user::", args.User.ID,
			" motivo::", decision.denyMessage)
		response := args.MakeErrCode(decision.denyMessage, decision.denyCode)
		return &response
	}

	chargedMethod := chargedMethodFor(args.Method, funcPath)
	// Sin créditos que cobrar ni accesos que verificar no hay nada que preguntar, y ese es el camino
	// más transitado: un GET sin acceso mapeado no manda frame.
	if chargedMethod == "" && len(decision.requiredAccess) == 0 {
		return nil
	}

	// Para una escritura el body ya está, así que el cargo sale completo de una vez. Para un GET el
	// tamaño que se cobra es el de la respuesta y todavía no existe: aquí sólo se cobra la base.
	payloadBytes := 0
	if isWriteMethod(chargedMethod) && args.Body != nil {
		payloadBytes = len(*args.Body)
	}

	requestContext := context.Background()
	if args.ReqContext != nil {
		requestContext = args.ReqContext.Context()
	}

	var err error
	cpuCredits := uint16(0)
	if chargedMethod == "" {
		err = core.ChargeAPIAccessOnly(
			requestContext, args.User.CompanyID, args.User.ID, args.RouteID, decision.requiredAccess)
	} else {
		// APICPUCredits(GET, 0) es exactamente la base, así que el mismo cálculo sirve para los dos.
		cpuCredits, _ = core.APICPUCredits(chargedMethod, payloadBytes)
		err = core.ChargeAPIUsage(
			requestContext, args.User.CompanyID, args.User.ID, args.RouteID, chargedMethod,
			payloadBytes, decision.requiredAccess)
	}

	if err != nil {
		core.Log("server utils rechazó::", " method::", args.Method, " company::", args.User.CompanyID,
			" user::", args.User.ID, " route::", args.RouteID, " bytes::", payloadBytes,
			" accesos::", len(decision.requiredAccess), " err::", err)
		var response core.HandlerResponse
		if core.IsAccessDeniedError(err) {
			// Los nombres salen de aquí: el daemon no conoce access_list.yml.
			response = args.MakeAccessDeniedResponse(err, decision.accessNames)
		} else {
			response = args.MakeCreditRateLimitResponse(err)
		}
		return &response
	}

	core.Log("server utils aceptó::", " method::", args.Method, " company::", args.User.CompanyID,
		" user::", args.User.ID, " route::", args.RouteID, " bytes::", payloadBytes,
		" cpu_credits::", cpuCredits, " accesos::", len(decision.requiredAccess))
	return nil
}

// chargeGetResponseTopUp cobra lo que la respuesta excedió sobre la base ya cobrada. Devuelve nil
// cuando no excedió nada, que es el caso normal: una respuesta de hasta 8 KB cuesta exactamente la
// base y no manda un segundo frame.
func chargeGetResponseTopUp(args *core.HandlerArgs, responseBytes int) *core.HandlerResponse {
	totalCredits, err := core.APICPUCredits("GET", responseBytes)
	if err != nil {
		core.Log("no se pudo calcular el excedente del GET::", err)
		return nil
	}
	baseCredits, _ := core.APICPUBaseCredits("GET")
	if totalCredits <= baseCredits {
		return nil
	}

	requestContext := context.Background()
	if args.ReqContext != nil {
		requestContext = args.ReqContext.Context()
	}
	topUpCredits := totalCredits - baseCredits
	if err := core.ChargeAPICredits(
		requestContext, args.User.CompanyID, args.User.ID, args.RouteID, topUpCredits,
	); err != nil {
		core.Log("credit rate limiter rejected GET top-up::", " company::", args.User.CompanyID,
			" user::", args.User.ID, " route::", args.RouteID, " bytes::", responseBytes,
			" credits::", topUpCredits, " err::", err)
		response := args.MakeCreditRateLimitResponse(err)
		return &response
	}
	core.Log("credit rate limiter accepted GET top-up::", " company::", args.User.CompanyID,
		" user::", args.User.ID, " route::", args.RouteID, " bytes::", responseBytes,
		" credits::", topUpCredits)
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
