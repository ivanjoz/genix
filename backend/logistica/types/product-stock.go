package types

import (
	"app/core"
	"app/db"
)

// ProductStock is the LEGACY stock table. Kept registered so the table
// stays deployed for historical reads, but writes and live logic moved
// to ProductStockV2 + ProductStockDetail + ProductStockLot.
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
		Name:            "warehouse_product_stock_legacy",
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
				Keys:     []db.Coln{e.WarehouseID, e.Status.DecimalSize(1), e.Updated.DecimalSize(9)},
				KeepPart: true,
			},
		},
	}
}

// ProductStockV2 is the current stock row per (Warehouse, Product, Presentation).
// - Quantity         : stock with no lot/serial tracking (the "free" bucket).
// - DetailQuantity   : sum of ProductStockDetail.Quantity rows linked to this row.
// - DetailComputed*  : async-precomputed snapshot (populated by a separate job).
type ProductStockV2 struct {
	db.TableStruct[ProductStockV2Table, ProductStockV2]
	ID                        int64
	CompanyID                 int32 `json:",omitempty"`
	WarehouseID               int32 `json:",omitempty"`
	ProductID                 int32 `json:",omitempty"`
	PresentationID            int16 `json:",omitempty"`
	Quantity                  int32 `json:",omitempty"`
	SubQuantity               int32 `json:",omitempty"`
	DetailQuantity            int32 `json:",omitempty"`
	DetailSubQuantity         int32 `json:",omitempty"`
	DetailComputedDate        int16 `json:",omitempty"`
	DetailComputedQuantity    int32 `json:",omitempty"`
	DetailComputedSubQuantity int32 `json:",omitempty"`
	StockStatus               int8  `json:",omitempty"`

	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Status    int8  `json:"ss,omitempty"`
}

// Derive Status from the two live buckets (Quantity + DetailQuantity).
func (e *ProductStockV2) SelfParse() {
	e.Status = core.If(e.Quantity == 0 && e.DetailQuantity == 0, int8(0), int8(1))
}

type ProductStockV2Table struct {
	db.TableStruct[ProductStockV2Table, ProductStockV2]
	CompanyID                 db.Col[ProductStockV2Table, int32]
	ID                        db.Col[ProductStockV2Table, int64]
	WarehouseID               db.Col[ProductStockV2Table, int32]
	ProductID                 db.Col[ProductStockV2Table, int32]
	PresentationID            db.Col[ProductStockV2Table, int16]
	Quantity                  db.Col[ProductStockV2Table, int32]
	SubQuantity               db.Col[ProductStockV2Table, int32]
	DetailQuantity            db.Col[ProductStockV2Table, int32]
	DetailSubQuantity         db.Col[ProductStockV2Table, int32]
	DetailComputedDate        db.Col[ProductStockV2Table, int16]
	DetailComputedQuantity    db.Col[ProductStockV2Table, int32]
	DetailComputedSubQuantity db.Col[ProductStockV2Table, int32]
	StockStatus               db.Col[ProductStockV2Table, int8]
	Created                   db.Col[ProductStockV2Table, int32]
	CreatedBy                 db.Col[ProductStockV2Table, int32]
	Updated                   db.Col[ProductStockV2Table, int32]
	UpdatedBy                 db.Col[ProductStockV2Table, int32]
	Status                    db.Col[ProductStockV2Table, int8]
}

func (e ProductStockV2Table) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:                 "warehouse_product_stock",
		Partition:            e.CompanyID,
		Keys:                 []db.Coln{e.ID},
		DisableUpdateCounter: true,
		// ID packs (WarehouseID, ProductID, PresentationID) into the single int64 key.
		KeyIntPacking: []db.Coln{
			e.WarehouseID.DecimalSize(5),
			e.ProductID.DecimalSize(9),
			e.PresentationID.DecimalSize(4),
		},
		Indexes: []db.Index{
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.WarehouseID, e.Status.DecimalSize(1), e.Updated.DecimalSize(10)},
				KeepPart: true,
			},
		},
	}
}

