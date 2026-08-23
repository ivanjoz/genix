package types

import "app/db"

// ExpenseScheduled is the recurring template. It only describes the cadence and the
// default amount; per-period payment state lives in the generated Expense rows.
type ExpenseScheduled struct {
	db.TableStruct[ExpenseScheduledTable, ExpenseScheduled]
	CompanyID      int32  `json:",omitempty"`
	ID             int32  `json:",omitempty"`
	Name           string `json:",omitempty"`
	Description    string `json:",omitempty"`
	CategoryID     int8   `json:",omitempty"` // Static expense category (code-defined list).
	SupplierID     int32  `json:",omitempty"` // Optional — who is paid.
	CurrencyType   int8   `json:",omitempty"` // 1 = PEN, 2 = USD.
	Amount         int32  `json:",omitempty"` // Default expected amount per period, in cents.
	Frequency      int16  `json:",omitempty"` // Packed cadence code CDD: cadence*100 + day.
	StartDate      int16  `json:",omitempty"` // UnixDay the schedule begins; anchors the month for N-monthly/yearly.
	EndDate        int16  `json:",omitempty"` // UnixDay the schedule stops (0 = open-ended).
	Status         int8   `json:"ss,omitempty"`
	Updated        int32  `json:"upd,omitempty"`
	UpdatedVersion int32  `json:"upv,omitempty"`
	UpdatedBy      int32  `json:",omitempty"`
	Created        int32  `json:",omitempty"`
	CreatedBy      int32  `json:",omitempty"`
}

type ExpenseScheduledTable struct {
	db.TableStruct[ExpenseScheduledTable, ExpenseScheduled]
	CompanyID      db.Col[ExpenseScheduledTable, int32]
	ID             db.Col[ExpenseScheduledTable, int32]
	Name           db.Col[ExpenseScheduledTable, string]
	Description    db.Col[ExpenseScheduledTable, string]
	CategoryID     db.Col[ExpenseScheduledTable, int8]
	SupplierID     db.Col[ExpenseScheduledTable, int32]
	CurrencyType   db.Col[ExpenseScheduledTable, int8]
	Amount         db.Col[ExpenseScheduledTable, int32]
	Frequency      db.Col[ExpenseScheduledTable, int16]
	StartDate      db.Col[ExpenseScheduledTable, int16]
	EndDate        db.Col[ExpenseScheduledTable, int16]
	Status         db.Col[ExpenseScheduledTable, int8]
	Updated        db.Col[ExpenseScheduledTable, int32]
	UpdatedVersion db.Col[ExpenseScheduledTable, int32]
	UpdatedBy      db.Col[ExpenseScheduledTable, int32]
	Created        db.Col[ExpenseScheduledTable, int32]
	CreatedBy      db.Col[ExpenseScheduledTable, int32]
}

