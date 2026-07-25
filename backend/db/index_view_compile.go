package db

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"
)

// indexPartitionColumnName returns the name of the partition override declared on an
// index, or "" when the index keeps the base table partition.
func indexPartitionColumnName(indexCfg Index) string {
	if indexCfg.Partition == nil {
		return ""
	}
	return indexCfg.Partition.GetInfo().Name
}

// resolveIndexPartitionColumn resolves the Partition override against the base table.
// Returns nil when no override is declared, meaning the view keeps the base partition.
func resolveIndexPartitionColumn(dbTable *ScyllaTable, viewCfg Index) IColInfo {
	partitionColumnName := indexPartitionColumnName(viewCfg)
	if partitionColumnName == "" {
		return nil
	}
	column := dbTable.columnsMap[partitionColumnName]
	if column == nil || column.IsNil() {
		panic(fmt.Sprintf(`Table "%v": Partition column "%v" was not found`, dbTable.name, partitionColumnName))
	}
	if column.GetInfo().IsVirtual || column.GetType().IsComplexType || column.GetType().IsSlice {
		panic(fmt.Sprintf(`Table "%v": Partition column "%v" cannot be virtual, a slice or a struct`, dbTable.name, column.GetName()))
	}
	return column
}

func compileSchemaViewTable(dbTable *ScyllaTable, viewCfg Index) {
	if indexPartitionColumnName(viewCfg) != "" {
		panic(fmt.Sprintf(`Table "%v": ViewTables always keep the base partition; remove Partition`, dbTable.name))
	}
	if viewCfg.UseHash {
		panic(fmt.Sprintf(`Table "%v": ViewTables does not support UseHash`, dbTable.name))
	}
	if len(viewCfg.Keys) == 0 {
		panic(fmt.Sprintf(`Table "%v": ViewTables entry must declare at least one key column`, dbTable.name))
	}
	if len(dbTable.keys) != 1 {
		panic(fmt.Sprintf(`Table "%v": ViewTables currently requires exactly one base key column for ID maintenance`, dbTable.name))
	}

	partKey := dbTable.GetPartKey()
	if partKey == nil || partKey.IsNil() {
		panic(fmt.Sprintf(`Table "%v": ViewTables requires a partition column`, dbTable.name))
	}

	declaredColumns := []IColInfo{}
	keyColumnNames := []string{}
	physicalColumns := []viewTableColumnInfo{
		makeViewTableColumn(partKey, false),
	}
	physicalKeyColumns := []viewTableColumnInfo{}
	rebuildColumnNames := map[string]bool{}
	fanoutColumnName := ""
	sliceKeyCount := 0

	for _, declaredColumn := range viewCfg.Keys {
		column := dbTable.columnsMap[declaredColumn.GetInfo().Name]
		if column == nil || column.IsNil() {
			panic(fmt.Sprintf(`Table "%v": ViewTables column "%v" was not found`, dbTable.name, declaredColumn.GetInfo().Name))
		}
		if column.GetType().IsComplexType {
			panic(fmt.Sprintf(`Table "%v": ViewTables column "%v" cannot be a complex type`, dbTable.name, column.GetName()))
		}
		if column.GetInfo().Name == dbTable.keys[0].GetName() {
			panic(fmt.Sprintf(`Table "%v": ViewTables key "%v" must not repeat the base ID column`, dbTable.name, column.GetName()))
		}

		useSliceElement := column.GetType().IsSlice
		if useSliceElement {
			sliceKeyCount++
			fanoutColumnName = column.GetName()
		}

		keyColumnNames = append(keyColumnNames, column.GetName())
		declaredColumns = append(declaredColumns, column)
		rebuildColumnNames[column.GetName()] = true

		physicalColumn := makeViewTableColumn(column, useSliceElement)
		physicalColumns = appendUniqueViewTableColumn(physicalColumns, physicalColumn)
		physicalKeyColumns = append(physicalKeyColumns, physicalColumn)
	}

	if sliceKeyCount > 1 {
		panic(fmt.Sprintf(`Table "%v": ViewTables currently supports only one slice-backed key column`, dbTable.name))
	}

	idColumn := dbTable.keys[0]
	physicalColumns = appendUniqueViewTableColumn(physicalColumns, makeViewTableColumn(idColumn, false))

	projectedColumns := []IColInfo{}
	if len(viewCfg.Cols) == 0 {
		for _, baseColumn := range dbTable.columnsMap {
			if baseColumn.GetInfo().IsVirtual {
				continue
			}
			if baseColumn.GetName() == fanoutColumnName {
				continue
			}
			projectedColumns = append(projectedColumns, baseColumn)
		}
	} else {
		for _, declaredProjectedColumn := range viewCfg.Cols {
			projectedColumn := dbTable.columnsMap[declaredProjectedColumn.GetInfo().Name]
			if projectedColumn == nil || projectedColumn.IsNil() {
				panic(fmt.Sprintf(`Table "%v": ViewTables projected column "%v" wasn't found`, dbTable.name, declaredProjectedColumn.GetInfo().Name))
			}
			if projectedColumn.GetInfo().IsVirtual {
				panic(fmt.Sprintf(`Table "%v": ViewTables projected column "%v" cannot be virtual`, dbTable.name, projectedColumn.GetName()))
			}
			if projectedColumn.GetName() == fanoutColumnName {
				continue
			}
			projectedColumns = append(projectedColumns, projectedColumn)
		}
	}

	for _, projectedColumn := range projectedColumns {
		physicalColumns = appendUniqueViewTableColumn(physicalColumns, makeViewTableColumn(projectedColumn, false))
		rebuildColumnNames[projectedColumn.GetName()] = true
	}

	viewColumns := append([]string{partKey.GetName()}, keyColumnNames...)
	viewName := fmt.Sprintf(`%v__%v_view`, dbTable.name, strings.Join(keyColumnNames, "_"))
	view := &viewInfo{
		Type:                9,
		name:                viewName,
		columns:             viewColumns,
		columnsNoPart:       append([]string{}, keyColumnNames...),
		column:              declaredColumns[0],
		availableColumns:    []string{},
		Operators:           []string{"=", "IN", "CONTAINS"},
		fanoutColumnName:    fanoutColumnName,
		tableColumns:        physicalColumns,
		tableKeyColumns:     physicalKeyColumns,
		maintenanceIDColumn: idColumn,
		rebuildColumnNames:  rebuildColumnNames,
	}

	selectableColumnNames := map[string]bool{}
	selectableColumnNames[partKey.GetName()] = true
	selectableColumnNames[idColumn.GetName()] = true
	for _, declaredColumn := range declaredColumns {
		if declaredColumn.GetName() == fanoutColumnName {
			continue
		}
		selectableColumnNames[declaredColumn.GetName()] = true
	}
	for _, projectedColumn := range projectedColumns {
		if projectedColumn.GetName() == fanoutColumnName {
			continue
		}
		selectableColumnNames[projectedColumn.GetName()] = true
	}
	for selectableColumnName := range selectableColumnNames {
		view.availableColumns = append(view.availableColumns, selectableColumnName)
	}
	slices.Sort(view.availableColumns)

	viewPtr := view
	view.getStatementPrepared = func(statements ...ColumnStatement) []boundWhereClause {
		whereClauses := []boundWhereClause{}
		for _, statement := range statements {
			if len(statement.From) > 0 {
				for idx := range statement.From {
					whereClauses = append(whereClauses,
						boundWhereClause{
							Clause: fmt.Sprintf("%v >= ?", statement.From[idx].Col),
							Values: []any{statement.From[idx].Value},
						},
						boundWhereClause{
							Clause: fmt.Sprintf("%v <= ?", statement.To[idx].Col),
							Values: []any{statement.To[idx].Value},
						},
					)
				}
				continue
			}

			operator := statement.Operator
			if viewPtr.fanoutColumnName == statement.Col && operator == "CONTAINS" {
				operator = "="
			}
			if operator == "IN" {
				placeholders := make([]string, 0, len(statement.Values))
				queryValues := make([]any, 0, len(statement.Values))
				for _, value := range statement.Values {
					placeholders = append(placeholders, "?")
					queryValues = append(queryValues, value)
				}
				whereClauses = append(whereClauses, boundWhereClause{
					Clause: fmt.Sprintf("%v IN (%v)", statement.Col, strings.Join(placeholders, ", ")),
					Values: queryValues,
				})
				continue
			}
			whereClauses = append(whereClauses, boundWhereClause{
				Clause: fmt.Sprintf("%v %v ?", statement.Col, operator),
				Values: []any{statement.Value},
			})
		}

		combinedClause := boundWhereClause{}
		for _, whereClause := range whereClauses {
			if combinedClause.Clause != "" {
				combinedClause.Clause += " AND "
			}
			combinedClause.Clause += whereClause.Clause
			combinedClause.Values = append(combinedClause.Values, whereClause.Values...)
		}
		return []boundWhereClause{combinedClause}
	}
	view.getCreateScript = func() string {
		columnDefinitions := make([]string, 0, len(viewPtr.tableColumns))
		for _, column := range viewPtr.tableColumns {
			columnDefinitions = append(columnDefinitions, fmt.Sprintf("%v %v",
				getViewTableColumnName(column),
				getViewTableColumnType(column.SourceColumn, column.UsesSliceElement).ColType,
			))
		}

		primaryKeyColumns := append([]string{}, keyColumnNames...)
		primaryKeyColumns = append(primaryKeyColumns, idColumn.GetName())
		return fmt.Sprintf(`CREATE TABLE %v.%v (
			%v,
			PRIMARY KEY ((%v), %v)
		)
		%v;`,
			dbTable.keyspace,
			viewPtr.name,
			strings.Join(columnDefinitions, ", "),
			partKey.GetName(),
			strings.Join(primaryKeyColumns, ", "),
			makeStatementWith,
		)
	}

	dbTable.views[view.name] = view
}

