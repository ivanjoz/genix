package db

import (
	"strings"
	"testing"
)

type updateCounterRecord struct {
	TableStruct[updateCounterSchema, updateCounterRecord]
	EmpresaID int32  `db:"empresa_id"`
	ID        int64  `db:"id"`
	Updated   int64  `db:"updated"`
	Nombre    string `db:"nombre"`
}

type updateCounterSchema struct {
	TableStruct[updateCounterSchema, updateCounterRecord]
	EmpresaID Col[updateCounterSchema, int32]
	ID        Col[updateCounterSchema, int64]
	Updated   Col[updateCounterSchema, int64]
	Nombre    Col[updateCounterSchema, string]
}

func (e updateCounterSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:         "test_keyspace",
		Name:             "update_counter_records",
		Partition:        e.EmpresaID,
		Keys:             []Coln{e.ID},
		UseUpdateCounter: e.Updated,
	}
}

func TestApplyWriteManagedColumnsSharesUpdateCounterAcrossBatch(t *testing.T) {
	scyllaTable := MakeScyllaTable[updateCounterRecord, updateCounterSchema]()
	records := []updateCounterRecord{
		{EmpresaID: 7, ID: 1, Nombre: "uno"},
		{EmpresaID: 7, ID: 2, Nombre: "dos"},
	}

	originalCounterFetcher := getWriteCounterValue
	t.Cleanup(func() {
		getWriteCounterValue = originalCounterFetcher
	})

	calls := 0
	getWriteCounterValue = func(keyspace string, name string, increment int) (int64, error) {
		calls++
		if keyspace != "test_keyspace" {
			t.Fatalf("unexpected keyspace: %s", keyspace)
		}
		if name != "update_counter_records_update_counter" {
			t.Fatalf("unexpected counter name: %s", name)
		}
		if increment != 1 {
			t.Fatalf("unexpected counter increment: %d", increment)
		}
		return 41, nil
	}

	if err := applyWriteManagedColumns(&records, scyllaTable); err != nil {
		t.Fatalf("applyWriteManagedColumns returned error: %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected a single counter fetch, got %d", calls)
	}
	for _, record := range records {
		if record.Updated != 41 {
			t.Fatalf("expected shared update counter 41, got %d for record %d", record.Updated, record.ID)
		}
	}
}

func TestMakeUpdateStatementsBaseAlwaysPersistsManagedUpdateCounter(t *testing.T) {
	records := []updateCounterRecord{
		{EmpresaID: 7, ID: 1, Updated: 41, Nombre: "nuevo"},
	}
	table := Table[updateCounterRecord, updateCounterSchema]()

	updateStatements := makeUpdateStatementsBase(&records, []Coln{table.Nombre}, nil, false)
	if len(updateStatements) != 1 {
		t.Fatalf("expected one update statement, got %d", len(updateStatements))
	}

	if !strings.Contains(updateStatements[0], "nombre = 'nuevo'") {
		t.Fatalf("expected explicit column update in statement: %s", updateStatements[0])
	}
	if !strings.Contains(updateStatements[0], "updated = 41") {
		t.Fatalf("expected managed update counter in statement: %s", updateStatements[0])
	}
}

func TestMakeUpdateStatementsBaseIgnoresExcludedManagedUpdateCounter(t *testing.T) {
	records := []updateCounterRecord{
		{EmpresaID: 7, ID: 1, Updated: 41, Nombre: "nuevo"},
	}
	table := Table[updateCounterRecord, updateCounterSchema]()

	updateStatements := makeUpdateStatementsBase(&records, nil, []Coln{table.Updated}, false)
	if len(updateStatements) != 1 {
		t.Fatalf("expected one update statement, got %d", len(updateStatements))
	}

	if !strings.Contains(updateStatements[0], "updated = 41") {
		t.Fatalf("expected managed update counter in statement despite exclusion: %s", updateStatements[0])
	}
}

func TestMakeInsertStatementIgnoresExcludedManagedUpdateCounter(t *testing.T) {
	records := []updateCounterRecord{
		{EmpresaID: 7, ID: 1, Updated: 41, Nombre: "nuevo"},
	}
	table := Table[updateCounterRecord, updateCounterSchema]()

	insertStatements := MakeInsertStatement(&records, table.Updated)
	if len(insertStatements) != 1 {
		t.Fatalf("expected one insert statement, got %d", len(insertStatements))
	}

	if !strings.Contains(insertStatements[0], "updated") {
		t.Fatalf("expected managed update counter column in insert statement despite exclusion: %s", insertStatements[0])
	}
	if !strings.Contains(insertStatements[0], "41") {
		t.Fatalf("expected managed update counter value in insert statement despite exclusion: %s", insertStatements[0])
	}
}
