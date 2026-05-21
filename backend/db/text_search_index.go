package db

import (
	"fmt"
	"slices"
	"strings"
	"time"
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

func getTextSearchRecordRows(record any, recordPointer unsafe.Pointer, textSearchIndex *textSearchIndexInfo) []textSearchIndexRow {
	partitionID := convertToInt64(textSearchIndex.partitionColumn.GetRawValue(recordPointer))
	baseID := convertToInt64(textSearchIndex.idColumn.GetRawValue(recordPointer))
	status := int8(0)
	if textSearchIndex.statusColumn != nil {
		status = int8(convertToInt64(textSearchIndex.statusColumn.GetRawValue(recordPointer)))
	}

	rawText := getTextSearchRecordText(record, recordPointer, textSearchIndex)
	return buildTextSearchRows(partitionID, baseID, status, rawText)
}

func getTextSearchRecordText(record any, recordPointer unsafe.Pointer, textSearchIndex *textSearchIndexInfo) string {
	if textSearchProvider, hasTextSearchProvider := record.(textSearchIndexProvider); hasTextSearchProvider {
		// Custom providers can combine denormalized fields that are not stored in the indexed column.
		return textSearchProvider.GetTextSearchIndex()
	}

	rawText, _ := textSearchIndex.sourceColumn.GetRawValue(recordPointer).(string)
	return rawText
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
	session := getScyllaConnection()

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

		// The index table's PRIMARY KEY is ((partition_id), hash, id), so `id` cannot be restricted
		// without `hash`. The local secondary index on ((partition_id), id) is used to resolve the
		// `IN` predicate; `ALLOW FILTERING` is required for the planner to accept it.
		query := fmt.Sprintf(`SELECT hash, id, bigrams, status FROM %v.%v WHERE partition_id = ? AND id IN (%v) ALLOW FILTERING`,
			keyspace,
			tableInfo.tableName,
			strings.Join(placeholders, ", "),
		)

		chunkStart := time.Now()
		iter := session.Query(query, queryArguments...).Iter()
		rowData, err := iter.RowData()
		if err != nil {
			return nil, err
		}

		chunkRows := 0
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
			chunkRows++
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
		fmt.Printf("TextSearchIndex fetch chunk: index=%s partition=%d ids=%d-%d (of %d) rows=%d elapsed=%s\n",
			tableInfo.tableName, partitionID, startIndex, endIndex, len(idValues), chunkRows, time.Since(chunkStart))
	}

	return existingRows, nil
}

func syncTextSearchIndexAfterWrite[T any](records *[]T, scyllaTable *ScyllaTable[any], preserveExistingStatus bool) error {
	if scyllaTable.textSearchIndex == nil || records == nil || len(*records) == 0 {
		return nil
	}

	fmt.Printf("TextSearchIndex sync start: index=%s records=%d\n",
		scyllaTable.textSearchIndex.tableName, len(*records))

	newRowsByKey := map[string]textSearchIndexRow{}
	idValuesByPartition := map[int64]map[int64]struct{}{}

	buildStart := time.Now()
	for recordIndex := range *records {
		recordPointer := xunsafe.AsPointer(&(*records)[recordIndex])
		partitionID := scyllaTable.GetPartValue(recordPointer)
		baseID := convertToInt64(scyllaTable.keys[0].GetRawValue(recordPointer))
		if _, exists := idValuesByPartition[partitionID]; !exists {
			idValuesByPartition[partitionID] = map[int64]struct{}{}
		}
		idValuesByPartition[partitionID][baseID] = struct{}{}

		for _, row := range getTextSearchRecordRows(&(*records)[recordIndex], recordPointer, scyllaTable.textSearchIndex) {
			newRowsByKey[makeTextSearchRowKey(row.partitionID, row.id, row.hash)] = row
		}
	}
	fmt.Printf("TextSearchIndex built new rows: index=%s partitions=%d new_rows=%d elapsed=%s\n",
		scyllaTable.textSearchIndex.tableName, len(idValuesByPartition), len(newRowsByKey), time.Since(buildStart))

	existingRowsByKey := map[string]textSearchIndexRow{}
	fetchStart := time.Now()
	for partitionID, idSet := range idValuesByPartition {
		idValues := make([]int64, 0, len(idSet))
		for idValue := range idSet {
			idValues = append(idValues, idValue)
		}

		partitionFetchStart := time.Now()
		existingRows, err := fetchExistingTextSearchRows(scyllaTable.keyspace, scyllaTable.textSearchIndex, partitionID, idValues)
		if err != nil {
			return fmt.Errorf("text search fetch existing rows %s: %w", scyllaTable.textSearchIndex.tableName, err)
		}
		fmt.Printf("TextSearchIndex fetched existing rows: index=%s partition=%d ids=%d rows=%d elapsed=%s\n",
			scyllaTable.textSearchIndex.tableName, partitionID, len(idValues), len(existingRows), time.Since(partitionFetchStart))
		for rowKey, row := range existingRows {
			existingRowsByKey[rowKey] = row
		}
	}
	fmt.Printf("TextSearchIndex fetched all existing rows: index=%s total_rows=%d elapsed=%s\n",
		scyllaTable.textSearchIndex.tableName, len(existingRowsByKey), time.Since(fetchStart))

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
	statements := make([]textSearchBatchStatement, 0, len(existingRowsByKey)+len(newRowsByKey))
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
		statements = append(statements, textSearchBatchStatement{
			query: deleteQuery,
			args:  []any{int32(existingRow.partitionID), existingRow.hash, makeNumericQueryValue(scyllaTable.textSearchIndex.idColumn, existingRow.id)},
		})
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
		statements = append(statements, textSearchBatchStatement{
			query: insertQuery,
			args:  []any{int32(newRow.partitionID), newRow.hash, newRow.bigrams, newRow.status, makeNumericQueryValue(scyllaTable.textSearchIndex.idColumn, newRow.id)},
		})
		if rowExists {
			updateStatementsCount++
		} else {
			insertStatementsCount++
		}
	}

	if len(statements) == 0 {
		fmt.Printf("TextSearchIndex sync no-op: index=%s records=%d\n",
			scyllaTable.textSearchIndex.tableName, len(*records))
		return nil
	}

	fmt.Printf("TextSearchIndex sync batch: base_table=%s index=%s deletes=%d inserts=%d updates=%d records=%d total_statements=%d\n",
		scyllaTable.name, scyllaTable.textSearchIndex.tableName, deleteStatementsCount, insertStatementsCount, updateStatementsCount, len(*records), len(statements))

	execStart := time.Now()
	if err := executeTextSearchBatches(session, statements); err != nil {
		return fmt.Errorf("text search index sync %s: %w", scyllaTable.textSearchIndex.tableName, err)
	}
	fmt.Printf("TextSearchIndex sync finished: index=%s statements=%d elapsed=%s\n",
		scyllaTable.textSearchIndex.tableName, len(statements), time.Since(execStart))

	return nil
}

