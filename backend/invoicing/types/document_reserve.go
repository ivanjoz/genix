// Numbering a document, which is the step that cannot be undone.
//
// It is split from writing it because the two happen either side of the sale
// being written. Everything that can fail — building the document, validating it,
// reserving the number — runs first, so a sale that cannot be invoiced is rejected
// with nothing persisted. The row is inserted afterwards, once the sale it bills
// exists and can be read back by whoever sends it.

package types

import (
	"app/core"
	"app/db"
	sales "app/sales/types"
	"fmt"

	"github.com/ivanjoz/facturago/model"
)

// PrepareDocumentForSale builds the document for a sale and gives the sale its id.
//
// **It assigns order.ID**, which is not a side effect but the point: the sale id
// carries the correlativo of its series, so minting one reserves the other. There
// is no separate document counter.
//
// It writes nothing, and the sale is not read from the database either — the caller
// holds it, and at this point it is not stored yet.
//
// There is no "does this sale already have a document" check: the only caller is
// the sale's own creation, and the id was just minted, so nothing can be keyed on
// it yet. A second document for the same sale would collide on the primary key,
// which is the guarantee that matters.
func PrepareDocumentForSale(companyID, userID int32, order *sales.SaleOrder,
	series *InvoiceSeries) (*InvoiceDocument, error) {

	document, _, err := SaleOrderToDocument(companyID, order, series)
	if err != nil {
		return nil, err
	}

	// Complete and validate before a number is spent: a document SUNAT would
	// reject should not consume one at all, and with the counters merged that
	// would also leave a gap in the series. The correlativo is the only thing not
	// yet known, so validation runs against a stand-in — the send validates again
	// with the real number in place.
	if err := model.CompleteTotals(document); err != nil {
		return nil, err
	}
	document.Correlativo = 1
	if problems := model.ValidateDocument(document); len(problems) > 0 {
		return nil, model.JoinProblems(problems)
	}

	// Reserved through the fareward allocator — the same path every id in the
	// system takes, and the reason two tills cannot be handed the same correlativo.
	saleOrderID, err := sales.MakeSaleOrderID(companyID, series.SeriesID)
	if err != nil {
		return nil, fmt.Errorf("error al reservar el correlativo: %w", err)
	}
	order.ID = saleOrderID
	document.Correlativo = order.Correlativo()

	row := rowFromDocument(companyID, userID, order, series, document)
	return &row, nil
}

// SaveNewDocument writes the reserved row.
//
// Called after the sale it bills has been inserted: a document whose sale cannot
// be read is a document nobody can send, because every send rebuilds it from the
// sale.
func SaveNewDocument(document *InvoiceDocument) error {
	rows := &[]InvoiceDocument{*document}
	if err := db.Insert(rows); err != nil {
		return fmt.Errorf("error al guardar el comprobante: %w", err)
	}
	*document = (*rows)[0]
	return nil
}

// rowFromDocument records what only the row knows: which sale, which number, and
// what it totalled.
//
// The lines are deliberately not copied. They are rebuilt from the sale on every
// send, and once SUNAT accepts the document the signed XML in object storage is
// the artifact that matters — a second copy in the row could only disagree with it.
func rowFromDocument(
	companyID, userID int32, order *sales.SaleOrder,
	series *InvoiceSeries, document *model.Document,
) InvoiceDocument {

	now := core.SUnixTime()
	return InvoiceDocument{
		CompanyID: companyID,
		ID:        DocumentIDForSale(order.ID, series.SeriesID),
		// Stored as well as derivable: every list and report reads it, and a note
		// is numbered in its own series, so the column cannot always be the head of
		// the id.
		Correlativo: int32(order.Correlativo()),
		IssueDate:   core.FechaUnix(),
		IssueTime:   now,

		Currency:         CurrencyPEN,
		TotalAmount:      int64(document.Totals.Payable),
		TaxAmount:        int64(document.Totals.TotalTaxes),
		TaxableAmount:    int64(document.Totals.Taxable),
		ExemptAmount:     int64(document.Totals.Exempt),
		UnaffectedAmount: int64(document.Totals.Unaffected),
		FreeAmount:       int64(document.Totals.Free),

		State:     InvoicePending,
		Status:    1,
		Created:   now,
		CreatedBy: userID,
		Updated:   now,
	}
}
