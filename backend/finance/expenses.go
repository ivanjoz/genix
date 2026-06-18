package finance

import (
	"app/core"
	"app/db"
	financeTypes "app/finance/types"
	"encoding/json"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"
)

// expensesQueryLimit caps the "Todos" and "Pagados" tabs; "Pend. Pago" stays uncapped.
const expensesQueryLimit int32 = 2000

// expenseCategoryIDs is the static, code-defined category set (mirrored on the
// frontend as `expenseCategories`). Used only to validate incoming CategoryID.
var expenseCategoryIDs = map[int8]bool{
	1: true, 2: true, 3: true, 4: true, 5: true,
	6: true, 7: true, 8: true, 9: true, 10: true,
}

// movementTypeExpensePayment is the CashBankMovement.Type used for expense payments
// (outflow). Mirrors `cajaMovimientoTipos` id 9 on the frontend.
const movementTypeExpensePayment int8 = 9

// validateExpenseCommon checks the fields shared by Expense and ExpenseScheduled.
func validateExpenseCommon(amount int32, currencyType, categoryID int8) error {
	if amount <= 0 {
		return core.Err("El monto debe ser mayor a 0.")
	}
	if currencyType != 1 && currencyType != 2 {
		return core.Err("Moneda inválida (debe ser 1=PEN o 2=USD).")
	}
	if !expenseCategoryIDs[categoryID] {
		return core.Err("Categoría de gasto inválida.")
	}
	return nil
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

// expensesComplementStatuses returns the ss values that have LEFT the given tab, so a
// delta sync can evict rows that no longer belong to it (mirrors GetSaleOrders).
//   - status 1 (Pend. Pago): evict paid (2) and removed (0)
//   - status 2 (Pagados):    evict pending (1) and removed (0)
//   - status 0 (Todos):      evict only removed (0)
func expensesComplementStatuses(statusFilter int8) []int8 {
	switch statusFilter {
	case 1:
		return []int8{2, 0}
	case 2:
		return []int8{1, 0}
	default:
		return []int8{0}
	}
}

// GetExpenses serves one status tab per request via the "status" query param
// (0 = Todos, 1 = Pend. Pago, 2 = Pagados) and supports delta sync via "updated".
// Each tab is its own delta-cache query; rows that change status are evicted through
// "records_IDsToRemove" (same strategy as GetSaleOrders).
func GetExpenses(req *core.HandlerArgs) core.HandlerResponse {
	statusFilter := int8(req.GetQueryInt("status"))
	updated := req.GetQueryInt("updated")
	queryGroup := errgroup.Group{}

	// Map the tab's status code to the concrete ss values to fetch.
	var statusToQuery []int8
	switch statusFilter {
	case 0:
		statusToQuery = []int8{1, 2} // Todos = all live
	case 1:
		statusToQuery = []int8{1}
	case 2:
		statusToQuery = []int8{2}
	default:
		return req.MakeErr("El status del gasto es incorrecto.")
	}

	// Todos / Pagados are capped; Pend. Pago stays uncapped.
	limit := int32(0)
	if statusFilter != 1 {
		limit = expensesQueryLimit
	}

	expensesByStatus := make([][]financeTypes.Expense, len(statusToQuery))
	for resultIndex, currentStatus := range statusToQuery {
		queryGroup.Go(func() error {
			query := db.Query(&expensesByStatus[resultIndex]).OrderDesc()
			if limit > 0 {
				query.Limit(limit)
			}
			query.CompanyID.Equals(req.User.CompanyID).
				Status.Equals(currentStatus).
				Updated.GreaterThan(updated)
			return query.Exec()
		})
	}

	// On delta syncs only, collect the IDs of rows that left this tab so the client cache drops them.
	statusToRemove := expensesComplementStatuses(statusFilter)
	expensesToRemoveIDsGroups := make([][]int32, len(statusToRemove))
	if updated > 0 {
		for resultIndex, currentStatus := range statusToRemove {
			queryGroup.Go(func() error {
				idsToSave := &expensesToRemoveIDsGroups[resultIndex]

				query := db.Query(&[]financeTypes.Expense{})
				query.Select(query.ID)
				query.CompanyID.Equals(req.User.CompanyID).
					Status.Equals(currentStatus).
					Updated.GreaterThan(updated)

				return query.ExecScan(func(record *financeTypes.Expense) bool {
					*idsToSave = append(*idsToSave, record.ID)
					// Only the ID is needed; skip storing the decoded row.
					return true
				})
			})
		}
	}

	if err := queryGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener los gastos:", err)
	}

	records := []financeTypes.Expense{}
	for _, expensesByCurrentStatus := range expensesByStatus {
		records = append(records, expensesByCurrentStatus...)
	}

	// Todos merges two status partitions, so cap the combined set to the most-recently
	// updated `limit` rows (each partition is already capped, but their sum is not).
	if limit > 0 && int32(len(records)) > limit {
		sort.Slice(records, func(i, j int) bool { return records[i].Updated > records[j].Updated })
		records = records[:limit]
	}

	expensesToRemoveIDs := []int32{}
	for _, idsToRemove := range expensesToRemoveIDsGroups {
		expensesToRemoveIDs = append(expensesToRemoveIDs, idsToRemove...)
	}

	response := map[string]any{
		"records":             &records,
		"records_IDsToRemove": &expensesToRemoveIDs,
	}
	return req.MakeResponse(&response)
}

