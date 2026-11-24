package types

import (
	finance "app/finance/types"
	"testing"
)

func TestSaleCashDocumentFrames(t *testing.T) {
	// A taxed line of 1 000.00, an exonerated one of 180.00 and a free line worth 59.00.
	sale := SaleOrder{Date: 20773, SiteID: 1, Status: OrderStatusDelivered, TotalAmount: 118000, DebtAmount: 68000,
		PaymentDueDate: 20803, CurrencyType: finance.CurrencyPEN, ClientID: 5000,
		Lines: []SaleOrderLine{
			{ProductID: 1, Quantity: 1000, Price: 100000, TaxGroup: finance.TaxGroupTaxed18},
			{ProductID: 2, Quantity: 1000, Price: 18000, TaxGroup: finance.TaxGroupExonerated},
			{ProductID: 3, Quantity: 1000, TaxGroup: finance.TaxGroupTaxed18},
			{ProductID: 4, Quantity: 1000, TaxGroup: finance.TaxGroupExonerated},
		},
		// A free exonerado line owes no IGV: it adds no memo row.
		ImposedTaxes: []SaleImposedTax{{Type: finance.TaxGroupFree, Line: 3, Amount: 5900}, {Type: finance.TaxGroupFree, Line: 4, Amount: 3000}},
	}
	type rowSummary struct {
		Kind, TaxGroup int8
		DueOffset      int16
		Amount         int32
	}
	summarize := func(lines []finance.CashDocumentFrame) []rowSummary {
		rows := []rowSummary{}
		for _, line := range lines {
			rows = append(rows, rowSummary{line.Kind, line.TaxGroup, line.DueOffset, line.Amount})
		}
		return rows
	}
	// The open 68 000 is spread over the charged groups; the free group is always settled.
	want := []rowSummary{
		{finance.CashDocumentKindSaleOpen, finance.TaxGroupTaxed18, 30, 57627},
		{finance.CashDocumentKindSaleSettled, finance.TaxGroupTaxed18, 0, 42373},
		{finance.CashDocumentKindSaleOpen, finance.TaxGroupExonerated, 30, 10373},
		{finance.CashDocumentKindSaleSettled, finance.TaxGroupExonerated, 0, 7627},
		{finance.CashDocumentKindSaleSettled, finance.TaxGroupFree, 0, 5900},
	}
	if got := summarize(sale.ToCashDocumentFrames(nil)); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] ||
		got[2] != want[2] || got[3] != want[3] || got[4] != want[4] {
		t.Fatalf("a partly collected credit sale: %+v", got)
	}
	debt := sale.ToSaleClientDebtFrames(nil)
	if len(debt) != 1 || debt[0].ClientBucket != 1 || debt[0].ClientID != 5000 || debt[0].DebtLocal != 68000 {
		t.Fatalf("its client's debt: %+v", debt)
	}

	sale.Status = OrderStatusCompleted
	for _, row := range summarize(sale.ToCashDocumentFrames(nil)) {
		if row.Kind != finance.CashDocumentKindSaleSettled {
			t.Fatalf("a paid sale is all settled: %+v", row)
		}
	}
	if debt := sale.ToSaleClientDebtFrames(nil); len(debt) != 0 {
		t.Fatalf("a paid sale owes nothing: %+v", debt)
	}
	sale.Status = OrderStatusAnnulled
	if lines := sale.ToCashDocumentFrames(nil); len(lines) != 0 {
		t.Fatalf("an annulled sale adds nothing: %+v", lines)
	}
}

func TestApplyFreeLines(t *testing.T) {
	newSale := func(imposedTaxes ...SaleImposedTax) SaleOrder {
		return SaleOrder{
			Lines: []SaleOrderLine{
				{ProductID: 1, Quantity: 2000, Price: 1180, TaxGroup: finance.TaxGroupTaxed18},
				{ProductID: 2, Quantity: 1000, Price: 500, TaxGroup: finance.TaxGroupExonerated},
				{ProductID: 3, Quantity: 1003, Price: 600, SubPrice: 100, TaxGroup: finance.TaxGroupTaxed18},
				{ProductID: 4, Quantity: 1000, Price: 700, TaxGroup: finance.TaxGroupExport},
			},
			ImposedTaxes: imposedTaxes,
		}
	}
	sale := newSale(SaleImposedTax{Type: finance.TaxGroupFree, Line: 1, Amount: 1})
	if err := sale.ApplyFreeLines(); err != nil {
		t.Fatal(err)
	}
	if sale.ImposedTaxes[0].Amount != 2360 || sale.Lines[0].Price != 0 || sale.Lines[0].SubPrice != 0 {
		t.Fatalf("the free line is charged 0 and its catalog value is the amount: %+v %+v", sale.ImposedTaxes, sale.Lines[0])
	}
	if amount, isFree := sale.FreeLineAmountOf(0); !isFree || amount != 2360 {
		t.Fatalf("FreeLineAmountOf(0) = %v, %v", amount, isFree)
	}
	exonerated := newSale(SaleImposedTax{Type: finance.TaxGroupFree, Line: 2})
	if err := exonerated.ApplyFreeLines(); err != nil || exonerated.ImposedTaxes[0].Amount != 500 {
		t.Fatalf("an exonerated line can be given away: %v %+v", err, exonerated.ImposedTaxes)
	}
	for name, imposedTax := range map[string]SaleImposedTax{
		"a retention from the client": {Type: finance.TaxGroupRetention, Line: 0},
		"a line past the end":         {Type: finance.TaxGroupFree, Line: 5},
		"an export line":              {Type: finance.TaxGroupFree, Line: 4},
		"a line with sub-units":       {Type: finance.TaxGroupFree, Line: 3},
	} {
		sale := newSale(imposedTax)
		if err := sale.ApplyFreeLines(); err == nil {
			t.Fatalf("%s must be refused", name)
		}
	}
	duplicated := newSale(SaleImposedTax{Type: finance.TaxGroupFree, Line: 1}, SaleImposedTax{Type: finance.TaxGroupFree, Line: 1})
	if err := duplicated.ApplyFreeLines(); err == nil {
		t.Fatal("a line marked free twice must be refused")
	}
}
