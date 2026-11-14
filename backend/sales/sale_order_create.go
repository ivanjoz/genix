package sales

import (
	"app/cloud"
	"app/core"
	"app/db"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	logistics "app/logistics/types"
	production "app/production/types"
	saleissue "app/sales/saleissue/types"
	"app/sales/types"
	"encoding/json"
	"fmt"
	"slices"
)

// PostSaleOrder creates a sale from the till (no ID), or registers the payment and/or the delivery
// of an existing one (ID).
func PostSaleOrder(req *core.HandlerArgs) core.HandlerResponse {
	saleRequest := types.SaleOrder{}
	if err := json.Unmarshal([]byte(*req.Body), &saleRequest); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	if saleRequest.ID > 0 {
		return updateSaleOrder(req, saleRequest)
	}
	return createSaleOrder(req, saleRequest)
}

// createSaleOrder prices the till's sale against the catalog and creates it through the shared
// path every sale is born from (saleissue.PrepareNewSale).
func createSaleOrder(req *core.HandlerArgs, sale types.SaleOrder) core.HandlerResponse {
	companyID := req.User.CompanyID
	// Every line names a product and a quantity. Its prices come from the catalog.
	for _, line := range sale.Lines {
		if line.ProductID <= 0 || line.Quantity <= 0 {
			return req.MakeErr("Hay un valor incorrecto.")
		}
	}
	// A web order becomes a sale only through its settlement: a till sale names none.
	sale.OnlineSaleOrderID = 0

	// Before the lines: the currency decides what the catalog prices are converted into.
	if err := types.ValidateSaleCurrency(companyID, &sale); err != nil {
		return req.MakeErr(err)
	}
	// Resolve the divisor and the prices against the catalog rather than trusting what the client
	// sent: it is what stops a crafted request from booking a sale at an invented price.
	saleProducts, err := validateSaleOrderLines(req, &sale)
	if err != nil {
		return req.MakeErr(err)
	}

	// Runs on the total the catalog just resolved, and before the client is created: a sale that
	// cannot be issued must not leave a client row behind.
	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return req.MakeErr("Error al leer la configuración de la empresa:", err)
	}
	issueSeries, err := saleissue.ResolveSaleIssueSeries(companyID, &sale, companyConfig.Sunat.Series)
	if err != nil {
		return req.MakeErr(err)
	}

	prepared, err := saleissue.PrepareNewSale(req, saleissue.NewSaleRequest{
		Sale: sale, Products: saleProducts, IssueSeries: issueSeries,
	})
	if err != nil {
		return req.MakeErr(err)
	}
	if err := db.RunTransaction(func(transaction *db.Transaction) error {
		return saleissue.WriteNewSaleIn(transaction, req, &prepared)
	}); err != nil {
		core.Log("Error al registrar la venta:", err)
		return req.MakeErr(err)
	}
	saleissue.ScheduleNewSaleDocument(&prepared)
	return req.MakeResponse(prepared.Sale)
}

