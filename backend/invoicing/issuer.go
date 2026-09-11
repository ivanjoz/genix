package invoicing

import (
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
// The identity, the credentials, the signing material and the fiscal address all
// come from the company config blob, which is one GET instead of a company read, a
// secrets read, a city read and three decryptions. The blob rebuilds itself when it
// is missing or stale, so this path does not need a fallback of its own.
//
// It takes no site: the address is the company's own, under establishment code
// 0000 — see fiscalAddress. A branch is declared by the annex code SUNAT assigned
// to it, which nothing stores yet.
func BuildIssuer(companyID int32) (model.Issuer, error) {
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

	address, err := fiscalAddress(companyConfig.Company)
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

// fiscalAddress is the domicilio fiscal every document declares for the issuer.
//
// It is the company's own address and not the branch's, because the establishment
// code that travels with it is 0000 — SUNAT's code for the main establishment, the
// fiscal address itself. Declaring a branch's street under 0000 is what produced
// the observations 4093 and 4096-4098 on the first documents this system emitted.
// A branch has to be declared under the annex code SUNAT assigned to it, and that
// is a column the Site table does not have yet.
//
// Everything it needs was resolved when the config blob was built, so this reads
// nothing.
func fiscalAddress(company config.CompanyConfigCompany) (model.Address, error) {
	ubigeo := company.Ubigeo()
	if ubigeo == "" {
		return model.Address{}, errors.New(
			"la empresa no tiene ciudad configurada: selecciónela en Mi Empresa para poder emitir")
	}
	// The names come from the same catalog row the ubigeo does, so a missing one
	// means the blob was built against a catalog that no longer has that district.
	if company.District == "" || company.Province == "" || company.Department == "" {
		return model.Address{}, fmt.Errorf(
			"el ubigeo %v de la empresa no corresponde a un distrito del catálogo", ubigeo)
	}

	return model.Address{
		Ubigeo:     ubigeo,
		AnnexCode:  defaultAnnexCode,
		Department: company.Department,
		Province:   company.Province,
		District:   company.District,
		Line:       company.Address,
	}, nil
}

// defaultAnnexCode is the establishment code of a main office. A company with
// annexes registered at SUNAT declares theirs per site, which is a column the
// Site table does not have yet.
const defaultAnnexCode = "0000"
