package types

import (
	"time"

	"app/core"
)

// The benefit runs of the general regime (plan §4): gratificación, CTS, settlement (liquidación),
// extraordinary payments and profit sharing. Pure like the monthly engine; each returns ok false
// when the employee has nothing to receive in the run, with a diagnostic saying why.
//
// Benefits are calculated on the computable remuneration (salary, asignación familiar and the
// variable earnings received in at least 3 of the reference months) and paid first against the
// provisions the monthly payslips accrued (ProvisionBalances).
//
// TODO(accountant): unjustified absences only reduce CTS through the worked days of the stored
// monthly payslips; periods before the first payroll in Genix count as fully worked.

// BenefitPayslipInput is everything one employee's benefit payslip depends on.
type BenefitPayslipInput struct {
	Run       PayrollRun
	Employee  Employee
	Contracts []ContractAgreement
	// History is every closed or paid payslip of the employee.
	History []Payslip
	// Records are the employee's active and settled records of every period (see MonthlyPayslipInput).
	Records  []EmployeeRecord
	Concepts map[int32]Concept
	Legal    LegalValues
	// ProfitShare is the employee's share of the profit sharing pool and CapExcess what the 18
	// salaries cap took from it (profit sharing run, see DistributeProfitSharing).
	ProfitShare int64
	CapExcess   int64
}

// Benefit diagnostics.
const (
	DiagnosticNothingToPay     = "HR-008"
	DiagnosticNetBelowLoans    = "HR-009"
	DiagnosticProfitSharingCap = "HR-011"
)

// GratificacionSemester is the first and last period of the gratificación semester a period
// belongs to: January–June (paid in July) or July–December (paid in December).
func GratificacionSemester(period int16) (int16, int16) {
	yearFirstPeriod := period - int16(PeriodMonth(period)) + 1
	if PeriodMonth(period) <= 6 {
		return yearFirstPeriod, yearFirstPeriod + 5
	}
	return yearFirstPeriod + 6, yearFirstPeriod + 11
}

// CtsSemester is the first and last period of the CTS semester a period belongs to: November–April
// (deposited in May) or May–October (deposited in November).
func CtsSemester(period int16) (int16, int16) {
	month, yearFirstPeriod := PeriodMonth(period), period-int16(PeriodMonth(period))+1
	switch {
	case month >= 5 && month <= 10:
		return yearFirstPeriod + 4, yearFirstPeriod + 9
	case month >= 11:
		return yearFirstPeriod + 10, yearFirstPeriod + 100 + 3
	default:
		return yearFirstPeriod - 100 + 10, yearFirstPeriod + 3
	}
}

// GratificacionRunSemester is the semester a gratificación run pays: July pays the one that ended
// in June, December the one ending with it.
func GratificacionRunSemester(runPeriod int16) (int16, int16) {
	if PeriodMonth(runPeriod) == 7 {
		return GratificacionSemester(runPeriod - 1)
	}
	return GratificacionSemester(runPeriod)
}

// CtsRunSemester is the semester a CTS deposit pays: the one that ended the month before.
func CtsRunSemester(runPeriod int16) (int16, int16) {
	return CtsSemester(core.PeriodOfDay(core.PeriodFirstDay(runPeriod) - 1))
}

// periodsBetween lists the periods from first to last, both included.
func periodsBetween(first, last int16) []int16 {
	periods := []int16{}
	for period := first; period <= last; period = core.NextPeriod(period) {
		periods = append(periods, period)
	}
	return periods
}

// fullMonthsEmployed counts the periods the employee was on the payroll the whole month.
func fullMonthsEmployed(employee Employee, periods []int16) int {
	months := 0
	for _, period := range periods {
		if EmployedDays(employee, period) == payrollMonthDays {
			months++
		}
	}
	return months
}

// computableDays sums the days of the periods that count for CTS and vacations: the paid days of
// the stored monthly payslip, or the days employed before payroll was kept in Genix.
func computableDays(employee Employee, periods []int16, history []Payslip) int {
	paidDaysByPeriod := map[int16]int{}
	for _, payslip := range history {
		paidDaysByPeriod[payslip.Period] = int(payslip.WorkedDays)
	}
	days := 0
	for _, period := range periods {
		if paidDays, found := paidDaysByPeriod[period]; found {
			days += paidDays
		} else {
			days += EmployedDays(employee, period)
		}
	}
	return days
}

