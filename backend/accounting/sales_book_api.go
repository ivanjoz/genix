// Reading the Registro de Ventas for a period.
//
// The handler reads and nothing else. It sends the period's comprobantes as numbers and leaves
// both joins to the browser: the series code and the document type come off the company record
// the frontend already holds, and the buyer's identity is resolved through the by-ids cache
// (frontend/packages/genix-ui/cache/CACHE_BY_IDS.md). Neither is re-read here.
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
	sales "app/sales/types"
)

// SalesBookDocument is a comprobante as the book needs it: the amounts, the dates, and the two
// ids the frontend joins on. The fat columns an InvoiceDocument also carries — the CDR notes,
// the ticket, the digest, the last transport error — are not part of a book and are left behind.
//
// SeriesID is not a field: it is the tail of the document id, the same way the invoicing page
// reads it.
type SalesBookDocument struct {
	ID               int64 `json:",omitempty"`
	IssueDate        int16 `json:",omitempty"`
	Correlativo      int32 `json:",omitempty"`
	ClientSnapshotID int32 `json:",omitempty"`
	Currency         int8  `json:",omitempty"`

	TaxableAmount    int64 `json:",omitempty"`
	TaxAmount        int64 `json:",omitempty"`
	ExemptAmount     int64 `json:",omitempty"`
	UnaffectedAmount int64 `json:",omitempty"`
	TotalAmount      int64 `json:",omitempty"`

	AffectedDocID int64 `json:",omitempty"`
	State         int8  `json:",omitempty"`
}

// UninvoicedSale is a sale of the period that no comprobante was ever issued for.
//
// It is NOT a book row — the Registro de Ventas is a register of comprobantes — and the page
// shows it in its own block. It is here because the question "what did we sell that we never
// invoiced?" is the one thing a month-end review cannot answer from the book itself.
type UninvoicedSale struct {
	ID          int64 `json:",omitempty"`
	Date        int16 `json:",omitempty"`
	TotalAmount int32 `json:",omitempty"`
	TaxAmount   int32 `json:",omitempty"`
	Status      int8  `json:"ss,omitempty"`
}

type SalesBookResponse struct {
	Period   string `json:",omitempty"`
	FirstDay int16  `json:",omitempty"`
	LastDay  int16  `json:",omitempty"`

	Documents []SalesBookDocument `json:",omitempty"`
	// AffectedDocuments are the comprobantes this period's notes correct and that the period
	// does not already contain — a note very often corrects an earlier month. They are sent
	// apart from Documents because they are not rows of this book; the page joins them.
	AffectedDocuments []SalesBookDocument `json:",omitempty"`
	UninvoicedSales   []UninvoicedSale    `json:",omitempty"`
}

// GetSalesBook reads a whole period, or the single day inside one.
//
// A single day is a working view, not a filable book. The period travels back so the page and
// a later export can never disagree about which period a day belongs to.
func GetSalesBook(req *core.HandlerArgs) core.HandlerResponse {
	firstDay, lastDay, period, argsError := resolveBookRange(req)
	if argsError != nil {
		return req.MakeErr(argsError.Error())
	}

	documents, documentsError := loadPeriodDocuments(req.User.CompanyID, firstDay, lastDay)
	if documentsError != nil {
		return req.MakeErr(documentsError.Error())
	}
	affected, affectedError := loadModifiedDocuments(req.User.CompanyID, documents)
	if affectedError != nil {
		return req.MakeErr(affectedError.Error())
	}
	uninvoiced, uninvoicedError := loadUninvoicedSales(req.User.CompanyID, firstDay, lastDay)
	if uninvoicedError != nil {
		return req.MakeErr(uninvoicedError.Error())
	}

	core.Log("GetSalesBook period:", period, "documents:", len(documents),
		"affected:", len(affected), "uninvoiced:", len(uninvoiced))
	return req.MakeResponse(SalesBookResponse{
		Period:            period,
		FirstDay:          firstDay,
		LastDay:           lastDay,
		Documents:         documents,
		AffectedDocuments: affected,
		UninvoicedSales:   uninvoiced,
	})
}

