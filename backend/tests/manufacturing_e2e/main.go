// manufacturing_e2e walks the manufacturing flows of docs/MANUFACTURING_PLAN.md end to end against
// the configured database, calling the production handlers in-process as a real user (the seeds'
// runner.CallAs), and checks the stock, the reservations and the values at every step.
//
//	cd backend && go run ./tests/manufacturing_e2e [companyID] [userID]
//
// It writes real, clearly named records ("E2E …") in the company: run it on a dev database only.
package main

import (
	businessModule "app/business"
	business "app/business/types"
	"app/core"
	crm "app/crm/types"
	"app/db"
	"app/exec"
	hr "app/hr/types"
	logisticsModule "app/logistics"
	logistics "app/logistics/types"
	production "app/production"
	"app/production/types"
	"app/seeds/runner"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"
)

var (
	actor    core.UsuarioToken
	failures []string
	runTag   = time.Now().Format("0102-150405")
)

func check(isOK bool, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isOK {
		fmt.Println("  ok   ", message)
		return
	}
	fmt.Println("  FAIL ", message)
	failures = append(failures, message)
}

func must(err error, step string) {
	if err != nil {
		fmt.Println("  FATAL", step, "::", err)
		failures = append(failures, step+": "+err.Error())
		summarize()
	}
}

