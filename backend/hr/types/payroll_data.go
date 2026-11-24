package types

import (
	"fmt"
	"slices"

	"app/db"

	"golang.org/x/sync/errgroup"
)

// PayrollData is everything the payroll reads, loaded once for every month and employee. An SME's
// HR tables fit in a few partition reads, so each table is read whole and split in memory; ForRun
// narrows it to one month. It lives in hr/types because the cash flow forecast (app/finance)
// projects the payroll from the same data.
type PayrollData struct {
	Employees   []Employee
	Departments []Department
	Contracts   []ContractAgreement
	// Records are every employee record of the company, cancelled and pending ones included: the
	// stamp must see a cancellation too. Only active and settled ones reach the engine.
	Records           []EmployeeRecord
	RecurringConcepts []EmployeeConcept
	Concepts          map[int32]Concept
	History           []Payslip // closed and paid payslips of every run
	Legal             LegalValues
}

func LoadPayrollData(companyID int32) (PayrollData, error) {
	data := PayrollData{}
	errGroup := errgroup.Group{}
	errGroup.Go(func() error {
		employees, err := LoadCompanyEmployees(companyID)
		// Only Planilla workers are paid by the payroll: one off payroll (fee receipt, temporary) is
		// left out silently, so no run, payslip, profit share or PLAME/T-Registro file names them.
		data.Employees = slices.DeleteFunc(employees, func(employee Employee) bool {
			return employee.EngagementType != EngagementPayroll
		})
		return err
	})
	errGroup.Go(func() (err error) {
		data.Departments, err = LoadCompanyDepartments(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.Contracts, err = LoadCompanyContracts(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.Records, err = LoadCompanyRecords(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.RecurringConcepts, err = LoadCompanyEmployeeConcepts(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.Concepts, err = LoadCompanyConcepts(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.History, err = LoadClosedPayslips(companyID)
		return err
	})
	errGroup.Go(func() (err error) {
		data.Legal, err = LoadLegalValues(CountryIDPeru)
		return err
	})
	return data, errGroup.Wait()
}

// ForRun narrows the data to one run: the history of every run but itself (a run never reads its
// own payslips). The records stay whole: each engine picks the periods it needs.
func (data PayrollData) ForRun(run PayrollRun) PayrollData {
	runData := data
	runData.History = slices.DeleteFunc(slices.Clone(data.History), func(payslip Payslip) bool { return payslip.RunID == run.ID })
	return runData
}

// InputsStamp is the newest Updated among the records a calculation reads. A payroll calculated with
// an older stamp is stale: some record it read has changed (or arrived) since.
func (data PayrollData) InputsStamp() int32 {
	stamp := data.Legal.NewestUpdated()
	newest := func(updated int32) { stamp = max(stamp, updated) }
	for _, employee := range data.Employees {
		newest(employee.Updated)
	}
	for _, department := range data.Departments {
		newest(department.Updated)
	}
	for _, contract := range data.Contracts {
		newest(contract.Updated)
	}
	for _, record := range data.Records {
		newest(record.Updated)
	}
	for _, recurring := range data.RecurringConcepts {
		newest(recurring.Updated)
	}
	for _, concept := range data.Concepts {
		newest(concept.Updated)
	}
	for _, payslip := range data.History {
		newest(payslip.Updated)
	}
	return stamp
}

// PeriodInputFor slices the run's data down to one employee for the payslip of the month.
func (data PayrollData) PeriodInputFor(employee Employee, run PayrollRun, profitShare, capExcess int64) PeriodPayslipInput {
	return PeriodPayslipInput{Run: run, Employee: employee, Contracts: ContractsOf(data.Contracts, employee.ID),
		Records: data.engineRecordsOf(employee.ID), Concepts: data.Concepts, History: data.HistoryOf(employee.ID),
		RecurringConcepts: filterByEmployee(data.RecurringConcepts, employee.ID, func(recurring EmployeeConcept) int32 { return recurring.EmployeeID }),
		Legal:             data.Legal, ProfitShare: profitShare, CapExcess: capExcess}
}

// engineRecordsOf are the employee's active and settled records: pending time off is not approved
// yet, and a cancelled record never happened.
func (data PayrollData) engineRecordsOf(employeeID int32) []EmployeeRecord {
	records := []EmployeeRecord{}
	for _, record := range data.Records {
		if record.EmployeeID == employeeID && (record.Status == RecordStatusActive || record.Status == RecordStatusSettled) {
			records = append(records, record)
		}
	}
	return records
}

func (data PayrollData) HistoryOf(employeeID int32) []Payslip {
	return filterByEmployee(data.History, employeeID, func(payslip Payslip) int32 { return payslip.EmployeeID })
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

func (data PayrollData) CostFunctionOf(departmentID int32) int8 {
	for _, department := range data.Departments {
		if department.ID == departmentID {
			return department.CostFunction
		}
	}
	return 0
}

func ContractsOf(contracts []ContractAgreement, employeeID int32) []ContractAgreement {
	employeeContracts := []ContractAgreement{}
	for _, contract := range contracts {
		if contract.EmployeeID == employeeID {
			employeeContracts = append(employeeContracts, contract)
		}
	}
	return employeeContracts
}

// LoadCompanyEmployees reads every employee of the company: the deletion guards need the whole set,
// and an SME's staff fits in one partition read.
func LoadCompanyEmployees(companyID int32) ([]Employee, error) {
	employees := []Employee{}
	query := db.Query(&employees)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer el personal: %v", err)
	}
	return employees, nil
}

func LoadCompanyDepartments(companyID int32) ([]Department, error) {
	departments := []Department{}
	query := db.Query(&departments)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las áreas: %v", err)
	}
	return departments, nil
}

func LoadCompanyContracts(companyID int32) ([]ContractAgreement, error) {
	contracts := []ContractAgreement{}
	query := db.Query(&contracts)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los contratos: %v", err)
	}
	return contracts, nil
}

// LoadCompanyRecords reads every record of the company, cancelled ones included: an SME's records
// fit in a partition read, and the payroll stamp must see a cancellation too.
func LoadCompanyRecords(companyID int32) ([]EmployeeRecord, error) {
	records := []EmployeeRecord{}
	query := db.Query(&records)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los registros del personal: %v", err)
	}
	return records, nil
}

// LoadCompanyEmployeeConcepts reads every recurring concept of the company.
func LoadCompanyEmployeeConcepts(companyID int32) ([]EmployeeConcept, error) {
	records := []EmployeeConcept{}
	query := db.Query(&records)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los conceptos fijos: %v", err)
	}
	return records, nil
}

// LoadCompanyConcepts reads every concept of the company, keyed by ID, deleted ones included: an
// old recurring item may still point at one.
func LoadCompanyConcepts(companyID int32) (map[int32]Concept, error) {
	concepts := []Concept{}
	query := db.Query(&concepts)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los conceptos: %v", err)
	}
	conceptsByID := map[int32]Concept{}
	for _, concept := range concepts {
		conceptsByID[concept.ID] = concept
	}
	return conceptsByID, nil
}

