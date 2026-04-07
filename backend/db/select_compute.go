package db

import (
	"fmt"
	"slices"
	"strings"
)

type QueryCapability struct {
	Signature string
	Source    *viewInfo
	Priority  int
	IsKey     bool // If it's the main table primary key
}

type selectScanColumn struct {
	ColumnName    string
	DecomposeView *viewInfo
}

type nativeGroupByPlan struct {
	ViewTableName     string
	SelectExpressions []string
	ScanColumns       []selectScanColumn
	GroupByColumns    []string
	WhereStatements   []boundWhereClause
	OrderColumn       IColInfo
}

func buildDefaultScanColumns(columnNames []string) []selectScanColumn {
	scanColumns := make([]selectScanColumn, 0, len(columnNames))
	for _, columnName := range columnNames {
		scanColumns = append(scanColumns, selectScanColumn{ColumnName: columnName})
	}
	return scanColumns
}

func splitGroupByColumns(columns []columnInfo) ([]columnInfo, []columnInfo) {
	groupedColumns := []columnInfo{}
	aggregateColumns := []columnInfo{}
	for _, column := range columns {
		if column.aggregateFn != "" {
			aggregateColumns = append(aggregateColumns, column)
			continue
		}
		groupedColumns = append(groupedColumns, column)
	}
	return groupedColumns, aggregateColumns
}

func canUseViewForAggregateColumns(aggregateColumns []columnInfo, view *viewInfo) bool {
	if view == nil {
		return false
	}
	for _, aggregateColumn := range aggregateColumns {
		if !slices.Contains(view.availableColumns, aggregateColumn.GetName()) {
			fmt.Printf("GroupBy aggregate view rejected: view=%s missing_column=%s available=%v\n",
				view.name, aggregateColumn.GetName(), view.availableColumns)
			return false
		}
	}
	return true
}

func makeGroupByAggregateProjection(column columnInfo) (string, error) {
	if column.aggregateFn == "" {
		return column.GetName(), nil
	}
	if column.aggregateFn == "AVG" && column.Type != 6 && column.Type != 7 {
		return "", fmt.Errorf(`Avg() requires a float32 or float64 destination column. Column: "%v"`, column.GetName())
	}
	return fmt.Sprintf("%s(%s) AS %s", column.aggregateFn, column.GetName(), column.GetName()), nil
}

func sumSlotDigits(slotDigits []int64, fromIndex int) int64 {
	totalDigits := int64(0)
	for i := fromIndex; i < len(slotDigits); i++ {
		totalDigits += slotDigits[i]
	}
	return totalDigits
}

func computePackedBound(slotDigits []int64, prefixValues []int64, rangeIndex int, rangeValue int64) int64 {
	componentValues := make([]int64, len(slotDigits))
	copy(componentValues, prefixValues)
	if rangeIndex >= 0 && rangeIndex < len(componentValues) {
		componentValues[rangeIndex] = rangeValue
	}
	return computePackedInt64ValueNonNegative(componentValues, slotDigits)
}

