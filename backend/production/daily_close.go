package production

import (
	"app/cloud"
	"app/core"
	"app/db"
	logistics "app/logistics/types"
	"app/production/types"
	sales "app/sales/types"
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

// The daily production close (docs/PRODUCTION_PLAN.md §5): the made products a warehouse delivered
// without their stock exit are produced from their recipes, consuming the supplies, and leave with
// their sales. types.PlanDailyClose is the calculation; this file reads its inputs and writes it.

// DailyClosePendingDay is a day of a warehouse with sales still waiting for the close.
type DailyClosePendingDay struct {
	Date      int16
	SaleCount int32
	Units     int32
}

// DailyClosePreview is the close of one day as it would run now.
type DailyClosePreview struct {
	WarehouseID         int32
	SuppliesWarehouseID int32
	Date                int16
	// MovementDate is the date its movements take: max(Date, yesterday).
	MovementDate    int16
	IsStrict        bool
	Products        []types.DailyCloseProduct
	Supplies        []types.DailyCloseSupply
	BlockingReasons []string
	SaleOrderIDs    []int64
	// PreviewHash is what the close is asked to run: it refuses when the day changed since.
	PreviewHash string
}

// dailyCloseRun is everything a close reads and computes before it writes.
type dailyCloseRun struct {
	preview                    DailyClosePreview
	plan                       types.DailyClosePlan
	sales                      []sales.SaleOrder
	productsByID               map[int32]types.Product
	unitsByProductPresentation map[int32]map[int16]int32
	costPerHourByLaborTypeID   map[int32]int32
}

// readPendingCloseSales are the delivered sales of warehouseID whose made products' stock exit waits
// for a close. The pending set only holds the sales since the last close, so it is read whole.
func readPendingCloseSales(companyID int32, warehouseID int32) ([]sales.SaleOrder, error) {
	pendingSales := []sales.SaleOrder{}
	query := db.Query(&pendingSales)
	// ClientID is read because the close writes it back with Lines (see PostDailyClose).
	query.Select(query.ID, query.Date, query.ClientID, query.WarehouseID, query.Lines, query.Status, query.OnlineSaleOrderID,
		query.PendingStockMovement, query.Updated).
		CompanyID.Equals(companyID).PendingStockMovement.Equals(1).
		Delta(db.DeltaCursor{}, sales.OrderStatusDelivered, sales.OrderStatusCompleted)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener las ventas pendientes del cierre:", err)
	}
	// A Delta() read ends with its metadata records, which carry a negative ID.
	return slices.DeleteFunc(pendingSales, func(sale sales.SaleOrder) bool {
		return sale.ID <= 0 || sale.WarehouseID != warehouseID
	}), nil
}

// pendingLineUnits is the whole units a pending line holds, as its reservation counted them: a
// sub-unit line holds one more.
func pendingLineUnits(line sales.SaleOrderLine) int32 {
	quantity := core.UnpackQuantityLine(line.Quantity)
	return quantity.Units + core.If(quantity.Sub > 0, int32(1), 0)
}

func GetDailyClosePending(req *core.HandlerArgs) core.HandlerResponse {
	warehouseID := int32(req.GetQueryInt("warehouse-id"))
	if warehouseID <= 0 {
		return req.MakeErr("Se requiere el almacén.")
	}
	pendingSales, err := readPendingCloseSales(req.User.CompanyID, warehouseID)
	if err != nil {
		return req.MakeErr(err)
	}
	dayByDate := map[int16]*DailyClosePendingDay{}
	for _, sale := range pendingSales {
		day := dayByDate[sale.Date]
		if day == nil {
			day = &DailyClosePendingDay{Date: sale.Date}
			dayByDate[sale.Date] = day
		}
		day.SaleCount++
		for _, line := range sales.LinesWithStockExitPending(sale.Lines, true) {
			day.Units += pendingLineUnits(line)
		}
	}
	pendingDays := []DailyClosePendingDay{}
	for _, day := range dayByDate {
		pendingDays = append(pendingDays, *day)
	}
	slices.SortFunc(pendingDays, func(a, b DailyClosePendingDay) int { return cmp.Compare(a.Date, b.Date) })
	return req.MakeResponse(pendingDays)
}

