package finance

import (
	"app/core"
	crm "app/crm/types"
	"app/db"
	"app/finance/types"
	"encoding/json"
)

// GetLoans syncs the company's active and paid-off loans, installments included.
func GetLoans(req *core.HandlerArgs) core.HandlerResponse {
	loans := []types.Loan{}
	query := db.Query(&loans)
	query.Select().CompanyID.Equals(req.User.CompanyID).
		Delta(req.GetDeltaCursor(), types.LoanStatusActive, types.LoanStatusPaidOff)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los préstamos:", err)
	}
	return core.MakeResponse(req, &loans)
}

// PostLoans creates, edits or (ss = 0) removes a loan. What the client cannot decide is taken from
// the stored row: the installments' paid amounts and whether the disbursement was registered, which
// also locks the principal, the account, the currency and the disbursement date.
func PostLoans(req *core.HandlerArgs) core.HandlerResponse {
	loan := types.Loan{}
	if err := json.Unmarshal([]byte(*req.Body), &loan); err != nil {
		return req.MakeErr("Error al deserializar el préstamo:", err)
	}
	loan.CompanyID = req.User.CompanyID
	if loan.Status == 0 {
		return removeLoan(req, loan.ID)
	}
	if err := validateLoanLender(req.User.CompanyID, loan.LenderID); err != nil {
		return req.MakeErr(err)
	}

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		nowTime := core.SUnixTime()
		isNew := loan.ID <= 0
		if isNew {
			for index := range loan.Installments {
				loan.Installments[index].PaidAmount = 0
			}
			loan.Created, loan.CreatedBy = nowTime, req.User.ID
			loan.IsDisbursementRegistered = loan.RegisterDisbursement
		} else {
			storedLoans := []types.Loan{{CompanyID: loan.CompanyID, ID: loan.ID}}
			if err := db.LockByKeys(transaction, &storedLoans); err != nil {
				return core.Err("Error al obtener el préstamo:", err)
			}
			if len(storedLoans) == 0 || storedLoans[0].Status == 0 {
				return core.Err("El préstamo no existe.")
			}
			if err := carryStoredLoanState(&loan, storedLoans[0]); err != nil {
				return err
			}
		}

		cashBank, err := types.GetCaja(loan.CompanyID, loan.CashBankID)
		if err != nil {
			return err
		}
		if cashBank.CurrencyType != loan.CurrencyType {
			return core.Err("La moneda del préstamo no coincide con la de la caja o banco.")
		}
		if err := types.ValidateLoan(&loan); err != nil {
			return err
		}
		loan.Status = core.If(loan.IsPaidOff(), types.LoanStatusPaidOff, types.LoanStatusActive)
		loan.Updated, loan.UpdatedBy = nowTime, req.User.ID

		if !isNew {
			table := db.TableOf[types.Loan]()
			return db.UpdateIn(transaction, &[]types.Loan{loan}, table.Name, table.LenderID, table.CashBankID,
				table.CurrencyType, table.Principal, table.DisbursementDate, table.AnnualRate, table.Method,
				table.GraceMonths, table.Installments, table.Notes, table.Status, table.Updated, table.UpdatedBy)
		}
		loans := []types.Loan{loan}
		if err := db.InsertIn(transaction, &loans); err != nil {
			return core.Err("Error al registrar el préstamo:", err)
		}
		loan = loans[0]
		if !loan.IsDisbursementRegistered {
			return nil
		}
		// The borrowed money arrives on the disbursement date, against the loan's payable (451).
		return types.ApplyCashBankMovementIn(transaction, req, []types.InternalCashMovement{{
			CashBankID: loan.CashBankID, DocumentID: int64(loan.ID), Date: loan.DisbursementDate,
			Type: types.CashMovementTypeLoanDisbursement, CounterAccountCode: types.PCGELoansReceived,
			Amount: loan.Principal,
		}})
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(loan)
}

