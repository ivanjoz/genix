package production

import (
	business "app/business/types"
	"app/core"
	"app/db"
	logistics "app/logistics/types"
	"app/production/types"
	"encoding/json"
	"fmt"
	"github.com/ivanjoz/genix-orm/db/cache"
	"maps"
	"slices"
	"strings"
)

// Production orders (docs/MANUFACTURING_PLAN.md §1, §3–§6): a draft plans products per presentation,
// the release freezes the recipes on it and reserves the supplies, each receipt consumes supplies and
// enters products, and completion or annulment releases what is still reserved.

// GetProductionOrders is the delta of the company's production orders, the daily closes' included
// (Kind 2, born finished): the orders page keeps Kind 1.
func GetProductionOrders(req *core.HandlerArgs) core.HandlerResponse {
	orders := []types.ProductionOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("records"),
		types.ProductionOrderStatusDraft, types.ProductionOrderStatusReleased, types.ProductionOrderStatusInProgress,
		types.ProductionOrderStatusFinished)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las órdenes de producción:", err)
	}
	return req.MakeResponse(map[string]any{"records": &orders})
}

// GetProductionOrder is one production order of either kind: the lot traceability of the stock screen
// reads the receipts of the orders a lot came from or went into (docs/MANUFACTURING_PLAN.md §10).
func GetProductionOrder(req *core.HandlerArgs) core.HandlerResponse {
	orderID := int32(req.GetQueryInt("id"))
	if orderID <= 0 {
		return req.MakeErr("Se requiere la orden de producción.")
	}
	orders := []types.ProductionOrder{}
	query := db.Query(&orders)
	if err := query.CompanyID.Equals(req.User.CompanyID).ID.Equals(orderID).Exec(); err != nil {
		return req.MakeErr("Error al obtener la orden de producción:", err)
	}
	if len(orders) == 0 {
		return req.MakeErr(fmt.Sprintf("La orden de producción %v no existe.", orderID))
	}
	return req.MakeResponse(orders[0])
}

// lockProductionOrderIn locks the company's manual order and answers it, refused unless its status is
// one of allowedStatuses.
func lockProductionOrderIn(transaction *db.Transaction, companyID int32, orderID int32, allowedStatuses ...int8) (*types.ProductionOrder, error) {
	orders := []types.ProductionOrder{{CompanyID: companyID, ID: orderID}}
	if err := db.LockByKeys(transaction, &orders); err != nil {
		return nil, core.Err("Error al obtener la orden de producción:", err)
	}
	if len(orders) == 0 || orders[0].Kind != types.ProductionOrderKindManual {
		return nil, core.Err(fmt.Sprintf("La orden de producción %v no existe.", orderID))
	}
	if !slices.Contains(allowedStatuses, orders[0].Status) {
		return nil, core.Err(fmt.Sprintf("La orden de producción %v no admite esta acción en su estado actual.", orderID))
	}
	return &orders[0], nil
}

// readOrderProducts reads the made products of the lines with what planning and receiving need.
func readOrderProducts(companyID int32, lines []types.ProductionOrderLine) (map[int32]types.Product, error) {
	productIDs := []int32{}
	for _, line := range lines {
		productIDs = append(productIDs, line.ProductID)
	}
	return readProductsByID(companyID, productIDs, func(query *types.ProductTable) []db.Coln {
		return db.Cols(query.ID, query.Name, query.Status, query.Origin, query.SbuQuantity, query.Presentations,
			query.ProductionYieldPortions, query.ProductionSupplies, query.ProductionExtraCostPerUnit,
			query.ProductionStandardCost, query.ProductionLabor, query.TracksLots, query.ShelfLifeDays, query.QualityHoldOnEntry)
	})
}

// validateCompanyWarehouses refuses a warehouse that is not an active one of the company.
func validateCompanyWarehouses(companyID int32, warehouseIDs ...int32) error {
	warehouses := []business.Warehouse{}
	query := db.Query(&warehouses)
	query.Select(query.ID, query.Status).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los almacenes:", err)
	}
	for _, warehouseID := range warehouseIDs {
		if !slices.ContainsFunc(warehouses, func(warehouse business.Warehouse) bool {
			return warehouse.ID == warehouseID && warehouse.Status > 0
		}) {
			return core.Err(fmt.Sprintf("El almacén %v no existe o no está activo.", warehouseID))
		}
	}
	return nil
}

