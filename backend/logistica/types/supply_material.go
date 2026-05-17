package types

import "app/db"

// SupplyMaterial is a catalog item (raw material / consumable / packaging) that
// can be purchased from one or more providers. Distinct from ProductSupply,
// which describes a product's supplier relationships.
type SupplyMaterial struct {
	db.TableStruct[SupplyMaterialTable, SupplyMaterial]
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
	db.TableStruct[SupplyMaterialTable, SupplyMaterial]
	CompanyID      db.Col[SupplyMaterialTable, int32]
	ID             db.Col[SupplyMaterialTable, int32]
	Name           db.Col[SupplyMaterialTable, string]
	Description    db.Col[SupplyMaterialTable, string]
	BrandID        db.Col[SupplyMaterialTable, int32]
	Price          db.Col[SupplyMaterialTable, int32]
	CurrencyID     db.Col[SupplyMaterialTable, int16]
	SKU            db.Col[SupplyMaterialTable, string]
	MinimunStock   db.Col[SupplyMaterialTable, int32]
	ProviderSupply db.Col[SupplyMaterialTable, []ProductSupplyProviderRow]
	Status         db.Col[SupplyMaterialTable, int8]
	Updated        db.Col[SupplyMaterialTable, int32]
	UpdatedBy      db.Col[SupplyMaterialTable, int32]
	Created        db.Col[SupplyMaterialTable, int32]
	CreatedBy      db.Col[SupplyMaterialTable, int32]
}

func (e SupplyMaterialTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "supply_material",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			// Status view → list active supplies cheaply.
			{Type: db.TypeView, Keys: []db.Coln{e.Status.DecimalSize(1)}, KeepPart: true},
			// Updated view → delta-cache watermark sync from the frontend.
			{Type: db.TypeView, Keys: []db.Coln{e.Updated.DecimalSize(10)}, KeepPart: true},
			// SKU lookup within a company (e.g. dedup during create).
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.SKU}},
		},
	}
}
