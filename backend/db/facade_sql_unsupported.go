//go:build !oceanbase && !postgres && !mariadb

package db

import "errors"

// SQL reports exist only on OceanBase (facade_oceanbase.go): on every other build they fail loudly.

var errSQLReportsNeedOceanBase = errors.New("SQL reports need the oceanbase build")

func QuerySQL[RowT any](statement string, args ...any) ([]RowT, error) {
	return nil, errSQLReportsNeedOceanBase
}

func SQLTable[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]]() string {
	return MakeSchema[RecordT]().Name
}
