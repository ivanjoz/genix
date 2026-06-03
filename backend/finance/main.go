package finance

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.cajas":                      GetCajas,
	"POST.cajas":                     PostCajas,
	"GET.cash-banks-movements":       GetCajaMovimientos,
	"GET.cash-bank-movement-by-id":   GetCashBankMovementByID,
	"GET.cash-banks-reconciliations": GetCajaCuadres,
	"POST.cash-banks-reconciliation": PostCajaCuadre,
	"POST.cash-banks-movement":       PostMovimientoCaja,
	// Expenses module
	"GET.expenses":                 GetExpenses,
	"POST.expenses":                PostExpenses,
	"GET.expenses-scheduled":       GetExpensesScheduled,
	"POST.expenses-scheduled":      PostExpensesScheduled,
	"GET.expense-schedule-periods": GetExpenseSchedulePeriods,
	"POST.expense-payment":         PostExpensePayment,
}