func buildPackedPrefixRangeClauses(view *viewInfo, statements []ColumnStatement, scyllaTable ScyllaTable[any]) ([]boundWhereClause, error) {
	if view == nil || len(view.packedSourceColumns) == 0 || len(view.packedSlotDigitsPerColumn) == 0 {
		return nil, fmt.Errorf("packed GroupBy requires a packed view")
	}

	partitionColumn := scyllaTable.GetPartKey()
	statementByColumn := map[string]ColumnStatement{}
	allowedColumns := map[string]bool{}
	if partitionColumn != nil && !partitionColumn.IsNil() {
		allowedColumns[partitionColumn.GetName()] = true
	}
	for _, sourceColumn := range view.packedSourceColumns {
		allowedColumns[sourceColumn.GetName()] = true
	}

	for _, statement := range statements {
		if !allowedColumns[statement.Col] {
			return nil, fmt.Errorf(`GroupBy packed view "%v" does not support filtering by column "%v"`, view.name, statement.Col)
		}
		if _, exists := statementByColumn[statement.Col]; exists {
			return nil, fmt.Errorf(`GroupBy packed view "%v" does not support repeated filters for column "%v"`, view.name, statement.Col)
		}
		statementByColumn[statement.Col] = statement
	}

	var partitionClause *boundWhereClause
	if partitionColumn != nil && !partitionColumn.IsNil() {
		partitionStatement, exists := statementByColumn[partitionColumn.GetName()]
		if !exists || partitionStatement.Operator != "=" {
			return nil, fmt.Errorf(`GroupBy packed view "%v" requires partition equality on "%v"`, view.name, partitionColumn.GetName())
		}
		partitionClause = &boundWhereClause{
			Clause: fmt.Sprintf("%v = ?", partitionColumn.GetName()),
			Values: []any{partitionStatement.Value},
		}
	}

	appendPartitionClause := func(clauses ...boundWhereClause) boundWhereClause {
		combined := boundWhereClause{}
		if partitionClause != nil {
			combined.Clause = partitionClause.Clause
			combined.Values = append(combined.Values, partitionClause.Values...)
		}
		for _, clause := range clauses {
			if clause.Clause == "" {
				continue
			}
			if combined.Clause != "" {
				combined.Clause += " AND "
			}
			combined.Clause += clause.Clause
			combined.Values = append(combined.Values, clause.Values...)
		}
		return combined
	}

	prefixValueGroups := [][]int64{{}}
	rangeStatementIndex := len(view.packedSourceColumns)
	var rangeStatement *ColumnStatement

	for sourceIndex, sourceColumn := range view.packedSourceColumns {
		statement, exists := statementByColumn[sourceColumn.GetName()]
		if !exists {
			rangeStatementIndex = sourceIndex
			break
		}

		switch statement.Operator {
		case "=":
			nextGroups := [][]int64{}
			for _, prefixValues := range prefixValueGroups {
				nextGroups = append(nextGroups, append(slices.Clone(prefixValues), convertToInt64(statement.Value)))
			}
			prefixValueGroups = nextGroups
		case "IN":
			if len(statement.Values) == 0 {
				return nil, fmt.Errorf(`GroupBy packed view "%v" received an empty IN for column "%v"`, view.name, sourceColumn.GetName())
			}
			nextGroups := [][]int64{}
			for _, prefixValues := range prefixValueGroups {
				for _, rawValue := range statement.Values {
					nextGroups = append(nextGroups, append(slices.Clone(prefixValues), convertToInt64(rawValue)))
				}
			}
			prefixValueGroups = nextGroups
		case "BETWEEN", ">", ">=", "<", "<=":
			rangeStatementIndex = sourceIndex
			rangeStatement = &statement
		default:
			return nil, fmt.Errorf(`GroupBy packed view "%v" does not support operator "%v" for column "%v"`, view.name, statement.Operator, sourceColumn.GetName())
		}

		if rangeStatement != nil {
			break
		}
	}

	for trailingIndex := rangeStatementIndex + 1; trailingIndex < len(view.packedSourceColumns); trailingIndex++ {
		trailingColumnName := view.packedSourceColumns[trailingIndex].GetName()
		if _, exists := statementByColumn[trailingColumnName]; exists {
			return nil, fmt.Errorf(`GroupBy packed view "%v" requires filters to follow the packed key order. Column "%v" is out of place`, view.name, trailingColumnName)
		}
	}

	packedColumnName := view.column.GetName()
	whereStatements := []boundWhereClause{}

	switch {
	case rangeStatement == nil && rangeStatementIndex == 0:
		whereStatements = append(whereStatements, appendPartitionClause())
	case rangeStatement == nil && rangeStatementIndex >= len(view.packedSourceColumns):
		for _, prefixValues := range prefixValueGroups {
			packedValue := computePackedBound(view.packedSlotDigitsPerColumn, prefixValues, -1, 0)
			whereStatements = append(whereStatements, appendPartitionClause(boundWhereClause{
				Clause: fmt.Sprintf("%v = ?", packedColumnName),
				Values: []any{packedValue},
			}))
		}
	case rangeStatement == nil:
		remainingDigits := sumSlotDigits(view.packedSlotDigitsPerColumn, rangeStatementIndex)
		for _, prefixValues := range prefixValueGroups {
			fromValue := computePackedBound(view.packedSlotDigitsPerColumn, prefixValues, -1, 0)
			toValue := fromValue + Pow10Int64(remainingDigits)
			whereStatements = append(whereStatements, appendPartitionClause(
				boundWhereClause{Clause: fmt.Sprintf("%v >= ?", packedColumnName), Values: []any{fromValue}},
				boundWhereClause{Clause: fmt.Sprintf("%v < ?", packedColumnName), Values: []any{toValue}},
			))
		}
	default:
		for _, prefixValues := range prefixValueGroups {
			switch rangeStatement.Operator {
			case "BETWEEN":
				if len(rangeStatement.From) == 0 || len(rangeStatement.To) == 0 {
					return nil, fmt.Errorf(`GroupBy packed view "%v" received an invalid BETWEEN for column "%v"`, view.name, rangeStatement.Col)
				}
				fromValue := computePackedBound(view.packedSlotDigitsPerColumn, prefixValues, rangeStatementIndex, convertToInt64(rangeStatement.From[0].Value))
				toValue := computePackedBound(view.packedSlotDigitsPerColumn, prefixValues, rangeStatementIndex, convertToInt64(rangeStatement.To[0].Value)+1)
				whereStatements = append(whereStatements, appendPartitionClause(
					boundWhereClause{Clause: fmt.Sprintf("%v >= ?", packedColumnName), Values: []any{fromValue}},
					boundWhereClause{Clause: fmt.Sprintf("%v < ?", packedColumnName), Values: []any{toValue}},
				))
			case ">":
				fromValue := computePackedBound(view.packedSlotDigitsPerColumn, prefixValues, rangeStatementIndex, convertToInt64(rangeStatement.Value)+1)
				whereStatements = append(whereStatements, appendPartitionClause(boundWhereClause{
					Clause: fmt.Sprintf("%v >= ?", packedColumnName),
					Values: []any{fromValue},
				}))
			case ">=":
				fromValue := computePackedBound(view.packedSlotDigitsPerColumn, prefixValues, rangeStatementIndex, convertToInt64(rangeStatement.Value))
				whereStatements = append(whereStatements, appendPartitionClause(boundWhereClause{
					Clause: fmt.Sprintf("%v >= ?", packedColumnName),
					Values: []any{fromValue},
				}))
			case "<":
				toValue := computePackedBound(view.packedSlotDigitsPerColumn, prefixValues, rangeStatementIndex, convertToInt64(rangeStatement.Value))
				whereStatements = append(whereStatements, appendPartitionClause(boundWhereClause{
					Clause: fmt.Sprintf("%v < ?", packedColumnName),
					Values: []any{toValue},
				}))
			case "<=":
				toValue := computePackedBound(view.packedSlotDigitsPerColumn, prefixValues, rangeStatementIndex, convertToInt64(rangeStatement.Value)+1)
				whereStatements = append(whereStatements, appendPartitionClause(boundWhereClause{
					Clause: fmt.Sprintf("%v < ?", packedColumnName),
					Values: []any{toValue},
				}))
			}
		}
	}

	if len(whereStatements) == 0 {
		return nil, fmt.Errorf(`GroupBy packed view "%v" could not build a packed where clause`, view.name)
	}

	return whereStatements, nil
}

