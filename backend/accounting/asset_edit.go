package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	logistics "app/logistics/types"
	"encoding/json"
	"fmt"
	"sort"
)

// AssetEditPayload corrects the acquisition data of an asset already on the register. Only the
// five fields a data-entry mistake can land in are here — the material, the warehouse, the
// supplier, the currency and the lot size are the asset's identity and are not editable (a
// warehouse move has its own path, PutAssetTransfer).
//
// AcquisitionValue and PurchaseAmount are the asset row's *totals*, not per-unit figures as in
// AssetAcquisitionPayload: that is what the columns hold, and dividing a stored total by Quantity
// would not round-trip for a lot of three.
type AssetEditPayload struct {
	AssetID          int32  `json:",omitempty"`
	SerialNumber     string `json:",omitempty"`
	AcquisitionDate  int16  `json:",omitempty"`
	DueDate          int16  `json:",omitempty"`
	AcquisitionValue int32  `json:",omitempty"`
	PurchaseAmount   int32  `json:",omitempty"`
}

// PutAssetEdit rewrites an asset's acquisition data, and everything that was derived from it.
//
// Two of the five fields are not cosmetic. AcquisitionValue and AcquisitionDate are the inputs
// DepreciationSchedule is a pure function of, so changing either makes every depreciation entry
// already posted describe a schedule that no longer exists — hence rewriteAssetDepreciation.
// SerialNumber is part of the ProductStockDetail key, so it cannot be updated in place at all;
// the units have to move.
func PutAssetEdit(req *core.HandlerArgs) core.HandlerResponse {
	payload := AssetEditPayload{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &payload); deserializeError != nil {
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}
	if payload.AssetID <= 0 {
		return req.MakeErr("Debe indicar el activo a editar.")
	}

	asset, loadError := loadAsset(req.User.CompanyID, payload.AssetID)
	if loadError != nil {
		return req.MakeErr(loadError)
	}
	if asset.Status == types.AssetStatusRemoved {
		return req.MakeErr("El activo fue eliminado y no puede editarse.")
	}
	if payload.AcquisitionValue <= 0 {
		return req.MakeErr("El valor en libros debe ser mayor a 0.")
	}
	if payload.PurchaseAmount < 0 {
		return req.MakeErr("El monto de compra no puede ser negativo.")
	}
	if payload.AcquisitionDate <= 0 {
		return req.MakeErr("Debe indicar la fecha de adquisición.")
	}
	// The payment ledger is authoritative and this handler never touches it, so lowering the
	// purchase amount below what was already paid would leave the asset over-paid with no record
	// of why. Reversing cash is not something an asset edit gets to do implicitly.
	if payload.PurchaseAmount < asset.PaidAmount {
		return req.MakeErr(fmt.Sprintf(
			"El monto de compra no puede ser menor a lo ya pagado (%.2f). Revierta el pago antes de reducirlo.",
			float64(asset.PaidAmount)/100,
		))
	}
	if asset.DisposalDate > 0 && payload.AcquisitionDate > asset.DisposalDate {
		return req.MakeErr("La fecha de adquisición no puede ser posterior a la fecha de baja.")
	}
	if payload.DueDate > 0 && payload.DueDate < payload.AcquisitionDate {
		return req.MakeErr("El vencimiento del pago no puede ser anterior a la fecha de adquisición.")
	}

	serialChanged := payload.SerialNumber != asset.SerialNumber
	if serialChanged {
		if asset.DisposalDate > 0 {
			return req.MakeErr("No se puede cambiar la serie de un activo dado de baja.")
		}
		if serialError := checkSerialIsFree(
			req.User.CompanyID, asset.ProductID, asset.ID, payload.SerialNumber,
		); serialError != nil {
			return req.MakeErr(serialError)
		}
		// The serial is part of the ProductStockDetail key, so the units move rather than the
		// row being relabelled: out of the old bucket, into the new one. An empty serial is the
		// no-detail bucket, so adding or clearing a serial is the same pair of movements.
		serialMovements := []logistics.InternalMovement{
			{
				ProductID:    asset.ProductID,
				WarehouseID:  asset.WarehouseID,
				SerialNumber: asset.SerialNumber,
				Quantity:     -asset.Quantity,
				DocumentID:   int64(asset.ID),
			},
			{
				ProductID:    asset.ProductID,
				WarehouseID:  asset.WarehouseID,
				SerialNumber: payload.SerialNumber,
				SupplierID:   asset.SupplierID,
				Quantity:     asset.Quantity,
				Price:        payload.AcquisitionValue / core.If(asset.Quantity > 0, asset.Quantity, 1),
				DocumentID:   int64(asset.ID),
			},
		}
		if movementError := logistics.ApplyMovimientos(req, serialMovements); movementError != nil {
			return req.MakeErr("Error al mover el activo a la nueva serie.", movementError)
		}
	}

	// Only these two invalidate the ledger. Compared before the asset is mutated.
	scheduleChanged := payload.AcquisitionValue != asset.AcquisitionValue ||
		payload.AcquisitionDate != asset.AcquisitionDate

	asset.SerialNumber = payload.SerialNumber
	asset.AcquisitionDate = payload.AcquisitionDate
	asset.DueDate = core.If(payload.DueDate > 0, payload.DueDate, payload.AcquisitionDate)
	asset.AcquisitionValue = payload.AcquisitionValue
	asset.PurchaseAmount = payload.PurchaseAmount
	asset.PaymentStatus = ResolveAssetPaymentStatus(payload.PurchaseAmount, asset.PaidAmount)

	if scheduleChanged {
		if rewriteError := rewriteAssetDepreciation(req, &asset); rewriteError != nil {
			return req.MakeErr(rewriteError)
		}
	}

	// Derived last, from the numbers the rewrite just settled: a value correction can move an
	// asset off FullyDepreciated and back onto Active.
	asset.Status = ResolveAssetStatus(&asset)
	asset.Updated = core.SUnixTime()
	asset.UpdatedBy = req.User.ID

	assetTable := db.TableOf[types.Asset]()
	assetRecords := []types.Asset{asset}
	if updateError := db.Update(&assetRecords,
		assetTable.SerialNumber, assetTable.AcquisitionDate, assetTable.DueDate,
		assetTable.AcquisitionValue, assetTable.PurchaseAmount, assetTable.PaymentStatus,
		assetTable.AccumulatedDepreciation, assetTable.LastDepreciationDate,
		assetTable.Status, assetTable.Updated, assetTable.UpdatedBy,
	); updateError != nil {
		return req.MakeErr("Error al actualizar el activo.", updateError)
	}

	core.Log("PutAssetEdit asset:", asset.ID, "schedule_rewritten:", scheduleChanged,
		"serial_moved:", serialChanged)
	return req.MakeResponse(assetRecords[0])
}

