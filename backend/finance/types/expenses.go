package types

import "app/db"

// ExpenseScheduled is the recurring template. It only describes the cadence and the
// default amount; per-period payment state lives in the generated Expense rows.
type ExpenseScheduled struct {
	db.TableStruct[ExpenseScheduledTable, ExpenseScheduled]
	CompanyID    int32  `cb:"CompanyID,1"`
	ID           int32  `cb:"ID,2"`
	Name         string `cb:"Name,3"`
	Description  string `cb:"Description,4"`
	CategoryID   int8   `cb:"CategoryID,5"`   // Static expense category (code-defined list).
	SupplierID   int32  `cb:"SupplierID,6"`   // Optional — who is paid.
	CurrencyType int8   `cb:"CurrencyType,7"` // 1 = PEN, 2 = USD.
	Amount       int32  `cb:"Amount,8"`       // Default expected amount per period, in cents.
	Frequency    int16  `cb:"Frequency,9"`    // Packed cadence code CDD: cadence*100 + day.
	StartDate    int16  `cb:"StartDate,10"`   // UnixDay the schedule begins; anchors the month for N-monthly/yearly.
	EndDate      int16  `cb:"EndDate,11"`     // UnixDay the schedule stops (0 = open-ended).
	Status       int8   `json:"ss" cb:"ss,12"`
	Updated      int32  `json:"upd" cb:"upd,13"`
	Version      int16  `json:"ver" cb:"ver,14"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,15"`
	UpdatedBy int32   `cb:"UpdatedBy,16"`
	Created   int32   `cb:"Created,17"`
	CreatedBy int32   `cb:"CreatedBy,18"`
}

type ExpenseScheduledTable struct {
	db.TableStruct[ExpenseScheduledTable, ExpenseScheduled]
	CompanyID    db.Col[*ExpenseScheduledTable, int32]
	ID           db.Col[*ExpenseScheduledTable, int32]
	Name         db.Col[*ExpenseScheduledTable, string]
	Description  db.Col[*ExpenseScheduledTable, string]
	CategoryID   db.Col[*ExpenseScheduledTable, int8]
	SupplierID   db.Col[*ExpenseScheduledTable, int32]
	CurrencyType db.Col[*ExpenseScheduledTable, int8]
	Amount       db.Col[*ExpenseScheduledTable, int32]
	Frequency    db.Col[*ExpenseScheduledTable, int16]
	StartDate    db.Col[*ExpenseScheduledTable, int16]
	EndDate      db.Col[*ExpenseScheduledTable, int16]
	Status       db.Col[*ExpenseScheduledTable, int8]
	Updated      db.Col[*ExpenseScheduledTable, int32]
	Version      db.Col[*ExpenseScheduledTable, int16]
	UpdatedBy    db.Col[*ExpenseScheduledTable, int32]
	Created      db.Col[*ExpenseScheduledTable, int32]
	CreatedBy    db.Col[*ExpenseScheduledTable, int32]
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
	CompanyID          int32  `cb:"CompanyID,1"`
	ID                 int32  `cb:"ID,2"`
	ExpenseScheduledID int32  `cb:"ExpenseScheduledID,3"` // 0 = one-time; otherwise → ExpenseScheduled.ID.
	PeriodDate         int16  `cb:"PeriodDate,4"`         // UnixDay identifying the period (dedupes period generation).
	Name               string `cb:"Name,5"`
	Description        string `cb:"Description,6"`
	Type               int8   `cb:"Type,7"` // ExpenseType* — what the money became.
	CategoryID         int8   `cb:"CategoryID,8"`
	SupplierID         int32  `cb:"SupplierID,9"`
	AssetID            int32  `cb:"AssetID,10"`   // Type 4 → the accounting.Asset being depreciated.
	ProductID          int32  `cb:"ProductID,11"` // Type 2 → the supply purchased.
	// Type 2 only: where the units landed and how many. Stored so the expense says what it
	// bought without having to reconstruct it from the movement ledger.
	WarehouseID  int32 `cb:"WarehouseID,12"`
	Quantity     int32 `cb:"Quantity,13"`
	CurrencyType int8  `cb:"CurrencyType,14"` // 1 = PEN, 2 = USD.
	Date         int16 `cb:"Date,15"`         // UnixDay the expense was incurred.
	DueDate      int16 `cb:"DueDate,16"`      // UnixDay payment is due.
	Amount       int32 `cb:"Amount,17"`       // Total owed for this expense/period, in cents.
	// The supplier's comprobante — the Registro de Compras row. See PurchaseDocument; the
	// document's currency is CurrencyType, and its total must equal Amount.
	ProviderSnapshotID int32  `cb:"ProviderSnapshotID,18"` // Frozen supplier identity (RCE fields 12-14).
	DocType            int8   `cb:"DocType,19"`
	DocSeries          string `cb:"DocSeries,20"`
	DocNumber          int64  `cb:"DocNumber,21"`
	DocIssueDate       int16  `cb:"DocIssueDate,22"`
	TaxableAmount      int32  `cb:"TaxableAmount,23"`
	TaxAmount          int32  `cb:"TaxAmount,24"`
	UntaxedAmount      int32  `cb:"UntaxedAmount,25"`
	OtherAmount        int32  `cb:"OtherAmount,26"`
	ExchangeRate       int32  `cb:"ExchangeRate,27"`
	PaidAmount         int32  `cb:"PaidAmount,28"`   // Positive running sum of payments applied (server-maintained).
	Status             int8   `json:"ss" cb:"ss,29"` // 0 removed · 1 pending · 2 fully paid · 3 posted (non-cash).
	Updated            int32  `json:"upd" cb:"upd,30"`
	Version            int16  `json:"ver" cb:"ver,31"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,32"`
	UpdatedBy int32   `cb:"UpdatedBy,33"`
	Created   int32   `cb:"Created,34"`
	CreatedBy int32   `cb:"CreatedBy,35"`
}

