package db

import (
	"errors"
	"fmt"
)

// Init creates the ORM internal tables required before sequence or cache-version features are used.
func Init() error {
	if err := initSequencesTable(); err != nil {
		return err
	}
	if err := InitCacheVersionTable(); err != nil {
		return err
	}
	return nil
}

// initSequencesTable ensures the shared autoincrement counter table exists before sequence-backed inserts run.
func initSequencesTable() error {
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
