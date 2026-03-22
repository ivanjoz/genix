package core

import (
	types "app/core/types"
	"app/db"
	"time"
)

type ActionHandler struct {
	ID   int16
	Name string
	Fn   func(args *ExecArgs) FuncResponse
}

var actionHandlerMap = map[int16]ActionHandler{}

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
	if frameLengthInMinutes == 0 {
		frameLengthInMinutes = 5
	}
	if frameLengthInMinutes%5 != 0 {
		panic("ScheduleCronAction: frameLengthInMinutes must be divisible by 5")
	}

	// Hash only the action payload so the schedule key stays stable for the same logical job.
	paramsHash := uint64(HashInt64(
		action.Params.Param1,
		action.Params.Param2,
		action.Params.Param3,
		action.Params.Param4,
		action.Params.Param5,
		action.Params.Param6,
	))

	// Reserve the high 16 bits for the action type and keep 48 hash bits for uniqueness.
	// This lets the executor recover the action namespace from the ID without extra columns.
	action.ID = (action.ID << 48) | int64(paramsHash&0x0000FFFFFFFFFFFF)

	currentUnixSeconds := time.Now().Unix()
	frameLengthInSeconds := int64(frameLengthInMinutes) * 60
	fiveMinuteFrameLength := int64(5 * 60)

	// Align to the next boundary of the requested interval, not to the nearest 5-minute slot.
	// Example: with a 20-minute interval we jump to the next 20-minute boundary and then
	// convert it back to 5-minute units, so repeated scheduling keeps the 20-minute cadence.
	nextAlignedUnixSeconds := ((currentUnixSeconds / frameLengthInSeconds) + 1) * frameLengthInSeconds
	action.UnixMinutesFrame = int32(nextAlignedUnixSeconds / fiveMinuteFrameLength)
	action.Updated = SUnixTime()

	if err := db.InsertOne(action); err != nil {
		panic(err)
	}
}
