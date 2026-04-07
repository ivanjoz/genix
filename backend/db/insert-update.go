package db

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
	"unsafe"

	"github.com/gocql/gocql"
	"github.com/viant/xunsafe"
	"golang.org/x/sync/errgroup"
)

const maxInsertBatchRows = 200

var getWriteCounterValue = GetCounter
var getManagedUnixTime = currentManagedUnixTime

type managedWriteValues struct {
	createdValues       []any
	updatedValues       []any
	updateCounterValues []any
}

type prefetchedManagedCounterValues struct {
	counterValueByPartition map[int64]any
	counterNameByPartition  map[int64]string
}

func currentManagedUnixTime() int32 {
	// Keep DB-managed audit timestamps aligned with the project SUnixTime convention without importing core.
	return int32((time.Now().Unix() - 1e9) / 2)
}

func (e managedWriteValues) slice(start int, end int) managedWriteValues {
	slicedValues := managedWriteValues{}
	if len(e.createdValues) > 0 {
		slicedValues.createdValues = e.createdValues[start:end]
	}
	if len(e.updatedValues) > 0 {
		slicedValues.updatedValues = e.updatedValues[start:end]
	}
	if len(e.updateCounterValues) > 0 {
		slicedValues.updateCounterValues = e.updateCounterValues[start:end]
	}
	return slicedValues
}

func (e managedWriteValues) getValueForColumn(recordIndex int, column IColInfo, isInsert bool) (any, bool) {
	switch column.GetName() {
	case managedCreatedColumnName:
		if isInsert && recordIndex < len(e.createdValues) && e.createdValues[recordIndex] != nil {
			return e.createdValues[recordIndex], true
		}
	case managedUpdatedColumnName:
		if recordIndex < len(e.updatedValues) && e.updatedValues[recordIndex] != nil {
			return e.updatedValues[recordIndex], true
		}
	case managedUpdateCounterColumnName:
		if recordIndex < len(e.updateCounterValues) && e.updateCounterValues[recordIndex] != nil {
			return e.updateCounterValues[recordIndex], true
		}
	}
	return nil, false
}

func coerceManagedIntegerValue(column IColInfo, value int64) any {
	switch column.GetType().FieldType {
	case "int8":
		return int8(value)
	case "int16":
		return int16(value)
	case "int32":
		return int32(value)
	case "int64":
		return value
	case "int":
		return int(value)
	default:
		return int32(value)
	}
}

func resolveManagedTimestampValue(column IColInfo, ptr unsafe.Pointer, fallbackValue int64) any {
	if column == nil {
		return nil
	}

	currentValue := column.GetRawValue(ptr)
	if currentValue != nil {
		currentValueInt := convertToInt64(currentValue)
		if currentValueInt > 0 {
			return coerceManagedIntegerValue(column, currentValueInt)
		}
	}

	return coerceManagedIntegerValue(column, fallbackValue)
}

type selfParser interface {
	SelfParse()
}

func runSelfParseIfDefined[T TableBaseInterface[E, T], E TableSchemaInterface[E]](records *[]T) {
	if len(*records) == 0 {
		return
	}

	firstRecordPointer := any(&(*records)[0])
	if _, hasSelfParse := firstRecordPointer.(selfParser); hasSelfParse {
		for i := range *records {
			any(&(*records)[i]).(selfParser).SelfParse()
		}
		return
	}
}

func fetchManagedCounterValues[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any],
) (prefetchedManagedCounterValues, error) {
	prefetchedValues := prefetchedManagedCounterValues{
		counterValueByPartition: map[int64]any{},
		counterNameByPartition:  map[int64]string{},
	}
	if len(*records) == 0 || scyllaTable.updateCounterCol == nil {
		return prefetchedValues, nil
	}

	partitionColumn := scyllaTable.GetPartKey()
	partitionValuesToFetch := map[int64]struct{}{}
	for recordIndex := range *records {
		recordPointer := xunsafe.AsPointer(&(*records)[recordIndex])
		partitionValue := int64(0)
		if partitionColumn != nil && !partitionColumn.IsNil() {
			partitionValue = convertToInt64(partitionColumn.GetRawValue(recordPointer))
		}
		partitionValuesToFetch[partitionValue] = struct{}{}
	}

	for partitionValue := range partitionValuesToFetch {
		counterName := fmt.Sprintf("x%v_%v_updated", partitionValue, scyllaTable.name)
		nextCounterValue, err := getWriteCounterValue(scyllaTable.keyspace, counterName, 1)
		if err != nil {
			return prefetchedManagedCounterValues{}, fmt.Errorf("write update counter %s: %w", counterName, err)
		}
		prefetchedValues.counterNameByPartition[partitionValue] = counterName
		prefetchedValues.counterValueByPartition[partitionValue] = coerceManagedIntegerValue(scyllaTable.updateCounterCol, nextCounterValue)
	}

	return prefetchedValues, nil
}

