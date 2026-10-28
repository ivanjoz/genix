package core

import "app/db"

type CronAction struct {
	db.TableStruct[CronActionTable, CronAction]
	ID               int64 `cb:"ID,1"`
	UnixMinutesFrame int32 `cb:"UnixMinutesFrame,2"`
	CompanyID        int32 `cb:"CompanyID,3"`
	ActionID         int16 `cb:"ActionID,4"`
	Updated          int32 `json:"upd" cb:"upd,5"`
	Status           int8  `json:"ss" cb:"ss,6"`
	InvocationCount  int16 `cb:"InvocationCount,7"`
	// Cadence in minutes, persisted so recurrence belongs to the row instead of to handler code:
	// the executor re-enqueues the next frame even when the handler panicked halfway through.
	// 0 means one-shot.
	FrameLengthMinutes int8 `json:"fl" cb:"fl,8"`
	// Best-effort lease. ClaimedAt is a SUnixTime stamp and ClaimedBy the token of the worker that
	// took it, so a concurrent worker can tell whose claim won the read-back.
	ClaimedAt int32    `json:"cat" cb:"cat,9"`
	ClaimedBy int32    `json:"cby" cb:"cby,10"`
	Params    ExecArgs `json:"" cb:"Params,11"`
	// What the handler reported through ExecArgs.AddMessage, persisted together with the final
	// status. Holds the last attempt only: every attempt starts from the row as loaded, and Params
	// carries no messages of its own, so a retry reports its own run instead of an accumulated log.
	// Pinned to a list because a slice column defaults to set<text>, which would sort the messages
	// alphabetically and destroy the order they were reported in.
	Messages []string `json:"messages" db:"messages,list" cb:"messages,12"`
}

type CronActionTable struct {
	db.TableStruct[CronActionTable, CronAction]
	UnixMinutesFrame   db.Col[*CronActionTable, int32]
	CompanyID          db.Col[*CronActionTable, int32]
	ID                 db.Col[*CronActionTable, int64]
	ActionID           db.Col[*CronActionTable, int16]
	Params             db.Col[*CronActionTable, ExecArgs]
	Updated            db.Col[*CronActionTable, int32]
	Status             db.Col[*CronActionTable, int8]
	InvocationCount    db.Col[*CronActionTable, int16]
	FrameLengthMinutes db.Col[*CronActionTable, int8]
	ClaimedAt          db.Col[*CronActionTable, int32]
	ClaimedBy          db.Col[*CronActionTable, int32]
	Messages           db.ColSlice[*CronActionTable, string]
}

func (cronActionTable CronActionTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:   10,
		Name: "cron_actions",
		Keys: db.Cols(cronActionTable.UnixMinutesFrame, cronActionTable.ID),
	}
}
