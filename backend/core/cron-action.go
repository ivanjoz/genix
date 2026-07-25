package core

import "github.com/ivanjoz/genix-orm/scylla"

type CronAction struct {
	scylla.TableStruct[CronActionTable, CronAction]
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
	scylla.TableStruct[CronActionTable, CronAction]
	UnixMinutesFrame scylla.Col[CronActionTable, int32]
	CompanyID        scylla.Col[CronActionTable, int32]
	ID               scylla.Col[CronActionTable, int64]
	ActionID         scylla.Col[CronActionTable, int16]
	Params           scylla.Col[CronActionTable, ExecArgs]
	Updated          scylla.Col[CronActionTable, int32]
	Status           scylla.Col[CronActionTable, int8]
	InvocationCount  scylla.Col[CronActionTable, int16]
}

func (cronActionTable CronActionTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:                 "cron_actions",
		Keys:                 scylla.Cols(cronActionTable.UnixMinutesFrame, cronActionTable.ID),
		DisableUpdateCounter: true,
	}
}