func applyPrefetchedManagedCounterValues[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any], managedValues *managedWriteValues, prefetchedValues prefetchedManagedCounterValues,
) {
	if scyllaTable.updateCounterCol == nil {
		return
	}

	partitionColumn := scyllaTable.GetPartKey()
	for recordIndex := range *records {
		recordPointer := xunsafe.AsPointer(&(*records)[recordIndex])
		partitionValue := int64(0)
		if partitionColumn != nil && !partitionColumn.IsNil() {
			partitionValue = convertToInt64(partitionColumn.GetRawValue(recordPointer))
		}

		counterValue := prefetchedValues.counterValueByPartition[partitionValue]
		managedValues.updateCounterValues[recordIndex] = counterValue
		scyllaTable.updateCounterCol.SetValue(recordPointer, counterValue)
	}

	if DebugFull {
		for partitionValue, counterName := range prefetchedValues.counterNameByPartition {
			fmt.Printf("Write update counter assigned: table=%s partition=%d column=%s counter=%s records=%d\n",
				scyllaTable.name, partitionValue, scyllaTable.updateCounterCol.GetName(), counterName, len(*records))
		}
	}
}

func applyWriteManagedColumnsWithPrefetchedCounters[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any], isInsert bool, prefetchedValues *prefetchedManagedCounterValues,
) (managedWriteValues, error) {
	managedValues := managedWriteValues{}
	if len(*records) == 0 {
		return managedValues, nil
	}

	managedValues.createdValues = make([]any, len(*records))
	managedValues.updatedValues = make([]any, len(*records))
	managedValues.updateCounterValues = make([]any, len(*records))
	currentWriteTime := int64(getManagedUnixTime())

	for recordIndex := range *records {
		recordPointer := xunsafe.AsPointer(&(*records)[recordIndex])

		if isInsert && scyllaTable.createdCol != nil {
			createdValue := resolveManagedTimestampValue(scyllaTable.createdCol, recordPointer, currentWriteTime)
			managedValues.createdValues[recordIndex] = createdValue
			scyllaTable.createdCol.SetValue(recordPointer, createdValue)
		}

		if scyllaTable.updatedCol != nil {
			updatedValue := resolveManagedTimestampValue(scyllaTable.updatedCol, recordPointer, currentWriteTime)
			managedValues.updatedValues[recordIndex] = updatedValue
			scyllaTable.updatedCol.SetValue(recordPointer, updatedValue)
		}
	}

	if scyllaTable.updateCounterCol != nil {
		valuesToApply := prefetchedManagedCounterValues{}
		if prefetchedValues == nil {
			fetchedValues, err := fetchManagedCounterValues(records, scyllaTable)
			if err != nil {
				return managedWriteValues{}, err
			}
			valuesToApply = fetchedValues
		} else {
			valuesToApply = *prefetchedValues
		}
		applyPrefetchedManagedCounterValues(records, scyllaTable, &managedValues, valuesToApply)
	}

	return managedValues, nil
}

func applyWriteManagedColumns[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any], isInsert bool,
) (managedWriteValues, error) {
	return applyWriteManagedColumnsWithPrefetchedCounters(records, scyllaTable, isInsert, nil)
}

func fetchAutoincrementCounterStarts[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any],
) (map[string]int64, error) {
	counterStartByGroup := map[string]int64{}
	if scyllaTable.autoincrementCol == nil {
		return counterStartByGroup, nil
	}

	partitionColumn := scyllaTable.GetPartKey()
	groups := map[string][]*T{}

	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)

		partitionValue := int32(0)
		if partitionColumn != nil {
			partitionValue = int32(convertToInt64(partitionColumn.GetRawValue(ptr)))
		}

		autoPartVal := int64(0)
		if scyllaTable.autoincrementPart != nil {
			autoPartVal = convertToInt64(scyllaTable.autoincrementPart.GetRawValue(ptr))
		}

		key := fmt.Sprintf("%d|%v", partitionValue, autoPartVal)
		groups[key] = append(groups[key], rec)
	}

	for groupKey, group := range groups {
		partValues := strings.Split(groupKey, "|")
		recordsNeedingAutoincrement := 0
		for _, rec := range group {
			ptr := xunsafe.AsPointer(rec)
			rawAutoincrementValue := scyllaTable.autoincrementCol.GetRawValue(ptr)
			if convertToInt64(rawAutoincrementValue) <= 0 {
				recordsNeedingAutoincrement++
			}
		}
		if recordsNeedingAutoincrement == 0 {
			continue
		}

		counterName := fmt.Sprintf("x%v_%v_%v", partValues[0], scyllaTable.name, partValues[1])
		keyspace := strings.Split(scyllaTable.GetFullName(), ".")[0]
		counterValue, err := GetCounter(keyspace, counterName, recordsNeedingAutoincrement)
		if err != nil {
			return nil, err
		}
		counterStartByGroup[groupKey] = counterValue - int64(recordsNeedingAutoincrement) + 1
	}

	return counterStartByGroup, nil
}

