package types

import (
	"app/core"
	"app/db"
	invoicing "app/invoicing/types"
)

type TAGS struct{}

type Company struct {
	db.TableStruct[CompanyTable, Company]
	ID                int32  `db:",pk" cb:"1"`
	Name              string `cb:"2"`
	LegalName         string `cb:"3"`
	RUC               string `cb:"4"`
	Email             string `cb:"5"`
	NotificationEmail string `cb:"6"`
	Phone             string `cb:"7"`
	Representative    string `cb:"8"`
	Address           string `cb:"9"`
	// CityID is the district the fiscal address sits in, and it is the INEI ubigeo
	// itself: the catalog is keyed by the code, so 150101 is Lima / Lima / Lima.
	// SUNAT reads it off every document the company issues, which is why this is a
	// picked district and not the free-text city it replaced.
	CityID        int32       `cb:"10"`
	FormApiKey    string      `cb:"11"`
	EmailVerified int8        `cb:"12"`
	PhoneVerified int8        `cb:"13"`
	CulqiConfig   CulqiConfig `cb:"14"`
	// InvoiceSeries are the SUNAT series this company issues under. Inline rather
	// than a table of their own: there are at most 99, every emission needs one,
	// and reading the company already brings them along.
	InvoiceSeries []invoicing.InvoiceSeries `cb:"15"`
	// Flags are the ids the company has checked in company_flags.toml, and only those:
	// an unchecked flag is absent, never stored as a zero. One flat list because the
	// catalog's ids are unique across its sections. core.HasCompanyFlag reads it.
	Flags []int16 `cb:"16"`
	// FlagValues carries the flags the catalog declares with a `type`: those hold a number
	// instead of an on/off, so their id never appears in Flags. The value is already scaled
	// by the flag's decimals — core.GetCompanyFlagValue unscales it back.
	FlagValues []core.CompanyFlagValue `cb:"17"`
	Updated    int32                   `json:"upd" cb:"upd,18"`
	Status     int8                    `json:"ss" cb:"ss,19"`
	// Created is the SUnixTime the company registered in the system: the payroll lists months from
	// a year before it.
	Created int32 `cb:"20"`

	// Tax setup, read by the cash flow forecast to project what the company owes SUNAT each month.
	// TaxRegime is one of the TaxRegime* constants (0 = not set up yet).
	TaxRegime int8 `cb:"21"`
	// IncomeTaxPrepaymentRate is the monthly renta pago a cuenta over net sales, in basis points
	// (1.5 % = 150). Defaulted from the regime in the form, editable for the coefficient method.
	IncomeTaxPrepaymentRate int32 `cb:"22"`
	// IsGoodTaxpayer: a buen contribuyente files on its own column of the SUNAT calendar.
	IsGoodTaxpayer bool `cb:"23"`
	// NrusCategory is the Nuevo RUS category (NrusCategory*), whose fixed cuota replaces IGV and renta.
	NrusCategory int8 `cb:"24"`
}

// The tax regimes of a Peruvian company.
const (
	TaxRegimeGeneral = int8(1) // Régimen General
	TaxRegimeMype    = int8(2) // Régimen MYPE Tributario
	TaxRegimeSpecial = int8(3) // Régimen Especial de Renta
	TaxRegimeNrus    = int8(4) // Nuevo RUS
)

// The Nuevo RUS categories and their monthly cuota in cents (D.Leg. 937, as amended by D.Leg. 1270).
const (
	NrusCategory1 = int8(1) // S/ 20
	NrusCategory2 = int8(2) // S/ 50
)

var NrusMonthlyQuotaCents = map[int8]int32{NrusCategory1: 2000, NrusCategory2: 5000}

// ValidateCompanyTaxSetup checks the tax fields of a company about to be saved. A regime of 0 is
// allowed (not set up yet: the forecast warns about it). NRUS needs its category and pays no renta
// prepayment; every other regime has no category.
func ValidateCompanyTaxSetup(company *Company) error {
	if company.TaxRegime < 0 || company.TaxRegime > TaxRegimeNrus {
		return core.Err("El régimen tributario no es válido:", company.TaxRegime)
	}
	if company.IncomeTaxPrepaymentRate < 0 || company.IncomeTaxPrepaymentRate > 10000 {
		return core.Err("La tasa del pago a cuenta debe estar entre 0 % y 100 %.")
	}
	if company.TaxRegime == TaxRegimeNrus {
		if _, isCategory := NrusMonthlyQuotaCents[company.NrusCategory]; !isCategory {
			return core.Err("Seleccione la categoría del Nuevo RUS.")
		}
		company.IncomeTaxPrepaymentRate = 0
		return nil
	}
	company.NrusCategory = 0
	return nil
}

