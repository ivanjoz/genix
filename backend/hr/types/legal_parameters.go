package types

import "app/db"

// Legal parameters are shared by every company of a country, so both tables are partitioned by
// CountryID (the pattern of business/types/generales.go) and synced whole through one cursor. Only
// the SaaS company edits them (System → Parámetros Laborales). Values change mid-year (the RMV) or
// monthly (the AFP rates), so every value is dated and the payroll reads the one in force on its
// period; a closed payslip stores the values it used.

const CountryIDPeru int32 = 604

// Rates are stored in hundredths of a percent (basis points): 9 % = 900, 1.37 % = 137.
const RateScale = 10000

// LegalParameterKey names a dated legal value. Money values are cents of PEN; rates are basis points.
const (
	LegalParamRMV                 int8 = 1 // remuneración mínima vital, cents
	LegalParamUIT                 int8 = 2 // unidad impositiva tributaria, cents
	LegalParamEssaludRate         int8 = 3 // employer health contribution, bp
	LegalParamEssaludEpsRate      int8 = 4 // EsSalud rate left when the worker has EPS, bp
	LegalParamOnpRate             int8 = 5 // public pension contribution, bp
	LegalParamMaxInsurableSalary  int8 = 6 // remuneración máxima asegurable (caps the AFP premium), cents
	LegalParamFamilyAllowanceRate int8 = 7 // asignación familiar as a share of the RMV, bp
	LegalParamGratiBonusRate      int8 = 8 // bonificación extraordinaria Ley 30334, bp
	LegalParamGratiBonusEpsRate   int8 = 9 // the same bonus when the worker has EPS, bp
)

// LegalParameterOption describes a key: Unit is "cents" or "bp", for the System page's inputs.
type LegalParameterOption struct {
	ID   int8   `cb:"1"`
	Name string `cb:"2"`
	Unit string `cb:"3"`
}

var LegalParameterOptions = []LegalParameterOption{
	{ID: LegalParamRMV, Name: "Minimum wage (RMV)|Remuneración mínima vital (RMV)", Unit: "cents"},
	{ID: LegalParamUIT, Name: "Tax unit (UIT)|Unidad impositiva tributaria (UIT)", Unit: "cents"},
	{ID: LegalParamEssaludRate, Name: "EsSalud rate|Tasa EsSalud", Unit: "bp"},
	{ID: LegalParamEssaludEpsRate, Name: "EsSalud rate with EPS|Tasa EsSalud con EPS", Unit: "bp"},
	{ID: LegalParamOnpRate, Name: "ONP rate|Tasa ONP", Unit: "bp"},
	{ID: LegalParamMaxInsurableSalary, Name: "Max. insurable salary (AFP premium)|Remuneración máxima asegurable", Unit: "cents"},
	{ID: LegalParamFamilyAllowanceRate, Name: "Family allowance (% of RMV)|Asignación familiar (% de la RMV)", Unit: "bp"},
	{ID: LegalParamGratiBonusRate, Name: "Gratificación bonus (Ley 30334)|Bonificación extraordinaria (Ley 30334)", Unit: "bp"},
	{ID: LegalParamGratiBonusEpsRate, Name: "Gratificación bonus with EPS|Bonificación extraordinaria con EPS", Unit: "bp"},
}

// LegalParameter is one dated value: Key has Value from ValidFrom until the next row of the same Key.
type LegalParameter struct {
	db.TableStruct[LegalParameterTable, LegalParameter]
	CountryID int32 `cb:"1"`
	ID        int32 `cb:"2"`
	Key       int8  `cb:"3"`
	ValidFrom int16 `cb:"4"` // UnixDay
	Value     int32 `cb:"5"`
	Status    int8  `json:"ss" cb:"ss,6"`
	Updated   int32 `json:"upd" cb:"upd,7"`
	Version   int16 `json:"ver" cb:"ver,8"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,9"`
	UpdatedBy int32   `cb:"10"`
}

type LegalParameterTable struct {
	db.TableStruct[LegalParameterTable, LegalParameter]
	CountryID db.Col[*LegalParameterTable, int32]
	ID        db.Col[*LegalParameterTable, int32]
	Key       db.Col[*LegalParameterTable, int8]
	ValidFrom db.Col[*LegalParameterTable, int16]
	Value     db.Col[*LegalParameterTable, int32]
	Status    db.Col[*LegalParameterTable, int8]
	Updated   db.Col[*LegalParameterTable, int32]
	Version   db.Col[*LegalParameterTable, int16]
	UpdatedBy db.Col[*LegalParameterTable, int32]
}

