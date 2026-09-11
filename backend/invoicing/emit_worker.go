package invoicing

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/invoicing/types"
	"context"
	"errors"
	"fmt"

	"github.com/ivanjoz/facturago"
	"github.com/ivanjoz/facturago/sunat"
)

// Sending, by whichever of the two paths asked for it.
//
// A document is written with its sale and waits in InvoicePending. Normally the
// sweep below picks it up on the next frame and nobody watches; when an operator
// presses "enviar ahora" the same send runs inside their request and they wait for
// SUNAT's answer. Both go through SendDocumentNow, so the lock and the state rules
// cannot differ between them.
//
// The per-document retry is the net underneath the sweep: it survives a panic,
// caps the attempts, and leaves a row somebody can read afterwards. Only a
// transport or configuration failure is retried. A rejection is final — the same
// XML will be refused forever, and re-sending it just fills a queue.
const (
	// EmitFunctionName is the command-line entry point, registered in exec.
	EmitFunctionName = "fn-emit-cpe"
	// emitRetryActionID is the per-document retry. Values 1 to 4 are taken by
	// other modules, and 6 is the sweep in invoicing/types.
	emitRetryActionID = int16(5)
	// retryFrameMinutes is the first retry delay; the executor keeps the cadence.
	retryFrameMinutes = int8(5)
	// maxRetries stops a document that keeps failing from being attempted forever.
	maxRetries = int8(6)
	// maxDocumentsPerSweep bounds one run. SUNAT answers in seconds, so an
	// unbounded batch would hold the cron worker for as long as the backlog is;
	// a full batch re-schedules instead.
	maxDocumentsPerSweep = 50
	// sendLockWaiters is how many callers may queue on one sale's lock. The sweep
	// and an operator pressing "enviar ahora" are the only contenders.
	sendLockWaiters = 2
)

// Both cron actions are registered here, next to what they run.
func init() {
	core.RegisterActionHandler(types.EmitPendingActionID,
		"Enviar comprobantes pendientes", EmitPendingDocumentsHandler)
	core.RegisterActionHandler(emitRetryActionID,
		"Reintentar envío de comprobante", EmitHandler)
}

// EmitPendingDocumentsHandler sends everything a company has waiting.
//
// It reads the pending bucket directly: the delta index on State serves a query
// that pins the partition and the state, so this is a range read and not a scan.
func EmitPendingDocumentsHandler(args *core.ExecArgs) core.FuncResponse {
	companyID := int32(args.Param1)
	if companyID == 0 {
		return args.MakeErr("falta la empresa para enviar los comprobantes pendientes")
	}

	pending, err := loadPendingDocuments(companyID)
	if err != nil {
		return args.MakeErr(err.Error())
	}
	args.AddMessage(core.Concat(" ", "Comprobantes pendientes:", len(pending)))

	for index := range pending {
		if _, sendErr := SendDocumentNow(companyID, pending[index].ID); sendErr != nil {
			core.Log("error al enviar el comprobante", pending[index].ID, sendErr)
			args.AddMessage(core.Concat(" ", "Comprobante", pending[index].ID, ":", sendErr))
		}
	}

	// A full batch means there is very likely more behind it. Anything that failed
	// left the pending bucket, so this cannot loop on the same rows.
	if len(pending) == maxDocumentsPerSweep {
		types.ScheduleEmitPendingSweep(companyID)
	}
	return core.FuncResponse{}
}

// loadPendingDocuments reads the documents waiting to be sent, oldest first.
func loadPendingDocuments(companyID int32) ([]types.InvoiceDocument, error) {
	documents := []types.InvoiceDocument{}
	query := db.Query(&documents)
	query.Select(query.ID).
		CompanyID.Equals(companyID).
		State.Equals(types.InvoicePending).
		Limit(maxDocumentsPerSweep)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los comprobantes pendientes: %w", err)
	}
	return documents, nil
}

