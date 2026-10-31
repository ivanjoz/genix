// Backfilling what GET.accounting-operations syncs a period through (frontend/routes/accounting/
// accounts-consolidation/DELTA_CACHE_PLAN.md), on every row written before the columns existed:
//
//   - Period (YYMM of the accounting date) on sale_order, invoice_document, purchase_order,
//     expenses, cash_bank_movements and warehouse_product_movement. Every row is rewritten, not
//     only those whose Period is wrong: the period delta views key on a packed column the ORM
//     computes on write, which no row written before the index has.
//   - The same-day settlement (finance.SameDaySettlement): SameDayMovement on the payments and
//     SameDayCashBankID on their sale, order, expense or asset.
//   - The counter-account a document's payment clears (finance.DocumentPaymentAccounts).
//   - PurchaseAccountCode on the receptions of purchase orders and inventory expenses.
//
// A backfill and not a read-time default (CLAUDE.md §3), applying the same rules the write paths
// apply from now on. Re-running is safe: every value is recomputed from the rows themselves.

package exec

import (
	accounting "app/accounting/types"
	config "app/config/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	logistics "app/logistics/types"
	sales "app/sales/types"
	"fmt"
	"slices"
)

// accountingDeltaBackfillChunk bounds one write: each one reads the stored versions of its keys first.
const accountingDeltaBackfillChunk = 500

type accountingDeltaBackfillReport struct {
	companies        int
	sales            int
	documents        int
	orders           int
	expenses         int
	assets           int
	cashMovements    int
	stockMovements   int
	sameDayPayments  int
	orphanPayments   int
	orphanReceptions int
	rowsWritten      int
}

// BackfillAccountingDelta stamps the period, same-day and account columns on every existing row.
//
// fn-backfill-accounting-delta [apply] [companyID]
//
// It reports without writing unless `apply` is given.
func BackfillAccountingDelta(args *core.ExecArgs) core.FuncResponse {
	shouldWrite, onlyCompanyID := parseIdentityBackfillArguments(args.Message)

	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("could not read the companies:", queryError)
	}

	report := accountingDeltaBackfillReport{}
	for _, company := range companies {
		if onlyCompanyID > 0 && company.ID != onlyCompanyID {
			continue
		}
		report.companies++
		if backfillError := backfillCompanyAccountingDelta(company.ID, shouldWrite, &report); backfillError != nil {
			return args.MakeErr(backfillError)
		}
	}

	mode := "DRY RUN (add 'apply' to write)"
	if shouldWrite {
		mode = "APPLIED"
	}
	message := fmt.Sprintf("Accounting delta backfill %v: %d company(ies) · sales %d · documents %d · "+
		"orders %d · expenses %d · assets %d · cash movements %d · stock movements %d · "+
		"same-day payments %d · payments with no document %d · receptions with no purchase %d · rows written %d",
		mode, report.companies, report.sales, report.documents, report.orders, report.expenses, report.assets,
		report.cashMovements, report.stockMovements, report.sameDayPayments, report.orphanPayments,
		report.orphanReceptions, report.rowsWritten)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

