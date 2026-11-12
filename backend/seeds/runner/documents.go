package runner

import (
	"app/core"
	financeTypes "app/finance/types"
	_ "embed"
	"math"
)

// PeriodTaxTotals accumulates what the month's IGV and income-tax prepayment are computed from,
// so the tax expenses booked the next month match the sales and purchases actually generated.
type PeriodTaxTotals struct {
	SalesGross     int64
	PurchaseTaxIGV int64
}

// MakePurchaseDocument builds a supplier comprobante for a gross total. A document with tax credit
// (factura, ticket, utility bill) splits it into base and 18% IGV; a boleta or fee receipt carries
// it whole as untaxed; no document returns the empty shape.
func MakePurchaseDocument(docType int8, series string, number int64, issueDay int16, total int32) financeTypes.PurchaseDocument {
	document := financeTypes.PurchaseDocument{DocType: docType, CurrencyType: financeTypes.CurrencyPEN}
	if docType == financeTypes.PurchaseDocTypeNone {
		return document
	}
	document.DocSeries, document.DocNumber, document.DocIssueDate = series, number, issueDay
	switch docType {
	case financeTypes.PurchaseDocTypeFactura, financeTypes.PurchaseDocTypeTicket, financeTypes.PurchaseDocTypeUtilityBill:
		document.TaxableAmount = int32(math.Round(float64(total) / 1.18))
		document.TaxAmount = total - document.TaxableAmount
	default:
		document.UntaxedAmount = total
	}
	return document
}

// nextDocNumber hands out a supplier's comprobante numbers in order. The first one starts at a
// random point, as a supplier's invoice numbering would.
func (run *Run) NextDocNumber(documentKey string) int64 {
	if run.State.NextDocNumberByKey[documentKey] == 0 {
		run.State.NextDocNumberByKey[documentKey] = int64(1000 + run.Random.Intn(90000))
	}
	run.State.NextDocNumberByKey[documentKey]++
	return run.State.NextDocNumberByKey[documentKey]
}

func (run *Run) PeriodTaxTotals(unixDay int16) *PeriodTaxTotals {
	period := core.PeriodOfDay(unixDay)
	if run.State.TaxTotalsByPeriod[period] == nil {
		run.State.TaxTotalsByPeriod[period] = &PeriodTaxTotals{}
	}
	return run.State.TaxTotalsByPeriod[period]
}

func (run *Run) AddPurchaseTax(issueDay int16, taxAmount int32) {
	run.PeriodTaxTotals(issueDay).PurchaseTaxIGV += int64(taxAmount)
}
