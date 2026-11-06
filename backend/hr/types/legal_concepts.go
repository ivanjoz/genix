package types

// A concept is a kind of thing that happens to an employee at a point in time: a payslip line
// (salary, overtime, a bonus, AFP), or a record with no line of its own that the engine reads
// (an absence, a vacation, a loan, an advance). One ID space for all of them:
//
//   - 1..LegalConceptMaxID: legal (system) concepts, fixed here and calculated by the engine.
//   - above LegalConceptMaxID: company concepts (bonuses, movilidad, other deductions) in hr_concepts,
//     with the same treatment flags; the table's autoincrement starts after the legal range.
//
// What the same amount costs, how it is taxed and where it is posted depends on the concept
// (plan/HR_PAYROLL_PLAN.md §4.1), so every concept carries its treatment. PLAME codes are the SUNAT
// table 22 codes the PLAME .rem file reports (see plame_codes.go).

// LegalConceptMaxID is the last ID reserved to legal concepts; company concepts start after it.
const LegalConceptMaxID int32 = 200

// IsLegalConcept tells a legal concept ID from a company one.
func IsLegalConcept(conceptID int32) bool { return conceptID > 0 && conceptID <= LegalConceptMaxID }

// Concept kinds.
const (
	ConceptKindEarning   int8 = 1
	ConceptKindDeduction int8 = 2
	ConceptKindEmployer  int8 = 3 // employer contribution: cost of the company, not on the net pay
	ConceptKindProvision int8 = 4 // monthly accrual of a future benefit: accounting only
	// ConceptKindRecord is registered on the employee and adds no payslip line of its own: the engine
	// derives its effect (absences cut the worked days, an installment becomes the loan deduction).
	ConceptKindRecord int8 = 5
)

var ConceptKindOptions = []HROption{
	{ID: ConceptKindEarning, Name: "Earning|Ingreso"},
	{ID: ConceptKindDeduction, Name: "Deduction|Descuento"},
	{ID: ConceptKindEmployer, Name: "Employer contribution|Aporte del empleador"},
	{ID: ConceptKindProvision, Name: "Provision|Provisión"},
	{ID: ConceptKindRecord, Name: "Record|Registro"},
}

// Concept groups: the screen an employee record of the concept is registered on. A concept with no
// group is never registered by hand (the engine adds it). Company concepts are always ConceptGroupEntry.
const (
	ConceptGroupEntry   int8 = 1 // Employee Records → Registros
	ConceptGroupAdvance int8 = 2 // Payroll → Weekly payments
	ConceptGroupLoan    int8 = 3 // Employee Records → Loans
	ConceptGroupTimeOff int8 = 4 // Time off
)

// How an earning enters the 5th-category income tax (renta de quinta categoría).
const (
	IncomeTaxNone          int8 = 0 // exempt or not income (CTS, legal indemnity)
	IncomeTaxOrdinary      int8 = 1 // projected over the year
	IncomeTaxExtraordinary int8 = 2 // not projected: taxed in the month it is paid
)

var IncomeTaxOptions = []HROption{
	{ID: IncomeTaxNone, Name: "Not taxed|Inafecto"},
	{ID: IncomeTaxOrdinary, Name: "Ordinary (projected)|Ordinaria (proyectada)"},
	{ID: IncomeTaxExtraordinary, Name: "Extraordinary|Extraordinaria"},
}

// ConceptTreatment is what the engine and the accounting need from a concept, legal or company.
type ConceptTreatment struct {
	Kind           int8
	PlameCode      string
	IsRemunerative bool // reported as remuneration; base of the employer and worker contributions
	AffectsEssalud bool
	AffectsPension bool
	IncomeTax      int8
	// ComputableForBenefits: enters the CTS / gratificación / vacation base. Variable concepts only
	// count when regular (3 of the 6 months of the semester), which the benefit runs check.
	ComputableForBenefits bool
	ExpenseAccountCode    int32 // debit of an earning or employer contribution (PCGE)
	PayableAccountCode    int32 // credit of a deduction or employer contribution: who is owed
}

