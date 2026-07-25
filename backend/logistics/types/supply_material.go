package types

import "github.com/ivanjoz/genix-orm/scylla"

// SupplyMaterial is a catalog item (raw material / consumable / packaging) that
// can be purchased from one or more providers. Distinct from ProductSupply,
// which describes a product's supplier relationships.
type SupplyMaterial struct {
	scylla.TableStruct[SupplyMaterialTable, SupplyMaterial]
	CompanyID      int32                      `json:",omitempty"`
	ID             int32                      `json:",omitempty"`
	Name           string                     `json:",omitempty"`
	Description    string                     `json:",omitempty"`
	BrandID        int32                      `json:",omitempty"`
	Price          int32                      `json:",omitempty"`
	CurrencyID     int16                      `json:",omitempty"`
	SKU            string                     `json:",omitempty"`
	MinimunStock   int32                      `json:",omitempty"`
	ProviderSupply []ProductSupplyProviderRow `json:",omitempty"`
	Status         int8                       `json:"ss,omitempty"`
	Updated        int32                      `json:"upd,omitempty"`
	UpdatedBy      int32                      `json:",omitempty"`
	Created        int32                      `json:",omitempty"`
	CreatedBy      int32                      `json:",omitempty"`
}

type SupplyMaterialTable struct {
	scylla.TableStruct[SupplyMaterialTable, SupplyMaterial]
	CompanyID      scylla.Col[SupplyMaterialTable, int32]
	ID             scylla.Col[SupplyMaterialTable, int32]
	Name           scylla.Col[SupplyMaterialTable, string]
	Description    scylla.Col[SupplyMaterialTable, string]
	BrandID        scylla.Col[SupplyMaterialTable, int32]
	Price          scylla.Col[SupplyMaterialTable, int32]
	CurrencyID     scylla.Col[SupplyMaterialTable, int16]
	SKU            scylla.Col[SupplyMaterialTable, string]
	MinimunStock   scylla.Col[SupplyMaterialTable, int32]
	ProviderSupply scylla.Col[SupplyMaterialTable, []ProductSupplyProviderRow]
	Status         scylla.Col[SupplyMaterialTable, int8]
	Updated        scylla.Col[SupplyMaterialTable, int32]
	UpdatedBy      scylla.Col[SupplyMaterialTable, int32]
	Created        scylla.Col[SupplyMaterialTable, int32]
	CreatedBy      scylla.Col[SupplyMaterialTable, int32]
}

func (e SupplyMaterialTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "supply_material",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			// Status view → list active supplies cheaply.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status.DecimalSize(1))},
			// Updated view → delta-cache watermark sync from the frontend.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated.DecimalSize(10))},
			// SKU lookup within a company (e.g. dedup during create).
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.SKU)},
		},
	}
}
