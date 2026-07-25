package types

import "github.com/ivanjoz/genix-orm/scylla"

type ProductSupplyProviderRow struct {
	ProviderID   int32 `json:",omitempty"`
	Capacity     int32 `json:",omitempty"`
	DeliveryTime int16 `json:",omitempty"`
	Price        int32 `json:",omitempty"`
}

type ProductSupply struct {
	scylla.TableStruct[ProductSupplyTable, ProductSupply]
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
	scylla.TableStruct[ProductSupplyTable, ProductSupply]
	CompanyID            scylla.Col[ProductSupplyTable, int32]
	ProductID            scylla.Col[ProductSupplyTable, int32]
	MinimunStock         scylla.Col[ProductSupplyTable, int32]
	SalesPerDayEstimated scylla.Col[ProductSupplyTable, int32]
	ProviderSupply       scylla.Col[ProductSupplyTable, []ProductSupplyProviderRow]
	Status               scylla.Col[ProductSupplyTable, int8]
	Updated              scylla.Col[ProductSupplyTable, int32]
	UpdatedBy            scylla.Col[ProductSupplyTable, int32]
}

func (productSupplyTable ProductSupplyTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "product_supply",
		Partition: productSupplyTable.CompanyID,
		Keys:      scylla.Cols(productSupplyTable.ProductID),
		Indexes: []scylla.Index{
			{Type: scylla.TypeView, Keys: scylla.Cols(productSupplyTable.Status)},
			{Type: scylla.TypeView, Keys: scylla.Cols(productSupplyTable.Updated)},
		},
	}
}