func handlePreInsert[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any], counterStartByGroup map[string]int64,
) error {
	partitionColumn := scyllaTable.GetPartKey()
	// Group records by composite key: partition value + autoincrementPart value
	groups := map[string][]*T{}

	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)

		// Get partition key value
		partitionValue := int32(0)
		if partitionColumn != nil {
			partitionValue = int32(convertToInt64(partitionColumn.GetRawValue(ptr)))
		}

		// Get autoincrement part value (0 if not defined)
		autoPartVal := int64(0)
		if scyllaTable.autoincrementPart != nil {
			autoPartVal = convertToInt64(scyllaTable.autoincrementPart.GetRawValue(ptr))
		}

		// Group key is concatenation of partition value + autoincrementPart value
		key := fmt.Sprintf("%d|%v", partitionValue, autoPartVal)
		groups[key] = append(groups[key], rec)
	}

	for partKey, group := range groups {
		// Filter records that need autoincrement.
		// Rule: autoincrement applies when the configured autoincrement column is <= 0.
		recordsNeedingAutoincrement := []*T{}
		recordNeedsAutoincrement := map[*T]bool{}

		for _, rec := range group {
			ptr := xunsafe.AsPointer(rec)
			autoincrementColumnValue := int64(0)
			if scyllaTable.autoincrementCol != nil {
				rawAutoincrementValue := scyllaTable.autoincrementCol.GetRawValue(ptr)
				autoincrementColumnValue = convertToInt64(rawAutoincrementValue)
			}

			if autoincrementColumnValue <= 0 {
				recordsNeedingAutoincrement = append(recordsNeedingAutoincrement, rec)
				recordNeedsAutoincrement[rec] = true
			}
		}

		counterVal := counterStartByGroup[partKey]

		for _, rec := range group {
			ptr := xunsafe.AsPointer(rec)
			var currentAutoVal int64

			// Only apply autoincrement when this record was marked as needing it (<= 0 rule).
			if scyllaTable.autoincrementCol != nil && recordNeedsAutoincrement[rec] {
				currentAutoVal = counterVal
				counterVal++

				colInfo := scyllaTable.autoincrementCol.(*columnInfo)
				if colInfo.autoincrementRandSize > 0 {
					suffix := GetRandomInt64(colInfo.autoincrementRandSize)
					currentAutoVal = currentAutoVal*Pow10Int64(int64(colInfo.autoincrementRandSize)) + suffix
				}

				// If not packing, set directly
				if len(scyllaTable.keyIntPacking) == 0 {
					scyllaTable.autoincrementCol.SetValue(ptr, currentAutoVal)
				}
			}

			if len(scyllaTable.keyIntPacking) > 0 {
				var packedValue int64
				remainingDigits := int64(19)
				for i, col := range scyllaTable.keyIntPacking {
					if col == nil {
						continue
					}
					var val int64
					if col == scyllaTable.autoincrementCol {
						val = currentAutoVal
					} else {
						val = convertToInt64(col.GetRawValue(ptr))
					}

					colPackingInfo := col.(*columnInfo)
					decSize := int64(colPackingInfo.decimalSize)
					// If it's the last one and size is 0, it takes all remaining space
					if i == len(scyllaTable.keyIntPacking)-1 && decSize == 0 {
						decSize = remainingDigits
					}

					remainingDigits -= decSize
					if remainingDigits < 0 {
						remainingDigits = 0
					}

					shift := Pow10Int64(remainingDigits)
					packedValue += val * shift
				}
				// Set into the only key
				scyllaTable.keys[0].SetValue(ptr, packedValue)
			}
		}
	}

	return nil
}

func MakeInsertStatement[T TableBaseInterface[E, T], E TableSchemaInterface[E]](records *[]T, columnsToExclude ...Coln) []string {
	refTable := initStructTable[E, T](new(E))
	scyllaTable := getOrCompileScyllaTable(refTable)
	managedValues, err := applyWriteManagedColumns(records, scyllaTable, true)
	if err != nil {
		panic(err)
	}

	columns := []IColInfo{}
	if len(columnsToExclude) > 0 {
		columsToExcludeNames := []string{}
		for _, e := range columnsToExclude {
			columsToExcludeNames = append(columsToExcludeNames, e.GetInfo().Name)
		}
		for _, col := range scyllaTable.columns {
			mustIncludeManagedColumn := col.GetName() == managedCreatedColumnName ||
				col.GetName() == managedUpdatedColumnName ||
				col.GetName() == managedUpdateCounterColumnName
			if mustIncludeManagedColumn || !slices.Contains(columsToExcludeNames, col.GetName()) {
				columns = append(columns, col)
			}
		}
	} else {
		columns = scyllaTable.columns
	}

	columnsNames := []string{}
	for _, col := range columns {
		columnsNames = append(columnsNames, col.GetName())
	}

	queryStrInsert := fmt.Sprintf(`INSERT INTO %v (%v) VALUES `,
		scyllaTable.GetFullName(), strings.Join(columnsNames, ", "))

	queryStatements := []string{}

	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)

		recordInsertValues := []string{}

		for _, col := range columns {
			value := any(nil)
			if managedValue, found := managedValues.getValueForColumn(i, col, true); found {
				value = normalizeEmptyStringWriteLiteral(managedValue)
			} else {
				value = getNormalizedWriteLiteral(col, ptr)
			}
			recordInsertValues = append(recordInsertValues, fmt.Sprintf("%v", value))
		}

		statement := /*" " +*/ queryStrInsert + "(" + strings.Join(recordInsertValues, ", ") + ")"
		queryStatements = append(queryStatements, statement)
	}
	return queryStatements
}

