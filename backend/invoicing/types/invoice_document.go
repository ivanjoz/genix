package types

import "app/db"

// Document types, SUNAT catalog 01, stored as the numeric part of the code.
const (
	DocTypeFactura    = int8(1)
	DocTypeBoleta     = int8(3)
	DocTypeCreditNote = int8(7)
	DocTypeDebitNote  = int8(8)
)

// States of an issued document. They are ordered so the delta index can size a
// single digit for them, and they mirror what SUNAT can answer.
const (
	// InvoiceVoided is a document cancelled after it was accepted.
	InvoiceVoided = int8(0)
	// InvoicePending is reserved and numbered but not yet signed or sent.
	InvoicePending = int8(1)
	// InvoiceQueued is signed and in flight, or waiting on a ticket.
	InvoiceQueued = int8(2)
	// InvoiceAccepted is a CDR with code 0.
	InvoiceAccepted = int8(3)
	// InvoiceObserved is accepted with remarks: valid, but worth reading.
	InvoiceObserved = int8(4)
	// InvoiceRejected is a CDR between 2000 and 3999. Never retried: the XML is
	// invalid and always will be. The number stays consumed.
	InvoiceRejected = int8(5)
	// InvoiceException is a transport or configuration failure. Retried.
	InvoiceException = int8(6)
)

// Currencies a document can be issued in.
const (
	CurrencyPEN = int8(1)
	CurrencyUSD = int8(2)
)

// docTypeSeriesFactor packs the document type and the series into one number.
// Six digits: two for the type, up to four for the series.
const docTypeSeriesFactor = 1000

// PackDocTypeSeries builds the value the primary key is partitioned by.
func PackDocTypeSeries(docType int8, seriesID int16) int32 {
	return int32(docType)*docTypeSeriesFactor + int32(seriesID)
}

// correlativoDigits is the width reserved for the number inside the packed key.
const correlativoDigits = 100_000_000