// prepareDailyClose reads the day's pending sales, the recipes, the supplies and their stock in the
// supplies warehouse, and plans the close. The close runs it again inside its transaction.
func prepareDailyClose(companyID int32, warehouseID int32, suppliesWarehouseID int32, date int16) (*dailyCloseRun, error) {
	if warehouseID <= 0 || suppliesWarehouseID <= 0 || date <= 0 {
		return nil, core.Err("Se requieren el almacén, el almacén de insumos y el día.")
	}
	today := core.FechaUnix()
	if date > today {
		return nil, core.Err("No se puede cerrar un día futuro.")
	}
	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return nil, core.Err("Error al leer la configuración de la empresa:", err)
	}
	pendingSales, err := readPendingCloseSales(companyID, warehouseID)
	if err != nil {
		return nil, err
	}
	run := &dailyCloseRun{
		preview: DailyClosePreview{
			WarehouseID: warehouseID, SuppliesWarehouseID: suppliesWarehouseID, Date: date,
			MovementDate: max(date, today-1),
			IsStrict:     core.HasCompanyFlag(companyConfig.Company.Flags, types.CompanyFlagStrictSupplyClose),
		},
		unitsByProductPresentation: map[int32]map[int16]int32{},
	}

	// Oldest first: a day is closed only once every earlier one is.
	soldUnitsByProduct := map[int32]int32{}
	for _, sale := range pendingSales {
		if sale.Date < date {
			run.preview.BlockingReasons = append(run.preview.BlockingReasons,
				fmt.Sprintf("Hay ventas pendientes de un día anterior (%v): ciérrelo primero.", core.DateUnixToTime(int64(sale.Date)*86400).UTC().Format("2006-01-02")))
			break
		}
	}
	for _, sale := range pendingSales {
		if sale.Date != date {
			continue
		}
		run.sales = append(run.sales, sale)
		for _, line := range sales.LinesWithStockExitPending(sale.Lines, true) {
			units := pendingLineUnits(line)
			soldUnitsByProduct[line.ProductID] += units
			if run.unitsByProductPresentation[line.ProductID] == nil {
				run.unitsByProductPresentation[line.ProductID] = map[int16]int32{}
			}
			run.unitsByProductPresentation[line.ProductID][line.PresentationID] += units
		}
	}
	slices.SortFunc(run.sales, func(a, b sales.SaleOrder) int { return cmp.Compare(a.ID, b.ID) })
	if len(run.sales) == 0 {
		run.preview.BlockingReasons = append(run.preview.BlockingReasons, "El día no tiene ventas pendientes del cierre en este almacén.")
		return run, nil
	}

	if run.productsByID, err = readProductsByID(companyID, slices.Collect(maps.Keys(soldUnitsByProduct)),
		func(query *types.ProductTable) []db.Coln {
			return db.Cols(query.ID, query.Name, query.SbuQuantity, query.ProductionYieldPortions,
				query.ProductionSupplies, query.ProductionExtraCostPerUnit, query.ProductionStandardCost, query.ProductionLabor)
		}); err != nil {
		return nil, err
	}
	supplyIDs := []int32{}
	for _, product := range run.productsByID {
		for _, recipeSupply := range product.ProductionSupplies {
			supplyIDs = append(supplyIDs, recipeSupply.SupplyID)
		}
	}
	suppliesByID, err := readProductsByID(companyID, supplyIDs, func(query *types.ProductTable) []db.Coln {
		return db.Cols(query.ID, query.Name, query.Status, query.Origin, query.SbuQuantity, query.SbuUnit, query.AccountCode)
	})
	if err != nil {
		return nil, err
	}
	supplyStockByID, err := readSupplyStock(companyID, suppliesWarehouseID, slices.Collect(maps.Keys(suppliesByID)))
	if err != nil {
		return nil, err
	}

	if run.costPerHourByLaborTypeID, err = readLaborTypeRates(companyID); err != nil {
		return nil, err
	}

	run.plan = types.PlanDailyClose(types.DailyCloseInput{
		IsStrict: run.preview.IsStrict, SoldUnitsByProduct: soldUnitsByProduct,
		ProductsByID: run.productsByID, SuppliesByID: suppliesByID, SupplyStockByID: supplyStockByID,
		CostPerHourByLaborTypeID: run.costPerHourByLaborTypeID,
	})
	run.preview.Products = run.plan.Products
	run.preview.Supplies = run.plan.Supplies
	run.preview.BlockingReasons = append(run.preview.BlockingReasons, run.plan.BlockingReasons...)

	// The hash covers the sales as they are (their Updated), what the close would consume and what
	// each product is valued at, so a sale delivered, paid or annulled meanwhile, a supply that
	// moved, or a recipe, standard cost or extra cost edited, asks for a new preview.
	hashValues := []any{warehouseID, suppliesWarehouseID, date}
	for _, sale := range run.sales {
		run.preview.SaleOrderIDs = append(run.preview.SaleOrderIDs, sale.ID)
		hashValues = append(hashValues, sale.ID, sale.Updated)
	}
	for _, closeProduct := range run.plan.Products {
		hashValues = append(hashValues, closeProduct.ProductID, closeProduct.CostSource, closeProduct.Value)
	}
	for _, consumption := range run.plan.Consumptions {
		hashValues = append(hashValues, consumption.ProductID, consumption.SupplyID, consumption.LotID,
			consumption.SerialNumber, consumption.SubUnits, consumption.EstimatedValue)
	}
	run.preview.PreviewHash = fmt.Sprint(core.HashInt64(hashValues...))
	return run, nil
}

