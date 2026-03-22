package types

import "app/db"

type CronActionParams struct {
	Param1 int64   `json:",omitempty" cbor:"1,keyasint,omitempty"`
	Param2 int64   `json:",omitempty" cbor:"2,keyasint,omitempty"`
	Param3 float64 `json:",omitempty" cbor:"3,keyasint,omitempty"`
	Param4 float64 `json:",omitempty" cbor:"4,keyasint,omitempty"`
	Param5 string  `json:",omitempty" cbor:"5,keyasint,omitempty"`
	Param6 string  `json:",omitempty" cbor:"6,keyasint,omitempty"`
}

type CronAction struct {
	db.TableStruct[CronActionTable, CronAction]
	ID               int64            `json:",omitempty"`
	UnixMinutesFrame int32            `json:",omitempty"`
	CompanyID        int32            `json:",omitempty"`
	ActionID         int16            `json:",omitempty"`
	Updated          int32            `json:",omitempty"`
	Status          int8            `json:",omitempty"`
	InvocationCount          int16            `json:",omitempty"`
	Params           CronActionParams `json:",omitempty"`
}

type CronActionTable struct {
	db.TableStruct[CronActionTable, CronAction]
	UnixMinutesFrame db.Col[CronActionTable, int32]
	CompanyID        db.Col[CronActionTable, int32]
	ID               db.Col[CronActionTable, int64]
	ActionID        db.Col[CronActionTable, int16]
	Params           db.Col[CronActionTable, CronActionParams]
	Updated          db.Col[CronActionTable, int32]
	Status        db.Col[CronActionTable, int8]
	InvocationCount        db.Col[CronActionTable, int16]
}

func (e CronActionTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name: "cron_actions",
		Keys: []db.Coln{e.UnixMinutesFrame, e.ID},
	}
}
