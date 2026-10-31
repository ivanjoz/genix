package types

import "testing"

func TestStockAfterInflow(t *testing.T) {
	cases := []struct {
		name            string
		stockValue      int64
		averageCost     int32
		balanceSubUnits int64
		inflowSubUnits  int64
		inflowValue     int32
		divisor         int16
		expectedValue   int64
		expectedAverage int32
	}{
		{"first inflow takes its own cost", 0, 0, 0, 10, 5000, 1, 5000, 500},
		{"equal lots average evenly", 5000, 500, 10, 10, 7000, 1, 12000, 600},
		{"weighted by quantity", 30000, 1000, 30, 10, 2000, 1, 32000, 800},
		{"no inflow units leaves the row", 3500, 700, 5, 0, 0, 1, 3500, 700},
		// Divisor 6: 12 sub-units = 2 boxes worth 1200, then 6 sub-units = 1 box worth 900.
		{"sub-units at a divisor", 1200, 600, 12, 6, 900, 6, 2100, 700},
		{"the average rounds, the value does not", 0, 0, 0, 3, 1000, 1, 1000, 333},
		// 2 units oversold, 4 come in at 500: the 2 that remain are worth 1000.
		{"negative balance covers the oversold units first", 0, 900, -2, 4, 2000, 1, 1000, 500},
		{"still negative after the inflow holds no value", 0, 900, -5, 2, 1000, 1, 0, 900},
	}
	for _, testCase := range cases {
		gotValue, gotAverage := StockAfterInflow(testCase.stockValue, testCase.averageCost,
			testCase.balanceSubUnits, testCase.inflowSubUnits, testCase.inflowValue, testCase.divisor)
		if gotValue != testCase.expectedValue || gotAverage != testCase.expectedAverage {
			t.Errorf("%s: expected value %v average %v, got value %v average %v", testCase.name,
				testCase.expectedValue, testCase.expectedAverage, gotValue, gotAverage)
		}
	}
}

func TestStockOutflowValue(t *testing.T) {
	cases := []struct {
		name            string
		stockValue      int64
		balanceSubUnits int64
		outflowSubUnits int64
		expected        int32
	}{
		{"share of the value", 35000, 20, 12, 21000},
		{"rounds to the nearest cent", 1000, 3, 1, 333},
		{"the emptying outflow takes the remainder", 667, 2, 2, 667},
		{"an oversold outflow takes no more than the row holds", 667, 2, 5, 667},
		{"a row with no value gives 0", 0, 4, 1, 0},
		{"a row with no units gives 0", 500, 0, 1, 0},
	}
	for _, testCase := range cases {
		got := StockOutflowValue(testCase.stockValue, testCase.balanceSubUnits, testCase.outflowSubUnits)
		if got != testCase.expected {
			t.Errorf("%s: expected %v, got %v", testCase.name, testCase.expected, got)
		}
	}
}

// Whatever the purchases and the split of the sales, selling everything takes out exactly what
// came in: the row ends at value 0 together with its units.
func TestStockValueEndsAtZeroWhenTheRowEmpties(t *testing.T) {
	type step struct {
		subUnits int64 // positive in, negative out
		value    int32 // inflow value; ignored on an outflow
	}
	scenarios := []struct {
		name    string
		divisor int16
		steps   []step
	}{
		// SALE_COST_PLAN §3.3: 10 cookies at 20.00, 10 at 15.00, sell 12 then 8.
		{"cookies", 1, []step{{10, 20000}, {10, 15000}, {-12, 0}, {-8, 0}}},
		{"three for ten", 1, []step{{3, 1000}, {-1, 0}, {-2, 0}}},
		{"one at a time", 1, []step{{7, 1000}, {-1, 0}, {-1, 0}, {-1, 0}, {-1, 0}, {-1, 0}, {-1, 0}, {-1, 0}}},
		// Divisor 12: buy 1 box for 10.00 and 5 loose units for 3.33, sell them in odd lots.
		{"sub-units", 12, []step{{12, 1000}, {5, 333}, {-7, 0}, {3, 101}, {-5, 0}, {-8, 0}}},
	}
	for _, scenario := range scenarios {
		stockValue, averageCost, balance := int64(0), int32(0), int64(0)
		valueIn, valueOut := int64(0), int64(0)
		for _, s := range scenario.steps {
			if s.subUnits > 0 {
				stockValue, averageCost = StockAfterInflow(stockValue, averageCost, balance, s.subUnits,
					s.value, scenario.divisor)
				valueIn += int64(s.value)
			} else {
				outflowValue := StockOutflowValue(stockValue, balance, -s.subUnits)
				stockValue -= int64(outflowValue)
				valueOut += int64(outflowValue)
			}
			balance += s.subUnits
		}
		if balance != 0 || stockValue != 0 || valueIn != valueOut {
			t.Errorf("%s: expected an empty row at value 0, got balance %v value %v (in %v, out %v)",
				scenario.name, balance, stockValue, valueIn, valueOut)
		}
	}
}
