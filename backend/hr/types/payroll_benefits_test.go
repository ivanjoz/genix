package types

import "testing"

// Worked cases of the benefit runs, calculated by hand in the comments. Money in cents.

func onpContract(salary int32, from int16) Contract {
	return Contract{ID: 7, ValidFrom: from, StartDate: from, ContractType: ContractTypeIndefinite, MonthlySalary: salary,
		DailyHours: 8, PensionSystem: PensionSystemONP, Status: StatusActive}
}

// monthlyPayslip is a closed monthly payslip of the history with the given lines.
func monthlyPayslip(period int16, lines ...PayslipLine) Payslip {
	return Payslip{Period: period, WorkedDays: 30, Lines: lines, Status: PayrollStatusClosed}
}

func provisionLine(account int32, amount int32) PayslipLine {
	return PayslipLine{Kind: ConceptKindProvision, PayableAccountCode: account, Amount: amount}
}

func TestBenefitSemesters(t *testing.T) {
	for _, testCase := range []struct {
		period, gratiFirst, gratiLast, ctsFirst, ctsLast int16
	}{
		{2603, 2601, 2606, 2511, 2604},
		{2607, 2607, 2612, 2605, 2610},
		{2611, 2607, 2612, 2611, 2704},
	} {
		gratiFirst, gratiLast := GratificacionSemester(testCase.period)
		ctsFirst, ctsLast := CtsSemester(testCase.period)
		if gratiFirst != testCase.gratiFirst || gratiLast != testCase.gratiLast || ctsFirst != testCase.ctsFirst || ctsLast != testCase.ctsLast {
			t.Errorf("%d: grati %d–%d cts %d–%d", testCase.period, gratiFirst, gratiLast, ctsFirst, ctsLast)
		}
	}
}

func TestGratificacionAgainstProvision(t *testing.T) {
	// S/ 2,000 the whole first semester, with overtime of S/ 100 in 3 of its months: computable
	// 2,000 + 300/6 = 2,050; gratificación 2,050; bonus 9 % = 184.50.
	history := []Payslip{}
	for period := int16(2601); period <= 2606; period++ {
		lines := []PayslipLine{provisionLine(4114, 36000)}
		if period <= 2603 {
			lines = append(lines, PayslipLine{LegalConceptID: LegalOvertime25, Kind: ConceptKindEarning, ComputableForBenefits: true, Amount: 10000})
		}
		history = append(history, monthlyPayslip(period, lines...))
	}
	payslip, ok := CalculateGratificacion(BenefitPayslipInput{
		Run:       PayrollRun{Period: 2607},
		Employee:  Employee{ID: 1, HireDate: unixDay(2025, 1, 1)},
		Contracts: []Contract{onpContract(200000, unixDay(2025, 1, 1))},
		History:   history, Legal: testLegalValues(),
	})
	if !ok || len(payslip.Diagnostics) > 0 {
		t.Fatalf("ok=%v diagnostics=%v", ok, payslip.Diagnostics)
	}
	if lineAmount(payslip, LegalGratificacion) != 205000 || lineAmount(payslip, LegalGratiBonus) != 18450 {
		t.Errorf("grati %d bonus %d", lineAmount(payslip, LegalGratificacion), lineAmount(payslip, LegalGratiBonus))
	}
	// The 6 × 360 provisioned cover the gratificación, and only 11 of its bonus.
	if payslip.Lines[0].ProvisionedAmount != 205000 || payslip.Lines[1].ProvisionedAmount != 11000 || payslip.Lines[1].ProvisionAccountCode != 4114 {
		t.Errorf("provisioned %+v", payslip.Lines)
	}
	if payslip.NetPay != 223450 || payslip.TotalDeductions != 0 {
		t.Errorf("net %d deductions %d", payslip.NetPay, payslip.TotalDeductions)
	}
}

func TestGratificacionCountsOnlyFullMonths(t *testing.T) {
	// Hired 15 Feb: March–June are the full months, 2,000 × 4/6 = 1,333.33.
	payslip, ok := CalculateGratificacion(BenefitPayslipInput{
		Run:       PayrollRun{Period: 2607},
		Employee:  Employee{ID: 1, HireDate: unixDay(2026, 2, 15)},
		Contracts: []Contract{onpContract(200000, unixDay(2026, 2, 15))},
		Legal:     testLegalValues(),
	})
	if !ok || lineAmount(payslip, LegalGratificacion) != 133333 {
		t.Errorf("ok=%v grati %d", ok, lineAmount(payslip, LegalGratificacion))
	}
}

func TestCtsWithGratificacionSixth(t *testing.T) {
	// May 2026 deposit, semester Nov 2025–Apr 2026, all 180 days. The December gratificación of
	// S/ 2,000 adds a sixth: computable 2,333.33; CTS 2,333.33 × 180/360 = 1,166.66.
	payslip, ok := CalculateCts(BenefitPayslipInput{
		Run:       PayrollRun{Period: 2605},
		Employee:  Employee{ID: 1, HireDate: unixDay(2025, 1, 1)},
		Contracts: []Contract{onpContract(200000, unixDay(2025, 1, 1))},
		History: []Payslip{{Period: 2512, WorkedDays: 30, Status: PayrollStatusPaid,
			Lines: []PayslipLine{{LegalConceptID: LegalGratificacion, Kind: ConceptKindEarning, Amount: 200000}}}},
		Legal: testLegalValues(),
	})
	if !ok || lineAmount(payslip, LegalCts) != 116666 {
		t.Errorf("ok=%v cts %d diagnostics %v", ok, lineAmount(payslip, LegalCts), payslip.Diagnostics)
	}

	partTime := onpContract(200000, unixDay(2025, 1, 1))
	partTime.DailyHours = 3
	if _, ok := CalculateCts(BenefitPayslipInput{Run: PayrollRun{Period: 2605}, Employee: Employee{ID: 1, HireDate: unixDay(2025, 1, 1)},
		Contracts: []Contract{partTime}, Legal: testLegalValues()}); ok {
		t.Error("a part-time worker has no CTS")
	}
}

