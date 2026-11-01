//go:build !js

package main

import (
	"app/core"
	"app/exec"
	"encoding/json"
	"fmt"
	"runtime/debug"
)

// exec is the server's admin package — backups, migrations, seeds — and it drives ScyllaDB
// directly, so it is wired here, out of the browser build, instead of in main-handlers.go.
func init() {
	appHandlersModules = append(appHandlersModules, exec.ModuleHandlers)
}

type ExecLambdaInput struct {
	ExecArgs core.ExecArgs `json:"fn_exec" cb:"fn_exec,1"`
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
