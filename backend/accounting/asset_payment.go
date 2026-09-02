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
	AssetID     int32 `json:",omitempty"`
	CashBankID  int32 `json:",omitempty"`
	Amount      int32 `json:",omitempty"` // Positive payment amount, in cents.
	Date        int16 `json:",omitempty"`
	IsFullyPaid bool  `json:",omitempty"` // Forces settled status (write-off case).
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

	asset, loadError := loadAsset(req.User.CompanyID, payload.AssetID)
	if loadError != nil {
		return req.MakeErr(loadError)
	}
	if asset.PurchaseAmount <= 0 {
		return req.MakeErr("Este activo no fue comprado, no hay nada que pagar.")
	}

	// Server-authoritative: this also blocks any payment on an already settled asset, whose
	// pending balance is 0.
	if payload.Amount > asset.PendingAmount() {
		return req.MakeErr("El monto del pago no puede ser mayor al monto pendiente.")
	}

	cashBank, cashBankError := finance.GetCaja(req.User.CompanyID, payload.CashBankID)
	if cashBankError != nil {
		return req.MakeErr(cashBankError)
	}
	if cashBank.CurrencyType != asset.CurrencyType {
		return req.MakeErr("La moneda de la caja no coincide con la del activo.")
	}
	if cashBank.CurrentAmount-payload.Amount < 0 {
		return req.MakeErr("El saldo de la caja no puede quedar negativo.")
	}

	// The movement is an outflow, so its amount is negative. FinalAmount stays 0 so the cash
	// side computes the resulting balance authoritatively.
	movement := finance.InternalCashMovement{
		CashBankID:  payload.CashBankID,
		DocumentID:  int64(asset.ID),
		ReferenceID: asset.ProductID,
		Date:        payload.Date,
		Type:        finance.CashMovementTypeAssetPayment,
		Amount:      -payload.Amount,
		FinalAmount: 0,
	}
	if movementError := finance.ApplyCashBankMovement(
		req, []finance.InternalCashMovement{movement},
	); movementError != nil {
		return req.MakeErr(movementError)
	}

	// Recompute PaidAmount from the ledger rather than incrementing it, so a retry or a
	// concurrent payment cannot drift the total. Type filters out anything that is not an
	// asset payment, since DocumentID alone is shared with the expense register.
	movements := []finance.CashBankMovement{}
	movementQuery := db.Query(&movements)
	movementQuery.Select().
		CompanyID.Equals(req.User.CompanyID).
		DocumentID.Equals(int64(asset.ID))

	if queryError := movementQuery.Exec(); queryError != nil {
		return req.MakeErr("Error al recalcular el monto pagado.", queryError)
	}

	paidAmount := int32(0)
	for _, cashMovement := range movements {
		if cashMovement.Type != finance.CashMovementTypeAssetPayment {
			continue
		}
		paidAmount += core.If(cashMovement.Amount < 0, -cashMovement.Amount, cashMovement.Amount)
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

	assetTable := db.TableOf[types.Asset]()
	assetRecords := []types.Asset{asset}
	// Status is written even though a payment never changes it: it shares the delta view's
	// composite key with UpdatedVersion, so the ORM requires the pair be updated together.
	if updateError := db.Update(&assetRecords,
		assetTable.PaidAmount, assetTable.PaymentStatus, assetTable.Status,
		assetTable.Updated, assetTable.UpdatedBy,
	); updateError != nil {
		return req.MakeErr("Error al actualizar el pago del activo.", updateError)
	}

	core.Log("PostAssetPayment asset:", asset.ID, "paid:", paidAmount, "of:", asset.PurchaseAmount)
	return req.MakeResponse(assetRecords[0])
}
