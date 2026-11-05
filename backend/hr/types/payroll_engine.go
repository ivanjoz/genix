package types

import (
	"fmt"
	"time"

	"app/core"
)

// The monthly payroll engine, general regime (plan §4). Pure: no DB, no clock. The handler loads
// the inputs, calls CalculateMonthlyPayslip, and stores the payslip it returns.
//
// Fixed order: earnings → bases → worker deductions → employer contributions → provisions.
// Money is cents (int64 while calculating), rates are basis points; the month is 30 days.

const payrollMonthDays = 30

// Workers under 4 hours a day are part-time: no CTS, and only the vacation of Convenio OIT 52.
const partTimeDailyHoursLimit = 4

// MonthlyPayslipInput is everything one employee's monthly payslip depends on.
type MonthlyPayslipInput struct {
	Period    int16
	Employee  Employee
	Contracts []Contract // every version of the employee; the engine picks the one in force
	// Records are the employee's active and settled records of every period (pending and cancelled
	// ones never reach the engine): it reads the period's novedades, advances and loan installments,
	// and the time off touching the month. Settled ones matter for the history: the vacation days
	// already taken, the paid medical leave days of the year.
	Records []EmployeeRecord
	// RecurringConcepts and Concepts: the employee's recurring items and the company's concepts.
	RecurringConcepts []EmployeeConcept
	Concepts          map[int32]Concept
	// History is every closed or paid payslip of the employee, any run type: the 5ta reads the
	// year's income and withholdings, the vacation pay the provision left.
	History []Payslip
	Legal   LegalValues
}

// PayslipDiagnostic codes.
const (
	DiagnosticNoContract      = "HR-001"
	DiagnosticNoPension       = "HR-002"
	DiagnosticAfpRatesMissing = "HR-003"
	DiagnosticLegalMissing    = "HR-004"
	DiagnosticAdvancesExceed  = "HR-005"
	DiagnosticBelowMinimum    = "HR-006"
	DiagnosticUnknownConcept  = "HR-007"
)

// FindContractInForce returns the contract version in force on a day: the latest ValidFrom not
// after it, among the versions whose contract covers the day.
func FindContractInForce(contracts []Contract, unixDay int16) (Contract, bool) {
	var found Contract
	for _, contract := range contracts {
		if contract.Status != StatusActive || contract.ValidFrom > unixDay || contract.StartDate > unixDay {
			continue
		}
		if contract.EndDate > 0 && contract.EndDate < unixDay {
			continue
		}
		if found.ID == 0 || contract.ValidFrom > found.ValidFrom {
			found = contract
		}
	}
	return found, found.ID != 0
}

// EmployedDays are the days of the period the employee was on the payroll, on the 30-day month: a
// whole month counts 30 whatever its length.
func EmployedDays(employee Employee, period int16) int {
	firstDay, lastDay := core.PeriodFirstDay(period), core.PeriodLastDay(period)
	from, to := max(employee.HireDate, firstDay), lastDay
	if employee.TerminationDate > 0 {
		to = min(employee.TerminationDate, lastDay)
	}
	if from > to {
		return 0
	}
	if from == firstDay && to == lastDay {
		return payrollMonthDays
	}
	return min(int(to-from)+1, payrollMonthDays)
}

// CompletedYearsOn is the age in completed years on a day, for UnixDays.
func CompletedYearsOn(birthDate, unixDay int16) int {
	birth := time.Unix(int64(birthDate)*86400, 0).UTC()
	day := time.Unix(int64(unixDay)*86400, 0).UTC()
	years := day.Year() - birth.Year()
	if day.Month() < birth.Month() || (day.Month() == birth.Month() && day.Day() < birth.Day()) {
		years--
	}
	return years
}

// periodRecords are the records of a period whose type matches.
func periodRecords(records []EmployeeRecord, period int16, matches func(recordType int8) bool) []EmployeeRecord {
	matching := []EmployeeRecord{}
	for _, record := range records {
		if record.Period == period && matches(record.Type) {
			matching = append(matching, record)
		}
	}
	return matching
}

func isRecordType(recordType int8) func(int8) bool {
	return func(other int8) bool { return other == recordType }
}

