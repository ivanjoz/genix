package db

import (
	"slices"
	"strings"
	"testing"
)

type textSearchRecord struct {
	TableStruct[textSearchRecordTable, textSearchRecord]
	CompanyID int32  `db:"empresa_id"`
	ID        int32  `db:"id"`
	Name      string `db:"nombre"`
	Status    int8   `db:"status"`
}

type textSearchRecordTable struct {
	TableStruct[textSearchRecordTable, textSearchRecord]
	CompanyID Col[textSearchRecordTable, int32]
	ID        Col[textSearchRecordTable, int32]
	Name      Col[textSearchRecordTable, string]
	Status    Col[textSearchRecordTable, int8]
}

func (e textSearchRecordTable) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:         "test_keyspace",
		Name:             "text_search_records",
		Partition:        e.CompanyID,
		Keys:             []Coln{e.ID},
		TextSearchColumn: e.Name,
	}
}

func TestParseTextSearchWordsNormalizesAndCapsWords(t *testing.T) {
	words := parseTextSearchWords("Aceite de Oliva 100 Pack12 Ñandú para a Extra Uno Dos Tres Cuatro Cinco Seis Siete Ocho")

	expectedWords := []string{
		"aceite", "oliva", "100", "pack12", "nandu", "extra",
		"uno", "dos", "tres", "cuatro", "cinco", "seis",
	}
	if !slices.Equal(words, expectedWords) {
		t.Fatalf("unexpected normalized words: got %v want %v", words, expectedWords)
	}
}

func TestMakeTextSearchBigramsSkipsWordsWithNumbers(t *testing.T) {
	words := []string{"aceite", "100", "pack12", "oliva", "nandu"}
	bigrams := makeTextSearchBigrams(words)

	expectedBigrams := []uint8{BigramMap["ac"], BigramMap["ol"], BigramMap["na"]}
	if !slices.Equal(bigrams, expectedBigrams) {
		t.Fatalf("unexpected bigrams: got %v want %v", bigrams, expectedBigrams)
	}
}

func TestBuildTextSearchRowsUsesAllOrderedCombinations(t *testing.T) {
	rows := buildTextSearchRows(7, 99, 1, "aceite oliva extra")
	expectedRows := 7 // C(3,1) + C(3,2) + C(3,3)
	if len(rows) != expectedRows {
		t.Fatalf("expected %d rows, got %d: %+v", expectedRows, len(rows), rows)
	}

	expectedHash := BasicHashInt(strings.Join([]string{"aceite", "extra"}, textSearchHashWordBoundary))
	foundExpectedPair := false
	for _, row := range rows {
		if row.partitionID != 7 || row.id != 99 || row.status != 1 {
			t.Fatalf("unexpected row metadata: %+v", row)
		}
		if row.hash == expectedHash {
			foundExpectedPair = true
		}
		if len(row.bigrams) != 3 {
			t.Fatalf("expected one bigram per word, got %+v", row.bigrams)
		}
	}
	if !foundExpectedPair {
		t.Fatalf("expected non-consecutive pair hash %d in rows %+v", expectedHash, rows)
	}
}

func TestTextSearchIndexCompilesMetadataAndDDL(t *testing.T) {
	scyllaTable := MakeScyllaTable[textSearchRecord, textSearchRecordTable]()
	if scyllaTable.textSearchIndex == nil {
		t.Fatal("expected text search index metadata")
	}
	if scyllaTable.textSearchIndex.tableName != "text_search_records_nombre_search_idx" {
		t.Fatalf("unexpected text search table name: %s", scyllaTable.textSearchIndex.tableName)
	}
	if scyllaTable.textSearchIndex.statusColumn == nil || scyllaTable.textSearchIndex.statusColumn.GetName() != "status" {
		t.Fatalf("expected status column metadata, got %+v", scyllaTable.textSearchIndex.statusColumn)
	}

	createScript := getTextSearchIndexCreateScript("test_keyspace", scyllaTable.textSearchIndex)
	if !strings.Contains(createScript, "bigrams list<tinyint>") {
		t.Fatalf("expected bigrams list<tinyint> column: %s", createScript)
	}
	if !strings.Contains(createScript, "id int") {
		t.Fatalf("expected id column to use the base key type: %s", createScript)
	}
	if !strings.Contains(createScript, "PRIMARY KEY ((partition_id), hash, id)") {
		t.Fatalf("unexpected primary key: %s", createScript)
	}

	indexScript := getTextSearchIndexMaintenanceIndexCreateScript("test_keyspace", scyllaTable.textSearchIndex)
	if !strings.Contains(indexScript, "((partition_id), id)") {
		t.Fatalf("unexpected maintenance index script: %s", indexScript)
	}
}
