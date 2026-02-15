package db

import (
	"fmt"
	"unsafe"

	"github.com/viant/xunsafe"
)

func makeMergeKeyStatements(recordPointer unsafe.Pointer, scyllaTable ScyllaTable[any]) []ColumnStatement {
	statements := []ColumnStatement{}
	columnsAdded := map[string]bool{}

	addEqualsStatement := func(column IColInfo) {
		if column == nil || column.IsNil() {
			return
		}

		columnName := column.GetName()
		if columnsAdded[columnName] {
			return
		}

		columnsAdded[columnName] = true
		statements = append(statements, ColumnStatement{
			Col:      columnName,
			Operator: "=",
			Value:    column.GetRawValue(recordPointer),
		})
	}

	// A record identity is partition + key columns, so merge lookups must include both.
	addEqualsStatement(scyllaTable.GetPartKey())
	for _, keyColumn := range scyllaTable.GetKeys() {
		addEqualsStatement(keyColumn)
	}

	return statements
}

func findExistingRecordByKey[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	record *T,
	scyllaTable ScyllaTable[any],
) (T, bool, error) {
	recordPointer := xunsafe.AsPointer(record)
	queryResult := []T{}
	queryInfo := &TableInfo{
		refSlice:   &queryResult,
		statements: makeMergeKeyStatements(recordPointer, scyllaTable),
		limit:      1,
	}

	queryTable := initStructTable[E, T](new(E))
	if err := execQuery[E, T](queryTable, queryInfo); err != nil {
		return *new(T), false, err
	}

	if len(queryResult) == 0 {
		return *new(T), false, nil
	}

	return queryResult[0], true, nil
}

func Merge[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T,
	columnsToExcludeUpdate []Coln,
	onUpdateHandler func(prev, current *T) bool /* need update? */,
	onInsertHandler func(e *T),
) error {
	if records == nil {
		return Err("merge received nil records pointer")
	}

	if len(*records) == 0 {
		return nil
	}

	queryTable := initStructTable[E, T](new(E))
	scyllaTable := makeTable(queryTable)
	if len(scyllaTable.GetKeys()) == 0 {
		return Err(`merge requires at least one key column in schema`)
	}

	recordsToInsert := []T{}
	recordsToUpdate := []T{}
	insertRecordIndexes := []int{}
	existingByRecordKey := map[string]T{}
	missingByRecordKey := map[string]bool{}

	for recordIndex := range *records {
		currentRecord := &(*records)[recordIndex]
		currentRecordPointer := xunsafe.AsPointer(currentRecord)
		recordKey := makePrimaryKeyRecordKey(currentRecordPointer, scyllaTable)

		previousRecord, wasFound := existingByRecordKey[recordKey]
		wasMissing := missingByRecordKey[recordKey]

		if !wasFound && !wasMissing {
			existingRecord, existsInDatabase, err := findExistingRecordByKey(currentRecord, scyllaTable)
			if err != nil {
				return Err("merge lookup failed for key", recordKey, ":", err)
			}

			if existsInDatabase {
				existingByRecordKey[recordKey] = existingRecord
				previousRecord = existingRecord
				wasFound = true
			} else {
				missingByRecordKey[recordKey] = true
				wasMissing = true
			}
		}

		if wasFound {
			recordNeedsUpdate := true
			if onUpdateHandler != nil {
				previousRecordCopy := previousRecord
				recordNeedsUpdate = onUpdateHandler(&previousRecordCopy, currentRecord)
			}

			if recordNeedsUpdate {
				recordsToUpdate = append(recordsToUpdate, *currentRecord)
			}
			continue
		}

		if wasMissing {
			if onInsertHandler != nil {
				onInsertHandler(currentRecord)
			}
			recordsToInsert = append(recordsToInsert, *currentRecord)
			insertRecordIndexes = append(insertRecordIndexes, recordIndex)
		}
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