// PostProductionOrder creates or edits a draft order (§4.2): its products per presentation, its
// warehouses, dates and supply adjustment. Only a draft is editable; the release freezes it.
func PostProductionOrder(req *core.HandlerArgs) core.HandlerResponse {
	order := types.ProductionOrder{}
	if err := json.Unmarshal([]byte(*req.Body), &order); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	if order.WarehouseID <= 0 || order.SuppliesWarehouseID <= 0 {
		return req.MakeErr("Se requieren el almacén de ingreso y el almacén de insumos.")
	}
	if err := validateCompanyWarehouses(companyID, order.WarehouseID, order.SuppliesWarehouseID); err != nil {
		return req.MakeErr(err)
	}
	// +10000 doubles the supplies; −10000 would plan none.
	if order.SupplyAdjustmentBasisPoints <= -int16(types.BasisPointsWhole) || int64(order.SupplyAdjustmentBasisPoints) > types.BasisPointsWhole {
		return req.MakeErr("El ajuste de insumos debe estar entre -99.99% y +100%.")
	}
	if order.PlannedStart > 0 && order.PlannedEnd > 0 && order.PlannedEnd < order.PlannedStart {
		return req.MakeErr("La fecha de fin planificada es anterior a la de inicio.")
	}
	if order.Series < 0 {
		return req.MakeErr("Las series no pueden ser negativas.")
	}
	if order.SubcontractorID > 0 {
		// The workshop must have its warehouse: the order sends material there and consumes it there.
		if _, err := readWorkshopWarehouseID(companyID, order.SubcontractorID); err != nil {
			return req.MakeErr(err)
		}
		if order.SubcontractPricePerUnit < 0 {
			return req.MakeErr("El precio por unidad del taller no puede ser negativo.")
		}
	} else {
		order.SubcontractPricePerUnit = 0
	}
	productsByID, err := readOrderProducts(companyID, order.Lines)
	if err != nil {
		return req.MakeErr(err)
	}
	if err := types.ValidateProductionOrderLines(order.Lines, productsByID); err != nil {
		return req.MakeErr(err)
	}
	// What only receipts and labor entries write starts empty.
	for index := range order.Lines {
		line := &order.Lines[index]
		line.ProducedUnits, line.ScrappedUnits, line.Value, line.CostSource, line.ShortSupplies = 0, 0, 0, 0, false
	}
	order.Notes = strings.TrimSpace(order.Notes)
	nowTime := core.SUnixTime()

	err = db.RunTransaction(func(transaction *db.Transaction) error {
		if order.ID <= 0 {
			order.ID, order.CompanyID, order.Kind = 0, companyID, types.ProductionOrderKindManual
			order.Status, order.Date = types.ProductionOrderStatusDraft, core.FechaUnix()
			order.Supplies, order.Labor, order.Receipts, order.SaleOrderIDs, order.LaborRecordedAmount = nil, nil, nil, nil, 0
			order.Created, order.CreatedBy, order.Updated, order.UpdatedBy = nowTime, req.User.ID, nowTime, req.User.ID
			orders := []types.ProductionOrder{order}
			if err := db.InsertIn(transaction, &orders); err != nil {
				return core.Err("Error al crear la orden de producción:", err)
			}
			order = orders[0]
			stampStoredVersion(&order)
			return nil
		}
		stored, err := lockProductionOrderIn(transaction, companyID, order.ID, types.ProductionOrderStatusDraft)
		if err != nil {
			return err
		}
		stored.TemplateID, stored.Series, stored.WarehouseID, stored.SuppliesWarehouseID = order.TemplateID, order.Series,
			order.WarehouseID, order.SuppliesWarehouseID
		stored.PlannedStart, stored.PlannedEnd, stored.StageID = order.PlannedStart, order.PlannedEnd, order.StageID
		stored.SubcontractorID, stored.SubcontractPricePerUnit = order.SubcontractorID, order.SubcontractPricePerUnit
		stored.SupplyAdjustmentBasisPoints, stored.Lines, stored.Notes = order.SupplyAdjustmentBasisPoints, order.Lines, order.Notes
		stored.Updated, stored.UpdatedBy = nowTime, req.User.ID
		orders := []types.ProductionOrder{*stored}
		table := db.TableOf[types.ProductionOrder]()
		if err := db.UpdateIn(transaction, &orders, table.TemplateID, table.Series, table.WarehouseID, table.SuppliesWarehouseID,
			table.PlannedStart, table.PlannedEnd, table.StageID, table.SupplyAdjustmentBasisPoints, table.Lines, table.Notes,
			table.SubcontractorID, table.SubcontractPricePerUnit,
			table.Status, table.Updated, table.UpdatedBy); err != nil {
			return core.Err("Error al guardar la orden de producción:", err)
		}
		order = orders[0]
		stampStoredVersion(&order)
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(order)
}

type productionOrderActionRequest struct {
	OrderID int32
	StageID int32
}

// PostProductionOrderRelease releases a draft (§4.2, M17): the recipes are exploded and frozen on the
// order and its supplies reserved. It is refused when any supply is short: the user buys first, or
// plans less.
func PostProductionOrderRelease(req *core.HandlerArgs) core.HandlerResponse {
	body := productionOrderActionRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	var releasedOrder types.ProductionOrder

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		order, err := lockProductionOrderIn(transaction, companyID, body.OrderID, types.ProductionOrderStatusDraft)
		if err != nil {
			return err
		}
		productsByID, err := readOrderProducts(companyID, order.Lines)
		if err != nil {
			return err
		}
		if err := types.ValidateProductionOrderLines(order.Lines, productsByID); err != nil {
			return err
		}
		supplyIDs := []int32{}
		for _, product := range productsByID {
			for _, recipeSupply := range product.ProductionSupplies {
				supplyIDs = append(supplyIDs, recipeSupply.SupplyID)
			}
		}
		suppliesByID, err := readProductsByID(companyID, supplyIDs, func(query *types.ProductTable) []db.Coln {
			return db.Cols(query.ID, query.Name, query.Status, query.SbuQuantity, query.SbuUnit)
		})
		if err != nil {
			return err
		}
		costPerHourByLaborTypeID, err := readLaborTypeRates(companyID)
		if err != nil {
			return err
		}
		orderSupplies, orderLabor, err := types.ExplodeProductionOrder(order.Lines, order.SupplyAdjustmentBasisPoints,
			productsByID, suppliesByID, costPerHourByLaborTypeID)
		if err != nil {
			return err
		}

		plannedSupplyIDs := []int32{}
		for _, orderSupply := range orderSupplies {
			plannedSupplyIDs = append(plannedSupplyIDs, orderSupply.SupplyID)
		}
		supplyStockByID, err := readSupplyStock(companyID, order.SuppliesWarehouseID, plannedSupplyIDs)
		if err != nil {
			return err
		}
		shortages := []string{}
		reservationChanges := []logistics.ProductionReservationChange{}
		for index := range orderSupplies {
			orderSupply := &orderSupplies[index]
			supply := suppliesByID[orderSupply.SupplyID]
			atDivisor, err := types.ReadSupplyStockAtDivisor(supplyStockByID[orderSupply.SupplyID], orderSupply.SubDivisor)
			if err != nil {
				return core.Err(fmt.Sprintf(`El stock del insumo "%s" está en una sub-unidad incompatible con la actual.`, supply.Name))
			}
			orderSupply.Available = int32(atDivisor.AvailableSubUnits)
			orderSupply.Reserved = orderSupply.Planned
			if atDivisor.AvailableSubUnits < int64(orderSupply.Planned) {
				shortages = append(shortages, fmt.Sprintf(`"%s": se necesitan %v y hay %v libres`, supply.Name,
					types.FormatSubUnits(int64(orderSupply.Planned), orderSupply.SubDivisor, supply.SbuUnit),
					types.FormatSubUnits(atDivisor.AvailableSubUnits, orderSupply.SubDivisor, supply.SbuUnit)))
			}
			reservationChanges = append(reservationChanges, logistics.ProductionReservationChange{
				WarehouseID: order.SuppliesWarehouseID, ProductID: orderSupply.SupplyID,
				SubUnits: orderSupply.Planned, SubDivisor: orderSupply.SubDivisor,
			})
		}
		if len(shortages) > 0 {
			return core.Err("Faltan insumos para liberar la orden (revise Necesidades de compra): " + strings.Join(shortages, "; ") + ".")
		}
		if err := logistics.ApplyProductionReservationIn(transaction, companyID, reservationChanges); err != nil {
			return err
		}

		order.Supplies, order.Labor, order.Status = orderSupplies, orderLabor, types.ProductionOrderStatusReleased
		order.Updated, order.UpdatedBy = core.SUnixTime(), req.User.ID
		orders := []types.ProductionOrder{*order}
		table := db.TableOf[types.ProductionOrder]()
		if err := db.UpdateIn(transaction, &orders, table.Supplies, table.Labor, table.Status, table.Updated, table.UpdatedBy); err != nil {
			return core.Err("Error al liberar la orden de producción:", err)
		}
		releasedOrder = orders[0]
		stampStoredVersion(&releasedOrder)
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(releasedOrder)
}

// orderReleaseChanges gives back everything an order still holds, for its completion or annulment:
// what it reserves in its supplies warehouse (zeroed) and, subcontracted, what it holds at its workshop.
// AtWorkshop keeps its value: the order shows the material left there.
func orderReleaseChanges(order *types.ProductionOrder, workshopWarehouseID int32) []logistics.ProductionReservationChange {
	changes := []logistics.ProductionReservationChange{}
	for index := range order.Supplies {
		orderSupply := &order.Supplies[index]
		if orderSupply.Reserved > 0 {
			changes = append(changes, logistics.ProductionReservationChange{
				WarehouseID: order.SuppliesWarehouseID, ProductID: orderSupply.SupplyID,
				SubUnits: -orderSupply.Reserved, SubDivisor: orderSupply.SubDivisor,
			})
			orderSupply.Reserved = 0
		}
		if orderSupply.AtWorkshop > 0 && workshopWarehouseID > 0 {
			changes = append(changes, logistics.ProductionReservationChange{
				WarehouseID: workshopWarehouseID, ProductID: orderSupply.SupplyID,
				SubUnits: -orderSupply.AtWorkshop, SubDivisor: orderSupply.SubDivisor,
			})
		}
	}
	return changes
}

// readWorkshopWarehouseID is the warehouse of a subcontractor's material: the one warehouse with its
// SubcontractorID (§8).
func readWorkshopWarehouseID(companyID int32, subcontractorID int32) (int32, error) {
	warehouses := []business.Warehouse{}
	query := db.Query(&warehouses)
	query.Select(query.ID, query.Status, query.SubcontractorID).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return 0, core.Err("Error al obtener los almacenes:", err)
	}
	for _, warehouse := range warehouses {
		if warehouse.Status > 0 && warehouse.SubcontractorID == subcontractorID {
			return warehouse.ID, nil
		}
	}
	return 0, core.Err("El taller de la orden no tiene almacén: cree uno con el taller asignado en Sedes y almacenes.")
}

type productionOrderReceiptRequest struct {
	OrderID int32
	// Version is the order's as the user saw it: a receipt registered meanwhile asks for a new proposal.
	Version int16
	Lines   []types.ProductionReceiptLine
	// Consumptions are the actual sub-units of each order supply, at the order supply's divisor.
	Consumptions []types.ProductionReceiptConsumption
	// Complete finishes the order with this receipt ("Completar"), whatever is left unreceived.
	Complete bool
}

// PostProductionOrderReceipt registers one receipt of a released order (§6.2), in one transaction:
// the order is locked and checked, its reservation decreases by what it consumes, the supplies leave
// split among the products by recipe share, and the products enter at their value.
func PostProductionOrderReceipt(req *core.HandlerArgs) core.HandlerResponse {
	body := productionOrderReceiptRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	var savedOrder types.ProductionOrder

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		order, err := lockProductionOrderIn(transaction, companyID, body.OrderID,
			types.ProductionOrderStatusReleased, types.ProductionOrderStatusInProgress)
		if err != nil {
			return err
		}
		if order.Version != body.Version {
			return core.Err("La orden cambió desde que se abrió el ingreso: revíselo de nuevo.")
		}
		productsByID, err := readOrderProducts(companyID, order.Lines)
		if err != nil {
			return err
		}
		costPerHourByLaborTypeID, err := readLaborTypeRates(companyID)
		if err != nil {
			return err
		}

		// The received units, checked against the order's lines. Over-production is allowed.
		lineIndexByKey := map[[2]int32]int{}
		for index, line := range order.Lines {
			lineIndexByKey[[2]int32{line.ProductID, int32(line.PresentationID)}] = index
		}
		receivedByProductID := map[int32]*types.ReceiptProductUnits{}
		unitsByProductPresentation := map[int32]map[int16]int32{}
		scrappedByProductPresentation := map[int32]map[int16]int32{}
		goodUnitsTotal := int32(0)
		for _, receiptLine := range body.Lines {
			if receiptLine.Units == 0 && receiptLine.Scrapped == 0 {
				continue
			}
			if _, isOrderLine := lineIndexByKey[[2]int32{receiptLine.ProductID, int32(receiptLine.PresentationID)}]; !isOrderLine {
				return core.Err(fmt.Sprintf("El producto %v (presentación %v) no está en la orden.", receiptLine.ProductID, receiptLine.PresentationID))
			}
			if receiptLine.Units < 0 || receiptLine.Scrapped < 0 {
				return core.Err("Las unidades del ingreso no pueden ser negativas.")
			}
			received := receivedByProductID[receiptLine.ProductID]
			if received == nil {
				received = &types.ReceiptProductUnits{ProductID: receiptLine.ProductID}
				receivedByProductID[receiptLine.ProductID] = received
				unitsByProductPresentation[receiptLine.ProductID] = map[int16]int32{}
				scrappedByProductPresentation[receiptLine.ProductID] = map[int16]int32{}
			}
			received.Good += receiptLine.Units
			received.Scrapped += receiptLine.Scrapped
			unitsByProductPresentation[receiptLine.ProductID][receiptLine.PresentationID] += receiptLine.Units
			scrappedByProductPresentation[receiptLine.ProductID][receiptLine.PresentationID] += receiptLine.Scrapped
			goodUnitsTotal += receiptLine.Units
		}
		// Scrapped units carry no value: a receipt needs good units to absorb what it consumed.
		if goodUnitsTotal == 0 {
			return core.Err("El ingreso necesita al menos una unidad buena.")
		}
		receivedProducts := []types.ReceiptProductUnits{}
		for _, productID := range slices.Sorted(maps.Keys(receivedByProductID)) {
			receivedProducts = append(receivedProducts, *receivedByProductID[productID])
		}

		// The lines after this receipt decide whether it finishes the order.
		for _, receiptLine := range body.Lines {
			if index, isOrderLine := lineIndexByKey[[2]int32{receiptLine.ProductID, int32(receiptLine.PresentationID)}]; isOrderLine {
				order.Lines[index].ProducedUnits += receiptLine.Units
				order.Lines[index].ScrappedUnits += receiptLine.Scrapped
			}
		}
		isFinishing := body.Complete || types.IsProductionOrderFullyReceived(order.Lines)

		// The order consumes from its own reservation first: it is decreased before the engine runs,
		// so the engine's reservation check only sees what other orders hold (§5). Finishing gives
		// back the rest.
		supplyIndexByID := map[int32]int{}
		for index, orderSupply := range order.Supplies {
			supplyIndexByID[orderSupply.SupplyID] = index
		}
		// A subcontracted order consumes at its workshop, from what it sent there (§8).
		consumptionWarehouseID, workshopWarehouseID := order.SuppliesWarehouseID, int32(0)
		if order.SubcontractorID > 0 {
			if workshopWarehouseID, err = readWorkshopWarehouseID(companyID, order.SubcontractorID); err != nil {
				return err
			}
			consumptionWarehouseID = workshopWarehouseID
		}
		reservationChanges := []logistics.ProductionReservationChange{}
		consumedBySupplyID := map[int32]int64{}
		for _, consumption := range body.Consumptions {
			index, isOrderSupply := supplyIndexByID[consumption.SupplyID]
			if !isOrderSupply {
				return core.Err(fmt.Sprintf("El insumo %v no está en la orden.", consumption.SupplyID))
			}
			if consumption.SubUnits < 0 {
				return core.Err("El consumo de un insumo no puede ser negativo.")
			}
			if consumption.SubUnits == 0 {
				continue
			}
			orderSupply := &order.Supplies[index]
			consumedBySupplyID[consumption.SupplyID] += int64(consumption.SubUnits)
			orderSupply.Consumed += consumption.SubUnits
			// What it consumes beyond its own reservation must fit in unreserved stock (the engine checks).
			heldSubUnits := core.If(order.SubcontractorID > 0, &orderSupply.AtWorkshop, &orderSupply.Reserved)
			releasedSubUnits := min(consumption.SubUnits, *heldSubUnits)
			*heldSubUnits -= releasedSubUnits
			reservationChanges = append(reservationChanges, logistics.ProductionReservationChange{
				WarehouseID: consumptionWarehouseID, ProductID: orderSupply.SupplyID,
				SubUnits: -releasedSubUnits, SubDivisor: orderSupply.SubDivisor,
			})
		}
		if isFinishing {
			reservationChanges = append(reservationChanges, orderReleaseChanges(order, workshopWarehouseID)...)
		}
		if err := logistics.ApplyProductionReservationIn(transaction, companyID, reservationChanges); err != nil {
			return err
		}

		// Each supply's consumption is split by recipe share, then drawn bucket by bucket.
		consumedSupplyIDs := slices.Sorted(maps.Keys(consumedBySupplyID))
		supplyStockByID, err := readSupplyStock(companyID, consumptionWarehouseID, consumedSupplyIDs)
		if err != nil {
			return err
		}
		receiptInput := productionReceiptInput{
			OrderID: order.ID, WarehouseID: order.WarehouseID, SuppliesWarehouseID: consumptionWarehouseID,
			N: int16(len(order.Receipts) + 1), MovementDate: core.FechaUnix(),
		}
		takenByProductSupply := map[[2]int32]int64{}
		for _, supplyID := range consumedSupplyIDs {
			orderSupply := order.Supplies[supplyIndexByID[supplyID]]
			stock := supplyStockByID[supplyID]
			atDivisor, err := types.ReadSupplyStockAtDivisor(stock, orderSupply.SubDivisor)
			if err != nil {
				return core.Err(fmt.Sprintf("El stock del insumo %v está en una sub-unidad incompatible con la de la orden.", supplyID))
			}
			takes := types.ShareReceiptSupply(supplyID, consumedBySupplyID[supplyID], receivedProducts, productsByID)
			consumptions, err := types.WalkSupplyBuckets(supplyID, orderSupply.SubDivisor, stock, atDivisor, takes)
			if err != nil {
				return err
			}
			for _, take := range takes {
				takenByProductSupply[[2]int32{take.ProductID, supplyID}] += take.SubUnits
			}
			for _, consumption := range consumptions {
				receiptInput.Consumptions = append(receiptInput.Consumptions, receiptConsumption{
					ProductID: consumption.ProductID, SupplyID: consumption.SupplyID, LotID: consumption.LotID,
					SerialNumber: consumption.SerialNumber, SubUnits: consumption.SubUnits, SubDivisor: consumption.SubDivisor,
				})
			}
		}

		// The labor logged since the previous receipt (§9.2), split by the recipes' labor.
		laborValue := order.LaborRecordedAmount
		for _, previousReceipt := range order.Receipts {
			laborValue -= previousReceipt.LaborValue
		}
		laborShares := types.ShareReceiptLabor(max(laborValue, 0), receivedProducts, productsByID, costPerHourByLaborTypeID)
		shortProductIDs := map[int32]bool{}
		for index, received := range receivedProducts {
			product := productsByID[received.ProductID]
			entry := receiptProduct{
				ProductID: received.ProductID, Name: product.Name, SubDivisor: product.SbuQuantity,
				UnitsByPresentation: unitsByProductPresentation[received.ProductID], ScrappedByPresentation: scrappedByProductPresentation[received.ProductID],
				CostSource: types.CostSourceRecipe,
				LaborValue: laborShares[index],
				// The workshop is paid for every unit it made; the overhead is per good unit.
				SubcontractValue: int64(order.SubcontractPricePerUnit) * int64(received.Good+received.Scrapped),
				OverheadValue:    int64(product.ProductionExtraCostPerUnit) * int64(received.Good),
				TracksLots:       product.TracksLots, ShelfLifeDays: product.ShelfLifeDays, QualityHoldOnEntry: product.QualityHoldOnEntry,
			}
			if product.ProductionStandardCost > 0 {
				entry.CostSource, entry.StandardValue = types.CostSourceStandard, int64(product.ProductionStandardCost)*int64(received.Good)
			}
			receiptInput.Products = append(receiptInput.Products, entry)
			// Short: it took less of a supply than its recipe asks for what was made.
			for _, recipeSupply := range product.ProductionSupplies {
				recipeNeed := recipeSupply.SubUnitsWithWaste(received.Good+received.Scrapped, product.ProductionYieldPortions)
				if takenByProductSupply[[2]int32{received.ProductID, recipeSupply.SupplyID}] < int64(recipeNeed) {
					shortProductIDs[received.ProductID] = true
				}
			}
		}
		receipt, err := applyProductionReceiptIn(transaction, req, receiptInput)
		if err != nil {
			return err
		}

		for _, receiptLine := range receipt.Lines {
			index := lineIndexByKey[[2]int32{receiptLine.ProductID, int32(receiptLine.PresentationID)}]
			line := &order.Lines[index]
			line.Value += receiptLine.Value
			line.CostSource = core.If(productsByID[line.ProductID].ProductionStandardCost > 0, types.CostSourceStandard, types.CostSourceRecipe)
			line.ShortSupplies = line.ShortSupplies || shortProductIDs[line.ProductID]
		}
		order.Receipts = append(order.Receipts, receipt)
		order.Status = core.If(isFinishing, types.ProductionOrderStatusFinished, types.ProductionOrderStatusInProgress)
		order.Updated, order.UpdatedBy = core.SUnixTime(), req.User.ID
		orders := []types.ProductionOrder{*order}
		table := db.TableOf[types.ProductionOrder]()
		if err := db.UpdateIn(transaction, &orders, table.Lines, table.Supplies, table.Receipts, table.Status,
			table.Updated, table.UpdatedBy); err != nil {
			return core.Err("Error al registrar el ingreso de la orden:", err)
		}
		savedOrder = orders[0]
		stampStoredVersion(&savedOrder)
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(savedOrder)
}

// PostProductionOrderComplete finishes an order in progress without another receipt: what it still
// reserves is released.
func PostProductionOrderComplete(req *core.HandlerArgs) core.HandlerResponse {
	return changeProductionOrderStatus(req, types.ProductionOrderStatusFinished, types.ProductionOrderStatusInProgress)
}

// PostProductionOrderAnnul annuls an order nothing was received of (§1): its reservations are released.
// An order in progress qualifies too while it has no receipt, no material at the workshop and no hours
// logged: an hour already booked must reach a product through a receipt.
func PostProductionOrderAnnul(req *core.HandlerArgs) core.HandlerResponse {
	return changeProductionOrderStatus(req, types.ProductionOrderStatusAnnulled,
		types.ProductionOrderStatusDraft, types.ProductionOrderStatusReleased, types.ProductionOrderStatusInProgress)
}

func changeProductionOrderStatus(req *core.HandlerArgs, nextStatus int8, allowedStatuses ...int8) core.HandlerResponse {
	body := productionOrderActionRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	var savedOrder types.ProductionOrder
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		order, err := lockProductionOrderIn(transaction, companyID, body.OrderID, allowedStatuses...)
		if err != nil {
			return err
		}
		if nextStatus == types.ProductionOrderStatusAnnulled && len(order.Receipts) > 0 {
			return core.Err("La orden ya tiene ingresos: no se puede anular.")
		}
		if nextStatus == types.ProductionOrderStatusAnnulled && order.LaborRecordedAmount != 0 {
			return core.Err("La orden tiene horas registradas: elimínelas o regístrelas en un ingreso antes de anularla.")
		}
		workshopWarehouseID := int32(0)
		for _, orderSupply := range order.Supplies {
			if orderSupply.AtWorkshop > 0 && nextStatus == types.ProductionOrderStatusAnnulled {
				return core.Err("La orden tiene material en el taller: devuélvalo antes de anularla.")
			}
			if orderSupply.AtWorkshop > 0 && workshopWarehouseID == 0 {
				if workshopWarehouseID, err = readWorkshopWarehouseID(companyID, order.SubcontractorID); err != nil {
					return err
				}
			}
		}
		if err := logistics.ApplyProductionReservationIn(transaction, companyID, orderReleaseChanges(order, workshopWarehouseID)); err != nil {
			return err
		}
		order.Status, order.Updated, order.UpdatedBy = nextStatus, core.SUnixTime(), req.User.ID
		orders := []types.ProductionOrder{*order}
		table := db.TableOf[types.ProductionOrder]()
		if err := db.UpdateIn(transaction, &orders, table.Supplies, table.Status, table.Updated, table.UpdatedBy); err != nil {
			return core.Err("Error al guardar la orden de producción:", err)
		}
		savedOrder = orders[0]
		stampStoredVersion(&savedOrder)
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(savedOrder)
}

// PostProductionOrderStage moves a released or in-progress order to another board column (§9.4).
// Information only: no stock moves and no cost changes.
func PostProductionOrderStage(req *core.HandlerArgs) core.HandlerResponse {
	body := productionOrderActionRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	if body.StageID > 0 {
		stages := []types.ProductionStage{}
		query := db.Query(&stages)
		query.Select(query.ID, query.Status).CompanyID.Equals(companyID).ID.Equals(body.StageID)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener la etapa:", err)
		}
		if len(stages) == 0 || stages[0].Status != 1 {
			return req.MakeErr("La etapa no existe o no está activa.")
		}
	}
	var savedOrder types.ProductionOrder
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		order, err := lockProductionOrderIn(transaction, companyID, body.OrderID,
			types.ProductionOrderStatusReleased, types.ProductionOrderStatusInProgress)
		if err != nil {
			return err
		}
		order.StageID, order.Updated, order.UpdatedBy = body.StageID, core.SUnixTime(), req.User.ID
		orders := []types.ProductionOrder{*order}
		table := db.TableOf[types.ProductionOrder]()
		if err := db.UpdateIn(transaction, &orders, table.StageID, table.Status, table.Updated, table.UpdatedBy); err != nil {
			return core.Err("Error al mover la orden de etapa:", err)
		}
		savedOrder = orders[0]
		stampStoredVersion(&savedOrder)
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(savedOrder)
}