func (e ExpenseScheduledTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           14,
		Name:         "expenses_scheduled",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// Expense.Type says what the money became, which is what keeps a purchase off the P&L when
// it belongs on the balance sheet instead.
//
// Buying a fixed asset is deliberately absent: an asset is not an expense at all, so it
// writes no row here. Its cost, its payment state and its supplier live on accounting.Asset,
// and the only thing it ever contributes to this table is its monthly depreciation.
const (
	ExpenseTypeSimple       int8 = 1 // Straight to P&L.
	ExpenseTypeInventory    int8 = 2 // Cash → stock. Becomes cost when consumed or sold.
	ExpenseTypeDepreciation int8 = 4 // Non-cash. One period of an asset's depreciation.
)

// ExpenseStatusPosted is the lifecycle slot for entries that are never paid, because no
// cash is involved. Type-4 depreciation rows live here, which is what keeps them out of
// the Pend. Pago and Pagados tabs without any of those queries having to know about Type.
const ExpenseStatusPosted int8 = 3

// Expense is one concrete amount owed: a one-time bill (ExpenseScheduledID = 0) or a
// single materialized period of a schedule, carrying its own (possibly adjusted)
// amount plus payment state.
type Expense struct {
	db.TableStruct[ExpenseTable, Expense]
	CompanyID          int32  `json:",omitempty"`
	ID                 int32  `json:",omitempty"`
	ExpenseScheduledID int32  `json:",omitempty"` // 0 = one-time; otherwise → ExpenseScheduled.ID.
	PeriodDate         int16  `json:",omitempty"` // UnixDay identifying the period (dedupes period generation).
	Name               string `json:",omitempty"`
	Description        string `json:",omitempty"`
	Type               int8   `json:",omitempty"` // ExpenseType* — what the money became.
	CategoryID         int8   `json:",omitempty"`
	SupplierID         int32  `json:",omitempty"`
	AssetID            int32  `json:",omitempty"` // Type 4 → the accounting.Asset being depreciated.
	ProductID          int32  `json:",omitempty"` // Type 2 → the supply purchased.
	// Type 2 only: where the units landed and how many. Stored so the expense says what it
	// bought without having to reconstruct it from the movement ledger.
	WarehouseID  int32 `json:",omitempty"`
	Quantity     int32 `json:",omitempty"`
	CurrencyType int8  `json:",omitempty"` // 1 = PEN, 2 = USD.
	Date         int16 `json:",omitempty"` // UnixDay the expense was incurred.
	DueDate      int16 `json:",omitempty"` // UnixDay payment is due.
	Amount       int32 `json:",omitempty"` // Total owed for this expense/period, in cents.
	// Book value, which is not the same as what was paid. A computer the owner donated to the
	// business has Amount = 0 and Value = 10000: it cost no cash and still depreciates.
	Value          int32 `json:",omitempty"`
	PaidAmount     int32 `json:",omitempty"`   // Positive running sum of payments applied (server-maintained).
	Status         int8  `json:"ss,omitempty"` // 0 removed · 1 pending · 2 fully paid · 3 posted (non-cash).
	Updated        int32 `json:"upd,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
	Created        int32 `json:",omitempty"`
	CreatedBy      int32 `json:",omitempty"`
}

type ExpenseTable struct {
	db.TableStruct[ExpenseTable, Expense]
	CompanyID          db.Col[ExpenseTable, int32]
	ID                 db.Col[ExpenseTable, int32]
	ExpenseScheduledID db.Col[ExpenseTable, int32]
	PeriodDate         db.Col[ExpenseTable, int16]
	Name               db.Col[ExpenseTable, string]
	Description        db.Col[ExpenseTable, string]
	Type               db.Col[ExpenseTable, int8]
	CategoryID         db.Col[ExpenseTable, int8]
	SupplierID         db.Col[ExpenseTable, int32]
	AssetID            db.Col[ExpenseTable, int32]
	ProductID          db.Col[ExpenseTable, int32]
	WarehouseID        db.Col[ExpenseTable, int32]
	Quantity           db.Col[ExpenseTable, int32]
	CurrencyType       db.Col[ExpenseTable, int8]
	Date               db.Col[ExpenseTable, int16]
	DueDate            db.Col[ExpenseTable, int16]
	Amount             db.Col[ExpenseTable, int32]
	PaidAmount         db.Col[ExpenseTable, int32]
	Status             db.Col[ExpenseTable, int8]
	Updated            db.Col[ExpenseTable, int32]
	UpdatedVersion     db.Col[ExpenseTable, int32]
	UpdatedBy          db.Col[ExpenseTable, int32]
	Created            db.Col[ExpenseTable, int32]
	CreatedBy          db.Col[ExpenseTable, int32]
}

func (e ExpenseTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           13,
		Name:         "expenses",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Sizes the Status slot of the delta index. 3 = posted non-cash (depreciation).
		FixedValues: []db.FixedValues{
			{Col: e.Status, Min: 0, Max: 3},
		},
		Indexes: []db.Index{
			// Fetch all periods belonging to a schedule (lazy generation + period listing).
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.ExpenseScheduledID)},
			// Fetch an asset's whole depreciation ledger.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.AssetID)},
			// The Register list reads one status bucket at a time, so its handler pins Status itself
			// and calls Delta(upv) for the watermark alone rather than the status fan-out.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
