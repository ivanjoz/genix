package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	"encoding/json"
)

// AssetPaymentPayload settles part or all of an asset's acquisition.
type AssetPaymentPayload struct {
	AssetID     int32 `cb:"1"`
	CashBankID  int32 `cb:"2"`
	Amount      int32 `cb:"3"` // Positive payment amount, in cents.
	Date        int16 `cb:"4"`
	IsFullyPaid bool  `cb:"5"` // Forces settled status (write-off case).
}

// PostAssetPayment records a payment against an asset. The asset is not an expense, so this
// does not go through the expense payment path: it writes the cash movement directly and
// recomputes the asset's own PaidAmount from the ledger.
func PostAssetPayment(req *core.HandlerArgs) core.HandlerResponse {
	payload := AssetPaymentPayload{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &payload); deserializeError != nil {
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}
	if payload.Amount <= 0 {
		return req.MakeErr("El monto del pago debe ser mayor a 0.")
	}
	if payload.AssetID <= 0 || payload.CashBankID <= 0 {
		return req.MakeErr("Faltan parámetros: (AssetID o CashBankID).")
	}

	// The asset and then the cash bank are locked, so the pending balance and the till's balance
	// checked here are the ones the payment is written against: a concurrent payment, edit or
	// depreciation of the asset waits for this one and sees it.
	var asset types.Asset
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var loadError error
		if asset, loadError = lockAssetIn(transaction, req.User.CompanyID, payload.AssetID); loadError != nil {
			return loadError
		}
		cashBanks := []finance.CashBank{{CompanyID: req.User.CompanyID, ID: payload.CashBankID}}
		if lockError := db.LockByKeys(transaction, &cashBanks); lockError != nil {
			return core.Err("Error al obtener la caja.", lockError)
		}
		if len(cashBanks) == 0 {
			return core.Err("No se encontró la caja.")
		}
		return payAssetIn(transaction, req, &asset, cashBanks[0], payload)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	core.Log("PostAssetPayment asset:", asset.ID, "paid:", asset.PaidAmount, "of:", asset.PurchaseAmount)
	return req.MakeResponse(asset)
}

// payAssetIn writes one payment of the asset from cashBank, both locked by the caller, and the
// asset's new paid amount and payment status.
func payAssetIn(transaction *db.Transaction, req *core.HandlerArgs, asset *types.Asset,
	cashBank finance.CashBank, payload AssetPaymentPayload,
) error {
	if asset.PurchaseAmount <= 0 {
		return core.Err("Este activo no fue comprado, no hay nada que pagar.")
	}
	// Server-authoritative: this also blocks any payment on an already settled asset, whose
	// pending balance is 0.
	if payload.Amount > asset.PendingAmount() {
		return core.Err("El monto del pago no puede ser mayor al monto pendiente.")
	}
	if cashBank.CurrencyType != asset.CurrencyType {
		return core.Err("La moneda de la caja no coincide con la del activo.")
	}
	if cashBank.CurrentAmount-payload.Amount < 0 {
		return core.Err("El saldo de la caja no puede quedar negativo.")
	}

	// PaidAmount is recomputed from the ledger rather than incremented, so it cannot drift: the
	// committed payments, plus this one (the read sees committed rows only, and under the asset's
	// lock no other payment of it is written meanwhile). Type filters out anything that is not an
	// asset payment, since DocumentID alone is shared with the expense register.
	movements := []finance.CashBankMovement{}
	movementQuery := db.Query(&movements)
	movementQuery.Select(movementQuery.Type, movementQuery.Amount).
		CompanyID.Equals(req.User.CompanyID).
		DocumentID.Equals(int64(asset.ID))
	if queryError := movementQuery.Exec(); queryError != nil {
		return core.Err("Error al recalcular el monto pagado.", queryError)
	}
	paidAmount := payload.Amount
	for _, cashMovement := range movements {
		if cashMovement.Type != finance.CashMovementTypeAssetPayment {
			continue
		}
		paidAmount += core.If(cashMovement.Amount < 0, -cashMovement.Amount, cashMovement.Amount)
	}

	// The movement is an outflow, so its amount is negative. FinalAmount stays 0 so the cash
	// side computes the resulting balance authoritatively. When it alone pays the whole purchase on
	// its accounting date, the accounting operations infer it from the asset instead.
	paymentDate := core.If(payload.Date != 0, payload.Date, core.FechaUnix())
	isSameDayPayment := finance.IsSameDaySettlement(asset.AccountingDate(), asset.PurchaseAmount,
		asset.PaidAmount, paymentDate, payload.Amount)
	movement := finance.InternalCashMovement{
		CashBankID:         payload.CashBankID,
		DocumentID:         int64(asset.ID),
		ReferenceID:        asset.ProductID,
		Date:               paymentDate,
		Type:               finance.CashMovementTypeAssetPayment,
		CounterAccountCode: finance.PCGEAssetPayables,
		Amount:             -payload.Amount,
		FinalAmount:        0,
		SameDayMovement:    int8(core.If(isSameDayPayment, 1, 0)),
	}
	if movementError := finance.ApplyCashBankMovementIn(
		transaction, req, []finance.InternalCashMovement{movement},
	); movementError != nil {
		return movementError
	}

	asset.PaidAmount = paidAmount
	// IsFullyPaid is the write-off override: the balance is declared settled without the cash
	// covering it, which is the one case the amounts alone cannot decide.
	asset.PaymentStatus = core.If(
		payload.IsFullyPaid,
		types.AssetPaymentPaid, ResolveAssetPaymentStatus(asset.PurchaseAmount, paidAmount),
	)
	asset.Updated = core.SUnixTime()
	asset.UpdatedBy = req.User.ID
	if isSameDayPayment {
		asset.SameDayCashBankID = payload.CashBankID
	}

	assetTable := db.TableOf[types.Asset]()
	if updateError := db.UpdateIn(transaction, &[]types.Asset{*asset},
		assetTable.PaidAmount, assetTable.PaymentStatus, assetTable.Status,
		assetTable.Updated, assetTable.UpdatedBy, assetTable.SameDayCashBankID,
	); updateError != nil {
		return core.Err("Error al actualizar el pago del activo.", updateError)
	}
	return nil
}