// textSearchBatchStatement holds a single prepared write that will be appended to one of the
// chunked UnloggedBatch executions used to bypass Scylla's per-batch statement limit.
type textSearchBatchStatement struct {
	query string
	args  []any
}

// executeTextSearchBatches splits derived-row writes across multiple UnloggedBatch executions so
// records that produce hundreds of hash combinations never exceed Scylla's per-batch cap.
func executeTextSearchBatches(session *gocql.Session, statements []textSearchBatchStatement) error {
	if len(statements) == 0 {
		return nil
	}
	const maxStatementsPerBatch = 2000
	for startIndex := 0; startIndex < len(statements); startIndex += maxStatementsPerBatch {
		endIndex := startIndex + maxStatementsPerBatch
		if endIndex > len(statements) {
			endIndex = len(statements)
		}
		batch := session.NewBatch(gocql.UnloggedBatch)
		for _, statement := range statements[startIndex:endIndex] {
			batch.Query(statement.query, statement.args...)
		}
		fmt.Printf("TextSearchIndex sending batch: %d / %d\n", endIndex, len(statements))
		batchStart := time.Now()
		if err := session.ExecuteBatch(batch); err != nil {
			return err
		}
		fmt.Printf("TextSearchIndex batch sent: %d / %d elapsed=%s\n", endIndex, len(statements), time.Since(batchStart))
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
	statements := make([]textSearchBatchStatement, 0, len(existingRowsByKey))
	updateQuery := fmt.Sprintf(`UPDATE %v.%v SET status = ? WHERE partition_id = ? AND hash = ? AND id = ?`,
		scyllaTable.keyspace,
		scyllaTable.textSearchIndex.tableName,
	)

	for _, existingRow := range existingRowsByKey {
		nextStatus, exists := statusByRecordKey[makeTextSearchRecordKey(existingRow.partitionID, existingRow.id)]
		if !exists || existingRow.status == nextStatus {
			continue
		}
		statements = append(statements, textSearchBatchStatement{
			query: updateQuery,
			args:  []any{nextStatus, int32(existingRow.partitionID), existingRow.hash, makeNumericQueryValue(scyllaTable.textSearchIndex.idColumn, existingRow.id)},
		})
	}

	if len(statements) == 0 {
		return nil
	}

	if DebugFull {
		fmt.Printf("TextSearchIndex status sync batch: base_table=%s index=%s updates=%d records=%d\n",
			scyllaTable.name, scyllaTable.textSearchIndex.tableName, len(statements), len(*records))
	}

	if err := executeTextSearchBatches(session, statements); err != nil {
		return fmt.Errorf("text search status sync %s: %w", scyllaTable.textSearchIndex.tableName, err)
	}

	return nil
}