// GetProductionTemplates is the delta of the company's production templates (§4.1).
func GetProductionTemplates(req *core.HandlerArgs) core.HandlerResponse {
	templates := []types.ProductionTemplate{}
	query := db.Query(&templates)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), int8(1))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las plantillas de producción:", err)
	}
	return req.MakeResponse(templates)
}

// PostProductionTemplate creates or updates one template; ss = 0 deletes it. Orders made from it keep
// their lines.
func PostProductionTemplate(req *core.HandlerArgs) core.HandlerResponse {
	template := types.ProductionTemplate{}
	if err := json.Unmarshal([]byte(*req.Body), &template); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	template.Name, template.Notes = strings.TrimSpace(template.Name), strings.TrimSpace(template.Notes)
	if template.Status != 0 {
		template.Status = 1
		templateLines := []types.ProductionOrderLine{}
		for _, templateLine := range template.Lines {
			templateLines = append(templateLines, types.ProductionOrderLine{ProductID: templateLine.ProductID})
		}
		productsByID, err := readOrderProducts(companyID, templateLines)
		if err != nil {
			return req.MakeErr(err)
		}
		if err := types.ValidateProductionTemplate(template, productsByID); err != nil {
			return req.MakeErr(err)
		}
		warehouseIDs := slices.DeleteFunc([]int32{template.WarehouseID, template.SuppliesWarehouseID}, func(id int32) bool { return id == 0 })
		if err := validateCompanyWarehouses(companyID, warehouseIDs...); err != nil {
			return req.MakeErr(err)
		}
	}
	template.CompanyID, template.Updated, template.UpdatedBy = companyID, core.SUnixTime(), req.User.ID

	records := []types.ProductionTemplate{template}
	if template.ID <= 0 {
		if template.Status == 0 {
			return req.MakeErr("No se puede eliminar una plantilla que no se ha guardado.")
		}
		records[0].ID, records[0].Created, records[0].CreatedBy = 0, template.Updated, req.User.ID
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar la plantilla:", err)
		}
		return req.MakeResponse(records[0])
	}
	stored := []types.ProductionTemplate{}
	query := db.Query(&stored)
	query.Select(query.ID).CompanyID.Equals(companyID).ID.Equals(template.ID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener la plantilla:", err)
	}
	if len(stored) == 0 {
		return req.MakeErr("La plantilla no existe.")
	}
	table := db.TableOf[types.ProductionTemplate]()
	if err := db.Update(&records, table.Name, table.Lines, table.DefaultSeries, table.WarehouseID, table.SuppliesWarehouseID,
		table.Notes, table.Status, table.Updated, table.UpdatedBy); err != nil {
		return req.MakeErr("Error al guardar la plantilla:", err)
	}
	return req.MakeResponse(records[0])
}

