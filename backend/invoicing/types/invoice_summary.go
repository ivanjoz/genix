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
	CompanyID int32 `cb:"1"`
	ID        int64 `cb:"2"`

	Type int8 `cb:"3"`
	// ReferenceDate is the day whose documents are being reported or cancelled.
	ReferenceDate int16 `cb:"4"`
	// IssueDate is the day the batch is sent, and what the identifier is built
	// from. Using the reference date instead is SUNAT rejection 2346.
	IssueDate int16  `cb:"5"`
	Sequence  int16  `cb:"6"`
	XmlName   string `cb:"7"`

	// DocumentIDs are the invoice_document rows this batch reports.
	DocumentIDs []int64 `db:",list" cb:"8"`

	Ticket           string   `cb:"9"`
	State            int8     `cb:"10"`
	SunatCode        string   `cb:"11"`
	SunatDescription string   `cb:"12"`
	SunatNotes       []string `db:",list" cb:"13"`

	XmlPath string `cb:"14"`
	CdrPath string `cb:"15"`

	RetryCount int8   `cb:"16"`
	LastError  string `cb:"17"`

	Status  int8  `json:"ss" cb:"ss,18"`
	Updated int32 `json:"upd" cb:"upd,19"`
	Version int16 `json:"ver" cb:"ver,20"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,21"`
	UpdatedBy int32   `cb:"22"`
	Created   int32   `cb:"23"`
	CreatedBy int32   `cb:"24"`
}

type InvoiceSummaryTable struct {
	db.TableStruct[InvoiceSummaryTable, InvoiceSummary]
	CompanyID        db.Col[*InvoiceSummaryTable, int32]
	ID               db.Col[*InvoiceSummaryTable, int64]
	Type             db.Col[*InvoiceSummaryTable, int8]
	ReferenceDate    db.Col[*InvoiceSummaryTable, int16]
	IssueDate        db.Col[*InvoiceSummaryTable, int16]
	Sequence         db.Col[*InvoiceSummaryTable, int16]
	XmlName          db.Col[*InvoiceSummaryTable, string]
	DocumentIDs      db.ColSlice[*InvoiceSummaryTable, int64]
	Ticket           db.Col[*InvoiceSummaryTable, string]
	State            db.Col[*InvoiceSummaryTable, int8]
	SunatCode        db.Col[*InvoiceSummaryTable, string]
	SunatDescription db.Col[*InvoiceSummaryTable, string]
	SunatNotes       db.ColSlice[*InvoiceSummaryTable, string]
	XmlPath          db.Col[*InvoiceSummaryTable, string]
	CdrPath          db.Col[*InvoiceSummaryTable, string]
	RetryCount       db.Col[*InvoiceSummaryTable, int8]
	LastError        db.Col[*InvoiceSummaryTable, string]
	Status           db.Col[*InvoiceSummaryTable, int8]
	Updated          db.Col[*InvoiceSummaryTable, int32]
	Version          db.Col[*InvoiceSummaryTable, int16]
	UpdatedBy        db.Col[*InvoiceSummaryTable, int32]
	Created          db.Col[*InvoiceSummaryTable, int32]
	CreatedBy        db.Col[*InvoiceSummaryTable, int32]
}

func (e InvoiceSummaryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:         54,
		Name:       "invoice_summary",
		Partition:  e.CompanyID,
		Keys:       db.Cols(e.ID),
		CacheByIDs: true,
		// A clean sequence: the ### of RC-YYYYMMDD-### counts the batches of the
		// day, so random digits would make no sense there either.
		KeyIntPacking:     db.Cols(e.IssueDate.Size(15), e.Autoincrement(0)),
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
