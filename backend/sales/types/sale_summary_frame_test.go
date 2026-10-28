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
		Date: 20000,
		// 3002: three units and two sixths; 2000: two whole units without a divisor.
		Lines: []SaleOrderLine{
			{ProductID: 77, Quantity: 3002, Price: 500, SubPrice: 100, SubDivisor: 6},
			{ProductID: 78, Quantity: 2000, Price: 1200},
			{ProductID: 0, Quantity: 1000, Price: 900},
		},
	}
	settledLines := []SaleSummaryFrame{
		{Date: 20000, ProductID: 77, SubDivisor: 6, AmountPEN: 1700, SoldUnits: 3, SoldSub: 2, SoldQuantity: 3 + 2.0/6},
		{Date: 20000, ProductID: 78, SubDivisor: core.QuantityDivisorNone, AmountPEN: 2400, SoldUnits: 2, SoldQuantity: 2},
	}
	withPending := func(lines []SaleSummaryFrame, isDeliveryPending, isPaymentPending bool) []SaleSummaryFrame {
		lines = append([]SaleSummaryFrame(nil), lines...)
		for i := range lines {
			if isDeliveryPending {
				lines[i].PendingUnits, lines[i].PendingSub, lines[i].PendingQuantity = lines[i].SoldUnits, lines[i].SoldSub, lines[i].SoldQuantity
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
		if got := sale.ToSaleSummaryFrames(nil); !reflect.DeepEqual(got, want) {
			t.Errorf("status %v:\n got %+v\nwant %+v", status, got, want)
		}
	}
}

// The client/product frame is the summary's lines, each with the sale's client: the two frames never
// disagree on what a sale adds.
func TestToSaleClientProductFramesAreTheSummaryLinesOfTheClient(t *testing.T) {
	sale := SaleOrder{
		Date: 20000, ClientID: 41, Status: OrderStatusPending,
		Lines: []SaleOrderLine{
			{ProductID: 77, Quantity: 3002, Price: 500, SubPrice: 100, SubDivisor: 6},
			{ProductID: 78, Quantity: 2000, Price: 1200},
		},
	}
	summaryLines := sale.ToSaleSummaryFrames(nil)
	clientLines := sale.ToSaleClientProductFrames(nil)
	if len(clientLines) != len(summaryLines) || len(clientLines) != 2 {
		t.Fatalf("%d client lines for %d summary lines", len(clientLines), len(summaryLines))
	}
	for i, summary := range summaryLines {
		want := SaleClientProductFrame{Date: summary.Date, ClientID: 41, ProductID: summary.ProductID,
			SubDivisor: summary.SubDivisor, AmountPEN: summary.AmountPEN, DebtAmountPEN: summary.DebtAmountPEN,
			SoldUnits: summary.SoldUnits, SoldSub: summary.SoldSub, PendingUnits: summary.PendingUnits, PendingSub: summary.PendingSub,
			SoldQuantity: summary.SoldQuantity, PendingQuantity: summary.PendingQuantity}
		if clientLines[i] != want {
			t.Errorf("line %d:\n got %+v\nwant %+v", i, clientLines[i], want)
		}
	}
}

// The sale_order DataFrame declarations compile with the table: a bad frame struct panics here.
func TestSaleOrderTableCompilesTheSaleSummaryFrame(t *testing.T) {
	if columnCount := len(db.MakeTable[SaleOrder]().GetColumns()); columnCount == 0 {
		t.Fatal("sale_order compiled without columns")
	}
}
