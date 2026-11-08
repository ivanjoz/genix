//go:build js

package db

import (
	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/idb"
)

// The browser build's database choice: IndexedDB. See driver_mariadb.go for why this is an alias.
type TableStruct[TableT Schema[TableT], RecordT orm.TableBaseInterface[TableT, RecordT]] = idb.TableStruct[TableT, RecordT]
