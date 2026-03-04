package db

import (
	"testing"

	"github.com/viant/xunsafe"
)

var ormBenchmarkSink any

type ormBenchmarkRecord struct {
	TableStruct[ormBenchmarkSchema, ormBenchmarkRecord]
	EmpresaID int32    `db:"empresa_id"`
	ID        int64    `db:"id"`
	SKU       string   `db:"sku"`
	Updated   int64    `db:"updated"`
	Active    bool     `db:"active"`
	Tags      []string `db:"tags"`
}

type ormBenchmarkSchema struct {
	TableStruct[ormBenchmarkSchema, ormBenchmarkRecord]
	EmpresaID Col[ormBenchmarkSchema, int32]
	ID        Col[ormBenchmarkSchema, int64]
	SKU       Col[ormBenchmarkSchema, string]
	Updated   Col[ormBenchmarkSchema, int64]
	Active    Col[ormBenchmarkSchema, bool]
	Tags      Col[ormBenchmarkSchema, []string]
}

func (e ormBenchmarkSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "orm_benchmark",
		Keys:      []Coln{e.ID},
		Partition: e.EmpresaID,
	}
}

func ptrTo[T any](value T) *T {
	return &value
}

func benchmarkTableAndColumns() (ScyllaTable[any], []string, []any) {
	schemaTable := initStructTable[ormBenchmarkSchema, ormBenchmarkRecord](new(ormBenchmarkSchema))
	scyllaTable := getOrCompileScyllaTable(schemaTable)

	columnNames := []string{"empresa_id", "id", "sku", "updated", "active", "tags"}
	rowValues := []any{
		ptrTo(int32(7)),
		ptrTo(int64(101)),
		ptrTo("sku-bench"),
		ptrTo(int64(1700000000)),
		ptrTo(true),
		&[]string{"tag-1", "tag-2"},
	}

	return scyllaTable, columnNames, rowValues
}

func BenchmarkORMInitStructAndTableWarm(b *testing.B) {
	// Warm-cache benchmark measures steady-state query setup overhead.
	resetORMTableCachesForTesting()
	seedSchema := initStructTable[ormBenchmarkSchema, ormBenchmarkRecord](new(ormBenchmarkSchema))
	_ = getOrCompileScyllaTable(seedSchema)

	b.ReportAllocs()
	b.ResetTimer()
	for benchmarkIndex := 0; benchmarkIndex < b.N; benchmarkIndex++ {
		schemaTable := initStructTable[ormBenchmarkSchema, ormBenchmarkRecord](new(ormBenchmarkSchema))
		ormBenchmarkSink = getOrCompileScyllaTable(schemaTable)
	}
}

func BenchmarkORMInitStructAndTableCold(b *testing.B) {
	// Cold benchmark forces a full metadata rebuild every iteration.
	b.ReportAllocs()
	b.ResetTimer()
	for benchmarkIndex := 0; benchmarkIndex < b.N; benchmarkIndex++ {
		resetORMTableCachesForTesting()
		schemaTable := initStructTable[ormBenchmarkSchema, ormBenchmarkRecord](new(ormBenchmarkSchema))
		ormBenchmarkSink = getOrCompileScyllaTable(schemaTable)
	}
}

func BenchmarkORMSelectRowMapping(b *testing.B) {
	// Row-mapping benchmark isolates scan-side SetValue costs without network I/O.
	resetORMTableCachesForTesting()
	scyllaTable, columnNames, rowValues := benchmarkTableAndColumns()

	b.ReportAllocs()
	b.ResetTimer()
	for benchmarkIndex := 0; benchmarkIndex < b.N; benchmarkIndex++ {
		record := new(ormBenchmarkRecord)
		recordPointer := xunsafe.AsPointer(record)
		for columnIndex, columnName := range columnNames {
			column := scyllaTable.columnsMap[columnName]
			column.SetValue(recordPointer, rowValues[columnIndex])
		}
		ormBenchmarkSink = record
	}
}

func BenchmarkORMInsertStatementValueExtraction(b *testing.B) {
	// Insert benchmark measures value extraction from typed structs into statement args.
	resetORMTableCachesForTesting()
	scyllaTable, _, _ := benchmarkTableAndColumns()

	records := make([]ormBenchmarkRecord, 512)
	for recordIndex := range records {
		records[recordIndex].EmpresaID = int32(recordIndex%20 + 1)
		records[recordIndex].ID = int64(recordIndex + 1)
		records[recordIndex].SKU = "sku-bench"
		records[recordIndex].Updated = int64(1700000000 + recordIndex)
		records[recordIndex].Active = recordIndex%2 == 0
		records[recordIndex].Tags = []string{"tag-1", "tag-2"}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for benchmarkIndex := 0; benchmarkIndex < b.N; benchmarkIndex++ {
		record := &records[benchmarkIndex%len(records)]
		recordPointer := xunsafe.AsPointer(record)

		values := make([]any, 0, len(scyllaTable.columns))
		for _, column := range scyllaTable.columns {
			values = append(values, column.GetStatementValue(recordPointer))
		}
		ormBenchmarkSink = values
	}
}
