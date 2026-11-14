package types

import "app/db"

// TableOrder is the open bill of one table (docs/DINING_PLAN.md §2). Modeled on OnlineSaleOrder,
// it is not a sale: it spends no comprobante number and enters no summary until it is charged.
// A table has at most one open order.
type TableOrder struct {
	db.TableStruct[TableOrderTable, TableOrder]
	CompanyID    int32 `cb:"1"`
	ID           int32 `cb:"2"`
	LayoutID     int32 `cb:"3"`
	TableShapeID int16 `cb:"4"`
	// WarehouseID is the layout's, copied when the order opens: its reservations live there.
	WarehouseID int32            `cb:"5"`
	Lines       []TableOrderLine `cb:"6"`
	Guests      int8             `cb:"7"`
	// WaiterID is the user who opened the order.
	WaiterID int32 `cb:"8"`
	// SaleOrderID is the sale the order became when it was charged.
	SaleOrderID int64 `cb:"9"`
	// ExchangeRate is the rate (×1000) the last round converted a USD-priced product with, 0 when
	// none was; the sale inherits it, as from a web order.
	ExchangeRate int32   `cb:"10"`
	Status       int8    `json:"ss" cb:"ss,11"`
	Created      int32   `cb:"12"`
	Updated      int32   `json:"upd" cb:"upd,13"`
	UpdatedBy    int32   `cb:"14"`
	Version      int16   `json:"ver" cb:"ver,15"`
	Metadata     *string `json:"mdt" cb:"mdt,16"`
	// AnnulReason is why an open order was annulled (Status 0).
	AnnulReason string `cb:"17"`
}

type TableOrderTable struct {
	db.TableStruct[TableOrderTable, TableOrder]
	CompanyID    db.Col[*TableOrderTable, int32]
	ID           db.Col[*TableOrderTable, int32]
	LayoutID     db.Col[*TableOrderTable, int32]
	TableShapeID db.Col[*TableOrderTable, int16]
	WarehouseID  db.Col[*TableOrderTable, int32]
	Lines        db.Col[*TableOrderTable, []TableOrderLine]
	Guests       db.Col[*TableOrderTable, int8]
	WaiterID     db.Col[*TableOrderTable, int32]
	SaleOrderID  db.Col[*TableOrderTable, int64]
	ExchangeRate db.Col[*TableOrderTable, int32]
	Status       db.Col[*TableOrderTable, int8]
	Created      db.Col[*TableOrderTable, int32]
	Updated      db.Col[*TableOrderTable, int32]
	UpdatedBy    db.Col[*TableOrderTable, int32]
	Version      db.Col[*TableOrderTable, int16]
	AnnulReason  db.Col[*TableOrderTable, string]
}

func (e TableOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          98,
		Name:        "dining_table_order",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 2}},
		Indexes: []db.Index{
			// A floor map syncs the open orders of its layout only; a charged or annulled one
			// arrives as a removed ID.
			{Type: db.TypeDelta, Keys: db.Cols(e.LayoutID, e.Status)},
		},
	}
}

// TableOrder.Status values.
const (
	TableOrderAnnulled = int8(0)
	TableOrderOpen     = int8(1)
	TableOrderCharged  = int8(2)
)

// "Atención de Mesas" (access.toml) and its sub-access "Cobrar mesa", which charges and annuls an
// order.
const (
	AccessIDTableService   = int32(61)
	SubAccessIDChargeTable = int32(2)
)

// TableOrderLine is one dish or drink of a round.
type TableOrderLine struct {
	LineID         int16 `cb:"1"`
	Round          int8  `cb:"2"`
	ProductID      int32 `cb:"3"`
	PresentationID int16 `cb:"4"`
	// Quantity is packed as SaleOrderLine's (core.QuantityLineScale), SubDivisor the product's.
	Quantity   int32 `cb:"5"`
	SubDivisor int16 `cb:"6"`
	// Price and SubPrice are fixed when ordered, from the catalog, in soles.
	Price    int32 `cb:"7"`
	SubPrice int32 `cb:"8"`
	// StationID is resolved once, when the line is ordered: later routing changes affect new lines only.
	StationID int16  `cb:"9"`
	Note      string `cb:"10"` // "sin cebolla"
	FromQR    bool   `cb:"11"`
	// Status is a LineStatus* value.
	Status    int8  `cb:"12"`
	OrderedAt int32 `cb:"13"` // SUnixTime it reached the kitchen
	ReadyAt   int32 `cb:"14"`
	ServedAt  int32 `cb:"15"`
	UpdatedBy int32 `cb:"16"`
}

// TableOrderLine.Status values (§2.1).
const (
	LineStatusCancelled = int8(0)
	LineStatusToConfirm = int8(1) // a QR round waiting for a waiter
	LineStatusOrdered   = int8(2) // in the kitchen
	LineStatusReady     = int8(3) // waiting at its pickup point
	LineStatusServed    = int8(4)
)
