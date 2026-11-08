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

// The employee records (types.EmployeeRecord): the month's novedades, time off, weekly payments and
// loans. One table, one read route; the writes stay one per screen because advances and loans move
// cash. Every write is refused once the run of a month it touches is closed: reopen the run first.

func queryPeriod(req *core.HandlerArgs) (int16, error) {
	period := int16(req.GetQueryInt("period"))
	if period <= 0 || types.PeriodMonth(period) < 1 || types.PeriodMonth(period) > 12 {
		return 0, fmt.Errorf("Debe enviar un periodo válido (AAMM).")
	}
	return period, nil
}

// GetEmployeeRecords syncs the records of one year (?year=YYYY: time off, and the month the payroll
// pages filter) or of one employee (?employee-id=: his loans and their installments).
func GetEmployeeRecords(req *core.HandlerArgs) core.HandlerResponse {
	records := []types.EmployeeRecord{}
	query := db.Query(&records)
	query.Select().CompanyID.Equals(req.User.CompanyID)
	year, employeeID := int16(req.GetQueryInt("year")), req.GetQueryInt("employee-id")
	switch {
	case year >= 2000 && year <= 2099:
		query.Year.Equals(year)
	case employeeID > 0:
		query.EmployeeID.Equals(employeeID)
	default:
		return req.MakeErr("Debe enviar un año válido o employee-id.")
	}
	query.Delta(req.GetDeltaCursor(), types.RecordStatusPending, types.RecordStatusActive, types.RecordStatusSettled)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros del personal:", err)
	}
	return core.MakeResponse(req, &records)
}

