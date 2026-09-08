package types

import "testing"

// The layout is [counter][rand:2][series:2]. SeriesID reads the tail, and the
// invoicing module builds a document id by replacing exactly those two digits —
// so if this drifts, documents stop pointing at their series.
func TestSeriesIDReadsTheTailOfTheID(t *testing.T) {
	cases := map[int64]int8{
		550342: 42,
		550300: 0,
		550399: 99,
		1:      1,
	}
	for saleOrderID, seriesID := range cases {
		sale := SaleOrder{ID: saleOrderID}
		if got := sale.SeriesID(); got != seriesID {
			t.Errorf("SeriesID() of %v = %v, want %v", saleOrderID, got, seriesID)
		}
	}
}

func TestMakeSaleOrderIDRejectsAnImpossibleSeries(t *testing.T) {
	for _, seriesID := range []int8{-1, 100} {
		if _, err := MakeSaleOrderID(1, seriesID); err == nil {
			t.Errorf("series %v was accepted", seriesID)
		}
	}
}

// The counter name has to stay what the ORM's autoincrement used: the sequence is
// live, and a fresh counter restarting at 1 would mint ids inside the range of
// rows already written.
func TestSaleOrderCounterKeepsTheORMName(t *testing.T) {
	if got := saleOrderCounterName(7); got != "x7_sale_order_0" {
		t.Errorf("counter name = %q, want x7_sale_order_0", got)
	}
}