type CompanyTable struct {
	db.TableStruct[CompanyTable, Company]
	ID                      db.Col[*CompanyTable, int32]
	Name                    db.Col[*CompanyTable, string]
	LegalName               db.Col[*CompanyTable, string]
	RUC                     db.Col[*CompanyTable, string]
	Email                   db.Col[*CompanyTable, string]
	NotificationEmail       db.Col[*CompanyTable, string]
	Phone                   db.Col[*CompanyTable, string]
	Representative          db.Col[*CompanyTable, string]
	Address                 db.Col[*CompanyTable, string]
	CityID                  db.Col[*CompanyTable, int32]
	FormApiKey              db.Col[*CompanyTable, string]
	EmailVerified           db.Col[*CompanyTable, int8]
	PhoneVerified           db.Col[*CompanyTable, int8]
	CulqiConfig             db.Col[*CompanyTable, CulqiConfig]
	InvoiceSeries           db.Col[*CompanyTable, []invoicing.InvoiceSeries]
	Flags                   db.ColSlice[*CompanyTable, int16]
	FlagValues              db.Col[*CompanyTable, []core.CompanyFlagValue]
	Updated                 db.Col[*CompanyTable, int32]
	Status                  db.Col[*CompanyTable, int8]
	Created                 db.Col[*CompanyTable, int32]
	TaxRegime               db.Col[*CompanyTable, int8]
	IncomeTaxPrepaymentRate db.Col[*CompanyTable, int32]
	IsGoodTaxpayer          db.Col[*CompanyTable, bool]
	NrusCategory            db.Col[*CompanyTable, int8]
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
	RsaKey     string `cb:"1"`
	RsaKeyID   string `cb:"2"`
	KeyLive    string `cb:"3"`
	PubKeyLive string `cb:"4"`
	KeyDev     string `cb:"5"`
	PubKeyDev  string `cb:"6"`
	// IsLive switches the web store between the live keys and the test keys: the public key the
	// storefront opens Culqi with and the secret key the backend charges with follow it together.
	IsLive bool `cb:"7"`
	// YapeEnabled offers Yape next to the card on the web store's payment step.
	YapeEnabled bool `cb:"8"`
}

// PublicKey is the browser's key for the company's current mode.
func (e CulqiConfig) PublicKey() string {
	return core.If(e.IsLive, e.PubKeyLive, e.PubKeyDev)
}

// CompanyPub is the public company file, empresas/e-{companyID}.json, that the storefront reads
// without a session. Everything here is visible to anyone: never put a secret in it.
type CompanyPub struct {
	ID            int32  `json:"id" cb:"id,1"`
	Name          string `cb:"2"`
	CulqiRsaKey   string `cb:"3"`
	CulqiRsaKeyID string `cb:"4"`
	CulqiLlave    string `cb:"5"`
	// PickupSites are the sites marked IsPickupPoint, offered on "Recojo en Tienda".
	PickupSites []CompanyPubSite `cb:"6"`
	// Flags are the company's [ecommerce] flags the storefront needs before calling the backend
	// (contact number required, cash on delivery, receipts). The backend never trusts this copy.
	Flags       []int16 `cb:"7"`
	YapeEnabled bool    `cb:"8"`
}

type CompanyPubSite struct {
	ID      int32  `cb:"1"`
	Name    string `cb:"2"`
	Address string `cb:"3"`
	CityID  int32  `cb:"4"`
}

// LockCompanyIn locks the company's row in the caller's transaction and returns it. It is the
// company-wide write lock: the company form and the invoice series hold it because they rewrite the
// row, and the client/provider and product saves hold it so their name uniqueness checks (plain
// reads of a mutable hash, which no unique index can cover) cannot interleave.
func LockCompanyIn(transaction *db.Transaction, companyID int32) (Company, error) {
	companies := []Company{{ID: companyID}}
	if err := db.LockByKeys(transaction, &companies); err != nil {
		return Company{}, core.Err("Error al obtener la empresa:", err)
	}
	if len(companies) == 0 {
		return Company{}, core.Err("No se encontró la empresa", companyID)
	}
	return companies[0], nil
}
