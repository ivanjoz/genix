package finance

import (
	accounting "app/accounting/types"
	"app/core"
	"app/db"
	"app/finance/types"
	production "app/production/types"
	"encoding/json"
	"fmt"
	"time"
)

// expensesQueryLimit caps the "Todos" and "Pagados" tabs; "Pend. Pago" stays uncapped.
const expensesQueryLimit int32 = 2000

// validateExpenseCommon checks the fields shared by Expense and ExpenseScheduled.
func validateExpenseCommon(amount int32, currencyType, categoryID int8) error {
	if amount <= 0 {
		return core.Err("El monto debe ser mayor a 0.")
	}
	if currencyType != 1 && currencyType != 2 {
		return core.Err("Moneda inválida (debe ser 1=PEN o 2=USD).")
	}
	if _, categoryExists := types.FindExpenseCategory(categoryID); !categoryExists {
		return core.Err("Categoría de gasto inválida.")
	}
	return nil
}

// resolveExpenseAccount returns the PCGE account to store on an expense: the one the user picked,
// or the default when they picked none. A simple expense defaults to its category's account and
// must use an expense account. An inventory expense bought a supply, so it defaults to the
// supply's own account and must use a supply account.
func resolveExpenseAccount(companyID int32, expense types.Expense) (int32, error) {
	if expense.Type == types.ExpenseTypeInventory {
		supplyAccountCode := int32(0)
		if expense.AccountCode == 0 {
			supplies := []production.Product{}
			supplyQuery := db.Query(&supplies)
			supplyQuery.Select(supplyQuery.ID, supplyQuery.AccountCode).
				CompanyID.Equals(companyID).ID.Equals(expense.ProductID)
			if queryError := supplyQuery.Exec(); queryError != nil {
				return 0, core.Err("Error al obtener la cuenta contable del insumo.", queryError)
			}
			if len(supplies) > 0 {
				supplyAccountCode = supplies[0].AccountCode
			}
		}
		return accounting.ResolveSupplyAccount(expense.AccountCode, supplyAccountCode)
	}
	category, _ := types.FindExpenseCategory(expense.CategoryID)
	accountCode := core.If(expense.AccountCode == 0, category.DefaultAccountCode, expense.AccountCode)
	return accountCode, accounting.ValidatePCGEAccount(accounting.PCGEKindExpense, accountCode)
}

// unixDayToUTC / utcToUnixDay round-trip a UnixDay (int16) through a UTC date so the
// cadence walker can reason about weekday / month / day-of-month deterministically.
func unixDayToUTC(day int16) time.Time { return time.Unix(int64(day)*86400, 0).UTC() }
func utcToUnixDay(t time.Time) int16   { return int16(t.UTC().Unix() / 86400) }

// daysInMonth returns the real length of a month, so DD can be clamped (e.g. 31 → 28/29).
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// --- GET / POST: one-time and per-period expenses ---------------------------------

// GetExpenses serves one status tab per request via the "status" query param
// (0 = Todos, 1 = Pend. Pago, 2 = Pagados) with the delta-cache protocol. Status is the delta
// index's pinned key, so the read covers exactly the tab's statuses: a row that moved to another
// tab comes back as a removed ID in the window Metadata.
func GetExpenses(req *core.HandlerArgs) core.HandlerResponse {
	statusFilter := int8(req.GetQueryInt("status"))

	// Map the tab's status code to the concrete ss values a first sync lists.
	var statusToQuery []any
	switch statusFilter {
	case 0:
		statusToQuery = []any{int8(1), int8(2)} // Todos = all live
	case 1:
		statusToQuery = []any{int8(1)}
	case 2:
		statusToQuery = []any{int8(2)}
	default:
		return req.MakeErr("El status del gasto es incorrecto.")
	}

	expenses := []types.Expense{}
	query := db.Query(&expenses).OrderDesc()
	// Todos / Pagados cap their first sync; Pend. Pago stays uncapped.
	if statusFilter != 1 {
		query.Limit(expensesQueryLimit)
	}
	query.CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("records"), statusToQuery...)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los gastos:", err)
	}

	response := map[string]any{"records": &expenses}
	return req.MakeResponse(&response)
}

