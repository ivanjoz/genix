package types

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
)

// ProductStock is the current stock row per (Warehouse, Product, Presentation).
// - Quantity         : stock with no lot/serial tracking (the "free" bucket).
// - DetailQuantity   : sum of ProductStockDetail.Quantity rows linked to this row.
// - DetailComputed*  : async-precomputed snapshot (populated by a separate job).
type ProductStock struct {
	scylla.TableStruct[ProductStockTable, ProductStock]
	ID                        int64
	CompanyID                 int32   `json:",omitempty"`
	WarehouseID               int32   `json:",omitempty"`
	ProductID                 int32   `json:",omitempty"`
	PresentationID            int16   `json:",omitempty"`
	Quantity                  int32   `json:",omitempty"`
	SubQuantity               int32   `json:",omitempty"`
	DetailQuantity            int32   `json:",omitempty"`
	DetailSubQuantity         int32   `json:",omitempty"`
	DetailComputedDate        int16   `json:",omitempty"`
	DetailComputedQuantity    int32   `json:",omitempty"`
	DetailComputedSubQuantity int32   `json:",omitempty"`
	LastPricesPrice           []int32 `json:",omitempty"`
	LastPricesQuantity        []int32 `json:",omitempty"`
	StockStatus               int8    `json:",omitempty"`

	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Status    int8  `json:"ss,omitempty"`
}

// Derive Status from the two live buckets (Quantity + DetailQuantity).
func (e *ProductStock) SelfParse() {
	e.Status = core.If(e.Quantity == 0 && e.DetailQuantity == 0, int8(0), int8(1))
}

type ProductStockTable struct {
	scylla.TableStruct[ProductStockTable, ProductStock]
	CompanyID                 scylla.Col[ProductStockTable, int32]
	ID                        scylla.Col[ProductStockTable, int64]
	WarehouseID               scylla.Col[ProductStockTable, int32]
	ProductID                 scylla.Col[ProductStockTable, int32]
	PresentationID            scylla.Col[ProductStockTable, int16]
	Quantity                  scylla.Col[ProductStockTable, int32]
	SubQuantity               scylla.Col[ProductStockTable, int32]
	DetailQuantity            scylla.Col[ProductStockTable, int32]
	DetailSubQuantity         scylla.Col[ProductStockTable, int32]
	DetailComputedDate        scylla.Col[ProductStockTable, int16]
	DetailComputedQuantity    scylla.Col[ProductStockTable, int32]
	DetailComputedSubQuantity scylla.Col[ProductStockTable, int32]
	LastPricesPrice           scylla.ColSlice[ProductStockTable, int32]
	LastPricesQuantity        scylla.ColSlice[ProductStockTable, int32]
	StockStatus               scylla.Col[ProductStockTable, int8]
	Created                   scylla.Col[ProductStockTable, int32]
	CreatedBy                 scylla.Col[ProductStockTable, int32]
	Updated                   scylla.Col[ProductStockTable, int32]
	UpdatedBy                 scylla.Col[ProductStockTable, int32]
	Status                    scylla.Col[ProductStockTable, int8]
}

func (e ProductStockTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:                 "warehouse_product_stock",
		Partition:            e.CompanyID,
		Keys:                 scylla.Cols(e.ID),
		DisableUpdateCounter: true,
		// ID packs (WarehouseID, ProductID, PresentationID) into the single int64 key.
		KeyIntPacking: scylla.Cols(
			e.WarehouseID.DecimalSize(5),
			e.ProductID.DecimalSize(9),
			e.PresentationID.DecimalSize(4),
		),
		Indexes: []scylla.Index{
			{
				Type: scylla.TypeView,
				Keys: scylla.Cols(e.WarehouseID, e.Status.DecimalSize(1), e.Updated.DecimalSize(10)),
			},
			{
				Type: scylla.TypeView,
				Keys: scylla.Cols(e.Status, e.Updated.DecimalSize(8)),
			},
		},
	}
}

// ProductStockDetail rows track stock by Lot and/or SerialNumber.
// Composite key is (ProductStockID, LotID, SerialNumber), so a row exists per
// distinct (warehouse-product-presentation, lot, serial) combination.
type ProductStockDetail struct {
	scylla.TableStruct[ProductStockDetailTable, ProductStockDetail]
	CompanyID      int32  `json:",omitempty"`
	ProductStockID int64  `json:",omitempty"`
	LotID          int32  `json:",omitempty"`
	SerialNumber   string `json:",omitempty"`
	WarehouseID    int32  `json:",omitempty"`
	ProductID      int32  `json:",omitempty"`
	Quantity       int32  `json:",omitempty"`
	SubQuantity    int32  `json:",omitempty"`
	ExpirationDate int16  `json:",omitempty"`

	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	Status    int8  `json:"ss,omitempty"`
}