// payslipBuilder accumulates lines and diagnostics while the engine runs. legal and legalDay are
// the legal values and the day they are read on.
type payslipBuilder struct {
	payslip  Payslip
	legal    LegalValues
	legalDay int16
	// provisionBalances: what is left of each provision account for this employee (see
	// ProvisionBalances), consumed as benefits are paid against it.
	provisionBalances map[int32]int64
}

// legalValue reads a legal parameter in force on the builder's day, diagnosing a missing one.
func (builder *payslipBuilder) legalValue(key int8) int64 {
	value, found := builder.legal.ValueOn(key, builder.legalDay)
	if !found {
		builder.diagnose(DiagnosticLegalMissing, true, "Falta el parámetro laboral %d vigente en el periodo.", key)
	}
	return int64(value)
}

// benefitProvisionAccount: the provision a benefit is paid against (plan §5.1 H3–H5).
var benefitProvisionAccount = map[int16]int32{
	LegalGratificacion: 4114,
	LegalGratiBonus:    4114,
	LegalCts:           415,
	LegalVacationPay:   4115,
}

// addBenefit adds a benefit line charged first to what is left of its provision.
func (builder *payslipBuilder) addBenefit(conceptID int16, base, rate int64, quantity float32, amount int64) {
	if amount <= 0 {
		return
	}
	builder.addLegal(conceptID, base, rate, quantity, amount)
	account := benefitProvisionAccount[conceptID]
	provisioned := min(amount, max(builder.provisionBalances[account], 0))
	line := &builder.payslip.Lines[len(builder.payslip.Lines)-1]
	line.ProvisionAccountCode, line.ProvisionedAmount = account, int32(provisioned)
	builder.provisionBalances[account] -= provisioned
}

// ProvisionBalances is what each provision account still holds for an employee: the provisions of
// his closed payslips less what the benefits paid since were charged to them.
func ProvisionBalances(history []Payslip) map[int32]int64 {
	balances := map[int32]int64{}
	for _, payslip := range history {
		for _, line := range payslip.Lines {
			if line.Kind == ConceptKindProvision {
				balances[line.PayableAccountCode] += int64(line.Amount)
			}
			if line.ProvisionAccountCode > 0 {
				balances[line.ProvisionAccountCode] -= int64(line.ProvisionedAmount)
			}
		}
	}
	return balances
}

// addPensionDeductions withholds the worker's pension contribution: ONP, or the AFP fund, the
// premium (capped by the maximum insurable remuneration) and the flow commission.
func (builder *payslipBuilder) addPensionDeductions(contract Contract, pensionBase int64, period int16, parameters *PayslipParameters) {
	parameters.PensionSystem = contract.PensionSystem
	if pensionBase <= 0 {
		return
	}
	switch {
	case contract.PensionSystem == PensionSystemONP:
		onpRate := builder.legalValue(LegalParamOnpRate)
		parameters.OnpRate = int32(onpRate)
		builder.addLegal(LegalOnp, pensionBase, onpRate, 0, roundRate(pensionBase, onpRate))
	case IsAfp(contract.PensionSystem):
		afpRate, found := builder.legal.AfpRateFor(contract.PensionSystem, period)
		if !found {
			builder.diagnose(DiagnosticAfpRatesMissing, true, "Faltan las tasas de la AFP del periodo %d.", period)
			return
		}
		maxInsurable := builder.legalValue(LegalParamMaxInsurableSalary)
		parameters.AfpFundRate, parameters.AfpPremiumRate, parameters.MaxInsurable = afpRate.FundRate, afpRate.PremiumRate, int32(maxInsurable)
		builder.addLegal(LegalAfpFund, pensionBase, int64(afpRate.FundRate), 0, roundRate(pensionBase, int64(afpRate.FundRate)))
		premiumBase := min(pensionBase, maxInsurable)
		builder.addLegal(LegalAfpPremium, premiumBase, int64(afpRate.PremiumRate), 0, roundRate(premiumBase, int64(afpRate.PremiumRate)))
		// A mixed commission is charged on the fund balance, not through payroll.
		if contract.AfpCommissionType == AfpCommissionFlow {
			parameters.AfpCommissionRate = afpRate.FlowCommission
			builder.addLegal(LegalAfpCommission, pensionBase, int64(afpRate.FlowCommission), 0, roundRate(pensionBase, int64(afpRate.FlowCommission)))
		}
	default:
		builder.diagnose(DiagnosticNoPension, true, "El contrato no tiene sistema de pensiones.")
	}
}

