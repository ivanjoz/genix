package main

import (
	"app/aws"
	"app/core"
	"app/exec"
	"app/handlers"
	"app/operaciones"
	"encoding/json"
	"fmt"
	"slices"

	"reflect"
	"runtime"
	"strings"
	"time"
)

// Agrupa los handlers de los módulos en uns solo map
var appHandlersModules = []core.AppRouterType{
	exec.ModuleHandlers,
	handlers.ModuleHandlers,
	operaciones.ModuleHandlers,
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

func mainHandler(args core.HandlerArgs) core.MainResponse {
	// coloca algunas variables de entorno que pueden ser utilizadas por otros handlers
	args.Authorization = core.MapGetKeys(args.Headers, "Authorization", "authorization")
	args.Encoding = core.MapGetKeys(args.Headers, "Accept-Encoding", "accept-encoding")

	// NOTE: The Lambda runtime processes one invocation at a time per execution environment,
	// but local/VPS HTTP mode is concurrent. Avoid mutating global per-request state when local.
	if !core.Env.IS_LOCAL {
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

	// Los es públicos comienzan con "p-" y no necesitan validacion del user Tocken
	isPublicPath := len(args.Route) > 2 && args.Route[0:2] == "p-"
	if !isPublicPath {
		args.Usuario = core.CheckUser(&args, 0)
	}

	funcPath := args.Method + "." + args.Route

	// Request header log is only meaningful in Lambda mode (it uses REQ_ID / REQ_LAMBDA_ID).
	if !core.Env.IS_LOCAL {
		reqPathsParsed := []string{}
		for _, name := range core.REQ_PATHS {
			hash := core.FnvHashString64(name, 64, 5)
			reqPathsParsed = append(reqPathsParsed, name+"="+hash)
		}

		logHeader := core.Concat("|", "$Req", core.Env.REQ_ID, core.Env.REQ_LAMBDA_ID,
			args.Usuario.ID, args.Usuario.Usuario, strings.Join(reqPathsParsed, "&"))
		fmt.Println(logHeader)
	}

	handlerResponse := core.HandlerResponse{Encoding: args.Encoding}

	// Si no es público, valida el usuario
	if !isPublicPath && len(args.Usuario.Error) > 0 {
		core.Log("Usuario Error::", args.Usuario.Error)
		handlerResponse.Error = args.Usuario.Error
		return prepareResponse(args, &handlerResponse)
	}

	handlerFunc, ok := appHandlers[funcPath]
	if !ok {
		core.Log("no hay una lambda para el path solicitado::", funcPath)
		handlerResponse.Error = "no hay una lambda para el path solicitado: " + funcPath
	} else {
		core.Log("Ejecutando Handler::", funcPath)
		handlerResponse = handlerFunc(&args)
		respLen := 0
		if handlerResponse.Body != nil {
			respLen = len(*handlerResponse.Body)
		}
		core.Log("Finalizado Handler::", funcPath, " | Len: ", respLen)
	}

	return prepareResponse(args, &handlerResponse)
}

func clearEnvVariables() {
	core.Usuario = core.IUsuario{}
	core.LogsSaved = []string{}
	core.REQ_PATHS = []string{}
	core.Env.USUARIO_ID = 0
	core.Env.LOGS_ONLY_SAVE = false
	core.Env.LOGS_FULL = false
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

type ExecLambdaInput struct {
	ExecArgs core.ExecArgs `json:"fn_exec"`
}

func ExecFuncHandler(lambdaInput string) core.FuncResponse {
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

	type FuncToInvoke struct {
		HourMin  string
		Name     string
		Priority int32
		Exec     func(args *core.ExecArgs) core.FuncResponse
	}

	// Revisa si hay una función asignada a esta hora
	if args.FuncToExec == "cron" {
		if len(args.Param6) == 0 {
			args.Param6 = core.GetHoursMinutes()
		}

		core.Log("*Search Time Function:: ", args.Param6)
		funcsToInvokeMap := map[string]FuncToInvoke{}

		for hourMin := range exec.ExecHandlersCron {
			addFuncToInvoke := func() {
				funcName := GetFunctionName(exec.ExecHandlers[hourMin])
				funcName = core.ToSnakeCase(strings.ReplaceAll(funcName, "app/", ""))
				funcsToInvokeMap[funcName] = FuncToInvoke{
					Name:    funcName,
					HourMin: hourMin,
					Exec:    (exec.ExecHandlersCron)[hourMin],
				}
			}
			if len(hourMin) < 5 {
				continue
			}
			if args.Param6 == hourMin[0:5] {
				addFuncToInvoke()
			} else if strings.Contains(hourMin, "|") {
				for _, h := range strings.Split(hourMin, "|") {
					if h[:2] != args.Param6[:2] || !strings.Contains(h, ":") {
						continue
					}
					minutes := strings.Split(h, ":")[1]
					isIncluded := strings.Contains(minutes, ",") && core.Contains(strings.Split(minutes, ","), args.Param6[3:])

					if minutes == "*" || args.Param6 == h || isIncluded {
						addFuncToInvoke()
						break
					}
				}
			} else if strings.Contains(hourMin, "-") {
				values := strings.Split(hourMin, "-")
				hourStart := values[0]
				hourEnd := values[1]
				if args.Param6 >= hourStart && args.Param6 <= hourEnd {
					addFuncToInvoke()
				}
			}
		}

		if len(funcsToInvokeMap) == 0 {
			return core.FuncResponse{}
		}

		messages := []string{}

		for funcName, funcToInvoke := range funcsToInvokeMap {
			nowTime := time.Now().Unix()
			core.Log("*Ejecutando función:: ", funcToInvoke.HourMin, " | ", funcName)
			core.LogsSaved = []string{}

			args := core.ExecArgs{Message: ""}
			funcMessage := funcToInvoke.Exec(&args).Message
			duration := int(time.Now().Unix() - nowTime)
			aws.PutFuncLog(funcName, funcMessage, duration)

			message := core.Concat(" | ", "Func: "+funcName, core.Concats(duration, "s"))
			messages = append(messages, message)
		}
		core.Log(messages)
		// Función a ejecutarse con nombre específico
	} else if len(args.FuncToExec) > 0 {
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

func prepareResponse(args core.HandlerArgs, handlerResponse *core.HandlerResponse) core.MainResponse {
	response := core.MainResponse{}
	if core.Env.IS_LOCAL {
		// core.Print(handlerResponse)
		core.SendLocalResponse(args, *handlerResponse)
		// In local/VPS HTTP mode we skip request-log persistence to avoid global
		// per-request state and Dynamo I/O on the hot path.
		return response
	} else {
		if len(handlerResponse.Error) > 0 {
			error := handlerResponse.Error
			response.LambdaResponse = core.MakeErrRespFinal(400, error)
		} else {
			response.LambdaResponse = core.MakeResponseFinal(handlerResponse)
		}
	}

	// Revisa si es necesario guardar el Log
	logRecord := core.MakeReqLog()
	apisToAvoid := []string{"logs-"}

	saveLogs := true
	if strings.Contains(logRecord.IP, "::1") || strings.Contains(logRecord.IP, "127.0.0.1") {
		saveLogs = false
	}
	if len(logRecord.PathName) < 4 || strings.Contains(logRecord.PathName, ".p-") {
		saveLogs = false
	}

	for _, e := range apisToAvoid {
		if strings.Contains(logRecord.ApiUrl, e) {
			saveLogs = false
			break
		}
	}
	if saveLogs {
		// core.Print(logRecord)
		aws.MakeTableLogs2().PutItem(&logRecord, 1)
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