// validateExpenseProductionOrder refuses a link to anything but one of the company's subcontracted
// production orders made by the expense's own supplier (docs/MANUFACTURING_PLAN.md §8).
func validateExpenseProductionOrder(companyID int32, expense types.Expense) error {
	if expense.Type != types.ExpenseTypeSimple {
		return core.Err("Solo un gasto simple se vincula a una orden de producción.")
	}
	orders := []production.ProductionOrder{}
	query := db.Query(&orders)
	query.Select(query.ID, query.Kind, query.SubcontractorID, query.Status).CompanyID.Equals(companyID).ID.Equals(expense.ProductionOrderID)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener la orden de producción:", err)
	}
	if len(orders) == 0 || orders[0].Kind != production.ProductionOrderKindManual || orders[0].Status == production.ProductionOrderStatusAnnulled {
		return core.Err(fmt.Sprintf("La orden de producción %v no existe o está anulada.", expense.ProductionOrderID))
	}
	if orders[0].SubcontractorID == 0 || orders[0].SubcontractorID != expense.SupplierID {
		return core.Err("La orden de producción no es de maquila de este proveedor.")
	}
	return nil
}

func PostExpenses(req *core.HandlerArgs) core.HandlerResponse {
	body := types.Expense{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}

	if err := validateExpenseCommon(body.Amount, body.CurrencyType, body.CategoryID); err != nil {
		return req.MakeErr(err)
	}

	// Assets are acquired from the Activos page and depreciation is generated by the server,
	// so neither type may be created here — accepting them would let a client write a
	// balance-sheet entry with no asset behind it.
	body.Type = core.If(body.Type == 0, types.ExpenseTypeSimple, body.Type)
	if body.Type != types.ExpenseTypeSimple && body.Type != types.ExpenseTypeInventory {
		return req.MakeErr("Tipo de gasto inválido: sólo se puede registrar Simple o Inventario.")
	}
	if body.Type == types.ExpenseTypeInventory && body.ProductID <= 0 {
		return req.MakeErr("Un gasto de inventario debe indicar el insumo o material comprado.")
	}
	// An asset link only belongs on the types the server itself writes.
	body.AssetID = 0
	if body.ProductionOrderID != 0 {
		if err := validateExpenseProductionOrder(req.User.CompanyID, body); err != nil {
			return req.MakeErr(err)
		}
	}

	if err := body.ApplyPurchaseDocument(req.User.CompanyID); err != nil {
		return req.MakeErr(err)
	}
	accountCode, err := resolveExpenseAccount(req.User.CompanyID, body)
	if err != nil {
		return req.MakeErr(err)
	}
	body.AccountCode = accountCode

	nowTime := core.SUnixTime()
	body.CompanyID = req.User.CompanyID
	body.Status = core.If(body.Status == 0, int8(1), body.Status)
	body.Updated = nowTime
	body.UpdatedBy = req.User.ID
	body.StampPeriod()
	// Only a payment decides it (PostExpensePayment), never the client.
	body.SameDayCashBankID = 0

	if body.ID <= 0 {
		// New record: ORM assigns the autoincrement ID in-place.
		body.Created = nowTime
		body.CreatedBy = req.User.ID
		records := []types.Expense{body}
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar el gasto:", err)
		}
		return req.MakeResponse(records[0])
	}

	// An edit runs on the expense read under its row lock, so a payment landing meanwhile is either
	// fully before it (and seen) or waits for it: the edit never writes back a stale Status or
	// PaidAmount.
	var saved types.Expense
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		existing := []types.Expense{{CompanyID: req.User.CompanyID, ID: body.ID}}
		if err := db.LockByKeys(transaction, &existing); err != nil {
			return core.Err("Error al obtener el gasto:", err)
		}
		if len(existing) == 0 {
			return core.Err("No se encontró el gasto.")
		}
		stored := existing[0]
		if stored.Status == 2 {
			// A fully-paid expense cannot be edited, except for its comprobante and its account: the
			// bill often arrives after the payment, and classifying it says nothing about the money.
			saved = stored
			return savePaidExpenseDocumentIn(transaction, req, &saved, body)
		}
		if stored.Status == types.ExpenseStatusPosted {
			return core.Err("No se puede editar un asiento de depreciación.")
		}
		if body.Amount < stored.PaidAmount {
			return core.Err("El monto no puede ser menor a lo ya pagado.")
		}
		// Payment state stays server-authoritative: Status and PaidAmount are set only by
		// PostExpensePayment, and the creation stamps never change.
		saved = body
		saved.Status = stored.Status
		saved.PaidAmount = stored.PaidAmount
		saved.Created = stored.Created
		saved.CreatedBy = stored.CreatedBy
		// The edit can move the accounting date, the total or the comprobante its payments clear.
		sameDayCashBankID, restampErr := restampExpensePaymentsIn(transaction, req.User.CompanyID, &saved)
		if restampErr != nil {
			return restampErr
		}
		saved.SameDayCashBankID = sameDayCashBankID
		if err := db.UpsertIn(transaction, &[]types.Expense{saved}); err != nil {
			return core.Err("Error al guardar el gasto:", err)
		}
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(saved)
}

