package core

import (
	"app/db"
	"math"
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
			actionHandlersInfo = append(actionHandlersInfo, ActionHandlerInfo{ e.ID, e.Name })
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

func ScheduleCronAction(action CronAction, frameLengthInMinutes int8) {
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
	fiveMinuteFrameLength := int64(5 * 60)

	// Align to the next boundary of the requested interval, not to the nearest 5-minute slot.
	// Example: with a 20-minute interval we jump to the next 20-minute boundary and then
	// convert it back to 5-minute units, so repeated scheduling keeps the 20-minute cadence.
	nextAlignedUnixSeconds := ((currentUnixSeconds / frameLengthInSeconds) + 1) * frameLengthInSeconds
	action.UnixMinutesFrame = int32(nextAlignedUnixSeconds / fiveMinuteFrameLength)
	action.Updated = SUnixTime()

	existingActions := []CronAction{}
	existingActionQuery := db.Query(&existingActions)
	existingActionQuery.Select(existingActionQuery.ID, existingActionQuery.Status).
		UnixMinutesFrame.Equals(action.UnixMinutesFrame).
		ID.Equals(action.ID)

	if err := existingActionQuery.Exec(); err != nil {
		panic(err)
	}
	if len(existingActions) > 0 && existingActions[0].Status == 0 {
		Log("ScheduleCronAction skipped existing pending action:", "company_id", action.CompanyID, "action_id", action.ActionID, "cron_id", action.ID)
		return
	}

	if err := db.InsertOne(action); err != nil {
		panic(err)
	}
}

var lastUnixMinutesFrame = int32(0)

func StartCronWatcher(){
	time.Sleep(10*time.Second)
	
	go func() {
		cronTick := time.NewTicker(time.Minute)

		// Run once on startup so already-due rows do not wait for the first ticker tick.
		runCronWatcherTick()

		for range cronTick.C {
			runCronWatcherTick()
		}
	}()
}

func runCronWatcherTick() {
	currentUnixMinutesFrame := int32(SUnix5Min())
	if currentUnixMinutesFrame == lastUnixMinutesFrame {
		return
	}

	firstFrameToProcess := currentUnixMinutesFrame - 12
	pendingActionsByID := map[int64][]CronAction{}
	
	pendingActionsGetted := []CronAction{}
	// Query the whole lookback window in one pass and keep one row per logical cron ID.
	pendingActionsQuery := db.Query(&pendingActionsGetted).
		UnixMinutesFrame.Between(firstFrameToProcess, currentUnixMinutesFrame).
		Status.Equals(0)

	if err := pendingActionsQuery.AllowFilter().Exec(); err != nil {
		Log("StartCronWatcher query error:", "from_frame", firstFrameToProcess, "error", err)
		return
	}

	for _, e := range pendingActionsGetted {
		pendingActionsByID[e .ID] = append(pendingActionsByID[e .ID], e)
	}

	for _, pendingActionsSameID := range pendingActionsByID {
		pendingAction := pendingActionsSameID[0]
		cronActionTable := db.Table[CronAction]()
		
		actionHandler, exists := actionHandlerMap[pendingAction.ActionID]
		if !exists {
			Log("StartCronWatcher missing handler:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID)
			continue
		}
		// Reuse the rows already loaded for this cron ID instead of querying them again.
		var markCronActionRowsAttempted_ = func(status int8) {
			for i := range pendingActionsSameID {
				e := &pendingActionsSameID[i]
				e.InvocationCount++
				e.Updated = SUnixTime()
				e.Status = status
			}
			
			if err := db.Update(
				&pendingActionsSameID,
				cronActionTable.Status,
				cronActionTable.InvocationCount,
				cronActionTable.Updated,
			); err != nil {
				Log("StartCronWatcher update rows error:", "cron_id", pendingAction.ID, "status", status, "error", err)
			}
		}

		// Pass the persisted ExecArgs payload directly so scheduling and execution use one shape.
		func() {
			defer func() {
				if recoveredValue := recover(); recoveredValue != nil {
					Log("StartCronWatcher panic executing action:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID, "panic", recoveredValue)
					markCronActionRowsAttempted_(0)
				}
			}()

			handlerResponse := actionHandler.Fn(&pendingAction.Params)
			if handlerResponse.Error != "" {
				Log("StartCronWatcher handler error:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID, "error", handlerResponse.Error)
				markCronActionRowsAttempted_(0)
				return
			}

			Log("StartCronWatcher action executed:", "cron_id", pendingAction.ID, "action_id", pendingAction.ActionID, "handler", actionHandler.Name)
			markCronActionRowsAttempted_(1)
		}()
	}

	// Advance the watermark only after the current frame window was processed.
	lastUnixMinutesFrame = currentUnixMinutesFrame
}
