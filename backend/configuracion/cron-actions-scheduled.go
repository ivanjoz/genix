package configuracion

import (
	"app/core"
	"app/db"
)

type ActionRegistered struct {
	ID   int16
	Name string
}

func GetCronActionsScheduled(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt("actionsScheduled")

	if updated == 0 {
		updated = core.SUnixTime() - int32((7*24*60*60)/2) // Last 7 days
	}
	
	cronActions := []core.CronAction{}
	cronActionsQuery := db.Query(&cronActions).Updated.GreaterThan(updated)

	core.Log("GetCronActionsScheduled query:", "updated", updated)

	if err := cronActionsQuery.AllowFilter().Exec(); err != nil {
		core.Log("GetCronActionsScheduled query error:", err)
		return req.MakeErr("Error al obtener las acciones programadas.", err)
	}

	actionsIDs := core.SliceSet[int16]{}
	
	for _, e := range cronActions {
		actionsIDs.Add(e.ActionID)
	}
	
	registeredActionHandlers := core.GetRegisteredActionHandlers(actionsIDs.Values...)

	response := map[string]any{
		"actionsScheduled":  cronActions,
		"actionsRegistered": registeredActionHandlers,
	}

	core.Log("GetCronActionsScheduled response count:", len(cronActions),"|",len(registeredActionHandlers))
	return req.MakeResponse(response)
}
