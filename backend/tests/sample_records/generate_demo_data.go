// fn-generate-demo-data seeds a whole ERP the way a two-store Lima minimarket would have lived it,
// in one pass over the days so every cash movement is written in date order: each movement stores
// its register's balance at write time, and writing one generator after another over the same
// days would leave those running balances out of order.
//
// One-time setup, before the first day:
//   - fn-generate-supply-data with the clock on the first day: catalog import, replenishment
//     settings, supplies and fixed assets.
//   - fn-generate-projects-flow.
//   - One till per store and a bank current account; sales planning, seasonality curves and
//     Lima shipping costs.
//
// Each day: purchase orders with the supplier's factura (paid from the bank) → expenses and their
// payments (from the bank) → sales issued as boletas/facturas (into the store's till) → a few
// annulments → the tills' takings transferred to the bank → the HR day (seed_hr.go: staff on the
// first day, advances, vacations, novedades, benefit, monthly and settlement runs paid from the
// bank) → at month end, the till counts and the depreciation run.
//
// Every generator draws from the fixed sampleDataSeed, so a wiped database refilled with the same
// arguments (and the same real date) gets the same data back.
package sample_records

import (
	"app/accounting"
	"app/business"
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	"app/finance"
	financeTypes "app/finance/types"
	"app/sales"
	salesTypes "app/sales/types"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

const (
	demoStateFileName = "genix_demo_data_state.json"
	demoBankName      = "BCP Cta. Cte. Soles"
	// What each till keeps overnight as change; the rest of the day's takings goes to the bank.
	demoTillFloat int32 = 50000
)

// demoAnnulReasons are what a cashier writes when voiding a sale.
var demoAnnulReasons = []string{
	"Error al registrar la venta",
	"El cliente desistió de la compra",
	"Producto con defecto de fábrica",
	"Venta registrada por duplicado",
}

// demoShippingCosts is the delivery price list for the Lima districts the stores serve: a flat fee
// plus a per-kilo charge, both in soles, rising with the distance from the stores.
var demoShippingCosts = []salesTypes.ShippingCost{
	{CityID: 150141, FlatCost: 5, CostPerKg: 0.5},  // Surquillo
	{CityID: 150122, FlatCost: 6, CostPerKg: 0.5},  // Miraflores
	{CityID: 150131, FlatCost: 6, CostPerKg: 0.5},  // San Isidro
	{CityID: 150130, FlatCost: 7, CostPerKg: 0.6},  // San Borja
	{CityID: 150140, FlatCost: 7, CostPerKg: 0.6},  // Santiago de Surco
	{CityID: 150104, FlatCost: 8, CostPerKg: 0.6},  // Barranco
	{CityID: 150116, FlatCost: 8, CostPerKg: 0.6},  // Lince
	{CityID: 150113, FlatCost: 9, CostPerKg: 0.7},  // Jesús María
	{CityID: 150115, FlatCost: 9, CostPerKg: 0.7},  // La Victoria
	{CityID: 150134, FlatCost: 9, CostPerKg: 0.7},  // San Luis
	{CityID: 150120, FlatCost: 10, CostPerKg: 0.8}, // Magdalena del Mar
	{CityID: 150121, FlatCost: 10, CostPerKg: 0.8}, // Pueblo Libre
	{CityID: 150108, FlatCost: 10, CostPerKg: 0.8}, // Chorrillos
	{CityID: 150101, FlatCost: 11, CostPerKg: 0.8}, // Lima
	{CityID: 150136, FlatCost: 12, CostPerKg: 0.9}, // San Miguel
	{CityID: 150114, FlatCost: 12, CostPerKg: 0.9}, // La Molina
	{CityID: 150103, FlatCost: 14, CostPerKg: 1},   // Ate
	{CityID: 150133, FlatCost: 14, CostPerKg: 1},   // San Juan de Miraflores
	{CityID: 150132, FlatCost: 16, CostPerKg: 1.2}, // San Juan de Lurigancho
	{CityID: 150135, FlatCost: 16, CostPerKg: 1.2}, // San Martín de Porres
}

func makeDemoDataDefaults() erpHistoryConfig {
	config := makeErpHistoryDefaults()
	config.days = 60
	config.salesMin, config.salesMax = 200, 400
	config.annulMin, config.annulMax = 2, 5
	// 200–400 tickets a day of 3–8 lines drain ~5,000 units; two orders of 150 products each
	// restock about that.
	config.purchaseOrderProducts = 150
	return config
}

// GenerateDemoData is the entry point behind `fn-generate-demo-data`.
func GenerateDemoData(args *core.ExecArgs) core.FuncResponse {
	config, configError := parseHistoryArgs(args.Message, makeDemoDataDefaults())
	if configError != nil {
		return args.MakeErr("Argumentos inválidos:", configError)
	}

	generator := newHistoryGenerator(config, demoStateFileName)
	defer core.SetHistoricalUnix(0)

	if err := generator.loadState(); err != nil {
		return args.MakeErr("No se pudo cargar el estado de la corrida:", err)
	}
	if !generator.state.DemoSetupDone && !config.dryRun {
		if err := generator.runDemoSetup(); err != nil {
			return args.MakeErr(err)
		}
		generator.state.DemoSetupDone = true
		if err := generator.saveState(); err != nil {
			return args.MakeErr(err)
		}
	}

	if err := generator.ensureDemoWarehouses(); err != nil {
		return args.MakeErr("No se pudieron preparar los almacenes:", err)
	}
	if err := generator.validateContext(); err != nil {
		return args.MakeErr("No se pudo validar el contexto:", err)
	}
	if err := generator.ensureDemoCashAccounts(); err != nil {
		return args.MakeErr("No se pudieron preparar las cajas:", err)
	}
	if err := generator.prepareErpCatalog(); err != nil {
		return args.MakeErr(err)
	}
	if err := generator.prepareExpenses(); err != nil {
		return args.MakeErr(err)
	}
	if config.dryRun {
		return generator.previewPayloads(args)
	}
	if err := generator.ensureSalesPlanning(); err != nil {
		return args.MakeErr("No se pudo crear la planificación de ventas:", err)
	}
	if err := generator.saveDemoShippingCosts(); err != nil {
		return args.MakeErr("No se pudieron guardar los costos de envío:", err)
	}

	payroll := newDemoPayroll(generator, core.TimeToFechaUnix(generator.realNow)-int16(config.days-1))
	generator.daySteps = []historyDayStep{
		{name: "purchase-orders", run: generator.runPurchaseOrdersStep},
		{name: "expenses", run: generator.runExpensesStep},
		{name: "sales", run: generator.runDailySales},
		{name: "annulments", run: generator.runAnnulmentsStep},
		{name: "till-transfers", run: generator.runTillTransfersStep},
		{name: "payroll", run: payroll.runPayrollStep},
		{name: "month-end", run: generator.runMonthEndStep},
	}
	if config.isErpOnly {
		generator.daySteps = slices.DeleteFunc(generator.daySteps, func(step historyDayStep) bool { return step.name == "payroll" })
	}
	historicalDates, err := generator.runDayLoop()
	if err != nil {
		return args.MakeErr(err)
	}

	core.Log("GenerateDemoData:: terminado", "días", len(historicalDates), "stats", generator.state.Stats)
	return core.FuncResponse{
		Message: "Datos de demostración generados correctamente.",
		Content: map[string]any{
			"days":  len(historicalDates),
			"stats": generator.state.Stats,
			// Sale summaries are rebuilt by cron action 2, which does not run in this process.
			"unixDaysToReprocess": generator.state.AffectedUnixDays,
		},
	}
}

// runDemoSetup runs the generators that are not day by day. The supply data is written with the
// clock on the first simulated morning, so the catalog, the supplies and the asset payments predate
// the history instead of landing after it.
func (generator *erpHistoryGenerator) runDemoSetup() error {
	firstUnixDay := core.TimeToFechaUnix(generator.realNow) - int16(generator.config.days-1)
	core.SetHistoricalUnix(generator.dayStartTime(firstUnixDay).Add(6 * time.Hour).Unix())
	if response := GenerateSupplyData(&core.ExecArgs{}); response.Error != "" {
		return core.Err("fn-generate-supply-data falló:", response.Error)
	}

	// Projects are not part of the dated history; they run on their own simulated timeline.
	core.SetHistoricalUnix(0)
	if generator.config.isErpOnly {
		return nil
	}
	if response := GenerateProjectsFlow(&core.ExecArgs{}); response.Error != "" {
		return core.Err("fn-generate-projects-flow falló:", response.Error)
	}
	return nil
}

// ensureDemoWarehouses creates the stores' warehouses a fresh company lacks: fn-init seeds only the
// first one. The new warehouse sits on the first one's site and must come out with the ID the
// configuration names, or the history would buy and sell into the wrong place.
func (generator *erpHistoryGenerator) ensureDemoWarehouses() error {
	warehouses := []businessTypes.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.SiteID, warehouseQuery.Status).
		CompanyID.Equals(generator.userToken.CompanyID)
	if err := warehouseQuery.Exec(); err != nil {
		return err
	}
	existingWarehouseIDs := core.SliceSet[int32]{}
	firstSiteID := int32(0)
	for _, warehouse := range warehouses {
		existingWarehouseIDs.Add(warehouse.ID)
		if warehouse.ID == generator.config.warehouseIDs[0] {
			firstSiteID = warehouse.SiteID
		}
	}
	if firstSiteID == 0 {
		return core.Err("no existe el almacén", generator.config.warehouseIDs[0], "; corre primero fn-init")
	}

	for warehouseIndex, warehouseID := range generator.config.warehouseIDs {
		if existingWarehouseIDs.Include(warehouseID) {
			continue
		}
		bodyBytes, err := json.Marshal(businessTypes.Warehouse{
			SiteID: firstSiteID, Name: fmt.Sprint("Almacén Tienda ", warehouseIndex+1), Status: 1,
		})
		if err != nil {
			return err
		}
		request := generator.makeRequest("POST.warehouses", nil, string(bodyBytes))
		createdWarehouse := businessTypes.Warehouse{}
		if err := decodeResponse(generator.send(request, business.PostWarehouse), &createdWarehouse); err != nil {
			return err
		}
		if createdWarehouse.ID != warehouseID {
			return core.Err("el almacén creado tiene ID", createdWarehouse.ID, "y se esperaba", warehouseID)
		}
	}
	return nil
}

