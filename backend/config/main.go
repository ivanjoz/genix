package config

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.p-hello":                HelloWorld,
	"GET.cron-actions-scheduled": GetCronActionsScheduled,
	"GET.empresas":               GetEmpresas,
	"GET.company-parametros":     GetEmpresaParametros,
	"POST.company-parametros":    PostEmpresaParametros,
	"POST.company":               PostEmpresa,
	"POST.parametros":            PostParametros,
	"GET.parametros":             GetParametros,
	"GET.system-parameters":      GetSystemParameters,
	"POST.system-parameters":     PostSystemParameters,
	"GET.system-metrics-stream":  GetSystemMetricsStream,
	"GET.system-memory-packages": GetSystemMemoryPackages,
}
