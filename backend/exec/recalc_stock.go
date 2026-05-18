package exec

import (
	"app/core"
	"app/logistics"
)

func RecalcStock(args *core.ExecArgs) core.FuncResponse {
	// Param1 is reserved for the company ID in compact cron/service calls.
	companyID := int32(args.Param1)
	if companyID <= 0 {
		companyID = 1
	}

	if err := logistics.RecalcProductStockByMovements(companyID); err != nil {
		return core.FuncResponse{Error: err.Error()}
	}

	return core.FuncResponse{Message: "Stock recalculado correctamente"}
}