// addEssalud adds the employer's EsSalud on its base; with an EPS part of it goes to the EPS,
// which invoices it apart.
func (builder *payslipBuilder) addEssalud(contract Contract, essaludBase int64, parameters *PayslipParameters) {
	essaludRate := builder.legalValue(LegalParamEssaludRate)
	if contract.HasEPS {
		essaludRate = builder.legalValue(LegalParamEssaludEpsRate)
	}
	parameters.EssaludRate = int32(essaludRate)
	builder.addLegal(LegalEssalud, essaludBase, essaludRate, 0, roundRate(essaludBase, essaludRate))
}

// gratiBonusRate is the bonificación extraordinaria rate: 9 %, or 6.75 % with an EPS.
func (builder *payslipBuilder) gratiBonusRate(contract Contract) int64 {
	if contract.HasEPS {
		return builder.legalValue(LegalParamGratiBonusEpsRate)
	}
	return builder.legalValue(LegalParamGratiBonusRate)
}

// finish computes the payslip totals.
func (builder *payslipBuilder) finish(parameters PayslipParameters) Payslip {
	builder.payslip.Parameters = parameters
	builder.payslip.SumTotals()
	return builder.payslip
}

// SumTotals computes the payslip totals from its lines.
func (payslip *Payslip) SumTotals() {
	earnings := sumLines(payslip.Lines, ConceptKindEarning)
	deductions := sumLines(payslip.Lines, ConceptKindDeduction)
	payslip.TotalEarnings, payslip.TotalDeductions = int32(earnings), int32(deductions)
	payslip.NetPay = int32(earnings - deductions - sumOutsideNetPay(payslip.Lines))
	payslip.EmployerCost = int32(earnings + sumLines(payslip.Lines, ConceptKindEmployer))
}

func (builder *payslipBuilder) addLegal(conceptID int16, base, rate int64, quantity float32, amount int64) {
	if amount <= 0 {
		return
	}
	concept, _ := FindLegalConcept(conceptID)
	builder.addLine(PayslipLine{LegalConceptID: conceptID}, concept.Treatment(), base, rate, quantity, amount)
}

func (builder *payslipBuilder) addCompany(concept Concept, amount int64) {
	if amount <= 0 {
		return
	}
	builder.addLine(PayslipLine{ConceptID: concept.ID}, concept.Treatment(), 0, 0, 0, amount)
}

func (builder *payslipBuilder) addLine(line PayslipLine, treatment ConceptTreatment, base, rate int64, quantity float32, amount int64) {
	line.Kind, line.Base, line.Rate, line.Quantity, line.Amount = treatment.Kind, int32(base), int32(rate), quantity, int32(amount)
	line.PlameCode, line.IsRemunerative, line.IncomeTax = treatment.PlameCode, treatment.IsRemunerative, treatment.IncomeTax
	line.ComputableForBenefits = treatment.ComputableForBenefits
	line.AccountCode, line.PayableAccountCode = treatment.ExpenseAccountCode, treatment.PayableAccountCode
	builder.payslip.Lines = append(builder.payslip.Lines, line)
}

func (builder *payslipBuilder) diagnose(code string, blocking bool, message string, args ...any) {
	builder.payslip.Diagnostics = append(builder.payslip.Diagnostics,
		PayslipDiagnostic{Code: code, Message: fmt.Sprintf(message, args...), Blocking: blocking})
}

// earningsWhere sums the earnings lines matching a treatment predicate.
func (builder *payslipBuilder) earningsWhere(matches func(treatment ConceptTreatment) bool, concepts map[int32]Concept) int64 {
	var total int64
	for _, line := range builder.payslip.Lines {
		if line.Kind == ConceptKindEarning && matches(lineTreatment(line, concepts)) {
			total += int64(line.Amount)
		}
	}
	return total
}

// lineTreatment is the treatment of the concept a line was calculated from.
func lineTreatment(line PayslipLine, concepts map[int32]Concept) ConceptTreatment {
	if line.LegalConceptID > 0 {
		concept, _ := FindLegalConcept(line.LegalConceptID)
		return concept.Treatment()
	}
	return concepts[line.ConceptID].Treatment()
}

