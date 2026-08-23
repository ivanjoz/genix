package types

import "app/db"

// Kinds of batch transmission.
const (
	// SummaryTypeDaily is the RC: a day's boletas reported together, and the
	// only way to cancel one.
	SummaryTypeDaily = int8(1)
	// SummaryTypeVoided is the RA: facturas cancelled after being accepted.
	SummaryTypeVoided = int8(2)
)

// InvoiceSummary is a daily summary (RC) or a voiding communication (RA).
//
// Both are asynchronous by SUNAT's design: the transmission is answered with a
// ticket, and the verdict has to be fetched afterwards. That is the whole reason
// this is a table and not a function — something has to remember the ticket
// across the minutes or hours until SUNAT finishes.
//
// The sequence inside the identifier — RC-20260822-001 — counts the batches sent
// on one day, so the key packs the date with an autoincrement scoped to it. RC
// and RA share that counter: SUNAT only requires the identifier not to repeat,
// and a shared counter leaves harmless gaps instead of costing a key column.
type InvoiceSummary struct {
	db.TableStruct[InvoiceSummaryTable, InvoiceSummary]
	CompanyID int32 `json:",omitempty"`
	ID        int64 `json:",omitempty"`

	Type int8 `json:",omitempty"`
	// ReferenceDate is the day whose documents are being reported or cancelled.
	ReferenceDate int16 `json:",omitempty"`
	// IssueDate is the day the batch is sent, and what the identifier is built
	// from. Using the reference date instead is SUNAT rejection 2346.
	IssueDate int16  `json:",omitempty"`
	Sequence  int16  `json:",omitempty"`
	XmlName   string `json:",omitempty" db:"xml_name"`

	// DocumentIDs are the invoice_document rows this batch reports.
	DocumentIDs []int64 `json:",omitempty" db:",list"`

	Ticket           string   `json:",omitempty"`
	State            int8     `json:",omitempty"`
	SunatCode        string   `json:",omitempty"`
	SunatDescription string   `json:",omitempty"`
	SunatNotes       []string `json:",omitempty" db:",list"`

	XmlPath string `json:",omitempty" db:"xml_path"`
	CdrPath string `json:",omitempty" db:"cdr_path"`

	RetryCount int8   `json:",omitempty"`
	LastError  string `json:",omitempty"`

	Status         int8  `json:"ss,omitempty"`
	Updated        int32 `json:"upd,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
	Created        int32 `json:",omitempty"`
	CreatedBy      int32 `json:",omitempty"`
}

type InvoiceSummaryTable struct {
	db.TableStruct[InvoiceSummaryTable, InvoiceSummary]
	CompanyID        db.Col[InvoiceSummaryTable, int32]
	ID               db.Col[InvoiceSummaryTable, int64]
	Type             db.Col[InvoiceSummaryTable, int8]
	ReferenceDate    db.Col[InvoiceSummaryTable, int16]
	IssueDate        db.Col[InvoiceSummaryTable, int16]
	Sequence         db.Col[InvoiceSummaryTable, int16]
	XmlName          db.Col[InvoiceSummaryTable, string] `db:"xml_name"`
	DocumentIDs      db.ColSlice[InvoiceSummaryTable, int64]
	Ticket           db.Col[InvoiceSummaryTable, string]
	State            db.Col[InvoiceSummaryTable, int8]
	SunatCode        db.Col[InvoiceSummaryTable, string]
	SunatDescription db.Col[InvoiceSummaryTable, string]
	SunatNotes       db.ColSlice[InvoiceSummaryTable, string]
	XmlPath          db.Col[InvoiceSummaryTable, string] `db:"xml_path"`
	CdrPath          db.Col[InvoiceSummaryTable, string] `db:"cdr_path"`
	RetryCount       db.Col[InvoiceSummaryTable, int8]
	LastError        db.Col[InvoiceSummaryTable, string]
	Status           db.Col[InvoiceSummaryTable, int8]
	Updated          db.Col[InvoiceSummaryTable, int32]
	UpdatedVersion   db.Col[InvoiceSummaryTable, int32]
	UpdatedBy        db.Col[InvoiceSummaryTable, int32]
	Created          db.Col[InvoiceSummaryTable, int32]
	CreatedBy        db.Col[InvoiceSummaryTable, int32]
}

func (e InvoiceSummaryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                 54,
		Name:               "invoice_summary",
		Partition:          e.CompanyID,
		Keys:               db.Cols(e.ID),
		SaveUpdatedVersion: true,
		// A clean sequence: the ### of RC-YYYYMMDD-### counts the batches of the
		// day, so random digits would make no sense there either.
		KeyIntPacking:     db.Cols(e.IssueDate.DecimalSize(5), e.Autoincrement(0)),
		AutoincrementPart: e.IssueDate,
		FixedValues: []db.FixedValues{
			{Col: e.State, Min: 0, Max: 6},
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeInheritFromKey, Keys: db.Cols(e.IssueDate), UseIndexGroup: true},
			// A ticket has to be findable on its own: the cron that polls for a
			// verdict starts from the ticket, not from the batch.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.Ticket)},
			{Type: db.TypeDelta, Keys: db.Cols(e.State)},
		},
	}
}
