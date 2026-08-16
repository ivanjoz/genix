package exec

import (
	"app/core"
	coreTypes "app/core/types"

	"github.com/ivanjoz/genix-orm/scylla"
)

const observabilityUserLogsTable = "user_logs"

// RebuildObservabilityLogView replaces only user_logs derived structures so projection changes
// become live without rebuilding views and indexes owned by unrelated tables.
func RebuildObservabilityLogView(args *core.ExecArgs) core.FuncResponse {
	if core.Env.DB_NAME == "" {
		return args.MakeErr("No se ha especificado db.name en config.toml.")
	}

	scylla.MakeScyllaConnection(makeConnParams())
	controller := makeDBController[coreTypes.UserLog]()
	core.Log("observability view rebuild started::", " table::", observabilityUserLogsTable)
	if err := controller.DeleteViewsAndIndexes(); err != nil {
		core.Log("observability view rebuild drop failed::", " table::", observabilityUserLogsTable, " err::", err)
		return args.MakeErr("No se pudo eliminar la view anterior de observabilidad.", err)
	}

	// Deploying one controller recreates only the declared user_logs view and keeps the base rows.
	scylla.DeployScylla(0, controller)
	message := "View de user_logs reconstruida para observabilidad."
	core.Log("observability view rebuild completed::", " table::", observabilityUserLogsTable)
	return core.FuncResponse{Message: message}
}
