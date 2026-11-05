package types

import "app/db"

// Contract is a dated version of a worker's employment terms (Odoo 19's hr.version): a raise, a
// change of pension system or a new bank account is a new row with its own ValidFrom, and a
// payslip uses the row whose ValidFrom covers its period. Everything here carries money or feeds
// the calculation, so it is served only by the payroll access (plan §3.0, rule 2).
type Contract struct {
	db.TableStruct[ContractTable, Contract]
	CompanyID  int32 `cb:"1"`
	ID         int32 `cb:"2"`
	EmployeeID int32 `cb:"3"`
	// ValidFrom is the UnixDay this version takes effect. Unique per employee.
	ValidFrom    int16 `cb:"4"`
	ContractType int8  `cb:"5"`
	// StartDate / EndDate are the contract's own dates (UnixDays). EndDate 0 = indefinite.
	StartDate int16 `cb:"6"`
	EndDate   int16 `cb:"7"`
	// MonthlySalary is in cents of PEN.
	MonthlySalary int32 `cb:"8"`
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
}

type ContractTable struct {
	db.TableStruct[ContractTable, Contract]
	CompanyID         db.Col[*ContractTable, int32]
	ID                db.Col[*ContractTable, int32]
	EmployeeID        db.Col[*ContractTable, int32]
	ValidFrom         db.Col[*ContractTable, int16]
	ContractType      db.Col[*ContractTable, int8]
	StartDate         db.Col[*ContractTable, int16]
	EndDate           db.Col[*ContractTable, int16]
	MonthlySalary     db.Col[*ContractTable, int32]
	DailyHours        db.Col[*ContractTable, int8]
	PensionSystem     db.Col[*ContractTable, int8]
	CUSPP             db.Col[*ContractTable, string]
	AfpCommissionType db.Col[*ContractTable, int8]
	HasEPS            db.Col[*ContractTable, bool]
	SalaryBankID      db.Col[*ContractTable, int8]
	SalaryAccount     db.Col[*ContractTable, string]
	CtsBankID         db.Col[*ContractTable, int8]
	CtsAccount        db.Col[*ContractTable, string]
	Status            db.Col[*ContractTable, int8]
	Updated           db.Col[*ContractTable, int32]
	Version           db.Col[*ContractTable, int16]
	UpdatedBy         db.Col[*ContractTable, int32]
	Created           db.Col[*ContractTable, int32]
	CreatedBy         db.Col[*ContractTable, int32]
}

// The whole list is a few rows per employee: one delta slice, and the browser resolves "the
// version valid on date D" locally. The local index serves the save handler's per-employee checks.
func (e ContractTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          73,
		Name:        "hr_contracts",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.EmployeeID)},
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
