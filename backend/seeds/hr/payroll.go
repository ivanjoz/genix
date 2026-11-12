package hr

import (
	"app/core"
	"app/db"
	hrModule "app/hr"
	"app/hr/types"
	"app/seeds/runner"
	"fmt"
	mrand "math/rand"
	"slices"
	"time"
)

// hrSeedConcepts are the company concepts the seed uses: a regular productivity bonus, movilidad
// tied to attendance (not remunerative), a one-off bonus and a judicial withholding owed to a third
// party (419). They are looked up by name, so a rerun reuses them.
var hrSeedConcepts = []types.Concept{
	{Name: "Bono de productividad", Kind: types.ConceptKindEarning, PlameCode: "0306", IsRemunerative: true,
		AffectsEssalud: true, AffectsPension: true, IncomeTax: types.IncomeTaxOrdinary, ExpenseAccountCode: 622},
	{Name: "Movilidad supeditada a asistencia", Kind: types.ConceptKindEarning, PlameCode: "0909",
		IncomeTax: types.IncomeTaxNone, ExpenseAccountCode: 625},
	{Name: "Bono extraordinario", Kind: types.ConceptKindEarning, PlameCode: "0304",
		IncomeTax: types.IncomeTaxExtraordinary, ExpenseAccountCode: 622},
	{Name: "Retención judicial", Kind: types.ConceptKindDeduction, PlameCode: "0703", PayableAccountCode: 419},
}

var Module = runner.Module{Name: "hr", Setup: Setup}

// Setup writes what the company has before the first day (staff, concepts, recurring items, a
// loan) and returns the payroll day step. The legal parameters come from base.
func Setup(run *runner.Run) ([]runner.DayStep, error) {
	firstDay := run.FirstDay()
	payroll := &demoPayroll{seedRun: run, seed: run.Config.Seed, lastDay: core.TimeToFechaUnix(run.RealNow),
		at: core.Now()}
	if err := payroll.setup(firstDay); err != nil {
		return nil, core.Err("preparación de RRHH:", err)
	}
	return []runner.DayStep{{Name: "payroll", Order: runner.OrderPayroll, Run: payroll.runPayrollStep}}, nil
}

// demoPayroll writes the HR side of each simulated day; seedRun supplies the bank account and the
// cash top-ups.
type demoPayroll struct {
	seedRun   *runner.Run
	seed      int64
	lastDay   int16
	at        time.Time // the frozen moment of the day's HR writes
	conceptID map[string]int32
}

// runPayrollStep is the day step. It runs late in the day, after the tills' takings reached the
// bank, so the payments find the day's deposits.
func (payroll *demoPayroll) runPayrollStep(dayStart time.Time) error {
	day := core.TimeToFechaUnix(dayStart)
	period := core.PeriodOfDay(day)
	payroll.at = dayStart.Add(21*time.Hour + 15*time.Minute)
	core.SetHistoricalUnix(payroll.at.Unix())
	random := mrand.New(mrand.NewSource(payroll.seed + int64(day)))

	var err error
	switch monthDay := dayStart.Day(); {
	case monthDay == 1:
		err = payroll.bookVacations(day, random)
	case monthDay%7 == 0:
		err = payroll.payWeeklyAdvances(day)
	case monthDay == 20:
		err = payroll.registerResignation(day, random)
	case monthDay == 25:
		err = payroll.registerMonthRecords(day, random)
	}
	if err != nil {
		return err
	}

	if runner.IsMonthDay(dayStart, 31) {
		return payroll.settleRun(period, true)
	} else if day == payroll.lastDay {
		return payroll.settleRun(period, false)
	}
	return nil
}

