package db

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/gocql/gocql"
	"github.com/viant/xunsafe"
)

type viewTableDeleteRow struct {
	idValue   int64
	keyValues []any
}

func appendUniqueViewTableColumn(target []viewTableColumnInfo, column viewTableColumnInfo) []viewTableColumnInfo {
	// ViewTable physical columns are derived from source columns, so source-name uniqueness is enough.
	for _, existingColumn := range target {
		if existingColumn.SourceColumn.GetName() == column.SourceColumn.GetName() {
			return target
		}
	}
	return append(target, column)
}

func getViewTableColumnType(sourceColumn IColInfo, useSliceElement bool) colType {
	columnType := sourceColumn.GetType()
	physicalType := *columnType

	if useSliceElement {
		// Fan-out columns persist one scalar per slice item, not the original collection type.
		elementFieldType := columnType.FieldType
		switch {
		case strings.HasPrefix(elementFieldType, "[]"):
			elementFieldType = strings.TrimPrefix(elementFieldType, "[]")
		case strings.HasPrefix(elementFieldType, "*[]"):
			elementFieldType = strings.TrimPrefix(elementFieldType, "*[]")
		default:
			panic(fmt.Sprintf(`ViewTables column "%v" must be a slice to fan out`, sourceColumn.GetName()))
		}

		physicalType = GetColTypeByName(elementFieldType, "")
		if physicalType.Type == 0 || physicalType.ColType == "" {
			panic(fmt.Sprintf(`ViewTables column "%v" fan-out element type "%v" is not supported`, sourceColumn.GetName(), elementFieldType))
		}
	}

	return physicalType
}

func getViewTableColumnName(column viewTableColumnInfo) string {
	return column.SourceColumn.GetName()
}

func makeViewTableColumn(sourceColumn IColInfo, useSliceElement bool) viewTableColumnInfo {
	// Keep only the source column reference plus the fan-out flag; name/type are derived on demand.
	return viewTableColumnInfo{
		SourceColumn:     sourceColumn,
		UsesSliceElement: useSliceElement,
	}
}

func getViewTableFanoutColumn(view *viewInfo) IColInfo {
	// Slice-backed views fan out on the declared slice key; scalar-only views still iterate once on the first key column.
	for _, keyColumn := range view.tableKeyColumns {
		if view.fanoutColumnName != "" && keyColumn.SourceColumn.GetName() == view.fanoutColumnName {
			return keyColumn.SourceColumn
		}
	}
	if len(view.tableKeyColumns) == 0 {
		return nil
	}
	return view.tableKeyColumns[0].SourceColumn
}

func getViewTableColumnWriteValue(view *viewInfo, column viewTableColumnInfo, recordPointer unsafe.Pointer, fanoutValue any) any {
	fanoutColumn := getViewTableFanoutColumn(view)
	if fanoutColumn != nil && fanoutColumn.GetName() == column.SourceColumn.GetName() {
		// The selected fan-out source column always binds from fanoutValue so scalar and slice paths share one insert flow.
		return normalizeEmptyStringWriteValue(fanoutValue)
	}

	return getSourceColumnWriteValue(column.SourceColumn, recordPointer)
}

func getSourceColumnWriteValue(sourceColumn IColInfo, recordPointer unsafe.Pointer) any {
	// Reuse the ORM's compiled statement/raw accessors so derived tables see the same normalized values as base writes.
	value := sourceColumn.GetStatementValue(recordPointer)
	if value == nil {
		value = sourceColumn.GetRawValue(recordPointer)
	}
	return normalizeEmptyStringWriteValue(value)
}

func getViewTableFanoutValues(view *viewInfo, recordPointer unsafe.Pointer) []any {
	fanoutColumn := getViewTableFanoutColumn(view)
	if fanoutColumn == nil {
		return nil
	}

	if !fanoutColumn.GetType().IsSlice {
		// Scalar-only views still need one derived row, so return the real scalar value as a single-item fan-out set.
		return []any{getSourceColumnWriteValue(fanoutColumn, recordPointer)}
	}

	// A ViewTable can fan out only one slice-backed key, so its collection defines the row expansion set.
	rawValue := fanoutColumn.GetRawValue(recordPointer)
	if rawValue == nil {
		return nil
	}

	valueRef := reflect.ValueOf(rawValue)
	for valueRef.Kind() == reflect.Pointer {
		if valueRef.IsNil() {
			return nil
		}
		valueRef = valueRef.Elem()
	}

	if valueRef.Kind() != reflect.Slice && valueRef.Kind() != reflect.Array {
		panic(fmt.Sprintf(`ViewTables column "%v" expected a slice value during fan-out`, fanoutColumn.GetName()))
	}

	fanoutValues := make([]any, 0, valueRef.Len())
	for index := 0; index < valueRef.Len(); index++ {
		fanoutValues = append(fanoutValues, valueRef.Index(index).Interface())
	}
	return fanoutValues
}

