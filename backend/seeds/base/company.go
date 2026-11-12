package base

import (
	"app/business"
	businessTypes "app/business/types"
	"app/cloud"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/finance"
	financeTypes "app/finance/types"
	"app/invoicing"
	invoicingTypes "app/invoicing/types"
	production "app/production/types"
	"app/seeds/runner"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

// checkAdminUser fails early when fn-init has not run for the company: every request is sent as
// its admin.
func checkAdminUser(run *runner.Run) error {
	users := []coreTypes.User{}
	userQuery := db.Query(&users)
	userQuery.Select(userQuery.ID, userQuery.Status).
		CompanyID.Equals(run.UserToken.CompanyID).ID.Equals(runner.UserID).Limit(1)
	if err := userQuery.Exec(); err != nil {
		return core.Err("error al consultar el usuario:", err)
	}
	if len(users) == 0 {
		return core.Err("no existe el usuario", runner.UserID, "en la empresa", run.UserToken.CompanyID, "; corre primero fn-init")
	}
	return nil
}

// loadProductPool fixes the working catalog once: the same products are reused every simulated
// day, which is what makes the generated history look like a real business restocking its shelves.
func loadProductPool(run *runner.Run) error {
	products := []production.Product{}
	productQuery := db.Query(&products)
	productQuery.Select(productQuery.ID, productQuery.Name, productQuery.Price,
		productQuery.FinalPrice, productQuery.Status).
		CompanyID.Equals(run.UserToken.CompanyID).Status.Equals(1)
	if err := productQuery.Exec(); err != nil {
		return core.Err("error al consultar los productos:", err)
	}

	// Only products the sale handler will price above zero. It recomputes every line from
	// FinalPrice and ignores the prices sent in the payload, so a product priced at 0 there would
	// book a line that charges nothing.
	eligibleProducts := make([]production.Product, 0, len(products))
	for _, product := range products {
		if product.ID > 0 && product.FinalPrice > 0 {
			eligibleProducts = append(eligibleProducts, product)
		}
	}
	if len(eligibleProducts) < run.Config.ProductPoolSize {
		return core.Err("sólo hay", len(eligibleProducts), "productos activos y se piden",
			run.Config.ProductPoolSize)
	}

	// A resumed run reuses the pool stored in the state file, so the same products keep repeating
	// across days; only a fresh run draws a new one.
	selectedProducts := eligibleProducts
	if len(run.State.ProductPool) > 0 {
		pooledProductIDs := core.SliceSet[int32]{}
		for _, productID := range run.State.ProductPool {
			pooledProductIDs.Add(productID)
		}
		selectedProducts = []production.Product{}
		for _, product := range eligibleProducts {
			if slices.Contains(pooledProductIDs.Values, product.ID) {
				selectedProducts = append(selectedProducts, product)
			}
		}
		if len(selectedProducts) == 0 {
			return core.Err("ninguno de los productos del estado sigue activo; usa --reset")
		}
	} else {
		run.Random.Shuffle(len(selectedProducts), func(i, j int) {
			selectedProducts[i], selectedProducts[j] = selectedProducts[j], selectedProducts[i]
		})
		selectedProducts = selectedProducts[:run.Config.ProductPoolSize]
	}

	run.ProductPool = make([]int32, 0, len(selectedProducts))
	for _, product := range selectedProducts {
		// Exactly what the sale handler will charge, with no fallback: any other number here makes
		// the run's totals disagree with the ones the server recomputes.
		salePrice := product.FinalPrice
		// Buying below the sale price is what leaves a margin for the reports to show.
		costPrice := salePrice * int32(60+run.Random.Intn(21)) / 100
		if costPrice <= 0 {
			costPrice = 1
		}

		run.ProductPool = append(run.ProductPool, product.ID)
		run.ProductNameByID[product.ID] = product.Name
		run.SalePriceByProduct[product.ID] = salePrice
		run.CostPriceByProduct[product.ID] = costPrice
	}

	run.State.ProductPool = run.ProductPool
	core.Log("Seed:: pool de productos", len(run.ProductPool))
	return run.SaveState()
}

// ─── Stock ledger ──────────────────────────────────────────────────────────────

// resolveInvoiceSeries finds the company's active default factura and boleta series, creating the
// missing ones (F001 / B001) through POST.invoice-series. Sales are issued under them, which writes
// their invoice_document; nothing is sent to SUNAT here.
func resolveInvoiceSeries(run *runner.Run) error {
	findDefaults := func(allSeries []invoicingTypes.InvoiceSeries) (int8, int8) {
		facturaSeriesID, boletaSeriesID := int8(0), int8(0)
		for _, series := range allSeries {
			if series.Status != 1 || series.IsDefault != 1 {
				continue
			}
			switch series.DocType {
			case invoicingTypes.DocTypeFactura:
				facturaSeriesID = series.SeriesID
			case invoicingTypes.DocTypeBoleta:
				boletaSeriesID = series.SeriesID
			}
		}
		return facturaSeriesID, boletaSeriesID
	}

	storedSeries, err := invoicing.LoadCompanySeries(run.UserToken.CompanyID)
	if err != nil {
		return err
	}
	facturaSeriesID, boletaSeriesID := findDefaults(storedSeries)
	for _, missingSeries := range []invoicingTypes.InvoiceSeries{
		{DocType: invoicingTypes.DocTypeFactura, SeriesCode: "F001", IsDefault: 1},
		{DocType: invoicingTypes.DocTypeBoleta, SeriesCode: "B001", IsDefault: 1},
	} {
		isMissing := (missingSeries.DocType == invoicingTypes.DocTypeFactura && facturaSeriesID == 0) ||
			(missingSeries.DocType == invoicingTypes.DocTypeBoleta && boletaSeriesID == 0)
		if !isMissing {
			continue
		}
		bodyBytes, err := json.Marshal(missingSeries)
		if err != nil {
			return err
		}
		request := run.MakeRequest("POST.invoice-series", nil, string(bodyBytes))
		savedSeries := map[string][]invoicingTypes.InvoiceSeries{}
		if err := runner.DecodeResponse(run.Send(request, invoicing.PostInvoiceSeries), &savedSeries); err != nil {
			return err
		}
		facturaSeriesID, boletaSeriesID = findDefaults(savedSeries["InvoiceSeries"])
	}
	if facturaSeriesID == 0 || boletaSeriesID == 0 {
		return core.Err("la empresa no tiene una serie activa por defecto de factura y de boleta")
	}

	// The sale handler reads the series from the cached company config, which may lag a fresh
	// write; wait until both are visible there rather than failing the first sales.
	for attempt := 0; attempt < 30; attempt++ {
		companyConfig, err := cloud.LoadCompanyConfig(run.UserToken.CompanyID)
		if err != nil {
			return err
		}
		if invoicingTypes.FindSeries(companyConfig.Sunat.Series, facturaSeriesID) != nil &&
			invoicingTypes.FindSeries(companyConfig.Sunat.Series, boletaSeriesID) != nil {
			run.FacturaSeriesID, run.BoletaSeriesID = facturaSeriesID, boletaSeriesID
			core.Log("Seed:: series de facturación", "factura", facturaSeriesID, "boleta", boletaSeriesID)
			return nil
		}
		time.Sleep(time.Second)
	}
	return core.Err("las series de facturación no aparecen en la configuración de la empresa")
}

// ensureWarehouses creates the stores' warehouses a fresh company lacks: fn-init seeds only the
// first one. The new warehouse sits on the first one's site and must come out with the ID the
// configuration names, or the history would buy and sell into the wrong place.
func ensureWarehouses(run *runner.Run) error {
	warehouses := []businessTypes.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.SiteID, warehouseQuery.Status).
		CompanyID.Equals(run.UserToken.CompanyID)
	if err := warehouseQuery.Exec(); err != nil {
		return err
	}
	existingWarehouseIDs := core.SliceSet[int32]{}
	firstSiteID := int32(0)
	for _, warehouse := range warehouses {
		if warehouse.Status != 1 && slices.Contains(run.Config.WarehouseIDs, warehouse.ID) {
			return core.Err("el almacén", warehouse.ID, "está inactivo")
		}
		existingWarehouseIDs.Add(warehouse.ID)
		if warehouse.ID == run.Config.WarehouseIDs[0] {
			firstSiteID = warehouse.SiteID
		}
	}
	if firstSiteID == 0 {
		return core.Err("no existe el almacén", run.Config.WarehouseIDs[0], "; corre primero fn-init")
	}

	for warehouseIndex, warehouseID := range run.Config.WarehouseIDs {
		if existingWarehouseIDs.Include(warehouseID) {
			continue
		}
		bodyBytes, err := json.Marshal(businessTypes.Warehouse{
			SiteID: firstSiteID, Name: fmt.Sprint("Almacén Tienda ", warehouseIndex+1), Status: 1,
		})
		if err != nil {
			return err
		}
		request := run.MakeRequest("POST.warehouses", nil, string(bodyBytes))
		createdWarehouse := businessTypes.Warehouse{}
		if err := runner.DecodeResponse(run.Send(request, business.PostWarehouse), &createdWarehouse); err != nil {
			return err
		}
		if createdWarehouse.ID != warehouseID {
			return core.Err("el almacén creado tiene ID", createdWarehouse.ID, "y se esperaba", warehouseID)
		}
	}
	return nil
}

