package hr

import "app/core"

// ModuleHandlers are the routes of the HR (RRHH) module. Route names carry the "hr-" prefix, and
// every route is mapped in backend/access.toml: people routes under "Personal", every route that
// carries money under "Planilla" (see plan/HR_PAYROLL_PLAN.md §2.2). The legal parameters are
// written only by the SaaS company (saasOnlyRoutes in main-handlers.go).
var ModuleHandlers = core.AppRouterType{
	"GET.hr-org":                    GetOrganization,
	"POST.hr-departments":           PostDepartment,
	"POST.hr-positions":             PostPosition,
	"GET.hr-employees":              GetEmployees,
	"GET.hr-employees-ids":          GetEmployeesByIDs,
	"POST.hr-employees":             PostEmployee,
	"GET.hr-dependents":             GetDependents,
	"POST.hr-dependents":            PostDependent,
	"GET.hr-contracts":              GetContracts,
	"POST.hr-contracts":             PostContract,
	"GET.hr-legal-parameters":       GetLegalParameters,
	"POST.hr-legal-parameters":      PostLegalParameter,
	"POST.hr-afp-rates":             PostAfpRates,
	"GET.hr-payroll-config":         GetPayrollConfig,
	"POST.hr-concepts":              PostConcept,
	"GET.hr-employee-concepts":      GetEmployeeConcepts,
	"POST.hr-employee-concepts":     PostEmployeeConcept,
	"GET.hr-employee-records":       GetEmployeeRecords,
	"POST.hr-employee-records":      PostEmployeeRecord,
	"POST.hr-loans":                 PostLoan,
	"POST.hr-pay-advances":          PostPayAdvances,
	"POST.hr-payroll-runs":          PostPayrollRun,
	"POST.hr-payroll-calculate":     PostPayrollCalculate,
	"POST.hr-payroll-ensure":        PostPayrollEnsure,
	"POST.hr-payroll-close":         PostPayrollClose,
	"POST.hr-payroll-reopen":        PostPayrollReopen,
	"POST.hr-payroll-paid":          PostPayrollPaid,
	"POST.hr-payroll-contributions": PostPayrollContributions,
	"GET.hr-payslips":               GetPayslips,
	"GET.hr-payslips-ids":           GetPayslipsByIDs,
	"GET.hr-payslips-employee":      GetEmployeePayslips,
}
