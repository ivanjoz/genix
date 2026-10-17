package types

import "app/db"

type ShippingCost struct {
	db.TableStruct[ShippingCostTable, ShippingCost]
	CompanyID int32   `json:",omitempty"`
	CityID    int32   `json:",omitempty"`
	FlatCost  float64 `json:",omitempty"`
	CostPerKg float64 `json:",omitempty"`
	Updated   int32   `json:"upd,omitempty"`
	Version   int16   `json:"ver,omitempty"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt,omitempty"`
	UpdatedBy int32   `json:",omitempty"`
	Created   int32   `json:",omitempty"`
	CreatedBy int32   `json:",omitempty"`
	// HasUpdated is request-only so the frontend can send a sparse batch without creating a DB column.
	HasUpdated bool `db:"-" json:"hasUpdated,omitempty"`
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
