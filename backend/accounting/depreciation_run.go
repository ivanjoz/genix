package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	"cmp"
	"slices"
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

	// The plain read only finds who is due, so a load with nothing due takes no lock. The due
	// assets are then locked and everything is recomputed from the locked rows: two page loads at
	// once post each period once, and an edit or payment of an asset never interleaves with it.
	dueAssets := []types.Asset{}
	for _, asset := range assets {
		if len(PendingDepreciationPeriods(&asset, throughDate)) > 0 {
			dueAssets = append(dueAssets, types.Asset{CompanyID: req.User.CompanyID, ID: asset.ID})
		}
	}
	if len(dueAssets) == 0 {
		core.Log("PostAssetDepreciationRun: nothing due")
		return req.MakeResponse(map[string]any{"PostedEntries": 0, "UpdatedAssets": 0})
	}
	// Locked in key order, so two runs never hold each other's rows.
	slices.SortFunc(dueAssets, func(left, right types.Asset) int { return cmp.Compare(left.ID, right.ID) })

	var postedEntries, updatedAssets int
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		lockedAssets := slices.Clone(dueAssets)
		if lockError := db.LockByKeys(transaction, &lockedAssets); lockError != nil {
			return core.Err("Error al obtener los activos.", lockError)
		}
		var runError error
		postedEntries, updatedAssets, runError = postDepreciationIn(transaction, req, lockedAssets, throughDate)
		return runError
	})
	if err != nil {
		return req.MakeErr(err)
	}
	core.Log("PostAssetDepreciationRun posted:", postedEntries, "assets:", updatedAssets)
	return req.MakeResponse(map[string]any{"PostedEntries": postedEntries, "UpdatedAssets": updatedAssets})
}

// postDepreciationIn posts every period due through throughDate of the assets its caller locked,
// and advances their watermarks, in the caller's transaction.
func postDepreciationIn(transaction *db.Transaction, req *core.HandlerArgs, assets []types.Asset, throughDate int16,
) (int, int, error) {
	currentTimestamp := core.SUnixTime()
	depreciationEntries := []finance.Expense{}
	assetsToUpdate := []types.Asset{}

	for assetIndex := range assets {
		asset := &assets[assetIndex]
		// Re-checked on the locked row: an edit or a disposal may have landed since the plain read.
		if asset.Status != types.AssetStatusActive {
			continue
		}
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
		return 0, 0, nil
	}

	// The entries and the watermarks commit together, so a period is never posted without the
	// watermark moving past it, nor the reverse.
	for index := range depreciationEntries {
		depreciationEntries[index].StampPeriod()
	}
	if insertError := db.InsertIn(transaction, &depreciationEntries); insertError != nil {
		return 0, 0, core.Err("Error al registrar los asientos de depreciación.", insertError)
	}
	assetTable := db.TableOf[types.Asset]()
	if updateError := db.UpdateIn(transaction, &assetsToUpdate,
		assetTable.AccumulatedDepreciation, assetTable.LastDepreciationDate,
		assetTable.Status, assetTable.Updated, assetTable.UpdatedBy,
	); updateError != nil {
		return 0, 0, core.Err("Error al actualizar los activos depreciados.", updateError)
	}
	return len(depreciationEntries), len(assetsToUpdate), nil
}

// GetAssetDepreciation returns one asset's posted depreciation ledger, for the schedule view.
func GetAssetDepreciation(req *core.HandlerArgs) core.HandlerResponse {
	assetID := int32(req.GetQueryInt("assetID"))
	if assetID <= 0 {
		return req.MakeErr("Debe indicar el activo.")
	}

	// The ledger is the asset's cache group: the browser downloads it again only after a run or an
	// edit wrote one of its entries. An edit that shortens the schedule retires entries with
	// Status 0 (rewriteAssetDepreciation); the browser drops them, since a filter outside the
	// group's Keys would not be part of the group's cache identity.
	heldGroups, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}
	entryGroups := []db.RecordGroup[finance.Expense]{}
	entryQuery := db.QueryIndexGroup(&entryGroups).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range heldGroups {
		entryQuery.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	entryQuery.AssetID.Between(assetID, assetID)
	if queryError := entryQuery.Exec(); queryError != nil {
		core.Log("Error querying asset depreciation group:", queryError)
		return req.MakeErr("Error al obtener la depreciación del activo.")
	}
	return core.MakeResponse(req, &entryGroups)
}
