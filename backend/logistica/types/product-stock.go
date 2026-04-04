package types

import "app/db"

type ProductStock struct {
	db.TableStruct[ProductStockTable, ProductStock]
	CompanyID      int32 `json:",omitempty"`
	ID             string
	SKU            string  `json:",omitempty"`
	Lote           string  `json:",omitempty"`
	WarehouseID      int32   `json:",omitempty"`
	ProductoID     int32   `json:",omitempty"`
	PresentacionID int16   `json:",omitempty"`
	Cantidad       int32   `json:",omitempty"`
	SubCantidad    int32   `json:",omitempty"`
	CostoUn        float32 `json:",omitempty"`
	Updated        int32   `json:"upd,omitempty"`
	UpdatedBy      int32   `json:",omitempty"`
	Status         int8    `json:"ss,omitempty"`

	IsWarehouseProductStatus int8  `json:",omitempty"`
	WarehouseProductQuantity int32 `json:",omitempty"`
	StockComputedFecha       int16 `json:",omitempty"`
	StockComputedQuantity    int32 `json:",omitempty"`
	IsNewRecord bool `json:",omitempty"`
}

func (e *ProductStock) SelfParse() {
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
	WarehouseID      db.Col[ProductStockTable, int32]
	ProductoID     db.Col[ProductStockTable, int32]
	PresentacionID db.Col[ProductStockTable, int16]
	Cantidad       db.Col[ProductStockTable, int32]
	SubCantidad    db.Col[ProductStockTable, int32]
	CostoUn        db.Col[ProductStockTable, float32]
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
		KeyConcatenated: []db.Coln{e.WarehouseID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote},
		LocalIndexes:    []db.Coln{e.SKU, e.Lote},
		Views: []db.View{
			{
				Keys:     []db.Coln{e.IsWarehouseProductStatus, e.Updated.DecimalSize(10)},
				Cols:     []db.Coln{e.WarehouseProductQuantity, e.PresentacionID},
				KeepPart: true,
			},
			{
				Keys:     []db.Coln{e.ProductoID.Int32(), e.Status.DecimalSize(1)},
				KeepPart: true,
			},
			{
				Keys:     []db.Coln{e.ProductoID.Int32(), e.Status.DecimalSize(1)},
				KeepPart: true,
			},
			{
				Keys:     []db.Coln{e.WarehouseID, e.Status.DecimalSize(1), e.Updated.DecimalSize(9)},
				KeepPart: true,
			},
		},
	}
}
