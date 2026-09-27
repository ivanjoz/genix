package sales

import "testing"

// The client says how much stays owed; the server derives what the payment collected from the
// debt it already holds, so a crafted DebtAmount can neither overpay nor raise the debt.
func TestSaleOrderCollectedAmount(t *testing.T) {
	cases := []struct {
		name          string
		pendingDebt   int32
		remainingDebt int32
		wantCollected int32
		wantErr       bool
	}{
		{"full payment of a new sale", 10000, 0, 10000, false},
		{"partial payment", 10000, 4000, 6000, false},
		{"second payment clears the rest", 4000, 0, 4000, false},
		{"free sale paid in full", 0, 0, 0, false},
		{"negative remaining debt is refused", 10000, -1, 0, true},
		{"remaining debt above the pending debt is refused", 4000, 5000, 0, true},
		{"a payment that collects nothing is refused", 4000, 4000, 0, true},
	}
	for _, testCase := range cases {
		collected, err := saleOrderCollectedAmount(testCase.pendingDebt, testCase.remainingDebt)
		if (err != nil) != testCase.wantErr {
			t.Errorf("%s: error = %v, want error %v", testCase.name, err, testCase.wantErr)
		}
		if collected != testCase.wantCollected {
			t.Errorf("%s: collected = %d, want %d", testCase.name, collected, testCase.wantCollected)
		}
	}
}
