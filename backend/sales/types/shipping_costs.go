package types

import "app/db"

type ShippingCost struct {
	db.TableStruct[ShippingCostTable, ShippingCost]
	CompanyID int32 `cb:"1"`
	// CityID is a department, province or district: a rate set on a level applies to every
	// district under it that has none of its own (ResolveShippingAmount).
	CityID int32 `cb:"2"`
	// FlatAmount is the delivery charge in cents. 0 means "not set" and falls back to the parent.
	// The columns replaced the float64 FlatCost/CostPerKg (soles), which the ORM cannot retype in
	// place; exec fn-shipping-costs-to-cents copied the old values over.
	FlatAmount int32 `cb:"3"`
	// AmountPerKg is cents per kg. Products carry no weight yet, so the checkout does not apply it.
	AmountPerKg int32 `cb:"4"`
	Updated     int32 `json:"upd" cb:"upd,5"`
	Version     int16 `json:"ver" cb:"ver,6"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,7"`
	UpdatedBy int32   `cb:"8"`
	Created   int32   `cb:"9"`
	CreatedBy int32   `cb:"10"`
	// HasUpdated is request-only so the frontend can send a sparse batch without creating a DB column.
	HasUpdated bool `db:"-" json:"hasUpdated" cb:"hasUpdated,11"`
}

type ShippingCostTable struct {
	db.TableStruct[ShippingCostTable, ShippingCost]
	CompanyID   db.Col[*ShippingCostTable, int32]
	CityID      db.Col[*ShippingCostTable, int32]
	FlatAmount  db.Col[*ShippingCostTable, int32]
	AmountPerKg db.Col[*ShippingCostTable, int32]
	Updated     db.Col[*ShippingCostTable, int32]
	Version     db.Col[*ShippingCostTable, int16]
	UpdatedBy   db.Col[*ShippingCostTable, int32]
	Created     db.Col[*ShippingCostTable, int32]
	CreatedBy   db.Col[*ShippingCostTable, int32]
}

func (e ShippingCostTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        30,
		Name:      "shipping_costs",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.CityID),
		Indexes: []db.Index{
			// Keyless: one log per partition and no sync filter. Read with Delta(cursor).
			{Type: db.TypeDelta},
		},
	}
}

// ShippingRateCityIDs are the ubigeo levels a district's rate is looked up at, nearest first: the
// district, its province and its department. The ubigeo is the id, and a district's code holds its
// parents' (150101 sits under 1501 under 15).
func ShippingRateCityIDs(districtID int32) []int32 {
	return []int32{districtID, districtID / 100, districtID / 10000}
}

// ResolveShippingAmount is the flat delivery charge for a district: the nearest level of
// ShippingRateCityIDs with a rate above 0. found is false when none has one, and then the store
// does not deliver there.
func ResolveShippingAmount(districtID int32, ratesByCityID map[int32]int32) (amount int32, found bool) {
	for _, cityID := range ShippingRateCityIDs(districtID) {
		if rate := ratesByCityID[cityID]; rate > 0 {
			return rate, true
		}
	}
	return 0, false
}
