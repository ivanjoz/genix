package types

import (
	"app/core"
	"app/db"
)

// EmployeeRecord is everything that happened to an employee on a date or in a month: the month's
// novedades (absences, overtime, one-off amounts, extraordinary payments), the payments made before
// the payslip (weekly, biweekly), loans and their installments, vacations and leaves. Type says
// which; each type reads only the columns its comment names. What does not belong to a date (a
// recurring concept) is an EmployeeConcept.

// Employee record types, grouped by the screen that registers them.
const (
	// The month's novedades (payroll → Entries).
	RecordAbsenceDays   int8 = 1 // Quantity: unpaid absence days, they reduce the worked days
	RecordOvertime25    int8 = 2 // Quantity: hours at 25 % (first two daily overtime hours)
	RecordOvertime35    int8 = 3 // Quantity: hours at 35 %
	RecordConceptAmount int8 = 4 // ConceptID, Amount: a company concept for this month only
	// RecordExtraordinaryPayment is a company earning of its own (ConceptID, Amount): its payslip
	// lines carry their own pension, 5ta and EsSalud, apart from the monthly salary's.
	RecordExtraordinaryPayment int8 = 5

	// Payments made before the payslip, which nets them (payroll → Weekly payments): Date paid,
	// FromDate–ToDate covered (one month), Amount, CashBankID it left from.
	RecordAdvanceWeekly   int8 = 10
	RecordAdvanceBiweekly int8 = 11
	RecordAdvanceOther    int8 = 12

	// RecordLoan is money lent (account 1411): Date granted, Amount, Quantity installments, Period of
	// the first installment, CashBankID it left from. Its installments are RecordLoanInstallment rows
	// (LoanID, Period, Amount), created with it; the monthly payslip of their Period recovers them.
	RecordLoan            int8 = 20
	RecordLoanInstallment int8 = 21

	// Time off, both dates included (FromDate, ToDate; Quantity the calendar days). The option Code is
	// the SUNAT "tipo de suspensión" the PLAME .snl file reports.
	// TODO(accountant): verify the codes against the current PLAME table 21.
	RecordVacation               int8 = 30
	RecordMedicalLeavePaid       int8 = 31 // incapacidad temporal, first 20 days of the year: paid by the employer
	RecordMedicalLeaveSubsidized int8 = 32 // incapacidad temporal subsidiada by EsSalud
	RecordMaternityLeave         int8 = 33 // subsidiada by EsSalud
	RecordUnpaidLeave            int8 = 34 // licencia sin goce de haber
)

var InputTypeOptions = []HROption{
	{ID: RecordAbsenceDays, Name: "Unpaid absence (days)|Inasistencias (días)"},
	{ID: RecordOvertime25, Name: "Overtime 25% (hours)|Horas extras 25% (horas)"},
	{ID: RecordOvertime35, Name: "Overtime 35% (hours)|Horas extras 35% (horas)"},
	{ID: RecordConceptAmount, Name: "Concept amount|Monto de concepto"},
	{ID: RecordExtraordinaryPayment, Name: "Extraordinary payment|Pago extraordinario"},
}

var AdvanceTypeOptions = []HROption{
	{ID: RecordAdvanceWeekly, Name: "Weekly|Semanal"},
	{ID: RecordAdvanceBiweekly, Name: "Biweekly|Quincenal"},
	{ID: RecordAdvanceOther, Name: "Other|Otro"},
}

var TimeOffTypeOptions = []HROption{
	{ID: RecordVacation, Code: "23", Name: "Vacation|Vacaciones"},
	{ID: RecordMedicalLeavePaid, Code: "20", Name: "Medical leave (paid by the company)|Descanso médico (pagado por la empresa)"},
	{ID: RecordMedicalLeaveSubsidized, Code: "21", Name: "Medical leave (EsSalud subsidy)|Descanso médico (subsidio EsSalud)"},
	{ID: RecordMaternityLeave, Code: "22", Name: "Maternity|Maternidad"},
	{ID: RecordUnpaidLeave, Code: "05", Name: "Unpaid leave|Licencia sin goce"},
}

func IsPeriodInputRecord(recordType int8) bool { return HasOption(InputTypeOptions, recordType) }
func IsAdvanceRecord(recordType int8) bool     { return HasOption(AdvanceTypeOptions, recordType) }
func IsTimeOffRecord(recordType int8) bool     { return HasOption(TimeOffTypeOptions, recordType) }

// IsUnpaidTimeOff: days the employer does not pay. EsSalud pays the subsidies, outside payroll.
func IsUnpaidTimeOff(recordType int8) bool {
	return recordType == RecordUnpaidLeave || recordType == RecordMedicalLeaveSubsidized || recordType == RecordMaternityLeave
}

// Employee record statuses, one numbering for every type. A cancelled record is Status 0.
const (
	RecordStatusPending int8 = 1 // a time off request waiting for approval
	RecordStatusActive  int8 = 2 // in force: the payroll reads it
	RecordStatusSettled int8 = 3 // consumed by the closed payslip PayslipID
)

