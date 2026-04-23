package types

import (
	"app/db"
)

// WarehouseProductMovement is the append-only movement ledger.
// LotID (0 = no lot) and SerialNumber (empty = no serial) together tell whether
// the movement targets a ProductStockDetail row or the plain ProductStockV2.Quantity bucket.
type WarehouseProductMovement struct {
	db.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID int32 `json:",omitempty"`
	// ID packs Fecha(5)+WarehouseID(5)+Autoincrement(3) into an int64.
	ID                   int64
	SerialNumber         string `json:",omitempty"`
	LotID                int32  `json:",omitempty"`
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
	UpdateCounter        int32  `json:",omitempty"`
}

type WarehouseProductMovementTable struct {
	db.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID            db.Col[WarehouseProductMovementTable, int32]
	ID                   db.Col[WarehouseProductMovementTable, int64]
	SerialNumber         db.Col[WarehouseProductMovementTable, string]
	LotID                db.Col[WarehouseProductMovementTable, int32]
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
	UpdateCounter        db.Col[WarehouseProductMovementTable, int32]
}

func (e WarehouseProductMovementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:                 "warehouse_product_movement",
		Partition:            e.CompanyID,
		Keys:                 []db.Coln{e.ID},
		KeyIntPacking: []db.Coln{
			e.Fecha.DecimalSize(5), e.WarehouseID.DecimalSize(5), e.Autoincrement(3),
		},
		AutoincrementPart: e.Fecha,
		Indexes: []db.Index{
			{
				Type: db.TypeInheritFromKey, Keys: []db.Coln{e.Fecha}, UseIndexGroup: true,
			},
			{
				Type: db.TypeInheritFromKey, Keys: []db.Coln{e.Fecha, e.WarehouseID}, UseIndexGroup: true,
			},
			{ 
				Type: db.TypeLocalIndex, Keys: []db.Coln{e.SerialNumber}, 
			},
			{
				Type: db.TypeLocalIndex, Keys: []db.Coln{e.LotID}, UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: []db.Coln{e.DocumentID}, UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: []db.Coln{e.Fecha, e.Tipo}, UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: []db.Coln{e.Fecha, e.Tipo, e.WarehouseID}, UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: []db.Coln{e.Fecha, e.ProductoID}, UseIndexGroup: true,
			},
			{
				Type: db.TypeView, 
				Keys: []db.Coln{e.Fecha, e.ProductoID.DecimalSize(9), e.Tipo.DecimalSize(1)},
				Cols: []db.Coln{e.Quantity},
			},
		},
	}
}

// MovimientoInterno is the ApplyMovimientos input unit. It targets a single
// (Warehouse, Product, Presentation) bucket, optionally scoped to a Lot and/or SerialNumber.
//
// Lot resolution rules:
//   - If LotID > 0: use it directly (required for outbound, Cantidad < 0).
//   - If LotID == 0 and LotName != "" and Cantidad > 0 (inbound): lot is resolved or
//     created from Hash(today, SupplierID, LotName). SupplierID is required.
//   - If LotID == 0 and LotName == "" and SerialNumber == "": treated as "no-detail",
//     mutating ProductStockV2.Quantity only.
type MovimientoInterno struct {
	ProductoID         int32
	PresentacionID     int16
	ReemplazarCantidad bool
	Tipo               int8
	SerialNumber       string
	LotName            string
	LotID              int32
	SupplierID         int32
	WarehouseID        int32
	AlmacenDestinoID   int32
	Cantidad           int32
	SubCantidad        int32
	DocumentID         int64
}

// HasDetail reports whether the movement targets a ProductStockDetail row
// (i.e. the non-free bucket keyed by LotID and/or SerialNumber).
func (e *MovimientoInterno) HasDetail() bool {
	return e.SerialNumber != "" || e.LotID > 0 || e.LotName != ""
}
