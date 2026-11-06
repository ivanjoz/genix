// Operating expenses of a two-store Lima minimarket, replayed day by day like the ERP history:
// recurring bills through expense schedules, one-off expenses, supplies bought into stock, the
// monthly taxes, and the payments that settle them. Every expense is classified under the PCGE
// account a Peruvian accountant would use and carries the supplier's comprobante (factura with
// IGV, utility bill, fee receipt, boleta, or none), so the P&L, the Registro de Compras and the
// cash reports come out the way a real month looks.
//
// Payroll is deliberately absent. TODO: pending development — HR (RRHH) is its own module, and
// staff costs (6211 sueldos, 627 EsSalud, 6214 gratificaciones, 629 CTS) belong to it, not here.
package sample_records

import (
	"app/accounting"
	"app/core"
	crmTypes "app/crm/types"
	"app/db"
	"app/finance"
	financeTypes "app/finance/types"
	production "app/production/types"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

//go:embed expense_suppliers.json
var expenseSupplierSeedJSON []byte

const expensesStateFileName = "genix_expenses_state.json"

type expenseCadence int8

const (
	// A monthly bill that lives as an expense schedule: the period is materialized by the
	// schedule, and the supplier's bill is attached to it when it arrives.
	cadenceScheduledMonthly expenseCadence = iota + 1
	cadenceMonthlyDay
	cadenceQuarterlyDay
	cadenceWeekday
	cadenceRandomDay
	// The month's IGV and income-tax prepayment, computed from the previous month's activity.
	cadenceMonthlyTax
)

// expenseTemplate is one kind of expense the business incurs. Amounts are gross, in cents.
type expenseTemplate struct {
	key         string
	name        string
	categoryID  int8
	accountCode int32
	supplierRUC string // empty: no supplier, which only a document-less expense allows
	docType     int8
	docSeries   string
	minAmount   int32
	maxAmount   int32
	cadence     expenseCadence
	// Day of the month for the monthly cadences (past the month's end means its last day),
	// weekday 1=Mon..7=Sun for the weekly one.
	day int
	// Percent chance per day, for cadenceRandomDay.
	dailyChance  int
	payAfterDays int16
	isInventory  bool
}

// expenseTemplates is the minimarket's cost structure. The accounts follow the PCGE: 63x services
// by third parties, 64x taxes that are a cost, 40x taxes that settle a liability, 60x supplies.
var expenseTemplates = []expenseTemplate{
	{key: "rent-store-1", name: "Alquiler local Tienda 1", categoryID: 1, accountCode: 6352, supplierRUC: "20604511201",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F001", minAmount: 850000, maxAmount: 850000,
		cadence: cadenceScheduledMonthly, day: 1, payAfterDays: 2},
	{key: "rent-store-2", name: "Alquiler local Tienda 2", categoryID: 1, accountCode: 6352, supplierRUC: "20604511202",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F001", minAmount: 620000, maxAmount: 620000,
		cadence: cadenceScheduledMonthly, day: 1, payAfterDays: 3},
	{key: "electricity", name: "Energía eléctrica", categoryID: 2, accountCode: 6361, supplierRUC: "20604511203",
		docType: financeTypes.PurchaseDocTypeUtilityBill, docSeries: "S001", minAmount: 240000, maxAmount: 330000,
		cadence: cadenceScheduledMonthly, day: 8, payAfterDays: 12},
	{key: "water", name: "Agua potable y alcantarillado", categoryID: 2, accountCode: 6363, supplierRUC: "20604511204",
		docType: financeTypes.PurchaseDocTypeUtilityBill, docSeries: "S002", minAmount: 38000, maxAmount: 52000,
		cadence: cadenceScheduledMonthly, day: 12, payAfterDays: 14},
	{key: "internet", name: "Internet fibra óptica", categoryID: 2, accountCode: 6365, supplierRUC: "20604511205",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F701", minAmount: 18990, maxAmount: 18990,
		cadence: cadenceScheduledMonthly, day: 5},
	{key: "phone", name: "Telefonía móvil", categoryID: 2, accountCode: 6364, supplierRUC: "20604511206",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F702", minAmount: 12900, maxAmount: 14500,
		cadence: cadenceScheduledMonthly, day: 18},
	{key: "accounting", name: "Honorarios contables", categoryID: 9, accountCode: 6323, supplierRUC: "10457812301",
		docType: financeTypes.PurchaseDocTypeFeeReceipt, docSeries: "E001", minAmount: 120000, maxAmount: 120000,
		cadence: cadenceScheduledMonthly, day: 28, payAfterDays: 2},
	{key: "security", name: "Servicio de seguridad", categoryID: 9, accountCode: 638, supplierRUC: "20604511208",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F003", minAmount: 350000, maxAmount: 350000,
		cadence: cadenceScheduledMonthly, day: 28, payAfterDays: 5},
	{key: "cleaning", name: "Servicio de limpieza", categoryID: 9, accountCode: 638, supplierRUC: "20604511209",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F004", minAmount: 180000, maxAmount: 180000,
		cadence: cadenceScheduledMonthly, day: 28, payAfterDays: 5},
	{key: "pos-software", name: "Licencia sistema POS", categoryID: 10, accountCode: 639, supplierRUC: "20604511214",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F005", minAmount: 9900, maxAmount: 9900,
		cadence: cadenceScheduledMonthly, day: 3},

	{key: "advertising", name: "Publicidad (volantes y redes sociales)", categoryID: 8, accountCode: 6371, supplierRUC: "20604511210",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F006", minAmount: 40000, maxAmount: 120000,
		cadence: cadenceRandomDay, dailyChance: 7, payAfterDays: 7},
	{key: "freight", name: "Fletes y movilidad", categoryID: 7, accountCode: 6311, supplierRUC: "20604511211",
		docType: financeTypes.PurchaseDocTypeBoleta, docSeries: "B001", minAmount: 3000, maxAmount: 15000,
		cadence: cadenceWeekday, day: 6},
	{key: "refrigeration", name: "Mantenimiento de equipos de frío", categoryID: 6, accountCode: 6343, supplierRUC: "20604511212",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F007", minAmount: 35000, maxAmount: 90000,
		cadence: cadenceRandomDay, dailyChance: 4, payAfterDays: 10},
	{key: "supplies", name: "Compra de insumos", categoryID: 4, supplierRUC: "20604511213",
		docType: financeTypes.PurchaseDocTypeFactura, docSeries: "F008", cadence: cadenceRandomDay, dailyChance: 30,
		payAfterDays: 15, isInventory: true},
	{key: "bank-fees", name: "Comisiones bancarias", categoryID: 10, accountCode: 639,
		minAmount: 2500, maxAmount: 6000, cadence: cadenceMonthlyDay, day: 31},
	{key: "itf", name: "Impuesto a las transacciones financieras", categoryID: 5, accountCode: 6412,
		minAmount: 1500, maxAmount: 4000, cadence: cadenceMonthlyDay, day: 31},
	{key: "municipal", name: "Arbitrios municipales", categoryID: 5, accountCode: 643,
		minAmount: 45000, maxAmount: 45000, cadence: cadenceQuarterlyDay, day: 28},
	{key: "igv", name: "Pago de IGV del mes anterior", categoryID: 5, accountCode: 40111,
		cadence: cadenceMonthlyTax, day: 15},
	{key: "income-tax", name: "Pago a cuenta del Impuesto a la Renta", categoryID: 5, accountCode: 40171,
		cadence: cadenceMonthlyTax, day: 15},
}

// plannedExpense is one expense of the day, drawn when the day starts and stored in the state, so
// a resumed day books exactly what it planned.
type plannedExpense struct {
	TemplateKey string
	Amount      int32
	// What gets paid and when: 0 leaves the expense pending forever.
	PayAmount int32
	PayDay    int16
	// Inventory purchases only.
	ProductID   int32
	WarehouseID int32
	Quantity    int32
	UnitPrice   int32
}

// pendingExpensePayment is a payment promised for a later day — a bill paid at its due date.
type pendingExpensePayment struct {
	ExpenseID int32
	PayDay    int16
	Amount    int32
}

// GenerateExpenses is the entry point behind `fn-generate-expenses`: the expense history alone,
// paid from the company's first cash register.
func GenerateExpenses(args *core.ExecArgs) core.FuncResponse {
	defaults := makeErpHistoryDefaults()
	defaults.days = 60
	config, configError := parseHistoryArgs(args.Message, defaults)
	if configError != nil {
		return args.MakeErr("Argumentos inválidos:", configError)
	}

	generator := newHistoryGenerator(config, expensesStateFileName)
	defer core.SetHistoricalUnix(0)

	if err := generator.loadState(); err != nil {
		return args.MakeErr("No se pudo cargar el estado de la corrida:", err)
	}
	if err := generator.validateContext(); err != nil {
		return args.MakeErr("No se pudo validar el contexto:", err)
	}
	if err := generator.prepareExpenses(); err != nil {
		return args.MakeErr(err)
	}
	if config.dryRun {
		firstDay := core.TimeToFechaUnix(generator.realNow) - int16(config.days-1)
		return core.FuncResponse{
			Message: "Dry-run: no se escribió nada.",
			Content: map[string]any{"firstDayPlan": generator.planDayExpenses(generator.dayStartTime(firstDay))},
		}
	}

	generator.daySteps = []historyDayStep{{name: "expenses", run: generator.runExpensesStep}}
	historicalDates, err := generator.runDayLoop()
	if err != nil {
		return args.MakeErr(err)
	}
	return core.FuncResponse{
		Message: "Gastos generados correctamente.",
		Content: map[string]any{"days": len(historicalDates), "stats": generator.state.Stats},
	}
}

// prepareExpenses seeds the expense suppliers, loads the supplies the inventory purchases buy, and
// makes sure every monthly bill has its expense schedule.
func (generator *erpHistoryGenerator) prepareExpenses() error {
	supplierSeeds := []partySeed{}
	if err := json.Unmarshal(expenseSupplierSeedJSON, &supplierSeeds); err != nil {
		return core.Err("error al leer el JSON de proveedores de gastos:", err)
	}
	suppliers, err := generator.saveAndLoadParties(supplierSeeds, crmTypes.ClientProviderTypeProvider)
	if err != nil {
		return core.Err("No se pudieron sembrar los proveedores de gastos:", err)
	}
	generator.expenseSupplierByRUC = map[string]int32{}
	for _, supplier := range suppliers {
		generator.expenseSupplierByRUC[supplier.RegistryNumber] = supplier.ID
	}
	for _, template := range expenseTemplates {
		if template.supplierRUC != "" && generator.expenseSupplierByRUC[template.supplierRUC] == 0 {
			return core.Err("el gasto", template.key, "apunta a un proveedor que no se sembró:", template.supplierRUC)
		}
	}

	// Consumables only: a depreciable supply is bought as a fixed asset, never as an expense.
	supplies := []production.Product{}
	supplyQuery := db.Query(&supplies)
	supplyQuery.Select(supplyQuery.ID, supplyQuery.Name, supplyQuery.Price, supplyQuery.DepreciationMonths).
		CompanyID.Equals(generator.userToken.CompanyID).Status.Equals(production.ProductStatusSupply)
	if err := supplyQuery.Exec(); err != nil {
		return core.Err("error al consultar los insumos:", err)
	}
	generator.expenseSupplies = []production.Product{}
	for _, supply := range supplies {
		if supply.DepreciationMonths == 0 && supply.Price > 0 {
			generator.expenseSupplies = append(generator.expenseSupplies, supply)
		}
	}
	if len(generator.expenseSupplies) == 0 {
		return core.Err("no hay insumos consumibles; corre primero fn-generate-supply-data")
	}

	// A dry run only previews the plan, so it does not create the schedules.
	if generator.config.dryRun {
		return nil
	}
	return generator.ensureExpenseSchedules()
}

// ensureExpenseSchedules creates the schedule of every monthly bill that does not exist yet,
// matched by name, starting on the first simulated day so each month gets its period.
func (generator *erpHistoryGenerator) ensureExpenseSchedules() error {
	request := generator.makeRequest("GET.expenses-scheduled", nil, "")
	storedSchedules := []financeTypes.ExpenseScheduled{}
	if err := decodeResponse(generator.send(request, finance.GetExpensesScheduled), &storedSchedules); err != nil {
		return err
	}
	generator.expenseScheduleByKey = map[string]int32{}
	scheduleIDByName := map[string]int32{}
	for _, schedule := range storedSchedules {
		if schedule.Status == 1 {
			scheduleIDByName[schedule.Name] = schedule.ID
		}
	}

	firstUnixDay := core.TimeToFechaUnix(generator.realNow) - int16(generator.config.days-1)
	for _, template := range expenseTemplates {
		if template.cadence != cadenceScheduledMonthly {
			continue
		}
		if scheduleID := scheduleIDByName[template.name]; scheduleID > 0 {
			generator.expenseScheduleByKey[template.key] = scheduleID
			continue
		}
		schedule := financeTypes.ExpenseScheduled{
			Name:         template.name,
			CategoryID:   template.categoryID,
			AccountCode:  template.accountCode,
			SupplierID:   generator.expenseSupplierByRUC[template.supplierRUC],
			CurrencyType: financeTypes.CurrencyPEN,
			Amount:       (template.minAmount + template.maxAmount) / 2,
			// Monthly cadence (2) on the template's day.
			Frequency: int16(200 + template.day),
			StartDate: firstUnixDay,
		}
		bodyBytes, err := json.Marshal(schedule)
		if err != nil {
			return err
		}
		createRequest := generator.makeRequest("POST.expenses-scheduled", nil, string(bodyBytes))
		createdSchedule := financeTypes.ExpenseScheduled{}
		if err := decodeResponse(finance.PostExpensesScheduled(&createRequest), &createdSchedule); err != nil {
			return core.Err("no se pudo crear el gasto programado", template.name, ":", err)
		}
		generator.expenseScheduleByKey[template.key] = createdSchedule.ID
	}
	return nil
}

// runExpensesStep settles the payments that fall due today, then books the day's expenses. Both
// checkpoint after every write.
func (generator *erpHistoryGenerator) runExpensesStep(dayStart time.Time) error {
	unixDay := core.TimeToFechaUnix(dayStart)

	paymentTime := dayStart.Add(10*time.Hour + 15*time.Minute)
	for paymentIndex := 0; paymentIndex < len(generator.state.PendingExpensePayments); {
		pendingPayment := generator.state.PendingExpensePayments[paymentIndex]
		if pendingPayment.PayDay > unixDay {
			paymentIndex++
			continue
		}
		if err := generator.payExpense(pendingPayment.ExpenseID, pendingPayment.Amount, paymentTime); err != nil {
			return err
		}
		paymentTime = paymentTime.Add(time.Second)
		generator.state.PendingExpensePayments = append(generator.state.PendingExpensePayments[:paymentIndex],
			generator.state.PendingExpensePayments[paymentIndex+1:]...)
		if err := generator.saveState(); err != nil {
			return err
		}
	}

	if !generator.state.ExpensePlanDrawn {
		generator.state.ExpensePlan = generator.planDayExpenses(dayStart)
		generator.state.ExpensePlanDrawn = true
		if err := generator.saveState(); err != nil {
			return err
		}
	}

	for planIndex := generator.state.CompletedExpenses; planIndex < len(generator.state.ExpensePlan); planIndex++ {
		bookedAt := dayStart.Add(10*time.Hour + 30*time.Minute + time.Duration(planIndex)*time.Minute)
		if err := generator.bookPlannedExpense(generator.state.ExpensePlan[planIndex], bookedAt); err != nil {
			return core.Err("gasto", generator.state.ExpensePlan[planIndex].TemplateKey, ":", err)
		}
		generator.state.CompletedExpenses = planIndex + 1
		if err := generator.saveState(); err != nil {
			return err
		}
	}
	return nil
}

// planDayExpenses decides which expenses fall on this day and draws their amounts and payment
// outcome: most are paid in full, some partially, a few stay pending, as a real payables list.
func (generator *erpHistoryGenerator) planDayExpenses(dayStart time.Time) []plannedExpense {
	unixDay := core.TimeToFechaUnix(dayStart)
	dayPlan := []plannedExpense{}

	for _, template := range expenseTemplates {
		if !generator.expenseFallsOnDay(template, dayStart) {
			continue
		}
		planned := plannedExpense{TemplateKey: template.key}

		switch {
		case template.cadence == cadenceMonthlyTax:
			planned.Amount = generator.previousMonthTax(template.key, dayStart)
			if planned.Amount <= 0 {
				continue
			}
		case template.isInventory:
			supply := generator.expenseSupplies[generator.random.Intn(len(generator.expenseSupplies))]
			planned.ProductID = supply.ID
			planned.WarehouseID = generator.config.warehouseIDs[generator.random.Intn(len(generator.config.warehouseIDs))]
			planned.Quantity = int32(5 + generator.random.Intn(26))
			planned.UnitPrice = supply.Price
			planned.Amount = planned.UnitPrice * planned.Quantity
		default:
			planned.Amount = int32(generator.randomInRange(int(template.minAmount), int(template.maxAmount)))
		}

		// A document-less charge (taxes, bank fees) is debited on the spot.
		paymentOutcome := generator.random.Intn(100)
		switch {
		case template.docType == financeTypes.PurchaseDocTypeNone || paymentOutcome < 85:
			planned.PayAmount = planned.Amount
		case paymentOutcome < 95:
			planned.PayAmount = planned.Amount / 2
		}
		planned.PayDay = unixDay + template.payAfterDays
		dayPlan = append(dayPlan, planned)
	}
	return dayPlan
}

func (generator *erpHistoryGenerator) expenseFallsOnDay(template expenseTemplate, dayStart time.Time) bool {
	switch template.cadence {
	case cadenceScheduledMonthly, cadenceMonthlyDay, cadenceMonthlyTax:
		return isMonthDay(dayStart, template.day)
	case cadenceQuarterlyDay:
		return int(dayStart.Month())%3 == 0 && isMonthDay(dayStart, template.day)
	case cadenceWeekday:
		// Go counts Sunday as 0; the templates count it as 7.
		return (int(dayStart.Weekday())+6)%7+1 == template.day
	case cadenceRandomDay:
		return generator.random.Intn(100) < template.dailyChance
	}
	return false
}

// isMonthDay matches a day of the month, a day past the month's end meaning its last day — the same
// clamp the expense schedules apply.
func isMonthDay(dayStart time.Time, monthDay int) bool {
	lastDayOfMonth := time.Date(dayStart.Year(), dayStart.Month()+1, 0, 0, 0, 0, 0, dayStart.Location()).Day()
	return dayStart.Day() == min(monthDay, lastDayOfMonth)
}

// previousMonthTax computes the tax paid this month for the previous one, from the sales and
// purchases the run generated: IGV is the sales IGV less the purchases' tax credit, and the income
// tax prepayment is 1.5% of the net sales. A month the run did not cover owes nothing here.
func (generator *erpHistoryGenerator) previousMonthTax(templateKey string, dayStart time.Time) int32 {
	previousMonthDay := core.TimeToFechaUnix(dayStart.AddDate(0, -1, 0))
	totals := generator.state.TaxTotalsByPeriod[core.PeriodOfDay(previousMonthDay)]
	if totals == nil {
		return 0
	}
	if templateKey == "igv" {
		salesIGV := totals.SalesGross * 18 / 118
		return int32(max(salesIGV-totals.PurchaseTaxIGV, 0))
	}
	return int32(totals.SalesGross * 100 / 118 * 15 / 1000)
}

// bookPlannedExpense writes one planned expense through the route its kind uses, then pays it now
// or queues the payment for its day.
func (generator *erpHistoryGenerator) bookPlannedExpense(planned plannedExpense, bookedAt time.Time) error {
	template := findExpenseTemplate(planned.TemplateKey)
	unixDay := core.TimeToFechaUnix(bookedAt)
	document := makePurchaseDocument(template.docType, template.docSeries,
		generator.nextDocNumber(template.key), unixDay, planned.Amount)
	dueDay := unixDay + template.payAfterDays

	core.SetHistoricalUnix(bookedAt.Unix())
	bookedExpense := financeTypes.Expense{}
	var err error
	switch {
	case template.cadence == cadenceScheduledMonthly:
		bookedExpense, err = generator.billSchedulePeriod(template, unixDay, dueDay, planned.Amount, document)
	case template.isInventory:
		bookedExpense, err = generator.postInventoryExpense(template, planned, unixDay, dueDay, document)
	default:
		expense := financeTypes.Expense{
			Name:         template.name,
			Type:         financeTypes.ExpenseTypeSimple,
			CategoryID:   template.categoryID,
			AccountCode:  template.accountCode,
			SupplierID:   generator.expenseSupplierByRUC[template.supplierRUC],
			CurrencyType: financeTypes.CurrencyPEN,
			Date:         unixDay,
			DueDate:      dueDay,
			Amount:       planned.Amount,
		}
		expense.SetPurchaseDocument(document)
		bookedExpense, err = generator.postExpense(expense)
	}
	if err != nil {
		return err
	}
	generator.addPurchaseTax(unixDay, document.TaxAmount)
	generator.state.Stats.Expenses++

	if planned.PayAmount <= 0 {
		return nil
	}
	if planned.PayDay <= unixDay {
		return generator.payExpense(bookedExpense.ID, planned.PayAmount, bookedAt.Add(30*time.Second))
	}
	generator.state.PendingExpensePayments = append(generator.state.PendingExpensePayments,
		pendingExpensePayment{ExpenseID: bookedExpense.ID, PayDay: planned.PayDay, Amount: planned.PayAmount})
	return nil
}

// billSchedulePeriod materializes the schedule's period for today (the clock is frozen on it) and
// attaches the supplier's bill: its actual amount, its comprobante and its due date.
func (generator *erpHistoryGenerator) billSchedulePeriod(template expenseTemplate, unixDay int16, dueDay int16,
	amount int32, document financeTypes.PurchaseDocument) (financeTypes.Expense, error) {

	scheduleID := generator.expenseScheduleByKey[template.key]
	request := generator.makeRequest("GET.expense-schedule-periods",
		map[string]string{"scheduleID": strconv.Itoa(int(scheduleID))}, "")
	response := map[string][]financeTypes.Expense{}
	if err := decodeResponse(generator.send(request, finance.GetExpenseSchedulePeriods), &response); err != nil {
		return financeTypes.Expense{}, err
	}
	for _, period := range response["Periods"] {
		if period.PeriodDate != unixDay {
			continue
		}
		period.Amount = amount
		period.DueDate = dueDay
		period.SetPurchaseDocument(document)
		return generator.postExpense(period)
	}
	return financeTypes.Expense{}, core.Err("el gasto programado", template.name, "no generó el periodo del día", unixDay)
}

func (generator *erpHistoryGenerator) postInventoryExpense(template expenseTemplate, planned plannedExpense,
	unixDay int16, dueDay int16, document financeTypes.PurchaseDocument) (financeTypes.Expense, error) {

	supplyName := ""
	for _, supply := range generator.expenseSupplies {
		if supply.ID == planned.ProductID {
			supplyName = supply.Name
		}
	}
	payload := accounting.InventoryExpensePayload{
		Name:             fmt.Sprint(template.name, ": ", supplyName),
		CategoryID:       template.categoryID,
		SupplierID:       generator.expenseSupplierByRUC[template.supplierRUC],
		Date:             unixDay,
		DueDate:          dueDay,
		ProductID:        planned.ProductID,
		WarehouseID:      planned.WarehouseID,
		Quantity:         planned.Quantity,
		UnitPrice:        planned.UnitPrice,
		PurchaseDocument: document,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return financeTypes.Expense{}, err
	}
	request := generator.makeRequest("POST.expense-inventory", nil, string(bodyBytes))
	createdExpense := financeTypes.Expense{}
	return createdExpense, decodeResponse(generator.send(request, accounting.PostInventoryExpense), &createdExpense)
}

func (generator *erpHistoryGenerator) postExpense(expense financeTypes.Expense) (financeTypes.Expense, error) {
	bodyBytes, err := json.Marshal(expense)
	if err != nil {
		return financeTypes.Expense{}, err
	}
	request := generator.makeRequest("POST.expenses", nil, string(bodyBytes))
	savedExpense := financeTypes.Expense{}
	return savedExpense, decodeResponse(generator.send(request, finance.PostExpenses), &savedExpense)
}

// payExpense pays from the expense cash register, topping it up first when it cannot cover it.
func (generator *erpHistoryGenerator) payExpense(expenseID int32, amount int32, paidAt time.Time) error {
	if err := generator.ensureCashBalance(generator.expenseCashBankID, amount, paidAt.Add(-time.Second)); err != nil {
		return err
	}
	bodyBytes, err := json.Marshal(map[string]any{
		"ExpenseID":  expenseID,
		"CashBankID": generator.expenseCashBankID,
		"Amount":     amount,
		"Date":       core.TimeToFechaUnix(paidAt),
	})
	if err != nil {
		return err
	}
	core.SetHistoricalUnix(paidAt.Unix())
	request := generator.makeRequest("POST.expense-payment", nil, string(bodyBytes))
	if response := generator.send(request, finance.PostExpensePayment); response.StatusCode != 200 {
		return core.Err("no se pudo pagar el gasto", expenseID, ":", response.Error)
	}
	generator.state.Stats.ExpensePayments++
	return nil
}

func findExpenseTemplate(templateKey string) expenseTemplate {
	for _, template := range expenseTemplates {
		if template.key == templateKey {
			return template
		}
	}
	panic("expense template not found: " + templateKey)
}
