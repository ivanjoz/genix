package main

import (
	"app/aws"
	"app/core"
	"app/exec"
	"app/handlers"
	"fmt"

	"reflect"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// Agrupa los handlers de los módulos en uns solo map
var appHandlersModules = []core.AppRouterType{
	handlers.ModuleHandlers,
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
func mainHandler(args core.HandlerArgs) core.MainResponse {
	// coloca algunas variables de entorno que pueden ser utilizadas por otros handlers
	args.Authorization = core.MapGetKeys(args.Headers, "Authorization", "authorization")
	req_encoding := core.MapGetKeys(args.Headers, "Accept-Encoding", "accept-encoding")
	core.StartTime = (time.Now()).UnixMilli()

	core.Env.REQ_ENCODING = req_encoding
	core.Env.REQ_PARAMS = core.MapGetKeys(args.Headers, "x-api-key", "X-Api-Key")
	core.Env.REQ_USER_AGENT = core.MapGetKeys(args.Headers, "User-Agent", "user-agent")
	core.Env.REQ_ID = core.IntToBase64(time.Now().UnixMilli(), 6)
	args.Usuario = core.CheckUser(&args, 0)

	apiNames := []string{"api", "go1", "go2", "go3", "go4", "go5"}
	pathSegments := []string{}
	for _, value := range strings.Split(args.Route, "/") {
		isContained := false
		for _, e := range apiNames {
			if value == e {
				isContained = true
				break
			}
		}
		if len(value) > 1 && !isContained {
			pathSegments = append(pathSegments, value)
		}
	}

	path := strings.Join(pathSegments, "/")
	funcPath := args.Method + "." + path
	core.Env.HANDLER_PARH = funcPath

	mergedRoutes := []core.MergedRoute{}
	if path == "merged" {
		mergedRoutes = core.ParseMergedUri(args.Query)
		for _, e := range mergedRoutes {
			core.REQ_PATHS = append(core.REQ_PATHS, e.FuncPath)
		}
	} else {
		core.REQ_PATHS = []string{funcPath}
	}

	reqPathsParsed := []string{}
	for _, name := range core.REQ_PATHS {
		hash := core.FnvHashString64(name, 64, 5)
		reqPathsParsed = append(reqPathsParsed, name+"="+hash)
	}

	logHeader := core.Concat("|", "$Req", core.Env.REQ_ID, core.Env.REQ_LAMBDA_ID,
		args.Usuario.ID, args.Usuario.Usuario, strings.Join(reqPathsParsed, "&"))

	fmt.Println(logHeader)

	handlerResponse := core.HandlerResponse{}

	// Los es públicos comienzan con "p-" y no necesitan validacion del user Tocken
	isPublicPath := len(path) > 2 && path[0:2] == "p-"
	// Si no es público, valida el usuario
	if !isPublicPath && len(args.Usuario.Error) > 0 {
		core.Log("Usuario Error::", args.Usuario.Error)
		handlerResponse.Error = args.Usuario.Error
		return prepareResponse(args, &handlerResponse)
	}

	appHandlers := *makeAppHandlers()

	// Revisa si es una query con varios handlers dentro de una
	if len(handlerResponse.Error) == 0 && len(mergedRoutes) > 0 {
		core.Log("Es un handler merged:", args.Query)
		mergedRoutes := core.ParseMergedUri(args.Query)
		if mergedRoutes == nil {
			core.Log("No se pudo interpretar la ruta enviada")
			handlerResponse.Error = "No se pudo interpretar la ruta enviada"
			return prepareResponse(args, &handlerResponse)
		}

		for _, route := range mergedRoutes {
			core.Log("ruta::", route)
			_, ok := appHandlers[route.FuncPath]
			if !ok {
				core.Log("no hay una lambda para el path solicitado 2::", route.FuncPath)
				handlerResponse.Error = "no hay una lambda para el path solicitado: " + route.FuncPath
				return prepareResponse(args, &handlerResponse)
			}
		}

		handlerResponses := make([]*core.HandlerResponse, len(mergedRoutes))
		group := errgroup.Group{}
		core.Log("Ejecutando las funciones de las rutas enviadas")

		for i := range mergedRoutes {
			route := mergedRoutes[i]
			core.Env.REQ_PATH = route.FuncPath

			handlerFunc := appHandlers[route.FuncPath]
			handerArgs := core.HandlerArgs{
				Headers:  args.Headers,
				Method:   args.Method,
				Query:    route.Query,
				Route:    route.Route,
				MergedID: route.Id,
				Usuario:  args.Usuario,
			}
			idx := i
			group.Go(func() error {
				response := handlerFunc(&handerArgs)
				handlerResponses[idx] = &response
				return nil
			})
			time.Sleep(4 * time.Millisecond)
		}

		group.Wait()
		handlerResponse = core.CombineResponses(handlerResponses)

		// revisa si hay una función que satisfaga la lambda requerida
	} else if len(handlerResponse.Error) == 0 {
		core.Env.REQ_PATH = funcPath
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

func ExecFuncHandler(funcToExec string) {
	core.Env.LOGS_ONLY_SAVE = true
	hourMinCurrent := ""

	if len(funcToExec) == 5 && strings.Contains(funcToExec, ":") {
		hourMinCurrent = funcToExec
		funcToExec = "cron"
	}

	core.Log("func to exec:: ", funcToExec)
	// Revisa si hay una función asignada a esta hora
	if funcToExec == "cron" {
		if len(hourMinCurrent) == 0 {
			hourMinCurrent = core.GetHoursMinutes()
		}

		core.Log("*Search Time Function:: ", hourMinCurrent)
		funcsToInvoke := map[string]func(args *core.ExecArgs) core.FuncResponse{}

		funcHandlers := []*exec.ExecRouterType{&exec.ExecHandlers}

		if core.Env.IS_PROD {
			funcHandlers = append(funcHandlers, &exec.ExecHandlersProd)
		} else {
			funcHandlers = append(funcHandlers, &exec.ExecHandlersQAs)
		}

		for _, funcHandler := range funcHandlers {
			for hourMin := range *funcHandler {
				setFuncToInvoke := func() {
					funcName := GetFunctionName(exec.ExecHandlers[hourMin])
					funcName = core.ToSnakeCase(strings.ReplaceAll(funcName, "app/", ""))
					funcsToInvoke[funcName] = exec.ExecHandlers[hourMin]
				}
				if len(hourMin) < 5 {
					continue
				}
				if hourMinCurrent == hourMin[0:5] {
					setFuncToInvoke()
				} else if strings.Contains(hourMin, "|") {
					for _, h := range strings.Split(hourMin, "|") {
						// core.Log("revisando horas: ", h)
						if hourMinCurrent == h {
							setFuncToInvoke()
							continue
						}
					}
				} else if strings.Contains(hourMin, "-") {
					values := strings.Split(hourMin, "-")
					hourStart := values[0]
					hourEnd := values[1]
					if hourMinCurrent >= hourStart && hourMinCurrent <= hourEnd {
						setFuncToInvoke()
					}
				}
			}
		}

		if len(funcsToInvoke) == 0 {
			return
		}

		messages := []string{}
		core.Log("*Funciones encontradas:: ", funcsToInvoke)

		for funcName, funcToInvoke := range funcsToInvoke {
			nowTime := time.Now().Unix()
			core.Log("*Ejecutando función:: ", hourMinCurrent, " | ", funcName)
			core.LogsSaved = []string{}

			args := core.ExecArgs{Message: ""}
			funcMessage := funcToInvoke(&args).Message
			duration := int(time.Now().Unix() - nowTime)

			// Guarda el log de la función
			aws.PutFuncLog(funcName, funcMessage, duration)

			message := core.Concat(" | ", "Func: "+funcName, core.Concats(duration, "s"))
			messages = append(messages, message)
		}
		core.Log(messages)
	}
}

func prepareResponse(args core.HandlerArgs, handlerResponse *core.HandlerResponse) core.MainResponse {
	response := core.MainResponse{}
	if core.Env.IS_LOCAL {
		core.SendLocalResponse(args, *handlerResponse)
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
