package invoicing

import (
	"app/core"
	"app/db"
	"app/invoicing/types"
	sales "app/sales/types"
	"errors"
	"fmt"
	"time"

	"github.com/ivanjoz/facturago/model"
)

// ReserveDocument turns a sale into a numbered, persisted document — and stops
// there.
//
// The split between reserving and sending is the whole design. Reserving is fast
// and transactional; sending takes seconds and can fail in ways that say nothing
// about the document. Persisting in between means a failed transmission is
// retried with the number it already has, instead of burning a new one on every
// attempt.
func ReserveDocument(
	companyID, userID int32, order *sales.SaleOrder, series *types.InvoiceSeries,
) (*types.InvoiceDocument, error) {

	if existing, err := types.FindBySaleOrder(companyID, order.ID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("la venta ya tiene el comprobante %v",
			existing.Number(series.SeriesCode))
	}

	document, _, err := SaleOrderToDocument(companyID, order, series)
	if err != nil {
		return nil, err
	}

	// Complete and validate before a number is spent: a document SUNAT would
	// reject should not consume one at all. The correlativo is the only thing not
	// yet known, so validation runs against a stand-in — Emit validates again at
	// send time with the real number in place.
	if err := model.CompleteTotals(document); err != nil {
		return nil, err
	}
	document.Correlativo = 1
	if problems := model.ValidateDocument(document); len(problems) > 0 {
		return nil, model.JoinProblems(problems)
	}

	// The number comes from the series' own counter, reserved through the
	// fareward allocator — the same path every id in the system takes, and the
	// reason two tills cannot be handed the same correlativo.
	correlativo, err := db.GetAutoincrementID(
		types.CorrelativoCounterName(companyID, series.SeriesID), 1)
	if err != nil {
		return nil, fmt.Errorf("error al reservar el correlativo: %w", err)
	}
	document.Correlativo = correlativo

	row := rowFromDocument(companyID, userID, order, series, document, correlativo)
	rows := &[]types.InvoiceDocument{row}
	if err := db.Insert(rows); err != nil {
		return nil, fmt.Errorf("error al guardar el comprobante: %w", err)
	}

	saved := (*rows)[0]
	return &saved, nil
}

// rowFromDocument records what only the row knows: which sale, which number, and
// what it totalled.
//
// The lines are deliberately not copied. They are rebuilt from the sale on every
// send, and once SUNAT accepts the document the signed XML in object storage is
// the artifact that matters — a second copy in the row could only disagree with it.
func rowFromDocument(
	companyID, userID int32, order *sales.SaleOrder,
	series *types.InvoiceSeries, document *model.Document, correlativo int64,
) types.InvoiceDocument {

	now := core.SUnixTime()
	return types.InvoiceDocument{
		CompanyID:   companyID,
		ID:          types.DocumentIDForSale(order.ID, series.SeriesID),
		Correlativo: int32(correlativo),
		IssueDate:   core.FechaUnix(),
		IssueTime:   now,

		Currency:         types.CurrencyPEN,
		TotalAmount:      int64(document.Totals.Payable),
		TaxAmount:        int64(document.Totals.TotalTaxes),
		TaxableAmount:    int64(document.Totals.Taxable),
		ExemptAmount:     int64(document.Totals.Exempt),
		UnaffectedAmount: int64(document.Totals.Unaffected),
		FreeAmount:       int64(document.Totals.Free),

		State:     types.InvoicePending,
		Status:    1,
		Created:   now,
		CreatedBy: userID,
		Updated:   now,
	}
}

// RebuildDocument reconstructs what was reserved, from the sale it bills.
//
// The issue date comes off the row, not the clock: a retry tomorrow must declare
// the day the document was issued, not the day it finally got through.
func RebuildDocument(companyID int32, row *types.InvoiceDocument,
	series *types.InvoiceSeries) (*model.Document, error) {

	order, err := loadSaleOrder(companyID, saleIDOfDocument(row))
	if err != nil {
		return nil, err
	}

	document, _, err := SaleOrderToDocument(companyID, order, series)
	if err != nil {
		return nil, err
	}
	document.Correlativo = int64(row.Correlativo)
	document.IssuedAt = time.Unix(core.SunixToUnix(row.IssueTime), 0)

	if err := model.CompleteTotals(document); err != nil {
		return nil, err
	}
	return document, nil
}

// LoadDocument reads one document by its id.
func LoadDocument(companyID int32, documentID int64) (*types.InvoiceDocument, error) {
	documents := []types.InvoiceDocument{}
	query := db.Query(&documents)
	query.Select().CompanyID.Equals(companyID).ID.Equals(documentID)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer el comprobante: %w", err)
	}
	if len(documents) == 0 {
		return nil, errors.New("el comprobante no existe")
	}
	return &documents[0], nil
}

// saleIDOfDocument is the sale a document bills.
//
// A document that bills a sale is keyed by that sale — the sale carries the series
// it will be issued under, so the two ids are the same value. A note is issued
// under its own series and lands elsewhere, so it reaches the sale through the
// document it corrects, which is the one keyed by the sale.
func saleIDOfDocument(row *types.InvoiceDocument) int64 {
	if row.AffectedDocID != 0 {
		return row.AffectedDocID
	}
	return row.ID
}

// seriesOfDocument resolves the series a document was issued under, from the tail
// of its own id.
func seriesOfDocument(companyID int32, row *types.InvoiceDocument) (*types.InvoiceSeries, error) {
	allSeries, err := LoadCompanySeries(companyID)
	if err != nil {
		return nil, err
	}
	series := types.FindSeries(allSeries, row.SeriesID())
	if series == nil {
		return nil, fmt.Errorf("la serie %v del comprobante ya no existe", row.SeriesID())
	}
	return series, nil
}