// savePaidExpenseDocumentIn writes only the supplier, the comprobante and the account of a paid
// expense its caller locked, the comprobante validated against the amount that was actually paid.
// The account was already resolved by the caller.
func savePaidExpenseDocumentIn(transaction *db.Transaction, req *core.HandlerArgs, stored *types.Expense, body types.Expense) error {
	document := body.PurchaseDocument()
	document.CurrencyType = stored.CurrencyType
	stored.SupplierID = body.SupplierID
	stored.AccountCode = body.AccountCode
	stored.SetPurchaseDocument(document)
	if err := stored.ApplyPurchaseDocument(req.User.CompanyID); err != nil {
		return err
	}
	stored.Updated = core.SUnixTime()
	stored.UpdatedBy = req.User.ID
	// The comprobante moves the accounting date and decides the payable its payments clear.
	stored.StampPeriod()
	sameDayCashBankID, restampErr := restampExpensePaymentsIn(transaction, req.User.CompanyID, stored)
	if restampErr != nil {
		return restampErr
	}
	stored.SameDayCashBankID = sameDayCashBankID

	q := db.TableOf[types.Expense]()
	if err := db.UpdateIn(transaction, &[]types.Expense{*stored},
		q.SupplierID, q.ProviderSnapshotID, q.DocType, q.DocSeries, q.DocNumber, q.DocIssueDate,
		q.TaxableAmount, q.TaxAmount, q.UntaxedAmount, q.OtherAmount, q.ExchangeRate,
		q.AccountCode, q.Updated, q.UpdatedBy, q.Status, q.Period, q.SameDayCashBankID,
	); err != nil {
		return core.Err("Error al guardar el comprobante del gasto:", err)
	}
	return nil
}

// restampExpensePaymentsIn re-applies the same-day rule and the payable to the expense's payments
// after an edit, and returns the cash bank of its same-day payment (0 when none).
func restampExpensePaymentsIn(transaction *db.Transaction, companyID int32, expense *types.Expense) (int32, error) {
	return types.RestampDocumentPaymentsIn(transaction, companyID, int64(expense.ID),
		[]types.CashMovementType{types.CashMovementTypeExpensePayment},
		expense.AccountingDate(), expense.Amount, types.PurchasePayableAccount(expense.DocType))
}

// GetExpensesByIDs serves the by-IDs cache: the accounting operations read through it the older
// inventory expenses a period's supplies receptions name.
func GetExpensesByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[types.Expense](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener los gastos.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}

// --- GET / POST: recurring schedules ----------------------------------------------

func GetExpensesScheduled(req *core.HandlerArgs) core.HandlerResponse {
	records := []types.ExpenseScheduled{}

	// Delta() keeps only active rows on a first sync and fans out over both statuses afterwards, so
	// the client can evict deleted ones — it replaces the per-status loop this handler used to run.
	query := db.Query(&records)
	query.CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los gastos programados:", err)
	}

	return core.MakeResponse(req, &records)
}

