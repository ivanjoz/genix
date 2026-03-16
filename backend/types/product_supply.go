package types

import "app/db"

type ProductSupplyProviderRow struct {
	ProviderID   int32 `json:",omitempty" cbor:"1,keyasint,omitempty"`
	Capacity     int32 `json:",omitempty" cbor:"2,keyasint,omitempty"`
	DeliveryTime int16 `json:",omitempty" cbor:"3,keyasint,omitempty"`
	Price        int32 `json:",omitempty" cbor:"4,keyasint,omitempty"`
}

type ProductSupply struct {
	db.TableStruct[ProductSupplyTable, ProductSupply]
	CompanyID            int32                      `json:",omitempty"`
	ProductID            int32                      `json:",omitempty"`
	MinimunStock         int32                      `json:",omitempty"`
	SalesPerDayEstimated int32                      `json:",omitempty"`
	ProviderSupply       []ProductSupplyProviderRow `json:",omitempty"`
	Status               int8                       `json:"ss,omitempty"`
	Updated              int32                      `json:"upd,omitempty"`
	UpdatedBy            int32                      `json:",omitempty"`
}

type ProductSupplyTable struct {
	db.TableStruct[ProductSupplyTable, ProductSupply]
	CompanyID            db.Col[ProductSupplyTable, int32]
	ProductID            db.Col[ProductSupplyTable, int32]
	MinimunStock         db.Col[ProductSupplyTable, int32]
	SalesPerDayEstimated db.Col[ProductSupplyTable, int32]
	ProviderSupply       db.Col[ProductSupplyTable, []ProductSupplyProviderRow]
	Status               db.Col[ProductSupplyTable, int8]
	Updated              db.Col[ProductSupplyTable, int32]
	UpdatedBy            db.Col[ProductSupplyTable, int32]
}

func (productSupplyTable ProductSupplyTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "product_supply",
		Partition: productSupplyTable.CompanyID,
		Keys:      []db.Coln{productSupplyTable.ProductID},
		Views: []db.View{
			{Cols: []db.Coln{productSupplyTable.Status}, KeepPart: true},
			{Cols: []db.Coln{productSupplyTable.Updated}, KeepPart: true},
		},
	}
}
