package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
)

// PostAssetDepreciationRun materializes every depreciation period that has come due and has
// not been posted yet, for every asset still depreciating. The Activos page calls it on load.
//
// Generation is lazy rather than scheduled: there is no cron to keep alive, and the run is
// idempotent because each period is filtered against the asset's own LastDepreciationDate
// watermark. Calling it twice in one day writes nothing the second time.
func PostAssetDepreciationRun(req *core.HandlerArgs) core.HandlerResponse {
	throughDate := core.FechaUnix()

	// Only assets that are still depreciating: a disposed or fully depreciated one has
	// nothing left to post.
	assets := []types.Asset{}
	assetQuery := db.Query(&assets)
	assetQuery.Select().
		CompanyID.Equals(req.User.CompanyID).
		Status.Equals(types.AssetStatusActive)

	if queryError := assetQuery.Exec(); queryError != nil {
		return req.MakeErr("Error al obtener los activos.", queryError)
	}

	currentTimestamp := core.SUnixTime()
	depreciationEntries := []finance.Expense{}
	assetsToUpdate := []types.Asset{}

	for assetIndex := range assets {
		asset := &assets[assetIndex]
		pendingPeriods := PendingDepreciationPeriods(asset, throughDate)
		if len(pendingPeriods) == 0 {
			continue
		}

		for _, period := range pendingPeriods {
			depreciationEntries = append(depreciationEntries, finance.Expense{
				CompanyID: req.User.CompanyID,
				// No Name. "Depreciación 7/60" is presentation, not data: the frontend
				// composes it from Date and the asset's DepreciationMonths, which is
				// what makes it translatable — a stored string would be Spanish forever.
				Type:         finance.ExpenseTypeDepreciation,
				AssetID:      asset.ID,
				ProductID:    asset.ProductID,
				CategoryID:   depreciationCategoryID,
				CurrencyType: asset.CurrencyType,
				// No PeriodDate either. That column is the scheduled-expense dedupe key;
				// depreciation dedupes on Asset.LastDepreciationDate instead, so on a Type-4
				// row it would only ever be a second copy of Date that nothing reads.
				Date:   period.PeriodDate,
				Amount: period.Amount,
				// Posted, not payable: a depreciation entry moves no cash, so it never
				// belongs in the Pend. Pago or Pagados tabs.
				Status:    finance.ExpenseStatusPosted,
				Updated:   currentTimestamp,
				UpdatedBy: req.User.ID,
				Created:   currentTimestamp,
				CreatedBy: req.User.ID,
			})
		}

		lastPeriod := pendingPeriods[len(pendingPeriods)-1]
		asset.AccumulatedDepreciation = lastPeriod.Accumulated
		asset.LastDepreciationDate = lastPeriod.PeriodDate
		asset.Status = ResolveAssetStatus(asset)
		asset.Updated = currentTimestamp
		asset.UpdatedBy = req.User.ID
		assetsToUpdate = append(assetsToUpdate, *asset)
	}

	if len(depreciationEntries) == 0 {
		core.Log("PostAssetDepreciationRun: nothing due")
		return req.MakeResponse(map[string]any{"PostedEntries": 0, "UpdatedAssets": 0})
	}

	// The ledger entries go first: if the asset update fails afterwards the watermark stays
	// behind and the next run re-posts them, which is visible. The reverse order would
	// advance the watermark past periods that were never written, losing them silently.
	if insertError := db.Insert(&depreciationEntries); insertError != nil {
		return req.MakeErr("Error al registrar los asientos de depreciación.", insertError)
	}

	assetTable := db.TableOf[types.Asset]()
	if updateError := db.Update(&assetsToUpdate,
		assetTable.AccumulatedDepreciation, assetTable.LastDepreciationDate,
		assetTable.Status, assetTable.Updated, assetTable.UpdatedBy,
	); updateError != nil {
		return req.MakeErr("Error al actualizar los activos depreciados.", updateError)
	}

	core.Log("PostAssetDepreciationRun posted:", len(depreciationEntries), "assets:", len(assetsToUpdate))
	return req.MakeResponse(map[string]any{
		"PostedEntries": len(depreciationEntries),
		"UpdatedAssets": len(assetsToUpdate),
	})
}

// GetAssetDepreciation returns one asset's posted depreciation ledger, for the schedule view.
func GetAssetDepreciation(req *core.HandlerArgs) core.HandlerResponse {
	assetID := int32(req.GetQueryInt("assetID"))
	if assetID <= 0 {
		return req.MakeErr("Debe indicar el activo.")
	}

	entries := []finance.Expense{}
	entryQuery := db.Query(&entries)
	entryQuery.Select().
		CompanyID.Equals(req.User.CompanyID).
		AssetID.Equals(assetID)

	if queryError := entryQuery.Exec(); queryError != nil {
		return req.MakeErr("Error al obtener la depreciación del activo.", queryError)
	}

	return req.MakeResponse(entries)
}