func TestSettlementOfADismissal(t *testing.T) {
	// Hired 1 Oct 2025, dismissed 15 Oct 2026, S/ 2,000, ONP, took 10 vacation days.
	payslip, ok := CalculateSettlement(BenefitPayslipInput{
		Run: PayrollRun{Period: 2610},
		Employee: Employee{ID: 1, HireDate: unixDay(2025, 10, 1), TerminationDate: unixDay(2026, 10, 15),
			TerminationReason: TerminationDismissal},
		Contracts: []Contract{onpContract(200000, unixDay(2025, 10, 1))},
		Records:   []EmployeeRecord{{Type: RecordVacation, Quantity: 10}},
		Legal:     testLegalValues(),
	})
	if !ok {
		t.Fatalf("diagnostics %v", payslip.Diagnostics)
	}
	expected := map[int16]int32{
		LegalGratificacion: 100000, // Jul–Sep: 2,000 × 3/6
		LegalGratiBonus:    9000,
		LegalCts:           91666,  // May–Sep 150 days + 15 of October: 2,000 × 165/360
		LegalVacationPay:   141667, // 375 days of service earn 31.25 days, 21.25 pending: 2,000 × 21.25/30
		LegalIndemnity:     312500, // 1.5 × 2,000 × 375/360
		LegalOnp:           18417,  // 13 % of the vacation pay
		LegalIncomeTax5ta:  0,      // S/ 2,506.67 of the year's taxed income < 7 UIT
		LegalEssalud:       12750,  // 9 % of the vacation pay
	}
	for conceptID, want := range expected {
		if got := lineAmount(payslip, conceptID); got != want {
			t.Errorf("concept %d = %d, want %d", conceptID, got, want)
		}
	}
}

func TestSettlementRecoversTheLoanBalance(t *testing.T) {
	// A S/ 900 loan in 3 installments from September: October's monthly payslip took the second,
	// the settlement recovers the third.
	payslip, _ := CalculateSettlement(BenefitPayslipInput{
		Run:       PayrollRun{Period: 2610},
		Employee:  Employee{ID: 1, HireDate: unixDay(2025, 10, 1), TerminationDate: unixDay(2026, 10, 31), TerminationReason: TerminationResignation},
		Contracts: []Contract{onpContract(200000, unixDay(2025, 10, 1))},
		Records:   LoanInstallments(EmployeeRecord{ID: 4, Type: RecordLoan, Amount: 90000, Quantity: 3}, 2609),
		Legal:     testLegalValues(),
	})
	if lineAmount(payslip, LegalLoanPayment) != 30000 || lineAmount(payslip, LegalIndemnity) != 0 {
		t.Errorf("loan %d indemnity %d", lineAmount(payslip, LegalLoanPayment), lineAmount(payslip, LegalIndemnity))
	}
}

func TestMonthlyPayslipWithVacation(t *testing.T) {
	// 10 vacation days in March: 20 days of salary and 10 of vacation pay, the vacation provision
	// left (S/ 500) charged first.
	payslip, ok := CalculateMonthlyPayslip(MonthlyPayslipInput{
		Period:    2603,
		Employee:  Employee{ID: 1, HireDate: unixDay(2025, 1, 1), Status: StatusActive},
		Contracts: []Contract{onpContract(200000, unixDay(2025, 1, 1))},
		Records:   []EmployeeRecord{{Type: RecordVacation, FromDate: unixDay(2026, 3, 10), ToDate: unixDay(2026, 3, 19), Quantity: 10}},
		History:   []Payslip{monthlyPayslip(2602, provisionLine(4115, 50000))},
		Legal:     testLegalValues(),
	})
	if !ok || lineAmount(payslip, LegalBasicSalary) != 133333 || lineAmount(payslip, LegalVacationPay) != 66666 || payslip.WorkedDays != 30 {
		t.Fatalf("ok=%v salary %d vacation %d days %d", ok, lineAmount(payslip, LegalBasicSalary),
			lineAmount(payslip, LegalVacationPay), payslip.WorkedDays)
	}
	for _, line := range payslip.Lines {
		if line.LegalConceptID == LegalVacationPay && (line.ProvisionAccountCode != 4115 || line.ProvisionedAmount != 50000) {
			t.Errorf("vacation line %+v", line)
		}
	}
}

func TestDistributeProfitSharing(t *testing.T) {
	// S/ 10,000: half by days (360 / 180), half by remuneration (24,000 / 12,000).
	bases := []ProfitShareBasis{
		{EmployeeID: 1, Days: 360, Remuneration: 2400000, Cap: 3600000},
		{EmployeeID: 2, Days: 180, Remuneration: 1200000, Cap: 3600000},
	}
	shares, excess := DistributeProfitSharing(1000000, bases)
	if shares[1] != 666666 || shares[2] != 333332 || len(excess) != 0 {
		t.Errorf("shares %v excess %v", shares, excess)
	}
	bases[0].Cap = 500000
	shares, excess = DistributeProfitSharing(1000000, bases)
	if shares[1] != 500000 || excess[1] != 166666 {
		t.Errorf("capped shares %v excess %v", shares, excess)
	}
}
