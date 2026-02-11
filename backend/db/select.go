package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
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
	whereStatements  []string
	handledColumns   map[string]bool
	filterStatements []ColumnStatement
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
		parts = append(parts, fmt.Sprintf("%s=%v", partKey.GetName(), partKey.GetRawValue(ptr)))
	}
	for _, keyColumn := range scyllaTable.keys {
		parts = append(parts, fmt.Sprintf("%s=%v", keyColumn.GetName(), keyColumn.GetRawValue(ptr)))
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

func scanSelectQueryRows[E any](
	queryStr string,
	queryValues []any,
	columnNames []string,
	scyllaTable ScyllaTable[any],
	refRecords *[]E,
	postFilterStatements []ColumnStatement,
) error {
	usePostFilter := len(postFilterStatements) > 0

	doScan := func() error {
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

		scanner := iter.Scanner()
		fmt.Println("starting iterator | columns::", len(scyllaTable.columns))

		for scanner.Next() {
			rowValues := rd.Values
			if err := scanner.Scan(rowValues...); err != nil {
				fmt.Println("Error on scan::", err)
				return err
			}

			record := new(E)
			recordPtr := xunsafe.AsPointer(record)
			shouldLog := ShouldLog()

			if shouldLog {
				fmt.Printf("\n--- Scanning record %d ---\n", atomic.LoadUint32(&LogCount)+1)
			}

			// Map each scanned DB column into the destination struct using precomputed column metadata.
			for columnIndex, columnName := range columnNames {
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
				if shouldLog {
					fmt.Printf("Calling SetValue for Col: %s\n", columnName)
				}
				column.SetValue(recordPtr, value)
			}

			if shouldLog {
				recordJSON, _ := json.MarshalIndent(record, "", "  ")
				fmt.Printf("Resulting Struct: %s\n", string(recordJSON))
				IncrementLogCount()
			}

			// Post-filter keeps exact semantics when indexed query planning intentionally overfetches.
			if usePostFilter && !recordMatchesPostFilter(xunsafe.AsPointer(record), postFilterStatements, scyllaTable) {
				continue
			}

			*refRecords = append(*refRecords, *record)
		}

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

func tryBuildCompositeBucketPlan(statements []ColumnStatement, scyllaTable ScyllaTable[any]) *compositeBucketQueryPlan {
	// Build a specialized plan for HashIndexes+CompositeBucketing before generic capability matching.
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

		whereStatements := []string{}
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
				whereStatements = append(whereStatements,
					fmt.Sprintf("%v = %v AND %v CONTAINS %v", partitionColumn.GetName(), partitionValue, virtualColumn.GetName(), hashValue),
				)
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
func selectExec[E any](recordsGetted *[]E, tableInfo *TableInfo, scyllaTable ScyllaTable[any]) error {

	viewTableName := scyllaTable.name

	if len(scyllaTable.keyspace) == 0 {
		scyllaTable.keyspace = connParams.Keyspace
	}
	if len(scyllaTable.keyspace) == 0 {
		return errors.New("no se ha especificado un keyspace")
	}

	columnNames := []string{}
	if len(tableInfo.columnsInclude) > 0 {
		for _, col := range tableInfo.columnsInclude {
			columnNames = append(columnNames, col.GetName())
		}
	} else {
		columnsExclude := []string{}
		for _, col := range tableInfo.columnsExclude {
			columnsExclude = append(columnsExclude, col.GetName())
		}
		for _, col := range scyllaTable.columns {
			if !slices.Contains(columnsExclude, col.GetName()) && !col.GetInfo().IsVirtual {
				columnNames = append(columnNames, col.GetName())
			}
		}
	}
	// Force internal selection of partition/key when cache-version is enabled to resolve group IDs reliably.
	columnNames = ensureCacheVersionColumnsForSelect(columnNames, scyllaTable)

	queryTemplate := fmt.Sprintf("SELECT %v ", strings.Join(columnNames, ", ")) + "FROM %v.%v %v"
	hashOperators := []string{"=", "IN"}
	isHash := true

	statements := []ColumnStatement{}
	colsWhereIdx := []int16{}

	for _, st := range tableInfo.statements {
		col := scyllaTable.columnsMap[st.Col]

		colsWhereIdx = append(colsWhereIdx, col.GetInfo().Idx)
		statements = append(statements, st)
		if isHash && !slices.Contains(hashOperators, st.Operator) {
			isHash = false
		}
	}

	// Between Statement
	if len(tableInfo.between.From) > 0 {
		statements = append(statements, tableInfo.between)
	}

	statementsSelected := []ColumnStatement{}
	statementsRemain := []ColumnStatement{}
	whereStatements := []string{}
	orderColumn := scyllaTable.keys[0]
	postFilterStatements := []ColumnStatement{}
	usePostFilter := false
	shouldDeduplicateFanoutResults := false

	compositePlan := tryBuildCompositeBucketPlan(statements, scyllaTable)
	if compositePlan != nil {
		// Composite plan handles the hash+bucket part; remaining statements are applied as normal SQL predicates.
		usePostFilter = true
		shouldDeduplicateFanoutResults = true
		whereStatements = compositePlan.whereStatements
		postFilterStatements = compositePlan.filterStatements
		for _, statement := range statements {
			if !compositePlan.handledColumns[statement.Col] {
				statementsRemain = append(statementsRemain, statement)
			}
		}
	} else {
		bestCap := MatchQueryCapability(statements, scyllaTable.capabilities)

		if bestCap != nil {
			fmt.Println("Best Match Signature:", bestCap.Signature)

			handledCols := make(map[string]bool)
			parts := strings.Split(bestCap.Signature, "|")
			for i := 0; i < len(parts); i += 2 {
				handledCols[parts[i]] = true
			}

			if bestCap.Source != nil {
				viewOrIndex := bestCap.Source
				fmt.Println("Selected Index/View:", viewOrIndex.name)
				if viewOrIndex.Type >= 6 {
					viewTableName = viewOrIndex.name
				}

				if viewOrIndex.getStatement != nil {
					// Identify which statements match the view columns
					for _, st := range statements {
						if slices.Contains(viewOrIndex.columns, st.Col) {
							statementsSelected = append(statementsSelected, st)
						} else if len(st.From) > 0 { // BETWEEN
							isIncluded := true
							for _, e := range st.From {
								if !slices.Contains(viewOrIndex.columns, e.Col) {
									isIncluded = false
									break
								}
							}
							if isIncluded {
								statementsSelected = append(statementsSelected, st)
							} else {
								statementsRemain = append(statementsRemain, st)
							}
						} else {
							statementsRemain = append(statementsRemain, st)
						}
					}
					orderColumn = viewOrIndex.column
					whereStatements = viewOrIndex.getStatement(statementsSelected...)

					if viewOrIndex.RequiresPostFilter {
						// Packed indexes with DecimalSize truncation can overfetch; enforce exact semantics in memory.
						usePostFilter = true
						shouldDeduplicateFanoutResults = true
						postFilterStatements = slices.Clone(statementsSelected)
					}
				} else {
					// Direct index usage, no special statement generator
					statementsRemain = statements
				}
			} else if bestCap.IsKey {
				// Key or KeyConcatenated logic
				keyCol := scyllaTable.keys[0]
				hasKeyColQuery := false
				for _, st := range statements {
					if st.Col == keyCol.GetName() {
						hasKeyColQuery = true
						break
					}
				}

				if !hasKeyColQuery && len(scyllaTable.keyConcatenated) > 0 {
					// Apply KeyConcatenated smart logic
					prefixValues := []any{}
					var rangeSt *ColumnStatement
					handledInKeyConcat := make(map[string]bool)

					for _, concatCol := range scyllaTable.keyConcatenated {
						found := false
						for _, st := range statements {
							if st.Col == concatCol.GetName() {
								if st.Operator == "=" {
									prefixValues = append(prefixValues, st.Value)
									handledInKeyConcat[st.Col] = true
									found = true
									break
								} else if slices.Contains(rangeOperators, st.Operator) || st.Operator == "BETWEEN" {
									rangeSt = &st
									handledInKeyConcat[st.Col] = true
									found = true
									break
								}
							}
						}
						if !found || rangeSt != nil {
							break
						}
					}

					if len(prefixValues) > 0 || rangeSt != nil {
						prefixStr := ""
						if len(prefixValues) > 0 {
							prefixStr = Concat62(prefixValues...)
						}

						var newSt ColumnStatement
						if rangeSt == nil {
							val := prefixStr
							if len(prefixValues) == len(scyllaTable.keyConcatenated) {
								newSt = ColumnStatement{Col: keyCol.GetName(), Operator: "=", Value: val}
							} else {
								newSt = ColumnStatement{
									Col:      keyCol.GetName(),
									Operator: "BETWEEN",
									From:     []ColumnStatement{{Col: keyCol.GetName(), Value: val + "_"}},
									To:       []ColumnStatement{{Col: keyCol.GetName(), Value: val + "_\uffff"}},
								}
							}
						} else {
							if rangeSt.Operator == "BETWEEN" {
								valFrom := Concat62(append(prefixValues, rangeSt.From[0].Value)...)
								valTo := Concat62(append(prefixValues, rangeSt.To[0].Value)...)
								newSt = ColumnStatement{
									Col:      keyCol.GetName(),
									Operator: "BETWEEN",
									From:     []ColumnStatement{{Col: keyCol.GetName(), Value: valFrom}},
									To:       []ColumnStatement{{Col: keyCol.GetName(), Value: valTo + "\uffff"}},
								}
							} else if trans, ok := smartRangeMap[rangeSt.Operator]; ok {
								valWithRange := Concat62(append(prefixValues, rangeSt.Value)...)
								prefixMin, prefixMax := "", "\uffff"
								if prefixStr != "" {
									prefixMin = prefixStr + "_"
									prefixMax = prefixStr + "_\uffff"
								}
								fromVal := trans.from(valWithRange, prefixMin, prefixMax)
								toVal := trans.to(valWithRange, prefixMin, prefixMax)
								newSt = ColumnStatement{Col: keyCol.GetName(), Operator: "BETWEEN"}
								if fromVal != "" {
									newSt.From = append(newSt.From, ColumnStatement{Col: keyCol.GetName(), Operator: ">=", Value: fromVal})
								}
								if toVal != "" {
									newSt.To = append(newSt.To, ColumnStatement{Col: keyCol.GetName(), Operator: "<", Value: toVal})
								}
							}
						}

						// Add the new PK statement and skip handled concatenated columns
						statementsRemain = append(statementsRemain, newSt)
						for _, st := range statements {
							if !handledInKeyConcat[st.Col] {
								statementsRemain = append(statementsRemain, st)
							}
						}
					} else {
						statementsRemain = statements
					}
				} else if !hasKeyColQuery && len(scyllaTable.keyIntPacking) > 0 {
					// Apply KeyIntPacking smart logic
					prefixValues := []any{}
					var rangeSt *ColumnStatement
					handledInKeyPack := make(map[string]bool)

					for _, packedCol := range scyllaTable.keyIntPacking {
						colName := packedCol.GetName()
						if colName == "autoincrement_placeholder" {
							break
						}
						found := false
						for _, st := range statements {
							if st.Col == colName {
								if st.Operator == "=" {
									prefixValues = append(prefixValues, st.Value)
									handledInKeyPack[st.Col] = true
									found = true
									break
								} else if slices.Contains(rangeOperators, st.Operator) || st.Operator == "BETWEEN" {
									rangeSt = &st
									handledInKeyPack[st.Col] = true
									found = true
									break
								}
							}
						}
						if !found || rangeSt != nil {
							break
						}
					}

					if len(prefixValues) > 0 || rangeSt != nil {
						// Calculation formula: packedValue = (packedValue * 10^col.decimalSize) + val
						// We need to calculate the value and the remaining shift to place it in the correct slot

						makePacked := func(vals []any, rSt *ColumnStatement) (fromVal int64, toVal int64, isEquality bool) {
							remainingDigits := int64(19)
							var currentPacked int64

							// Process equality prefix
							for i, col := range scyllaTable.keyIntPacking {
								colInfo := col.(*columnInfo)
								decSize := int64(colInfo.decimalSize)
								if i == len(scyllaTable.keyIntPacking)-1 && decSize == 0 {
									decSize = remainingDigits
								}
								remainingDigits -= decSize

								if i < len(vals) {
									val := convertToInt64(vals[i])
									currentPacked += val * Pow10Int64(remainingDigits)
								} else if rSt != nil && col.GetName() == rSt.Col {
									// Handle range on current column
									if rSt.Operator == "BETWEEN" {
										fromVal = currentPacked + convertToInt64(rSt.From[0].Value)*Pow10Int64(remainingDigits)
										toVal = currentPacked + (convertToInt64(rSt.To[0].Value)+1)*Pow10Int64(remainingDigits)
									} else {
										// Handle other range operators if needed, or fallback to simple prefix range
										val := convertToInt64(rSt.Value)
										fromVal = currentPacked + val*Pow10Int64(remainingDigits)
										// For range operators like >, < we might need more complex logic,
										// but for now let's focus on the common prefix range case.
										toVal = fromVal + Pow10Int64(remainingDigits)
									}
									return fromVal, toVal, false
								} else {
									// End of provided values
									fromVal = currentPacked
									toVal = currentPacked + Pow10Int64(remainingDigits+decSize)
									isEquality = (i == len(scyllaTable.keyIntPacking))
									return fromVal, toVal, isEquality
								}
							}
							return currentPacked, currentPacked, true
						}

						from, to, isEq := makePacked(prefixValues, rangeSt)

						var newSt ColumnStatement
						if isEq {
							newSt = ColumnStatement{Col: keyCol.GetName(), Operator: "=", Value: from}
						} else {
							newSt = ColumnStatement{
								Col:      keyCol.GetName(),
								Operator: "BETWEEN",
								From:     []ColumnStatement{{Col: keyCol.GetName(), Value: from}},
								To:       []ColumnStatement{{Col: keyCol.GetName(), Value: to}},
							}
						}

						// Add the new PK statement and skip handled packed columns
						statementsRemain = append(statementsRemain, newSt)
						for _, st := range statements {
							if !handledInKeyPack[st.Col] {
								statementsRemain = append(statementsRemain, st)
							}
						}
					} else {
						statementsRemain = statements
					}
				} else {
					statementsRemain = statements
				}
			}
		} else {
			// No match found, fallback to default behavior (all remain)
			statementsRemain = statements
		}
	}

	if len(whereStatements) == 0 {
		whereStatements = append(whereStatements, "")
	}

	// fmt.Println("where statements::", whereStatements)
	whereStatementsRemainSects := []string{}
	for _, st := range statementsRemain {
		if len(st.From) > 0 {
			for i := range st.From {
				colname := st.From[i].Col
				w1 := fmt.Sprintf("%v >= %v", colname, st.From[i].GetValue())
				whereStatementsRemainSects = append(whereStatementsRemainSects, w1)
				w2 := fmt.Sprintf("%v < %v", colname, st.To[i].GetValue())
				whereStatementsRemainSects = append(whereStatementsRemainSects, w2)
			}
		} else {
			whereStatementsRemainSects = append(whereStatementsRemainSects,
				fmt.Sprintf("%v %v %v", st.Col, st.Operator, st.GetValue()))
		}
	}

	whereStatementsRemain := strings.Join(whereStatementsRemainSects, " AND ")
	queryWhereStatements := []string{}

	for _, whereStatement := range whereStatements {
		whereRemainClause := whereStatementsRemain
		if whereStatementsRemain != "" {
			if whereStatement != "" {
				whereRemainClause = " AND " + whereRemainClause
			}
			whereStatement += whereRemainClause
		}
		if whereStatement != "" {
			whereStatement = " WHERE " + whereStatement
		}
		if len(tableInfo.orderBy) > 0 {
			whereStatement += " " + fmt.Sprintf(tableInfo.orderBy, orderColumn.GetName())
		}
		if tableInfo.limit > 0 {
			whereStatement += fmt.Sprintf(" LIMIT %v", tableInfo.limit)
		}
		// Explicitly opt into filtered queries when the handler requested it.
		if tableInfo.allowFilter {
			whereStatement += " ALLOW FILTERING"
		}
		queryWhereStatements = append(queryWhereStatements, whereStatement)
	}

	recordsMap := map[int]*[]E{}
	for i := range queryWhereStatements {
		recordsMap[i] = &[]E{}
	}

	eg := errgroup.Group{}

	for i, whereStatement := range queryWhereStatements {
		queryStr := fmt.Sprintf(queryTemplate, scyllaTable.keyspace, viewTableName, whereStatement)
		records := recordsMap[i]
		recordsTarget := records
		if len(queryWhereStatements) == 1 {
			recordsTarget = recordsGetted
		}

		fmt.Println("Query::", queryStr)
		queryValues := []any{}
		queryPostFilterStatements := []ColumnStatement{}
		if usePostFilter {
			queryPostFilterStatements = postFilterStatements
		}

		eg.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("Panic in select goroutine: %v", r)
				}
			}()

			err = scanSelectQueryRows(queryStr, queryValues, columnNames, scyllaTable, recordsTarget, queryPostFilterStatements)
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if len(queryWhereStatements) > 1 {
		if shouldDeduplicateFanoutResults {
			// Deduplicate fan-out results by partition+primary-key tuple before returning.
			recordsUniqueMap := map[string]E{}
			for _, records := range recordsMap {
				for _, rec := range *records {
					recPtr := xunsafe.AsPointer(&rec)
					recordKey := makePrimaryKeyRecordKey(recPtr, scyllaTable)
					recordsUniqueMap[recordKey] = rec
				}
			}
			for _, rec := range recordsUniqueMap {
				(*recordsGetted) = append((*recordsGetted), rec)
			}
		} else {
			for _, records := range recordsMap {
				(*recordsGetted) = append((*recordsGetted), *records...)
			}
		}
	}

	if err := assignCacheVersionsAfterSelect(recordsGetted, scyllaTable); err != nil {
		return err
	}

	return nil
}
