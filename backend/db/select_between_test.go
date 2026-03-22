package db

import (
	"reflect"
	"testing"
)

func TestBuildRemainingWhereClausesKeepsBetweenInclusive(t *testing.T) {
	clauses := buildRemainingWhereClauses([]ColumnStatement{
		{
			Col:      "unix_minutes_frame",
			Operator: "BETWEEN",
			From:     []ColumnStatement{{Col: "unix_minutes_frame", Value: int64(2580727)}},
			To:       []ColumnStatement{{Col: "unix_minutes_frame", Value: int64(2580739)}},
		},
		{Col: "status", Operator: "=", Value: int8(0)},
	})

	expectedClauses := []string{
		"unix_minutes_frame >= 2580727",
		"unix_minutes_frame <= 2580739",
		"status = 0",
	}

	// Lock the raw clause rendering so BETWEEN keeps inclusive upper bounds in Scylla queries.
	if !reflect.DeepEqual(clauses, expectedClauses) {
		t.Fatalf("unexpected clauses: got %v want %v", clauses, expectedClauses)
	}
}
