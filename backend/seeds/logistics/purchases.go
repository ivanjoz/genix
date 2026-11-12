package logistics

import (
	"app/core"
	financeTypes "app/finance/types"
	logisticsModule "app/logistics"
	logisticsTypes "app/logistics/types"
	"app/seeds/runner"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// erpProductClass is how a product is received in one purchase order. The same product can be
// free on one day and lot-tracked on another, exactly like a real catalog.
type erpProductClass int8

const (
	erpProductFree erpProductClass = iota
	erpProductLotted
	erpProductSerialized
)

const (
	erpLotCodePrefix = "L"
	erpSerialPrefix  = "SN"
)

// RunPurchaseOrdersStep buys one order per warehouse and re-reads the stock they received.
func RunPurchaseOrdersStep(run *runner.Run, dayStart time.Time) error {
	for purchaseOrderIndex, warehouseID := range run.Config.WarehouseIDs {
		// Already written before the interruption; replaying it would duplicate the order.
		if purchaseOrderIndex < run.State.CompletedPurchaseOrders {
			continue
		}
		if err := RunPurchaseOrder(run, dayStart, purchaseOrderIndex, warehouseID); err != nil {
			return err
		}
		run.State.CompletedPurchaseOrders = purchaseOrderIndex + 1
		if err := run.SaveState(); err != nil {
			return err
		}
	}

	// Reception changed the warehouses, so the ledger is re-read from the API before selling.
	for _, warehouseID := range run.Config.WarehouseIDs {
		if err := run.ReloadWarehouseLedger(warehouseID); err != nil {
			return core.Err("no se pudo recargar el stock del almacén", warehouseID, ":", err)
		}
	}
	return nil
}

// RunPurchaseOrder walks one order through its full lifecycle: create → confirm → pay → receive.
func RunPurchaseOrder(run *runner.Run, dayStart time.Time, purchaseOrderIndex int, warehouseID int32) error {
	orderSlot := time.Duration(purchaseOrderIndex) * 5 * time.Minute
	selectedProductIDs := run.PickDistinctFromPool(run.Config.PurchaseOrderProducts)
	classByProduct := classifyPurchaseProducts(run, selectedProductIDs)

	quantityByProduct := map[int32]int32{}
	orderPayload := logisticsTypes.PurchaseOrder{
		ProviderID:  run.ProviderIDs[run.Random.Intn(len(run.ProviderIDs))],
		WarehouseID: warehouseID,
	}
	for _, productID := range selectedProductIDs {
		// Serialized products move in single units, so they are bought in small numbers; the rest
		// arrive in case-sized quantities that outpace one day of demand.
		quantity := int32(12 + run.Random.Intn(29))
		if classByProduct[productID] == erpProductSerialized {
			quantity = int32(2 + run.Random.Intn(3))
		}
		quantityByProduct[productID] = quantity

		orderPayload.Lines = append(orderPayload.Lines, logisticsTypes.PurchaseOrderLine{
			ProductID: productID, Quantity: quantity, Price: run.CostPriceByProduct[productID],
		})
		orderPayload.TotalAmount += quantity * run.CostPriceByProduct[productID]
	}

	createdOrder, err := createPurchaseOrder(run, dayStart.Add(8*time.Hour+orderSlot), orderPayload)
	if err != nil {
		return core.Err("error al crear la orden de compra:", err)
	}

	if err := confirmPurchaseOrder(run, dayStart.Add(8*time.Hour+orderSlot+time.Minute), createdOrder.ID); err != nil {
		return core.Err("error al confirmar la orden de compra:", err)
	}

	// The supplier's factura arrives with the goods and is booked before paying, so the payment
	// clears the invoices payable (4212) instead of an undocumented payable.
	if err := registerPurchaseOrderDocument(run, dayStart.Add(8*time.Hour+orderSlot+90*time.Second), createdOrder); err != nil {
		return core.Err("error al registrar la factura de la orden de compra:", err)
	}

	if createdOrder.DebtAmount > 0 {
		if err := payPurchaseOrder(run, dayStart, orderSlot, createdOrder.ID, createdOrder.DebtAmount); err != nil {
			return core.Err("error al pagar la orden de compra:", err)
		}
	}

	receivedUnits, err := receivePurchaseOrder(run,
		dayStart.Add(10*time.Hour+orderSlot), createdOrder.ID, warehouseID, selectedProductIDs, classByProduct, quantityByProduct)
	if err != nil {
		return core.Err("error al recibir la orden de compra:", err)
	}

	run.State.Stats.PurchaseOrders++
	run.State.Stats.ReceivedUnits += receivedUnits
	core.Log("Seed:: OC recibida", "id", createdOrder.ID, "almacén", warehouseID,
		"total", createdOrder.TotalAmount, "unidades", receivedUnits)
	return nil
}

// classifyPurchaseProducts splits the order's products into free, lot-tracked and serial-tracked.
func classifyPurchaseProducts(run *runner.Run, productIDs []int32) map[int32]erpProductClass {
	classByProduct := map[int32]erpProductClass{}
	for productIndex, productID := range productIDs {
		switch {
		case productIndex < run.Config.LotProducts:
			classByProduct[productID] = erpProductLotted
		case productIndex < run.Config.LotProducts+run.Config.SerialProducts:
			classByProduct[productID] = erpProductSerialized
		default:
			classByProduct[productID] = erpProductFree
		}
	}
	return classByProduct
}

func createPurchaseOrder(run *runner.Run, at time.Time, payload logisticsTypes.PurchaseOrder) (logisticsTypes.PurchaseOrder, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return logisticsTypes.PurchaseOrder{}, err
	}
	core.SetHistoricalUnix(at.Unix())
	request := run.MakeRequest("POST.purchase-orders", nil, string(bodyBytes))
	createdOrder := logisticsTypes.PurchaseOrder{}
	if err := runner.DecodeResponse(run.Send(request, logisticsModule.PostPurchaseOrder), &createdOrder); err != nil {
		return logisticsTypes.PurchaseOrder{}, err
	}
	if createdOrder.ID == 0 {
		return logisticsTypes.PurchaseOrder{}, core.Err("la orden de compra no devolvió ID")
	}
	return createdOrder, nil
}