// LegalConcept IDs, contiguous and ordered by kind. They are persisted on payslip lines and employee
// records: a renumbering needs the HR data reseeded. New ones take the next free ID.
const (
	// Earnings
	LegalBasicSalary     int32 = 1
	LegalFamilyAllowance int32 = 2
	LegalOvertime25      int32 = 3 // registered by hand: Quantity hours (first two daily overtime hours)
	LegalOvertime35      int32 = 4 // registered by hand: Quantity hours
	LegalVacationPay     int32 = 5
	LegalGratificacion   int32 = 6
	LegalGratiBonus      int32 = 7 // bonificación extraordinaria Ley 30334
	LegalCts             int32 = 8
	LegalProfitSharing   int32 = 9 // utilidades
	LegalIndemnity       int32 = 10

	// Worker deductions
	LegalAfpFund       int32 = 11
	LegalAfpPremium    int32 = 12
	LegalAfpCommission int32 = 13
	LegalOnp           int32 = 14
	LegalIncomeTax5ta  int32 = 15
	LegalAdvancePaid   int32 = 16 // weekly / biweekly payments already made (§3.6)
	LegalLoanPayment   int32 = 17 // the installment a payslip recovers

	// Employer contributions
	LegalEssalud int32 = 18

	// Provisions
	LegalProvisionGratificacion int32 = 19
	LegalProvisionGratiBonus    int32 = 20
	LegalProvisionCts           int32 = 21
	LegalProvisionVacation      int32 = 22

	// Records (ConceptKindRecord): what the employee records hold besides overtime and company concepts.
	LegalAbsence int32 = 23 // Quantity: unpaid absence days, they reduce the worked days
	// Payments made before the payslip, which nets them as LegalAdvancePaid: Date paid, FromDate–ToDate
	// covered (one month), Amount, CashBankID it left from.
	LegalAdvanceWeekly   int32 = 24
	LegalAdvanceBiweekly int32 = 25
	LegalAdvanceOther    int32 = 26
	// LegalLoan is money lent (account 1411): Date granted, Amount, Quantity installments, Period of the
	// first installment, CashBankID it left from. Its installments are LegalLoanInstallment records
	// (LoanID, Period, Amount), created with it; the monthly payslip of their Period recovers them.
	LegalLoan            int32 = 27
	LegalLoanInstallment int32 = 28
	// Time off, both dates included (FromDate, ToDate; Quantity the calendar days). SuspensionCode is the
	// SUNAT "tipo de suspensión" the PLAME .snl file reports.
	// TODO(accountant): verify the codes against the current PLAME table 21.
	LegalVacation               int32 = 29
	LegalMedicalLeavePaid       int32 = 30 // incapacidad temporal, first 20 days of the year: paid by the employer
	LegalMedicalLeaveSubsidized int32 = 31 // incapacidad temporal subsidiada by EsSalud
	LegalMaternityLeave         int32 = 32 // subsidiada by EsSalud
	LegalUnpaidLeave            int32 = 33 // licencia sin goce de haber
)

type LegalConcept struct {
	ID                    int32  `cb:"1"`
	Name                  string `cb:"2"`
	Kind                  int8   `cb:"3"`
	PlameCode             string `cb:"4"`
	IsRemunerative        bool   `cb:"5"`
	AffectsEssalud        bool   `cb:"6"`
	AffectsPension        bool   `cb:"7"`
	IncomeTax             int8   `cb:"8"`
	ComputableForBenefits bool   `cb:"9"`
	ExpenseAccountCode    int32  `cb:"10"`
	PayableAccountCode    int32  `cb:"11"`
	Group                 int8   `cb:"12"` // ConceptGroup* screen that registers it; 0 = engine only
	SuspensionCode        string `cb:"13"` // time off: SUNAT tipo de suspensión (PLAME table 21)
}

func (concept LegalConcept) Treatment() ConceptTreatment {
	return ConceptTreatment{Kind: concept.Kind, PlameCode: concept.PlameCode, IsRemunerative: concept.IsRemunerative,
		AffectsEssalud: concept.AffectsEssalud, AffectsPension: concept.AffectsPension, IncomeTax: concept.IncomeTax,
		ComputableForBenefits: concept.ComputableForBenefits, ExpenseAccountCode: concept.ExpenseAccountCode,
		PayableAccountCode: concept.PayableAccountCode}
}

