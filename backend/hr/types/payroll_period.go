package types

import (
	"slices"

	"app/core"
)

// The payslip of the month. A company runs one payroll a month, so one payslip per employee holds
// every concept the month brings: the salary, the gratificación (July, December), the CTS (May,
// November), the settlement of a termination, extraordinary payments and the profit sharing. Each
// concept still has its own engine; CalculatePeriodPayslip decides which apply and joins their lines.

// PeriodPayslipInput is everything one employee's payslip of the month depends on.
type PeriodPayslipInput struct {
	Run               PayrollRun
	Employee          Employee
	Contracts         []Contract
	Records           []EmployeeRecord // active and settled, every period: each engine reads what concerns it
	RecurringConcepts []EmployeeConcept
	Concepts          map[int32]Concept
	// History is every closed or paid payslip of the employee.
	History []Payslip
	Legal   LegalValues
	// ProfitShare and CapExcess are the employee's part of the profit sharing the run distributes.
	ProfitShare int64
	CapExcess   int64
}

// IsOnPayroll: hired by the end of the period and not terminated before it starts.
func IsOnPayroll(employee Employee, period int16) bool {
	if employee.Status == StatusDeleted || employee.HireDate > core.PeriodLastDay(period) {
		return false
	}
	return employee.TerminationDate == 0 || employee.TerminationDate >= core.PeriodFirstDay(period)
}

// IsTerminatedIn: the employee's last day falls in the period, so the month brings a settlement.
func IsTerminatedIn(employee Employee, period int16) bool {
	return employee.TerminationDate > 0 && core.PeriodOfDay(employee.TerminationDate) == period
}

// isOwedSemesterBenefit: employed when the semester ended. One terminated within the semester is
// paid his truncated benefit by the settlement.
func isOwedSemesterBenefit(employee Employee, semesterLastDay int16) bool {
	return employee.HireDate <= semesterLastDay && (employee.TerminationDate == 0 || employee.TerminationDate > semesterLastDay)
}

// PaysGratificacion: the gratificación is paid in July and December.
func PaysGratificacion(employee Employee, period int16) bool {
	month := PeriodMonth(period)
	if month != 7 && month != 12 {
		return false
	}
	_, lastPeriod := GratificacionRunSemester(period)
	return isOwedSemesterBenefit(employee, core.PeriodLastDay(lastPeriod))
}

// PaysCts: the CTS is deposited in May and November.
func PaysCts(employee Employee, period int16) bool {
	month := PeriodMonth(period)
	if month != 5 && month != 11 {
		return false
	}
	_, lastPeriod := CtsRunSemester(period)
	return isOwedSemesterBenefit(employee, core.PeriodLastDay(lastPeriod))
}

// CalculatePeriodPayslip builds the employee's payslip of the month. ok is false when the month
// brings nothing to pay; the payslip then carries only the diagnostics worth reporting (none when
// the employee simply has no concept this month).
//
// The engines run gratificación and CTS first, then the monthly one, then settlement, extraordinary
// payments and profit sharing; each reads the pieces already calculated as closed history, so the
// 5ta, the provisions and the benefit bases see the whole month and never withhold twice.
func CalculatePeriodPayslip(input PeriodPayslipInput) (Payslip, bool) {
	period := input.Run.Period
	history := slices.Clone(input.History)
	diagnostics := []PayslipDiagnostic{}
	pieces := map[periodPiece]Payslip{}
	// keep stores the piece of one engine. A piece that is not ok (nothing to pay, as the part-time
	// worker's CTS) only reports its blocking diagnostics.
	keep := func(kind periodPiece, piece Payslip, ok bool) {
		if !ok {
			for _, diagnostic := range piece.Diagnostics {
				if diagnostic.Blocking {
					diagnostics = append(diagnostics, diagnostic)
				}
			}
			return
		}
		pieces[kind] = piece
		closedPiece := piece
		closedPiece.Status = PayrollStatusClosed
		history = append(history, closedPiece)
	}
	benefitInput := func() BenefitPayslipInput {
		return BenefitPayslipInput{Run: input.Run, Employee: input.Employee, Contracts: input.Contracts,
			History: history, Records: input.Records, Concepts: input.Concepts, Legal: input.Legal,
			ProfitShare: input.ProfitShare, CapExcess: input.CapExcess}
	}

	if PaysGratificacion(input.Employee, period) {
		piece, ok := CalculateGratificacion(benefitInput())
		keep(pieceGratificacion, piece, ok)
	}
	if PaysCts(input.Employee, period) {
		piece, ok := CalculateCts(benefitInput())
		keep(pieceCts, piece, ok)
	}
	if IsOnPayroll(input.Employee, period) {
		piece, ok := CalculateMonthlyPayslip(MonthlyPayslipInput{Period: period, Employee: input.Employee,
			Contracts: input.Contracts, Records: input.Records, RecurringConcepts: input.RecurringConcepts,
			Concepts: input.Concepts, History: history, Legal: input.Legal})
		keep(pieceMonthly, piece, ok)
	}
	if IsTerminatedIn(input.Employee, period) {
		piece, ok := CalculateSettlement(benefitInput())
		keep(pieceSettlement, piece, ok)
	}
	if len(periodRecords(input.Records, period, isRecordType(RecordExtraordinaryPayment))) > 0 {
		piece, ok := CalculateExtraordinaryPayment(benefitInput())
		keep(pieceExtraordinary, piece, ok)
	}
	if input.ProfitShare > 0 {
		piece, ok := CalculateProfitSharing(benefitInput())
		keep(pieceProfitSharing, piece, ok)
	}

	merged := Payslip{EmployeeID: input.Employee.ID, Period: period, DepartmentID: input.Employee.DepartmentID}
	hasParameters := false
	for _, kind := range periodPieceDisplayOrder {
		piece, found := pieces[kind]
		if !found {
			continue
		}
		merged.Lines = append(merged.Lines, piece.Lines...)
		diagnostics = append(diagnostics, piece.Diagnostics...)
		if merged.ContractID == 0 {
			merged.ContractID = piece.ContractID
		}
		if !hasParameters {
			merged.Parameters, hasParameters = piece.Parameters, true
		}
	}
	merged.Diagnostics = diagnostics
	// The worked days are the month's salary days: the benefit pieces carry the days they were
	// calculated on, which the next semester's CTS must not read as this month's.
	merged.WorkedDays = pieces[pieceMonthly].WorkedDays
	if len(merged.Lines) == 0 {
		return merged, false
	}
	merged.SumTotals()
	return merged, true
}

// periodPiece is one engine's share of the month's payslip.
type periodPiece int8

const (
	pieceMonthly periodPiece = iota + 1
	pieceGratificacion
	pieceCts
	pieceSettlement
	pieceExtraordinary
	pieceProfitSharing
)

// periodPieceDisplayOrder is the order the lines appear in the payslip: the salary first.
var periodPieceDisplayOrder = []periodPiece{pieceMonthly, pieceGratificacion, pieceCts, pieceSettlement,
	pieceExtraordinary, pieceProfitSharing}
