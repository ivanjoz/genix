package db

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

type groupedMovementRecord struct {
	TableStruct[groupedMovementSchema, groupedMovementRecord]
	EmpresaID  int32   `db:"empresa_id"`
	ID         int64   `db:"id"`
	Fecha      int16   `db:"fecha"`
	ProductoID int32   `db:"producto_id"`
	Cantidad   int32   `db:"cantidad"`
	Promedio   float64 `db:"promedio"`
}

type groupedMovementSchema struct {
	TableStruct[groupedMovementSchema, groupedMovementRecord]
	EmpresaID  Col[groupedMovementSchema, int32]
	ID         Col[groupedMovementSchema, int64]
	Fecha      Col[groupedMovementSchema, int16]
	ProductoID Col[groupedMovementSchema, int32]
	Cantidad   Col[groupedMovementSchema, int32]
	Promedio   Col[groupedMovementSchema, float64]
}

func (e groupedMovementSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "grouped_movements",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
		Views: []View{
			// Keep the packed grouping view minimal: partition + packed key + aggregated payload column.
			{Keys: []Coln{e.Fecha, e.ProductoID.DecimalSize(10)}, Cols: []Coln{e.Cantidad}, KeepPart: true},
		},
	}
}

type fullViewRecord struct {
	TableStruct[fullViewSchema, fullViewRecord]
	EmpresaID int32  `db:"empresa_id"`
	ID        int64  `db:"id"`
	Status    int8   `db:"status"`
	Updated   int32  `db:"updated"`
	Nombre    string `db:"nombre"`
}

type fullViewSchema struct {
	TableStruct[fullViewSchema, fullViewRecord]
	EmpresaID Col[fullViewSchema, int32]
	ID        Col[fullViewSchema, int64]
	Status    Col[fullViewSchema, int8]
	Updated   Col[fullViewSchema, int32]
	Nombre    Col[fullViewSchema, string]
}

func (e fullViewSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "full_view_records",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
		Views: []View{
			// No Cols means the MV keeps the full base payload via SELECT *.
			{Keys: []Coln{e.Status, e.Updated.DecimalSize(9)}, KeepPart: true},
		},
	}
}

func findPackedGroupView(t *testing.T, scyllaTable ScyllaTable[any]) *viewInfo {
	t.Helper()
	for _, view := range scyllaTable.indexViews {
		if view.Type == 8 && slices.Equal(view.columnsNoPart, []string{"fecha", "producto_id"}) {
			return view
		}
	}
	t.Fatal("packed group view not found")
	return nil
}

func TestBuildNativeGroupByPlanWithPackedView(t *testing.T) {
	scyllaTable := MakeScyllaTable[groupedMovementRecord, groupedMovementSchema]()
	records := []groupedMovementRecord{}
	query := Query[groupedMovementRecord, groupedMovementSchema](&records)

	query.EmpresaID.Equals(7)
	query.Fecha.GreaterEqual(15)
	query.GroupBy(query.Fecha, query.ProductoID, query.Cantidad.Sum())

	plan, err := buildNativeGroupByPlan(query.GetTableInfo(), query.GetTableInfo().statements, scyllaTable)
	if err != nil {
		t.Fatalf("buildNativeGroupByPlan returned error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected a GroupBy plan")
	}
	if plan.ViewTableName == "" {
		t.Fatal("expected a packed view to be selected")
	}
	if len(plan.GroupByColumns) != 1 {
		t.Fatalf("expected a single packed GroupBy column, got %v", plan.GroupByColumns)
	}
	if len(plan.SelectExpressions) != 2 {
		t.Fatalf("expected packed key plus aggregate projection, got %v", plan.SelectExpressions)
	}
	if plan.SelectExpressions[1] != "SUM(cantidad) AS cantidad" {
		t.Fatalf("unexpected aggregate projection: %v", plan.SelectExpressions[1])
	}
	if len(plan.ScanColumns) == 0 || plan.ScanColumns[0].DecomposeView == nil {
		t.Fatal("expected the packed key scan column to decompose into primitive fields")
	}

	packedView := plan.ScanColumns[0].DecomposeView
	expectedLowerBound := computePackedInt64ValueNonNegative([]int64{15, 0}, packedView.packedSlotDigitsPerColumn)
	expectedClause := fmt.Sprintf("empresa_id = 7 AND %s >= %d", packedView.column.GetName(), expectedLowerBound)
	if len(plan.WhereStatements) != 1 || plan.WhereStatements[0] != expectedClause {
		t.Fatalf("unexpected packed where clauses: %v", plan.WhereStatements)
	}
}

