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

// Reserving a document and sending it are separate steps, and the split is the
// whole design. Reserving happens with the sale, is fast, and lives in
// invoicing/types so `sales` can reach it. Sending takes seconds and can fail in
// ways that say nothing about the document, so it runs from a cron, off the
// request — and because the row was already persisted, a failed transmission is
// retried with the number it already has instead of burning a new one.

// RebuildDocument reconstructs what was reserved, from the sale it bills.
//
// The issue date comes off the row, not the clock: a retry tomorrow must declare
// the day the document was issued, not the day it finally got through.
func RebuildDocument(companyID int32, row *types.InvoiceDocument,
	series *types.InvoiceSeries) (*model.Document, error) {

	order, err := loadSaleOrder(companyID, row.SaleOrderID())
	if err != nil {
		return nil, err
	}

	document, _, err := types.SaleOrderToDocument(companyID, order, series)
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

// loadSaleOrder reads the sale being invoiced.
func loadSaleOrder(companyID int32, saleOrderID int64) (*sales.SaleOrder, error) {
	orders := []sales.SaleOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(companyID).ID.Equals(saleOrderID)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer la venta: %w", err)
	}
	if len(orders) == 0 {
		return nil, errors.New("la venta no existe")
	}
	if orders[0].Status == sales.OrderStatusAnnulled {
		return nil, errors.New("la venta está anulada")
	}
	return &orders[0], nil
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
