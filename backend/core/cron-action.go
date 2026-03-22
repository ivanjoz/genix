package core

import "app/db"

type CronAction struct {
	db.TableStruct[CronActionTable, CronAction]
	ID               int64    `json:",omitempty"`
	UnixMinutesFrame int32    `json:",omitempty"`
	CompanyID        int32    `json:",omitempty"`
	ActionID         int16    `json:",omitempty"`
	Updated          int32    `json:"upd,omitempty"` 
	Status           int8     `json:"ss,omitempty"`
	InvocationCount  int16    `json:",omitempty"`
	Params           ExecArgs `json:",omitempty"`
}

type CronActionTable struct {
	db.TableStruct[CronActionTable, CronAction]
	UnixMinutesFrame db.Col[CronActionTable, int32]
	CompanyID        db.Col[CronActionTable, int32]
	ID               db.Col[CronActionTable, int64]
	ActionID         db.Col[CronActionTable, int16]
	Params           db.Col[CronActionTable, ExecArgs]
	Updated          db.Col[CronActionTable, int32]
	Status           db.Col[CronActionTable, int8]
	InvocationCount  db.Col[CronActionTable, int16]
}

func (cronActionTable CronActionTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name: "cron_actions",
		Keys: []db.Coln{cronActionTable.UnixMinutesFrame, cronActionTable.ID},
	}
}
