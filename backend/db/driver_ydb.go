//go:build ydb && !js

package db

import (
	"time"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/ydbdriver"
)

// The YDB build's database choice (`go build -tags ydb`): every table embeds the ydbdriver
// TableStruct. See driver_scylla.go for why this is a type alias and not a runtime switch, and
// docs/YDB_PLAN.md for what the driver serves so far.
type TableStruct[TableT Schema[TableT], RecordT orm.TableBaseInterface[TableT, RecordT]] = ydbdriver.TableStruct[TableT, RecordT]

// SetDataFrameLock is accepted and unused until the DataFrame step (docs/YDB_PLAN.md Step 7).
func SetDataFrameLock(acquire func(lockName string, wait, lease time.Duration) (release func(), isTaken bool, err error)) {
}
