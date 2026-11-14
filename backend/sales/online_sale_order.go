package sales

import (
	"app/core"
	"app/db"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	logistics "app/logistics/types"
	saleissue "app/sales/saleissue/types"
	"app/sales/types"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Web store orders (OnlineSaleOrder) and the SaleOrder each becomes. See
// docs/ONLINE_SALE_ORDER_PLAN.md.
//
// An online order becomes a sale the moment it is paid or its goods are delivered, whichever comes
// first: that is when the sale's number — the boleta's correlativo — is minted, so an order that is
// declined or never paid spends none. Until then every action runs on the online order; after, they
// run on the SaleOrder (PostSaleOrder, PostSaleOrderAnnul), which copies its status back here.

// onlineSaleOrderStatusGroups are the Pedidos Web tabs: pending, in progress, completed, annulled.
var onlineSaleOrderStatusGroups = map[int32][]int8{
	1: {types.OrderStatusPending},
	2: {types.OrderStatusPaid, types.OrderStatusDelivered},
	3: {types.OrderStatusCompleted},
	0: {types.OrderStatusAnnulled},
}

// GetOnlineSaleOrders lists one Pedidos Web tab (?status-group=), delta-synced under "records".
func GetOnlineSaleOrders(req *core.HandlerArgs) core.HandlerResponse {
	statuses, isKnownGroup := onlineSaleOrderStatusGroups[req.GetQueryInt("status-group")]
	if !isKnownGroup {
		return req.MakeErr("El grupo de estados es incorrecto.")
	}
	orders := []types.OnlineSaleOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("records"), core.ToAny(statuses)...)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los pedidos web:", err)
	}
	return req.MakeResponse(map[string]any{"records": &orders})
}

type onlineSaleOrderSettleRequest struct {
	ID int32
	// ActionsIncluded is what happened to the order: 2 paid, 3 delivered, or both.
	ActionsIncluded []int8
	CashBankID      int32
	WarehouseID     int32
}

// PostOnlineSaleOrderSettle registers, from Pedidos Web, the payment and/or the delivery of an order
// that has no sale yet, which creates it.
func PostOnlineSaleOrderSettle(req *core.HandlerArgs) core.HandlerResponse {
	settleRequest := onlineSaleOrderSettleRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &settleRequest); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	onlineOrder, sale, err := settleOnlineSaleOrder(req, settleRequest, "")
	if lockErr, isLockErr := err.(*core.LockError); isLockErr {
		return lockErr.Response(req)
	}
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(map[string]any{"Order": onlineOrder, "SaleOrder": sale})
}

// PostOnlineSaleOrderAnnul annuls, from Pedidos Web, an order that has no sale yet. One with a sale is
// annulled through the sale, in Gestión Ventas.
func PostOnlineSaleOrderAnnul(req *core.HandlerArgs) core.HandlerResponse {
	annulRequest := struct {
		ID     int32
		Reason string
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &annulRequest); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	reason := strings.TrimSpace(annulRequest.Reason)
	if reason == "" {
		return req.MakeErr("Se requiere el motivo de la anulación.")
	}
	if len(reason) > types.MaxAnnulReasonLength {
		return req.MakeErr("El motivo de la anulación no puede superar los", types.MaxAnnulReasonLength, "caracteres.")
	}
	if !req.User.HasSubAcceso(types.AccesoIDPedidosWeb, types.SubAccesoIDAnularPedidoWeb) {
		return req.MakeErr("No tiene permiso para anular pedidos web.")
	}
	onlineOrder, err := annulOnlineSaleOrder(req.User.CompanyID, annulRequest.ID, reason, req.User.ID)
	if lockErr, isLockErr := err.(*core.LockError); isLockErr {
		return lockErr.Response(req)
	}
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(onlineOrder)
}

