//go:build !js && !fdb && !ydb

package db

import (
	"fmt"
	"strings"

	"github.com/gocql/gocql"
	"github.com/ivanjoz/genix-orm/scylla"
)

// The functions below are the driver-neutral facade: the rest of the backend calls them and never
// names the database. Each build tag has its own file with the same signatures (see driver_scylla.go).

// Connect configures the connection. Scripting entry points dial right away; the server dials on
// the first query.
func Connect(params ConnectParams) {
	connectedKeyspace = params.Keyspace
	lastScyllaParams = scylla.ConnParams{
		Host:             params.Host,
		Port:             params.Port,
		User:             params.User,
		Password:         params.Password,
		Keyspace:         params.Keyspace,
		MaxClusteringKey: params.MaxClusteringKey,
	}
	if params.Scripting {
		scylla.MakeScyllaConnection(lastScyllaParams)
		return
	}
	scylla.SetScyllaConnection(lastScyllaParams)
}

var lastScyllaParams scylla.ConnParams

// Deploy creates the keyspace and the ORM's internal tables, then homologates the controllers'
// tables, views and indexes. Every step is idempotent.
func Deploy(controllers ...Controller) error {
	if err := scylla.Init(); err != nil {
		return err
	}
	scylla.DeployScylla(0, controllers...)
	return nil
}

// MakeController builds the controller of one table: export/restore, recalculation, text search
// and DataFrame maintenance. The keyspace is the one Connect received.
func MakeController[RecordT scylla.RecordOf[TableT, RecordT], TableT Schema[TableT]]() Controller {
	schema := MakeSchema[RecordT]()
	keyspace := schema.Namespace
	if keyspace == "" {
		keyspace = connectedKeyspace
	}
	return &scylla.ScyllaController[RecordT, TableT]{
		TableName: fmt.Sprintf("%s.%s", keyspace, schema.Name),
		Table:     scylla.MakeScyllaTable[RecordT](),
		Schema:    schema,
	}
}

// ListCounters returns the names of every autoincrement counter.
func ListCounters() ([]string, error) {
	counterNames := []string{}
	counterIterator := scyllaSession().Query(fmt.Sprintf(`SELECT name FROM %s.sequences`, connectedKeyspace)).Iter()
	for counterName := ""; counterIterator.Scan(&counterName); {
		counterNames = append(counterNames, counterName)
	}
	return counterNames, counterIterator.Close()
}

// WipePartition deletes every row of one partition of a table: the records, the ORM's side tables
// for it (index_updated, delta moves) and its by-IDs cache slots. It does not touch text search,
// DataFrames or snapshot files, which the controller and the file store own.
func WipePartition(table TableDescription, partition int32) error {
	session := scyllaSession()
	keyspace := connectedKeyspace

	existingTables := map[string]bool{}
	tableIterator := session.Query(`SELECT table_name FROM system_schema.tables WHERE keyspace_name = ?`, keyspace).Iter()
	for tableName := ""; tableIterator.Scan(&tableName); {
		existingTables[tableName] = true
	}
	if err := tableIterator.Close(); err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	statements := []string{fmt.Sprintf(`DELETE FROM %s.%s WHERE %s = %d`, keyspace, table.Name, table.Partition, partition)}
	if existingTables[table.Name+"__index_updated"] {
		statements = append(statements, fmt.Sprintf(`DELETE FROM %s.%s__index_updated WHERE partition_id = %d`, keyspace, table.Name, partition))
	}
	statements = append(statements, fmt.Sprintf(`DELETE FROM %s.cache_by_ids_slots WHERE table_id = %d AND partition_id = %d`,
		keyspace, table.ID, partition))
	// Delta moves are partitioned by (partition_id, index_tag, pinned_prefix): list the
	// partition's rows, then delete each one.
	if deltaTable := table.Name + "__delta_moves"; existingTables[deltaTable] {
		iterator := session.Query(fmt.Sprintf(`SELECT DISTINCT partition_id, index_tag, pinned_prefix FROM %s.%s`, keyspace, deltaTable)).Iter()
		// Typed as deltaMovesCreateScript declares them: gocql cannot scan a bigint into an any.
		var partitionID, pinnedPrefix int64
		var indexTag string
		for iterator.Scan(&partitionID, &indexTag, &pinnedPrefix) {
			if partitionID == int64(partition) {
				statements = append(statements, fmt.Sprintf(`DELETE FROM %s.%s WHERE partition_id = %d AND index_tag = %s AND pinned_prefix = %s`,
					keyspace, deltaTable, partition, cqlLiteral(indexTag), cqlLiteral(pinnedPrefix)))
			}
		}
		if err := iterator.Close(); err != nil {
			return fmt.Errorf("read %s: %w", deltaTable, err)
		}
	}
	for _, statement := range statements {
		if err := scylla.QueryExec(statement); err != nil {
			return fmt.Errorf("%s: %w", statement, err)
		}
	}
	return nil
}

// cqlLiteral writes a delta-moves key value back into a statement.
func cqlLiteral(value any) string {
	switch typed := value.(type) {
	case string:
		return "'" + strings.ReplaceAll(typed, "'", "''") + "'"
	case []byte:
		return fmt.Sprintf("0x%x", typed)
	default:
		return fmt.Sprint(typed)
	}
}

// scyllaSession dials (or returns) the session for the raw-CQL helpers. MakeScyllaConnection hands
// back the cached session when one exists, so this is cheap after the first call.
func scyllaSession() *gocql.Session { return scylla.MakeScyllaConnection(lastScyllaParams) }
