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
	// never today's catalog.
	nowTime, today := core.SUnixTime(), core.FechaUnix()
	sale := types.SaleOrder{
		CompanyID: companyID, Lines: onlineOrder.Lines, TotalAmount: onlineOrder.TotalAmount,
		DebtAmount: onlineOrder.TotalAmount, ShippingAmount: onlineOrder.ShippingAmount,
		ExchangeRate: onlineOrder.ExchangeRate, CurrencyType: finance.CurrencyPEN, ClientID: onlineOrder.ClientID,
		OnlineSaleOrderID: onlineOrder.ID, ActionsIncluded: settleRequest.ActionsIncluded,
		Date: today, Created: nowTime, Updated: nowTime, UpdatedBy: req.User.ID, Status: types.OrderStatusPending,
	}
	sale.TaxAmount = invoicing.SaleOrderTaxAmount(&sale)
	saleProducts, err := readSaleProducts(companyID, sale.Lines)
	if err != nil {
		return onlineOrder, sale, err
	}

	collectedAmount := int32(0)
	if isPayment {
		cashBank, err := finance.GetCaja(companyID, settleRequest.CashBankID)
		if err != nil {
			return onlineOrder, sale, core.Err("Error al obtener la caja del pago:", err)
		}
		if cashBank.Status == 0 {
			return onlineOrder, sale, core.Err("La caja seleccionada está inactiva.")
		}
		if normalizeCurrency(cashBank.CurrencyType) != finance.CurrencyPEN {
			return onlineOrder, sale, core.Err(fmt.Sprintf(`El pedido está en soles pero la caja "%v" no.`, cashBank.Name))
		}
		collectedAmount, sale.DebtAmount = sale.DebtAmount, 0
		if err := sale.AddStatus(types.OrderStatusPaid); err != nil {
			return onlineOrder, sale, err
		}
		sale.LastPaymentCajaID = settleRequest.CashBankID
		sale.LastPaymentTime, sale.LastPaymentUser = nowTime, req.User.ID
	}
	isSameDayCollection := isPayment && finance.IsSameDaySettlement(sale.Date, sale.TotalAmount, 0, today, collectedAmount)
	if isSameDayCollection {
		sale.SameDayCashBankID = sale.LastPaymentCajaID
	}
	if isDelivery {
		if settleRequest.WarehouseID <= 0 {
			return onlineOrder, sale, core.Err("Se requiere el almacén de la entrega.")
		}
		sale.WarehouseID = settleRequest.WarehouseID
		// An early check before the number is minted; the movements re-check under their row locks.
		if err := validateSaleStock(req, sale); err != nil {
			return onlineOrder, sale, err
		}
		if err := sale.AddStatus(types.OrderStatusDelivered); err != nil {
			return onlineOrder, sale, err
		}
		sale.DeliveryTime, sale.DeliveryUser = nowTime, req.User.ID
	}
	sale.StampPeriod()

	// Mint the sale's id. With flag 12 it is the next boleta of the "Usar en Tienda Web" series and the
	// boleta is issued now; without it the sale names no comprobante (series 0).
	var document *invoicing.InvoiceDocument
	if core.HasCompanyFlag(companyConfig.Company.Flags, types.CompanyFlagWebReceiptAutomatic) {
		receiptSeries := invoicing.FindWebStoreSeries(companyConfig.Sunat.Series)
		if receiptSeries == nil {
			return onlineOrder, sale, core.Err(`La tienda no tiene una serie de boleta marcada "Usar en Tienda Web".`)
		}
		if document, err = invoicing.PrepareDocumentForSale(companyID, req.User.ID, &sale, receiptSeries, saleProducts); err != nil {
			return onlineOrder, sale, core.Err("No se pudo generar la boleta del pedido:", err)
		}
	} else if sale.ID, err = types.MakeSaleOrderID(companyID, 0); err != nil {
		return onlineOrder, sale, core.Err("Error al obtener el ID de la venta:", err)
	}

	cashMovements := []finance.InternalCashMovement{}
	if collectedAmount > 0 {
		cashMovements = append(cashMovements, finance.InternalCashMovement{
			CashBankID: sale.LastPaymentCajaID, DocumentID: sale.ID, Date: today,
			Type: finance.CashMovementTypeSaleCollection, CounterAccountCode: finance.PCGEReceivable,
			Amount: collectedAmount, SameDayMovement: int8(core.If(isSameDayCollection, 1, 0)),
		})
	}
	stockMovements := []logistics.InternalMovement{}
	if isDelivery {
		stockMovements = saleDeliveryMovements(&sale)
	}

	err = db.RunTransaction(func(transaction *db.Transaction) error {
		lockedOrder, err := lockOnlineSaleOrderRowIn(transaction, companyID, onlineOrder.ID)
		if err != nil {
			return err
		}
		if err := requireUnsettledOnlineSaleOrder(&lockedOrder); err != nil {
			return err
		}
		if err := db.InsertIn(transaction, &[]types.SaleOrder{sale}); err != nil {
			return core.Err("Error al registrar la venta del pedido:", err)
		}
		if document != nil {
			if err := invoicing.SaveNewDocument(transaction, document); err != nil {
				return core.Err("No se pudo guardar la boleta del pedido:", err)
			}
		}
		if err := finance.ApplyCashBankMovementIn(transaction, req, cashMovements); err != nil {
			return core.Err("Error al registrar el cobro:", err)
		}
		if len(stockMovements) > 0 {
			if err := logistics.ApplyMovimientosIn(transaction, req, stockMovements); err != nil {
				return core.Err("Error al procesar la salida de almacén:", err)
			}
		}
		if err := applyClientWeekSaleChangesIn(transaction, companyID, types.ClientWeekSaleChanges(nil, &sale)); err != nil {
			return err
		}

		lockedOrder.SaleOrderID, lockedOrder.Status, lockedOrder.DebtAmount = sale.ID, sale.Status, sale.DebtAmount
		lockedOrder.CulqiChargeID = core.If(culqiChargeID != "", culqiChargeID, lockedOrder.CulqiChargeID)
		lockedOrder.Updated, lockedOrder.UpdatedBy = nowTime, req.User.ID
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
	if document != nil {
		invoicing.ScheduleEmitPendingSweep(companyID)
	}
	return onlineOrder, sale, nil
}

// annulOnlineSaleOrder annuls an order that never became a sale: a declined charge, or one the store
// cancels. Nothing else exists to undo — no number, no document, no money, no stock.
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
		lockedOrder.AnnulReason = core.If(len(reason) > types.MaxAnnulReasonLength, reason[:types.MaxAnnulReasonLength], reason)
		lockedOrder.Updated, lockedOrder.UpdatedBy = core.SUnixTime(), userID
		annulledOrder = lockedOrder
		orderTable := db.TableOf[types.OnlineSaleOrder]()
		if err := db.UpdateIn(transaction, &[]types.OnlineSaleOrder{lockedOrder}, orderTable.Status,
			orderTable.AnnulReason, orderTable.Updated, orderTable.UpdatedBy); err != nil {
			return core.Err("Error al anular el pedido web:", err)
		}
		return nil
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