// PostOnlineSaleOrderFulfillment moves an order forward to "in preparation" or "on the way / ready
// for pickup". Only a status label: no stock moves and no sale is created. It works with or without a
// SaleOrder, since the SaleOrder knows nothing of these steps. The delivery itself is a settle (no
// sale yet) or a SaleOrder delivery.
func PostOnlineSaleOrderFulfillment(req *core.HandlerArgs) core.HandlerResponse {
	fulfillmentRequest := struct {
		ID                int32
		FulfillmentStatus int8
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &fulfillmentRequest); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	nextStatus := fulfillmentRequest.FulfillmentStatus
	if nextStatus != types.FulfillmentPreparing && nextStatus != types.FulfillmentInTransit {
		return req.MakeErr("FulfillmentStatus solo permite 1 (en preparación) y 2 (en camino / para recoger).")
	}
	// The same lock as a settle: a delivery being registered cannot interleave with this write.
	settleLock, err := lockOnlineSaleOrder(req.User.CompanyID, fulfillmentRequest.ID)
	if lockErr, isLockErr := err.(*core.LockError); isLockErr {
		return lockErr.Response(req)
	}
	if err != nil {
		return req.MakeErr(err)
	}
	defer settleLock.Release()

	var updatedOrder types.OnlineSaleOrder
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		lockedOrder, err := lockOnlineSaleOrderRowIn(transaction, req.User.CompanyID, fulfillmentRequest.ID)
		if err != nil {
			return err
		}
		if lockedOrder.Status == types.OrderStatusAnnulled {
			return core.Err(fmt.Sprintf("El pedido N° %v está anulado.", lockedOrder.ID))
		}
		if lockedOrder.Status == types.OrderStatusDelivered || lockedOrder.Status == types.OrderStatusCompleted {
			return core.Err(fmt.Sprintf("El pedido N° %v ya fue entregado.", lockedOrder.ID))
		}
		// Forward only: a step can be skipped but never undone.
		if nextStatus <= lockedOrder.FulfillmentStatus {
			return core.Err(fmt.Sprintf("El pedido N° %v ya está en ese estado o en uno posterior.", lockedOrder.ID))
		}
		lockedOrder.FulfillmentStatus = nextStatus
		lockedOrder.Updated, lockedOrder.UpdatedBy = core.SUnixTime(), req.User.ID
		updatedOrder = lockedOrder
		orderTable := db.TableOf[types.OnlineSaleOrder]()
		if err := db.UpdateIn(transaction, &[]types.OnlineSaleOrder{lockedOrder}, orderTable.FulfillmentStatus,
			orderTable.Updated, orderTable.UpdatedBy); err != nil {
			return core.Err("Error al actualizar el estado del pedido web:", err)
		}
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(updatedOrder)
}

// lockOnlineSaleOrder takes the order's settle lock (core.ActionOnlineSaleOrderSettle).
func lockOnlineSaleOrder(companyID, onlineOrderID int32) (core.Lock, error) {
	lock, lockErr := core.AcquireLock(context.Background(), core.ActionOnlineSaleOrderSettle,
		int64(companyID)<<32|int64(onlineOrderID), 2)
	if lockErr != nil {
		return nil, lockErr
	}
	return lock, nil
}

// lockOnlineSaleOrderRowIn reads the order under its row lock, inside the caller's transaction.
func lockOnlineSaleOrderRowIn(transaction *db.Transaction, companyID, onlineOrderID int32) (types.OnlineSaleOrder, error) {
	lockedOrders := []types.OnlineSaleOrder{{CompanyID: companyID, ID: onlineOrderID}}
	if err := db.LockByKeys(transaction, &lockedOrders); err != nil {
		return types.OnlineSaleOrder{}, core.Err("Error al obtener el pedido web:", err)
	}
	if len(lockedOrders) == 0 {
		return types.OnlineSaleOrder{}, core.Err(fmt.Sprintf("No se encontró el pedido web N° %v.", onlineOrderID))
	}
	return lockedOrders[0], nil
}

// requireUnsettledOnlineSaleOrder refuses an order that already became a sale or is not pending.
func requireUnsettledOnlineSaleOrder(onlineOrder *types.OnlineSaleOrder) error {
	if onlineOrder.SaleOrderID > 0 {
		return core.Err(fmt.Sprintf("El pedido N° %v ya tiene la venta %v: continúe en Gestión Ventas.",
			onlineOrder.ID, onlineOrder.SaleOrderID))
	}
	if onlineOrder.Status != types.OrderStatusPending {
		return core.Err(fmt.Sprintf("El pedido N° %v no está pendiente.", onlineOrder.ID))
	}
	return nil
}

