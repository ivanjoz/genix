package finance

import (
	accounting "app/accounting/types"
	config "app/config/types"
	"app/core"
	"app/db"
	"app/finance/types"
	hr "app/hr/types"
	sales "app/sales/types"
	"fmt"
	"math"
	"slices"
)

// GET.cash-flow-commitments?horizon=N (plan/CASH_FLOW_FORECAST_PLAN.md §12): the Known and
// Scheduled layers of the cash flow forecast for the next N days, each commitment already dated, and
// the warnings describing what that dating assumed. One computed response, read when the page opens.
// The open documents (sales, expenses, purchase orders, assets) and the taxes they generate are not
// here: the browser computes them from the cash-documents frame (plan/CASH_DOCUMENTS_FRAME_PLAN.md).

const (
	cashFlowDefaultHorizonDays = 91
	cashFlowMaxHorizonDays     = 365
	// The closed periods whose taxes the browser checks.
	cashFlowClosedTaxPeriods = 3
	// The days of the cash-documents frame the browser reads; older open payables come as one total.
	cashFlowDocumentWindowDays = 365
	// How far back an unmaterialized schedule period still counts as owed.
	cashFlowScheduleLookbackDays = 60
	// How far back the manual cash movement warnings look.
	cashFlowMovementLookbackDays = 90
)

type CashFlowCommitmentsResponse struct {
	Commitments []types.CashFlowCommitment
	Warnings    []types.CashFlowWarning
	TaxOutlook  types.CashFlowTaxOutlook
	OldPayables types.CashFlowOldPayables
}

// cashFlowBuild is one computation: the window, what several streams share, and the output.
type cashFlowBuild struct {
	companyID   int32
	today       int16
	end         int16
	company     config.Company
	taxCalendar map[int16]accounting.TaxDueCalendar // by tax period
	// periodsWithoutCalendar raised tax-calendar-missing already: several taxes share a period.
	periodsWithoutCalendar map[int16]bool
	paydays                []int16
	holidays               map[int16]bool
	commitments            []types.CashFlowCommitment
	warnings               types.CashFlowWarnings
	taxOutlook             types.CashFlowTaxOutlook
	oldPayables            types.CashFlowOldPayables
}

func GetCashFlowCommitments(req *core.HandlerArgs) core.HandlerResponse {
	horizon := int16(req.GetQueryInt("horizon"))
	if horizon <= 0 {
		horizon = cashFlowDefaultHorizonDays
	}
	horizon = min(horizon, cashFlowMaxHorizonDays)
	today := core.FechaUnix()
	build := &cashFlowBuild{companyID: req.User.CompanyID, today: today, end: today + horizon,
		periodsWithoutCalendar: map[int16]bool{}}

	for _, step := range []func() error{
		build.loadShared, build.addExpenseSchedules, build.addLoanInstallments, build.addPayroll, build.addTaxPeriods,
		build.addOldPayables, build.warnCashMovements,
	} {
		if err := step(); err != nil {
			return req.MakeErr(err)
		}
	}
	return req.MakeResponse(CashFlowCommitmentsResponse{Commitments: build.commitments, Warnings: build.warnings.List(),
		TaxOutlook: build.taxOutlook, OldPayables: build.oldPayables})
}

// add keeps a commitment that falls inside the window.
func (build *cashFlowBuild) add(commitment types.CashFlowCommitment) {
	if commitment.Amount != 0 && commitment.Date <= build.end {
		build.commitments = append(build.commitments, commitment)
	}
}

