// Reading the Registro de Compras for a period.
//
// Like the sales book, the handler only reads. A purchase is booked through the supplier's
// comprobante, and three tables carry one — purchase orders, expenses and fixed assets. Orders
// and expenses are synced through their period delta logs (Period = the YYMM of DocIssueDate,
// the same logs the accounting operations read); the asset register is small and synced whole,
// and the browser keeps the assets whose DocIssueDate falls in the period. The supplier's
// identity is resolved in the browser through the by-ids snapshot cache, and the Anexo 11
// mapping lives in frontend/routes/accounting/books/books.purchases.ts.
//
// Purchases without a comprobante come back too, flagged by DocType 0: a purchase without one
// stores its own date as DocIssueDate, so it lands in that date's period. They are not book rows;
// the page lists them behind a checkbox so the month can be reviewed.

package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	logistics "app/logistics/types"

	"golang.org/x/sync/errgroup"
)

// Where a purchase-book row came from. The page needs it to say what the purchase was and to
// link back to it; the book itself does not care.
const (
	PurchaseSourceOrder   int8 = 1
	PurchaseSourceExpense int8 = 2
	PurchaseSourceAsset   int8 = 3
)

// PurchaseBookDocument is one purchase as the book reads it. The document columns are
// finance.PurchaseDocument, identical on the three source tables.
type PurchaseBookDocument struct {
	Source             int8   `cb:"1"`
	SourceID           int32  `cb:"2"`
	Name               string `cb:"3"` // What was bought, for the review list.
	ProviderID         int32  `cb:"4"`
	ProviderSnapshotID int32  `cb:"5"`
	DueDate            int16  `cb:"6"` // Field 6, for a recibo de servicios públicos.
	// The record's own total. For a purchase without a comprobante it is the only amount there is.
	RecordAmount int32 `cb:"7"`
	// What the accounting operations (Consolidado & Cuentas) post the purchase to. The book
	// itself does not read them.
	AccountCode int32                         `cb:"8"`  // an expense's nature account
	ExpenseType int8                          `cb:"9"`  // finance.ExpenseType*
	ProductID   int32                         `cb:"10"` // an inventory expense's supply
	Lines       []logistics.PurchaseOrderLine `cb:"11"` // an order's lines, each with its account
	// A depreciation entry (ExpenseType 4) has no comprobante: its asset and its own date.
	AssetID int32 `cb:"12"`
	Date    int16 `cb:"13"`
	// The cash bank of the payment that settled it on its accounting date (finance.SameDaySettlement):
	// the accounting operations infer that payment instead of reading it.
	SameDayCashBankID int32 `cb:"14"`
	// Carried through for GET.accounting-operations, which syncs orders and expenses by delta log.
	Status   int8    `json:"ss" cb:"ss,15"`
	Version  int16   `json:"ver" cb:"ver,16"`
	Metadata *string `json:"mdt" cb:"mdt,17"`
	// The comprobante (finance.PurchaseDocument), flat: colbin does not flatten an embedded
	// struct, so embedding it sent these fields nested under "PurchaseDocument" to the browser.
	DocType       int8   `cb:"18"`
	DocSeries     string `cb:"19"`
	DocNumber     int64  `cb:"20"`
	DocIssueDate  int16  `cb:"21"`
	TaxableAmount int32  `cb:"22"`
	TaxAmount     int32  `cb:"23"`
	UntaxedAmount int32  `cb:"24"`
	OtherAmount   int32  `cb:"25"`
	CurrencyType  int8   `cb:"26"`
	ExchangeRate  int32  `cb:"27"`
}

// withPurchaseDocument copies a source's comprobante onto the row.
func (document PurchaseBookDocument) withPurchaseDocument(purchase finance.PurchaseDocument) PurchaseBookDocument {
	document.DocType, document.DocSeries, document.DocNumber = purchase.DocType, purchase.DocSeries, purchase.DocNumber
	document.DocIssueDate, document.TaxableAmount, document.TaxAmount = purchase.DocIssueDate, purchase.TaxableAmount, purchase.TaxAmount
	document.UntaxedAmount, document.OtherAmount = purchase.UntaxedAmount, purchase.OtherAmount
	document.CurrencyType, document.ExchangeRate = purchase.CurrencyType, purchase.ExchangeRate
	return document
}

