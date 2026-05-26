package db

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"app/db/text_search"

	"github.com/viant/xunsafe"
)

// configureTextSearchIndex validates that the ORM table declaring a
// TextSearchColumn has the shape the Sonic backend assumes: an int32
// partition (company_id), exactly one numeric primary key, a string
// source column, and an optional int8 status column. Schema mistakes
// panic at compile time of the table — they're never recoverable at
// runtime.
func configureTextSearchIndex(scyllaTable *ScyllaTable[any], schema TableSchema) {
	if schema.TextSearchColumn == nil {
		return
	}
	if scyllaTable.partKey == nil || scyllaTable.partKey.IsNil() {
		panic(fmt.Sprintf(`Table "%v": TextSearchColumn requires a partition column`, scyllaTable.name))
	}
	partitionFieldType := scyllaTable.partKey.GetType().FieldType
	if partitionFieldType != "int32" && partitionFieldType != "int" {
		panic(fmt.Sprintf(`Table "%v": TextSearchColumn partition column "%v" must be int32. Found: %v`,
			scyllaTable.name, scyllaTable.partKey.GetName(), partitionFieldType))
	}
	if len(scyllaTable.keys) != 1 {
		panic(fmt.Sprintf(`Table "%v": TextSearchColumn requires exactly one key column. Found: %v`, scyllaTable.name, len(scyllaTable.keys)))
	}

	sourceColumn := scyllaTable.columnsMap[schema.TextSearchColumn.GetName()]
	if sourceColumn == nil {
		panic(fmt.Sprintf(`Table "%v": TextSearchColumn "%v" was not found`, scyllaTable.name, schema.TextSearchColumn.GetName()))
	}
	if sourceColumn.GetType().FieldType != "string" {
		panic(fmt.Sprintf(`Table "%v": TextSearchColumn "%v" must be string. Found: %v`, scyllaTable.name, sourceColumn.GetName(), sourceColumn.GetType().FieldType))
	}

	idColumn := scyllaTable.keys[0]
	idFieldType := idColumn.GetType().FieldType
	if idFieldType != "int16" && idFieldType != "int32" && idFieldType != "int64" && idFieldType != "int" {
		panic(fmt.Sprintf(`Table "%v": TextSearchColumn key column "%v" must be numeric. Found: %v`, scyllaTable.name, idColumn.GetName(), idFieldType))
	}

	var statusColumn IColInfo
	if column, exists := scyllaTable.columnsMap["status"]; exists {
		if column.GetType().FieldType != "int8" {
			panic(fmt.Sprintf(`Table "%v": TextSearchColumn status column must be int8. Found: %v`, scyllaTable.name, column.GetType().FieldType))
		}
		statusColumn = column
	}

	scyllaTable.textSearchIndex = &textSearchIndexInfo{
		sourceColumn:    sourceColumn,
		partitionColumn: scyllaTable.partKey,
		idColumn:        idColumn,
		statusColumn:    statusColumn,
	}
}

// buildSearchText extracts the indexable text for one record. Records
// implementing textSearchIndexProvider override the default (a single
// column read) so they can stitch together denormalized fields like
// {name brand sku} before normalization. The result is always run
// through text_search.NormalizeSearchText so tokenize='ascii' gets the
// pre-tokenized input it expects.
func buildSearchText(record any, recordPointer unsafe.Pointer, info *textSearchIndexInfo) string {
	if provider, ok := record.(textSearchIndexProvider); ok {
		return text_search.NormalizeSearchText(provider.GetTextSearchIndex())
	}
	raw, _ := info.sourceColumn.GetRawValue(recordPointer).(string)
	return text_search.NormalizeSearchText(raw)
}

// textSearchAffectedColumns reports whether the columns touched by an
// update overlap with the indexed text column or the status column.
// Both flags drive different sync paths in insert-update.go.
func textSearchAffectedColumns(scyllaTable *ScyllaTable[any], affectedColumns []IColInfo) (bool, bool) {
	if scyllaTable.textSearchIndex == nil {
		return false, false
	}
	textChanged := false
	statusChanged := false
	for _, affectedColumn := range affectedColumns {
		if affectedColumn == nil {
			continue
		}
		if affectedColumn.GetName() == scyllaTable.textSearchIndex.sourceColumn.GetName() {
			textChanged = true
		}
		if scyllaTable.textSearchIndex.statusColumn != nil &&
			affectedColumn.GetName() == scyllaTable.textSearchIndex.statusColumn.GetName() {
			statusChanged = true
		}
	}
	return textChanged, statusChanged
}

