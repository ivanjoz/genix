package exec

import (
	"app/business"
	"app/core"
	"fmt"
)

// SyncImageAssetsHandler exposes the repository import through the existing executable-function registry.
func SyncImageAssetsHandler(_ *core.ExecArgs) core.FuncResponse {
	result, err := business.SyncImageAssets()
	if err != nil {
		return core.FuncResponse{Error: err.Error()}
	}
	return core.FuncResponse{
		Message: fmt.Sprintf(
			"Image assets synced: %d images inserted, %d categories inserted, %d categories fetched, %d categories skipped",
			result.RecordsInserted,
			result.CategoriesInserted,
			result.CategoriesFetched,
			result.CategoriesSkipped,
		),
	}
}
