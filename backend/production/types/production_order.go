package types

import "app/db"

// ProductionOrder is one production, whatever made it (docs/MANUFACTURING_PLAN.md §3, M4): an order
// the user plans and receives in parts (Kind 1), or a daily close that settles a day of sales (Kind 2,
// born Terminada with one receipt). Both share the recipe, the consumption and valuation, the stock
// movement types 15/16 and the accounting. Its ID is the DocumentID of every stock movement it makes.
type ProductionOrder struct {
	db.TableStruct[ProductionOrderTable, ProductionOrder]
	CompanyID int32 `cb:"1"`
	ID        int32 `cb:"2"`
	Kind      int8  `cb:"3"`
	// TemplateID is the template the order came from, 0 if none. Series is how many series of it (M16).
	TemplateID int32 `cb:"4"`
	Series     int32 `cb:"5"`
	// WarehouseID is where the products enter (the close: where its sales left). SuppliesWarehouseID is
	// the company warehouse the supplies are reserved in and come from.
	WarehouseID         int32 `cb:"6"`
	SuppliesWarehouseID int32 `cb:"7"`
	// SubcontractorID is the workshop (client_provider Type 2) making it, 0 = made in-house (§8): its
	// supplies are reserved in SuppliesWarehouseID, sent to the workshop's warehouse (the one with that
	// SubcontractorID) and consumed there. SubcontractPricePerUnit is what was agreed with it, PEN cents
	// net of IGV.
	SubcontractorID         int32 `cb:"8"`
	SubcontractPricePerUnit int32 `cb:"9"`
	// Date is the creation day (UnixDay); the close's is the day it closed.
	Date         int16 `cb:"10"`
	PlannedStart int16 `cb:"11"`
	PlannedEnd   int16 `cb:"12"`
	// StageID is the board column (§9.4), 0 = none. Information only.
	StageID int32 `cb:"13"`
	// SupplyAdjustmentBasisPoints is signed: +1000 = use 10% more supplies than the recipe, −500 = 5%
	// less (M5). It applies on top of each supply's waste.
	SupplyAdjustmentBasisPoints int16                   `cb:"14"`
	Lines                       []ProductionOrderLine   `cb:"15"`
	Supplies                    []ProductionOrderSupply `cb:"16"`
	Labor                       []ProductionOrderLabor  `cb:"17"`
	// LaborRecordedAmount is Σ Amount of the order's labor entries (§9.2), PEN cents.
	LaborRecordedAmount int64               `cb:"18"`
	Receipts            []ProductionReceipt `cb:"19"`
	// SaleOrderIDs are the sales a close settled.
	SaleOrderIDs []int64 `cb:"20"`
	// IsStrict is company flag 15 when a close ran: a short supply blocked it instead of being shared.
	IsStrict  bool   `cb:"21"`
	Notes     string `cb:"22"`
	Status    int8   `json:"ss" cb:"ss,23"`
	Created   int32  `cb:"24"`
	CreatedBy int32  `cb:"25"`
	Updated   int32  `json:"upd" cb:"upd,26"`
	UpdatedBy int32  `cb:"27"`
	Version   int16  `json:"ver" cb:"ver,28"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,29"`
}

type ProductionOrderTable struct {
	db.TableStruct[ProductionOrderTable, ProductionOrder]
	CompanyID                   db.Col[*ProductionOrderTable, int32]
	ID                          db.Col[*ProductionOrderTable, int32]
	Kind                        db.Col[*ProductionOrderTable, int8]
	TemplateID                  db.Col[*ProductionOrderTable, int32]
	Series                      db.Col[*ProductionOrderTable, int32]
	WarehouseID                 db.Col[*ProductionOrderTable, int32]
	SuppliesWarehouseID         db.Col[*ProductionOrderTable, int32]
	SubcontractorID             db.Col[*ProductionOrderTable, int32]
	SubcontractPricePerUnit     db.Col[*ProductionOrderTable, int32]
	Date                        db.Col[*ProductionOrderTable, int16]
	PlannedStart                db.Col[*ProductionOrderTable, int16]
	PlannedEnd                  db.Col[*ProductionOrderTable, int16]
	StageID                     db.Col[*ProductionOrderTable, int32]
	SupplyAdjustmentBasisPoints db.Col[*ProductionOrderTable, int16]
	Lines                       db.Col[*ProductionOrderTable, []ProductionOrderLine]
	Supplies                    db.Col[*ProductionOrderTable, []ProductionOrderSupply]
	Labor                       db.Col[*ProductionOrderTable, []ProductionOrderLabor]
	LaborRecordedAmount         db.Col[*ProductionOrderTable, int64]
	Receipts                    db.Col[*ProductionOrderTable, []ProductionReceipt]
	SaleOrderIDs                db.ColSlice[*ProductionOrderTable, int64]
	IsStrict                    db.Col[*ProductionOrderTable, bool]
	Notes                       db.Col[*ProductionOrderTable, string]
	Status                      db.Col[*ProductionOrderTable, int8]
	Created                     db.Col[*ProductionOrderTable, int32]
	CreatedBy                   db.Col[*ProductionOrderTable, int32]
	Updated                     db.Col[*ProductionOrderTable, int32]
	UpdatedBy                   db.Col[*ProductionOrderTable, int32]
	Version                     db.Col[*ProductionOrderTable, int16]
}

