package invoicing

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/invoicing/types"
)

// There is no endpoint that issues a document.
//
// A document is created with the sale it bills — `sales` reserves and writes it in
// the same request — and the cron sweep sends it. What is left here is reading the
// documents, downloading their artifacts, and pushing one out ahead of the sweep.

// GetInvoices lists issued documents, with the delta the frontend caches on.
//
// The fan-out names every state on purpose. The delta index is keyed on State, so
// a first sync (watermark 0) returns only the states named here — asking for the
// pending ones alone, as this did, hid every document the moment it was sent.
func GetInvoices(req *core.HandlerArgs) core.HandlerResponse {
	updatedVersion := req.GetUpVersion()

	documents := []types.InvoiceDocument{}
	query := db.Query(&documents)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(updatedVersion, types.AllInvoiceStates...)

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

// PostInvoiceRetry sends a document now, with the number it already has, and waits
// for SUNAT.
//
// It covers both "this one is waiting for the sweep and I want it out now" and
// "this one failed and I fixed whatever caused it". The operator pressed the button
// to learn the answer, so the request holds until there is one and returns the
// document with SUNAT's verdict already on it — no polling, no second click to find
// out what happened.
//
// The wait is real: SUNAT regularly takes tens of seconds and the client gives up
// at 45. On the standalone server nothing cuts the request short; behind an API
// Gateway with a 30-second limit the caller can time out while the send finishes
// anyway, which is safe — the row is written by the send, not by this handler, so
// the state is correct whether or not the answer got back.
//
// A rejection is not retryable by definition: SUNAT read the XML and refused it, so
// the fix is a corrected document, not another attempt at this one.
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
	case types.InvoiceVoided:
		return req.MakeErr("El comprobante fue anulado y no puede enviarse.")
	}

	// The attempt counter is cleared because this one is a person's decision, not
	// the automatic retry: it must not inherit a spent budget and stop halfway.
	document.RetryCount = 0
	saveDocument(document)

	sent, sendErr := SendDocumentNow(req.User.CompanyID, documentID)
	if sendErr != nil {
		return req.MakeErr("SUNAT rechazó o no recibió el comprobante:", sendErr)
	}
	return req.MakeResponse(sent)
}
