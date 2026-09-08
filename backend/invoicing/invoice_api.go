package invoicing

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/invoicing/types"
	sales "app/sales/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// PostInvoiceBody is what the till sends to issue a document for a sale.
type PostInvoiceBody struct {
	SaleOrderID int64 `json:"SaleOrderID"`
	// DocType is optional and only asserts what the caller believes it is issuing.
	// The series is not a parameter: it is fixed when the sale is created, because
	// the document is keyed by the sale.
	DocType int8 `json:"DocType"`
}

// PostInvoice issues an electronic document for a sale.
//
// It returns as soon as the document has a number, without waiting for SUNAT.
// The transmission takes seconds and the answer is not needed to hand the
// customer their receipt — the document exists, is numbered, and the state moves
// on its own. The alternative, holding the request open, means the till freezes
// whenever SUNAT is slow.
func PostInvoice(req *core.HandlerArgs) core.HandlerResponse {
	body := PostInvoiceBody{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	if body.SaleOrderID == 0 {
		return req.MakeErr("Debe indicar la venta a facturar.")
	}

	companyID := req.User.CompanyID

	// Two clicks on the same sale must not produce two documents. The lock is
	// keyed on the sale, so unrelated sales never queue behind each other.
	lock, lockErr := core.AcquireLock(context.Background(), core.ActionInvoiceSaleOrder, body.SaleOrderID, 2)
	if lockErr != nil {
		return lockErr.Response(req)
	}
	defer lock.Release()

	order, err := loadSaleOrder(companyID, body.SaleOrderID)
	if err != nil {
		return req.MakeErr(err.Error())
	}
	series, err := seriesOfSale(companyID, order, body.DocType)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	document, err := ReserveDocument(companyID, req.User.ID, order, series)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	// From here the document exists and is numbered; everything that follows can
	// be retried without the caller.
	SendDocumentAsync(companyID, document.ID)

	return req.MakeResponse(document)
}

// GetInvoices lists issued documents, with the delta the frontend caches on.
func GetInvoices(req *core.HandlerArgs) core.HandlerResponse {
	updatedVersion := req.GetQueryInt("upv")

	documents := []types.InvoiceDocument{}
	query := db.Query(&documents)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(updatedVersion, 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los comprobantes:", err)
	}
	return core.MakeResponse(req, &map[string]any{"Invoices": documents})
}

// GetInvoiceXML downloads the signed document or the CDR SUNAT returned.
//
// Both have to be kept for five years, and this is how they are retrieved. The
// row holds neither the bytes nor a path: the object name is derived from the
// document id, which is already unique per sale and series.
func GetInvoiceXML(req *core.HandlerArgs) core.HandlerResponse {
	documentID := req.GetQueryInt64("id")
	if documentID == 0 {
		return req.MakeErr("Debe indicar el comprobante.")
	}

	document, err := LoadDocument(req.User.CompanyID, documentID)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	// The artifact names are derived from the document id, so there is no stored
	// path to consult — but there is also nothing to tell us the file exists yet.
	// A document that never reached the signing step has no XML, and object
	// storage answering "not found" is what says so.
	kind, contentType := xmlArtifactName, "application/xml"
	if req.Query["tipo"] == "cdr" {
		kind, contentType = cdrArtifactName, "application/zip"
	}
	if document.State == types.InvoicePending {
		return req.MakeErr("El comprobante todavía no ha sido firmado ni enviado.")
	}

	content, err := cloud.GetFile(cloud.SaveFileArgs{
		Bucket: core.Env.S3_BUCKET,
		Path:   ArtifactFolder(req.User.CompanyID),
		Name:   ArtifactName(document.ID, kind),
	})
	if err != nil {
		return req.MakeErr("No se pudo leer el archivo del comprobante:", err)
	}

	response := req.MakeResponsePlain(&content)
	response.Headers = map[string]string{"Content-Type": contentType}
	return response
}

// PostInvoiceRetry sends a document again, with the number it already has.
//
// Only a document that failed on the way out can be retried. A rejection is not
// retryable by definition: SUNAT read the XML and refused it, so the fix is a
// corrected document, not another attempt at this one.
func PostInvoiceRetry(req *core.HandlerArgs) core.HandlerResponse {
	documentID := req.GetQueryInt64("id")
	if documentID == 0 {
		return req.MakeErr("Debe indicar el comprobante.")
	}

	document, err := LoadDocument(req.User.CompanyID, documentID)
	if err != nil {
		return req.MakeErr(err.Error())
	}
	switch document.State {
	case types.InvoiceAccepted, types.InvoiceObserved:
		return req.MakeErr("El comprobante ya fue aceptado por SUNAT.")
	case types.InvoiceRejected:
		return req.MakeErr("El comprobante fue rechazado por SUNAT. Debe emitir uno corregido.")
	}

	document.RetryCount = 0
	saveDocument(document)
	SendDocumentAsync(req.User.CompanyID, document.ID)

	return req.MakeResponse(document)
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

// seriesOfSale is the series a sale's document is numbered in, and it is not a
// choice: the sale carries it in the last two digits of its id, and the document
// is keyed by the sale, so any other series would key the document somewhere the
// sale cannot be found.
//
// A sale whose tail is 00 was never registered for electronic invoicing — the
// company had none configured when it was made — and cannot be invoiced now
// without changing its id, which is its identity everywhere else in the system.
func seriesOfSale(companyID int32, order *sales.SaleOrder, assertedDocType int8) (*types.InvoiceSeries, error) {
	seriesID := order.SeriesID()
	if seriesID == 0 {
		return nil, errors.New(
			"la venta no fue registrada para facturación electrónica y no puede facturarse")
	}

	allSeries, err := LoadCompanySeries(companyID)
	if err != nil {
		return nil, err
	}
	series := types.FindSeries(allSeries, seriesID)
	if series == nil {
		return nil, fmt.Errorf("la serie %v con la que se registró la venta ya no existe", seriesID)
	}
	if series.Status != 1 {
		return nil, fmt.Errorf("la serie %v con la que se registró la venta está inactiva", series.SeriesCode)
	}
	// The caller may say what it expects to issue; it may not choose something else.
	if assertedDocType != 0 && assertedDocType != series.DocType {
		return nil, fmt.Errorf(
			"la venta se registró para la serie %v y no puede emitirse como otro tipo de comprobante",
			series.SeriesCode)
	}
	return series, nil
}