func (e ProductionOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          101,
		Name:        "production_order",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 4}, {Col: e.Kind, Min: 1, Max: 2}},
		Indexes: []db.Index{
			// The orders list and the board, by status.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
			// The close history of a warehouse: Kind and WarehouseID pinned by equalities.
			{Type: db.TypeDelta, Keys: db.Cols(e.Kind, e.WarehouseID)},
		},
	}
}

// ProductionOrder.Kind values.
const (
	ProductionOrderKindManual     = int8(1)
	ProductionOrderKindDailyClose = int8(2)
)

// ProductionOrder.Status values (§1).
const (
	ProductionOrderStatusAnnulled   = int8(0) // no receipt and no material at a workshop; reservations released
	ProductionOrderStatusDraft      = int8(1) // editable; nothing planned or reserved
	ProductionOrderStatusReleased   = int8(2) // recipe exploded and frozen on the order, supplies reserved
	ProductionOrderStatusInProgress = int8(3) // a receipt, a material shipment or a labor entry happened
	ProductionOrderStatusFinished   = int8(4) // completed: reservations released, labor entries locked
)

// ProductionOrderLine is one (product, presentation) of the order. Presentations are the sizes (M3).
type ProductionOrderLine struct {
	ProductID      int32 `cb:"1"`
	PresentationID int16 `cb:"2"`
	PlannedUnits   int32 `cb:"3"`
	// ProducedUnits, ScrappedUnits and Value are Σ the receipts. A scrapped unit was made but defective,
	// and never entered stock. Value is PEN cents.
	ProducedUnits int32 `cb:"4"`
	ScrappedUnits int32 `cb:"5"`
	Value         int32 `cb:"6"`
	// CostSource is CostSourceStandard or CostSourceRecipe.
	CostSource int8 `cb:"7"`
	// ShortSupplies: some receipt consumed less than the recipe asked.
	ShortSupplies bool `cb:"8"`
}

// ProductionOrderSupply is one supply the order consumes, in sub-units of SubDivisor (the supply's
// own sub-unit).
type ProductionOrderSupply struct {
	SupplyID   int32 `cb:"1"`
	SubDivisor int16 `cb:"2"`
	// Planned is what the recipe asks, waste and the order's adjustment included.
	Planned int32 `cb:"3"`
	// Available is what the supplies warehouse held at release (the close: when it ran).
	Available int32 `cb:"4"`
	// Consumed is Σ the receipts.
	Consumed int32 `cb:"5"`
	// Reserved is what the order still holds of the supply on ProductStock (§5), in SuppliesWarehouseID.
	Reserved int32 `cb:"6"`
	// AtWorkshop is what a subcontracted order sent to its workshop and the workshop has not returned
	// or consumed (§8). It is reserved on the workshop warehouse's row while the order is open.
	AtWorkshop int32 `cb:"7"`
}

// ProductionOrderLabor is one labor type the order plans and records.
type ProductionOrderLabor struct {
	LaborTypeID    int32 `cb:"1"`
	PlannedMinutes int32 `cb:"2"`
	// PlannedCost is at the type's CostPerHour when released, PEN cents.
	PlannedCost int32 `cb:"3"`
	// RecordedMinutes and RecordedCost are Σ the order's labor entries of the type.
	RecordedMinutes int32 `cb:"4"`
	RecordedCost    int32 `cb:"5"`
}

// ProductionReceipt is one partial receipt of an order (§6): what entered stock and what it consumed.
type ProductionReceipt struct {
	// N is 1, 2, 3… within the order.
	N            int16                          `cb:"1"`
	Date         int16                          `cb:"2"`
	Lines        []ProductionReceiptLine        `cb:"3"`
	Consumptions []ProductionReceiptConsumption `cb:"4"`
	// The value's parts, PEN cents: the supplies the engine valued, the labor logged since the
	// previous receipt (the close: at standard), the workshop's price and the overhead per unit.
	SuppliesValue    int64 `cb:"5"`
	LaborValue       int64 `cb:"6"`
	SubcontractValue int64 `cb:"7"`
	OverheadValue    int64 `cb:"8"`
	CreatedBy        int32 `cb:"9"`
}

// ProductionReceiptLine is what a receipt produced of one (product, presentation).
type ProductionReceiptLine struct {
	ProductID      int32 `cb:"1"`
	PresentationID int16 `cb:"2"`
	Units          int32 `cb:"3"`
	Scrapped       int32 `cb:"4"`
	// LotID is the lot the units entered in, for a product that tracks lots (§10).
	LotID int32 `cb:"5"`
	// Value is what the units entered at, PEN cents.
	Value int32 `cb:"6"`
}

// ProductionReceiptConsumption is one consumption exit of a receipt: a supply taken from one bucket.
type ProductionReceiptConsumption struct {
	SupplyID     int32  `cb:"1"`
	LotID        int32  `cb:"2"`
	SerialNumber string `cb:"3"`
	SubUnits     int32  `cb:"4"`
	SubDivisor   int16  `cb:"5"`
	// Value is what the stock engine valued the exit at, PEN cents.
	Value int32 `cb:"6"`
}

// The cost sources of a production line.
const (
	CostSourceStandard = int8(1) // the product's ProductionStandardCost × units
	CostSourceRecipe   = int8(2) // its consumed supplies' value + labor + subcontract + overhead
)