// CalculateMonthlyPayslip runs the monthly payroll of one employee. ok is false when there is
// nothing to pay (no contract in the period); the diagnostics then say why.
func CalculateMonthlyPayslip(input MonthlyPayslipInput) (payslip Payslip, ok bool) {
	lastDay := core.PeriodLastDay(input.Period)
	builder := newPayslipBuilder(input.Employee, input.Period, input.Legal, lastDay, input.History)

	// The contract in force on the last employed day of the month.
	employedUntil := lastDay
	if input.Employee.TerminationDate > 0 && input.Employee.TerminationDate < lastDay {
		employedUntil = input.Employee.TerminationDate
	}
	contract, found := FindContractInForce(input.Contracts, employedUntil)
	if !found {
		builder.diagnose(DiagnosticNoContract, true, "No tiene un contrato vigente en el periodo.")
		return builder.payslip, false
	}
	builder.payslip.ContractID = contract.ID

	minimumWage, foundMinimumWage := input.Legal.ValueOn(LegalParamRMV, lastDay)
	taxUnit, foundTaxUnit := input.Legal.ValueOn(LegalParamUIT, lastDay)
	if !foundMinimumWage || !foundTaxUnit {
		builder.diagnose(DiagnosticLegalMissing, true, "Faltan la RMV o la UIT vigentes en el periodo (Sistema → Parámetros laborales).")
		return builder.payslip, true
	}
	parameters := PayslipParameters{MinimumWage: minimumWage, TaxUnit: taxUnit}

	// 1. Earnings. Vacation days are paid as vacation pay instead of salary; unpaid absences and
	// unpaid or subsidized leave are not paid at all.
	absenceDays, overtime25Hours, overtime35Hours := 0, float32(0), float32(0)
	periodInputs := periodRecords(input.Records, input.Period, IsPeriodInputRecord)
	for _, periodInput := range periodInputs {
		switch periodInput.Type {
		case RecordAbsenceDays:
			absenceDays += int(periodInput.Quantity)
		case RecordOvertime25:
			overtime25Hours += periodInput.Quantity
		case RecordOvertime35:
			overtime35Hours += periodInput.Quantity
		}
	}
	employedDays := EmployedDays(input.Employee, input.Period)
	vacationDays := 0
	for _, timeOff := range input.Records {
		if !IsTimeOffRecord(timeOff.Type) {
			continue
		}
		days := timeOff.DaysWithin(core.PeriodFirstDay(input.Period), employedUntil)
		switch {
		case timeOff.Type == RecordVacation:
			vacationDays += days
		case IsUnpaidTimeOff(timeOff.Type):
			absenceDays += days
		}
	}
	vacationDays = min(vacationDays, employedDays)
	salaryDays := max(employedDays-absenceDays-vacationDays, 0)
	paidDays := salaryDays + vacationDays
	builder.payslip.WorkedDays = int8(paidDays)
	salary := int64(contract.MonthlySalary)
	builder.addLegal(LegalBasicSalary, salary, 0, float32(salaryDays), salary*int64(salaryDays)/payrollMonthDays)
	builder.addBenefit(LegalVacationPay, salary, 0, float32(vacationDays), salary*int64(vacationDays)/payrollMonthDays)

	familyAllowance := builder.familyAllowance(input.Employee, lastDay, int64(minimumWage))

	// Overtime is paid on the hourly value of the ordinary remuneration (salary + asignación).
	if overtime25Hours > 0 || overtime35Hours > 0 {
		dailyHours := int64(max(contract.DailyHours, 1))
		hourlyValue := float64(salary+familyAllowance) / payrollMonthDays / float64(dailyHours)
		builder.addLegal(LegalOvertime25, int64(hourlyValue), 12500, overtime25Hours, int64(hourlyValue*1.25*float64(overtime25Hours)+0.5))
		builder.addLegal(LegalOvertime35, int64(hourlyValue), 13500, overtime35Hours, int64(hourlyValue*1.35*float64(overtime35Hours)+0.5))
	}

	// Company concepts: recurring ones, then this month's amounts.
	companyConcept := func(conceptID int32) (Concept, bool) {
		concept, found := input.Concepts[conceptID]
		if !found || concept.Status != StatusActive {
			builder.diagnose(DiagnosticUnknownConcept, true, "El concepto %d no existe o fue eliminado.", conceptID)
		}
		return concept, found && concept.Status == StatusActive
	}
	companyDeductions := []PayslipLine{}
	addCompanyAmount := func(conceptID int32, amount int64) {
		concept, found := companyConcept(conceptID)
		if !found {
			return
		}
		if concept.Kind == ConceptKindEarning {
			builder.addCompany(concept, amount)
		} else {
			companyDeductions = append(companyDeductions, PayslipLine{ConceptID: concept.ID, Amount: int32(amount)})
		}
	}
	for _, recurring := range input.RecurringConcepts {
		if recurring.AppliesTo(input.Period) {
			addCompanyAmount(recurring.ConceptID, int64(recurring.Amount))
		}
	}
	for _, periodInput := range periodInputs {
		if periodInput.Type == RecordConceptAmount {
			addCompanyAmount(periodInput.ConceptID, int64(periodInput.Amount))
		}
	}

	// 2. Bases.
	pensionBase := builder.earningsWhere(func(treatment ConceptTreatment) bool { return treatment.AffectsPension }, input.Concepts)
	essaludBase := builder.earningsWhere(func(treatment ConceptTreatment) bool { return treatment.AffectsEssalud }, input.Concepts)
	ordinaryIncome := builder.earningsWhere(func(treatment ConceptTreatment) bool {
		return treatment.IncomeTax == IncomeTaxOrdinary
	}, input.Concepts)
	extraordinaryIncome := builder.earningsWhere(func(treatment ConceptTreatment) bool {
		return treatment.IncomeTax == IncomeTaxExtraordinary
	}, input.Concepts)

	isPartTime := contract.DailyHours < partTimeDailyHoursLimit
	isFullMonthFullTime := !isPartTime && employedDays == payrollMonthDays
	if isFullMonthFullTime && salary < int64(minimumWage) {
		builder.diagnose(DiagnosticBelowMinimum, false, "La remuneración (%s) es menor a la RMV (%s).",
			formatCents(salary), formatCents(int64(minimumWage)))
	}

	// 3. Worker deductions: pension, 5ta, loans, advances, company deductions.
	builder.addPensionDeductions(contract, pensionBase, input.Period, &parameters)

	gratiBonusRate := builder.gratiBonusRate(contract)
	taxInput := buildIncomeTaxInput(input.Period, input.History, int64(taxUnit))
	taxInput.CurrentOrdinary, taxInput.CurrentExtraordinary = ordinaryIncome, extraordinaryIncome
	taxInput.FixedMonthly, taxInput.GratiBonusRate = salary+familyAllowance, gratiBonusRate
	builder.addLegal(LegalIncomeTax5ta, ordinaryIncome+extraordinaryIncome, 0, 0, MonthlyIncomeTax(taxInput))

	for _, installment := range periodRecords(input.Records, input.Period, isRecordType(RecordLoanInstallment)) {
		builder.addRecordDeduction(LegalLoanPayment, installment)
	}
	for _, deduction := range companyDeductions {
		builder.addCompany(input.Concepts[deduction.ConceptID], int64(deduction.Amount))
	}

	// Advances come last: they settle what is left of the net pay.
	netBeforeAdvances := sumLines(builder.payslip.Lines, ConceptKindEarning) - sumLines(builder.payslip.Lines, ConceptKindDeduction)
	var advancesTotal int64
	for _, advance := range periodRecords(input.Records, input.Period, IsAdvanceRecord) {
		builder.addRecordDeduction(LegalAdvancePaid, advance)
		advancesTotal += int64(advance.Amount)
	}
	if advancesTotal > netBeforeAdvances {
		builder.diagnose(DiagnosticAdvancesExceed, true, "Los adelantos (%s) superan el neto del mes (%s): reduzca un adelanto o conviértalo en préstamo.",
			formatCents(advancesTotal), formatCents(netBeforeAdvances))
	}

	// 4. Employer contributions: EsSalud on at least the RMV for a full month of a full-time worker.
	if isFullMonthFullTime {
		essaludBase = max(essaludBase, int64(minimumWage))
	}
	builder.addEssalud(contract, essaludBase, &parameters)

	// 5. Provisions, on the computable remuneration of the days paid.
	computable := (salary + familyAllowance) * int64(paidDays) / payrollMonthDays
	gratificacionProvision := computable / 6
	builder.addLegal(LegalProvisionGratificacion, computable, 0, 0, gratificacionProvision)
	builder.addLegal(LegalProvisionGratiBonus, gratificacionProvision, gratiBonusRate, 0, roundRate(gratificacionProvision, gratiBonusRate))
	if !isPartTime {
		builder.addLegal(LegalProvisionCts, computable+gratificacionProvision, 0, 0, (computable+gratificacionProvision)/12)
	}
	builder.addLegal(LegalProvisionVacation, computable, 0, 0, computable*int64(annualVacationDays(contract))/payrollMonthDays/12)
	return builder.finish(parameters), true
}

