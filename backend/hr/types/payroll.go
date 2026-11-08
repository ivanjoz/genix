package types

import "app/db"

// One status numbering for runs, payslips and advances (plan §3.0 rule 6). The accounting reads
// payslips with Delta(cursor, 2, 3), so a reopened payslip (2 → 1) leaves its slice as a removed ID.
const (
	PayrollStatusOpen   int8 = 1 // draft or calculated (told apart by CalculatedAt)
	PayrollStatusClosed int8 = 2
	PayrollStatusPaid   int8 = 3
)

// PayrollRun is the one payroll of a month: a payslip per employee holding every concept of the month
// (salary, gratificación, CTS, settlement, extraordinary payments, profit sharing). Its payslips are
// calculated, closed and paid together.
type PayrollRun struct {
	db.TableStruct[PayrollRunTable, PayrollRun]
	CompanyID int32 `cb:"1"`
	ID        int32 `cb:"2"`
	Period    int16 `cb:"3"` // YYMM
	// InputsStamp is the newest Updated among every record the last calculation read: a later one
	// means the payslips are stale and the run is recalculated (see PayrollInputsStamp).
	InputsStamp int32 `cb:"4"`
	// PayDate is the UnixDay the run is paid on; the 5ta and the boleta date use it.
	PayDate int16  `cb:"5"`
	Note    string `cb:"6"`
	// Totals of the last calculation, for the run list. Cents.
	EmployeeCount     int32 `cb:"7"`
	TotalNetPay       int32 `cb:"8"`
	TotalEmployerCost int32 `cb:"9"`
	// SUnixTimes and users of the last transitions.
	CalculatedAt int32 `cb:"10"`
	ClosedAt     int32 `cb:"11"`
	ClosedBy     int32 `cb:"12"`
	PaidAt       int32 `cb:"13"`
	Status       int8  `json:"ss" cb:"ss,14"`
	Updated      int32 `json:"upd" cb:"upd,15"`
	Version      int16 `json:"ver" cb:"ver,16"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,17"`
	UpdatedBy int32   `cb:"18"`
	Created   int32   `cb:"19"`
	CreatedBy int32   `cb:"20"`
	// CashBankID paid the net pay: a Payroll payment movement against 4111 whose DocumentID is the run.
	CashBankID int32 `cb:"21"`
	// Profit sharing only (plan §4.3): the fiscal year is the one before the run's period; its net
	// income before tax comes from the annual return, and the sharing rate (basis points) from the
	// company's sector.
	NetIncomeBeforeTax int64 `cb:"22"`
	SharingRate        int32 `cb:"23"`
}

type PayrollRunTable struct {
	db.TableStruct[PayrollRunTable, PayrollRun]
	CompanyID          db.Col[*PayrollRunTable, int32]
	ID                 db.Col[*PayrollRunTable, int32]
	Period             db.Col[*PayrollRunTable, int16]
	InputsStamp        db.Col[*PayrollRunTable, int32]
	PayDate            db.Col[*PayrollRunTable, int16]
	Note               db.Col[*PayrollRunTable, string]
	EmployeeCount      db.Col[*PayrollRunTable, int32]
	TotalNetPay        db.Col[*PayrollRunTable, int32]
	TotalEmployerCost  db.Col[*PayrollRunTable, int32]
	CalculatedAt       db.Col[*PayrollRunTable, int32]
	ClosedAt           db.Col[*PayrollRunTable, int32]
	ClosedBy           db.Col[*PayrollRunTable, int32]
	PaidAt             db.Col[*PayrollRunTable, int32]
	Status             db.Col[*PayrollRunTable, int8]
	Updated            db.Col[*PayrollRunTable, int32]
	Version            db.Col[*PayrollRunTable, int16]
	UpdatedBy          db.Col[*PayrollRunTable, int32]
	Created            db.Col[*PayrollRunTable, int32]
	CreatedBy          db.Col[*PayrollRunTable, int32]
	CashBankID         db.Col[*PayrollRunTable, int32]
	NetIncomeBeforeTax db.Col[*PayrollRunTable, int64]
	SharingRate        db.Col[*PayrollRunTable, int32]
}

func (e PayrollRunTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          81,
		Name:        "hr_payroll_runs",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1, 2, 3}}},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
			// One run per month: two page loads that both find a month without its run both try to
			// create it, and the loser's insert fails here and reruns onto the winner's run. A run is
			// never deleted, so the month stays taken.
			{Type: db.TypeUniqueIndex, Keys: db.Cols(e.Period)},
		},
	}
}

// PayslipLine is one concept of a payslip, frozen at calculation: the treatment and accounts of the
// concept are stamped, so the boleta, PLAME and the accounting never read the concept again.
type PayslipLine struct {
	ConceptID int32 `cb:"2"` // legal (1..LegalConceptMaxID) or company concept (hr_concepts)
	Kind      int8  `cb:"3"`
	Base      int32 `cb:"4"` // cents the rate applied to, when there is one
	Rate      int32 `cb:"5"` // basis points
	// Quantity is days or hours, for the boleta (e.g. 2.5 overtime hours).
	Quantity              float32 `cb:"6"`
	Amount                int32   `cb:"7"` // cents, always positive; Kind says the sign
	PlameCode             string  `cb:"8"`
	IsRemunerative        bool    `cb:"9"`
	IncomeTax             int8    `cb:"10"`
	ComputableForBenefits bool    `cb:"11"`
	AccountCode           int32   `cb:"12"`
	PayableAccountCode    int32   `cb:"13"`
	// SourceID is the advance or loan a deduction settles, so closing the run can mark it.
	SourceID int32 `cb:"14"`
	// A benefit paid against its monthly provision (gratificación 4114, CTS 415, vacations 4115):
	// ProvisionedAmount of the line is charged to ProvisionAccountCode and only the rest to
	// AccountCode (plan §5.1 H4/H5), so the accounting projection needs no history.
	ProvisionAccountCode int32 `cb:"15"`
	ProvisionedAmount    int32 `cb:"16"`
	// OutsideNetPay marks an earning the worker is not paid through the payslip (the CTS is
	// deposited in his bank): it adds to the earnings and the employer cost but not to the net pay.
	OutsideNetPay bool `cb:"17"`
}

