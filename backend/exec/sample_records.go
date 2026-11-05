package exec

import (
	"app/core"
	"app/tests/sample_records"
)

// GenerateErpHistory replays N past days of purchases, receptions and sales with the process
// clock frozen on each simulated day.
func GenerateErpHistory(args *core.ExecArgs) core.FuncResponse {
	return sample_records.GenerateErpHistory(args)
}

// GenerateProjectsFlow replays the life of agile projects through the projects handlers, checking
// every step against the module's invariants.
func GenerateProjectsFlow(args *core.ExecArgs) core.FuncResponse {
	return sample_records.GenerateProjectsFlow(args)
}

// GenerateSupplyData seeds the static provider list and a replenishment configuration for the
// first N products.
func GenerateSupplyData(args *core.ExecArgs) core.FuncResponse {
	return sample_records.GenerateSupplyData(args)
}

// GenerateExpenses replays N past days of a minimarket's operating expenses and their payments.
func GenerateExpenses(args *core.ExecArgs) core.FuncResponse {
	return sample_records.GenerateExpenses(args)
}

// SeedHR lives the demo company's HR day by day over the last N months: staff, advances, vacations,
// novedades and every run paid from the bank, then recomputes the bank's running balances.
func SeedHR(args *core.ExecArgs) core.FuncResponse {
	return sample_records.SeedHR(args)
}

// SeedHRLegalParameters loads the Peruvian legal payroll values and the AFP rates of 2026.
func SeedHRLegalParameters(args *core.ExecArgs) core.FuncResponse {
	return sample_records.SeedHRLegalParameters(args)
}

// GenerateDemoData seeds the whole ERP: catalog, supplies, assets and projects once, then purchases,
// expenses, invoiced sales, annulments, bank deposits and month-end closes day by day.
func GenerateDemoData(args *core.ExecArgs) core.FuncResponse {
	return sample_records.GenerateDemoData(args)
}