func buildNativeGroupByPlan(
	tableInfo *TableInfo,
	statements []ColumnStatement,
	scyllaTable ScyllaTable[any],
) (*nativeGroupByPlan, error) {
	if len(tableInfo.groupByColumns) == 0 {
		return nil, nil
	}
	if len(tableInfo.columnsInclude) > 0 || len(tableInfo.columnsExclude) > 0 {
		return nil, fmt.Errorf("GroupBy cannot be combined with Select() or Exclude()")
	}

	groupedColumns, aggregateColumns := splitGroupByColumns(tableInfo.groupByColumns)
	if len(groupedColumns) == 0 {
		return nil, fmt.Errorf("GroupBy requires at least one non-aggregated column")
	}

	selectExpressions := []string{}
	scanColumns := []selectScanColumn{}
	for _, groupedColumn := range groupedColumns {
		selectExpressions = append(selectExpressions, groupedColumn.GetName())
		scanColumns = append(scanColumns, selectScanColumn{ColumnName: groupedColumn.GetName()})
	}
	for _, aggregateColumn := range aggregateColumns {
		selectExpression, err := makeGroupByAggregateProjection(aggregateColumn)
		if err != nil {
			return nil, err
		}
		selectExpressions = append(selectExpressions, selectExpression)
		scanColumns = append(scanColumns, selectScanColumn{ColumnName: aggregateColumn.GetName()})
	}

	if len(groupedColumns) == 1 {
		groupedColumn := scyllaTable.columnsMap[groupedColumns[0].GetName()]
		if groupedColumn == nil {
			return nil, fmt.Errorf(`GroupBy column "%v" was not found`, groupedColumns[0].GetName())
		}
		fmt.Printf("Native GroupBy plan selected on base table: group=%v aggregates=%d\n", groupedColumns[0].GetName(), len(aggregateColumns))
		return &nativeGroupByPlan{
			SelectExpressions: selectExpressions,
			ScanColumns:       scanColumns,
			GroupByColumns:    []string{groupedColumns[0].GetName()},
			OrderColumn:       groupedColumn,
		}, nil
	}

	groupedColumnNames := make([]string, 0, len(groupedColumns))
	for _, groupedColumn := range groupedColumns {
		groupedColumnNames = append(groupedColumnNames, groupedColumn.GetName())
	}

	for _, view := range scyllaTable.indexViews {
		if view.Type != 8 || len(view.columnsNoPart) != len(groupedColumnNames) {
			continue
		}
		if !slices.Equal(view.columnsNoPart, groupedColumnNames) {
			fmt.Printf("GroupBy packed view skipped: view=%s expected=%v got=%v\n",
				view.name, groupedColumnNames, view.columnsNoPart)
			continue
		}
		if !canUseViewForAggregateColumns(aggregateColumns, view) {
			continue
		}

		whereStatements, err := buildPackedPrefixRangeClauses(view, statements, scyllaTable)
		if err != nil {
			fmt.Printf("GroupBy packed view rejected: view=%s err=%v\n", view.name, err)
			return nil, err
		}

		selectExpressionsPacked := []string{view.column.GetName()}
		scanColumnsPacked := []selectScanColumn{{ColumnName: view.column.GetName(), DecomposeView: view}}
		for _, aggregateColumn := range aggregateColumns {
			selectExpression, err := makeGroupByAggregateProjection(aggregateColumn)
			if err != nil {
				return nil, err
			}
			selectExpressionsPacked = append(selectExpressionsPacked, selectExpression)
			scanColumnsPacked = append(scanColumnsPacked, selectScanColumn{ColumnName: aggregateColumn.GetName()})
		}

		fmt.Printf("Native GroupBy packed view selected: view=%s groups=%v aggregates=%d\n", view.name, groupedColumnNames, len(aggregateColumns))
		return &nativeGroupByPlan{
			ViewTableName:     view.name,
			SelectExpressions: selectExpressionsPacked,
			ScanColumns:       scanColumnsPacked,
			GroupByColumns:    []string{view.column.GetName()},
			WhereStatements:   whereStatements,
			OrderColumn:       view.column,
		}, nil
	}

	return nil, fmt.Errorf(`GroupBy requires a compatible packed view for columns "%v"`, strings.Join(groupedColumnNames, ", "))
}

