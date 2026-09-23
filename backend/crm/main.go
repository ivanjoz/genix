package crm

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.client-provider":              GetClientProviders,
	"GET.client-provider-ids":          GetClientProvidersByIDs,
	"GET.client-provider-snapshot-ids": GetClientProviderSnapshotsByIDs,
	"POST.client-provider":             PostClientProviders,
}
