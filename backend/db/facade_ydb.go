//go:build ydb && !js

package db

import (
	"errors"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/ydbdriver"
)

// The driver-neutral facade on YDB: the same signatures as every other facade_*.go.

// Connect records the endpoint; the driver dials on first use. Scripting entry points ping the
// server right away so a wrong endpoint fails before any work.
func Connect(params ConnectParams) {
	connectedKeyspace = params.Keyspace
	ydbdriver.SetConnection(ydbdriver.ConnParams{Endpoint: params.YDBEndpoint, Database: params.YDBDatabase, Keyspace: params.Keyspace})
	if params.Scripting {
		if err := ydbdriver.Ping(); err != nil {
			panic(err)
		}
	}
}

// Deploy creates or homologates the tables, indexes and side tables of every controller.
func Deploy(controllers ...Controller) error {
	return ydbdriver.Deploy(controllers...)
}

// MakeController builds the controller of one table.
func MakeController[RecordT orm.TableBaseInterface[TableT, RecordT], TableT Schema[TableT]]() Controller {
	return &ydbdriver.YdbController[RecordT, TableT]{Schema: MakeSchema[RecordT]()}
}

// ListCounters returns the names of every sequence.
func ListCounters() ([]string, error) { return ydbdriver.ListCounters() }

// WipePartition deletes one partition of a table and of its side tables.
func WipePartition(table TableDescription, partition int32) error {
	return ydbdriver.WipePartition(table.Namespace, table.ID, int64(partition))
}

// SetCounter moves a sequence to an absolute value and returns what it held. Sequences belong to
// YDB under this build (decision Y11), not to fareward.
func SetCounter(counterName string, value int64) (int64, error) {
	return ydbdriver.SetCounter(counterName, value)
}

// BypassFarewardSequences does nothing here: fareward never owns the sequences of a ydb build.
func BypassFarewardSequences() {}

// RenameColumns has no YDB implementation yet (docs/YDB_PLAN.md Step 7).
func RenameColumns(apply bool) error {
	return errors.New("fn-rename-columns is not implemented on YDB yet")
}

// UseTableKeyBlocks does nothing here: only the OceanBase driver reserves table keys in blocks.
func UseTableKeyBlocks() {}
