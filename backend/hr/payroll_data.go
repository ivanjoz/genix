package hr

import (
	"fmt"

	"app/db"
	"app/hr/types"
)

// The payroll's read model (types.PayrollData and the company loaders) lives in hr/types, shared
// with the cash flow forecast. What stays here is what only the payroll's writes need.

// lockRunIn reads a run under its row lock. Every write of a run or of what it settles holds it, in
// this order: employee, run(s), records, cash bank. So a close, a reopen, a payment, a
// recalculation and a record edit of its month apply one after the other, each on what the
// previous one left.
func lockRunIn(transaction *db.Transaction, companyID, runID int32) (types.PayrollRun, error) {
	runs := []types.PayrollRun{{CompanyID: companyID, ID: runID}}
	if err := db.LockByKeys(transaction, &runs); err != nil {
		return types.PayrollRun{}, fmt.Errorf("error al leer la planilla: %v", err)
	}
	if len(runs) == 0 || runs[0].Status == types.StatusDeleted {
		return types.PayrollRun{}, fmt.Errorf("la planilla %d no existe", runID)
	}
	return runs[0], nil
}

func loadCompanyRuns(companyID int32) ([]types.PayrollRun, error) {
	runs := []types.PayrollRun{}
	query := db.Query(&runs)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las planillas: %v", err)
	}
	return runs, nil
}

func loadRunPayslips(companyID, runID int32) ([]types.Payslip, error) {
	payslips := []types.Payslip{}
	query := db.Query(&payslips)
	query.CompanyID.Equals(companyID).RunID.Equals(runID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las boletas: %v", err)
	}
	return payslips, nil
}
