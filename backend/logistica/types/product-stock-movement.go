package types

import (
	"app/core"
	"app/db"
)

type AlmacenMovimiento struct {
	db.TableStruct[AlmacenMovimientoTable, AlmacenMovimiento]
	EmpresaID int32 `json:",omitempty"`
	// [Almacen-ID] + [Created] + [Ramdom Number]
	ID                 int64
	SKU                string `json:",omitempty"`
	Lote               string `json:",omitempty"`
	AlmacenID          int32  `json:",omitempty"`
	AlmacenRefID       int32  `json:",omitempty"`
	AlmacenRefCantidad int32  `json:",omitempty"`
	Fecha              int16  `json:",omitempty"`
	DocumentID         int64  `json:",omitempty"`
	ProductoID         int32  `json:",omitempty"`
	PresentacionID     int16  `json:",omitempty"`
	Cantidad           int32  `json:",omitempty"`
	AlmacenCantidad    int32  `json:",omitempty"`
	SubCantidad        int32  `json:",omitempty"`
	Tipo               int8   `json:",omitempty"`
	Created            int32  `json:",omitempty"`
	CreatedBy          int32  `json:",omitempty"`

	// Extra fields
	Inflows_  int32 `json:"-"`
	Outflows_ int32 `json:"-"`
}

type AlmacenMovimientoTable struct {
	db.TableStruct[AlmacenMovimientoTable, AlmacenMovimiento]
	EmpresaID          db.Col[AlmacenMovimientoTable, int32]
	ID                 db.Col[AlmacenMovimientoTable, int64]
	SKU                db.Col[AlmacenMovimientoTable, string]
	Lote               db.Col[AlmacenMovimientoTable, string]
	AlmacenID          db.Col[AlmacenMovimientoTable, int32]
	AlmacenRefID       db.Col[AlmacenMovimientoTable, int32]
	AlmacenRefCantidad db.Col[AlmacenMovimientoTable, int32]
	DocumentID         db.Col[AlmacenMovimientoTable, int64]
	ProductoID         db.Col[AlmacenMovimientoTable, int32]
	PresentacionID     db.Col[AlmacenMovimientoTable, int16]
	Cantidad           db.Col[AlmacenMovimientoTable, int32]
	AlmacenCantidad    db.Col[AlmacenMovimientoTable, int32]
	SubCantidad        db.Col[AlmacenMovimientoTable, int32]
	Tipo               db.Col[AlmacenMovimientoTable, int8]
	Created            db.Col[AlmacenMovimientoTable, int32]
	CreatedBy          db.Col[AlmacenMovimientoTable, int32]
	Fecha              db.Col[AlmacenMovimientoTable, int16]
}

func (e AlmacenMovimientoTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "almacen_movimiento",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ID},
		KeyIntPacking: []db.Coln{
			e.AlmacenID.DecimalSize(5), e.Fecha.DecimalSize(5), e.Autoincrement(3),
		},
		AutoincrementPart: e.Fecha,
		Indexes:           [][]db.Coln{{e.SKU}, {e.Lote}},
		Views: []db.View{
			{Keys: []db.Coln{e.Created}, KeepPart: true},
			{Keys: []db.Coln{e.AlmacenRefID, e.Created.DecimalSize(9)}, KeepPart: true},
			{Keys: []db.Coln{e.Fecha, e.ProductoID.DecimalSize(10)}, Cols: []db.Coln{e.Cantidad}, KeepPart: true},
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
	AlmacenID            int32
	AlmacenDestinoID     int32
	Cantidad             int32
	SubCantidad          int32
	ModificarCantidad    int32
	ModificarSubCantidad int32
	DocumentID           int64
	Stock                *ProductoStock
}

func (e *MovimientoInterno) GetAlmacenProductoID() string {
	return core.Concat62(e.AlmacenID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote)
}
