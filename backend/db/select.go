package db

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
	"unsafe"

	"github.com/viant/xunsafe"
	"golang.org/x/sync/errgroup"
)

type smartRangeDef struct {
	from func(v, min, max string) string
	to   func(v, min, max string) string
}

var smartRangeMap = map[string]smartRangeDef{
	">=": {
		from: func(v, min, max string) string { return v },
		to:   func(v, min, max string) string { return max },
	},
	">": {
		from: func(v, min, max string) string { return v + "\uffff" },
		to:   func(v, min, max string) string { return max },
	},
	"<=": {
		from: func(v, min, max string) string { return min },
		to:   func(v, min, max string) string { return v + "\uffff" },
	},
	"<": {
		from: func(v, min, max string) string { return min },
		to:   func(v, min, max string) string { return v },
	},
}

type posibleIndex struct {
	indexView    *viewInfo
	colsIncluded []int16
	colsMissing  []int16
	priority     int8
}

type compositeBucketSelection struct {
	bucketSize int8
	bucketID   int64
}

// compositeBucketQueryPlan captures the generated indexed where-clauses plus source statements to keep for final exact filtering.
type compositeBucketQueryPlan struct {
	whereStatements  []boundWhereClause
	handledColumns   map[string]bool
	filterStatements []ColumnStatement
}

func canUseProjectedView(selectedColumnNames []string, view *viewInfo) bool {
	if view == nil || len(view.availableColumns) == 0 {
		return true
	}

	for _, selectedColumnName := range selectedColumnNames {
		if !slices.Contains(view.availableColumns, selectedColumnName) {
			fmt.Printf("Projected view rejected: view=%s missing_column=%s available=%v\n",
				view.name, selectedColumnName, view.availableColumns)
			return false
		}
	}

	return true
}

func deduplicateRecordsByPrimaryKey[E any](records *[]E, scyllaTable ScyllaTable[any]) {
	if records == nil || len(*records) <= 1 {
		return
	}

	recordsUniqueMap := map[string]E{}
	for _, record := range *records {
		recordPointer := xunsafe.AsPointer(&record)
		recordKey := makePrimaryKeyRecordKey(recordPointer, scyllaTable)
		recordsUniqueMap[recordKey] = record
	}

	recordsDeduplicated := make([]E, 0, len(recordsUniqueMap))
	for _, record := range recordsUniqueMap {
		recordsDeduplicated = append(recordsDeduplicated, record)
	}
	*records = recordsDeduplicated
}

func buildRemainingWhereClauses(statements []ColumnStatement) []boundWhereClause {
	clauses := []boundWhereClause{}
	for _, statement := range statements {
		if len(statement.From) > 0 {
			for idx := range statement.From {
				columnName := statement.From[idx].Col
				// Keep BETWEEN inclusive to match the fluent API and post-filter behavior.
				clauses = append(clauses, boundWhereClause{
					Clause: fmt.Sprintf("%v >= ?", columnName),
					Values: []any{statement.From[idx].Value},
				})
				clauses = append(clauses, boundWhereClause{
					Clause: fmt.Sprintf("%v <= ?", columnName),
					Values: []any{statement.To[idx].Value},
				})
			}
			continue
		}

		if statement.Operator == "IN" {
			placeholders := make([]string, 0, len(statement.Values))
			queryValues := make([]any, 0, len(statement.Values))
			for _, value := range statement.Values {
				placeholders = append(placeholders, "?")
				queryValues = append(queryValues, value)
			}
			clauses = append(clauses, boundWhereClause{
				Clause: fmt.Sprintf("%v IN (%v)", statement.Col, strings.Join(placeholders, ", ")),
				Values: queryValues,
			})
			continue
		}

		clauses = append(clauses, boundWhereClause{
			Clause: fmt.Sprintf("%v %v ?", statement.Col, statement.Operator),
			Values: []any{statement.Value},
		})
	}
	return clauses
}

