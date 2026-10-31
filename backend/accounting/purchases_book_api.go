// Reading the Registro de Compras for a period.
//
// Like the sales book, the handler only reads. A purchase is booked through the supplier's
// comprobante, and three tables carry one — purchase orders, expenses and fixed assets — so
// this reads the period from all three through their DocIssueDate index and sends the rows as
// numbers. The supplier's identity is resolved in the browser through the by-ids snapshot
// cache, and the Anexo 11 mapping lives in frontend/routes/accounting/books/books.purchases.ts.
//
// Purchases without a comprobante come back too, flagged by DocType 0: a purchase without one
// stores its own date as DocIssueDate, so the same index finds it. They are not book rows; the
// page lists them behind a checkbox so the month can be reviewed.

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

type PurchasesBookResponse struct {
	Period    string                 `cb:"1"`
	FirstDay  int16                  `cb:"2"`
	LastDay   int16                  `cb:"3"`
	Documents []PurchaseBookDocument `cb:"4"`
}

// GetPurchasesBook reads a whole period, or the single day inside one.
func GetPurchasesBook(req *core.HandlerArgs) core.HandlerResponse {
	firstDay, lastDay, period, argsError := resolveBookRange(req)
	if argsError != nil {
		return req.MakeErr(argsError.Error())
	}

	companyID := req.User.CompanyID
	var orders, expenses, assets []PurchaseBookDocument
	readGroup := errgroup.Group{}
	readGroup.Go(func() (err error) { orders, err = loadPeriodOrders(companyID, firstDay, lastDay); return })
	readGroup.Go(func() (err error) { expenses, err = loadPeriodExpenses(companyID, firstDay, lastDay); return })
	readGroup.Go(func() (err error) { assets, err = loadPeriodAssets(companyID, firstDay, lastDay); return })
	if readError := readGroup.Wait(); readError != nil {
		return req.MakeErr(readError.Error())
	}

	documents := append(append(orders, expenses...), assets...)
	core.Log("GetPurchasesBook period:", period, "orders:", len(orders),
		"expenses:", len(expenses), "assets:", len(assets))
	return req.MakeResponse(PurchasesBookResponse{
		Period: period, FirstDay: firstDay, LastDay: lastDay, Documents: documents,
	})
}

// loadPeriodOrders reads the purchase orders of the range. A canceled order bought nothing.
func loadPeriodOrders(companyID int32, firstDay, lastDay int16) ([]PurchaseBookDocument, error) {
	groups := []db.RecordGroup[logistics.PurchaseOrder]{}
	query := db.QueryIndexGroup(&groups).CompanyID.Equals(companyID)
	query.DocIssueDate.Between(firstDay, lastDay)
	if queryError := query.Exec(); queryError != nil {
		return nil, core.Err("Error al obtener las órdenes de compra del periodo.", queryError)
	}

	documents := []PurchaseBookDocument{}
	for groupIndex := range groups {
		for recordIndex := range groups[groupIndex].Records {
			order := &groups[groupIndex].Records[recordIndex]
			if order.Status == logistics.PurchaseOrderStatusCanceled {
				continue
			}
			documents = append(documents, makeOrderBookDocument(order))
		}
	}
	return documents, nil
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

// loadPeriodExpenses reads the expenses of the range. A removed expense never happened, and
// depreciation is not a purchase — it carries no DocIssueDate, so the index never returns it.
func loadPeriodExpenses(companyID int32, firstDay, lastDay int16) ([]PurchaseBookDocument, error) {
	groups := []db.RecordGroup[finance.Expense]{}
	query := db.QueryIndexGroup(&groups).CompanyID.Equals(companyID)
	query.DocIssueDate.Between(firstDay, lastDay)
	if queryError := query.Exec(); queryError != nil {
		return nil, core.Err("Error al obtener los gastos del periodo.", queryError)
	}

	documents := []PurchaseBookDocument{}
	for groupIndex := range groups {
		for recordIndex := range groups[groupIndex].Records {
			expense := &groups[groupIndex].Records[recordIndex]
			if expense.Status == 0 || expense.Type == finance.ExpenseTypeDepreciation {
				continue
			}
			documents = append(documents, makeExpenseBookDocument(expense))
		}
	}
	return documents, nil
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

// loadPeriodAssets reads the fixed assets acquired in the range. A donated asset without a
// comprobante was not bought, so it is not a purchase to review.
func loadPeriodAssets(companyID int32, firstDay, lastDay int16) ([]PurchaseBookDocument, error) {
	groups := []db.RecordGroup[types.Asset]{}
	query := db.QueryIndexGroup(&groups).CompanyID.Equals(companyID)
	query.DocIssueDate.Between(firstDay, lastDay)
	if queryError := query.Exec(); queryError != nil {
		return nil, core.Err("Error al obtener los activos del periodo.", queryError)
	}

	documents := []PurchaseBookDocument{}
	for groupIndex := range groups {
		for recordIndex := range groups[groupIndex].Records {
			asset := &groups[groupIndex].Records[recordIndex]
			isDonation := asset.PurchaseAmount == 0 && asset.DocType == finance.PurchaseDocTypeNone
			if asset.Status == types.AssetStatusRemoved || isDonation {
				continue
			}
			documents = append(documents, PurchaseBookDocument{
				Source: PurchaseSourceAsset, SourceID: asset.ID, Name: asset.Name,
				ProviderID: asset.SupplierID, ProviderSnapshotID: asset.ProviderSnapshotID,
				DueDate: asset.DueDate, RecordAmount: asset.PurchaseAmount,
			}.withPurchaseDocument(asset.PurchaseDocument()))
		}
	}
	return documents, nil
}