// setup writes what the company has before the first day: the staff (unless the company already
// has one), the concepts, the recurring items and a loan.
func (payroll *demoPayroll) setup(day int16) error {
	employees, err := loadSeedEmployees()
	if err != nil {
		return err
	}
	if len(employees) == 0 {
		if err := seedHRStaff(payroll.seed, day); err != nil {
			return err
		}
	}
	if err := payroll.ensureConcepts(); err != nil {
		return err
	}

	// The loan is written last, so a company that has one already went through this block.
	records, err := loadSeedRecords()
	if err != nil {
		return err
	}
	if slices.ContainsFunc(records, func(record types.EmployeeRecord) bool { return record.ConceptID == types.LegalLoan }) {
		return nil
	}
	activeEmployees, err := activeEmployeesOn(day)
	if err != nil {
		return err
	}
	period := core.PeriodOfDay(day)
	// Movilidad for one in three, a judicial withholding for the fifth worker.
	for i, employee := range activeEmployees {
		var recurring *types.EmployeeConcept
		switch {
		case i%3 == 0:
			recurring = &types.EmployeeConcept{ConceptID: payroll.conceptID["Movilidad supeditada a asistencia"], Amount: 15000}
		case i == 4:
			recurring = &types.EmployeeConcept{ConceptID: payroll.conceptID["Retención judicial"], Amount: 25000, Note: "Pensión de alimentos"}
		}
		if recurring == nil {
			continue
		}
		recurring.EmployeeID, recurring.FromPeriod = employee.ID, period
		if err := callHRHandler("POST.hr-employee-concepts", hrModule.PostEmployeeConcept, nil, recurring, nil); err != nil {
			return err
		}
	}
	if len(activeEmployees) < 2 {
		return nil
	}
	// A personal loan recovered in three installments.
	loan := types.EmployeeRecord{EmployeeID: activeEmployees[1].ID, ConceptID: types.LegalLoan, Date: day, Amount: 90000,
		Quantity: 3, Period: period, Note: "Préstamo personal", CashBankID: payroll.seedRun.BankCashBankID}
	if err := payroll.seedRun.EnsureCashBalance(loan.CashBankID, loan.Amount, payroll.at); err != nil {
		return err
	}
	return callHRHandler("POST.hr-loans", hrModule.PostLoan, nil, loan, nil)
}

// ensureConcepts creates the seed concepts the company lacks and keeps their IDs by name.
func (payroll *demoPayroll) ensureConcepts() error {
	concepts := []types.Concept{}
	query := db.Query(&concepts)
	query.CompanyID.Equals(runner.CompanyID)
	if err := query.Exec(); err != nil {
		return err
	}
	payroll.conceptID = map[string]int32{}
	for _, concept := range concepts {
		if concept.Status != types.StatusDeleted {
			payroll.conceptID[concept.Name] = concept.ID
		}
	}
	for _, conceptSeed := range hrSeedConcepts {
		if payroll.conceptID[conceptSeed.Name] > 0 {
			continue
		}
		created := types.Concept{}
		if err := callHRHandler("POST.hr-concepts", hrModule.PostConcept, nil, conceptSeed, &created); err != nil {
			return err
		}
		payroll.conceptID[created.Name] = created.ID
	}
	return nil
}

// bookVacations gives two workers with a year of service ten days of vacation this month.
func (payroll *demoPayroll) bookVacations(day int16, random *mrand.Rand) error {
	activeEmployees, err := activeEmployeesOn(day)
	if err != nil {
		return err
	}
	entitled := slices.DeleteFunc(activeEmployees, func(employee types.Employee) bool { return day-employee.HireDate < 365 })
	if len(entitled) == 0 {
		return nil
	}
	timeOff, err := loadSeedRecords()
	if err != nil {
		return err
	}
	vacationStart := day + 9
	for range 2 {
		employee := entitled[random.Intn(len(entitled))]
		overlaps := slices.ContainsFunc(timeOff, func(leave types.EmployeeRecord) bool {
			return types.IsTimeOffRecord(leave.ConceptID) && leave.EmployeeID == employee.ID && leave.Status != types.StatusDeleted &&
				leave.FromDate <= vacationStart+9 && vacationStart <= leave.ToDate
		})
		if overlaps || (employee.TerminationDate > 0 && employee.TerminationDate <= vacationStart+9) {
			continue
		}
		vacation := types.EmployeeRecord{EmployeeID: employee.ID, ConceptID: types.LegalVacation, FromDate: vacationStart,
			ToDate: vacationStart + 9, Note: "Vacaciones programadas"}
		if err := callHRHandler("POST.hr-employee-records", hrModule.PostEmployeeRecord, nil, vacation, &vacation); err != nil {
			return err
		}
		timeOff = append(timeOff, vacation)
	}
	return nil
}

