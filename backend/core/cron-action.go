package core

import "app/db"

type CronAction struct {
	db.TableStruct[CronActionTable, CronAction]
	ID               int64 `json:",omitempty"`
	UnixMinutesFrame int32 `json:",omitempty"`
	CompanyID        int32 `json:",omitempty"`
	ActionID         int16 `json:",omitempty"`
	Updated          int32 `json:"upd,omitempty"`
	Status           int8  `json:"ss,omitempty"`
	InvocationCount  int16 `json:",omitempty"`
	// Cadence in minutes, persisted so recurrence belongs to the row instead of to handler code:
	// the executor re-enqueues the next frame even when the handler panicked halfway through.
	// 0 means one-shot.
	FrameLengthMinutes int8 `json:"fl,omitempty"`
	// Best-effort lease. ClaimedAt is a SUnixTime stamp and ClaimedBy the token of the worker that
	// took it, so a concurrent worker can tell whose claim won the read-back.
	ClaimedAt int32    `json:"cat,omitempty"`
	ClaimedBy int32    `json:"cby,omitempty"`
	Params    ExecArgs `json:""`
	// What the handler reported through ExecArgs.AddMessage, persisted together with the final
	// status. Holds the last attempt only: every attempt starts from the row as loaded, and Params
	// carries no messages of its own, so a retry reports its own run instead of an accumulated log.
	// Pinned to a list because a slice column defaults to set<text>, which would sort the messages
	// alphabetically and destroy the order they were reported in.
	Messages []string `json:"messages,omitempty" db:"messages,list"`
}

type CronActionTable struct {
	db.TableStruct[CronActionTable, CronAction]
	UnixMinutesFrame   db.Col[CronActionTable, int32]
	CompanyID          db.Col[CronActionTable, int32]
	ID                 db.Col[CronActionTable, int64]
	ActionID           db.Col[CronActionTable, int16]
	Params             db.Col[CronActionTable, ExecArgs]
	Updated            db.Col[CronActionTable, int32]
	Status             db.Col[CronActionTable, int8]
	InvocationCount    db.Col[CronActionTable, int16]
	FrameLengthMinutes db.Col[CronActionTable, int8]
	ClaimedAt          db.Col[CronActionTable, int32]
	ClaimedBy          db.Col[CronActionTable, int32]
	Messages           db.ColSlice[CronActionTable, string]
}

func (cronActionTable CronActionTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:   10,
		Name: "cron_actions",
		Keys: db.Cols(cronActionTable.UnixMinutesFrame, cronActionTable.ID),
	}
}
