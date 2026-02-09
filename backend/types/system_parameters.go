package types

import "app/db"

type SystemParameters struct {
	db.TableStruct[SystemParametersTable, SystemParameters]
	ID        int32   `json:",omitempty" db:"parameter_id,pk"`
	ValueText string  `json:",omitempty" db:"value_text"`
	ValueInts []int32 `json:",omitempty" db:"value_ints"`
	Value     int32   `json:",omitempty" db:"value"`
	EmpresaID int32   `json:",omitempty" db:"empresa_id"`
	Updated   int64   `json:"upd," db:"updated"`
	UpdatedBy int32   `json:",omitempty" db:"updated_by"`
}

type SystemParametersTable struct {
	db.TableStruct[SystemParametersTable, SystemParameters]
	EmpresaID db.Col[SystemParametersTable, int32]
	ID        db.Col[SystemParametersTable, int32]
	ValueText db.Col[SystemParametersTable, string]
	ValueInts db.ColSlice[SystemParametersTable, int32]
	Value     db.Col[SystemParametersTable, int32]
	Updated   db.Col[SystemParametersTable, int64]
	UpdatedBy db.Col[SystemParametersTable, int32]
}

func (e SystemParametersTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "system_parameters",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ID},
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
