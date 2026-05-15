package agent

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.agent-models": GetAgentModels,
}