// regularVariableEarnings is the variable part of the computable remuneration: each variable
// earning computable for benefits (overtime, commissions, regular bonuses) received in at least 3
// of the reference periods counts as its sum over 6 (TUO CTS art. 16, Ley 27735 reglamento art. 3).
func regularVariableEarnings(history []Payslip, periods []int16) int64 {
	inReference := map[int16]bool{}
	for _, period := range periods {
		inReference[period] = true
	}
	monthsReceived, totals := map[int32]map[int16]bool{}, map[int32]int64{}
	for _, payslip := range history {
		if !inReference[payslip.Period] {
			continue
		}
		for _, line := range payslip.Lines {
			isFixed := line.ConceptID == LegalBasicSalary || line.ConceptID == LegalFamilyAllowance
			if line.Kind != ConceptKindEarning || !line.ComputableForBenefits || isFixed {
				continue
			}
			key := line.ConceptID
			if monthsReceived[key] == nil {
				monthsReceived[key] = map[int16]bool{}
			}
			monthsReceived[key][payslip.Period] = true
			totals[key] += int64(line.Amount)
		}
	}
	var variable int64
	for key, months := range monthsReceived {
		if len(months) >= 3 {
			variable += totals[key] / 6
		}
	}
	return variable
}

// gratificacionReceived sums the gratificaciones paid in the periods (for the CTS sixth).
func gratificacionReceived(history []Payslip, periods []int16) int64 {
	inPeriods := map[int16]bool{}
	for _, period := range periods {
		inPeriods[period] = true
	}
	var total int64
	for _, payslip := range history {
		if !inPeriods[payslip.Period] {
			continue
		}
		for _, line := range payslip.Lines {
			if line.ConceptID == LegalGratificacion {
				total += int64(line.Amount)
			}
		}
	}
	return total
}

// benefitStart prepares the builder of a benefit payslip: the contract in force on onDay, and the
// minimum wage and tax unit of that day. ok is false when the payslip cannot be calculated.
func benefitStart(input BenefitPayslipInput, onDay int16) (*payslipBuilder, ContractAgreement, PayslipParameters, bool) {
	builder := newPayslipBuilder(input.Employee, input.Run.Period, input.Legal, onDay, input.History)
	contract, found := FindContractInForce(input.Contracts, onDay)
	if !found {
		builder.diagnose(DiagnosticNoContract, true, "No tiene un contrato vigente al %s.", formatDay(onDay))
		return builder, contract, PayslipParameters{}, false
	}
	builder.payslip.ContractID = contract.ID
	minimumWage, foundMinimumWage := input.Legal.ValueOn(LegalParamRMV, onDay)
	taxUnit, foundTaxUnit := input.Legal.ValueOn(LegalParamUIT, onDay)
	if !foundMinimumWage || !foundTaxUnit {
		builder.diagnose(DiagnosticLegalMissing, true, "Faltan la RMV o la UIT vigentes (Sistema → Parámetros laborales).")
		return builder, contract, PayslipParameters{}, false
	}
	return builder, contract, PayslipParameters{MinimumWage: minimumWage, TaxUnit: taxUnit, PensionSystem: contract.PensionSystem}, true
}

// computableRemuneration is salary + asignación familiar on onDay + the regular variable earnings
// of the reference periods. It adds no line: the benefit lines carry it as their base.
func computableRemuneration(input BenefitPayslipInput, builder *payslipBuilder, contract ContractAgreement, onDay int16, referencePeriods []int16, minimumWage int64) int64 {
	computable := int64(contract.PayAmount) + regularVariableEarnings(input.History, referencePeriods)
	if input.Employee.HasFamilyAllowanceOn(onDay) {
		computable += roundRate(minimumWage, builder.legalValue(LegalParamFamilyAllowanceRate))
	}
	return computable
}

