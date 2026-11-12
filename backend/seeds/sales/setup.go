package sales

import (
	"app/core"
	"app/db"
	salesModule "app/sales"
	salesTypes "app/sales/types"
	"app/seeds/runner"
	"encoding/json"
	"slices"
	"time"
)

var Module = runner.Module{Name: "sales", Requires: []string{"logistics"}, Setup: Setup}

// Setup writes the sales planning and the delivery price list, then hands the day its two steps:
// the day's sales (invoiced as boletas and facturas, collected in the store's till) and, in the
// evening, the annulment of the ones flagged for it. Sales require logistics: without its purchase
// orders the stores would sell out on the first day.
func Setup(run *runner.Run) ([]runner.DayStep, error) {
	if err := ensureSalesPlanning(run); err != nil {
		return nil, core.Err("No se pudo crear la planificación de ventas:", err)
	}
	if err := saveDemoShippingCosts(run); err != nil {
		return nil, core.Err("No se pudieron guardar los costos de envío:", err)
	}
	return []runner.DayStep{
		{Name: "sales", Order: runner.OrderSales, Run: func(dayStart time.Time) error { return runDailySales(run, dayStart) }},
		{Name: "annulments", Order: runner.OrderAnnulments, Run: func(dayStart time.Time) error { return runAnnulmentsStep(run, dayStart) }},
	}, nil
}

// demoShippingCosts is the delivery price list for the Lima districts the stores serve: a flat fee
// plus a per-kilo charge, both in cents, rising with the distance from the stores.
var demoShippingCosts = []salesTypes.ShippingCost{
	{CityID: 150141, FlatAmount: 500, AmountPerKg: 50},   // Surquillo
	{CityID: 150122, FlatAmount: 600, AmountPerKg: 50},   // Miraflores
	{CityID: 150131, FlatAmount: 600, AmountPerKg: 50},   // San Isidro
	{CityID: 150130, FlatAmount: 700, AmountPerKg: 60},   // San Borja
	{CityID: 150140, FlatAmount: 700, AmountPerKg: 60},   // Santiago de Surco
	{CityID: 150104, FlatAmount: 800, AmountPerKg: 60},   // Barranco
	{CityID: 150116, FlatAmount: 800, AmountPerKg: 60},   // Lince
	{CityID: 150113, FlatAmount: 900, AmountPerKg: 70},   // Jesús María
	{CityID: 150115, FlatAmount: 900, AmountPerKg: 70},   // La Victoria
	{CityID: 150134, FlatAmount: 900, AmountPerKg: 70},   // San Luis
	{CityID: 150120, FlatAmount: 1000, AmountPerKg: 80},  // Magdalena del Mar
	{CityID: 150121, FlatAmount: 1000, AmountPerKg: 80},  // Pueblo Libre
	{CityID: 150108, FlatAmount: 1000, AmountPerKg: 80},  // Chorrillos
	{CityID: 150101, FlatAmount: 1100, AmountPerKg: 80},  // Lima
	{CityID: 150136, FlatAmount: 1200, AmountPerKg: 90},  // San Miguel
	{CityID: 150114, FlatAmount: 1200, AmountPerKg: 90},  // La Molina
	{CityID: 150103, FlatAmount: 1400, AmountPerKg: 100}, // Ate
	{CityID: 150133, FlatAmount: 1400, AmountPerKg: 100}, // San Juan de Miraflores
	{CityID: 150132, FlatAmount: 1600, AmountPerKg: 120}, // San Juan de Lurigancho
	{CityID: 150135, FlatAmount: 1600, AmountPerKg: 120}, // San Martín de Porres
}

// ensureSalesPlanning writes three seasonality curves and a weekly plan for every product of the
// pool, unless the company already has planning.
func ensureSalesPlanning(run *runner.Run) error {
	existingPlanning := []salesTypes.SalesPlanning{}
	planningQuery := db.Query(&existingPlanning)
	planningQuery.Select(planningQuery.ID).CompanyID.Equals(run.UserToken.CompanyID).Limit(1)
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
	request := run.MakeRequest("POST.seasonality-curve", nil, string(bodyBytes))
	savedCurves := []salesTypes.SeasonalityCurve{}
	if err := runner.DecodeResponse(run.Send(request, salesModule.PostSeasonalityCurve), &savedCurves); err != nil {
		return err
	}

	planning := make([]salesTypes.SalesPlanning, 0, len(run.ProductPool))
	for _, productID := range run.ProductPool {
		curve := savedCurves[run.Random.Intn(len(savedCurves))]
		baseQuantity := int32(5 + run.Random.Intn(56))
		productPlanning := salesTypes.SalesPlanning{ProductID: productID, BaseQuantity: baseQuantity, SeasonalityCurveID: curve.ID}
		for _, curveWeek := range curve.Curve {
			productPlanning.WeeklyQuantity = append(productPlanning.WeeklyQuantity, salesTypes.SalesPlanningWeek{
				Week: curveWeek.Week, Quantity: int16(baseQuantity * int32(curveWeek.Percent) / 1000),
			})
		}
		planning = append(planning, productPlanning)
	}
	for batchStart := 0; batchStart < len(planning); batchStart += runner.PostBatchSize {
		batchBytes, err := json.Marshal(planning[batchStart:min(batchStart+runner.PostBatchSize, len(planning))])
		if err != nil {
			return err
		}
		batchRequest := run.MakeRequest("POST.sales-planning", nil, string(batchBytes))
		if response := salesModule.PostSalesPlanning(&batchRequest); response.StatusCode != 200 {
			return core.Err(response.Error)
		}
	}
	core.Log("Seed:: planificación de ventas", "curvas", len(savedCurves), "productos", len(planning))
	return nil
}

// saveDemoShippingCosts writes the Lima delivery price list. The handler merges by district, so a
// rerun with the same prices writes nothing.
func saveDemoShippingCosts(run *runner.Run) error {
	shippingCosts := slices.Clone(demoShippingCosts)
	for costIndex := range shippingCosts {
		shippingCosts[costIndex].HasUpdated = true
	}
	bodyBytes, err := json.Marshal(shippingCosts)
	if err != nil {
		return err
	}
	request := run.MakeRequest("POST.shipping-costs", nil, string(bodyBytes))
	if response := run.Send(request, salesModule.PostShippingCosts); response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	return nil
}
