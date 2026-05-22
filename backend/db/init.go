package db

import (
	"errors"
	"fmt"
)

// Init creates the ORM internal tables required before sequence or cache-version features are used.
//
// Sonic text-search backend: callers wire the Sonic endpoint via
// text_search.Configure(host, port, password) before the first write.
// The db package can't do it here without creating a core ->
// core/types -> db -> core import cycle, so the application entry
// points (main.go, exec/init.go) call Configure themselves after
// core.PopulateVariables.
func Init() error {
	if err := CreateKeyspaceIfNotExists(); err != nil {
		return err
	}
	if err := InitSequencesTable(); err != nil {
		return err
	}
	if err := InitCacheVersionTable(); err != nil {
		return err
	}
	return nil
}

// CreateKeyspaceIfNotExists ensures the configured keyspace exists in ScyllaDB,
// creating it with SimpleStrategy / replication_factor=1 when missing.
func CreateKeyspaceIfNotExists() error {
	keyspace := connParams.Keyspace
	if keyspace == "" {
		return errors.New("CreateKeyspaceIfNotExists: no keyspace configured")
	}
	stmt := fmt.Sprintf(
		"CREATE KEYSPACE IF NOT EXISTS %v WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}",
		keyspace,
	)
	return QueryExec(stmt)
}

// InitSequencesTable ensures the shared autoincrement counter table exists before sequence-backed inserts run.
func InitSequencesTable() error {
	keyspace := connParams.Keyspace
	if keyspace == "" {
		return errors.New("Init: no keyspace configured")
	}

	createTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %v.sequences (
			name text, current_value counter,
			PRIMARY KEY (name)
		)
		%v;`,
		keyspace, makeStatementWith)

	return QueryExec(createTableQuery)
}