// checkSerialIsFree rejects a serial another live asset of the same material already carries.
// Two assets on one serial would both claim the same ProductStockDetail row.
func checkSerialIsFree(companyID, productID, assetID int32, serialNumber string) error {
	if len(serialNumber) == 0 {
		return nil
	}
	assets := []types.Asset{}
	assetQuery := db.Query(&assets)
	assetQuery.Select(assetQuery.ID, assetQuery.ProductID, assetQuery.SerialNumber, assetQuery.Status).
		CompanyID.Equals(companyID).
		SerialNumber.Equals(serialNumber)

	if queryError := assetQuery.Exec(); queryError != nil {
		return core.Err("Error al verificar el número de serie.", queryError)
	}
	for _, existing := range assets {
		if existing.ID == assetID || existing.ProductID != productID {
			continue
		}
		if existing.Status != types.AssetStatusRemoved {
			return core.Err("El número de serie ya está registrado en otro activo.")
		}
	}
	return nil
}

// rewriteAssetDepreciation reconciles the posted ledger against the schedule the asset's *new*
// acquisition value and date produce, and sets the asset's accumulated total and watermark to
// match. It mutates the asset but writes no asset row — the caller does that once.
//
// The rows are rewritten in place rather than retired and regenerated. The ORM has no Delete, so
// a wholesale rebuild would leave one dead row per period per edit in `expenses` forever; reusing
// the rows keeps their IDs stable and shows the expense register an amount change instead of a
// removal followed by an insert. Only the tail that no longer has a period is retired.
func rewriteAssetDepreciation(req *core.HandlerArgs, asset *types.Asset) error {
	postedEntries := []finance.Expense{}
	entryQuery := db.Query(&postedEntries)
	entryQuery.Select().
		CompanyID.Equals(req.User.CompanyID).
		AssetID.Equals(asset.ID)

	if queryError := entryQuery.Exec(); queryError != nil {
		return core.Err("Error al obtener los asientos de depreciación.", queryError)
	}

	// Type is not indexed here, and one asset's ledger is at most a few dozen rows. Rows this
	// function retired on an earlier edit are kept: a schedule that shrank and then grew again
	// reuses them rather than leaving them dead and inserting replacements. Status 0 on a Type-4
	// row is only ever written here, so there is no other retirement to resurrect by mistake.
	reusableEntries := make([]finance.Expense, 0, len(postedEntries))
	for _, entry := range postedEntries {
		if entry.Type == finance.ExpenseTypeDepreciation {
			reusableEntries = append(reusableEntries, entry)
		}
	}
	// Live rows first, each group by Date: period N of the new schedule then lands on the row
	// that already held period N, and only what is left over reaches the retired rows.
	sort.Slice(reusableEntries, func(a, b int) bool {
		leftIsLive, rightIsLive := reusableEntries[a].Status != 0, reusableEntries[b].Status != 0
		if leftIsLive != rightIsLive {
			return leftIsLive
		}
		return reusableEntries[a].Date < reusableEntries[b].Date
	})

	newSchedule := DepreciationSchedule(asset, core.FechaUnix())
	currentTimestamp := core.SUnixTime()

	entriesToUpdate := []finance.Expense{}
	entriesToInsert := []finance.Expense{}

	for periodIndex := range newSchedule {
		period := newSchedule[periodIndex]
		if periodIndex < len(reusableEntries) {
			entry := reusableEntries[periodIndex]
			entry.Date = period.PeriodDate
			entry.Amount = period.Amount
			// Posted, not payable — and the line that brings a previously retired row back.
			entry.Status = finance.ExpenseStatusPosted
			entry.Updated = currentTimestamp
			entry.UpdatedBy = req.User.ID
			entriesToUpdate = append(entriesToUpdate, entry)
			continue
		}
		// More periods than there are rows to hold them: the asset has never depreciated this
		// far, so the tail is written for the first time.
		entriesToInsert = append(entriesToInsert, finance.Expense{
			CompanyID:    req.User.CompanyID,
			Type:         finance.ExpenseTypeDepreciation,
			AssetID:      asset.ID,
			ProductID:    asset.ProductID,
			CategoryID:   depreciationCategoryID,
			CurrencyType: asset.CurrencyType,
			Date:         period.PeriodDate,
			Amount:       period.Amount,
			Status:       finance.ExpenseStatusPosted,
			Updated:      currentTimestamp,
			UpdatedBy:    req.User.ID,
			Created:      currentTimestamp,
			CreatedBy:    req.User.ID,
		})
	}

	// Rows the new schedule has no period for: the acquisition date moved forward, so months that
	// were posted never happened. Status 0 is the removed slot, and already-retired rows past
	// this point simply stay retired.
	for periodIndex := len(newSchedule); periodIndex < len(reusableEntries); periodIndex++ {
		entry := reusableEntries[periodIndex]
		if entry.Status == 0 {
			continue
		}
		entry.Status = 0
		entry.Updated = currentTimestamp
		entry.UpdatedBy = req.User.ID
		entriesToUpdate = append(entriesToUpdate, entry)
	}

	if len(entriesToInsert) > 0 {
		if insertError := db.Insert(&entriesToInsert); insertError != nil {
			return core.Err("Error al registrar los nuevos asientos de depreciación.", insertError)
		}
	}
	if len(entriesToUpdate) > 0 {
		expenseTable := db.TableOf[finance.Expense]()
		// Status is written on every row, changed or not: it shares the delta view's composite
		// key with UpdatedVersion, so the ORM requires the pair be updated together.
		if updateError := db.Update(&entriesToUpdate,
			expenseTable.Date, expenseTable.Amount, expenseTable.Status,
			expenseTable.Updated, expenseTable.UpdatedBy,
		); updateError != nil {
			return core.Err("Error al reescribir los asientos de depreciación.", updateError)
		}
	}

	if len(newSchedule) == 0 {
		// Nothing has elapsed under the new dates — the asset is back to depreciating from zero.
		asset.AccumulatedDepreciation = 0
		asset.LastDepreciationDate = 0
	} else {
		lastPeriod := newSchedule[len(newSchedule)-1]
		asset.AccumulatedDepreciation = lastPeriod.Accumulated
		asset.LastDepreciationDate = lastPeriod.PeriodDate
	}

	core.Log("rewriteAssetDepreciation asset:", asset.ID, "updated:", len(entriesToUpdate),
		"inserted:", len(entriesToInsert), "periods:", len(newSchedule))
	return nil
}
