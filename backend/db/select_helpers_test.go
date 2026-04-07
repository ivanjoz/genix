package db

import (
	"fmt"
	"strings"
	"testing"
)

func TestSelectStatementCacheReusesSameShape(t *testing.T) {
	// Reset metadata caches so this test validates one deterministic compile+cache lifecycle.
	resetORMTableCachesForTesting()

	scyllaTable := MakeScyllaTable[int32PackedViewRecord, int32PackedViewSchema]()
	scyllaTable.keyspace = "genix_test"

	recordsFirst := []int32PackedViewRecord{}
	queryFirst := Query[int32PackedViewRecord, int32PackedViewSchema](&recordsFirst)
	queryFirst.Select(queryFirst.EmpresaID, queryFirst.ID, queryFirst.StatusTrace, queryFirst.Updated)
	queryFirst.EmpresaID.Equals(7)
	queryFirst.StatusTrace.Equals(int8(2))
	queryFirst.Updated.GreaterEqual(int32(15))

	recordsSecond := []int32PackedViewRecord{}
	querySecond := Query[int32PackedViewRecord, int32PackedViewSchema](&recordsSecond)
	querySecond.Select(querySecond.EmpresaID, querySecond.ID, querySecond.StatusTrace, querySecond.Updated)
	querySecond.EmpresaID.Equals(9)
	querySecond.StatusTrace.Equals(int8(4))
	querySecond.Updated.GreaterEqual(int32(21))

	hashFirst := computeSelectShapeHash(queryFirst.GetTableInfo(), scyllaTable)
	hashSecond := computeSelectShapeHash(querySecond.GetTableInfo(), scyllaTable)
	if hashFirst != hashSecond {
		t.Fatalf("expected the same select shape hash, got %d and %d", hashFirst, hashSecond)
	}

	planFirst, err := tryGetOrCompileSelectStatement(queryFirst.GetTableInfo(), scyllaTable)
	if err != nil {
		t.Fatalf("unexpected compile error on first shape: %v", err)
	}
	if planFirst == nil {
		t.Fatal("expected the first query shape to compile into a cached select statement")
	}

	planSecond, err := tryGetOrCompileSelectStatement(querySecond.GetTableInfo(), scyllaTable)
	if err != nil {
		t.Fatalf("unexpected compile error on second shape: %v", err)
	}
	if planSecond == nil {
		t.Fatal("expected the second query shape to reuse the cached select statement")
	}

	if planFirst != planSecond {
		t.Fatal("expected identical shapes to reuse the same compiled select statement pointer")
	}
}

func TestCompiledSelectStatementBindsPackedViewFanout(t *testing.T) {
	// Keep the bind test explicit so multi-statement packed-view expansion is validated without a live DB.
	resetORMTableCachesForTesting()

	scyllaTable := MakeScyllaTable[int32PackedViewRecord, int32PackedViewSchema]()
	scyllaTable.keyspace = "genix_test"

	records := []int32PackedViewRecord{}
	query := Query[int32PackedViewRecord, int32PackedViewSchema](&records)
	query.Select(query.EmpresaID, query.ID, query.StatusTrace, query.Updated)
	query.EmpresaID.Equals(7)
	query.StatusTrace.In(int8(2), int8(4))
	query.Updated.GreaterEqual(int32(15))

	compiledStatement, err := tryGetOrCompileSelectStatement(query.GetTableInfo(), scyllaTable)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if compiledStatement == nil {
		t.Fatal("expected the packed-view query to compile into a cached select statement")
	}
	if compiledStatement.route != selectRouteViewStatements {
		t.Fatalf("expected a view-backed cached route, got %d", compiledStatement.route)
	}
	if compiledStatement.sourceView == nil {
		t.Fatal("expected the cached select statement to keep the selected packed view")
	}

	boundPlan, err := compiledStatement.Compute(query.GetTableInfo(), scyllaTable)
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}
	if len(boundPlan.Statements) != 2 {
		t.Fatalf("expected 2 bound statements from packed-view fanout, got %d", len(boundPlan.Statements))
	}

	selectedStatements := pickStatementsByIndexes(collectSelectStatements(query.GetTableInfo()), compiledStatement.selectedStatementIndexes)
	expectedWhereStatements := compiledStatement.sourceView.getStatement(selectedStatements...)
	expectedQueries := map[string]bool{}
	for _, whereStatement := range expectedWhereStatements {
		expectedQueries[fmt.Sprintf(compiledStatement.queryTemplate, " WHERE "+whereStatement)] = true
	}

	for _, boundStatement := range boundPlan.Statements {
		if !expectedQueries[boundStatement.QueryStr] {
			t.Fatalf("unexpected bound query: %q", boundStatement.QueryStr)
		}
	}
}

