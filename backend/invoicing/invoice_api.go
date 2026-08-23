package invoicing

import (
	"app/cloud"
	"app/core"
	"app/db"
	invoicingTypes "app/invoicing/types"
	salesTypes "app/sales/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// PostInvoiceBody is what the till sends to issue a document for a sale.
type PostInvoiceBody struct {
	SaleOrderID int64 `json:"SaleOrderID"`
	// SeriesID names the series to number the document in. Omitted, the
	// company's default series for that document type is used.
	SeriesID int16 `json:"SeriesID"`
	DocType  int8  `json:"DocType"`
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
	series, err := resolveSeries(companyID, body.DocType, body.SeriesID, order)
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

	documents := []invoicingTypes.InvoiceDocument{}
	query := db.Query(&documents)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(updatedVersion, 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los comprobantes:", err)
	}
	return core.MakeResponse(req, &map[string]any{"Invoices": documents})
}

// GetInvoiceXML downloads the signed document or the CDR SUNAT returned.
//
// Both have to be kept for five years, and this is how they are retrieved: the
// row holds a path, not the bytes.
func GetInvoiceXML(req *core.HandlerArgs) core.HandlerResponse {
	documentID := req.GetQueryInt64("id")
	if documentID == 0 {
		return req.MakeErr("Debe indicar el comprobante.")
	}

	document, err := LoadDocument(req.User.CompanyID, documentID)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	path, contentType := document.XmlPath, "application/xml"
	if req.Query["tipo"] == "cdr" {
		path, contentType = document.CdrPath, "application/zip"
	}
	if path == "" {
		return req.MakeErr("El comprobante todavía no tiene ese archivo.")
	}

	content, err := cloud.GetFile(cloud.SaveFileArgs{
		Bucket: core.Env.S3_BUCKET,
		Path:   pathFolder(path),
		Name:   pathName(path),
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
	case invoicingTypes.InvoiceAccepted, invoicingTypes.InvoiceObserved:
		return req.MakeErr("El comprobante ya fue aceptado por SUNAT.")
	case invoicingTypes.InvoiceRejected:
		return req.MakeErr("El comprobante fue rechazado por SUNAT. Debe emitir uno corregido.")
	}

	document.RetryCount = 0
	saveDocument(document)
	SendDocumentAsync(req.User.CompanyID, document.ID)

	return req.MakeResponse(document)
}

// loadSaleOrder reads the sale being invoiced.
func loadSaleOrder(companyID int32, saleOrderID int64) (*salesTypes.SaleOrder, error) {
	orders := []salesTypes.SaleOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(companyID).ID.Equals(saleOrderID)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer la venta: %w", err)
	}
	if len(orders) == 0 {
		return nil, errors.New("la venta no existe")
	}
	if orders[0].Status == salesTypes.OrderStatusAnnulled {
		return nil, errors.New("la venta está anulada")
	}
	return &orders[0], nil
}

// resolveSeries picks the series to number the document in.
//
// Naming one explicitly wins. Otherwise the company's default for that document
// type is used, which is what a till does: it knows it is selling, not which
// series the accountant set up.
func resolveSeries(companyID int32, docType int8, seriesID int16,
	order *salesTypes.SaleOrder) (*invoicingTypes.InvoiceSeries, error) {

	if docType == 0 {
		docType = invoicingTypes.DocTypeBoleta
	}

	series := []invoicingTypes.InvoiceSeries{}
	query := db.Query(&series)
	query.Select().CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las series: %w", err)
	}

	var chosen, fallback *invoicingTypes.InvoiceSeries
	for index := range series {
		candidate := &series[index]
		if candidate.Status != 1 || candidate.DocType != docType {
			continue
		}
		if seriesID != 0 && candidate.SeriesID == seriesID {
			chosen = candidate
			break
		}
		if seriesID == 0 && (candidate.IsDefault == 1 || fallback == nil) {
			fallback = candidate
			if candidate.IsDefault == 1 {
				break
			}
		}
	}
	if chosen == nil {
		chosen = fallback
	}
	if chosen == nil {
		return nil, errors.New("la empresa no tiene una serie configurada para ese tipo de comprobante")
	}
	return chosen, nil
}

// The stored artifact path is "folder/name"; object storage wants them apart.
func pathFolder(path string) string {
	for index := len(path) - 1; index >= 0; index-- {
		if path[index] == '/' {
			return path[:index]
		}
	}
	return ""
}

func pathName(path string) string {
	for index := len(path) - 1; index >= 0; index-- {
		if path[index] == '/' {
			return path[index+1:]
		}
	}
	return path
}