// ensureDemoCashAccounts gives each store its own till and the company a bank current account,
// creating the missing ones by name. The first store keeps the company's first till. Suppliers and
// expenses are paid from the bank; each store's sales are collected in its till.
func (generator *erpHistoryGenerator) ensureDemoCashAccounts() error {
	cashBanks := []financeTypes.CashBank{}
	cashBankQuery := db.Query(&cashBanks)
	cashBankQuery.Select(cashBankQuery.ID, cashBankQuery.Name, cashBankQuery.Type,
		cashBankQuery.CurrencyType, cashBankQuery.Status).
		CompanyID.Equals(generator.userToken.CompanyID).Status.Equals(1)
	if err := cashBankQuery.Exec(); err != nil {
		return err
	}
	cashBankIDByName := map[string]int32{}
	for _, cashBank := range cashBanks {
		if cashBank.CurrencyType == financeTypes.CurrencyPEN {
			cashBankIDByName[cashBank.Name] = cashBank.ID
		}
	}

	warehouses := []businessTypes.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.SiteID).
		CompanyID.Equals(generator.userToken.CompanyID).ID.In(generator.config.warehouseIDs...)
	if err := warehouseQuery.Exec(); err != nil {
		return err
	}
	siteByWarehouse := map[int32]int32{}
	for _, warehouse := range warehouses {
		siteByWarehouse[warehouse.ID] = warehouse.SiteID
	}

	ensureCashBank := func(name string, cashBankType int32, siteID int32) (int32, error) {
		if cashBankID := cashBankIDByName[name]; cashBankID > 0 {
			return cashBankID, nil
		}
		bodyBytes, err := json.Marshal(financeTypes.CashBank{
			ID: -1, Name: name, Type: cashBankType, SiteID: siteID, CurrencyType: financeTypes.CurrencyPEN, Status: 1,
		})
		if err != nil {
			return 0, err
		}
		request := generator.makeRequest("POST.cash-banks", nil, string(bodyBytes))
		createdCashBank := financeTypes.CashBank{}
		if err := decodeResponse(generator.send(request, finance.PostCashBanks), &createdCashBank); err != nil {
			return 0, core.Err("no se pudo crear la caja", name, ":", err)
		}
		return createdCashBank.ID, nil
	}

	for warehouseIndex, warehouseID := range generator.config.warehouseIDs {
		if warehouseIndex == 0 {
			generator.tillByWarehouse[warehouseID] = generator.cashBankID
			continue
		}
		tillID, err := ensureCashBank(fmt.Sprint("Caja Tienda ", warehouseIndex+1), 1, siteByWarehouse[warehouseID])
		if err != nil {
			return err
		}
		generator.tillByWarehouse[warehouseID] = tillID
	}
	bankID, err := ensureCashBank(demoBankName, 2, siteByWarehouse[generator.config.warehouseIDs[0]])
	if err != nil {
		return err
	}
	generator.bankCashBankID = bankID
	generator.purchaseCashBankID = bankID
	generator.expenseCashBankID = bankID
	core.Log("GenerateDemoData:: cajas", "tiendas", generator.tillByWarehouse, "banco", bankID)
	return nil
}

