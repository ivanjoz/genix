package types

import (
	"app/core"
	"app/db"
)

type ProductStock struct {
	db.TableStruct[ProductStockTable, ProductStock]
	CompanyID      int32 `json:",omitempty"`
	ID             string
	WarehouseID    int32  `json:",omitempty"`
	ProductID      int32  `json:",omitempty"`
	PresentationID int16  `json:",omitempty"`
	SKU            string `json:",omitempty"`
	Lote           string `json:",omitempty"`
	Quantity       int32  `json:",omitempty"`
	SubQuantity    int32  `json:",omitempty"`
	Updated        int32  `json:"upd,omitempty"`
	UpdatedBy      int32  `json:",omitempty"`
	Status         int8   `json:"ss,omitempty"`

	IsWarehouseProductStatus int8  `json:",omitempty"`
	WarehouseProductQuantity int32 `json:",omitempty"`
	StockComputedFecha       int16 `json:",omitempty"`
	StockComputedQuantity    int32 `json:",omitempty"`
	IsNewRecord              bool  `json:",omitempty"`
}

func (e *ProductStock) SelfParse() {
	e.Status = core.If(e.Quantity == 0 && e.SubQuantity == 0, int8(0), int8(1))
	if e.SKU == "" && e.Lote == "" {
		e.IsWarehouseProductStatus = 1
		if e.WarehouseProductQuantity > 0 {
			e.IsWarehouseProductStatus = 2
		}
	} else {
		e.IsWarehouseProductStatus = 0
	}
}

func MakeProductoStockByKey(key string) ProductStock {
	return ProductStock{}
}

type ProductStockTable struct {
	db.TableStruct[ProductStockTable, ProductStock]
	CompanyID      db.Col[ProductStockTable, int32]
	ID             db.Col[ProductStockTable, string]
	SKU            db.Col[ProductStockTable, string]
	Lote           db.Col[ProductStockTable, string]
	WarehouseID    db.Col[ProductStockTable, int32]
	ProductID      db.Col[ProductStockTable, int32]
	PresentationID db.Col[ProductStockTable, int16]
	Quantity       db.Col[ProductStockTable, int32]
	SubQuantity    db.Col[ProductStockTable, int32]
	Updated        db.Col[ProductStockTable, int32]
	UpdatedBy      db.Col[ProductStockTable, int32]
	Status         db.Col[ProductStockTable, int8]

	IsWarehouseProductStatus db.Col[ProductStockTable, int8]
	WarehouseProductQuantity db.Col[ProductStockTable, int32]
	StockComputedFecha       db.Col[ProductStockTable, int16]
	StockComputedQuantity    db.Col[ProductStockTable, int32]
}

func (e ProductStockTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:            "warehouse_product_stock",
		Partition:       e.CompanyID,
		Keys:            []db.Coln{e.ID},
		KeyConcatenated: []db.Coln{e.WarehouseID, e.ProductID, e.PresentationID, e.SKU, e.Lote},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.SKU}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.Lote}},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.IsWarehouseProductStatus, e.Updated.DecimalSize(10)},
				Cols:     []db.Coln{e.WarehouseProductQuantity, e.PresentationID},
				KeepPart: true,
			},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.ProductID.Int32(), e.Status.DecimalSize(1)},
				KeepPart: true,
			},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.ProductID.Int32(), e.Status.DecimalSize(1)},
				KeepPart: true,
			},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.WarehouseID, e.Status.DecimalSize(1), e.Updated.DecimalSize(9)},
				KeepPart: true,
			},
		},
	}
}
