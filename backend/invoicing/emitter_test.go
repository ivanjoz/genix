package invoicing

import (
	"app/invoicing/types"
	sales "app/sales/types"
	"testing"
	"time"

	"github.com/ivanjoz/facturago"
	"github.com/ivanjoz/facturago/model"
)

// TestSplitGrossAmount covers the one piece of arithmetic this module owns.
//
// The ERP prices with IGV included and SUNAT wants the two halves declared
// separately. The split has to be exact — net plus tax back to the cent — or the
// invoice totals something other than what the customer paid.
func TestSplitGrossAmount(t *testing.T) {
	cases := []struct {
		gross, net, tax int64
	}{
		{11_800, 10_000, 1_800}, // S/ 118.00 → 100.00 + 18.00
		{10_000, 8_475, 1_525},  // S/ 100.00 → the price that does not divide evenly
		{100, 85, 15},
		{1, 1, 0},
		{0, 0, 0},
	}

	for _, testCase := range cases {
		net, tax := splitGrossAmount(testCase.gross)
		if net != testCase.net || tax != testCase.tax {
			t.Errorf("splitGrossAmount(%d) = (%d, %d), want (%d, %d)",
				testCase.gross, net, tax, testCase.net, testCase.tax)
		}
	}
}

// TestSplitGrossAmountIsExact is the property that matters more than any single
// case: whatever the amount, the two halves add back to it.
func TestSplitGrossAmountIsExact(t *testing.T) {
	for gross := int64(0); gross < 50_000; gross += 7 {
		net, tax := splitGrossAmount(gross)
		if net+tax != gross {
			t.Fatalf("splitGrossAmount(%d) = (%d, %d), which sums to %d", gross, net, tax, net+tax)
		}
	}
}

func testSeries() *types.InvoiceSeries {
	return &types.InvoiceSeries{
		DocType: types.DocTypeFactura, SeriesID: 1,
		SeriesCode: "F001", SiteID: 1, Status: 1,
	}
}

func testDocument() *model.Document {
	net, tax := splitGrossAmount(23_600) // two units at S/ 118.00
	return &model.Document{
		Type:     model.Factura,
		Series:   "F001",
		IssuedAt: time.Unix(1_700_000_000, 0),
		Currency: model.DefaultCurrency,
		Payment:  model.PaymentCash,
		Customer: model.Party{
			DocType: model.IDDocRUC, DocNumber: "20000000002", LegalName: "CLIENTE S.A.C.",
		},
		Lines: []model.Line{{
			Description: "PRODUCTO 1",
			Quantity:    model.Units(2),
			UnitCode:    model.DefaultUnitCode,
			IgvType:     model.IgvTaxed,
			IgvPercent:  model.Percent(igvPercentHundredths),
			Value:       model.Cents(net),
			IGV:         model.Cents(tax),
		}},
	}
}

// A note is issued under its own series, so it is not keyed by the sale — it
// reaches it through the document it corrects.
func TestANoteReachesTheSaleThroughTheDocumentItCorrects(t *testing.T) {
	const saleOrderID = int64(550301)
	note := types.InvoiceDocument{
		ID:            types.DocumentIDForSale(saleOrderID, 3), // FC01
		AffectedDocID: saleOrderID,
	}
	if note.ID == saleOrderID {
		t.Fatal("the note took the same key as the document it corrects")
	}
	if saleIDOfDocument(&note) != saleOrderID {
		t.Errorf("saleIDOfDocument = %v, want %v", saleIDOfDocument(&note), saleOrderID)
	}
}

