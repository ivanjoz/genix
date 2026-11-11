package config

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.p-hello":                     HelloWorld,
	"GET.p-company-names-by-ids":      GetPublicCompanyNamesByIDs,
	"GET.cron-actions-scheduled":      GetCronActionsScheduled,
	"GET.empresas":                    GetEmpresas,
	"GET.company-parametros":          GetEmpresaParametros,
	"POST.company-parametros":         PostEmpresaParametros,
	"POST.company":                    PostEmpresa,
	"GET.system-parameters":           GetSystemParameters,
	"POST.system-parameters":          PostSystemParameters,
	"GET.system-metrics-stream":       GetSystemMetricsStream,
	"GET.server-metrics":              GetServerMetrics,
	"GET.observability":               GetObservability,
	"GET.credit-consumption":          GetCreditConsumption,
	"GET.company-credit-usage-report": GetCompanyCreditUsageReport,
	"GET.company-credit-usage":        GetCompanyCreditUsage,
	"GET.company-users-by-ids":        GetCompanyUsersByIDs,
	"GET.company-credit-budget":       GetCompanyCreditBudget,
	"POST.company-credit-budget":      PostCompanyCreditBudget,
	"GET.request-errors-by-ids":       GetRequestErrorsByIDs,
	"GET.system-memory-packages":      GetSystemMemoryPackages,
	"GET.credit-usage":                GetCreditUsage,
	"GET.sent-emails":                 GetSentEmails,
	"GET.sent-email":                  GetSentEmail,
	"GET.records-by-ids":              GetTableRecordsByIDs,
}
