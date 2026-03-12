package db

import (
	"strings"
	"testing"
)

type singleViewRecord struct {
	TableStruct[singleViewSchema, singleViewRecord]
	CompanyID int32 `db:"company_id"`
	ID        int32 `db:"id"`
}

type singleViewSchema struct {
	TableStruct[singleViewSchema, singleViewRecord]
	CompanyID Col[singleViewSchema, int32]
	ID        Col[singleViewSchema, int32]
}

func (schema singleViewSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "single_view_table",
		Partition: schema.CompanyID,
		Keys:      []Coln{schema.ID},
		Views: []View{
			{
				Cols:     []Coln{schema.ID},
				KeepPart: false,
			},
		},
	}
}

func TestSingleDeclaredColumnViewDoesNotCreateVirtualHashColumn(t *testing.T) {
	// Reset caches so the test inspects a fresh schema compilation path.
	resetORMTableCachesForTesting()

	schemaTable := MakeScyllaTable[singleViewRecord, singleViewSchema]()

	if _, exists := schemaTable.columnsMap["zz_company_id_id"]; exists {
		t.Fatalf("unexpected virtual hash column generated for single-column view")
	}

	view, exists := schemaTable.views["single_view_table__id_view"]
	if !exists {
		t.Fatalf("expected simple view metadata to be registered")
	}

	if view.column.GetName() != "id" {
		t.Fatalf("expected view column to reuse id, got %q", view.column.GetName())
	}

	if view.Type != 6 {
		t.Fatalf("expected simple view type 6, got %d", view.Type)
	}

	createScript := view.getCreateScript()
	if strings.Contains(createScript, "zz_company_id_id") {
		t.Fatalf("create script should not reference a synthetic hash column: %s", createScript)
	}

	if !strings.Contains(createScript, "PRIMARY KEY (id,company_id)") {
		t.Fatalf("unexpected primary key definition for single-column view: %s", createScript)
	}
}

type singlePackedKeepPartRecord struct {
	TableStruct[singlePackedKeepPartSchema, singlePackedKeepPartRecord]
	CompanyID int32 `db:"company_id"`
	ID        int32 `db:"id"`
	ListID    int32 `db:"list_id"`
	Status    int8  `db:"status"`
}

type singlePackedKeepPartSchema struct {
	TableStruct[singlePackedKeepPartSchema, singlePackedKeepPartRecord]
	CompanyID Col[singlePackedKeepPartSchema, int32]
	ID        Col[singlePackedKeepPartSchema, int32]
	ListID    Col[singlePackedKeepPartSchema, int32]
	Status    Col[singlePackedKeepPartSchema, int8]
}

func (schema singlePackedKeepPartSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "single_packed_keep_part",
		Partition: schema.CompanyID,
		Keys:      []Coln{schema.ID},
		Views: []View{
			{Cols: []Coln{schema.ListID.Int32(), schema.Status.DecimalSize(2)}},
		},
	}
}

func TestPackedKeepPartViewUsesDeclaredComponentsOnly(t *testing.T) {
	// Reset caches so the test exercises the full packed-view compilation path.
	resetORMTableCachesForTesting()

	schemaTable := MakeScyllaTable[singlePackedKeepPartRecord, singlePackedKeepPartSchema]()
	view, exists := schemaTable.views["single_packed_keep_part__pk_list_id_status_rng_view"]
	if !exists {
		t.Fatalf("expected packed keep-part view metadata to be registered")
	}

	if view.Type != 8 {
		t.Fatalf("expected packed range view type 8, got %d", view.Type)
	}

	whereStatements := view.getStatement(
		ColumnStatement{Col: "company_id", Operator: "=", Value: int32(1)},
		ColumnStatement{Col: "list_id", Operator: "=", Value: int32(2)},
		ColumnStatement{Col: "status", Operator: "=", Value: int8(1)},
	)

	if len(whereStatements) != 1 {
		t.Fatalf("expected a single packed where statement, got %v", whereStatements)
	}

	if !strings.Contains(whereStatements[0], "company_id = 1 AND") {
		t.Fatalf("expected company_id predicate in packed where statement: %s", whereStatements[0])
	}

	if strings.Contains(whereStatements[0], "zz_pk_list_id_status_rng = 0") {
		t.Fatalf("unexpected zero packed value in where statement: %s", whereStatements[0])
	}
}
