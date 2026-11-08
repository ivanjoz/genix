package hr

import (
	"encoding/json"
	"errors"
	"fmt"

	"app/cloud"
	config "app/config/types"
	"app/core"
	"app/db"
	"app/hr/types"
)

// How far the payroll list reaches: from this many months before the company registered in the
// system (the history it may load) to the month after the current one (paid in advance).
const payrollMonthsBeforeRegistration = 12

// skippedEmployee is an employee of the month the payroll could not pay, reported to the page.
type skippedEmployee struct {
	Period     int16
	EmployeeID int32
	Message    string
}

// runCalculation is the outcome of calculating one run.
type runCalculation struct {
	run      types.PayrollRun
	payslips []types.Payslip
	skipped  []skippedEmployee
}

// PostPayrollCalculate recalculates an open run now, whether or not its inputs changed.
func PostPayrollCalculate(req *core.HandlerArgs) core.HandlerResponse {
	if !req.User.HasSubAcceso(accessPayroll, subAccessCalculatePay) {
		return req.MakeErr("No tiene permiso para calcular la planilla.")
	}
	request := runRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	data, err := loadPayrollData(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	// Under the run's lock: a close waits for the recalculation, so it never freezes payslips
	// half rewritten, and a recalculation after a close is refused.
	var calculation runCalculation
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		run, err := lockRunIn(transaction, companyID, request.RunID)
		if err != nil {
			return err
		}
		if run.Status != types.PayrollStatusOpen {
			return errors.New("Solo se puede calcular una planilla abierta.")
		}
		calculation, err = calculateRunIn(transaction, run, data, req.User.ID)
		return err
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(map[string]any{"Run": calculation.run, "Payslips": calculation.payslips, "Skipped": calculation.skipped})
}

// PostPayrollEnsure brings every month of the payroll list up to date, so the user never asks for a
// calculation: it creates the month's run when there is something to pay and recalculates the open
// ones whose inputs changed since they were calculated. Closed and paid runs are never touched.
// The response says the months the list covers and the employees left out of a month.
func PostPayrollEnsure(req *core.HandlerArgs) core.HandlerResponse {
	if !req.User.HasSubAcceso(accessPayroll, subAccessCalculatePay) {
		return req.MakeErr("No tiene permiso para calcular la planilla.")
	}
	companyID := req.User.CompanyID
	registeredPeriod, err := loadCompanyRegistrationPeriod(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	firstPeriod, lastPeriod := payrollListWindow(registeredPeriod, core.PeriodOfDay(core.FechaUnix()))

	runs, err := loadCompanyRuns(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	runByPeriod := map[int16]types.PayrollRun{}
	for _, run := range runs {
		if run.Status != types.StatusDeleted {
			runByPeriod[run.Period] = run
		}
	}
	data, err := loadPayrollData(companyID)
	if err != nil {
		return req.MakeErr(err)
	}

	skipped := []skippedEmployee{}
	for period := firstPeriod; period <= lastPeriod; period = core.NextPeriod(period) {
		// The snapshot only skips the months with nothing to do, so a load that finds every month up
		// to date takes no lock. The rest is decided again inside the month's transaction.
		run, exists := runByPeriod[period]
		if exists && (run.Status != types.PayrollStatusOpen ||
			(run.CalculatedAt > 0 && run.InputsStamp == data.forRun(run).inputsStamp())) {
			continue
		}
		var skippedOfRun []skippedEmployee
		err := db.RunTransaction(func(transaction *db.Transaction) error {
			var ensureError error
			skippedOfRun, ensureError = ensurePeriodRunIn(transaction, req, period, data)
			return ensureError
		})
		if err != nil {
			return req.MakeErr(err)
		}
		skipped = append(skipped, skippedOfRun...)
	}
	return req.MakeResponse(map[string]any{"FirstPeriod": firstPeriod, "LastPeriod": lastPeriod, "Skipped": skipped})
}

// ensurePeriodRunIn creates the month's run when there is something to pay, or recalculates it when
// it is open and its inputs changed. An existing run is locked first; a new one is a plain insert
// that the unique index on Period turns into a rerun of this body when another load created it
// first, and the rerun finds and locks that run.
func ensurePeriodRunIn(transaction *db.Transaction, req *core.HandlerArgs, period int16, data payrollData,
) ([]skippedEmployee, error) {
	companyID := req.User.CompanyID
	periodRuns := []types.PayrollRun{}
	periodQuery := db.Query(&periodRuns)
	periodQuery.Select(periodQuery.ID).CompanyID.Equals(companyID).Period.Equals(period)
	if err := periodQuery.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer la planilla del periodo %d: %v", period, err)
	}

	exists := len(periodRuns) > 0
	run := types.PayrollRun{CompanyID: companyID, Period: period, PayDate: core.PeriodLastDay(period),
		Status: types.PayrollStatusOpen}
	if exists {
		var err error
		if run, err = lockRunIn(transaction, companyID, periodRuns[0].ID); err != nil {
			return nil, err
		}
		if run.Status != types.PayrollStatusOpen {
			return nil, nil
		}
	}
	runData := data.forRun(run)
	if exists && run.CalculatedAt > 0 && run.InputsStamp == runData.inputsStamp() {
		return nil, nil
	}
	payslips, skippedOfRun := calculateRunPayslips(run, runData, req.User.ID)
	// A month nobody is paid in has no run: the list shows it empty.
	if !exists && len(payslips) == 0 && len(skippedOfRun) == 0 {
		return nil, nil
	}
	if !exists {
		run.Created, run.CreatedBy, run.UpdatedBy = core.SUnixTime(), req.User.ID, req.User.ID
		created := []types.PayrollRun{run}
		if err := db.InsertIn(transaction, &created); err != nil {
			return nil, fmt.Errorf("Error al guardar la planilla: %v", err)
		}
		run = created[0]
		for i := range payslips {
			payslips[i].StampRun(run)
		}
	}
	if _, err := saveRunCalculationIn(transaction, run, runData.inputsStamp(), payslips, req.User.ID); err != nil {
		return nil, err
	}
	return skippedOfRun, nil
}

// payrollListWindow is the first and last month of the payroll list.
func payrollListWindow(registeredPeriod, currentPeriod int16) (firstPeriod, lastPeriod int16) {
	return core.AddPeriods(registeredPeriod, -payrollMonthsBeforeRegistration), core.AddPeriods(currentPeriod, 1)
}

// loadCompanyRegistrationPeriod is the month the company registered in the system. The company is
// read here rather than through the config module, which a module body may not import; the mirror
// branch is the one the other modules carry.
func loadCompanyRegistrationPeriod(companyID int32) (int16, error) {
	var company *config.Company
	var err error
	if cloud.IsDataMirrorEnabled() {
		company, err = cloud.GetByID(config.Company{ID: companyID})
	} else {
		companies := []config.Company{}
		query := db.Query(&companies)
		query.ID.Equals(companyID).Limit(1)
		if err = query.Exec(); err == nil && len(companies) > 0 {
			company = &companies[0]
		}
	}
	if err != nil {
		return 0, fmt.Errorf("error al leer la empresa: %w", err)
	}
	if company == nil {
		return 0, errors.New("no se encontró la empresa")
	}
	if company.Created <= 0 {
		return 0, errors.New("la empresa no tiene fecha de registro")
	}
	return core.PeriodOfDay(core.SUnixTimeToUnixDay(company.Created)), nil
}

// calculateRunIn recalculates an open run its caller locked with the whole payroll data, and saves
// the result.
func calculateRunIn(transaction *db.Transaction, run types.PayrollRun, data payrollData, userID int32) (runCalculation, error) {
	runData := data.forRun(run)
	payslips, skipped := calculateRunPayslips(run, runData, userID)
	savedRun, err := saveRunCalculationIn(transaction, run, runData.inputsStamp(), payslips, userID)
	return runCalculation{run: savedRun, payslips: payslips, skipped: skipped}, err
}

// calculateRunPayslips builds the payslip of every employee with something to pay in the run's
// month, and reports the ones that could not be calculated.
func calculateRunPayslips(run types.PayrollRun, data payrollData, userID int32) ([]types.Payslip, []skippedEmployee) {
	profitShares, capExcess := distributeProfitSharing(run, data)
	calculatedAt := core.SUnixTime()
	payslips, skipped := []types.Payslip{}, []skippedEmployee{}
	for _, employee := range data.employees {
		if employee.Status == types.StatusDeleted {
			continue
		}
		payslip, ok := types.CalculatePeriodPayslip(data.periodInputFor(employee, run, profitShares[employee.ID], capExcess[employee.ID]))
		if !ok {
			if len(payslip.Diagnostics) > 0 {
				skipped = append(skipped, skippedEmployee{Period: run.Period, EmployeeID: employee.ID, Message: payslip.Diagnostics[0].Message})
			}
			continue
		}
		payslip.CompanyID, payslip.Status, payslip.CalculatedAt, payslip.UpdatedBy =
			run.CompanyID, types.PayrollStatusOpen, calculatedAt, userID
		payslip.CostFunction = data.costFunctionOf(employee.DepartmentID)
		payslip.StampRun(run)
		payslips = append(payslips, payslip)
	}
	return payslips, skipped
}

// saveRunCalculationIn stores the payslips of a calculation and the totals and stamp on its run.
func saveRunCalculationIn(transaction *db.Transaction, run types.PayrollRun, inputsStamp int32, payslips []types.Payslip, userID int32) (types.PayrollRun, error) {
	if err := savePayslipsIn(transaction, run.CompanyID, run.ID, payslips, userID); err != nil {
		return run, err
	}
	run.CalculatedAt, run.InputsStamp = core.SUnixTime(), inputsStamp
	run.EmployeeCount, run.TotalNetPay, run.TotalEmployerCost = int32(len(payslips)), 0, 0
	for _, payslip := range payslips {
		run.TotalNetPay += payslip.NetPay
		run.TotalEmployerCost += payslip.EmployerCost
	}
	run.UpdatedBy = userID
	runs := []types.PayrollRun{run}
	table := db.TableOf[types.PayrollRun]()
	if err := db.UpdateIn(transaction, &runs, table.CalculatedAt, table.InputsStamp, table.EmployeeCount, table.TotalNetPay,
		table.TotalEmployerCost, table.UpdatedBy); err != nil {
		return run, fmt.Errorf("Error al guardar la planilla: %v", err)
	}
	return run, nil
}

// distributeProfitSharing splits the run's pool over the workers of the fiscal year (the year
// before the run's period), each on his days and remunerations of that year. A run with no net
// income distributes nothing.
func distributeProfitSharing(run types.PayrollRun, data payrollData) (shares, capExcess map[int32]int64) {
	if run.NetIncomeBeforeTax <= 0 {
		return map[int32]int64{}, map[int32]int64{}
	}
	fiscalYear := 2000 + run.Period/100 - 1
	yearLastDay := core.PeriodLastDay((fiscalYear%100)*100 + 12)
	bases := []types.ProfitShareBasis{}
	for _, employee := range data.employees {
		if employee.Status == types.StatusDeleted {
			continue
		}
		// Former workers share too: anyone with a payslip of the fiscal year.
		for _, payslip := range data.historyOf(employee.ID) {
			if payslip.Period/100 == fiscalYear%100 {
				bases = append(bases, types.ProfitShareBasisOf(employee.ID, data.historyOf(employee.ID), fiscalYear,
					contractsOf(data.contracts, employee.ID), yearLastDay))
				break
			}
		}
	}
	pool := run.NetIncomeBeforeTax * int64(run.SharingRate) / types.RateScale
	return types.DistributeProfitSharing(pool, bases)
}
