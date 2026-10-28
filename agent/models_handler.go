package agent

import (
	"app/core"
	"genix-agent/llm"
)

func GetAgentModels(req *core.HandlerArgs) core.HandlerResponse {
	models := llm.ListModels()
	core.Log("agent.models list requested user::", req.User.ID, " count::", len(models))
	return req.MakeResponse(core.WithDeltaReplace(models))
}
