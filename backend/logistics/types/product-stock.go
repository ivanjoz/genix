package types

import (
	"app/core"
	"app/db"
)

// ProductStock is the current stock row per (Warehouse, Product, Presentation).
// - Quantity         : stock with no lot/serial tracking (the "free" bucket).
// - DetailQuantity   : sum of ProductStockDetail.Quantity rows linked to this row.
// - DetailComputed*  : async-precomputed snapshot (populated by a separate job).
//
// Every Quantity has a SubQuantity companion: together they are the core.Quantity pair
// {Units, Sub} read at SubDivisor. Sub is NOT normalized — it may exceed the divisor and go
// negative (a shop that buys boxes and sells candies runs a negative sub balance against
// whole-unit inflows), which is what lets movements accumulate with a plain +.
type ProductStock struct {
	db.TableStruct[ProductStockTable, ProductStock]
	ID                int64 `cb:"1"`
	CompanyID         int32 `cb:"2"`
	WarehouseID       int32 `cb:"3"`
	ProductID         int32 `cb:"4"`
	PresentationID    int16 `cb:"5"`
	Quantity          int32 `cb:"6"`
	SubQuantity       int32 `cb:"7"`
	DetailQuantity    int32 `cb:"8"`
	DetailSubQuantity int32 `cb:"9"`
	// SubDivisor is the divisor every SubQuantity on this row (and on its detail rows, which
	// are the same product) is expressed in. Kept here rather than read from the product so a
	// divisor refinement converts this row on the fly instead of rewriting history.
	SubDivisor                int16   `cb:"10"`
	DetailComputedDate        int16   `cb:"11"`
	DetailComputedQuantity    int32   `cb:"12"`
	DetailComputedSubQuantity int32   `cb:"13"`
	LastPricesPrice           []int32 `cb:"14"`
	LastPricesQuantity        []int32 `cb:"15"`
	// AverageCost is the moving weighted average cost of one whole unit, in PEN cents, net of
	// recoverable IGV. ApplyMovimientos moves it on every costed inflow; outflows and uncosted
	// inflows leave it as it is. It is an operational figure: a purchase whose comprobante is
	// registered after the goods arrive does not restate it — the accounting valuation does.
	AverageCost int32 `cb:"16"`
	StockStatus int8  `cb:"17"`

	Created   int32 `cb:"18"`
	CreatedBy int32 `cb:"19"`
	Updated   int32 `json:"upd" cb:"upd,20"`
	Version   int16 `json:"ver" cb:"ver,21"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,22"`
	UpdatedBy int32   `cb:"23"`
	Status    int8    `json:"ss" cb:"ss,24"`
}

// Derive Status from the two live buckets (Quantity + DetailQuantity). The sub-unit halves
// count: a row holding no whole units but four loose candies still has stock, and a Status
// of 0 would evict it from the delta sync as if the warehouse were empty.
func (e *ProductStock) SelfParse() {
	freeBucket := core.Quantity{Units: e.Quantity, Sub: e.SubQuantity}
	detailBucket := core.Quantity{Units: e.DetailQuantity, Sub: e.DetailSubQuantity}
	e.Status = core.If(freeBucket.IsZero() && detailBucket.IsZero(), int8(0), int8(1))
}

type ProductStockTable struct {
	db.TableStruct[ProductStockTable, ProductStock]
	CompanyID                 db.Col[*ProductStockTable, int32]
	ID                        db.Col[*ProductStockTable, int64]
	WarehouseID               db.Col[*ProductStockTable, int32]
	ProductID                 db.Col[*ProductStockTable, int32]
	PresentationID            db.Col[*ProductStockTable, int16]
	Quantity                  db.Col[*ProductStockTable, int32]
	SubQuantity               db.Col[*ProductStockTable, int32]
	DetailQuantity            db.Col[*ProductStockTable, int32]
	DetailSubQuantity         db.Col[*ProductStockTable, int32]
	SubDivisor                db.Col[*ProductStockTable, int16]
	DetailComputedDate        db.Col[*ProductStockTable, int16]
	DetailComputedQuantity    db.Col[*ProductStockTable, int32]
	DetailComputedSubQuantity db.Col[*ProductStockTable, int32]
	LastPricesPrice           db.ColSlice[*ProductStockTable, int32]
	LastPricesQuantity        db.ColSlice[*ProductStockTable, int32]
	AverageCost               db.Col[*ProductStockTable, int32]
	StockStatus               db.Col[*ProductStockTable, int8]
	Created                   db.Col[*ProductStockTable, int32]
	CreatedBy                 db.Col[*ProductStockTable, int32]
	Updated                   db.Col[*ProductStockTable, int32]
	Version                   db.Col[*ProductStockTable, int16]
	UpdatedBy                 db.Col[*ProductStockTable, int32]
	Status                    db.Col[*ProductStockTable, int8]
}

