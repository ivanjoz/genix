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

// SeriesDigits is the width the series occupies at the tail of an id, in both a
// sale id and a document id. It is what caps a company at 99 series.
const SeriesDigits = int64(100)

// DocumentIDForSale is the id a document takes: the sale's id with its last two
// digits replaced by the series this document is issued under.
//
// For the document that bills the sale those are the same digits — a sale already
// carries the series it will be issued under — so its id *is* the sale's id. A note
// is issued under its own series (FC01 corrects a factura, BC01 a boleta) and so
// lands on a neighbouring id, which is what lets a sale hold a boleta and the
// credit note correcting it without a second key column.
func DocumentIDForSale(saleOrderID int64, seriesID int8) int64 {
	return SalePrefix(saleOrderID)*SeriesDigits + int64(seriesID)
}

// SalePrefix is the counter and the random digits a sale shares with every
// document issued for it.
func SalePrefix(id int64) int64 {
	return id / SeriesDigits
}

// CorrelativoCounterName names the sequence a series numbers from: one counter
// per company and series, which is exactly the scope SUNAT requires the numbering
// to be unique in. Reserved through the fareward-backed allocator like every
// other counter in the system.
func CorrelativoCounterName(companyID int32, seriesID int8) string {
	return "cpe_" + itoa(int64(companyID)) + "_" + itoa(int64(seriesID))
}