type ExpenseTable struct {
	db.TableStruct[ExpenseTable, Expense]
	CompanyID          db.Col[*ExpenseTable, int32]
	ID                 db.Col[*ExpenseTable, int32]
	ExpenseScheduledID db.Col[*ExpenseTable, int32]
	PeriodDate         db.Col[*ExpenseTable, int16]
	Name               db.Col[*ExpenseTable, string]
	Description        db.Col[*ExpenseTable, string]
	Type               db.Col[*ExpenseTable, int8]
	CategoryID         db.Col[*ExpenseTable, int8]
	SupplierID         db.Col[*ExpenseTable, int32]
	AssetID            db.Col[*ExpenseTable, int32]
	ProductID          db.Col[*ExpenseTable, int32]
	WarehouseID        db.Col[*ExpenseTable, int32]
	Quantity           db.Col[*ExpenseTable, int32]
	CurrencyType       db.Col[*ExpenseTable, int8]
	Date               db.Col[*ExpenseTable, int16]
	DueDate            db.Col[*ExpenseTable, int16]
	Amount             db.Col[*ExpenseTable, int32]
	ProviderSnapshotID db.Col[*ExpenseTable, int32]
	DocType            db.Col[*ExpenseTable, int8]
	DocSeries          db.Col[*ExpenseTable, string]
	DocNumber          db.Col[*ExpenseTable, int64]
	DocIssueDate       db.Col[*ExpenseTable, int16]
	TaxableAmount      db.Col[*ExpenseTable, int32]
	TaxAmount          db.Col[*ExpenseTable, int32]
	UntaxedAmount      db.Col[*ExpenseTable, int32]
	OtherAmount        db.Col[*ExpenseTable, int32]
	ExchangeRate       db.Col[*ExpenseTable, int32]
	PaidAmount         db.Col[*ExpenseTable, int32]
	Status             db.Col[*ExpenseTable, int8]
	Updated            db.Col[*ExpenseTable, int32]
	Version            db.Col[*ExpenseTable, int16]
	UpdatedBy          db.Col[*ExpenseTable, int32]
	Created            db.Col[*ExpenseTable, int32]
	CreatedBy          db.Col[*ExpenseTable, int32]
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
			// Status is the sync filter: a first sync lists the statuses the handler asks for.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
			// The Registro de Compras reads a period of documents, one indexed read per day.
			{Keys: db.Cols(e.DocIssueDate), UseIndexGroup: true},
		},
	}
}

func (e *Expense) PurchaseDocument() PurchaseDocument {
	return PurchaseDocument{
		DocType: e.DocType, DocSeries: e.DocSeries, DocNumber: e.DocNumber, DocIssueDate: e.DocIssueDate,
		TaxableAmount: e.TaxableAmount, TaxAmount: e.TaxAmount, UntaxedAmount: e.UntaxedAmount,
		OtherAmount: e.OtherAmount, CurrencyType: e.CurrencyType, ExchangeRate: e.ExchangeRate,
	}
}

func (e *Expense) SetPurchaseDocument(document PurchaseDocument) {
	e.DocType, e.DocSeries, e.DocNumber = document.DocType, document.DocSeries, document.DocNumber
	e.DocIssueDate = document.DocIssueDate
	e.TaxableAmount, e.TaxAmount = document.TaxableAmount, document.TaxAmount
	e.UntaxedAmount, e.OtherAmount = document.UntaxedAmount, document.OtherAmount
	e.CurrencyType, e.ExchangeRate = document.CurrencyType, document.ExchangeRate
}
