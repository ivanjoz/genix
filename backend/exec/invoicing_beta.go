package exec

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/invoicing"
	invoicingTypes "app/invoicing/types"
	sales "app/sales/types"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ivanjoz/facturago"
	"github.com/ivanjoz/facturago/sunat"
)

// Command-line access to electronic invoicing, for the two things that have no
// interface yet: configuring a company against SUNAT's test environment, and
// issuing a document for a sale.
//
// Both are thin wrappers over the invoicing package — they exist so the path can
// be exercised end to end before the frontend does, and so support can reissue a
// document without a browser.

// betaCertificatePath is the self-signed certificate facturago tests with. It is
// valid in SUNAT's beta environment and worthless anywhere else, which is
// exactly what a setup command for beta should install.
const (
	betaCertificatePath = "facturago/testdata/cert-aes256.pfx"
	betaCertPassword    = "demo123"
	betaSolUser         = "MODDATOS"
	betaSolPassword     = "MODDATOS"
	// betaRUC is the taxpayer SUNAT's test service answers for, and the one the
	// test certificate is issued to. A company issuing against beta has to be it.
	betaRUC = "20000000001"
)

// SetupSunatBeta configures a company to issue against SUNAT's test service.
//
//	go run . fn-sunat-beta-setup <company-id> [site-id]
//
// It writes the credentials encrypted, exactly as the upload form would, and
// creates a factura and a boleta series if the company has none.
func SetupSunatBeta(args *core.ExecArgs) core.FuncResponse {
	numbers := execNumbers(args, 2)
	companyID := int32(numbers[0])
	if companyID == 0 {
		return args.MakeErr("indique la empresa: fn-sunat-beta-setup <company-id> [site-id]")
	}
	siteID := int32(numbers[1])
	if siteID == 0 {
		siteID = 1
	}

	certificate, err := os.ReadFile(betaCertificatePath)
	if err != nil {
		return args.MakeErr(fmt.Sprintf("no se pudo leer el certificado de pruebas %v: %v",
			betaCertificatePath, err))
	}

	if err := storeBetaCredentials(companyID, certificate); err != nil {
		return args.MakeErr(err.Error())
	}
	created, err := ensureBetaSeries(companyID, siteID)
	if err != nil {
		return args.MakeErr(err.Error())
	}

	// This path writes the rows directly instead of going through the endpoints, so
	// it has to refresh the config blob itself: an existing blob would keep serving
	// the credentials that were just replaced.
	companyConfig, err := cloud.CompactAndStoreCompanyConfig(companyID)
	if err != nil {
		return args.MakeErr(fmt.Sprintf("credenciales guardadas, pero no se pudo actualizar la configuración: %v", err))
	}

	args.AddMessage(fmt.Sprintf(
		"empresa %v configurada contra SUNAT beta (usuario %v, %v series creadas)",
		companyID, betaSolUser, created))

	// SUNAT beta only answers for its own taxpayer, and BuildIssuer refuses a
	// certificate whose RUC is not the company's. Saying so here beats discovering
	// it as a rejected emission.
	if companyConfig.Company.RUC != betaRUC {
		args.AddMessage(fmt.Sprintf(
			"ATENCIÓN: el RUC de la empresa es %q y el certificado de pruebas es de %v. "+
				"Cambie el RUC de la empresa a %v para poder emitir contra beta.",
			companyConfig.Company.RUC, betaRUC, betaRUC))
	}
	return core.FuncResponse{}
}

