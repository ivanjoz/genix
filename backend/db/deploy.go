package db

import "fmt"

type ScyllaColumns struct {
	Keyspace string
	Table    string
	Name     string
	Type     string
}

func (e ScyllaColumns) Keyspace_() CoStr { return CoStr{"keyspace_name"} }
func (e ScyllaColumns) Table_() CoStr    { return CoStr{"keyspace_name"} }
func (e ScyllaColumns) Name_() CoStr     { return CoStr{"keyspace_name"} }
func (e ScyllaColumns) Type_() CoStr     { return CoStr{"keyspace_name"} }

func (e ScyllaColumns) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:   "system_schema",
		Name:       "columns",
		PrimaryKey: e.Keyspace_(),
	}
}

type ScyllaViews struct {
	ViewName string
	Table    string
}

func (e ScyllaViews) ViewName_() CoStr { return CoStr{"view_name"} }
func (e ScyllaViews) Table_() CoStr    { return CoStr{"base_table_name"} }

func DeployScylla(tables ...TableSchemaInterface) {

	fmt.Println("ejecutando select...")
	result := Select(func(q *Query[ScyllaColumns], col ScyllaColumns) {
		q.Where(col.Keyspace_().Equals(connParams.Keyspace))
	})

	if result.Error != nil {
		fmt.Println("Error:", result.Error)
	}

	fmt.Print(result.Records)
}
