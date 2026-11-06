// The HR seed lives the demo company's payroll one day at a time, through the production handlers
// of app/hr, with the process clock frozen on each simulated day:
//
//   - First day: the legal parameters, the staff as it stands that day, the company concepts,
//     recurring movilidad and a judicial withholding, and one loan.
//   - Day 1 of a month: two workers book ten days of vacation (10th–19th).
//   - Days 7, 14, 21 and 28: the two last workers on the staff list are paid their weekly advance.
//   - Day 20: now and then a worker resigns.
//   - Day 25: the month's novedades (overtime, absences, bonuses).
//   - Month end: the month's single run (its payslips carry the gratificación, the CTS and the
//     leavers' settlements in their months), closed and paid.
//   - Last day, unless it is a month end: the month's run, calculated and left open.
//
// Every payment comes out of the demo bank account. Each action first checks what is stored, so
// running the same days again writes nothing twice. The randomness of a day is seeded by the day
// itself, so a resumed or repeated run draws the same novedades.
//
// It runs inside fn-generate-demo-data (one more step of the day loop) and on its own as
// fn-seed-hr, over the last N months of a database that already has the ERP data.
package sample_records

import (
	"cmp"
	"fmt"
	mrand "math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	"app/core"
	"app/db"
	financeTypes "app/finance/types"
	"app/hr"
	"app/hr/types"
)

const hrSeedStateFileName = "genix_seed_hr_state.json"

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

// demoPayroll writes the HR side of each simulated day; history supplies the bank account, the
// clock helpers and the cash top-ups.
type demoPayroll struct {
	history   *erpHistoryGenerator
	seed      int64
	firstDay  int16
	lastDay   int16
	at        time.Time // the frozen moment of the day's HR writes
	conceptID map[string]int32
	paidRuns  []string
}

func newDemoPayroll(history *erpHistoryGenerator, firstDay int16) *demoPayroll {
	return &demoPayroll{history: history, seed: history.config.seed, firstDay: firstDay,
		lastDay: core.TimeToFechaUnix(history.realNow)}
}

// runPayrollStep is the day step. It runs late in the day, after the tills' takings reached the
// bank, so the payments find the day's deposits.
func (payroll *demoPayroll) runPayrollStep(dayStart time.Time) error {
	day := core.TimeToFechaUnix(dayStart)
	period := core.PeriodOfDay(day)
	payroll.at = dayStart.Add(21*time.Hour + 15*time.Minute)
	core.SetHistoricalUnix(payroll.at.Unix())
	random := mrand.New(mrand.NewSource(payroll.seed + int64(day)))

	if day == payroll.firstDay {
		if err := payroll.setup(day); err != nil {
			return core.Err("preparación de RRHH:", err)
		}
	}
	if payroll.conceptID == nil {
		if err := payroll.ensureConcepts(); err != nil {
			return err
		}
	}

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

	if isMonthDay(dayStart, 31) {
		return payroll.settleRun(period, true)
	} else if day == payroll.lastDay {
		return payroll.settleRun(period, false)
	}
	return nil
}

// setup writes what the company has before the first day: the legal values, the staff (unless the
// company already has one), the concepts, the recurring items and a loan.
func (payroll *demoPayroll) setup(day int16) error {
	if response := SeedHRLegalParameters(&core.ExecArgs{}); response.Error != "" {
		return core.Err("parámetros laborales:", response.Error)
	}
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
		if err := callHRHandler("POST.hr-employee-concepts", hr.PostEmployeeConcept, nil, recurring, nil); err != nil {
			return err
		}
	}
	if len(activeEmployees) < 2 {
		return nil
	}
	// A personal loan recovered in three installments.
	loan := types.EmployeeRecord{EmployeeID: activeEmployees[1].ID, ConceptID: types.LegalLoan, Date: day, Amount: 90000,
		Quantity: 3, Period: period, Note: "Préstamo personal", CashBankID: payroll.history.bankCashBankID}
	if err := payroll.history.ensureCashBalance(loan.CashBankID, loan.Amount, payroll.at); err != nil {
		return err
	}
	return callHRHandler("POST.hr-loans", hr.PostLoan, nil, loan, nil)
}