func TestBuildBoundSelectPlanSplitsLargeInQueriesWithDefaultLimit(t *testing.T) {
	// Keep single-column IN queries below Scylla's clustering-key default restriction.
	t.Setenv("MAX_CLUSTERING_KEY", "")

	inValues := make([]any, 0, 120)
	for currentID := 1; currentID <= 120; currentID++ {
		inValues = append(inValues, currentID)
	}

	boundPlan := buildBoundSelectPlan(
		"SELECT id FROM genix.sale_order %v",
		nil,
		false,
		false,
		[]string{"company_id = 1"},
		[]ColumnStatement{{Col: "id", Operator: "IN", Values: inValues}},
		nil,
		nil,
		"",
		"",
		0,
		false,
	)

	if len(boundPlan.Statements) != 2 {
		t.Fatalf("expected 2 batched queries, got %d", len(boundPlan.Statements))
	}
	if !strings.Contains(boundPlan.Statements[0].QueryStr, "id IN (1, 2") {
		t.Fatalf("expected first batch to keep the first IDs, got %q", boundPlan.Statements[0].QueryStr)
	}
	if !strings.Contains(boundPlan.Statements[1].QueryStr, "id IN (101, 102") {
		t.Fatalf("expected second batch to continue after the default limit, got %q", boundPlan.Statements[1].QueryStr)
	}
}

func TestBuildBoundSelectPlanSplitsCartesianInQueriesByEnvLimit(t *testing.T) {
	// Split only enough to keep the IN cartesian product within the configured budget.
	t.Setenv("MAX_CLUSTERING_KEY", "6")

	boundPlan := buildBoundSelectPlan(
		"SELECT id FROM genix.sale_order %v",
		nil,
		false,
		false,
		[]string{"company_id = 1"},
		[]ColumnStatement{
			{Col: "status", Operator: "IN", Values: []any{1, 2, 3}},
			{Col: "id", Operator: "IN", Values: []any{10, 11, 12, 13}},
		},
		nil,
		nil,
		"",
		"",
		0,
		false,
	)

	if len(boundPlan.Statements) != 2 {
		t.Fatalf("expected 2 cartesian batches, got %d", len(boundPlan.Statements))
	}
	for _, boundStatement := range boundPlan.Statements {
		if !strings.Contains(boundStatement.QueryStr, "status IN (1, 2, 3)") {
			t.Fatalf("expected status IN clause to stay intact, got %q", boundStatement.QueryStr)
		}
	}
	if !strings.Contains(boundPlan.Statements[0].QueryStr, "id IN (10, 11)") {
		t.Fatalf("expected first cartesian batch to keep the first ID chunk, got %q", boundPlan.Statements[0].QueryStr)
	}
	if !strings.Contains(boundPlan.Statements[1].QueryStr, "id IN (12, 13)") {
		t.Fatalf("expected second cartesian batch to keep the second ID chunk, got %q", boundPlan.Statements[1].QueryStr)
	}
}

func TestCompiledSelectStatementSkipsWriteOnlyManagedColumns(t *testing.T) {
	resetORMTableCachesForTesting()

	scyllaTable := MakeScyllaTable[Increment, IncrementTable]()
	scyllaTable.keyspace = "genix_test"

	records := []Increment{}
	query := Query[Increment, IncrementTable](&records)
	query.Name.Equals("x1_sale_order_updated")

	compiledStatement, err := tryGetOrCompileSelectStatement(query.GetTableInfo(), scyllaTable)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if compiledStatement == nil {
		t.Fatal("expected a compiled select statement")
	}

	if strings.Contains(compiledStatement.queryTemplate, managedCreatedColumnName) {
		t.Fatalf("did not expect %q in query template: %s", managedCreatedColumnName, compiledStatement.queryTemplate)
	}
	if strings.Contains(compiledStatement.queryTemplate, managedUpdatedColumnName) {
		t.Fatalf("did not expect %q in query template: %s", managedUpdatedColumnName, compiledStatement.queryTemplate)
	}
	if strings.Contains(compiledStatement.queryTemplate, managedUpdateCounterColumnName) {
		t.Fatalf("did not expect %q in query template: %s", managedUpdateCounterColumnName, compiledStatement.queryTemplate)
	}
	if !strings.Contains(compiledStatement.queryTemplate, "FROM genix_test.sequences") {
		t.Fatalf("unexpected query template: %s", compiledStatement.queryTemplate)
	}
	if !strings.Contains(compiledStatement.queryTemplate, "name") || !strings.Contains(compiledStatement.queryTemplate, "current_value") {
		t.Fatalf("unexpected query template: %s", compiledStatement.queryTemplate)
	}
}
