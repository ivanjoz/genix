package types

import "testing"

// A stock row is live (Status 1) while it holds units or owes deliveries.
func TestProductStockSelfParseStatus(t *testing.T) {
	cases := []struct {
		name  string
		stock ProductStock
		want  int8
	}{
		{"empty", ProductStock{}, 0},
		{"units on hand", ProductStock{Quantity: 2}, 1},
		{"loose sub-units only", ProductStock{SubQuantity: 4}, 1},
		{"nothing on hand, owes a store delivery", ProductStock{PendingDeliveryStore: 3}, 1},
		{"nothing on hand, owes a web delivery", ProductStock{PendingDeliveryOnline: 1}, 1},
	}
	for _, testCase := range cases {
		testCase.stock.SelfParse()
		if testCase.stock.Status != testCase.want {
			t.Errorf("%s: Status %v, want %v", testCase.name, testCase.stock.Status, testCase.want)
		}
	}
}
