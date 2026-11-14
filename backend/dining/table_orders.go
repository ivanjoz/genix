package dining

import (
	"app/core"
	"app/db"
	"app/dining/types"
	finance "app/finance/types"
	logistics "app/logistics/types"
	production "app/production/types"
	sales "app/sales/types"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// GetDiningTableOrders is the delta of a layout's open orders (?layout-id): a charged or annulled
// order arrives as a removed ID.
func GetDiningTableOrders(req *core.HandlerArgs) core.HandlerResponse {
	layoutID := int32(req.GetQueryInt("layout-id"))
	if layoutID <= 0 {
		return req.MakeErr("Debe indicar el salón.")
	}
	orders := []types.TableOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(req.User.CompanyID).LayoutID.Equals(layoutID).
		Delta(req.GetDeltaCursor("records"), types.TableOrderOpen)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los pedidos de las mesas:", err)
	}
	return req.MakeResponse(map[string]any{"records": &orders})
}

type tableOrderRoundRequest struct {
	LayoutID     int32
	TableShapeID int16
	// Guests is kept when the round opens the order.
	Guests int8
	Lines  []tableOrderRoundLine
}

type tableOrderRoundLine struct {
	ProductID      int32
	PresentationID int16
	Quantity       int32 // packed (core.QuantityLineScale)
	Note           string
}

// PostDiningTableOrderRound adds a waiter's round to a table, opening its order when it has none
// (docs/DINING_PLAN.md §2): the lines are priced from the catalog in soles, routed to their station,
// checked against the web store's sellable rule on the order's warehouse, and their units reserved
// there in the same transaction as the order.
func PostDiningTableOrderRound(req *core.HandlerArgs) core.HandlerResponse {
	round := tableOrderRoundRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &round); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID, userID := req.User.CompanyID, req.User.ID
	if len(round.Lines) == 0 || len(round.Lines) > types.MaxRoundLines {
		return req.MakeErr(fmt.Sprintf("Un pedido lleva entre 1 y %v productos.", types.MaxRoundLines))
	}
	if round.Guests < 0 {
		return req.MakeErr("El número de comensales no es válido.")
	}
	productIDs := core.SliceSet[int32]{}
	for index := range round.Lines {
		line := &round.Lines[index]
		line.Note = strings.TrimSpace(line.Note)
		if err := types.ValidateRoundLine(line.ProductID, line.Quantity, line.Note); err != nil {
			return req.MakeErr(err)
		}
		productIDs.Add(line.ProductID)
	}

	layout, err := readActiveLayout(companyID, round.LayoutID)
	if err != nil {
		return req.MakeErr(err)
	}
	if !slices.ContainsFunc(layout.Shapes, func(shape types.LayoutShape) bool {
		return shape.ShapeID == round.TableShapeID && shape.Kind == types.ShapeTable
	}) {
		return req.MakeErr("La mesa no existe en el salón.")
	}

	tableLock, lockErr := lockDiningTable(companyID, round.LayoutID, round.TableShapeID)
	if lockErr != nil {
		return lockErr.Response(req)
	}
	defer tableLock.Release()

	openOrder, err := readOpenTableOrder(companyID, round.LayoutID, round.TableShapeID)
	if err != nil {
		return req.MakeErr(err)
	}
	// An open order keeps the warehouse it opened with, where its reservations are.
	warehouseID := layout.WarehouseID
	if openOrder != nil {
		warehouseID = openOrder.WarehouseID
	}

	products, err := readDiningProducts(companyID, productIDs.Values)
	if err != nil {
		return req.MakeErr(err)
	}
	productByID := map[int32]production.Product{}
	for _, product := range products {
		productByID[product.ID] = product
	}
	exchangeRate, err := resolveDiningExchangeRate(companyID, products)
	if err != nil {
		return req.MakeErr(err)
	}
	roundLines, err := priceRoundLines(round.Lines, productByID, layout.Stations, exchangeRate)
	if err != nil {
		return req.MakeErr(err)
	}
	if err := validateRoundStock(companyID, warehouseID, roundLines, productByID); err != nil {
		return req.MakeErr(err)
	}

	nowTime := core.SUnixTime()
	savedOrder := types.TableOrder{}
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		order := types.TableOrder{
			CompanyID: companyID, LayoutID: round.LayoutID, TableShapeID: round.TableShapeID, WarehouseID: warehouseID,
			Guests: round.Guests, WaiterID: userID, Status: types.TableOrderOpen, Created: nowTime,
		}
		if openOrder != nil {
			lockedOrders := []types.TableOrder{{CompanyID: companyID, ID: openOrder.ID}}
			if err := db.LockByKeys(transaction, &lockedOrders); err != nil {
				return core.Err("Error al obtener el pedido de la mesa:", err)
			}
			if len(lockedOrders) == 0 || lockedOrders[0].Status != types.TableOrderOpen {
				return core.Err("El pedido de la mesa acaba de cerrarse. Vuelva a enviar el pedido.")
			}
			order = lockedOrders[0]
		}
		types.AddRoundLines(&order, roundLines, false, nowTime, userID)
		order.ExchangeRate = core.Coalesce(exchangeRate, order.ExchangeRate)
		order.Updated, order.UpdatedBy = nowTime, userID

		orders := []types.TableOrder{order}
		if openOrder == nil {
			if err := db.InsertIn(transaction, &orders); err != nil {
				return core.Err("Error al abrir el pedido de la mesa:", err)
			}
		} else if err := updateTableOrderLinesIn(transaction, &orders, true); err != nil {
			return err
		}
		savedOrder = orders[0]
		// Only the round's lines reserve now: the earlier rounds already hold theirs.
		roundOnly := savedOrder.Lines[len(savedOrder.Lines)-len(roundLines):]
		return logistics.ApplyPendingDeliveryIn(transaction, companyID, userID,
			types.TableOrderReservationChanges(roundOnly, warehouseID, 1))
	})
	if err != nil {
		return req.MakeErr(err)
	}
	core.Log("PostDiningTableOrderRound:: mesa", round.TableShapeID, "salón", round.LayoutID, "pedido", savedOrder.ID,
		"líneas", len(roundLines))
	publishDiningLayoutChange(companyID, savedOrder.LayoutID, savedOrder.Updated)
	return req.MakeResponse(savedOrder)
}

