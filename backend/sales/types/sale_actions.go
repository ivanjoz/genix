package types

import (
	"app/core"
	finance "app/finance/types"
	logistics "app/logistics/types"
)

// The payment (2) and delivery (3) actions of a sale, shared by the till's create and update, the
// web order's settlement and the restaurant table's charge: what each stamps on the sale and the
// cash and stock movements it writes.

// SaleOrderCollectedAmount is what a payment collects when the sale owed pendingDebt and the
// client says remainingDebt stays owed. A payment must collect something while there is debt,
// and can neither overpay nor raise what is owed.
func SaleOrderCollectedAmount(pendingDebt int32, remainingDebt int32) (int32, error) {
	if remainingDebt < 0 {
		return 0, core.Err("El monto adeudado no puede ser negativo.")
	}
	if remainingDebt > pendingDebt {
		return 0, core.Err("El monto adeudado no puede superar la deuda pendiente de la venta.")
	}
	if pendingDebt > 0 && remainingDebt == pendingDebt {
		return 0, core.Err("El pago debe cobrar un monto mayor a 0.")
	}
	return pendingDebt - remainingDebt, nil
}

// StampSalePayment records a payment action on the sale, its DebtAmount already reduced: only the
// payment that clears the debt marks it Pagado, a partial one leaves it unpaid.
func StampSalePayment(sale *SaleOrder, userID int32, nowTime int32) error {
	if sale.DebtAmount == 0 {
		if err := sale.AddStatus(OrderStatusPaid); err != nil {
			return err
		}
	}
	sale.LastPaymentTime, sale.LastPaymentUser = nowTime, userID
	if sale.LastPaymentCajaID == 0 {
		return core.Err("Se requiere LastPaymentCajaID para procesar el pago.")
	}
	return nil
}

// StampSameDayCollection is whether this collection alone pays the whole sale on the sale's day:
// then the accounting operations infer it from the sale instead of downloading it, and the sale
// names the cash-bank it went to.
func StampSameDayCollection(sale *SaleOrder, collectedAmount int32, paidBefore int32, paymentDate int16) bool {
	isSameDayCollection := collectedAmount != 0 &&
		finance.IsSameDaySettlement(sale.Date, sale.TotalAmount, paidBefore, paymentDate, collectedAmount)
	if isSameDayCollection {
		sale.SameDayCashBankID = sale.LastPaymentCajaID
	}
	return isSameDayCollection
}

// StampSaleDelivery records a delivery action on the sale. It refuses a sale already delivered: a
// second delivery would move its stock twice.
func StampSaleDelivery(sale *SaleOrder, userID int32, nowTime int32) error {
	if err := sale.AddStatus(OrderStatusDelivered); err != nil {
		return err
	}
	sale.DeliveryTime, sale.DeliveryUser = nowTime, userID
	if sale.WarehouseID == 0 {
		return core.Err("Se requiere WarehouseID para procesar la entrega.")
	}
	if len(sale.Lines) == 0 {
		return core.Err("No hay productos en el detalle para procesar la entrega.")
	}
	return nil
}

// SaleCollectionMovements is the cash-bank entry of a collection, none when it collected nothing.
// The sale must have its ID: the movement names it.
func SaleCollectionMovements(sale *SaleOrder, collectedAmount int32, paymentDate int16, isSameDayCollection bool) []finance.InternalCashMovement {
	if collectedAmount == 0 {
		return []finance.InternalCashMovement{}
	}
	return []finance.InternalCashMovement{{
		CashBankID:         sale.LastPaymentCajaID,
		DocumentID:         sale.ID,
		Date:               paymentDate,
		Type:               finance.CashMovementTypeSaleCollection,
		CounterAccountCode: finance.PCGEReceivable,
		Amount:             collectedAmount,
		SameDayMovement:    int8(core.If(isSameDayCollection, 1, 0)),
	}}
}

// SaleDeliveryMovements is the warehouse exit of every line of the sale, from sale.WarehouseID, but
// the made products' lines: their exit is the daily production close's (StockExitPending).
func SaleDeliveryMovements(sale *SaleOrder) []logistics.InternalMovement {
	stockMovements := []logistics.InternalMovement{}
	for _, line := range sale.Lines {
		if line.Quantity == 0 || line.StockExitPending {
			continue
		}
		// The order line is packed; the ledger keeps the halves apart so it can accumulate
		// and be SUM()-ed. This is the single conversion point between the two forms.
		lineQuantity := core.UnpackQuantityLine(line.Quantity).Negate() // Salida de almacén
		lineDivisor := line.SubDivisor
		if lineDivisor <= 0 {
			lineDivisor = core.QuantityDivisorNone
		}
		stockMovements = append(stockMovements, logistics.InternalMovement{
			WarehouseID:    sale.WarehouseID,
			ProductID:      line.ProductID,
			PresentationID: line.PresentationID,
			SerialNumber:   line.Sku,
			LotID:          line.LotID,
			DocumentID:     sale.ID,
			Type:           logistics.StockMovementTypeSaleDelivery,
			Quantity:       lineQuantity.Units,
			SubQuantity:    lineQuantity.Sub,
			SubDivisor:     lineDivisor,
		})
	}
	return stockMovements
}
