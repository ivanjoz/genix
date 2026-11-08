package hr

import (
	"fmt"
	"slices"

	"app/db"
	"app/hr/types"

	"golang.org/x/sync/errgroup"
)

// payrollData is everything the payroll reads, loaded once for every month and employee. An SME's
// HR tables fit in a few partition reads, so each table is read whole and split in memory; forRun
// narrows it to one month.
type payrollData struct {
	employees   []types.Employee
	departments []types.Department
	contracts   []types.Contract
	// records are every employee record of the company, cancelled and pending ones included: the
	// stamp must see a cancellation too. Only active and settled ones reach the engine.
	records           []types.EmployeeRecord
	recurringConcepts []types.EmployeeConcept
	concepts          map[int32]types.Concept
	history           []types.Payslip // closed and paid payslips of every run
	legal             types.LegalValues
}

func loadPayrollData(companyID int32) (payrollData, error) {
	data := payrollData{}
	errGroup := errgroup.Group{}
	errGroup.Go(func() (err error) {
		data.employees, err = loadCompanyEmployees(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.departments, err = loadCompanyDepartments(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.contracts, err = loadCompanyContracts(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.records, err = loadCompanyRecords(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.recurringConcepts, err = loadCompanyEmployeeConcepts(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.concepts, err = loadCompanyConcepts(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.history, err = loadClosedPayslips(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.legal, err = loadLegalValues(types.CountryIDPeru)
		return err
	})
	return data, errGroup.Wait()
}

// forRun narrows the data to one run: the history of every run but itself (a run never reads its
// own payslips). The records stay whole: each engine picks the periods it needs.
func (data payrollData) forRun(run types.PayrollRun) payrollData {
	runData := data
	runData.history = slices.DeleteFunc(slices.Clone(data.history), func(payslip types.Payslip) bool { return payslip.RunID == run.ID })
	return runData
}

// inputsStamp is the newest Updated among the records a calculation reads. A payroll calculated with
// an older stamp is stale: some record it read has changed (or arrived) since.
func (data payrollData) inputsStamp() int32 {
	stamp := data.legal.NewestUpdated()
	newest := func(updated int32) { stamp = max(stamp, updated) }
	for _, employee := range data.employees {
		newest(employee.Updated)
	}
	for _, department := range data.departments {
		newest(department.Updated)
	}
	for _, contract := range data.contracts {
		newest(contract.Updated)
	}
	for _, record := range data.records {
		newest(record.Updated)
	}
	for _, recurring := range data.recurringConcepts {
		newest(recurring.Updated)
	}
	for _, concept := range data.concepts {
		newest(concept.Updated)
	}
	for _, payslip := range data.history {
		newest(payslip.Updated)
	}
	return stamp
}

// periodInputFor slices the run's data down to one employee for the payslip of the month.
func (data payrollData) periodInputFor(employee types.Employee, run types.PayrollRun, profitShare, capExcess int64) types.PeriodPayslipInput {
	return types.PeriodPayslipInput{Run: run, Employee: employee, Contracts: contractsOf(data.contracts, employee.ID),
		Records: data.engineRecordsOf(employee.ID), Concepts: data.concepts, History: data.historyOf(employee.ID),
		RecurringConcepts: filterByEmployee(data.recurringConcepts, employee.ID, func(recurring types.EmployeeConcept) int32 { return recurring.EmployeeID }),
		Legal:             data.legal, ProfitShare: profitShare, CapExcess: capExcess}
}

// engineRecordsOf are the employee's active and settled records: pending time off is not approved
// yet, and a cancelled record never happened.
func (data payrollData) engineRecordsOf(employeeID int32) []types.EmployeeRecord {
	records := []types.EmployeeRecord{}
	for _, record := range data.records {
		if record.EmployeeID == employeeID && (record.Status == types.RecordStatusActive || record.Status == types.RecordStatusSettled) {
			records = append(records, record)
		}
	}
	return records
}

func (data payrollData) historyOf(employeeID int32) []types.Payslip {
	return filterByEmployee(data.history, employeeID, func(payslip types.Payslip) int32 { return payslip.EmployeeID })
}

func filterByEmployee[T any](records []T, employeeID int32, employeeOf func(T) int32) []T {
	filtered := []T{}
	for _, record := range records {
		if employeeOf(record) == employeeID {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func (data payrollData) costFunctionOf(departmentID int32) int8 {
	for _, department := range data.departments {
		if department.ID == departmentID {
			return department.CostFunction
		}
	}
	return 0
}

// loadClosedPayslips reads the company's closed and paid payslips: the 5ta, the provisions and the
// benefits read the employee's history from them.
func loadClosedPayslips(companyID int32) ([]types.Payslip, error) {
	payslips := []types.Payslip{}
	query := db.Query(&payslips)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las boletas: %v", err)
	}
	return slices.DeleteFunc(payslips, func(payslip types.Payslip) bool { return payslip.Status < types.PayrollStatusClosed }), nil
}

func contractsOf(contracts []types.Contract, employeeID int32) []types.Contract {
	employeeContracts := []types.Contract{}
	for _, contract := range contracts {
		if contract.EmployeeID == employeeID {
			employeeContracts = append(employeeContracts, contract)
		}
	}
	return employeeContracts
}

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

func loadCompanyContracts(companyID int32) ([]types.Contract, error) {
	contracts := []types.Contract{}
	query := db.Query(&contracts)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los contratos: %v", err)
	}
	return contracts, nil
}

func loadCompanyDepartments(companyID int32) ([]types.Department, error) {
	departments := []types.Department{}
	query := db.Query(&departments)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las áreas: %v", err)
	}
	return departments, nil
}
