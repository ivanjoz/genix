package invoicing

import (
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

	isNew := body.ID <= 0
	if isNew {
		record.ID = -1
		record.Created = now
		record.CreatedBy = req.User.ID
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

	records := &[]types.CompanySecrets{record}
	var err error
	if isNew {
		err = db.Insert(records)
	} else {
		// A save that does not re-upload the certificate must not blank it, so
		// the untouched columns are excluded rather than written as empty.
		table := db.TableOf[types.CompanySecrets]()
		excluded := []db.Coln{table.Created, table.CreatedBy}
		if len(record.SolPasswordEnc) == 0 {
			excluded = append(excluded, table.SolPasswordEnc)
		}
		if len(record.CertificateEnc) == 0 {
			excluded = append(excluded, table.CertificateEnc, table.CertPasswordEnc,
				table.CertSubject, table.CertIssuer, table.CertSerial, table.CertRUC,
				table.CertValidFrom, table.CertValidTo)
		}
		err = db.UpdateExclude(records, excluded...)
	}
	if err != nil {
		return req.MakeErr("Error al guardar las credenciales:", err)
	}

	saved := (*records)[0]
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
// It asks for the CDR of a document that does not exist. SUNAT answers that it
// has no record of it, which is a successful round trip: the credentials were
// accepted. Anything else is the real problem, reported before a real sale hits
// it.
func PostCompanySecretsTest(req *core.HandlerArgs) core.HandlerResponse {
	issuer, err := buildIssuerForTest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	status, err := new(sunat.Client).StatusOf(issuer, model.Factura, "F001", 1)
	if err != nil {
		var sunatError *sunat.Error
		if errors.As(err, &sunatError) {
			return req.MakeErr("SUNAT rechazó las credenciales: " + sunatError.Error())
		}
		return req.MakeErr("No se pudo contactar a SUNAT: " + err.Error())
	}

	return req.MakeResponse(map[string]any{
		"Ok":      true,
		"Code":    status.Code,
		"Message": sunat.ErrorDescription(status.Code),
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