func Table[T TableBaseInterface[E, T], E TableSchemaInterface[E]]() *E {
	return initStructTable[E, T](new(E))
}

func normalizeEmptyStringWriteValue(value any) any {
	// Rationale: Scylla should persist empty strings as NULL on write operations to keep storage semantics consistent.
	switch typedValue := value.(type) {
	case string:
		if typedValue == "" {
			return nil
		}
	case *string:
		if typedValue == nil || *typedValue == "" {
			return nil
		}
	}

	return value
}

func normalizeEmptyStringWriteLiteral(value any) any {
	// Rationale: UPDATE/statement helpers use raw CQL literals, so NULL must be emitted as the keyword instead of a Go nil.
	if normalizedValue := normalizeEmptyStringWriteValue(value); normalizedValue == nil {
		return "null"
	}

	return value
}

func getNormalizedWriteLiteral(column IColInfo, ptr unsafe.Pointer) any {
	// Rationale: virtual columns may not have a backing field, so updates must normalize the computed statement value
	// instead of treating a missing raw field as NULL before the virtual accessor runs.
	writeValue := column.GetStatementValue(ptr)
	if writeValue == nil {
		writeValue = column.GetRawValue(ptr)
	}
	if normalizedValue := normalizeEmptyStringWriteLiteral(writeValue); normalizedValue == "null" {
		return normalizedValue
	}

	return column.GetValue(ptr)
}

func makeInsertBatch[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any], managedValues managedWriteValues, columnsToExclude ...Coln,
) *gocql.Batch {

	columns := []IColInfo{}
	if len(columnsToExclude) > 0 {
		columsToExcludeNames := []string{}
		for _, e := range columnsToExclude {
			columsToExcludeNames = append(columsToExcludeNames, e.GetInfo().Name)
		}
		for _, col := range scyllaTable.columns {
			mustIncludeManagedColumn := col.GetName() == managedCreatedColumnName ||
				col.GetName() == managedUpdatedColumnName ||
				col.GetName() == managedUpdateCounterColumnName
			if mustIncludeManagedColumn || !slices.Contains(columsToExcludeNames, col.GetName()) {
				columns = append(columns, col)
			}
		}
	} else {
		columns = scyllaTable.columns
	}

	columnsNames := []string{}
	columnPlaceholders := []string{}
	for _, col := range columns {
		columnsNames = append(columnsNames, col.GetName())
		columnPlaceholders = append(columnPlaceholders, "?")
	}

	session := getScyllaConnection()
	batch := session.NewBatch(gocql.UnloggedBatch)

	queryStrInsert := fmt.Sprintf(`INSERT INTO %v (%v) VALUES (%v)`,
		scyllaTable.GetFullName(), strings.Join(columnsNames, ", "), strings.Join(columnPlaceholders, ", "))

	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)
		values := []any{}

		for _, col := range columns {
			var value any
			if managedValue, found := managedValues.getValueForColumn(i, col, true); found {
				value = managedValue
			} else {
				value = col.GetStatementValue(ptr)
			}
			if value == nil {
				value = col.GetValue(ptr)
			}
			value = normalizeEmptyStringWriteValue(value)
			values = append(values, value)
		}

		// fmt.Println("VALUES::")
		// fmt.Println(values)
		batch.Query(queryStrInsert, values...)
	}
	return batch
}

func collectAllWritableColumns(scyllaTable *ScyllaTable[any]) []IColInfo {
	affectedColumns := []IColInfo{}
	for _, column := range scyllaTable.columns {
		if column.GetInfo().IsVirtual {
			continue
		}
		affectedColumns = append(affectedColumns, column)
	}
	return affectedColumns
}

func collectAffectedColumnsForInclude(scyllaTable *ScyllaTable[any], columnsToInclude []Coln) []IColInfo {
	affectedColumns := []IColInfo{}
	for _, columnToInclude := range columnsToInclude {
		column := scyllaTable.columnsMap[columnToInclude.GetName()]
		if column == nil || column.IsNil() || column.GetInfo().IsVirtual {
			continue
		}
		affectedColumns = append(affectedColumns, column)
	}
	if scyllaTable.updatedCol != nil {
		updatedAlreadyIncluded := false
		for _, affectedColumn := range affectedColumns {
			if affectedColumn.GetName() == scyllaTable.updatedCol.GetName() {
				updatedAlreadyIncluded = true
				break
			}
		}
		if !updatedAlreadyIncluded {
			affectedColumns = append(affectedColumns, scyllaTable.updatedCol)
		}
	}
	return affectedColumns
}

