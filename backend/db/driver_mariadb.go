//go:build !js && !fdb && !ydb && !oceanbase && !postgres

package db

import (
	"time"

	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/mariadbdriver"
)

// ─────────────────────────────────────────────────────────────────────────────
// THIS IS THE PROJECT'S DATABASE CHOICE — the only declaration in the entire
// codebase that names a database. There is one per build target:
//
//	driver_mariadb.go (default)   ->  mariadbdriver.TableStruct: MariaDB, the server build
//	driver_postgres.go (postgres) ->  pgdriver.TableStruct:      PostgreSQL
//	driver_js.go       (js)       ->  idb.TableStruct:           IndexedDB, the browser build
//
// (fdb, ydb and oceanbase are experimental builds with their own files.)
//
// It has to be a type alias rather than a constructor function, because the driver
// is a type argument of the embedded TableStruct — that is what keeps every query
// statically typed. Go cannot instantiate a generic type from a runtime value, so
// the driver must be named where the table and record types are statically known,
// and the build tag picks which alias exists.
// ─────────────────────────────────────────────────────────────────────────────
type TableStruct[TableT Schema[TableT], RecordT orm.TableBaseInterface[TableT, RecordT]] = mariadbdriver.TableStruct[TableT, RecordT]

// SetDataFrameLock is accepted and unused: this driver keeps its frames as aggregate tables, written
// in the record's transaction, so nothing compacts under a lock.
func SetDataFrameLock(acquire func(lockName string, wait, lease time.Duration) (release func(), isTaken bool, err error)) {
}
