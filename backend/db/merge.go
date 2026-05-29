package db

import (
	"fmt"

	"github.com/viant/xunsafe"
)

type bulkLookupRecord struct {
	recordKey string
	keyToken  string
}

type bulkLookupGroup struct {
	partitionValue   any
	keyValues        []any
	seenKeyValues    map[string]bool
	recordsToResolve []bulkLookupRecord
}

func isNonPositiveNumericValue(value any) bool {
	switch typedValue := value.(type) {
	case int:
		return typedValue <= 0
	case int8:
		return typedValue <= 0
	case int16:
		return typedValue <= 0
	case int32:
		return typedValue <= 0
	case int64:
		return typedValue <= 0
	}
	return false
}

func preloadExistingRecordsBySingleKey[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T,
	scyllaTable ScyllaTable,
) (map[string]*T, error) {
	existingByRecordKey := map[string]*T{}

	partitionColumn := scyllaTable.GetPartKey()
	keyColumn := scyllaTable.GetKeys()[0]
	queryTable := initStructTable[E, T](new(E))
	groupsByPartition := map[string]*bulkLookupGroup{}

	for recordIndex := range *records {
		currentRecord := &(*records)[recordIndex]
		recordPointer := xunsafe.AsPointer(currentRecord)
		recordKey := makePrimaryKeyRecordKey(recordPointer, scyllaTable)
		keyValue := keyColumn.GetRawValue(recordPointer)

		// Temporary numeric keys are always inserts, so skip DB lookups.
		if isNonPositiveNumericValue(keyValue) {
			continue
		}

		partitionValue := any(nil)
		partitionToken := "__no_partition__"
		if partitionColumn != nil {
			partitionValue = partitionColumn.GetRawValue(recordPointer)
			partitionToken = partitionColumn.GetValueString(recordPointer)
		}
		group, groupExists := groupsByPartition[partitionToken]
		if !groupExists {
			group = &bulkLookupGroup{
				partitionValue:   partitionValue,
				keyValues:        []any{},
				seenKeyValues:    map[string]bool{},
				recordsToResolve: []bulkLookupRecord{},
			}
			groupsByPartition[partitionToken] = group
		}

		keyToken := keyColumn.GetValueString(recordPointer)
		if !group.seenKeyValues[keyToken] {
			group.keyValues = append(group.keyValues, keyValue)
			group.seenKeyValues[keyToken] = true
		}
		group.recordsToResolve = append(group.recordsToResolve, bulkLookupRecord{
			recordKey: recordKey,
			keyToken:  keyToken,
		})
	}

	for _, group := range groupsByPartition {
		if len(group.keyValues) == 0 {
			continue
		}

		queryResult := []T{}
		statements := []ColumnStatement{
			{Col: keyColumn.GetName(), Operator: "IN", Values: group.keyValues},
		}
		if partitionColumn != nil {
			// Keep tenant isolation by constraining the optional partition key when present.
			statements = append(statements, ColumnStatement{
				Col: partitionColumn.GetName(), Operator: "=", Value: group.partitionValue,
			})
		}

		queryInfo := &TableInfo{
			refSlice:   &queryResult,
			statements: statements,
		}

		if err := execQuery[E, T](queryTable, queryInfo, nil); err != nil {
			return nil, err
		}

		foundByKeyToken := map[string]*T{}
		for resultIndex := range queryResult {
			foundRecord := &queryResult[resultIndex]
			foundPointer := xunsafe.AsPointer(foundRecord)
			foundKeyToken := keyColumn.GetValueString(foundPointer)
			if _, alreadyMapped := foundByKeyToken[foundKeyToken]; !alreadyMapped {
				foundByKeyToken[foundKeyToken] = foundRecord
			}
		}

		for _, lookupRecord := range group.recordsToResolve {
			if foundRecord, wasFound := foundByKeyToken[lookupRecord.keyToken]; wasFound {
				existingByRecordKey[lookupRecord.recordKey] = foundRecord
			}
		}
	}

	return existingByRecordKey, nil
}

