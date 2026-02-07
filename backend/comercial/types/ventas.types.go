package types

import "app/db"


type Venta struct {
	db.TableStruct[VentaTable, Venta]
	EmpresaID int32 `json:",omitempty"`
	AlmacenID int32   `json:",omitempty"`
	ID             int64
	SKU            string  `json:",omitempty"`
	Lote           string  `json:",omitempty"`
	ProductoID     int32   `json:",omitempty"`
	PresentacionID int16   `json:",omitempty"`
	Cantidad       int32   `json:",omitempty"`
	SubCantidad    int32   `json:",omitempty"`
	CostoUn        float32 `json:",omitempty"`
	Updated        int32   `json:"upd,omitempty"`
	UpdatedBy      int32   `json:",omitempty"`
	Status         int8    `json:"ss,omitempty"`
}

type VentaTable struct {
	db.TableStruct[VentaTable, Venta]
	EmpresaID          db.Col[VentaTable, int32]
	ID                 db.Col[VentaTable, int64]
		Fecha  db.Col[VentaTable, int16]
				AlmacenID  db.Col[VentaTable, int32]
}


func (e VentaTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:            "ventas",
		Partition:       e.EmpresaID,
		Keys:            []db.Coln{e.ID},
		KeyIntPacking: []db.Coln{
			e.Fecha.DecimalSize(5), e.AlmacenID.DecimalSize(4), e.Autoincrement(3),
		},
		AutoincrementPart: e.Fecha,
	}
}
