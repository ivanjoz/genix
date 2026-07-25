package types

import "github.com/ivanjoz/genix-orm/scylla"

type ShippingCost struct {
	scylla.TableStruct[ShippingCostTable, ShippingCost]
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
	scylla.TableStruct[ShippingCostTable, ShippingCost]
	CompanyID scylla.Col[ShippingCostTable, int32]
	CityID    scylla.Col[ShippingCostTable, int32]
	FlatCost  scylla.Col[ShippingCostTable, float64]
	CostPerKg scylla.Col[ShippingCostTable, float64]
	Updated   scylla.Col[ShippingCostTable, int32]
	UpdatedBy scylla.Col[ShippingCostTable, int32]
	Created   scylla.Col[ShippingCostTable, int32]
	CreatedBy scylla.Col[ShippingCostTable, int32]
}

func (e ShippingCostTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "shipping_costs",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.CityID),
		Indexes: []scylla.Index{
			// Delta-cache fetches query by company partition and updated watermark.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}
