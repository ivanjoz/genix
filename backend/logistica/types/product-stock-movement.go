package types

import (
	"app/db"
)

type WarehouseProductMovement struct {
	db.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID int32 `json:",omitempty"`
	// e.Fecha.DecimalSize(5), e.WarehouseID.DecimalSize(5), e.Autoincrement(3),
	ID                   int64
	SKU                  string `json:",omitempty"`
	Lote                 string `json:",omitempty"`
	WarehouseID          int32  `json:",omitempty"`
	WarehouseRefID       int32  `json:",omitempty"`
	WarehouseRefQuantity int32  `json:",omitempty"`
	Fecha                int16  `json:",omitempty"`
	DocumentID           int64  `json:",omitempty"`
	ProductoID           int32  `json:",omitempty"`
	PresentacionID       int16  `json:",omitempty"`
	Quantity             int32  `json:",omitempty"`
	WarehouseQuantity    int32  `json:",omitempty"`
	SubQuantity          int32  `json:",omitempty"`
	MonetaryValue        int32  `json:",omitempty"`
	Tipo                 int8   `json:",omitempty"`
	Created              int32  `json:",omitempty"`
	CreatedBy            int32  `json:",omitempty"`
}

type WarehouseProductMovementTable struct {
	db.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID            db.Col[WarehouseProductMovementTable, int32]
	ID                   db.Col[WarehouseProductMovementTable, int64]
	SKU                  db.Col[WarehouseProductMovementTable, string]
	Lote                 db.Col[WarehouseProductMovementTable, string]
	WarehouseID          db.Col[WarehouseProductMovementTable, int32]
	WarehouseRefID       db.Col[WarehouseProductMovementTable, int32]
	WarehouseRefQuantity db.Col[WarehouseProductMovementTable, int32]
	DocumentID           db.Col[WarehouseProductMovementTable, int64]
	ProductoID           db.Col[WarehouseProductMovementTable, int32]
	PresentacionID       db.Col[WarehouseProductMovementTable, int16]
	Quantity             db.Col[WarehouseProductMovementTable, int32]
	WarehouseQuantity    db.Col[WarehouseProductMovementTable, int32]
	SubQuantity          db.Col[WarehouseProductMovementTable, int32]
	Tipo                 db.Col[WarehouseProductMovementTable, int8]
	Created              db.Col[WarehouseProductMovementTable, int32]
	MonetaryValue        db.Col[WarehouseProductMovementTable, int32]
	CreatedBy            db.Col[WarehouseProductMovementTable, int32]
	Fecha                db.Col[WarehouseProductMovementTable, int16]
}

func (e WarehouseProductMovementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "warehouse_product_movement",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID},
		KeyIntPacking: []db.Coln{
			e.Fecha.DecimalSize(5), e.WarehouseID.DecimalSize(5), e.Autoincrement(3),
		},
		AutoincrementPart: e.Fecha,
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.SKU}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.Lote}},
			{Type: db.TypeView, Keys: []db.Coln{e.Created}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.WarehouseRefID, e.Created.DecimalSize(9)}, KeepPart: true},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.Fecha, e.ProductoID.DecimalSize(9), e.Tipo.DecimalSize(1)},
				Cols:     []db.Coln{e.Quantity},
				KeepPart: true,
			},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.Fecha, e.WarehouseID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote},
				Cols:     []db.Coln{e.Quantity},
				KeepPart: true,
			},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.Fecha, e.WarehouseID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote},
				Cols:     []db.Coln{e.Quantity},
				KeepPart: true,
			},
		},
	}
}

type MovimientoInterno struct {
	ProductoID           int32
	PresentacionID       int16
	ReemplazarCantidad   bool
	Tipo                 int8
	SKU                  string
	Lote                 string
	WarehouseID          int32
	AlmacenDestinoID     int32
	Cantidad             int32
	SubCantidad          int32
	ModificarCantidad    int32
	ModificarSubCantidad int32
	DocumentID           int64
}

func (e *MovimientoInterno) GetAlmacenProductoID() string {
	return db.MakeKeyConcat(e.WarehouseID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote)
}
func (e *MovimientoInterno) GetAlmacenProductoGrupoID() string {
	return db.MakeKeyConcat(e.WarehouseID, e.ProductoID, e.PresentacionID)
}