type tableOrderLinesRequest struct {
	OrderID int32
	LineIDs []int16
	// Status is where the lines go (§2.1): 2 confirm or take back a "ready", 3 ready, 4 served,
	// 0 cancel.
	Status int8
}

// PostDiningTableOrderLines moves lines of an open order to a new status (§2.1). A cancelled line
// releases its reservation in the same transaction.
func PostDiningTableOrderLines(req *core.HandlerArgs) core.HandlerResponse {
	request := tableOrderLinesRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if request.OrderID <= 0 || len(request.LineIDs) == 0 {
		return req.MakeErr("Debe indicar el pedido y sus productos.")
	}
	companyID, userID := req.User.CompanyID, req.User.ID
	nowTime := core.SUnixTime()

	savedOrder := types.TableOrder{}
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		orders := []types.TableOrder{{CompanyID: companyID, ID: request.OrderID}}
		if err := db.LockByKeys(transaction, &orders); err != nil {
			return core.Err("Error al obtener el pedido de la mesa:", err)
		}
		if len(orders) == 0 || orders[0].Status != types.TableOrderOpen {
			return core.Err("El pedido de la mesa no existe o ya se cerró.")
		}
		order := &orders[0]
		// The lines as they were, of those this request cancels: what their reservation releases.
		cancelledLines := []types.TableOrderLine{}
		for _, lineID := range request.LineIDs {
			lineIndex := slices.IndexFunc(order.Lines, func(line types.TableOrderLine) bool { return line.LineID == lineID })
			if lineIndex < 0 {
				return core.Err(fmt.Sprintf("El pedido no tiene el producto %v.", lineID))
			}
			lineBefore := order.Lines[lineIndex]
			if err := types.ApplyLineTransition(&order.Lines[lineIndex], request.Status, nowTime, userID); err != nil {
				return err
			}
			if request.Status == types.LineStatusCancelled {
				cancelledLines = append(cancelledLines, lineBefore)
			}
		}
		order.Updated, order.UpdatedBy = nowTime, userID
		if err := updateTableOrderLinesIn(transaction, &orders, false); err != nil {
			return err
		}
		savedOrder = orders[0]
		return logistics.ApplyPendingDeliveryIn(transaction, companyID, userID,
			types.TableOrderReservationChanges(cancelledLines, order.WarehouseID, -1))
	})
	if err != nil {
		return req.MakeErr(err)
	}
	publishDiningLayoutChange(companyID, savedOrder.LayoutID, savedOrder.Updated)
	return req.MakeResponse(savedOrder)
}

// lockDiningTable takes the table's lock: one writer per table, so two devices ordering at once
// cannot each open an order for it, and a round cannot land on a bill being charged.
func lockDiningTable(companyID int32, layoutID int32, tableShapeID int16) (core.Lock, *core.LockError) {
	return core.AcquireLock(context.Background(), core.ActionDiningTable,
		int64(companyID)<<32|int64(uint16(layoutID))<<16|int64(uint16(tableShapeID)), 4)
}

// updateTableOrderLinesIn writes an order's lines; a round also writes the rate it converted with.
// LayoutID and Status go too: they are the delta index's keys, which the write needs with the row.
func updateTableOrderLinesIn(transaction *db.Transaction, orders *[]types.TableOrder, withExchangeRate bool) error {
	orderTable := db.TableOf[types.TableOrder]()
	columns := []db.Coln{orderTable.Lines, orderTable.Updated, orderTable.UpdatedBy, orderTable.LayoutID, orderTable.Status}
	if withExchangeRate {
		columns = append(columns, orderTable.ExchangeRate)
	}
	if err := db.UpdateIn(transaction, orders, columns...); err != nil {
		return core.Err("Error al guardar el pedido de la mesa:", err)
	}
	return nil
}