// InvoiceDocument is one electronic document that has been issued, or is about
// to be.
//
// The row holds what only it knows: the number it was issued under, what SUNAT
// answered, and where it is in the send cycle. Everything describing *what* was
// sold is read back from the sale, so nothing is stored twice. The artifacts that
// must survive five years are the signed XML and the CDR, and those live in object
// storage under a path derived from the id.
//
// There is no SaleOrderID column: the document that bills a sale is keyed by that
// sale, and a note reaches it through AffectedDocID — the document it corrects is
// the one whose id is the sale's.
type InvoiceDocument struct {
	db.TableStruct[InvoiceDocumentTable, InvoiceDocument]
	CompanyID int32 `json:",omitempty"`
	// ID is the sale's id with the series in its last two digits. Set by the
	// caller, not the ORM — see DocumentIDForSale.
	ID int64 `json:",omitempty"`

	// Correlativo is the document number within its series, reserved from the
	// series counter when the row is written.
	Correlativo int32 `json:",omitempty"`

	IssueDate int16 `json:",omitempty"`
	IssueTime int32 `json:",omitempty"`

	Currency int8 `json:",omitempty"`

	// Amounts in cents. They aggregate the lines, so they are 64 bits. Kept even
	// though the lines are not: every list and report reads these, and joining to
	// the sale for each row would be the wrong trade.
	TotalAmount      int64 `json:",omitempty"`
	TaxAmount        int64 `json:",omitempty"`
	TaxableAmount    int64 `json:",omitempty"`
	ExemptAmount     int64 `json:",omitempty"`
	UnaffectedAmount int64 `json:",omitempty"`
	FreeAmount       int64 `json:",omitempty"`

	// Notes only: what this document corrects, and why. AffectedDocID is also how a
	// note reaches its sale — the document it corrects is the one whose id is the
	// sale's.
	AffectedDocID  int64  `json:",omitempty"`
	NoteReasonCode string `json:",omitempty"`
	NoteReason     string `json:",omitempty"`

	State     int8   `json:",omitempty"`
	SunatCode string `json:",omitempty"`
	// SunatNotes are the CDR's observations. Unlike the description, which is
	// derivable from the code, these exist nowhere but in the CDR.
	SunatNotes       []string `json:",omitempty" db:",list"`
	Ticket           string   `json:",omitempty"`
	InvoiceSummaryID int64    `json:",omitempty"`
	// DigestValue is SUNAT's hash of the document. It is the last field of the
	// QR code, so it has to survive as long as the document does.
	DigestValue string `json:",omitempty"`

	RetryCount int8   `json:",omitempty"`
	LastError  string `json:",omitempty"`

	Status int8 `json:"ss,omitempty"`
	// UpdatedVersion drives the delta list: State moves several times after the
	// row is written and the frontend syncs on those transitions.
	Updated        int32 `json:"upd,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
	Created        int32 `json:",omitempty"`
	CreatedBy      int32 `json:",omitempty"`
}

// SeriesID is the series this document was issued under, read off the tail of the
// id. It resolves against the company's inline series for the type and the code.
func (e *InvoiceDocument) SeriesID() int8 {
	return int8(e.ID % SeriesDigits)
}

// Number is the identifier the way SUNAT writes it: F001-123. The code is not
// stored — it belongs to the series — so the caller supplies it.
func (e *InvoiceDocument) Number(seriesCode string) string {
	return seriesCode + "-" + itoa(int64(e.Correlativo))
}

type InvoiceDocumentTable struct {
	db.TableStruct[InvoiceDocumentTable, InvoiceDocument]
	CompanyID        db.Col[*InvoiceDocumentTable, int32]
	ID               db.Col[*InvoiceDocumentTable, int64]
	Correlativo      db.Col[*InvoiceDocumentTable, int32]
	IssueDate        db.Col[*InvoiceDocumentTable, int16]
	IssueTime        db.Col[*InvoiceDocumentTable, int32]
	Currency         db.Col[*InvoiceDocumentTable, int8]
	TotalAmount      db.Col[*InvoiceDocumentTable, int64]
	TaxAmount        db.Col[*InvoiceDocumentTable, int64]
	TaxableAmount    db.Col[*InvoiceDocumentTable, int64]
	ExemptAmount     db.Col[*InvoiceDocumentTable, int64]
	UnaffectedAmount db.Col[*InvoiceDocumentTable, int64]
	FreeAmount       db.Col[*InvoiceDocumentTable, int64]
	AffectedDocID    db.Col[*InvoiceDocumentTable, int64]
	NoteReasonCode   db.Col[*InvoiceDocumentTable, string]
	NoteReason       db.Col[*InvoiceDocumentTable, string]
	State            db.Col[*InvoiceDocumentTable, int8]
	SunatCode        db.Col[*InvoiceDocumentTable, string]
	SunatNotes       db.Col[*InvoiceDocumentTable, []string]
	Ticket           db.Col[*InvoiceDocumentTable, string]
	InvoiceSummaryID db.Col[*InvoiceDocumentTable, int64]
	DigestValue      db.Col[*InvoiceDocumentTable, string]
	RetryCount       db.Col[*InvoiceDocumentTable, int8]
	LastError        db.Col[*InvoiceDocumentTable, string]
	Status           db.Col[*InvoiceDocumentTable, int8]
	Updated          db.Col[*InvoiceDocumentTable, int32]
	UpdatedVersion   db.Col[*InvoiceDocumentTable, int32]
	Created          db.Col[*InvoiceDocumentTable, int32]
	CreatedBy        db.Col[*InvoiceDocumentTable, int32]
}

func (e InvoiceDocumentTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                 42,
		Name:               "invoice_document",
		Partition:          e.CompanyID,
		SaveUpdatedVersion: true,
		// A plain key. The id is built by the caller from the sale and the series,
		// so there is nothing for the ORM to allocate or pack — and no autoincrement
		// declaration that could be spelled wrong.
		Keys: db.Cols(e.ID),
		FixedValues: []db.FixedValues{
			{Col: e.State, Min: 0, Max: 6},
		},
		// No index on the series: it lives in the tail of the id rather than in a
		// column, so there is nothing to index. Filtering a list by series happens
		// on the client, over the set the delta view already synced.
		// "Was this sale invoiced?" needs no index: the documents of a sale are a
		// contiguous range of the key, so the question is a slice of the partition.
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.IssueDate)},
			{Type: db.TypeDelta, Keys: db.Cols(e.State)},
		},
	}
}