func collectAffectedColumnsForExclude(scyllaTable *ScyllaTable[any], columnsToExclude []Coln) []IColInfo {
	excludedColumnNames := map[string]bool{}
	for _, columnToExclude := range columnsToExclude {
		excludedColumnNames[columnToExclude.GetName()] = true
	}

	affectedColumns := []IColInfo{}
	for _, column := range scyllaTable.columns {
		mustIncludeUpdated := scyllaTable.updatedCol != nil && column.GetName() == scyllaTable.updatedCol.GetName()
		if column.GetInfo().IsVirtual || (!mustIncludeUpdated && excludedColumnNames[column.GetName()]) {
			continue
		}
		affectedColumns = append(affectedColumns, column)
	}
	return affectedColumns
}

func hasUsableIndexSourceValue(rawValue any) bool {
	if rawValue == nil {
		return false
	}

	valueRef := reflect.ValueOf(rawValue)
	for valueRef.Kind() == reflect.Pointer {
		if valueRef.IsNil() {
			return false
		}
		valueRef = valueRef.Elem()
	}

	switch valueRef.Kind() {
	case reflect.String:
		return valueRef.Len() > 0
	case reflect.Slice, reflect.Array:
		return valueRef.Len() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return valueRef.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return valueRef.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return valueRef.Float() != 0
	case reflect.Bool:
		return valueRef.Bool()
	default:
		return true
	}
}

func syncTableBackedViews[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable *ScyllaTable[any], affectedColumns []IColInfo,
) error {
	if len(*records) == 0 || len(scyllaTable.views) == 0 {
		return nil
	}

	session := getScyllaConnection()
	partColumn := scyllaTable.GetPartKey()
	if partColumn == nil || partColumn.IsNil() {
		return nil
	}

	for _, view := range scyllaTable.views {
		if view.Type != 9 {
			continue
		}
		if len(affectedColumns) > 0 {
			shouldSyncCurrentView := false
			for _, affectedColumn := range affectedColumns {
				if view.rebuildColumnNames[affectedColumn.GetName()] {
					shouldSyncCurrentView = true
					break
				}
			}
			if !shouldSyncCurrentView {
				continue
			}
		}

		for start := 0; start < len(*records); start += maxInsertBatchRows {
			end := start + maxInsertBatchRows
			if end > len(*records) {
				end = len(*records)
			}

			recordsChunk := (*records)[start:end]
			if err := executeViewTableSyncChunk(view, &recordsChunk, session, scyllaTable); err != nil {
				return err
			}
		}
	}

	return nil
}

func Insert[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToExclude ...Coln,
) error {
	refTable := initStructTable[E, T](new(E))
	scyllaTable := getOrCompileScyllaTable(refTable)

	runSelfParseIfDefined(records)

	prefetchedManagedCounters := prefetchedManagedCounterValues{}
	autoincrementCounterStarts := map[string]int64{}
	var fetchGroup errgroup.Group
	if scyllaTable.updateCounterCol != nil {
		fetchGroup.Go(func() error {
			values, err := fetchManagedCounterValues(records, scyllaTable)
			if err != nil {
				return err
			}
			prefetchedManagedCounters = values
			return nil
		})
	}
	if scyllaTable.autoincrementCol != nil {
		fetchGroup.Go(func() error {
			values, err := fetchAutoincrementCounterStarts(records, scyllaTable)
			if err != nil {
				return err
			}
			autoincrementCounterStarts = values
			return nil
		})
	}
	if err := fetchGroup.Wait(); err != nil {
		return err
	}

	managedValues, err := applyWriteManagedColumnsWithPrefetchedCounters(records, scyllaTable, true, &prefetchedManagedCounters)
	if err != nil {
		return err
	}

	if err := handlePreInsert(records, scyllaTable, autoincrementCounterStarts); err != nil {
		return err
	}

	session := getScyllaConnection()
	for start := 0; start < len(*records); start += maxInsertBatchRows {
		end := start + maxInsertBatchRows
		if end > len(*records) {
			end = len(*records)
		}

		recordsChunk := (*records)[start:end]
		// Keep unlogged batches small to reduce coordinator pressure and avoid write timeouts.
		queryBatch := makeInsertBatch(&recordsChunk, scyllaTable, managedValues.slice(start, end), columnsToExclude...)
		// fmt.Println(queryBatch)

		if err := session.ExecuteBatch(queryBatch); err != nil {
			fmt.Println("Error inserting records:", err)
			return err
		}
	}

	affectedColumns := collectAllWritableColumns(&scyllaTable)
	if err := syncIndexGroupsAfterWrite(records, &scyllaTable, managedValues); err != nil {
		fmt.Println("Error syncing index groups after insert:", err)
		return err
	}
	if err := syncTableBackedViews(records, &scyllaTable, affectedColumns); err != nil {
		fmt.Println("Error syncing view tables after insert:", err)
		return err
	}

	// Cache-version is updated only after a successful write to keep counters consistent with persisted rows.
	if err := updateCacheVersionsAfterWrite(records, scyllaTable); err != nil {
		fmt.Println("Error updating cache versions after insert:", err)
		return err
	}

	return nil
}

