package types

import "app/db"

// The cash-documents DataFrame (plan/CASH_DOCUMENTS_FRAME_PLAN.md): what the cash flow forecast needs
// from every document that moves cash — sales, expenses, purchase orders, assets — precomputed on each
// write into one frame several tables share (db.SharedFrame). Per day and site, one row per Kind
// (the source and whether the part is still open), TaxGroup (how to get the tax out of the amount),
// DocumentOffset (the currency, and which day's rate converts it) and DueOffset (when it falls due).
// The browser reads it instead of the documents themselves.

// Kind: which table wrote the row and whether its amount is still open (unpaid) or settled. Each
// table owns its Kinds (DataFrame.OwnedRows): a payment moves an amount from the open Kind to the
// settled one, by the write itself.
const (
	CashDocumentKindSaleOpen                int8 = 1
	CashDocumentKindSaleSettled             int8 = 2
	CashDocumentKindExpenseOpen             int8 = 10
	CashDocumentKindExpenseSettled          int8 = 11
	CashDocumentKindScheduledExpenseOpen    int8 = 12 // a period of an expense schedule
	CashDocumentKindScheduledExpenseSettled int8 = 13
	CashDocumentKindPurchaseOrderOpen       int8 = 20
	CashDocumentKindPurchaseOrderSettled    int8 = 21
	CashDocumentKindAssetOpen               int8 = 30
	CashDocumentKindAssetSettled            int8 = 31
)

// TaxGroup: how the tax comes out of a row's amount, so no tax column is stored. 1–7 are IGV
// treatments of a gross amount; 8 onward are taxes charged besides the IGV, whose row amount is the
// tax itself (their cash behavior is in the plan, §9.1). 1–7 are written so far.
const (
	TaxGroupTaxed18    int8 = 1 // IGV = amount × 18/118
	TaxGroupTaxed10    int8 = 2 // IGV = amount × 10/110 (MYPE restaurants and hotels)
	TaxGroupExonerated int8 = 3
	TaxGroupUnaffected int8 = 4 // inafecto; also a purchase's untaxed and other charges
	TaxGroupExport     int8 = 5
	TaxGroupNoDocument int8 = 6 // a purchase without comprobante: no crédito fiscal
	TaxGroupFree       int8 = 7 // gratuito: moves no cash, the seller still owes its IGV
	TaxGroupIsc        int8 = 8
	TaxGroupIcbper     int8 = 9
	TaxGroupDetraction int8 = 10
	TaxGroupPerception int8 = 11
	TaxGroupRetention  int8 = 12
)

const (
	// CashDocumentNoDueDate is the DueOffset of an open part without a due date.
	CashDocumentNoDueDate int16 = 1023
	// cashDocumentMaxDueOffset is the farthest due date a DueOffset holds, in days after Date.
	cashDocumentMaxDueOffset = 1022
	// CashDocumentSameDayOffset is the DocumentOffset of a foreign-currency document dated on the
	// record's own Date; 0 is a local-currency row.
	CashDocumentSameDayOffset = 128
	// cashDocumentMaxDocumentDays is how far a document's date may lie from the record's Date.
	cashDocumentMaxDocumentDays = 127
)

// CashDocumentFrame is one row of the cash-documents frame. Amount is the gross amount (taxes
// included) in local currency, or in dollars when DocumentOffset > 0.
type CashDocumentFrame struct {
	// Date is the record's date, week spread: partition_id = CompanyID << 13 | its epoch week.
	Date   int16 `frame:"part,13"`
	SiteID int16 `frame:"key,14"`
	Kind   int8  `frame:"row,6"`
	// TaxGroup holds up to 64 values: the IGV treatments and the taxes charged besides the IGV.
	TaxGroup int8 `frame:"row,6"`
	// DocumentOffset is 0 for local currency; for dollars, the rate of day Date + DocumentOffset − 128
	// (the comprobante's date) converts it.
	DocumentOffset uint8 `frame:"row,8"`
	// DueOffset is the due date − Date, 0..1022; CashDocumentNoDueDate without one; 0 on settled rows.
	DueOffset int16 `frame:"row,10"`
	Amount    int32 `frame:"sum"`
}

