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
	business "app/business/types"
	"app/core"
	invoicing "app/invoicing/types"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/ivanjoz/facturago"
)

// CompanyConfigVersion is the blob format. A reader that does not recognise it
// rebuilds instead of guessing at the payload.
//
// 2: the sites travel in the blob. Bumped rather than relying on a missing field
// decoding as empty, because an issuer reading a version-1 blob would find no site
// and refuse to build a document that is perfectly issuable.
const CompanyConfigVersion = int8(2)

type CompanyConfig struct {
	Version       int8                     `cb:"1"`
	CompanyID     int32                    `cb:"2"`
	SourceUpdated int32                    `cb:"3"` // max Updated of the rows it was built from
	GeneratedAt   int32                    `cb:"4"`
	Company       CompanyConfigCompany     `cb:"5"`
	Sunat         CompanyConfigSunat       `cb:"6"`
	Culqi         CompanyConfigCulqi       `cb:"7"`
	Parameters    []CompanyConfigParameter `cb:"8"`
	Sites         []CompanyConfigSite      `cb:"9"`
}

// CompanyConfigSite is an establishment, carrying what a document has to declare
// about the place it was issued from and nothing else.
//
// The sites are in the blob because every emission needs one: the series names the
// establishment, and resolving it used to be a cluster read per document. CityID is
// the six-digit INEI code, which is exactly the ubigeo SUNAT expects.
//
// Status travels with them because a series can name a site that was later retired,
// and a document already issued under it still has to build.
type CompanyConfigSite struct {
	ID      int32  `cb:"1"`
	Name    string `cb:"2"`
	Address string `cb:"3"`
	CityID  int32  `cb:"4"`
	Status  int8   `cb:"5"`
}

// CompanyConfigCompany is the taxpayer, as every document declares it.
//
// The fiscal address is carried resolved — the ubigeo and the three names SUNAT
// asks for — rather than as the district id alone. The names live in the city
// catalog, which is a table of 2000 rows nobody should read to sign one document,
// and they only change when the catalog is re-imported.
type CompanyConfigCompany struct {
	RUC               string `cb:"1"`
	LegalName         string `cb:"2"`
	TradeName         string `cb:"3"`
	Address           string `cb:"4"`
	CityID            int32  `cb:"5"`
	Phone             string `cb:"6"`
	Email             string `cb:"7"`
	NotificationEmail string `cb:"8"`
	District          string `cb:"9"`
	Province          string `cb:"10"`
	Department        string `cb:"11"`
}

// Ubigeo is the six-digit INEI code of the fiscal address, which is what SUNAT
// validates. The catalog is keyed by the code but stores it as an integer, so the
// leading zeros of a department like Amazonas (01) have to be put back.
func (e CompanyConfigCompany) Ubigeo() string {
	if e.CityID <= 0 {
		return ""
	}
	return fmt.Sprintf("%06d", e.CityID)
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
	parameters []Parameters, sites []business.Site,
	cities []business.CityLocation) (*CompanyConfig, error) {

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
			CityID:            company.CityID,
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
	fillFiscalAddressNames(&config.Company, cities)

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

	// Retired sites are kept, unlike retired parameters: a series can name a site
	// that was deactivated afterwards, and the documents issued under it still have
	// to declare the address they were issued from.
	for _, site := range sites {
		config.Sites = append(config.Sites, CompanyConfigSite{
			ID:      site.ID,
			Name:    site.Name,
			Address: site.Address,
			CityID:  site.CityID,
			Status:  site.Status,
		})
		if site.Updated > config.SourceUpdated {
			config.SourceUpdated = site.Updated
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

// fillFiscalAddressNames resolves the district, province and department of the
// company's ubigeo.
//
// The three are read off the catalog rows the caller passed, matched by id: the
// ubigeo *is* the id, and a district's parents are its own code truncated — 150101
// sits under 1501 under 15 — which is how the rest of the codebase walks it too.
// Nothing is filled when the company has no district picked, so a company that has
// not completed its address produces a blob that says so rather than a plausible
// wrong address.
func fillFiscalAddressNames(company *CompanyConfigCompany, cities []business.CityLocation) {
	if company.CityID <= 0 {
		return
	}

	districtID := company.CityID
	provinceID := districtID / 100
	departmentID := provinceID / 100

	for _, city := range cities {
		switch city.ID {
		case districtID:
			company.District = city.Name
		case provinceID:
			company.Province = city.Name
		case departmentID:
			company.Department = city.Name
		}
	}
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
