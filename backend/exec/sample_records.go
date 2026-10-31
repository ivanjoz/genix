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