// loadShared reads the company's tax setup, the tax calendar and the calendar events.
func (build *cashFlowBuild) loadShared() error {
	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Select(companyQuery.ID, companyQuery.RUC, companyQuery.TaxRegime, companyQuery.IncomeTaxPrepaymentRate,
		companyQuery.IsGoodTaxpayer, companyQuery.NrusCategory).ID.Equals(build.companyID)
	if err := companyQuery.Exec(); err != nil || len(companies) == 0 {
		return core.Err("Error al obtener la empresa:", err)
	}
	build.company = companies[0]

	calendars := []accounting.TaxDueCalendar{}
	calendarQuery := db.Query(&calendars)
	calendarQuery.Select().CountryID.Equals(hr.CountryIDPeru)
	if err := calendarQuery.Exec(); err != nil {
		return core.Err("Error al obtener el calendario tributario:", err)
	}
	build.taxCalendar = map[int16]accounting.TaxDueCalendar{}
	for _, calendar := range calendars {
		if calendar.Status > 0 {
			build.taxCalendar[calendar.Period] = calendar
		}
	}

	// The global holidays live in partition 0; the company's paydays in its own.
	build.holidays = map[int16]bool{}
	for _, partitionID := range []int32{sales.CalendarGlobalCompanyID, build.companyID} {
		events := []sales.CalendarEvent{}
		eventQuery := db.Query(&events)
		eventQuery.Select(eventQuery.Date, eventQuery.EventType).CompanyID.Equals(partitionID).Status.Equals(1)
		if err := eventQuery.Exec(); err != nil {
			return core.Err("Error al obtener los eventos del calendario:", err)
		}
		for _, event := range events {
			switch event.EventType {
			case sales.CalendarEventHoliday:
				build.holidays[event.Date] = true
			case sales.CalendarEventPayday:
				if partitionID == build.companyID {
					build.paydays = append(build.paydays, event.Date)
				}
			}
		}
	}
	slices.Sort(build.paydays)
	return nil
}

// taxDueDay is the SUNAT deadline of a tax period for this company; without a calendar row, the
// 15th of the following month (tax-calendar-missing).
func (build *cashFlowBuild) taxDueDay(period int16) int16 {
	if calendar, found := build.taxCalendar[period]; found {
		if dueDay := calendar.DueDayFor(build.company.RUC, build.company.IsGoodTaxpayer); dueDay > 0 {
			return dueDay
		}
	}
	if !build.periodsWithoutCalendar[period] {
		build.periodsWithoutCalendar[period] = true
		build.warnings.Add("tax-calendar-missing", 0, int64(period), fmt.Sprintf("Periodo %d", period))
	}
	return core.PeriodFirstDay(core.NextPeriod(period)) + 14
}

// addExpenseSchedules walks every active schedule to the end of the window. A period that already
// has its expense is Known (the cash-documents frame has it, Kinds 12/13); a recent past period that
// never got one is owed today. A schedule without an amount projects the mean of its last 3 periods, or nothing.
func (build *cashFlowBuild) addExpenseSchedules() error {
	schedules := []types.ExpenseScheduled{}
	query := db.Query(&schedules)
	query.Select().CompanyID.Equals(build.companyID).Status.Equals(1)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los gastos programados:", err)
	}
	for _, schedule := range schedules {
		periodExpenses := []types.Expense{}
		expenseQuery := db.Query(&periodExpenses)
		expenseQuery.Select(expenseQuery.PeriodDate, expenseQuery.Amount, expenseQuery.Status).
			CompanyID.Equals(build.companyID).ExpenseScheduledID.Equals(schedule.ID)
		if err := expenseQuery.Exec(); err != nil {
			return core.Err("Error al obtener los periodos del gasto programado:", err)
		}
		materialized := map[int16]bool{}
		recentAmounts := []int32{}
		slices.SortFunc(periodExpenses, func(left, right types.Expense) int { return int(left.PeriodDate) - int(right.PeriodDate) })
		for _, expense := range periodExpenses {
			if expense.Status > 0 {
				materialized[expense.PeriodDate] = true
				recentAmounts = append(recentAmounts, expense.Amount)
			}
		}
		amount := int64(schedule.Amount)
		if amount == 0 {
			recentAmounts = recentAmounts[max(0, len(recentAmounts)-3):]
			for _, recentAmount := range recentAmounts {
				amount += int64(recentAmount)
			}
			if len(recentAmounts) == 0 {
				build.warnings.Add("schedule-amount-zero", 0, int64(schedule.ID), schedule.Name)
				continue
			}
			amount /= int64(len(recentAmounts))
		}
		for _, periodDate := range schedulePeriodDates(&schedule, build.end) {
			if materialized[periodDate] || periodDate < build.today-cashFlowScheduleLookbackDays {
				continue
			}
			cashDate, _ := types.DocumentCashDate(periodDate, periodDate, build.today)
			build.add(types.CashFlowCommitment{Stream: types.CashFlowStreamExpenseSchedule, Layer: types.CashFlowLayerScheduled,
				SiteID: schedule.SiteID, SourceID: int64(schedule.ID), Label: schedule.Name, Date: cashDate, DueDate: periodDate,
				Amount: -amount, CurrencyType: schedule.CurrencyType})
		}
	}
	return nil
}

