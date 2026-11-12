// Package base seeds what every other module writes against, and loads it into the run: the HR
// legal parameters (global), the stores' warehouses and tills, the bank account, the default invoice
// series, clients and providers, the product catalog from products.xlsx and the supplies catalog.
// Every step matches what is already stored, so it runs again on every invocation and only fills
// what is missing.
package base

import (
	"app/core"
	"app/seeds/runner"
)

// bankName is the current account suppliers, expenses and payroll are paid from.
const bankName = "BCP Cta. Cte. Soles"

var Module = runner.Module{Name: "base", Setup: Setup}

// Setup writes the missing catalog and loads the run's company context.
func Setup(run *runner.Run) ([]runner.DayStep, error) {
	if err := seedHRLegalParameters(run); err != nil {
		return nil, core.Err("parámetros laborales:", err)
	}
	return nil, SetupCompany(run)
}

// SetupCompany is base without the legal parameters, which are per country rather than per company.
// The load test runs it over HTTP to onboard each of its companies, many at once, and would race on
// the legal parameters (it never runs a payroll).
func SetupCompany(run *runner.Run) error {
	if err := checkAdminUser(run); err != nil {
		return err
	}
	if err := ensureWarehouses(run); err != nil {
		return core.Err("No se pudieron preparar los almacenes:", err)
	}
	if err := ensureCashAccounts(run); err != nil {
		return core.Err("No se pudieron preparar las cajas:", err)
	}
	if err := seedProvidersAndClients(run); err != nil {
		return core.Err("No se pudieron sembrar proveedores/clientes:", err)
	}
	if err := resolveInvoiceSeries(run); err != nil {
		return core.Err("No se pudieron resolver las series de facturación:", err)
	}
	// The catalog comes before the supplies and the pool: both read the products it creates.
	productsImport, err := importProductsWorkbook(run)
	if err != nil {
		return core.Err("No se pudieron importar los productos de products.xlsx:", err)
	}
	core.Log("Seed:: catálogo", "creados", productsImport.Created, "existentes", productsImport.SkippedExisting,
		"categoríasNuevas", productsImport.NewCategories, "marcasNuevas", productsImport.NewBrands)
	if err := seedSupplyProviders(run); err != nil {
		return core.Err("No se pudieron sembrar los proveedores de abastecimiento:", err)
	}
	if err := seedSupplyMaterials(run); err != nil {
		return core.Err("No se pudieron generar los insumos:", err)
	}
	if err := loadProductPool(run); err != nil {
		return core.Err("No se pudo armar el pool de productos:", err)
	}
	for _, warehouseID := range run.Config.WarehouseIDs {
		if err := run.ReloadWarehouseLedger(warehouseID); err != nil {
			return core.Err("No se pudo cargar el stock del almacén", warehouseID, ":", err)
		}
	}
	return nil
}
