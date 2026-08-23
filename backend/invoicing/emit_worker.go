package invoicing

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/invoicing/types"
	"errors"
	"fmt"

	"github.com/ivanjoz/facturago"
	"github.com/ivanjoz/facturago/sunat"
)

// Sending happens outside the request that asked for it.
//
// SUNAT regularly takes tens of seconds and the API gateway gives up at thirty,
// so an emission cannot finish inline. It runs as an asynchronous invoke, which
// starts immediately, and the cron action is the net underneath: it survives a
// panic, caps at ten attempts, and leaves a row somebody can read afterwards.
//
// Only a transport or configuration failure is retried. A rejection is final —
// the same XML will be refused forever, and re-sending it just fills a queue.
const (
	// EmitFunctionName is the async entry point, registered in exec.
	EmitFunctionName = "fn-emit-cpe"
	// emitRetryActionID is this module's cron action. Values 1 to 4 are taken.
	emitRetryActionID = int16(5)
	// retryFrameMinutes is the first retry delay; the executor keeps the cadence.
	retryFrameMinutes = int8(5)
	// maxRetries stops a document that keeps failing from being attempted forever.
	maxRetries = int8(6)
)

// SendDocumentAsync starts the transmission without waiting for it.
func SendDocumentAsync(companyID int32, documentID int64) {
	cloud.ExecLambda(core.ExecArgs{
		LambdaName:    core.Env.LAMBDA_NAME,
		FuncToExec:    EmitFunctionName,
		InvokeAsEvent: true,
		Param1:        int64(companyID),
		Param2:        documentID,
	})
}

// EmitHandler is the async entry point and the body of the retry. Both paths are
// the same work, which is why they are the same function.
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

	issuer, err := BuildIssuer(companyID, siteOfDocument(companyID, row))
	if err != nil {
		return recordFailure(row, err, false)
	}

	// Signing regenerates the XML from the row rather than reusing a stored
	// one, so a document that never got past this point is reproduced exactly.
	document := DocumentFromRow(row)
	emission, err := facturago.Emit(issuer, document)
	if err != nil {
		// A build or validation failure is about the document, not the network.
		return recordFailure(row, err, false)
	}

	xmlPath, err := storeArtifact(companyID, row, emission.FileName+".xml", emission.XML, "application/xml")
	if err != nil {
		return recordFailure(row, err, true)
	}
	row.XmlPath = xmlPath
	row.DigestValue = emission.Digest
	row.State = types.InvoiceQueued
	saveDocument(row)

	cdr, err := new(sunat.Client).SendBill(issuer, emission)
	if err != nil {
		return recordFailure(row, err, sunat.Retryable(err))
	}

	if len(cdr.Zip) > 0 {
		cdrPath, storeErr := storeArtifact(companyID, row,
			facturago.CDRName(emission.FileName)+".zip", cdr.Zip, "application/zip")
		if storeErr != nil {
			core.Log("no se pudo guardar la CDR del comprobante", row.ID, storeErr)
		} else {
			row.CdrPath = cdrPath
		}
	}

	row.SunatCode = cdr.Code
	row.SunatDescription = cdr.Description
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
		row.SunatDescription = sunatError.Description
		row.State = stateFromSeverity(sunatError.Severity)
	} else if retryable {
		row.State = types.InvoiceException
	} else {
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
		table.SunatCode, table.SunatDescription, table.SunatNotes, table.Ticket,
		table.DigestValue, table.XmlPath, table.CdrPath,
		table.RetryCount, table.LastError,
	)
	if err != nil {
		core.Log("no se pudo actualizar el comprobante", row.ID, err)
	}
}

// storeArtifact keeps the XML and the CDR. Both have to survive five years, so
// they go to object storage and the row keeps the path.
func storeArtifact(companyID int32, row *types.InvoiceDocument,
	name string, content []byte, contentType string) (string, error) {

	path := fmt.Sprintf("cpe/%v/%v", companyID, row.IssueDate/30)
	err := cloud.SaveFile(cloud.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        path,
		Name:        name,
		FileContent: content,
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("no se pudo guardar %v: %w", name, err)
	}
	return path + "/" + name, nil
}

// siteOfDocument finds the establishment a document is issued from, which is the
// site its series belongs to.
func siteOfDocument(companyID int32, row *types.InvoiceDocument) int32 {
	series := []types.InvoiceSeries{}
	query := db.Query(&series)
	query.Select().CompanyID.Equals(companyID).
		ID.Equals(types.PackDocTypeSeries(row.DocType, row.SeriesID))

	if err := query.Exec(); err != nil || len(series) == 0 {
		return 0
	}
	return series[0].SiteID
}
