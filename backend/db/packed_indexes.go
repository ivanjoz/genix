package db

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"
)

type packedIndexScope int8

const (
	packedIndexScopeLocal  packedIndexScope = 1
	packedIndexScopeGlobal packedIndexScope = 2
)

type packedIndexBuildConfig struct {
	scope packedIndexScope
	// schemaFieldName is used only for panic/error messages ("Indexes" vs "GlobalIndexes").
	schemaFieldName string
	// virtualColumnPrefix differentiates local/global packed columns (e.g. "zz_ixp_" vs "zz_gixp_").
	virtualColumnPrefix string
	// indexType is viewInfo.Type: 2 for local index, 1 for global index.
	indexType int8
	// requireInOnlyFirst restricts fan-out when using global indexes.
	// If true, IN is allowed only on the first component.
	requireInOnlyFirst bool
}

func isSupportedPackedIndexNumericFieldType(fieldType string) bool {
	// Packed indexes only support scalar integers. Slices are rejected by callers.
	switch fieldType {
	case "int8", "int16", "int32", "int64", "int":
		return true
	default:
		return false
	}
}

func registerPackedIndex(
	dbTable *ScyllaTable[any],
	idxCount *int8,
	indexColumns []Coln,
	cfg packedIndexBuildConfig,
) {
	// Build a stored packed numeric column plus an index on it.
	// Rationale: this enables composite predicates like "Status IN (...) + Updated BETWEEN/GT".
	if len(indexColumns) < 2 {
		panic(fmt.Sprintf(`Table "%v": %v entries must have at least 2 columns. Found: %v`, dbTable.name, cfg.schemaFieldName, len(indexColumns)))
	}

	sourceColumns := make([]IColInfo, 0, len(indexColumns))
	sourceColumnNames := make([]string, 0, len(indexColumns))
	slotDigitsPerColumn := make([]int64, 0, len(indexColumns))

	isInt32Packed := false
	totalDigits := int64(19)

	for columnIndex, indexColumnConfig := range indexColumns {
		configInfo := indexColumnConfig.GetInfo()
		column := dbTable.columnsMap[configInfo.Name]
		if column == nil {
			panic(fmt.Sprintf(`Table "%v": %v column "%v" was not found`, dbTable.name, cfg.schemaFieldName, configInfo.Name))
		}

		if column.GetType().IsComplexType || column.GetType().IsSlice || !isSupportedPackedIndexNumericFieldType(column.GetType().FieldType) {
			panic(fmt.Sprintf(`Table "%v": %v packed column "%v" must be a scalar integer. Found: %v`, dbTable.name, cfg.schemaFieldName, column.GetName(), column.GetType().FieldType))
		}

		if configInfo.useInt32Packing {
			isInt32Packed = true
			totalDigits = 9
		}

		// DecimalSize rules:
		// - First component MUST NOT set DecimalSize() (its width is implied by remaining digit budget).
		// - All remaining components MUST set DecimalSize().
		if columnIndex == 0 && configInfo.decimalSize > 0 {
			panic(fmt.Sprintf(`Table "%v": %v first column "%v" must not set DecimalSize()`, dbTable.name, cfg.schemaFieldName, configInfo.Name))
		}
		if columnIndex > 0 && configInfo.decimalSize <= 0 {
			panic(fmt.Sprintf(`Table "%v": %v requires DecimalSize() for column "%v" (all columns after the first must set DecimalSize)`, dbTable.name, cfg.schemaFieldName, configInfo.Name))
		}

		sourceColumns = append(sourceColumns, column)
		sourceColumnNames = append(sourceColumnNames, column.GetName())
		slotDigitsPerColumn = append(slotDigitsPerColumn, int64(configInfo.decimalSize)) // first column set after digit budget calc
	}

	sumTrailingDigits := int64(0)
	for i := 1; i < len(slotDigitsPerColumn); i++ {
		sumTrailingDigits += slotDigitsPerColumn[i]
	}

	if isInt32Packed {
		if sumTrailingDigits > 8 {
			panic(fmt.Sprintf(`Table "%v": int32 packed %v requires sum(DecimalSize(columns[1:])) <= 8. Got: %v`, dbTable.name, cfg.schemaFieldName, sumTrailingDigits))
		}
	} else {
		if sumTrailingDigits > 18 {
			panic(fmt.Sprintf(`Table "%v": int64 packed %v requires sum(DecimalSize(columns[1:])) <= 18. Got: %v`, dbTable.name, cfg.schemaFieldName, sumTrailingDigits))
		}
	}

	firstSlotDigits := totalDigits - sumTrailingDigits
	if firstSlotDigits <= 0 {
		panic(fmt.Sprintf(`Table "%v": %v invalid digit budget: totalDigits=%v sumTrailingDigits=%v`, dbTable.name, cfg.schemaFieldName, totalDigits, sumTrailingDigits))
	}
	slotDigitsPerColumn[0] = firstSlotDigits

	virtualPackedColName := fmt.Sprintf("%s%s", cfg.virtualColumnPrefix, strings.Join(sourceColumnNames, "_"))
	if _, exists := dbTable.columnsMap[virtualPackedColName]; exists {
		panic(fmt.Sprintf(`Table "%v": generated packed column already exists: %v`, dbTable.name, virtualPackedColName))
	}

	packedColumnTypeName := "int64"
	if isInt32Packed {
		packedColumnTypeName = "int32"
	}

	sourceColumnsLocal := slices.Clone(sourceColumns)
	slotDigitsLocal := slices.Clone(slotDigitsPerColumn)
	isInt32PackedLocal := isInt32Packed

	virtualPackedColumn := &columnInfo{
		colInfo: colInfo{
			Name:      virtualPackedColName,
			FieldName: virtualPackedColName,
			IsVirtual: true,
			Idx:       dbTable._maxColIdx,
		},
		colType: GetColTypeByName(packedColumnTypeName, ""),
	}
	virtualPackedColumn.getRawValue = func(ptr unsafe.Pointer) any {
		componentValues := make([]int64, 0, len(sourceColumnsLocal))
		for _, sourceColumn := range sourceColumnsLocal {
			valueI64 := convertToInt64(sourceColumn.GetRawValue(ptr))
			if valueI64 < 0 {
				panic(fmt.Sprintf(`Table "%v": packed %v column "%v" produced negative value %d`, dbTable.name, cfg.schemaFieldName, sourceColumn.GetName(), valueI64))
			}
			componentValues = append(componentValues, valueI64)
		}

		packed := computePackedInt64ValueNonNegative(componentValues, slotDigitsLocal)
		if !isInt32PackedLocal {
			return any(packed)
		}

		// Keep stored packed int within int32 constraints by trimming on the right side.
		// Reads must post-filter for exact semantics because trimming can overfetch.
		packedTrimmed := trimRightToDigitsNonNegative(packed, 9)
		return any(int32(packedTrimmed))
	}
	virtualPackedColumn.getValue = virtualPackedColumn.getRawValue

	dbTable._maxColIdx++
	dbTable.columnsMap[virtualPackedColumn.GetName()] = virtualPackedColumn

	indexNameSuffix := "index_0"
	if cfg.indexType == 2 {
		indexNameSuffix = "index_1"
	}
	indexName := fmt.Sprintf(`%v__%v_%v`, dbTable.name, virtualPackedColName, indexNameSuffix)
	if _, exists := dbTable.indexes[indexName]; exists {
		panic(fmt.Sprintf(`Table "%v": %v index name already exists: %v`, dbTable.name, cfg.schemaFieldName, indexName))
	}

	index := &viewInfo{
		Type:               cfg.indexType,
		name:               indexName,
		idx:                *idxCount,
		column:             virtualPackedColumn,
		columns:            slices.Clone(sourceColumnNames),
		RequiresPostFilter: true,
	}

	switch cfg.scope {
	case packedIndexScopeLocal:
		partitionColumn := dbTable.GetPartKey()
		if partitionColumn == nil || partitionColumn.IsNil() {
			panic(fmt.Sprintf(`Table "%v": %v requires a Partition column`, dbTable.name, cfg.schemaFieldName))
		}
		partitionName := partitionColumn.GetName()

		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v ((%v),%v)`, indexName, dbTable.GetFullName(), partitionName, virtualPackedColName)
		}

		dbTable.packedIndexes = append(dbTable.packedIndexes, &packedIndexInfo{
			indexName:           index.name,
			packedColumnName:    virtualPackedColName,
			sourceColumnNames:   slices.Clone(sourceColumnNames),
			partitionColumnName: partitionName,
			slotDigitsPerColumn: slices.Clone(slotDigitsPerColumn),
			totalDigits:         totalDigits,
			isInt32Packed:       isInt32Packed,
		})

	case packedIndexScopeGlobal:
		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v (%v)`, indexName, dbTable.GetFullName(), virtualPackedColName)
		}

		dbTable.packedIndexes = append(dbTable.packedIndexes, &packedIndexInfo{
			indexName:           index.name,
			packedColumnName:    virtualPackedColName,
			sourceColumnNames:   slices.Clone(sourceColumnNames),
			partitionColumnName: "",
			slotDigitsPerColumn: slices.Clone(slotDigitsPerColumn),
			totalDigits:         totalDigits,
			isInt32Packed:       isInt32Packed,
		})
	default:
		panic(fmt.Sprintf(`Table "%v": unknown packedIndexScope=%v`, dbTable.name, cfg.scope))
	}

	packedColumnNameLocal := virtualPackedColName
	lastSourceColNameLocal := sourceColumnNames[len(sourceColumnNames)-1]

	index.getStatement = func(statements ...ColumnStatement) []string {
		statementByColumn := map[string]ColumnStatement{}
		for _, st := range statements {
			statementByColumn[st.Col] = st
		}

		// Expand prefix value groups (all but the last column). This can produce fan-out.
		prefixValueGroups := [][]int64{{}}
		for i := 0; i < len(sourceColumnNames)-1; i++ {
			colName := sourceColumnNames[i]
			st, ok := statementByColumn[colName]
			if !ok {
				return nil
			}

			if cfg.requireInOnlyFirst && i > 0 && st.Operator == "IN" {
				return nil
			}

			values := []int64{}
			switch st.Operator {
			case "=":
				values = append(values, convertToInt64(st.Value))
			case "IN":
				for _, v := range st.Values {
					values = append(values, convertToInt64(v))
				}
			default:
				return nil
			}

			nextGroups := [][]int64{}
			for _, group := range prefixValueGroups {
				for _, v := range values {
					nextGroup := append(slices.Clone(group), v)
					nextGroups = append(nextGroups, nextGroup)
				}
			}
			prefixValueGroups = nextGroups
		}

		lastStatement, ok := statementByColumn[lastSourceColNameLocal]
		if !ok {
			return nil
		}

		finalizePackedForStorageType := func(packed int64) int64 {
			if !isInt32PackedLocal {
				return packed
			}
			return trimRightToDigitsNonNegative(packed, 9)
		}

		emitRangeClause := func(prefixValues []int64, operator string, boundValue int64) string {
			componentValues := append(slices.Clone(prefixValues), boundValue)
			packed := finalizePackedForStorageType(computePackedInt64ValueNonNegative(componentValues, slotDigitsLocal))

			// Avoid underfetch due to truncation on strict bounds.
			switch operator {
			case ">":
				operator = ">="
			case "<":
				operator = "<="
			}

			return fmt.Sprintf("%v %v %v", packedColumnNameLocal, operator, packed)
		}

		whereStatements := []string{}
		for _, prefixValues := range prefixValueGroups {
			switch lastStatement.Operator {
			case "=":
				componentValues := append(slices.Clone(prefixValues), convertToInt64(lastStatement.Value))
				packed := finalizePackedForStorageType(computePackedInt64ValueNonNegative(componentValues, slotDigitsLocal))
				whereStatements = append(whereStatements, fmt.Sprintf("%v = %v", packedColumnNameLocal, packed))
			case "BETWEEN":
				if len(lastStatement.From) == 0 || len(lastStatement.To) == 0 {
					return nil
				}
				fromValue := convertToInt64(lastStatement.From[0].Value)
				toValue := convertToInt64(lastStatement.To[0].Value)
				fromClause := emitRangeClause(prefixValues, ">=", fromValue)
				toClause := emitRangeClause(prefixValues, "<=", toValue)
				whereStatements = append(whereStatements, fromClause+" AND "+toClause)
			case ">", ">=", "<", "<=":
				whereStatements = append(whereStatements, emitRangeClause(prefixValues, lastStatement.Operator, convertToInt64(lastStatement.Value)))
			default:
				return nil
			}
		}

		return whereStatements
	}

	*idxCount++
	dbTable.indexes[index.name] = index

	fmt.Printf("Packed index registered: table=%s scope=%v index=%s packedCol=%s isInt32=%v slotDigits=%v\n",
		dbTable.name, cfg.scope, index.name, virtualPackedColName, isInt32Packed, slotDigitsPerColumn)
}
