package types

import (
	"app/core"
	"app/db"
	"reflect"
	"testing"
)

// The sale's status alone decides what it adds to its day: undelivered quantities stay pending,
// unpaid amounts stay as debt, an annulled sale adds nothing (the frame takes back what it added).
func TestToSaleSummaryFramesFollowsTheStatus(t *testing.T) {
	sale := SaleOrder{
		Date:              20000,
		DetailProductsIDs: []int32{77, 78, 0},
		// 3002: three units and two sixths; 2000: two whole units without a divisor.
		DetailQuantities: []int32{3002, 2000, 1000},
		DetailPrices:     []int32{500, 1200, 900},
		DetailSubPrices:  []int32{100, 0, 0},
		DetailSubDivisor: []int16{6, 0, 0},
	}
	settledLines := []SaleSummaryFrame{
		{Date: 20000, ProductID: 77, SubDivisor: 6, AmountPEN: 1700, SoldUnits: 3, SoldSub: 2},
		{Date: 20000, ProductID: 78, SubDivisor: core.QuantityDivisorNone, AmountPEN: 2400, SoldUnits: 2},
	}
	withPending := func(lines []SaleSummaryFrame, isDeliveryPending, isPaymentPending bool) []SaleSummaryFrame {
		lines = append([]SaleSummaryFrame(nil), lines...)
		for i := range lines {
			if isDeliveryPending {
				lines[i].PendingUnits, lines[i].PendingSub = lines[i].SoldUnits, lines[i].SoldSub
			}
			if isPaymentPending {
				lines[i].DebtAmountPEN = lines[i].AmountPEN
			}
		}
		return lines
	}
	for status, want := range map[int8][]SaleSummaryFrame{
		OrderStatusPending:   withPending(settledLines, true, true),
		OrderStatusPaid:      withPending(settledLines, true, false),
		OrderStatusDelivered: withPending(settledLines, false, true),
		OrderStatusCompleted: settledLines,
		OrderStatusAnnulled:  nil,
	} {
		sale.Status = status
		if got := sale.ToSaleSummaryFrames(); !reflect.DeepEqual(got, want) {
			t.Errorf("status %v:\n got %+v\nwant %+v", status, got, want)
		}
	}
}

// The "sale-summary" declaration compiles with the table: a bad frame struct panics here.
func TestSaleOrderTableCompilesTheSaleSummaryFrame(t *testing.T) {
	if columnCount := len(db.MakeTable[SaleOrder]().GetColumns()); columnCount == 0 {
		t.Fatal("sale_order compiled without columns")
	}
}
