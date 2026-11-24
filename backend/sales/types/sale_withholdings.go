package types

import (
	"app/core"
	finance "app/finance/types"
	production "app/production/types"
	"math"
)

// The detracción (SPOT) and the retención of a sale (plan/CASH_DOCUMENTS_FRAME_PLAN.md §9.1): both are
// shares of a factura the customer does not pay to the company. The detracción is deposited into the
// company's Banco de la Nación detracciones account, so it stays owed until that deposit is recorded;
// the retención is paid to SUNAT by the customer, so the company never collects it (it becomes an IGV
// credit).

// SaleWithholdingContext is what decides the withholdings besides the sale itself.
type SaleWithholdingContext struct {
	// IsFactura: only a factura carries a detracción or a retención.
	IsFactura bool
	// IsRetentionAgent: the client withholds the IGV retention (crm.ClientProvider.IsRetentionAgent).
	IsRetentionAgent bool
	// IsGoodTaxpayer: a buen contribuyente is exempt from the retention (config.Company.IsGoodTaxpayer).
	IsGoodTaxpayer bool
}

// ApplySaleWithholdings adds the sale's detracción or retención to ImposedTaxes, after its lines were
// priced (Product.DetractionCode is read from products). The detracción applies to a factura above
// S/ 700 with a line subject to the SPOT system: the code with the largest amount among the sale's
// lines is the one the factura declares, at its rate over the whole total, rounded to whole soles as
// the deposit is made. Without a detracción, a retention agent withholds 3 % of a factura above S/ 700
// unless the company is a buen contribuyente. Amounts are in the sale's currency.
func (e *SaleOrder) ApplySaleWithholdings(products map[int32]production.Product, context SaleWithholdingContext) {
	totalPEN := e.TotalInPEN()
	if !context.IsFactura || totalPEN <= finance.DetractionMinAmountPEN {
		return
	}
	amountByDetractionCode := map[int16]int32{}
	for _, line := range e.Lines {
		if code := products[line.ProductID].DetractionCode; code > 0 {
			amountByDetractionCode[code] += core.QuantityAmount(core.UnpackQuantityLine(line.Quantity), line.Price, line.SubPrice)
		}
	}
	detractionCode, largestAmount := int16(0), int32(0)
	for code, amount := range amountByDetractionCode {
		// The lower code breaks a tie, so the same sale always declares the same code.
		if amount > largestAmount || (amount == largestAmount && code < detractionCode) {
			detractionCode, largestAmount = code, amount
		}
	}
	if detraction := finance.DetractionCodeOf(detractionCode); detraction != nil {
		amountPEN := int32(math.Round(float64(totalPEN)*float64(detraction.Rate)/10_000/100) * 100)
		e.ImposedTaxes = append(e.ImposedTaxes, SaleImposedTax{Type: finance.TaxGroupDetraction,
			Amount: e.amountFromPEN(amountPEN), Code: detraction.ID, Rate: detraction.Rate})
		return
	}
	if context.IsRetentionAgent && !context.IsGoodTaxpayer && totalPEN > finance.RetentionMinAmountPEN {
		e.ImposedTaxes = append(e.ImposedTaxes, SaleImposedTax{Type: finance.TaxGroupRetention,
			Amount: int32(math.Round(float64(e.TotalAmount) * finance.RetentionRate / 10_000)), Rate: finance.RetentionRate})
	}
}

// amountFromPEN restates a PEN amount in the sale's currency, at the rate the sale was converted with.
func (e *SaleOrder) amountFromPEN(amountPEN int32) int32 {
	if e.CurrencyType != finance.CurrencyUSD {
		return amountPEN
	}
	return finance.ConvertUnitPrice(amountPEN, finance.CurrencyPEN, finance.CurrencyUSD, e.ExchangeRate)
}

// CollectibleAmount is what the customer owes the company when the sale is created: the total plus the
// taxes charged on top, less those withheld (finance.ImposedTaxCashBehaviorOf). A detracción stays in it.
func (e *SaleOrder) CollectibleAmount() int32 {
	collectible := e.TotalAmount
	for _, imposedTax := range e.ImposedTaxes {
		switch finance.ImposedTaxCashBehaviorOf(imposedTax.Type) {
		case finance.ImposedTaxOnTop:
			collectible += imposedTax.Amount
		case finance.ImposedTaxWithheld:
			collectible -= imposedTax.Amount
		}
	}
	return collectible
}

// detractionEntry is the sale's detracción, nil without one.
func (e *SaleOrder) detractionEntry() *SaleImposedTax {
	for index := range e.ImposedTaxes {
		if e.ImposedTaxes[index].Type == finance.TaxGroupDetraction {
			return &e.ImposedTaxes[index]
		}
	}
	return nil
}

// OpenDetractionAmount is what of the detracción the customer has not deposited yet; part of DebtAmount.
func (e *SaleOrder) OpenDetractionAmount() int32 {
	if detraction := e.detractionEntry(); detraction != nil {
		return detraction.Amount - detraction.SettledAmount
	}
	return 0
}

// ApplyDetractionDeposit records a collection of the sale. Into the detracciones account it is the
// customer's deposit of the detracción: it may not exceed what is still open of it, and it is added to
// the entry's SettledAmount. Into any other account it may not collect the detracción: what stays owed
// must still cover the open detracción.
func (e *SaleOrder) ApplyDetractionDeposit(collectedAmount int32, remainingDebt int32, isDetractionsAccount bool) error {
	openDetraction := e.OpenDetractionAmount()
	if !isDetractionsAccount {
		if remainingDebt < openDetraction {
			return core.Err("La detracción pendiente se cobra solo en la cuenta de detracciones del Banco de la Nación.")
		}
		return nil
	}
	if collectedAmount > openDetraction {
		return core.Err("La cuenta de detracciones solo recibe la detracción pendiente de la venta.")
	}
	e.detractionEntry().SettledAmount += collectedAmount
	return nil
}