// readLaborTypeRates is the estimate cost per hour of each of the company's labor types, deleted ones
// included: a recipe line naming one still values at its last rate.
func readLaborTypeRates(companyID int32) (map[int32]int32, error) {
	laborTypes := []types.ProductionLaborType{}
	query := db.Query(&laborTypes)
	query.Select(query.ID, query.CostPerHour).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los tipos de trabajo:", err)
	}
	costPerHourByLaborTypeID := map[int32]int32{}
	for _, laborType := range laborTypes {
		costPerHourByLaborTypeID[laborType.ID] = laborType.CostPerHour
	}
	return costPerHourByLaborTypeID, nil
}

// readProductsByID reads the company's products of productIDs, with the columns selectColumns names.
func readProductsByID(companyID int32, productIDs []int32, selectColumns func(query *types.ProductTable) []db.Coln) (map[int32]types.Product, error) {
	productsByID := map[int32]types.Product{}
	if len(productIDs) == 0 {
		return productsByID, nil
	}
	products := []types.Product{}
	query := db.Query(&products)
	query.Select(selectColumns(query)...).CompanyID.Equals(companyID).ID.In(core.MakeSliceInclude(productIDs).Values...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los productos del cierre:", err)
	}
	for _, product := range products {
		productsByID[product.ID] = product
	}
	return productsByID, nil
}