type productionOrderMaterialRequest struct {
	OrderID int32
	// IsReturn sends the lines back from the workshop to the order's supplies warehouse.
	IsReturn bool
	// Lines are sub-units of each order supply, at the order supply's divisor.
	Lines []types.ProductionReceiptConsumption
}

// PostProductionOrderMaterial sends a subcontracted order's supplies to its workshop or returns the
// leftovers (§8), as type 17 transfers. The reservation travels with them: a shipment takes what it
// sends out of the order's reservation in the supplies warehouse and reserves it on the workshop's row;
// a return gives the workshop's reservation back and leaves the returned stock free.
func PostProductionOrderMaterial(req *core.HandlerArgs) core.HandlerResponse {
	body := productionOrderMaterialRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	var savedOrder types.ProductionOrder

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		order, err := lockProductionOrderIn(transaction, companyID, body.OrderID,
			types.ProductionOrderStatusReleased, types.ProductionOrderStatusInProgress)
		if err != nil {
			return err
		}
		if order.SubcontractorID == 0 {
			return core.Err("La orden no es de maquila: no tiene taller.")
		}
		workshopWarehouseID, err := readWorkshopWarehouseID(companyID, order.SubcontractorID)
		if err != nil {
			return err
		}
		sourceWarehouseID, destinationWarehouseID := order.SuppliesWarehouseID, workshopWarehouseID
		if body.IsReturn {
			sourceWarehouseID, destinationWarehouseID = workshopWarehouseID, order.SuppliesWarehouseID
		}

		// The reservation leaving the source is released before the engine runs, so its check only sees
		// what others hold; the one at the destination is taken after, on stock that has arrived.
		releasedAtSource, reservedAtDestination := []logistics.ProductionReservationChange{}, []logistics.ProductionReservationChange{}
		movements := []logistics.InternalMovement{}
		for _, line := range body.Lines {
			index := slices.IndexFunc(order.Supplies, func(orderSupply types.ProductionOrderSupply) bool {
				return orderSupply.SupplyID == line.SupplyID
			})
			if index < 0 {
				return core.Err(fmt.Sprintf("El insumo %v no está en la orden.", line.SupplyID))
			}
			if line.SubUnits < 0 {
				return core.Err("La cantidad a trasladar no puede ser negativa.")
			}
			if line.SubUnits == 0 {
				continue
			}
			orderSupply := &order.Supplies[index]
			if body.IsReturn {
				if line.SubUnits > orderSupply.AtWorkshop {
					return core.Err(fmt.Sprintf("Del insumo %v solo hay %v en el taller.", line.SupplyID, orderSupply.AtWorkshop))
				}
				orderSupply.AtWorkshop -= line.SubUnits
				releasedAtSource = append(releasedAtSource, logistics.ProductionReservationChange{
					WarehouseID: workshopWarehouseID, ProductID: line.SupplyID, SubUnits: -line.SubUnits, SubDivisor: orderSupply.SubDivisor,
				})
			} else {
				// What it sends beyond its reservation must fit in unreserved stock (the engine checks).
				releasedSubUnits := min(line.SubUnits, orderSupply.Reserved)
				orderSupply.Reserved -= releasedSubUnits
				orderSupply.AtWorkshop += line.SubUnits
				releasedAtSource = append(releasedAtSource, logistics.ProductionReservationChange{
					WarehouseID: order.SuppliesWarehouseID, ProductID: line.SupplyID, SubUnits: -releasedSubUnits, SubDivisor: orderSupply.SubDivisor,
				})
				reservedAtDestination = append(reservedAtDestination, logistics.ProductionReservationChange{
					WarehouseID: workshopWarehouseID, ProductID: line.SupplyID, SubUnits: line.SubUnits, SubDivisor: orderSupply.SubDivisor,
				})
			}
			movements = append(movements, logistics.InternalMovement{
				WarehouseID: sourceWarehouseID, DestWarehouseID: destinationWarehouseID, ProductID: line.SupplyID,
				Quantity: line.SubUnits / int32(orderSupply.SubDivisor), SubQuantity: line.SubUnits % int32(orderSupply.SubDivisor),
				SubDivisor: orderSupply.SubDivisor, Type: logistics.StockMovementTypeProductionTransfer,
				DocumentID: int64(order.ID),
			})
		}
		if len(movements) == 0 {
			return core.Err("No hay material para trasladar.")
		}
		if err := logistics.ApplyProductionReservationIn(transaction, companyID, releasedAtSource); err != nil {
			return err
		}
		if err := logistics.ApplyMovimientosIn(transaction, req, movements); err != nil {
			return core.Err("Error al trasladar el material:", err)
		}
		if err := logistics.ApplyProductionReservationIn(transaction, companyID, reservedAtDestination); err != nil {
			return err
		}

		order.Status, order.Updated, order.UpdatedBy = types.ProductionOrderStatusInProgress, core.SUnixTime(), req.User.ID
		orders := []types.ProductionOrder{*order}
		table := db.TableOf[types.ProductionOrder]()
		if err := db.UpdateIn(transaction, &orders, table.Supplies, table.Status, table.Updated, table.UpdatedBy); err != nil {
			return core.Err("Error al guardar el material de la orden:", err)
		}
		savedOrder = orders[0]
		stampStoredVersion(&savedOrder)
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(savedOrder)
}

// stampStoredVersion gives the answered order the Version its write stored: the ORM raises it in SQL
// (cache.NextVersion), not on the record it was handed. A receipt sends the version back, and is
// refused unless it is the stored one, so a proposal made on an older order is never registered.
func stampStoredVersion(order *types.ProductionOrder) {
	order.Version = int16(cache.NextVersion(int64(order.Version)))
}