// CashDocumentsFrame is the shared declaration: SaleOrder, Expense, PurchaseOrder and Asset each
// write their Kinds of it.
var CashDocumentsFrame = db.DeclareSharedFrame[CashDocumentFrame](db.SharedFrame{
	Name: "cash-documents", Version: 1, UseWeekSpread: true, OwnerRow: "Kind"})

// CashDocumentAmount is one tax group's part of a document, in the document's currency.
type CashDocumentAmount struct {
	TaxGroup int8
	Amount   int32
}

// CashDocument is what a record says about itself to the frame.
type CashDocument struct {
	Date                  int16
	SiteID                int16
	OpenKind, SettledKind int8
	// DueDate is when the open part falls due, 0 without one; DocumentDate is the comprobante's date.
	DueDate, DocumentDate int16
	IsForeignCurrency     bool
	// Groups split the document's total by IGV treatment.
	Groups []CashDocumentAmount
	// OpenAmount is what is still unpaid, in the document's currency.
	OpenAmount int32
	// Memos are rows beside the total, never part of it: a sale's free lines (TaxGroupFree), its
	// detracción still to deposit (TaxGroupDetraction, open) and its retención (TaxGroupRetention).
	// A reader that sums a document leaves the TaxGroups of the memos out.
	Memos []CashDocumentMemo
}

// CashDocumentMemo is a memo row: open (its Kind the open one, due on DueDate) or settled.
type CashDocumentMemo struct {
	TaxGroup int8
	Amount   int32
	IsOpen   bool
}

// AppendCashDocumentFrames appends a document's rows: per tax group, its open part (due on DueOffset)
// and its settled part, then its memo rows. The open amount is spread over the groups in proportion
// to their amounts.
func AppendCashDocumentFrames(lines []CashDocumentFrame, document CashDocument) []CashDocumentFrame {
	totalAmount := int64(0)
	for _, group := range document.Groups {
		totalAmount += int64(group.Amount)
	}
	openAmount := min(max(int64(document.OpenAmount), 0), max(totalAmount, 0))
	documentOffset := cashDocumentOffsetOf(document)
	dueOffset := cashDueOffsetOf(document.Date, document.DueDate)

	spreadOpen := int64(0)
	for groupIndex, group := range document.Groups {
		groupOpen := int64(0)
		if totalAmount > 0 {
			groupOpen = openAmount * int64(group.Amount) / totalAmount
			// The last group takes the rounding left, so the open parts add up to the open amount.
			if groupIndex == len(document.Groups)-1 {
				groupOpen = min(openAmount-spreadOpen, int64(group.Amount))
			}
		}
		spreadOpen += groupOpen
		row := CashDocumentFrame{Date: document.Date, SiteID: document.SiteID, TaxGroup: group.TaxGroup, DocumentOffset: documentOffset}
		if groupOpen != 0 {
			row.Kind, row.DueOffset, row.Amount = document.OpenKind, dueOffset, int32(groupOpen)
			lines = append(lines, row)
		}
		if settled := int64(group.Amount) - groupOpen; settled != 0 {
			row.Kind, row.DueOffset, row.Amount = document.SettledKind, 0, int32(settled)
			lines = append(lines, row)
		}
	}
	for _, memo := range document.Memos {
		if memo.Amount == 0 {
			continue
		}
		row := CashDocumentFrame{Date: document.Date, SiteID: document.SiteID, TaxGroup: memo.TaxGroup,
			DocumentOffset: documentOffset, Kind: document.SettledKind, Amount: memo.Amount}
		if memo.IsOpen {
			row.Kind, row.DueOffset = document.OpenKind, dueOffset
		}
		lines = append(lines, row)
	}
	return lines
}

// IsCashDocumentMemoTaxGroup: a row of this TaxGroup is a memo beside its document's total.
func IsCashDocumentMemoTaxGroup(taxGroup int8) bool {
	return taxGroup >= TaxGroupFree
}

