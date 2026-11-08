package hr

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"app/core"
	"app/db"
	finance "app/finance/types"
	"app/hr/types"
)

// Payroll access (backend/access.toml id 41) and the sub-accesses that gate the run transitions.
const (
	accessPayroll         = int32(41)
	subAccessCalculatePay = int32(2)
	subAccessClosePayroll = int32(3)
	subAccessRegisterPaid = int32(4)
)

type runRequest struct {
	RunID int32
}

// Profit sharing rates by sector (DL 892 art. 2), basis points.
// TODO(accountant): verify the 5 % "other activities" rate against the decree text.
var profitSharingRates = []int32{500, 800, 1000}

// PostPayrollRun edits an open run: its pay date, note and the profit sharing of the year. Runs are
// created by PostPayrollEnsure, one per month. A profit sharing change recalculates the run.
func PostPayrollRun(req *core.HandlerArgs) core.HandlerResponse {
	run := types.PayrollRun{}
	if err := json.Unmarshal([]byte(*req.Body), &run); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	data, err := loadPayrollData(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	var calculation runCalculation
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		stored, err := lockRunIn(transaction, companyID, run.ID)
		if err != nil {
			return err
		}
		if stored.Status != types.PayrollStatusOpen {
			return fmt.Errorf("Solo se puede modificar una planilla abierta.")
		}
		stored.PayDate, stored.Note, stored.UpdatedBy = run.PayDate, run.Note, req.User.ID
		stored.NetIncomeBeforeTax, stored.SharingRate = run.NetIncomeBeforeTax, run.SharingRate
		if err := validateProfitSharing(stored); err != nil {
			return err
		}
		table := db.TableOf[types.PayrollRun]()
		if err := db.UpdateIn(transaction, &[]types.PayrollRun{stored}, table.PayDate, table.Note,
			table.NetIncomeBeforeTax, table.SharingRate, table.UpdatedBy); err != nil {
			return fmt.Errorf("Error al guardar la planilla: %v", err)
		}
		calculation, err = calculateRunIn(transaction, stored, data, req.User.ID)
		return err
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(map[string]any{"Run": calculation.run, "Payslips": calculation.payslips, "Skipped": calculation.skipped})
}

// validateProfitSharing checks the inputs of the distribution, taken from the annual return. A run
// with no net income distributes nothing.
func validateProfitSharing(run types.PayrollRun) error {
	if run.NetIncomeBeforeTax < 0 {
		return fmt.Errorf("La renta neta del ejercicio no puede ser negativa.")
	}
	if run.NetIncomeBeforeTax > 0 && !slices.Contains(profitSharingRates, run.SharingRate) {
		return fmt.Errorf("El porcentaje de participación debe ser 5 %%, 8 %% o 10 %% según la actividad de la empresa.")
	}
	return nil
}

// savePayslipsIn upserts the run's payslips by employee: an employee keeps the ID of his payslip
// across recalculations, and one no longer on the run is deleted. Its caller holds the run's lock,
// so the stored payslips it reads are not being written meanwhile.
func savePayslipsIn(transaction *db.Transaction, companyID, runID int32, payslips []types.Payslip, userID int32) error {
	stored, err := loadRunPayslips(companyID, runID)
	if err != nil {
		return err
	}
	storedByEmployee := map[int32]types.Payslip{}
	for _, payslip := range stored {
		if payslip.Status != types.StatusDeleted {
			storedByEmployee[payslip.EmployeeID] = payslip
		}
	}
	newPayslips, changedPayslips := []types.Payslip{}, []types.Payslip{}
	for i := range payslips {
		if previous, found := storedByEmployee[payslips[i].EmployeeID]; found {
			payslips[i].ID = previous.ID
			changedPayslips = append(changedPayslips, payslips[i])
			delete(storedByEmployee, payslips[i].EmployeeID)
		} else {
			newPayslips = append(newPayslips, payslips[i])
		}
	}
	for _, removed := range storedByEmployee {
		removed.Status, removed.UpdatedBy = types.StatusDeleted, userID
		changedPayslips = append(changedPayslips, removed)
	}
	if len(newPayslips) > 0 {
		if err := db.InsertIn(transaction, &newPayslips); err != nil {
			return fmt.Errorf("Error al guardar las boletas: %v", err)
		}
		// Hand the new IDs back to the caller's slice.
		for _, inserted := range newPayslips {
			for i := range payslips {
				if payslips[i].EmployeeID == inserted.EmployeeID {
					payslips[i].ID = inserted.ID
				}
			}
		}
	}
	if len(changedPayslips) > 0 {
		table := db.TableOf[types.Payslip]()
		if err := db.UpdateIn(transaction, &changedPayslips, table.ContractID, table.DepartmentID, table.CostFunction, table.WorkedDays,
			table.Lines, table.TotalEarnings, table.TotalDeductions, table.NetPay, table.EmployerCost, table.Parameters,
			table.Diagnostics, table.CalculatedAt, table.Status, table.UpdatedBy); err != nil {
			return fmt.Errorf("Error al guardar las boletas: %v", err)
		}
	}
	return nil
}

// PostPayrollClose freezes a calculated run: its payslips become the accounting fact, the period's
// novedades are consumed and the advances it netted are settled.
func PostPayrollClose(req *core.HandlerArgs) core.HandlerResponse {
	if !req.User.HasSubAcceso(accessPayroll, subAccessClosePayroll) {
		return req.MakeErr("No tiene permiso para cerrar la planilla.")
	}
	request := runRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	var run types.PayrollRun
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		if run, err = lockRunIn(transaction, companyID, request.RunID); err != nil {
			return err
		}
		if run.Status != types.PayrollStatusOpen || run.CalculatedAt == 0 {
			return fmt.Errorf("Solo se puede cerrar una planilla abierta y calculada.")
		}
		payslips, err := loadLiveRunPayslips(companyID, run.ID)
		if err != nil {
			return err
		}
		for _, payslip := range payslips {
			for _, diagnostic := range payslip.Diagnostics {
				if diagnostic.Blocking {
					return fmt.Errorf("Hay boletas con observaciones que impiden el cierre (%s: %s).",
						diagnostic.Code, diagnostic.Message)
				}
			}
		}

		// Anything that changed after the calculation (an input, an advance, a contract, a legal
		// value) is not in the payslips yet. A record write of this month holds this run's lock, so
		// none lands between this check and the close.
		data, err := loadPayrollData(companyID)
		if err != nil {
			return err
		}
		runData := data.forRun(run)
		if runData.inputsStamp() != run.InputsStamp {
			return fmt.Errorf("Hay datos que cambiaron después del cálculo: vuelva a calcular la planilla.")
		}
		if err := setRunStatusIn(transaction, &run, payslips, types.PayrollStatusClosed, req.User.ID); err != nil {
			return err
		}
		return settleRunRecordsIn(transaction, run, payslips, runData.records, true, req.User.ID)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(run)
}

// loadLiveRunPayslips reads a run's payslips, without the ones a recalculation deleted.
func loadLiveRunPayslips(companyID, runID int32) ([]types.Payslip, error) {
	payslips, err := loadRunPayslips(companyID, runID)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(payslips, func(payslip types.Payslip) bool { return payslip.Status == types.StatusDeleted }), nil
}

// PostPayrollReopen takes a closed (not paid) run back to open: its payslips leave the accounting,
// and the records it settled (novedades, advances, installments) are active again.
func PostPayrollReopen(req *core.HandlerArgs) core.HandlerResponse {
	if !req.User.HasSubAcceso(accessPayroll, subAccessClosePayroll) {
		return req.MakeErr("No tiene permiso para reabrir la planilla.")
	}
	request := runRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	var run types.PayrollRun
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		if run, err = lockRunIn(transaction, companyID, request.RunID); err != nil {
			return err
		}
		if run.Status != types.PayrollStatusClosed {
			return fmt.Errorf("Solo se puede reabrir una planilla cerrada que aún no fue pagada.")
		}
		payslips, err := loadLiveRunPayslips(companyID, run.ID)
		if err != nil {
			return err
		}
		companyRecords, err := loadCompanyRecords(companyID)
		if err != nil {
			return err
		}
		if err := setRunStatusIn(transaction, &run, payslips, types.PayrollStatusOpen, req.User.ID); err != nil {
			return err
		}
		return settleRunRecordsIn(transaction, run, payslips, companyRecords, false, req.User.ID)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(run)
}

// PostPayrollPaid pays a closed run's net pay from a cash or bank account (Dr 4111 · Cr 101/1041,
// H7) on its pay date, and marks the run and its payslips paid.
func PostPayrollPaid(req *core.HandlerArgs) core.HandlerResponse {
	if !req.User.HasSubAcceso(accessPayroll, subAccessRegisterPaid) {
		return req.MakeErr("No tiene permiso para registrar el pago de la planilla.")
	}
	request := struct {
		RunID      int32
		CashBankID int32
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if request.CashBankID <= 0 {
		return req.MakeErr("Seleccione la caja o cuenta bancaria de la que se paga la planilla.")
	}
	companyID := req.User.CompanyID
	// Under the run's lock a second payment waits, then finds it Paid: the net pay leaves once.
	var run types.PayrollRun
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		if run, err = lockRunIn(transaction, companyID, request.RunID); err != nil {
			return err
		}
		if run.Status != types.PayrollStatusClosed {
			return fmt.Errorf("Solo se puede pagar una planilla cerrada.")
		}
		payslips, err := loadLiveRunPayslips(companyID, run.ID)
		if err != nil {
			return err
		}
		var netPay int32
		for _, payslip := range payslips {
			netPay += payslip.NetPay
		}
		if netPay > 0 {
			if err := recordPayrollCashIn(transaction, req, request.CashBankID, finance.CashMovementTypePayrollPayment, run.ID,
				finance.PCGESalariesPayable, run.PayDate, -netPay); err != nil {
				return err
			}
		}
		run.CashBankID = request.CashBankID
		return setRunStatusIn(transaction, &run, payslips, types.PayrollStatusPaid, req.User.ID)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(run)
}

// setRunStatusIn moves a run and its payslips to a status together, stamping the transition.
func setRunStatusIn(transaction *db.Transaction, run *types.PayrollRun, payslips []types.Payslip, status int8, userID int32) error {
	now := core.SUnixTime()
	run.Status, run.UpdatedBy = status, userID
	switch status {
	case types.PayrollStatusClosed:
		run.ClosedAt, run.ClosedBy = now, userID
	case types.PayrollStatusOpen:
		run.ClosedAt, run.ClosedBy = 0, 0
	case types.PayrollStatusPaid:
		run.PaidAt = now
	}
	for i := range payslips {
		payslips[i].Status, payslips[i].UpdatedBy = status, userID
	}
	if len(payslips) > 0 {
		payslipTable := db.TableOf[types.Payslip]()
		if err := db.UpdateIn(transaction, &payslips, payslipTable.Status, payslipTable.UpdatedBy); err != nil {
			return fmt.Errorf("Error al actualizar las boletas: %v", err)
		}
	}
	runs := []types.PayrollRun{*run}
	runTable := db.TableOf[types.PayrollRun]()
	if err := db.UpdateIn(transaction, &runs, runTable.Status, runTable.ClosedAt, runTable.ClosedBy, runTable.PaidAt,
		runTable.CashBankID, runTable.UpdatedBy); err != nil {
		return fmt.Errorf("Error al actualizar la planilla: %v", err)
	}
	return nil
}

// settleRunRecords marks what a closing run consumed as settled by its payslip: the period's
// novedades of its employees, and the advances and loan installments its lines net (by SourceID: a
// settlement recovers installments of later months too). Time off stays active: it is a fact of
// the calendar, not something the payslip pays off. Reopening gives every record settled by the
// run's payslips back to active.
//
// The records to change are picked from companyRecords, then locked and picked again from their
// locked rows: a loan deleted or a record edited since the read is never settled or revived.
func settleRunRecordsIn(transaction *db.Transaction, run types.PayrollRun, payslips []types.Payslip,
	companyRecords []types.EmployeeRecord, closing bool, userID int32,
) error {
	payslipByEmployee, payslipByRecord := map[int32]int32{}, map[int32]int32{}
	for _, payslip := range payslips {
		payslipByEmployee[payslip.EmployeeID] = payslip.ID
		for _, line := range payslip.Lines {
			if line.SourceID > 0 {
				payslipByRecord[line.SourceID] = payslip.ID
			}
		}
	}
	runPayslipIDs := slices.Collect(maps.Values(payslipByEmployee))

	// settled is the record as this close or reopen leaves it, false when it does not change it.
	settled := func(record types.EmployeeRecord) (types.EmployeeRecord, bool) {
		switch {
		case closing && record.Status == types.RecordStatusActive:
			payslipID, netted := payslipByRecord[record.ID]
			if !netted && types.IsEntryRecord(record.ConceptID) && record.Period == run.Period {
				payslipID, netted = payslipByEmployee[record.EmployeeID]
			}
			if !netted {
				return record, false
			}
			record.Status, record.PayslipID = types.RecordStatusSettled, payslipID
		case !closing && record.Status == types.RecordStatusSettled && slices.Contains(runPayslipIDs, record.PayslipID):
			record.Status, record.PayslipID = types.RecordStatusActive, 0
		default:
			return record, false
		}
		record.UpdatedBy = userID
		return record, true
	}

	candidateIDs := []int32{}
	for _, record := range companyRecords {
		if _, changes := settled(record); changes {
			candidateIDs = append(candidateIDs, record.ID)
		}
	}
	if len(candidateIDs) == 0 {
		return nil
	}
	lockedRecords, err := lockRecordsIn(transaction, run.CompanyID, candidateIDs)
	if err != nil {
		return err
	}
	changedRecords := []types.EmployeeRecord{}
	for _, record := range lockedRecords {
		if changed, changes := settled(record); changes {
			changedRecords = append(changedRecords, changed)
		}
	}
	if len(changedRecords) == 0 {
		return nil
	}
	table := db.TableOf[types.EmployeeRecord]()
	if err := db.UpdateIn(transaction, &changedRecords, table.Status, table.PayslipID, table.UpdatedBy); err != nil {
		return fmt.Errorf("Error al actualizar los registros del personal: %v", err)
	}
	return nil
}

// GetPayslips syncs the payslips of one period (?period=YYMM).
func GetPayslips(req *core.HandlerArgs) core.HandlerResponse {
	period, err := queryPeriod(req)
	if err != nil {
		return req.MakeErr(err)
	}
	payslips := []types.Payslip{}
	query := db.Query(&payslips)
	query.Select().CompanyID.Equals(req.User.CompanyID).Period.Equals(period).
		Delta(req.GetDeltaCursor(), types.PayrollStatusOpen, types.PayrollStatusClosed, types.PayrollStatusPaid)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las boletas:", err)
	}
	return core.MakeResponse(req, &payslips)
}

// GetEmployeePayslips syncs one worker's closed and paid payslips (?employee-id=).
func GetEmployeePayslips(req *core.HandlerArgs) core.HandlerResponse {
	employeeID := req.GetQueryInt("employee-id")
	if employeeID <= 0 {
		return req.MakeErr("Debe enviar employee-id.")
	}
	payslips := []types.Payslip{}
	query := db.Query(&payslips)
	query.Select().CompanyID.Equals(req.User.CompanyID).EmployeeID.Equals(employeeID).
		Delta(req.GetDeltaCursor(), types.PayrollStatusClosed, types.PayrollStatusPaid)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las boletas:", err)
	}
	return core.MakeResponse(req, &payslips)
}

// GetPayslipsByIDs resolves payslips referenced from other pages (accounting, cash movements).
func GetPayslipsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[types.Payslip](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener las boletas.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}
