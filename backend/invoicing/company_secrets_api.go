package invoicing

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/invoicing/types"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/ivanjoz/facturago"
	"github.com/ivanjoz/facturago/model"
	"github.com/ivanjoz/facturago/sunat"
)

// PostCompanySecretsBody is the credential upload. The certificate arrives as
// base64 because it is a binary file coming through a JSON form.
type PostCompanySecretsBody struct {
	ID             int32  `json:"ID"`
	Name           string `json:"Name"`
	SolUser        string `json:"SolUser"`
	SolPassword    string `json:"SolPassword"`
	CertificateB64 string `json:"Certificate"`
	CertPassword   string `json:"CertPassword"`
	Environment    int8   `json:"Environment"`
}

// CompanySecretsView is what a caller may see: everything except the secrets.
//
// No endpoint returns key material — not the certificate, not either password,
// not even to the company that owns them. What a form needs is whose certificate
// it is and when it stops working, which is all here.
type CompanySecretsView struct {
	ID            int32  `json:"ID"`
	Name          string `json:"Name"`
	SolUser       string `json:"SolUser"`
	Environment   int8   `json:"Environment"`
	CertSubject   string `json:"CertSubject"`
	CertIssuer    string `json:"CertIssuer"`
	CertRUC       string `json:"CertRUC"`
	CertValidFrom int32  `json:"CertValidFrom"`
	CertValidTo   int32  `json:"CertValidTo"`
	HasCert       bool   `json:"HasCert"`
	Status        int8   `json:"ss"`
	Updated       int32  `json:"upd"`
}

// GetCompanySecrets lists the company's SUNAT credentials, without the secrets.
func GetCompanySecrets(req *core.HandlerArgs) core.HandlerResponse {
	secrets := []types.CompanySecrets{}
	query := db.Query(&secrets)
	query.Select().CompanyID.Equals(req.User.CompanyID)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las credenciales:", err)
	}

	views := make([]CompanySecretsView, 0, len(secrets))
	for index := range secrets {
		record := &secrets[index]
		views = append(views, CompanySecretsView{
			ID: record.ID, Name: record.Name, SolUser: record.SolUser,
			Environment: record.Environment,
			CertSubject: record.CertSubject, CertIssuer: record.CertIssuer,
			CertRUC:       record.CertRUC,
			CertValidFrom: record.CertValidFrom, CertValidTo: record.CertValidTo,
			HasCert: len(record.CertificateEnc) > 0,
			Status:  record.Status, Updated: record.Updated,
		})
	}
	return core.MakeResponse(req, &map[string]any{"CompanySecrets": views})
}