func newPayslipBuilder(employee Employee, period int16, legal LegalValues, legalDay int16, history []Payslip) *payslipBuilder {
	builder := &payslipBuilder{legal: legal, legalDay: legalDay, provisionBalances: ProvisionBalances(history)}
	builder.payslip.EmployeeID, builder.payslip.Period = employee.ID, period
	builder.payslip.DepartmentID = employee.DepartmentID
	return builder
}

// addRecordDeduction deducts a record's amount (an advance, a loan installment) and points the line
// at it, so closing the run settles the record.
func (builder *payslipBuilder) addRecordDeduction(conceptID int16, record EmployeeRecord) {
	builder.addLegal(conceptID, 0, 0, 0, int64(record.Amount))
	builder.payslip.Lines[len(builder.payslip.Lines)-1].SourceID = record.ID
}

// familyAllowance adds the asignación familiar when a child gives the right to it on the day.
func (builder *payslipBuilder) familyAllowance(employee Employee, onDay int16, minimumWage int64) int64 {
	if !employee.HasFamilyAllowanceOn(onDay) {
		return 0
	}
	rate := builder.legalValue(LegalParamFamilyAllowanceRate)
	amount := roundRate(minimumWage, rate)
	builder.addLegal(LegalFamilyAllowance, minimumWage, rate, 0, amount)
	return amount
}