func getStatementValueListForHash(statement ColumnStatement) ([]int64, bool) {
	// Only numeric operators that can be converted into deterministic hash inputs are allowed in composite hash planning.
	switch statement.Operator {
	case "=", "CONTAINS":
		return []int64{convertToInt64(statement.Value)}, true
	case "IN":
		if len(statement.Values) == 0 {
			return nil, false
		}
		values := make([]int64, 0, len(statement.Values))
		for _, value := range statement.Values {
			values = append(values, convertToInt64(value))
		}
		return values, true
	default:
		return nil, false
	}
}

func flattenRecordInt64Values(rawValue any) []int64 {
	// Normalize scalar/slice/pointer numeric values for uniform in-memory predicate checks.
	if rawValue == nil {
		return nil
	}
	rv := reflect.ValueOf(rawValue)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []int64{convertToInt64(rv.Interface())}
	}
	values := make([]int64, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		values = append(values, convertToInt64(rv.Index(i).Interface()))
	}
	return values
}

func getBetweenRange(statement ColumnStatement, isWeek bool) (int64, int64, bool) {
	// Composite planner currently targets BETWEEN ranges; this keeps behavior explicit and predictable.
	if statement.Operator != "BETWEEN" || len(statement.From) == 0 || len(statement.To) == 0 {
		return 0, 0, false
	}
	from, to := normalizeCompositeRange(convertToInt64(statement.From[0].Value), convertToInt64(statement.To[0].Value), isWeek)
	return from, to, true
}

func planCompositeBuckets(from, to int64, bucketSizes []int8) []compositeBucketSelection {
	// Choose the minimum statement-count bucket cover while enforcing max total overfetch of 2 units.
	if to < from || len(bucketSizes) == 0 {
		return nil
	}

	type interval struct {
		selection compositeBucketSelection
		start     int64
		end       int64
		overfetch int64
	}

	maxOverfetch := int64(0)
	for _, size := range bucketSizes {
		maxOverfetch += int64(size)
	}

	maxOverfetch = maxOverfetch / int64(len(bucketSizes))
	minCoverage := from - maxOverfetch
	maxCoverage := to + maxOverfetch

	intervals := []interval{}
	// Precompute candidate bucket intervals that intersect the target range and satisfy overfetch constraints.
	for _, bucketSize := range bucketSizes {
		size := int64(bucketSize)
		if size <= 0 {
			continue
		}

		minBucketID := minCoverage / size
		maxBucketID := maxCoverage / size
		for bucketID := minBucketID; bucketID <= maxBucketID; bucketID++ {
			start := bucketID * size
			end := start + size - 1
			if end < from || start > to {
				continue
			}
			overfetch := int64(0)
			if start < from {
				overfetch += from - start
			}
			if end > to {
				overfetch += end - to
			}
			if overfetch > maxOverfetch {
				continue
			}
			intervals = append(intervals, interval{
				selection: compositeBucketSelection{bucketSize: bucketSize, bucketID: bucketID},
				start:     start,
				end:       end,
				overfetch: overfetch,
			})
		}
	}

	type planState struct {
		count      int
		overfetch  int64
		selections []compositeBucketSelection
		valid      bool
	}

	memo := map[string]planState{}
	var solve func(currentWeek, usedOverfetch int64) planState
	solve = func(currentWeek, usedOverfetch int64) planState {
		// DP state represents the best plan from currentWeek to end, given accumulated overfetch.
		if currentWeek > to {
			return planState{valid: true}
		}

		memoKey := fmt.Sprintf("%d|%d", currentWeek, usedOverfetch)
		if state, ok := memo[memoKey]; ok {
			return state
		}

		best := planState{valid: false}
		for _, intervalDef := range intervals {
			if intervalDef.start > currentWeek || intervalDef.end < currentWeek {
				continue
			}
			nextOverfetch := usedOverfetch + intervalDef.overfetch
			if nextOverfetch > maxOverfetch {
				continue
			}

			nextState := solve(intervalDef.end+1, nextOverfetch)
			if !nextState.valid {
				continue
			}

			candidate := planState{
				count:      nextState.count + 1,
				overfetch:  nextState.overfetch + intervalDef.overfetch,
				selections: append([]compositeBucketSelection{intervalDef.selection}, nextState.selections...),
				valid:      true,
			}

			if !best.valid || candidate.count < best.count ||
				(candidate.count == best.count && candidate.overfetch < best.overfetch) {
				best = candidate
			}
		}

		memo[memoKey] = best
		return best
	}

	plan := solve(from, 0)
	if !plan.valid {
		return nil
	}

	return plan.selections
}

