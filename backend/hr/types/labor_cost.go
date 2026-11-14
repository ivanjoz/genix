package types

import (
	"app/core"
	"app/db"
	"fmt"
	"slices"
)

// The real cost of an hour of a worker, known beforehand (docs/MANUFACTURING_PLAN.md §9.2, §9.5, M18,
// M24, M29): what production values each logged hour at. Production imports hr/types, never the hr body.

// MonthlyHoursDays is how many working days a month counts: a 6-day week (M24).
const MonthlyHoursDays = 26

// EstimatedMonthlyEmployerCost is what a Planilla worker costs the company in a full month with no
// absence, novedad or advance: the payroll engine's EmployerCost (earnings + employer contributions)
// plus its provisions (gratificación, CTS, vacation). The worker is taken as employed the whole period
// under the agreement given, so a mid-month hire or a contract ending soon still estimates a full month.
func EstimatedMonthlyEmployerCost(employee Employee, agreement ContractAgreement, period int16,
	recurringConcepts []EmployeeConcept, concepts map[int32]Concept, legal LegalValues) (int64, error) {
	firstDay := core.PeriodFirstDay(period)
	employee.HireDate, employee.TerminationDate, employee.TerminationReason = min(employee.HireDate, firstDay), 0, 0
	agreement.ValidFrom, agreement.StartDate, agreement.EndDate, agreement.Status = firstDay, firstDay, 0, StatusActive
	payslip, ok := CalculateMonthlyPayslip(MonthlyPayslipInput{
		Period: period, Employee: employee, Contracts: []ContractAgreement{agreement},
		RecurringConcepts: recurringConcepts, Concepts: concepts, Legal: legal,
	})
	for _, diagnostic := range payslip.Diagnostics {
		if diagnostic.Blocking {
			ok = false
		}
	}
	if !ok {
		message := "no se pudo estimar su costo mensual"
		if len(payslip.Diagnostics) > 0 {
			message = payslip.Diagnostics[0].Message
		}
		return 0, core.Err(fmt.Sprintf("Trabajador %v: %s", employee.ID, message))
	}
	return int64(payslip.EmployerCost) + sumLines(payslip.Lines, ConceptKindProvision), nil
}

// ContractAgreementHourlyCost is the cost of one hour of the worker under the agreement, PEN cents:
//
//	Planilla               estimated monthly employer cost ÷ (DailyHours × 26)
//	monthly, off payroll   PayAmount ÷ (DailyHours × 26)
//	weekly                 PayAmount ÷ (DailyHours × 6)
//	hourly                 PayAmount
//
// estimatedMonthlyEmployerCost is only read for a Planilla worker.
func ContractAgreementHourlyCost(employee Employee, agreement ContractAgreement, estimatedMonthlyEmployerCost int64) int32 {
	dailyHours := int64(max(agreement.DailyHours, 1))
	switch {
	case employee.EngagementType == EngagementPayroll:
		return int32((estimatedMonthlyEmployerCost + dailyHours*MonthlyHoursDays/2) / (dailyHours * MonthlyHoursDays))
	case agreement.PayBasis == PayBasisHourly:
		return agreement.PayAmount
	case agreement.PayBasis == PayBasisWeekly:
		return int32((int64(agreement.PayAmount) + dailyHours*3) / (dailyHours * 6))
	default:
		return int32((int64(agreement.PayAmount) + dailyHours*MonthlyHoursDays/2) / (dailyHours * MonthlyHoursDays))
	}
}

// WorkerHourCost is what one hour of a worker costs on a day, and the most a month of their labor may
// book (MonthlyCap; −1 = no cap, a worker off payroll).
type WorkerHourCost struct {
	Employee    Employee
	CostPerHour int32
	MonthlyCap  int64
	// HasAgreement is false for a worker off payroll with no agreement in force: CostPerHour is then 0
	// and the caller falls back to the labor type's rate (M29).
	HasAgreement bool
}

// LoadWorkerHourCost reads what LaborEntry costing needs of one of the company's workers and costs an
// hour of them on day. A Planilla worker needs a contract in force; their cap is their estimated
// monthly employer cost of the day's period.
func LoadWorkerHourCost(companyID int32, employeeID int32, day int16) (WorkerHourCost, error) {
	employees := []Employee{}
	employeeQuery := db.Query(&employees)
	employeeQuery.CompanyID.Equals(companyID).ID.Equals(employeeID)
	if err := employeeQuery.Exec(); err != nil {
		return WorkerHourCost{}, core.Err("Error al obtener el trabajador:", err)
	}
	if len(employees) == 0 || employees[0].Status != StatusActive {
		return WorkerHourCost{}, core.Err(fmt.Sprintf("El trabajador %v no existe o no está activo.", employeeID))
	}
	employee := employees[0]
	if employee.HireDate > day || (employee.TerminationDate > 0 && employee.TerminationDate < day) {
		return WorkerHourCost{}, core.Err(fmt.Sprintf(`"%s %s" no trabajaba en la empresa ese día.`, employee.FirstName, employee.LastName))
	}
	agreements := []ContractAgreement{}
	agreementQuery := db.Query(&agreements)
	agreementQuery.CompanyID.Equals(companyID).EmployeeID.Equals(employeeID)
	if err := agreementQuery.Exec(); err != nil {
		return WorkerHourCost{}, core.Err("Error al obtener los contratos del trabajador:", err)
	}
	agreement, hasAgreement := FindContractInForce(agreements, day)
	hourCost := WorkerHourCost{Employee: employee, MonthlyCap: -1, HasAgreement: hasAgreement}
	if employee.EngagementType != EngagementPayroll {
		if hasAgreement {
			hourCost.CostPerHour = ContractAgreementHourlyCost(employee, agreement, 0)
		}
		return hourCost, nil
	}
	if !hasAgreement {
		return WorkerHourCost{}, core.Err(fmt.Sprintf(`"%s %s" no tiene un contrato vigente ese día.`, employee.FirstName, employee.LastName))
	}

	recurringConcepts := []EmployeeConcept{}
	recurringQuery := db.Query(&recurringConcepts)
	recurringQuery.CompanyID.Equals(companyID)
	concepts := []Concept{}
	conceptQuery := db.Query(&concepts)
	conceptQuery.CompanyID.Equals(companyID)
	parameters, afpRates := []LegalParameter{}, []AfpRate{}
	parameterQuery := db.Query(&parameters)
	parameterQuery.CountryID.Equals(CountryIDPeru)
	afpQuery := db.Query(&afpRates)
	afpQuery.CountryID.Equals(CountryIDPeru)
	for _, err := range []error{recurringQuery.Exec(), conceptQuery.Exec(), parameterQuery.Exec(), afpQuery.Exec()} {
		if err != nil {
			return WorkerHourCost{}, core.Err("Error al obtener los datos de planilla del trabajador:", err)
		}
	}
	employeeConcepts := slices.DeleteFunc(recurringConcepts, func(recurring EmployeeConcept) bool { return recurring.EmployeeID != employeeID })
	conceptsByID := map[int32]Concept{}
	for _, concept := range concepts {
		conceptsByID[concept.ID] = concept
	}
	monthlyCost, err := EstimatedMonthlyEmployerCost(employee, agreement, core.PeriodOfDay(day), employeeConcepts, conceptsByID,
		NewLegalValues(parameters, afpRates))
	if err != nil {
		return WorkerHourCost{}, err
	}
	hourCost.CostPerHour = ContractAgreementHourlyCost(employee, agreement, monthlyCost)
	hourCost.MonthlyCap = monthlyCost
	return hourCost, nil
}
