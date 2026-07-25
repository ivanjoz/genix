package exec

import (
	"app/core"
	"app/db"
	"fmt"
	"github.com/ivanjoz/genix-orm/scylla"
)

// makeDBController creates a ScyllaController for db2 package types using generics.
// This function automatically handles queries for any db2 table type using the
// TableQueryInterface for clean, simple query building.
func makeDBController[T scylla.RecordOf[E, T], E db.Schema[E]]() db.Controller {
	// Get the table struct instance
	schema := db.MakeSchema[T]()
	scyllaTable := scylla.MakeScyllaTable[T]()

	// Get table name and keyspace
	tableName := schema.Name
	keyspace := schema.Namespace
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
