package types

// NextMovingAverageCost is the moving weighted average after a costed inflow lands on a stock
// row: the value already held plus the value coming in, over the units held afterwards.
//
// Quantities are in sub-units at the row's divisor and costs are per whole unit, so the held
// value is balanceSubUnits × averageCost / divisor and the formula below is that expression
// with the divisor multiplied through. A balance at or below zero holds no value to average
// with, so the row takes the inflow's own unit cost.
func NextMovingAverageCost(averageCost int32, balanceSubUnits int64,
	inflowSubUnits int64, inflowValue int32, divisor int16) int32 {

	if inflowSubUnits <= 0 {
		return averageCost
	}
	if balanceSubUnits <= 0 {
		return roundedDivision(int64(inflowValue)*int64(divisor), inflowSubUnits)
	}
	heldValueTimesDivisor := balanceSubUnits * int64(averageCost)
	inflowValueTimesDivisor := int64(inflowValue) * int64(divisor)
	return roundedDivision(heldValueTimesDivisor+inflowValueTimesDivisor, balanceSubUnits+inflowSubUnits)
}

// roundedDivision rounds half away from zero; costs here are never negative.
func roundedDivision(numerator int64, denominator int64) int32 {
	return int32((numerator + denominator/2) / denominator)
}