func TestPackedGroupByDecomposesVirtualValue(t *testing.T) {
	scyllaTable := MakeScyllaTable[groupedMovementRecord, groupedMovementSchema]()
	packedView := findPackedGroupView(t, scyllaTable)

	packedValue := computePackedInt64ValueNonNegative([]int64{31, 4567}, packedView.packedSlotDigitsPerColumn)
	values := packedView.decomposeVirtualValue(packedValue)
	if len(values) != 2 {
		t.Fatalf("expected 2 decomposed values, got %v", values)
	}
	if values[0].(int64) != 31 || values[1].(int64) != 4567 {
		t.Fatalf("unexpected decomposed values: %v", values)
	}
}

func TestBuildNativeGroupByPlanRejectsAvgOnIntegerColumn(t *testing.T) {
	scyllaTable := MakeScyllaTable[groupedMovementRecord, groupedMovementSchema]()
	records := []groupedMovementRecord{}
	query := Query[groupedMovementRecord, groupedMovementSchema](&records)

	query.EmpresaID.Equals(1)
	query.GroupBy(query.Fecha, query.Cantidad.Avg())

	_, err := buildNativeGroupByPlan(query.GetTableInfo(), query.GetTableInfo().statements, scyllaTable)
	if err == nil {
		t.Fatal("expected Avg on an integer destination column to fail")
	}
	if !strings.Contains(err.Error(), "Avg() requires a float32 or float64") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMatchQueryCapabilityReturnsSelectedCapability(t *testing.T) {
	capabilities := []QueryCapability{
		{Signature: "empresa_id|=", Priority: 10, IsKey: true},
		{
			Signature: "empresa_id|=|status|=",
			Priority:  25,
			Source:    &viewInfo{name: "product_supply__status_view"},
		},
	}
	statements := []ColumnStatement{
		{Col: "empresa_id", Operator: "=", Value: 1},
		{Col: "status", Operator: "=", Value: 1},
	}

	bestCapability := MatchQueryCapability(statements, capabilities)
	if bestCapability == nil {
		t.Fatal("expected a capability match")
	}
	if bestCapability.Signature != "empresa_id|=|status|=" {
		t.Fatalf("unexpected capability selected: %s", bestCapability.Signature)
	}
	if bestCapability.Source == nil || bestCapability.Source.name != "product_supply__status_view" {
		t.Fatalf("unexpected capability source: %+v", bestCapability.Source)
	}
}

func TestVirtualViewWithoutProjectedColsKeepsFullSelectablePayload(t *testing.T) {
	scyllaTable := MakeScyllaTable[fullViewRecord, fullViewSchema]()

	var packedView *viewInfo
	for _, view := range scyllaTable.indexViews {
		if view.Type == 8 {
			packedView = view
			break
		}
	}
	if packedView == nil {
		t.Fatal("expected a packed range view")
	}

	for _, expectedColumn := range []string{"empresa_id", "id", "status", "updated", "nombre"} {
		if !slices.Contains(packedView.availableColumns, expectedColumn) {
			t.Fatalf("expected view to expose column %q, available=%v", expectedColumn, packedView.availableColumns)
		}
	}
}