// ensureCashAccounts gives each store its own till and the company a bank current account,
// creating the missing ones by name. The first store keeps the company's first till. Suppliers and
// expenses are paid from the bank; each store's sales are collected in its till.
func ensureCashAccounts(run *runner.Run) error {
	cashBanks := []financeTypes.CashBank{}
	cashBankQuery := db.Query(&cashBanks)
	cashBankQuery.Select(cashBankQuery.ID, cashBankQuery.Name, cashBankQuery.Type,
		cashBankQuery.CurrencyType, cashBankQuery.Status).
		CompanyID.Equals(run.UserToken.CompanyID).Status.Equals(1)
	if err := cashBankQuery.Exec(); err != nil {
		return err
	}
	// The history is in soles, and a sale must be collected in its own currency. The lowest soles
	// account other than the bank is the first store's till (fn-init's cash register).
	cashBankIDByName, firstTillID := map[string]int32{}, int32(0)
	for _, cashBank := range cashBanks {
		if cashBank.CurrencyType != financeTypes.CurrencyPEN {
			continue
		}
		cashBankIDByName[cashBank.Name] = cashBank.ID
		if cashBank.Name != bankName && (firstTillID == 0 || cashBank.ID < firstTillID) {
			firstTillID = cashBank.ID
		}
	}
	if firstTillID == 0 {
		return core.Err("no hay ninguna caja activa en soles en la empresa", run.UserToken.CompanyID)
	}

	warehouses := []businessTypes.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.SiteID).
		CompanyID.Equals(run.UserToken.CompanyID).ID.In(run.Config.WarehouseIDs...)
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
		request := run.MakeRequest("POST.cash-banks", nil, string(bodyBytes))
		createdCashBank := financeTypes.CashBank{}
		if err := runner.DecodeResponse(run.Send(request, finance.PostCashBanks), &createdCashBank); err != nil {
			return 0, core.Err("no se pudo crear la caja", name, ":", err)
		}
		return createdCashBank.ID, nil
	}

	for warehouseIndex, warehouseID := range run.Config.WarehouseIDs {
		if warehouseIndex == 0 {
			run.TillByWarehouse[warehouseID] = firstTillID
			continue
		}
		tillID, err := ensureCashBank(fmt.Sprint("Caja Tienda ", warehouseIndex+1), 1, siteByWarehouse[warehouseID])
		if err != nil {
			return err
		}
		run.TillByWarehouse[warehouseID] = tillID
	}
	bankID, err := ensureCashBank(bankName, 2, siteByWarehouse[run.Config.WarehouseIDs[0]])
	if err != nil {
		return err
	}
	run.BankCashBankID = bankID
	core.Log("Seed:: cajas", "tiendas", run.TillByWarehouse, "banco", bankID)
	return nil
}
