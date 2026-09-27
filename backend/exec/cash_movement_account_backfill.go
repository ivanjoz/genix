// Backfilling CashBankMovement.AccountCode on the movements written before the column existed.
//
// Withdrawals, losses and physical counts now carry the PCGE account the user classified them
// under, and the Libro Diario posts the other side of the cash against it. Every row written
// before the column is 0, so without this those entries would have no counter-account.
//
// A backfill and not a read-time default on purpose (CLAUDE.md §3). Each row gets the first
// option of its type and sign in finance.CashMovementAccountOptions — the same default the form
// preselects — because the choice the user would have made was never recorded.
//
// What it leaves alone: "Unspecified" movements (type 1). No account is correct for a movement
// that never said why the money moved, so they are only counted.
//
// It also counts the movements written with Date = 0 (the reconciliation handler never set it).
// Those need re-keying, not an update, so they are reported and not written.

package exec

import (
	config "app/config/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	"fmt"
)

type cashMovementAccountBackfillReport struct {
	companies          int
	accountsPending    int
	undatedMovements   int
	movementsWritten   int
	unspecifiedSkipped int
}

// BackfillCashMovementAccounts stamps the default account on classified movements with none.
//
// fn-backfill-cash-movement-accounts [apply] [companyID]
//
// It reports without writing unless `apply` is given. Re-running is safe: it only touches rows
// whose AccountCode is still 0.
func BackfillCashMovementAccounts(args *core.ExecArgs) core.FuncResponse {
	shouldWrite, onlyCompanyID := parseIdentityBackfillArguments(args.Message)

	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", queryError)
	}

	report := cashMovementAccountBackfillReport{}
	for _, company := range companies {
		if onlyCompanyID > 0 && company.ID != onlyCompanyID {
			continue
		}
		report.companies++
		if backfillError := backfillCompanyCashMovementAccounts(
			company.ID, shouldWrite, &report); backfillError != nil {
			return args.MakeErr(backfillError)
		}
	}

	mode := "SIMULACIÓN (agregue 'apply' para escribir)"
	if shouldWrite {
		mode = "APLICADO"
	}
	message := fmt.Sprintf(
		"Backfill de cuentas contables de movimientos de caja %v: %d empresa(s) · "+
			"pendientes %d, escritos %d · sin especificar (no tocados) %d · sin fecha (no tocados) %d",
		mode, report.companies, report.accountsPending, report.movementsWritten,
		report.unspecifiedSkipped, report.undatedMovements,
	)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

func backfillCompanyCashMovementAccounts(
	companyID int32, shouldWrite bool, report *cashMovementAccountBackfillReport,
) error {

	movements := []finance.CashBankMovement{}
	query := db.Query(&movements)
	query.Select().CompanyID.Equals(companyID)
	if queryError := query.Exec(); queryError != nil {
		return core.Err("Error al leer los movimientos de caja de la empresa", companyID, queryError)
	}

	movementsToUpdate := []finance.CashBankMovement{}
	for _, movement := range movements {
		// Only counted: the ID packs CashBankID + Date, so an undated row sits under date 0 in
		// the key and a period query never reaches it. Writing Date alone would leave the column
		// and the key disagreeing; the row has to be re-keyed, which the ORM cannot delete for.
		if movement.Date == 0 {
			report.undatedMovements++
		}
		if movement.AccountCode != 0 {
			continue
		}
		if movement.Type == finance.CashMovementTypeUnspecified {
			report.unspecifiedSkipped++
			continue
		}
		// A type the user does not classify, or a zero-amount count, has no option and stays 0.
		options := finance.CashMovementAccountOptionsFor(movement.Type, movement.Amount)
		if len(options) == 0 {
			continue
		}
		movement.AccountCode = options[0].AccountCode
		movementsToUpdate = append(movementsToUpdate, movement)
	}
	report.accountsPending += len(movementsToUpdate)

	if len(movementsToUpdate) == 0 || !shouldWrite {
		return nil
	}
	movementTable := db.TableOf[finance.CashBankMovement]()
	if updateError := db.Update(&movementsToUpdate, movementTable.AccountCode); updateError != nil {
		return core.Err("Error al actualizar los movimientos de caja de la empresa", companyID, updateError)
	}
	report.movementsWritten += len(movementsToUpdate)
	return nil
}