// CalculateGratificacion pays the July or December gratificación (Ley 27735): a sixth of the
// computable remuneration per full month of the semester, plus the bonificación extraordinaria
// (Ley 30334). It is free of pension and EsSalud; its 5ta is withheld by the monthly payslip,
// whose projection reads it as paid income.
func CalculateGratificacion(input BenefitPayslipInput) (Payslip, bool) {
	firstPeriod, lastPeriod := GratificacionRunSemester(input.Run.Period)
	semesterLastDay := core.PeriodLastDay(lastPeriod)
	builder, contract, parameters, ok := benefitStart(input, semesterLastDay)
	if !ok {
		return builder.payslip, false
	}
	semester := periodsBetween(firstPeriod, lastPeriod)
	months := fullMonthsEmployed(input.Employee, semester)
	if months == 0 {
		builder.diagnose(DiagnosticNothingToPay, false, "No tiene meses completos en el semestre.")
		return builder.payslip, false
	}
	computable := computableRemuneration(input, builder, contract, semesterLastDay, semester, int64(parameters.MinimumWage))
	gratificacion := computable * int64(months) / 6
	builder.addBenefit(LegalGratificacion, computable, 0, float32(months), gratificacion)
	bonusRate := builder.gratiBonusRate(contract)
	builder.addBenefit(LegalGratiBonus, gratificacion, bonusRate, 0, roundRate(gratificacion, bonusRate))
	builder.payslip.WorkedDays = int8(min(months*payrollMonthDays, 127))
	return builder.finish(parameters), true
}

// CalculateCts pays the CTS of the semester ending the month before the deposit (DS 001-97-TR):
// the computable remuneration plus a sixth of the gratificación received in the semester, a
// twelfth per month and a 360th per day. Part-time workers have no CTS.
func CalculateCts(input BenefitPayslipInput) (Payslip, bool) {
	firstPeriod, lastPeriod := CtsRunSemester(input.Run.Period)
	semesterLastDay := core.PeriodLastDay(lastPeriod)
	builder, contract, parameters, ok := benefitStart(input, semesterLastDay)
	if !ok {
		return builder.payslip, false
	}
	if contract.DailyHours < partTimeDailyHoursLimit {
		builder.diagnose(DiagnosticNothingToPay, false, "Trabajador a tiempo parcial: no tiene CTS.")
		return builder.payslip, false
	}
	semester := periodsBetween(firstPeriod, lastPeriod)
	days := computableDays(input.Employee, semester, input.History)
	if days == 0 {
		builder.diagnose(DiagnosticNothingToPay, false, "No tiene días computables en el semestre.")
		return builder.payslip, false
	}
	computable := computableRemuneration(input, builder, contract, semesterLastDay, semester, int64(parameters.MinimumWage)) +
		gratificacionReceived(input.History, semester)/6
	builder.addBenefit(LegalCts, computable, 0, float32(days), computable*int64(days)/360)
	// The deposit goes to the worker's bank, so the line is informative for the net pay.
	for lineIndex := range builder.payslip.Lines {
		builder.payslip.Lines[lineIndex].OutsideNetPay = true
	}
	builder.payslip.WorkedDays = int8(min(days, 127))
	return builder.finish(parameters), true
}