func PostExpenses(req *core.HandlerArgs) core.HandlerResponse {
	body := financeTypes.Expense{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}

	if err := validateExpenseCommon(body.Amount, body.CurrencyType, body.CategoryID); err != nil {
		return req.MakeErr(err)
	}

	nowTime := core.SUnixTime()
	body.CompanyID = req.User.CompanyID
	body.Status = core.If(body.Status == 0, int8(1), body.Status)
	body.Updated = nowTime
	body.UpdatedBy = req.User.ID

	records := &[]financeTypes.Expense{body}
	var err error
	if body.ID <= 0 {
		// New record: ORM assigns the autoincrement ID in-place.
		body.Created = nowTime
		body.CreatedBy = req.User.ID
		(*records)[0] = body
		err = db.Insert(records)
	} else {
		// Lock rule (server-authoritative, never trust the client): a fully-paid expense
		// (Status == 2) cannot be edited. Load the current Status to enforce it.
		existing := []financeTypes.Expense{}
		eq := db.Query(&existing)
		eq.Select(eq.Status).CompanyID.Equals(req.User.CompanyID).ID.Equals(body.ID)
		if err := eq.Exec(); err != nil {
			return req.MakeErr("Error al obtener el gasto:", err)
		}
		if len(existing) == 0 {
			return req.MakeErr("No se encontró el gasto.")
		}
		if existing[0].Status == 2 {
			return req.MakeErr("No se puede editar un gasto que ya fue pagado.")
		}
		// Keep payment state server-authoritative: write back the stored Status (set only by
		// PostExpensePayment) instead of the client's. Status shares a composite view index with
		// Updated, so it must be written together — only PaidAmount/Created/CreatedBy are excluded.
		(*records)[0].Status = existing[0].Status
		q := db.Table[financeTypes.Expense]()
		err = db.UpdateExclude(records, q.PaidAmount, q.Created, q.CreatedBy)
	}
	if err != nil {
		return req.MakeErr("Error al guardar el gasto:", err)
	}
	return req.MakeResponse((*records)[0])
}

// --- GET / POST: recurring schedules ----------------------------------------------

func GetExpensesScheduled(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt("updated")

	records := []financeTypes.ExpenseScheduled{}

	// Initial load: active only. Delta: also fetch Status=0 so the client can evict.
	status := []int8{1}
	if updated > 0 {
		status = append(status, 0)
	}

	for _, statusCode := range status {
		query := db.Query(&records)
		query.CompanyID.Equals(req.User.CompanyID).
			Status.Equals(statusCode).
			Updated.GreaterThan(updated) // updated=0 on initial → matches all rows

		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener los gastos programados:", err)
		}
	}

	return core.MakeResponse(req, &records)
}

