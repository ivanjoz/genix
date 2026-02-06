package types

import "app/db"

type SystemParameters struct {
	db.TableStruct[SystemParametersTable, SystemParameters]
	EmpresaID   int32   `json:"empresa_id,omitempty" db:"empresa_id"`
	Updated     int64   `json:"upd,omitempty" db:"updated"`
	UpdatedBy   int32   `json:"updated_by,omitempty" db:"updated_by"`
	ParameterId int32   `json:"parameter_id,omitempty" db:"parameter_id,pk"`
	ValueText   string  `json:"value_text,omitempty" db:"value_text"`
	ValueInts   []int32 `json:"value_ints,omitempty" db:"value_ints"`
	ValueExtra  string  `json:"value_extra,omitempty" db:"value_extra"`
}

type SystemParametersTable struct {
	db.TableStruct[SystemParametersTable, SystemParameters]
	EmpresaID   db.Col[SystemParametersTable, int32]
	Updated     db.Col[SystemParametersTable, int64]
	UpdatedBy   db.Col[SystemParametersTable, int32]
	ParameterId db.Col[SystemParametersTable, int32]
	ValueText   db.Col[SystemParametersTable, string]
	ValueInts   db.ColSlice[SystemParametersTable, int32]
	ValueExtra  db.Col[SystemParametersTable, string]
}

func (e SystemParametersTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "system_parameters",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ParameterId},
		Views: []db.View{
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
