package hr

import (
	"encoding/json"
	"fmt"
	"slices"

	"app/core"
	"app/db"
	finance "app/finance/types"
	"app/hr/types"
)

// Payroll money goes through the cash ledger of app/finance (finance/types.ApplyCashBankMovement):
// the accounting projection posts each movement against the payable it carries (plan §5.2).

const currencyPEN = 1

// recordPayrollCash writes one payroll cash movement. A negative amount is money out (a payment),
// a positive one gives it back (a voided advance, a deleted loan).
func recordPayrollCash(req *core.HandlerArgs, cashBankID int32, movementType finance.CashMovementType,
	documentID int32, counterAccountCode int32, date int16, amount int32) error {
	cashBank, err := finance.GetCaja(req.User.CompanyID, cashBankID)
	if err != nil {
		return err
	}
	if cashBank.CurrencyType != currencyPEN {
		return fmt.Errorf("La planilla se paga en soles: elija una caja o cuenta en soles.")
	}
	return finance.ApplyCashBankMovement(req, []finance.InternalCashMovement{{
		CashBankID: cashBankID, DocumentID: int64(documentID), Date: date, Type: movementType,
		CounterAccountCode: counterAccountCode, Amount: amount,
	}})
}

// contributionsRequest pays one payable a closed run owes: EsSalud (4031), ONP (4032), 5ta (40173),
// AFP (417) or a judicial withholding (419).
type contributionsRequest struct {
	RunID       int32
	CashBankID  int32
	AccountCode int32
	Amount      int32
	Date        int16
}

// PostPayrollContributions registers the payment of a run's contributions or withholdings to a
// third party (PLAME, AFPnet, a beneficiary). It refuses to pay more than the run still owes on
// that account.
func PostPayrollContributions(req *core.HandlerArgs) core.HandlerResponse {
	if !req.User.HasSubAcceso(accessPayroll, subAccessRegisterPaid) {
		return req.MakeErr("No tiene permiso para registrar pagos de planilla.")
	}
	request := contributionsRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	payableAccounts := finance.DocumentPaymentAccounts[finance.CashMovementTypePayrollContributions]
	if !slices.Contains(payableAccounts, request.AccountCode) {
		return req.MakeErr(fmt.Sprintf("La cuenta %d no es un aporte o retención de planilla.", request.AccountCode))
	}
	if request.Amount <= 0 || request.CashBankID <= 0 {
		return req.MakeErr("Indique el monto y la caja o cuenta de pago.")
	}
	companyID := req.User.CompanyID
	run, err := loadRun(companyID, request.RunID)
	if err != nil {
		return req.MakeErr(err)
	}
	if run.Status < types.PayrollStatusClosed {
		return req.MakeErr("Solo se pagan los aportes de una planilla cerrada.")
	}

	owed, err := runPayableBalance(companyID, run, request.AccountCode)
	if err != nil {
		return req.MakeErr(err)
	}
	if request.Amount > owed {
		return req.MakeErr(fmt.Sprintf("El monto supera lo pendiente de la cuenta %d en esta planilla (S/ %d.%02d).",
			request.AccountCode, owed/100, owed%100))
	}
	if request.Date == 0 {
		request.Date = core.FechaUnix()
	}
	if err := recordPayrollCash(req, request.CashBankID, finance.CashMovementTypePayrollContributions, run.ID,
		request.AccountCode, request.Date, -request.Amount); err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(map[string]any{"Pending": owed - request.Amount})
}

// runPayableBalance is what a run's payslips credited to a payable, less what was already paid on
// it: deductions credit their payable, employer contributions theirs.
func runPayableBalance(companyID int32, run types.PayrollRun, accountCode int32) (int32, error) {
	payslips, err := loadRunPayslips(companyID, run.ID)
	if err != nil {
		return 0, err
	}
	var owed int32
	for _, payslip := range payslips {
		if payslip.Status < types.PayrollStatusClosed {
			continue
		}
		for _, line := range payslip.Lines {
			if line.PayableAccountCode == accountCode &&
				(line.Kind == types.ConceptKindDeduction || line.Kind == types.ConceptKindEmployer) {
				owed += line.Amount
			}
		}
	}
	movements := []finance.CashBankMovement{}
	query := db.Query(&movements)
	query.Select().CompanyID.Equals(companyID).DocumentID.Equals(int64(run.ID))
	if err := query.Exec(); err != nil {
		return 0, fmt.Errorf("error al leer los pagos de la planilla: %v", err)
	}
	for _, movement := range movements {
		if movement.Type == finance.CashMovementTypePayrollContributions && movement.CounterAccountCode == accountCode {
			owed += movement.Amount // payments are negative
		}
	}
	return owed, nil
}
