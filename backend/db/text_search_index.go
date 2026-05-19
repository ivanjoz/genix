package db

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"github.com/gocql/gocql"
	"github.com/viant/xunsafe"
)

func configureTextSearchIndex(scyllaTable *ScyllaTable[any], schema TableSchema) {
	if schema.TextSearchColumn == nil {
		return
	}
	if scyllaTable.partKey == nil || scyllaTable.partKey.IsNil() {
		panic(fmt.Sprintf(`Table "%v": TextSearchColumn requires a partition column`, scyllaTable.name))
	}
	partitionFieldType := scyllaTable.partKey.GetType().FieldType
	if partitionFieldType != "int32" && partitionFieldType != "int" {
		panic(fmt.Sprintf(`Table "%v": TextSearchColumn partition column "%v" must be int32 because search partition_id is int. Found: %v`,
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
		tableName:       makeTextSearchIndexTableName(scyllaTable.name, sourceColumn.GetName()),
		sourceColumn:    sourceColumn,
		partitionColumn: scyllaTable.partKey,
		idColumn:        idColumn,
		statusColumn:    statusColumn,
	}
}

func makeTextSearchIndexTableName(tableName string, columnName string) string {
	return fmt.Sprintf("%s_%s_search_idx", tableName, columnName)
}

func getTextSearchIndexCreateScript(keyspace string, tableInfo *textSearchIndexInfo) string {
	return fmt.Sprintf(`CREATE TABLE %v.%v (
		partition_id int,
		hash int,
		bigrams list<tinyint>,
		status tinyint,
		id %v,
		PRIMARY KEY ((partition_id), hash, id)
	)
	%v;`,
		keyspace,
		tableInfo.tableName,
		getTextSearchIndexIDColumnType(tableInfo),
		makeStatementWith,
	)
}

func getTextSearchIndexIDColumnType(tableInfo *textSearchIndexInfo) string {
	return tableInfo.idColumn.GetType().ColType
}

func getTextSearchIndexMaintenanceIndexName(tableInfo *textSearchIndexInfo) string {
	return fmt.Sprintf(`%v__id_index`, tableInfo.tableName)
}

func getTextSearchIndexMaintenanceIndexCreateScript(keyspace string, tableInfo *textSearchIndexInfo) string {
	return fmt.Sprintf(`CREATE INDEX %v ON %v.%v ((partition_id), id)`,
		getTextSearchIndexMaintenanceIndexName(tableInfo),
		keyspace,
		tableInfo.tableName,
	)
}

func getTextSearchRecordRows(recordPointer unsafe.Pointer, textSearchIndex *textSearchIndexInfo) []textSearchIndexRow {
	partitionID := convertToInt64(textSearchIndex.partitionColumn.GetRawValue(recordPointer))
	baseID := convertToInt64(textSearchIndex.idColumn.GetRawValue(recordPointer))
	status := int8(0)
	if textSearchIndex.statusColumn != nil {
		status = int8(convertToInt64(textSearchIndex.statusColumn.GetRawValue(recordPointer)))
	}

	rawText, _ := textSearchIndex.sourceColumn.GetRawValue(recordPointer).(string)
	return buildTextSearchRows(partitionID, baseID, status, rawText)
}

func makeTextSearchRowKey(partitionID int64, baseID int64, hashValue int32) string {
	return fmt.Sprintf("%d|%d|%d", partitionID, baseID, hashValue)
}

func makeTextSearchRecordKey(partitionID int64, baseID int64) string {
	return fmt.Sprintf("%d|%d", partitionID, baseID)
}

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

func fetchExistingTextSearchRows(
	keyspace string,
	tableInfo *textSearchIndexInfo,
	partitionID int64,
	idValues []int64,
) (map[string]textSearchIndexRow, error) {
	existingRows := map[string]textSearchIndexRow{}
	if len(idValues) == 0 {
		return existingRows, nil
	}

	maxIDsPerQuery := getMaxClusteringKeyRestrictionsPerQuery()
	if maxIDsPerQuery <= 0 {
		maxIDsPerQuery = 100
	}
	slices.Sort(idValues)

	for startIndex := 0; startIndex < len(idValues); startIndex += maxIDsPerQuery {
		endIndex := startIndex + maxIDsPerQuery
		if endIndex > len(idValues) {
			endIndex = len(idValues)
		}

		currentIDs := idValues[startIndex:endIndex]
		placeholders := make([]string, 0, len(currentIDs))
		queryArguments := make([]any, 0, len(currentIDs)+1)
		queryArguments = append(queryArguments, int32(partitionID))
		for _, idValue := range currentIDs {
			placeholders = append(placeholders, "?")
			queryArguments = append(queryArguments, makeNumericQueryValue(tableInfo.idColumn, idValue))
		}

		query := fmt.Sprintf(`SELECT hash, id, bigrams, status FROM %v.%v WHERE partition_id = ? AND id IN (%v)`,
			keyspace,
			tableInfo.tableName,
			strings.Join(placeholders, ", "),
		)

		iter := getScyllaConnection().Query(query, queryArguments...).Iter()
		rowData, err := iter.RowData()
		if err != nil {
			return nil, err
		}

		scanner := iter.Scanner()
		for scanner.Next() {
			rowValues := rowData.Values
			if err := scanner.Scan(rowValues...); err != nil {
				return nil, err
			}

			hashValue := convertToInt32(normalizeScannedValue(rowValues[0]))
			baseID := convertToInt64(normalizeScannedValue(rowValues[1]))
			bigrams, _ := normalizeScannedValue(rowValues[2]).([]int8)
			status := int8(convertToInt64(normalizeScannedValue(rowValues[3])))
			row := textSearchIndexRow{
				partitionID: partitionID,
				id:          baseID,
				hash:        hashValue,
				bigrams:     slices.Clone(bigrams),
				status:      status,
			}
			existingRows[makeTextSearchRowKey(partitionID, baseID, hashValue)] = row
		}
		if err := iter.Close(); err != nil {
			return nil, err
		}
	}

	return existingRows, nil
}

func syncTextSearchIndexAfterWrite[T any](records *[]T, scyllaTable *ScyllaTable[any], preserveExistingStatus bool) error {
	if scyllaTable.textSearchIndex == nil || records == nil || len(*records) == 0 {
		return nil
	}

	newRowsByKey := map[string]textSearchIndexRow{}
	idValuesByPartition := map[int64]map[int64]struct{}{}

	for recordIndex := range *records {
		recordPointer := xunsafe.AsPointer(&(*records)[recordIndex])
		partitionID := scyllaTable.GetPartValue(recordPointer)
		baseID := convertToInt64(scyllaTable.keys[0].GetRawValue(recordPointer))
		if _, exists := idValuesByPartition[partitionID]; !exists {
			idValuesByPartition[partitionID] = map[int64]struct{}{}
		}
		idValuesByPartition[partitionID][baseID] = struct{}{}

		for _, row := range getTextSearchRecordRows(recordPointer, scyllaTable.textSearchIndex) {
			newRowsByKey[makeTextSearchRowKey(row.partitionID, row.id, row.hash)] = row
		}
	}

	existingRowsByKey := map[string]textSearchIndexRow{}
	for partitionID, idSet := range idValuesByPartition {
		idValues := make([]int64, 0, len(idSet))
		for idValue := range idSet {
			idValues = append(idValues, idValue)
		}

		existingRows, err := fetchExistingTextSearchRows(scyllaTable.keyspace, scyllaTable.textSearchIndex, partitionID, idValues)
		if err != nil {
			return fmt.Errorf("text search fetch existing rows %s: %w", scyllaTable.textSearchIndex.tableName, err)
		}
		for rowKey, row := range existingRows {
			existingRowsByKey[rowKey] = row
		}
	}

	if preserveExistingStatus {
		statusByRecordKey := map[string]int8{}
		for _, existingRow := range existingRowsByKey {
			statusByRecordKey[makeTextSearchRecordKey(existingRow.partitionID, existingRow.id)] = existingRow.status
		}
		for rowKey, newRow := range newRowsByKey {
			if existingStatus, exists := statusByRecordKey[makeTextSearchRecordKey(newRow.partitionID, newRow.id)]; exists {
				newRow.status = existingStatus
				newRowsByKey[rowKey] = newRow
			}
		}
	}

	session := getScyllaConnection()
	batch := session.NewBatch(gocql.UnloggedBatch)
	deleteStatementsCount := 0
	insertStatementsCount := 0
	updateStatementsCount := 0

	deleteQuery := fmt.Sprintf(`DELETE FROM %v.%v WHERE partition_id = ? AND hash = ? AND id = ?`,
		scyllaTable.keyspace,
		scyllaTable.textSearchIndex.tableName,
	)
	for rowKey, existingRow := range existingRowsByKey {
		if _, shouldKeepRow := newRowsByKey[rowKey]; shouldKeepRow {
			continue
		}
		batch.Query(deleteQuery, int32(existingRow.partitionID), existingRow.hash, makeNumericQueryValue(scyllaTable.textSearchIndex.idColumn, existingRow.id))
		deleteStatementsCount++
	}

	insertQuery := fmt.Sprintf(`INSERT INTO %v.%v (partition_id, hash, bigrams, status, id) VALUES (?, ?, ?, ?, ?)`,
		scyllaTable.keyspace,
		scyllaTable.textSearchIndex.tableName,
	)
	for rowKey, newRow := range newRowsByKey {
		existingRow, rowExists := existingRowsByKey[rowKey]
		if rowExists && existingRow.status == newRow.status && slices.Equal(existingRow.bigrams, newRow.bigrams) {
			continue
		}
		batch.Query(insertQuery, int32(newRow.partitionID), newRow.hash, newRow.bigrams, newRow.status, makeNumericQueryValue(scyllaTable.textSearchIndex.idColumn, newRow.id))
		if rowExists {
			updateStatementsCount++
		} else {
			insertStatementsCount++
		}
	}

	if deleteStatementsCount == 0 && insertStatementsCount == 0 && updateStatementsCount == 0 {
		return nil
	}

	if DebugFull {
		fmt.Printf("TextSearchIndex sync batch: base_table=%s index=%s deletes=%d inserts=%d updates=%d records=%d\n",
			scyllaTable.name, scyllaTable.textSearchIndex.tableName, deleteStatementsCount, insertStatementsCount, updateStatementsCount, len(*records))
	}

	if err := session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("text search index sync %s: %w", scyllaTable.textSearchIndex.tableName, err)
	}

	return nil
}

func syncTextSearchStatusAfterWrite[T any](records *[]T, scyllaTable *ScyllaTable[any]) error {
	if scyllaTable.textSearchIndex == nil || scyllaTable.textSearchIndex.statusColumn == nil || records == nil || len(*records) == 0 {
		return nil
	}

	idValuesByPartition := map[int64]map[int64]struct{}{}
	statusByRecordKey := map[string]int8{}
	for recordIndex := range *records {
		recordPointer := xunsafe.AsPointer(&(*records)[recordIndex])
		partitionID := scyllaTable.GetPartValue(recordPointer)
		baseID := convertToInt64(scyllaTable.keys[0].GetRawValue(recordPointer))
		if _, exists := idValuesByPartition[partitionID]; !exists {
			idValuesByPartition[partitionID] = map[int64]struct{}{}
		}
		idValuesByPartition[partitionID][baseID] = struct{}{}
		statusByRecordKey[makeTextSearchRecordKey(partitionID, baseID)] = int8(convertToInt64(scyllaTable.textSearchIndex.statusColumn.GetRawValue(recordPointer)))
	}

	existingRowsByKey := map[string]textSearchIndexRow{}
	for partitionID, idSet := range idValuesByPartition {
		idValues := make([]int64, 0, len(idSet))
		for idValue := range idSet {
			idValues = append(idValues, idValue)
		}

		existingRows, err := fetchExistingTextSearchRows(scyllaTable.keyspace, scyllaTable.textSearchIndex, partitionID, idValues)
		if err != nil {
			return fmt.Errorf("text search fetch status rows %s: %w", scyllaTable.textSearchIndex.tableName, err)
		}
		for rowKey, row := range existingRows {
			existingRowsByKey[rowKey] = row
		}
	}

	session := getScyllaConnection()
	batch := session.NewBatch(gocql.UnloggedBatch)
	updateStatementsCount := 0
	updateQuery := fmt.Sprintf(`UPDATE %v.%v SET status = ? WHERE partition_id = ? AND hash = ? AND id = ?`,
		scyllaTable.keyspace,
		scyllaTable.textSearchIndex.tableName,
	)

	for _, existingRow := range existingRowsByKey {
		nextStatus, exists := statusByRecordKey[makeTextSearchRecordKey(existingRow.partitionID, existingRow.id)]
		if !exists || existingRow.status == nextStatus {
			continue
		}
		batch.Query(updateQuery, nextStatus, int32(existingRow.partitionID), existingRow.hash, makeNumericQueryValue(scyllaTable.textSearchIndex.idColumn, existingRow.id))
		updateStatementsCount++
	}

	if updateStatementsCount == 0 {
		return nil
	}

	if DebugFull {
		fmt.Printf("TextSearchIndex status sync batch: base_table=%s index=%s updates=%d records=%d\n",
			scyllaTable.name, scyllaTable.textSearchIndex.tableName, updateStatementsCount, len(*records))
	}

	if err := session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("text search status sync %s: %w", scyllaTable.textSearchIndex.tableName, err)
	}

	return nil
}