func PostExpensesScheduled(req *core.HandlerArgs) core.HandlerResponse {
	body := financeTypes.ExpenseScheduled{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}

	if err := validateExpenseCommon(body.Amount, body.CurrencyType, body.CategoryID); err != nil {
		return req.MakeErr(err)
	}
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

	records := &[]financeTypes.ExpenseScheduled{body}
	var err error
	if body.ID <= 0 {
		body.Created = nowTime
		body.CreatedBy = req.User.ID
		(*records)[0] = body
		err = db.Insert(records)
	} else {
		q := db.Table[financeTypes.ExpenseScheduled]()
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
func schedulePeriodDates(schedule *financeTypes.ExpenseScheduled, end int16) []int16 {
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

	// 1. Load the schedule (multi-tenant scoped).
	schedules := []financeTypes.ExpenseScheduled{}
	sq := db.Query(&schedules)
	sq.Select().CompanyID.Equals(req.User.CompanyID).ID.Equals(scheduleID)
	if err := sq.Exec(); err != nil {
		return req.MakeErr("Error al obtener el gasto programado:", err)
	}
	if len(schedules) == 0 {
		return req.MakeErr("No se encontró el gasto programado.")
	}
	schedule := schedules[0]

	// 2. Load existing periods for this schedule (local index on ExpenseScheduledID).
	periods := []financeTypes.Expense{}
	pq := db.Query(&periods)
	pq.Select().CompanyID.Equals(req.User.CompanyID).ExpenseScheduledID.Equals(scheduleID)
	if err := pq.Exec(); err != nil {
		return req.MakeErr("Error al obtener los periodos del gasto:", err)
	}

	existing := core.SliceSet[int16]{}
	for _, p := range periods {
		existing.Add(p.PeriodDate)
	}

	// 3. Materialize any missing period up to today (deduped by PeriodDate).
	nowTime := core.SUnixTime()
	today := req.EffectiveFechaUnix()
	newPeriods := []financeTypes.Expense{}
	for _, periodDate := range schedulePeriodDates(&schedule, today) {
		if existing.Include(periodDate) {
			continue
		}
		newPeriods = append(newPeriods, financeTypes.Expense{
			CompanyID:          req.User.CompanyID,
			ExpenseScheduledID: schedule.ID,
			PeriodDate:         periodDate,
			Name:               schedule.Name,
			Description:        schedule.Description,
			CategoryID:         schedule.CategoryID,
			SupplierID:         schedule.SupplierID,
			CurrencyType:       schedule.CurrencyType,
			Date:               periodDate,
			DueDate:            periodDate,
			Amount:             schedule.Amount,
			Status:             1,
			Updated:            nowTime,
			UpdatedBy:          req.User.ID,
			Created:            nowTime,
			CreatedBy:          req.User.ID,
		})
	}

	if len(newPeriods) > 0 {
		if err := db.Insert(&newPeriods); err != nil {
			return req.MakeErr("Error al generar los periodos del gasto:", err)
		}
		periods = append(periods, newPeriods...)
	}

	response := map[string]any{"Periods": periods}
	return core.MakeResponse(req, &response)
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

	// 1. Load the expense being paid.
	expenses := []financeTypes.Expense{}
	eq := db.Query(&expenses)
	eq.Select().CompanyID.Equals(req.User.CompanyID).ID.Equals(body.ExpenseID)
	if err := eq.Exec(); err != nil {
		return req.MakeErr("Error al obtener el gasto:", err)
	}
	if len(expenses) == 0 {
		return req.MakeErr("No se encontró el gasto.")
	}
	expense := expenses[0]

	// Reject a payment larger than the outstanding balance (server-authoritative). This also
	// blocks any payment on an already fully-paid expense, whose pending balance is 0.
	pendingAmount := expense.Amount - expense.PaidAmount
	if body.Amount > pendingAmount {
		return req.MakeErr("El monto del pago no puede ser mayor al monto pendiente.")
	}

	// 2. Load the source cash bank and enforce a matching currency.
	cashBank, err := GetCaja(req.User.CompanyID, body.CashBankID)
	if err != nil {
		return req.MakeErr(err)
	}
	if cashBank.CurrencyType != expense.CurrencyType {
		return req.MakeErr("La moneda de la caja no coincide con la del gasto.")
	}

	// 3. The movement is an outflow, so its amount is negative. We don't validate the client's
	//    expected balance; we only reject a payment that would drive the cash-bank balance
	//    negative. The resulting balance is computed server-side by ApplyCajaMovimientos.
	movementAmount := -body.Amount
	if cashBank.CurrentAmount+movementAmount < 0 {
		return req.MakeErr("El saldo de la caja no puede quedar negativo.")
	}

	// 4. Record the outflow movement (auto-updates the cash-bank balance). FinalAmount is left
	//    at 0 so ApplyCajaMovimientos computes it from the current balance authoritatively.
	movement := financeTypes.InternalCashMovement{
		CashBankID:  body.CashBankID,
		DocumentID:  int64(expense.ID),
		ReferenceID: expense.ExpenseScheduledID,
		Date:        body.Date,
		Type:        movementTypeExpensePayment,
		Amount:      movementAmount,
		FinalAmount: 0,
	}
	if err := ApplyCashBankMovement(req, []financeTypes.InternalCashMovement{movement}); err != nil {
		return req.MakeErr(err)
	}

	// 5. Recompute PaidAmount = Σ |movement.Amount| over this expense's movements.
	movements := []financeTypes.CashBankMovement{}
	mq := db.Query(&movements)
	mq.Select().CompanyID.Equals(req.User.CompanyID).DocumentID.Equals(int64(expense.ID))
	if err := mq.Exec(); err != nil {
		return req.MakeErr("Error al recalcular el monto pagado:", err)
	}
	var paidAmount int32
	for _, m := range movements {
		paidAmount += core.If(m.Amount < 0, -m.Amount, m.Amount)
	}

	// 6. Resolve the payment lifecycle on Status (honoring the "Is Fully Paid" write-off flag):
	//    2 = fully paid, 1 = pending (unpaid/partial). This moves the row between status tabs.
	expense.PaidAmount = paidAmount
	if body.IsFullyPaid || paidAmount >= expense.Amount {
		expense.Status = 2
	} else {
		expense.Status = 1
	}
	expense.Updated = core.SUnixTime()
	expense.UpdatedBy = req.User.ID

	q := db.Table[financeTypes.Expense]()
	if err := db.Update(&[]financeTypes.Expense{expense}, q.PaidAmount, q.Status, q.Updated, q.UpdatedBy); err != nil {
		return req.MakeErr("Error al actualizar el gasto:", err)
	}

	return req.MakeResponse(expense)
}