func (e ProductStockTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        37,
		Name:      "warehouse_product_stock",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		// ID packs (WarehouseID, ProductID, PresentationID) into the single int64 key.
		KeyIntPacking: db.Cols(
			e.WarehouseID.Size(17),
			e.ProductID.Size(30),
			e.PresentationID.Size(14),
		),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		// Two delta indexes for the two read shapes. Delta() picks by what the query already pinned:
		// one warehouse's stock routes to the first, a whole-company sweep to the second. WarehouseID
		// reuses the 17-bit width the packed ID key gives it.
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.WarehouseID.Size(17), e.Status)},
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// ProductStockDetail rows track stock by Lot and/or SerialNumber.
// Composite key is (ProductStockID, LotID, SerialNumber), so a row exists per
// distinct (warehouse-product-presentation, lot, serial) combination.
type ProductStockDetail struct {
	db.TableStruct[ProductStockDetailTable, ProductStockDetail]
	CompanyID      int32  `cb:"1"`
	ProductStockID int64  `cb:"2"`
	LotID          int32  `cb:"3"`
	SerialNumber   string `cb:"4"`
	WarehouseID    int32  `cb:"5"`
	ProductID      int32  `cb:"6"`
	Quantity       int32  `cb:"7"`
	SubQuantity    int32  `cb:"8"`
	ExpirationDate int16  `cb:"9"`

	Updated   int32 `json:"upd" cb:"upd,10"`
	UpdatedBy int32 `cb:"11"`
	Created   int32 `cb:"12"`
	CreatedBy int32 `cb:"13"`
	Status    int8  `json:"ss" cb:"ss,14"`
}

type ProductStockDetailTable struct {
	db.TableStruct[ProductStockDetailTable, ProductStockDetail]
	CompanyID      db.Col[*ProductStockDetailTable, int32]
	ProductStockID db.Col[*ProductStockDetailTable, int64]
	LotID          db.Col[*ProductStockDetailTable, int32]
	SerialNumber   db.Col[*ProductStockDetailTable, string]
	WarehouseID    db.Col[*ProductStockDetailTable, int32]
	ProductID      db.Col[*ProductStockDetailTable, int32]
	Quantity       db.Col[*ProductStockDetailTable, int32]
	SubQuantity    db.Col[*ProductStockDetailTable, int32]
	ExpirationDate db.Col[*ProductStockDetailTable, int16]
	Updated        db.Col[*ProductStockDetailTable, int32]
	UpdatedBy      db.Col[*ProductStockDetailTable, int32]
	Created        db.Col[*ProductStockDetailTable, int32]
	CreatedBy      db.Col[*ProductStockDetailTable, int32]
	Status         db.Col[*ProductStockDetailTable, int8]
}

func (e ProductStockDetailTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        38,
		Name:      "warehouse_product_stock_detail",
		Partition: e.CompanyID,
		// One detail row per stock-record + lot + serial.
		Keys: db.Cols(e.ProductStockID, e.LotID, e.SerialNumber),
		// A composite key, which a delta log can't name: the route answers one warehouse's active
		// details whole, with the replace Metadata (core.WithDeltaReplace).
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: db.Cols(e.WarehouseID, e.Status)},
		},
	}
}