// textSearchRecordChanged reports whether a record's searchable content or status group
// differs between the previously-stored record and the incoming one. When neither changed,
// the update-phase GenixSearch re-index for that record is redundant and can be skipped.
func textSearchRecordChanged[T any](info *textSearchIndexInfo, previous, current *T) bool {
	previousPointer := xunsafe.AsPointer(previous)
	currentPointer := xunsafe.AsPointer(current)

	if !info.sourceColumn.FieldsEqual(previousPointer, currentPointer) {
		return true
	}
	if info.statusColumn != nil &&
		!info.statusColumn.FieldsEqual(previousPointer, currentPointer) {
		return true
	}
	return false
}

func Merge[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T,
	columnsToExcludeUpdate []Coln,
	onUpdateHandler func(prev, current *T) bool, /* need update? */
	onInsertHandler func(e *T),
) error {
	if onUpdateHandler == nil || onInsertHandler == nil {
		panic("Merge requires non-nil onUpdateHandler and onInsertHandler")
	}

	if records == nil {
		return Err("merge received nil records pointer")
	}

	if len(*records) == 0 {
		return nil
	}

	queryTable := initStructTable[E, T](new(E))
	scyllaTable := getOrCompileScyllaTable(queryTable)
	if len(scyllaTable.GetKeys()) != 1 {
		return Err(`merge requires exactly one key column in schema`)
	}

	recordsToInsert := []T{}
	recordsToUpdate := []T{}
	insertRecordIndexes := []int{}
	skipTextSearchRecordsIDs := []int64{}
	textSearchInfo := scyllaTable.textSearchIndex
	existingByRecordKey, err := preloadExistingRecordsBySingleKey(records, scyllaTable)
	if err != nil {
		return Err("merge bulk lookup failed:", err)
	}

	for recordIndex := range *records {
		currentRecord := &(*records)[recordIndex]
		currentRecordPointer := xunsafe.AsPointer(currentRecord)
		recordKey := makePrimaryKeyRecordKey(currentRecordPointer, scyllaTable)

		if previousRecord, wasFound := existingByRecordKey[recordKey]; wasFound {
			if recordNeedsUpdate := onUpdateHandler(previousRecord, currentRecord); recordNeedsUpdate {
				recordsToUpdate = append(recordsToUpdate, *currentRecord)
				// We already hold the previous record, so decide here whether the searchable
				// content actually changed and let the write path skip unchanged re-indexes.
				if textSearchInfo != nil && !textSearchRecordChanged(textSearchInfo, previousRecord, currentRecord) {
					recordID := convertToInt64(scyllaTable.keys[0].GetRawValue(currentRecordPointer))
					skipTextSearchRecordsIDs = append(skipTextSearchRecordsIDs, recordID)
				}
			}
			continue
		}

		onInsertHandler(currentRecord)
		recordsToInsert = append(recordsToInsert, *currentRecord)
		insertRecordIndexes = append(insertRecordIndexes, recordIndex)
	}

	if len(recordsToInsert) == 0 && len(recordsToUpdate) == 0 {
		return nil
	}

	// Combine inserts and updates into a single batch so the merge issues one Scylla round-trip,
	// one managed-counter prefetch, one cache-version bump, and one text-search sync.
	fmt.Println("Merge | records to Insert:", len(recordsToInsert), "Update:", len(recordsToUpdate),"| Text skip:", len(skipTextSearchRecordsIDs))
	
	if err := executeInsertUpdateBatch(insertUpdateBatchParams[T, E]{
		recordsForInsert:         &recordsToInsert,
		recordsForUpdate:         &recordsToUpdate,
		columnsToUpdate:            columnsToExcludeUpdate,
		skipTextSearchRecordsIDs: skipTextSearchRecordsIDs,
	}); err != nil {
		return err
	}

	// Insert may mutate records (e.g. autoincrement IDs), so propagate persisted values back.
	for insertIndex, originalRecordIndex := range insertRecordIndexes {
		(*records)[originalRecordIndex] = recordsToInsert[insertIndex]
	}

	return nil
}
