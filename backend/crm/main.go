package crm

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.client-provider":     GetClientProviders,
	"GET.client-provider-ids": GetClientProvidersByIDs,
	"POST.client-provider":    PostClientProviders,
}
