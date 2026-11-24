package types

import (
	"app/core"
	finance "app/finance/types"
	production "app/production/types"
	sales "app/sales/types"
	"testing"

	"github.com/ivanjoz/facturago/model"
)

func candyBoxProducts() map[int32]production.Product {
	return map[int32]production.Product{
		101: {ID: 101, Name: "CAJA DE CARAMELOS", SbuUnit: "unidad"},
	}
}

// A sale line packs both halves, and they were charged at different prices, so the document
// has to carry them as two lines. Emitting one line would either misprice the sub-units or
// push a fraction of a box into the XML.
func TestBuildLinesSplitsASubUnitSale(t *testing.T) {
	packed, err := core.PackQuantityLine(core.Quantity{Units: 1, Sub: 4}, 6)
	if err != nil {
		t.Fatalf("unexpected pack error: %v", err)
	}
	if packed != 1004 {
		t.Fatalf("expected the line to pack as 1004, got %v", packed)
	}

	order := &sales.SaleOrder{
		// Price is a box, SubPrice one candy.
		Lines: []sales.SaleOrderLine{{ProductID: 101, TaxGroup: finance.TaxGroupTaxed18, Quantity: packed, Price: 5000, SubPrice: 850, SubDivisor: 6}},
	}

	lines, lineProductIDs, err := buildLines(order, candyBoxProducts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected the line to split in two, got %v", len(lines))
	}
	if len(lineProductIDs) != 2 || lineProductIDs[0] != 101 || lineProductIDs[1] != 101 {
		t.Fatalf("both lines must map back to product 101, got %v", lineProductIDs)
	}

	// Every emitted quantity is a whole count — no sixth of a box reaches the XML.
	if lines[0].Quantity != model.Units(1) {
		t.Fatalf("expected 1 whole unit, got %v", lines[0].Quantity)
	}
	if lines[1].Quantity != model.Units(4) {
		t.Fatalf("expected 4 sub-units, got %v", lines[1].Quantity)
	}

	// Each half is valued at the price it was actually charged at.
	if got := int64(lines[0].Value + lines[0].IGV); got != 5000 {
		t.Fatalf("expected the box line to gross 5000, got %v", got)
	}
	if got := int64(lines[1].Value + lines[1].IGV); got != 3400 {
		t.Fatalf("expected the candy line to gross 4x850=3400, got %v", got)
	}

	// The sub-unit name is carried in the description: SUNAT catalog 03 has no entry for it.
	if lines[1].Description != "CAJA DE CARAMELOS (unidad)" {
		t.Fatalf("unexpected sub-unit description: %q", lines[1].Description)
	}
	if lines[0].Description != "CAJA DE CARAMELOS" {
		t.Fatalf("unexpected whole-unit description: %q", lines[0].Description)
	}
}

