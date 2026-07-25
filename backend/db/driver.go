package db

import (
	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/scylla"
)

// ─────────────────────────────────────────────────────────────────────────────
// THIS IS THE PROJECT'S DATABASE CHOICE — the only declaration in the entire
// codebase that names a database.
//
// Point TableStruct at another driver and the whole project switches: no table
// declaration changes, no query changes, nothing else in this package changes.
//
//	scylla.TableStruct  ->  ScyllaDB / Cassandra
//	dynamo.TableStruct  ->  DynamoDB
//
// It has to be a type alias rather than a constructor function, because the driver
// is a type argument of the embedded TableStruct — that is what keeps every query
// statically typed. Go cannot instantiate a generic type from a runtime value, so
// the driver must be named where the table and record types are statically known,
// and this alias is that one place.
//
// To read one table from a *second* database at runtime, leave this alone and pass
// that driver's executor per query: Query(&rows).Via(otherExecutor).
// ─────────────────────────────────────────────────────────────────────────────

type TableStruct[TableT Schema[TableT], RecordT orm.TableBaseInterface[TableT, RecordT]] = scylla.TableStruct[TableT, RecordT]

// Schema and Record are the constraints every wrapper in db.go is written against.
// Record does not name a driver: it takes the executor as a parameter, which Go
// infers from the record type. That is what keeps db.go driver-free.
type (
	Schema[TableT any]                     = orm.TableSchemaInterface[TableT]
	Record[TableT any, RecordT any, D any] = orm.RecordWithExecutor[TableT, RecordT, D]
)