func backfillCompanyAccountingDelta(companyID int32, shouldWrite bool, report *accountingDeltaBackfillReport) error {
	saleOrders, documents, orders, expenses := []sales.SaleOrder{}, []invoicing.InvoiceDocument{}, []logistics.PurchaseOrder{}, []finance.Expense{}
	assets, cashMovements, stockMovements := []accounting.Asset{}, []finance.CashBankMovement{}, []logistics.WarehouseProductMovement{}
	for _, readError := range []error{
		db.Query(&saleOrders).CompanyID.Equals(companyID).Exec(),
		db.Query(&documents).CompanyID.Equals(companyID).Exec(),
		db.Query(&orders).CompanyID.Equals(companyID).Exec(),
		db.Query(&expenses).CompanyID.Equals(companyID).Exec(),
		db.Query(&assets).CompanyID.Equals(companyID).Exec(),
		db.Query(&cashMovements).CompanyID.Equals(companyID).Exec(),
		db.Query(&stockMovements).CompanyID.Equals(companyID).Exec(),
	} {
		if readError != nil {
			return core.Err("Error reading the records of company", companyID, readError)
		}
	}
	report.sales += len(saleOrders)
	report.documents += len(documents)
	report.orders += len(orders)
	report.expenses += len(expenses)
	report.assets += len(assets)
	report.cashMovements += len(cashMovements)
	report.stockMovements += len(stockMovements)

	// Each register's payments, by the document they pay: DocumentID alone is shared between them.
	paymentIndexesByDocument := map[string][]int{}
	paymentKey := func(movementType finance.CashMovementType, documentID int64) string {
		family := movementType
		if movementType == finance.CashMovementTypeSaleRefund {
			family = finance.CashMovementTypeSaleCollection
		}
		return fmt.Sprintf("%d|%d", family, documentID)
	}
	for index, movement := range cashMovements {
		if _, isDocumentPayment := finance.DocumentPaymentAccounts[movement.Type]; isDocumentPayment {
			key := paymentKey(movement.Type, movement.DocumentID)
			paymentIndexesByDocument[key] = append(paymentIndexesByDocument[key], index)
		}
		cashMovements[index].Period = core.PeriodOfDay(movement.Date)
		cashMovements[index].SameDayMovement = 0
	}
	paidDocuments := map[string]bool{}

	// settle stamps one document's payments, the same way finance.RestampDocumentPayments does, and
	// returns its same-day cash bank.
	settle := func(paymentType finance.CashMovementType, documentID int64, accountingDate int16,
		documentTotal int32, counterAccountCode int32) int32 {
		key := paymentKey(paymentType, documentID)
		paidDocuments[key] = true
		paymentIndexes := paymentIndexesByDocument[key]
		payments := make([]finance.CashBankMovement, len(paymentIndexes))
		for position, movementIndex := range paymentIndexes {
			cashMovements[movementIndex].CounterAccountCode = counterAccountCode
			payments[position] = cashMovements[movementIndex]
		}
		settledPayment := finance.SameDaySettlement(accountingDate, documentTotal, payments)
		if settledPayment == nil {
			return 0
		}
		for _, movementIndex := range paymentIndexes {
			if cashMovements[movementIndex].ID == settledPayment.ID {
				cashMovements[movementIndex].SameDayMovement = 1
			}
		}
		report.sameDayPayments++
		return settledPayment.CashBankID
	}

	for index := range saleOrders {
		sale := &saleOrders[index]
		sale.StampPeriod()
		// An annulled sale settles nothing (see PostSaleOrderAnnul).
		saleTotal := core.If(sale.Status == sales.OrderStatusAnnulled, int32(0), sale.TotalAmount)
		sale.SameDayCashBankID = settle(finance.CashMovementTypeSaleCollection, sale.ID, sale.Date,
			saleTotal, finance.PCGEReceivable)
	}
	for index := range documents {
		documents[index].StampPeriod()
	}
	ordersByID := map[int64]*logistics.PurchaseOrder{}
	for index := range orders {
		order := &orders[index]
		order.StampPeriod()
		order.SameDayCashBankID = settle(finance.CashMovementTypeSupplierPayment, int64(order.ID),
			order.DocIssueDate, order.TotalAmount, finance.PurchasePayableAccount(order.DocType))
		ordersByID[int64(order.ID)] = order
	}
	expensesByID := map[int64]*finance.Expense{}
	for index := range expenses {
		expense := &expenses[index]
		expense.StampPeriod()
		expense.SameDayCashBankID = settle(finance.CashMovementTypeExpensePayment, int64(expense.ID),
			expense.AccountingDate(), expense.Amount, finance.PurchasePayableAccount(expense.DocType))
		expensesByID[int64(expense.ID)] = expense
	}
	for index := range assets {
		asset := &assets[index]
		asset.SameDayCashBankID = settle(finance.CashMovementTypeAssetPayment, int64(asset.ID),
			asset.AccountingDate(), asset.PurchaseAmount, finance.PCGEAssetPayables)
	}
	for key, paymentIndexes := range paymentIndexesByDocument {
		if !paidDocuments[key] {
			report.orphanPayments += len(paymentIndexes)
		}
	}

	for index := range stockMovements {
		movement := &stockMovements[index]
		movement.Period = core.PeriodOfDay(movement.Date)
		switch movement.Type {
		case logistics.StockMovementTypePurchaseReception:
			if order := ordersByID[movement.DocumentID]; order != nil {
				movement.PurchaseAccountCode = order.ReceptionAccountOf(movement.ProductID)
			} else {
				report.orphanReceptions++
			}
		case logistics.StockMovementTypeSuppliesPurchase:
			if expense := expensesByID[movement.DocumentID]; expense != nil {
				movement.PurchaseAccountCode = expense.AccountCode
			} else {
				report.orphanReceptions++
			}
		}
	}

	if !shouldWrite {
		return nil
	}
	saleTable, documentTable := db.TableOf[sales.SaleOrder](), db.TableOf[invoicing.InvoiceDocument]()
	orderTable, expenseTable := db.TableOf[logistics.PurchaseOrder](), db.TableOf[finance.Expense]()
	assetTable, cashTable := db.TableOf[accounting.Asset](), db.TableOf[finance.CashBankMovement]()
	stockTable := db.TableOf[logistics.WarehouseProductMovement]()
	// Each write names the columns a period delta index packs together (Period with its sync filter).
	writes := []func() error{
		func() error {
			return updateInChunks(saleOrders, func(chunk *[]sales.SaleOrder) error {
				return db.Update(chunk, saleTable.Period, saleTable.Status, saleTable.SameDayCashBankID)
			})
		},
		func() error {
			return updateInChunks(documents, func(chunk *[]invoicing.InvoiceDocument) error {
				return db.Update(chunk, documentTable.Period, documentTable.State)
			})
		},
		func() error {
			return updateInChunks(orders, func(chunk *[]logistics.PurchaseOrder) error {
				return db.Update(chunk, orderTable.Period, orderTable.Status, orderTable.SameDayCashBankID)
			})
		},
		func() error {
			return updateInChunks(expenses, func(chunk *[]finance.Expense) error {
				return db.Update(chunk, expenseTable.Period, expenseTable.Status, expenseTable.SameDayCashBankID)
			})
		},
		func() error {
			return updateInChunks(assets, func(chunk *[]accounting.Asset) error {
				return db.Update(chunk, assetTable.SameDayCashBankID, assetTable.Status)
			})
		},
		func() error {
			return updateInChunks(cashMovements, func(chunk *[]finance.CashBankMovement) error {
				return db.Update(chunk, cashTable.Period, cashTable.SameDayMovement, cashTable.Type,
					cashTable.CounterAccountCode)
			})
		},
		func() error {
			return updateInChunks(stockMovements, func(chunk *[]logistics.WarehouseProductMovement) error {
				// Type is also packed in the (Date, ProductID, Type, SubDivisor) view.
				return db.Update(chunk, stockTable.Period, stockTable.Type, stockTable.PurchaseAccountCode,
					stockTable.Date, stockTable.ProductID, stockTable.SubDivisor)
			})
		},
	}
	for _, write := range writes {
		if writeError := write(); writeError != nil {
			return core.Err("Error writing the records of company", companyID, writeError)
		}
	}
	report.rowsWritten += len(saleOrders) + len(documents) + len(orders) + len(expenses) + len(assets) +
		len(cashMovements) + len(stockMovements)
	return nil
}

// updateInChunks hands the records to write in slices of accountingDeltaBackfillChunk.
func updateInChunks[RecordT any](records []RecordT, write func(chunk *[]RecordT) error) error {
	for chunkStart := 0; chunkStart < len(records); chunkStart += accountingDeltaBackfillChunk {
		chunk := slices.Clone(records[chunkStart:min(chunkStart+accountingDeltaBackfillChunk, len(records))])
		if err := write(&chunk); err != nil {
			return err
		}
	}
	return nil
}