func getViewTableKeyValues(view *viewInfo, recordPointer unsafe.Pointer, fanoutValue any) []any {
	// Build the derived clustering key exactly as it will be inserted so delete filtering can compare full row identity.
	keyValues := make([]any, 0, len(view.tableKeyColumns))
	for _, keyColumn := range view.tableKeyColumns {
		keyValues = append(keyValues, getViewTableColumnWriteValue(view, keyColumn, recordPointer, fanoutValue))
	}
	return keyValues
}

func makeViewTableRowKey(partValue int64, idValue int64, keyValues []any) string {
	// String keys keep the stale-row filter simple and deterministic across mixed primitive clustering columns.
	rowKeyParts := []string{
		fmt.Sprintf("p=%d", partValue),
		fmt.Sprintf("id=%d", idValue),
	}
	for _, keyValue := range keyValues {
		rowKeyParts = append(rowKeyParts, fmt.Sprintf("k=%v", keyValue))
	}
	return strings.Join(rowKeyParts, "|")
}

func getViewTableMaintenanceIndexName(view *viewInfo) string {
	return fmt.Sprintf(`%v__%v_index_0`, view.name, view.maintenanceIDColumn.GetName())
}

func getViewTableMaintenanceIndexCreateScript(view *viewInfo, scyllaTable ScyllaTable[any]) string {
	// Maintenance reads are always scoped by partition + base ID, so the helper derives the one required local index.
	partKey := scyllaTable.GetPartKey()
	return fmt.Sprintf(`CREATE INDEX %v ON %v.%v ((%v), %v)`,
		getViewTableMaintenanceIndexName(view), scyllaTable.keyspace, view.name, partKey.GetName(), view.maintenanceIDColumn.GetName())
}

func makeNumericQueryValue(column IColInfo, value int64) any {
	// Query binds must match the physical CQL numeric type even though ORM helpers normalize partition/ID values to int64.
	switch column.GetType().FieldType {
	case "int8":
		return int8(value)
	case "int16":
		return int16(value)
	case "int32":
		return int32(value)
	case "int":
		return int(value)
	default:
		return value
	}
}

