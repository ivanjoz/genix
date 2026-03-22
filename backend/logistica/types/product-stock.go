package types

import "app/db"

type ProductoStock struct {
	db.TableStruct[ProductoStockTable, ProductoStock]
	EmpresaID      int32 `json:",omitempty"`
	ID             string
	SKU            string  `json:",omitempty"`
	Lote           string  `json:",omitempty"`
	AlmacenID      int32   `json:",omitempty"`
	ProductoID     int32   `json:",omitempty"`
	PresentacionID int16   `json:",omitempty"`
	Cantidad       int32   `json:",omitempty"`
	SubCantidad    int32   `json:",omitempty"`
	CostoUn        float32 `json:",omitempty"`
	Updated        int32   `json:"upd,omitempty"`
	UpdatedBy      int32   `json:",omitempty"`
	Status         int8    `json:"ss,omitempty"`
}

type ProductoStockTable struct {
	db.TableStruct[ProductoStockTable, ProductoStock]
	EmpresaID      db.Col[ProductoStockTable, int32]
	ID             db.Col[ProductoStockTable, string]
	SKU            db.Col[ProductoStockTable, string]
	Lote           db.Col[ProductoStockTable, string]
	AlmacenID      db.Col[ProductoStockTable, int32]
	ProductoID     db.Col[ProductoStockTable, int32]
	PresentacionID db.Col[ProductoStockTable, int16]
	Cantidad       db.Col[ProductoStockTable, int32]
	SubCantidad    db.Col[ProductoStockTable, int32]
	CostoUn        db.Col[ProductoStockTable, float32]
	Updated        db.Col[ProductoStockTable, int32]
	UpdatedBy      db.Col[ProductoStockTable, int32]
	Status         db.Col[ProductoStockTable, int8]
}

func (e ProductoStockTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:            "almacen_producto",
		Partition:       e.EmpresaID,
		Keys:            []db.Coln{e.ID},
		KeyConcatenated: []db.Coln{e.AlmacenID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote},
		LocalIndexes:    []db.Coln{e.SKU, e.Lote},
		Views: []db.View{
			{
				Cols:     []db.Coln{e.ProductoID.Int32(), e.Status.DecimalSize(1)},
				KeepPart: true,
			},
			{
				Cols:     []db.Coln{e.AlmacenID, e.Status.DecimalSize(1), e.Updated.DecimalSize(9)},
				KeepPart: true,
			},
		},
	}
}
