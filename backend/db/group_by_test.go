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

type int32PackedViewRecord struct {
	TableStruct[int32PackedViewSchema, int32PackedViewRecord]
	EmpresaID   int32 `db:"empresa_id"`
	ID          int64 `db:"id"`
	StatusTrace int8  `db:"status_trace"`
	Updated     int32 `db:"updated"`
}

type int32PackedViewSchema struct {
	TableStruct[int32PackedViewSchema, int32PackedViewRecord]
	EmpresaID   Col[int32PackedViewSchema, int32]
	ID          Col[int32PackedViewSchema, int64]
	StatusTrace Col[int32PackedViewSchema, int8]
	Updated     Col[int32PackedViewSchema, int32]
}

func (e int32PackedViewSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "int32_packed_view_records",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
		Views: []View{
			// Match the sale-order status trace view: a small enum prefix packed with an 8-digit updated slot.
			{Keys: []Coln{e.StatusTrace.Int32(), e.Updated.DecimalSize(8)}, KeepPart: true},
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

func TestPackedViewCapabilityMatchesEqualityPrefixPlusRange(t *testing.T) {
	scyllaTable := MakeScyllaTable[fullViewRecord, fullViewSchema]()
	results := []fullViewRecord{}
	query := Query[fullViewRecord, fullViewSchema](&results)

	query.EmpresaID.Equals(1)
	query.Status.Equals(6)
	query.Updated.GreaterEqual(0)

	bestCapability := MatchQueryCapability(query.GetTableInfo().statements, scyllaTable.capabilities)
	if bestCapability == nil {
		t.Fatal("expected a packed view capability match")
	}
	if bestCapability.Source == nil || bestCapability.Source.Type != 8 {
		t.Fatalf("expected a packed range view, got %+v", bestCapability.Source)
	}

	whereStatements := bestCapability.Source.getStatement(query.GetTableInfo().statements...)
	if len(whereStatements) != 1 {
		t.Fatalf("expected a single packed where clause, got %v", whereStatements)
	}

	packedView := bestCapability.Source
	expectedLowerBound := computePackedInt64ValueNonNegative([]int64{6, 0}, packedView.packedSlotDigitsPerColumn)
	expectedUpperBound := computePackedInt64ValueNonNegative([]int64{7, 0}, packedView.packedSlotDigitsPerColumn)
	expectedWhere := fmt.Sprintf("empresa_id = 1 AND %s >= %d AND %s < %d",
		packedView.column.GetName(), expectedLowerBound,
		packedView.column.GetName(), expectedUpperBound,
	)
	if whereStatements[0] != expectedWhere {
		t.Fatalf("unexpected packed where clause: %v", whereStatements[0])
	}
}

func TestInt32PackedViewUpperBoundKeepsCarryDigit(t *testing.T) {
	scyllaTable := MakeScyllaTable[int32PackedViewRecord, int32PackedViewSchema]()
	results := []int32PackedViewRecord{}
	query := Query[int32PackedViewRecord, int32PackedViewSchema](&results)

	query.EmpresaID.Equals(1)
	query.StatusTrace.Equals(9)
	query.Updated.GreaterEqual(38768176)

	bestCapability := MatchQueryCapability(query.GetTableInfo().statements, scyllaTable.capabilities)
	if bestCapability == nil || bestCapability.Source == nil || bestCapability.Source.Type != 8 {
		t.Fatalf("expected an int32 packed range view, got %+v", bestCapability)
	}
	if !bestCapability.Source.RequiresPostFilter {
		t.Fatal("expected int32 packed range views to enable exact post-filtering")
	}

	whereStatements := bestCapability.Source.getStatement(query.GetTableInfo().statements...)
	if len(whereStatements) != 1 {
		t.Fatalf("expected a single packed where clause, got %v", whereStatements)
	}

	expectedWhere := fmt.Sprintf("empresa_id = 1 AND %s >= 938768176 AND %s < 1000000000",
		bestCapability.Source.column.GetName(),
		bestCapability.Source.column.GetName(),
	)
	if whereStatements[0] != expectedWhere {
		t.Fatalf("unexpected int32 packed where clause: %v", whereStatements[0])
	}
}