// LegalConcepts is the §4.1 treatment matrix in code; the frontend copy comes from //CATALOG:.
var LegalConcepts = []LegalConcept{
	// Earnings
	{ID: LegalBasicSalary, Name: "Basic salary|Remuneración básica", Kind: ConceptKindEarning, PlameCode: "0121",
		IsRemunerative: true, AffectsEssalud: true, AffectsPension: true, IncomeTax: IncomeTaxOrdinary,
		ComputableForBenefits: true, ExpenseAccountCode: 6211},
	{ID: LegalFamilyAllowance, Name: "Family allowance|Asignación familiar", Kind: ConceptKindEarning, PlameCode: "0201",
		IsRemunerative: true, AffectsEssalud: true, AffectsPension: true, IncomeTax: IncomeTaxOrdinary,
		ComputableForBenefits: true, ExpenseAccountCode: 6211},
	{ID: LegalOvertime25, Name: "Overtime 25%|Horas extras 25%", Kind: ConceptKindEarning, PlameCode: "0105",
		IsRemunerative: true, AffectsEssalud: true, AffectsPension: true, IncomeTax: IncomeTaxOrdinary,
		ComputableForBenefits: true, ExpenseAccountCode: 6211, Group: ConceptGroupEntry},
	{ID: LegalOvertime35, Name: "Overtime 35%|Horas extras 35%", Kind: ConceptKindEarning, PlameCode: "0106",
		IsRemunerative: true, AffectsEssalud: true, AffectsPension: true, IncomeTax: IncomeTaxOrdinary,
		ComputableForBenefits: true, ExpenseAccountCode: 6211, Group: ConceptGroupEntry},
	{ID: LegalVacationPay, Name: "Vacation pay|Remuneración vacacional", Kind: ConceptKindEarning, PlameCode: "0118",
		IsRemunerative: true, AffectsEssalud: true, AffectsPension: true, IncomeTax: IncomeTaxOrdinary,
		ExpenseAccountCode: 6215},
	{ID: LegalGratificacion, Name: "Gratificación|Gratificación", Kind: ConceptKindEarning, PlameCode: "0406",
		IsRemunerative: true, IncomeTax: IncomeTaxOrdinary, ExpenseAccountCode: 6214},
	{ID: LegalGratiBonus, Name: "Gratificación bonus (Ley 30334)|Bonificación extraordinaria (Ley 30334)",
		Kind: ConceptKindEarning, PlameCode: "0312", IncomeTax: IncomeTaxOrdinary, ExpenseAccountCode: 6214},
	{ID: LegalCts, Name: "CTS|CTS", Kind: ConceptKindEarning, PlameCode: "0904", ExpenseAccountCode: 629,
		// The deposit is owed to the worker's bank, not paid through the payslip (OutsideNetPay).
		PayableAccountCode: 415},
	{ID: LegalProfitSharing, Name: "Profit sharing|Participación en utilidades", Kind: ConceptKindEarning,
		PlameCode: "0910", IncomeTax: IncomeTaxExtraordinary, ExpenseAccountCode: 622},
	{ID: LegalIndemnity, Name: "Dismissal indemnity|Indemnización por despido", Kind: ConceptKindEarning,
		PlameCode: "0501", ExpenseAccountCode: 623},

	// Worker deductions
	{ID: LegalAfpFund, Name: "AFP fund contribution|Aporte obligatorio AFP", Kind: ConceptKindDeduction,
		PlameCode: "0608", PayableAccountCode: 417},
	{ID: LegalAfpPremium, Name: "AFP insurance premium|Prima de seguro AFP", Kind: ConceptKindDeduction,
		PlameCode: "0606", PayableAccountCode: 417},
	{ID: LegalAfpCommission, Name: "AFP commission|Comisión AFP", Kind: ConceptKindDeduction,
		PlameCode: "0601", PayableAccountCode: 417},
	{ID: LegalOnp, Name: "ONP|ONP (SNP)", Kind: ConceptKindDeduction, PlameCode: "0607", PayableAccountCode: 4032},
	{ID: LegalIncomeTax5ta, Name: "Income tax 5th category|Renta de quinta categoría", Kind: ConceptKindDeduction,
		PlameCode: "0605", PayableAccountCode: 40173},
	{ID: LegalAdvancePaid, Name: "Advances paid|Adelantos pagados", Kind: ConceptKindDeduction,
		PlameCode: "0701", PayableAccountCode: 1412},
	{ID: LegalLoanPayment, Name: "Loan deduction|Descuento de préstamo", Kind: ConceptKindDeduction,
		PlameCode: "0706", PayableAccountCode: 1411},

	// Employer contributions
	{ID: LegalEssalud, Name: "EsSalud|EsSalud", Kind: ConceptKindEmployer, PlameCode: "0804",
		ExpenseAccountCode: 6271, PayableAccountCode: 4031},

	// Provisions
	{ID: LegalProvisionGratificacion, Name: "Gratificación provision|Provisión de gratificación",
		Kind: ConceptKindProvision, ExpenseAccountCode: 6214, PayableAccountCode: 4114},
	{ID: LegalProvisionGratiBonus, Name: "Gratificación bonus provision|Provisión de bonificación extraordinaria",
		Kind: ConceptKindProvision, ExpenseAccountCode: 6214, PayableAccountCode: 4114},
	{ID: LegalProvisionCts, Name: "CTS provision|Provisión de CTS", Kind: ConceptKindProvision,
		ExpenseAccountCode: 629, PayableAccountCode: 415},
	{ID: LegalProvisionVacation, Name: "Vacation provision|Provisión de vacaciones", Kind: ConceptKindProvision,
		ExpenseAccountCode: 6215, PayableAccountCode: 4115},

	// Records
	{ID: LegalAbsence, Name: "Unpaid absence (days)|Inasistencia (días)", Kind: ConceptKindRecord, Group: ConceptGroupEntry},
	{ID: LegalAdvanceWeekly, Name: "Weekly advance|Adelanto semanal", Kind: ConceptKindRecord, Group: ConceptGroupAdvance},
	{ID: LegalAdvanceBiweekly, Name: "Biweekly advance|Adelanto quincenal", Kind: ConceptKindRecord, Group: ConceptGroupAdvance},
	{ID: LegalAdvanceOther, Name: "Other advance|Adelanto otro", Kind: ConceptKindRecord, Group: ConceptGroupAdvance},
	{ID: LegalLoan, Name: "Loan|Préstamo", Kind: ConceptKindRecord, Group: ConceptGroupLoan},
	{ID: LegalLoanInstallment, Name: "Loan installment|Cuota de préstamo", Kind: ConceptKindRecord, Group: ConceptGroupLoan},
	{ID: LegalVacation, Name: "Vacation|Vacaciones", Kind: ConceptKindRecord, Group: ConceptGroupTimeOff, SuspensionCode: "23"},
	{ID: LegalMedicalLeavePaid, Name: "Medical leave (paid by the company)|Descanso médico (pagado por la empresa)",
		Kind: ConceptKindRecord, Group: ConceptGroupTimeOff, SuspensionCode: "20"},
	{ID: LegalMedicalLeaveSubsidized, Name: "Medical leave (EsSalud subsidy)|Descanso médico (subsidio EsSalud)",
		Kind: ConceptKindRecord, Group: ConceptGroupTimeOff, SuspensionCode: "21"},
	{ID: LegalMaternityLeave, Name: "Maternity|Maternidad", Kind: ConceptKindRecord, Group: ConceptGroupTimeOff, SuspensionCode: "22"},
	{ID: LegalUnpaidLeave, Name: "Unpaid leave|Licencia sin goce", Kind: ConceptKindRecord, Group: ConceptGroupTimeOff, SuspensionCode: "05"},
}