// payWeeklyAdvances pays the week that ends today to the two workers on a weekly wage.
func (payroll *demoPayroll) payWeeklyAdvances(day int16) error {
	activeEmployees, err := activeEmployeesOn(day - 6)
	if err != nil {
		return err
	}
	stored, err := loadSeedRecords()
	if err != nil {
		return err
	}
	advances := []types.EmployeeRecord{}
	for _, employee := range activeEmployees[len(activeEmployees)-min(2, len(activeEmployees)):] {
		alreadyPaid := slices.ContainsFunc(stored, func(advance types.EmployeeRecord) bool {
			return types.IsAdvanceRecord(advance.ConceptID) && advance.EmployeeID == employee.ID &&
				advance.Status != types.StatusDeleted && advance.ToDate == day
		})
		if alreadyPaid || (employee.TerminationDate > 0 && employee.TerminationDate < day) {
			continue
		}
		advances = append(advances, types.EmployeeRecord{EmployeeID: employee.ID, ConceptID: types.LegalAdvanceWeekly, Amount: 25000,
			FromDate: day - 6, ToDate: day, Date: day, CashBankID: payroll.seedRun.BankCashBankID})
	}
	if len(advances) == 0 {
		return nil
	}
	if err := payroll.seedRun.EnsureCashBalance(payroll.seedRun.BankCashBankID, int32(len(advances))*25000, payroll.at); err != nil {
		return err
	}
	return callHRHandler("POST.hr-pay-advances", hrModule.PostPayAdvances, nil, advances, nil)
}

// registerResignation makes one worker in about three months resign today. The first worker
// (management), the loan holder and the weekly-paid pair are kept, so the payroll's other flows
// stay populated.
func (payroll *demoPayroll) registerResignation(day int16, random *mrand.Rand) error {
	if random.Intn(3) > 0 {
		return nil
	}
	employees, err := loadSeedEmployees()
	if err != nil {
		return err
	}
	period := core.PeriodOfDay(day)
	if slices.ContainsFunc(employees, func(employee types.Employee) bool {
		return employee.TerminationDate > 0 && core.PeriodOfDay(employee.TerminationDate) == period
	}) {
		return nil
	}
	activeEmployees, err := activeEmployeesOn(day)
	if err != nil || len(activeEmployees) < 5 {
		return err
	}
	// A boss with active reports never resigns: the server refuses a terminated boss on his
	// reports' next save, and the org chart keeps its shape.
	activeManagerIDs := map[int32]bool{}
	for _, employee := range activeEmployees {
		activeManagerIDs[employee.ManagerEmployeeID] = true
	}
	candidates := slices.DeleteFunc(slices.Clone(activeEmployees[2:len(activeEmployees)-2]), func(employee types.Employee) bool {
		return activeManagerIDs[employee.ID]
	})
	if len(candidates) == 0 {
		return nil
	}
	leaver := candidates[random.Intn(len(candidates))]
	leaver.TerminationDate, leaver.TerminationReason = day, types.TerminationResignation
	return callHRHandler("POST.hr-employees", hrModule.PostEmployee, nil, leaver, nil)
}

