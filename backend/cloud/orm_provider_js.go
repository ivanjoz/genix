//go:build js

package cloud

import (
	"app/db"
	"errors"
)

// The browser build keeps every table in IndexedDB only: there is no cloud data mirror, so the
// DynamoDB and D1 providers are not linked at all.
func getProviderORM[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]]() (ORM[RecordT], error) {
	return nil, errors.New("the cloud data mirror does not exist in the browser build")
}

func IsDataMirrorEnabled() bool { return false }
