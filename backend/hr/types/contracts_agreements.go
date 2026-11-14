package types

import "app/db"

// ContractAgreement is a payroll contract, or the agreement of a worker off payroll with the payroll-only
// fields empty (docs/MANUFACTURING_PLAN.md §9.5, M28, M30). It is a dated version of a worker's terms (Odoo 19's hr.version): a raise, a
// change of pension system or a new bank account is a new row with its own ValidFrom, and a
// payslip uses the row whose ValidFrom covers its period. Everything here carries money or feeds
// the calculation, so it is served only by the payroll access (plan §3.0, rule 2).
type ContractAgreement struct {
	db.TableStruct[ContractAgreementTable, ContractAgreement]
	CompanyID  int32 `cb:"1"`
	ID         int32 `cb:"2"`
	EmployeeID int32 `cb:"3"`
	// ValidFrom is the UnixDay this version takes effect. Unique per employee.
	ValidFrom    int16 `cb:"4"`
	ContractType int8  `cb:"5"`
	// StartDate / EndDate are the contract's own dates (UnixDays). EndDate 0 = indefinite.
	StartDate int16 `cb:"6"`
	EndDate   int16 `cb:"7"`
	// PayAmount is PEN cents per month, week or hour, by PayBasis. A payroll contract is always monthly.
	PayAmount int32 `cb:"8"`
	// DailyHours below 4 makes a part-time worker (no CTS, reduced vacations).
	DailyHours        int8   `cb:"9"`
	PensionSystem     int8   `cb:"10"`
	CUSPP             string `cb:"11"` // AFP affiliate code, only for an AFP
	AfpCommissionType int8   `cb:"12"` // only for an AFP
	HasEPS            bool   `cb:"13"`
	SalaryBankID      int8   `cb:"14"`
	SalaryAccount     string `cb:"15"`
	CtsBankID         int8   `cb:"16"`
	CtsAccount        string `cb:"17"`
	Status            int8   `json:"ss" cb:"ss,18"`
	Updated           int32  `json:"upd" cb:"upd,19"`
	Version           int16  `json:"ver" cb:"ver,20"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,21"`
	UpdatedBy int32   `cb:"22"`
	Created   int32   `cb:"23"`
	CreatedBy int32   `cb:"24"`
	// PayBasis is PayBasisMonthly, PayBasisWeekly or PayBasisHourly.
	PayBasis int8 `cb:"25"`
	// PayDayOfWeek is the weekly pay day, 1 Monday … 7 Sunday; 0 unless weekly.
	PayDayOfWeek int8 `cb:"26"`
}

type ContractAgreementTable struct {
	db.TableStruct[ContractAgreementTable, ContractAgreement]
	CompanyID         db.Col[*ContractAgreementTable, int32]
	ID                db.Col[*ContractAgreementTable, int32]
	EmployeeID        db.Col[*ContractAgreementTable, int32]
	ValidFrom         db.Col[*ContractAgreementTable, int16]
	ContractType      db.Col[*ContractAgreementTable, int8]
	StartDate         db.Col[*ContractAgreementTable, int16]
	EndDate           db.Col[*ContractAgreementTable, int16]
	PayAmount         db.Col[*ContractAgreementTable, int32]
	DailyHours        db.Col[*ContractAgreementTable, int8]
	PensionSystem     db.Col[*ContractAgreementTable, int8]
	CUSPP             db.Col[*ContractAgreementTable, string]
	AfpCommissionType db.Col[*ContractAgreementTable, int8]
	HasEPS            db.Col[*ContractAgreementTable, bool]
	SalaryBankID      db.Col[*ContractAgreementTable, int8]
	SalaryAccount     db.Col[*ContractAgreementTable, string]
	CtsBankID         db.Col[*ContractAgreementTable, int8]
	CtsAccount        db.Col[*ContractAgreementTable, string]
	Status            db.Col[*ContractAgreementTable, int8]
	Updated           db.Col[*ContractAgreementTable, int32]
	Version           db.Col[*ContractAgreementTable, int16]
	UpdatedBy         db.Col[*ContractAgreementTable, int32]
	Created           db.Col[*ContractAgreementTable, int32]
	CreatedBy         db.Col[*ContractAgreementTable, int32]
	PayBasis          db.Col[*ContractAgreementTable, int8]
	PayDayOfWeek      db.Col[*ContractAgreementTable, int8]
}

// The whole list is a few rows per employee: one delta slice, and the browser resolves "the
// version valid on date D" locally. The local index serves the save handler's per-employee checks.
func (e ContractAgreementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          73,
		Name:        "hr_contracts_agreements",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.EmployeeID)},
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
