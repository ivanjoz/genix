package db

import (
	"regexp"
	"strings"
	"testing"
)

type compositeBucketUpdateRecord struct {
	TableStruct[compositeBucketUpdateSchema, compositeBucketUpdateRecord]
	EmpresaID         int32   `db:"empresa_id"`
	ID                int64   `db:"id"`
	Fecha             int16   `db:"fecha"`
	DetailProductsIDs []int32 `db:"detail_products_ids,list"`
	Updated           int32   `db:"updated"`
}

type compositeBucketUpdateSchema struct {
	TableStruct[compositeBucketUpdateSchema, compositeBucketUpdateRecord]
	EmpresaID         Col[compositeBucketUpdateSchema, int32]
	ID                Col[compositeBucketUpdateSchema, int64]
	Fecha             Col[compositeBucketUpdateSchema, int16]
	DetailProductsIDs Col[compositeBucketUpdateSchema, []int32]
	Updated           Col[compositeBucketUpdateSchema, int32]
}

func (e compositeBucketUpdateSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "composite_bucket_update_test",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
		HashIndexes: [][]Coln{
			{e.DetailProductsIDs, e.Fecha.CompositeBucketing(2, 6)},
		},
	}
}

func TestMakeUpdateStatementsFormatsCompositeBucketHashesAsCQLList(t *testing.T) {
	// Regression: virtual composite hash columns must use CQL list syntax "[a,b]" (not Go "[a b]").
	resetORMTableCachesForTesting()

	recordRows := []compositeBucketUpdateRecord{
		{
			EmpresaID:         1,
			ID:                29672,
			Fecha:             20516,
			DetailProductsIDs: []int32{4, 1},
			Updated:           386426941,
		},
	}

	table := Table[compositeBucketUpdateRecord]()
	updateStatements := MakeUpdateStatements(&recordRows,
		table.Fecha,
		table.DetailProductsIDs,
		table.Updated,
	)

	if len(updateStatements) != 1 {
		t.Fatalf("expected 1 update statement, got=%d", len(updateStatements))
	}

	statement := updateStatements[0]
	if !strings.Contains(statement, "zz_hb_detail_products_ids_fecha_b2 = [") {
		t.Fatalf("expected virtual composite bucket column in update statement, got=%s", statement)
	}

	spaceSeparatedCollection := regexp.MustCompile(`zz_hb_[a-z0-9_]+ = \[[^,\]]+ [^\]]+\]`)
	if spaceSeparatedCollection.MatchString(statement) {
		t.Fatalf("expected comma-separated CQL list literal, got=%s", statement)
	}
}
