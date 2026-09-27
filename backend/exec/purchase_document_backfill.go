// Backfilling the purchase-document columns on the purchases written before they existed.
//
// The Registro de Compras reads purchase orders, expenses and assets through their DocIssueDate
// index. A purchase with no comprobante stores its own date there (finance.NormalizePurchaseDocument),
// which is how the book's "sin comprobante" list finds it. Every row written before the column
// holds 0 and would be invisible to the book, so this runs each row through the same
// normalization the save paths use, and pins the supplier's identity snapshot on expenses and
// assets, which did not carry one.
//
// None of the existing rows has a comprobante to recover: PurchaseOrder.InvoiceNumber, the only
// document field there was, is empty on every row, so it was dropped rather than migrated.
//
// Depreciation expenses (type 4) are not purchases and are left alone.

package exec

import (
	accounting "app/accounting/types"
	config "app/config/types"
	"app/core"
	crm "app/crm/types"
	"app/db"
	finance "app/finance/types"
	logistics "app/logistics/types"
	"fmt"
)

type purchaseDocumentBackfillReport struct {
	companies      int
	ordersPending  int
	expensePending int
	assetsPending  int
	rowsWritten    int
}

// BackfillPurchaseDocuments normalizes the document columns of every purchase and pins its supplier.
//
// fn-backfill-purchase-documents [apply] [companyID]
//
// It reports without writing unless `apply` is given. Re-running is safe: a row already in its
// normalized shape is not written again.
func BackfillPurchaseDocuments(args *core.ExecArgs) core.FuncResponse {
	shouldWrite, onlyCompanyID := parseIdentityBackfillArguments(args.Message)

	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", queryError)
	}

	report := purchaseDocumentBackfillReport{}
	for _, company := range companies {
		if onlyCompanyID > 0 && company.ID != onlyCompanyID {
			continue
		}
		report.companies++
		snapshots := providerSnapshotCache{companyID: company.ID, snapshotIDs: map[int32]int32{}}
		for _, backfill := range []func(int32, bool, *providerSnapshotCache, *purchaseDocumentBackfillReport) error{
			backfillOrderDocuments, backfillExpenseDocuments, backfillAssetDocuments,
		} {
			if backfillError := backfill(company.ID, shouldWrite, &snapshots, &report); backfillError != nil {
				return args.MakeErr(backfillError)
			}
		}
	}

	mode := "SIMULACIÓN (agregue 'apply' para escribir)"
	if shouldWrite {
		mode = "APLICADO"
	}
	message := fmt.Sprintf(
		"Backfill de comprobantes de compra %v: %d empresa(s) · pendientes: órdenes %d, gastos %d, "+
			"activos %d · escritos %d",
		mode, report.companies, report.ordersPending, report.expensePending, report.assetsPending,
		report.rowsWritten,
	)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

// providerSnapshotCache reads each supplier's snapshot once per company.
type providerSnapshotCache struct {
	companyID   int32
	snapshotIDs map[int32]int32
}

func (cache *providerSnapshotCache) resolve(providerID int32) int32 {
	if snapshotID, isKnown := cache.snapshotIDs[providerID]; isKnown {
		return snapshotID
	}
	snapshotID, err := crm.ProviderSnapshotID(cache.companyID, providerID)
	if err != nil {
		// A supplier deleted since the purchase has no identity to pin; the book flags the row.
		core.Log("backfill: proveedor sin identidad", providerID, err)
		snapshotID = 0
	}
	cache.snapshotIDs[providerID] = snapshotID
	return snapshotID
}

func backfillOrderDocuments(
	companyID int32, shouldWrite bool, snapshots *providerSnapshotCache, report *purchaseDocumentBackfillReport,
) error {
	orders := []logistics.PurchaseOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(companyID)
	if queryError := query.Exec(); queryError != nil {
		return core.Err("Error al leer las órdenes de compra de la empresa", companyID, queryError)
	}

	ordersToUpdate := []logistics.PurchaseOrder{}
	for _, order := range orders {
		document := order.PurchaseDocument()
		if err := finance.NormalizePurchaseDocument(&document, order.ProviderID, order.Date, order.PaymentDate); err != nil {
			return core.Err("Orden de compra", order.ID, err)
		}
		document.CurrencyType = core.If(document.CurrencyType == 0, finance.CurrencyPEN, document.CurrencyType)
		// An order pins its snapshot at creation; only the orders older than the snapshot lack one.
		snapshotID := core.If(order.ProviderSnapshotID > 0, order.ProviderSnapshotID, snapshots.resolve(order.ProviderID))
		if document == order.PurchaseDocument() && snapshotID == order.ProviderSnapshotID {
			continue
		}
		order.SetPurchaseDocument(document)
		order.ProviderSnapshotID = snapshotID
		ordersToUpdate = append(ordersToUpdate, order)
	}
	report.ordersPending += len(ordersToUpdate)
	if len(ordersToUpdate) == 0 || !shouldWrite {
		return nil
	}

	table := db.TableOf[logistics.PurchaseOrder]()
	if updateError := db.Update(&ordersToUpdate,
		table.ProviderSnapshotID, table.DocType, table.DocSeries, table.DocNumber, table.DocIssueDate,
		table.TaxableAmount, table.TaxAmount, table.UntaxedAmount, table.OtherAmount, table.CurrencyType,
		table.ExchangeRate,
	); updateError != nil {
		return core.Err("Error al actualizar las órdenes de compra de la empresa", companyID, updateError)
	}
	report.rowsWritten += len(ordersToUpdate)
	return nil
}

