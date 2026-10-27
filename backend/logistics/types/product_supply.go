package types

import "app/db"

type ProductSupplyProviderRow struct {
	ProviderID   int32 `json:",omitempty" cb:"ProviderID,1"`
	Capacity     int32 `json:",omitempty" cb:"Capacity,2"`
	DeliveryTime int16 `json:",omitempty" cb:"DeliveryTime,3"`
	Price        int32 `json:",omitempty" cb:"Price,4"`
}

type ProductSupply struct {
	db.TableStruct[ProductSupplyTable, ProductSupply]
	CompanyID            int32                      `json:",omitempty" cb:"CompanyID,1"`
	ProductID            int32                      `json:",omitempty" cb:"ProductID,2"`
	MinimunStock         int32                      `json:",omitempty" cb:"MinimunStock,3"`
	SalesPerDayEstimated int32                      `json:",omitempty" cb:"SalesPerDayEstimated,4"`
	ProviderSupply       []ProductSupplyProviderRow `json:",omitempty" cb:"ProviderSupply,5"`
	Status               int8                       `json:"ss,omitempty" cb:"ss,6"`
	Updated              int32                      `json:"upd,omitempty" cb:"upd,7"`
	UpdatedBy            int32                      `json:",omitempty" cb:"UpdatedBy,8"`
}

type ProductSupplyTable struct {
	db.TableStruct[ProductSupplyTable, ProductSupply]
	CompanyID            db.Col[*ProductSupplyTable, int32]
	ProductID            db.Col[*ProductSupplyTable, int32]
	MinimunStock         db.Col[*ProductSupplyTable, int32]
	SalesPerDayEstimated db.Col[*ProductSupplyTable, int32]
	ProviderSupply       db.Col[*ProductSupplyTable, []ProductSupplyProviderRow]
	Status               db.Col[*ProductSupplyTable, int8]
	Updated              db.Col[*ProductSupplyTable, int32]
	UpdatedBy            db.Col[*ProductSupplyTable, int32]
}

func (productSupplyTable ProductSupplyTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        21,
		Name:      "product_supply",
		Partition: productSupplyTable.CompanyID,
		Keys:      db.Cols(productSupplyTable.ProductID),
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: db.Cols(productSupplyTable.Status)},
			{Type: db.TypeView, Keys: db.Cols(productSupplyTable.Updated)},
		},
	}
}