// PayslipParameters are the legal values a payslip was calculated with, kept for the boleta and
// so that a later correction of a legal value never rewrites a closed month.
type PayslipParameters struct {
	MinimumWage       int32 `cb:"1"`
	TaxUnit           int32 `cb:"2"`
	PensionSystem     int8  `cb:"3"`
	AfpFundRate       int32 `cb:"4"`
	AfpPremiumRate    int32 `cb:"5"`
	AfpCommissionRate int32 `cb:"6"`
	OnpRate           int32 `cb:"7"`
	EssaludRate       int32 `cb:"8"`
	MaxInsurable      int32 `cb:"9"`
}

// PayslipDiagnostic is a problem found while calculating: Blocking ones stop the run from closing.
type PayslipDiagnostic struct {
	Code     string `cb:"1"`
	Message  string `cb:"2"`
	Blocking bool   `cb:"3"`
}

// Payslip is the calculated-then-frozen document of one employee in one run (plan §3.5).
type Payslip struct {
	db.TableStruct[PayslipTable, Payslip]
	CompanyID  int32 `cb:"1"`
	ID         int32 `cb:"2"`
	RunID      int32 `cb:"3"`
	Period     int16 `cb:"5"` // YYMM of the run, stamped
	EmployeeID int32 `cb:"6"`
	ContractID int32 `cb:"7"`
	// Snapshot of the employee's placement, so the cost report never joins history.
	DepartmentID int32 `cb:"8"`
	CostFunction int8  `cb:"9"`
	WorkedDays   int8  `cb:"10"`
	// Lines holds earnings, deductions, employer contributions and provisions, in calculation order.
	Lines           []PayslipLine       `cb:"11"`
	TotalEarnings   int32               `cb:"12"`
	TotalDeductions int32               `cb:"13"`
	NetPay          int32               `cb:"14"`
	EmployerCost    int32               `cb:"15"` // earnings + employer contributions
	Parameters      PayslipParameters   `cb:"16"`
	Diagnostics     []PayslipDiagnostic `cb:"17"`
	CalculatedAt    int32               `cb:"18"`
	Status          int8                `json:"ss" cb:"ss,19"`
	Updated         int32               `json:"upd" cb:"upd,20"`
	Version         int16               `json:"ver" cb:"ver,21"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,22"`
	UpdatedBy int32   `cb:"23"`
}

type PayslipTable struct {
	db.TableStruct[PayslipTable, Payslip]
	CompanyID       db.Col[*PayslipTable, int32]
	ID              db.Col[*PayslipTable, int32]
	RunID           db.Col[*PayslipTable, int32]
	Period          db.Col[*PayslipTable, int16]
	EmployeeID      db.Col[*PayslipTable, int32]
	ContractID      db.Col[*PayslipTable, int32]
	DepartmentID    db.Col[*PayslipTable, int32]
	CostFunction    db.Col[*PayslipTable, int8]
	WorkedDays      db.Col[*PayslipTable, int8]
	Lines           db.Col[*PayslipTable, []PayslipLine]
	TotalEarnings   db.Col[*PayslipTable, int32]
	TotalDeductions db.Col[*PayslipTable, int32]
	NetPay          db.Col[*PayslipTable, int32]
	EmployerCost    db.Col[*PayslipTable, int32]
	Parameters      db.Col[*PayslipTable, PayslipParameters]
	Diagnostics     db.Col[*PayslipTable, []PayslipDiagnostic]
	CalculatedAt    db.Col[*PayslipTable, int32]
	Status          db.Col[*PayslipTable, int8]
	Updated         db.Col[*PayslipTable, int32]
	Version         db.Col[*PayslipTable, int16]
	UpdatedBy       db.Col[*PayslipTable, int32]
}

func (e PayslipTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:         82,
		Name:       "hr_payslips",
		Partition:  e.CompanyID,
		Keys:       db.Cols(e.ID.Autoincrement(0)),
		CacheByIDs: true,
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1, 2, 3}},
			{Col: e.Period, Min: 0, Max: 9912},
		},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.RunID)},
			// The payroll grid of a period, and the accounting (closed and paid only).
			{Type: db.TypeDelta, Keys: db.Cols(e.Period, e.Status)},
			// One worker's history.
			{Type: db.TypeDelta, Keys: db.Cols(e.EmployeeID, e.Status)},
			// The multi-period cost report: one group per month.
			{Type: db.TypeCacheGroup, Keys: db.Cols(e.Period)},
		},
	}
}

// StampRun copies the run's ID and period onto the payslip: Period is a key its indexes slice by.
func (e *Payslip) StampRun(run PayrollRun) {
	e.RunID, e.Period = run.ID, run.Period
}

// PeriodMonth is the month (1–12) of a YYMM period.
func PeriodMonth(period int16) int {
	return int(period % 100)
}
