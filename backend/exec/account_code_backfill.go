// Backfilling the PCGE accounts on the rows written before the AccountCode / Origin columns.
//
// Every movement needs an account for its journal entry to balance, and the defaults are stored
// when a record is written, so rows from before the columns read 0 — "unclassified" — even
// where the default is obvious. This stamps the same default the handlers would have stored:
//
//   - product: Origin = bought (no product was made before the column existed);
//   - supply: accounting.DefaultSupplyAccountCode (equipment if it depreciates, else suministros);
//   - simple expense and expense schedule: the category's default (0 for Taxes / Other, which
//     stay unclassified for the user to decide);
//   - inventory expense and purchase-order supply line: the supply's account;
//   - asset: accounting.ResolveAssetAccount (the supply's 33x, else generic equipment).
//
// Depreciation expenses (Type 4) and product lines keep 0: their accounts are derived.
// Supplies run first, so the expenses, lines and assets read the supply defaults just stamped.
//
// fn-backfill-account-codes [apply] [companyID]
//
// It reports without writing unless `apply` is given. Re-running is safe: it only touches rows
// whose value is still 0.

package exec

import (
	accounting "app/accounting/types"
	config "app/config/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	logistics "app/logistics/types"
	production "app/production/types"
	"fmt"
)

type accountCodeBackfillReport struct {
	companies, products, supplies, expenses, schedules, purchaseOrders, assets int
	expensesLeftUnclassified, schedulesLeftUnclassified                        int
}

func BackfillAccountCodes(args *core.ExecArgs) core.FuncResponse {
	shouldWrite, onlyCompanyID := parseIdentityBackfillArguments(args.Message)

	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", queryError)
	}

	report := accountCodeBackfillReport{}
	for _, company := range companies {
		if onlyCompanyID > 0 && company.ID != onlyCompanyID {
			continue
		}
		report.companies++
		if backfillError := backfillCompanyAccountCodes(company.ID, shouldWrite, &report); backfillError != nil {
			return args.MakeErr(backfillError)
		}
	}

	mode := core.If(shouldWrite, "APLICADO", "SIMULACIÓN (agregue 'apply' para escribir)")
	message := fmt.Sprintf(
		"Backfill de cuentas contables %v: %d empresa(s) · productos %d · insumos %d · gastos %d "+
			"(sin clasificar %d) · programados %d (sin clasificar %d) · órdenes de compra %d · activos %d",
		mode, report.companies, report.products, report.supplies, report.expenses,
		report.expensesLeftUnclassified, report.schedules, report.schedulesLeftUnclassified,
		report.purchaseOrders, report.assets,
	)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