// ensureConcepts creates the seed concepts the company lacks and keeps their IDs by name.
func (payroll *demoPayroll) ensureConcepts() error {
	concepts := []types.Concept{}
	query := db.Query(&concepts)
	query.CompanyID.Equals(sampleCompanyID)
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
		if err := callHRHandler("POST.hr-concepts", hr.PostConcept, nil, conceptSeed, &created); err != nil {
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
		if err := callHRHandler("POST.hr-employee-records", hr.PostEmployeeRecord, nil, vacation, &vacation); err != nil {
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
			FromDate: day - 6, ToDate: day, Date: day, CashBankID: payroll.history.bankCashBankID})
	}
	if len(advances) == 0 {
		return nil
	}
	if err := payroll.history.ensureCashBalance(payroll.history.bankCashBankID, int32(len(advances))*25000, payroll.at); err != nil {
		return err
	}
	return callHRHandler("POST.hr-pay-advances", hr.PostPayAdvances, nil, advances, nil)
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
	return callHRHandler("POST.hr-employees", hr.PostEmployee, nil, leaver, nil)
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
			if err := callHRHandler("POST.hr-employee-records", hr.PostEmployeeRecord, nil, input, nil); err != nil {
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
	runQuery.CompanyID.Equals(sampleCompanyID)
	if err := runQuery.Exec(); err != nil {
		return err
	}
	runIndex := slices.IndexFunc(runs, func(run types.PayrollRun) bool {
		return run.Period == period && run.Status != types.StatusDeleted
	})
	run := types.PayrollRun{CompanyID: sampleCompanyID, Period: period, PayDate: core.PeriodLastDay(period),
		Status: types.PayrollStatusOpen, Created: core.SUnixTime()}
	if runIndex >= 0 {
		run = runs[runIndex]
	} else {
		created := []types.PayrollRun{run}
		if err := db.Insert(&created); err != nil {
			return err
		}
		run = created[0]
	}

	if run.Status == types.PayrollStatusOpen {
		calculation := struct{ Run types.PayrollRun }{}
		if err := callHRHandler("POST.hr-payroll-calculate", hr.PostPayrollCalculate, nil, map[string]int32{"RunID": run.ID}, &calculation); err != nil {
			return err
		}
		run = calculation.Run
	}
	if !closeAndPay || run.Status == types.PayrollStatusPaid {
		return nil
	}
	if run.Status == types.PayrollStatusOpen {
		if err := callHRHandler("POST.hr-payroll-close", hr.PostPayrollClose, nil, map[string]int32{"RunID": run.ID}, nil); err != nil {
			return err
		}
	}
	if err := payroll.history.ensureCashBalance(payroll.history.bankCashBankID, int32(run.TotalNetPay), payroll.at); err != nil {
		return err
	}
	payment := map[string]int32{"RunID": run.ID, "CashBankID": payroll.history.bankCashBankID}
	if err := callHRHandler("POST.hr-payroll-paid", hr.PostPayrollPaid, nil, payment, nil); err != nil {
		return err
	}
	payroll.paidRuns = append(payroll.paidRuns, fmt.Sprintf("%d: %d boletas, neto S/ %.2f", period,
		run.EmployeeCount, float64(run.TotalNetPay)/100))
	return nil
}

