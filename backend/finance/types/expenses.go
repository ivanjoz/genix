package types

import "github.com/ivanjoz/genix-orm/scylla"

// ExpenseScheduled is the recurring template. It only describes the cadence and the
// default amount; per-period payment state lives in the generated Expense rows.
type ExpenseScheduled struct {
	scylla.TableStruct[ExpenseScheduledTable, ExpenseScheduled]
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
	scylla.TableStruct[ExpenseScheduledTable, ExpenseScheduled]
	CompanyID    scylla.Col[ExpenseScheduledTable, int32]
	ID           scylla.Col[ExpenseScheduledTable, int32]
	Name         scylla.Col[ExpenseScheduledTable, string]
	Description  scylla.Col[ExpenseScheduledTable, string]
	CategoryID   scylla.Col[ExpenseScheduledTable, int8]
	SupplierID   scylla.Col[ExpenseScheduledTable, int32]
	CurrencyType scylla.Col[ExpenseScheduledTable, int8]
	Amount       scylla.Col[ExpenseScheduledTable, int32]
	Frequency    scylla.Col[ExpenseScheduledTable, int16]
	StartDate    scylla.Col[ExpenseScheduledTable, int16]
	EndDate      scylla.Col[ExpenseScheduledTable, int16]
	Status       scylla.Col[ExpenseScheduledTable, int8]
	Updated      scylla.Col[ExpenseScheduledTable, int32]
	UpdatedBy    scylla.Col[ExpenseScheduledTable, int32]
	Created      scylla.Col[ExpenseScheduledTable, int32]
	CreatedBy    scylla.Col[ExpenseScheduledTable, int32]
}

func (e ExpenseScheduledTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "expenses_scheduled",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			// Delta-cache view: frontend syncs active schedules by watermark.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status.Int32(), e.Updated.DecimalSize(8))},
		},
	}
}

// Expense is one concrete amount owed: a one-time bill (ExpenseScheduledID = 0) or a
// single materialized period of a schedule, carrying its own (possibly adjusted)
// amount plus payment state.
type Expense struct {
	scylla.TableStruct[ExpenseTable, Expense]
	CompanyID          int32  `json:",omitempty"`
	ID                 int32  `json:",omitempty"`
	ExpenseScheduledID int32  `json:",omitempty"` // 0 = one-time; otherwise → ExpenseScheduled.ID.
	PeriodDate         int16  `json:",omitempty"` // UnixDay identifying the period (dedupes period generation).
	Name               string `json:",omitempty"`
	Description        string `json:",omitempty"`
	CategoryID         int8   `json:",omitempty"`
	SupplierID         int32  `json:",omitempty"`
	CurrencyType       int8   `json:",omitempty"`   // 1 = PEN, 2 = USD.
	Date               int16  `json:",omitempty"`   // UnixDay the expense was incurred.
	DueDate            int16  `json:",omitempty"`   // UnixDay payment is due.
	Amount             int32  `json:",omitempty"`   // Total owed for this expense/period, in cents.
	PaidAmount         int32  `json:",omitempty"`   // Positive running sum of payments applied (server-maintained).
	Status             int8   `json:"ss,omitempty"` // Payment lifecycle: 0 removed · 1 created/pending · 2 fully paid.
	Updated            int32  `json:"upd,omitempty"`
	UpdatedBy          int32  `json:",omitempty"`
	Created            int32  `json:",omitempty"`
	CreatedBy          int32  `json:",omitempty"`
}

type ExpenseTable struct {
	scylla.TableStruct[ExpenseTable, Expense]
	CompanyID          scylla.Col[ExpenseTable, int32]
	ID                 scylla.Col[ExpenseTable, int32]
	ExpenseScheduledID scylla.Col[ExpenseTable, int32]
	PeriodDate         scylla.Col[ExpenseTable, int16]
	Name               scylla.Col[ExpenseTable, string]
	Description        scylla.Col[ExpenseTable, string]
	CategoryID         scylla.Col[ExpenseTable, int8]
	SupplierID         scylla.Col[ExpenseTable, int32]
	CurrencyType       scylla.Col[ExpenseTable, int8]
	Date               scylla.Col[ExpenseTable, int16]
	DueDate            scylla.Col[ExpenseTable, int16]
	Amount             scylla.Col[ExpenseTable, int32]
	PaidAmount         scylla.Col[ExpenseTable, int32]
	Status             scylla.Col[ExpenseTable, int8]
	Updated            scylla.Col[ExpenseTable, int32]
	UpdatedBy          scylla.Col[ExpenseTable, int32]
	Created            scylla.Col[ExpenseTable, int32]
	CreatedBy          scylla.Col[ExpenseTable, int32]
}

func (e ExpenseTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "expenses",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			// Delta-cache view for the Register list.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status.Int32(), e.Updated.DecimalSize(8))},
			// Fetch all periods belonging to a schedule (lazy generation + period listing).
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.ExpenseScheduledID)},
		},
	}
}
