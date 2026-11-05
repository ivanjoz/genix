package types

// A payroll concept is one line of a payslip. What the same amount costs, how it is taxed and where it
// is posted depends on the concept (plan/HR_PAYROLL_PLAN.md §4.1), so every concept carries its
// treatment. Legal concepts are fixed here and calculated by the engine; companies add their own
// (bonuses, movilidad, other deductions) in hr_concepts with the same flags.
//
// PLAME codes are the SUNAT table 22 codes the PLAME .rem file reports (see plame_codes.go).

// Concept kinds.
const (
	ConceptKindEarning   int8 = 1
	ConceptKindDeduction int8 = 2
	ConceptKindEmployer  int8 = 3 // employer contribution: cost of the company, not on the net pay
	ConceptKindProvision int8 = 4 // monthly accrual of a future benefit: accounting only
)

var ConceptKindOptions = []HROption{
	{ID: ConceptKindEarning, Name: "Earning|Ingreso"},
	{ID: ConceptKindDeduction, Name: "Deduction|Descuento"},
	{ID: ConceptKindEmployer, Name: "Employer contribution|Aporte del empleador"},
	{ID: ConceptKindProvision, Name: "Provision|Provisión"},
}

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

// LegalConcept IDs. They are persisted on payslip lines: never renumber.
const (
	LegalBasicSalary     int16 = 1
	LegalFamilyAllowance int16 = 2
	LegalOvertime25      int16 = 3
	LegalOvertime35      int16 = 4
	LegalVacationPay     int16 = 5
	LegalGratificacion   int16 = 6
	LegalGratiBonus      int16 = 7 // bonificación extraordinaria Ley 30334
	LegalCts             int16 = 8
	LegalProfitSharing   int16 = 9 // utilidades
	LegalIndemnity       int16 = 10

	LegalAfpFund       int16 = 20
	LegalAfpPremium    int16 = 21
	LegalAfpCommission int16 = 22
	LegalOnp           int16 = 23
	LegalIncomeTax5ta  int16 = 24
	LegalAdvancePaid   int16 = 25 // weekly / biweekly payments already made (§3.6)
	LegalLoanPayment   int16 = 26

	LegalEssalud int16 = 40

	LegalProvisionGratificacion int16 = 50
	LegalProvisionGratiBonus    int16 = 51
	LegalProvisionCts           int16 = 52
	LegalProvisionVacation      int16 = 53
)

type LegalConcept struct {
	ID                    int16  `cb:"1"`
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
		ComputableForBenefits: true, ExpenseAccountCode: 6211},
	{ID: LegalOvertime35, Name: "Overtime 35%|Horas extras 35%", Kind: ConceptKindEarning, PlameCode: "0106",
		IsRemunerative: true, AffectsEssalud: true, AffectsPension: true, IncomeTax: IncomeTaxOrdinary,
		ComputableForBenefits: true, ExpenseAccountCode: 6211},
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
	{ID: LegalLoanPayment, Name: "Loan installment|Cuota de préstamo", Kind: ConceptKindDeduction,
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
}

// FindLegalConcept returns the legal concept with this ID.
func FindLegalConcept(id int16) (LegalConcept, bool) {
	for _, concept := range LegalConcepts {
		if concept.ID == id {
			return concept, true
		}
	}
	return LegalConcept{}, false
}