func backfillExpenseDocuments(
	companyID int32, shouldWrite bool, snapshots *providerSnapshotCache, report *purchaseDocumentBackfillReport,
) error {
	expenses := []finance.Expense{}
	query := db.Query(&expenses)
	query.Select().CompanyID.Equals(companyID)
	if queryError := query.Exec(); queryError != nil {
		return core.Err("Error al leer los gastos de la empresa", companyID, queryError)
	}

	expensesToUpdate := []finance.Expense{}
	for _, expense := range expenses {
		if expense.Type == finance.ExpenseTypeDepreciation {
			continue
		}
		document := expense.PurchaseDocument()
		if err := finance.NormalizePurchaseDocument(&document, expense.SupplierID, expense.Date, expense.DueDate); err != nil {
			return core.Err("Gasto", expense.ID, err)
		}
		snapshotID := snapshots.resolve(expense.SupplierID)
		if document == expense.PurchaseDocument() && snapshotID == expense.ProviderSnapshotID {
			continue
		}
		expense.SetPurchaseDocument(document)
		expense.ProviderSnapshotID = snapshotID
		expensesToUpdate = append(expensesToUpdate, expense)
	}
	report.expensePending += len(expensesToUpdate)
	if len(expensesToUpdate) == 0 || !shouldWrite {
		return nil
	}

	table := db.TableOf[finance.Expense]()
	if updateError := db.Update(&expensesToUpdate,
		table.ProviderSnapshotID, table.DocType, table.DocSeries, table.DocNumber, table.DocIssueDate,
		table.TaxableAmount, table.TaxAmount, table.UntaxedAmount, table.OtherAmount, table.ExchangeRate,
	); updateError != nil {
		return core.Err("Error al actualizar los gastos de la empresa", companyID, updateError)
	}
	report.rowsWritten += len(expensesToUpdate)
	return nil
}

func backfillAssetDocuments(
	companyID int32, shouldWrite bool, snapshots *providerSnapshotCache, report *purchaseDocumentBackfillReport,
) error {
	assets := []accounting.Asset{}
	query := db.Query(&assets)
	query.Select().CompanyID.Equals(companyID)
	if queryError := query.Exec(); queryError != nil {
		return core.Err("Error al leer los activos de la empresa", companyID, queryError)
	}

	assetsToUpdate := []accounting.Asset{}
	for _, asset := range assets {
		document := asset.PurchaseDocument()
		if err := finance.NormalizePurchaseDocument(&document, asset.SupplierID, asset.AcquisitionDate, asset.DueDate); err != nil {
			return core.Err("Activo", asset.ID, err)
		}
		snapshotID := snapshots.resolve(asset.SupplierID)
		if document == asset.PurchaseDocument() && snapshotID == asset.ProviderSnapshotID {
			continue
		}
		asset.SetPurchaseDocument(document)
		asset.ProviderSnapshotID = snapshotID
		assetsToUpdate = append(assetsToUpdate, asset)
	}
	report.assetsPending += len(assetsToUpdate)
	if len(assetsToUpdate) == 0 || !shouldWrite {
		return nil
	}

	table := db.TableOf[accounting.Asset]()
	if updateError := db.Update(&assetsToUpdate,
		table.ProviderSnapshotID, table.DocType, table.DocSeries, table.DocNumber, table.DocIssueDate,
		table.TaxableAmount, table.TaxAmount, table.UntaxedAmount, table.OtherAmount, table.ExchangeRate,
	); updateError != nil {
		return core.Err("Error al actualizar los activos de la empresa", companyID, updateError)
	}
	report.rowsWritten += len(assetsToUpdate)
	return nil
}
