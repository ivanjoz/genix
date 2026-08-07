package agent

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.agent-models": GetAgentModels,
	// p- prefix: no acceso requirement (the chat is available to any signed in
	// user), but PostAgentTurn validates the session token itself. See turn.go.
	"POST.p-agent-turn": PostAgentTurn,
}
