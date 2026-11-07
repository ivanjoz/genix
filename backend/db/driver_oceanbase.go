//go:build oceanbase && !js

package db

import (
	"time"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/obdriver"
)

// The OceanBase build's database choice (`go build -tags oceanbase`): every table embeds the
// obdriver TableStruct. See driver_scylla.go for why this is a type alias and not a runtime switch,
// and docs/OCEANBASE_PLAN.md for what the driver serves so far.
type TableStruct[TableT Schema[TableT], RecordT orm.TableBaseInterface[TableT, RecordT]] = obdriver.TableStruct[TableT, RecordT]

// SetDataFrameLock installs the lock a DataFrame compaction takes (core installs fareward's at boot).
func SetDataFrameLock(acquire func(lockName string, wait, lease time.Duration) (release func(), isTaken bool, err error)) {
	obdriver.AcquireFrameLock = acquire
}
