package types

import "app/db"

type ShippingCost struct {
	db.TableStruct[ShippingCostTable, ShippingCost]
	CompanyID int32   `json:",omitempty"`
	CityID    int32   `json:",omitempty"`
	FlatCost  float64 `json:",omitempty"`
	CostPerKg float64 `json:",omitempty"`
	Updated   int32   `json:"upd,omitempty"`
	UpdatedBy int32   `json:",omitempty"`
	Created   int32   `json:",omitempty"`
	CreatedBy int32   `json:",omitempty"`
	// HasUpdated is request-only so the frontend can send a sparse batch without creating a DB column.
	HasUpdated bool `db:"-" json:"hasUpdated,omitempty"`
}

type ShippingCostTable struct {
	db.TableStruct[ShippingCostTable, ShippingCost]
	CompanyID db.Col[ShippingCostTable, int32]
	CityID    db.Col[ShippingCostTable, int32]
	FlatCost  db.Col[ShippingCostTable, float64]
	CostPerKg db.Col[ShippingCostTable, float64]
	Updated   db.Col[ShippingCostTable, int32]
	UpdatedBy db.Col[ShippingCostTable, int32]
	Created   db.Col[ShippingCostTable, int32]
	CreatedBy db.Col[ShippingCostTable, int32]
}

func (e ShippingCostTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "shipping_costs",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.CityID},
		Indexes: []db.Index{
			// Delta-cache fetches query by company partition and updated watermark.
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
