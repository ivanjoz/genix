package dining

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/dining/types"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	logistics "app/logistics/types"
	saleissue "app/sales/saleissue/types"
	sales "app/sales/types"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type tableOrderChargeRequest struct {
	OrderID       int32
	IssueSeriesID int8
	// ClientID names an existing client; ClientInfo registers one, as at the till.
	ClientID   int32
	ClientInfo *sales.SaleOrderClientInfo
	// CashBankID collects the whole bill; 0 leaves it owed by the client.
	CashBankID int32
}

// PostDiningTableOrderCharge closes a table's bill into a SaleOrder (docs/DINING_PLAN.md §4), through
// the path every sale is created by (saleissue.PrepareNewSale). The sale bills the served lines at
// the prices they were ordered at, and is delivered: the diners already ate. Its bought lines leave
// the order's warehouse now and release their reservation; its made lines keep theirs until the
// daily production close takes their stock.
func PostDiningTableOrderCharge(req *core.HandlerArgs) core.HandlerResponse {
	request := tableOrderChargeRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if !req.User.HasSubAcceso(types.AccessIDTableService, types.SubAccessIDChargeTable) {
		return req.MakeErr("No tiene permiso para cobrar mesas.")
	}
	companyID := req.User.CompanyID

	order, tableLock, err := lockOpenTableOrder(companyID, request.OrderID)
	if lockErr, isLockErr := err.(*core.LockError); isLockErr {
		return lockErr.Response(req)
	}
	if err != nil {
		return req.MakeErr(err)
	}
	defer tableLock.Release()

	saleLines, err := types.ChargeSaleLines(order.Lines)
	if err != nil {
		return req.MakeErr(err)
	}
	// A table order is always in soles; the sale inherits the rate its rounds converted with.
	sale := sales.SaleOrder{
		Lines: saleLines, CurrencyType: finance.CurrencyPEN, ExchangeRate: order.ExchangeRate,
		WarehouseID: order.WarehouseID, IssueSeriesID: request.IssueSeriesID,
		ClientID: request.ClientID, ClientInfo: request.ClientInfo,
		ActionsIncluded: []int8{3}, LastPaymentCajaID: request.CashBankID,
	}
	if request.CashBankID > 0 {
		sale.ActionsIncluded = []int8{2, 3}
	}
	for _, line := range sale.Lines {
		sale.TotalAmount += core.QuantityAmount(core.UnpackQuantityLine(line.Quantity), line.Price, line.SubPrice)
	}
	sale.TaxAmount = invoicing.SaleOrderTaxAmount(&sale)
	// Checks the cash-bank collects in soles.
	if err := sales.ValidateSaleCurrency(companyID, &sale); err != nil {
		return req.MakeErr(err)
	}

	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return req.MakeErr("Error al leer la configuración de la empresa:", err)
	}
	issueSeries, err := saleissue.ResolveSaleIssueSeries(companyID, &sale, companyConfig.Sunat.Series)
	if err != nil {
		return req.MakeErr(err)
	}
	saleProducts, err := sales.ReadSaleProducts(companyID, sale.Lines)
	if err != nil {
		return req.MakeErr(err)
	}
	// The order's lines are reserved on its warehouse, which the sale delivers from.
	prepared, err := saleissue.PrepareNewSale(req, saleissue.NewSaleRequest{
		Sale: sale, Products: saleProducts, IssueSeries: issueSeries, ReservedWarehouseID: order.WarehouseID,
	})
	if err != nil {
		return req.MakeErr(err)
	}

	err = db.RunTransaction(func(transaction *db.Transaction) error {
		// The table lock keeps rounds out and served or cancelled lines never move again, but a
		// writer that skips the table lock must not slip a line past the bill the sale was built from.
		lockedOrder, err := lockTableOrderRowIn(transaction, companyID, order.ID)
		if err != nil {
			return err
		}
		if !slices.Equal(lockedOrder.Lines, order.Lines) {
			return core.Err("El pedido de la mesa cambió mientras se cobraba. Vuelva a cobrarlo.")
		}
		if err := saleissue.WriteNewSaleIn(transaction, req, &prepared); err != nil {
			return err
		}
		order.Status, order.SaleOrderID = types.TableOrderCharged, prepared.Sale.ID
		order.Updated, order.UpdatedBy = prepared.Sale.Updated, req.User.ID
		orderTable := db.TableOf[types.TableOrder]()
		if err := db.UpdateIn(transaction, &[]types.TableOrder{order}, orderTable.Status, orderTable.SaleOrderID,
			orderTable.Updated, orderTable.UpdatedBy, orderTable.LayoutID); err != nil {
			return core.Err("Error al cerrar el pedido de la mesa:", err)
		}
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	saleissue.ScheduleNewSaleDocument(&prepared)
	core.Log("PostDiningTableOrderCharge:: pedido", order.ID, "-> venta", prepared.Sale.ID)
	publishDiningLayoutChange(companyID, order.LayoutID, order.Updated)
	return req.MakeResponse(map[string]any{"Order": order, "SaleOrder": prepared.Sale})
}

// PostDiningTableOrderAnnul annuls an open order (§4): every line's reservation is released.
// Nothing was sold, so nothing else moves.
func PostDiningTableOrderAnnul(req *core.HandlerArgs) core.HandlerResponse {
	request := struct {
		OrderID int32
		Reason  string
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		return req.MakeErr("Se requiere el motivo de la anulación.")
	}
	if len(reason) > sales.MaxAnnulReasonLength {
		return req.MakeErr("El motivo de la anulación no puede superar los", sales.MaxAnnulReasonLength, "caracteres.")
	}
	if !req.User.HasSubAcceso(types.AccessIDTableService, types.SubAccessIDChargeTable) {
		return req.MakeErr("No tiene permiso para anular pedidos de mesa.")
	}
	companyID, userID := req.User.CompanyID, req.User.ID

	order, tableLock, err := lockOpenTableOrder(companyID, request.OrderID)
	if lockErr, isLockErr := err.(*core.LockError); isLockErr {
		return lockErr.Response(req)
	}
	if err != nil {
		return req.MakeErr(err)
	}
	defer tableLock.Release()

	err = db.RunTransaction(func(transaction *db.Transaction) error {
		lockedOrder, err := lockTableOrderRowIn(transaction, companyID, order.ID)
		if err != nil {
			return err
		}
		// The lines as they are now: a line cancelled meanwhile already released its units.
		order = lockedOrder
		order.Status = types.TableOrderAnnulled
		order.AnnulReason = reason
		order.Updated, order.UpdatedBy = core.SUnixTime(), userID
		orderTable := db.TableOf[types.TableOrder]()
		if err := db.UpdateIn(transaction, &[]types.TableOrder{order}, orderTable.Status, orderTable.AnnulReason,
			orderTable.Updated, orderTable.UpdatedBy, orderTable.LayoutID); err != nil {
			return core.Err("Error al anular el pedido de la mesa:", err)
		}
		return logistics.ApplyPendingDeliveryIn(transaction, companyID, userID,
			types.TableOrderReservationChanges(order.Lines, order.WarehouseID, -1))
	})
	if err != nil {
		return req.MakeErr(err)
	}
	core.Log("PostDiningTableOrderAnnul:: pedido", order.ID, "anulado")
	publishDiningLayoutChange(companyID, order.LayoutID, order.Updated)
	return req.MakeResponse(order)
}

// lockOpenTableOrder reads an open order and takes its table's lock, then reads it again under the
// lock: the first read only names the table to lock. A lock failure is returned as *core.LockError.
func lockOpenTableOrder(companyID int32, orderID int32) (types.TableOrder, core.Lock, error) {
	order, err := readOpenTableOrderByID(companyID, orderID)
	if err != nil {
		return order, nil, err
	}
	tableLock, lockErr := lockDiningTable(companyID, order.LayoutID, order.TableShapeID)
	if lockErr != nil {
		return order, nil, lockErr
	}
	if order, err = readOpenTableOrderByID(companyID, orderID); err != nil {
		tableLock.Release()
		return order, nil, err
	}
	return order, tableLock, nil
}

// readOpenTableOrderByID is an order that is still open, or an error saying it is not.
func readOpenTableOrderByID(companyID int32, orderID int32) (types.TableOrder, error) {
	orders := []types.TableOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(companyID).ID.Equals(orderID)
	if err := query.Exec(); err != nil {
		return types.TableOrder{}, core.Err("Error al obtener el pedido de la mesa:", err)
	}
	if len(orders) == 0 || orders[0].Status != types.TableOrderOpen {
		return types.TableOrder{}, core.Err(fmt.Sprintf("El pedido de mesa N° %v no existe o ya se cerró.", orderID))
	}
	return orders[0], nil
}

// lockTableOrderRowIn reads an order under its row lock, inside the caller's transaction, and
// refuses one that is no longer open.
func lockTableOrderRowIn(transaction *db.Transaction, companyID int32, orderID int32) (types.TableOrder, error) {
	orders := []types.TableOrder{{CompanyID: companyID, ID: orderID}}
	if err := db.LockByKeys(transaction, &orders); err != nil {
		return types.TableOrder{}, core.Err("Error al obtener el pedido de la mesa:", err)
	}
	if len(orders) == 0 || orders[0].Status != types.TableOrderOpen {
		return types.TableOrder{}, core.Err("El pedido de la mesa no existe o ya se cerró.")
	}
	return orders[0], nil
}