// carryStoredLoanState copies onto an edit what only the server knows, and refuses the changes a
// registered disbursement or a paid installment rules out.
func carryStoredLoanState(loan *types.Loan, stored types.Loan) error {
	loan.IsDisbursementRegistered, loan.RegisterDisbursement = stored.IsDisbursementRegistered, false
	loan.Created, loan.CreatedBy = stored.Created, stored.CreatedBy
	if stored.IsDisbursementRegistered && (loan.Principal != stored.Principal || loan.CashBankID != stored.CashBankID ||
		loan.CurrencyType != stored.CurrencyType || loan.DisbursementDate != stored.DisbursementDate) {
		return core.Err("El desembolso ya se registró en caja: el monto, la caja, la moneda y la fecha del desembolso no pueden cambiar.")
	}
	for index := range loan.Installments {
		loan.Installments[index].PaidAmount = 0
		if storedInstallment := stored.FindInstallment(loan.Installments[index].Number); storedInstallment != nil {
			loan.Installments[index].PaidAmount = storedInstallment.PaidAmount
		}
	}
	for _, storedInstallment := range stored.Installments {
		if storedInstallment.PaidAmount > 0 && loan.FindInstallment(storedInstallment.Number) == nil {
			return core.Err("La cuota", storedInstallment.Number, "ya tiene pagos y no puede quitarse.")
		}
	}
	return nil
}

// removeLoan deletes a loan nothing has moved cash for yet: once its disbursement or a payment is in
// a cash account, removing it would leave those movements explaining nothing.
func removeLoan(req *core.HandlerArgs, loanID int32) core.HandlerResponse {
	if loanID <= 0 {
		return req.MakeErr("No se puede eliminar un préstamo que no se ha guardado.")
	}
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		loans := []types.Loan{{CompanyID: req.User.CompanyID, ID: loanID}}
		if err := db.LockByKeys(transaction, &loans); err != nil {
			return core.Err("Error al obtener el préstamo:", err)
		}
		if len(loans) == 0 {
			return core.Err("El préstamo no existe.")
		}
		loan := loans[0]
		hasPayments := false
		for _, installment := range loan.Installments {
			hasPayments = hasPayments || installment.PaidAmount > 0
		}
		if loan.IsDisbursementRegistered || hasPayments {
			return core.Err("El préstamo ya movió caja (desembolso o pagos) y no puede eliminarse.")
		}
		loan.Status, loan.Updated, loan.UpdatedBy = 0, core.SUnixTime(), req.User.ID
		table := db.TableOf[types.Loan]()
		return db.UpdateIn(transaction, &[]types.Loan{loan}, table.Status, table.Updated, table.UpdatedBy)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(map[string]int32{"ID": loanID})
}

// validateLoanLender: the lender must be an active provider of the company.
func validateLoanLender(companyID, lenderID int32) error {
	lenders := []crm.ClientProvider{}
	query := db.Query(&lenders)
	query.Select(query.ID, query.Type, query.Status).CompanyID.Equals(companyID).ID.Equals(lenderID)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener la entidad que presta:", err)
	}
	if len(lenders) == 0 || lenders[0].Status == 0 || lenders[0].Type != crm.ClientProviderTypeProvider {
		return core.Err("La entidad que presta debe ser un proveedor activo.")
	}
	return nil
}

// LoanPaymentBody is one payment of one installment.
type LoanPaymentBody struct {
	LoanID            int32
	InstallmentNumber int16
	CashBankID        int32
	Amount            int32 // positive, cents of the loan's currency
	Date              int16 // UnixDay; today when 0
}

