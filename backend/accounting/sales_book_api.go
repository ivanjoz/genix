// Reading the Registro de Ventas for a period.
//
// The handler reads and nothing else. It syncs the period's comprobantes and sales through their
// period delta logs (the same the accounting operations read), so a reopened period only travels
// what changed. Every join is the browser's: the series code and the document type come off the
// company record, the buyer's identity and the older comprobantes a note corrects come through
// the by-ids cache (frontend/packages/genix-ui/cache/CACHE_BY_IDS.md). The single-day view and the
// "sales never invoiced" list are filters the browser applies to the period.
//
// The SUNAT mapping — the S/ 700 threshold, the daily consolidation of boletas, the zeroing of a
// refused comprobante — therefore lives in frontend/routes/accounting/books/books.ts, where it
// is pure and unit-tested. See that folder's PLAN.md and RATIONALE.md.

package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	invoicing "app/invoicing/types"

	"golang.org/x/sync/errgroup"
)

// SalesBookDocument is a comprobante as the book needs it: the amounts, the dates, and the two
// ids the frontend joins on. The fat columns an InvoiceDocument also carries — the CDR notes,
// the ticket, the digest, the last transport error — are not part of a book and are left behind.
//
// SeriesID is not a field: it is the tail of the document id, the same way the invoicing page
// reads it.
type SalesBookDocument struct {
	ID               int64 `cb:"1"`
	IssueDate        int16 `cb:"2"`
	Correlativo      int32 `cb:"3"`
	ClientSnapshotID int32 `cb:"4"`
	Currency         int8  `cb:"5"`

	TaxableAmount    int64 `cb:"6"`
	TaxAmount        int64 `cb:"7"`
	ExemptAmount     int64 `cb:"8"`
	UnaffectedAmount int64 `cb:"9"`
	TotalAmount      int64 `cb:"10"`

	AffectedDocID int64 `cb:"11"`
	State         int8  `cb:"12"`
	// Carried through for GET.accounting-operations, which syncs comprobantes by delta log.
	Version  int16   `json:"ver" cb:"ver,13"`
	Metadata *string `json:"mdt" cb:"mdt,14"`
}

// SalesBookResponse is one key per period delta log; each key travels its own cursor (?<Key>=…).
type SalesBookResponse struct {
	SalesDocuments []SalesBookDocument `cb:"1"`
	// Every sale of the period, annulled ones aside: the page lists the ones no comprobante was
	// ever issued for (a sale whose id ends in 00). They are not book rows, but "what did we sell
	// that we never invoiced?" is the one thing a month-end review cannot answer from the book.
	Sales []OperationSale `cb:"2"`
}

// GetSalesBook syncs a whole period (?period=YYYYMM).
func GetSalesBook(req *core.HandlerArgs) core.HandlerResponse {
	firstDay, _, boundsError := types.PeriodBounds(req.Query["period"])
	if boundsError != nil {
		return req.MakeErr(boundsError.Error())
	}
	companyID, periodKey := req.User.CompanyID, core.PeriodOfDay(firstDay)

	response := SalesBookResponse{}
	readGroup := errgroup.Group{}
	readGroup.Go(func() (err error) {
		response.SalesDocuments, err = readPeriodSalesDocuments(companyID, periodKey, req.GetDeltaCursor("SalesDocuments"))
		return err
	})
	readGroup.Go(func() (err error) {
		response.Sales, err = readPeriodSales(companyID, periodKey, req.GetDeltaCursor("Sales"))
		return err
	})
	if readError := readGroup.Wait(); readError != nil {
		return req.MakeErr(readError.Error())
	}
	return req.MakeResponse(response)
}

// GetInvoiceDocumentsByIDs serves the by-IDs cache of comprobantes: the Registro de Ventas reads
// through it the older comprobantes its period's notes correct.
func GetInvoiceDocumentsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[invoicing.InvoiceDocument](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener los comprobantes.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}

func makeBookDocument(document *invoicing.InvoiceDocument) SalesBookDocument {
	return SalesBookDocument{
		ID:               document.ID,
		IssueDate:        document.IssueDate,
		Correlativo:      document.Correlativo,
		ClientSnapshotID: document.ClientSnapshotID,
		Currency:         document.Currency,
		TaxableAmount:    document.TaxableAmount,
		TaxAmount:        document.TaxAmount,
		ExemptAmount:     document.ExemptAmount,
		UnaffectedAmount: document.UnaffectedAmount,
		TotalAmount:      document.TotalAmount,
		AffectedDocID:    document.AffectedDocID,
		State:            document.State,
		Version:          document.Version,
		Metadata:         document.Metadata,
	}
}
