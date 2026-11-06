//go:build fdb && !js

package db

import (
	"errors"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/fdbdriver"
)

// The driver-neutral facade on FoundationDB: the same signatures as facade_scylla.go.

// Connect records the fdbgate socket; the driver dials on first use. Scripting entry points ping
// the daemon right away so a wrong socket fails before any work.
func Connect(params ConnectParams) {
	connectedKeyspace = params.Keyspace
	fdbdriver.SetConnection(fdbdriver.ConnParams{SocketPath: params.FDBSocket, Keyspace: params.Keyspace})
	if params.Scripting {
		if err := fdbdriver.Ping(); err != nil {
			panic(err)
		}
	}
}

// Deploy compiles every controller's table, which validates the schemas. FoundationDB has no schema
// to create: keys are written as records arrive.
func Deploy(controllers ...Controller) error {
	for _, controller := range controllers {
		controller.GetTable()
	}
	return nil
}

// MakeController builds the controller of one table.
func MakeController[RecordT orm.TableBaseInterface[TableT, RecordT], TableT Schema[TableT]]() Controller {
	return &fdbdriver.FdbController[RecordT, TableT]{Schema: MakeSchema[RecordT]()}
}

// ListCounters returns the names of every autoincrement counter.
func ListCounters() ([]string, error) { return fdbdriver.ListCounters() }

// WipePartition deletes every key of one partition of a table: records, index entries and delta
// moves. By-IDs slot files, text search, DataFrames and snapshots are not touched, as on Scylla.
func WipePartition(table TableDescription, partition int32) error {
	return fdbdriver.WipePartition(table.Namespace, table.ID, int64(partition))
}

// SetCounter moves a counter to an absolute value and returns what it held. Sequences belong to
// FoundationDB under this build (decision D3), not to fareward.
func SetCounter(counterName string, value int64) (int64, error) {
	return fdbdriver.SetCounter(counterName, value)
}

// BypassFarewardSequences does nothing here: fareward never owns the sequences of an fdb build.
func BypassFarewardSequences() {}

// RenameColumns has nothing to rename: a row keys its columns by a hash of the name, and the rename
// tooling is Scylla's (column_rename_scylla.go).
func RenameColumns(apply bool) error {
	return errors.New("fn-rename-columns is a Scylla tool; it has no FoundationDB counterpart yet")
}