// PostLoanPayment pays an installment, fully or in part, as one cash movement split into the fees,
// interest and principal it clears (in that order). The loan and the cash account are locked, so two
// payments of one installment apply one after the other.
func PostLoanPayment(req *core.HandlerArgs) core.HandlerResponse {
	body := LoanPaymentBody{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el pago:", err)
	}
	if body.LoanID <= 0 || body.InstallmentNumber <= 0 || body.CashBankID <= 0 {
		return req.MakeErr("Faltan parámetros: (LoanID, InstallmentNumber o CashBankID).")
	}
	paymentDate := core.If(body.Date != 0, body.Date, core.FechaUnix())

	var loan types.Loan
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		loans := []types.Loan{{CompanyID: req.User.CompanyID, ID: body.LoanID}}
		if err := db.LockByKeys(transaction, &loans); err != nil {
			return core.Err("Error al obtener el préstamo:", err)
		}
		if len(loans) == 0 || loans[0].Status != types.LoanStatusActive {
			return core.Err("El préstamo no existe o ya está cancelado.")
		}
		loan = loans[0]
		if paymentDate < loan.DisbursementDate {
			return core.Err("El pago no puede ser anterior al desembolso.")
		}
		if err := recomputeLoanPaidAmounts(req.User.CompanyID, &loan); err != nil {
			return err
		}
		installment := loan.FindInstallment(body.InstallmentNumber)
		if installment == nil {
			return core.Err("El préstamo no tiene la cuota", body.InstallmentNumber)
		}
		splits, err := types.LoanPaymentSplits(*installment, body.Amount)
		if err != nil {
			return err
		}

		cashBanks := []types.CashBank{{CompanyID: req.User.CompanyID, ID: body.CashBankID}}
		if err := db.LockByKeys(transaction, &cashBanks); err != nil {
			return core.Err("Error al obtener la caja:", err)
		}
		if len(cashBanks) == 0 {
			return core.Err("No se encontró la caja.")
		}
		if cashBanks[0].CurrencyType != loan.CurrencyType {
			return core.Err("La moneda de la caja no coincide con la del préstamo.")
		}
		if cashBanks[0].CurrentAmount-body.Amount < 0 {
			return core.Err("El saldo de la caja no puede quedar negativo.")
		}

		if err := types.ApplyCashBankMovementIn(transaction, req, []types.InternalCashMovement{{
			CashBankID: body.CashBankID, DocumentID: int64(loan.ID), ReferenceID: int32(installment.Number),
			Date: paymentDate, Type: types.CashMovementTypeLoanPayment, CounterAccountCode: types.PCGELoansReceived,
			Amount: -body.Amount, DocumentDate: installment.DueDate, CounterAccountSplits: splits,
		}}); err != nil {
			return err
		}

		installment.PaidAmount += body.Amount
		loan.Status = core.If(loan.IsPaidOff(), types.LoanStatusPaidOff, types.LoanStatusActive)
		loan.Updated, loan.UpdatedBy = core.SUnixTime(), req.User.ID
		table := db.TableOf[types.Loan]()
		return db.UpdateIn(transaction, &[]types.Loan{loan}, table.Installments, table.Status, table.Updated, table.UpdatedBy)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(loan)
}

// recomputeLoanPaidAmounts sets every installment's PaidAmount from the loan's committed payments, so
// it cannot drift from the ledger. Type filters out the other documents sharing the DocumentID.
func recomputeLoanPaidAmounts(companyID int32, loan *types.Loan) error {
	movements := []types.CashBankMovement{}
	query := db.Query(&movements)
	query.Select(query.Type, query.Amount, query.ReferenceID).CompanyID.Equals(companyID).DocumentID.Equals(int64(loan.ID))
	if err := query.Exec(); err != nil {
		return core.Err("Error al recalcular lo pagado del préstamo:", err)
	}
	paidByInstallment := map[int16]int32{}
	for _, movement := range movements {
		if movement.Type == types.CashMovementTypeLoanPayment {
			paidByInstallment[int16(movement.ReferenceID)] += -movement.Amount
		}
	}
	for index := range loan.Installments {
		loan.Installments[index].PaidAmount = paidByInstallment[loan.Installments[index].Number]
	}
	return nil
}
