package types

import "github.com/ivanjoz/genix-orm/scylla"

type Parameters struct {
	scylla.TableStruct[ParametersTable, Parameters]
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
	scylla.TableStruct[ParametersTable, Parameters]
	CompanyID scylla.Col[ParametersTable, int32]
	Group     scylla.Col[ParametersTable, int32]
	Key       scylla.Col[ParametersTable, string]
	Value     scylla.Col[ParametersTable, string]
	ValueInt  scylla.Col[ParametersTable, int32]
	Values    scylla.ColSlice[ParametersTable, int32]
	Status    scylla.Col[ParametersTable, int8]
	Updated   scylla.Col[ParametersTable, int32]
	UpdatedBy scylla.Col[ParametersTable, int32]
}

func (e ParametersTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "parameters",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         scylla.Cols(e.Group, e.Key),
		Indexes:      []scylla.Index{},
	}
}
