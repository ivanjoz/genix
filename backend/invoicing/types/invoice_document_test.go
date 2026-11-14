package types

import "testing"

// A sale id is [correlativo][series:2]. A document keeps the correlativo and puts
// its own series in the tail.
func TestDocumentIDForSaleSwapsOnlyTheSeries(t *testing.T) {
	const saleOrderID = int64(5501) // correlativo 55, sale series 01

	if got := DocumentIDForSale(saleOrderID, 1); got != saleOrderID {
		t.Errorf("boleta id = %v, want the sale's own id %v", got, saleOrderID)
	}
	if got := DocumentIDForSale(saleOrderID, 3); got != 5503 {
		t.Errorf("id under series 3 = %v, want 5503", got)
	}
}

func TestSeriesIDReadsOffTheTail(t *testing.T) {
	cases := map[int8]int64{1: 5501, 3: 5503, 99: 5599, 0: 5500}
	for seriesID, documentID := range cases {
		document := InvoiceDocument{ID: documentID}
		if got := document.SeriesID(); got != seriesID {
			t.Errorf("SeriesID() of %v = %v, want %v", documentID, got, seriesID)
		}
	}
}

// Issuing the same sale under the same series twice must land on the same key —
// that is what makes a duplicate a collision rather than a second document.
func TestDocumentIDIsStableForOneSaleAndSeries(t *testing.T) {
	const saleOrderID = int64(987654321)
	if DocumentIDForSale(saleOrderID, 7) != DocumentIDForSale(saleOrderID, 7) {
		t.Error("the same sale and series produced two different ids")
	}
}

// The invariant the whole key rests on: a sale carries the series it will be
// issued under, so the document that bills it takes the sale's own id.
func TestTheBillingDocumentIsKeyedByTheSale(t *testing.T) {
	// A sale created under series 2 (B001).
	const saleOrderID = int64(5502)
	if got := DocumentIDForSale(saleOrderID, 2); got != saleOrderID {
		t.Errorf("billing document id = %v, want the sale's own id %v", got, saleOrderID)
	}
}

// The correlativo is the head of the sale's id, so a document numbered from it
// reads its own number back without a column lookup. The counter that produces
// both is tested in sales/types.
func TestTheCorrelativoIsReadableFromTheDocumentID(t *testing.T) {
	const saleOrderID = int64(13002) // correlativo 130, series 02
	documentID := DocumentIDForSale(saleOrderID, 2)

	if SalePrefix(documentID) != 130 {
		t.Errorf("correlativo in %v = %v, want 130", documentID, SalePrefix(documentID))
	}
}

func TestNumberIsTheSunatSpelling(t *testing.T) {
	document := InvoiceDocument{ID: 5501, Correlativo: 123}
	if got := document.Number("F001"); got != "F001-123" {
		t.Errorf("Number() = %q, want F001-123", got)
	}
}
