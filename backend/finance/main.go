package finance

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.cash-banks":                      GetCashBanks,
	"POST.cash-banks":                     PostCashBanks,
	"GET.cash-banks-movements":       GetCashBankMovements,
	"GET.cash-bank-movement-by-id":   GetCashBankMovementByID,
	"GET.cash-banks-reconciliations": GetCashReconciliation,
	"POST.cash-banks-reconciliation": PostCashReconciliation,
	"POST.cash-banks-movement":       PostCashBankMovement,
	// Expenses module
	"GET.expenses":                 GetExpenses,
	"POST.expenses":                PostExpenses,
	"GET.expenses-scheduled":       GetExpensesScheduled,
	"POST.expenses-scheduled":      PostExpensesScheduled,
	"GET.expense-schedule-periods": GetExpenseSchedulePeriods,
	"POST.expense-payment":         PostExpensePayment,
}