// PostEmployeeRecord creates, edits or cancels (Status 0) a Registros record (overtime, an absence, a
// company concept amount) or a time off, or moves a loan installment to another month. Advances and
// loans have their own routes: they move cash.
func PostEmployeeRecord(req *core.HandlerArgs) core.HandlerResponse {
	request := types.EmployeeRecord{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	var saved types.EmployeeRecord
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		saved, err = saveEmployeeRecordIn(transaction, req, request)
		return err
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(saved)
}

// saveEmployeeRecordIn validates and writes one record under the locks of its employee, of the runs
// of every month it touches before and after the change, and of the record itself.
func saveEmployeeRecordIn(transaction *db.Transaction, req *core.HandlerArgs, request types.EmployeeRecord) (types.EmployeeRecord, error) {
	companyID := req.User.CompanyID
	companyRecords, err := loadCompanyRecords(companyID)
	if err != nil {
		return types.EmployeeRecord{}, err
	}

	// The record is rebuilt from what the client may set: never its cash, loan or payslip links.
	record := types.EmployeeRecord{CompanyID: companyID, EmployeeID: request.EmployeeID, ConceptID: request.ConceptID}
	touchedPeriods := []int16{}
	if request.ID > 0 {
		stored, found := findRecord(companyRecords, request.ID)
		if !found {
			return record, fmt.Errorf("El registro %d no existe.", request.ID)
		}
		// Advances and loans move cash: they are voided on their own routes. An installment only moves.
		isOwnScreen := types.IsEntryRecord(stored.ConceptID) || types.IsTimeOffRecord(stored.ConceptID) ||
			(stored.ConceptID == types.LegalLoanInstallment && request.Status != types.StatusDeleted)
		if !isOwnScreen {
			return record, fmt.Errorf("Este registro se modifica desde su propia pantalla.")
		}
		if recordGroup(stored.ConceptID) != recordGroup(request.ConceptID) && request.Status != types.StatusDeleted {
			return record, fmt.Errorf("No se puede cambiar el registro a un concepto de otra pantalla.")
		}
		// The months it touched before the change must still be open.
		touchedPeriods = append(touchedPeriods, recordPeriods(stored)...)
		record = stored
		if request.Status == types.StatusDeleted {
			record.Status = types.StatusDeleted
		} else {
			record.ConceptID = request.ConceptID
		}
	}
	// The employee's lock comes first: the time off overlap check below reads his other records.
	if _, err := lockEmployeeIn(transaction, companyID, record.EmployeeID); err != nil {
		return record, err
	}

	if record.Status != types.StatusDeleted || record.ID == 0 {
		record.Period, record.FromDate, record.ToDate, record.Note = request.Period, request.FromDate, request.ToDate, request.Note
		var validationError error
		switch {
		case types.IsEntryRecord(record.ConceptID):
			record.Date, record.Quantity, record.Amount = request.Date, request.Quantity, request.Amount
			validationError = validateEntryRecord(companyID, &record)
		case types.IsTimeOffRecord(record.ConceptID):
			validationError = validateTimeOff(companyID, &record, companyRecords)
			record.ApprovedBy = req.User.ID
		case record.ConceptID == types.LegalLoanInstallment && record.ID > 0:
			// Only its month moves; the amount and the loan stay.
			validationError = validateInstallmentMove(record, companyRecords)
		default:
			validationError = fmt.Errorf("Este concepto se registra desde su propia pantalla.")
		}
		if validationError != nil {
			return record, validationError
		}
		record.SelfParse()
		touchedPeriods = append(touchedPeriods, recordPeriods(record)...)
		record.Status = types.RecordStatusActive
	}
	if err := lockOpenPeriodsIn(transaction, companyID, touchedPeriods...); err != nil {
		return record, err
	}

	record.UpdatedBy = req.User.ID
	records := []types.EmployeeRecord{record}
	if record.ID == 0 {
		records[0].Created, records[0].CreatedBy = core.SUnixTime(), req.User.ID
		if err := db.InsertIn(transaction, &records); err != nil {
			return record, fmt.Errorf("Error al guardar el registro: %v", err)
		}
		return records[0], nil
	}
	// Re-checked on the locked row: a close of its month may have settled it since the read.
	lockedRecords, err := lockRecordsIn(transaction, companyID, []int32{record.ID})
	if err != nil {
		return record, err
	}
	if len(lockedRecords) == 0 || lockedRecords[0].Status == types.StatusDeleted {
		return record, fmt.Errorf("El registro %d no existe.", record.ID)
	}
	if lockedRecords[0].Status == types.RecordStatusSettled {
		return record, fmt.Errorf("El registro ya fue usado por una planilla cerrada.")
	}
	table := db.TableOf[types.EmployeeRecord]()
	if err := db.UpdateIn(transaction, &records, table.ConceptID, table.Period, table.Year, table.Date, table.FromDate, table.ToDate,
		table.Quantity, table.Amount, table.Note, table.ApprovedBy, table.Status, table.UpdatedBy); err != nil {
		return record, fmt.Errorf("Error al guardar el registro: %v", err)
	}
	return records[0], nil
}

// recordGroup is the screen a record's concept belongs to: a record never moves between screens.
func recordGroup(conceptID int32) string {
	switch {
	case types.IsEntryRecord(conceptID):
		return "entry"
	case types.IsTimeOffRecord(conceptID):
		return "time-off"
	case types.IsAdvanceRecord(conceptID):
		return "advance"
	}
	return fmt.Sprint(conceptID)
}

// validateEntryRecord checks a Registros record: the day it happened, the payroll month that pays it
// and its quantity (overtime hours, absence days) or amount (company concept).
func validateEntryRecord(companyID int32, record *types.EmployeeRecord) error {
	if _, err := loadEmployee(companyID, record.EmployeeID); err != nil {
		return err
	}
	if record.Date <= 0 {
		return fmt.Errorf("Indique la fecha del registro.")
	}
	if record.Period <= 0 || types.PeriodMonth(record.Period) < 1 || types.PeriodMonth(record.Period) > 12 {
		return fmt.Errorf("Indique el mes de planilla del registro.")
	}
	// Date is the day it happened; Period the payroll month that pays it, which may be a later month
	// (a late report) but always of the date's year.
	if record.Period/100 != core.PeriodOfDay(record.Date)/100 {
		return fmt.Errorf("El mes de planilla debe ser del mismo año que la fecha del registro.")
	}
	switch record.ConceptID {
	case types.LegalAbsence:
		if record.Quantity < 1 || record.Quantity > 31 || record.Quantity != float32(int(record.Quantity)) {
			return fmt.Errorf("Los días de inasistencia deben ser un número entero entre 1 y 31.")
		}
		record.Amount = 0
	case types.LegalOvertime25, types.LegalOvertime35:
		if record.Quantity <= 0 || record.Quantity > 200 {
			return fmt.Errorf("Las horas extras deben estar entre 0 y 200.")
		}
		record.Amount = 0
	default:
		// IsEntryRecord let only company concepts through besides the legal ones above.
		if _, err := loadConcept(companyID, record.ConceptID); err != nil {
			return err
		}
		if record.Amount <= 0 {
			return fmt.Errorf("El monto debe ser mayor a cero.")
		}
		record.Quantity = 0
	}
	record.FromDate, record.ToDate = 0, 0
	return nil
}

func validateTimeOff(companyID int32, record *types.EmployeeRecord, companyRecords []types.EmployeeRecord) error {
	employee, err := loadEmployee(companyID, record.EmployeeID)
	if err != nil {
		return err
	}
	if record.FromDate <= 0 || record.ToDate < record.FromDate {
		return fmt.Errorf("La fecha final no puede ser anterior a la inicial.")
	}
	if record.ToDate-record.FromDate >= 366 {
		return fmt.Errorf("Registre una ausencia de más de un año en varios tramos.")
	}
	if record.FromDate < employee.HireDate || (employee.TerminationDate > 0 && record.ToDate > employee.TerminationDate) {
		return fmt.Errorf("Las fechas deben estar dentro del periodo laboral del trabajador.")
	}
	for _, other := range companyRecords {
		isLive := other.Status == types.RecordStatusPending || other.Status == types.RecordStatusActive
		if isLive && types.IsTimeOffRecord(other.ConceptID) && other.ID != record.ID && other.EmployeeID == record.EmployeeID &&
			other.FromDate <= record.ToDate && record.FromDate <= other.ToDate {
			return fmt.Errorf("Se cruza con otra ausencia registrada del trabajador.")
		}
	}
	record.Amount = 0
	return nil
}

// validateInstallmentMove lets an installment go to a later month than the loan was granted in.
func validateInstallmentMove(installment types.EmployeeRecord, companyRecords []types.EmployeeRecord) error {
	loan, found := findRecord(companyRecords, installment.LoanID)
	if !found || loan.Status != types.RecordStatusActive {
		return fmt.Errorf("El préstamo de la cuota no existe.")
	}
	if installment.Period < core.PeriodOfDay(loan.Date) || types.PeriodMonth(installment.Period) < 1 || types.PeriodMonth(installment.Period) > 12 {
		return fmt.Errorf("La cuota no puede descontarse antes del mes del préstamo.")
	}
	return nil
}

// PostPayAdvances registers the weekly / biweekly payments of several employees at once, or voids
// one (Status 0) not settled yet. A payment is never edited: void it and register it again.
func PostPayAdvances(req *core.HandlerArgs) core.HandlerResponse {
	advances := []types.EmployeeRecord{}
	if err := json.Unmarshal([]byte(*req.Body), &advances); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if len(advances) == 0 {
		return req.MakeErr("No se enviaron adelantos.")
	}
	var saved []types.EmployeeRecord
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		if len(advances) == 1 && advances[0].ID > 0 {
			saved, err = voidAdvanceIn(transaction, req, advances[0])
		} else {
			saved, err = payAdvancesIn(transaction, req, slices.Clone(advances))
		}
		return err
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(saved)
}

// voidAdvanceIn voids one advance not settled yet and gives its money back, under the locks of its
// employee, its month's run and the advance: a double void refunds once.
func voidAdvanceIn(transaction *db.Transaction, req *core.HandlerArgs, request types.EmployeeRecord) ([]types.EmployeeRecord, error) {
	companyID := req.User.CompanyID
	companyRecords, err := loadCompanyRecords(companyID)
	if err != nil {
		return nil, err
	}
	stored, found := findRecord(companyRecords, request.ID)
	if !found || !types.IsAdvanceRecord(stored.ConceptID) {
		return nil, fmt.Errorf("El adelanto %d no existe.", request.ID)
	}
	if _, err := lockEmployeeIn(transaction, companyID, stored.EmployeeID); err != nil {
		return nil, err
	}
	if err := lockOpenPeriodsIn(transaction, companyID, stored.Period); err != nil {
		return nil, err
	}
	lockedRecords, err := lockRecordsIn(transaction, companyID, []int32{stored.ID})
	if err != nil {
		return nil, err
	}
	if len(lockedRecords) == 0 {
		return nil, fmt.Errorf("El adelanto %d no existe.", request.ID)
	}
	stored = lockedRecords[0]
	if request.Status != types.StatusDeleted || stored.Status != types.RecordStatusActive {
		return nil, fmt.Errorf("Solo se puede anular un adelanto que aún no fue descontado en planilla.")
	}
	// The money comes back to the account that paid it.
	if err := recordPayrollCashIn(transaction, req, stored.CashBankID, finance.CashMovementTypePayrollPayment, stored.ID,
		finance.PCGEStaffAdvances, core.FechaUnix(), stored.Amount); err != nil {
		return nil, err
	}
	stored.Status, stored.UpdatedBy = types.StatusDeleted, req.User.ID
	records := []types.EmployeeRecord{stored}
	table := db.TableOf[types.EmployeeRecord]()
	if err := db.UpdateIn(transaction, &records, table.Status, table.UpdatedBy); err != nil {
		return nil, fmt.Errorf("Error al anular el adelanto: %v", err)
	}
	return records, nil
}

// payAdvancesIn registers new advances and pays them, under the locks of their employees (in key
// order) and of their months' runs: the overlap check reads what every other payment of these
// employees committed, so a double submit cannot pay the same days twice.
func payAdvancesIn(transaction *db.Transaction, req *core.HandlerArgs, advances []types.EmployeeRecord) ([]types.EmployeeRecord, error) {
	companyID := req.User.CompanyID
	advanceEmployeeIDs := []int32{}
	for _, advance := range advances {
		if !slices.Contains(advanceEmployeeIDs, advance.EmployeeID) {
			advanceEmployeeIDs = append(advanceEmployeeIDs, advance.EmployeeID)
		}
	}
	slices.Sort(advanceEmployeeIDs)
	for _, employeeID := range advanceEmployeeIDs {
		if _, err := lockEmployeeIn(transaction, companyID, employeeID); err != nil {
			return nil, err
		}
	}
	companyRecords, err := loadCompanyRecords(companyID)
	if err != nil {
		return nil, err
	}
	employees, err := loadCompanyEmployees(companyID)
	if err != nil {
		return nil, err
	}
	employeesByID := map[int32]types.Employee{}
	for _, employee := range employees {
		employeesByID[employee.ID] = employee
	}
	contracts, err := loadCompanyContracts(companyID)
	if err != nil {
		return nil, err
	}

	advancePeriods := []int16{}
	for i := range advances {
		request := advances[i]
		employee, found := employeesByID[request.EmployeeID]
		if !found || employee.Status == types.StatusDeleted {
			return nil, core.Err(fmt.Sprintf("El trabajador %d no existe.", request.EmployeeID))
		}
		name := employee.FirstName + " " + employee.LastName
		if request.ID != 0 {
			return nil, core.Err("Un adelanto registrado no se edita: anúlelo y regístrelo de nuevo.")
		}
		if !types.IsAdvanceRecord(request.ConceptID) {
			return nil, core.Err("Seleccione el tipo de adelanto.")
		}
		if request.Amount <= 0 || request.Date <= 0 || request.FromDate <= 0 || request.ToDate < request.FromDate {
			return nil, core.Err(fmt.Sprintf("%s: indique el monto, la fecha de pago y las fechas que cubre el adelanto.", name))
		}
		if request.CashBankID <= 0 {
			return nil, core.Err("Seleccione la caja o cuenta de la que se paga el adelanto.")
		}
		advance := types.EmployeeRecord{CompanyID: companyID, EmployeeID: employee.ID, ConceptID: request.ConceptID, Date: request.Date,
			FromDate: request.FromDate, ToDate: request.ToDate, Amount: request.Amount, CashBankID: request.CashBankID,
			Note: request.Note, Status: types.RecordStatusActive, Created: core.SUnixTime(), CreatedBy: req.User.ID,
			UpdatedBy: req.User.ID}
		advance.SelfParse()
		if core.PeriodOfDay(advance.FromDate) != advance.Period {
			return nil, core.Err(fmt.Sprintf("%s: el adelanto debe cubrir días de un solo mes.", name))
		}
		if _, found := types.FindContractInForce(contractsOf(contracts, employee.ID), advance.ToDate); !found {
			return nil, core.Err(fmt.Sprintf("%s no tiene un contrato vigente en las fechas del adelanto.", name))
		}
		for _, other := range append(slices.Clone(companyRecords), advances[:i]...) {
			if types.IsAdvanceRecord(other.ConceptID) && other.EmployeeID == advance.EmployeeID && other.Status != types.StatusDeleted &&
				other.FromDate <= advance.ToDate && advance.FromDate <= other.ToDate {
				return nil, core.Err(fmt.Sprintf("%s ya tiene un adelanto que cubre esas fechas.", name))
			}
		}
		advancePeriods = append(advancePeriods, advance.Period)
		advances[i] = advance
	}
	if err := lockOpenPeriodsIn(transaction, companyID, advancePeriods...); err != nil {
		return nil, err
	}
	if err := db.InsertIn(transaction, &advances); err != nil {
		return nil, core.Err("Error al guardar los adelantos:", err)
	}
	// Each payment leaves its account on its pay date: Dr 1412 · Cr 101/1041 (H14).
	for _, advance := range advances {
		if err := recordPayrollCashIn(transaction, req, advance.CashBankID, finance.CashMovementTypePayrollPayment, advance.ID,
			finance.PCGEStaffAdvances, advance.Date, -advance.Amount); err != nil {
			return nil, err
		}
	}
	return advances, nil
}

// PostLoan grants a loan with its installments, or deletes one (Status 0) none of whose
// installments was recovered by a closed payslip. A granted loan is never edited: delete and grant
// it again, or move an installment (PostEmployeeRecord).
func PostLoan(req *core.HandlerArgs) core.HandlerResponse {
	request := types.EmployeeRecord{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if request.ID > 0 && request.Status != types.StatusDeleted {
		return req.MakeErr("Un préstamo otorgado no se edita: elimínelo y regístrelo de nuevo.")
	}
	var saved []types.EmployeeRecord
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		if request.ID > 0 {
			saved, err = deleteLoanIn(transaction, req, request.ID)
		} else {
			saved, err = grantLoanIn(transaction, req, request)
		}
		return err
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(saved)
}

// grantLoanIn writes a loan, its installments and its disbursement together, under the locks of
// its employee and of its installments' months' runs.
func grantLoanIn(transaction *db.Transaction, req *core.HandlerArgs, request types.EmployeeRecord) ([]types.EmployeeRecord, error) {
	companyID := req.User.CompanyID
	if _, err := lockEmployeeIn(transaction, companyID, request.EmployeeID); err != nil {
		return nil, err
	}
	if request.Amount <= 0 || request.Quantity < 1 || request.Quantity > 60 || request.Quantity != float32(int(request.Quantity)) {
		return nil, fmt.Errorf("Indique el monto y entre 1 y 60 cuotas.")
	}
	if request.Date <= 0 || request.CashBankID <= 0 {
		return nil, fmt.Errorf("Indique la fecha del préstamo y la caja o cuenta de la que se entrega.")
	}
	if request.Period < core.PeriodOfDay(request.Date) || types.PeriodMonth(request.Period) < 1 || types.PeriodMonth(request.Period) > 12 {
		return nil, fmt.Errorf("La primera cuota no puede ser anterior al mes del préstamo.")
	}
	loan := types.EmployeeRecord{CompanyID: companyID, EmployeeID: request.EmployeeID, ConceptID: types.LegalLoan,
		Period: request.Period, Date: request.Date, Amount: request.Amount, Quantity: request.Quantity,
		CashBankID: request.CashBankID, Note: request.Note, Status: types.RecordStatusActive,
		Created: core.SUnixTime(), CreatedBy: req.User.ID, UpdatedBy: req.User.ID}
	loan.SelfParse()
	installments := types.LoanInstallments(loan, loan.Period)
	installmentPeriods := []int16{}
	for _, installment := range installments {
		installmentPeriods = append(installmentPeriods, installment.Period)
	}
	if err := lockOpenPeriodsIn(transaction, companyID, installmentPeriods...); err != nil {
		return nil, err
	}
	loans := []types.EmployeeRecord{loan}
	if err := db.InsertIn(transaction, &loans); err != nil {
		return nil, fmt.Errorf("Error al guardar el préstamo: %v", err)
	}
	for i := range installments {
		installments[i].LoanID, installments[i].Created, installments[i].CreatedBy, installments[i].UpdatedBy =
			loans[0].ID, loan.Created, req.User.ID, req.User.ID
	}
	if err := db.InsertIn(transaction, &installments); err != nil {
		return nil, fmt.Errorf("Error al guardar las cuotas del préstamo: %v", err)
	}
	// The disbursement: Dr 1411 · Cr 101/1041 (H13).
	if err := recordPayrollCashIn(transaction, req, loan.CashBankID, finance.CashMovementTypePayrollPayment, loans[0].ID,
		finance.PCGEStaffLoans, loan.Date, -loan.Amount); err != nil {
		return nil, err
	}
	return append(loans, installments...), nil
}

// deleteLoanIn cancels a loan and its installments and gives the money back to the account it left,
// under the locks of its employee, its installments' months' runs and the records themselves: a
// double delete refunds once, and a close cannot settle an installment the delete cancels.
func deleteLoanIn(transaction *db.Transaction, req *core.HandlerArgs, loanID int32) ([]types.EmployeeRecord, error) {
	companyID := req.User.CompanyID
	companyRecords, err := loadCompanyRecords(companyID)
	if err != nil {
		return nil, err
	}
	loan, found := findRecord(companyRecords, loanID)
	if !found || loan.ConceptID != types.LegalLoan {
		return nil, fmt.Errorf("El préstamo %d no existe.", loanID)
	}
	if _, err := lockEmployeeIn(transaction, companyID, loan.EmployeeID); err != nil {
		return nil, err
	}
	recordIDs, installmentPeriods := []int32{loan.ID}, []int16{}
	for _, installment := range companyRecords {
		if installment.LoanID == loan.ID {
			recordIDs = append(recordIDs, installment.ID)
			installmentPeriods = append(installmentPeriods, installment.Period)
		}
	}
	if err := lockOpenPeriodsIn(transaction, companyID, installmentPeriods...); err != nil {
		return nil, err
	}
	// Decided on the locked rows: a close or a first delete may have landed since the read.
	lockedRecords, err := lockRecordsIn(transaction, companyID, recordIDs)
	if err != nil {
		return nil, err
	}
	cancelled := []types.EmployeeRecord{}
	for _, record := range lockedRecords {
		if record.ID == loan.ID {
			if record.Status == types.StatusDeleted {
				return nil, fmt.Errorf("El préstamo %d no existe.", loanID)
			}
			loan = record
		} else if record.Status == types.StatusDeleted {
			continue
		} else if record.Status == types.RecordStatusSettled {
			return nil, fmt.Errorf("Ya se descontaron cuotas del préstamo: no se puede eliminar.")
		}
		cancelled = append(cancelled, record)
	}
	if err := recordPayrollCashIn(transaction, req, loan.CashBankID, finance.CashMovementTypePayrollPayment, loan.ID,
		finance.PCGEStaffLoans, core.FechaUnix(), loan.Amount); err != nil {
		return nil, err
	}
	for i := range cancelled {
		cancelled[i].Status, cancelled[i].UpdatedBy = types.StatusDeleted, req.User.ID
	}
	table := db.TableOf[types.EmployeeRecord]()
	if err := db.UpdateIn(transaction, &cancelled, table.Status, table.UpdatedBy); err != nil {
		return nil, fmt.Errorf("Error al eliminar el préstamo: %v", err)
	}
	return cancelled, nil
}

// lockOpenPeriodsIn locks the runs of the given months and refuses a change to one already closed.
// Under those locks a close of the month waits for the change, or the change waits for the close
// and is refused: a record is never changed under a payslip being frozen. A month with no run yet
// has nothing to freeze; a run created later is calculated from what is committed.
func lockOpenPeriodsIn(transaction *db.Transaction, companyID int32, periods ...int16) error {
	runs, err := loadCompanyRuns(companyID)
	if err != nil {
		return err
	}
	periodRuns := []types.PayrollRun{}
	for _, run := range runs {
		if slices.Contains(periods, run.Period) && run.Status != types.StatusDeleted {
			periodRuns = append(periodRuns, types.PayrollRun{CompanyID: companyID, ID: run.ID})
		}
	}
	if err := db.LockByKeys(transaction, &periodRuns); err != nil {
		return fmt.Errorf("error al leer las planillas: %v", err)
	}
	for _, run := range periodRuns {
		if run.Status >= types.PayrollStatusClosed {
			return fmt.Errorf("La planilla del periodo %d ya está cerrada; reábrala para hacer cambios.", run.Period)
		}
	}
	return nil
}

// recordPeriods is every month a record touches: every month a time off covers, the period of the
// rest.
func recordPeriods(record types.EmployeeRecord) []int16 {
	if !types.IsTimeOffRecord(record.ConceptID) {
		return []int16{record.Period}
	}
	periods := []int16{}
	for period := core.PeriodOfDay(record.FromDate); period <= core.PeriodOfDay(record.ToDate); period = core.NextPeriod(period) {
		periods = append(periods, period)
	}
	return periods
}

// lockEmployeeIn reads an employee under its row lock: the first lock of a record write, so two
// writes of one employee's records (overlapping advances, overlapping time off) apply one after
// the other and each checks the overlaps against what the other committed.
func lockEmployeeIn(transaction *db.Transaction, companyID, employeeID int32) (types.Employee, error) {
	employees := []types.Employee{{CompanyID: companyID, ID: employeeID}}
	if err := db.LockByKeys(transaction, &employees); err != nil {
		return types.Employee{}, fmt.Errorf("error al leer el trabajador: %v", err)
	}
	if len(employees) == 0 || employees[0].Status == types.StatusDeleted {
		return types.Employee{}, fmt.Errorf("el trabajador %d no existe", employeeID)
	}
	return employees[0], nil
}

// lockRecordsIn reads records under their row locks, in key order.
func lockRecordsIn(transaction *db.Transaction, companyID int32, recordIDs []int32) ([]types.EmployeeRecord, error) {
	slices.Sort(recordIDs)
	records := make([]types.EmployeeRecord, len(recordIDs))
	for index, recordID := range recordIDs {
		records[index] = types.EmployeeRecord{CompanyID: companyID, ID: recordID}
	}
	if err := db.LockByKeys(transaction, &records); err != nil {
		return nil, fmt.Errorf("error al leer los registros del personal: %v", err)
	}
	return records, nil
}

// loadCompanyRecords reads every record of the company, cancelled ones included: an SME's records
// fit in a partition read, and the payroll stamp must see a cancellation too.
func loadCompanyRecords(companyID int32) ([]types.EmployeeRecord, error) {
	records := []types.EmployeeRecord{}
	query := db.Query(&records)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los registros del personal: %v", err)
	}
	return records, nil
}

// findRecord finds a live record by ID; a cancelled one counts as missing.
func findRecord(records []types.EmployeeRecord, id int32) (types.EmployeeRecord, bool) {
	for _, record := range records {
		if record.ID == id && record.Status != types.StatusDeleted {
			return record, true
		}
	}
	return types.EmployeeRecord{}, false
}

// GetEmployeeConcepts syncs one employee's fixed concepts (?employee-id=).
func GetEmployeeConcepts(req *core.HandlerArgs) core.HandlerResponse {
	employeeID := req.GetQueryInt("employee-id")
	if employeeID <= 0 {
		return req.MakeErr("Debe enviar employee-id.")
	}
	records := []types.EmployeeConcept{}
	query := db.Query(&records)
	query.Select().CompanyID.Equals(req.User.CompanyID).EmployeeID.Equals(employeeID).
		Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los conceptos fijos:", err)
	}
	return core.MakeResponse(req, &records)
}

