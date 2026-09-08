package invoicing

import (
	business "app/business/types"
	config "app/config/types"
	"app/core"
	"app/db"
	"app/invoicing/types"
	"errors"
	"fmt"

	"github.com/ivanjoz/facturago/model"
)

// BuildIssuer assembles the taxpayer that signs and transmits one document.
//
// It is rebuilt on every emission and never cached. A warm Lambda serves many
// companies, so an issuer held in a package variable is a document signed with
// somebody else's certificate — the worst failure this module could have, and
// one that would only be discovered by whoever received the invoice.
//
// The fiscal address comes from the site the sale was made at, because that is
// what SUNAT wants declared: a company with three branches issues from three
// addresses, each with the establishment code SUNAT assigned to it.
func BuildIssuer(companyID int32, siteID int32) (model.Issuer, error) {
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
		return model.Issuer{}, errors.New("no se pudo descifrar la clave SOL de la empresa")
	}
	certificate, err := core.Decrypt(secrets.CertificateEnc)
	if err != nil {
		return model.Issuer{}, errors.New("no se pudo descifrar el certificado de la empresa")
	}
	certPassword, err := core.Decrypt(secrets.CertPasswordEnc)
	if err != nil {
		return model.Issuer{}, errors.New("no se pudo descifrar la clave del certificado")
	}

	environment := model.EnvBeta
	if secrets.Environment == types.SunatEnvProduction {
		environment = model.EnvProduction
	}

	issuer := model.Issuer{
		RUC:            company.RUC,
		LegalName:      company.LegalName,
		TradeName:      company.Name,
		SolUser:        secrets.SolUser,
		SolPassword:    string(solPassword),
		PKCS12:         certificate,
		PKCS12Password: string(certPassword),
		Environment:    environment,
	}
	if issuer.LegalName == "" {
		issuer.LegalName = company.Name
	}

	address, err := loadSiteAddress(companyID, siteID)
	if err != nil {
		return model.Issuer{}, err
	}
	issuer.Address = address

	// The certificate has to belong to the company issuing. SUNAT rejects the
	// mismatch, and catching it here names the real problem instead of leaving
	// a code to look up.
	if secrets.CertRUC != "" && secrets.CertRUC != company.RUC {
		return model.Issuer{}, fmt.Errorf(
			"el certificado pertenece al RUC %v y la empresa es %v", secrets.CertRUC, company.RUC)
	}
	return issuer, nil
}

// LoadActiveSecrets returns the company's current SUNAT credentials.
//
// Several rows can exist — replacing an expiring certificate leaves the old one
// behind for the documents it signed — so the active one is the newest with
// Status 1.
func LoadActiveSecrets(companyID int32) (*types.CompanySecrets, error) {
	secrets := []types.CompanySecrets{}
	query := db.Query(&secrets)
	query.Select().CompanyID.Equals(companyID).Type.Equals(types.SecretTypeSunatCPE)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las credenciales SUNAT: %w", err)
	}

	var active *types.CompanySecrets
	for index := range secrets {
		candidate := &secrets[index]
		if candidate.Status != 1 {
			continue
		}
		if active == nil || candidate.ID > active.ID {
			active = candidate
		}
	}
	if active == nil {
		return nil, errors.New("la empresa no tiene credenciales SUNAT configuradas")
	}
	if len(active.CertificateEnc) == 0 {
		return nil, errors.New("la empresa no tiene un certificado digital cargado")
	}
	return active, nil
}

// loadCompany reads the issuer's own company. The RUC check lives here rather
// than in loadCompanyRecord because it is an issuing requirement: reading a
// company's series is legitimate before anyone has typed a RUC.
func loadCompany(companyID int32) (*config.Company, error) {
	company, err := loadCompanyRecord(companyID)
	if err != nil {
		return nil, err
	}
	if len(company.RUC) != 11 {
		return nil, errors.New("la empresa no tiene un RUC válido de 11 dígitos")
	}
	return company, nil
}

// loadSiteAddress turns a site into the fiscal address the XML declares.
//
// CityID is the six-digit INEI code, which is exactly the ubigeo SUNAT expects,
// and its prefixes are the province and the department — the same decomposition
// the rest of the codebase already does.
func loadSiteAddress(companyID int32, siteID int32) (model.Address, error) {
	if siteID == 0 {
		return model.Address{}, errors.New("no se pudo determinar la sede que emite el comprobante")
	}

	sites := []business.Site{}
	query := db.Query(&sites)
	query.Select().CompanyID.Equals(companyID).ID.Equals(siteID)

	if err := query.Exec(); err != nil {
		return model.Address{}, fmt.Errorf("error al leer la sede: %w", err)
	}
	if len(sites) == 0 {
		return model.Address{}, errors.New("la sede que emite el comprobante no existe")
	}
	site := sites[0]

	address := model.Address{
		Line:      site.Address,
		AnnexCode: defaultAnnexCode,
	}
	if site.CityID > 0 {
		address.Ubigeo = fmt.Sprintf("%06d", site.CityID)
	}
	return address, nil
}

// defaultAnnexCode is the establishment code of a main office. A company with
// annexes registered at SUNAT declares theirs per site, which is a column the
// Site table does not have yet.
const defaultAnnexCode = "0000"
