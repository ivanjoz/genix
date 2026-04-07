package db

import (
	"slices"
	"strings"
	"testing"

	"github.com/viant/xunsafe"
)

type fanoutViewRecord struct {
	TableStruct[fanoutViewRecordTable, fanoutViewRecord]
	EmpresaID  int32
	ID         int64
	ProductIDs []int32 `db:",list"`
	Fecha      int16
	Updated    int32
}

type fanoutViewRecordTable struct {
	TableStruct[fanoutViewRecordTable, fanoutViewRecord]
	EmpresaID  Col[fanoutViewRecordTable, int32]
	ID         Col[fanoutViewRecordTable, int64]
	ProductIDs Col[fanoutViewRecordTable, []int32]
	Fecha      Col[fanoutViewRecordTable, int16]
	Updated    Col[fanoutViewRecordTable, int32]
}

func (e fanoutViewRecordTable) GetSchema() TableSchema {
	return TableSchema{
		Name:      "fanout_view_record",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
		Indexes: []Index{
			{
				Type:     TypeViewTable,
				Keys:     []Coln{e.ProductIDs, e.Fecha},
				Cols:     []Coln{e.Updated},
				KeepPart: true,
			},
		},
	}
}

func getRequiredViewTable(t *testing.T, scyllaTable ScyllaTable[any], name string) *viewInfo {
	t.Helper()
	view := scyllaTable.views[name]
	if view == nil {
		t.Fatalf("view table %q not found", name)
	}
	return view
}

func TestViewTablesCompileAsTableBackedViews(t *testing.T) {
	scyllaTable := MakeScyllaTable[fanoutViewRecord, fanoutViewRecordTable]()
	view := getRequiredViewTable(t, scyllaTable, "fanout_view_record__product_ids_fecha_view")

	if view.Type != 9 {
		t.Fatalf("expected view table type 9, got %d", view.Type)
	}
	if view.fanoutColumnName != "product_ids" {
		t.Fatalf("expected product_ids as fanout column, got %q", view.fanoutColumnName)
	}
	if slices.Contains(view.availableColumns, "product_ids") {
		t.Fatalf("expected product_ids to be excluded from selectable columns, got %v", view.availableColumns)
	}
	if !slices.Contains(view.availableColumns, "updated") {
		t.Fatalf("expected updated to be selectable, got %v", view.availableColumns)
	}

	createScript := view.getCreateScript()
	if !strings.Contains(createScript, "CREATE TABLE") {
		t.Fatalf("expected CREATE TABLE script, got %q", createScript)
	}
	if !strings.Contains(createScript, "PRIMARY KEY ((empresa_id), product_ids, fecha, id)") {
		t.Fatalf("unexpected primary key for view table: %q", createScript)
	}

	secondaryIndexScript := getViewTableMaintenanceIndexCreateScript(view, scyllaTable)
	if !strings.Contains(secondaryIndexScript, "((empresa_id), id)") {
		t.Fatalf("unexpected secondary index script: %q", secondaryIndexScript)
	}

	bestCapability := MatchQueryCapability([]ColumnStatement{
		{Col: "empresa_id", Operator: "=", Value: int32(7)},
		{Col: "product_ids", Operator: "CONTAINS", Value: int64(9)},
		{Col: "fecha", Operator: ">=", Value: int16(3)},
	}, scyllaTable.capabilities)
	if bestCapability == nil || bestCapability.Source != view {
		t.Fatalf("expected view table capability match, got %+v", bestCapability)
	}

	whereStatements := view.getStatementPrepared(
		ColumnStatement{Col: "empresa_id", Operator: "=", Value: int32(7)},
		ColumnStatement{Col: "product_ids", Operator: "CONTAINS", Value: int64(9)},
		ColumnStatement{Col: "fecha", Operator: ">=", Value: int16(3)},
	)
	if len(whereStatements) != 1 || whereStatements[0].Clause != "empresa_id = ? AND product_ids = ? AND fecha >= ?" {
		t.Fatalf("unexpected translated where clause: %v", whereStatements)
	}
	if got := whereStatements[0].Values; len(got) != 3 || convertToInt64(got[0]) != 7 || convertToInt64(got[1]) != 9 || convertToInt64(got[2]) != 3 {
		t.Fatalf("unexpected translated where values: %v", got)
	}
}

func TestViewTablesFanOutSliceValues(t *testing.T) {
	scyllaTable := MakeScyllaTable[fanoutViewRecord, fanoutViewRecordTable]()
	view := getRequiredViewTable(t, scyllaTable, "fanout_view_record__product_ids_fecha_view")

	record := fanoutViewRecord{
		EmpresaID:  3,
		ID:         91,
		ProductIDs: []int32{11, 17, 17},
		Fecha:      5,
		Updated:    42,
	}

	fanoutValues := getViewTableFanoutValues(view, xunsafe.AsPointer(&record))
	if len(fanoutValues) != 3 {
		t.Fatalf("expected 3 fanout values, got %d", len(fanoutValues))
	}
	if fanoutValues[0] != int32(11) || fanoutValues[1] != int32(17) || fanoutValues[2] != int32(17) {
		t.Fatalf("unexpected fanout values: %#v", fanoutValues)
	}

	productColumnFound := false
	for _, column := range view.tableColumns {
		if getViewTableColumnName(column) != "product_ids" {
			continue
		}
		productColumnFound = true
		columnType := getViewTableColumnType(column.SourceColumn, column.UsesSliceElement)
		if columnType.FieldType != "int32" || columnType.ColType != "int" {
			t.Fatalf("expected scalar int32/int physical column, got %+v", columnType)
		}
		if !column.UsesSliceElement {
			t.Fatal("expected product_ids to use slice fan-out values")
		}
	}
	if !productColumnFound {
		t.Fatal("expected physical product_ids column in the view table")
	}
}
