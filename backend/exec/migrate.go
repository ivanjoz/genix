package exec

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
	"fmt"
)

// makeDBController creates a ScyllaController for db2 package types using generics.
// This function automatically handles queries for any db2 table type using the
// TableQueryInterface for clean, simple query building.
func makeDBController[T scylla.TableBaseInterface[E, T], E scylla.TableSchemaInterface[E]]() scylla.ScyllaControllerInterface {
	// Get the table struct instance
	schema := scylla.MakeSchema[T]()
	scyllaTable := scylla.MakeScyllaTable[T]()

	// Get table name and keyspace
	tableName := schema.Name
	keyspace := schema.Keyspace
	if keyspace == "" {
		keyspace = core.Env.DB_NAME
	}
	fullTableName := fmt.Sprintf("%s.%s", keyspace, tableName)

	contoller := scylla.ScyllaController[T, E]{
		TableName: fullTableName,
		Table:     scyllaTable,
		Schema:    schema,
	}
	return &contoller
}
