package types

import "app/db"

type ProductSupplyProviderRow struct {
	ProviderID   int32 `cb:"ProviderID,1"`
	Capacity     int32 `cb:"Capacity,2"`
	DeliveryTime int16 `cb:"DeliveryTime,3"`
	Price        int32 `cb:"Price,4"`
}

type ProductSupply struct {
	db.TableStruct[ProductSupplyTable, ProductSupply]
	CompanyID            int32                      `cb:"CompanyID,1"`
	ProductID            int32                      `cb:"ProductID,2"`
	MinimunStock         int32                      `cb:"MinimunStock,3"`
	SalesPerDayEstimated int32                      `cb:"SalesPerDayEstimated,4"`
	ProviderSupply       []ProductSupplyProviderRow `cb:"ProviderSupply,5"`
	Status               int8                       `json:"ss" cb:"ss,6"`
	Updated              int32                      `json:"upd" cb:"upd,7"`
	UpdatedBy            int32                      `cb:"UpdatedBy,8"`
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