// InvoiceDocument is one electronic document that has been issued, or is about
// to be.
//
// The SUNAT identity of a document is its type, series and number, and here that
// identity *is* the primary key: ID packs the type and series with an
// autoincrement, and the ORM allocates the number from a Scylla counter
// partitioned by DocTypeSeries. Two things follow. Numbering is atomic without
// any lock being held across a call to SUNAT, and a duplicate becomes a
// primary-key collision rather than a rule somebody has to remember to check.
//
// A failed emission still consumes its number. That is deliberate: the row is
// written before the document is signed, so a retry reuses its own number
// instead of taking a fresh one, and SUNAT tolerates gaps in a series.
type InvoiceDocument struct {
	db.TableStruct[InvoiceDocumentTable, InvoiceDocument]
	CompanyID int32 `json:",omitempty"`
	ID        int64 `json:",omitempty"`

	// DocTypeSeries is the packed key part; the two that follow are the same
	// identity spelled out, because a report should not have to unpack a key.
	// The number itself is not stored — see Correlativo.
	DocTypeSeries int32  `json:",omitempty"`
	DocType       int8   `json:",omitempty"`
	SeriesID      int16  `json:",omitempty"`
	SeriesCode    string `json:",omitempty"`

	IssueDate int16 `json:",omitempty"`
	IssueTime int32 `json:",omitempty"`

	// SaleOrderID is the sale this document bills. Zero for a document issued
	// on its own, such as a note correcting an earlier one.
	SaleOrderID int64 `json:",omitempty"`

	// The customer as it was at the moment of issue. Copied rather than joined:
	// a document is immutable, and the client record is not.
	ClientID        int32  `json:",omitempty"`
	ClientDocType   int8   `json:",omitempty"`
	ClientDocNumber string `json:",omitempty"`
	ClientName      string `json:",omitempty"`

	Currency int8 `json:",omitempty"`

	// Amounts in cents. They aggregate the lines, so they are 64 bits.
	TotalAmount      int64 `json:",omitempty"`
	TaxAmount        int64 `json:",omitempty"`
	TaxableAmount    int64 `json:",omitempty"`
	ExemptAmount     int64 `json:",omitempty"`
	UnaffectedAmount int64 `json:",omitempty"`
	FreeAmount       int64 `json:",omitempty"`

	// The lines as they were sent, in parallel slices. Amounts are per-line, so
	// they are 32 bits, and quantities are whole units as everywhere else.
	DetailProductIDs  []int32  `json:",omitempty" db:",list"`
	DetailQuantity    []int32  `json:",omitempty" db:",list"`
	DetailUnitValue   []int32  `json:",omitempty" db:",list"`
	DetailValue       []int32  `json:",omitempty" db:",list"`
	DetailIgvAmount   []int32  `json:",omitempty" db:",list"`
	DetailIgvType     []int8   `json:",omitempty" db:",list"`
	DetailUnitCode    []string `json:",omitempty" db:",list"`
	DetailDescription []string `json:",omitempty" db:",list"`

	// Notes only: what this document corrects, and why.
	AffectedDocID  int64  `json:",omitempty"`
	NoteReasonCode string `json:",omitempty"`
	NoteReason     string `json:",omitempty"`

	State            int8     `json:",omitempty"`
	SunatCode        string   `json:",omitempty"`
	SunatDescription string   `json:",omitempty"`
	SunatNotes       []string `json:",omitempty" db:",list"`
	Ticket           string   `json:",omitempty"`
	InvoiceSummaryID int64    `json:",omitempty"`
	// DigestValue is SUNAT's hash of the document. It is the last field of the
	// QR code, so it has to survive as long as the document does.
	DigestValue string `json:",omitempty"`

	// Where the artifacts live. Keeping both for five years is a legal
	// obligation, which is why the paths are columns and not a convention.
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

// Correlativo is the document number, read off the key rather than stored.
//
// It cannot be a column: the ORM assigns the key during the insert, after
// SelfParse has run, so a stored copy would be written as zero and then disagree
// with the number the document was actually issued under. Deriving it leaves one
// source of truth.
func (e *InvoiceDocument) Correlativo() int32 {
	return int32(e.ID % correlativoDigits)
}

// Number is the document identifier the way SUNAT writes it: F001-123.
func (e *InvoiceDocument) Number() string {
	return e.SeriesCode + "-" + itoa(int64(e.Correlativo()))
}

// SelfParse derives the packed key part, so a caller sets the identity once and
// cannot leave the two disagreeing.
func (e *InvoiceDocument) SelfParse() {
	if e.DocTypeSeries == 0 {
		e.DocTypeSeries = PackDocTypeSeries(e.DocType, e.SeriesID)
	}
}

type InvoiceDocumentTable struct {
	db.TableStruct[InvoiceDocumentTable, InvoiceDocument]
	CompanyID         db.Col[*InvoiceDocumentTable, int32]
	ID                db.Col[*InvoiceDocumentTable, int64]
	DocTypeSeries     db.Col[*InvoiceDocumentTable, int32]
	DocType           db.Col[*InvoiceDocumentTable, int8]
	SeriesID          db.Col[*InvoiceDocumentTable, int16]
	SeriesCode        db.Col[*InvoiceDocumentTable, string]
	IssueDate         db.Col[*InvoiceDocumentTable, int16]
	IssueTime         db.Col[*InvoiceDocumentTable, int32]
	SaleOrderID       db.Col[*InvoiceDocumentTable, int64]
	ClientID          db.Col[*InvoiceDocumentTable, int32]
	ClientDocType     db.Col[*InvoiceDocumentTable, int8]
	ClientDocNumber   db.Col[*InvoiceDocumentTable, string]
	ClientName        db.Col[*InvoiceDocumentTable, string]
	Currency          db.Col[*InvoiceDocumentTable, int8]
	TotalAmount       db.Col[*InvoiceDocumentTable, int64]
	TaxAmount         db.Col[*InvoiceDocumentTable, int64]
	TaxableAmount     db.Col[*InvoiceDocumentTable, int64]
	ExemptAmount      db.Col[*InvoiceDocumentTable, int64]
	UnaffectedAmount  db.Col[*InvoiceDocumentTable, int64]
	FreeAmount        db.Col[*InvoiceDocumentTable, int64]
	DetailProductIDs  db.ColSlice[*InvoiceDocumentTable, int32]
	DetailQuantity    db.ColSlice[*InvoiceDocumentTable, int32]
	DetailUnitValue   db.ColSlice[*InvoiceDocumentTable, int32]
	DetailValue       db.ColSlice[*InvoiceDocumentTable, int32]
	DetailIgvAmount   db.ColSlice[*InvoiceDocumentTable, int32]
	DetailIgvType     db.ColSlice[*InvoiceDocumentTable, int8]
	DetailUnitCode    db.ColSlice[*InvoiceDocumentTable, string]
	DetailDescription db.ColSlice[*InvoiceDocumentTable, string]
	AffectedDocID     db.Col[*InvoiceDocumentTable, int64]
	NoteReasonCode    db.Col[*InvoiceDocumentTable, string]
	NoteReason        db.Col[*InvoiceDocumentTable, string]
	State             db.Col[*InvoiceDocumentTable, int8]
	SunatCode         db.Col[*InvoiceDocumentTable, string]
	SunatDescription  db.Col[*InvoiceDocumentTable, string]
	SunatNotes        db.ColSlice[*InvoiceDocumentTable, string]
	Ticket            db.Col[*InvoiceDocumentTable, string]
	InvoiceSummaryID  db.Col[*InvoiceDocumentTable, int64]
	DigestValue       db.Col[*InvoiceDocumentTable, string]
	XmlPath           db.Col[*InvoiceDocumentTable, string] `db:"xml_path"`
	CdrPath           db.Col[*InvoiceDocumentTable, string] `db:"cdr_path"`
	RetryCount        db.Col[*InvoiceDocumentTable, int8]
	LastError         db.Col[*InvoiceDocumentTable, string]
	Status            db.Col[*InvoiceDocumentTable, int8]
	Updated           db.Col[*InvoiceDocumentTable, int32]
	UpdatedVersion    db.Col[*InvoiceDocumentTable, int32]
	UpdatedBy         db.Col[*InvoiceDocumentTable, int32]
	Created           db.Col[*InvoiceDocumentTable, int32]
	CreatedBy         db.Col[*InvoiceDocumentTable, int32]
}

func (e InvoiceDocumentTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                 42,
		Name:               "invoice_document",
		Partition:          e.CompanyID,
		Keys:               db.Cols(e.ID),
		SaveUpdatedVersion: true,
		// The number is the autoincrement, counted per series rather than per
		// company: F001 and B001 advance independently, which is what SUNAT
		// requires and what makes the counter contention-free.
		//
		// Autoincrement(0) is not a default — it is the clean 1, 2, 3 sequence a
		// correlativo has to be. Any other value appends that many random digits,
		// which is right for an opaque id and wrong for a document number.
		KeyIntPacking:     db.Cols(e.DocTypeSeries.DecimalSize(6), e.Autoincrement(0)),
		AutoincrementPart: e.DocTypeSeries,
		FixedValues: []db.FixedValues{
			{Col: e.State, Min: 0, Max: 6},
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			// Free: a range scan over the key prefix lists a whole series in
			// order, which is how an accountant reads them.
			{Type: db.TypeInheritFromKey, Keys: db.Cols(e.DocTypeSeries), UseIndexGroup: true},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.IssueDate)},
			// "Has this sale been invoiced?" — asked before every emission.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.SaleOrderID)},
			{Type: db.TypeDelta, Keys: db.Cols(e.State)},
		},
	}
}