type EmployeeRecord struct {
	db.TableStruct[EmployeeRecordTable, EmployeeRecord]
	CompanyID  int32 `cb:"1"`
	ID         int32 `cb:"2"`
	EmployeeID int32 `cb:"3"`
	Type       int8  `cb:"4"`
	// Period is the YYMM the payroll applies the record to; Year, derived from it, is the key the
	// date lists sync by (the time off page shows a year, the payroll pages filter its month).
	Period     int16   `cb:"5"`
	Year       int16   `cb:"6"`
	Date       int16   `cb:"7"` // UnixDay paid (advance) or granted (loan)
	FromDate   int16   `cb:"8"` // UnixDay
	ToDate     int16   `cb:"9"` // UnixDay
	Quantity   float32 `cb:"10"`
	Amount     int32   `cb:"11"` // cents
	ConceptID  int32   `cb:"12"`
	CashBankID int32   `cb:"13"`
	LoanID     int32   `cb:"14"` // the loan of an installment
	PayslipID  int32   `cb:"15"` // the closed payslip that settled it
	Note       string  `cb:"16"`
	ApprovedBy int32   `cb:"17"`
	Status     int8    `json:"ss" cb:"ss,18"`
	Updated    int32   `json:"upd" cb:"upd,19"`
	Version    int16   `json:"ver" cb:"ver,20"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,21"`
	UpdatedBy int32   `cb:"22"`
	Created   int32   `cb:"23"`
	CreatedBy int32   `cb:"24"`
}

type EmployeeRecordTable struct {
	db.TableStruct[EmployeeRecordTable, EmployeeRecord]
	CompanyID  db.Col[*EmployeeRecordTable, int32]
	ID         db.Col[*EmployeeRecordTable, int32]
	EmployeeID db.Col[*EmployeeRecordTable, int32]
	Type       db.Col[*EmployeeRecordTable, int8]
	Period     db.Col[*EmployeeRecordTable, int16]
	Year       db.Col[*EmployeeRecordTable, int16]
	Date       db.Col[*EmployeeRecordTable, int16]
	FromDate   db.Col[*EmployeeRecordTable, int16]
	ToDate     db.Col[*EmployeeRecordTable, int16]
	Quantity   db.Col[*EmployeeRecordTable, float32]
	Amount     db.Col[*EmployeeRecordTable, int32]
	ConceptID  db.Col[*EmployeeRecordTable, int32]
	CashBankID db.Col[*EmployeeRecordTable, int32]
	LoanID     db.Col[*EmployeeRecordTable, int32]
	PayslipID  db.Col[*EmployeeRecordTable, int32]
	Note       db.Col[*EmployeeRecordTable, string]
	ApprovedBy db.Col[*EmployeeRecordTable, int32]
	Status     db.Col[*EmployeeRecordTable, int8]
	Updated    db.Col[*EmployeeRecordTable, int32]
	Version    db.Col[*EmployeeRecordTable, int16]
	UpdatedBy  db.Col[*EmployeeRecordTable, int32]
	Created    db.Col[*EmployeeRecordTable, int32]
	CreatedBy  db.Col[*EmployeeRecordTable, int32]
}

func (e EmployeeRecordTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		// 85: new, so nothing keyed by the retired tables' IDs (76 loans, 79 time off, 80 inputs,
		// 84 advances) is read back as this one.
		ID:        85,
		Name:      "hr_employee_records",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1, 2, 3}},
			{Col: e.Year, Min: 2000, Max: 2099},
		},
		Indexes: []db.Index{
			// One employee's history (loans and their installments, the employee page).
			{Type: db.TypeDelta, Keys: db.Cols(e.EmployeeID, e.Status)},
			// One year of every employee (time off, and the month the payroll pages filter).
			{Type: db.TypeDelta, Keys: db.Cols(e.Year, e.Status)},
		},
	}
}

// SelfParse derives what the dates fix: a time off's period and days, an advance's period, and the
// year of every record.
func (e *EmployeeRecord) SelfParse() {
	switch {
	case IsTimeOffRecord(e.Type):
		e.Period, e.Quantity = core.PeriodOfDay(e.FromDate), float32(e.ToDate-e.FromDate+1)
	case IsAdvanceRecord(e.Type):
		e.Period = core.PeriodOfDay(e.ToDate)
	}
	e.Year = 2000 + e.Period/100
}

// DaysWithin counts the days of a time off inside [fromDay, toDay].
func (e EmployeeRecord) DaysWithin(fromDay, toDay int16) int {
	from, to := max(e.FromDate, fromDay), min(e.ToDate, toDay)
	if from > to {
		return 0
	}
	return int(to-from) + 1
}

// LoanInstallments splits a loan into equal monthly installments from firstPeriod on; the last one
// absorbs the rounding. They are saved with the loan, so one can be moved to a later month.
func LoanInstallments(loan EmployeeRecord, firstPeriod int16) []EmployeeRecord {
	count := int32(loan.Quantity)
	installments := []EmployeeRecord{}
	installmentAmount := loan.Amount / count
	period := firstPeriod
	for installmentIndex := int32(0); installmentIndex < count; installmentIndex++ {
		amount := installmentAmount
		if installmentIndex == count-1 {
			amount = loan.Amount - installmentAmount*(count-1)
		}
		installment := EmployeeRecord{CompanyID: loan.CompanyID, EmployeeID: loan.EmployeeID, Type: RecordLoanInstallment,
			Period: period, Amount: amount, LoanID: loan.ID, Status: RecordStatusActive}
		installment.SelfParse()
		installments = append(installments, installment)
		period = core.NextPeriod(period)
	}
	return installments
}