func capabilityOpForStatement(statement ColumnStatement) string {
	// Normalize query operators into capability tokens so matching supports range and contains uniformly.
	if statement.Operator == "CONTAINS" {
		return "@"
	}
	if slices.Contains(rangeOperators, statement.Operator) || statement.Operator == "BETWEEN" {
		return "~"
	}
	return "="
}

func capabilityDefaultOpForColumn(column IColInfo) string {
	// Slice-backed columns default to CONTAINS semantics in Scylla index lookups.
	if column != nil && column.GetType().IsSlice {
		return "@"
	}
	return "="
}

// GetQuerySignature generates a signature for a set of ColumnStatements
func GetQuerySignature(statements []ColumnStatement) string {
	// Sort statements by column name to ensure consistent signatures for hashing/matching
	// but for Scylla we usually care about the order of keys.
	// Actually, the matching logic should probably be smarter than just string matching
	// because the order in WHERE doesn't matter, but the order in the index DOES matter.

	// For now, let's just collect what we have.
	type colOp struct {
		col string
		op  string
	}
	ops := []colOp{}
	for _, st := range statements {
		op := capabilityOpForStatement(st)
		ops = append(ops, colOp{st.Col, op})
	}

	// We need to match these ops against the capabilities.
	return "" // Will be used differently
}

