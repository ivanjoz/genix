package types

import (
	"app/core"
	"app/db"
)

// RestampReceptionAccounts rewrites the PurchaseAccountCode of the receptions of one purchase after
// its account was reclassified: accountOfProduct is the account each received product now posts to.
// receptionType is the reception's movement type (3 an order, 4 an inventory expense), since
// DocumentID alone is shared between the registers.
func RestampReceptionAccounts(
	companyID int32, documentID int64, receptionType StockMovementType, accountOfProduct func(productID int32) int32,
) error {
	stored := []WarehouseProductMovement{}
	query := db.Query(&stored)
	query.Select(query.ID, query.Type, query.ProductID, query.PurchaseAccountCode).
		CompanyID.Equals(companyID).DocumentID.Equals(documentID)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los ingresos de la compra:", documentID, err)
	}
	changed := []WarehouseProductMovement{}
	for _, movement := range stored {
		accountCode := accountOfProduct(movement.ProductID)
		if movement.Type != receptionType || movement.PurchaseAccountCode == accountCode {
			continue
		}
		movement.CompanyID = companyID
		movement.PurchaseAccountCode = accountCode
		changed = append(changed, movement)
	}
	if len(changed) == 0 {
		return nil
	}
	table := db.TableOf[WarehouseProductMovement]()
	if err := db.Update(&changed, table.PurchaseAccountCode); err != nil {
		return core.Err("Error al actualizar la cuenta de los ingresos de la compra:", documentID, err)
	}
	return nil
}

// WarehouseProductMovement is the append-only movement ledger.
// LotID (0 = no lot) and SerialNumber (empty = no serial) together tell whether
// the movement targets a ProductStockDetail row or the plain ProductStockV2.Quantity bucket.
type WarehouseProductMovement struct {
	db.TableStruct[WarehouseProductMovementTable, WarehouseProductMovement]
	CompanyID int32 `cb:"1"`
	// ID packs Date(15 bits)+WarehouseID(17 bits)+Autoincrement(8 random bits) into an int64.
	ID                   int64  `cb:"2"`
	SerialNumber         string `cb:"3"`
	LotID                int32  `cb:"4"`
	WarehouseID          int32  `cb:"5"`
	WarehouseRefID       int32  `cb:"6"`
	WarehouseRefQuantity int32  `cb:"7"`
	Date                 int16  `cb:"8"`
	DocumentID           int64  `cb:"9"`
	ProductID            int32  `cb:"10"`
	PresentationID       int16  `cb:"11"`
	Quantity             int32  `cb:"12"`
	WarehouseQuantity    int32  `cb:"13"`
	SubQuantity          int32  `cb:"14"`
	// SubDivisor is the divisor SubQuantity was recorded in. It must live on the ledger row
	// itself: RecalcProductStockByMovements replays the whole history, and without it a
	// SUM(SubQuantity) spanning a divisor refinement would add sixths to twelfths.
	SubDivisor    int16             `cb:"15"`
	MonetaryValue int32             `cb:"16"`
	Type          StockMovementType `cb:"17"`
	Created       int32             `cb:"18"`
	CreatedBy     int32             `cb:"19"`
	// Period is the YYMM of Date (core.PeriodOfDay): the pinned key of the period delta index the
	// accounting operations sync a month through.
	Period int16 `cb:"21"`
	// PurchaseAccountCode is the PCGE account the purchase behind a reception posted to, stamped
	// when the reception is written: an order line's AccountCode, else 6011, on a purchase
	// reception (3); the expense's AccountCode on a supplies purchase (4); 0 on every other type.
	// The accounting operations post the reception to the stock / variation pair of this account
	// without reading the purchase back.
	PurchaseAccountCode int32 `cb:"22"`
	Version             int16 `json:"ver" cb:"ver,23"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,24"`
	// Updated is the ORM-stamped write tick: the period delta index packs it into its key and a
	// Delta() read hands it to the client as the cursor.
	Updated int32 `json:"upd" cb:"upd,25"`
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
	Period               db.Col[*WarehouseProductMovementTable, int16]
	PurchaseAccountCode  db.Col[*WarehouseProductMovementTable, int32]
	Version              db.Col[*WarehouseProductMovementTable, int16]
	Updated              db.Col[*WarehouseProductMovementTable, int32]
}