// The sale's TaxAmount is the sum of the IGV the document lines carry, never a split of the
// total: splitting 5000+3400 at once could round a cent away from what the comprobante declares.
func TestSaleOrderTaxAmountMatchesTheDocumentLines(t *testing.T) {
	order := &sales.SaleOrder{
		Lines: []sales.SaleOrderLine{
			{ProductID: 101, TaxGroup: finance.TaxGroupTaxed18, Quantity: 1004, Price: 5000, SubPrice: 850, SubDivisor: 6},
			{ProductID: 101, TaxGroup: finance.TaxGroupTaxed18, Quantity: 3000, Price: 1999, SubDivisor: 6},
		},
	}
	lines, _, err := buildLines(order, candyBoxProducts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	documentTax := int64(0)
	for _, line := range lines {
		documentTax += int64(line.IGV)
	}
	if got := SaleOrderTaxAmount(order); int64(got) != documentTax {
		t.Fatalf("expected the sale's tax to equal the document's %v, got %v", documentTax, got)
	}
	if SaleOrderTaxAmount(&sales.SaleOrder{}) != 0 {
		t.Fatal("expected an empty sale to carry no tax")
	}
}

// A whole-unit sale must stay a single line, so ordinary invoices are unchanged.
func TestBuildLinesKeepsWholeUnitSalesAsOneLine(t *testing.T) {
	packed, err := core.PackQuantityLine(core.Quantity{Units: 3}, core.QuantityDivisorNone)
	if err != nil {
		t.Fatalf("unexpected pack error: %v", err)
	}

	order := &sales.SaleOrder{
		Lines: []sales.SaleOrderLine{{ProductID: 101, TaxGroup: finance.TaxGroupTaxed18, Quantity: packed, Price: 5000, SubDivisor: core.QuantityDivisorNone}},
	}

	lines, _, err := buildLines(order, candyBoxProducts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected one line, got %v", len(lines))
	}
	if lines[0].Quantity != model.Units(3) {
		t.Fatalf("expected 3 units, got %v", lines[0].Quantity)
	}
}

// Selling only sub-units produces only the sub-unit line — a zero whole-unit half must not
// emit an empty line that SUNAT would reject.
func TestBuildLinesEmitsOnlyTheSubUnitLineWhenNoWholeUnitsSold(t *testing.T) {
	packed, err := core.PackQuantityLine(core.Quantity{Sub: 4}, 6)
	if err != nil {
		t.Fatalf("unexpected pack error: %v", err)
	}

	order := &sales.SaleOrder{
		Lines: []sales.SaleOrderLine{{ProductID: 101, TaxGroup: finance.TaxGroupTaxed18, Quantity: packed, Price: 5000, SubPrice: 850, SubDivisor: 6}},
	}

	lines, _, err := buildLines(order, candyBoxProducts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected one line, got %v", len(lines))
	}
	if lines[0].Quantity != model.Units(4) {
		t.Fatalf("expected 4 sub-units, got %v", lines[0].Quantity)
	}
	if lines[0].Description != "CAJA DE CARAMELOS (unidad)" {
		t.Fatalf("unexpected description: %q", lines[0].Description)
	}
}

// Each line is billed under its own tax group, and a free line declares its reference value under
// its group's free code (11, 21, 37) without charging it: the comprobante totals what the sale charged, and its IGV is
// the sale's TaxAmount.
func TestBuildLinesFollowsTheTaxGroups(t *testing.T) {
	order := &sales.SaleOrder{
		Lines: []sales.SaleOrderLine{
			{ProductID: 101, Quantity: 1000, Price: 11800, TaxGroup: finance.TaxGroupTaxed18},
			{ProductID: 101, Quantity: 1000, Price: 11000, TaxGroup: finance.TaxGroupTaxed10},
			{ProductID: 101, Quantity: 1000, Price: 5000, TaxGroup: finance.TaxGroupExonerated},
			{ProductID: 101, Quantity: 1000, Price: 4000, TaxGroup: finance.TaxGroupUnaffected},
			{ProductID: 101, Quantity: 1000, Price: 3000, TaxGroup: finance.TaxGroupExport},
			{ProductID: 101, Quantity: 2000, TaxGroup: finance.TaxGroupTaxed18},
			{ProductID: 101, Quantity: 1000, TaxGroup: finance.TaxGroupExonerated},
			{ProductID: 101, Quantity: 1000, TaxGroup: finance.TaxGroupUnaffected},
		},
		ImposedTaxes: []sales.SaleImposedTax{{Type: finance.TaxGroupFree, Line: 6, Amount: 2360},
			{Type: finance.TaxGroupFree, Line: 7, Amount: 700}, {Type: finance.TaxGroupFree, Line: 8, Amount: 300}},
	}
	lines, _, err := buildLines(order, candyBoxProducts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantTypes := []string{model.IgvTaxed, model.IgvTaxed, model.IgvExempt, model.IgvUnaffected, model.IgvExport, model.IgvFreeTaxed,
		model.IgvFreeExempt, igvFreeUnaffectedGift}
	wantIGV := []model.Cents{1800, 1000, 0, 0, 0, 360, 0, 0}
	for index, line := range lines {
		if line.IgvType != wantTypes[index] || line.IGV != wantIGV[index] {
			t.Fatalf("line %d: IgvType %q IGV %v, want %q %v", index+1, line.IgvType, line.IGV, wantTypes[index], wantIGV[index])
		}
	}
	if lines[6].IgvPercent != 0 || lines[7].IgvPercent != 0 {
		t.Fatalf("a free exonerado or inafecto line declares a zero rate: %+v %+v", lines[6], lines[7])
	}
	if lines[5].FreeUnitValue != 1000 || lines[5].UnitValue != 0 {
		t.Fatalf("the free line's reference unit value: %+v", lines[5])
	}

	document := testDocument()
	document.Lines = lines
	if err := model.CompleteTotals(document); err != nil {
		t.Fatalf("CompleteTotals: %v", err)
	}
	totals := document.Totals
	if totals.Payable != 34800 || totals.IGV != 2800 || totals.IGVFree != 360 || totals.Exempt != 5000 ||
		totals.Unaffected != 4000 || totals.Export != 3000 || totals.Free != 3000 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
	if got := SaleOrderTaxAmount(order); got != 2800 {
		t.Fatalf("the sale's IGV must be the charged IGV 2800, got %v", got)
	}
}

// A credit factura declares what the customer owes the seller: the open detracción is deposited into
// the detracciones account, so a factura owing only it is Contado.
func TestCreditTermsLeaveTheDetractionOut(t *testing.T) {
	order := &sales.SaleOrder{DebtAmount: 100000, PaymentDueDate: 20030,
		ImposedTaxes: []sales.SaleImposedTax{{Type: finance.TaxGroupDetraction, Amount: 12000}}}
	if terms := CreditTermsForSale(order, DocTypeFactura, 20000); terms.PendingAmount != 88000 {
		t.Fatalf("pending amount %v, want 88000", terms.PendingAmount)
	}
	order.DebtAmount = 12000
	if terms := CreditTermsForSale(order, DocTypeFactura, 20000); terms.PendingAmount != 0 {
		t.Fatalf("a factura owing only its detracción is Contado: %+v", terms)
	}
}
