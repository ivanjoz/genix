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
	Source             int8   `json:",omitempty"`
	SourceID           int32  `json:",omitempty"`
	Name               string `json:",omitempty"` // What was bought, for the review list.
	ProviderID         int32  `json:",omitempty"`
	ProviderSnapshotID int32  `json:",omitempty"`
	DueDate            int16  `json:",omitempty"` // Field 6, for a recibo de servicios públicos.
	// The record's own total. For a purchase without a comprobante it is the only amount there is.
	RecordAmount int32 `json:",omitempty"`
	finance.PurchaseDocument
}

type PurchasesBookResponse struct {
	Period    string                 `json:",omitempty"`
	FirstDay  int16                  `json:",omitempty"`
	LastDay   int16                  `json:",omitempty"`
	Documents []PurchaseBookDocument `json:",omitempty"`
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
			documents = append(documents, PurchaseBookDocument{
				Source: PurchaseSourceOrder, SourceID: order.ID, ProviderID: order.ProviderID,
				ProviderSnapshotID: order.ProviderSnapshotID, DueDate: order.PaymentDate,
				RecordAmount: order.TotalAmount, PurchaseDocument: order.PurchaseDocument(),
			})
		}
	}
	return documents, nil
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
			documents = append(documents, PurchaseBookDocument{
				Source: PurchaseSourceExpense, SourceID: expense.ID, Name: expense.Name,
				ProviderID: expense.SupplierID, ProviderSnapshotID: expense.ProviderSnapshotID,
				DueDate: expense.DueDate, RecordAmount: expense.Amount,
				PurchaseDocument: expense.PurchaseDocument(),
			})
		}
	}
	return documents, nil
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
				PurchaseDocument: asset.PurchaseDocument(),
			})
		}
	}
	return documents, nil
}
