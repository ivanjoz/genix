package core

import (
	"app/db"
	"math"
	mrand "math/rand"
	"time"
)

type ActionHandler struct {
	ID   int16
	Name string
	Fn   func(args *ExecArgs) FuncResponse
}

type ActionHandlerInfo struct {
	ID   int16
	Name string
}

var actionHandlerMap = map[int16]ActionHandler{}

func GetRegisteredActionHandlers(actionIDs ...int16) []ActionHandlerInfo {
	actionHandlersInfo := []ActionHandlerInfo{}

	for _, actionID := range actionIDs {
		if e, ok := actionHandlerMap[actionID]; ok {
			actionHandlersInfo = append(actionHandlersInfo, ActionHandlerInfo{e.ID, e.Name})
		}
	}

	return actionHandlersInfo
}

func RegisterActionHandler(id int16, name string, handler func(args *ExecArgs) FuncResponse) {
	if id <= 0 {
		panic("RegisterActionHandler: id must be between 1 and 32767")
	}
	if name == "" {
		panic("RegisterActionHandler: name is required")
	}
	if handler == nil {
		panic("RegisterActionHandler: handler is required")
	}

	if existingHandler, exists := actionHandlerMap[id]; exists {
		panic(Concat(" ", "RegisterActionHandler: duplicated id", id, "existing:", existingHandler.Name, "new:", name))
	}

	// Keep the registration keyed by the same 16-bit action namespace used in the cron row ID.
	// This guarantees the executor can resolve the stored action prefix directly to the handler.
	actionHandlerMap[id] = ActionHandler{ID: id, Name: name, Fn: handler}
}

const fiveMinuteFrameLength = int64(5 * 60)

// Persisted Status values of a cron row.
const (
	statusCronActionPending   = int8(0)
	statusCronActionDone      = int8(1)
	statusCronActionAbandoned = int8(2)
)

// SUnixTime advances in two-second units, so a sixty-second lease is thirty of them.
const cronClaimLeaseSUnits = int32(30)

// Attempts a row gets before it is abandoned. Without a cap, a payload that always panics keeps
// consuming a slot on every tick for the whole lookback window and then vanishes silently at
// Status 0; at statusCronActionAbandoned it stops being retried but stays auditable.
const cronMaxInvocations = int16(10)

// ScheduleCronAction enqueues a one-shot action. frameLengthInMinutes only says where the row
// lands: the action runs once and is not re-enqueued. Use ScheduleRecurringCronAction for a job
// that has to keep a cadence.
func ScheduleCronAction(action CronAction, frameLengthInMinutes int8) {
	scheduleCronActionRow(action, frameLengthInMinutes, false)
}

// ScheduleRecurringCronAction enqueues an action that repeats every frameLengthInMinutes. The
// cadence is persisted on the row, so the executor enqueues each following frame by itself and the
// handler never has to reschedule itself — which is what keeps a panicking handler from breaking
// the chain.
func ScheduleRecurringCronAction(action CronAction, frameLengthInMinutes int8) {
	scheduleCronActionRow(action, frameLengthInMinutes, true)
}