func (e LegalParameterTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        77,
		Name:      "hr_legal_parameters",
		Partition: e.CountryID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		// Keyless: one log for the country's whole list. Deleted rows leave it as removed IDs.
		Indexes: []db.Index{{Type: db.TypeDelta}},
	}
}

// AfpRate is the SBS table of one AFP for one month (Period = YYMM of the accrual month).
type AfpRate struct {
	db.TableStruct[AfpRateTable, AfpRate]
	CountryID     int32 `cb:"1"`
	ID            int32 `cb:"2"`
	Period        int16 `cb:"3"`
	PensionSystem int8  `cb:"4"` // the AFP
	// All in basis points.
	FundRate        int32 `cb:"5"`
	PremiumRate     int32 `cb:"6"`
	FlowCommission  int32 `cb:"7"`
	MixedCommission int32 `cb:"8"` // charged on the balance, not through payroll; kept for reference
	Status          int8  `json:"ss" cb:"ss,9"`
	Updated         int32 `json:"upd" cb:"upd,10"`
	Version         int16 `json:"ver" cb:"ver,11"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,12"`
	UpdatedBy int32   `cb:"13"`
}

type AfpRateTable struct {
	db.TableStruct[AfpRateTable, AfpRate]
	CountryID       db.Col[*AfpRateTable, int32]
	ID              db.Col[*AfpRateTable, int32]
	Period          db.Col[*AfpRateTable, int16]
	PensionSystem   db.Col[*AfpRateTable, int8]
	FundRate        db.Col[*AfpRateTable, int32]
	PremiumRate     db.Col[*AfpRateTable, int32]
	FlowCommission  db.Col[*AfpRateTable, int32]
	MixedCommission db.Col[*AfpRateTable, int32]
	Status          db.Col[*AfpRateTable, int8]
	Updated         db.Col[*AfpRateTable, int32]
	Version         db.Col[*AfpRateTable, int16]
	UpdatedBy       db.Col[*AfpRateTable, int32]
}

func (e AfpRateTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        78,
		Name:      "hr_afp_rates",
		Partition: e.CountryID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes:   []db.Index{{Type: db.TypeDelta}},
	}
}

// LegalValues answers "what was the value on this day" over the stored rows. Built once per
// calculation from every active row of the country.
type LegalValues struct {
	parameters []LegalParameter
	afpRates   []AfpRate
}

func NewLegalValues(parameters []LegalParameter, afpRates []AfpRate) LegalValues {
	return LegalValues{parameters: parameters, afpRates: afpRates}
}

// NewestUpdated is the latest Updated among the legal rows: a change of law or an AFP table that
// arrives later makes the payrolls calculated before it stale.
func (values LegalValues) NewestUpdated() int32 {
	var newest int32
	for _, parameter := range values.parameters {
		newest = max(newest, parameter.Updated)
	}
	for _, afpRate := range values.afpRates {
		newest = max(newest, afpRate.Updated)
	}
	return newest
}

// ValueOn returns the value of key in force on unixDay: the row with the latest ValidFrom not after
// it. ok is false when the key has no value yet on that day.
func (values LegalValues) ValueOn(key int8, unixDay int16) (value int32, ok bool) {
	var latestValidFrom int16
	for _, parameter := range values.parameters {
		if parameter.Key != key || parameter.Status != StatusActive || parameter.ValidFrom > unixDay {
			continue
		}
		if !ok || parameter.ValidFrom > latestValidFrom {
			value, latestValidFrom, ok = parameter.Value, parameter.ValidFrom, true
		}
	}
	return value, ok
}

// AfpRateFor returns the SBS rates of an AFP for a YYMM period.
func (values LegalValues) AfpRateFor(pensionSystem int8, period int16) (AfpRate, bool) {
	for _, afpRate := range values.afpRates {
		if afpRate.PensionSystem == pensionSystem && afpRate.Period == period && afpRate.Status == StatusActive {
			return afpRate, true
		}
	}
	return AfpRate{}, false
}
