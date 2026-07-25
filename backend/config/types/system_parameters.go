package types

import "github.com/ivanjoz/genix-orm/scylla"

type SystemParameters struct {
	scylla.TableStruct[SystemParametersTable, SystemParameters]
	ID        int32   `json:",omitempty" db:"parameter_id,pk"`
	ValueText string  `json:",omitempty" db:"value_text"`
	ValueInts []int32 `json:",omitempty" db:"value_ints"`
	Value     int32   `json:",omitempty" db:"value"`
	CompanyID int32   `json:",omitempty" db:"empresa_id"`
	Updated   int32   `json:"upd," db:"updated"`
	UpdatedBy int32   `json:",omitempty" db:"updated_by"`
}

type SystemParametersTable struct {
	scylla.TableStruct[SystemParametersTable, SystemParameters]
	CompanyID scylla.Col[SystemParametersTable, int32]
	ID        scylla.Col[SystemParametersTable, int32]
	ValueText scylla.Col[SystemParametersTable, string]
	ValueInts scylla.ColSlice[SystemParametersTable, int32]
	Value     scylla.Col[SystemParametersTable, int32]
	Updated   scylla.Col[SystemParametersTable, int32]
	UpdatedBy scylla.Col[SystemParametersTable, int32]
}

func (e SystemParametersTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "system_parameters",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID),
		Indexes: []scylla.Index{
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}