func makePrimaryKeyRecordKey(ptr unsafe.Pointer, scyllaTable ScyllaTable[any]) string {
	// Dedupe key always includes partition key plus clustering keys to preserve table key semantics.
	parts := []string{}
	partKey := scyllaTable.GetPartKey()
	if partKey != nil && !partKey.IsNil() {
		parts = append(parts, fmt.Sprintf("%s=%v", partKey.GetName(), scyllaTable.GetPartValue(ptr)))
	}
	keyValues := scyllaTable.GetKeyValues(ptr)
	for keyIndex, keyColumn := range scyllaTable.keys {
		parts = append(parts, fmt.Sprintf("%s=%v", keyColumn.GetName(), keyValues[keyIndex]))
	}
	return strings.Join(parts, "|")
}

func recordMatchesPostFilter(ptr unsafe.Pointer, statements []ColumnStatement, scyllaTable ScyllaTable[any]) bool {
	// Final in-memory filtering guarantees exact semantics after overfetch (e.g. packed indexes with DecimalSize truncation).
	for _, statement := range statements {
		column := scyllaTable.columnsMap[statement.Col]
		if column == nil {
			return false
		}
		rawValue := column.GetRawValue(ptr)

		switch statement.Operator {
		case "=":
			if convertToInt64(rawValue) != convertToInt64(statement.Value) {
				return false
			}
		case ">":
			if convertToInt64(rawValue) <= convertToInt64(statement.Value) {
				return false
			}
		case ">=":
			if convertToInt64(rawValue) < convertToInt64(statement.Value) {
				return false
			}
		case "<":
			if convertToInt64(rawValue) >= convertToInt64(statement.Value) {
				return false
			}
		case "<=":
			if convertToInt64(rawValue) > convertToInt64(statement.Value) {
				return false
			}
		case "IN":
			matched := false
			rawValueInt := convertToInt64(rawValue)
			for _, value := range statement.Values {
				if rawValueInt == convertToInt64(value) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		case "CONTAINS":
			contains := false
			statementValue := convertToInt64(statement.Value)
			for _, value := range flattenRecordInt64Values(rawValue) {
				if value == statementValue {
					contains = true
					break
				}
			}
			if !contains {
				return false
			}
		case "BETWEEN":
			if len(statement.From) == 0 || len(statement.To) == 0 {
				return false
			}
			valueI64 := convertToInt64(rawValue)
			fromI64 := convertToInt64(statement.From[0].Value)
			toI64 := convertToInt64(statement.To[0].Value)
			if valueI64 < fromI64 || valueI64 > toI64 {
				return false
			}
		}
	}
	return true
}

func formatDebugQuery(queryStr string, queryValues []any) string {
	// Keep debug output short by collapsing the projected column list before injecting values.
	minifiedQuery := minifySelectColumns(queryStr)
	if len(queryValues) == 0 {
		return minifiedQuery
	}

	var formattedQuery strings.Builder
	formattedQuery.Grow(len(minifiedQuery) + len(queryValues)*8)

	valueIndex := 0
	for _, queryChar := range minifiedQuery {
		if queryChar == '?' && valueIndex < len(queryValues) {
			formattedQuery.WriteString(formatDebugValue(queryValues[valueIndex]))
			valueIndex++
			continue
		}
		formattedQuery.WriteRune(queryChar)
	}

	if valueIndex < len(queryValues) {
		formattedQuery.WriteString(" /* extra_values=")
		formattedQuery.WriteString(formatDebugValues(queryValues[valueIndex:]))
		formattedQuery.WriteString(" */")
	}

	return formattedQuery.String()
}

func minifySelectColumns(queryStr string) string {
	// Only rewrite the top-level SELECT projection so the rest of the query stays intact.
	upperQuery := strings.ToUpper(queryStr)
	selectIndex := strings.Index(upperQuery, "SELECT ")
	fromIndex := strings.Index(upperQuery, " FROM ")
	if selectIndex < 0 || fromIndex <= selectIndex+len("SELECT ") {
		return queryStr
	}

	selectColumns := queryStr[selectIndex+len("SELECT ") : fromIndex]
	columnCount := countTopLevelCSVColumns(selectColumns)
	if columnCount == 0 {
		return queryStr
	}

	return queryStr[:selectIndex] + fmt.Sprintf("SELECT (%d)", columnCount) + queryStr[fromIndex:]
}

func countTopLevelCSVColumns(columnsText string) int {
	// Count only commas at the top level so function calls like foo(a, b) stay as one projected column.
	trimmedColumns := strings.TrimSpace(columnsText)
	if len(trimmedColumns) == 0 {
		return 0
	}

	columnCount := 1
	parenthesisDepth := 0
	for _, currentChar := range trimmedColumns {
		switch currentChar {
		case '(':
			parenthesisDepth++
		case ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
		case ',':
			if parenthesisDepth == 0 {
				columnCount++
			}
		}
	}

	return columnCount
}

func formatDebugValues(queryValues []any) string {
	formattedValues := make([]string, 0, len(queryValues))
	for _, queryValue := range queryValues {
		formattedValues = append(formattedValues, formatDebugValue(queryValue))
	}
	return "[" + strings.Join(formattedValues, ", ") + "]"
}

func formatDebugValue(rawValue any) string {
	// Render bound values as valid-enough CQL literals for debugging and copy-paste inspection.
	if rawValue == nil {
		return "null"
	}

	switch value := rawValue.(type) {
	case string:
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	case []byte:
		return fmt.Sprintf("0x%x", value)
	case fmt.Stringer:
		return "'" + strings.ReplaceAll(value.String(), "'", "''") + "'"
	case time.Time:
		return "'" + value.Format(time.RFC3339Nano) + "'"
	case bool:
		if value {
			return "true"
		}
		return "false"
	}

	valueRef := reflect.ValueOf(rawValue)
	for valueRef.Kind() == reflect.Pointer {
		if valueRef.IsNil() {
			return "null"
		}
		valueRef = valueRef.Elem()
	}

	switch valueRef.Kind() {
	case reflect.String:
		return "'" + strings.ReplaceAll(valueRef.String(), "'", "''") + "'"
	case reflect.Slice, reflect.Array:
		formattedValues := make([]string, 0, valueRef.Len())
		for idx := 0; idx < valueRef.Len(); idx++ {
			formattedValues = append(formattedValues, formatDebugValue(valueRef.Index(idx).Interface()))
		}
		return "[" + strings.Join(formattedValues, ", ") + "]"
	}

	return fmt.Sprintf("%v", valueRef.Interface())
}

func normalizeScannedValue(value any) any {
	// RowData stores pointers for scalar columns; grouped virtual decomposition works with the plain value.
	if value == nil {
		return nil
	}
	valueRef := reflect.ValueOf(value)
	for valueRef.Kind() == reflect.Pointer {
		if valueRef.IsNil() {
			return nil
		}
		valueRef = valueRef.Elem()
	}
	return valueRef.Interface()
}

func scanSelectQueryRows[E any](
	queryStr string,
	queryValues []any,
	scanColumns []selectScanColumn,
	scyllaTable ScyllaTable[any],
	refRecords *[]E,
	postFilterStatements []ColumnStatement,
	scanHandler func(record *E) bool,
	queryNoticeTime time.Time,
) error {
	usePostFilter := len(postFilterStatements) > 0
	
	
	doScan := func() error {
		if ShouldLog() {
			// Log the executable statement with compact projection output to keep noisy SELECTs readable.
			fmt.Printf("Select Elapsed: %d | %s\n",
				time.Since(queryNoticeTime).Milliseconds(),
				formatDebugQuery(queryStr, queryValues),
			)
		}

		iter := getScyllaConnection().Query(queryStr, queryValues...).Iter()
		rd, err := iter.RowData()
		if err != nil {
			fmt.Println("Error on RowData::", err)
			if strings.Contains(err.Error(), "use ALLOW FILTERING") {
				fmt.Println("Posible Indexes or Views::")
				for _, capability := range scyllaTable.capabilities {
					fmt.Println(capability.Signature)
				}
			}
			return err
		}
		
		rowsScaned := 0
		rowsFinal := 0

		scanner := iter.Scanner()
		for scanner.Next() {
			rowsScaned++
			
			rowValues := rd.Values
			if err := scanner.Scan(rowValues...); err != nil {
				fmt.Println("Error on scan::", err)
				return err
			}

			record := new(E)
			recordPtr := xunsafe.AsPointer(record)
			shouldLog := ShouldLogFull()

			/* 
			if shouldLog {
				fmt.Printf("\n--- Scanning record %d ---\n", atomic.LoadUint32(&LogCount)+1)
			}
			*/

			// Map each scanned DB column into the destination struct using precomputed column metadata.
			for columnIndex, scanColumn := range scanColumns {
				columnName := scanColumn.ColumnName
				column := scyllaTable.columnsMap[columnName]
				value := rowValues[columnIndex]

				if shouldLog {
					valueText := "nil"
					if value != nil {
						valueText = fmt.Sprintf("%v", reflect.Indirect(reflect.ValueOf(value)).Interface())
					}
					offset := uintptr(0)
					if column.GetInfo().Field != nil {
						offset = column.GetInfo().Field.Offset
					}
					fmt.Printf("Col: %-20s (Field: %-20s) | Offset: %-4d | DB Value: %s\n", columnName, column.GetInfo().FieldName, offset, valueText)
				}

				if value == nil {
					continue
				}
				if scanColumn.DecomposeView != nil && scanColumn.DecomposeView.decomposeVirtualValue != nil {
					// Packed group keys are scanned as one virtual column and decomposed back into the primitive fields.
					decomposedValues := scanColumn.DecomposeView.decomposeVirtualValue(normalizeScannedValue(value))
					for valueIndex, sourceColumn := range scanColumn.DecomposeView.packedSourceColumns {
						if valueIndex >= len(decomposedValues) {
							break
						}
						sourceColumn.SetValue(recordPtr, decomposedValues[valueIndex])
					}
					continue
				}
				if shouldLog {
					fmt.Printf("Calling SetValue for Col: %s\n", columnName)
				}
				column.SetValue(recordPtr, value)
			}
			/* 
			if shouldLog {
				recordJSON, _ := json.MarshalIndent(record, "", "  ")
				fmt.Printf("Resulting Struct: %s\n", string(recordJSON))
			}
			*/

			// Post-filter keeps exact semantics when indexed query planning intentionally overfetches.
			if usePostFilter && !recordMatchesPostFilter(xunsafe.AsPointer(record), postFilterStatements, scyllaTable) {
				continue
			}

			// Let callers process the decoded row and optionally avoid storing it to reduce peak memory.
			if scanHandler != nil && scanHandler(record) {
				continue
			}

			rowsFinal++
			*refRecords = append(*refRecords, *record)
		}

		fmt.Printf(`Table %v | Rows Scanned %v | Final %v`+"\n",scyllaTable.name, rowsScaned, rowsFinal)
		
		return nil
	}

	err := doScan()
	if err != nil && strings.Contains(err.Error(), "no hosts available") {
		scyllaSession = nil
		fmt.Println("Reconectando con ScyllaDB...")
		getScyllaConnection()
		err = doScan()
	}

	return err
}

func executeBoundSelectQueries[E any](
	recordsGetted *[]E,
	boundStatements []BoundSelectStatement,
	scanColumns []selectScanColumn,
	requiresDeduplication bool,
	scanHandler func(record *E) bool,
	scyllaTable ScyllaTable[any],
	queryNoticeTime time.Time,
) error {
	// Keep query execution shared so cached and legacy planning paths use the same scan, merge, and dedup behavior.
	if len(boundStatements) == 0 {
		return nil
	}

	recordsMap := map[int]*[]E{}
	for statementIndex := range boundStatements {
		recordsMap[statementIndex] = &[]E{}
	}

	eg := errgroup.Group{}
	for statementIndex, boundStatement := range boundStatements {
		records := recordsMap[statementIndex]
		recordsTarget := records
		if len(boundStatements) == 1 {
			recordsTarget = recordsGetted
		}

		// fmt.Println("Query::", boundStatement.QueryStr)
		boundStatement := boundStatement
		eg.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("Panic in select goroutine: %v", r)
				}
			}()

			return scanSelectQueryRows(
				boundStatement.QueryStr,
				boundStatement.QueryValues,
				scanColumns,
				scyllaTable,
				recordsTarget,
				boundStatement.PostFilterStatements,
				scanHandler,
				queryNoticeTime,
			)
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if len(boundStatements) > 1 {
		if requiresDeduplication {
			recordsMerged := []E{}
			for _, records := range recordsMap {
				recordsMerged = append(recordsMerged, (*records)...)
			}
			*recordsGetted = append(*recordsGetted, recordsMerged...)
			deduplicateRecordsByPrimaryKey(recordsGetted, scyllaTable)
		} else {
			for _, records := range recordsMap {
				*recordsGetted = append(*recordsGetted, *records...)
			}
		}
	} else if requiresDeduplication {
		deduplicateRecordsByPrimaryKey(recordsGetted, scyllaTable)
	}

	return nil
}

