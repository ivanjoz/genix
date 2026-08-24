package types

import "app/db"

// Kinds of secret a company can store. Only SUNAT credentials exist today; the
// column is here so a payment gateway or a courier account does not need a
// table of its own later.
const (
	SecretTypeSunatCPE = int8(1)
)

// Environments a company can issue against.
const (
	SunatEnvBeta       = int8(1)
	SunatEnvProduction = int8(2)
)

// CompanySecrets holds what a company needs in order to sign and transmit an
// electronic document: its SOL credentials and its digital certificate.
//
// The certificate lives in a column rather than in S3. A .pfx is three to six
// kilobytes, so the row costs nothing, and keeping it here means an emission
// reads one partition instead of making a round trip to object storage while a
// customer waits at the till.
//
// Rotation is why the key is an autoincrement rather than the company id: a
// company replaces an expiring certificate by inserting a new row and retiring
// the old one to Status 0, which keeps the documents signed with it auditable.
type CompanySecrets struct {
	db.TableStruct[CompanySecretsTable, CompanySecrets]
	CompanyID int32  `json:",omitempty"`
	ID        int32  `json:",omitempty"`
	Type      int8   `json:",omitempty"`
	Name      string `json:",omitempty"`

	// SolUser is the secondary SOL user. It travels in clear because on its own
	// it authenticates nothing.
	SolUser string `json:",omitempty"`

	// Encrypted with core.Encrypt before they reach the database, and excluded
	// from every response: no endpoint returns key material, not even to the
	// company that owns it.
	SolPasswordEnc  []byte `json:"-"`
	CertificateEnc  []byte `json:"-"`
	CertPasswordEnc []byte `json:"-"`

	// Read out of the certificate when it is uploaded, so the UI can show whose
	// it is and warn before it expires without decrypting anything.
	CertSubject   string `json:",omitempty"`
	CertIssuer    string `json:",omitempty"`
	CertSerial    string `json:",omitempty"`
	CertRUC       string `json:",omitempty" db:"cert_ruc"`
	CertValidFrom int32  `json:",omitempty"`
	CertValidTo   int32  `json:",omitempty"`

	Environment int8 `json:",omitempty"`

	Status         int8  `json:"ss,omitempty"`
	Updated        int32 `json:"upd,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
	Created        int32 `json:",omitempty"`
	CreatedBy      int32 `json:",omitempty"`
}

type CompanySecretsTable struct {
	db.TableStruct[CompanySecretsTable, CompanySecrets]
	CompanyID       db.Col[*CompanySecretsTable, int32]
	ID              db.Col[*CompanySecretsTable, int32]
	Type            db.Col[*CompanySecretsTable, int8]
	Name            db.Col[*CompanySecretsTable, string]
	SolUser         db.Col[*CompanySecretsTable, string]
	SolPasswordEnc  db.Col[*CompanySecretsTable, []byte]
	CertificateEnc  db.Col[*CompanySecretsTable, []byte]
	CertPasswordEnc db.Col[*CompanySecretsTable, []byte]
	CertSubject     db.Col[*CompanySecretsTable, string]
	CertIssuer      db.Col[*CompanySecretsTable, string]
	CertSerial      db.Col[*CompanySecretsTable, string]
	CertRUC         db.Col[*CompanySecretsTable, string] `db:"cert_ruc"`
	CertValidFrom   db.Col[*CompanySecretsTable, int32]
	CertValidTo     db.Col[*CompanySecretsTable, int32]
	Environment     db.Col[*CompanySecretsTable, int8]
	Status          db.Col[*CompanySecretsTable, int8]
	Updated         db.Col[*CompanySecretsTable, int32]
	UpdatedVersion  db.Col[*CompanySecretsTable, int32]
	UpdatedBy       db.Col[*CompanySecretsTable, int32]
	Created         db.Col[*CompanySecretsTable, int32]
	CreatedBy       db.Col[*CompanySecretsTable, int32]
}

func (e CompanySecretsTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                 26,
		Name:               "company_secrets",
		Partition:          e.CompanyID,
		Keys:               db.Cols(e.ID.Autoincrement(0)),
		SaveUpdatedVersion: true,
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			// Resolves "the active SUNAT credentials of this company" in one read,
			// which is what every emission starts with.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.Type)},
		},
	}
}