// settleOnlineSaleOrder creates the SaleOrder of an online order that was just paid and/or
// delivered, and links both, in one transaction. culqiChargeID is set by the checkout after an
// approved charge; Pedidos Web passes "".
//
// The order's settle lock is held from before the sale's number is minted to after the commit, so
// two settlements of one order (a double click, or Culqi and the back office at once) cannot both
// spend a comprobante number; the row is re-checked under its row lock inside the transaction.
func settleOnlineSaleOrder(req *core.HandlerArgs, settleRequest onlineSaleOrderSettleRequest,
	culqiChargeID string) (types.OnlineSaleOrder, types.SaleOrder, error) {

	companyID := req.User.CompanyID
	isPayment := slices.Contains(settleRequest.ActionsIncluded, 2)
	isDelivery := slices.Contains(settleRequest.ActionsIncluded, 3)
	if (!isPayment && !isDelivery) || slices.ContainsFunc(settleRequest.ActionsIncluded, func(action int8) bool {
		return action != 2 && action != 3
	}) {
		return types.OnlineSaleOrder{}, types.SaleOrder{}, core.Err("ActionsIncluded solo permite 2 (pago) y 3 (entrega).")
	}

	settleLock, err := lockOnlineSaleOrder(companyID, settleRequest.ID)
	if err != nil {
		return types.OnlineSaleOrder{}, types.SaleOrder{}, err
	}
	defer settleLock.Release()

	onlineOrders := []types.OnlineSaleOrder{}
	query := db.Query(&onlineOrders)
	query.Select().CompanyID.Equals(companyID).ID.Equals(settleRequest.ID)
	if err := query.Exec(); err != nil {
		return types.OnlineSaleOrder{}, types.SaleOrder{}, core.Err("Error al obtener el pedido web:", err)
	}
	if len(onlineOrders) == 0 {
		return types.OnlineSaleOrder{}, types.SaleOrder{}, core.Err(fmt.Sprintf("No se encontró el pedido web N° %v.", settleRequest.ID))
	}
	onlineOrder := onlineOrders[0]
	if err := requireUnsettledOnlineSaleOrder(&onlineOrder); err != nil {
		return onlineOrder, types.SaleOrder{}, err
	}
	companyConfig, err := loadWebStoreConfig(companyID)
	if err != nil {
		return onlineOrder, types.SaleOrder{}, err
	}

	// The sale bills what the shopper agreed to at checkout: the order's priced lines and shipping,
	// never today's catalog. It owes nothing after a payment, which collects the whole order.
	sale := types.SaleOrder{
		Lines: onlineOrder.Lines, TotalAmount: onlineOrder.TotalAmount, ShippingAmount: onlineOrder.ShippingAmount,
		DebtAmount: core.If(isPayment, int32(0), onlineOrder.TotalAmount), ExchangeRate: onlineOrder.ExchangeRate,
		CurrencyType: finance.CurrencyPEN, ClientID: onlineOrder.ClientID, OnlineSaleOrderID: onlineOrder.ID,
		ActionsIncluded: settleRequest.ActionsIncluded,
	}
	sale.TaxAmount = invoicing.SaleOrderTaxAmount(&sale)
	saleProducts, err := types.ReadSaleProducts(companyID, sale.Lines)
	if err != nil {
		return onlineOrder, sale, err
	}

	if isPayment {
		cashBank, err := finance.GetCaja(companyID, settleRequest.CashBankID)
		if err != nil {
			return onlineOrder, sale, core.Err("Error al obtener la caja del pago:", err)
		}
		if cashBank.Status == 0 {
			return onlineOrder, sale, core.Err("La caja seleccionada está inactiva.")
		}
		if types.NormalizeCurrency(cashBank.CurrencyType) != finance.CurrencyPEN {
			return onlineOrder, sale, core.Err(fmt.Sprintf(`El pedido está en soles pero la caja "%v" no.`, cashBank.Name))
		}
		sale.LastPaymentCajaID = settleRequest.CashBankID
	}
	// The order's units stay reserved on its PendingWarehouseID until the delivery. A sale that is
	// only paid takes that warehouse as its own, so its later delivery or annulment (PostSaleOrder,
	// PostSaleOrderAnnul) releases them from there.
	sale.WarehouseID = onlineOrder.PendingWarehouseID
	if isDelivery {
		if settleRequest.WarehouseID <= 0 {
			return onlineOrder, sale, core.Err("Se requiere el almacén de la entrega.")
		}
		sale.WarehouseID = settleRequest.WarehouseID
	}

	// With flag 12 the sale is the next boleta of the "Usar en Tienda Web" series and the boleta is
	// issued now; without it the sale names no comprobante (series 0).
	var receiptSeries *invoicing.InvoiceSeries
	if core.HasCompanyFlag(companyConfig.Company.Flags, types.CompanyFlagWebReceiptAutomatic) {
		if receiptSeries = invoicing.FindWebStoreSeries(companyConfig.Sunat.Series); receiptSeries == nil {
			return onlineOrder, sale, core.Err(`La tienda no tiene una serie de boleta marcada "Usar en Tienda Web".`)
		}
	}

	prepared, err := saleissue.PrepareNewSale(req, saleissue.NewSaleRequest{
		Sale: sale, Products: saleProducts, IssueSeries: receiptSeries,
		ReservedWarehouseID: onlineOrder.PendingWarehouseID, IsOnlineReservation: true,
	})
	if err != nil {
		return onlineOrder, sale, err
	}
	sale = prepared.Sale

	err = db.RunTransaction(func(transaction *db.Transaction) error {
		lockedOrder, err := lockOnlineSaleOrderRowIn(transaction, companyID, onlineOrder.ID)
		if err != nil {
			return err
		}
		if err := requireUnsettledOnlineSaleOrder(&lockedOrder); err != nil {
			return err
		}
		if err := saleissue.WriteNewSaleIn(transaction, req, &prepared); err != nil {
			return err
		}

		lockedOrder.SaleOrderID, lockedOrder.Status, lockedOrder.DebtAmount = sale.ID, sale.Status, sale.DebtAmount
		lockedOrder.CulqiChargeID = core.If(culqiChargeID != "", culqiChargeID, lockedOrder.CulqiChargeID)
		lockedOrder.Updated, lockedOrder.UpdatedBy = sale.Updated, req.User.ID
		onlineOrder = lockedOrder
		orderTable := db.TableOf[types.OnlineSaleOrder]()
		if err := db.UpdateIn(transaction, &[]types.OnlineSaleOrder{lockedOrder}, orderTable.SaleOrderID,
			orderTable.Status, orderTable.DebtAmount, orderTable.CulqiChargeID, orderTable.Updated, orderTable.UpdatedBy); err != nil {
			return core.Err("Error al actualizar el pedido web:", err)
		}
		return nil
	})
	if err != nil {
		return onlineOrder, sale, err
	}
	core.Log("settleOnlineSaleOrder:: pedido", onlineOrder.ID, "-> venta", sale.ID, "acciones", settleRequest.ActionsIncluded)
	saleissue.ScheduleNewSaleDocument(&prepared)
	return onlineOrder, sale, nil
}