func PostExpensesScheduled(req *core.HandlerArgs) core.HandlerResponse {
	body := types.ExpenseScheduled{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}

	if err := validateExpenseCommon(body.Amount, body.CurrencyType, body.CategoryID); err != nil {
		return req.MakeErr(err)
	}
	// A schedule only generates simple expenses, so its account is an expense account.
	accountCode, err := resolveExpenseAccount(req.User.CompanyID, types.Expense{
		Type: types.ExpenseTypeSimple, CategoryID: body.CategoryID, AccountCode: body.AccountCode,
	})
	if err != nil {
		return req.MakeErr(err)
	}
	body.AccountCode = accountCode
	// Frequency is the packed cadence code CDD: cadence (1..6) * 100 + day.
	cadence, day := body.Frequency/100, body.Frequency%100
	if cadence < 1 || cadence > 6 {
		return req.MakeErr("Cadencia inválida (debe estar entre 1 y 6).")
	}
	maxDay := int16(31)
	if cadence == 1 { // weekly: day is a weekday 1=Mon..7=Sun
		maxDay = 7
	}
	if day < 1 || day > maxDay {
		return req.MakeErr("El día de la cadencia está fuera de rango.")
	}
	if body.StartDate == 0 {
		return req.MakeErr("Debe especificar una fecha de inicio.")
	}

	nowTime := core.SUnixTime()
	body.CompanyID = req.User.CompanyID
	body.Status = core.If(body.Status == 0, int8(1), body.Status)
	body.Updated = nowTime
	body.UpdatedBy = req.User.ID

	records := &[]types.ExpenseScheduled{body}
	if body.ID <= 0 {
		body.Created = nowTime
		body.CreatedBy = req.User.ID
		(*records)[0] = body
		err = db.Upsert(records)
	} else {
		q := db.TableOf[types.ExpenseScheduled]()
		err = db.UpdateExclude(records, q.Created, q.CreatedBy)
	}
	if err != nil {
		return req.MakeErr("Error al guardar el gasto programado:", err)
	}
	return req.MakeResponse((*records)[0])
}

// --- Lazy period generation -------------------------------------------------------

// schedulePeriodDates walks the cadence from StartDate to `end` (inclusive) and returns
// every period's UnixDay. See EXPENSES.md §5 for the anchor rules.
func schedulePeriodDates(schedule *types.ExpenseScheduled, end int16) []int16 {
	cadence, day := schedule.Frequency/100, int(schedule.Frequency%100)
	start := schedule.StartDate
	if schedule.EndDate > 0 && schedule.EndDate < end {
		end = schedule.EndDate
	}
	periods := []int16{}
	if start > end {
		return periods
	}

	if cadence == 1 {
		// Weekly: first period is the first occurrence of weekday `day` on/after StartDate.
		startT := unixDayToUTC(start)
		targetWeekday := day % 7 // our 7=Sun maps to Go's 0=Sunday
		offset := (targetWeekday - int(startT.Weekday()) + 7) % 7
		periodT := startT.AddDate(0, 0, offset)
		for i := 0; i < 5000; i++ { // guard against runaway loops
			periodDay := utcToUnixDay(periodT)
			if periodDay > end {
				break
			}
			periods = append(periods, periodDay)
			periodT = periodT.AddDate(0, 0, 7)
		}
		return periods
	}

	// Monthly / N-monthly / yearly: StartDate anchors the month(s); `day` is day-of-month.
	stepMonths := map[int16]int{2: 1, 3: 2, 4: 3, 5: 4, 6: 12}[cadence]
	startT := unixDayToUTC(start)
	year, month := startT.Year(), startT.Month()
	for i := 0; i < 5000; i++ {
		clampedDay := day
		if dim := daysInMonth(year, month); clampedDay > dim {
			clampedDay = dim
		}
		periodDay := utcToUnixDay(time.Date(year, month, clampedDay, 0, 0, 0, 0, time.UTC))
		if periodDay > end {
			break
		}
		if periodDay >= start { // skip a first period that lands before StartDate
			periods = append(periods, periodDay)
		}
		// advance by stepMonths, handling year rollover
		m := int(month) - 1 + stepMonths
		year += m / 12
		month = time.Month(m%12) + 1
	}
	return periods
}

func GetExpenseSchedulePeriods(req *core.HandlerArgs) core.HandlerResponse {
	scheduleID := req.GetQueryInt("scheduleID")
	if scheduleID == 0 {
		return req.MakeErr("No se envió el scheduleID.")
	}

	// The schedule's row lock serializes two loads of its periods: the second waits, then finds the
	// periods the first one created, so a period is never materialized twice. (A unique index on
	// (ExpenseScheduledID, PeriodDate) cannot do it: every one-time expense stores (0, 0).)
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		schedules := []types.ExpenseScheduled{{CompanyID: req.User.CompanyID, ID: scheduleID}}
		if err := db.LockByKeys(transaction, &schedules); err != nil {
			return core.Err("Error al obtener el gasto programado:", err)
		}
		if len(schedules) == 0 {
			return core.Err("No se encontró el gasto programado.")
		}
		return materializeSchedulePeriodsIn(transaction, req, schedules[0])
	})
	if err != nil {
		return req.MakeErr(err)
	}

	// The periods are read as the schedule's cache group after the commit, so the browser
	// downloads them again only when a period was just created or has changed since.
	heldGroups, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}
	periodGroups := []db.RecordGroup[types.Expense]{}
	query := db.QueryIndexGroup(&periodGroups).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range heldGroups {
		query.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	query.ExpenseScheduledID.Between(scheduleID, scheduleID)
	if err := query.Exec(); err != nil {
		core.Log("Error querying expense schedule period group:", err)
		return req.MakeErr("Error al obtener los periodos del gasto.")
	}
	return core.MakeResponse(req, &periodGroups)
}

