package business

import (
	"app/core"
)

// A single global tick rebuilds every dirty company's products .db snapshot at most every 30 min.
// "Dirty" companies are those registered in any cache_global group (productos/marcas/categorías);
// maybeRebuildProductsDbFile then decides per company whether the source actually advanced.
const (
	productsDbRebuildActionID        = int16(3)
	productsDbRebuildSystemCompanyID = int32(1)
	productsDbRebuildFrameMinutes    = int8(30)
)

func init() {
	core.RegisterActionHandler(productsDbRebuildActionID, "Reconstruir .db de productos", RebuildProductsDbHandler)
}

// ScheduleProductsDbRebuildCron seeds the recurring rebuild tick. Only the initial seed needs it:
// the row stores its own cadence, so the cron executor enqueues every following frame. Safe to
// call repeatedly (ScheduleCronAction dedupes the same logical action within a frame).
func ScheduleProductsDbRebuildCron() {
	core.ScheduleRecurringCronAction(core.CronAction{
		ActionID:  productsDbRebuildActionID,
		CompanyID: productsDbRebuildSystemCompanyID,
	}, productsDbRebuildFrameMinutes)
}

// RebuildProductsDbHandler rebuilds the snapshot for every dirty company. It does not reschedule
// itself: the executor re-enqueues the next 30-minute frame whatever the outcome here, so a panic
// halfway through no longer breaks the cadence.
func RebuildProductsDbHandler(args *core.ExecArgs) core.FuncResponse {
	dirtyCompanyIDs := collectDirtyCompanyIDs()
	core.Log("RebuildProductsDbHandler:: dirty companies", len(dirtyCompanyIDs))
	// AddMessage lands on the cron row when the executor writes the final status, so a per-company
	// failure is readable from the cron actions page instead of only from the process log.
	args.AddMessage(core.Concat(" ", "Empresas con cambios:", len(dirtyCompanyIDs)))

	for _, companyID := range dirtyCompanyIDs {
		if rebuildErr := maybeRebuildProductsDbFile(companyID, false); rebuildErr != nil {
			core.Log("RebuildProductsDbHandler:: rebuild error", "| companyID:", companyID, "| err:", rebuildErr)
			args.AddMessage(core.Concat(" ", "Error reconstruyendo empresa", companyID, ":", rebuildErr))
		}
	}

	return core.FuncResponse{}
}

// collectDirtyCompanyIDs unions the company IDs registered across all product-related cache groups.
func collectDirtyCompanyIDs() []int32 {
	seenCompanyIDs := map[int32]bool{}
	orderedCompanyIDs := []int32{}
	for _, groupID := range []int16{cacheGroupProducts, cacheGroupBrands, cacheGroupCategories} {
		rows, err := core.GetCacheGlobal(groupID)
		if err != nil {
			core.Log("collectDirtyCompanyIDs:: error leyendo grupo", "| group:", groupID, "| err:", err)
			continue
		}
		for _, row := range rows {
			if row.ID > 0 && !seenCompanyIDs[row.ID] {
				seenCompanyIDs[row.ID] = true
				orderedCompanyIDs = append(orderedCompanyIDs, row.ID)
			}
		}
	}
	return orderedCompanyIDs
}