// updateSaleOrder registers the payment (2) and/or the delivery (3) of an existing sale.
func updateSaleOrder(req *core.HandlerArgs, saleRequest types.SaleOrder) core.HandlerResponse {
	companyID, userID := req.User.CompanyID, req.User.ID
	nowTime, today := core.SUnixTime(), core.FechaUnix()
	if len(saleRequest.ActionsIncluded) == 0 {
		return req.MakeErr("Se requiere ActionsIncluded para actualizar la venta.")
	}
	for _, actionID := range saleRequest.ActionsIncluded {
		if actionID != 2 && actionID != 3 {
			return req.MakeErr("ActionsIncluded solo permite 2 (pago) y 3 (entrega).")
		}
	}
	isPayment := slices.Contains(saleRequest.ActionsIncluded, 2)
	isDelivery := slices.Contains(saleRequest.ActionsIncluded, 3)

	core.Log("PostSaleOrder update requested. SaleID:", saleRequest.ID, "ActionsIncluded:", saleRequest.ActionsIncluded)
	existingSales := []types.SaleOrder{}
	query := db.Query(&existingSales)
	query.CompanyID.Equals(companyID).ID.Equals(saleRequest.ID).Limit(1)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener la venta a actualizar:", err)
	}
	if len(existingSales) == 0 {
		return req.MakeErr("No se encontró la venta a actualizar.")
	}
	// storedSale is the sale before this update: its client week rows move by what this write
	// changes from it, and its reservation is what a delivery releases.
	storedSale := existingSales[0]
	sale := existingSales[0]
	sale.ActionsIncluded = saleRequest.ActionsIncluded
	// Preserve existing payment cashBank on delivery-only updates (payload may omit LastPaymentCajaID).
	if saleRequest.LastPaymentCajaID > 0 {
		sale.LastPaymentCajaID = saleRequest.LastPaymentCajaID
	}
	if saleRequest.WarehouseID > 0 {
		sale.WarehouseID = saleRequest.WarehouseID
	}
	// PaymentDueDate is editable post-creation (e.g., reschedule a payment).
	sale.PaymentDueDate = saleRequest.PaymentDueDate

	// The server owns the debt: the client only says how much stays owed, and the amount collected
	// is the drop from what the sale owed before. paidBefore = 0 makes this payment the first one,
	// the only kind that can settle the sale the same day (finance.IsSameDaySettlement).
	collectedAmount, paidBefore := int32(0), int32(0)
	if isPayment {
		if sale.DebtAmount <= 0 {
			return req.MakeErr("La venta no tiene deuda pendiente.")
		}
		paidBefore = sale.TotalAmount - sale.DebtAmount
		var err error
		if collectedAmount, err = types.SaleOrderCollectedAmount(sale.DebtAmount, saleRequest.DebtAmount); err != nil {
			return req.MakeErr(err)
		}
		sale.DebtAmount = saleRequest.DebtAmount
		// The sale's amounts are fixed in its own currency, so a later payment must land in a
		// cash-bank of that currency.
		if err := types.ValidateSaleCurrency(companyID, &sale); err != nil {
			return req.MakeErr(err)
		}
	}

	if saleRequest.ClientInfo != nil {
		clientID, err := saleissue.ResolveSaleClientID(saleRequest.ClientInfo, companyID, userID)
		if err != nil {
			return req.MakeErr("Error al guardar el cliente de la venta:", err)
		}
		sale.ClientID = clientID
	}

	// Delivering a pending sale releases its units from the warehouse they were reserved on, which
	// the delivery may not ship from. A made product's line is delivered without its stock exit,
	// which waits for the daily production close: it keeps a reservation on the delivery warehouse.
	pendingDeliveryChanges := []logistics.PendingDeliveryChange{}
	if isDelivery {
		saleProducts, err := types.ReadSaleProducts(companyID, sale.Lines)
		if err != nil {
			return req.MakeErr(err)
		}
		sale.MarkStockExitPending(types.MadeProductIDsOf(saleProducts))
		isReserved := storedSale.IsDeliveryPending()
		reservedWarehouseID, isOnlineReservation := int32(0), sale.OnlineSaleOrderID > 0
		if isReserved {
			reservedWarehouseID, isOnlineReservation = storedSale.WarehouseID, storedSale.OnlineSaleOrderID > 0
		}
		pendingDeliveryChanges = types.DeliveryPendingChanges(sale.Lines, reservedWarehouseID, sale.WarehouseID, isOnlineReservation)
		// An early check on a plain read; the movements re-check against the rows they lock inside
		// the transaction. A made line takes the margin when it reserves on a new warehouse.
		checksMadeLines := !isReserved || storedSale.WarehouseID != sale.WarehouseID
		if err := types.ValidateSaleStock(companyID, sale, isReserved, saleProducts, checksMadeLines); err != nil {
			return req.MakeErr(err)
		}
	}

	sale.Updated, sale.UpdatedBy = nowTime, userID
	if isPayment {
		if err := types.StampSalePayment(&sale, userID, nowTime); err != nil {
			return req.MakeErr(err)
		}
	}
	isSameDayCollection := types.StampSameDayCollection(&sale, collectedAmount, paidBefore, today)
	sale.StampPeriod()
	if isDelivery {
		if err := types.StampSaleDelivery(&sale, userID, nowTime); err != nil {
			return req.MakeErr(err)
		}
	}

	cashMovements := types.SaleCollectionMovements(&sale, collectedAmount, today, isSameDayCollection)
	stockMovements := []logistics.InternalMovement{}
	if isDelivery {
		stockMovements = types.SaleDeliveryMovements(&sale)
	}

	// The update, the collection with the till's balance, and the delivery with the stock it takes
	// commit at once. A deadlock reruns the body, and each step derives its writes from what it
	// reads inside it.
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		if err := finance.ApplyCashBankMovementIn(transaction, req, cashMovements); err != nil {
			return core.Err("Error al registrar el movimiento de cashBank:", err)
		}
		if len(stockMovements) > 0 {
			if err := logistics.ApplyMovimientosIn(transaction, req, stockMovements); err != nil {
				return core.Err("Error al procesar la salida de almacén:", err)
			}
		}
		if err := logistics.ApplyPendingDeliveryIn(transaction, companyID, userID, pendingDeliveryChanges); err != nil {
			return err
		}
		if err := types.ApplyClientWeekSaleChangesIn(transaction, companyID, types.ClientWeekSaleChanges(&storedSale, &sale)); err != nil {
			return err
		}
		saleTable := db.TableOf[types.SaleOrder]()
		if err := db.UpdateIn(transaction, &[]types.SaleOrder{sale},
			// Keep composite view columns in sync: {Date, Updated} must be updated together.
			saleTable.WarehouseID,
			saleTable.LastPaymentCajaID,
			saleTable.DebtAmount,
			saleTable.Updated,
			saleTable.UpdatedBy,
			saleTable.Status,
			saleTable.LastPaymentTime,
			saleTable.LastPaymentUser,
			saleTable.DeliveryTime,
			saleTable.DeliveryUser,
			saleTable.ClientID,
			saleTable.PaymentDueDate,
			// The period delta index packs Period with Status: they are written together.
			saleTable.Period,
			saleTable.SameDayCashBankID,
			// A delivery flags the made lines whose stock exit waits for the production close.
			saleTable.Lines,
			saleTable.PendingStockMovement,
		); err != nil {
			return core.Err("Error al actualizar la venta:", err)
		}
		// A sale created from a web store order carries its payment or delivery back to it.
		return syncOnlineSaleOrderIn(transaction, &sale)
	})
	if err != nil {
		core.Log("Error al actualizar la venta:", err)
		return req.MakeErr(err)
	}
	return req.MakeResponse(sale)
}

