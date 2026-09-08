// The company configuration blob: everything an API needs to know about a company,
// in one record that can be sealed and parked in object storage.
//
// It lives in config/types rather than in the config module because both `config`
// and `invoicing` write it and `invoicing` reads it, and a module body may never
// import another module body (docs/MODULE_BOUNDARIES.md). Only the assembly is
// here — reading the rows and storing the sealed file is cloud/company_config_blob.go,
// because L2 may not import `cloud`.
//
// Numeric `cb` ids are explicit on every type declared here. Colbin's hashed ids are
// already deterministic, so this is not about that: it is so renaming a field cannot
// move an id, and so every type stays inside colbin's 4-bit id window (all ids <= 14),
// which is why this is four small structs instead of one flat one.
//
// Nothing already persisted by the ORM is tagged. genix-orm colbin-marshals struct
// columns with hashed ids, so InvoiceSeries and CulqiConfig are on disk under those
// ids — giving them explicit ones would orphan every stored company row.

package types

import (
	"app/core"
	invoicing "app/invoicing/types"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/ivanjoz/facturago"
)

// CompanyConfigVersion is the blob format. A reader that does not recognise it
// rebuilds instead of guessing at the payload.
const CompanyConfigVersion = int8(1)

type CompanyConfig struct {
	Version       int8                     `cb:"1"`
	CompanyID     int32                    `cb:"2"`
	SourceUpdated int32                    `cb:"3"` // max Updated of the rows it was built from
	GeneratedAt   int32                    `cb:"4"`
	Company       CompanyConfigCompany     `cb:"5"`
	Sunat         CompanyConfigSunat       `cb:"6"`
	Culqi         CompanyConfigCulqi       `cb:"7"`
	Parameters    []CompanyConfigParameter `cb:"8"`
}

type CompanyConfigCompany struct {
	RUC               string `cb:"1"`
	LegalName         string `cb:"2"`
	TradeName         string `cb:"3"`
	Address           string `cb:"4"`
	City              string `cb:"5"`
	Phone             string `cb:"6"`
	Email             string `cb:"7"`
	NotificationEmail string `cb:"8"`
}

// CompanyConfigSunat is what signing and transmitting a document needs.
//
// The private key and the leaf certificate are carried instead of the .pfx they
// came from: facturago signs with exactly those two, and the container, its MAC,
// its bag attributes, the CA chain and the container password are transport
// wrapping that ParsePKCS12 discards. Storing the pair is smaller and means the
// certificate password never reaches the blob at all.
type CompanyConfigSunat struct {
	SolUser        string                    `cb:"1"`
	SolPassword    string                    `cb:"2"`
	PrivateKeyDER  []byte                    `cb:"3"` // PKCS#8 DER
	CertificateDER []byte                    `cb:"4"` // leaf X509 DER
	CertRUC        string                    `cb:"5"`
	CertValidTo    int32                     `cb:"6"`
	Environment    int8                      `cb:"7"`
	Series         []invoicing.InvoiceSeries `cb:"8"`
}

// CompanyConfigCulqi carries the keys the backend charges with. PubKeyLive and
// PubKeyDev are the browser's half and are deliberately absent: nothing server-side
// reads them, and a secret store should not hold what is already public.
type CompanyConfigCulqi struct {
	RsaKey   string `cb:"1"`
	RsaKeyID string `cb:"2"`
	KeyLive  string `cb:"3"`
	KeyDev   string `cb:"4"`
}

// CompanyConfigParameter is a parameters row minus what the blob already knows
// (CompanyID) and what no reader of it needs (Status, Updated, UpdatedBy).
type CompanyConfigParameter struct {
	Group    int32   `cb:"1"`
	Key      string  `cb:"2"`
	Value    string  `cb:"3"`
	ValueInt int32   `cb:"4"`
	Values   []int32 `cb:"5"`
}