// materializeSchedulePeriodsIn creates the missing periods of the schedule up to today, under the
// schedule's lock its caller holds.
func materializeSchedulePeriodsIn(transaction *db.Transaction, req *core.HandlerArgs, schedule types.ExpenseScheduled) error {
	// Load existing periods for this schedule (local index on ExpenseScheduledID).
	periods := []types.Expense{}
	pq := db.Query(&periods)
	pq.Select(pq.PeriodDate).CompanyID.Equals(req.User.CompanyID).ExpenseScheduledID.Equals(schedule.ID)
	if err := pq.Exec(); err != nil {
		return core.Err("Error al obtener los periodos del gasto:", err)
	}

	existing := core.SliceSet[int16]{}
	for _, p := range periods {
		existing.Add(p.PeriodDate)
	}

	// Materialize any missing period up to today (deduped by PeriodDate).
	nowTime := core.SUnixTime()
	today := core.FechaUnix()
	newPeriods := []types.Expense{}
	for _, periodDate := range schedulePeriodDates(&schedule, today) {
		if existing.Include(periodDate) {
			continue
		}
		newPeriods = append(newPeriods, types.Expense{
			CompanyID:          req.User.CompanyID,
			ExpenseScheduledID: schedule.ID,
			PeriodDate:         periodDate,
			Name:               schedule.Name,
			Description:        schedule.Description,
			Type:               types.ExpenseTypeSimple,
			CategoryID:         schedule.CategoryID,
			AccountCode:        schedule.AccountCode,
			SupplierID:         schedule.SupplierID,
			CurrencyType:       schedule.CurrencyType,
			Date:               periodDate,
			DueDate:            periodDate,
			// A period is born without its bill; the "sin comprobante" list finds it by this date.
			DocIssueDate: periodDate,
			Amount:       schedule.Amount,
			Status:       1,
			Updated:      nowTime,
			UpdatedBy:    req.User.ID,
			Created:      nowTime,
			CreatedBy:    req.User.ID,
		})
	}

	if len(newPeriods) > 0 {
		for index := range newPeriods {
			newPeriods[index].StampPeriod()
		}
		if err := db.InsertIn(transaction, &newPeriods); err != nil {
			return core.Err("Error al generar los periodos del gasto:", err)
		}
	}
	return nil
}

// --- Payment ----------------------------------------------------------------------

func PostExpensePayment(req *core.HandlerArgs) core.HandlerResponse {
	body := struct {
		ExpenseID   int32 `json:"ExpenseID"`
		CashBankID  int32 `json:"CashBankID"`
		Amount      int32 `json:"Amount"`      // positive payment amount, in cents
		Date        int16 `json:"Date"`        // payment date (UnixDay)
		IsFullyPaid bool  `json:"IsFullyPaid"` // forces full-paid status (write-off case)
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}

	if body.Amount <= 0 {
		return req.MakeErr("El monto del pago debe ser mayor a 0.")
	}
	if body.ExpenseID == 0 || body.CashBankID == 0 {
		return req.MakeErr("Faltan parámetros: (ExpenseID o CashBankID).")
	}

	// The expense and then the cash bank are locked, so the pending balance and the till's balance
	// checked here are the ones the payment is written against: two payments of one expense apply
	// one after the other, and the second sees what the first paid.
	var expense types.Expense
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		expenses := []types.Expense{{CompanyID: req.User.CompanyID, ID: body.ExpenseID}}
		if err := db.LockByKeys(transaction, &expenses); err != nil {
			return core.Err("Error al obtener el gasto:", err)
		}
		if len(expenses) == 0 {
			return core.Err("No se encontró el gasto.")
		}
		expense = expenses[0]
		cashBanks := []types.CashBank{{CompanyID: req.User.CompanyID, ID: body.CashBankID}}
		if err := db.LockByKeys(transaction, &cashBanks); err != nil {
			return core.Err("Error al obtener la caja:", err)
		}
		if len(cashBanks) == 0 {
			return core.Err("No se encontró la caja.")
		}
		return payExpenseIn(transaction, req, &expense, cashBanks[0], body.Amount, body.Date, body.IsFullyPaid)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(expense)
}

