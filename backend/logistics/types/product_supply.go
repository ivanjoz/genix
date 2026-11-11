package types

import "app/db"

type ProductSupplyProviderRow struct {
	ProviderID   int32 `cb:"1"`
	Capacity     int32 `cb:"2"`
	DeliveryTime int16 `cb:"3"`
	Price        int32 `cb:"4"`
}

type ProductSupply struct {
	db.TableStruct[ProductSupplyTable, ProductSupply]
	CompanyID            int32                      `cb:"1"`
	ProductID            int32                      `cb:"2"`
	MinimunStock         int32                      `cb:"3"`
	SalesPerDayEstimated int32                      `cb:"4"`
	ProviderSupply       []ProductSupplyProviderRow `cb:"5"`
	Status               int8                       `json:"ss" cb:"ss,6"`
	Updated              int32                      `json:"upd" cb:"upd,7"`
	UpdatedBy            int32                      `cb:"8"`
	Version              int16                      `json:"ver" cb:"ver,9"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,10"`
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
	Version              db.Col[*ProductSupplyTable, int16]
}

func (productSupplyTable ProductSupplyTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        21,
		Name:      "product_supply",
		Partition: productSupplyTable.CompanyID,
		Keys:      db.Cols(productSupplyTable.ProductID),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: productSupplyTable.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(productSupplyTable.Status)},
		},
	}
}
