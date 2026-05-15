package agent

import (
	"app/agent/llm"
	"app/core"
)

func GetAgentModels(req *core.HandlerArgs) core.HandlerResponse {
	models := llm.ListModels()
	core.Log("agent.models list requested user::", req.Usuario.ID, " count::", len(models))
	return req.MakeResponse(models)
}
