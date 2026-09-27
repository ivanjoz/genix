package types

import "testing"

func TestNextMovingAverageCost(t *testing.T) {
	cases := []struct {
		name            string
		averageCost     int32
		balanceSubUnits int64
		inflowSubUnits  int64
		inflowValue     int32
		divisor         int16
		expected        int32
	}{
		{"first inflow takes its own cost", 0, 0, 10, 5000, 1, 500},
		{"equal lots average evenly", 500, 10, 10, 7000, 1, 600},
		{"weighted by quantity", 1000, 30, 10, 2000, 1, 800},
		{"negative balance restarts at the inflow cost", 900, -2, 4, 2000, 1, 500},
		{"no inflow units leaves the average", 700, 5, 0, 0, 1, 700},
		// Divisor 6: 12 sub-units = 2 boxes at 600, then 6 sub-units = 1 box worth 900.
		{"sub-units at a divisor", 600, 12, 6, 900, 6, 700},
		{"rounds to the nearest cent", 100, 2, 1, 101, 1, 100},
	}
	for _, testCase := range cases {
		got := NextMovingAverageCost(testCase.averageCost, testCase.balanceSubUnits,
			testCase.inflowSubUnits, testCase.inflowValue, testCase.divisor)
		if got != testCase.expected {
			t.Errorf("%s: expected %v, got %v", testCase.name, testCase.expected, got)
		}
	}
}
