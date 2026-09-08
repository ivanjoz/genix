package sales

import (
	"app/core"
	"app/db"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	logistics "app/logistics/types"
	"app/sales/types"
	"context"
	"encoding/json"
	"strings"

	"golang.org/x/sync/errgroup"
)

// warehouseMovementTypeSaleDelivery and warehouseMovementTypeSaleAnnulment are the two
// WarehouseProductMovement.Type values a sale writes. The ledger's type vocabulary has no Go enum
// (unlike finance.CashMovementType); these two are named here because the annulment nets them
// against each other and a bare 8 and 9 in that arithmetic would say nothing.
const (
	warehouseMovementTypeSaleDelivery  = int8(8)
	warehouseMovementTypeSaleAnnulment = int8(9)
)

type saleOrderAnnulRequest struct {
	ID int64
	// RefundCashBankID is where the money is withdrawn from, chosen by the operator: the refund
	// need not leave the register or account that collected it. Only the destination is theirs to
	// choose — the amount comes from the ledger.
	RefundCashBankID int32
	Reason           string
}

// PostSaleOrderAnnul voids a sale: it gives back what was collected, returns what was delivered,
// takes the sale out of the day summary and marks it annulled.
//
// What to give back is read from the two ledgers rather than derived from Status, and every
// reversal already recorded is netted out of the total. That is what makes the endpoint safe to
// retry after a partial failure: a second call gives back what the first one did not.
func PostSaleOrderAnnul(req *core.HandlerArgs) core.HandlerResponse {
	annulRequest := saleOrderAnnulRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &annulRequest); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}

	if annulRequest.ID <= 0 {
		return req.MakeErr("Se requiere el ID de la venta a anular.")
	}
	annulReason := strings.TrimSpace(annulRequest.Reason)
	if annulReason == "" {
		return req.MakeErr("Se requiere el motivo de la anulación.")
	}
	if len(annulReason) > types.MaxAnnulReasonLength {
		return req.MakeErr("El motivo de la anulación no puede superar los",
			types.MaxAnnulReasonLength, "caracteres.")
	}

	if !req.User.HasSubAcceso(types.AccesoIDGestionVentas, types.SubAccesoIDAnularVenta) {
		return req.MakeErr("No tiene permiso para anular ventas.")
	}

	// Both clicks would otherwise read a zero reversal and both write one, refunding twice.
	// Keyed on the sale, so unrelated annulments never queue behind each other.
	lock, lockErr := core.AcquireLock(context.Background(), core.ActionAnnulSaleOrder, annulRequest.ID, 2)
	if lockErr != nil {
		return lockErr.Response(req)
	}
	defer lock.Release()

	saleOrders := []types.SaleOrder{}
	saleQuery := db.Query(&saleOrders)
	saleQuery.CompanyID.Equals(req.User.CompanyID).ID.Equals(annulRequest.ID).Limit(1)
	if err := saleQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener la venta a anular:", err)
	}
	if len(saleOrders) == 0 {
		return req.MakeErr("No se encontró la venta a anular.")
	}

	sale := saleOrders[0]
	if sale.Status == types.OrderStatusAnnulled {
		return req.MakeErr("La venta ya está anulada.")
	}

	// A sale with a live electronic document cannot simply vanish: SUNAT requires a credit note,
	// which this system does not emit yet.
	invoiceDocument, invoiceErr := invoicing.FindBySaleOrder(req.User.CompanyID, sale.ID)
	if invoiceErr != nil {
		return req.MakeErr("Error al verificar si la venta fue facturada:", invoiceErr)
	}
	if invoiceDocument != nil {
		// The series code is not named here on purpose: it lives on the company
		// record, and reaching it from this module would cross a module body. The
		// series number and the correlativo identify the document well enough for
		// somebody to go find it.
		return req.MakeErr("La venta tiene un comprobante emitido (serie",
			invoiceDocument.SeriesID(), "correlativo", invoiceDocument.Correlativo,
			"); requiere una nota de crédito.")
	}

	// Read both ledgers before anything is written: the refund amount decides whether a cash bank
	// is required at all, so demanding one on a sale that was never paid would be wrong.
	refundAmount, stockReturns, ledgerErr := readSaleOrderReversal(req.User.CompanyID, sale.ID)
	if ledgerErr != nil {
		return req.MakeErr(ledgerErr)
	}

	refundMovements := []finance.InternalCashMovement{}
	if refundAmount > 0 {
		if annulRequest.RefundCashBankID == 0 {
			return req.MakeErr("Se requiere la caja de la cual se devolverá el dinero.")
		}
		// GetCaja already rejects a cash bank of another company or one that does not exist;
		// ApplyCashBankMovement checks no status, so an inactive one is refused here.
		refundCashBank, cashBankErr := finance.GetCaja(req.User.CompanyID, annulRequest.RefundCashBankID)
		if cashBankErr != nil {
			return req.MakeErr("Error al obtener la caja de la devolución:", cashBankErr)
		}
		if refundCashBank.Status == 0 {
			return req.MakeErr("La caja seleccionada está inactiva.")
		}

		refundMovements = append(refundMovements, finance.InternalCashMovement{
			CashBankID: annulRequest.RefundCashBankID,
			DocumentID: sale.ID,
			Type:       finance.CashMovementTypeSaleRefund,
			Amount:     -refundAmount,
		})
	}

	core.Log("PostSaleOrderAnnul reversing sale:", "saleID", sale.ID, "status", sale.Status,
		"refundAmount", refundAmount, "stockReturns", len(stockReturns))

	// Money and stock first, and the status last: a failure here leaves the sale un-annulled and
	// retryable, where the reverse order would leave a sale marked void with the money still taken.
	eg := errgroup.Group{}
	if len(refundMovements) > 0 {
		eg.Go(func() error {
			if err := finance.ApplyCashBankMovement(req, refundMovements); err != nil {
				return core.Err("Error al registrar la devolución del dinero:", err)
			}
			return nil
		})
	}
	if len(stockReturns) > 0 {
		eg.Go(func() error {
			if err := logistics.ApplyMovimientos(req, stockReturns); err != nil {
				return core.Err("Error al reingresar los productos al almacén:", err)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return req.MakeErr(err)
	}

	// Sequenced after the two ledgers rather than beside them, because it is the one reversal
	// that cannot be netted: the summary keeps per-day totals, not per-sale rows, so nothing
	// records that this sale was already taken out of them. Running it only once the reversals
	// that CAN be retried have succeeded is what keeps a failed annulment from subtracting the
	// sale from the day twice. The counters clamp at zero, so the residual window — a failure
	// between here and the status write below — can under-count a day but never invent stock.
	if err := annulSaleSummary(sale); err != nil {
		return req.MakeErr("Error al actualizar el resumen de ventas:", err)
	}

	sale.Status = types.OrderStatusAnnulled
	sale.AnnulReason = annulReason
	sale.Updated = core.SUnixTime()
	sale.UpdatedBy = req.User.ID

	saleTable := db.TableOf[types.SaleOrder]()
	salesToUpdate := []types.SaleOrder{sale}
	if err := db.Update(&salesToUpdate,
		saleTable.Status,
		saleTable.AnnulReason,
		saleTable.Updated,
		saleTable.UpdatedBy,
	); err != nil {
		return req.MakeErr("Error al anular la venta:", err)
	}

	return req.MakeResponse(sale)
}

// readSaleOrderReversal asks both ledgers what this sale still owes back.
func readSaleOrderReversal(companyID int32, saleOrderID int64) (int32, []logistics.InternalMovement, error) {
	cashMovements := []finance.CashBankMovement{}
	stockMovements := []logistics.WarehouseProductMovement{}

	eg := errgroup.Group{}
	eg.Go(func() error {
		query := db.Query(&cashMovements)
		query.Select(query.Type, query.Amount).
			CompanyID.Equals(companyID).
			DocumentID.Equals(saleOrderID)
		if err := query.Exec(); err != nil {
			return core.Err("Error al obtener los movimientos de caja de la venta:", err)
		}
		return nil
	})
	eg.Go(func() error {
		query := db.Query(&stockMovements)
		query.Select(query.Type, query.WarehouseID, query.ProductID, query.PresentationID,
			query.LotID, query.SerialNumber, query.Quantity, query.SubQuantity, query.SubDivisor).
			CompanyID.Equals(companyID).
			DocumentID.Equals(saleOrderID)
		if err := query.Exec(); err != nil {
			return core.Err("Error al obtener los movimientos de almacén de la venta:", err)
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return 0, nil, err
	}

	return netCashRefundAmount(cashMovements), netStockReturns(stockMovements, saleOrderID), nil
}

// netCashRefundAmount is what the sale collected minus what it has already given back, so the
// result is what is still owed. It sums across cash banks on purpose: the refund may be paid from
// a different register than the one that collected, and a per-bank sum would then see an unpaid
// collection next to an unexplained outflow instead of a settled pair.
func netCashRefundAmount(cashMovements []finance.CashBankMovement) int32 {
	refundAmount := int32(0)
	for _, cashMovement := range cashMovements {
		if cashMovement.Type != finance.CashMovementTypeSaleCollection &&
			cashMovement.Type != finance.CashMovementTypeSaleRefund {
			continue
		}
		refundAmount += cashMovement.Amount
	}
	if refundAmount < 0 {
		return 0
	}
	return refundAmount
}

// netStockReturns is what left the warehouse minus what has already come back, per physical
// bucket. Reversing the ledger rather than rebuilding from the sale's detail lines is what returns
// each unit to the exact lot, serial and presentation it left from — and to the warehouse it
// actually left, which the sale's own WarehouseID may no longer name.
func netStockReturns(stockMovements []logistics.WarehouseProductMovement,
	saleOrderID int64) []logistics.InternalMovement {

	// One bucket per physically distinct location. SubDivisor joins the key because two rows
	// counted in different divisors cannot be added without converting first, and a divisor
	// refinement mid-sale would otherwise add sixths to twelfths.
	type stockBucket struct {
		warehouseID    int32
		productID      int32
		presentationID int16
		lotID          int32
		serialNumber   string
		subDivisor     int16
	}

	deliveredByBucket := map[stockBucket]core.Quantity{}
	bucketOrder := []stockBucket{}

	for _, stockMovement := range stockMovements {
		if stockMovement.Type != warehouseMovementTypeSaleDelivery &&
			stockMovement.Type != warehouseMovementTypeSaleAnnulment {
			continue
		}
		// A row written before the product had a sub-unit carries divisor 0, which means the same
		// thing as 1 — normalized here so both land in one bucket and the arithmetic below has a
		// divisor it can multiply by.
		bucketDivisor := stockMovement.SubDivisor
		if bucketDivisor <= 0 {
			bucketDivisor = core.QuantityDivisorNone
		}
		bucket := stockBucket{
			warehouseID:    stockMovement.WarehouseID,
			productID:      stockMovement.ProductID,
			presentationID: stockMovement.PresentationID,
			lotID:          stockMovement.LotID,
			serialNumber:   stockMovement.SerialNumber,
			subDivisor:     bucketDivisor,
		}
		if _, seen := deliveredByBucket[bucket]; !seen {
			bucketOrder = append(bucketOrder, bucket)
		}
		deliveredByBucket[bucket] = deliveredByBucket[bucket].Add(
			core.Quantity{Units: stockMovement.Quantity, Sub: stockMovement.SubQuantity})
	}

	stockReturns := []logistics.InternalMovement{}
	for _, bucket := range bucketOrder {
		// A delivery is negative on the ledger, so what is still out is a negative sum and the
		// return is its negation. A bucket already fully returned nets to zero and is skipped.
		// IsNegative rather than a Units check: Add carries nothing, so a partially returned
		// bucket can sit at {0, -4}, which is still four sub-units owed.
		stillDelivered := deliveredByBucket[bucket]
		if !stillDelivered.IsNegative(bucket.subDivisor) {
			continue
		}
		returnQuantity := stillDelivered.Negate()

		stockReturns = append(stockReturns, logistics.InternalMovement{
			WarehouseID:    bucket.warehouseID,
			ProductID:      bucket.productID,
			PresentationID: bucket.presentationID,
			LotID:          bucket.lotID,
			SerialNumber:   bucket.serialNumber,
			SubDivisor:     bucket.subDivisor,
			Quantity:       returnQuantity.Units,
			SubQuantity:    returnQuantity.Sub,
			// The reversal has to carry the sale it undoes, or the next annulment of this sale
			// could not find it and would return the stock a second time.
			DocumentID: saleOrderID,
			Type:       warehouseMovementTypeSaleAnnulment,
		})
	}
	return stockReturns
}