// ensureSalesPlanning writes three seasonality curves and a weekly plan for every product of the
// pool, unless the company already has planning.
func (generator *erpHistoryGenerator) ensureSalesPlanning() error {
	existingPlanning := []salesTypes.SalesPlanning{}
	planningQuery := db.Query(&existingPlanning)
	planningQuery.Select(planningQuery.ID).CompanyID.Equals(generator.userToken.CompanyID).Limit(1)
	if err := planningQuery.Exec(); err != nil {
		return err
	}
	if len(existingPlanning) > 0 {
		return nil
	}

	// Week multipliers ×1000: steady sellers, a December peak, and summer drinks (Jan–Mar high,
	// winter low).
	makeCurve := func(name string, percentForWeek func(week int16) int16) salesTypes.SeasonalityCurve {
		curve := salesTypes.SeasonalityCurve{Name: name}
		for week := int16(1); week <= 52; week++ {
			curve.Curve = append(curve.Curve, salesTypes.SeasonalityCurveWeek{Week: week, Percent: percentForWeek(week)})
		}
		return curve
	}
	curves := []salesTypes.SeasonalityCurve{
		makeCurve("Estable", func(week int16) int16 { return 1000 }),
		makeCurve("Campaña navideña", func(week int16) int16 { return core.If(week >= 47, int16(1000+(week-46)*150), 1000) }),
		makeCurve("Verano", func(week int16) int16 {
			return core.If(week <= 12, int16(1350), core.If(week >= 24 && week <= 35, int16(800), int16(1000)))
		}),
	}
	bodyBytes, err := json.Marshal(curves)
	if err != nil {
		return err
	}
	request := generator.makeRequest("POST.seasonality-curve", nil, string(bodyBytes))
	savedCurves := []salesTypes.SeasonalityCurve{}
	if err := decodeResponse(generator.send(request, sales.PostSeasonalityCurve), &savedCurves); err != nil {
		return err
	}

	planning := make([]salesTypes.SalesPlanning, 0, len(generator.productPool))
	for _, productID := range generator.productPool {
		curve := savedCurves[generator.random.Intn(len(savedCurves))]
		baseQuantity := int32(5 + generator.random.Intn(56))
		productPlanning := salesTypes.SalesPlanning{ProductID: productID, BaseQuantity: baseQuantity, SeasonalityCurveID: curve.ID}
		for _, curveWeek := range curve.Curve {
			productPlanning.WeeklyQuantity = append(productPlanning.WeeklyQuantity, salesTypes.SalesPlanningWeek{
				Week: curveWeek.Week, Quantity: int16(baseQuantity * int32(curveWeek.Percent) / 1000),
			})
		}
		planning = append(planning, productPlanning)
	}
	for batchStart := 0; batchStart < len(planning); batchStart += supplyPostBatchSize {
		batchBytes, err := json.Marshal(planning[batchStart:min(batchStart+supplyPostBatchSize, len(planning))])
		if err != nil {
			return err
		}
		batchRequest := generator.makeRequest("POST.sales-planning", nil, string(batchBytes))
		if response := sales.PostSalesPlanning(&batchRequest); response.StatusCode != 200 {
			return core.Err(response.Error)
		}
	}
	core.Log("GenerateDemoData:: planificación de ventas", "curvas", len(savedCurves), "productos", len(planning))
	return nil
}

