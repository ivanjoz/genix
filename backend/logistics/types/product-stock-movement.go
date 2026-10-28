package types

import (
	"app/db"
)

// WarehouseProductMovement is the append-only movement ledger.
// LotID (0 = no lot) and SerialNumber (empty = no serial) together tell whether
// the movement targets a ProductStockDetail row or the plain ProductStockV2.Quantity bucket.
type WarehouseProductMovement struct {
	db.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID int32 `cb:"CompanyID,1"`
	// ID packs Date(15 bits)+WarehouseID(17 bits)+Autoincrement(8 random bits) into an int64.
	ID                   int64  `cb:"ID,2"`
	SerialNumber         string `cb:"SerialNumber,3"`
	LotID                int32  `cb:"LotID,4"`
	WarehouseID          int32  `cb:"WarehouseID,5"`
	WarehouseRefID       int32  `cb:"WarehouseRefID,6"`
	WarehouseRefQuantity int32  `cb:"WarehouseRefQuantity,7"`
	Date                 int16  `cb:"Date,8"`
	DocumentID           int64  `cb:"DocumentID,9"`
	ProductID            int32  `cb:"ProductID,10"`
	PresentationID       int16  `cb:"PresentationID,11"`
	Quantity             int32  `cb:"Quantity,12"`
	WarehouseQuantity    int32  `cb:"WarehouseQuantity,13"`
	SubQuantity          int32  `cb:"SubQuantity,14"`
	// SubDivisor is the divisor SubQuantity was recorded in. It must live on the ledger row
	// itself: RecalcProductStockByMovements replays the whole history, and without it a
	// SUM(SubQuantity) spanning a divisor refinement would add sixths to twelfths.
	SubDivisor    int16             `cb:"SubDivisor,15"`
	MonetaryValue int32             `cb:"MonetaryValue,16"`
	Type          StockMovementType `cb:"Type,17"`
	Created       int32             `cb:"Created,18"`
	CreatedBy     int32             `cb:"CreatedBy,19"`
	UpdateCounter int32             `cb:"UpdateCounter,20"`
}

type WarehouseProductMovementTable struct {
	db.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID            db.Col[*WarehouseProductMovementTable, int32]
	ID                   db.Col[*WarehouseProductMovementTable, int64]
	SerialNumber         db.Col[*WarehouseProductMovementTable, string]
	LotID                db.Col[*WarehouseProductMovementTable, int32]
	WarehouseID          db.Col[*WarehouseProductMovementTable, int32]
	WarehouseRefID       db.Col[*WarehouseProductMovementTable, int32]
	WarehouseRefQuantity db.Col[*WarehouseProductMovementTable, int32]
	DocumentID           db.Col[*WarehouseProductMovementTable, int64]
	ProductID            db.Col[*WarehouseProductMovementTable, int32]
	PresentationID       db.Col[*WarehouseProductMovementTable, int16]
	Quantity             db.Col[*WarehouseProductMovementTable, int32]
	WarehouseQuantity    db.Col[*WarehouseProductMovementTable, int32]
	SubQuantity          db.Col[*WarehouseProductMovementTable, int32]
	SubDivisor           db.Col[*WarehouseProductMovementTable, int16]
	Type                 db.Col[*WarehouseProductMovementTable, StockMovementType]
	Created              db.Col[*WarehouseProductMovementTable, int32]
	MonetaryValue        db.Col[*WarehouseProductMovementTable, int32]
	CreatedBy            db.Col[*WarehouseProductMovementTable, int32]
	Date                 db.Col[*WarehouseProductMovementTable, int16]
	UpdateCounter        db.Col[*WarehouseProductMovementTable, int32]
}

func (e WarehouseProductMovementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        36,
		Name:      "warehouse_product_movement",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		KeyIntPacking: db.Cols(
			e.Date.Size(15), e.WarehouseID.Size(17), e.Autoincrement(8),
		),
		AutoincrementPart: e.Date,
		Indexes: []db.Index{
			{
				Type: db.TypeInheritFromKey, Keys: db.Cols(e.Date), UseIndexGroup: true,
			},
			{
				Type: db.TypeInheritFromKey, Keys: db.Cols(e.Date, e.WarehouseID), UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: db.Cols(e.SerialNumber),
			},
			{
				Type: db.TypeLocalIndex, Keys: db.Cols(e.LotID), UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: db.Cols(e.DocumentID), UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: db.Cols(e.Date, e.Type), UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: db.Cols(e.Date, e.Type, e.WarehouseID), UseIndexGroup: true,
			},
			{
				Type: db.TypeLocalIndex, Keys: db.Cols(e.Date, e.ProductID), UseIndexGroup: true,
			},
			// Backs the grouped movement report. SubDivisor joins the key so a SUM(SubQuantity)
			// spanning a divisor refinement never adds sixths to twelfths; the caller folds the
			// per-divisor groups back together. Out of the 64-bit budget: ProductID 27 (134M products
			// per company), Type 7 (127 StockMovementTypes — the catalog is never capped by this view),
			// SubDivisor 14 (core.MaxQuantityDivisor fits), and Date takes the remaining 16.
			{
				Type: db.TypeView,
				Keys: db.Cols(e.Date, e.ProductID.Size(27), e.Type.Size(7), e.SubDivisor.Size(14)),
				Cols: db.Cols(e.Quantity, e.SubQuantity),
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
	Type            StockMovementType
	SerialNumber    string
	LotName         string
	LotID           int32
	SupplierID      int32
	WarehouseID     int32
	DestWarehouseID int32
	Quantity        int32
	SubQuantity     int32
	SubDivisor      int16
	// Price is the unit cost of an inflow, in PEN cents net of recoverable IGV. It is ignored on
	// an outflow: stock leaves at the row's AverageCost, not at a price someone typed.
	Price      int32
	DocumentID int64
	// The company's manual-stock policy (company flags 6 and 9). They are judged on the delta,
	// which only the engine knows for a "set stock to X" movement, and inside its lock, so the
	// check and the write see the same balance.
	RejectInbound      bool // the company only lets stock in through a purchase order
	RequireInboundCost bool // every inflow must carry a Price

	// transferPeerWarehouseID is set by expandStockTransfers on both legs of a transfer: the
	// other warehouse, written to the ledger as WarehouseRefID.
	transferPeerWarehouseID int32
}

// HasDetail reports whether the movement targets a ProductStockDetail row
// (i.e. the non-free bucket keyed by LotID and/or SerialNumber).
func (e *InternalMovement) HasDetail() bool {
	return e.SerialNumber != "" || e.LotID > 0 || e.LotName != ""
}
