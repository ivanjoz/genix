package types

import (
	"app/core"
	"app/db"
	invoicing "app/invoicing/types"
)

type TAGS struct{}

type Company struct {
	db.TableStruct[CompanyTable, Company]
	ID                int32  `db:"id,pk" cb:"ID,1"`
	Name              string `json:",omitempty" cb:"Name,2"`
	LegalName         string `json:",omitempty" cb:"LegalName,3"`
	RUC               string `json:",omitempty" db:"ruc" cb:"RUC,4"`
	Email             string `json:",omitempty" db:"email" cb:"Email,5"`
	NotificationEmail string `json:",omitempty" cb:"NotificationEmail,6"`
	Phone             string `json:",omitempty" cb:"Phone,7"`
	Representative    string `json:",omitempty" cb:"Representative,8"`
	Address           string `json:",omitempty" cb:"Address,9"`
	// CityID is the district the fiscal address sits in, and it is the INEI ubigeo
	// itself: the catalog is keyed by the code, so 150101 is Lima / Lima / Lima.
	// SUNAT reads it off every document the company issues, which is why this is a
	// picked district and not the free-text city it replaced.
	CityID        int32       `json:",omitempty" cb:"CityID,10"`
	FormApiKey    string      `json:",omitempty" db:"form_api_key" cb:"FormApiKey,11"`
	EmailVerified int8        `json:",omitempty" cb:"EmailVerified,12"`
	PhoneVerified int8        `json:",omitempty" cb:"PhoneVerified,13"`
	CulqiConfig   CulqiConfig `json:",omitempty" db:"culqui_config" cb:"CulqiConfig,14"`
	// InvoiceSeries are the SUNAT series this company issues under. Inline rather
	// than a table of their own: there are at most 99, every emission needs one,
	// and reading the company already brings them along.
	InvoiceSeries []invoicing.InvoiceSeries `json:",omitempty" cb:"InvoiceSeries,15"`
	// Flags are the ids the company has checked in company_flags.toml, and only those:
	// an unchecked flag is absent, never stored as a zero. One flat list because the
	// catalog's ids are unique across its sections. core.HasCompanyFlag reads it.
	Flags []int16 `json:",omitempty" cb:"Flags,16"`
	// FlagValues carries the flags the catalog declares with a `type`: those hold a number
	// instead of an on/off, so their id never appears in Flags. The value is already scaled
	// by the flag's decimals — core.GetCompanyFlagValue unscales it back.
	FlagValues []core.CompanyFlagValue `json:",omitempty" cb:"FlagValues,17"`
	Updated    int32                   `json:"upd" db:"updated" cb:"upd,18"`
	Status     int8                    `json:"ss" db:"status" cb:"ss,19"`
}

type CompanyTable struct {
	db.TableStruct[CompanyTable, Company]
	ID                db.Col[*CompanyTable, int32]
	Name              db.Col[*CompanyTable, string]
	LegalName         db.Col[*CompanyTable, string]
	RUC               db.Col[*CompanyTable, string]
	Email             db.Col[*CompanyTable, string]
	NotificationEmail db.Col[*CompanyTable, string]
	Phone             db.Col[*CompanyTable, string]
	Representative    db.Col[*CompanyTable, string]
	Address           db.Col[*CompanyTable, string]
	CityID            db.Col[*CompanyTable, int32]
	FormApiKey        db.Col[*CompanyTable, string]
	EmailVerified     db.Col[*CompanyTable, int8]
	PhoneVerified     db.Col[*CompanyTable, int8]
	CulqiConfig       db.Col[*CompanyTable, CulqiConfig]
	InvoiceSeries     db.Col[*CompanyTable, []invoicing.InvoiceSeries]
	Flags             db.ColSlice[*CompanyTable, int16]
	FlagValues        db.Col[*CompanyTable, []core.CompanyFlagValue]
	Updated           db.Col[*CompanyTable, int32]
	Status            db.Col[*CompanyTable, int8]
}

func (e CompanyTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           9,
		Name:         "companies",
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Companies are global, so these indexes carry no tenant prefix. Email must be a
		// global index and not a local one: a local index is scoped to the table's partition
		// column, and this table has none. Order is the cloud mirror's slot order (ix1, ix2),
		// so new entries go at the end.
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: db.Cols(e.Updated)},
			// Public sign-up enforces one company per email address, which needs this lookup.
			{Type: db.TypeGlobalIndex, Keys: db.Cols(e.Email)},
		},
	}
}

type CulqiConfig struct {
	RsaKey     string `json:",omitempty" cb:"RsaKey,1"`
	RsaKeyID   string `json:",omitempty" cb:"RsaKeyID,2"`
	KeyLive    string `json:",omitempty" cb:"KeyLive,3"`
	PubKeyLive string `json:",omitempty" cb:"PubKeyLive,4"`
	KeyDev     string `json:",omitempty" cb:"KeyDev,5"`
	PubKeyDev  string `json:",omitempty" cb:"PubKeyDev,6"`
}

type CompanyPub struct {
	ID            int32  `json:"id" cb:"id,1"`
	Name          string `json:",omitempty" cb:"Name,2"`
	CulqiRsaKey   string `json:",omitempty" cb:"CulqiRsaKey,3"`
	CulqiRsaKeyID string `json:",omitempty" cb:"CulqiRsaKeyID,4"`
	CulqiLlave    string `json:",omitempty" cb:"CulqiLlave,5"`
}
