package types

// Renta de quinta categoría (LIR art. 53 and Reglamento art. 40): the employer withholds each
// month a share of the worker's projected annual tax. Pure; money in cents.
//
// TODO(accountant): verify the projection base (here: fixed monthly remuneration for the remaining
// months, actual ordinary income for the current one) against a SUNAT worked example.

// incomeTaxBracket: income up to UpperUIT tax units is taxed at Rate (basis points).
type incomeTaxBracket struct {
	UpperUIT int64
	Rate     int64
}

// Brackets of LIR art. 53, in UIT of the year.
var incomeTaxBrackets = []incomeTaxBracket{
	{UpperUIT: 5, Rate: 800},
	{UpperUIT: 20, Rate: 1400},
	{UpperUIT: 35, Rate: 1700},
	{UpperUIT: 45, Rate: 2000},
	{UpperUIT: 0, Rate: 3000}, // 0 = no upper limit
}

// The first 7 UIT of the year's employment income are deducted (LIR art. 46).
const incomeTaxDeductionUIT = 7

// AnnualIncomeTax is the tax on a year's gross employment income.
func AnnualIncomeTax(annualIncome int64, taxUnit int64) int64 {
	taxable := annualIncome - incomeTaxDeductionUIT*taxUnit
	if taxable <= 0 {
		return 0
	}
	var tax, lowerLimit int64
	for _, bracket := range incomeTaxBrackets {
		upperLimit := bracket.UpperUIT * taxUnit
		if bracket.UpperUIT == 0 || taxable < upperLimit {
			return tax + roundRate(taxable-lowerLimit, bracket.Rate)
		}
		tax += roundRate(upperLimit-lowerLimit, bracket.Rate)
		lowerLimit = upperLimit
	}
	return tax
}

// IncomeTaxInput is what the monthly withholding needs, all in cents.
type IncomeTaxInput struct {
	Month   int   // 1–12
	TaxUnit int64 // UIT of the year
	// PriorIncome is every taxable income (ordinary and extraordinary) paid in the year's previous
	// months; PriorWithheld, the 5ta withheld on it, by month (index 1–12).
	PriorIncome   int64
	PriorWithheld [13]int64
	// CurrentOrdinary is this month's ordinary taxable income; FixedMonthly, the fixed monthly
	// remuneration projected over the remaining months; GratificacionsToProject, how many
	// gratificaciones (July, December) are still to be paid this year, each projected as
	// FixedMonthly plus its bonus (GratiBonusRate).
	CurrentOrdinary         int64
	FixedMonthly            int64
	GratificacionsToProject int64
	GratiBonusRate          int64
	// CurrentExtraordinary is this month's extraordinary income (one-off bonus, utilidades): it is
	// taxed whole this month, never projected.
	CurrentExtraordinary int64
}

// projectedAnnualIncome is the year's income as projected this month: what was paid, this month's
// ordinary income, the fixed remuneration of the remaining months and the gratificaciones to come.
func projectedAnnualIncome(input IncomeTaxInput) int64 {
	remainingMonths := int64(12 - input.Month)
	gratificacion := input.FixedMonthly + roundRate(input.FixedMonthly, input.GratiBonusRate)
	return input.PriorIncome + input.CurrentOrdinary + input.FixedMonthly*remainingMonths +
		gratificacion*input.GratificacionsToProject
}

// MonthlyIncomeTax is the 5ta to withhold this month.
func MonthlyIncomeTax(input IncomeTaxInput) int64 {
	projectedIncome := projectedAnnualIncome(input)
	projectedTax := AnnualIncomeTax(projectedIncome, input.TaxUnit)

	// Reglamento art. 40: the projected tax, less what earlier months withheld, spread over a divisor
	// that depends on the month.
	withheldUntil := func(lastMonth int) int64 {
		var withheld int64
		for month := 1; month <= lastMonth; month++ {
			withheld += input.PriorWithheld[month]
		}
		return withheld
	}
	var ordinary int64
	switch {
	case input.Month <= 3:
		ordinary = projectedTax / 12
	case input.Month == 4:
		ordinary = (projectedTax - withheldUntil(3)) / 9
	case input.Month <= 7:
		ordinary = (projectedTax - withheldUntil(4)) / 8
	case input.Month == 8:
		ordinary = (projectedTax - withheldUntil(7)) / 5
	case input.Month <= 11:
		ordinary = (projectedTax - withheldUntil(8)) / 4
	default:
		ordinary = projectedTax - withheldUntil(11)
	}
	if ordinary < 0 {
		ordinary = 0
	}

	// Extraordinary income adds the tax it causes on top of the projection, all this month.
	extraordinary := AnnualIncomeTax(projectedIncome+input.CurrentExtraordinary, input.TaxUnit) - projectedTax
	return ordinary + extraordinary
}

// roundRate is amount × rate (basis points), rounded half up.
func roundRate(amount int64, rate int64) int64 {
	return (amount*rate + RateScale/2) / RateScale
}
