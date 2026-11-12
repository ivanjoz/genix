// Package logistics seeds the replenishment settings of the catalog and, day by day, one purchase
// order per warehouse walked through create → confirm → supplier factura → pay → reception.
package logistics

import (
	"app/core"
	"app/seeds/runner"
	"time"
)

var Module = runner.Module{Name: "logistics", Setup: Setup}

// Setup writes the replenishment configuration (minimum stock, providers and their quotes) of the
// first products. product_supply is keyed by ProductID and written through db.Merge, so a rerun
// overwrites the same rows.
func Setup(run *runner.Run) ([]runner.DayStep, error) {
	products, err := loadReplenishedProducts(run)
	if err != nil {
		return nil, core.Err("No se pudieron cargar los productos:", err)
	}
	if err := postSupplyBatches(run, makeSupplyRecords(run, products)); err != nil {
		return nil, core.Err("No se pudo guardar el abastecimiento:", err)
	}
	core.Log("Seed:: abastecimiento configurado", "productos", len(products))
	return []runner.DayStep{{Name: "purchase-orders", Order: runner.OrderPurchases,
		Run: func(dayStart time.Time) error { return RunPurchaseOrdersStep(run, dayStart) }}}, nil
}
