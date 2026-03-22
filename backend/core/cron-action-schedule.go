package core

import (
	types "app/core/types"
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

func ScheduleCronAction(action types.CronAction, frameLengthInMinutes int8) {
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

	existingActions := []types.CronAction{}
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