func (e WarehouseProductMovementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        36,
		Name:      "warehouse_product_movement",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		KeyIntPacking: db.Cols(
			e.Date.Size(16), e.WarehouseID.Size(16), e.Type.Size(6), e.Autoincrement(6),
		),
		AutoincrementPart: e.Date,
		// Size the slots of the period delta index. Type leaves room for new movement types.
		FixedValues: []db.FixedValues{
			{Col: e.Period, Min: 0, Max: 9912},
			{Col: e.Type, Min: 0, Max: 63},
		},
		Indexes: []db.Index{
			// One log per accounting period: GET.accounting-operations syncs a month's receptions,
			// deliveries and returns through it. Movements are never deleted, so Type is just the
			// sync filter a first sync lists. The view copies only what that accounting read projects
			// (OperationStockMovement): every sale line writes a ledger row, and each copied column
			// is paid again on that write.
			{
				Type: db.TypeDelta, Keys: db.Cols(e.Period, e.Type),
				Cols: db.Cols(e.Date, e.DocumentID, e.ProductID, e.MonetaryValue, e.PurchaseAccountCode),
			},
			{
				Type: db.TypeCacheGroupFromKey, Keys: db.Cols(e.Date),
			},
			{
				Type: db.TypeCacheGroupFromKey, Keys: db.Cols(e.Date, e.WarehouseID),
			},
			{
				Type: db.TypeCacheGroupFromKey, Keys: db.Cols(e.Date, e.WarehouseID, e.Type),
			},
			{
				Type: db.TypeLocalIndex, Keys: db.Cols(e.SerialNumber),
			},
			{
				Type: db.TypeCacheGroup, Keys: db.Cols(e.LotID),
			},
			{
				Type: db.TypeCacheGroup, Keys: db.Cols(e.DocumentID),
			},
			// Type is not a prefix of the ID after Date: the group reads the (Date) range and filters
			// Type in memory, so a sale writes nothing for it.
			{
				Type: db.TypeCacheGroupNoIndex, Keys: db.Cols(e.Date, e.Type),
			},
			{
				Type: db.TypeCacheGroup, Keys: db.Cols(e.Date, e.ProductID),
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
	// an outflow: stock leaves at its share of the row's StockValue, not at a price someone typed.
	// An inflow with no Price enters at the row's AverageCost.
	Price int32
	// InflowValue is the inflow's total value when the caller knows it exactly — a sale annulment
	// returns the value its deliveries took out. It wins over Price: a unit price re-multiplied
	// by the quantity would round, and the returned value would no longer cancel the delivered one.
	InflowValue int32
	DocumentID  int64
	// PurchaseAccountCode is the account the purchase behind a reception posted to, written on the
	// ledger row (WarehouseProductMovement.PurchaseAccountCode). Set by the reception's caller.
	PurchaseAccountCode int32
	// The manual-stock policy: company flag 6, and opening stock's cost. They are judged on the
	// delta, which only the engine knows for a "set stock to X" movement, and against the stock row it
	// locked, so the check and the write see the same balance.
	RejectInbound      bool // the company only lets stock in through a purchase order
	RequireInboundCost bool // the inflow must carry a Price even if the row has an AverageCost

	// transferPeerWarehouseID is set by expandStockTransfers on both legs of a transfer: the
	// other warehouse, written to the ledger as WarehouseRefID.
	transferPeerWarehouseID int32
}

// HasDetail reports whether the movement targets a ProductStockDetail row
// (i.e. the non-free bucket keyed by LotID and/or SerialNumber).
func (e *InternalMovement) HasDetail() bool {
	return e.SerialNumber != "" || e.LotID > 0 || e.LotName != ""
}