// PurchasesBookResponse is one key per delta log; each key travels its own cursor (?<Key>=…). A
// canceled order and a removed expense or asset leave their log as removed IDs: they bought nothing.
type PurchasesBookResponse struct {
	PurchaseOrders []PurchaseBookDocument `cb:"1"`
	// Only the payable statuses: depreciation (Posted) is not a purchase.
	Expenses []PurchaseBookDocument `cb:"2"`
	// The whole register; the browser keeps the assets acquired in the period.
	Assets []PurchaseBookDocument `cb:"3"`
}

// GetPurchasesBook syncs a whole period (?period=YYYYMM).
func GetPurchasesBook(req *core.HandlerArgs) core.HandlerResponse {
	firstDay, _, boundsError := types.PeriodBounds(req.Query["period"])
	if boundsError != nil {
		return req.MakeErr(boundsError.Error())
	}
	companyID, periodKey := req.User.CompanyID, core.PeriodOfDay(firstDay)

	response := PurchasesBookResponse{}
	readGroup := errgroup.Group{}
	readGroup.Go(func() (err error) {
		response.PurchaseOrders, err = readPeriodPurchaseOrders(companyID, periodKey, req.GetDeltaCursor("PurchaseOrders"))
		return err
	})
	readGroup.Go(func() (err error) {
		response.Expenses, err = readPeriodExpenses(companyID, periodKey, req.GetDeltaCursor("Expenses"), []any{int8(1), int8(2)})
		return err
	})
	readGroup.Go(func() error {
		assets := []types.Asset{}
		query := db.Query(&assets)
		query.Select().CompanyID.Equals(companyID).Delta(req.GetDeltaCursor("Assets"),
			types.AssetStatusActive, types.AssetStatusFullyDepreciated, types.AssetStatusDisposed)
		if queryError := query.Exec(); queryError != nil {
			return core.Err("Error al obtener los activos.", queryError)
		}
		response.Assets = projectRecords(assets, makeAssetBookDocument)
		return nil
	})
	if readError := readGroup.Wait(); readError != nil {
		return req.MakeErr(readError.Error())
	}
	return req.MakeResponse(response)
}

func makeOrderBookDocument(order *logistics.PurchaseOrder) PurchaseBookDocument {
	return PurchaseBookDocument{
		Source: PurchaseSourceOrder, SourceID: order.ID, ProviderID: order.ProviderID,
		ProviderSnapshotID: order.ProviderSnapshotID, DueDate: order.PaymentDate,
		RecordAmount: order.TotalAmount, Lines: order.Lines,
		SameDayCashBankID: order.SameDayCashBankID, Status: order.Status, Version: order.Version,
		Metadata: order.Metadata,
	}.withPurchaseDocument(order.PurchaseDocument())
}

func makeExpenseBookDocument(expense *finance.Expense) PurchaseBookDocument {
	return PurchaseBookDocument{
		Source: PurchaseSourceExpense, SourceID: expense.ID, Name: expense.Name,
		ProviderID: expense.SupplierID, ProviderSnapshotID: expense.ProviderSnapshotID,
		DueDate: expense.DueDate, RecordAmount: expense.Amount, AccountCode: expense.AccountCode,
		ExpenseType: expense.Type, ProductID: expense.ProductID,
		AssetID: expense.AssetID, Date: expense.Date, SameDayCashBankID: expense.SameDayCashBankID,
		Status: expense.Status, Version: expense.Version, Metadata: expense.Metadata,
	}.withPurchaseDocument(expense.PurchaseDocument())
}

func makeAssetBookDocument(asset types.Asset) PurchaseBookDocument {
	return PurchaseBookDocument{
		Source: PurchaseSourceAsset, SourceID: asset.ID, Name: asset.Name,
		ProviderID: asset.SupplierID, ProviderSnapshotID: asset.ProviderSnapshotID,
		DueDate: asset.DueDate, RecordAmount: asset.PurchaseAmount,
		Status: asset.Status, Version: asset.Version, Metadata: asset.Metadata,
	}.withPurchaseDocument(asset.PurchaseDocument())
}