// saveDemoShippingCosts writes the Lima delivery price list. The handler merges by district, so a
// rerun with the same prices writes nothing.
func (generator *erpHistoryGenerator) saveDemoShippingCosts() error {
	shippingCosts := slices.Clone(demoShippingCosts)
	for costIndex := range shippingCosts {
		shippingCosts[costIndex].HasUpdated = true
	}
	bodyBytes, err := json.Marshal(shippingCosts)
	if err != nil {
		return err
	}
	request := generator.makeRequest("POST.shipping-costs", nil, string(bodyBytes))
	if response := generator.send(request, sales.PostShippingCosts); response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	return nil
}

// runAnnulmentsStep voids the sales flagged for annulment, refunding into the till that collected
// them. The handler checks the annul sub-access, which a real request learns from the gate; the
// generator grants it to itself for this call only.
func (generator *erpHistoryGenerator) runAnnulmentsStep(dayStart time.Time) error {
	annulTime := dayStart.Add(20*time.Hour + 30*time.Minute)
	for len(generator.state.PendingAnnulments) > 0 {
		annulment := generator.state.PendingAnnulments[0]
		bodyBytes, err := json.Marshal(map[string]any{
			"ID":               annulment.SaleID,
			"RefundCashBankID": annulment.CashBankID,
			"Reason":           demoAnnulReasons[generator.random.Intn(len(demoAnnulReasons))],
		})
		if err != nil {
			return err
		}
		core.SetHistoricalUnix(annulTime.Unix())
		request := generator.makeRequest("POST.sale-order-annul", nil, string(bodyBytes))
		annulToken := generator.userToken
		annulToken.SubAccesos = map[int32]uint16{salesTypes.AccesoIDGestionVentas: 1 << (core.SubAccesoTodosID - 1)}
		request.User = &annulToken
		if response := generator.send(request, sales.PostSaleOrderAnnul); response.StatusCode != 200 {
			return core.Err("no se pudo anular la venta", annulment.SaleID, ":", response.Error)
		}

		generator.state.Stats.SalesAnnulled++
		generator.state.PendingAnnulments = generator.state.PendingAnnulments[1:]
		if err := generator.saveState(); err != nil {
			return err
		}
		annulTime = annulTime.Add(time.Minute)
	}
	return nil
}

