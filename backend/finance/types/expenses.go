package types

import "app/db"

// ExpenseScheduled is the recurring template. It only describes the cadence and the
// default amount; per-period payment state lives in the generated Expense rows.
type ExpenseScheduled struct {
	db.TableStruct[ExpenseScheduledTable, ExpenseScheduled]
	CompanyID    int32  `json:",omitempty"`
	ID           int32  `json:",omitempty"`
	Name         string `json:",omitempty"`
	Description  string `json:",omitempty"`
	CategoryID   int8   `json:",omitempty"` // Static expense category (code-defined list).
	SupplierID   int32  `json:",omitempty"` // Optional — who is paid.
	CurrencyType int8   `json:",omitempty"` // 1 = PEN, 2 = USD.
	Amount       int32  `json:",omitempty"` // Default expected amount per period, in cents.
	Frequency    int16  `json:",omitempty"` // Packed cadence code CDD: cadence*100 + day.
	StartDate    int16  `json:",omitempty"` // UnixDay the schedule begins; anchors the month for N-monthly/yearly.
	EndDate      int16  `json:",omitempty"` // UnixDay the schedule stops (0 = open-ended).
	Status       int8   `json:"ss,omitempty"`
	Updated      int32  `json:"upd,omitempty"`
	UpdatedBy    int32  `json:",omitempty"`
	Created      int32  `json:",omitempty"`
	CreatedBy    int32  `json:",omitempty"`
}

type ExpenseScheduledTable struct {
	db.TableStruct[ExpenseScheduledTable, ExpenseScheduled]
	CompanyID    db.Col[ExpenseScheduledTable, int32]
	ID           db.Col[ExpenseScheduledTable, int32]
	Name         db.Col[ExpenseScheduledTable, string]
	Description  db.Col[ExpenseScheduledTable, string]
	CategoryID   db.Col[ExpenseScheduledTable, int8]
	SupplierID   db.Col[ExpenseScheduledTable, int32]
	CurrencyType db.Col[ExpenseScheduledTable, int8]
	Amount       db.Col[ExpenseScheduledTable, int32]
	Frequency    db.Col[ExpenseScheduledTable, int16]
	StartDate    db.Col[ExpenseScheduledTable, int16]
	EndDate      db.Col[ExpenseScheduledTable, int16]
	Status       db.Col[ExpenseScheduledTable, int8]
	Updated      db.Col[ExpenseScheduledTable, int32]
	UpdatedBy    db.Col[ExpenseScheduledTable, int32]
	Created      db.Col[ExpenseScheduledTable, int32]
	CreatedBy    db.Col[ExpenseScheduledTable, int32]
}

func (e ExpenseScheduledTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "expenses_scheduled",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			// Delta-cache view: frontend syncs active schedules by watermark.
			{Type: db.TypeView, Keys: []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)}, KeepPart: true},
		},
	}
}

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
	CategoryID         int8   `json:",omitempty"`
	SupplierID         int32  `json:",omitempty"`
	CurrencyType       int8   `json:",omitempty"` // 1 = PEN, 2 = USD.
	Date               int16  `json:",omitempty"` // UnixDay the expense was incurred.
	DueDate            int16  `json:",omitempty"` // UnixDay payment is due.
	Amount             int32  `json:",omitempty"` // Total owed for this expense/period, in cents.
	PaidAmount         int32  `json:",omitempty"` // Positive running sum of payments applied (server-maintained).
	Status             int8   `json:"ss,omitempty"` // Payment lifecycle: 0 removed · 1 created/pending · 2 fully paid.
	Updated            int32  `json:"upd,omitempty"`
	UpdatedBy          int32  `json:",omitempty"`
	Created            int32  `json:",omitempty"`
	CreatedBy          int32  `json:",omitempty"`
}

type ExpenseTable struct {
	db.TableStruct[ExpenseTable, Expense]
	CompanyID          db.Col[ExpenseTable, int32]
	ID                 db.Col[ExpenseTable, int32]
	ExpenseScheduledID db.Col[ExpenseTable, int32]
	PeriodDate         db.Col[ExpenseTable, int16]
	Name               db.Col[ExpenseTable, string]
	Description        db.Col[ExpenseTable, string]
	CategoryID         db.Col[ExpenseTable, int8]
	SupplierID         db.Col[ExpenseTable, int32]
	CurrencyType       db.Col[ExpenseTable, int8]
	Date               db.Col[ExpenseTable, int16]
	DueDate            db.Col[ExpenseTable, int16]
	Amount             db.Col[ExpenseTable, int32]
	PaidAmount         db.Col[ExpenseTable, int32]
	Status             db.Col[ExpenseTable, int8]
	Updated            db.Col[ExpenseTable, int32]
	UpdatedBy          db.Col[ExpenseTable, int32]
	Created            db.Col[ExpenseTable, int32]
	CreatedBy          db.Col[ExpenseTable, int32]
}

func (e ExpenseTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "expenses",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			// Delta-cache view for the Register list.
			{Type: db.TypeView, Keys: []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)}, KeepPart: true},
			// Fetch all periods belonging to a schedule (lazy generation + period listing).
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.ExpenseScheduledID}},
		},
	}
}