// CalculateSettlement is the liquidación of an employee terminated in the run's period: the
// truncated gratificación and CTS of the semesters in progress, the vacation earned and not
// taken, the indemnity of an arbitrary dismissal, and the loans still owed. The last month's
// salary is paid by the monthly run.
func CalculateSettlement(input BenefitPayslipInput) (Payslip, bool) {
	terminationDay := input.Employee.TerminationDate
	builder, contract, parameters, ok := benefitStart(input, terminationDay)
	if !ok {
		return builder.payslip, false
	}
	terminationPeriod := core.PeriodOfDay(terminationDay)
	minimumWage := int64(parameters.MinimumWage)
	lastSixPeriods := periodsBetween(core.PeriodOfDay(core.PeriodFirstDay(terminationPeriod)-150), terminationPeriod)
	computable := computableRemuneration(input, builder, contract, terminationDay, lastSixPeriods, minimumWage)

	// Truncated gratificación: the full months of the semester in progress.
	gratificacionFirst, _ := GratificacionSemester(terminationPeriod)
	gratificacionMonths := fullMonthsEmployed(input.Employee, periodsBetween(gratificacionFirst, terminationPeriod))
	gratificacion := computable * int64(gratificacionMonths) / 6
	builder.addBenefit(LegalGratificacion, computable, 0, float32(gratificacionMonths), gratificacion)
	bonusRate := builder.gratiBonusRate(contract)
	builder.addBenefit(LegalGratiBonus, gratificacion, bonusRate, 0, roundRate(gratificacion, bonusRate))

	// Truncated CTS: the days of the CTS semester in progress.
	if contract.DailyHours >= partTimeDailyHoursLimit {
		ctsFirst, _ := CtsSemester(terminationPeriod)
		ctsPeriods := periodsBetween(ctsFirst, terminationPeriod)
		ctsDays := computableDays(input.Employee, ctsPeriods, input.History)
		ctsComputable := computable + gratificacionReceived(input.History, ctsPeriods)/6
		builder.addBenefit(LegalCts, ctsComputable, 0, float32(ctsDays), ctsComputable*int64(ctsDays)/360)
	}

	// Vacation earned over the whole service and not taken, truncated year included.
	servicePeriods := periodsBetween(core.PeriodOfDay(input.Employee.HireDate), terminationPeriod)
	serviceDays := computableDays(input.Employee, servicePeriods, input.History)
	pendingVacationDays := float64(serviceDays*annualVacationDays(contract))/360 - float64(VacationDaysTaken(input.Records))
	if pendingVacationDays > 0 {
		builder.addBenefit(LegalVacationPay, computable, 0, float32(pendingVacationDays),
			int64(float64(computable)*pendingVacationDays/payrollMonthDays+0.5))
	}

	// Indemnity for an arbitrary dismissal (TUO DL 728 arts. 38 and 76): 1.5 remunerations per year
	// of service, or per month left of a fixed-term contract, capped at 12 remunerations.
	if input.Employee.TerminationReason == TerminationDismissal {
		indemnity := computable * 3 / 2 * int64(serviceDays) / 360
		if contract.ContractType != ContractTypeIndefinite && contract.EndDate > terminationDay {
			indemnity = computable * 3 / 2 * int64(contract.EndDate-terminationDay) / payrollMonthDays
		}
		builder.addLegal(LegalIndemnity, computable, 0, 0, min(indemnity, 12*computable))
	}

	// Pension and 5ta on what is remunerative or taxed.
	pensionBase := builder.earningsWhere(func(treatment ConceptTreatment) bool { return treatment.AffectsPension }, input.Concepts)
	builder.addPensionDeductions(contract, pensionBase, input.Run.Period, &parameters)
	ordinaryIncome := builder.earningsWhere(func(treatment ConceptTreatment) bool { return treatment.IncomeTax != IncomeTaxNone }, input.Concepts)
	builder.addLegal(LegalIncomeTax5ta, ordinaryIncome, 0, 0, finalIncomeTax(input.Run.Period, input.History, int64(parameters.TaxUnit), ordinaryIncome))

	// The installments due after the termination month (whose own one the monthly payslip takes)
	// are recovered here.
	for _, installment := range input.Records {
		if installment.ConceptID == LegalLoanInstallment && installment.Period > terminationPeriod {
			builder.addRecordDeduction(LegalLoanPayment, installment)
		}
	}

	essaludBase := builder.earningsWhere(func(treatment ConceptTreatment) bool { return treatment.AffectsEssalud }, input.Concepts)
	if essaludBase > 0 {
		builder.addEssalud(contract, essaludBase, &parameters)
	}
	payslip := builder.finish(parameters)
	if payslip.TotalEarnings == 0 {
		builder.diagnose(DiagnosticNothingToPay, false, "No tiene beneficios pendientes.")
		return builder.payslip, false
	}
	if payslip.NetPay < 0 {
		builder.diagnose(DiagnosticNetBelowLoans, true, "Los descuentos (%s) superan la liquidación (%s): ajuste el préstamo.",
			formatCents(int64(payslip.TotalDeductions)), formatCents(int64(payslip.TotalEarnings)))
		payslip.Diagnostics = builder.payslip.Diagnostics
	}
	return payslip, true
}

// VacationDaysTaken sums the vacation days of the records (active and settled).
func VacationDaysTaken(records []EmployeeRecord) int {
	days := 0
	for _, leave := range records {
		if leave.ConceptID == LegalVacation {
			days += int(leave.Quantity)
		}
	}
	return days
}

