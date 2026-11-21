package types

import (
	"reflect"
	"testing"
)

// A payment covers fees first, then interest, then principal, picking up where earlier partial
// payments of the installment left off.
func TestLoanPaymentSplitsApplyFeesThenInterestThenPrincipal(t *testing.T) {
	installment := LoanInstallment{Number: 4, Principal: 6120, Interest: 1300, Fees: 80}

	full, err := LoanPaymentSplits(installment, 7500)
	if err != nil {
		t.Fatal(err)
	}
	wantFull := []CashMovementAccountSplit{{PCGELoanCharges, -80}, {PCGELoanInterest, -1300}, {PCGELoansReceived, -6120}}
	if !reflect.DeepEqual(full, wantFull) {
		t.Errorf("full payment splits = %v, want %v", full, wantFull)
	}

	// 1000 already paid: the 80 of fees and 920 of the interest. The next 2000 finishes the
	// interest (380) and starts the principal (1620).
	installment.PaidAmount = 1000
	partial, err := LoanPaymentSplits(installment, 2000)
	if err != nil {
		t.Fatal(err)
	}
	wantPartial := []CashMovementAccountSplit{{PCGELoanInterest, -380}, {PCGELoansReceived, -1620}}
	if !reflect.DeepEqual(partial, wantPartial) {
		t.Errorf("partial payment splits = %v, want %v", partial, wantPartial)
	}

	if _, err := LoanPaymentSplits(installment, 6501); err == nil {
		t.Error("a payment above what is pending was accepted")
	}
	if err := ValidateCashMovementSplits(CashMovementTypeLoanPayment, -2000, partial); err != nil {
		t.Errorf("the splits a payment builds should pass the writer's check: %v", err)
	}
}

func TestValidateCashMovementSplits(t *testing.T) {
	splits := []CashMovementAccountSplit{{PCGELoansReceived, -600}, {PCGELoanInterest, -400}}
	if err := ValidateCashMovementSplits(CashMovementTypeLoanPayment, -1000, splits); err != nil {
		t.Fatalf("valid splits refused: %v", err)
	}
	for name, testCase := range map[string]struct {
		movementType CashMovementType
		amount       int32
		splits       []CashMovementAccountSplit
	}{
		"no splits on a split type": {CashMovementTypeLoanPayment, -1000, nil},
		"splits on a plain type":    {CashMovementTypeExpensePayment, -1000, splits},
		"splits not adding up":      {CashMovementTypeLoanPayment, -900, splits},
		"account not allowed":       {CashMovementTypeLoanPayment, -1000, []CashMovementAccountSplit{{PCGESuppliers, -1000}}},
		"split against the sign":    {CashMovementTypeLoanPayment, -1000, []CashMovementAccountSplit{{PCGELoansReceived, -1200}, {PCGELoanInterest, 200}}},
	} {
		if ValidateCashMovementSplits(testCase.movementType, testCase.amount, testCase.splits) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func validLoan() Loan {
	return Loan{
		Name: "BCP capital de trabajo", LenderID: 3, CashBankID: 3, Principal: 10000, DisbursementDate: 20000,
		Installments: []LoanInstallment{
			{Number: 1, DueDate: 20030, Principal: 5000, Interest: 200},
			{Number: 2, DueDate: 20060, Principal: 5000, Interest: 100},
		},
	}
}

func TestValidateLoan(t *testing.T) {
	loan := validLoan()
	if err := ValidateLoan(&loan); err != nil {
		t.Fatalf("valid loan refused: %v", err)
	}
	for name, breakLoan := range map[string]func(*Loan){
		"principals not adding up":    func(loan *Loan) { loan.Installments[1].Principal = 4000 },
		"due dates not increasing":    func(loan *Loan) { loan.Installments[1].DueDate = 20030 },
		"due before the disbursement": func(loan *Loan) { loan.Installments[0].DueDate = 19990 },
		"numbers out of order":        func(loan *Loan) { loan.Installments[1].Number = 3 },
		"negative interest":           func(loan *Loan) { loan.Installments[0].Interest = -1 },
		"paid above the installment":  func(loan *Loan) { loan.Installments[0].PaidAmount = 5201 },
		"no lender":                   func(loan *Loan) { loan.LenderID = 0 },
	} {
		brokenLoan := validLoan()
		breakLoan(&brokenLoan)
		if ValidateLoan(&brokenLoan) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestLoanIsPaidOff(t *testing.T) {
	loan := validLoan()
	if loan.IsPaidOff() {
		t.Fatal("an unpaid loan reads as paid off")
	}
	loan.Installments[0].PaidAmount, loan.Installments[1].PaidAmount = 5200, 5100
	if !loan.IsPaidOff() {
		t.Error("a fully paid loan does not read as paid off")
	}
}