// addLoanInstallments: what every installment of an active loan still owes, in the company lane.
func (build *cashFlowBuild) addLoanInstallments() error {
	loans := []types.Loan{}
	query := db.Query(&loans)
	query.Select().CompanyID.Equals(build.companyID).Status.Equals(types.LoanStatusActive)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los préstamos:", err)
	}
	for _, loan := range loans {
		if loan.AnnualRate == 0 {
			build.warnings.Add("loan-rate-missing", 0, int64(loan.ID), loan.Name)
		}
		for _, installment := range loan.Installments {
			pending := int64(installment.Total() - installment.PaidAmount)
			if pending <= 0 {
				continue
			}
			label := fmt.Sprintf("%s · cuota %d", loan.Name, installment.Number)
			cashDate, isOverdue := types.DocumentCashDate(installment.DueDate, installment.DueDate, build.today)
			if isOverdue {
				build.warnings.Add("loan-installment-overdue", pending, int64(loan.ID), label)
			}
			build.add(types.CashFlowCommitment{Stream: types.CashFlowStreamLoanInstallment,
				Layer: core.If(isOverdue, types.CashFlowLayerKnown, types.CashFlowLayerScheduled), SourceID: int64(loan.ID),
				Label: label, Date: cashDate, DueDate: installment.DueDate, Amount: -pending, CurrencyType: loan.CurrencyType})
		}
	}
	return nil
}

// addPayroll covers last month to the end of the window. A closed run is Known: what it still owes
// (net pay until it is paid, each contribution less its payments). A month without a closed run is
// Scheduled: the payroll engine projects it over the current staff and contracts, each projected
// month feeding the next one's history (5ta, provisions).
func (build *cashFlowBuild) addPayroll() error {
	data, err := hr.LoadPayrollData(build.companyID)
	if err != nil {
		return err
	}
	runs := []hr.PayrollRun{}
	runQuery := db.Query(&runs)
	runQuery.Select().CompanyID.Equals(build.companyID)
	if err := runQuery.Exec(); err != nil {
		return core.Err("Error al obtener las planillas:", err)
	}
	runByPeriod := map[int16]hr.PayrollRun{}
	for _, run := range runs {
		if run.Status != hr.StatusDeleted {
			runByPeriod[run.Period] = run
		}
	}

	currentPeriod := core.PeriodOfDay(build.today)
	for _, employee := range data.Employees {
		if !hr.IsOnPayroll(employee, currentPeriod) {
			continue
		}
		if _, hasContract := hr.FindContractInForce(hr.ContractsOf(data.Contracts, employee.ID), build.today); !hasContract {
			build.warnings.Add("payroll-contract-missing", 0, int64(employee.ID), employee.FirstName+" "+employee.LastName)
		}
	}

	firstPeriod := core.AddPeriods(currentPeriod, -1)
	for period := firstPeriod; period <= core.PeriodOfDay(build.end); period = core.NextPeriod(period) {
		run, hasRun := runByPeriod[period]
		periodLabel := fmt.Sprintf("Planilla %d", period)
		payday, hasPayday := types.PaydayOf(period, build.paydays)
		if hasRun && run.PayDate > 0 {
			payday, hasPayday = run.PayDate, true
		}

		var cash types.PayrollCash
		layer := types.CashFlowLayerScheduled
		if hasRun && run.Status >= hr.PayrollStatusClosed {
			layer = types.CashFlowLayerKnown
			runPayslips := slices.DeleteFunc(slices.Clone(data.History), func(payslip hr.Payslip) bool { return payslip.RunID != run.ID })
			cash = types.PayrollCashOf(runPayslips)
			if run.Status == hr.PayrollStatusPaid {
				cash.NetPay, cash.Cts = 0, 0
			}
			if err := build.subtractContributionPayments(run.ID, cash.Contributions); err != nil {
				return err
			}
		} else {
			if period == firstPeriod {
				build.warnings.Add("payroll-run-open", 0, int64(period), periodLabel)
			}
			cash = types.PayrollCashOf(projectPayrollMonth(&data, period))
		}
		if cash.NetPay == 0 && cash.Cts == 0 && len(cash.Contributions) == 0 {
			continue
		}
		if !hasPayday {
			build.warnings.Add("payday-missing", 0, int64(period), periodLabel)
		}

		addPayrollFlow := func(stream int8, label string, dueDate int16, amount int64) {
			cashDate, _ := types.DocumentCashDate(dueDate, dueDate, build.today)
			build.add(types.CashFlowCommitment{Stream: stream, Layer: layer, SourceID: int64(period), Label: label,
				Date: cashDate, DueDate: dueDate, Amount: -amount, CurrencyType: types.CurrencyPEN})
		}
		addPayrollFlow(types.CashFlowStreamPayrollNetPay, periodLabel+" · neto", payday, cash.NetPay)
		// The CTS is deposited by the 15th of May and November, from the month's payslip.
		addPayrollFlow(types.CashFlowStreamPayrollCts, periodLabel+" · CTS", core.PeriodFirstDay(period)+14, cash.Cts)
		for accountCode, amount := range cash.Contributions {
			dueDate := build.taxDueDay(period)
			switch accountCode {
			case types.PCGEAfpPayable:
				dueDate = types.NthBusinessDay(core.NextPeriod(period), 5, build.holidays)
			case types.PCGEOtherStaffPayable:
				dueDate = payday
			}
			addPayrollFlow(types.CashFlowStreamPayrollContribution, fmt.Sprintf("%s · %d", periodLabel, accountCode), dueDate, amount)
		}
	}
	return nil
}