func InsertOne[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	record T, columnsToExclude ...Coln,
) error {
	return Insert(&[]T{record}, columnsToExclude...)
}

func makeUpdateStatementsBase[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, managedValues managedWriteValues, columnsToInclude []Coln, columnsToExclude []Coln, onlyVirtual bool,
) []string {

	refTable := initStructTable[E, T](new(E))
	scyllaTable := getOrCompileScyllaTable(refTable)
	columnsToUpdate := []IColInfo{}

	if len(columnsToInclude) > 0 {
		updatedAlreadyIncluded := scyllaTable.updatedCol == nil
		updateCounterAlreadyIncluded := scyllaTable.updateCounterCol == nil
		for _, col_ := range columnsToInclude {
			col := scyllaTable.columnsMap[col_.GetName()]
			if col == nil {
				Print(col)
				panic("No se encontró la columna (update):" + col_.GetName())
			}
			if slices.Contains(scyllaTable.keysIdx, col.GetInfo().Idx) {
				msg := fmt.Sprintf(`Table "%v": The column "%v" can't be updated because is part of primary key.`, scyllaTable.name, col.GetName())
				panic(msg)
			}
			columnsToUpdate = append(columnsToUpdate, col)
			if scyllaTable.updatedCol != nil && col.GetName() == scyllaTable.updatedCol.GetName() {
				updatedAlreadyIncluded = true
			}
			if scyllaTable.updateCounterCol != nil && col.GetName() == scyllaTable.updateCounterCol.GetName() {
				updateCounterAlreadyIncluded = true
			}
		}
		if !updatedAlreadyIncluded {
			columnsToUpdate = append(columnsToUpdate, scyllaTable.updatedCol)
		}
		if !updateCounterAlreadyIncluded {
			// The managed update counter must always persist, even when callers provide an explicit include list.
			columnsToUpdate = append(columnsToUpdate, scyllaTable.updateCounterCol)
		}
	} else {
		columnsToExcludeNames := []string{}
		for _, c := range columnsToExclude {
			columnsToExcludeNames = append(columnsToExcludeNames, c.GetName())
		}
		for _, col := range scyllaTable.columns {
			isExcluded := slices.Contains(columnsToExcludeNames, col.GetName())
			mustIncludeManagedColumn := (scyllaTable.updatedCol != nil && col.GetName() == scyllaTable.updatedCol.GetName()) ||
				(scyllaTable.updateCounterCol != nil && col.GetName() == scyllaTable.updateCounterCol.GetName())
			if !col.GetInfo().IsVirtual && (mustIncludeManagedColumn || !isExcluded) && !slices.Contains(scyllaTable.keysIdx, col.GetInfo().Idx) {
				columnsToUpdate = append(columnsToUpdate, col)
			}
		}
	}

	columnsIdx := []int16{}
	for _, col := range columnsToUpdate {
		columnsIdx = append(columnsIdx, col.GetInfo().Idx)
	}
	columnsIncluded := slices.Concat(scyllaTable.keysIdx, columnsIdx)
	pk := scyllaTable.GetPartKey()
	if pk != nil && !pk.IsNil() {
		columnsIncluded = append(columnsIncluded, pk.GetInfo().Idx)
	}

	//Revisa si hay columnas que deben actualizarse juntas para los índices calculados
	for _, indexViews := range scyllaTable.indexViews {
		if indexViews.column.GetInfo().IsVirtual {
			if indexViews.Type == 3 {
				// Index groups own their own validation and virtual-column recomputation below.
				continue
			}
			includedCols := []int16{}
			notIncludedCols := []int16{}

			for _, colIdx := range indexViews.columnsIdx {
				if slices.Contains(columnsIncluded, colIdx) {
					includedCols = append(includedCols, colIdx)
				} else {
					notIncludedCols = append(notIncludedCols, colIdx)
				}
			}

			if len(includedCols) > 0 && len(notIncludedCols) > 0 {
				colnames := []string{}
				for _, colname := range indexViews.columns {
					if pk != nil && !pk.IsNil() && pk.GetName() == colname {
						continue
					}
					colnames = append(colnames, fmt.Sprintf(`"%v"`, colname))
				}

				includedColsNames := []string{}
				for _, idx := range notIncludedCols {
					includedColsNames = append(includedColsNames, scyllaTable.columnsIdxMap[idx].GetName())
				}

				msg := fmt.Sprintf(`Table "%v": A composit index/view requires the columns %v be updated together. Not Included: %v`, scyllaTable.name, strings.Join(colnames, ", "), strings.Join(includedColsNames, ", "))
				panic(msg)
			} else if len(includedCols) > 0 {
				columnsToUpdate = append(columnsToUpdate, indexViews.column)
			}
		}
	}

	columnsToUpdateByName := map[string]IColInfo{}
	// Track already-selected columns to avoid duplicate SET clauses when adding virtual bucket columns.
	for _, col := range columnsToUpdate {
		columnsToUpdateByName[col.GetName()] = col
	}

	for _, compositeBucketIndex := range scyllaTable.compositeBucketIndexes {
		// Composite bucket hashes become inconsistent if only part of the source tuple is updated.
		includedCols := []string{}
		notIncludedCols := []string{}

		for _, sourceColumn := range compositeBucketIndex.sourceColumns {
			if slices.Contains(columnsIncluded, sourceColumn.GetInfo().Idx) {
				includedCols = append(includedCols, sourceColumn.GetName())
			} else {
				notIncludedCols = append(notIncludedCols, sourceColumn.GetName())
			}
		}

		if len(includedCols) > 0 && len(notIncludedCols) > 0 {
			panic(fmt.Sprintf(`Table "%v": CompositeBucketing index "%v" requires updating all source columns together. Included: %v | Missing: %v`,
				scyllaTable.name,
				compositeBucketIndex.name,
				strings.Join(includedCols, ", "),
				strings.Join(notIncludedCols, ", "),
			))
		}

		if len(includedCols) > 0 {
			// Recompute all generated bucket columns whenever any source tuple is updated.
			for _, virtualColumn := range compositeBucketIndex.virtualColumnsBySize {
				if _, exists := columnsToUpdateByName[virtualColumn.GetName()]; !exists {
					columnsToUpdate = append(columnsToUpdate, virtualColumn)
					columnsToUpdateByName[virtualColumn.GetName()] = virtualColumn
				}
			}
		}
	}

	for _, indexGroup := range scyllaTable.indexGroups {
		includedSourceColumns := []string{}
		missingSourceColumns := []string{}

		for _, sourceColumn := range indexGroup.sourceColumns {
			if slices.Contains(columnsIncluded, sourceColumn.column.GetInfo().Idx) {
				includedSourceColumns = append(includedSourceColumns, sourceColumn.column.GetName())
			} else {
				missingSourceColumns = append(missingSourceColumns, sourceColumn.column.GetName())
			}
		}

		if len(includedSourceColumns) > 0 && len(missingSourceColumns) > 0 {
			for recordIndex := range *records {
				recordPointer := xunsafe.AsPointer(&(*records)[recordIndex])
				missingValuesInStruct := []string{}

				for _, sourceColumn := range indexGroup.sourceColumns {
					if slices.Contains(columnsIncluded, sourceColumn.column.GetInfo().Idx) {
						continue
					}
					if !hasUsableIndexSourceValue(sourceColumn.column.GetRawValue(recordPointer)) {
						missingValuesInStruct = append(missingValuesInStruct, sourceColumn.column.GetName())
					}
				}

				if len(missingValuesInStruct) > 0 {
					panic(fmt.Sprintf(`Table "%v": IndexGroup "%v" needs struct values for omitted source columns. Included in update: %v | Missing in struct: %v`,
						scyllaTable.name,
						indexGroup.name,
						strings.Join(includedSourceColumns, ", "),
						strings.Join(missingValuesInStruct, ", "),
					))
				}
			}
		}

		if len(includedSourceColumns) > 0 && indexGroup.virtualColumn != nil && !indexGroup.virtualColumn.IsNil() {
			if _, exists := columnsToUpdateByName[indexGroup.virtualColumn.GetName()]; !exists {
				columnsToUpdate = append(columnsToUpdate, indexGroup.virtualColumn)
				columnsToUpdateByName[indexGroup.virtualColumn.GetName()] = indexGroup.virtualColumn
			}
		}
	}

	if onlyVirtual {
		cols := columnsToUpdate
		columnsToUpdate = nil
		for _, col := range cols {
			if col.GetInfo().IsVirtual {
				columnsToUpdate = append(columnsToUpdate, col)
			}
		}
	}

	columnsWhere := scyllaTable.keys

	pk = scyllaTable.GetPartKey()
	if pk != nil && !pk.IsNil() {
		columnsWhere = append([]IColInfo{pk}, columnsWhere...)
	}

	queryStatements := []string{}

	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)

		setStatements := []string{}
		for _, col := range columnsToUpdate {
			v := any(nil)
			if managedValue, found := managedValues.getValueForColumn(i, col, false); found {
				v = normalizeEmptyStringWriteLiteral(managedValue)
			} else {
				v = getNormalizedWriteLiteral(col, ptr)
			}
			setStatements = append(setStatements, fmt.Sprintf(`%v = %v`, col.GetName(), v))
		}

		whereStatements := []string{}
		for _, col := range columnsWhere {
			v := col.GetValue(ptr)
			whereStatements = append(whereStatements, fmt.Sprintf(`%v = %v`, col.GetName(), v))
		}

		queryStatement := fmt.Sprintf(
			"UPDATE %v SET %v WHERE %v",
			scyllaTable.GetFullName(), Concatx(", ", setStatements), Concatx(" and ", whereStatements),
		)

		queryStatements = append(queryStatements, queryStatement)
	}

	return queryStatements
}

