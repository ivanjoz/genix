package types

import (
	"app/core"
	"testing"
)

func TestNormalizePurchaseDocument(t *testing.T) {
	today := core.FechaUnix()
	factura := func() PurchaseDocument {
		return PurchaseDocument{
			DocType: PurchaseDocTypeFactura, DocSeries: " f001 ", DocNumber: 123,
			DocIssueDate: today - 3, TaxableAmount: 10000, TaxAmount: 1800,
		}
	}

	valid := factura()
	if err := NormalizePurchaseDocument(&valid, 7, today, 0); err != nil {
		t.Fatalf("valid factura: %v", err)
	}
	if valid.DocSeries != "F001" || valid.CurrencyType != CurrencyPEN || valid.Total() != 11800 {
		t.Errorf("normalized factura = %+v", valid)
	}

	// No comprobante: the record's own date becomes the booking date and everything else clears.
	undocumented := PurchaseDocument{DocSeries: "X", TaxAmount: 500, CurrencyType: CurrencyUSD, ExchangeRate: 3700}
	if err := NormalizePurchaseDocument(&undocumented, 0, today-10, 0); err != nil {
		t.Fatalf("no comprobante: %v", err)
	}
	if undocumented != (PurchaseDocument{DocIssueDate: today - 10, CurrencyType: CurrencyUSD}) {
		t.Errorf("no comprobante = %+v", undocumented)
	}

	boleta := PurchaseDocument{
		DocType: PurchaseDocTypeBoleta, DocSeries: "B001", DocNumber: 9, DocIssueDate: today, UntaxedAmount: 5000,
	}
	if err := NormalizePurchaseDocument(&boleta, 7, today, 0); err != nil {
		t.Errorf("boleta with its amount untaxed: %v", err)
	}

	rejected := []struct {
		name     string
		mutate   func(*PurchaseDocument)
		supplier int32
		dueDate  int16
	}{
		{"unknown type", func(d *PurchaseDocument) { d.DocType = 7 }, 7, 0},
		{"no supplier", func(d *PurchaseDocument) {}, 0, 0},
		{"no number", func(d *PurchaseDocument) { d.DocNumber = 0 }, 7, 0},
		{"pipe in series", func(d *PurchaseDocument) { d.DocSeries = "F0|1" }, 7, 0},
		{"future issue date", func(d *PurchaseDocument) { d.DocIssueDate = today + 1 }, 7, 0},
		{"no issue date", func(d *PurchaseDocument) { d.DocIssueDate = 0 }, 7, 0},
		{"negative amount", func(d *PurchaseDocument) { d.OtherAmount = -1 }, 7, 0},
		{"IGV without base", func(d *PurchaseDocument) { d.TaxableAmount = 0; d.UntaxedAmount = 100 }, 7, 0},
		{"boleta with crédito fiscal", func(d *PurchaseDocument) { d.DocType = PurchaseDocTypeBoleta }, 7, 0},
		{"utility bill without due date", func(d *PurchaseDocument) { d.DocType = PurchaseDocTypeUtilityBill }, 7, 0},
		{"dollars without a rate", func(d *PurchaseDocument) { d.CurrencyType = CurrencyUSD }, 7, 0},
		{"unknown currency", func(d *PurchaseDocument) { d.CurrencyType = 9 }, 7, 0},
	}
	for _, testCase := range rejected {
		document := factura()
		testCase.mutate(&document)
		if err := NormalizePurchaseDocument(&document, testCase.supplier, today, testCase.dueDate); err == nil {
			t.Errorf("%s: expected an error", testCase.name)
		}
	}
}

func TestInventoryUnitCost(t *testing.T) {
	cases := []struct {
		name           string
		document       PurchaseDocument
		grossUnitPrice int32
		expected       int32
	}{
		{"factura drops the IGV", PurchaseDocument{DocType: PurchaseDocTypeFactura, TaxableAmount: 10000, TaxAmount: 1800}, 1180, 1000},
		{"untaxed lines share the proportion", PurchaseDocument{DocType: PurchaseDocTypeFactura, TaxableAmount: 10000, TaxAmount: 1800, UntaxedAmount: 2000}, 1380, 1200},
		{"boleta keeps the whole price", PurchaseDocument{DocType: PurchaseDocTypeBoleta, UntaxedAmount: 5000}, 500, 500},
		{"no comprobante keeps the whole price", PurchaseDocument{}, 750, 750},
		{"dollar factura is restated in soles", PurchaseDocument{DocType: PurchaseDocTypeFactura, TaxableAmount: 10000, TaxAmount: 1800, CurrencyType: CurrencyUSD, ExchangeRate: 3700}, 118, 370},
		{"dollars without a rate enter uncosted", PurchaseDocument{CurrencyType: CurrencyUSD}, 118, 0},
	}
	for _, testCase := range cases {
		if got := testCase.document.InventoryUnitCost(testCase.grossUnitPrice); got != testCase.expected {
			t.Errorf("%s: expected %v, got %v", testCase.name, testCase.expected, got)
		}
	}
}
