package db

import "fmt"

type ScyllaColumns struct {
	Keyspace string
	Table    string
	Name     string
	Type     string
}

type ScyllaColumnsSchema struct {
	Keyspace Col[string] `db:"keyspace_name"`
	Table    Col[string] `db:"table_name"`
	Name     Col[string] `db:"column_name"`
	Type     Col[string] `db:"type"`
}

func (e ScyllaColumnsSchema) GetSchema() TableSchema[ScyllaColumns] {
	return TableSchema[ScyllaColumns]{
		Keyspace:   "system_schema",
		Name:       "columns",
		PrimaryKey: e.Table,
	}
}

type ScyllaViews struct {
	ViewName string
	Table    string
}

type ScyllaViewsSchema struct {
	ViewName Col[string] `db:"view_name"`
	Table    Col[string] `db:"base_table_name"`
}

func (e ScyllaViewsSchema) GetSchema() TableSchema[ScyllaViews] {
	return TableSchema[ScyllaViews]{
		Name: "system_schema.views",
	}
}
func (e ScyllaViews) ViewName_() CoStr { return CoStr{"table_name", e.ViewName} }
func (e ScyllaViews) Table_() CoStr    { return CoStr{"column_name", e.Table} }

func DeployScylla[T any](tables ...TableSchemaInterface[T]) {

	fmt.Println("ejecutando select...")
	result := QuerySelect(func(q *Query[ScyllaColumns], col ScyllaColumnsSchema) {
		q.Where(col.Keyspace.Equals(connParams.Keyspace))
	})

	if result.Error != nil {
		fmt.Println("Error:", result.Error)
	}

	fmt.Print(result.Records)
}
