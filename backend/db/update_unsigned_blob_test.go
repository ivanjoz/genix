package db

import (
	"strings"
	"testing"
)

type unsignedBlobUpdateRecord struct {
	TableStruct[unsignedBlobUpdateSchema, unsignedBlobUpdateRecord]
	EmpresaID       int32    `db:"empresa_id"`
	ID              int64    `db:"id"`
	AccesosComputed []uint16 `db:"accesos_computed"`
}

type unsignedBlobUpdateSchema struct {
	TableStruct[unsignedBlobUpdateSchema, unsignedBlobUpdateRecord]
	EmpresaID       Col[unsignedBlobUpdateSchema, int32]
	ID              Col[unsignedBlobUpdateSchema, int64]
	AccesosComputed Col[unsignedBlobUpdateSchema, []uint16]
}

func (e unsignedBlobUpdateSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "unsigned_blob_update_test",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
	}
}

func TestMakeUpdateStatementsFormatsUint16SlicesAsBlobLiterals(t *testing.T) {
	// Regression: blob-backed []uint16 fields must serialize as 0x... in UPDATE statements.
	resetORMTableCachesForTesting()

	recordRows := []unsignedBlobUpdateRecord{
		{
			EmpresaID:       1,
			ID:              5,
			AccesosComputed: []uint16{8, 12, 24},
		},
	}

	table := Table[unsignedBlobUpdateRecord]()
	updateStatements := MakeUpdateStatements(&recordRows, table.AccesosComputed)

	if len(updateStatements) != 1 {
		t.Fatalf("expected 1 update statement, got=%d", len(updateStatements))
	}

	statement := updateStatements[0]
	if !strings.Contains(statement, "accesos_computed = 0x08000c001800") {
		t.Fatalf("expected blob literal for []uint16 update, got=%s", statement)
	}
}
