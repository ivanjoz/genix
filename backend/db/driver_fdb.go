//go:build fdb && !js

package db

import (
	"time"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/fdbdriver"
)

// The FoundationDB build's database choice (`go build -tags fdb`): every table embeds the fdbdriver
// TableStruct. See driver_scylla.go for why this is a type alias and not a runtime switch, and
// docs/FOUNDATIONDB_PLAN.md for what the driver serves so far.
type TableStruct[TableT Schema[TableT], RecordT orm.TableBaseInterface[TableT, RecordT]] = fdbdriver.TableStruct[TableT, RecordT]

// SetDataFrameLock is accepted and unused: the FDB driver does not keep DataFrames yet.
func SetDataFrameLock(acquire func(lockName string, wait, lease time.Duration) (release func(), isTaken bool, err error)) {
}
