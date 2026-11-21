package types

import (
	finance "app/finance/types"
	"testing"
)

func TestToPurchaseOrderSummaryFrames(t *testing.T) {
	order := PurchaseOrder{Status: PurchaseOrderStatusConfirmed, Date: 20000, SiteID: 1,
		CurrencyType: finance.CurrencyPEN, TotalAmount: 23600, TaxAmount: 3600}
	want := PurchaseOrderSummaryFrame{Date: 20000, SiteID: 1, CategoryID: finance.ExpenseCategoryMerchandise,
		AmountPEN: 23600, TaxAmountPEN: 3600}
	if lines := order.ToPurchaseOrderSummaryFrames(nil); len(lines) != 1 || lines[0] != want {
		t.Fatalf("confirmed order: want %+v, got %+v", want, lines)
	}

	order.Status = PurchaseOrderStatusFulfilled
	if lines := order.ToPurchaseOrderSummaryFrames(nil); len(lines) != 1 {
		t.Errorf("a fulfilled order still counts, got %+v", lines)
	}

	for _, status := range []int8{PurchaseOrderStatusPending, PurchaseOrderStatusCanceled} {
		order.Status = status
		if lines := order.ToPurchaseOrderSummaryFrames(nil); len(lines) != 0 {
			t.Errorf("status %d is not committed, got %+v", status, lines)
		}
	}

	dollarOrder := PurchaseOrder{Status: PurchaseOrderStatusConfirmed, CurrencyType: finance.CurrencyUSD, TotalAmount: 1000}
	if lines := dollarOrder.ToPurchaseOrderSummaryFrames(nil); len(lines) != 0 {
		t.Errorf("a dollar order without a rate adds nothing, got %+v", lines)
	}
	dollarOrder.ExchangeRate = 3700
	if lines := dollarOrder.ToPurchaseOrderSummaryFrames(nil); len(lines) != 1 || lines[0].AmountPEN != 3700 {
		t.Errorf("a dollar order is restated at its rate, got %+v", lines)
	}
}
