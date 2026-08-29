package invoicing

import (
	business "app/business/types"
	"app/core"
	sales "app/sales/types"
	"testing"

	"github.com/ivanjoz/facturago/model"
)

func candyBoxProducts() map[int32]business.Product {
	return map[int32]business.Product{
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
		DetailProductsIDs: []int32{101},
		DetailQuantities:  []int32{packed},
		DetailPrices:      []int32{5000}, // a box
		DetailSubPrices:   []int32{850},  // one candy
		DetailSubDivisor:  []int16{6},
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

// A whole-unit sale must stay a single line, so ordinary invoices are unchanged.
func TestBuildLinesKeepsWholeUnitSalesAsOneLine(t *testing.T) {
	packed, err := core.PackQuantityLine(core.Quantity{Units: 3}, core.QuantityDivisorNone)
	if err != nil {
		t.Fatalf("unexpected pack error: %v", err)
	}

	order := &sales.SaleOrder{
		DetailProductsIDs: []int32{101},
		DetailQuantities:  []int32{packed},
		DetailPrices:      []int32{5000},
		DetailSubDivisor:  []int16{core.QuantityDivisorNone},
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
		DetailProductsIDs: []int32{101},
		DetailQuantities:  []int32{packed},
		DetailPrices:      []int32{5000},
		DetailSubPrices:   []int32{850},
		DetailSubDivisor:  []int16{6},
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