func scheduleCronActionRow(action CronAction, frameLengthInMinutes int8, isRecurring bool) {
	if action.ActionID == 0 || action.CompanyID == 0 {
		panic("ActionID and CompanyID needed for: ScheduleCronAction")
	}
	if action.CompanyID > math.MaxUint16 {
		panic("ScheduleCronAction: CompanyID must fit in 16 bits")
	}
	if frameLengthInMinutes == 0 {
		frameLengthInMinutes = 5
	}
	if frameLengthInMinutes%5 != 0 {
		panic("ScheduleCronAction: frameLengthInMinutes must be divisible by 5")
	}

	// Compose the row ID as company namespace + action namespace + payload hash.
	// This keeps duplicate detection stable for the same logical cron job.
	paramsHash := uint32(HashInt32(
		action.Params.Param1,
		action.Params.Param2,
		action.Params.Param3,
		action.Params.Param4,
		action.Params.Param5,
		action.Params.Param6,
	))
	action.ID = int64(uint64(uint16(action.CompanyID))<<48 | uint64(uint16(action.ActionID))<<32 | uint64(paramsHash))

	currentUnixSeconds := time.Now().Unix()
	frameLengthInSeconds := int64(frameLengthInMinutes) * 60
	// Align to the next boundary of the requested interval, not to the nearest 5-minute slot.
	// Example: with a 20-minute interval we jump to the next 20-minute boundary and then
	// convert it back to 5-minute units, so repeated scheduling keeps the 20-minute cadence.
	nextAlignedUnixSeconds := ((currentUnixSeconds / frameLengthInSeconds) + 1) * frameLengthInSeconds
	action.UnixMinutesFrame = int32(nextAlignedUnixSeconds / fiveMinuteFrameLength)
	action.Updated = SUnixTime()

	// Persist the cadence only for a recurring job: a zero here is what tells the executor this row
	// must not be re-enqueued. Reset the per-attempt state here rather than at every call site,
	// because rescheduleCronAction passes back a copy of the row that just ran, which still carries
	// its attempt count and its spent lease.
	action.FrameLengthMinutes = If(isRecurring, frameLengthInMinutes, int8(0))
	action.Status = statusCronActionPending
	action.InvocationCount = 0
	action.ClaimedAt = 0
	action.ClaimedBy = 0
	// A fresh occurrence reports its own run: never inherit the messages of the one that spawned it.
	action.Messages = nil
	action.Params.ProcessMessages = nil

	existingActions := []CronAction{}
	existingActionQuery := db.Query(&existingActions)
	existingActionQuery.Select(existingActionQuery.ID, existingActionQuery.Status).
		UnixMinutesFrame.Equals(action.UnixMinutesFrame).
		ID.Equals(action.ID)

	if err := existingActionQuery.Exec(); err != nil {
		panic(err)
	}
	if len(existingActions) > 0 && existingActions[0].Status == statusCronActionPending {
		Log("ScheduleCronAction skipped existing pending action:", "company_id", action.CompanyID, "action_id", action.ActionID, "cron_id", action.ID)
		return
	}

	if err := db.InsertOne(action); err != nil {
		panic(err)
	}
}

var lastUnixMinutesFrame = int32(0)

func StartCronWatcher() {
	go func() {
		// Inside the goroutine, not before it: the caller is main, and blocking it here delayed
		// the HTTP listener by ten seconds on every boot.
		time.Sleep(10 * time.Second)

		cronTick := time.NewTicker(time.Minute)

		// Run once on startup so already-due rows do not wait for the first ticker tick.
		runCronWatcherTick()

		for range cronTick.C {
			runCronWatcherTick()
		}
	}()
}

// runCronWatcherTick is the VPS ticker entry point. The ticker fires every minute while frames
// last five, so the watermark is what keeps the same frame from being processed five times.
func runCronWatcherTick() {
	currentUnixMinutesFrame := int32(time.Now().Unix() / fiveMinuteFrameLength)
	if currentUnixMinutesFrame == lastUnixMinutesFrame {
		return
	}

	RunPendingCronActions()

	// Advance the watermark only after the current frame window was processed.
	lastUnixMinutesFrame = currentUnixMinutesFrame
}

