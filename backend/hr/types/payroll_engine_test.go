package types

import (
	"testing"
	"time"
)

// Worked cases, calculated by hand in the comments. Money in cents.

func unixDay(year int, month time.Month, day int) int16 {
	return int16(time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// testLegalValues: RMV S/ 1,130 until Sep 2026 and S/ 1,230 from Oct 2026; UIT S/ 5,500; Integra
// Oct 2026 at 10 % fund, 1.37 % premium, 1.55 % flow commission.
func testLegalValues() LegalValues {
	parameters := []LegalParameter{
		{Key: LegalParamRMV, ValidFrom: unixDay(2025, 1, 1), Value: 113000},
		{Key: LegalParamRMV, ValidFrom: unixDay(2026, 10, 1), Value: 123000},
		{Key: LegalParamUIT, ValidFrom: unixDay(2026, 1, 1), Value: 550000},
		{Key: LegalParamEssaludRate, ValidFrom: unixDay(2025, 1, 1), Value: 900},
		{Key: LegalParamEssaludEpsRate, ValidFrom: unixDay(2025, 1, 1), Value: 675},
		{Key: LegalParamOnpRate, ValidFrom: unixDay(2025, 1, 1), Value: 1300},
		{Key: LegalParamMaxInsurableSalary, ValidFrom: unixDay(2026, 10, 1), Value: 1273270},
		{Key: LegalParamFamilyAllowanceRate, ValidFrom: unixDay(2025, 1, 1), Value: 1000},
		{Key: LegalParamGratiBonusRate, ValidFrom: unixDay(2025, 1, 1), Value: 900},
		{Key: LegalParamGratiBonusEpsRate, ValidFrom: unixDay(2025, 1, 1), Value: 675},
	}
	for i := range parameters {
		parameters[i].Status = StatusActive
	}
	afpRates := []AfpRate{{Period: 2610, PensionSystem: PensionSystemIntegra, FundRate: 1000, PremiumRate: 137,
		FlowCommission: 155, Status: StatusActive}}
	return NewLegalValues(parameters, afpRates)
}

func lineAmount(payslip Payslip, legalConceptID int16) int32 {
	var total int32
	for _, line := range payslip.Lines {
		if line.LegalConceptID == legalConceptID {
			total += line.Amount
		}
	}
	return total
}

func TestMonthlyPayslipOnp(t *testing.T) {
	// S/ 2,000, ONP, no children, whole of March 2026.
	payslip, ok := CalculateMonthlyPayslip(MonthlyPayslipInput{
		Period:   2603,
		Employee: Employee{ID: 1, HireDate: unixDay(2025, 1, 1), Status: StatusActive},
		Contracts: []Contract{{ID: 7, ValidFrom: unixDay(2025, 1, 1), StartDate: unixDay(2025, 1, 1),
			MonthlySalary: 200000, DailyHours: 8, PensionSystem: PensionSystemONP, Status: StatusActive}},
		Legal: testLegalValues(),
	})
	if !ok || len(payslip.Diagnostics) > 0 {
		t.Fatalf("unexpected result: ok=%v diagnostics=%v", ok, payslip.Diagnostics)
	}
	expected := map[int16]int32{
		LegalBasicSalary:  200000,
		LegalOnp:          26000, // 13 %
		LegalIncomeTax5ta: 0,     // S/ 24,360 projected < 7 UIT
		LegalEssalud:      18000, // 9 %
		// Provisions: grati 1/6, its 9 % bonus, CTS (salary + 1/6 grati) / 12, vacations 1/12.
		LegalProvisionGratificacion: 33333,
		LegalProvisionGratiBonus:    3000,
		LegalProvisionCts:           19444,
		LegalProvisionVacation:      16666,
	}
	for conceptID, want := range expected {
		if got := lineAmount(payslip, conceptID); got != want {
			t.Errorf("concept %d = %d, want %d", conceptID, got, want)
		}
	}
	if payslip.NetPay != 174000 || payslip.EmployerCost != 218000 || payslip.ContractID != 7 || payslip.WorkedDays != 30 {
		t.Errorf("net %d cost %d contract %d days %d", payslip.NetPay, payslip.EmployerCost, payslip.ContractID, payslip.WorkedDays)
	}
}

func TestMonthlyPayslipAfpWithFamilyAllowance(t *testing.T) {
	// S/ 6,000, Integra on flow, a 10-year-old child, October 2026 (RMV S/ 1,230).
	payslip, ok := CalculateMonthlyPayslip(MonthlyPayslipInput{
		Period: 2610,
		Employee: Employee{ID: 2, HireDate: unixDay(2024, 3, 1), Status: StatusActive,
			FamilyAllowance: FamilyAllowanceRanges([]Dependent{{Relationship: RelationshipChild, BirthDate: unixDay(2016, 5, 5), Status: StatusActive}})},
		Contracts: []Contract{{ID: 9, ValidFrom: unixDay(2024, 3, 1), StartDate: unixDay(2024, 3, 1),
			MonthlySalary: 600000, DailyHours: 8, PensionSystem: PensionSystemIntegra,
			AfpCommissionType: AfpCommissionFlow, Status: StatusActive}},
		Legal: testLegalValues(),
	})
	if !ok || len(payslip.Diagnostics) > 0 {
		t.Fatalf("unexpected result: ok=%v diagnostics=%v", ok, payslip.Diagnostics)
	}
	expected := map[int16]int32{
		LegalFamilyAllowance: 12300, // 10 % of S/ 1,230
		LegalAfpFund:         61230, // 10 % of 612,300
		LegalAfpPremium:      8389,  // 1.37 %, under the RMA
		LegalAfpCommission:   9491,  // 1.55 %
		LegalEssalud:         55107, // 9 %
	}
	for conceptID, want := range expected {
		if got := lineAmount(payslip, conceptID); got != want {
			t.Errorf("concept %d = %d, want %d", conceptID, got, want)
		}
	}
}

func TestMonthlyPayslipDiagnostics(t *testing.T) {
	employee := Employee{ID: 3, HireDate: unixDay(2026, 10, 16), Status: StatusActive}
	contract := Contract{ID: 1, ValidFrom: unixDay(2026, 10, 16), StartDate: unixDay(2026, 10, 16),
		MonthlySalary: 300000, DailyHours: 8, PensionSystem: PensionSystemPrima, Status: StatusActive}

	if _, ok := CalculateMonthlyPayslip(MonthlyPayslipInput{Period: 2610, Employee: employee, Legal: testLegalValues()}); ok {
		t.Error("an employee without contract must not get a payslip")
	}

	// Hired on Oct 16: 16 days. Prima has no AFP rates for the period, and the advance exceeds the net.
	payslip, _ := CalculateMonthlyPayslip(MonthlyPayslipInput{Period: 2610, Employee: employee,
		Contracts: []Contract{contract}, Records: []EmployeeRecord{{ID: 4, Type: RecordAdvanceWeekly, Period: 2610, Amount: 200000}},
		Legal: testLegalValues()})
	if payslip.WorkedDays != 16 || lineAmount(payslip, LegalBasicSalary) != 160000 {
		t.Errorf("days %d basic %d, want 16 and 160000", payslip.WorkedDays, lineAmount(payslip, LegalBasicSalary))
	}
	codes := map[string]bool{}
	for _, diagnostic := range payslip.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes[DiagnosticAfpRatesMissing] || !codes[DiagnosticAdvancesExceed] {
		t.Errorf("diagnostics %v, want %s and %s", payslip.Diagnostics, DiagnosticAfpRatesMissing, DiagnosticAdvancesExceed)
	}
}

func TestAnnualIncomeTax(t *testing.T) {
	// S/ 100,000 − 7 UIT = S/ 61,500: 8 % of 27,500 + 14 % of 34,000 = S/ 6,960.
	if got := AnnualIncomeTax(10000000, 550000); got != 696000 {
		t.Errorf("AnnualIncomeTax = %d, want 696000", got)
	}
	if got := AnnualIncomeTax(3000000, 550000); got != 0 {
		t.Errorf("income under 7 UIT must not be taxed, got %d", got)
	}
}

func TestMonthlyIncomeTax(t *testing.T) {
	// January, S/ 10,000 a month: projected 12 × 10,000 + 2 gratificaciones of 10,900 = S/ 141,800.
	// Taxable 103,300 → 2,200 + 14 % of 75,800 = S/ 12,812; January withholds 1/12.
	input := IncomeTaxInput{Month: 1, TaxUnit: 550000, CurrentOrdinary: 1000000, FixedMonthly: 1000000,
		GratificacionsToProject: 2, GratiBonusRate: 900}
	if got := MonthlyIncomeTax(input); got != 106766 {
		t.Errorf("January withholding = %d, want 106766", got)
	}
	// A S/ 5,000 one-off bonus adds 14 % of it on top, all in the month.
	input.CurrentExtraordinary = 500000
	if got := MonthlyIncomeTax(input); got != 176766 {
		t.Errorf("January withholding with bonus = %d, want 176766", got)
	}
	// December settles the year: the projected tax less everything withheld.
	december := IncomeTaxInput{Month: 12, TaxUnit: 550000, PriorIncome: 11*1000000 + 1090000, CurrentOrdinary: 1000000,
		FixedMonthly: 1000000, GratificacionsToProject: 1, GratiBonusRate: 900}
	for month := 1; month <= 11; month++ {
		december.PriorWithheld[month] = 106766
	}
	if got := MonthlyIncomeTax(december); got != 1281200-11*106766 {
		t.Errorf("December withholding = %d, want %d", got, 1281200-11*106766)
	}
}

func TestLoanInstallments(t *testing.T) {
	// S/ 1,000 in 3 installments from Nov 2026: the last one absorbs the rounding.
	installments := LoanInstallments(EmployeeRecord{ID: 8, EmployeeID: 2, Type: RecordLoan, Amount: 100000, Quantity: 3}, 2611)
	got := map[int16]int32{}
	for _, installment := range installments {
		if installment.LoanID != 8 || installment.Type != RecordLoanInstallment || installment.Status != RecordStatusActive {
			t.Errorf("installment %+v", installment)
		}
		got[installment.Period] = installment.Amount
	}
	if len(got) != 3 || got[2611] != 33333 || got[2612] != 33333 || got[2701] != 33334 || installments[2].Year != 2027 {
		t.Errorf("installments %v", got)
	}
}

func TestFamilyAllowance(t *testing.T) {
	day := unixDay(2026, 10, 31)
	child := Dependent{Relationship: RelationshipChild, Status: StatusActive}
	cases := []struct {
		name       string
		birthDate  int16
		isStudying bool
		want       bool
	}{
		{"minor", unixDay(2010, 1, 1), false, true},
		{"turns 18 tomorrow", unixDay(2008, 11, 1), false, true},
		{"turned 18 today", unixDay(2008, 10, 31), false, false},
		{"adult not studying", unixDay(2006, 1, 1), false, false},
		{"adult studying, 23", unixDay(2003, 1, 1), true, true},
		{"adult studying, 24", unixDay(2002, 1, 1), true, false},
	}
	for _, testCase := range cases {
		child.BirthDate, child.IsStudying = testCase.birthDate, testCase.isStudying
		employee := Employee{FamilyAllowance: FamilyAllowanceRanges([]Dependent{child})}
		if got := employee.HasFamilyAllowanceOn(day); got != testCase.want {
			t.Errorf("%s: HasFamilyAllowanceOn = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestFamilyAllowanceRangesMergeChildrenAndKeepGaps(t *testing.T) {
	// Child A covers 2008-01-01 → 2025-12-31; child B, born 2026-03-01, opens a second range; child C
	// (registered from 2020, born 2019) overlaps A and merges into it. A spouse gives no right.
	dependents := []Dependent{
		{Relationship: RelationshipChild, BirthDate: unixDay(2008, 1, 1), Status: StatusActive},
		{Relationship: RelationshipChild, BirthDate: unixDay(2026, 3, 1), Status: StatusActive},
		{Relationship: RelationshipChild, BirthDate: unixDay(2019, 6, 1), ValidFrom: unixDay(2020, 1, 1), ValidTo: unixDay(2021, 1, 1), Status: StatusActive},
		{Relationship: RelationshipSpouse, BirthDate: unixDay(1990, 1, 1), Status: StatusActive},
	}
	ranges := FamilyAllowanceRanges(dependents)
	want := []DayRange{{From: unixDay(2008, 1, 1), To: unixDay(2025, 12, 31)}, {From: unixDay(2026, 3, 1), To: unixDay(2044, 2, 29)}}
	if len(ranges) != 2 || ranges[0] != want[0] || ranges[1] != want[1] {
		t.Errorf("ranges %v, want %v", ranges, want)
	}
	employee := Employee{}
	employee.SetDependentsSummary(dependents)
	if employee.DependentsCount != 4 || !employee.HasFamilyAllowanceOn(unixDay(2026, 3, 1)) || employee.HasFamilyAllowanceOn(unixDay(2026, 1, 15)) {
		t.Errorf("summary %d %v", employee.DependentsCount, employee.FamilyAllowance)
	}
}