// readSupplyStock reads each supply's stock row in the supplies warehouse (presentation 0), its
// detail rows and their lots' status and expiry, the buckets in the order a consumption walks them.
func readSupplyStock(companyID int32, suppliesWarehouseID int32, supplyIDs []int32) (map[int32]types.SupplyStock, error) {
	supplyStockByID := map[int32]types.SupplyStock{}
	if len(supplyIDs) == 0 {
		return supplyStockByID, nil
	}
	stockIDs := []int64{}
	for _, supplyID := range supplyIDs {
		stockIDs = append(stockIDs, logistics.PackProductStockID(suppliesWarehouseID, supplyID, 0))
	}
	stockRows := []logistics.ProductStock{}
	stockQuery := db.Query(&stockRows)
	stockQuery.Select(stockQuery.ID, stockQuery.ProductID, stockQuery.Quantity, stockQuery.SubQuantity,
		stockQuery.SubDivisor, stockQuery.StockValue, stockQuery.ReservedProductionQuantity, stockQuery.ReservedProductionSubQuantity).
		CompanyID.Equals(companyID).ID.In(stockIDs...)
	if err := stockQuery.Exec(); err != nil {
		return nil, core.Err("Error al obtener el stock de los insumos:", err)
	}
	details := []logistics.ProductStockDetail{}
	detailQuery := db.Query(&details)
	detailQuery.Select(detailQuery.ProductStockID, detailQuery.LotID, detailQuery.SerialNumber,
		detailQuery.Quantity, detailQuery.SubQuantity).
		CompanyID.Equals(companyID).ProductStockID.In(stockIDs...)
	if err := detailQuery.Exec(); err != nil {
		return nil, core.Err("Error al obtener el detalle de stock de los insumos:", err)
	}
	lotByID := map[int32]logistics.ProductStockLot{}
	if lotIDs := core.Map(details, func(detail logistics.ProductStockDetail) int32 { return detail.LotID }); len(lotIDs) > 0 {
		lots := []logistics.ProductStockLot{}
		lotQuery := db.Query(&lots)
		lotQuery.Select(lotQuery.ID, lotQuery.QualityStatus, lotQuery.ExpirationDate).
			CompanyID.Equals(companyID).ID.In(lotIDs...)
		if err := lotQuery.Exec(); err != nil {
			return nil, core.Err("Error al obtener los lotes de los insumos:", err)
		}
		for _, lot := range lots {
			lotByID[lot.ID] = lot
		}
	}
	today := core.FechaUnix()
	for _, stockRow := range stockRows {
		supplyStock := types.SupplyStock{
			SubDivisor: stockRow.SubDivisor, StockValue: stockRow.StockValue,
			ReservedProduction: core.Quantity{Units: stockRow.ReservedProductionQuantity, Sub: stockRow.ReservedProductionSubQuantity},
			Buckets:            []types.SupplyStockBucket{{Quantity: core.Quantity{Units: stockRow.Quantity, Sub: stockRow.SubQuantity}}},
		}
		for _, detail := range details {
			if detail.ProductStockID == stockRow.ID && (detail.Quantity != 0 || detail.SubQuantity != 0) {
				lot := lotByID[detail.LotID]
				supplyStock.Buckets = append(supplyStock.Buckets, types.SupplyStockBucket{
					LotID: detail.LotID, SerialNumber: detail.SerialNumber,
					Quantity:      core.Quantity{Units: detail.Quantity, Sub: detail.SubQuantity},
					QualityStatus: lot.QualityStatus, ExpirationDate: lot.ExpirationDate,
				})
			}
		}
		types.OrderSupplyStockBuckets(supplyStock.Buckets, today)
		supplyStockByID[stockRow.ProductID] = supplyStock
	}
	return supplyStockByID, nil
}

func GetDailyClosePreview(req *core.HandlerArgs) core.HandlerResponse {
	run, err := prepareDailyClose(req.User.CompanyID, int32(req.GetQueryInt("warehouse-id")),
		int32(req.GetQueryInt("supplies-warehouse-id")), int16(req.GetQueryInt("date")))
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(run.preview)
}

type dailyCloseRequest struct {
	WarehouseID         int32
	SuppliesWarehouseID int32
	Date                int16
	PreviewHash         string
}