// claimCronActionRows takes a best-effort lease on a cron group so two workers do not run the same
// action at once. It is a lease and not a mutex on purpose: the scylla driver exposes no
// lightweight transaction (QueryExec drops the [applied] row a CAS needs), so the write-then-read
// back below narrows the double-execution window from the whole handler duration to a single round
// trip. Handlers still have to be idempotent.
func claimCronActionRows(rows []CronAction, cronActionTable *CronActionTable, claimToken int32) bool {
	currentSUnix := SUnixTime()
	leadRow := rows[0]

	// A lease still inside its window means another worker is running this action right now.
	// Nothing expires it explicitly: once cronClaimLeaseSUnits pass the row is claimable again, so
	// a process that dies mid-handler blocks the action for a minute and no longer, and no sweeper
	// is needed to clean up after it.
	if leadRow.ClaimedAt > 0 && currentSUnix-leadRow.ClaimedAt < cronClaimLeaseSUnits {
		Log("claimCronActionRows skipped leased action:", "cron_id", leadRow.ID, "claimed_at", leadRow.ClaimedAt)
		return false
	}

	for i := range rows {
		rows[i].ClaimedAt = currentSUnix
		rows[i].ClaimedBy = claimToken
	}

	if err := db.Update(&rows, cronActionTable.ClaimedAt, cronActionTable.ClaimedBy); err != nil {
		Log("claimCronActionRows update error:", "cron_id", leadRow.ID, "error", err)
		return false
	}

	// Read the token back: scylla resolves concurrent writes last-write-wins, so two workers that
	// both wrote read the same winner and only one of them recognises itself here.
	claimCheckRows := []CronAction{}
	claimCheckQuery := db.Query(&claimCheckRows)
	claimCheckQuery.Select(claimCheckQuery.ClaimedBy).
		UnixMinutesFrame.Equals(leadRow.UnixMinutesFrame).
		ID.Equals(leadRow.ID)

	if err := claimCheckQuery.Exec(); err != nil {
		Log("claimCronActionRows read back error:", "cron_id", leadRow.ID, "error", err)
		return false
	}
	if len(claimCheckRows) == 0 || claimCheckRows[0].ClaimedBy != claimToken {
		Log("claimCronActionRows lost the claim:", "cron_id", leadRow.ID)
		return false
	}

	return true
}

// rescheduleCronAction enqueues the next frame of a recurring row. It runs after the attempt and
// regardless of its outcome, which is the whole point: the cadence no longer depends on the
// handler reaching a self-rescheduling tail call that a panic would skip. ScheduleCronAction
// panics on a database error, so the recover keeps a lost link from aborting the rest of the queue,
// and its in-frame dedup keeps repeated retries from piling up duplicate rows.
func rescheduleCronAction(action CronAction, params ExecArgs) {
	if action.FrameLengthMinutes <= 0 {
		return
	}

	defer func() {
		if recoveredValue := recover(); recoveredValue != nil {
			Log("rescheduleCronAction error:", "cron_id", action.ID, "action_id", action.ActionID, "panic", recoveredValue)
		}
	}()

	// Reschedule from the payload as it was loaded, not as the handler left it: the handler
	// received a pointer to Params and any mutation would move the ID hash to a different row.
	action.Params = params
	ScheduleRecurringCronAction(action, action.FrameLengthMinutes)
}