// resolveBookRange reads either a whole period or the single day inside one.
func resolveBookRange(req *core.HandlerArgs) (int16, int16, string, error) {
	if singleDay := req.GetQueryInt16("date"); singleDay > 0 {
		return singleDay, singleDay, types.PeriodOf(singleDay), nil
	}

	period := req.Query["period"]
	if period == "" {
		return 0, 0, "", core.Err("Debe indicar el periodo (YYYYMM) o la fecha.")
	}
	firstDay, lastDay, boundsError := types.PeriodBounds(period)
	if boundsError != nil {
		return 0, 0, "", boundsError
	}
	return firstDay, lastDay, period, nil
}

// loadPeriodDocuments reads every comprobante issued inside the range.
//
// Through the index group and not a plain range: Scylla only takes equality on a secondary
// index, so a BETWEEN on IssueDate is refused. The group turns the range into one indexed read
// per day, which is also why a book is asked for a month and never for a year.
func loadPeriodDocuments(companyID int32, firstDay, lastDay int16) ([]SalesBookDocument, error) {
	groups := []db.RecordGroup[invoicing.InvoiceDocument]{}
	query := db.QueryIndexGroup(&groups).CompanyID.Equals(companyID)
	query.IssueDate.Between(firstDay, lastDay)

	if queryError := query.Exec(); queryError != nil {
		return nil, core.Err("Error al obtener los comprobantes del periodo.", queryError)
	}

	documents := []SalesBookDocument{}
	for groupIndex := range groups {
		for recordIndex := range groups[groupIndex].Records {
			documents = append(documents, makeBookDocument(&groups[groupIndex].Records[recordIndex]))
		}
	}
	return documents, nil
}

// loadModifiedDocuments reads the comprobantes this period's notes correct and that are not
// already in the period.
func loadModifiedDocuments(
	companyID int32, documents []SalesBookDocument,
) ([]SalesBookDocument, error) {

	inPeriod := make(map[int64]bool, len(documents))
	for index := range documents {
		inPeriod[documents[index].ID] = true
	}

	wantedIDs := core.SliceSet[int64]{}
	for index := range documents {
		affectedID := documents[index].AffectedDocID
		if affectedID != 0 && !inPeriod[affectedID] {
			wantedIDs.Add(affectedID)
		}
	}
	if wantedIDs.IsEmpty() {
		return nil, nil
	}

	earlier := []invoicing.InvoiceDocument{}
	query := db.Query(&earlier)
	query.Select().CompanyID.Equals(companyID).ID.In(wantedIDs.Values...)
	if queryError := query.Exec(); queryError != nil {
		return nil, core.Err("Error al leer los comprobantes modificados por las notas.", queryError)
	}

	affected := make([]SalesBookDocument, 0, len(earlier))
	for index := range earlier {
		affected = append(affected, makeBookDocument(&earlier[index]))
	}
	return affected, nil
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
	}
}

// loadUninvoicedSales reads the sales of the range that carry no comprobante.
//
// A sale whose id ends in 00 was never registered for electronic issuance, so no document was
// ever reserved for it. An annulled sale is left out: nothing was sold.
func loadUninvoicedSales(companyID int32, firstDay, lastDay int16) ([]UninvoicedSale, error) {
	groups := []db.RecordGroup[sales.SaleOrder]{}
	query := db.QueryIndexGroup(&groups).CompanyID.Equals(companyID)
	query.Date.Between(firstDay, lastDay)

	if queryError := query.Exec(); queryError != nil {
		return nil, core.Err("Error al obtener las ventas del periodo.", queryError)
	}

	uninvoiced := []UninvoicedSale{}
	for groupIndex := range groups {
		for recordIndex := range groups[groupIndex].Records {
			order := &groups[groupIndex].Records[recordIndex]
			if order.SeriesID() != 0 || order.Status == sales.OrderStatusAnnulled {
				continue
			}
			uninvoiced = append(uninvoiced, UninvoicedSale{
				ID:          order.ID,
				Date:        order.Date,
				TotalAmount: order.TotalAmount,
				TaxAmount:   order.TaxAmount,
				Status:      order.Status,
			})
		}
	}
	return uninvoiced, nil
}
