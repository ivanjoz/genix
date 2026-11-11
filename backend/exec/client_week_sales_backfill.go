// Rebuilding client_week_sales from the sales already stored.
//
// The table was added after the sales existed, and from then on every sale write keeps it by
// increments. This rebuilds a company's rows whole from its sale orders (sales.BuildClientWeekSales,
// the same shares the increments add), so it also repairs rows that drifted. It overwrites: run it
// while nothing sells for the company, or a sale written meanwhile is lost from its row.

package exec

import (
	config "app/config/types"
	"app/core"
	"app/db"
	sales "app/sales/types"
	"fmt"
	"slices"
)

type clientWeekSalesBackfillReport struct {
	companies     int
	saleOrders    int
	rowsBuilt     int
	rowsZeroed    int
	rowsWritten   int
	rowsUnchanged int
}

// BackfillClientWeekSales rebuilds the client week rows of every company (or one).
//
// fn-backfill-client-week-sales [apply] [companyID]
//
// Without `apply` it only reports what it would write. Re-running is safe: it rebuilds the same rows.
func BackfillClientWeekSales(args *core.ExecArgs) core.FuncResponse {
	shouldWrite, onlyCompanyID := parseIdentityBackfillArguments(args.Message)

	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", queryError)
	}

	report := clientWeekSalesBackfillReport{}
	for _, company := range companies {
		if onlyCompanyID > 0 && company.ID != onlyCompanyID {
			continue
		}
		report.companies++
		if err := backfillCompanyClientWeekSales(company.ID, shouldWrite, &report); err != nil {
			return args.MakeErr(err)
		}
	}

	mode := "SIMULACIÓN (agregue 'apply' para escribir)"
	if shouldWrite {
		mode = "APLICADO"
	}
	message := fmt.Sprintf("Backfill de ventas semanales por cliente %v: %d empresa(s) · %d venta(s) · "+
		"%d fila(s) calculadas, %d a cero, %d sin cambios, %d escritas",
		mode, report.companies, report.saleOrders, report.rowsBuilt, report.rowsZeroed, report.rowsUnchanged, report.rowsWritten)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

// backfillCompanyClientWeekSales writes the rows the company's sales add up to, and zeroes the stored
// rows no sale accounts for any more. A row already holding its value is not written.
func backfillCompanyClientWeekSales(companyID int32, shouldWrite bool, report *clientWeekSalesBackfillReport) error {
	saleOrders := []sales.SaleOrder{}
	saleQuery := db.Query(&saleOrders)
	saleQuery.Select().CompanyID.Equals(companyID)
	if err := saleQuery.Exec(); err != nil {
		return core.Err("Error al leer las ventas de la empresa", companyID, err)
	}
	report.saleOrders += len(saleOrders)

	storedRows := []sales.ClientWeekSales{}
	rowQuery := db.Query(&storedRows)
	rowQuery.Select().CompanyID.Equals(companyID)
	if err := rowQuery.Exec(); err != nil {
		return core.Err("Error al leer las ventas semanales de la empresa", companyID, err)
	}
	storedByKey := map[[2]int32]sales.ClientWeekSales{}
	for _, row := range storedRows {
		storedByKey[[2]int32{int32(row.Week), row.ClientID}] = row
	}

	builtRows := sales.BuildClientWeekSales(companyID, saleOrders)
	report.rowsBuilt += len(builtRows)
	rowsToWrite := []sales.ClientWeekSales{}
	for _, row := range builtRows {
		key := [2]int32{int32(row.Week), row.ClientID}
		stored, isStored := storedByKey[key]
		delete(storedByKey, key)
		if isStored && slices.Equal(stored.Purchases, row.Purchases) && slices.Equal(stored.Payments, row.Payments) {
			report.rowsUnchanged++
			continue
		}
		rowsToWrite = append(rowsToWrite, row)
	}
	// A stored row no sale accounts for (its sales annulled or moved) goes back to zero.
	isNonZero := func(amount int32) bool { return amount != 0 }
	for _, stored := range storedByKey {
		// Already zero: nothing to write.
		if !slices.ContainsFunc(stored.Purchases, isNonZero) && !slices.ContainsFunc(stored.Payments, isNonZero) {
			report.rowsUnchanged++
			continue
		}
		zeroed := sales.NewClientWeekSales(companyID, sales.ClientWeekSaleShare{ClientID: stored.ClientID})
		zeroed.Week = stored.Week
		rowsToWrite = append(rowsToWrite, zeroed)
		report.rowsZeroed++
	}
	if !shouldWrite || len(rowsToWrite) == 0 {
		return nil
	}
	if err := db.Upsert(&rowsToWrite); err != nil {
		return core.Err("Error al escribir las ventas semanales de la empresa", companyID, err)
	}
	report.rowsWritten += len(rowsToWrite)
	return nil
}
