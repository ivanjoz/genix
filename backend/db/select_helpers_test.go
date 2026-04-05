package db

import (
	"fmt"
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