// payExpenseIn writes one payment of the expense from cashBank, both locked by the caller, and the
// expense's new paid amount and status.
func payExpenseIn(transaction *db.Transaction, req *core.HandlerArgs, expense *types.Expense,
	cashBank types.CashBank, amount int32, date int16, isFullyPaid bool,
) error {
	// A depreciation entry moves no cash, so there is nothing to pay against it.
	if expense.Status == types.ExpenseStatusPosted || expense.Type == types.ExpenseTypeDepreciation {
		return core.Err("Un asiento de depreciación no se paga.")
	}

	// Reject a payment larger than the outstanding balance (server-authoritative). This also
	// blocks any payment on an already fully-paid expense, whose pending balance is 0.
	pendingAmount := expense.Amount - expense.PaidAmount
	if amount > pendingAmount {
		return core.Err("El monto del pago no puede ser mayor al monto pendiente.")
	}
	if cashBank.CurrencyType != expense.CurrencyType {
		return core.Err("La moneda de la caja no coincide con la del gasto.")
	}

	// The movement is an outflow, so its amount is negative. We don't validate the client's
	// expected balance; we only reject a payment that would drive the cash-bank balance
	// negative. The resulting balance is computed server-side by ApplyCashBankMovementIn.
	movementAmount := -amount
	if cashBank.CurrentAmount+movementAmount < 0 {
		return core.Err("El saldo de la caja no puede quedar negativo.")
	}

	// PaidAmount = Σ |movement.Amount| over this expense's committed payments, plus this one: the
	// read sees committed rows only, and under the expense's lock no other payment of it is written
	// meanwhile. Type filters out the other registers' movements, since DocumentID alone is shared.
	movements := []types.CashBankMovement{}
	mq := db.Query(&movements)
	mq.Select(mq.Type, mq.Amount).CompanyID.Equals(req.User.CompanyID).DocumentID.Equals(int64(expense.ID))
	if err := mq.Exec(); err != nil {
		return core.Err("Error al recalcular el monto pagado:", err)
	}
	paidAmount := amount
	for _, m := range movements {
		if m.Type != types.CashMovementTypeExpensePayment {
			continue
		}
		paidAmount += core.If(m.Amount < 0, -m.Amount, m.Amount)
	}

	// Record the outflow movement (it updates the cash-bank balance). FinalAmount is left at 0 so
	// ApplyCashBankMovementIn computes it from the locked balance. When it alone pays the whole
	// expense on its accounting date, the accounting operations infer it from the expense instead
	// of downloading it.
	paymentDate := core.If(date != 0, date, core.FechaUnix())
	isSameDayPayment := types.IsSameDaySettlement(expense.AccountingDate(), expense.Amount,
		expense.PaidAmount, paymentDate, amount)
	movement := types.InternalCashMovement{
		CashBankID:         cashBank.ID,
		DocumentID:         int64(expense.ID),
		ReferenceID:        expense.ExpenseScheduledID,
		Date:               paymentDate,
		Type:               types.CashMovementTypeExpensePayment,
		CounterAccountCode: types.PurchasePayableAccount(expense.DocType),
		Amount:             movementAmount,
		FinalAmount:        0,
		SameDayMovement:    int8(core.If(isSameDayPayment, 1, 0)),
	}
	if err := types.ApplyCashBankMovementIn(transaction, req, []types.InternalCashMovement{movement}); err != nil {
		return err
	}

	// Resolve the payment lifecycle on Status (honoring the "Is Fully Paid" write-off flag):
	// 2 = fully paid, 1 = pending (unpaid/partial). This moves the row between status tabs.
	expense.PaidAmount = paidAmount
	if isFullyPaid || paidAmount >= expense.Amount {
		expense.Status = 2
	} else {
		expense.Status = 1
	}
	expense.Updated = core.SUnixTime()
	expense.UpdatedBy = req.User.ID
	if isSameDayPayment {
		expense.SameDayCashBankID = cashBank.ID
	}
	expense.StampPeriod()

	q := db.TableOf[types.Expense]()
	if err := db.UpdateIn(transaction, &[]types.Expense{*expense},
		q.PaidAmount, q.Status, q.Updated, q.UpdatedBy, q.Period, q.SameDayCashBankID,
	); err != nil {
		return core.Err("Error al actualizar el gasto:", err)
	}
	return nil
}