func MakeUpdateStatements[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToInclude ...Coln,
) []string {
	scyllaTable := MakeScyllaTable[T, E]()
	managedValues, err := applyWriteManagedColumns(records, scyllaTable, false)
	if err != nil {
		panic(err)
	}
	return makeUpdateStatementsBase(records, managedValues, columnsToInclude, nil, false)
}

func Update[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToInclude ...Coln,
) error {

	if len(columnsToInclude) == 0 {
		panic("No se incluyeron columnas a actualizar.")
	}

	refTable := initStructTable[E, T](new(E))
	scyllaTable := getOrCompileScyllaTable(refTable)

	runSelfParseIfDefined(records)

	managedValues, err := applyWriteManagedColumns(records, scyllaTable, false)
	if err != nil {
		return err
	}

	queryStatements := makeUpdateStatementsBase(records, managedValues, columnsToInclude, nil, false)
	queryUpdate := makeQueryStatement(queryStatements)

	fmt.Println(queryUpdate)

	if err := QueryExec(queryUpdate); err != nil {
		fmt.Println("Error updating records:", err)
		return err
	}

	affectedColumns := collectAffectedColumnsForInclude(&scyllaTable, columnsToInclude)
	if err := syncIndexGroupsAfterWrite(records, &scyllaTable, managedValues); err != nil {
		fmt.Println("Error syncing index groups after update:", err)
		return err
	}
	if err := syncTableBackedViews(records, &scyllaTable, affectedColumns); err != nil {
		fmt.Println("Error syncing view tables after update:", err)
		return err
	}

	// Version groups are incremented after update commit using the same record IDs as the update payload.
	if err := updateCacheVersionsAfterWrite(records, scyllaTable); err != nil {
		fmt.Println("Error updating cache versions after update:", err)
		return err
	}
	return nil
}