// storeBetaCredentials writes the SOL credentials and the certificate, encrypted.
func storeBetaCredentials(companyID int32, certificate []byte) error {
	existing, err := invoicing.LoadActiveSecrets(companyID)
	if err == nil && existing != nil {
		core.Log("la empresa ya tenía credenciales; se reemplazan por las de beta")
	}

	solPassword, err := core.Encrypt([]byte(betaSolPassword))
	if err != nil {
		return fmt.Errorf("no se pudo cifrar la clave SOL: %w", err)
	}
	certificateEnc, err := core.Encrypt(certificate)
	if err != nil {
		return fmt.Errorf("no se pudo cifrar el certificado: %w", err)
	}
	certPassword, err := core.Encrypt([]byte(betaCertPassword))
	if err != nil {
		return fmt.Errorf("no se pudo cifrar la clave del certificado: %w", err)
	}

	now := core.SUnixTime()
	record := invoicingTypes.CompanySecrets{
		CompanyID:       companyID,
		Type:            invoicingTypes.SecretTypeSunatCPE,
		Name:            "SUNAT beta",
		SolUser:         betaSolUser,
		SolPasswordEnc:  solPassword,
		CertificateEnc:  certificateEnc,
		CertPasswordEnc: certPassword,
		Environment:     invoicingTypes.SunatEnvBeta,
		Status:          1,
		Created:         now,
		Updated:         now,
	}
	if existing != nil {
		record.ID = existing.ID
	} else {
		record.ID = -1
	}

	// The metadata the form would have read off the certificate.
	if credential, err := facturago.ParsePKCS12(certificate, betaCertPassword); err == nil {
		record.CertSubject = credential.Subject
		record.CertIssuer = credential.Issuer
		record.CertSerial = credential.SerialNumber
		record.CertRUC = credential.RUC
		record.CertValidFrom = core.UnixToSunix(credential.NotBefore.Unix())
		record.CertValidTo = core.UnixToSunix(credential.NotAfter.Unix())
	}

	records := &[]invoicingTypes.CompanySecrets{record}
	if existing != nil {
		return db.Update(records)
	}
	return db.Insert(records)
}

// ensureBetaSeries gives the company one series per document type if it has none.
func ensureBetaSeries(companyID, siteID int32) (int, error) {
	allSeries, err := invoicing.LoadCompanySeries(companyID)
	if err != nil {
		return 0, err
	}

	wanted := []struct {
		docType int8
		code    string
	}{
		{invoicingTypes.DocTypeFactura, "F001"},
		{invoicingTypes.DocTypeBoleta, "B001"},
	}

	created := 0
	for _, want := range wanted {
		found := false
		for index := range allSeries {
			if allSeries[index].DocType == want.docType {
				found = true
				break
			}
		}
		if found {
			continue
		}
		allSeries = append(allSeries, invoicingTypes.InvoiceSeries{
			// Ids are unique across the whole set now, not per document type: the id
			// is what partitions the correlativo counter.
			SeriesID:   invoicingTypes.NextSeriesID(allSeries),
			DocType:    want.docType,
			SeriesCode: want.code,
			SiteID:     siteID,
			IsDefault:  1,
			Status:     1,
		})
		created++
	}
	if created == 0 {
		return 0, nil
	}
	if err := invoicing.SaveCompanySeries(companyID, allSeries); err != nil {
		return 0, err
	}
	return created, nil
}