// annualVacationDays: 30 days a year, or the 6 of Convenio OIT 52 for a part-time worker.
func annualVacationDays(contract Contract) int {
	if contract.DailyHours < partTimeDailyHoursLimit {
		return 6
	}
	return payrollMonthDays
}

// buildIncomeTaxInput reads the year's earlier payslips (the closed ones of the period's year in
// the history) for the 5ta projection; the caller sets the current month's income.
func buildIncomeTaxInput(period int16, history []Payslip, taxUnit int64) IncomeTaxInput {
	month := PeriodMonth(period)
	taxInput := IncomeTaxInput{Month: month, TaxUnit: taxUnit}
	gratificacionPaid := map[int]bool{}
	yearFirstPeriod := period - int16(month) + 1
	for _, payslip := range history {
		if payslip.Period < yearFirstPeriod || payslip.Period > period {
			continue
		}
		payslipMonth := PeriodMonth(payslip.Period)
		for _, line := range payslip.Lines {
			switch {
			case line.Kind == ConceptKindEarning && line.IncomeTax != IncomeTaxNone:
				taxInput.PriorIncome += int64(line.Amount)
			case line.LegalConceptID == LegalIncomeTax5ta:
				taxInput.PriorWithheld[payslipMonth] += int64(line.Amount)
			}
			if line.LegalConceptID == LegalGratificacion {
				gratificacionPaid[payslipMonth] = true
			}
		}
	}
	for _, gratificacionMonth := range []int{7, 12} {
		if gratificacionMonth >= month && !gratificacionPaid[gratificacionMonth] {
			taxInput.GratificacionsToProject++
		}
	}
	return taxInput
}

func sumLines(lines []PayslipLine, kind int8) int64 {
	var total int64
	for _, line := range lines {
		if line.Kind == kind {
			total += int64(line.Amount)
		}
	}
	return total
}

// sumOutsideNetPay is what the payslip earns but does not pay the worker through it (the CTS).
func sumOutsideNetPay(lines []PayslipLine) int64 {
	var total int64
	for _, line := range lines {
		if line.OutsideNetPay {
			total += int64(line.Amount)
		}
	}
	return total
}

// formatCents renders cents as soles for a diagnostic message: 123456 → "S/ 1234.56".
func formatCents(cents int64) string {
	return fmt.Sprintf("S/ %d.%02d", cents/100, cents%100)
}
