package types

import "testing"

func TestEstimatedMonthlyEmployerCostAndHourlyCost(t *testing.T) {
	// A worker hired mid-month still estimates a full month: 2000.00 salary, EsSalud 9% (180.00), and
	// the month's provisions. Whatever the exact provisions, the estimate covers more than the salary.
	employee := Employee{ID: 3, HireDate: unixDay(2026, 3, 15), EngagementType: EngagementPayroll}
	agreement := onpContract(200000, unixDay(2026, 3, 15))
	monthlyCost, err := EstimatedMonthlyEmployerCost(employee, agreement, 2603, nil, map[int32]Concept{}, testLegalValues())
	if err != nil {
		t.Fatal(err)
	}
	if monthlyCost <= 200000+18000 {
		t.Fatalf("monthly cost %v should include salary, EsSalud and provisions", monthlyCost)
	}
	// 8 hours × 26 days = 208 hours.
	if got := ContractAgreementHourlyCost(employee, agreement, 208000); got != 1000 {
		t.Fatalf("payroll hourly cost: got %v, want 1000", got)
	}

	offPayroll := Employee{EngagementType: EngagementTemporaryOff}
	cases := []struct {
		name      string
		agreement ContractAgreement
		want      int32
	}{
		{"monthly", ContractAgreement{PayBasis: PayBasisMonthly, PayAmount: 156000, DailyHours: 6}, 1000},
		{"weekly", ContractAgreement{PayBasis: PayBasisWeekly, PayAmount: 48000, DailyHours: 8}, 1000},
		{"hourly", ContractAgreement{PayBasis: PayBasisHourly, PayAmount: 1250}, 1250},
	}
	for _, testCase := range cases {
		if got := ContractAgreementHourlyCost(offPayroll, testCase.agreement, 0); got != testCase.want {
			t.Errorf("%s: got %v, want %v", testCase.name, got, testCase.want)
		}
	}
}
