package exec

import (
	"app/core"
	"app/tests/sample_records"
)

// GenerateSampleSaleOrders delegates the heavy sample data generation to the dedicated tests/sample_records package.
func GenerateSampleSaleOrders(args *core.ExecArgs) core.FuncResponse {
	return sample_records.GenerateSaleOrders(args)
}
