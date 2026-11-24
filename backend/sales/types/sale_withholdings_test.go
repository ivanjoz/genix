package types

import (
	finance "app/finance/types"
	production "app/production/types"
	"testing"
)

func TestApplySaleWithholdings(t *testing.T) {
	products := map[int32]production.Product{
		1: {ID: 1, DetractionCode: 37}, // demás servicios, 12 %
		2: {ID: 2, DetractionCode: 27}, // transporte de carga, 4 %
		3: {ID: 3},
	}
	newSale := func(lines ...SaleOrderLine) SaleOrder {
		sale := SaleOrder{CurrencyType: finance.CurrencyPEN, Lines: lines}
		for _, line := range lines {
			sale.TotalAmount += line.Price * (line.Quantity / 1000)
		}
		return sale
	}
	factura := SaleWithholdingContext{IsFactura: true}

	// S/ 1 000.50: 12 % of the whole total, the code of the larger SPOT amount, in whole soles.
	sale := newSale(SaleOrderLine{ProductID: 1, Quantity: 1000, Price: 70000}, SaleOrderLine{ProductID: 2, Quantity: 1000, Price: 20000},
		SaleOrderLine{ProductID: 3, Quantity: 1000, Price: 10050})
	sale.ApplySaleWithholdings(products, factura)
	if len(sale.ImposedTaxes) != 1 || sale.ImposedTaxes[0] != (SaleImposedTax{Type: finance.TaxGroupDetraction, Amount: 12000, Code: 37, Rate: 1200}) {
		t.Fatalf("detracción: %+v", sale.ImposedTaxes)
	}
	if sale.CollectibleAmount() != sale.TotalAmount || sale.OpenDetractionAmount() != 12000 {
		t.Fatalf("a detracción stays owed: collectible %v, open %v", sale.CollectibleAmount(), sale.OpenDetractionAmount())
	}

	for name, context := range map[string]SaleWithholdingContext{"a boleta": {}, "S/ 700 or less": factura} {
		small := newSale(SaleOrderLine{ProductID: 1, Quantity: 1000, Price: 70000})
		if name == "a boleta" {
			small = newSale(SaleOrderLine{ProductID: 1, Quantity: 2000, Price: 70000})
		}
		small.ApplySaleWithholdings(products, context)
		if len(small.ImposedTaxes) != 0 {
			t.Fatalf("%s carries no detracción: %+v", name, small.ImposedTaxes)
		}
	}

	// No SPOT line: a retention agent withholds 3 %, unless the company is a buen contribuyente.
	retained := newSale(SaleOrderLine{ProductID: 3, Quantity: 1000, Price: 100000})
	retained.ApplySaleWithholdings(products, SaleWithholdingContext{IsFactura: true, IsRetentionAgent: true})
	if len(retained.ImposedTaxes) != 1 || retained.ImposedTaxes[0].Type != finance.TaxGroupRetention || retained.ImposedTaxes[0].Amount != 3000 ||
		retained.CollectibleAmount() != 97000 {
		t.Fatalf("retención: %+v, collectible %v", retained.ImposedTaxes, retained.CollectibleAmount())
	}
	goodTaxpayer := newSale(SaleOrderLine{ProductID: 3, Quantity: 1000, Price: 100000})
	goodTaxpayer.ApplySaleWithholdings(products, SaleWithholdingContext{IsFactura: true, IsRetentionAgent: true, IsGoodTaxpayer: true})
	if len(goodTaxpayer.ImposedTaxes) != 0 {
		t.Fatalf("a buen contribuyente is not retained: %+v", goodTaxpayer.ImposedTaxes)
	}
	// A sale under detracción is never retained as well.
	both := newSale(SaleOrderLine{ProductID: 1, Quantity: 1000, Price: 100000})
	both.ApplySaleWithholdings(products, SaleWithholdingContext{IsFactura: true, IsRetentionAgent: true})
	if len(both.ImposedTaxes) != 1 || both.ImposedTaxes[0].Type != finance.TaxGroupDetraction {
		t.Fatalf("detracción and retención never together: %+v", both.ImposedTaxes)
	}
}

func TestApplyDetractionDeposit(t *testing.T) {
	sale := SaleOrder{TotalAmount: 100000, DebtAmount: 100000,
		ImposedTaxes: []SaleImposedTax{{Type: finance.TaxGroupDetraction, Amount: 12000}}}
	if err := sale.ApplyDetractionDeposit(95000, 5000, false); err == nil {
		t.Fatal("a till may not collect the open detracción")
	}
	if err := sale.ApplyDetractionDeposit(88000, 12000, false); err != nil {
		t.Fatal(err)
	}
	if err := sale.ApplyDetractionDeposit(13000, 0, true); err == nil {
		t.Fatal("the detracciones account takes only the open detracción")
	}
	if err := sale.ApplyDetractionDeposit(12000, 0, true); err != nil || sale.OpenDetractionAmount() != 0 {
		t.Fatalf("the deposit settles the detracción: %v, open %v", err, sale.OpenDetractionAmount())
	}
}

func TestSaleCashDocumentMemos(t *testing.T) {
	sale := SaleOrder{Date: 20773, Status: OrderStatusPending, TotalAmount: 100000, DebtAmount: 100000, PaymentDueDate: 20803,
		Lines:        []SaleOrderLine{{ProductID: 1, Quantity: 1000, Price: 100000, TaxGroup: finance.TaxGroupTaxed18}},
		ImposedTaxes: []SaleImposedTax{{Type: finance.TaxGroupDetraction, Amount: 12000, SettledAmount: 2000}}}
	lines := sale.ToCashDocumentFrames(nil)
	if len(lines) != 2 || lines[0].Amount != 100000 || lines[1].TaxGroup != finance.TaxGroupDetraction ||
		lines[1].Kind != finance.CashDocumentKindSaleOpen || lines[1].Amount != 10000 || lines[1].DueOffset != 30 {
		t.Fatalf("the open detracción is an open memo beside the debt: %+v", lines)
	}
	selection := finance.CashDocumentSelection{Kind: finance.CashDocumentKindSaleOpen, Dates: map[int16]bool{20773: true}, AllSites: true}
	if selection.OpenAmountOf(lines) != 100000 {
		t.Fatalf("a memo is never summed into the document: %v", selection.OpenAmountOf(lines))
	}

	retained := SaleOrder{Date: 20773, Status: OrderStatusCompleted, TotalAmount: 100000,
		Lines:        []SaleOrderLine{{ProductID: 1, Quantity: 1000, Price: 100000, TaxGroup: finance.TaxGroupTaxed18}},
		ImposedTaxes: []SaleImposedTax{{Type: finance.TaxGroupRetention, Amount: 3000}}}
	if lines := retained.ToCashDocumentFrames(nil); len(lines) != 2 || lines[1].TaxGroup != finance.TaxGroupRetention ||
		lines[1].Kind != finance.CashDocumentKindSaleSettled || lines[1].Amount != 3000 {
		t.Fatalf("the retención is a settled memo: %+v", lines)
	}
}