// PostEmployeeConcept creates or edits a recurring concept; Status 0 deletes it. Ending it is a
// ToPeriod, so the months already paid keep it.
func PostEmployeeConcept(req *core.HandlerArgs) core.HandlerResponse {
	record := types.EmployeeConcept{}
	if err := json.Unmarshal([]byte(*req.Body), &record); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	if record.ID > 0 {
		stored, err := loadEmployeeConcept(companyID, record.ID)
		if err != nil {
			return req.MakeErr(err)
		}
		record.EmployeeID = stored.EmployeeID
	}
	if record.Status != types.StatusDeleted || record.ID == 0 {
		if _, err := loadEmployee(companyID, record.EmployeeID); err != nil {
			return req.MakeErr(err)
		}
		if _, err := loadConcept(companyID, record.ConceptID); err != nil {
			return req.MakeErr(err)
		}
		if record.Amount <= 0 {
			return req.MakeErr("El monto mensual debe ser mayor a cero.")
		}
		if record.FromPeriod <= 0 || (record.ToPeriod > 0 && record.ToPeriod < record.FromPeriod) {
			return req.MakeErr("Indique el periodo inicial, y un periodo final posterior a él.")
		}
		record.Status = types.StatusActive
	}

	record.CompanyID, record.UpdatedBy = companyID, req.User.ID
	records := []types.EmployeeConcept{record}
	if record.ID == 0 {
		if err := db.Upsert(&records); err != nil {
			return req.MakeErr("Error al guardar el concepto fijo:", err)
		}
		return req.MakeResponse(records[0])
	}
	table := db.TableOf[types.EmployeeConcept]()
	if err := db.Update(&records, table.ConceptID, table.Amount, table.FromPeriod, table.ToPeriod, table.Note,
		table.Status, table.UpdatedBy); err != nil {
		return req.MakeErr("Error al guardar el concepto fijo:", err)
	}
	return req.MakeResponse(records[0])
}

func loadEmployeeConcept(companyID, id int32) (types.EmployeeConcept, error) {
	records := []types.EmployeeConcept{}
	query := db.Query(&records)
	query.CompanyID.Equals(companyID).ID.Equals(id)
	if err := query.Exec(); err != nil {
		return types.EmployeeConcept{}, fmt.Errorf("error al leer el concepto fijo: %v", err)
	}
	if len(records) == 0 || records[0].Status == types.StatusDeleted {
		return types.EmployeeConcept{}, fmt.Errorf("el concepto fijo %d no existe", id)
	}
	return records[0], nil
}

// loadCompanyEmployeeConcepts reads every recurring concept of the company.
func loadCompanyEmployeeConcepts(companyID int32) ([]types.EmployeeConcept, error) {
	records := []types.EmployeeConcept{}
	query := db.Query(&records)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los conceptos fijos: %v", err)
	}
	return records, nil
}
