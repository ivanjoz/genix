package exec

import (
	"app/core"
	"app/logistica"
	"strconv"
)

func RecalcStock(args *core.ExecArgs) core.FuncResponse {
	// Let's assume the first parameter in Params is the empresa_id, or we parse it from param2 if needed
	var empresaID int32 = 1 // Default or we can extract from args
	if val, ok := args.Params["empresa_id"]; ok {
		parsed, _ := strconv.Atoi(val)
		empresaID = int32(parsed)
	}

	if err := logistica.RecalcProductStockByMovements(empresaID); err != nil {
		return core.FuncResponse{Error: err.Error()}
	}

	return core.FuncResponse{Message: "Stock recalculado correctamente"}
}