func (dbTable *ScyllaTable[T]) ComputeCapabilities() []QueryCapability {
	caps := []QueryCapability{}

	// 1. Main Table Primary Key
	pk := dbTable.partKey
	if pk != nil && !pk.IsNil() {
		// Just partition
		caps = append(caps, QueryCapability{
			Signature: fmt.Sprintf("%v|=", pk.GetName()),
			Priority:  10,
			IsKey:     true,
		})

		// Partition + Clustering Keys
		currentSig := fmt.Sprintf("%v|=", pk.GetName())
		for i, key := range dbTable.keys {
			colName := key.GetName()
			// Equality
			caps = append(caps, QueryCapability{
				Signature: currentSig + fmt.Sprintf("|%v|=", colName),
				Priority:  20 + i*2,
				IsKey:     true,
			})
			// Range (only supported on clustering keys)
			caps = append(caps, QueryCapability{
				Signature: currentSig + fmt.Sprintf("|%v|~", colName),
				Priority:  15 + i*2,
				IsKey:     true,
			})
			currentSig += fmt.Sprintf("|%v|=", colName)
		}
	}

	// 2. Local Indexes
	for _, idx := range dbTable.indexes {
		if idx.Type == 2 { // Local Index
			pkName := dbTable.GetPartKey().GetName()
			colName := idx.column.GetName()
			// Use slice-aware operator token so generated index signatures match CONTAINS queries.
			colOp := capabilityDefaultOpForColumn(idx.column)
			// Equality
			caps = append(caps, QueryCapability{
				Signature: fmt.Sprintf("%v|=|%v|%v", pkName, colName, colOp),
				Source:    idx,
				Priority:  12,
			})
			// Range
			caps = append(caps, QueryCapability{
				Signature: fmt.Sprintf("%v|=|%v|~", pkName, colName),
				Source:    idx,
				Priority:  11,
			})
		} else if idx.Type == 1 { // Global Index
			colName := idx.column.GetName()
			// Global set indexes must advertise CONTAINS-compatible signatures.
			colOp := capabilityDefaultOpForColumn(idx.column)
			caps = append(caps, QueryCapability{
				Signature: fmt.Sprintf("%v|%v", colName, colOp),
				Source:    idx,
				Priority:  10,
			})
		}
	}

	// 2.5 Packed Indexes (local and global)
	// These are backed by:
	// - local:  CREATE INDEX ... ON table ((partition), packed_column)
	// - global: CREATE INDEX ... ON table (packed_column)
	// but capabilities must match on the source predicates (status/updated/etc).
	if len(dbTable.packedIndexes) > 0 {
		for _, packedIndex := range dbTable.packedIndexes {
			if len(packedIndex.sourceColumnNames) == 0 {
				continue
			}

			source := dbTable.indexes[packedIndex.indexName]
			if source == nil {
				continue
			}

			signatureParts := []string{}
			isLocal := packedIndex.partitionColumnName != ""
			if isLocal {
				signatureParts = append(signatureParts, packedIndex.partitionColumnName, "=")
			}

			// Single-column packed index: match equality/range directly on that source column (plus pk if local).
			if len(packedIndex.sourceColumnNames) == 1 {
				colName := packedIndex.sourceColumnNames[0]
				sigPrefix := strings.Join(signatureParts, "|")
				if sigPrefix != "" {
					sigPrefix += "|"
				}
				caps = append(caps, QueryCapability{
					Signature: sigPrefix + fmt.Sprintf("%v|=", colName),
					Source:    source,
					Priority:  14,
				})
				caps = append(caps, QueryCapability{
					Signature: sigPrefix + fmt.Sprintf("%v|~", colName),
					Source:    source,
					Priority:  13,
				})
				continue
			}

			// Composite: equality on all prefix columns + (range/equality) on last column.
			// Note: IN is normalized to "=" in capabilityOpForStatement, so it matches these signatures.
			for i, colName := range packedIndex.sourceColumnNames {
				if i < len(packedIndex.sourceColumnNames)-1 {
					signatureParts = append(signatureParts, colName, "=")
					continue
				}

				signaturePrefix := strings.Join(signatureParts, "|")
				if signaturePrefix != "" {
					signaturePrefix += "|"
				}

				priorityBase := 20
				if isLocal {
					// Local packed indexes are typically more selective when partition is provided.
					priorityBase = 26
				}

				caps = append(caps, QueryCapability{
					Signature: signaturePrefix + colName + "|=",
					Source:    source,
					Priority:  priorityBase + len(packedIndex.sourceColumnNames)*2,
				})
				// Global secondary indexes are not reliable for range scans; Scylla can demand ALLOW FILTERING.
				// We only advertise "~" for local packed indexes (partition-scoped index table supports range on packed col).
				if isLocal {
					caps = append(caps, QueryCapability{
						Signature: signaturePrefix + colName + "|~",
						Source:    source,
						Priority:  priorityBase - 2 + len(packedIndex.sourceColumnNames)*2,
					})
				}
			}
		}
	}

	// 3. Views
	for _, view := range dbTable.indexViews {
		if view.Type < 6 {
			continue
		}

		sigBase := ""
		if view.Type == 7 || view.Type == 3 { // Hash
			cols := []string{}
			for _, col := range view.columns {
				column := dbTable.columnsMap[col]
				// Hash view signatures inherit source column operator semantics.
				cols = append(cols, col+"|"+capabilityDefaultOpForColumn(column))
			}
			sigBase = strings.Join(cols, "|")
			caps = append(caps, QueryCapability{
				Signature: sigBase,
				Source:    view,
				Priority:  30 + len(view.columns)*2,
			})
		} else if view.Type == 6 { // Simple View
			currentSig := ""
			for i, col := range view.columns {
				if i > 0 {
					currentSig += "|"
				}
				currentSig += col + "|="

				priority := 10
				if i > 0 {
					priority = 25 + (i-1)*2
				}

				// Prefix equality
				caps = append(caps, QueryCapability{
					Signature: currentSig,
					Source:    view,
					Priority:  priority,
				})

				// Range on clustering keys (everything after partition key)
				if i > 0 {
					caps = append(caps, QueryCapability{
						Signature: currentSig[:len(currentSig)-1] + "~",
						Source:    view,
						Priority:  priority - 5,
					})
				}
			}
		} else if view.Type == 9 { // Table-backed view
			currentSig := ""
			for i, col := range view.columns {
				if i > 0 {
					currentSig += "|"
				}

				operatorToken := capabilityDefaultOpForColumn(dbTable.columnsMap[col])
				currentSig += col + "|" + operatorToken

				priority := 10
				if i > 0 {
					priority = 25 + (i-1)*2
				}

				caps = append(caps, QueryCapability{
					Signature: currentSig,
					Source:    view,
					Priority:  priority,
				})

				if i > 0 {
					caps = append(caps, QueryCapability{
						Signature: currentSig[:len(currentSig)-1] + "~",
						Source:    view,
						Priority:  priority - 5,
					})
				}
			}
		} else if view.Type == 8 { // Range/Radix
			// Radix views always have equality on prefix, range on last.
			// They also usually keep part.
			cols := []string{}
			for i, col := range view.columns {
				if i < len(view.columns)-1 {
					cols = append(cols, col+"|=")
				} else {
					// Last one can be = or ~
					sigPrefix := strings.Join(cols, "|")
					if sigPrefix != "" {
						sigPrefix += "|"
					}

					caps = append(caps, QueryCapability{
						Signature: sigPrefix + col + "|=",
						Source:    view,
						Priority:  35 + len(view.columns)*2,
					})
					caps = append(caps, QueryCapability{
						Signature: sigPrefix + col + "|~",
						Source:    view,
						Priority:  30 + len(view.columns)*2,
					})
				}
			}
		}
	}

	// 4. KeyIntPacking
	if len(dbTable.keyIntPacking) > 0 {
		pk := dbTable.partKey
		if pk != nil && !pk.IsNil() {
			pkName := pk.GetName()
			currentSig := fmt.Sprintf("%v|=", pkName)

			for i, col := range dbTable.keyIntPacking {
				colName := col.GetName()
				if colName == "autoincrement_placeholder" {
					continue
				}

				// Equality
				caps = append(caps, QueryCapability{
					Signature: currentSig + fmt.Sprintf("|%v|=", colName),
					Priority:  40 + i*2,
					IsKey:     true,
				})

				// Range
				caps = append(caps, QueryCapability{
					Signature: currentSig + fmt.Sprintf("|%v|~", colName),
					Priority:  35 + i*2,
					IsKey:     true,
				})

				currentSig += fmt.Sprintf("|%v|=", colName)
			}
		}
	}

	// 5. KeyConcatenated
	if len(dbTable.keyConcatenated) > 0 {
		pk := dbTable.partKey
		if pk != nil && !pk.IsNil() {
			pkName := pk.GetName()
			currentSig := fmt.Sprintf("%v|=", pkName)

			for i, col := range dbTable.keyConcatenated {
				colName := col.GetName()
				// Equality on this prefix maps to a range or equality on the actual PK
				// If it's the last column of KeyConcatenated, it's equality on PK
				// Otherwise it's a range prefix search on PK.

				isLast := i == len(dbTable.keyConcatenated)-1

				// Equality
				caps = append(caps, QueryCapability{
					Signature: currentSig + fmt.Sprintf("|%v|=", colName),
					Priority:  25 + i*2,
					IsKey:     true, // It's handled by PK smart logic
				})

				// Range on this column
				caps = append(caps, QueryCapability{
					Signature: currentSig + fmt.Sprintf("|%v|~", colName),
					Priority:  20 + i*2,
					IsKey:     true,
				})

				if isLast {
					// All concatenated columns provided with equality = Equality on PK
				}

				currentSig += fmt.Sprintf("|%v|=", colName)
			}
		}
	}

	return caps
}