func loadSeedEmployees() ([]types.Employee, error) {
	employees := []types.Employee{}
	query := db.Query(&employees)
	query.CompanyID.Equals(sampleCompanyID)
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
	query.CompanyID.Equals(sampleCompanyID)
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

// SeedHR is the entry point behind `fn-seed-hr`: the HR day steps over the last N months, up to
// today, on a database that already has the ERP data. Those payments are written into days the
// bank account already has movements for, so the account's running balances are recomputed at
// the end (rebalanceCashBank).
func SeedHR(args *core.ExecArgs) core.FuncResponse {
	months, reset := 7, false
	for _, argument := range strings.Fields(args.Message) {
		name, rawValue, _ := strings.Cut(strings.TrimPrefix(argument, "--"), "=")
		value, err := strconv.Atoi(rawValue)
		switch {
		case name == "reset":
			reset = true
		case name == "months" && err == nil && value >= 1 && value <= 12:
			months = value
		default:
			return args.MakeErr("Argumento desconocido o inválido:", argument, "(--months=1..12, --reset)")
		}
	}

	config := makeErpHistoryDefaults()
	generator := newHistoryGenerator(config, hrSeedStateFileName)
	today := core.TimeToFechaUnix(generator.realNow)
	firstPeriod := core.PeriodOfDay(today)
	for range months - 1 {
		firstPeriod = core.PeriodOfDay(core.PeriodFirstDay(firstPeriod) - 1)
	}
	firstDay := core.PeriodFirstDay(firstPeriod)
	generator.config.days, generator.config.reset = int(today-firstDay)+1, reset
	defer core.SetHistoricalUnix(0)

	if err := generator.loadState(); err != nil {
		return args.MakeErr("No se pudo cargar el estado de la corrida:", err)
	}
	if err := generator.ensureDemoWarehouses(); err != nil {
		return args.MakeErr("No se pudieron preparar los almacenes:", err)
	}
	if err := generator.validateContext(); err != nil {
		return args.MakeErr("No se pudo validar el contexto:", err)
	}
	if err := generator.ensureDemoCashAccounts(); err != nil {
		return args.MakeErr("No se pudieron preparar las cajas:", err)
	}

	payroll := newDemoPayroll(generator, firstDay)
	generator.daySteps = []historyDayStep{{name: "payroll", run: payroll.runPayrollStep}}
	if _, err := generator.runDayLoop(); err != nil {
		return args.MakeErr(err)
	}
	core.SetHistoricalUnix(0)
	rewritten, err := generator.rebalanceCashBank(generator.bankCashBankID, generator.dayStartTime(firstDay-1))
	if err != nil {
		return args.MakeErr("No se pudieron recalcular los saldos del banco:", err)
	}

	message := fmt.Sprintf("RRHH sembrado del %s al %s; %d saldos del banco recalculados.\n%s",
		generator.dayStartTime(firstDay).Format("2006-01-02"), generator.dayStartTime(today).Format("2006-01-02"),
		rewritten, strings.Join(payroll.paidRuns, "\n"))
	return core.FuncResponse{Message: message}
}

// rebalanceCashBank recomputes the running balance (FinalAmount) of every movement of a cash
// account in ID order — account, date, then write order within the day — and rewrites the ones
// that changed. The opening balance is the account's current balance less every movement. If the
// trail dips below zero somewhere, a capital contribution dated the day before openingDayStart
// covers the deepest point and the trail is recomputed once more.
func (generator *erpHistoryGenerator) rebalanceCashBank(cashBankID int32, openingDayStart time.Time) (int, error) {
	rewritten := 0
	for range 2 {
		movements := []financeTypes.CashBankMovement{}
		query := db.Query(&movements)
		query.CompanyID.Equals(sampleCompanyID).CashBankID.Equals(cashBankID)
		if err := query.Exec(); err != nil {
			return rewritten, err
		}
		slices.SortFunc(movements, func(left, right financeTypes.CashBankMovement) int { return cmp.Compare(left.ID, right.ID) })
		currentBalance, err := generator.readCashBankBalance(cashBankID)
		if err != nil {
			return rewritten, err
		}
		running := currentBalance
		for _, movement := range movements {
			running -= movement.Amount
		}
		lowestBalance := running
		changed := []financeTypes.CashBankMovement{}
		for _, movement := range movements {
			running += movement.Amount
			lowestBalance = min(lowestBalance, running)
			if movement.FinalAmount != running {
				movement.FinalAmount = running
				changed = append(changed, movement)
			}
		}
		if len(changed) > 0 {
			table := db.TableOf[financeTypes.CashBankMovement]()
			if err := db.Update(&changed, table.FinalAmount); err != nil {
				return rewritten, err
			}
			rewritten += len(changed)
		}
		if lowestBalance >= 0 {
			return rewritten, nil
		}
		if err := generator.injectCash(cashBankID, openingDayStart.Add(8*time.Hour), currentBalance, -lowestBalance); err != nil {
			return rewritten, err
		}
		core.SetHistoricalUnix(0)
	}
	return rewritten, core.Err("el saldo de la caja", cashBankID, "sigue quedando negativo")
}
