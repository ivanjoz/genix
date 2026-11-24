package types

import (
	"slices"
	"testing"
)

func TestAppendCashDocumentFramesSplitsOpenAndSettled(t *testing.T) {
	// A S/ 1 180.00 credit sale with S/ 500.00 collected, due 30 days after its date.
	lines := AppendCashDocumentFrames(nil, CashDocument{
		Date: 20773, SiteID: 1, OpenKind: CashDocumentKindSaleOpen, SettledKind: CashDocumentKindSaleSettled,
		DueDate: 20803, Groups: []CashDocumentAmount{{TaxGroup: TaxGroupTaxed18, Amount: 118000}}, OpenAmount: 68000,
	})
	expected := []CashDocumentFrame{
		{Date: 20773, SiteID: 1, Kind: CashDocumentKindSaleOpen, TaxGroup: TaxGroupTaxed18, DueOffset: 30, Amount: 68000},
		{Date: 20773, SiteID: 1, Kind: CashDocumentKindSaleSettled, TaxGroup: TaxGroupTaxed18, DueOffset: 0, Amount: 50000},
	}
	if !slices.Equal(lines, expected) {
		t.Fatalf("got %+v", lines)
	}
}

func TestAppendCashDocumentFramesSpreadsOpenOverGroups(t *testing.T) {
	// 300 open of 1 000: 3/4 taxed, 1/4 untaxed; the parts add up exactly.
	lines := AppendCashDocumentFrames(nil, CashDocument{
		Date: 100, OpenKind: CashDocumentKindExpenseOpen, SettledKind: CashDocumentKindExpenseSettled,
		Groups:     []CashDocumentAmount{{TaxGroup: TaxGroupTaxed18, Amount: 750}, {TaxGroup: TaxGroupUnaffected, Amount: 250}},
		OpenAmount: 301,
	})
	openTotal, settledTotal := int32(0), int32(0)
	for _, line := range lines {
		if line.Kind == CashDocumentKindExpenseOpen {
			openTotal += line.Amount
			if line.DueOffset != CashDocumentNoDueDate {
				t.Fatalf("an open part without a due date takes CashDocumentNoDueDate: %+v", line)
			}
		} else {
			settledTotal += line.Amount
		}
	}
	if openTotal != 301 || settledTotal != 699 || len(lines) != 4 {
		t.Fatalf("open %d settled %d lines %+v", openTotal, settledTotal, lines)
	}
}

func TestCashDocumentOffsets(t *testing.T) {
	local := CashDocument{Date: 1000, DocumentDate: 1007}
	if offset := cashDocumentOffsetOf(local); offset != 0 {
		t.Fatalf("local currency: %d", offset)
	}
	dollars := CashDocument{Date: 1000, DocumentDate: 1007, IsForeignCurrency: true}
	if offset := cashDocumentOffsetOf(dollars); offset != 135 {
		t.Fatalf("a dollar document a week later: %d", offset)
	}
	dollars.DocumentDate = 0
	if offset := cashDocumentOffsetOf(dollars); offset != CashDocumentSameDayOffset {
		t.Fatalf("without a document date: %d", offset)
	}
	dollars.DocumentDate = 500
	if offset := cashDocumentOffsetOf(dollars); offset != 1 {
		t.Fatalf("clipped to −127: %d", offset)
	}
	if offset := cashDueOffsetOf(1000, 990); offset != 0 {
		t.Fatalf("a due date before the date: %d", offset)
	}
	if offset := cashDueOffsetOf(1000, 3000); offset != 1022 {
		t.Fatalf("clipped to 1022: %d", offset)
	}
}

func TestPurchaseCashGroups(t *testing.T) {
	if groups := PurchaseCashGroups(PurchaseDocument{}, 5000); !slices.Equal(groups, []CashDocumentAmount{{TaxGroup: TaxGroupNoDocument, Amount: 5000}}) {
		t.Fatalf("without comprobante: %+v", groups)
	}
	factura := PurchaseDocument{DocType: PurchaseDocTypeFactura, TaxableAmount: 10000, TaxAmount: 1800, UntaxedAmount: 500, OtherAmount: 200}
	expected := []CashDocumentAmount{{TaxGroup: TaxGroupTaxed18, Amount: 11800}, {TaxGroup: TaxGroupUnaffected, Amount: 700}}
	if groups := PurchaseCashGroups(factura, 12500); !slices.Equal(groups, expected) {
		t.Fatalf("factura: %+v", groups)
	}
	reduced := PurchaseDocument{DocType: PurchaseDocTypeFactura, TaxableAmount: 10000, TaxAmount: 1000}
	if groups := PurchaseCashGroups(reduced, 11000); groups[0].TaxGroup != TaxGroupTaxed10 {
		t.Fatalf("10 %% IGV: %+v", groups)
	}
}

func TestExpenseCashDocumentFrames(t *testing.T) {
	expense := Expense{Type: ExpenseTypeSimple, Status: 1, Date: 200, DueDate: 215, SiteID: 2, Amount: 1000, PaidAmount: 400}
	lines := expense.ToCashDocumentFrames(nil)
	if len(lines) != 2 || lines[0].Kind != CashDocumentKindExpenseOpen || lines[0].Amount != 600 || lines[0].TaxGroup != TaxGroupNoDocument {
		t.Fatalf("a pending expense without comprobante: %+v", lines)
	}
	expense.ExpenseScheduledID = 9
	if lines := expense.ToCashDocumentFrames(nil); lines[0].Kind != CashDocumentKindScheduledExpenseOpen {
		t.Fatalf("a schedule's period: %+v", lines)
	}
	expense.AccountCode = 40111
	if lines := expense.ToCashDocumentFrames(nil); len(lines) != 0 {
		t.Fatalf("a tax payment adds nothing: %+v", lines)
	}
}

func TestCashDocumentSelectionOpenAmount(t *testing.T) {
	// A pending expense with two tax groups: its open part is split over two lines.
	expense := Expense{Type: ExpenseTypeSimple, Status: 1, Date: 200, SiteID: 2, Amount: 1000, PaidAmount: 400,
		DocType: PurchaseDocTypeFactura, TaxableAmount: 500, TaxAmount: 90, UntaxedAmount: 410}
	lines := expense.ToCashDocumentFrames(nil)
	selection := CashDocumentSelection{Kind: CashDocumentKindExpenseOpen, Dates: map[int16]bool{200: true}, SiteID: 2}
	if amount := selection.OpenAmountOf(lines); amount != 600 {
		t.Fatalf("the open parts of every group: %d", amount)
	}
	selection.SiteID = 3
	if amount := selection.OpenAmountOf(lines); amount != 0 {
		t.Fatalf("another site: %d", amount)
	}
	selection.AllSites, selection.Kind = true, CashDocumentKindExpenseSettled
	if amount := selection.OpenAmountOf(lines); amount != 400 {
		t.Fatalf("the settled Kind, every site: %d", amount)
	}
}