func fetchExistingViewTableDeleteRows(
	view *viewInfo, scyllaTable ScyllaTable[any], partValue int64, idValues []int64,
) ([]viewTableDeleteRow, error) {
	if len(idValues) == 0 {
		return nil, nil
	}

	// Fetch stale candidates in one indexed pass per partition so updates do not issue one SELECT per base record.
	selectColumnNames := make([]string, 0, len(view.tableKeyColumns)+1)
	for _, keyColumn := range view.tableKeyColumns {
		selectColumnNames = append(selectColumnNames, getViewTableColumnName(keyColumn))
	}
	selectColumnNames = append(selectColumnNames, view.maintenanceIDColumn.GetName())

	idPlaceholders := make([]string, 0, len(idValues))
	queryArguments := make([]any, 0, len(idValues)+1)
	queryArguments = append(queryArguments, makeNumericQueryValue(scyllaTable.GetPartKey(), partValue))
	for _, idValue := range idValues {
		idPlaceholders = append(idPlaceholders, "?")
		queryArguments = append(queryArguments, makeNumericQueryValue(view.maintenanceIDColumn, idValue))
	}

	query := fmt.Sprintf(`SELECT %v FROM %v.%v WHERE %v = ? AND %v IN (%v)`,
		strings.Join(selectColumnNames, ", "),
		scyllaTable.keyspace,
		view.name,
		scyllaTable.GetPartKey().GetName(),
		view.maintenanceIDColumn.GetName(),
		strings.Join(idPlaceholders, ", "),
	)

	iter := getScyllaConnection().Query(query, queryArguments...).Iter()
	rowData, err := iter.RowData()
	if err != nil {
		return nil, err
	}

	scanner := iter.Scanner()
	deleteRows := []viewTableDeleteRow{}
	for scanner.Next() {
		rowValues := rowData.Values
		if err := scanner.Scan(rowValues...); err != nil {
			return nil, err
		}

		keyValues := make([]any, 0, len(view.tableKeyColumns))
		for valueIndex := range view.tableKeyColumns {
			keyValues = append(keyValues, normalizeScannedValue(rowValues[valueIndex]))
		}
		deleteRows = append(deleteRows, viewTableDeleteRow{
			idValue:   convertToInt64(normalizeScannedValue(rowValues[len(view.tableKeyColumns)])),
			keyValues: keyValues,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}
	return deleteRows, nil
}

func executeViewTableSyncChunk[T any](
	view *viewInfo, recordsChunk *[]T, session *gocql.Session, scyllaTable *ScyllaTable[any],
) error {
	batch := session.NewBatch(gocql.UnloggedBatch)
	deleteStatementsCount := 0
	insertStatementsCount := 0

	// Group deletes by partition and base ID so the maintenance read can use the local index efficiently.
	deleteGroups := map[int64]map[int64]struct{}{}
	// Track rows that will be reinserted with the same derived primary key to avoid unnecessary tombstones.
	rowsToKeep := map[string]struct{}{}

	for recordIndex := range *recordsChunk {
		recordPointer := xunsafe.AsPointer(&(*recordsChunk)[recordIndex])
		partValue := scyllaTable.GetPartValue(recordPointer)
		keyValues := scyllaTable.GetKeyValues(recordPointer)
		if len(keyValues) == 0 {
			continue
		}
		idValue := convertToInt64(keyValues[0])

		groupRows := deleteGroups[partValue]
		if groupRows == nil {
			groupRows = map[int64]struct{}{}
			deleteGroups[partValue] = groupRows
		}
		groupRows[idValue] = struct{}{}

		// Precompute the future derived row identities so unchanged fan-out rows are preserved during update sync.
		fanoutValues := getViewTableFanoutValues(view, recordPointer)
		for _, fanoutValue := range fanoutValues {
			rowKey := makeViewTableRowKey(partValue, idValue, getViewTableKeyValues(view, recordPointer, fanoutValue))
			rowsToKeep[rowKey] = struct{}{}
		}
	}

	partColumn := scyllaTable.GetPartKey()
	for partValue, groupRows := range deleteGroups {
		idValues := make([]int64, 0, len(groupRows))
		for idValue := range groupRows {
			idValues = append(idValues, idValue)
		}

		// Read current derived rows first, then delete only the ones not covered by the new fan-out payload.
		deleteRows, err := fetchExistingViewTableDeleteRows(view, *scyllaTable, partValue, idValues)
		if err != nil {
			return fmt.Errorf("view table fetch existing rows %s: %w", view.name, err)
		}

		for _, deleteRow := range deleteRows {
			rowKey := makeViewTableRowKey(partValue, deleteRow.idValue, deleteRow.keyValues)
			if _, shouldKeepRow := rowsToKeep[rowKey]; shouldKeepRow {
				continue
			}

			whereColumnNames := []string{partColumn.GetName()}
			whereArguments := []any{makeNumericQueryValue(partColumn, partValue)}
			for keyIndex, keyColumn := range view.tableKeyColumns {
				whereColumnNames = append(whereColumnNames, getViewTableColumnName(keyColumn))
				whereArguments = append(whereArguments, deleteRow.keyValues[keyIndex])
			}
			whereColumnNames = append(whereColumnNames, view.maintenanceIDColumn.GetName())
			whereArguments = append(whereArguments, makeNumericQueryValue(view.maintenanceIDColumn, deleteRow.idValue))

			whereClauses := make([]string, 0, len(whereColumnNames))
			for _, whereColumnName := range whereColumnNames {
				whereClauses = append(whereClauses, fmt.Sprintf("%v = ?", whereColumnName))
			}

			// Deletes target the full derived primary key so only stale fan-out rows are removed.
			deleteQuery := fmt.Sprintf(`DELETE FROM %v.%v WHERE %v`,
				scyllaTable.keyspace,
				view.name,
				strings.Join(whereClauses, " AND "),
			)
			batch.Query(deleteQuery, whereArguments...)
			deleteStatementsCount++
		}
	}

	columnNames := make([]string, 0, len(view.tableColumns))
	columnPlaceholders := make([]string, 0, len(view.tableColumns))
	for _, column := range view.tableColumns {
		columnNames = append(columnNames, getViewTableColumnName(column))
		columnPlaceholders = append(columnPlaceholders, "?")
	}

	// Reuse one insert statement shape per chunk; only the bound values vary across fan-out rows.
	insertQuery := fmt.Sprintf(`INSERT INTO %v.%v (%v) VALUES (%v)`,
		scyllaTable.keyspace,
		view.name,
		strings.Join(columnNames, ", "),
		strings.Join(columnPlaceholders, ", "),
	)

	for recordIndex := range *recordsChunk {
		recordPointer := xunsafe.AsPointer(&(*recordsChunk)[recordIndex])
		fanoutValues := getViewTableFanoutValues(view, recordPointer)
		if len(fanoutValues) == 0 {
			if DebugFull {
				fmt.Printf("ViewTable sync skipped empty fanout: table=%s view=%s\n", scyllaTable.name, view.name)
			}
			continue
		}

		// Insert every derived row from the current source payload after stale rows have been cleared.
		for _, fanoutValue := range fanoutValues {
			values := make([]any, 0, len(view.tableColumns))
			for _, column := range view.tableColumns {
				values = append(values, getViewTableColumnWriteValue(view, column, recordPointer, fanoutValue))
			}
			batch.Query(insertQuery, values...)
			insertStatementsCount++
		}
	}

	if deleteStatementsCount == 0 && insertStatementsCount == 0 {
		return nil
	}

	if DebugFull {
		fmt.Printf("ViewTable sync batch: base_table=%s view=%s deletes=%d inserts=%d records=%d\n",
			scyllaTable.name, view.name, deleteStatementsCount, insertStatementsCount, len(*recordsChunk))
	}

	// Stale deletes and current inserts no longer target the same derived key, so one batch is enough.
	if err := session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("view table sync %s: %w", view.name, err)
	}

	return nil
}