// RunPendingCronActions executes every pending row inside the lookback window and returns how
// many ran successfully. It carries no watermark of its own: the caller owns the cadence, which
// is what lets the Lambda EventBridge tick reuse it across cold containers.
func RunPendingCronActions() int {
	currentUnixMinutesFrame := int32(time.Now().Unix() / fiveMinuteFrameLength)
	executedActionsCount := 0

	// One token per invocation rather than per process: a warm Lambda container serving two
	// overlapping invocations would share a process-level token and both would read their own
	// claim back as a win. Zero is reserved for "unclaimed", so it can never be the token.
	claimToken := mrand.Int31()
	if claimToken == 0 {
		claimToken = 1
	}

	// A 60-minute lookback, independent of the caller's cadence: it exists so actions that failed
	// and stayed at Status 0 get retried on the following ticks.
	firstFrameToProcess := currentUnixMinutesFrame - 12
	pendingActionsByID := map[int64][]CronAction{}

	pendingActionsGetted := []CronAction{}
	// Query the whole lookback window in one pass and keep one row per logical cron ID.
	pendingActionsQuery := db.Query(&pendingActionsGetted).
		UnixMinutesFrame.Between(firstFrameToProcess, currentUnixMinutesFrame).
		Status.Equals(0)

	if err := pendingActionsQuery.AllowFilter().Exec(); err != nil {
		Log("RunPendingCronActions query error:", "from_frame", firstFrameToProcess, "error", err)
		return executedActionsCount
	}

	for _, e := range pendingActionsGetted {
		pendingActionsByID[e.ID] = append(pendingActionsByID[e.ID], e)
	}

	for _, pendingActionsSameID := range pendingActionsByID {
		pendingAction := pendingActionsSameID[0]
		cronActionTable := db.TableOf[CronAction]()

		actionHandler, exists := actionHandlerMap[pendingAction.ActionID]
		if !exists {
			Log("RunPendingCronActions missing handler:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID)
			continue
		}

		if !claimCronActionRows(pendingActionsSameID, cronActionTable, claimToken) {
			continue
		}

		// Snapshot the payload before the handler can touch it, so the reschedule below lands on
		// the same logical row this one came from.
		scheduledParams := pendingAction.Params

		// Reuse the rows already loaded for this cron ID instead of querying them again.
		var markCronActionRowsAttempted_ = func(status int8) {
			for i := range pendingActionsSameID {
				e := &pendingActionsSameID[i]
				e.InvocationCount++
				e.Updated = SUnixTime()
				e.Status = status
				// Whatever the handler reported through AddMessage lands here. It is read off the
				// payload the handler was given, so a run that panicked halfway still persists the
				// messages it had already added before dying.
				e.Messages = pendingAction.Params.ProcessMessages
				// Release the lease explicitly so a failed row is retryable on the very next tick
				// instead of sitting out the rest of its minute.
				e.ClaimedAt = 0
				e.ClaimedBy = 0
			}

			if err := db.Update(
				&pendingActionsSameID,
				cronActionTable.Status,
				cronActionTable.InvocationCount,
				cronActionTable.Updated,
				cronActionTable.ClaimedAt,
				cronActionTable.ClaimedBy,
				cronActionTable.Messages,
			); err != nil {
				Log("RunPendingCronActions update rows error:", "cron_id", pendingAction.ID, "status", status, "error", err)
			}
		}

		// Keep the row pending so the lookback window retries it, unless it already burned every
		// attempt it was allowed.
		var markCronActionRowsFailed_ = func() {
			if pendingAction.InvocationCount+1 >= cronMaxInvocations {
				Log("RunPendingCronActions action abandoned:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID, "invocations", pendingAction.InvocationCount+1)
				markCronActionRowsAttempted_(statusCronActionAbandoned)
				return
			}
			markCronActionRowsAttempted_(statusCronActionPending)
		}

		// Pass the persisted ExecArgs payload directly so scheduling and execution use one shape.
		func() {
			defer func() {
				if recoveredValue := recover(); recoveredValue != nil {
					Log("RunPendingCronActions panic executing action:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID, "panic", recoveredValue)
					// Record why it died next to the handler's own messages: the row is the only
					// place a VPS operator can read this without digging through CloudWatch.
					pendingAction.Params.AddMessage(Concat(" ", "panic:", recoveredValue))
					markCronActionRowsFailed_()
				}
			}()

			handlerResponse := actionHandler.Fn(&pendingAction.Params)
			if handlerResponse.Message != "" {
				pendingAction.Params.AddMessage(handlerResponse.Message)
			}

			if handlerResponse.Error != "" {
				Log("RunPendingCronActions handler error:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID, "error", handlerResponse.Error)
				pendingAction.Params.AddMessage(Concat(" ", "error:", handlerResponse.Error))
				markCronActionRowsFailed_()
				return
			}

			Log("RunPendingCronActions action executed:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID, "handler", actionHandler.Name)
			markCronActionRowsAttempted_(statusCronActionDone)
			executedActionsCount++
		}()

		// Outside the closure so it runs on every outcome, including the recovered panic.
		rescheduleCronAction(pendingAction, scheduledParams)
	}

	return executedActionsCount
}