func summarize() {
	fmt.Printf("\n%d failure(s)\n", len(failures))
	for _, failure := range failures {
		fmt.Println(" -", failure)
	}
	if len(failures) > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

// call runs a handler as the actor. The answer omits zero fields, so the target is reset first: a
// value that dropped to 0 (a released reservation, an annulled status) must not keep its old value.
func call(route string, handler func(*core.HandlerArgs) core.HandlerResponse, payload any, target any) error {
	if target != nil {
		targetValue := reflect.ValueOf(target).Elem()
		targetValue.Set(reflect.Zero(targetValue.Type()))
	}
	return runner.CallAs(actor, route, handler, nil, payload, target)
}

func callQuery(route string, handler func(*core.HandlerArgs) core.HandlerResponse, query map[string]string, target any) error {
	return runner.CallAs(actor, route, handler, query, nil, target)
}

// stockOf is the (balance, reserved, blocked) of a product's presentation-0 row, in sub-units of divisor.
func stockOf(warehouseID, productID int32, presentationID int16, divisor int16) (int64, int64, int64, int64) {
	rows := []logistics.ProductStock{}
	query := db.Query(&rows)
	query.Select().CompanyID.Equals(actor.CompanyID).ID.Equals(logistics.PackProductStockID(warehouseID, productID, presentationID))
	must(query.Exec(), "read stock")
	if len(rows) == 0 {
		return 0, 0, 0, 0
	}
	row := rows[0]
	rowDivisor := max(row.SubDivisor, 1)
	toDivisor := func(quantity core.Quantity) int64 {
		converted, err := quantity.ConvertToDivisor(rowDivisor, divisor)
		must(err, "convert stock")
		return converted.TotalSubUnits(divisor)
	}
	balance := toDivisor(core.Quantity{Units: row.Quantity, Sub: row.SubQuantity}.Add(core.Quantity{Units: row.DetailQuantity, Sub: row.DetailSubQuantity}))
	return balance, toDivisor(core.Quantity{Units: row.ReservedProductionQuantity, Sub: row.ReservedProductionSubQuantity}),
		toDivisor(core.Quantity{Units: row.BlockedQuantity, Sub: row.BlockedSubQuantity}), row.StockValue
}

func readOrder(orderID int32) types.ProductionOrder {
	orders := []types.ProductionOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(actor.CompanyID).ID.Equals(orderID)
	must(query.Exec(), "read order")
	if len(orders) == 0 {
		must(fmt.Errorf("order %v not found", orderID), "read order")
	}
	return orders[0]
}

func orderSupply(order types.ProductionOrder, supplyID int32) types.ProductionOrderSupply {
	for _, orderSupply := range order.Supplies {
		if orderSupply.SupplyID == supplyID {
			return orderSupply
		}
	}
	return types.ProductionOrderSupply{}
}

func main() {
	companyID, userID := int32(1), int32(1)
	if len(os.Args) > 1 {
		parsed, _ := strconv.Atoi(os.Args[1])
		companyID = int32(parsed)
	}
	if len(os.Args) > 2 {
		parsed, _ := strconv.Atoi(os.Args[2])
		userID = int32(parsed)
	}
	core.PopulateVariables()
	if err := core.ConfigureFareward(core.Env.FAREWARD_ADDRESS, core.Env.INTERNAL_APIKEY); err != nil {
		panic(err)
	}
	core.ConfigureTextSearch()
	core.ConfigureFileStore()
	db.Connect(exec.MakeConnParams())
	actor = core.UsuarioToken{CompanyID: companyID, ID: userID}
	stockReq := &core.HandlerArgs{User: &actor}

	// ── Setup: two company warehouses ─────────────────────────────────────────────────────────────
	warehouses := []business.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select().CompanyID.Equals(companyID)
	must(warehouseQuery.Exec(), "read warehouses")
	companyWarehouses := []business.Warehouse{}
	for _, warehouse := range warehouses {
		if warehouse.Status > 0 && warehouse.SubcontractorID == 0 {
			companyWarehouses = append(companyWarehouses, warehouse)
		}
	}
	if len(companyWarehouses) < 1 {
		must(fmt.Errorf("the company has no warehouse"), "setup")
	}
	suppliesWarehouseID, productsWarehouseID := companyWarehouses[0].ID, companyWarehouses[0].ID
	if len(companyWarehouses) > 1 {
		productsWarehouseID = companyWarehouses[1].ID
	}
	fmt.Println("warehouses: supplies", suppliesWarehouseID, "products", productsWarehouseID, "run", runTag)

	// ── Step 1: settings and recipes ──────────────────────────────────────────────────────────────
	fmt.Println("\n[1] labor type, stage, supplies, semi-finished and product recipes")
	laborType := types.ProductionLaborType{}
	must(call("POST.production-labor-type", production.PostProductionLaborType,
		types.ProductionLaborType{Name: "E2E Zapatero " + runTag, CostPerHour: 1200, Status: 1}, &laborType), "labor type")
	stage := types.ProductionStage{}
	must(call("POST.production-stage", production.PostProductionStage,
		types.ProductionStage{Name: "E2E Corte " + runTag, Position: 10, Status: 1}, &stage), "stage")
	check(laborType.ID > 0 && stage.ID > 0, "labor type %v and stage %v created", laborType.ID, stage.ID)

	supplyIDs := []struct{ ID, TempID int32 }{}
	must(call("POST.supply-material", production.PostSupplyMaterial, []production.SupplyMaterialPayload{
		{ID: -1, Name: "E2E Cuero " + runTag, Price: 4000, CurrencyID: 1, UnitID: 1, SbuQuantity: 1000, SbuUnit: "g", Status: 2, AccountCode: 6032},
		{ID: -2, Name: "E2E Hilo " + runTag, Price: 500, CurrencyID: 1, UnitID: 1, Status: 2, AccountCode: 6032},
	}, &supplyIDs), "supplies")
	leatherID, threadID := supplyIDs[0].ID, supplyIDs[1].ID
	check(leatherID > 0 && threadID > 0, "supplies leather %v thread %v", leatherID, threadID)

	// A recipe that names a missing labor type is refused.
	badErr := call("POST.supply-material", production.PostSupplyMaterial, []production.SupplyMaterialPayload{{
		ID: -3, Name: "E2E Malo " + runTag, Status: 2, Origin: types.ProductOriginMade, ProductionYieldPortions: 1,
		ProductionLabor: []types.ProductionLabor{{LaborTypeID: 999999, Minutes: 5}},
	}}, nil)
	check(badErr != nil, "a recipe with an unknown labor type is refused (%v)", badErr)

	// Semi-finished uppers: 200 g of leather and 10 minutes each.
	must(call("POST.supply-material", production.PostSupplyMaterial, []production.SupplyMaterialPayload{{
		ID: -4, Name: "E2E Capellada " + runTag, CurrencyID: 1, UnitID: 1, Status: 2, Origin: types.ProductOriginMade,
		ProductionYieldPortions: 1,
		ProductionSupplies:      []types.ProductionSupply{{SupplyID: leatherID, Quantity: 200, SubDivisor: 1000}},
		ProductionLabor:         []types.ProductionLabor{{LaborTypeID: laborType.ID, Minutes: 10}},
	}}, &supplyIDs), "semi-finished")
	uppersID := supplyIDs[0].ID
	check(uppersID > 0, "semi-finished uppers %v", uppersID)

	// A recipe reaching itself through a semi-finished supply is refused.
	cycleErr := call("POST.supply-material", production.PostSupplyMaterial, []production.SupplyMaterialPayload{{
		ID: uppersID, Name: "E2E Capellada " + runTag, CurrencyID: 1, UnitID: 1, Status: 2, Origin: types.ProductOriginMade,
		ProductionYieldPortions: 1, ProductionSupplies: []types.ProductionSupply{{SupplyID: uppersID, Quantity: 1000, SubDivisor: 1}},
	}}, nil)
	check(cycleErr != nil, "a recipe containing itself is refused (%v)", cycleErr)

	// The shoe: 10 pairs need 10 uppers, 500 g of leather with 10% waste, 1 thread; 60 minutes.
	products := []types.Product{}
	must(call("POST.products", production.PostProducts, []types.Product{{
		ID: -5, Name: "E2E Zapato " + runTag, Origin: types.ProductOriginMade, Price: 15000, CurrencyID: 1, UnitID: 1,
		TracksLots: true, ShelfLifeDays: 365,
		ProductionYieldPortions: 10,
		ProductionSupplies: []types.ProductionSupply{
			{SupplyID: uppersID, Quantity: 10000, SubDivisor: 1},
			{SupplyID: leatherID, Quantity: 500, SubDivisor: 1000, WasteBasisPoints: 1000},
			{SupplyID: threadID, Quantity: 1000, SubDivisor: 1},
		},
		ProductionLabor:            []types.ProductionLabor{{LaborTypeID: laborType.ID, Minutes: 60}},
		ProductionExtraCostPerUnit: 100,
	}}, &products), "product")
	shoeID := products[0].ID
	check(shoeID > 0, "product shoe %v", shoeID)

	// Opening stock of the supplies: 10 kg of leather at 40.00/kg, 20 threads at 5.00.
	must(logistics.ApplyMovimientos(stockReq, []logistics.InternalMovement{
		{WarehouseID: suppliesWarehouseID, ProductID: leatherID, Quantity: 10, SubDivisor: 1000, Price: 4000, Type: logistics.StockMovementTypeManualEntry},
		{WarehouseID: suppliesWarehouseID, ProductID: threadID, Quantity: 20, SubDivisor: 1, Price: 500, Type: logistics.StockMovementTypeManualEntry},
	}), "opening stock")
	leatherBalance, _, _, _ := stockOf(suppliesWarehouseID, leatherID, 0, 1000)
	check(leatherBalance == 10000, "leather stock 10000 g, got %v", leatherBalance)

	// ── Step 2: an order for the semi-finished uppers, released and received in one go ────────────
	fmt.Println("\n[2] order of 20 uppers: release, reserve, receive")
	uppersOrder := types.ProductionOrder{}
	must(call("POST.production-order", production.PostProductionOrder, types.ProductionOrder{
		WarehouseID: suppliesWarehouseID, SuppliesWarehouseID: suppliesWarehouseID,
		Lines: []types.ProductionOrderLine{{ProductID: uppersID, PlannedUnits: 20}},
	}, &uppersOrder), "uppers draft")
	check(uppersOrder.Status == types.ProductionOrderStatusDraft, "draft created %v", uppersOrder.ID)
	must(call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": uppersOrder.ID}, &uppersOrder), "release uppers")
	leatherSupply := orderSupply(uppersOrder, leatherID)
	check(leatherSupply.Planned == 4000 && leatherSupply.Reserved == 4000, "uppers plan 4000 g of leather reserved, got %+v", leatherSupply)
	check(len(uppersOrder.Labor) == 1 && uppersOrder.Labor[0].PlannedMinutes == 200 && uppersOrder.Labor[0].PlannedCost == 4000,
		"uppers plan 200 min costing 40.00, got %+v", uppersOrder.Labor)
	_, leatherReserved, _, _ := stockOf(suppliesWarehouseID, leatherID, 0, 1000)
	check(leatherReserved == 4000, "the stock row reserves 4000 g, got %v", leatherReserved)

	// Somebody else's manual exit may not eat the reservation: 7 kg out of 10 with 4 reserved.
	exitErr := logistics.ApplyMovimientos(stockReq, []logistics.InternalMovement{
		{WarehouseID: suppliesWarehouseID, ProductID: leatherID, Quantity: -7, SubDivisor: 1000, Type: logistics.StockMovementTypeManualExit},
	})
	check(exitErr != nil, "a manual exit into reserved stock is refused (%v)", exitErr)

	must(call("POST.production-order-receipt", production.PostProductionOrderReceipt, map[string]any{
		"OrderID": uppersOrder.ID, "Version": uppersOrder.Version, "Complete": true,
		"Lines":        []map[string]any{{"ProductID": uppersID, "PresentationID": 0, "Units": 20, "Scrapped": 0}},
		"Consumptions": []map[string]any{{"SupplyID": leatherID, "SubUnits": 4000}},
	}, &uppersOrder), "receive uppers")
	check(uppersOrder.Status == types.ProductionOrderStatusFinished, "uppers order finished, status %v", uppersOrder.Status)
	leatherBalance, leatherReserved, _, _ = stockOf(suppliesWarehouseID, leatherID, 0, 1000)
	check(leatherBalance == 6000 && leatherReserved == 0, "leather 6000 g left, nothing reserved, got %v / %v", leatherBalance, leatherReserved)
	uppersBalance, _, _, uppersValue := stockOf(suppliesWarehouseID, uppersID, 0, 1)
	// 4 kg of leather at 40.00 = 160.00; no labor logged yet, no overhead.
	check(uppersBalance == 20 && uppersValue == 16000, "20 uppers worth 160.00, got %v units / %v", uppersBalance, uppersValue)
	if len(uppersOrder.Receipts) == 1 {
		check(uppersOrder.Receipts[0].SuppliesValue == 16000, "receipt supplies value 160.00, got %v", uppersOrder.Receipts[0].SuppliesValue)
	}

	// ── Step 3: an order of 30 shoes, labor, two partial receipts with lots ───────────────────────
	fmt.Println("\n[3] order of 30 shoes: release, labor entry, two receipts, lots")
	shoeOrder := types.ProductionOrder{}
	must(call("POST.production-order", production.PostProductionOrder, types.ProductionOrder{
		WarehouseID: productsWarehouseID, SuppliesWarehouseID: suppliesWarehouseID, StageID: stage.ID,
		Lines: []types.ProductionOrderLine{{ProductID: shoeID, PlannedUnits: 30}},
	}, &shoeOrder), "shoe draft")
	// 30 uppers are needed and only 20 exist: the release is refused and reserves nothing (M17).
	shortErr := call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": shoeOrder.ID}, nil)
	check(shortErr != nil, "release refused for the short uppers (%v)", shortErr)
	_, uppersReserved, _, _ := stockOf(suppliesWarehouseID, uppersID, 0, 1)
	check(uppersReserved == 0, "the refused release reserved nothing, got %v", uppersReserved)
	shoeOrder = readOrder(shoeOrder.ID)

	if shoeOrder.Status != types.ProductionOrderStatusReleased {
		// Expected: 30 uppers needed, 20 in stock. Make 10 more and release again.
		fmt.Println("  (release refused for the short uppers, as expected; making 10 more)")
		extraUppers := types.ProductionOrder{}
		must(call("POST.production-order", production.PostProductionOrder, types.ProductionOrder{
			WarehouseID: suppliesWarehouseID, SuppliesWarehouseID: suppliesWarehouseID,
			Lines: []types.ProductionOrderLine{{ProductID: uppersID, PlannedUnits: 10}},
		}, &extraUppers), "extra uppers draft")
		must(call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": extraUppers.ID}, &extraUppers), "release extra uppers")
		must(call("POST.production-order-receipt", production.PostProductionOrderReceipt, map[string]any{
			"OrderID": extraUppers.ID, "Version": extraUppers.Version, "Complete": true,
			"Lines":        []map[string]any{{"ProductID": uppersID, "PresentationID": 0, "Units": 10}},
			"Consumptions": []map[string]any{{"SupplyID": leatherID, "SubUnits": 2000}},
		}, &extraUppers), "receive extra uppers")
		must(call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": shoeOrder.ID}, &shoeOrder), "release shoes again")
	}
	check(shoeOrder.Status == types.ProductionOrderStatusReleased, "shoe order released, status %v", shoeOrder.Status)
	// Leather: 30 × 500 g ÷ 10 × 1.1 = 1650 g; uppers 30; thread 3.
	check(orderSupply(shoeOrder, leatherID).Planned == 1650, "shoes plan 1650 g of leather, got %v", orderSupply(shoeOrder, leatherID).Planned)
	check(orderSupply(shoeOrder, uppersID).Planned == 30, "shoes plan 30 uppers, got %v", orderSupply(shoeOrder, uppersID).Planned)
	check(orderSupply(shoeOrder, threadID).Planned == 3, "shoes plan 3 threads, got %v", orderSupply(shoeOrder, threadID).Planned)

	// Labor: a Planilla worker logs 2 hours.
	employees := []hr.Employee{}
	employeeQuery := db.Query(&employees)
	employeeQuery.Select().CompanyID.Equals(companyID)
	must(employeeQuery.Exec(), "read employees")
	worker := hr.Employee{}
	for _, employee := range employees {
		if employee.Status == hr.StatusActive && employee.EngagementType == hr.EngagementPayroll && employee.TerminationDate == 0 {
			worker = employee
			break
		}
	}
	laborEntry := types.ProductionLaborEntry{}
	laborErr := call("POST.production-labor-entry", production.PostProductionLaborEntry, types.ProductionLaborEntry{
		OrderID: shoeOrder.ID, EmployeeID: worker.ID, LaborTypeID: laborType.ID, Date: core.FechaUnix(), Minutes: 120, Status: 1,
	}, &laborEntry)
	check(laborErr == nil, "labor entry of worker %v saved (%v)", worker.ID, laborErr)
	check(laborEntry.CostPerHour > 0 && laborEntry.Amount > 0, "the entry is valued: %v/h, amount %v", laborEntry.CostPerHour, laborEntry.Amount)
	shoeOrder = readOrder(shoeOrder.ID)
	check(shoeOrder.Status == types.ProductionOrderStatusInProgress, "a labor entry puts the order in progress, status %v", shoeOrder.Status)
	check(shoeOrder.LaborRecordedAmount == int64(laborEntry.Amount), "order recorded labor %v = entry amount %v", shoeOrder.LaborRecordedAmount, laborEntry.Amount)

	// Receipt 1: 10 good + 1 scrapped, consuming the proposal for 11 of 30.
	must(call("POST.production-order-receipt", production.PostProductionOrderReceipt, map[string]any{
		"OrderID": shoeOrder.ID, "Version": shoeOrder.Version,
		"Lines": []map[string]any{{"ProductID": shoeID, "PresentationID": 0, "Units": 10, "Scrapped": 1}},
		"Consumptions": []map[string]any{
			{"SupplyID": uppersID, "SubUnits": 11}, {"SupplyID": leatherID, "SubUnits": 605}, {"SupplyID": threadID, "SubUnits": 1},
		},
	}, &shoeOrder), "shoe receipt 1")
	check(len(shoeOrder.Receipts) == 1 && shoeOrder.Status == types.ProductionOrderStatusInProgress, "receipt 1 recorded, in progress")
	if len(shoeOrder.Receipts) == 1 {
		receipt := shoeOrder.Receipts[0]
		check(receipt.LaborValue == int64(laborEntry.Amount), "receipt 1 absorbs the logged labor %v, got %v", laborEntry.Amount, receipt.LaborValue)
		check(receipt.OverheadValue == 1000, "overhead 10 good × 1.00 = 10.00, got %v", receipt.OverheadValue)
		check(len(receipt.Lines) == 1 && receipt.Lines[0].LotID > 0, "the shoes entered a lot, got %+v", receipt.Lines)
		lineValue := int64(0)
		for _, line := range receipt.Lines {
			lineValue += int64(line.Value)
		}
		check(lineValue == receipt.SuppliesValue+receipt.LaborValue+receipt.OverheadValue,
			"value = supplies %v + labor %v + overhead %v, got %v", receipt.SuppliesValue, receipt.LaborValue, receipt.OverheadValue, lineValue)
		if len(receipt.Lines) == 1 && receipt.Lines[0].LotID > 0 {
			lots := []logistics.ProductStockLot{}
			lotQuery := db.Query(&lots)
			lotQuery.Select().CompanyID.Equals(companyID).ID.Equals(receipt.Lines[0].LotID)
			must(lotQuery.Exec(), "read lot")
			check(len(lots) == 1 && lots[0].Name == fmt.Sprintf("OP%v-1", shoeOrder.ID) && lots[0].ExpirationDate > 0 &&
				lots[0].QualityStatus == logistics.LotQualityReleased && lots[0].OriginDocumentID == int64(shoeOrder.ID),
				"lot OP%v-1 with expiry, released, origin the order: %+v", shoeOrder.ID, lots)
		}
	}
	shoeBalance, _, _, _ := stockOf(productsWarehouseID, shoeID, 0, 1)
	check(shoeBalance == 10, "10 shoes in stock, got %v", shoeBalance)
	check(orderSupply(shoeOrder, uppersID).Reserved == 19, "uppers still reserved 30 − 11 = 19, got %v", orderSupply(shoeOrder, uppersID).Reserved)

	// A stale version is refused.
	staleErr := call("POST.production-order-receipt", production.PostProductionOrderReceipt, map[string]any{
		"OrderID": shoeOrder.ID, "Version": shoeOrder.Version - 1,
		"Lines": []map[string]any{{"ProductID": shoeID, "PresentationID": 0, "Units": 1}},
	}, nil)
	check(staleErr != nil, "a receipt on a stale version is refused (%v)", staleErr)

	// Receipt 2 completes with 22 good (over-production of 2) and the remainder of the supplies.
	must(call("POST.production-order-receipt", production.PostProductionOrderReceipt, map[string]any{
		"OrderID": shoeOrder.ID, "Version": shoeOrder.Version, "Complete": true,
		"Lines": []map[string]any{{"ProductID": shoeID, "PresentationID": 0, "Units": 22}},
		"Consumptions": []map[string]any{
			{"SupplyID": uppersID, "SubUnits": 19}, {"SupplyID": leatherID, "SubUnits": 1045}, {"SupplyID": threadID, "SubUnits": 2},
		},
	}, &shoeOrder), "shoe receipt 2")
	check(shoeOrder.Status == types.ProductionOrderStatusFinished, "shoe order finished, status %v", shoeOrder.Status)
	check(shoeOrder.Lines[0].ProducedUnits == 32 && shoeOrder.Lines[0].ScrappedUnits == 1, "32 good and 1 scrapped, got %+v", shoeOrder.Lines[0])
	_, uppersReserved, _, _ = stockOf(suppliesWarehouseID, uppersID, 0, 1)
	_, leatherReserved, _, _ = stockOf(suppliesWarehouseID, leatherID, 0, 1000)
	check(uppersReserved == 0 && leatherReserved == 0, "nothing reserved after completion: %v / %v", uppersReserved, leatherReserved)
	laborAfterFinish := call("POST.production-labor-entry", production.PostProductionLaborEntry, types.ProductionLaborEntry{
		OrderID: shoeOrder.ID, EmployeeID: worker.ID, LaborTypeID: laborType.ID, Date: core.FechaUnix(), Minutes: 30, Status: 1,
	}, nil)
	check(laborAfterFinish != nil, "a labor entry on a finished order is refused (%v)", laborAfterFinish)

	// ── Step 4: subcontracting ────────────────────────────────────────────────────────────────────
	fmt.Println("\n[4] subcontracted uppers: workshop warehouse, send, receive at the workshop, return")
	providers := []crm.ClientProvider{}
	providerQuery := db.Query(&providers)
	providerQuery.Select().CompanyID.Equals(companyID)
	must(providerQuery.Exec(), "read providers")
	workshop := crm.ClientProvider{}
	for _, provider := range providers {
		if provider.Type == crm.ClientProviderTypeProvider && provider.Status > 0 {
			workshop = provider
		}
	}
	workshopWarehouseID := int32(0)
	for _, warehouse := range warehouses {
		if warehouse.Status > 0 && warehouse.SubcontractorID == workshop.ID {
			workshopWarehouseID = warehouse.ID
		}
	}
	if workshopWarehouseID == 0 {
		savedWarehouse := business.Warehouse{}
		must(call("POST.warehouses", businessModule.PostWarehouse, business.Warehouse{
			SiteID: companyWarehouses[0].SiteID, Name: "E2E Taller " + workshop.Name, Status: 1, SubcontractorID: workshop.ID,
		}, &savedWarehouse), "workshop warehouse")
		workshopWarehouseID = savedWarehouse.ID
		check(savedWarehouse.ExcludeFromOnlineSales, "a workshop warehouse is never sold online")
	}
	duplicateErr := call("POST.warehouses", businessModule.PostWarehouse, business.Warehouse{
		SiteID: companyWarehouses[0].SiteID, Name: "E2E Taller duplicado", Status: 1, SubcontractorID: workshop.ID,
	}, nil)
	check(duplicateErr != nil, "a second warehouse for the same workshop is refused (%v)", duplicateErr)

	subcontracted := types.ProductionOrder{}
	must(call("POST.production-order", production.PostProductionOrder, types.ProductionOrder{
		WarehouseID: suppliesWarehouseID, SuppliesWarehouseID: suppliesWarehouseID,
		SubcontractorID: workshop.ID, SubcontractPricePerUnit: 300,
		Lines: []types.ProductionOrderLine{{ProductID: uppersID, PlannedUnits: 5}},
	}, &subcontracted), "subcontracted draft")
	must(call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": subcontracted.ID}, &subcontracted), "release subcontracted")
	must(call("POST.production-order-material", production.PostProductionOrderMaterial, map[string]any{
		"OrderID": subcontracted.ID, "IsReturn": false, "Lines": []map[string]any{{"SupplyID": leatherID, "SubUnits": 1200}},
	}, &subcontracted), "send material")
	atWorkshop, workshopReserved, _, _ := stockOf(workshopWarehouseID, leatherID, 0, 1000)
	check(atWorkshop == 1200 && workshopReserved == 1200, "1200 g at the workshop, reserved there, got %v / %v", atWorkshop, workshopReserved)
	check(orderSupply(subcontracted, leatherID).AtWorkshop == 1200 && orderSupply(subcontracted, leatherID).Reserved == 0,
		"order supply: 1200 at the workshop, 0 left reserved at home, got %+v", orderSupply(subcontracted, leatherID))
	annulErr := call("POST.production-order-annul", production.PostProductionOrderAnnul, map[string]any{"OrderID": subcontracted.ID}, nil)
	check(annulErr != nil, "annul is refused with material at the workshop (%v)", annulErr)
	must(call("POST.production-order-receipt", production.PostProductionOrderReceipt, map[string]any{
		"OrderID": subcontracted.ID, "Version": subcontracted.Version,
		"Lines":        []map[string]any{{"ProductID": uppersID, "PresentationID": 0, "Units": 5}},
		"Consumptions": []map[string]any{{"SupplyID": leatherID, "SubUnits": 1000}},
	}, &subcontracted), "subcontracted receipt")
	if len(subcontracted.Receipts) == 1 {
		check(subcontracted.Receipts[0].SubcontractValue == 1500, "workshop value 5 × 3.00, got %v", subcontracted.Receipts[0].SubcontractValue)
	}
	check(subcontracted.Status == types.ProductionOrderStatusFinished, "fully received: finished, status %v", subcontracted.Status)
	atWorkshop, workshopReserved, _, _ = stockOf(workshopWarehouseID, leatherID, 0, 1000)
	check(atWorkshop == 200 && workshopReserved == 0, "200 g left at the workshop, unreserved after completion, got %v / %v", atWorkshop, workshopReserved)

	// ── Step 5: annul a released order ────────────────────────────────────────────────────────────
	fmt.Println("\n[5] annul a released order")
	annulled := types.ProductionOrder{}
	must(call("POST.production-order", production.PostProductionOrder, types.ProductionOrder{
		WarehouseID: suppliesWarehouseID, SuppliesWarehouseID: suppliesWarehouseID,
		Lines: []types.ProductionOrderLine{{ProductID: uppersID, PlannedUnits: 3}},
	}, &annulled), "draft to annul")
	must(call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": annulled.ID}, &annulled), "release to annul")
	_, leatherReserved, _, _ = stockOf(suppliesWarehouseID, leatherID, 0, 1000)
	check(leatherReserved == 600, "600 g reserved by the order to annul, got %v", leatherReserved)
	must(call("POST.production-order-annul", production.PostProductionOrderAnnul, map[string]any{"OrderID": annulled.ID}, &annulled), "annul")
	_, leatherReserved, _, _ = stockOf(suppliesWarehouseID, leatherID, 0, 1000)
	check(annulled.Status == types.ProductionOrderStatusAnnulled && leatherReserved == 0, "annulled and released, reserved %v", leatherReserved)

	// ── Step 6: templates ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n[6] template")
	template := types.ProductionTemplate{}
	templateErr := call("POST.production-template", production.PostProductionTemplate, types.ProductionTemplate{
		Name: "E2E Plantilla " + runTag, DefaultSeries: 4, WarehouseID: productsWarehouseID, SuppliesWarehouseID: suppliesWarehouseID, Status: 1,
		Lines: []types.ProductionTemplateLine{{ProductID: shoeID, Run: []types.PresentationQuantity{{PresentationID: 0, Units: 6}}}},
	}, &template)
	check(templateErr == nil && template.ID > 0, "template saved %v (%v)", template.ID, templateErr)

	// ── Step 7: reservation recompute agrees with the counters ───────────────────────────────────
	fmt.Println("\n[7] the recompute finds nothing to change")
	recompute := exec.RecomputeProductionReservation(&core.ExecArgs{Message: strconv.Itoa(int(companyID))})
	check(recompute.Error == "", "recompute ran: %v %v", recompute.Message, recompute.Error)

	// ── Step 8: the daily close writes a production order of Kind 2 ──────────────────────────────
	fmt.Println("\n[8] daily close, if a warehouse has a pending day")
	for _, warehouse := range companyWarehouses {
		pendingDays := []production.DailyClosePendingDay{}
		if err := callQuery("GET.production-close-pending", production.GetDailyClosePending,
			map[string]string{"warehouse-id": strconv.Itoa(int(warehouse.ID))}, &pendingDays); err != nil || len(pendingDays) == 0 {
			continue
		}
		preview := production.DailyClosePreview{}
		// The close reads the company configuration (DynamoDB): without AWS access it cannot run here.
		if err := callQuery("GET.production-close-preview", production.GetDailyClosePreview, map[string]string{
			"warehouse-id": strconv.Itoa(int(warehouse.ID)), "supplies-warehouse-id": strconv.Itoa(int(warehouse.ID)),
			"date": strconv.Itoa(int(pendingDays[0].Date)),
		}, &preview); err != nil {
			fmt.Println("  SKIP  close preview unavailable:", core.StrCut(err.Error(), 120))
			break
		}
		fmt.Println("  preview of warehouse", warehouse.ID, "day", preview.Date, "products", len(preview.Products), "blocking", preview.BlockingReasons)
		if len(preview.BlockingReasons) > 0 {
			continue
		}
		closeOrder := types.ProductionOrder{}
		closeErr := call("POST.production-close", production.PostDailyClose, map[string]any{
			"WarehouseID": warehouse.ID, "SuppliesWarehouseID": warehouse.ID, "Date": preview.Date, "PreviewHash": preview.PreviewHash,
		}, &closeOrder)
		check(closeErr == nil, "close ran (%v)", closeErr)
		check(closeOrder.Kind == types.ProductionOrderKindDailyClose && closeOrder.Status == types.ProductionOrderStatusFinished &&
			len(closeOrder.Receipts) == 1 && len(closeOrder.Lines) > 0,
			"the close is a finished Kind 2 order with one receipt: kind %v status %v receipts %v lines %v",
			closeOrder.Kind, closeOrder.Status, len(closeOrder.Receipts), len(closeOrder.Lines))
		closes := map[string]any{}
		must(callQuery("GET.production-closes", production.GetProductionCloses, map[string]string{"warehouse-id": strconv.Itoa(int(warehouse.ID))}, &closes), "close history")
		history, _ := closes["records"].([]any)
		check(len(history) > 0, "the close history lists %v close(s)", len(history))
		break
	}

	// ── Step 9: a semi-finished made in a quarantined lot is not available until released ────────
	fmt.Println("\n[9] quarantine on entry: blocked stock, refused consumption, release of the lot")
	liningIDs := []struct{ ID, TempID int32 }{}
	must(call("POST.supply-material", production.PostSupplyMaterial, []production.SupplyMaterialPayload{{
		ID: -9, Name: "E2E Forro " + runTag, CurrencyID: 1, UnitID: 1, Status: 2, Origin: types.ProductOriginMade,
		TracksLots: true, QualityHoldOnEntry: true, ProductionYieldPortions: 1,
		ProductionSupplies: []types.ProductionSupply{{SupplyID: leatherID, Quantity: 100, SubDivisor: 1000}},
	}}, &liningIDs), "lining")
	liningID := liningIDs[0].ID
	liningOrder := types.ProductionOrder{}
	must(call("POST.production-order", production.PostProductionOrder, types.ProductionOrder{
		WarehouseID: suppliesWarehouseID, SuppliesWarehouseID: suppliesWarehouseID,
		Lines: []types.ProductionOrderLine{{ProductID: liningID, PlannedUnits: 5}},
	}, &liningOrder), "lining draft")
	must(call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": liningOrder.ID}, &liningOrder), "release lining")
	must(call("POST.production-order-receipt", production.PostProductionOrderReceipt, map[string]any{
		"OrderID": liningOrder.ID, "Version": liningOrder.Version, "Complete": true,
		"Lines":        []map[string]any{{"ProductID": liningID, "PresentationID": 0, "Units": 5}},
		"Consumptions": []map[string]any{{"SupplyID": leatherID, "SubUnits": 500}},
	}, &liningOrder), "receive lining")
	liningBalance, _, liningBlocked, _ := stockOf(suppliesWarehouseID, liningID, 0, 1)
	check(liningBalance == 5 && liningBlocked == 5, "5 linings in stock, all blocked in quarantine, got %v / %v", liningBalance, liningBlocked)
	liningLotID := int32(0)
	if len(liningOrder.Receipts) == 1 && len(liningOrder.Receipts[0].Lines) == 1 {
		liningLotID = liningOrder.Receipts[0].Lines[0].LotID
	}
	check(liningLotID > 0, "the linings entered lot %v", liningLotID)

	// A product consuming linings cannot be released while the lot is in quarantine.
	bagIDs := []types.Product{}
	must(call("POST.products", production.PostProducts, []types.Product{{
		ID: -10, Name: "E2E Bolso " + runTag, Origin: types.ProductOriginMade, Price: 9000, CurrencyID: 1, UnitID: 1,
		ProductionYieldPortions: 1, ProductionSupplies: []types.ProductionSupply{{SupplyID: liningID, Quantity: 1000, SubDivisor: 1}},
	}}, &bagIDs), "bag")
	bagOrder := types.ProductionOrder{}
	must(call("POST.production-order", production.PostProductionOrder, types.ProductionOrder{
		WarehouseID: productsWarehouseID, SuppliesWarehouseID: suppliesWarehouseID,
		Lines: []types.ProductionOrderLine{{ProductID: bagIDs[0].ID, PlannedUnits: 2}},
	}, &bagOrder), "bag draft")
	quarantineErr := call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": bagOrder.ID}, nil)
	check(quarantineErr != nil, "a release needing quarantined linings is refused (%v)", quarantineErr)
	must(call("POST.product-stock-lot-status", logisticsModule.PostProductStockLotStatus,
		map[string]any{"LotID": liningLotID, "QualityStatus": logistics.LotQualityReleased}, nil), "release the lot")
	_, _, liningBlocked, _ = stockOf(suppliesWarehouseID, liningID, 0, 1)
	check(liningBlocked == 0, "the released lot unblocks the stock, blocked %v", liningBlocked)
	releaseErr := call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": bagOrder.ID}, &bagOrder)
	check(releaseErr == nil && bagOrder.Status == types.ProductionOrderStatusReleased, "now the bag order releases (%v)", releaseErr)
	if bagOrder.Status == types.ProductionOrderStatusReleased {
		must(call("POST.production-order-annul", production.PostProductionOrderAnnul, map[string]any{"OrderID": bagOrder.ID}, nil), "annul bag order")
	}

	// ── Step 10: the monthly cap of a Planilla worker ────────────────────────────────────────────
	fmt.Println("\n[10] labor cap: a worker's month never books more than their estimated payroll cost")
	capOrder := types.ProductionOrder{}
	must(call("POST.production-order", production.PostProductionOrder, types.ProductionOrder{
		WarehouseID: suppliesWarehouseID, SuppliesWarehouseID: suppliesWarehouseID,
		Lines: []types.ProductionOrderLine{{ProductID: uppersID, PlannedUnits: 1}},
	}, &capOrder), "cap draft")
	must(call("POST.production-order-release", production.PostProductionOrderRelease, map[string]any{"OrderID": capOrder.ID}, &capOrder), "release cap order")
	bigEntry := types.ProductionLaborEntry{}
	must(call("POST.production-labor-entry", production.PostProductionLaborEntry, types.ProductionLaborEntry{
		OrderID: capOrder.ID, EmployeeID: worker.ID, LaborTypeID: laborType.ID, Date: core.FechaUnix(), Minutes: 24 * 60, Status: 1,
	}, &bigEntry), "first day")
	monthTotal := func() int64 {
		entries := []types.ProductionLaborEntry{}
		query := db.Query(&entries)
		query.Select(query.Amount, query.Status).CompanyID.Equals(companyID).EmployeeID.Equals(worker.ID).Period.Equals(bigEntry.Period)
		must(query.Exec(), "month entries")
		total := int64(0)
		for _, entry := range entries {
			if entry.Status == 1 {
				total += int64(entry.Amount)
			}
		}
		return total
	}
	monthlyCap := int64(bigEntry.CostPerHour) * 8 * 26 // ≈ the estimate the rate came from (8 h × 26 days)
	for day := 0; day < 30 && monthTotal() < monthlyCap; day++ {
		must(call("POST.production-labor-entry", production.PostProductionLaborEntry, types.ProductionLaborEntry{
			OrderID: capOrder.ID, EmployeeID: worker.ID, LaborTypeID: laborType.ID, Date: core.FechaUnix(), Minutes: 24 * 60, Status: 1,
		}, nil), "another day")
	}
	finalEntry := types.ProductionLaborEntry{}
	must(call("POST.production-labor-entry", production.PostProductionLaborEntry, types.ProductionLaborEntry{
		OrderID: capOrder.ID, EmployeeID: worker.ID, LaborTypeID: laborType.ID, Date: core.FechaUnix(), Minutes: 60, Status: 1,
	}, &finalEntry), "past the cap")
	check(finalEntry.Amount == 0, "an hour past the monthly cap books 0, got %v (month total %v, cap ≈ %v)", finalEntry.Amount, monthTotal(), monthlyCap)
	check(monthTotal() <= monthlyCap+int64(bigEntry.CostPerHour), "the month stays within the cap: %v ≤ ≈%v", monthTotal(), monthlyCap)
	capOrder = readOrder(capOrder.ID)
	check(capOrder.LaborRecordedAmount == monthTotalOfOrder(capOrder.ID), "the order's recorded labor equals its entries")
	// The test entries would distort this month's labor report: they are deleted again.
	entries := []types.ProductionLaborEntry{}
	entryQuery := db.Query(&entries)
	entryQuery.Select().CompanyID.Equals(companyID).OrderID.Equals(capOrder.ID)
	must(entryQuery.Exec(), "cap entries")
	for _, entry := range entries {
		if entry.Status == 1 {
			entry.Status = 0
			must(call("POST.production-labor-entry", production.PostProductionLaborEntry, entry, nil), "delete cap entry")
		}
	}
	capOrder = readOrder(capOrder.ID)
	check(capOrder.LaborRecordedAmount == 0, "deleting the entries leaves no recorded labor, got %v", capOrder.LaborRecordedAmount)
	must(call("POST.production-order-annul", production.PostProductionOrderAnnul, map[string]any{"OrderID": capOrder.ID}, nil), "annul cap order")

	summarize()
}

func monthTotalOfOrder(orderID int32) int64 {
	entries := []types.ProductionLaborEntry{}
	query := db.Query(&entries)
	query.Select(query.Amount, query.Status).CompanyID.Equals(actor.CompanyID).OrderID.Equals(orderID)
	must(query.Exec(), "order entries")
	total := int64(0)
	for _, entry := range entries {
		if entry.Status == 1 {
			total += int64(entry.Amount)
		}
	}
	return total
}
