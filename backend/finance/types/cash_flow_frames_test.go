package types

import "testing"

func TestToExpenseSummaryFrames(t *testing.T) {
	base := Expense{Type: ExpenseTypeSimple, Status: 1, Date: 20000, SiteID: 2, CategoryID: 1,
		CurrencyType: CurrencyPEN, Amount: 11800, TaxAmount: 1800, AccountCode: 6352}

	cases := []struct {
		name     string
		mutate   func(expense *Expense)
		wantLine *ExpenseSummaryFrame
	}{
		{"a pending simple expense counts", func(*Expense) {},
			&ExpenseSummaryFrame{Date: 20000, SiteID: 2, CategoryID: 1, AmountPEN: 11800, TaxAmountPEN: 1800}},
		{"a paid one too", func(expense *Expense) { expense.Status = 2 },
			&ExpenseSummaryFrame{Date: 20000, SiteID: 2, CategoryID: 1, AmountPEN: 11800, TaxAmountPEN: 1800}},
		{"a schedule's period is left to its schedule", func(expense *Expense) { expense.ExpenseScheduledID = 4 }, nil},
		{"a tax payment is left to the tax stream", func(expense *Expense) { expense.AccountCode = 40111 }, nil},
		{"a removed expense adds nothing", func(expense *Expense) { expense.Status = 0 }, nil},
		{"depreciation moves no cash", func(expense *Expense) {
			expense.Type, expense.Status = ExpenseTypeDepreciation, ExpenseStatusPosted
		}, nil},
		{"a dollar expense is restated at its rate", func(expense *Expense) {
			expense.CurrencyType, expense.ExchangeRate, expense.Amount, expense.TaxAmount = CurrencyUSD, 3750, 1000, 0
		}, &ExpenseSummaryFrame{Date: 20000, SiteID: 2, CategoryID: 1, AmountPEN: 3750}},
		{"a dollar expense without a rate is left out", func(expense *Expense) {
			expense.CurrencyType, expense.ExchangeRate = CurrencyUSD, 0
		}, nil},
	}
	for _, testCase := range cases {
		expense := base
		testCase.mutate(&expense)
		lines := expense.ToExpenseSummaryFrames(nil)
		if testCase.wantLine == nil {
			if len(lines) != 0 {
				t.Errorf("%s: want no line, got %+v", testCase.name, lines)
			}
			continue
		}
		if len(lines) != 1 || lines[0] != *testCase.wantLine {
			t.Errorf("%s: want %+v, got %+v", testCase.name, *testCase.wantLine, lines)
		}
	}
}

func TestToCashDelayFrames(t *testing.T) {
	collection := CashBankMovement{CashBankID: 3, Date: 20010, DocumentDate: 20000,
		Type: CashMovementTypeSaleCollection, Amount: 5000}
	lines := collection.ToCashDelayFrames(nil)
	want := CashDelayFrame{Date: 20010, CashBankID: 3, Type: CashMovementTypeSaleCollection, DelayDays: 10, Amount: 5000}
	if len(lines) != 1 || lines[0] != want {
		t.Fatalf("collection: want %+v, got %+v", want, lines)
	}

	advance := collection
	advance.Date = 19990
	if lines := advance.ToCashDelayFrames(nil); len(lines) != 1 || lines[0].DelayDays != 0 {
		t.Errorf("a payment before its document counts at delay 0, got %+v", lines)
	}

	straggler := collection
	straggler.Date = 20000 + 400
	if lines := straggler.ToCashDelayFrames(nil); len(lines) != 1 || lines[0].DelayDays != CashDelayMaxDays {
		t.Errorf("a late payment is clipped to %d days, got %+v", CashDelayMaxDays, lines)
	}

	for _, movementType := range []CashMovementType{CashMovementTypeTransfer, CashMovementTypeAssetPayment,
		CashMovementTypeSaleRefund, CashMovementTypePayrollPayment} {
		movement := collection
		movement.Type = movementType
		if lines := movement.ToCashDelayFrames(nil); len(lines) != 0 {
			t.Errorf("type %d is not forecast from its delay, got %+v", movementType, lines)
		}
	}
}

func TestIsTaxPayableAccount(t *testing.T) {
	for code, want := range map[int32]bool{40111: true, 40171: true, 4011: true, 6412: false, 643: false, 0: false} {
		if IsTaxPayableAccount(code) != want {
			t.Errorf("IsTaxPayableAccount(%d) = %v, want %v", code, !want, want)
		}
	}
}
