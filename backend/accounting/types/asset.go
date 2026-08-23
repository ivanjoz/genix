package types

import "app/db"

// Payment lifecycle of the acquisition, tracked on the asset itself. An asset is not an
// expense: buying one is cash turning into a balance-sheet item, not a cost, so the money
// owed for it lives here rather than in the expense register. Only depreciation is an expense.
const (
	AssetPaymentNone    int8 = 0 // Donated or contributed — no cash was ever owed.
	AssetPaymentPending int8 = 1 // Owed, in full or in part.
	AssetPaymentPaid    int8 = 2 // Settled.
)

// Asset lifecycle. An asset that is fully depreciated is still owned and still in the
// warehouse; only disposal removes it from the balance sheet.
const (
	AssetStatusRemoved          int8 = 0
	AssetStatusActive           int8 = 1
	AssetStatusFullyDepreciated int8 = 2
	AssetStatusDisposed         int8 = 3
)

// Asset is the accounting overlay on a stock row. The asset's *existence* is its stock —
// a ProductStock bucket when it is tracked as a group, a ProductStockDetail row when it
// has a serial — and this table holds only what stock cannot:
//
//   - Stock has no monetary or acquisition-date columns.
//   - The stock key packs WarehouseID, so a transfer would change the row's identity.
//   - ProductStock.SelfParse zeroes Status at quantity 0, so disposal would delete the
//     history the balance sheet still needs to read.
//
// Hence an autoincrement ID of its own, with (ProductID, SerialNumber) as the stable
// pointer back at the stock row and WarehouseID as a mutable mirror of where it is now.
type Asset struct {
	db.TableStruct[AssetTable, Asset]
	CompanyID int32 `json:",omitempty"`
	ID        int32 `json:",omitempty"`
	// The supply this instantiates: a Product row with Status = 2 and DepreciationMonths > 0.
	ProductID int32 `json:",omitempty"`
	// Empty when the acquisition is tracked as a group rather than per unit. Whether a serial
	// was entered is what decides granularity: one Asset per unit, or one per acquisition lot.
	SerialNumber string `json:",omitempty"`
	Quantity     int32  `json:",omitempty"` // 1 when serial-tracked; N for a grouped lot.
	WarehouseID  int32  `json:",omitempty"`
	Name         string `json:",omitempty"`
	Description  string `json:",omitempty"`
	SupplierID   int32  `json:",omitempty"`
	// The acquisition's own money, kept here rather than in an Expense row. 0 = donated:
	// nothing was owed, and PaymentStatus stays AssetPaymentNone.
	PurchaseAmount int32 `json:",omitempty"`
	PaidAmount     int32 `json:",omitempty"` // Server-maintained sum of payments applied.
	PaymentStatus  int8  `json:",omitempty"`
	DueDate        int16 `json:",omitempty"`
	// The real acquisition date, which is not Created: a donated or backdated asset is
	// registered long after it was acquired, and the schedule has to run from the latter.
	AcquisitionDate int16 `json:",omitempty"`
	// Book value at acquisition, in cents. Independent of what was paid.
	AcquisitionValue int32 `json:",omitempty"`
	// Copied from the Product at acquisition, so editing the catalog later never rewrites
	// the schedule of an asset already in service.
	DepreciationMonths int16 `json:",omitempty"`
	CurrencyType       int8  `json:",omitempty"` // 1 = PEN, 2 = USD.
	// Server-maintained running sum, and the UnixDay of the last period generated — which is
	// what makes lazy generation idempotent.
	AccumulatedDepreciation int32 `json:",omitempty"`
	LastDepreciationDate    int16 `json:",omitempty"`
	DisposalDate            int16 `json:",omitempty"` // 0 = still held.

	Status         int8  `json:"ss,omitempty"`
	Updated        int32 `json:"upd,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
	Created        int32 `json:",omitempty"`
	CreatedBy      int32 `json:",omitempty"`
}

type AssetTable struct {
	db.TableStruct[AssetTable, Asset]
	CompanyID               db.Col[AssetTable, int32]
	ID                      db.Col[AssetTable, int32]
	ProductID               db.Col[AssetTable, int32]
	SerialNumber            db.Col[AssetTable, string]
	Quantity                db.Col[AssetTable, int32]
	WarehouseID             db.Col[AssetTable, int32]
	Name                    db.Col[AssetTable, string]
	Description             db.Col[AssetTable, string]
	SupplierID              db.Col[AssetTable, int32]
	PurchaseAmount          db.Col[AssetTable, int32]
	PaidAmount              db.Col[AssetTable, int32]
	PaymentStatus           db.Col[AssetTable, int8]
	DueDate                 db.Col[AssetTable, int16]
	AcquisitionDate         db.Col[AssetTable, int16]
	AcquisitionValue        db.Col[AssetTable, int32]
	DepreciationMonths      db.Col[AssetTable, int16]
	CurrencyType            db.Col[AssetTable, int8]
	AccumulatedDepreciation db.Col[AssetTable, int32]
	LastDepreciationDate    db.Col[AssetTable, int16]
	DisposalDate            db.Col[AssetTable, int16]
	Status                  db.Col[AssetTable, int8]
	Updated                 db.Col[AssetTable, int32]
	UpdatedVersion          db.Col[AssetTable, int32]
	UpdatedBy               db.Col[AssetTable, int32]
	Created                 db.Col[AssetTable, int32]
	CreatedBy               db.Col[AssetTable, int32]
}

func (e AssetTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                 52,
		Name:               "accounting_asset",
		Partition:          e.CompanyID,
		SaveUpdatedVersion: true,
		Keys:               db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Min: 0, Max: 3},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
			// "Which assets exist for this supply?" — the Activos list groups by catalog row.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.ProductID)},
			// Serial lookup, for resolving an asset from the stock detail row it shadows.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.SerialNumber)},
		},
	}
}

// PendingAmount is what is still owed on the acquisition. Nothing is owed on a donation.
func (e *Asset) PendingAmount() int32 {
	pending := e.PurchaseAmount - e.PaidAmount
	if pending < 0 {
		return 0
	}
	return pending
}

// BookValue is what the asset is still worth: acquisition cost less everything depreciated
// so far. Never negative — the final period is sized to land exactly on zero.
func (e *Asset) BookValue() int32 {
	remaining := e.AcquisitionValue - e.AccumulatedDepreciation
	if remaining < 0 {
		return 0
	}
	return remaining
}
