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

func makeLookupToken(value any) string {
	return fmt.Sprintf("%v", value)
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
	scyllaTable ScyllaTable[any],
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
			partitionToken = makeLookupToken(partitionValue)
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

		keyToken := makeLookupToken(keyValue)
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

		if err := execQuery[E, T](queryTable, queryInfo, nil, nil); err != nil {
			return nil, err
		}

		foundByKeyToken := map[string]*T{}
		for resultIndex := range queryResult {
			foundRecord := &queryResult[resultIndex]
			foundPointer := xunsafe.AsPointer(foundRecord)
			foundKeyToken := makeLookupToken(keyColumn.GetRawValue(foundPointer))
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
			}
			continue
		}

		onInsertHandler(currentRecord)
		recordsToInsert = append(recordsToInsert, *currentRecord)
		insertRecordIndexes = append(insertRecordIndexes, recordIndex)
	}

	if len(recordsToUpdate) > 0 {
		fmt.Println("Merge | records to update:", len(recordsToUpdate))
		if err := UpdateExclude(&recordsToUpdate, columnsToExcludeUpdate...); err != nil {
			return err
		}
	}

	if len(recordsToInsert) > 0 {
		fmt.Println("Merge | records to insert:", len(recordsToInsert))
		if err := Insert(&recordsToInsert); err != nil {
			return err
		}

		// Insert may mutate records (e.g. autoincrement IDs), so propagate persisted values back.
		for insertIndex, originalRecordIndex := range insertRecordIndexes {
			(*records)[originalRecordIndex] = recordsToInsert[insertIndex]
		}
	}

	return nil
}
