package types

import "app/db"

type CronActionParams struct {
	Param1 int64   `json:",omitempty" cbor:"1,keyasint"`
	Param2 int64   `json:",omitempty" cbor:"2,keyasint"`
	Param3 float64 `json:",omitempty" cbor:"3,keyasint"`
	Param4 float64 `json:",omitempty" cbor:"4,keyasint"`
	Param5 string  `json:",omitempty" cbor:"5,keyasint"`
	Param6 string  `json:",omitempty" cbor:"6,keyasint"`
}

type CronAction struct {
	db.TableStruct[CronActionTable, CronAction]
	ID               int64            `json:",omitempty"`
	UnixMinutesFrame int32            `json:",omitempty"`
	CompanyID        int32            `json:",omitempty"`
	ActionID         int16            `json:",omitempty"`
	Updated          int32            `json:",omitempty"`
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
}

func (e CronActionTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name: "cron_actions",
		Keys: []db.Coln{e.UnixMinutesFrame, e.ID},
	}
}