// readOpenTableOrder is the table's open order, nil when it has none.
func readOpenTableOrder(companyID int32, layoutID int32, tableShapeID int16) (*types.TableOrder, error) {
	orders, err := readOpenTableOrders(companyID, layoutID)
	if err != nil {
		return nil, err
	}
	for index := range orders {
		if orders[index].TableShapeID == tableShapeID {
			return &orders[index], nil
		}
	}
	return nil, nil
}

// readOpenTableOrders are the layout's open orders.
func readOpenTableOrders(companyID int32, layoutID int32) ([]types.TableOrder, error) {
	orders := []types.TableOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(companyID).LayoutID.Equals(layoutID).Status.Equals(types.TableOrderOpen)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los pedidos abiertos del salón:", err)
	}
	return orders, nil
}

// priceRoundLines turns the requested lines into order lines: priced in soles by the till's rule,
// on a presentation the web store sells, routed to their station.
func priceRoundLines(requestLines []tableOrderRoundLine, productByID map[int32]production.Product,
	stations []types.DiningStation, exchangeRate int32) ([]types.TableOrderLine, error) {
	roundLines := make([]types.TableOrderLine, 0, len(requestLines))
	for _, requestLine := range requestLines {
		product, found := productByID[requestLine.ProductID]
		if !found {
			return nil, core.Err(fmt.Sprintf("El producto %v no existe o no está a la venta.", requestLine.ProductID))
		}
		presentationIDs := product.WebPresentationIDs()
		if !slices.Contains(presentationIDs, requestLine.PresentationID) {
			return nil, core.Err(fmt.Sprintf(`La presentación elegida de "%v" no está a la venta.`, product.Name))
		}
		pricedLine := sales.SaleOrderLine{ProductID: product.ID, Quantity: requestLine.Quantity}
		if err := sales.PriceSaleOrderLine(&pricedLine, product, finance.CurrencyPEN, exchangeRate); err != nil {
			return nil, err
		}
		// Live (ordered) from here: the stock check counts it. AddRoundLines settles the status.
		roundLines = append(roundLines, types.TableOrderLine{
			Status:    types.LineStatusOrdered,
			ProductID: product.ID, PresentationID: requestLine.PresentationID, Quantity: requestLine.Quantity,
			SubDivisor: pricedLine.SubDivisor, Price: pricedLine.Price, SubPrice: pricedLine.SubPrice,
			StationID: types.StationForProduct(stations, product.ID, product.CategoryIDs), Note: requestLine.Note,
		})
	}
	return roundLines, nil
}

// validateRoundStock checks the round against the web store's sellable rule on the order's
// warehouse alone (§3): what it reserves, added to what is already oversold, must stay within each
// product's margin ("Sobreventa permitida"). A made product has no stock, so the margin is its menu.
func validateRoundStock(companyID int32, warehouseID int32, roundLines []types.TableOrderLine, productByID map[int32]production.Product) error {
	orderedUnitsByProduct := map[int32]map[int16]int32{}
	presentationIDsByProduct := map[int32][]int16{}
	for _, change := range types.TableOrderReservationChanges(roundLines, warehouseID, 1) {
		if orderedUnitsByProduct[change.ProductID] == nil {
			orderedUnitsByProduct[change.ProductID] = map[int16]int32{}
			product := productByID[change.ProductID]
			presentationIDsByProduct[change.ProductID] = product.WebPresentationIDs()
		}
		orderedUnitsByProduct[change.ProductID][change.PresentationID] += change.Store
	}
	stockRowsByProduct, err := logistics.ReadWebStockRows(companyID, []int32{warehouseID}, presentationIDsByProduct)
	if err != nil {
		return err
	}
	for productID, orderedUnits := range orderedUnitsByProduct {
		product := productByID[productID]
		stockRows, presentationIDs := stockRowsByProduct[productID], presentationIDsByProduct[productID]
		if logistics.WebOrderFits(stockRows, presentationIDs, product.WebExtraStock, orderedUnits) {
			continue
		}
		// One presentation ordered: say how many are left. Several share the margin, so no single
		// figure is right for each.
		if len(orderedUnits) == 1 {
			sellableUnits := logistics.WebSellableStock(stockRows, presentationIDs, product.WebExtraStock)
			for presentationID := range orderedUnits {
				return core.Err(fmt.Sprintf(`No hay stock suficiente de "%v": quedan %v.`, product.Name, sellableUnits[presentationID]))
			}
		}
		return core.Err(fmt.Sprintf(`No hay stock suficiente de "%v" para las presentaciones elegidas.`, product.Name))
	}
	return nil
}
