package types

import "testing"

// A sale id is [counter][rand:2][series:2]. A document keeps the counter and the
// random digits and swaps in its own series, which is the whole reason a sale can
// carry more than one document without a second key column.
func TestDocumentIDForSaleSwapsOnlyTheSeries(t *testing.T) {
	const saleOrderID = int64(550301) // counter 55, random 03, sale series 01

	boleta := DocumentIDForSale(saleOrderID, 1)
	creditNote := DocumentIDForSale(saleOrderID, 3)

	if boleta != saleOrderID {
		t.Errorf("boleta id = %v, want the sale's own id %v", boleta, saleOrderID)
	}
	if creditNote != 550303 {
		t.Errorf("credit note id = %v, want 550303", creditNote)
	}
	if boleta == creditNote {
		t.Error("two documents for one sale collided on the key")
	}
	// The counter and the random digits are what they share.
	if SalePrefix(boleta) != SalePrefix(creditNote) {
		t.Error("the two documents no longer share the sale's counter")
	}
}

func TestSeriesIDReadsOffTheTail(t *testing.T) {
	cases := map[int8]int64{1: 550301, 3: 550303, 99: 550399, 0: 550300}
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
	const saleOrderID = int64(550302)
	if got := DocumentIDForSale(saleOrderID, 2); got != saleOrderID {
		t.Errorf("billing document id = %v, want the sale's own id %v", got, saleOrderID)
	}
}

// The correlativo is the head of the sale's id, so a document numbered from it
// reads its own number back without a column lookup. The counter that produces
// both is tested in sales/types.
func TestTheCorrelativoIsReadableFromTheDocumentID(t *testing.T) {
	const saleOrderID = int64(1301102) // correlativo 130, random 11, series 02
	documentID := DocumentIDForSale(saleOrderID, 2)

	if SalePrefix(documentID)/100 != 130 {
		t.Errorf("correlativo in %v = %v, want 130", documentID, SalePrefix(documentID)/100)
	}
}

func TestNumberIsTheSunatSpelling(t *testing.T) {
	document := InvoiceDocument{ID: 550301, Correlativo: 123}
	if got := document.Number("F001"); got != "F001-123" {
		t.Errorf("Number() = %q, want F001-123", got)
	}
}
