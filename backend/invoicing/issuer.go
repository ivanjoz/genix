package invoicing

import (
	business "app/business/types"
	"app/cloud"
	config "app/config/types"
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
// The identity, the credentials and the signing material all come from the
// company config blob, which is one GET instead of a company read, a secrets read
// and three decryptions. The blob rebuilds itself when it is missing or stale, so
// this path does not need a fallback of its own.
func BuildIssuer(companyID int32, siteID int32) (model.Issuer, error) {
	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return model.Issuer{}, err
	}
	if len(companyConfig.Company.RUC) != 11 {
		return model.Issuer{}, errors.New("la empresa no tiene un RUC válido de 11 dígitos")
	}
	sunat := companyConfig.Sunat
	if len(sunat.CertificateDER) == 0 {
		return model.Issuer{}, errors.New("la empresa no tiene un certificado digital cargado")
	}

	environment := model.EnvBeta
	if sunat.Environment == types.SunatEnvProduction {
		environment = model.EnvProduction
	}

	// The key and the certificate travel unwrapped: the .pfx container around them
	// is transport the signer discards, so the blob never carried it.
	issuer := model.Issuer{
		RUC:            companyConfig.Company.RUC,
		LegalName:      companyConfig.Company.LegalName,
		TradeName:      companyConfig.Company.TradeName,
		SolUser:        sunat.SolUser,
		SolPassword:    sunat.SolPassword,
		PrivateKeyDER:  sunat.PrivateKeyDER,
		CertificateDER: sunat.CertificateDER,
		Environment:    environment,
	}

	address, err := loadSiteAddress(companyID, siteID)
	if err != nil {
		return model.Issuer{}, err
	}
	issuer.Address = address

	// The certificate has to belong to the company issuing. SUNAT rejects the
	// mismatch, and catching it here names the real problem instead of leaving
	// a code to look up.
	if sunat.CertRUC != "" && sunat.CertRUC != issuer.RUC {
		return model.Issuer{}, fmt.Errorf(
			"el certificado pertenece al RUC %v y la empresa es %v", sunat.CertRUC, issuer.RUC)
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
