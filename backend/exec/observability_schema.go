package exec

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
)

const observabilityUserLogsTable = "user_logs"

// RebuildObservabilityLogView replaces only user_logs derived structures so projection changes
// become live without rebuilding views and indexes owned by unrelated tables.
func RebuildObservabilityLogView(args *core.ExecArgs) core.FuncResponse {
	if core.Env.DB_NAME == "" {
		return args.MakeErr("No se ha especificado db.name en config.toml.")
	}

	db.Connect(MakeConnParams())
	controller := db.MakeController[coreTypes.UserLog]()
	core.Log("observability view rebuild started::", " table::", observabilityUserLogsTable)
	if err := controller.DeleteViewsAndIndexes(); err != nil {
		core.Log("observability view rebuild drop failed::", " table::", observabilityUserLogsTable, " err::", err)
		return args.MakeErr("No se pudo eliminar la view anterior de observabilidad.", err)
	}

	// Deploying one controller recreates only the declared user_logs view and keeps the base rows.
	if err := db.Deploy(controller); err != nil {
		return args.MakeErr(err)
	}
	message := "View de user_logs reconstruida para observabilidad."
	core.Log("observability view rebuild completed::", " table::", observabilityUserLogsTable)
	return core.FuncResponse{Message: message}
}
