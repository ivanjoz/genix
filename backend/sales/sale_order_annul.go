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
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"
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
// reversal already recorded is netted out of the total, so a reversal is never given twice.
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

	// Every sale issued under a series carries a document from birth, so what decides this is
	// whether it was sent. One still waiting for the sweep is voided and never goes out; one that
	// reached SUNAT cannot simply vanish, because SUNAT requires a credit note and this system
	// does not emit them yet.
	//
	// Under the invoicing lock, which is the one the sweep takes before sending, held until the
	// annulment commits: without it the sweep could send the document a moment after this decided
	// to void it, or right after a failed annulment rolled the void back.
	invoiceLock, invoiceLockErr := core.AcquireLock(
		context.Background(), core.ActionInvoiceSaleOrder, annulRequest.ID, 2)
	if invoiceLockErr != nil {
		return invoiceLockErr.Response(req)
	}
	defer invoiceLock.Release()

	// One transaction under the sale's row lock: a second annulment waits on the row, then finds it
	// annulled, so two clicks can never both read a zero reversal and both refund. Everything — the
	// void, the refund, the stock return, the restamp and the status — commits or rolls back
	// together, and a deadlock reruns the body from the reads.
	var sale types.SaleOrder
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		saleOrders := []types.SaleOrder{{CompanyID: req.User.CompanyID, ID: annulRequest.ID}}
		if err := db.LockByKeys(transaction, &saleOrders); err != nil {
			return core.Err("Error al obtener la venta a anular:", err)
		}
		if len(saleOrders) == 0 {
			return core.Err("No se encontró la venta a anular.")
		}
		sale = saleOrders[0]
		storedSale := saleOrders[0]
		if sale.Status == types.OrderStatusAnnulled {
			return core.Err("La venta ya está anulada.")
		}

		invoiceDocument, invoiceErr := invoicing.VoidPendingDocumentForSaleIn(transaction, req.User.CompanyID, sale.ID)
		if invoiceErr != nil {
			return core.Err("Error al verificar si la venta fue facturada:", invoiceErr)
		}
		if invoiceDocument != nil {
			// The series code is not named here on purpose: it lives on the company
			// record, and reaching it from this module would cross a module body. The
			// series number and the correlativo identify the document well enough for
			// somebody to go find it.
			return core.Err("La venta tiene un comprobante emitido (serie",
				invoiceDocument.SeriesID(), "correlativo", invoiceDocument.Correlativo,
				"); requiere una nota de crédito.")
		}

		// Read both ledgers before anything is written: the refund amount decides whether a cash bank
		// is required at all, so demanding one on a sale that was never paid would be wrong. Under the
		// sale's lock no reversal of it is written meanwhile.
		refundAmount, stockReturns, ledgerErr := readSaleOrderReversal(req.User.CompanyID, sale.ID)
		if ledgerErr != nil {
			return ledgerErr
		}

		if refundAmount > 0 {
			if annulRequest.RefundCashBankID == 0 {
				return core.Err("Se requiere la caja de la cual se devolverá el dinero.")
			}
			// GetCaja already rejects a cash bank of another company or one that does not exist;
			// ApplyCashBankMovementIn checks no status, so an inactive one is refused here.
			refundCashBank, cashBankErr := finance.GetCaja(req.User.CompanyID, annulRequest.RefundCashBankID)
			if cashBankErr != nil {
				return core.Err("Error al obtener la caja de la devolución:", cashBankErr)
			}
			if refundCashBank.Status == 0 {
				return core.Err("La caja seleccionada está inactiva.")
			}
			// The refund gives back the collected amount as it was booked: in the sale's currency.
			saleCurrency := types.NormalizeCurrency(sale.CurrencyType)
			if refundCurrency := types.NormalizeCurrency(refundCashBank.CurrencyType); refundCurrency != saleCurrency {
				return core.Err(fmt.Sprintf(`La venta está en %v pero la caja "%v" es en %v.`,
					types.CurrencyName(saleCurrency), refundCashBank.Name, types.CurrencyName(refundCurrency)))
			}

			// Written already final (no same-day flag, the receivable account), as the restamp below
			// reads committed payments only.
			refundMovements := []finance.InternalCashMovement{{
				CashBankID:         annulRequest.RefundCashBankID,
				DocumentID:         sale.ID,
				Type:               finance.CashMovementTypeSaleRefund,
				CounterAccountCode: finance.PCGEReceivable,
				Amount:             -refundAmount,
			}}
			if err := finance.ApplyCashBankMovementIn(transaction, req, refundMovements); err != nil {
				return core.Err("Error al registrar la devolución del dinero:", err)
			}
		}

		core.Log("PostSaleOrderAnnul reversing sale:", "saleID", sale.ID, "status", sale.Status,
			"refundAmount", refundAmount, "stockReturns", len(stockReturns))

		if err := logistics.ApplyMovimientosIn(transaction, req, stockReturns); err != nil {
			return core.Err("Error al reingresar los productos al almacén:", err)
		}
		// An undelivered sale gives back the units it held on its warehouse; a delivered one, those its
		// made products' lines held until the production close took their stock. Those never left the
		// warehouse, so the ledger return above has nothing of them.
		releasedLines := []types.SaleOrderLine{}
		if storedSale.IsDeliveryPending() {
			releasedLines = storedSale.Lines
		} else if storedSale.PendingStockMovement == 1 {
			releasedLines = types.LinesWithStockExitPending(storedSale.Lines, true)
		}
		if err := logistics.ApplyPendingDeliveryIn(transaction, req.User.CompanyID, req.User.ID,
			types.PendingDeliveryChanges(releasedLines, storedSale.WarehouseID, storedSale.OnlineSaleOrderID > 0, -1)); err != nil {
			return err
		}

		// An annulled sale leaves the accounting operations' cache, so a collection they inferred from it
		// must become a movement they download, like its refund: the two cancel in 101 and 1212. A zero
		// total is what says the annulled sale settles nothing.
		sameDayCashBankID, restampErr := finance.RestampDocumentPaymentsIn(transaction, req.User.CompanyID, sale.ID,
			[]finance.CashMovementType{finance.CashMovementTypeSaleCollection, finance.CashMovementTypeSaleRefund},
			sale.Date, 0, finance.PCGEReceivable)
		if restampErr != nil {
			return restampErr
		}

		sale.Status = types.OrderStatusAnnulled
		sale.AnnulReason = annulReason
		sale.Updated = core.SUnixTime()
		sale.UpdatedBy = req.User.ID
		sale.SameDayCashBankID = sameDayCashBankID
		// Nothing is left for the production close to settle.
		sale.PendingStockMovement = 0
		sale.StampPeriod()

		saleTable := db.TableOf[types.SaleOrder]()
		salesToUpdate := []types.SaleOrder{sale}
		if err := db.UpdateIn(transaction, &salesToUpdate,
			saleTable.Status,
			saleTable.AnnulReason,
			saleTable.Updated,
			saleTable.UpdatedBy,
			// The period delta index packs Period with Status: they are written together.
			saleTable.Period,
			saleTable.SameDayCashBankID,
			saleTable.PendingStockMovement,
		); err != nil {
			return core.Err("Error al anular la venta:", err)
		}
		// A sale created from a web store order annuls that order too.
		if err := syncOnlineSaleOrderIn(transaction, &sale); err != nil {
			return err
		}
		// The annulled sale's purchase and payments leave its client's week, from the sale's day.
		return types.ApplyClientWeekSaleChangesIn(transaction, req.User.CompanyID, types.ClientWeekSaleChanges(&storedSale, &sale))
	})
	if err != nil {
		return req.MakeErr(err)
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
			query.LotID, query.SerialNumber, query.Quantity, query.SubQuantity, query.SubDivisor,
			query.MonetaryValue).
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
	// The value still out per bucket: what the deliveries took minus what returns brought back.
	// MonetaryValue is positive on both sides, so the type decides the sign.
	deliveredValueByBucket := map[stockBucket]int32{}
	bucketOrder := []stockBucket{}

	for _, stockMovement := range stockMovements {
		if stockMovement.Type != logistics.StockMovementTypeSaleDelivery &&
			stockMovement.Type != logistics.StockMovementTypeSaleAnnulmentReturn {
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
		deliveredValueByBucket[bucket] += core.If(
			stockMovement.Type == logistics.StockMovementTypeSaleDelivery,
			stockMovement.MonetaryValue, -stockMovement.MonetaryValue)
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
			// The goods come back at the value they left with, not re-priced at today's average,
			// so the sale's cost (deliveries minus returns) nets to exactly 0. An uncosted delivery
			// has no value to return; the engine then values the return at the row's average.
			InflowValue: max(deliveredValueByBucket[bucket], 0),
			// The reversal has to carry the sale it undoes, or the next annulment of this sale
			// could not find it and would return the stock a second time.
			DocumentID: saleOrderID,
			Type:       logistics.StockMovementTypeSaleAnnulmentReturn,
		})
	}
	return stockReturns
}
