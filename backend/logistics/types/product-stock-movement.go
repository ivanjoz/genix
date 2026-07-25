package types

import (
	"github.com/ivanjoz/genix-orm/scylla"
)

// WarehouseProductMovement is the append-only movement ledger.
// LotID (0 = no lot) and SerialNumber (empty = no serial) together tell whether
// the movement targets a ProductStockDetail row or the plain ProductStockV2.Quantity bucket.
type WarehouseProductMovement struct {
	scylla.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID int32 `json:",omitempty"`
	// ID packs Date(5)+WarehouseID(5)+Autoincrement(3) into an int64.
	ID                   int64
	SerialNumber         string `json:",omitempty"`
	LotID                int32  `json:",omitempty"`
	WarehouseID          int32  `json:",omitempty"`
	WarehouseRefID       int32  `json:",omitempty"`
	WarehouseRefQuantity int32  `json:",omitempty"`
	Date                 int16  `json:",omitempty"`
	DocumentID           int64  `json:",omitempty"`
	ProductID            int32  `json:",omitempty"`
	PresentationID       int16  `json:",omitempty"`
	Quantity             int32  `json:",omitempty"`
	WarehouseQuantity    int32  `json:",omitempty"`
	SubQuantity          int32  `json:",omitempty"`
	MonetaryValue        int32  `json:",omitempty"`
	Type                 int8   `json:",omitempty"`
	Created              int32  `json:",omitempty"`
	CreatedBy            int32  `json:",omitempty"`
	UpdateCounter        int32  `json:",omitempty"`
}

type WarehouseProductMovementTable struct {
	scylla.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID            scylla.Col[WarehouseProductMovementTable, int32]
	ID                   scylla.Col[WarehouseProductMovementTable, int64]
	SerialNumber         scylla.Col[WarehouseProductMovementTable, string]
	LotID                scylla.Col[WarehouseProductMovementTable, int32]
	WarehouseID          scylla.Col[WarehouseProductMovementTable, int32]
	WarehouseRefID       scylla.Col[WarehouseProductMovementTable, int32]
	WarehouseRefQuantity scylla.Col[WarehouseProductMovementTable, int32]
	DocumentID           scylla.Col[WarehouseProductMovementTable, int64]
	ProductID            scylla.Col[WarehouseProductMovementTable, int32]
	PresentationID       scylla.Col[WarehouseProductMovementTable, int16]
	Quantity             scylla.Col[WarehouseProductMovementTable, int32]
	WarehouseQuantity    scylla.Col[WarehouseProductMovementTable, int32]
	SubQuantity          scylla.Col[WarehouseProductMovementTable, int32]
	Type                 scylla.Col[WarehouseProductMovementTable, int8]
	Created              scylla.Col[WarehouseProductMovementTable, int32]
	MonetaryValue        scylla.Col[WarehouseProductMovementTable, int32]
	CreatedBy            scylla.Col[WarehouseProductMovementTable, int32]
	Date                 scylla.Col[WarehouseProductMovementTable, int16]
	UpdateCounter        scylla.Col[WarehouseProductMovementTable, int32]
}

func (e WarehouseProductMovementTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "warehouse_product_movement",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID),
		KeyIntPacking: scylla.Cols(
			e.Date.DecimalSize(5), e.WarehouseID.DecimalSize(5), e.Autoincrement(3),
		),
		AutoincrementPart: e.Date,
		Indexes: []scylla.Index{
			{
				Type: scylla.TypeInheritFromKey, Keys: scylla.Cols(e.Date), UseIndexGroup: true,
			},
			{
				Type: scylla.TypeInheritFromKey, Keys: scylla.Cols(e.Date, e.WarehouseID), UseIndexGroup: true,
			},
			{
				Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.SerialNumber),
			},
			{
				Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.LotID), UseIndexGroup: true,
			},
			{
				Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.DocumentID), UseIndexGroup: true,
			},
			{
				Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.Date, e.Type), UseIndexGroup: true,
			},
			{
				Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.Date, e.Type, e.WarehouseID), UseIndexGroup: true,
			},
			{
				Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.Date, e.ProductID), UseIndexGroup: true,
			},
			{
				Type: scylla.TypeView,
				Keys: scylla.Cols(e.Date, e.ProductID.DecimalSize(9), e.Type.DecimalSize(1)),
				Cols: scylla.Cols(e.Quantity),
			},
		},
	}
}

// InternalMovement is the ApplyMovements input unit. It targets a single
// (Warehouse, Product, Presentation) bucket, optionally scoped to a Lot and/or SerialNumber.
//
// Lot resolution rules:
//   - If LotID > 0: use it directly (required for outbound, Quantity < 0).
//   - If LotID == 0 and LotName != "" and Quantity > 0 (inbound): lot is resolved or
//     created from Hash(today, SupplierID, LotName). SupplierID is required.
//   - If LotID == 0 and LotName == "" and SerialNumber == "": treated as "no-detail",
//     mutating ProductStockV2.Quantity only.
type InternalMovement struct {
	ProductID       int32
	PresentationID  int16
	ReplaceQuantity bool
	Type            int8
	SerialNumber    string
	LotName         string
	LotID           int32
	SupplierID      int32
	WarehouseID     int32
	DestWarehouseID int32
	Quantity        int32
	SubQuantity     int32
	Price           int32
	DocumentID      int64
}

// HasDetail reports whether the movement targets a ProductStockDetail row
// (i.e. the non-free bucket keyed by LotID and/or SerialNumber).
func (e *InternalMovement) HasDetail() bool {
	return e.SerialNumber != "" || e.LotID > 0 || e.LotName != ""
}
