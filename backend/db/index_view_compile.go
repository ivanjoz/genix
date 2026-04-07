package db

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"
)

func compileSchemaViewTable(dbTable *ScyllaTable[any], viewCfg Index) {
	if !viewCfg.KeepPart {
		panic(fmt.Sprintf(`Table "%v": ViewTables requires KeepPart = true to preserve the base partition`, dbTable.name))
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
	view.getStatement = func(statements ...ColumnStatement) []string {
		whereClauses := []string{}
		for _, statement := range statements {
			if len(statement.From) > 0 {
				for idx := range statement.From {
					whereClauses = append(whereClauses, fmt.Sprintf("%v >= %v", statement.From[idx].Col, statement.From[idx].GetValue()))
					whereClauses = append(whereClauses, fmt.Sprintf("%v <= %v", statement.To[idx].Col, statement.To[idx].GetValue()))
				}
				continue
			}

			operator := statement.Operator
			if viewPtr.fanoutColumnName == statement.Col && operator == "CONTAINS" {
				operator = "="
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%v %v %v", statement.Col, operator, statement.GetValue()))
		}
		return []string{strings.Join(whereClauses, " AND ")}
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

func compileSchemaView(dbTable *ScyllaTable[any], viewCfg Index) {
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
	if isRangeView {
		viewCfg.KeepPart = true
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
	pk := dbTable.GetPartKey()
	if pk != nil && !pk.IsNil() {
		if viewCfg.KeepPart {
			colNames = append([]string{pk.GetName()}, colNames...)
			colNamesJoined = "pk_" + colNamesJoined
		} else if !isSingleDeclaredSimpleView {
			colNames = append([]string{pk.GetName()}, colNames...)
			colNamesJoined = pk.GetName() + "_" + colNamesJoined
			columns = append([]IColInfo{pk}, columns...)
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
		if !viewCfg.KeepPart {
			view.columns = colNamesNoPart
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
		viewCfgPtr := &viewCfg
		view.getStatement = func(statements ...ColumnStatement) []string {
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
			if viewCfgPtr.KeepPart {
				pk := dbTable.GetPartKey()
				if pk != nil && !pk.IsNil() {
					partStatement = statementsMap[pk.GetName()].from
				}
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
					if st.Col == dbTable.GetPartKey().GetName() && viewCfgPtr.KeepPart {
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

			whereStatements := []string{}
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
				whereStatement := fmt.Sprintf("%v >= %v AND %v < %v",
					viewPtr.column.GetName(), makeValue(valuesFrom),
					viewPtr.column.GetName(), makeValue(valuesTo)+1,
				)
				if partStatement != nil {
					pk := dbTable.GetPartKey()
					if pk != nil && !pk.IsNil() {
						whereStatement = fmt.Sprintf("%v = %v AND %v",
							pk.GetName(), convertToInt64(partStatement.Value), whereStatement)
					}
				}
				return []string{whereStatement}
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
					whereStatements = append(whereStatements, fmt.Sprintf("%v >= %v AND %v < %v",
						viewPtr.column.GetName(), makeValue(valuesFrom),
						viewPtr.column.GetName(), upperBound,
					))
				}
			} else {
				valuesGroups, _ := getValuesGroups()
				hashValues := []int64{}
				for _, values := range valuesGroups {
					hashValues = append(hashValues, makeValue(values))
				}
				whereStatements = append(whereStatements,
					fmt.Sprintf("%v IN (%v)", viewPtr.column.GetName(), Concatx(", ", hashValues)),
				)
			}

			if partStatement != nil {
				pk := dbTable.GetPartKey()
				if pk != nil && !pk.IsNil() {
					for i, ws := range whereStatements {
						whereStatements[i] = fmt.Sprintf("%v = %v AND %v",
							pk.GetName(), convertToInt64(partStatement.Value), ws)
					}
				}
			}
			return whereStatements
		}
	} else {
		viewPtr := view
		viewCfgPtr := &viewCfg
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

		view.getStatement = func(statements ...ColumnStatement) []string {
			valuesGroups := [][]any{{}}
			statement := ""
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

			hashValues := []string{}
			for _, values := range valuesGroups {
				hashValues = append(hashValues, fmt.Sprintf("%v", HashInt(values...)))
			}

			if len(hashValues) == 1 {
				statement = fmt.Sprintf("%v = %v", viewPtr.column.GetName(), hashValues[0])
			} else {
				values := strings.Join(hashValues, ", ")
				statement = fmt.Sprintf("%v IN (%v)", viewPtr.column.GetName(), values)
			}

			if viewCfgPtr.KeepPart {
				pk := dbTable.GetPartKey()
				if pk != nil && !pk.IsNil() {
					for _, st := range statements {
						if st.Col == pk.GetName() {
							statement = fmt.Sprintf("%v = %v AND ", st.Col, st.Value) + statement
						}
					}
				}
			}
			return []string{statement}
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
		partKeyColumn := dbTable.GetPartKey()
		if partKeyColumn != nil && !partKeyColumn.IsNil() {
			selectableColumns = appendUniqueColumn(selectableColumns, partKeyColumn)
		}
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
		var wherePartCol IColInfo

		pk_ := dbTable.GetPartKey()
		if pk_ != nil && !pk_.IsNil() {
			if viewCfg.KeepPart {
				wherePartCol = pk_
			} else {
				whereCols = appendUniqueColumn(whereCols, pk_)
			}
		}
		for _, keyColumn := range dbTable.keys {
			whereCols = appendUniqueColumn(whereCols, keyColumn)
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
				whereColumnsNotNull = append(whereColumnsNotNull, wherePartCol.GetName()+" > ''")
			} else {
				whereColumnsNotNull = append(whereColumnsNotNull, wherePartCol.GetName()+" > 0")
			}
		}
		for _, col := range whereCols {
			if col.GetType().ColType == "text" {
				whereColumnsNotNull = append(whereColumnsNotNull, col.GetName()+" > ''")
			} else {
				whereColumnsNotNull = append(whereColumnsNotNull, col.GetName()+" > 0")
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
