//go:build postgres && !js

package db

import (
	"errors"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/pgdriver"
)

// The driver-neutral facade on PostgreSQL: the same signatures as facade_scylla.go.

// Connect records the server; the pool opens on first use. Scripting entry points ping the server
// right away so a wrong address fails before any work. The keyspace is a PostgreSQL schema.
func Connect(params ConnectParams) {
	connectedKeyspace = params.Keyspace
	pgdriver.SetConnection(pgdriver.ConnParams{
		Host:         params.Postgres.Host,
		Port:         params.Postgres.Port,
		User:         params.Postgres.User,
		Password:     params.Postgres.Password,
		DatabaseName: params.Postgres.Database,
		Database:     params.Keyspace,
		MaxConns:     params.Postgres.MaxConns,
	})
	if params.Scripting {
		if err := pgdriver.Ping(); err != nil {
			panic(err)
		}
	}
}

// Deploy creates or homologates the tables and indexes of every controller.
func Deploy(controllers ...Controller) error {
	return pgdriver.Deploy(controllers...)
}

// MakeController builds the controller of one table.
func MakeController[RecordT orm.TableBaseInterface[TableT, RecordT], TableT Schema[TableT]]() Controller {
	return &pgdriver.PgController[RecordT, TableT]{Schema: MakeSchema[RecordT]()}
}

// ListCounters returns the names of every sequence.
func ListCounters() ([]string, error) { return pgdriver.ListCounters() }

// WipePartition deletes one partition of a table and of its side tables.
func WipePartition(table TableDescription, partition int32) error {
	return pgdriver.WipePartition(table.Namespace, table.ID, int64(partition))
}

// SetCounter moves a sequence to an absolute value and returns what it held. Sequences belong to
// PostgreSQL under this build, not to fareward.
func SetCounter(counterName string, value int64) (int64, error) {
	return pgdriver.SetCounter(counterName, value)
}

// UseTableKeyBlocks makes the server reserve table keys 64 at a time and hand them out from memory:
// one sequence statement per block instead of one per write (pgdriver.reserveTableKeys).
func UseTableKeyBlocks() { pgdriver.UseTableKeyBlocks() }

// BypassFarewardSequences does nothing here: fareward never owns the sequences of a postgres build.
func BypassFarewardSequences() {}

// RenameColumns has no PostgreSQL implementation.
func RenameColumns(apply bool) error {
	return errors.New("fn-rename-columns is not implemented on PostgreSQL")
}

// QuerySQL runs a read-only SQL report and scans each row into a RowT: every result column names a
// field of RowT by its db tag or its name. `?` placeholders are numbered for PostgreSQL.
func QuerySQL[RowT any](statement string, args ...any) ([]RowT, error) {
	return pgdriver.QuerySQL[RowT](statement, args...)
}

// SQLTable is the qualified name of a table, for the FROM and JOIN of a SQL report.
func SQLTable[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]]() string {
	return pgdriver.QualifiedTableName(TableOf[RecordT]())
}