// FindLegalConcept returns the legal concept with this ID.
func FindLegalConcept(id int32) (LegalConcept, bool) {
	for _, concept := range LegalConcepts {
		if concept.ID == id {
			return concept, true
		}
	}
	return LegalConcept{}, false
}

// legalConceptGroup is the screen group of a legal concept; 0 for company concepts and engine-only ones.
func legalConceptGroup(conceptID int32) int8 {
	concept, _ := FindLegalConcept(conceptID)
	return concept.Group
}

// IsEntryRecord: a record of the Registros screen, a company concept or a legal one registered by
// hand (overtime, absences).
func IsEntryRecord(conceptID int32) bool {
	return conceptID > LegalConceptMaxID || legalConceptGroup(conceptID) == ConceptGroupEntry
}
func IsAdvanceRecord(conceptID int32) bool {
	return legalConceptGroup(conceptID) == ConceptGroupAdvance
}
func IsTimeOffRecord(conceptID int32) bool {
	return legalConceptGroup(conceptID) == ConceptGroupTimeOff
}

// IsUnpaidTimeOff: days the employer does not pay. EsSalud pays the subsidies, outside payroll.
func IsUnpaidTimeOff(conceptID int32) bool {
	return conceptID == LegalUnpaidLeave || conceptID == LegalMedicalLeaveSubsidized || conceptID == LegalMaternityLeave
}