type ProductStockDetailTable struct {
	scylla.TableStruct[ProductStockDetailTable, ProductStockDetail]
	CompanyID      scylla.Col[ProductStockDetailTable, int32]
	ProductStockID scylla.Col[ProductStockDetailTable, int64]
	LotID          scylla.Col[ProductStockDetailTable, int32]
	SerialNumber   scylla.Col[ProductStockDetailTable, string]
	WarehouseID    scylla.Col[ProductStockDetailTable, int32]
	ProductID      scylla.Col[ProductStockDetailTable, int32]
	Quantity       scylla.Col[ProductStockDetailTable, int32]
	SubQuantity    scylla.Col[ProductStockDetailTable, int32]
	ExpirationDate scylla.Col[ProductStockDetailTable, int16]
	Updated        scylla.Col[ProductStockDetailTable, int32]
	UpdatedBy      scylla.Col[ProductStockDetailTable, int32]
	Created        scylla.Col[ProductStockDetailTable, int32]
	CreatedBy      scylla.Col[ProductStockDetailTable, int32]
	Status         scylla.Col[ProductStockDetailTable, int8]
}

func (e ProductStockDetailTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:                 "warehouse_product_stock_detail",
		Partition:            e.CompanyID,
		DisableUpdateCounter: true,
		// One detail row per stock-record + lot + serial.
		Keys: scylla.Cols(e.ProductStockID, e.LotID, e.SerialNumber),
		Indexes: []scylla.Index{
			// Hash index for dedup lookups when resolving LotID from (Date, SupplierID, Name).
			{
				Type: scylla.TypeView,
				Keys: scylla.Cols(e.WarehouseID, e.Status.DecimalSize(1), e.Updated.DecimalSize(10)),
			},
		},
	}
}

// ProductStockLot is a dedupe-able lot catalog. Hash = (Date, SupplierID, Name).
type ProductStockLot struct {
	scylla.TableStruct[ProductStockLotTable, ProductStockLot]
	CompanyID        int32  `json:",omitempty"`
	ID               int32  `json:",omitempty"`
	Date             int16  `json:",omitempty"`
	Name             string `json:",omitempty"`
	SupplierID       int32  `json:",omitempty"`
	DeliveryNoteID   int32  `json:",omitempty"`
	DeliveryNoteCode string `json:",omitempty"`
	Hash             string `json:",omitempty"`

	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	Status    int8  `json:"ss,omitempty"`
}

func (e *ProductStockLot) SelfParse() {
	e.Hash = scylla.MakeKeyConcat(e.Date, e.SupplierID, e.Name)
}

type ProductStockLotTable struct {
	scylla.TableStruct[ProductStockLotTable, ProductStockLot]
	CompanyID        scylla.Col[ProductStockLotTable, int32]
	ID               scylla.Col[ProductStockLotTable, int32]
	Date             scylla.Col[ProductStockLotTable, int16]
	Name             scylla.Col[ProductStockLotTable, string]
	SupplierID       scylla.Col[ProductStockLotTable, int32]
	DeliveryNoteID   scylla.Col[ProductStockLotTable, int32]
	DeliveryNoteCode scylla.Col[ProductStockLotTable, string]
	Hash             scylla.Col[ProductStockLotTable, string]
	Created          scylla.Col[ProductStockLotTable, int32]
	CreatedBy        scylla.Col[ProductStockLotTable, int32]
	Status           scylla.Col[ProductStockLotTable, int8]
}

func (e ProductStockLotTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "product_stock_lot",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			// Hash index for dedup lookups when resolving LotID from (Date, SupplierID, Name).
			{Type: scylla.TypeGlobalIndex, Keys: scylla.Cols(e.Hash)},
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.Name)},
		},
	}
}

// DeliveryOrderNote is modeled but not wired up yet.
type DeliveryOrderNote struct {
	scylla.TableStruct[DeliveryOrderNoteTable, DeliveryOrderNote]
	CompanyID   int32  `json:",omitempty"`
	ID          int32  `json:",omitempty"`
	Date        int16  `json:",omitempty"`
	Code        string `json:",omitempty"`
	SupplierID  int32  `json:",omitempty"`
	Description string `json:",omitempty"`
	Hash        string `json:",omitempty"`

	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	Updated   int32 `json:",omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Status    int8  `json:"ss,omitempty"`
}

type DeliveryOrderNoteTable struct {
	scylla.TableStruct[DeliveryOrderNoteTable, DeliveryOrderNote]
	CompanyID   scylla.Col[DeliveryOrderNoteTable, int32]
	ID          scylla.Col[DeliveryOrderNoteTable, int32]
	Date        scylla.Col[DeliveryOrderNoteTable, int16]
	Code        scylla.Col[DeliveryOrderNoteTable, string]
	SupplierID  scylla.Col[DeliveryOrderNoteTable, int32]
	Description scylla.Col[DeliveryOrderNoteTable, string]
	Hash        scylla.Col[DeliveryOrderNoteTable, string]
	Created     scylla.Col[DeliveryOrderNoteTable, int32]
	CreatedBy   scylla.Col[DeliveryOrderNoteTable, int32]
	Updated     scylla.Col[DeliveryOrderNoteTable, int32]
	UpdatedBy   scylla.Col[DeliveryOrderNoteTable, int32]
	Status      scylla.Col[DeliveryOrderNoteTable, int8]
}

func (e DeliveryOrderNoteTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "delivery_order_note",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{Type: scylla.TypeGlobalIndex, Keys: scylla.Cols(e.Hash)},
		},
	}
}

func (e *DeliveryOrderNote) SelfParse() {
	e.Hash = scylla.MakeKeyConcat(e.Date, e.SupplierID, e.Code)
}
