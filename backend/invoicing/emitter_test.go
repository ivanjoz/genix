package invoicing

import (
	"app/invoicing/types"
	sales "app/sales/types"
	"testing"

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

// TestRowRoundTripsIntoAValidDocument is the check that matters for this module:
// a document flattened into its row and rebuilt from it has to still be one
// SUNAT would accept.
//
// It is not a formality. The row is what a retry works from — possibly days
// later, after the product was renamed — so anything the flattening loses is
// lost from every retry, and the failure would only appear as a rejection.
func TestRowRoundTripsIntoAValidDocument(t *testing.T) {
	original := testDocument()
	if err := model.CompleteTotals(original); err != nil {
		t.Fatalf("CompleteTotals: %v", err)
	}

	order := &sales.SaleOrder{ID: 55, ClientID: 7, DetailProductsIDs: []int32{101}}
	row := rowFromDocument(1, 1, order, testSeries(), original)
	// The ORM assigns the key on insert, and the number is read off it; the test
	// stands in for that.
	row.ID = int64(row.DocTypeSeries)*100_000_000_00000 + 123

	rebuilt := DocumentFromRow(&row)
	if err := model.CompleteTotals(rebuilt); err != nil {
		t.Fatalf("CompleteTotals on the rebuilt document: %v", err)
	}
	if problems := model.ValidateDocument(rebuilt); len(problems) > 0 {
		t.Fatalf("the rebuilt document is not valid: %v", problems)
	}

	if rebuilt.Totals.Payable != original.Totals.Payable {
		t.Errorf("payable = %d, want %d", rebuilt.Totals.Payable, original.Totals.Payable)
	}
	if rebuilt.Totals.IGV != original.Totals.IGV {
		t.Errorf("IGV = %d, want %d", rebuilt.Totals.IGV, original.Totals.IGV)
	}
	if rebuilt.Number() != "F001-123" {
		t.Errorf("number = %q, want F001-123", rebuilt.Number())
	}

	// And it has to serialize: a document that cannot be built cannot be sent.
	issuer := model.Issuer{
		RUC: "20000000001", LegalName: "EMPRESA DEMO S.A.C.",
		Address: model.Address{Ubigeo: "150101", Line: "AV. LIMA 100"},
	}
	if _, err := facturago.BuildXML(issuer, rebuilt); err != nil {
		t.Fatalf("the rebuilt document does not serialize: %v", err)
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

func TestIdentityDocumentCodesRoundTrip(t *testing.T) {
	for _, docType := range []string{
		model.IDDocNone, model.IDDocDNI, model.IDDocForeign, model.IDDocRUC, model.IDDocPassport,
	} {
		if got := identityDocTypeOfCode(identityDocTypeCode(docType)); got != docType {
			t.Errorf("identity doc %q round-tripped to %q", docType, got)
		}
	}
}

func TestIgvTypeCodesRoundTrip(t *testing.T) {
	for _, igvType := range []string{
		model.IgvTaxed, model.IgvExempt, model.IgvUnaffected,
		model.IgvExport, model.IgvFreeTaxed, model.IgvIVAP,
	} {
		if got := igvTypeOfCode(igvTypeCode(igvType)); got != igvType {
			t.Errorf("IGV type %q round-tripped to %q", igvType, got)
		}
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
