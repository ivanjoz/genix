package types

// Moving weighted average kept as a cost pool: the row carries its total value (StockValue) and
// the per-unit AverageCost is derived from it. Quantities are in sub-units at the row's divisor
// and AverageCost is per whole unit, so the divisor is multiplied through.

// StockAfterInflow is the row's value and average after an inflow lands on it. A negative
// balance (oversold history, only reachable in a replay of old data) holds no value: the
// oversold units are covered at the inflow's own cost and only what remains is kept.
func StockAfterInflow(stockValue int64, averageCost int32, balanceSubUnits int64,
	inflowSubUnits int64, inflowValue int32, divisor int16) (int64, int32) {

	if inflowSubUnits <= 0 {
		return stockValue, averageCost
	}
	balanceAfter := balanceSubUnits + inflowSubUnits
	nextValue := stockValue + int64(inflowValue)
	if balanceSubUnits < 0 {
		nextValue = int64(roundedDivision(int64(inflowValue)*max(balanceAfter, 0), inflowSubUnits))
	}
	if balanceAfter <= 0 {
		return 0, averageCost
	}
	return nextValue, roundedDivision(nextValue*int64(divisor), balanceAfter)
}

// StockOutflowValue is the value that leaves with an outflow: its share of the row's value, and
// all of it when the outflow empties the row, so the rounding of every earlier outflow is
// absorbed and Σ outflows = Σ inflows exactly. A row with no units or no value gives 0.
func StockOutflowValue(stockValue int64, balanceSubUnits int64, outflowSubUnits int64) int32 {
	if balanceSubUnits <= 0 || stockValue <= 0 || outflowSubUnits <= 0 {
		return 0
	}
	if outflowSubUnits >= balanceSubUnits {
		return int32(stockValue)
	}
	return roundedDivision(stockValue*outflowSubUnits, balanceSubUnits)
}

// roundedDivision rounds half away from zero; costs here are never negative.
func roundedDivision(numerator int64, denominator int64) int32 {
	return int32((numerator + denominator/2) / denominator)
}