// projectPayrollMonth runs the payroll engine for a month nobody closed yet, and adds the result to
// the history as closed so the following projected months read it.
func projectPayrollMonth(data *hr.PayrollData, period int16) []hr.Payslip {
	run := hr.PayrollRun{Period: period, ID: -int32(period)}
	payslips := []hr.Payslip{}
	for _, employee := range data.Employees {
		payslip, ok := hr.CalculatePeriodPayslip(data.PeriodInputFor(employee, run, 0, 0))
		if !ok {
			continue
		}
		payslip.RunID, payslip.Status = run.ID, hr.PayrollStatusClosed
		payslips = append(payslips, payslip)
	}
	data.History = append(data.History, payslips...)
	return payslips
}

// subtractContributionPayments takes off each payable what the run's contribution payments paid.
func (build *cashFlowBuild) subtractContributionPayments(runID int32, contributions map[int32]int64) error {
	movements := []types.CashBankMovement{}
	query := db.Query(&movements)
	query.Select(query.Type, query.CounterAccountCode, query.Amount).CompanyID.Equals(build.companyID).
		DocumentID.Equals(int64(runID))
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los pagos de la planilla:", err)
	}
	for _, movement := range movements {
		if movement.Type == types.CashMovementTypePayrollContributions {
			contributions[movement.CounterAccountCode] += int64(movement.Amount) // payments are negative
		}
	}
	for accountCode, amount := range contributions {
		if amount <= 0 {
			delete(contributions, accountCode)
		}
	}
	return nil
}

// addTaxPeriods lists the tax periods the browser computes from the cash-documents frame: the last
// closed ones and the open and following ones whose due date falls in the window, each with its SUNAT
// due date and what the 401x expenses already paid for it (option B, plan/CASH_DOCUMENTS_FRAME_PLAN.md
// §6). Whether a period is paid can't be computed from the documents, so the backend reads it.
func (build *cashFlowBuild) addTaxPeriods() error {
	if build.company.TaxRegime == 0 {
		build.warnings.Add("tax-regime-missing", 0, int64(build.companyID), "Empresa")
	}
	build.taxOutlook.Rules = build.taxRules()
	currentPeriod := core.PeriodOfDay(build.today)
	for period := core.AddPeriods(currentPeriod, -cashFlowClosedTaxPeriods); ; period = core.NextPeriod(period) {
		dueDate := build.taxDueDay(period)
		if period >= currentPeriod && dueDate > build.end {
			break
		}
		build.taxOutlook.Periods = append(build.taxOutlook.Periods, types.CashFlowTaxPeriod{Period: period, DueDate: dueDate})
	}

	// A tax payment is dated in the month its period falls due: it pays the period before its own.
	paymentPeriods := []int16{}
	periodIndexByPaymentPeriod := map[int16]int{}
	for index, taxPeriod := range build.taxOutlook.Periods {
		paymentPeriod := core.NextPeriod(taxPeriod.Period)
		paymentPeriods = append(paymentPeriods, paymentPeriod)
		periodIndexByPaymentPeriod[paymentPeriod] = index
	}
	expenses := []types.Expense{}
	query := db.Query(&expenses)
	query.Select(query.Period, query.Type, query.AccountCode, query.Amount, query.CurrencyType, query.ExchangeRate).
		CompanyID.Equals(build.companyID).Period.In(paymentPeriods...).Status.In(1, 2)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los pagos de impuestos:", err)
	}
	for _, expense := range expenses {
		index, found := periodIndexByPaymentPeriod[expense.Period]
		if !found || expense.Type == types.ExpenseTypeDepreciation || !types.IsTaxPayableAccount(expense.AccountCode) {
			continue
		}
		taxPeriod := &build.taxOutlook.Periods[index]
		amountLocal := types.AmountInPEN(expense.CurrencyType, expense.ExchangeRate, int64(expense.Amount))
		if types.IsIncomeTaxPayableAccount(expense.AccountCode) {
			taxPeriod.PaidIncomeLocal += amountLocal
		} else if types.IsIgvPayableAccount(expense.AccountCode) || build.company.TaxRegime == config.TaxRegimeNrus {
			taxPeriod.PaidIgvLocal += amountLocal
		}
	}
	return nil
}

