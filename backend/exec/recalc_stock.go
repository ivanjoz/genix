package exec

import (
	"app/core"
	"app/logistica"
)

func RecalcStock(args *core.ExecArgs) core.FuncResponse {
	// Param1 is reserved for the company ID in compact cron/service calls.
	empresaID := int32(args.Param1)
	if empresaID <= 0 {
		empresaID = 1
	}

	if err := logistica.RecalcProductStockByMovements(empresaID, false); err != nil {
		return core.FuncResponse{Error: err.Error()}
	}

	return core.FuncResponse{Message: "Stock recalculado correctamente"}
}