func compileSchemaView(dbTable *ScyllaTable, viewCfg Index) {
	appendUniqueColumn := func(target []IColInfo, column IColInfo) []IColInfo {
		if column == nil || column.IsNil() {
			return target
		}
		for _, existingColumn := range target {
			if existingColumn.GetName() == column.GetName() {
				return target
			}
		}
		return append(target, column)
	}
	orderColumnsBySchemaIndex := func(columns []IColInfo) []IColInfo {
		orderedColumns := slices.Clone(columns)
		slices.SortFunc(orderedColumns, func(leftColumn, rightColumn IColInfo) int {
			if idxDiff := int(leftColumn.GetInfo().Idx - rightColumn.GetInfo().Idx); idxDiff != 0 {
				return idxDiff
			}
			return strings.Compare(leftColumn.GetName(), rightColumn.GetName())
		})
		return orderedColumns
	}

	colNames := []string{}
	declaredColumns := []IColInfo{}
	columns := []IColInfo{}
	viewColumnsConfig := make([]columnInfo, 0, len(viewCfg.Keys))
	packedViewHintFound := false
	for _, declaredColumn := range viewCfg.Keys {
		columnConfig := declaredColumn.GetInfo()
		viewColumnsConfig = append(viewColumnsConfig, columnConfig)
		if columnConfig.decimalSize > 0 || columnConfig.useInt32Packing {
			packedViewHintFound = true
		}
	}

	isRangeView := len(viewCfg.Keys) > 1 && packedViewHintFound

	// Views keep the base table partition unless the schema declares another column.
	basePartCol := dbTable.GetPartKey()
	if basePartCol != nil && basePartCol.IsNil() {
		basePartCol = nil
	}
	viewPartCol := resolveIndexPartitionColumn(dbTable, viewCfg)
	keepsBasePart := viewPartCol == nil ||
		(basePartCol != nil && viewPartCol.GetName() == basePartCol.GetName())
	if viewPartCol == nil {
		viewPartCol = basePartCol
	}

	for _, colInfo := range viewCfg.Keys {
		column := dbTable.columnsMap[colInfo.GetInfo().Name]
		if column.GetType().IsComplexType {
			panic("No puede usar un struct como columna de una view.")
		}
		colNames = append(colNames, column.GetName())
		declaredColumns = append(declaredColumns, column)
		columns = append(columns, column)
	}

	colNamesNoPart := colNames
	declaredColumnCount := len(declaredColumns)
	isSingleDeclaredSimpleView := declaredColumnCount == 1 && !isRangeView

	colNamesJoined := strings.Join(colNames, "_")
	if viewPartCol != nil {
		if keepsBasePart {
			colNames = append([]string{viewPartCol.GetName()}, colNames...)
			colNamesJoined = "pk_" + colNamesJoined
		} else {
			// The override leads the view key; it must not be repeated as a clustering column.
			partColName := viewPartCol.GetName()
			if isRangeView && slices.Contains(colNames, partColName) {
				panic(fmt.Sprintf(`Table "%v": the Partition column "%v" cannot also be a packed view key`,
					dbTable.name, partColName))
			}
			clusteringColNames := make([]string, 0, len(colNames))
			for _, colName := range colNames {
				if colName != partColName {
					clusteringColNames = append(clusteringColNames, colName)
				}
			}
			colNames = append([]string{partColName}, clusteringColNames...)
			colNamesJoined = strings.Join(colNames, "_")
		}
	}
	if isRangeView {
		colNamesJoined = colNamesJoined + "_rng"
	}

	view := &viewInfo{
		Type:          6,
		name:          fmt.Sprintf(`%v__%v_view`, dbTable.name, colNamesJoined),
		columns:       colNames,
		columnsNoPart: colNamesNoPart,
	}

	if len(columns) > 1 {
		view.column = &columnInfo{
			colInfo: colInfo{
				IsVirtual: true,
				Idx:       dbTable._maxColIdx,
			},
			colType: colType{
				FieldType: "int32", ColType: "int",
			},
		}
		view.column.GetInfo().Name = fmt.Sprintf(`zz_%v`, colNamesJoined)
		dbTable._maxColIdx++
		dbTable.columnsMap[view.column.GetName()] = view.column
	}

	if isSingleDeclaredSimpleView {
		view.column = declaredColumns[0]
		view.getStatementPrepared = func(statements ...ColumnStatement) []boundWhereClause {
			// Simple MVs keep their source columns, so predicates bind without key rewriting.
			sourceClauses := buildRemainingWhereClauses(statements)
			combinedClause := boundWhereClause{}
			for _, sourceClause := range sourceClauses {
				if combinedClause.Clause != "" {
					combinedClause.Clause += " AND "
				}
				combinedClause.Clause += sourceClause.Clause
				combinedClause.Values = append(combinedClause.Values, sourceClause.Values...)
			}
			return []boundWhereClause{combinedClause}
		}
	} else if len(columns) == 1 {
		view.column = columns[0]
	} else if isRangeView {
		view.Type = 8
		view.column.GetType().FieldType = "int64"
		view.column.GetType().ColType = "bigint"

		if len(columns) < 2 {
			panic(fmt.Sprintf(`The view "%v" in "%v" requires at least 2 columns for DecimalSize() packed range views`, view.name, dbTable.name))
		}
		if viewColumnsConfig[0].decimalSize > 0 {
			panic(fmt.Sprintf(`The view "%v" in "%v" cannot set DecimalSize() on the first column; it is inferred from the remaining columns`, view.name, dbTable.name))
		}

		isInt32PackedView := viewColumnsConfig[0].useInt32Packing
		if isInt32PackedView {
			view.column.GetType().FieldType = "int32"
			view.column.GetType().ColType = "int"
		}

		radixSlotsByColumn := make([]int8, 0, len(viewColumnsConfig)-1)
		for columnIndex := 1; columnIndex < len(viewColumnsConfig); columnIndex++ {
			decimalSize := viewColumnsConfig[columnIndex].decimalSize
			if decimalSize <= 0 {
				panic(fmt.Sprintf(`The view "%v" in "%v" must set DecimalSize() on column "%v" (only the first column can be inferred)`,
					view.name, dbTable.name, columns[columnIndex].GetName()))
			}
			radixSlotsByColumn = append(radixSlotsByColumn, decimalSize)
		}

		radixes := append(radixSlotsByColumn, 0)
		slices.Reverse(radixes)
		sum := int8(0)
		for i, v := range radixes {
			radixes[i] = v + sum
			sum += v
		}
		slices.Reverse(radixes)
		if radixes[0] > 17 {
			panic(fmt.Sprintf(`For view "%v" in "%v" the max radix must not be greater than 17.`, view.name, dbTable.name))
		}

		totalDigitsForPackedView := int64(19)
		if isInt32PackedView {
			totalDigitsForPackedView = 9
		}
		slotDigitsPerColumn := make([]int64, 0, len(viewColumnsConfig))
		sumTrailingDigits := int64(0)
		for _, decimalSize := range radixSlotsByColumn {
			sumTrailingDigits += int64(decimalSize)
		}
		slotDigitsPerColumn = append(slotDigitsPerColumn, totalDigitsForPackedView-sumTrailingDigits)
		for _, decimalSize := range radixSlotsByColumn {
			slotDigitsPerColumn = append(slotDigitsPerColumn, int64(decimalSize))
		}
		view.packedSourceColumns = append([]IColInfo{}, columns...)
		view.packedSlotDigitsPerColumn = append([]int64{}, slotDigitsPerColumn...)

		supportedTypes := []string{"int8", "int16", "int32", "int64", "int"}
		for _, col := range columns {
			if col.GetType().IsSlice || !slices.Contains(supportedTypes, col.GetType().FieldType) {
				panic(fmt.Sprintf(`For view "%v" in "%v" need the column %v need to be a int type for the radix value be computed.`,
					view.name, dbTable.name, col.GetName()))
			}
		}

		makeValue := func(values []int64) int64 {
			return computePackedInt64ValueNonNegative(values, slotDigitsPerColumn)
		}

		slotDigitsCopy := append([]int64{}, slotDigitsPerColumn...)
		viewColsCopy := append([]IColInfo{}, columns...)
		view.decomposeVirtualValue = func(rawValue any) []any {
			packedValues := decomposePackedInt64ValueNonNegative(convertToInt64(rawValue), slotDigitsCopy)
			values := make([]any, 0, len(viewColsCopy))
			for _, packedValue := range packedValues {
				values = append(values, packedValue)
			}
			return values
		}

		viewCols := columns
		useInt32Output := isInt32PackedView
		view.column.(*columnInfo).getValue = func(ptr unsafe.Pointer) any {
			values := []int64{}
			for _, col := range viewCols {
				values = append(values, convertToInt64(col.GetValue(ptr)))
			}
			sumValue := makeValue(values)
			if useInt32Output {
				return any(int32(sumValue))
			}
			return any(sumValue)
		}

		viewPtr := view
		viewPartColPtr := viewPartCol
		view.getStatementPrepared = func(statements ...ColumnStatement) []boundWhereClause {
			statementsMap := map[string]statementRangeGroup{}
			useBeetween := false

			for i := range statements {
				st := &statements[i]
				if st.Operator == "BETWEEN" {
					useBeetween = true
					for i := range st.From {
						statementsMap[st.From[i].Col] = statementRangeGroup{
							from:      &st.From[i],
							betweenTo: &st.To[i],
						}
					}
				} else {
					statementsMap[st.Col] = statementRangeGroup{from: st}
				}
			}

			var partStatement *ColumnStatement
			if viewPartColPtr != nil {
				partStatement = statementsMap[viewPartColPtr.GetName()].from
			}

			getValuesGroups := func() ([][]int64, []IColInfo) {
				valuesGroups := [][]int64{}
				rangeColumns := []IColInfo{}
				for _, col := range viewCols {
					stRange, ok := statementsMap[col.GetName()]
					if !ok || stRange.from == nil {
						stRange = statementRangeGroup{from: &ColumnStatement{Value: int64(0)}}
					}
					st := stRange.from
					if viewPartColPtr != nil && st.Col == viewPartColPtr.GetName() {
						continue
					}
					if len(rangeColumns) > 0 || slices.Contains(rangeOperators, st.Operator) {
						rangeColumns = append(rangeColumns, col)
						continue
					}

					valuesToAdd := []int64{}
					if len(st.Values) > 0 {
						for _, value := range st.Values {
							valuesToAdd = append(valuesToAdd, convertToInt64(value))
						}
					} else {
						valuesToAdd = append(valuesToAdd, convertToInt64(st.Value))
					}

					if len(valuesGroups) > 0 {
						valuesGroupsCurrent := valuesGroups
						valuesGroups = [][]int64{}
						for _, vg := range valuesGroupsCurrent {
							for _, value := range valuesToAdd {
								valuesGroups = append(valuesGroups, append(append([]int64{}, vg...), value))
							}
						}
					} else {
						for _, value := range valuesToAdd {
							valuesGroups = append(valuesGroups, []int64{value})
						}
					}
				}
				return valuesGroups, rangeColumns
			}

			whereStatements := []boundWhereClause{}
			if useBeetween {
				valuesFrom, valuesTo := []int64{}, []int64{}
				for _, col := range viewCols {
					srg := statementsMap[col.GetName()]
					valuesFrom = append(valuesFrom, convertToInt64(srg.from.Value))
					if srg.betweenTo != nil {
						valuesTo = append(valuesTo, convertToInt64(srg.betweenTo.Value))
					} else {
						valuesTo = append(valuesTo, convertToInt64(srg.from.Value))
					}
				}
				whereStatement := boundWhereClause{
					Clause: fmt.Sprintf("%v >= ? AND %v < ?", viewPtr.column.GetName(), viewPtr.column.GetName()),
					Values: []any{makeValue(valuesFrom), makeValue(valuesTo) + 1},
				}
				if partStatement != nil {
					whereStatement = boundWhereClause{
						Clause: fmt.Sprintf("%v = ? AND %v", viewPartColPtr.GetName(), whereStatement.Clause),
						Values: append([]any{convertToInt64(partStatement.Value)}, whereStatement.Values...),
					}
				}
				return []boundWhereClause{whereStatement}
			} else if len(statements) > 0 && slices.Contains(rangeOperators, statements[len(statements)-1].Operator) {
				valuesGroups, rangeColumns := getValuesGroups()
				for _, prefixValues := range valuesGroups {
					valuesFrom := slices.Clone(prefixValues)
					prefixFloorValues := slices.Clone(prefixValues)

					for _, col := range rangeColumns {
						st := statementsMap[col.GetName()]
						valuesFrom = append(valuesFrom, convertToInt64(st.from.Value))
						prefixFloorValues = append(prefixFloorValues, 0)
					}

					upperBound := makeValue(prefixFloorValues) + Pow10Int64(sumSlotDigits(slotDigitsPerColumn, len(prefixValues)))
					whereStatements = append(whereStatements, boundWhereClause{
						Clause: fmt.Sprintf("%v >= ? AND %v < ?", viewPtr.column.GetName(), viewPtr.column.GetName()),
						Values: []any{makeValue(valuesFrom), upperBound},
					})
				}
			} else {
				valuesGroups, _ := getValuesGroups()
				hashValues := make([]any, 0, len(valuesGroups))
				placeholders := make([]string, 0, len(valuesGroups))
				for _, values := range valuesGroups {
					hashValues = append(hashValues, makeValue(values))
					placeholders = append(placeholders, "?")
				}
				whereStatements = append(whereStatements, boundWhereClause{
					Clause: fmt.Sprintf("%v IN (%v)", viewPtr.column.GetName(), strings.Join(placeholders, ", ")),
					Values: hashValues,
				})
			}

			if partStatement != nil {
				for i, ws := range whereStatements {
					whereStatements[i] = boundWhereClause{
						Clause: fmt.Sprintf("%v = ? AND %v", viewPartColPtr.GetName(), ws.Clause),
						Values: append([]any{convertToInt64(partStatement.Value)}, ws.Values...),
					}
				}
			}
			return whereStatements
		}
	} else {
		viewPtr := view
		viewPartColPtr := viewPartCol
		viewCols := columns
		view.Operators = []string{"=", "IN"}
		view.Type = 7
		view.column.(*columnInfo).getValue = func(ptr unsafe.Pointer) any {
			values := []any{}
			for _, e := range viewCols {
				values = append(values, e.GetValue(ptr))
			}
			return HashInt(values...)
		}

		view.getStatementPrepared = func(statements ...ColumnStatement) []boundWhereClause {
			valuesGroups := [][]any{{}}
			for _, e := range viewCols {
				for _, st := range statements {
					if st.Col == e.GetName() {
						if len(st.Values) >= 2 {
							valuesGroupsCurrent := valuesGroups
							valuesGroups = [][]any{}
							for _, vg := range valuesGroupsCurrent {
								for _, value := range st.Values {
									valuesGroups = append(valuesGroups, append(vg, value))
								}
							}
						} else {
							if len(st.Values) == 1 {
								st.Value = st.Values[0]
							}
							for i := range valuesGroups {
								valuesGroups[i] = append(valuesGroups[i], st.Value)
							}
						}
						break
					}
				}
			}

			hashValues := make([]any, 0, len(valuesGroups))
			for _, values := range valuesGroups {
				hashValues = append(hashValues, HashInt(values...))
			}

			statement := boundWhereClause{}
			if len(hashValues) == 1 {
				statement = boundWhereClause{
					Clause: fmt.Sprintf("%v = ?", viewPtr.column.GetName()),
					Values: []any{hashValues[0]},
				}
			} else {
				placeholders := make([]string, 0, len(hashValues))
				for range hashValues {
					placeholders = append(placeholders, "?")
				}
				statement = boundWhereClause{
					Clause: fmt.Sprintf("%v IN (%v)", viewPtr.column.GetName(), strings.Join(placeholders, ", ")),
					Values: hashValues,
				}
			}

			if viewPartColPtr != nil {
				for _, st := range statements {
					if st.Col == viewPartColPtr.GetName() {
						statement = boundWhereClause{
							Clause: fmt.Sprintf("%v = ? AND %v", st.Col, statement.Clause),
							Values: append([]any{st.Value}, statement.Values...),
						}
						break
					}
				}
			}
			return []boundWhereClause{statement}
		}
	}

	projectedColumnsConfig := viewCfg.Cols
	projectedColumns := []IColInfo{}
	for _, declaredProjectedColumn := range projectedColumnsConfig {
		projectedColumn := dbTable.columnsMap[declaredProjectedColumn.GetInfo().Name]
		if projectedColumn == nil || projectedColumn.IsNil() {
			panic(fmt.Sprintf(`The projected column "%v" for view "%v" in "%v" wasn't found.`,
				declaredProjectedColumn.GetInfo().Name, view.name, dbTable.name))
		}
		if projectedColumn.GetInfo().IsVirtual {
			panic(fmt.Sprintf(`The projected column "%v" for view "%v" in "%v" cannot be virtual.`,
				projectedColumn.GetName(), view.name, dbTable.name))
		}
		projectedColumns = appendUniqueColumn(projectedColumns, projectedColumn)
	}

	selectableColumns := []IColInfo{}
	if len(projectedColumns) == 0 {
		for _, baseColumn := range dbTable.columnsMap {
			if baseColumn.GetInfo().IsVirtual {
				continue
			}
			selectableColumns = appendUniqueColumn(selectableColumns, baseColumn)
		}
	} else {
		selectableColumns = appendUniqueColumn(selectableColumns, basePartCol)
		// A relocated partition column must always travel with the projected view.
		selectableColumns = appendUniqueColumn(selectableColumns, viewPartCol)
		if view.Type == 6 {
			for _, declaredViewColumn := range declaredColumns {
				selectableColumns = appendUniqueColumn(selectableColumns, declaredViewColumn)
			}
		}
		for _, keyColumn := range dbTable.keys {
			selectableColumns = appendUniqueColumn(selectableColumns, keyColumn)
		}
		if view.column != nil && !view.column.IsNil() && !view.column.GetInfo().IsVirtual {
			selectableColumns = appendUniqueColumn(selectableColumns, view.column)
		}
		for _, projectedColumn := range projectedColumns {
			selectableColumns = appendUniqueColumn(selectableColumns, projectedColumn)
		}
	}
	for _, selectableColumn := range selectableColumns {
		view.availableColumns = append(view.availableColumns, selectableColumn.GetName())
	}

	viewPtr := view
	view.getCreateScript = func() string {
		whereCols := []IColInfo{}
		if viewPtr.Type == 6 && !viewPtr.column.GetInfo().IsVirtual {
			for _, declaredViewColumn := range declaredColumns {
				whereCols = appendUniqueColumn(whereCols, declaredViewColumn)
			}
		} else {
			whereCols = appendUniqueColumn(whereCols, viewPtr.column)
		}
		wherePartCol := viewPartCol
		if !keepsBasePart {
			// The base partition becomes a clustering column of the relocated view.
			whereCols = appendUniqueColumn(whereCols, basePartCol)
		}
		for _, keyColumn := range dbTable.keys {
			whereCols = appendUniqueColumn(whereCols, keyColumn)
		}
		if wherePartCol != nil {
			whereCols = slices.DeleteFunc(whereCols, func(column IColInfo) bool {
				return column.GetName() == wherePartCol.GetName()
			})
		}

		keyNames := []string{}
		for _, col := range whereCols {
			keyNames = append(keyNames, col.GetName())
		}

		primaryKey := strings.Join(keyNames, ",")
		if wherePartCol != nil {
			primaryKey = fmt.Sprintf("(%v), %v", wherePartCol.GetName(), primaryKey)
		}

		whereColumnsNotNull := []string{}
		if wherePartCol != nil {
			if wherePartCol.GetType().ColType == "text" {
				whereColumnsNotNull = append(whereColumnsNotNull, wherePartCol.GetName()+" IS NOT NULL")
			} else {
				// whereColumnsNotNull = append(whereColumnsNotNull, wherePartCol.GetName()+" > 0")
				whereColumnsNotNull = append(whereColumnsNotNull, wherePartCol.GetName()+" IS NOT NULL")
			}
		}
		for _, col := range whereCols {
			if col.GetType().ColType == "text" {
				whereColumnsNotNull = append(whereColumnsNotNull, col.GetName()+" IS NOT NULL")
			} else {
				// whereColumnsNotNull = append(whereColumnsNotNull, col.GetName()+" > 0")
				whereColumnsNotNull = append(whereColumnsNotNull, col.GetName()+" IS NOT NULL")
			}
		}

		selectClause := "*"
		if len(projectedColumns) > 0 {
			projectedColumnNames := make([]string, 0, len(projectedColumns))
			for _, projectedColumn := range projectedColumns {
				projectedColumnNames = append(projectedColumnNames, projectedColumn.GetName())
			}
			selectClause = strings.Join(projectedColumnNames, ", ")
		} else {
			selectColumns := slices.Clone(selectableColumns)
			for _, whereColumn := range whereCols {
				if whereColumn != nil && !whereColumn.IsNil() && whereColumn.GetInfo().IsVirtual {
					selectColumns = appendUniqueColumn(selectColumns, whereColumn)
				}
			}
			selectColumns = orderColumnsBySchemaIndex(selectColumns)

			selectColumnNames := make([]string, 0, len(selectColumns))
			for _, selectColumn := range selectColumns {
				selectColumnNames = append(selectColumnNames, selectColumn.GetName())
			}
			selectClause = strings.Join(selectColumnNames, ", ")
		}

		return fmt.Sprintf(`CREATE MATERIALIZED VIEW %v.%v AS
			SELECT %v FROM %v
			WHERE %v
			PRIMARY KEY (%v)
			%v;`,
			dbTable.keyspace, viewPtr.name, selectClause, dbTable.GetFullName(),
			strings.Join(whereColumnsNotNull, " AND "), primaryKey, makeStatementWith)
	}

	dbTable.views[view.name] = view
}