// addOldPayables totals the open payables dated before the window the browser reads of the
// cash-documents frame (cashFlowDocumentWindowDays): they are overdue, so the browser adds them on
// today and raises payables-before-window. Receivables that old are stale and stay out.
func (build *cashFlowBuild) addOldPayables() error {
	expenseTable := db.TableOf[types.Expense]()
	rows, _, err := db.QueryFrame[types.Expense](build.companyID, types.CashDocumentsFrame.Name).
		Between(expenseTable.Date, 1, int64(build.today-cashFlowDocumentWindowDays)).
		Between(expenseTable.SiteID, 0, math.MaxInt16).
		ExecRows()
	if err != nil {
		return core.Err("Error al obtener las cuentas por pagar antiguas:", err)
	}
	for _, row := range rows {
		// RowValues follow CashDocumentFrame: Kind, TaxGroup, DocumentOffset, DueOffset.
		kind, documentOffset := int8(row.RowValues[0]), row.RowValues[2]
		if !types.IsOpenPayableCashDocumentKind(kind) {
			continue
		}
		if documentOffset > 0 {
			build.oldPayables.Dollars += row.Sums[0]
		} else {
			build.oldPayables.Local += row.Sums[0]
		}
	}
	return nil
}

// taxRules resolves the company's regime: Nuevo RUS pays only its cuota; every other regime charges
// IGV and prepays renta at its rate. A company with no regime set gets IGV only (tax-regime-missing).
func (build *cashFlowBuild) taxRules() types.TaxRules {
	if build.company.TaxRegime == config.TaxRegimeNrus {
		return types.TaxRules{NrusQuota: int64(config.NrusMonthlyQuotaCents[build.company.NrusCategory])}
	}
	return types.TaxRules{ChargesIgv: true, IncomeTaxRate: build.company.IncomeTaxPrepaymentRate}
}

// warnCashMovements reads the last 90 days' manual movements: an unspecified one (type 1) feeds the
// history as is, and a Cobro against 451 is a loan nobody registered, whose repayments are unknown.
func (build *cashFlowBuild) warnCashMovements() error {
	firstDay := build.today - cashFlowMovementLookbackDays
	periods := []int16{}
	for period := core.PeriodOfDay(firstDay); period <= core.PeriodOfDay(build.today); period = core.NextPeriod(period) {
		periods = append(periods, period)
	}
	movements := []types.CashBankMovement{}
	query := db.Query(&movements)
	query.Select(query.ID, query.Date, query.Type, query.Amount, query.CounterAccountCode).CompanyID.Equals(build.companyID).
		Period.In(periods...).SameDayMovement.In(0, 1).
		Type.In(types.CashMovementTypeUnspecified, types.CashMovementTypeCollection)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los movimientos de caja manuales:", err)
	}
	for _, movement := range movements {
		if movement.Date < firstDay {
			continue
		}
		label := fmt.Sprintf("Movimiento %d", movement.ID)
		if movement.Type == types.CashMovementTypeUnspecified {
			build.warnings.Add("unspecified-cash-movements", int64(movement.Amount), movement.ID, label)
		} else if movement.CounterAccountCode == types.PCGELoansReceived {
			build.warnings.Add("loan-received-as-cobro", int64(movement.Amount), movement.ID, label)
		}
	}
	return nil
}
