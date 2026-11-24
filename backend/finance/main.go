package finance

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.cash-banks":                 GetCashBanks,
	"POST.cash-banks":                PostCashBanks,
	"GET.cash-banks-movements":       GetCashBankMovements,
	"GET.cash-bank-movement-by-id":   GetCashBankMovementByID,
	"GET.cash-banks-reconciliations": GetCashReconciliation,
	"POST.cash-banks-reconciliation": PostCashReconciliation,
	"POST.cash-banks-movement":       PostCashBankMovement,
	// Expenses module
	"GET.expenses":                 GetExpenses,
	"GET.expense-ids":              GetExpensesByIDs,
	"POST.expenses":                PostExpenses,
	"GET.expenses-scheduled":       GetExpensesScheduled,
	"POST.expenses-scheduled":      PostExpensesScheduled,
	"GET.expense-schedule-periods": GetExpenseSchedulePeriods,
	"POST.expense-payment":         PostExpensePayment,
	// Cash flow forecast history frames
	"GET.cash-delay":     GetCashDelay,
	"GET.cash-documents": GetCashDocuments,
	// Cash flow drill-down: the open documents behind a cell
	"GET.cash-documents-day": GetCashDocumentsDay,
	// Cash flow forecast: Known + Scheduled layers, planned entries
	"GET.cash-flow-commitments": GetCashFlowCommitments,
	"GET.cash-flow-entries":     GetCashFlowEntries,
	"POST.cash-flow-entries":    PostCashFlowEntries,
	// Loans
	"GET.loans":         GetLoans,
	"POST.loans":        PostLoans,
	"POST.loan-payment": PostLoanPayment,
	// Exchange rates module
	"GET.exchange-rates":  GetExchangeRates,
	"POST.exchange-rates": PostExchangeRates,
}