// AssembleCompanyConfig builds the record from rows already read.
//
// Taking the rows as arguments rather than reading them is what keeps this
// testable without a database, and it is also what lets the caller decide whether
// the company comes from the data mirror or from the cluster.
//
// secrets may be nil: a company that has not configured SUNAT yet still has a
// config worth caching, it just cannot issue. Everything else is required.
func AssembleCompanyConfig(company *Company, secrets *invoicing.CompanySecrets,
	parameters []Parameters) (*CompanyConfig, error) {

	if company == nil {
		return nil, errors.New("no se puede compactar la configuración de una empresa vacía")
	}

	config := CompanyConfig{
		Version:       CompanyConfigVersion,
		CompanyID:     company.ID,
		GeneratedAt:   core.SUnixTime(),
		SourceUpdated: company.Updated,
		Company: CompanyConfigCompany{
			RUC:               company.RUC,
			LegalName:         company.LegalName,
			TradeName:         company.Name,
			Address:           company.Address,
			City:              company.City,
			Phone:             company.Phone,
			Email:             company.Email,
			NotificationEmail: company.NotificationEmail,
		},
		Culqi: CompanyConfigCulqi{
			RsaKey:   company.CulqiConfig.RsaKey,
			RsaKeyID: company.CulqiConfig.RsaKeyID,
			KeyLive:  company.CulqiConfig.KeyLive,
			KeyDev:   company.CulqiConfig.KeyDev,
		},
		Sunat: CompanyConfigSunat{Series: company.InvoiceSeries},
	}

	// The blob is a legal-name document holder: BuildIssuer falls back to the trade
	// name, so the same fallback belongs here rather than at every reader.
	if config.Company.LegalName == "" {
		config.Company.LegalName = company.Name
	}

	for _, parameter := range parameters {
		if parameter.Status != 1 {
			continue
		}
		config.Parameters = append(config.Parameters, CompanyConfigParameter{
			Group:    parameter.Group,
			Key:      parameter.Key,
			Value:    parameter.Value,
			ValueInt: parameter.ValueInt,
			Values:   parameter.Values,
		})
		if parameter.Updated > config.SourceUpdated {
			config.SourceUpdated = parameter.Updated
		}
	}

	if secrets != nil {
		if err := fillSunatSecrets(&config.Sunat, secrets); err != nil {
			return nil, err
		}
		if secrets.Updated > config.SourceUpdated {
			config.SourceUpdated = secrets.Updated
		}
	}

	return &config, nil
}

// fillSunatSecrets decrypts the credentials and reduces the .pfx to the two pieces
// a signature actually consumes.
//
// The whole blob is sealed, so what lands here is plaintext. Re-encrypting the
// fields inside an encrypted file would mean two keys to rotate for one artifact
// and would buy nothing: anyone who can open the blob can open the fields.
func fillSunatSecrets(sunat *CompanyConfigSunat, secrets *invoicing.CompanySecrets) error {
	sunat.SolUser = secrets.SolUser
	sunat.CertRUC = secrets.CertRUC
	sunat.CertValidTo = secrets.CertValidTo
	sunat.Environment = secrets.Environment

	if len(secrets.SolPasswordEnc) > 0 {
		solPassword, err := core.Decrypt(secrets.SolPasswordEnc)
		if err != nil {
			return errors.New("no se pudo descifrar la clave SOL de la empresa")
		}
		sunat.SolPassword = string(solPassword)
	}

	if len(secrets.CertificateEnc) == 0 {
		return nil
	}

	certificate, err := core.Decrypt(secrets.CertificateEnc)
	if err != nil {
		return errors.New("no se pudo descifrar el certificado de la empresa")
	}
	certPassword, err := core.Decrypt(secrets.CertPasswordEnc)
	if err != nil {
		return errors.New("no se pudo descifrar la clave del certificado")
	}

	credential, err := facturago.ParsePKCS12(certificate, string(certPassword))
	if err != nil {
		return fmt.Errorf("no se pudo leer el certificado de la empresa: %w", err)
	}
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(credential.PrivateKey)
	if err != nil {
		return fmt.Errorf("no se pudo serializar la llave privada del certificado: %w", err)
	}

	sunat.PrivateKeyDER = privateKeyDER
	sunat.CertificateDER = credential.Certificate.Raw
	// Read off the certificate rather than trusted from the row: the row's copy is
	// what the upload recorded, and the blob is what will actually sign.
	if credential.RUC != "" {
		sunat.CertRUC = credential.RUC
	}
	sunat.CertValidTo = core.UnixToSunix(credential.NotAfter.Unix())
	return nil
}