func UpdateOne[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	record T, columnsToInclude ...Coln,
) error {
	return Update(&[]T{record}, columnsToInclude...)
}

func UpdateExclude[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToExclude ...Coln,
) error {

	runSelfParseIfDefined(records)

	refTable := initStructTable[E, T](new(E))
	scyllaTable := getOrCompileScyllaTable(refTable)

	managedValues, err := applyWriteManagedColumns(records, scyllaTable, false)
	if err != nil {
		return err
	}

	queryStatements := makeUpdateStatementsBase(records, managedValues, nil, columnsToExclude, false)
	queryInsert := makeQueryStatement(queryStatements)

	if err := QueryExec(queryInsert); err != nil {
		fmt.Println(queryInsert)
		fmt.Println("Error inserting records:", err)
		return err
	}

	affectedColumns := collectAffectedColumnsForExclude(&scyllaTable, columnsToExclude)
	if err := syncIndexGroupsAfterWrite(records, &scyllaTable, managedValues); err != nil {
		fmt.Println("Error syncing index groups after update-exclude:", err)
		return err
	}
	if err := syncTableBackedViews(records, &scyllaTable, affectedColumns); err != nil {
		fmt.Println("Error syncing view tables after update-exclude:", err)
		return err
	}

	// UpdateExclude still mutates records, so it participates in the same cache-version increment flow.
	if err := updateCacheVersionsAfterWrite(records, scyllaTable); err != nil {
		fmt.Println("Error updating cache versions after update-exclude:", err)
		return err
	}
	return nil
}

func InsertOrUpdate[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T,
	isRecordForInsert func(e *T) bool,
	columnsToExcludeUpdate []Coln,
	columnsToExcludeInsert ...Coln,
) error {

	recordsToInsert := []T{}
	recordsToUpdate := []T{}

	for _, e := range *records {
		if isRecordForInsert(&e) {
			recordsToInsert = append(recordsToInsert, e)
		} else {
			recordsToUpdate = append(recordsToUpdate, e)
		}
	}

	if len(recordsToUpdate) > 0 {
		fmt.Println("Registros a actualizar:", len(recordsToUpdate))
		if err := UpdateExclude(&recordsToUpdate, columnsToExcludeUpdate...); err != nil {
			return err
		}
	}

	if len(recordsToInsert) > 0 {
		fmt.Println("Registros a insertar:", len(recordsToInsert))
		if err := Insert(&recordsToInsert, columnsToExcludeInsert...); err != nil {
			return err
		}
	}

	return nil
}
