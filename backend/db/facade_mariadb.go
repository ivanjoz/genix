//go:build mariadb && !js

package db

import (
	"errors"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/mariadbdriver"
)

// The driver-neutral facade on MariaDB: the same signatures as facade_scylla.go.

// Connect records the server; the pool opens on first use. Scripting entry points ping the server
// right away so a wrong address fails before any work. The keyspace is the MariaDB database.
func Connect(params ConnectParams) {
	connectedKeyspace = params.Keyspace
	mariadbdriver.SetConnection(mariadbdriver.ConnParams{
		Host:     params.MariaDB.Host,
		Port:     params.MariaDB.Port,
		User:     params.MariaDB.User,
		Password: params.MariaDB.Password,
		Database: params.Keyspace,
		MaxConns: params.MariaDB.MaxConns,
	})
	if params.Scripting {
		if err := mariadbdriver.Ping(); err != nil {
			panic(err)
		}
	}
}

// Deploy creates or homologates the tables and indexes of every controller.
func Deploy(controllers ...Controller) error {
	return mariadbdriver.Deploy(controllers...)
}

// MakeController builds the controller of one table.
func MakeController[RecordT orm.TableBaseInterface[TableT, RecordT], TableT Schema[TableT]]() Controller {
	return &mariadbdriver.MariaController[RecordT, TableT]{Schema: MakeSchema[RecordT]()}
}

// ListCounters returns the names of every sequence.
func ListCounters() ([]string, error) { return mariadbdriver.ListCounters() }

// WipePartition deletes one partition of a table and of its side tables.
func WipePartition(table TableDescription, partition int32) error {
	return mariadbdriver.WipePartition(table.Namespace, table.ID, int64(partition))
}

// SetCounter moves a sequence to an absolute value and returns what it held. Sequences belong to
// MariaDB under this build, not to fareward.
func SetCounter(counterName string, value int64) (int64, error) {
	return mariadbdriver.SetCounter(counterName, value)
}

// UseTableKeyBlocks makes the server reserve table keys 64 at a time and hand them out from memory:
// one sequence transaction per block instead of one per write (mariadbdriver.reserveTableKeys).
func UseTableKeyBlocks() { mariadbdriver.UseTableKeyBlocks() }

// BypassFarewardSequences does nothing here: fareward never owns the sequences of a mariadb build.
func BypassFarewardSequences() {}

// RenameColumns has no MariaDB implementation.
func RenameColumns(apply bool) error {
	return errors.New("fn-rename-columns is not implemented on MariaDB")
}

// QuerySQL runs a read-only SQL report and scans each row into a RowT: every result column names a
// field of RowT by its db tag or its name. Writes never go here.
func QuerySQL[RowT any](statement string, args ...any) ([]RowT, error) {
	return mariadbdriver.QuerySQL[RowT](statement, args...)
}

// SQLTable is the qualified name of a table, for the FROM and JOIN of a SQL report.
func SQLTable[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]]() string {
	return mariadbdriver.QualifiedTableName(TableOf[RecordT]())
}
