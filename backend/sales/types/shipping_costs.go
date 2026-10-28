package types

import "app/db"

type ShippingCost struct {
	db.TableStruct[ShippingCostTable, ShippingCost]
	CompanyID int32   `cb:"1"`
	CityID    int32   `cb:"2"`
	FlatCost  float64 `cb:"3"`
	CostPerKg float64 `cb:"4"`
	Updated   int32   `json:"upd" cb:"upd,5"`
	Version   int16   `json:"ver" cb:"ver,6"`
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
	CompanyID db.Col[*ShippingCostTable, int32]
	CityID    db.Col[*ShippingCostTable, int32]
	FlatCost  db.Col[*ShippingCostTable, float64]
	CostPerKg db.Col[*ShippingCostTable, float64]
	Updated   db.Col[*ShippingCostTable, int32]
	Version   db.Col[*ShippingCostTable, int16]
	UpdatedBy db.Col[*ShippingCostTable, int32]
	Created   db.Col[*ShippingCostTable, int32]
	CreatedBy db.Col[*ShippingCostTable, int32]
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
