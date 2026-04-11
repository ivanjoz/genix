package db

import "testing"

func TestFormatDebugQueryMinifiesProjectionAndInjectsValues(t *testing.T) {
	queryStr := "SELECT col_a, col_b, fn(col_c, col_d) FROM sale_order WHERE company_id = ? AND status IN (?, ?) AND customer_name = ?"
	queryValues := []any{int32(7), int8(2), int8(3), "O'Hara"}

	formattedQuery := formatDebugQuery(queryStr, queryValues)
	expectedQuery := "SELECT (3) FROM sale_order WHERE company_id = 7 AND status IN (2, 3) AND customer_name = 'O''Hara'"

	if formattedQuery != expectedQuery {
		t.Fatalf("unexpected debug query.\nexpected: %s\nactual:   %s", expectedQuery, formattedQuery)
	}
}

func TestFormatDebugQueryPreservesExtraValuesComment(t *testing.T) {
	queryStr := "SELECT col_a FROM sale_order WHERE company_id = ?"
	queryValues := []any{int32(7), "extra"}

	formattedQuery := formatDebugQuery(queryStr, queryValues)
	expectedQuery := "SELECT (1) FROM sale_order WHERE company_id = 7 /* extra_values=['extra'] */"

	if formattedQuery != expectedQuery {
		t.Fatalf("unexpected debug query.\nexpected: %s\nactual:   %s", expectedQuery, formattedQuery)
	}
}