// finalIncomeTax is the 5ta that closes the year at a termination: the tax on the year's income,
// this payslip's included, less what was already withheld.
func finalIncomeTax(period int16, history []Payslip, taxUnit int64, currentIncome int64) int64 {
	taxInput := buildIncomeTaxInput(period, history, taxUnit)
	var withheld int64
	for _, monthWithheld := range taxInput.PriorWithheld {
		withheld += monthWithheld
	}
	return max(AnnualIncomeTax(taxInput.PriorIncome+currentIncome, taxUnit)-withheld, 0)
}

// extraordinaryIncomeTax is the 5ta of income paid outside the monthly payslip: the tax it adds on
// top of the year's projection, withheld whole (Reglamento LIR art. 40 inc. d).
func extraordinaryIncomeTax(period int16, history []Payslip, taxUnit int64, fixedMonthly int64, gratiBonusRate int64, extraordinaryIncome int64) int64 {
	taxInput := buildIncomeTaxInput(period, history, taxUnit)
	taxInput.FixedMonthly, taxInput.GratiBonusRate = fixedMonthly, gratiBonusRate
	projected := projectedAnnualIncome(taxInput)
	return AnnualIncomeTax(projected+extraordinaryIncome, taxUnit) - AnnualIncomeTax(projected, taxUnit)
}

// CalculateExtraordinaryPayment pays the period's extraordinary payments of the employee (one-off
// bonuses paid on their own date): each company concept with its own contributions, and the 5ta
// they add on top of the year's projection.
func CalculateExtraordinaryPayment(input BenefitPayslipInput) (Payslip, bool) {
	payDay := input.Run.PayDate
	builder, contract, parameters, ok := benefitStart(input, payDay)
	if !ok {
		return builder.payslip, false
	}
	for _, periodInput := range separatePayslipRecords(input.Records, input.Run.Period, input.Concepts) {
		concept, found := input.Concepts[periodInput.ConceptID]
		if !found || concept.Status != StatusActive || concept.Kind != ConceptKindEarning {
			builder.diagnose(DiagnosticUnknownConcept, true, "El concepto %d no existe o no es un ingreso.", periodInput.ConceptID)
			continue
		}
		builder.addCompany(concept, int64(periodInput.Amount))
	}
	if len(builder.payslip.Lines) == 0 {
		builder.diagnose(DiagnosticNothingToPay, false, "No tiene pagos extraordinarios en el periodo.")
		return builder.payslip, false
	}
	pensionBase := builder.earningsWhere(func(treatment ConceptTreatment) bool { return treatment.AffectsPension }, input.Concepts)
	builder.addPensionDeductions(contract, pensionBase, input.Run.Period, &parameters)
	taxedIncome := builder.earningsWhere(func(treatment ConceptTreatment) bool { return treatment.IncomeTax != IncomeTaxNone }, input.Concepts)
	builder.addLegal(LegalIncomeTax5ta, taxedIncome, 0, 0, extraordinaryIncomeTax(input.Run.Period, input.History,
		int64(parameters.TaxUnit), int64(contract.PayAmount), builder.gratiBonusRate(contract), taxedIncome))
	if essaludBase := builder.earningsWhere(func(treatment ConceptTreatment) bool { return treatment.AffectsEssalud }, input.Concepts); essaludBase > 0 {
		builder.addEssalud(contract, essaludBase, &parameters)
	}
	return builder.finish(parameters), true
}

// ProfitShareBasis is what one worker's share of the profit sharing is distributed on.
type ProfitShareBasis struct {
	EmployeeID   int32
	Days         int64 // days worked in the fiscal year
	Remuneration int64 // remunerations received in the fiscal year, cents
	Cap          int64 // 18 monthly remunerations, cents
}

