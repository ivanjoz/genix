package exec

import (
	"app/core"
	"app/db"
	salesTypes "app/sales/types"
	"math"
)

// fn-shipping-costs-to-cents
//
// ShippingCost moved from float64 soles (flat_cost, cost_per_kg) to int32 cents (flat_amount,
// amount_per_kg). The ORM refuses to retype a column, so the new columns were added beside the old
// ones and this copies every row across: round(soles * 100). Only rows whose new columns are still
// empty are written, so a second run changes nothing. The old columns stay, unread.
func MigrateShippingCostsToCents(args *core.ExecArgs) core.FuncResponse {
	db.Connect(MakeConnParams())

	type legacyShippingCost struct {
		CompanyID int32   `db:"company_id"`
		CityID    int32   `db:"city_id"`
		FlatCost  float64 `db:"flat_cost"`
		CostPerKg float64 `db:"cost_per_kg"`
	}
	legacyRows, err := db.QuerySQL[legacyShippingCost](
		"SELECT company_id, city_id, COALESCE(flat_cost, 0) AS flat_cost, COALESCE(cost_per_kg, 0) AS cost_per_kg FROM " +
			db.SQLTable[salesTypes.ShippingCost]() +
			" WHERE (flat_amount IS NULL OR flat_amount = 0) AND (amount_per_kg IS NULL OR amount_per_kg = 0)")
	if err != nil {
		return args.MakeErr("No se pudieron leer los costos de envío anteriores:", err)
	}

	migratedRows := []salesTypes.ShippingCost{}
	for _, legacyRow := range legacyRows {
		if legacyRow.FlatCost == 0 && legacyRow.CostPerKg == 0 {
			continue
		}
		migratedRows = append(migratedRows, salesTypes.ShippingCost{
			CompanyID:   legacyRow.CompanyID,
			CityID:      legacyRow.CityID,
			FlatAmount:  int32(math.Round(legacyRow.FlatCost * 100)),
			AmountPerKg: int32(math.Round(legacyRow.CostPerKg * 100)),
			Updated:     core.SUnixTime(),
		})
	}
	core.Log("MigrateShippingCostsToCents:: filas a migrar", len(migratedRows), "de", len(legacyRows))
	if len(migratedRows) == 0 {
		return core.FuncResponse{Message: "ok: nada que migrar"}
	}

	shippingCostTable := db.TableOf[salesTypes.ShippingCost]()
	if err := db.Update(&migratedRows, shippingCostTable.FlatAmount, shippingCostTable.AmountPerKg,
		shippingCostTable.Updated); err != nil {
		return args.MakeErr("No se pudieron guardar los costos de envío en céntimos:", err)
	}
	return core.FuncResponse{Message: core.Concat(" ", "ok:", len(migratedRows), "costos migrados")}
}