// annulOnlineSaleOrder annuls an order that never became a sale: a declined charge, or one the store
// cancels. No number, document, money or stock movement exists to undo; only its reserved units
// are released.
func annulOnlineSaleOrder(companyID, onlineOrderID int32, reason string, userID int32) (types.OnlineSaleOrder, error) {
	settleLock, err := lockOnlineSaleOrder(companyID, onlineOrderID)
	if err != nil {
		return types.OnlineSaleOrder{}, err
	}
	defer settleLock.Release()

	var annulledOrder types.OnlineSaleOrder
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		lockedOrder, err := lockOnlineSaleOrderRowIn(transaction, companyID, onlineOrderID)
		if err != nil {
			return err
		}
		if err := requireUnsettledOnlineSaleOrder(&lockedOrder); err != nil {
			return err
		}
		lockedOrder.Status = types.OrderStatusAnnulled
		lockedOrder.AnnulReason = reason
		lockedOrder.Updated, lockedOrder.UpdatedBy = core.SUnixTime(), userID
		annulledOrder = lockedOrder
		orderTable := db.TableOf[types.OnlineSaleOrder]()
		if err := db.UpdateIn(transaction, &[]types.OnlineSaleOrder{lockedOrder}, orderTable.Status,
			orderTable.AnnulReason, orderTable.Updated, orderTable.UpdatedBy); err != nil {
			return core.Err("Error al anular el pedido web:", err)
		}
		return logistics.ApplyPendingDeliveryIn(transaction, companyID, userID,
			types.PendingDeliveryChanges(lockedOrder.Lines, lockedOrder.PendingWarehouseID, true, -1))
	})
	return annulledOrder, err
}

// syncOnlineSaleOrderIn copies a sale's state to the online order it was created from, in the
// sale's own transaction: the SaleOrder rules once it exists, so this is the only way the two
// statuses move after that. A till sale (no OnlineSaleOrderID) writes nothing.
func syncOnlineSaleOrderIn(transaction *db.Transaction, sale *types.SaleOrder) error {
	if sale.OnlineSaleOrderID == 0 {
		return nil
	}
	onlineOrder := types.OnlineSaleOrder{
		CompanyID: sale.CompanyID, ID: sale.OnlineSaleOrderID, Status: sale.Status, DebtAmount: sale.DebtAmount,
		AnnulReason: sale.AnnulReason, Updated: sale.Updated, UpdatedBy: sale.UpdatedBy,
	}
	orderTable := db.TableOf[types.OnlineSaleOrder]()
	if err := db.UpdateIn(transaction, &[]types.OnlineSaleOrder{onlineOrder}, orderTable.Status,
		orderTable.DebtAmount, orderTable.AnnulReason, orderTable.Updated, orderTable.UpdatedBy); err != nil {
		return core.Err("Error al actualizar el pedido web de la venta:", err)
	}
	return nil
}