// cashDocumentOffsetOf is a document's DocumentOffset: 0 in local currency; in dollars, 128 plus the
// days from the record's Date to the comprobante's, clipped to ±127.
func cashDocumentOffsetOf(document CashDocument) uint8 {
	if !document.IsForeignCurrency {
		return 0
	}
	documentDate := document.DocumentDate
	if documentDate == 0 {
		documentDate = document.Date
	}
	days := min(max(int(documentDate)-int(document.Date), -cashDocumentMaxDocumentDays), cashDocumentMaxDocumentDays)
	return uint8(CashDocumentSameDayOffset + days)
}

// cashDueOffsetOf is the days from the record's Date to its due date, clipped to 0..1022;
// CashDocumentNoDueDate without a due date.
func cashDueOffsetOf(date, dueDate int16) int16 {
	if dueDate == 0 {
		return CashDocumentNoDueDate
	}
	return int16(min(max(int(dueDate)-int(date), 0), cashDocumentMaxDueOffset))
}

// PurchaseCashGroups splits a purchase's amount by tax group from its comprobante: without one, all of
// it has no crédito fiscal; with one, the taxed base and its IGV are one group (18 % or 10 % by their
// ratio) and the rest — untaxed and other charges — is inafecto.
func PurchaseCashGroups(document PurchaseDocument, amount int32) []CashDocumentAmount {
	if document.DocType == PurchaseDocTypeNone {
		return []CashDocumentAmount{{TaxGroup: TaxGroupNoDocument, Amount: amount}}
	}
	groups := []CashDocumentAmount{}
	taxedAmount := document.TaxableAmount + document.TaxAmount
	if taxedAmount != 0 {
		taxedGroup := TaxGroupTaxed18
		// The IGV over its base is 0.18 or 0.10: below 0.14 it is the reduced rate.
		if int64(document.TaxAmount)*100 < int64(document.TaxableAmount)*14 {
			taxedGroup = TaxGroupTaxed10
		}
		groups = append(groups, CashDocumentAmount{TaxGroup: taxedGroup, Amount: taxedAmount})
	}
	if untaxedAmount := amount - taxedAmount; untaxedAmount != 0 {
		groups = append(groups, CashDocumentAmount{TaxGroup: TaxGroupUnaffected, Amount: untaxedAmount})
	}
	return groups
}

// OwnedCashDocumentKinds lists Kinds as a DataFrame's OwnedRows.
func OwnedCashDocumentKinds(kinds ...int8) []int64 {
	owned := make([]int64, len(kinds))
	for position, kind := range kinds {
		owned[position] = int64(kind)
	}
	return owned
}

// IsOpenPayableCashDocumentKind: what the company still owes on an expense, a schedule's period, a
// purchase order or an asset purchase.
func IsOpenPayableCashDocumentKind(kind int8) bool {
	return kind == CashDocumentKindExpenseOpen || kind == CashDocumentKindScheduledExpenseOpen ||
		kind == CashDocumentKindPurchaseOrderOpen || kind == CashDocumentKindAssetOpen
}

// CashDocumentSelection is what a cash flow drill-down asks for (GET.cash-documents-day): the open
// documents of one Kind dated on one of Dates, on one site unless AllSites.
type CashDocumentSelection struct {
	Kind     int8
	Dates    map[int16]bool
	SiteID   int16
	AllSites bool
}

// OpenAmountOf sums a record's cash-documents lines that fall in the selection: the same lines the
// frame summed, so the drill-down lists exactly what the cell added up.
func (selection CashDocumentSelection) OpenAmountOf(lines []CashDocumentFrame) int64 {
	openAmount := int64(0)
	for _, line := range lines {
		if line.Kind == selection.Kind && selection.Dates[line.Date] && (selection.AllSites || line.SiteID == selection.SiteID) &&
			!IsCashDocumentMemoTaxGroup(line.TaxGroup) {
			openAmount += int64(line.Amount)
		}
	}
	return openAmount
}

// CashDocumentDetail is one open document of a drill-down: Amount is what it still owes, in cents of
// CurrencyType; DueDate is 0 without one.
type CashDocumentDetail struct {
	ID           int64
	Label        string
	Date         int16
	DueDate      int16
	Amount       int64
	CurrencyType int8
}