func tryBuildCompositeBucketPlan(statements []ColumnStatement, scyllaTable ScyllaTable[any]) *compositeBucketQueryPlan {
	// Build a specialized plan for composite-bucket indexes before generic capability matching.
	partitionColumn := scyllaTable.GetPartKey()
	if partitionColumn == nil || partitionColumn.IsNil() {
		return nil
	}

	statementByColumn := map[string][]ColumnStatement{}
	for _, statement := range statements {
		statementByColumn[statement.Col] = append(statementByColumn[statement.Col], statement)
	}

	partitionStatements := statementByColumn[partitionColumn.GetName()]
	if len(partitionStatements) == 0 || partitionStatements[0].Operator != "=" {
		return nil
	}

	for _, compositeIndex := range scyllaTable.compositeBucketIndexes {
		// The composite strategy requires a range on the bucketed column and hashable predicates for remaining source columns.
		bucketStatements := statementByColumn[compositeIndex.bucketColumn.GetName()]
		if len(bucketStatements) == 0 {
			continue
		}

		bucketIsWeek := compositeIndex.bucketIsWeek
		from, to, ok := getBetweenRange(bucketStatements[0], bucketIsWeek)
		if !ok {
			continue
		}

		valueOptionsByColumn := map[string][]int64{}
		filterStatements := []ColumnStatement{bucketStatements[0]}
		handledColumns := map[string]bool{
			compositeIndex.bucketColumn.GetName(): true,
			partitionColumn.GetName():             true,
		}
		isValid := true

		for _, sourceColumn := range compositeIndex.sourceColumns {
			if sourceColumn.GetName() == compositeIndex.bucketColumn.GetName() {
				continue
			}

			sourceStatements := statementByColumn[sourceColumn.GetName()]
			if len(sourceStatements) == 0 {
				isValid = false
				break
			}

			values, valueOk := getStatementValueListForHash(sourceStatements[0])
			if !valueOk || len(values) == 0 {
				isValid = false
				break
			}

			valueOptionsByColumn[sourceColumn.GetName()] = values
			filterStatements = append(filterStatements, sourceStatements[0])
			handledColumns[sourceColumn.GetName()] = true
		}

		if !isValid {
			continue
		}

		bucketSelections := planCompositeBuckets(from, to, compositeIndex.bucketSizes)
		if len(bucketSelections) == 0 {
			continue
		}

		queryValueGroups := []map[string]int64{{}}
		// Expand all hash input combinations for IN/equality source predicates.
		for _, sourceColumn := range compositeIndex.sourceColumns {
			if sourceColumn.GetName() == compositeIndex.bucketColumn.GetName() {
				continue
			}
			nextGroups := []map[string]int64{}
			for _, queryValueGroup := range queryValueGroups {
				for _, value := range valueOptionsByColumn[sourceColumn.GetName()] {
					nextGroup := map[string]int64{}
					for key, currentValue := range queryValueGroup {
						nextGroup[key] = currentValue
					}
					nextGroup[sourceColumn.GetName()] = value
					nextGroups = append(nextGroups, nextGroup)
				}
			}
			queryValueGroups = nextGroups
		}

		whereStatements := []boundWhereClause{}
		// Emit one indexed CONTAINS statement per (bucket selection x source tuple) combination.
		partitionValue := partitionStatements[0].GetValue()
		for _, bucketSelection := range bucketSelections {
			virtualColumn := compositeIndex.virtualColumnsBySize[bucketSelection.bucketSize]
			if virtualColumn == nil {
				isValid = false
				break
			}

			for _, queryValueGroup := range queryValueGroups {
				hashValues := make([]int64, 0, len(compositeIndex.sourceColumns))
				for _, sourceColumn := range compositeIndex.sourceColumns {
					if sourceColumn.GetName() == compositeIndex.bucketColumn.GetName() {
						hashValues = append(hashValues, bucketSelection.bucketID)
						continue
					}
					hashValues = append(hashValues, queryValueGroup[sourceColumn.GetName()])
				}

				hashValue := HashInt64(hashValues...)
				whereStatements = append(whereStatements, boundWhereClause{
					Clause: fmt.Sprintf("%v = ? AND %v CONTAINS ?", partitionColumn.GetName(), virtualColumn.GetName()),
					Values: []any{partitionValue, hashValue},
				})
			}
		}

		if !isValid || len(whereStatements) == 0 {
			continue
		}

		rawFrom := convertToInt64(bucketStatements[0].From[0].Value)
		rawTo := convertToInt64(bucketStatements[0].To[0].Value)
		fmt.Printf("CompositeBucketing select plan selected: index=%s buckets=%d statements=%d rangeNorm=[%d,%d] rangeRaw=[%d,%d] isWeek=%v\n",
			compositeIndex.name, len(bucketSelections), len(whereStatements), from, to, rawFrom, rawTo, bucketIsWeek)

		return &compositeBucketQueryPlan{
			whereStatements:  whereStatements,
			handledColumns:   handledColumns,
			filterStatements: filterStatements,
		}
	}

	return nil
}

// selectExec executes a query using TableSchema and TableInfo
func selectExec[E any](
	recordsGetted *[]E,
	tableInfo *TableInfo,
	scyllaTable ScyllaTable[any],
	scanHandler func(record *E) bool,
	selectStartTime time.Time,
) error {
	if len(scyllaTable.keyspace) == 0 {
		scyllaTable.keyspace = connParams.Keyspace
	}
	if len(scyllaTable.keyspace) == 0 {
		return errors.New("no se ha especificado un keyspace")
	}

	compiledSelect, err := tryGetOrCompileSelectStatement(tableInfo, scyllaTable)
	if err != nil {
		return err
	}

	boundPlan, err := compiledSelect.Compute(tableInfo, scyllaTable)
	if err != nil {
		return err
	}

	if err := executeBoundSelectQueries(
		recordsGetted,
		boundPlan.Statements,
		boundPlan.ScanColumns,
		boundPlan.RequiresDeduplication,
		scanHandler,
		scyllaTable,
		selectStartTime,
	); err != nil {
		return err
	}

	if boundPlan.AssignCacheVersions {
		if err := assignCacheVersionsAfterSelect(recordsGetted, scyllaTable); err != nil {
			return err
		}
	}

	return nil
}
