package types

import "testing"

// The layout is [correlativo][rand:2][series:2]. SeriesID reads the tail, and the
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

// The id carries the correlativo of the series, so reading it back is what lets a
// sale name its own comprobante — 1301102 is F001-130 — with nothing to look up.
func TestCorrelativoIsTheHeadOfTheID(t *testing.T) {
	cases := map[int64]int64{
		1301102: 130,
		20701:   2,
		10002:   1,
		102:     0, // a legacy id from before the layout: no correlativo in it
	}
	for saleOrderID, correlativo := range cases {
		sale := SaleOrder{ID: saleOrderID}
		if got := sale.Correlativo(); got != correlativo {
			t.Errorf("Correlativo() of %v = %v, want %v", saleOrderID, got, correlativo)
		}
	}
}

// One counter per company and series — the scope SUNAT requires the numbering to be
// unique in — and it is the same sequence the comprobante is numbered from, because
// they are the same number.
func TestIssueSeriesCounterIsPerCompanyAndSeries(t *testing.T) {
	if got := IssueSeriesCounterName(7, 2); got != "cpe_7_2" {
		t.Errorf("counter name = %q, want cpe_7_2", got)
	}
	if IssueSeriesCounterName(7, 1) == IssueSeriesCounterName(7, 2) {
		t.Error("two series shared a counter")
	}
	// The sale that issues nothing still needs ids, and they come from their own
	// bucket so they never consume a correlativo anybody declares.
	if IssueSeriesCounterName(7, 0) == IssueSeriesCounterName(7, 1) {
		t.Error("the no-series sales shared the boleta counter")
	}
}
