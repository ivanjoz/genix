package types

import "app/db"

type SystemParameters struct {
	db.TableStruct[SystemParametersTable, SystemParameters]
	EmpresaID   int32   `json:",omitempty" db:"empresa_id"`
	Updated     int64   `json:"upd," db:"updated"`
	UpdatedBy   int32   `json:",omitempty" db:"updated_by"`
	ParameterID int32   `json:",omitempty" db:"parameter_id,pk"`
	ValueText   string  `json:",omitempty" db:"value_text"`
	ValueInts   []int32 `json:",omitempty" db:"value_ints"`
	ValueExtra  string  `json:",omitempty" db:"value_extra"`
}

type SystemParametersTable struct {
	db.TableStruct[SystemParametersTable, SystemParameters]
	EmpresaID   db.Col[SystemParametersTable, int32]
	Updated     db.Col[SystemParametersTable, int64]
	UpdatedBy   db.Col[SystemParametersTable, int32]
	ParameterID db.Col[SystemParametersTable, int32]
	ValueText   db.Col[SystemParametersTable, string]
	ValueInts   db.ColSlice[SystemParametersTable, int32]
	ValueExtra  db.Col[SystemParametersTable, string]
}

func (e SystemParametersTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "system_parameters",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ParameterID},
		Views: []db.View{
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