func MatchQueryCapability(statements []ColumnStatement, capabilities []QueryCapability) *QueryCapability {
	// Create a map of available columns and their operators in the query
	queryOps := make(map[string][]string)
	for _, st := range statements {
		op := capabilityOpForStatement(st)
		queryOps[st.Col] = append(queryOps[st.Col], op)
	}

	var bestMatch *QueryCapability

	for capabilityIndex := range capabilities {
		cap := &capabilities[capabilityIndex]
		parts := strings.Split(cap.Signature, "|")
		match := true

		// Every column in the signature MUST be in the query with the correct operator
		colMatches := make(map[string]int)
		for i := 0; i < len(parts); i += 2 {
			col := parts[i]
			op := parts[i+1]

			qOps, ok := queryOps[col]
			if !ok {
				match = false
				break
			}

			// Check if we have enough of this op in the query
			found := false
			startIdx := colMatches[col]
			for j := startIdx; j < len(qOps); j++ {
				if qOps[j] == op || (op == "~" && qOps[j] == "=") {
					found = true
					colMatches[col] = j + 1
					break
				}
			}

			if !found {
				match = false
				break
			}
		}

		if match {
			sourceName := "base-key"
			if cap.Source != nil {
				sourceName = cap.Source.name
			}
			fmt.Printf("Capability match: signature=%s priority=%d source=%s isKey=%v\n",
				cap.Signature, cap.Priority, sourceName, cap.IsKey)
			if bestMatch == nil || cap.Priority > bestMatch.Priority {
				bestMatch = cap
			} else if cap.Priority == bestMatch.Priority {
				// Prefer signature with more columns if priorities are equal
				if len(parts) > len(strings.Split(bestMatch.Signature, "|")) {
					bestMatch = cap
				}
			}
		}
	}

	if bestMatch != nil {
		sourceName := "base-key"
		if bestMatch.Source != nil {
			sourceName = bestMatch.Source.name
		}
		fmt.Printf("Capability selected: signature=%s priority=%d source=%s isKey=%v\n",
			bestMatch.Signature, bestMatch.Priority, sourceName, bestMatch.IsKey)
	}

	return bestMatch
}