// ftsBucket groups records that share a (partition, statusGroup) pair
// so we can issue one transactional batch per virtual table.
type ftsBucket struct {
	partition   int32
	statusGroup int8
	records     []text_search.Record
}

// syncTextSearchIndexAfterWrite upserts each record into its matching
// Sonic bucket (one per (partition, statusGroup) pair). The
// preserveExistingStatus flag is accepted for call-site compatibility
// but ignored here: status is encoded in the bucket name, and the
// Sonic upsert path already clears the opposite-status bucket on every
// write, so stale rows from a previous status group cannot linger.
func syncTextSearchIndexAfterWrite[T any](records *[]T, scyllaTable *ScyllaTable[any], _ bool) error {
	if scyllaTable.textSearchIndex == nil || records == nil || len(*records) == 0 {
		return nil
	}
	buckets := groupRecordsForTextSearch(records, scyllaTable)
	if len(buckets) == 0 {
		return nil
	}

	start := time.Now()
	totalRecords := 0
	ctx := context.Background()
	for _, bucket := range buckets {
		totalRecords += len(bucket.records)
		if err := text_search.UpsertBatch(ctx, scyllaTable.name, bucket.partition, bucket.statusGroup, bucket.records); err != nil {
			return fmt.Errorf("text search upsert %s_p%d_s%d: %w", scyllaTable.name, bucket.partition, bucket.statusGroup, err)
		}
	}
	if DebugFull {
		fmt.Printf("TextSearchIndex sync: table=%s buckets=%d records=%d elapsed=%s\n",
			scyllaTable.name, len(buckets), totalRecords, time.Since(start))
	}
	return nil
}

// syncTextSearchStatusAfterWrite reroutes records whose status changed
// into the Sonic bucket implied by the new status. Same upsert path as
// the general sync — the opposite-bucket FLUSHO inside UpsertBatch
// guarantees the stale row in the previous bucket is gone.
func syncTextSearchStatusAfterWrite[T any](records *[]T, scyllaTable *ScyllaTable[any]) error {
	return syncTextSearchIndexAfterWrite(records, scyllaTable, false)
}

// groupRecordsForTextSearch walks the record slice once and assembles
// the per-bucket Record lists used by UpsertBatch. Iteration order is
// not preserved across buckets, but it is preserved within each bucket,
// which matches the SQLite WAL writer's single-threaded commit order.
func groupRecordsForTextSearch[T any](records *[]T, scyllaTable *ScyllaTable[any]) []ftsBucket {
	info := scyllaTable.textSearchIndex
	indexByKey := map[uint32]int{}
	buckets := make([]ftsBucket, 0, 4)

	for i := range *records {
		recordPointer := xunsafe.AsPointer(&(*records)[i])
		partition := int32(scyllaTable.GetPartValue(recordPointer))
		recordID := convertToInt32(info.idColumn.GetRawValue(recordPointer))
		status := int8(0)
		if info.statusColumn != nil {
			status = int8(convertToInt64(info.statusColumn.GetRawValue(recordPointer)))
		}
		statusGroup := text_search.PickStatusGroup(status)
		searchText := buildSearchText(&(*records)[i], recordPointer, info)

		// Pack (partition, statusGroup) into a single uint32 key. Partition
		// is the company_id which is comfortably below 2^24 in practice;
		// statusGroup is 0 or 1.
		key := uint32(partition)<<1 | uint32(statusGroup&1)
		idx, exists := indexByKey[key]
		if !exists {
			idx = len(buckets)
			indexByKey[key] = idx
			buckets = append(buckets, ftsBucket{partition: partition, statusGroup: statusGroup})
		}
		buckets[idx].records = append(buckets[idx].records, text_search.Record{
			ID:         recordID,
			SearchText: searchText,
		})
	}
	return buckets
}
