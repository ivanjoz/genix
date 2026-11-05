package types

import "testing"

func periodInputOf(period int16, employee Employee) PeriodPayslipInput {
	return PeriodPayslipInput{
		Run:       PayrollRun{Period: period},
		Employee:  employee,
		Contracts: []Contract{onpContract(200000, unixDay(2025, 1, 1))},
		Concepts:  map[int32]Concept{},
		Legal:     testLegalValues(),
	}
}

func TestPeriodPayslipMonthOfCtsKeepsTheDepositOutOfTheNetPay(t *testing.T) {
	// May 2026: the monthly payslip and the CTS of Nov 2025–Apr 2026 (2,000 × 180/360 = 1,000.00).
	payslip, ok := CalculatePeriodPayslip(periodInputOf(2605, Employee{ID: 1, Status: StatusActive, HireDate: unixDay(2025, 1, 1)}))
	if !ok {
		t.Fatalf("diagnostics %v", payslip.Diagnostics)
	}
	if lineAmount(payslip, LegalCts) != 100000 || lineAmount(payslip, LegalBasicSalary) != 200000 {
		t.Errorf("cts %d salary %d", lineAmount(payslip, LegalCts), lineAmount(payslip, LegalBasicSalary))
	}
	monthly, _ := CalculateMonthlyPayslip(MonthlyPayslipInput{Period: 2605, Employee: Employee{ID: 1, Status: StatusActive, HireDate: unixDay(2025, 1, 1)},
		Contracts: []Contract{onpContract(200000, unixDay(2025, 1, 1))}, Concepts: map[int32]Concept{}, Legal: testLegalValues()})
	if payslip.NetPay != monthly.NetPay {
		t.Errorf("net pay %d, the monthly one alone is %d", payslip.NetPay, monthly.NetPay)
	}
	if payslip.TotalEarnings != monthly.TotalEarnings+100000 || payslip.EmployerCost != monthly.EmployerCost+100000 {
		t.Errorf("earnings %d cost %d", payslip.TotalEarnings, payslip.EmployerCost)
	}
	if payslip.WorkedDays != 30 {
		t.Errorf("worked days %d", payslip.WorkedDays)
	}
}

func TestPeriodPayslipJulyBringsTheGratificacion(t *testing.T) {
	// July 2026 pays the first semester: 2,000 × 6/6, plus the 9 % bonus.
	payslip, ok := CalculatePeriodPayslip(periodInputOf(2607, Employee{ID: 1, Status: StatusActive, HireDate: unixDay(2025, 1, 1)}))
	if !ok || lineAmount(payslip, LegalGratificacion) != 200000 || lineAmount(payslip, LegalGratiBonus) != 18000 {
		t.Errorf("ok=%v grati %d bonus %d diagnostics %v", ok, lineAmount(payslip, LegalGratificacion),
			lineAmount(payslip, LegalGratiBonus), payslip.Diagnostics)
	}
	if lineAmount(payslip, LegalCts) != 0 {
		t.Error("July has no CTS")
	}
}

func TestPeriodPayslipOrdinaryMonthHasOnlyTheSalary(t *testing.T) {
	payslip, ok := CalculatePeriodPayslip(periodInputOf(2603, Employee{ID: 1, Status: StatusActive, HireDate: unixDay(2025, 1, 1)}))
	if !ok || lineAmount(payslip, LegalGratificacion) != 0 || lineAmount(payslip, LegalCts) != 0 || lineAmount(payslip, LegalBasicSalary) != 200000 {
		t.Errorf("ok=%v diagnostics %v", ok, payslip.Diagnostics)
	}
}

func TestPeriodPayslipTerminatedInTheMonthAddsTheSettlementOnce(t *testing.T) {
	// Terminated on 15 Oct 2026: the month has the salary of the 15 days and the settlement, whose
	// 5ta reads the monthly piece, so nothing is withheld twice.
	employee := Employee{ID: 1, Status: StatusActive, HireDate: unixDay(2025, 1, 1), TerminationDate: unixDay(2026, 10, 15)}
	payslip, ok := CalculatePeriodPayslip(periodInputOf(2610, employee))
	if !ok || lineAmount(payslip, LegalBasicSalary) == 0 || lineAmount(payslip, LegalVacationPay) == 0 {
		t.Fatalf("ok=%v diagnostics %v", ok, payslip.Diagnostics)
	}
}

func TestPeriodPayslipWithNothingToPayReportsNothing(t *testing.T) {
	// Hired after the month: no concept applies, and that is not a problem to report.
	payslip, ok := CalculatePeriodPayslip(periodInputOf(2603, Employee{ID: 1, Status: StatusActive, HireDate: unixDay(2026, 8, 1)}))
	if ok || len(payslip.Diagnostics) > 0 {
		t.Errorf("ok=%v diagnostics %v", ok, payslip.Diagnostics)
	}
}