// registerMonthRecords writes the month's overtime, absences and bonuses, unless it has some already.
func (payroll *demoPayroll) registerMonthRecords(day int16, random *mrand.Rand) error {
	period := core.PeriodOfDay(day)
	stored, err := loadSeedRecords()
	if err != nil {
		return err
	}
	if slices.ContainsFunc(stored, func(record types.EmployeeRecord) bool {
		return types.IsEntryRecord(record.ConceptID) && record.Period == period
	}) {
		return nil
	}
	activeEmployees, err := activeEmployeesOn(day)
	if err != nil {
		return err
	}
	for _, employee := range activeEmployees {
		inputs := []types.EmployeeRecord{}
		if random.Intn(3) == 0 {
			inputs = append(inputs, types.EmployeeRecord{ConceptID: types.LegalOvertime25, Quantity: float32(2 + random.Intn(10))})
		}
		if random.Intn(8) == 0 {
			inputs = append(inputs, types.EmployeeRecord{ConceptID: types.LegalOvertime35, Quantity: float32(1 + random.Intn(4))})
		}
		if random.Intn(10) == 0 {
			inputs = append(inputs, types.EmployeeRecord{ConceptID: types.LegalAbsence, Quantity: float32(1 + random.Intn(2))})
		}
		if random.Intn(6) == 0 {
			inputs = append(inputs, types.EmployeeRecord{
				ConceptID: payroll.conceptID["Bono de productividad"], Amount: int32(10000 + random.Intn(20)*1000)})
		}
		if random.Intn(20) == 0 {
			inputs = append(inputs, types.EmployeeRecord{
				ConceptID: payroll.conceptID["Bono extraordinario"], Amount: int32(50000 + random.Intn(10)*10000), Note: "Bono por campaña"})
		}
		for _, input := range inputs {
			// Runs on the 25th: each record happened on some earlier day of the same month.
			input.EmployeeID, input.Period, input.Date = employee.ID, period, day-int16(random.Intn(20))
			if err := callHRHandler("POST.hr-employee-records", hrModule.PostEmployeeRecord, nil, input, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// settleRun takes the period's run as far as it should go: created, calculated and, when
// closeAndPay, closed and paid from the bank. It resumes a run a previous pass left halfway. The
// run is inserted here, as the payroll page's ensure would, because the seed walks the months
// one day at a time and must not create the ones ahead of it.
func (payroll *demoPayroll) settleRun(period int16, closeAndPay bool) error {
	runs := []types.PayrollRun{}
	runQuery := db.Query(&runs)
	runQuery.CompanyID.Equals(runner.CompanyID)
	if err := runQuery.Exec(); err != nil {
		return err
	}
	runIndex := slices.IndexFunc(runs, func(run types.PayrollRun) bool {
		return run.Period == period && run.Status != types.StatusDeleted
	})
	run := types.PayrollRun{CompanyID: runner.CompanyID, Period: period, PayDate: core.PeriodLastDay(period),
		Status: types.PayrollStatusOpen, Created: core.SUnixTime()}
	if runIndex >= 0 {
		run = runs[runIndex]
	} else {
		created := []types.PayrollRun{run}
		if err := db.Upsert(&created); err != nil {
			return err
		}
		run = created[0]
	}

	if run.Status == types.PayrollStatusOpen {
		calculation := struct{ Run types.PayrollRun }{}
		if err := callHRHandler("POST.hr-payroll-calculate", hrModule.PostPayrollCalculate, nil, map[string]int32{"RunID": run.ID}, &calculation); err != nil {
			return err
		}
		run = calculation.Run
	}
	if !closeAndPay || run.Status == types.PayrollStatusPaid {
		return nil
	}
	if run.Status == types.PayrollStatusOpen {
		if err := callHRHandler("POST.hr-payroll-close", hrModule.PostPayrollClose, nil, map[string]int32{"RunID": run.ID}, nil); err != nil {
			return err
		}
	}
	if err := payroll.seedRun.EnsureCashBalance(payroll.seedRun.BankCashBankID, int32(run.TotalNetPay), payroll.at); err != nil {
		return err
	}
	payment := map[string]int32{"RunID": run.ID, "CashBankID": payroll.seedRun.BankCashBankID}
	if err := callHRHandler("POST.hr-payroll-paid", hrModule.PostPayrollPaid, nil, payment, nil); err != nil {
		return err
	}
	core.Log("SeedHR:: planilla pagada", "periodo", period, "boletas", run.EmployeeCount,
		"neto", fmt.Sprintf("S/ %.2f", float64(run.TotalNetPay)/100))
	return nil
}

func loadSeedEmployees() ([]types.Employee, error) {
	employees := []types.Employee{}
	query := db.Query(&employees)
	query.CompanyID.Equals(runner.CompanyID)
	if err := query.Exec(); err != nil {
		return nil, core.Err("no se pudo leer el personal:", err)
	}
	slices.SortFunc(employees, func(left, right types.Employee) int { return int(left.ID - right.ID) })
	return employees, nil
}

// loadSeedRecords reads every employee record of the demo company: what a rerun must not repeat.
func loadSeedRecords() ([]types.EmployeeRecord, error) {
	records := []types.EmployeeRecord{}
	query := db.Query(&records)
	query.CompanyID.Equals(runner.CompanyID)
	if err := query.Exec(); err != nil {
		return nil, core.Err("no se pudieron leer los registros del personal:", err)
	}
	return records, nil
}

// activeEmployeesOn lists, by ID, the workers hired by the day and not gone before it.
func activeEmployeesOn(day int16) ([]types.Employee, error) {
	employees, err := loadSeedEmployees()
	return slices.DeleteFunc(employees, func(employee types.Employee) bool {
		return employee.Status == types.StatusDeleted || employee.HireDate > day ||
			(employee.TerminationDate > 0 && employee.TerminationDate < day)
	}), err
}