// PostDailyClose runs the close the user accepted (§5.5), in one transaction: the day's pending sales
// are locked, the plan recomputed and checked against the preview, then the supplies are consumed,
// the products produced and the sales' exits written, their reservations released and their flags
// cleared.
func PostDailyClose(req *core.HandlerArgs) core.HandlerResponse {
	body := dailyCloseRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	var savedClose types.ProductionOrder

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		run, err := prepareDailyClose(companyID, body.WarehouseID, body.SuppliesWarehouseID, body.Date)
		if err != nil {
			return err
		}
		// Lock the sales, then plan again from what the locks guard: a payment, delivery or annulment
		// of one of them commits before or after the close, never in between.
		lockedSales := []sales.SaleOrder{}
		for _, sale := range run.sales {
			lockedSales = append(lockedSales, sales.SaleOrder{CompanyID: companyID, ID: sale.ID})
		}
		if err := db.LockByKeys(transaction, &lockedSales); err != nil {
			return core.Err("Error al bloquear las ventas del cierre:", err)
		}
		if run, err = prepareDailyClose(companyID, body.WarehouseID, body.SuppliesWarehouseID, body.Date); err != nil {
			return err
		}
		if run.preview.PreviewHash != body.PreviewHash {
			return core.Err("El día cambió desde que se revisó el resumen: revíselo de nuevo.")
		}
		if len(run.preview.BlockingReasons) > 0 {
			return core.Err(run.preview.BlockingReasons[0])
		}
		movementDate := run.preview.MovementDate
		nowTime := core.SUnixTime()

		// The close is a production order born finished (docs/MANUFACTURING_PLAN.md §3, M4). It is
		// inserted first: its ID is the DocumentID of every stock movement it writes.
		closeOrder := []types.ProductionOrder{{
			CompanyID: companyID, Kind: types.ProductionOrderKindDailyClose,
			WarehouseID: body.WarehouseID, SuppliesWarehouseID: body.SuppliesWarehouseID,
			Date: body.Date, IsStrict: run.preview.IsStrict, SaleOrderIDs: run.preview.SaleOrderIDs,
			Status:  types.ProductionOrderStatusFinished,
			Created: nowTime, CreatedBy: req.User.ID, Updated: nowTime, UpdatedBy: req.User.ID,
		}}
		if err := db.InsertIn(transaction, &closeOrder); err != nil {
			return core.Err("Error al registrar el cierre:", err)
		}

		receiptInput := productionReceiptInput{
			OrderID: closeOrder[0].ID, WarehouseID: body.WarehouseID, SuppliesWarehouseID: body.SuppliesWarehouseID,
			N: 1, MovementDate: movementDate,
		}
		for _, consumption := range run.plan.Consumptions {
			receiptInput.Consumptions = append(receiptInput.Consumptions, receiptConsumption{
				ProductID: consumption.ProductID, SupplyID: consumption.SupplyID, LotID: consumption.LotID,
				SerialNumber: consumption.SerialNumber, SubUnits: consumption.SubUnits, SubDivisor: consumption.SubDivisor,
			})
		}
		// Each product enters at its value right before its sales leave: the engine applies them in
		// order, so the exits leave at the cost the entries set. Its labor is at standard (M26).
		// It enters the free bucket even when it tracks lots: the sale exits that follow carry no lot
		// and only draw from the free bucket, so units entered into a lot would be stranded there and
		// the exits would fail. What a close makes leaves the same day, so there is nothing to trace.
		for _, closeProduct := range run.plan.Products {
			product := run.productsByID[closeProduct.ProductID]
			receiptInput.Products = append(receiptInput.Products, receiptProduct{
				ProductID: closeProduct.ProductID, Name: product.Name, SubDivisor: product.SbuQuantity,
				UnitsByPresentation: run.unitsByProductPresentation[closeProduct.ProductID],
				CostSource:          closeProduct.CostSource,
				StandardValue:       int64(closeProduct.Value),
				LaborValue:          types.RecipeLaborStandardCost(product, closeProduct.Units, run.costPerHourByLaborTypeID),
				OverheadValue:       int64(product.ProductionExtraCostPerUnit) * int64(closeProduct.Units),
			})
		}
		movements := []logistics.InternalMovement{}
		pendingDeliveryChanges := []logistics.PendingDeliveryChange{}
		salesToUpdate := []sales.SaleOrder{}
		for _, sale := range run.sales {
			pendingLines := sales.LinesWithStockExitPending(sale.Lines, true)
			for _, line := range pendingLines {
				lineQuantity := core.UnpackQuantityLine(line.Quantity).Negate()
				movements = append(movements, logistics.InternalMovement{
					WarehouseID: sale.WarehouseID, ProductID: line.ProductID, PresentationID: line.PresentationID,
					Quantity: lineQuantity.Units, SubQuantity: lineQuantity.Sub, SubDivisor: max(line.SubDivisor, core.QuantityDivisorNone),
					Type: logistics.StockMovementTypeSaleDelivery, DocumentID: sale.ID, Date: movementDate,
				})
			}
			pendingDeliveryChanges = append(pendingDeliveryChanges,
				sales.PendingDeliveryChanges(pendingLines, sale.WarehouseID, sale.OnlineSaleOrderID > 0, -1)...)

			// Settled: the lines' exits are done. Status does not change, so the sale's frames, client
			// week rows and web order mirror stay as they are.
			sale.Lines = slices.Clone(sale.Lines)
			for lineIndex := range sale.Lines {
				sale.Lines[lineIndex].StockExitPending = false
			}
			sale.RefreshPendingStockMovement()
			sale.CompanyID, sale.Updated, sale.UpdatedBy = companyID, nowTime, req.User.ID
			sale.StampPeriod()
			salesToUpdate = append(salesToUpdate, sale)
		}
		receiptInput.FollowingMovements = movements
		receipt, err := applyProductionReceiptIn(transaction, req, receiptInput)
		if err != nil {
			return err
		}
		if err := logistics.ApplyPendingDeliveryIn(transaction, companyID, req.User.ID, pendingDeliveryChanges); err != nil {
			return err
		}
		saleTable := db.TableOf[sales.SaleOrder]()
		if err := db.UpdateIn(transaction, &salesToUpdate,
			// Lines feeds the (Date, ClientID, DetailProductIDs) index group, and the ORM reads a ClientID
			// of 0 (a sale without a client) as missing unless it is written too: it is, unchanged.
			saleTable.Lines, saleTable.ClientID, saleTable.Updated, saleTable.UpdatedBy,
			// The delta indexes pack PendingStockMovement and Period with Status: written together.
			saleTable.PendingStockMovement, saleTable.Status, saleTable.Period,
		); err != nil {
			return core.Err("Error al actualizar las ventas del cierre:", err)
		}

		closeOrder[0].Receipts = []types.ProductionReceipt{receipt}
		closeOrder[0].Lines = types.DailyCloseOrderLines(run.plan.Products, receipt)
		for _, closeSupply := range run.plan.Supplies {
			closeOrder[0].Supplies = append(closeOrder[0].Supplies, types.ProductionOrderSupply{
				SupplyID: closeSupply.SupplyID, SubDivisor: closeSupply.SubDivisor,
				Planned: closeSupply.Needed, Available: closeSupply.Available, Consumed: closeSupply.Consumed,
			})
		}
		orderTable := db.TableOf[types.ProductionOrder]()
		if err := db.UpdateIn(transaction, &closeOrder, orderTable.Lines, orderTable.Supplies, orderTable.Receipts); err != nil {
			return core.Err("Error al registrar el valor de la producción:", err)
		}
		savedClose = closeOrder[0]
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(savedClose)
}

// GetProductionCloses is the delta of a warehouse's closes, the history under the close: its
// production orders of Kind 2.
func GetProductionCloses(req *core.HandlerArgs) core.HandlerResponse {
	warehouseID := int32(req.GetQueryInt("warehouse-id"))
	if warehouseID <= 0 {
		return req.MakeErr("Se requiere el almacén.")
	}
	closes := []types.ProductionOrder{}
	query := db.Query(&closes)
	query.Select().CompanyID.Equals(req.User.CompanyID).Kind.Equals(types.ProductionOrderKindDailyClose).
		WarehouseID.Equals(warehouseID).Delta(req.GetDeltaCursor("records"))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los cierres de producción:", err)
	}
	return req.MakeResponse(&map[string]any{"records": &closes})
}