// ProductStockLot is a dedupe-able lot catalog. Hash = (Date, SupplierID, Name).
type ProductStockLot struct {
	db.TableStruct[ProductStockLotTable, ProductStockLot]
	CompanyID        int32  `cb:"1"`
	ID               int32  `cb:"2"`
	Date             int16  `cb:"3"`
	Name             string `cb:"4"`
	SupplierID       int32  `cb:"5"`
	DeliveryNoteID   int32  `cb:"6"`
	DeliveryNoteCode string `cb:"7"`
	Hash             string `cb:"8"`

	Created   int32 `cb:"9"`
	CreatedBy int32 `cb:"10"`
	Status    int8  `json:"ss" cb:"ss,11"`
}

func (e *ProductStockLot) SelfParse() {
	e.Hash = db.MakeKeyConcat(e.Date, e.SupplierID, e.Name)
}

type ProductStockLotTable struct {
	db.TableStruct[ProductStockLotTable, ProductStockLot]
	CompanyID        db.Col[*ProductStockLotTable, int32]
	ID               db.Col[*ProductStockLotTable, int32]
	Date             db.Col[*ProductStockLotTable, int16]
	Name             db.Col[*ProductStockLotTable, string]
	SupplierID       db.Col[*ProductStockLotTable, int32]
	DeliveryNoteID   db.Col[*ProductStockLotTable, int32]
	DeliveryNoteCode db.Col[*ProductStockLotTable, string]
	Hash             db.Col[*ProductStockLotTable, string]
	Created          db.Col[*ProductStockLotTable, int32]
	CreatedBy        db.Col[*ProductStockLotTable, int32]
	Status           db.Col[*ProductStockLotTable, int8]
}

func (e ProductStockLotTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        20,
		Name:      "product_stock_lot",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			// Hash index for dedup lookups when resolving LotID from (Date, SupplierID, Name).
			{Type: db.TypeGlobalIndex, Keys: db.Cols(e.Hash)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.Name)},
		},
	}
}

// DeliveryOrderNote is modeled but not wired up yet.
type DeliveryOrderNote struct {
	db.TableStruct[DeliveryOrderNoteTable, DeliveryOrderNote]
	CompanyID   int32  `cb:"1"`
	ID          int32  `cb:"2"`
	Date        int16  `cb:"3"`
	Code        string `cb:"4"`
	SupplierID  int32  `cb:"5"`
	Description string `cb:"6"`
	Hash        string `cb:"7"`

	Created   int32 `cb:"8"`
	CreatedBy int32 `cb:"9"`
	Updated   int32 `cb:"10"`
	UpdatedBy int32 `cb:"11"`
	Status    int8  `json:"ss" cb:"ss,12"`
}

type DeliveryOrderNoteTable struct {
	db.TableStruct[DeliveryOrderNoteTable, DeliveryOrderNote]
	CompanyID   db.Col[*DeliveryOrderNoteTable, int32]
	ID          db.Col[*DeliveryOrderNoteTable, int32]
	Date        db.Col[*DeliveryOrderNoteTable, int16]
	Code        db.Col[*DeliveryOrderNoteTable, string]
	SupplierID  db.Col[*DeliveryOrderNoteTable, int32]
	Description db.Col[*DeliveryOrderNoteTable, string]
	Hash        db.Col[*DeliveryOrderNoteTable, string]
	Created     db.Col[*DeliveryOrderNoteTable, int32]
	CreatedBy   db.Col[*DeliveryOrderNoteTable, int32]
	Updated     db.Col[*DeliveryOrderNoteTable, int32]
	UpdatedBy   db.Col[*DeliveryOrderNoteTable, int32]
	Status      db.Col[*DeliveryOrderNoteTable, int8]
}

func (e DeliveryOrderNoteTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        11,
		Name:      "delivery_order_note",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			{Type: db.TypeGlobalIndex, Keys: db.Cols(e.Hash)},
		},
	}
}

func (e *DeliveryOrderNote) SelfParse() {
	e.Hash = db.MakeKeyConcat(e.Date, e.SupplierID, e.Code)
}