// LoadClosedPayslips reads the company's closed and paid payslips: the 5ta, the provisions and the
// benefits read the employee's history from them.
func LoadClosedPayslips(companyID int32) ([]Payslip, error) {
	payslips := []Payslip{}
	query := db.Query(&payslips)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las boletas: %v", err)
	}
	return slices.DeleteFunc(payslips, func(payslip Payslip) bool { return payslip.Status < PayrollStatusClosed }), nil
}

// LoadLegalRecords reads every legal value and AFP rate of a country; both lists are small (a few
// rows per year), so the payroll and the validations read them whole.
func LoadLegalRecords(countryID int32) ([]LegalParameter, []AfpRate, error) {
	parameters, afpRates := []LegalParameter{}, []AfpRate{}
	parameterQuery := db.Query(&parameters)
	parameterQuery.CountryID.Equals(countryID)
	if err := parameterQuery.Exec(); err != nil {
		return nil, nil, fmt.Errorf("error al leer los parámetros laborales: %v", err)
	}
	afpQuery := db.Query(&afpRates)
	afpQuery.CountryID.Equals(countryID)
	if err := afpQuery.Exec(); err != nil {
		return nil, nil, fmt.Errorf("error al leer las tasas de AFP: %v", err)
	}
	return parameters, afpRates, nil
}

// LoadLegalValues builds the dated lookup the payroll engine reads.
func LoadLegalValues(countryID int32) (LegalValues, error) {
	parameters, afpRates, err := LoadLegalRecords(countryID)
	if err != nil {
		return LegalValues{}, err
	}
	return NewLegalValues(parameters, afpRates), nil
}