// PostCompanySecrets stores or replaces a company's SUNAT credentials.
//
// The certificate is opened here, before anything is saved. That turns three
// failures that would otherwise appear as a SUNAT rejection days later — wrong
// password, expired certificate, certificate belonging to another taxpayer —
// into an error on the form that uploaded it.
func PostCompanySecrets(req *core.HandlerArgs) core.HandlerResponse {
	body := PostCompanySecretsBody{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	if body.SolUser == "" {
		return req.MakeErr("Debe indicar el usuario SOL.")
	}

	companyID := req.User.CompanyID
	now := core.SUnixTime()

	// The row the form is editing, when it is editing one. It is read before anything
	// is written, because a rotation both inherits the SOL password the form did not
	// ask again for and has to retire this row afterwards.
	var previous *types.CompanySecrets
	if body.ID > 0 {
		found, err := loadSecretsByID(companyID, body.ID)
		if err != nil {
			return req.MakeErr(err.Error())
		}
		previous = found
	}

	record := types.CompanySecrets{
		CompanyID:   companyID,
		ID:          body.ID,
		Type:        types.SecretTypeSunatCPE,
		Name:        body.Name,
		SolUser:     body.SolUser,
		Environment: body.Environment,
		Status:      1,
		Updated:     now,
		UpdatedBy:   req.User.ID,
	}
	if record.Environment == 0 {
		record.Environment = types.SunatEnvBeta
	}

	if body.SolPassword != "" {
		encrypted, err := core.Encrypt([]byte(body.SolPassword))
		if err != nil {
			return req.MakeErr("No se pudo cifrar la clave SOL.")
		}
		record.SolPasswordEnc = encrypted
	}

	if body.CertificateB64 != "" {
		certificate, err := base64.StdEncoding.DecodeString(body.CertificateB64)
		if err != nil {
			return req.MakeErr("El certificado enviado no es un base64 válido.")
		}

		credential, err := facturago.ParsePKCS12(certificate, body.CertPassword)
		if err != nil {
			return req.MakeErr("No se pudo abrir el certificado: " + err.Error())
		}
		if credential.Expired(core.Now()) {
			return req.MakeErr("El certificado está vencido o todavía no es válido.")
		}
		if err := checkCertificateRUC(companyID, credential.RUC); err != nil {
			return req.MakeErr(err.Error())
		}

		encryptedCert, err := core.Encrypt(certificate)
		if err != nil {
			return req.MakeErr("No se pudo cifrar el certificado.")
		}
		encryptedPassword, err := core.Encrypt([]byte(body.CertPassword))
		if err != nil {
			return req.MakeErr("No se pudo cifrar la clave del certificado.")
		}

		record.CertificateEnc = encryptedCert
		record.CertPasswordEnc = encryptedPassword
		record.CertSubject = credential.Subject
		record.CertIssuer = credential.Issuer
		record.CertSerial = credential.SerialNumber
		record.CertRUC = credential.RUC
		record.CertValidFrom = core.UnixToSunix(credential.NotBefore.Unix())
		record.CertValidTo = core.UnixToSunix(credential.NotAfter.Unix())
	}

	// A new certificate never overwrites the one in place: it goes into a new row and
	// the previous one is retired, so a document signed six months ago keeps the
	// certificate that signed it. Everything else edits the row it came from.
	isRotation := previous != nil && len(record.CertificateEnc) > 0
	isNew := previous == nil

	if isRotation && len(record.SolPasswordEnc) == 0 {
		record.SolPasswordEnc = previous.SolPasswordEnc
	}
	// Without it the row cannot authenticate, and the failure would only surface when
	// SUNAT rejects the first real sale.
	if isNew && len(record.SolPasswordEnc) == 0 {
		return req.MakeErr("Debe indicar la clave SOL.")
	}

	table := db.TableOf[types.CompanySecrets]()
	records := &[]types.CompanySecrets{record}
	var err error

	if isNew || isRotation {
		record.ID = -1
		record.Created = now
		record.CreatedBy = req.User.ID
		(*records)[0] = record
		err = db.Insert(records)
	} else {
		// A save that does not upload a certificate must not blank the one this row
		// already holds, so those columns are excluded rather than written as empty.
		excluded := []db.Coln{table.Created, table.CreatedBy,
			table.CertificateEnc, table.CertPasswordEnc, table.CertSubject, table.CertIssuer,
			table.CertSerial, table.CertRUC, table.CertValidFrom, table.CertValidTo}
		if len(record.SolPasswordEnc) == 0 {
			excluded = append(excluded, table.SolPasswordEnc)
		}
		err = db.UpdateExclude(records, excluded...)
	}
	if err != nil {
		return req.MakeErr("Error al guardar las credenciales:", err)
	}

	// Retired after the new row is in, never before: if this write fails, the company
	// has two active rows and emission takes the newest, which is the right one. The
	// other order would leave it with none.
	if isRotation {
		previous.Status = 0
		previous.Updated = now
		previous.UpdatedBy = req.User.ID
		retired := &[]types.CompanySecrets{*previous}
		if err := db.Update(retired, table.Status, table.Updated, table.UpdatedBy); err != nil {
			return req.MakeErr("El certificado se guardó, pero no se pudo retirar el anterior:", err)
		}
	}

	saved := (*records)[0]
	cloud.StoreCompanyConfigAsync(req.User.CompanyID)
	return req.MakeResponse(CompanySecretsView{
		ID: saved.ID, Name: saved.Name, SolUser: saved.SolUser,
		Environment: saved.Environment, CertSubject: saved.CertSubject,
		CertIssuer: saved.CertIssuer, CertRUC: saved.CertRUC,
		CertValidFrom: saved.CertValidFrom, CertValidTo: saved.CertValidTo,
		HasCert: len(saved.CertificateEnc) > 0, Status: saved.Status, Updated: saved.Updated,
	})
}

// PostCompanySecretsTest checks the credentials against SUNAT without issuing.
//
// The check itself is facturago's VerifyCredentials, which sends a file that cannot
// be accepted and reads whether SUNAT refused the credentials or the file. It used
// to ask for the CDR of a document that does not exist, which only works in
// production: beta does not host the CDR lookup service at all.
func PostCompanySecretsTest(req *core.HandlerArgs) core.HandlerResponse {
	issuer, err := buildIssuerForTest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	if err := new(sunat.Client).VerifyCredentials(issuer); err != nil {
		var authError *sunat.AuthError
		if errors.As(err, &authError) {
			return req.MakeErr("SUNAT rechazó las credenciales: revise el usuario y la clave SOL.")
		}
		return req.MakeErr("No se pudo contactar a SUNAT: " + err.Error())
	}

	return req.MakeResponse(map[string]any{
		"Ok":      true,
		"Message": "SUNAT aceptó las credenciales.",
	})
}

// buildIssuerForTest assembles just enough of an issuer to authenticate: the
// test never builds a document, so it needs no address.
func buildIssuerForTest(companyID int32) (model.Issuer, error) {
	company, err := loadCompany(companyID)
	if err != nil {
		return model.Issuer{}, err
	}
	secrets, err := LoadActiveSecrets(companyID)
	if err != nil {
		return model.Issuer{}, err
	}
	solPassword, err := core.Decrypt(secrets.SolPasswordEnc)
	if err != nil {
		return model.Issuer{}, err
	}

	environment := model.EnvBeta
	if secrets.Environment == types.SunatEnvProduction {
		environment = model.EnvProduction
	}
	return model.Issuer{
		RUC:         company.RUC,
		LegalName:   company.LegalName,
		SolUser:     secrets.SolUser,
		SolPassword: string(solPassword),
		Environment: environment,
	}, nil
}

// loadSecretsByID reads one credential row of this company. The company id is part
// of the lookup, not checked afterwards, so an id from another tenant finds nothing.
func loadSecretsByID(companyID int32, secretsID int32) (*types.CompanySecrets, error) {
	records := []types.CompanySecrets{}
	query := db.Query(&records)
	query.Select().CompanyID.Equals(companyID).ID.Equals(secretsID)

	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al leer las credenciales:", err)
	}
	if len(records) == 0 {
		return nil, core.Err("Las credenciales indicadas no existen.")
	}
	return &records[0], nil
}

// checkCertificateRUC refuses a certificate issued to another taxpayer. SUNAT
// would reject every document signed with it, with a code that does not say so.
func checkCertificateRUC(companyID int32, certificateRUC string) error {
	if certificateRUC == "" {
		return nil // not every certificate carries one
	}
	company, err := loadCompany(companyID)
	if err != nil {
		return err
	}
	if company.RUC != certificateRUC {
		return core.Err("El certificado pertenece al RUC ", certificateRUC,
			" y la empresa tiene el RUC ", company.RUC)
	}
	return nil
}