// DistributeProfitSharing splits the pool (DL 892 art. 2): half by days worked, half by
// remunerations received. A share above its cap is cut; the excess goes to FONDOEMPLEO.
func DistributeProfitSharing(pool int64, bases []ProfitShareBasis) (shares, capExcess map[int32]int64) {
	shares, capExcess = map[int32]int64{}, map[int32]int64{}
	var totalDays, totalRemuneration int64
	for _, basis := range bases {
		totalDays += basis.Days
		totalRemuneration += basis.Remuneration
	}
	for _, basis := range bases {
		var share int64
		if totalDays > 0 {
			share += pool / 2 * basis.Days / totalDays
		}
		if totalRemuneration > 0 {
			share += pool / 2 * basis.Remuneration / totalRemuneration
		}
		if share > basis.Cap {
			capExcess[basis.EmployeeID] = share - basis.Cap
			share = basis.Cap
		}
		shares[basis.EmployeeID] = share
	}
	return shares, capExcess
}

// ProfitShareBasisOf reads one worker's basis from his payslips of the fiscal year.
func ProfitShareBasisOf(employeeID int32, history []Payslip, fiscalYear int16, contracts []ContractAgreement, yearLastDay int16) ProfitShareBasis {
	basis := ProfitShareBasis{EmployeeID: employeeID}
	firstPeriod := (fiscalYear%100)*100 + 1
	for _, payslip := range history {
		if payslip.Period < firstPeriod || payslip.Period > firstPeriod+11 {
			continue
		}
		basis.Days += int64(payslip.WorkedDays)
		for _, line := range payslip.Lines {
			if line.Kind == ConceptKindEarning && line.IsRemunerative {
				basis.Remuneration += int64(line.Amount)
			}
		}
	}
	// The cap reads the last contract of the year (or of the employment, if it ended earlier).
	var lastContract ContractAgreement
	for _, contract := range contracts {
		if contract.Status == StatusActive && contract.ValidFrom <= yearLastDay && contract.ValidFrom >= lastContract.ValidFrom {
			lastContract = contract
		}
	}
	basis.Cap = 18 * int64(lastContract.PayAmount)
	return basis
}

// CalculateProfitSharing pays one worker's share of the profit sharing (the distribution is done
// by DistributeProfitSharing over all the workers of the fiscal year, former ones included). It is
// not remunerative; its 5ta is extraordinary income of the payment year.
func CalculateProfitSharing(input BenefitPayslipInput) (Payslip, bool) {
	builder := newPayslipBuilder(input.Employee, input.Run.Period, input.Legal, input.Run.PayDate, input.History)
	if input.ProfitShare <= 0 {
		builder.diagnose(DiagnosticNothingToPay, false, "No trabajó en el ejercicio.")
		return builder.payslip, false
	}
	taxUnit, found := input.Legal.ValueOn(LegalParamUIT, input.Run.PayDate)
	if !found {
		builder.diagnose(DiagnosticLegalMissing, true, "Falta la UIT vigente (Sistema → Parámetros laborales).")
		return builder.payslip, false
	}
	parameters := PayslipParameters{TaxUnit: taxUnit}
	// Posted to 622 in the payment period. The fiscal-year accrual (Dr 622 · Cr 413, plan §5.1 H6)
	// is a year-end adjustment of the accounting: the run only exists after the annual return.
	builder.addLegal(LegalProfitSharing, 0, 0, 0, input.ProfitShare)
	if input.CapExcess > 0 {
		builder.diagnose(DiagnosticProfitSharingCap, false, "La participación supera el tope de 18 remuneraciones: %s van al FONDOEMPLEO.",
			formatCents(input.CapExcess))
	}
	// A current worker's projection uses his salary; a former worker has no more ordinary income.
	var fixedMonthly int64
	if contract, inForce := FindContractInForce(input.Contracts, input.Run.PayDate); inForce && input.Employee.TerminationDate == 0 {
		fixedMonthly = int64(contract.PayAmount)
		builder.payslip.ContractID = contract.ID
	}
	builder.addLegal(LegalIncomeTax5ta, input.ProfitShare, 0, 0, extraordinaryIncomeTax(input.Run.Period, input.History,
		int64(taxUnit), fixedMonthly, builder.legalValue(LegalParamGratiBonusRate), input.ProfitShare))
	return builder.finish(parameters), true
}

// formatDay renders a UnixDay for a diagnostic message.
func formatDay(unixDay int16) string {
	return time.Unix(int64(unixDay)*86400, 0).UTC().Format("02-01-2006")
}
