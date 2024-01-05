package handlers

import (
	"app/core"
	"os"
	"strings"
)

func HelloWorld(req *core.HandlerArgs) core.HandlerResponse {

	body1 := `
		Todo está funcionando bien!
		ENVIROMENT = $2
		DB_HOST = $3
		APP_CODE = $4
		EXEC_ARGS = $5
	`
	body1 = strings.Replace(body1, "$2", core.Env.ENVIROMENT, -1)
	body1 = strings.Replace(body1, "$3", core.Env.DB_HOST, -1)
	body1 = strings.Replace(body1, "$4", os.Getenv("APP_CODE"), -1)
	body1 = strings.Replace(body1, "$5", strings.Join(os.Args, " | "), -1)

	return req.MakeResponsePlain(&body1)
}
