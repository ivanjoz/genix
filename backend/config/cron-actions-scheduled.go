package config

import (
	"app/core"
	"app/db"
)

type ActionRegistered struct {
	ID   int16
	Name string
}

func GetCronActionsScheduled(req *core.HandlerArgs) core.HandlerResponse {
	// Last 7 days, whole every sync: there is no delta log for cron actions.
	updatedSince := core.SUnixTime() - int32((7*24*60*60)/2)
	cronActions := []core.CronAction{}
	cronActionsQuery := db.Query(&cronActions).Updated.GreaterThan(updatedSince)

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
		"actionsScheduled":  core.WithDeltaReplace(cronActions),
		"actionsRegistered": registeredActionHandlers,
	}

	core.Log("GetCronActionsScheduled response count:", len(cronActions), "|", len(registeredActionHandlers))
	return req.MakeResponse(response)
}