// EmitInvoice issues an electronic document for a sale, from the command line.
//
//	go run . fn-emit-invoice <company-id> <sale-order-id> [doc-type]
//
// doc-type is the SUNAT code: 1 for a factura, 3 for a boleta. It runs the same
// two steps the handler does — reserve, then send — but waits for SUNAT instead
// of returning early, because a command line has nobody to notify later.
func EmitInvoice(args *core.ExecArgs) core.FuncResponse {
	numbers := execNumbers(args, 3)
	companyID, saleOrderID := int32(numbers[0]), numbers[1]
	if companyID == 0 || saleOrderID == 0 {
		return args.MakeErr("indique la empresa y la venta: fn-emit-invoice <company-id> <sale-order-id> [doc-type]")
	}
	docType := int8(numbers[2])
	if docType == 0 {
		docType = invoicingTypes.DocTypeBoleta
	}

	order, err := loadOrderForEmission(companyID, saleOrderID)
	if err != nil {
		return args.MakeErr(err.Error())
	}
	series, err := findSeries(companyID, docType)
	if err != nil {
		return args.MakeErr(err.Error())
	}

	// A sale created after the deferred-emission change already carries its
	// document. This command is what issues one for a sale that predates it, or
	// re-issues after the first attempt was discarded.
	existing, err := invoicingTypes.FindBySaleOrder(companyID, saleOrderID)
	if err != nil {
		return args.MakeErr(err.Error())
	}
	if existing != nil {
		return args.MakeErr(fmt.Sprintf("la venta ya tiene el comprobante %v",
			existing.Number(series.SeriesCode)))
	}

	document, err := invoicingTypes.PrepareDocumentForSale(companyID, 1, order, series)
	if err != nil {
		return args.MakeErr(fmt.Sprintf("no se pudo reservar el correlativo: %v", err))
	}
	if err := invoicingTypes.SaveNewDocument(document); err != nil {
		return args.MakeErr(err.Error())
	}
	args.AddMessage(fmt.Sprintf("reservado %v (id %v)",
		document.Number(series.SeriesCode), document.ID))

	if err := invoicing.SendDocument(companyID, document.ID); err != nil {
		return args.MakeErr(fmt.Sprintf("SUNAT: %v", err))
	}

	issued, err := invoicing.LoadDocument(companyID, document.ID)
	if err != nil {
		return args.MakeErr(err.Error())
	}
	args.AddMessage(fmt.Sprintf("SUNAT respondió %v: %v",
		issued.SunatCode, sunat.ErrorDescription(issued.SunatCode)))
	args.AddMessage(fmt.Sprintf("estado %v | artefactos en %v/%v",
		issued.State, invoicing.ArtifactFolder(companyID),
		invoicing.ArtifactName(issued.ID, "xml")))

	return core.FuncResponse{}
}

// execNumbers reads the numeric arguments of a command.
//
// A command line arrives joined in Message, while a programmatic invoke fills
// Param1 and up. Both are accepted so the same function serves a person typing
// and a Lambda calling.
func execNumbers(args *core.ExecArgs, count int) []int64 {
	numbers := make([]int64, count)
	for index, field := range strings.Fields(args.Message) {
		if index >= count {
			break
		}
		value, err := strconv.ParseInt(field, 10, 64)
		if err == nil {
			numbers[index] = value
		}
	}

	params := []int64{args.Param1, args.Param2, args.Param3}
	for index := range numbers {
		if numbers[index] == 0 && index < len(params) {
			numbers[index] = params[index]
		}
	}
	return numbers
}

// SendInvoice transmits a document that was already numbered.
//
//	go run . fn-send-invoice <company-id> <document-id>
//
// This is the retry path by hand: the same work the cron action does, for when
// support needs to push one document rather than wait for the schedule.
func SendInvoice(args *core.ExecArgs) core.FuncResponse {
	numbers := execNumbers(args, 2)
	companyID, documentID := int32(numbers[0]), numbers[1]
	if companyID == 0 || documentID == 0 {
		return args.MakeErr("indique la empresa y el comprobante: fn-send-invoice <company-id> <document-id>")
	}

	if err := invoicing.SendDocument(companyID, documentID); err != nil {
		return args.MakeErr(fmt.Sprintf("SUNAT: %v", err))
	}

	issued, err := invoicing.LoadDocument(companyID, documentID)
	if err != nil {
		return args.MakeErr(err.Error())
	}
	args.AddMessage(fmt.Sprintf("serie %v correlativo %v | SUNAT %v: %v | estado %v",
		issued.SeriesID(), issued.Correlativo, issued.SunatCode,
		sunat.ErrorDescription(issued.SunatCode), issued.State))
	return core.FuncResponse{}
}

func loadOrderForEmission(companyID int32, saleOrderID int64) (*sales.SaleOrder, error) {
	orders := []sales.SaleOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(companyID).ID.Equals(saleOrderID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer la venta: %w", err)
	}
	if len(orders) == 0 {
		return nil, errors.New("la venta no existe")
	}
	return &orders[0], nil
}

func findSeries(companyID int32, docType int8) (*invoicingTypes.InvoiceSeries, error) {
	allSeries, err := invoicing.LoadCompanySeries(companyID)
	if err != nil {
		return nil, err
	}
	return invoicingTypes.ResolveSeries(allSeries, docType, 0)
}
