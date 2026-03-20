package configuracion

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.p-hello":                HelloWorld,
	"GET.empresas":               GetEmpresas,
	"GET.empresa-parametros":     GetEmpresaParametros,
	"POST.empresa-parametros":    PostEmpresaParametros,
	"POST.empresa":               PostEmpresa,
	"POST.parametros":            PostParametros,
	"GET.parametros":             GetParametros,
	"GET.system-parameters":      GetSystemParameters,
	"POST.system-parameters":     PostSystemParameters,
	"GET.system-metrics-stream":  GetSystemMetricsStream,
	"GET.system-memory-packages": GetSystemMemoryPackages,
}
