//go:build !js && !fdb

package db

import (
	"time"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/scylla"
)

// ─────────────────────────────────────────────────────────────────────────────
// THIS IS THE PROJECT'S DATABASE CHOICE — the only declaration in the entire
// codebase that names a database. There is one per build target:
//
//	driver_scylla.go (!js)  ->  scylla.TableStruct: ScyllaDB, the server build
//	driver_js.go     (js)   ->  idb.TableStruct:    IndexedDB, the browser build
//
// Point TableStruct at another driver and the whole project switches: no table
// declaration changes, no query changes, nothing else in this package changes.
//
// It has to be a type alias rather than a constructor function, because the driver
// is a type argument of the embedded TableStruct — that is what keeps every query
// statically typed. Go cannot instantiate a generic type from a runtime value, so
// the driver must be named where the table and record types are statically known,
// and the build tag picks which of the two aliases exists.
//
// To read one table from a *second* database at runtime, leave this alone and pass
// that driver's executor per query: Query(&rows).Via(otherExecutor).
// ─────────────────────────────────────────────────────────────────────────────

type TableStruct[TableT Schema[TableT], RecordT orm.TableBaseInterface[TableT, RecordT]] = scylla.TableStruct[TableT, RecordT]

// SetDataFrameLock installs the lock a DataFrame compaction takes (core installs fareward's at boot,
// with the file store: the lock's action number lives in core). It names the driver, which is why it
// is not in db.go.
func SetDataFrameLock(acquire func(lockName string, wait, lease time.Duration) (release func(), isTaken bool, err error)) {
	scylla.AcquireFrameLock = acquire
}
