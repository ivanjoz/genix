package types

import "testing"

func TestSameDaySettlement(t *testing.T) {
	const accountingDate, documentTotal = int16(20740), int32(11800)
	cases := []struct {
		name        string
		payments    []CashBankMovement
		wantSettled bool
	}{
		{"collection of the whole sale on its day", []CashBankMovement{{Date: 20740, Amount: 11800}}, true},
		{"supplier payment of the whole order on its day", []CashBankMovement{{Date: 20740, Amount: -11800}}, true},
		{"paid the next day", []CashBankMovement{{Date: 20741, Amount: 11800}}, false},
		{"paid in part", []CashBankMovement{{Date: 20740, Amount: 5000}}, false},
		{"paid in two movements", []CashBankMovement{{Date: 20740, Amount: 5000}, {Date: 20740, Amount: 6800}}, false},
		{"collected and then refunded", []CashBankMovement{{Date: 20740, Amount: 11800}, {Date: 20740, Amount: -11800}}, false},
		{"no payment", nil, false},
	}
	for _, testCase := range cases {
		settled := SameDaySettlement(accountingDate, documentTotal, testCase.payments)
		if (settled != nil) != testCase.wantSettled {
			t.Errorf("%s: settled = %v, want %v", testCase.name, settled != nil, testCase.wantSettled)
		}
	}
	if SameDaySettlement(accountingDate, 0, []CashBankMovement{{Date: 20740, Amount: 0}}) != nil {
		t.Error("a document with no total is never settled by a payment")
	}
}

func TestIsSameDaySettlement(t *testing.T) {
	if !IsSameDaySettlement(20740, 11800, 0, 20740, -11800) {
		t.Error("the first payment of the whole total on the day settles it")
	}
	if IsSameDaySettlement(20740, 11800, 5000, 20740, -6800) {
		t.Error("a payment after another one never settles alone")
	}
}