// SendDocumentNow sends one document under the sale's lock and waits for SUNAT.
//
// It is what both the sweep and the "enviar ahora" endpoint run, so a document is
// never sent by two paths with different rules. The document it returns is the row
// as it stands after the attempt — including when the attempt failed, because a
// rejection is a state the caller has to show.
//
// The lock is the same one an annulment takes, and it is why a sale annulled a
// moment ago does not get its document sent anyway: the state is re-read inside
// it, after whoever was voiding the document has finished.
func SendDocumentNow(companyID int32, documentID int64) (*types.InvoiceDocument, error) {
	document, err := LoadDocument(companyID, documentID)
	if err != nil {
		return nil, err
	}

	lock, lockErr := core.AcquireLock(context.Background(),
		core.ActionInvoiceSaleOrder, document.SaleOrderID(), sendLockWaiters)
	if lockErr != nil {
		return nil, lockErr
	}
	defer lock.Release()

	current, err := LoadDocument(companyID, documentID)
	if err != nil {
		return nil, err
	}
	if !types.CanSendInvoice(current.State) {
		// Voided while we waited, or sent by whoever held the lock before us. The
		// row as it stands is the answer.
		return current, nil
	}

	sendErr := SendDocument(companyID, documentID)

	// Reloaded either way: SendDocument records what SUNAT said before it returns
	// the rejection, and that row is what the caller has to show.
	sent, loadErr := LoadDocument(companyID, documentID)
	if loadErr != nil {
		return nil, loadErr
	}
	return sent, sendErr
}

// EmitHandler is the body of the automatic retry (cron action 5) and the
// command-line entry point registered in exec as "fn-emit-cpe". Both are the same
// work on the same arguments, which is why they are the same function.
//
// It does not take the sale's lock, because the retry runs on a document that
// already failed and the CLI is a person deciding. SendDocumentNow is the path
// everything else goes through.
func EmitHandler(args *core.ExecArgs) core.FuncResponse {
	companyID, documentID := int32(args.Param1), args.Param2
	if companyID == 0 || documentID == 0 {
		return args.MakeErr("faltan los parámetros de la emisión")
	}

	if err := SendDocument(companyID, documentID); err != nil {
		core.Log("error al emitir el comprobante", documentID, err)
		return args.MakeErr(err.Error())
	}
	return core.FuncResponse{}
}

// SendDocument signs a reserved document, transmits it, and records what SUNAT
// answered.
func SendDocument(companyID int32, documentID int64) error {
	row, err := LoadDocument(companyID, documentID)
	if err != nil {
		return err
	}
	if row.State == types.InvoiceAccepted || row.State == types.InvoiceObserved {
		return nil // already issued; nothing to do
	}
	if row.State == types.InvoiceRejected {
		return errors.New("el comprobante fue rechazado por SUNAT y no puede reenviarse")
	}

	series, err := seriesOfDocument(companyID, row)
	if err != nil {
		return recordFailure(row, err, false)
	}

	issuer, err := BuildIssuer(companyID)
	if err != nil {
		return recordFailure(row, err, false)
	}

	// The document is rebuilt from the sale on every attempt rather than kept in
	// the row. Nothing that reaches the XML can drift between attempts: the number,
	// the issue date and the series come off the row, and the sale is frozen while
	// it has a live document.
	document, err := RebuildDocument(companyID, row, series)
	if err != nil {
		return recordFailure(row, err, false)
	}

	emission, err := facturago.Emit(issuer, document)
	if err != nil {
		// A build or validation failure is about the document, not the network.
		return recordFailure(row, err, false)
	}

	if err := storeArtifact(companyID, row, xmlArtifactName, emission.XML, "application/xml"); err != nil {
		return recordFailure(row, err, true)
	}
	row.DigestValue = emission.Digest
	row.State = types.InvoiceQueued
	saveDocument(row)

	cdr, err := new(sunat.Client).SendBill(issuer, emission)
	if err != nil {
		return recordFailure(row, err, sunat.Retryable(err))
	}

	if len(cdr.Zip) > 0 {
		if storeErr := storeArtifact(companyID, row, cdrArtifactName, cdr.Zip, "application/zip"); storeErr != nil {
			core.Log("no se pudo guardar la CDR del comprobante", row.ID, storeErr)
		}
	}

	row.SunatCode = cdr.Code
	row.SunatNotes = cdr.Notes
	row.State = stateFromSeverity(cdr.Severity)
	row.LastError = ""
	saveDocument(row)

	if !cdr.Accepted() {
		return cdr.Err()
	}
	return nil
}