// ProductStockDetail rows track stock by Lot and/or SerialNumber.
// Composite key is (ProductStockID, LotID, SerialNumber), so a row exists per
// distinct (warehouse-product-presentation, lot, serial) combination.
type ProductStockDetail struct {
	db.TableStruct[ProductStockDetailTable, ProductStockDetail]
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
	db.TableStruct[ProductStockDetailTable, ProductStockDetail]
	CompanyID      db.Col[ProductStockDetailTable, int32]
	ProductStockID db.Col[ProductStockDetailTable, int64]
	LotID          db.Col[ProductStockDetailTable, int32]
	SerialNumber   db.Col[ProductStockDetailTable, string]
	WarehouseID    db.Col[ProductStockDetailTable, int32]
	ProductID      db.Col[ProductStockDetailTable, int32]
	Quantity       db.Col[ProductStockDetailTable, int32]
	SubQuantity    db.Col[ProductStockDetailTable, int32]
	ExpirationDate db.Col[ProductStockDetailTable, int16]
	Updated        db.Col[ProductStockDetailTable, int32]
	UpdatedBy      db.Col[ProductStockDetailTable, int32]
	Created        db.Col[ProductStockDetailTable, int32]
	CreatedBy      db.Col[ProductStockDetailTable, int32]
	Status         db.Col[ProductStockDetailTable, int8]
}

func (e ProductStockDetailTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:                 "warehouse_product_stock_detail",
		Partition:            e.CompanyID,
		DisableUpdateCounter: true,
		// One detail row per stock-record + lot + serial.
		Keys: []db.Coln{e.ProductStockID, e.LotID, e.SerialNumber},
		Indexes: []db.Index{
			// Hash index for dedup lookups when resolving LotID from (Date, SupplierID, Name).
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.WarehouseID, e.Status.DecimalSize(1), e.Updated.DecimalSize(10)},
				KeepPart: true,
			},
		},
	}
}

// ProductStockLot is a dedupe-able lot catalog. Hash = (Date, SupplierID, Name).
type ProductStockLot struct {
	db.TableStruct[ProductStockLotTable, ProductStockLot]
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
	e.Hash = db.MakeKeyConcat(e.Date, e.SupplierID, e.Name)
}

type ProductStockLotTable struct {
	db.TableStruct[ProductStockLotTable, ProductStockLot]
	CompanyID        db.Col[ProductStockLotTable, int32]
	ID               db.Col[ProductStockLotTable, int32]
	Date             db.Col[ProductStockLotTable, int16]
	Name             db.Col[ProductStockLotTable, string]
	SupplierID       db.Col[ProductStockLotTable, int32]
	DeliveryNoteID   db.Col[ProductStockLotTable, int32]
	DeliveryNoteCode db.Col[ProductStockLotTable, string]
	Hash             db.Col[ProductStockLotTable, string]
	Created          db.Col[ProductStockLotTable, int32]
	CreatedBy        db.Col[ProductStockLotTable, int32]
	Status           db.Col[ProductStockLotTable, int8]
}

func (e ProductStockLotTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "product_stock_lot",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			// Hash index for dedup lookups when resolving LotID from (Date, SupplierID, Name).
			{Type: db.TypeGlobalIndex, Keys: []db.Coln{e.Hash}, KeepPart: true},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.Name}},
		},
	}
}

// DeliveryOrderNote is modeled but not wired up yet.
type DeliveryOrderNote struct {
	db.TableStruct[DeliveryOrderNoteTable, DeliveryOrderNote]
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
	db.TableStruct[DeliveryOrderNoteTable, DeliveryOrderNote]
	CompanyID   db.Col[DeliveryOrderNoteTable, int32]
	ID          db.Col[DeliveryOrderNoteTable, int32]
	Date        db.Col[DeliveryOrderNoteTable, int16]
	Code        db.Col[DeliveryOrderNoteTable, string]
	SupplierID  db.Col[DeliveryOrderNoteTable, int32]
	Description db.Col[DeliveryOrderNoteTable, string]
	Hash        db.Col[DeliveryOrderNoteTable, string]
	Created     db.Col[DeliveryOrderNoteTable, int32]
	CreatedBy   db.Col[DeliveryOrderNoteTable, int32]
	Updated     db.Col[DeliveryOrderNoteTable, int32]
	UpdatedBy   db.Col[DeliveryOrderNoteTable, int32]
	Status      db.Col[DeliveryOrderNoteTable, int8]
}

func (e DeliveryOrderNoteTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "delivery_order_note",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeGlobalIndex, Keys: []db.Coln{e.Hash}},
		},
	}
}

func (e *DeliveryOrderNote) SelfParse() {
	e.Hash = db.MakeKeyConcat(e.Date, e.SupplierID, e.Code)
}