// runTillTransfersStep deposits each till's takings in the bank, keeping the change float. It
// re-reads the balance before each transfer, so a resumed step only moves what is still there.
func (generator *erpHistoryGenerator) runTillTransfersStep(dayStart time.Time) error {
	transferTime := dayStart.Add(21 * time.Hour)
	for _, warehouseID := range generator.config.warehouseIDs {
		tillID := generator.tillByWarehouse[warehouseID]
		if tillID == generator.bankCashBankID {
			continue
		}
		tillBalance, err := generator.readCashBankBalance(tillID)
		if err != nil {
			return err
		}
		transferAmount := tillBalance - demoTillFloat
		if transferAmount <= 0 {
			continue
		}
		if err := generator.postManualCashMovement(transferTime, financeTypes.CashBankMovement{
			CashBankID:    tillID,
			CashBankRefID: generator.bankCashBankID,
			Type:          financeTypes.CashMovementTypeTransfer,
			Amount:        -transferAmount,
			FinalAmount:   demoTillFloat,
		}); err != nil {
			return core.Err("no se pudo transferir la caja", tillID, "al banco:", err)
		}
		generator.state.Stats.TillTransfers++
		transferTime = transferTime.Add(time.Minute)
	}
	return nil
}

// runMonthEndStep closes the month: each till is counted — usually exact, sometimes a few soles
// short (charged to the cashier) or over (other income) — and the assets' depreciation is posted.
func (generator *erpHistoryGenerator) runMonthEndStep(dayStart time.Time) error {
	if !isMonthDay(dayStart, 31) {
		return nil
	}
	countTime := dayStart.Add(21*time.Hour + 30*time.Minute)
	for _, warehouseID := range generator.config.warehouseIDs {
		tillID := generator.tillByWarehouse[warehouseID]
		if tillID == generator.bankCashBankID {
			continue
		}
		systemAmount, err := generator.readCashBankBalance(tillID)
		if err != nil {
			return err
		}
		difference, counterAccountCode := int32(0), int32(0)
		switch countOutcome := generator.random.Intn(100); {
		case countOutcome < 20:
			difference, counterAccountCode = -int32(100+generator.random.Intn(1901)), financeTypes.PCGEOtherStaffReceivables
		case countOutcome < 30:
			difference, counterAccountCode = int32(100+generator.random.Intn(701)), financeTypes.PCGEOtherOperatingIncome
		}

		reconciliation := struct {
			financeTypes.CashReconciliation
			CounterAccountCode int32
		}{
			CashReconciliation: financeTypes.CashReconciliation{
				CashBankID: tillID, SystemAmount: systemAmount, ActualAmount: systemAmount + difference,
			},
			CounterAccountCode: counterAccountCode,
		}
		bodyBytes, err := json.Marshal(reconciliation)
		if err != nil {
			return err
		}
		core.SetHistoricalUnix(countTime.Unix())
		request := generator.makeRequest("POST.cash-banks-reconciliation", nil, string(bodyBytes))
		responseBody := map[string]any{}
		if err := decodeResponse(generator.send(request, finance.PostCashReconciliation), &responseBody); err != nil {
			return core.Err("no se pudo cuadrar la caja", tillID, ":", err)
		}
		if storedBalance, isStale := responseBody["NeedUpdateSaldo"]; isStale {
			return core.Err("el saldo de la caja", tillID, "no coincide al cuadrar; saldo guardado:", storedBalance)
		}
		generator.state.Stats.Reconciliations++
		countTime = countTime.Add(time.Minute)
	}

	core.SetHistoricalUnix(countTime.Unix())
	request := generator.makeRequest("POST.asset-depreciation-run", nil, "")
	if response := generator.send(request, accounting.PostAssetDepreciationRun); response.StatusCode != 200 {
		return core.Err("no se pudo correr la depreciación:", response.Error)
	}
	return nil
}