// recordFailure writes down what went wrong and decides whether to try again.
func recordFailure(row *types.InvoiceDocument, cause error, retryable bool) error {
	row.LastError = core.StrCut(cause.Error(), 500)

	var sunatError *sunat.Error
	if errors.As(cause, &sunatError) {
		row.SunatCode = sunatError.Code
		row.State = stateFromSeverity(sunatError.Severity)
	} else {
		// Retryable or not, the state is the same; what differs is whether a retry
		// is scheduled below.
		row.State = types.InvoiceException
	}

	if retryable && row.RetryCount < maxRetries {
		row.RetryCount++
		core.ScheduleCronAction(core.CronAction{
			ActionID:  emitRetryActionID,
			CompanyID: row.CompanyID,
			Params: core.ExecArgs{
				Param1: int64(row.CompanyID),
				Param2: row.ID,
			},
		}, retryFrameMinutes)
	}

	saveDocument(row)
	return cause
}

// stateFromSeverity maps SUNAT's verdict onto the row's state.
func stateFromSeverity(severity sunat.Severity) int8 {
	switch severity {
	case sunat.SeverityAccepted:
		return types.InvoiceAccepted
	case sunat.SeverityObserved:
		return types.InvoiceObserved
	case sunat.SeverityRejected:
		return types.InvoiceRejected
	}
	return types.InvoiceException
}

// saveDocument writes a state transition.
//
// The columns are listed rather than writing the whole row: everything that
// identifies the document — its number, its customer, its lines — is immutable
// once reserved, and naming only what moves makes that explicit. State travels
// with the delta view's key, which the ORM requires to be written together.
func saveDocument(row *types.InvoiceDocument) {
	row.Updated = core.SUnixTime()
	rows := &[]types.InvoiceDocument{*row}

	table := db.TableOf[types.InvoiceDocument]()
	err := db.Update(rows,
		table.State, table.Status, table.Updated,
		table.SunatCode, table.SunatNotes, table.Ticket,
		table.DigestValue, table.RetryCount, table.LastError,
	)
	if err != nil {
		core.Log("no se pudo actualizar el comprobante", row.ID, err)
	}
}

// The artifacts a document must keep for five years, named by convention rather
// than by a stored path. Both are derived from the document id, which is unique
// per sale and series, so a sale's boleta and its credit note never collide.
const (
	xmlArtifactName = "xml"
	cdrArtifactName = "cdr"
)

// ArtifactFolder is where a company's documents live. Kept next to the naming so
// the read path and the write path cannot drift.
func ArtifactFolder(companyID int32) string {
	return fmt.Sprintf("cpe/%v", companyID)
}

// ArtifactName is the object a document's XML or CDR is stored under.
func ArtifactName(documentID int64, kind string) string {
	if kind == cdrArtifactName {
		return fmt.Sprintf("%v-cdr.zip", documentID)
	}
	return fmt.Sprintf("%v.xml", documentID)
}

// storeArtifact keeps the XML and the CDR. Both have to survive five years, so
// they go to object storage; the row keeps no path because the name is derived.
func storeArtifact(companyID int32, row *types.InvoiceDocument,
	kind string, content []byte, contentType string) error {

	err := cloud.SaveFile(cloud.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        ArtifactFolder(companyID),
		Name:        ArtifactName(row.ID, kind),
		FileContent: content,
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("no se pudo guardar el archivo del comprobante: %w", err)
	}
	return nil
}
