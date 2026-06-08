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

// ScheduleProductsDbRebuildCron enqueues the next global rebuild tick. Safe to call repeatedly
// (ScheduleCronAction dedupes the same logical action within a frame) — call it once at startup.
func ScheduleProductsDbRebuildCron() {
	core.ScheduleCronAction(core.CronAction{
		ActionID:  productsDbRebuildActionID,
		CompanyID: productsDbRebuildSystemCompanyID,
	}, productsDbRebuildFrameMinutes)
}

// RebuildProductsDbHandler rebuilds the snapshot for every dirty company, then reschedules itself
// for the next 30-minute frame to keep the cadence going.
func RebuildProductsDbHandler(args *core.ExecArgs) core.FuncResponse {
	dirtyCompanyIDs := collectDirtyCompanyIDs()
	core.Log("RebuildProductsDbHandler:: dirty companies", len(dirtyCompanyIDs))

	for _, companyID := range dirtyCompanyIDs {
		if rebuildErr := maybeRebuildProductsDbFile(companyID); rebuildErr != nil {
			core.Log("RebuildProductsDbHandler:: rebuild error", "| companyID:", companyID, "| err:", rebuildErr)
		}
	}

	// Keep the recurring cadence alive by enqueuing the next frame.
	ScheduleProductsDbRebuildCron()
	return core.FuncResponse{}
}

// collectDirtyCompanyIDs unions the company IDs registered across all product-related cache groups.
func collectDirtyCompanyIDs() []int32 {
	seenCompanyIDs := map[int32]bool{}
	orderedCompanyIDs := []int32{}
	for _, groupID := range []int16{cacheGroupProductos, cacheGroupMarcas, cacheGroupCategorias} {
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