// validateSaleOrderLines resolves each line against the product catalog: the sub-unit
// divisor, whether a sub-unit part is even allowed, and the prices. Without it the line prices and
// TotalAmount arrive straight off the request body unchecked — a client could book a sale at any
// price. It returns the catalog rows, which the sale's document describes its lines from.
func validateSaleOrderLines(req *core.HandlerArgs, sale *types.SaleOrder) (map[int32]production.Product, error) {
	if len(sale.Lines) == 0 {
		return nil, nil
	}
	productByID, err := types.ReadSaleProducts(req.User.CompanyID, sale.Lines)
	if err != nil {
		return nil, err
	}

	// A USD sale always carries its rate, even when every product is already in dollars:
	// the sale summaries restate it in PEN with it. A PEN sale only needs one for a USD product.
	needsExchangeRate := sale.CurrencyType == finance.CurrencyUSD
	for _, product := range productByID {
		if types.NormalizeCurrency(int8(product.CurrencyID)) != sale.CurrencyType {
			needsExchangeRate = true
		}
	}

	sale.ExchangeRate = 0
	if needsExchangeRate {
		exchangeRate, err := resolveSaleExchangeRate(req.User.CompanyID, sale.CurrencyType)
		if err != nil {
			return nil, err
		}
		sale.ExchangeRate = exchangeRate
	}

	for lineIndex := range sale.Lines {
		line := &sale.Lines[lineIndex]
		product, found := productByID[line.ProductID]
		if !found {
			return nil, core.Err(fmt.Sprintf("El producto %v de la venta no existe.", line.ProductID))
		}
		if err := types.PriceSaleOrderLine(line, product, sale.CurrencyType, sale.ExchangeRate); err != nil {
			return nil, err
		}
	}

	// Recompute the order total from the resolved lines for the same reason.
	totalAmount := int32(0)
	for _, line := range sale.Lines {
		totalAmount += core.QuantityAmount(core.UnpackQuantityLine(line.Quantity), line.Price, line.SubPrice)
	}
	sale.TotalAmount = totalAmount
	sale.TaxAmount = invoicing.SaleOrderTaxAmount(sale)
	return productByID, nil
}

func GetSaleOrderByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	// Only the records whose slot moved past the tick the client holds come back.
	partitions, err := db.QueryCachedIDs[types.SaleOrder](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener las ventas.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}