// TestRowRecordsWhatOnlyItKnows checks the flattening that is left now that the
// lines are rebuilt from the sale rather than copied.
//
// What the row must carry is the identity and the money: the id that ties it to a
// sale and a series, the number it was issued under, and the totals every list
// reads. Anything it got wrong here would be wrong on every retry and in every
// report.
func TestRowRecordsWhatOnlyItKnows(t *testing.T) {
	original := testDocument()
	if err := model.CompleteTotals(original); err != nil {
		t.Fatalf("CompleteTotals: %v", err)
	}

	// The sale was created under series 1, which is the series it is issued in —
	// so the document is keyed by the sale itself.
	order := &sales.SaleOrder{ID: 550301, ClientID: 7, DetailProductsIDs: []int32{101}}
	row := rowFromDocument(1, 1, order, testSeries(), original, 123)

	if row.ID != order.ID {
		t.Errorf("id = %v, want the sale's own id %v", row.ID, order.ID)
	}
	if row.SeriesID() != testSeries().SeriesID {
		t.Errorf("series = %v, want %v", row.SeriesID(), testSeries().SeriesID)
	}
	// And that is how the sale is found again: no stored reference.
	if saleIDOfDocument(&row) != order.ID {
		t.Errorf("saleIDOfDocument = %v, want %v", saleIDOfDocument(&row), order.ID)
	}
	if row.Correlativo != 123 {
		t.Errorf("correlativo = %v, want 123", row.Correlativo)
	}
	if row.Number("F001") != "F001-123" {
		t.Errorf("number = %q, want F001-123", row.Number("F001"))
	}
	if row.TotalAmount != int64(original.Totals.Payable) {
		t.Errorf("total = %v, want %v", row.TotalAmount, original.Totals.Payable)
	}
	if row.TaxAmount != int64(original.Totals.TotalTaxes) {
		t.Errorf("tax = %v, want %v", row.TaxAmount, original.Totals.TotalTaxes)
	}
	if row.State != types.InvoicePending {
		t.Errorf("state = %v, want pending", row.State)
	}
}

// The document still has to serialize — that is what the reserve path validates
// before it spends a number.
func TestDocumentSerializes(t *testing.T) {
	document := testDocument()
	if err := model.CompleteTotals(document); err != nil {
		t.Fatalf("CompleteTotals: %v", err)
	}
	document.Correlativo = 123
	if problems := model.ValidateDocument(document); len(problems) > 0 {
		t.Fatalf("the document is not valid: %v", problems)
	}

	issuer := model.Issuer{
		RUC: "20000000001", LegalName: "EMPRESA DEMO S.A.C.",
		Address: model.Address{Ubigeo: "150101", Line: "AV. LIMA 100"},
	}
	if _, err := facturago.BuildXML(issuer, document); err != nil {
		t.Fatalf("the document does not serialize: %v", err)
	}
}

// TestRowKeepsTheAmountCharged guards the reason the split exists: the invoice
// must total exactly what the till charged, not a re-derived approximation.
func TestRowKeepsTheAmountCharged(t *testing.T) {
	const charged = int64(10_000) // S/ 100.00, a price that does not divide evenly

	net, tax := splitGrossAmount(charged)
	document := testDocument()
	document.Lines = []model.Line{{
		Description: "PRODUCTO 1",
		Quantity:    model.Units(1),
		UnitCode:    model.DefaultUnitCode,
		IgvType:     model.IgvTaxed,
		IgvPercent:  model.Percent(igvPercentHundredths),
		Value:       model.Cents(net),
		IGV:         model.Cents(tax),
	}}

	if err := model.CompleteTotals(document); err != nil {
		t.Fatalf("CompleteTotals: %v", err)
	}
	if int64(document.Totals.Payable) != charged {
		t.Errorf("payable = %d, want exactly the %d that was charged",
			document.Totals.Payable, charged)
	}
}

func TestIdentityDocTypeOfInfersFromShape(t *testing.T) {
	cases := map[string]string{
		"20000000002": model.IDDocRUC,
		"12345678":    model.IDDocDNI,
		"":            "",
		"X1234":       model.IDDocForeign,
	}
	for registryNumber, want := range cases {
		if got := identityDocTypeOf(registryNumber); got != want {
			t.Errorf("identityDocTypeOf(%q) = %q, want %q", registryNumber, got, want)
		}
	}
}
