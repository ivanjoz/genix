//go:build postgres && !js

package db

import (
	"time"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/pgdriver"
)

// The PostgreSQL build's database choice (`go build -tags postgres`): every table embeds the
// pgdriver TableStruct. See driver_scylla.go for why this is a type alias and not a runtime switch.
// It exists for the sale-flow benchmarks (tests/loadtest/plan.md).
type TableStruct[TableT Schema[TableT], RecordT orm.TableBaseInterface[TableT, RecordT]] = pgdriver.TableStruct[TableT, RecordT]

// SetDataFrameLock is accepted and unused: this driver keeps its frames as aggregate tables, written
// in the record's transaction, so nothing compacts under a lock.
func SetDataFrameLock(acquire func(lockName string, wait, lease time.Duration) (release func(), isTaken bool, err error)) {
}