func backfillCompanyAccountCodes(companyID int32, shouldWrite bool, report *accountCodeBackfillReport) error {
	// 1. The catalog: products get their origin, supplies their default account.
	catalogRows := []production.Product{}
	catalogQuery := db.Query(&catalogRows)
	// CompanyID is the partition key: without it the update below would upsert into company 0.
	catalogQuery.Select(catalogQuery.CompanyID, catalogQuery.ID, catalogQuery.Status, catalogQuery.DepreciationMonths,
		catalogQuery.Origin, catalogQuery.AccountCode).CompanyID.Equals(companyID)
	if queryError := catalogQuery.Exec(); queryError != nil {
		return core.Err("Error al leer los productos de la empresa", companyID, queryError)
	}
	supplyAccountByID := map[int32]int32{}
	catalogToUpdate := []production.Product{}
	for _, catalogRow := range catalogRows {
		switch {
		case catalogRow.Status == production.ProductStatusSupply && catalogRow.AccountCode == 0:
			catalogRow.AccountCode = accounting.DefaultSupplyAccountCode(catalogRow.DepreciationMonths)
			catalogToUpdate = append(catalogToUpdate, catalogRow)
			report.supplies++
		case catalogRow.Status != production.ProductStatusSupply && catalogRow.Origin == 0:
			catalogRow.Origin = production.ProductOriginBought
			catalogToUpdate = append(catalogToUpdate, catalogRow)
			report.products++
		}
		if catalogRow.Status == production.ProductStatusSupply {
			supplyAccountByID[catalogRow.ID] = catalogRow.AccountCode
		}
	}

	// 2. Expenses: the category's default, or the supply's account for an inventory purchase.
	expenses := []finance.Expense{}
	expenseQuery := db.Query(&expenses)
	expenseQuery.Select(expenseQuery.CompanyID, expenseQuery.ID,expenseQuery.Type, expenseQuery.CategoryID,
		expenseQuery.ProductID, expenseQuery.AccountCode).CompanyID.Equals(companyID)
	if queryError := expenseQuery.Exec(); queryError != nil {
		return core.Err("Error al leer los gastos de la empresa", companyID, queryError)
	}
	expensesToUpdate := []finance.Expense{}
	for _, expense := range expenses {
		if expense.AccountCode != 0 || expense.Type == finance.ExpenseTypeDepreciation {
			continue
		}
		if expense.Type == finance.ExpenseTypeInventory {
			expense.AccountCode = supplyAccountByID[expense.ProductID]
		} else {
			category, _ := finance.FindExpenseCategory(expense.CategoryID)
			expense.AccountCode = category.DefaultAccountCode
		}
		if expense.AccountCode == 0 {
			report.expensesLeftUnclassified++
			continue
		}
		expensesToUpdate = append(expensesToUpdate, expense)
	}
	report.expenses += len(expensesToUpdate)

	schedules := []finance.ExpenseScheduled{}
	scheduleQuery := db.Query(&schedules)
	scheduleQuery.Select(scheduleQuery.CompanyID, scheduleQuery.ID,scheduleQuery.CategoryID, scheduleQuery.AccountCode).
		CompanyID.Equals(companyID)
	if queryError := scheduleQuery.Exec(); queryError != nil {
		return core.Err("Error al leer los gastos programados de la empresa", companyID, queryError)
	}
	schedulesToUpdate := []finance.ExpenseScheduled{}
	for _, schedule := range schedules {
		if schedule.AccountCode != 0 {
			continue
		}
		category, _ := finance.FindExpenseCategory(schedule.CategoryID)
		if category.DefaultAccountCode == 0 {
			report.schedulesLeftUnclassified++
			continue
		}
		schedule.AccountCode = category.DefaultAccountCode
		schedulesToUpdate = append(schedulesToUpdate, schedule)
	}
	report.schedules += len(schedulesToUpdate)

	// 3. Purchase orders: every supply line takes its supply's account.
	orders := []logistics.PurchaseOrder{}
	orderQuery := db.Query(&orders)
	orderQuery.Select(orderQuery.CompanyID, orderQuery.ID,orderQuery.Lines).CompanyID.Equals(companyID)
	if queryError := orderQuery.Exec(); queryError != nil {
		return core.Err("Error al leer las órdenes de compra de la empresa", companyID, queryError)
	}
	ordersToUpdate := []logistics.PurchaseOrder{}
	for _, order := range orders {
		orderChanged := false
		for lineIndex := range order.Lines {
			line := &order.Lines[lineIndex]
			if supplyAccountCode, isSupply := supplyAccountByID[line.ProductID]; isSupply && line.AccountCode == 0 {
				line.AccountCode = supplyAccountCode
				orderChanged = true
			}
		}
		if orderChanged {
			ordersToUpdate = append(ordersToUpdate, order)
		}
	}
	report.purchaseOrders += len(ordersToUpdate)

	// 4. Assets: the supply's 33x, else generic equipment.
	assets := []accounting.Asset{}
	assetQuery := db.Query(&assets)
	assetQuery.Select(assetQuery.CompanyID, assetQuery.ID,assetQuery.ProductID, assetQuery.AccountCode).CompanyID.Equals(companyID)
	if queryError := assetQuery.Exec(); queryError != nil {
		return core.Err("Error al leer los activos de la empresa", companyID, queryError)
	}
	assetsToUpdate := []accounting.Asset{}
	for _, asset := range assets {
		if asset.AccountCode != 0 {
			continue
		}
		asset.AccountCode, _ = accounting.ResolveAssetAccount(0, supplyAccountByID[asset.ProductID])
		assetsToUpdate = append(assetsToUpdate, asset)
	}
	report.assets += len(assetsToUpdate)

	if !shouldWrite {
		return nil
	}
	updateError := func(table string, err error) error {
		return core.Err("Error al actualizar", table, "de la empresa", companyID, err)
	}
	if len(catalogToUpdate) > 0 {
		productTable := db.TableOf[production.Product]()
		if err := db.Update(&catalogToUpdate, productTable.Origin, productTable.AccountCode); err != nil {
			return updateError("los productos", err)
		}
	}
	if len(expensesToUpdate) > 0 {
		if err := db.Update(&expensesToUpdate, db.TableOf[finance.Expense]().AccountCode); err != nil {
			return updateError("los gastos", err)
		}
	}
	if len(schedulesToUpdate) > 0 {
		if err := db.Update(&schedulesToUpdate, db.TableOf[finance.ExpenseScheduled]().AccountCode); err != nil {
			return updateError("los gastos programados", err)
		}
	}
	if len(ordersToUpdate) > 0 {
		if err := db.Update(&ordersToUpdate, db.TableOf[logistics.PurchaseOrder]().Lines); err != nil {
			return updateError("las órdenes de compra", err)
		}
	}
	if len(assetsToUpdate) > 0 {
		if err := db.Update(&assetsToUpdate, db.TableOf[accounting.Asset]().AccountCode); err != nil {
			return updateError("los activos", err)
		}
	}
	return nil
}
