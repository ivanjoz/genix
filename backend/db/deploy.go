package db

import (
	"fmt"
)

type ScyllaColumns struct {
	Keyspace string `db:"keyspace_name"`
	Table    string `db:"table_name"`
	Name     string `db:"column_name"`
	Type     string `db:"type"`
}

func (e ScyllaColumns) GetTableSchema() TableSchema {
	return TableSchema{
		Name: "system_schema.columns",
	}
}
func (e ScyllaColumns) Keyspace_() CoStr { return CoStr{"keyspace_name", e.Keyspace} }
func (e ScyllaColumns) Table_() CoStr    { return CoStr{"table_name", e.Table} }
func (e ScyllaColumns) Name_() CoStr     { return CoStr{"column_name", e.Name} }
func (e ScyllaColumns) Type_() CoStr     { return CoStr{"type", e.Type} }

type ScyllaViews struct {
	ViewName string `db:"view_name"`
	Table    string `db:"base_table_name"`
}

func (e ScyllaViews) GetTableSchema() TableSchema {
	return TableSchema{
		Name: "system_schema.views",
	}
}
func (e ScyllaViews) ViewName_() CoStr { return CoStr{"table_name", e.ViewName} }
func (e ScyllaViews) Table_() CoStr    { return CoStr{"column_name", e.Table} }

func DeployScylla(tables ...TableSchemaInterface) {

	result := QuerySelect(func(query *Query[ScyllaColumns], t *ScyllaColumns) {
		query.Where(t.Keyspace_().Equals("")).Exec()
	})

	fmt.Printf("%v | %v", result.Records, result.Error)
}
