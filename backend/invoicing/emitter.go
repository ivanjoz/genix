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
//
// The correlativo is not chosen here. The ORM allocates it from a Scylla counter
// partitioned by series when the row is inserted, so two concurrent sales cannot
// take the same number without one of them failing on the primary key.
func ReserveDocument(
	companyID, userID int32, order *sales.SaleOrder, series *types.InvoiceSeries,
) (*types.InvoiceDocument, error) {

	if existing, err := FindBySaleOrder(companyID, order.ID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("la venta ya tiene el comprobante %v", existing.Number())
	}

	document, lineProductIDs, err := SaleOrderToDocument(companyID, order, series)
	if err != nil {
		return nil, err
	}

	// Complete and validate before anything is written: a document SUNAT would
	// reject should not consume a number at all.
	//
	// The number is the one thing that cannot be checked yet — the ORM assigns
	// it on insert — so validation runs against a stand-in. Emit validates again
	// at send time, by which point the real number is in place.
	if err := model.CompleteTotals(document); err != nil {
		return nil, err
	}
	document.Correlativo = 1
	if problems := model.ValidateDocument(document); len(problems) > 0 {
		return nil, model.JoinProblems(problems)
	}
	document.Correlativo = 0

	row := rowFromDocument(companyID, userID, order, series, document, lineProductIDs)
	rows := &[]types.InvoiceDocument{row}
	if err := db.Insert(rows); err != nil {
		return nil, fmt.Errorf("error al reservar el correlativo: %w", err)
	}

	saved := (*rows)[0]
	return &saved, nil
}

// rowFromDocument flattens a document into the row that will outlive it.
//
// The lines are copied out in full rather than referenced, because the row has
// to be able to rebuild the document on its own: a retry that happens tomorrow
// must produce the same XML even if the product was renamed in between.
func rowFromDocument(
	companyID, userID int32, order *sales.SaleOrder,
	series *types.InvoiceSeries, document *model.Document, lineProductIDs []int32,
) types.InvoiceDocument {

	now := core.SUnixTime()
	row := types.InvoiceDocument{
		CompanyID:     companyID,
		DocTypeSeries: types.PackDocTypeSeries(series.DocType, series.SeriesID),
		DocType:       series.DocType,
		SeriesID:      series.SeriesID,
		SeriesCode:    series.SeriesCode,
		IssueDate:     core.FechaUnix(),
		IssueTime:     now,
		SaleOrderID:   order.ID,
		ClientID:      order.ClientID,

		ClientDocNumber: document.Customer.DocNumber,
		ClientName:      document.Customer.LegalName,
		ClientDocType:   identityDocTypeCode(document.Customer.DocType),

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
		UpdatedBy: userID,
	}

	for index := range document.Lines {
		line := &document.Lines[index]
		// Not order.DetailProductsIDs[index]: a sale line with a sub-unit part produces two
		// document lines, so the two slices are no longer index-aligned.
		row.DetailProductIDs = append(row.DetailProductIDs, core.GetIndex(lineProductIDs, index))
		quantity := int32(line.Quantity / model.QuantityScale)
		row.DetailQuantity = append(row.DetailQuantity, quantity)
		// The unit value is derived when the caller priced the line as a whole,
		// which is the usual path here: a column that always stored zero would
		// only mislead whoever reads the row later.
		unitValue := line.UnitValue
		if unitValue == 0 && quantity > 0 {
			unitValue = line.Value / model.Cents(quantity)
		}
		row.DetailUnitValue = append(row.DetailUnitValue, int32(unitValue))
		row.DetailValue = append(row.DetailValue, int32(line.Value))
		row.DetailIgvAmount = append(row.DetailIgvAmount, int32(line.IGV))
		row.DetailIgvType = append(row.DetailIgvType, igvTypeCode(line.IgvType))
		row.DetailUnitCode = append(row.DetailUnitCode, line.UnitCode)
		row.DetailDescription = append(row.DetailDescription, line.Description)
	}
	return row
}

// DocumentFromRow rebuilds what was reserved, so a transmission that never
// happened can be attempted again without consulting the sale.
func DocumentFromRow(row *types.InvoiceDocument) *model.Document {
	document := &model.Document{
		Type:        sunatDocType(row.DocType),
		Series:      row.SeriesCode,
		Correlativo: int64(row.Correlativo()),
		IssuedAt:    time.Unix(core.SunixToUnix(row.IssueTime), 0),
		Currency:    model.DefaultCurrency,
		Payment:     model.PaymentCash,
		Customer: model.Party{
			DocType:   identityDocTypeOfCode(row.ClientDocType),
			DocNumber: row.ClientDocNumber,
			LegalName: row.ClientName,
		},
	}

	for index := range row.DetailDescription {
		document.Lines = append(document.Lines, model.Line{
			ProductCode: "",
			Description: core.GetIndex(row.DetailDescription, index),
			Quantity:    model.Units(int(core.GetIndex(row.DetailQuantity, index))),
			UnitCode:    core.GetIndex(row.DetailUnitCode, index),
			IgvType:     igvTypeOfCode(core.GetIndex(row.DetailIgvType, index)),
			IgvPercent:  model.Percent(igvPercentHundredths),
			Value:       model.Cents(core.GetIndex(row.DetailValue, index)),
			IGV:         model.Cents(core.GetIndex(row.DetailIgvAmount, index)),
		})
	}
	return document
}

// FindBySaleOrder answers whether a sale has already been invoiced. Asked before
// every emission, because the second document for one sale is the mistake that
// cannot be undone without a credit note.
func FindBySaleOrder(companyID int32, saleOrderID int64) (*types.InvoiceDocument, error) {
	documents := []types.InvoiceDocument{}
	query := db.Query(&documents)
	query.Select().CompanyID.Equals(companyID).SaleOrderID.Equals(saleOrderID)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al verificar si la venta ya fue facturada: %w", err)
	}
	for index := range documents {
		// A rejected document does not exist for SUNAT, and neither does one
		// that was discarded, so in both cases the sale may be invoiced again.
		if documents[index].Status == 0 || documents[index].State == types.InvoiceRejected {
			continue
		}
		return &documents[index], nil
	}
	return nil, nil
}

// LoadDocument reads one document by its packed key.
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

// Identity document codes are stored as small integers and travel as catalog-06
// strings. The two conversions live together so they cannot drift apart.
func identityDocTypeCode(docType string) int8 {
	switch docType {
	case model.IDDocDNI:
		return 1
	case model.IDDocForeign:
		return 4
	case model.IDDocRUC:
		return 6
	case model.IDDocPassport:
		return 7
	}
	return 0
}

func identityDocTypeOfCode(code int8) string {
	switch code {
	case 1:
		return model.IDDocDNI
	case 4:
		return model.IDDocForeign
	case 6:
		return model.IDDocRUC
	case 7:
		return model.IDDocPassport
	}
	return model.IDDocNone
}

// The IGV affectation is catalog 07, stored numerically.
func igvTypeCode(igvType string) int8 {
	switch igvType {
	case model.IgvExempt:
		return 20
	case model.IgvUnaffected:
		return 30
	case model.IgvExport:
		return 40
	case model.IgvFreeTaxed:
		return 11
	case model.IgvIVAP:
		return 17
	}
	return 10
}

func igvTypeOfCode(code int8) string {
	switch code {
	case 20:
		return model.IgvExempt
	case 30:
		return model.IgvUnaffected
	case 40:
		return model.IgvExport
	case 11:
		return model.IgvFreeTaxed
	case 17:
		return model.IgvIVAP
	}
	return model.IgvTaxed
}