func confirmPurchaseOrder(run *runner.Run, at time.Time, orderID int32) error {
	core.SetHistoricalUnix(at.Unix())
	request := run.MakeRequest("PUT.purchase-orders", map[string]string{
		"action": strconv.Itoa(logisticsModule.PurchaseOrderActionConfirm),
		"id":     strconv.Itoa(int(orderID)),
	}, "")
	if response := run.Send(request, logisticsModule.PutPurchaseOrder); response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	return nil
}

// registerPurchaseOrderDocument books the supplier's factura (PUT.purchase-orders action 5): one
// series per supplier and an increasing number, the order total split into base and 18% IGV.
func registerPurchaseOrderDocument(run *runner.Run, at time.Time, order logisticsTypes.PurchaseOrder) error {
	issueDay := core.TimeToFechaUnix(at)
	documentKey := fmt.Sprint("po-", order.ProviderID)
	document := runner.MakePurchaseDocument(financeTypes.PurchaseDocTypeFactura,
		fmt.Sprintf("F%03d", order.ProviderID%1000), run.NextDocNumber(documentKey), issueDay, order.TotalAmount)

	patch := logisticsTypes.PurchaseOrder{}
	patch.SetPurchaseDocument(document)
	bodyBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	core.SetHistoricalUnix(at.Unix())
	request := run.MakeRequest("PUT.purchase-orders", map[string]string{
		"action": strconv.Itoa(logisticsModule.PurchaseOrderActionDocument),
		"id":     strconv.Itoa(int(order.ID)),
	}, string(bodyBytes))
	if response := run.Send(request, logisticsModule.PutPurchaseOrder); response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	run.AddPurchaseTax(issueDay, document.TaxAmount)
	return nil
}

// payPurchaseOrder tops the cash register up first when the balance cannot cover the invoice:
// ApplyCashBankMovement does not stop a negative balance, so keeping it solvent is on the caller.
func payPurchaseOrder(run *runner.Run, dayStart time.Time, orderSlot time.Duration, orderID int32, amountToPay int32) error {
	if err := run.EnsureCashBalance(run.BankCashBankID, amountToPay, dayStart.Add(7*time.Hour+orderSlot)); err != nil {
		return err
	}

	payloadBytes, err := json.Marshal(map[string]any{
		"CashBankID": run.BankCashBankID,
		"Amount":     amountToPay,
	})
	if err != nil {
		return err
	}
	core.SetHistoricalUnix(dayStart.Add(8*time.Hour + orderSlot + 2*time.Minute).Unix())
	request := run.MakeRequest("PUT.purchase-orders", map[string]string{
		"action": strconv.Itoa(logisticsModule.PurchaseOrderActionPay),
		"id":     strconv.Itoa(int(orderID)),
	}, string(payloadBytes))
	if response := run.Send(request, logisticsModule.PutPurchaseOrder); response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	return nil
}

// receivePurchaseOrder expands the order into stock entry items. Serial-tracked products become
// one item of quantity 1 per unit, which is what creates an individually traceable unit.
func receivePurchaseOrder(run *runner.Run,
	at time.Time, orderID int32, warehouseID int32, productIDs []int32,
	classByProduct map[int32]erpProductClass, quantityByProduct map[int32]int32,
) (int32, error) {
	entryItems := []logisticsModule.PostStockAdjustItem{}
	receivedUnits := int32(0)
	lotDateToken := strconv.Itoa(int(core.TimeToFechaUnix(at)))

	for _, productID := range productIDs {
		quantity := quantityByProduct[productID]
		receivedUnits += quantity

		switch classByProduct[productID] {
		case erpProductLotted:
			// LotCode with no LotID makes the backend resolve or create the lot by
			// Hash(date, supplier, name) — dated on the simulated day thanks to the frozen clock.
			entryItems = append(entryItems, logisticsModule.PostStockAdjustItem{
				WarehouseID: warehouseID,
				ProductID:   productID,
				Quantity:    quantity,
				LotCode:     fmt.Sprint(erpLotCodePrefix, lotDateToken, "-", productID),
			})
		case erpProductSerialized:
			for unitIndex := int32(0); unitIndex < quantity; unitIndex++ {
				entryItems = append(entryItems, logisticsModule.PostStockAdjustItem{
					WarehouseID:  warehouseID,
					ProductID:    productID,
					Quantity:     1,
					SerialNumber: fmt.Sprint(erpSerialPrefix, "-", lotDateToken, "-", productID, "-", unitIndex+1),
				})
			}
		default:
			entryItems = append(entryItems, logisticsModule.PostStockAdjustItem{
				WarehouseID: warehouseID,
				ProductID:   productID,
				Quantity:    quantity,
			})
		}
	}

	payloadBytes, err := json.Marshal(map[string]any{
		"PurchaseOrderID": orderID,
		"WarehouseID":     warehouseID,
		"Items":           entryItems,
	})
	if err != nil {
		return 0, err
	}

	core.SetHistoricalUnix(at.Unix())
	request := run.MakeRequest("POST.purchase-order-entry", nil, string(payloadBytes))
	if response := run.Send(request, logisticsModule.PostPurchaseOrderEntry); response.StatusCode != 200 {
		return 0, core.Err(response.Error)
	}
	return receivedUnits, nil
}

// ─── Sales ─────────────────────────────────────────────────────────────────────
