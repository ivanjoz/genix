package types

import "app/db"

type Parameters struct {
	db.TableStruct[ParametersTable, Parameters]
	CompanyID int32
	Group     int32
	Key       string
	Value     string
	ValueInt  int32
	Values    []int32
	Status    int8
	Updated   int32
	UpdatedBy int32
}

type ParametersTable struct {
	db.TableStruct[ParametersTable, Parameters]
	CompanyID db.Col[ParametersTable, int32]
	Group     db.Col[ParametersTable, int32]
	Key       db.Col[ParametersTable, string]
	Value     db.Col[ParametersTable, string]
	ValueInt  db.Col[ParametersTable, int32]
	Values    db.ColSlice[ParametersTable, int32]
	Status    db.Col[ParametersTable, int8]
	Updated   db.Col[ParametersTable, int32]
	UpdatedBy db.Col[ParametersTable, int32]
}

func (e ParametersTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "parameters",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.Group, e.Key},
		Indexes:      []db.Index{},
	}
}
