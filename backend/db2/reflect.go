package db2

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unsafe"

	"github.com/fxamacker/cbor/v2"
	"github.com/viant/xunsafe"
)

type columnInfo struct {
	Name      string
	FieldType string
	FieldName string
	//FieldName      string
	NameAlias string
	Type      string
	// RefType        reflect.Value
	FieldIdx          int
	Idx               int16
	RefType           reflect.Type
	IsPrimaryKey      int8
	IsSlice           bool
	IsPointer         bool
	IsVirtual         bool
	HasView           bool
	IsComplexType     bool
	ViewIdx           int8
	Field             *xunsafe.Field
	getValue          func(ptr unsafe.Pointer) any
	getStatementValue func(ptr unsafe.Pointer) any
	setValue          func(ptr unsafe.Pointer, v any)
}

var scyllaFieldToColumnTypesMap = map[string]string{
	"string":  "text",
	"int":     "int",
	"int32":   "int",
	"int64":   "bigint",
	"int16":   "smallint",
	"int8":    "tinyint",
	"float32": "float",
	"float64": "double",
}

var indexTypes = map[int8]string{
	1: "GLOBAL INDEX",
	2: "LOCAL INDEX",
	3: "HASH INDEX",
	4: "HASH INDEX w/COLLECTIONS",
	6: "VIEW",
	7: "HASH VIEW",
	8: "RANGE VIEW",
}

var rangeOperators = []string{">", "<", ">=", "<="}

var makeStatementWith string = `	WITH caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
	and compaction = {'class': 'SizeTieredCompactionStrategy'}
	and compression = {'compression_level': '3', 'sstable_compression': 'org.apache.cassandra.io.compress.ZstdCompressor'}
	and dclocal_read_repair_chance = 0
	and speculative_retry = '99.0PERCENTILE'`

// https://forum.scylladb.com/t/what-is-the-difference-between-clustering-primary-partition-and-composite-or-compound-keys-in-scylladb/41
type statementRangeGroup struct {
	from      *ColumnStatement
	betweenTo *ColumnStatement
}

func makeTable[T TableSchemaInterface[T]](structType *T) ScyllaTable[any] {

	schema := (*structType).GetSchema()
	structRefValue := reflect.ValueOf(structType).Elem()

	if len(schema.Keys) == 0 {
		panic("No se ha especificado una PrimaryKey")
	}

	dbTable := ScyllaTable[any]{
		keyspace:      schema.Keyspace,
		name:          schema.Name,
		columnsMap:    map[string]*columnInfo{},
		columnsIdxMap: map[int16]*columnInfo{},
		indexes:       map[string]*viewInfo{},
		views:         map[string]*viewInfo{},
		_maxColIdx:    int16(structRefValue.NumField()) + 1,
	}

	if dbTable.keyspace == "" {
		dbTable.keyspace = connParams.Keyspace
	}

	sequenceColumn := ""
	if schema.SequenceColumn != nil {
		sequenceColumn = schema.SequenceColumn.GetName()
	}

	// New implementation: iterate over struct fields and process Col[T,E] fields
	for i := 0; i < structRefValue.NumField(); i++ {
		field := structRefValue.Field(i)

		// Skip if field cannot be addressed or interfaced
		if !field.CanAddr() || !field.Addr().CanInterface() {
			panic("No es un interface!: " + field.Type().Name())
		}

		// Check if field implements Coln interface
		fieldAddr := field.Addr()
		colInterface, ok := fieldAddr.Interface().(Coln)
		if !ok {
			fieldName := field.Type().Name()
			if !(len(fieldName) > 12 && fieldName[0:12] == "TableStruct[") {
				fmt.Println("No es una columna:", field.Type().Name())
			}
			continue
		}

		// Get column info from the field
		column := new(columnInfo)
		*column = colInterface.GetInfo()

		fieldMappingFunc := fieldMapping[makeMappingKey(column)]
		if fieldMappingFunc != nil {
			f := column.Field
			column.setValue = func(ptr unsafe.Pointer, v any) {
				fieldMappingFunc(f, ptr, v)
			}
		}

		if column.setValue == nil {
			fmt.Println("Unrecognized type for column:", column.FieldName, "|", column.FieldType)
		}

		if DebugFull {
			offset := uintptr(0)
			if column.Field != nil {
				offset = column.Field.Offset
			}
			fmt.Printf("Mapped Table Column: %-20s | Field: %-20s | Type: %-10s | Offset: %d\n", column.Name, column.FieldName, column.FieldType, offset)
		}

		// Seteando "getStatementValue" (para los batchs)
		column.Type = scyllaFieldToColumnTypesMap[column.FieldType]
		if sequenceColumn == column.Name {
			column.Type = "counter"
		} else if column.Type == "" {
			column.IsComplexType = true
			column.IsSlice = false
			column.Type = "blob"
		} else if column.IsSlice {
			column.Type = fmt.Sprintf("set<%v>", column.Type)
		}

		if column.IsPointer {
			f := column.Field
			column.getStatementValue = func(ptr unsafe.Pointer) any {
				if f.Addr(ptr) == nil || f.IsNil(ptr) {
					return nil
				}
				return f.Interface(ptr)
			}
		} else if column.IsComplexType {
			f := column.Field
			fName := column.FieldName
			column.getStatementValue = func(ptr unsafe.Pointer) any {
				fieldValue := f.Interface(ptr)
				recordBytes, err := cbor.Marshal(fieldValue)
				if err != nil {
					fmt.Println("Error al encodeding .cbor:: ", fName, err)
					return ""
				}
				return recordBytes
			}
		} else {
			f := column.Field
			column.getStatementValue = func(ptr unsafe.Pointer) any {
				return f.Interface(ptr)
			}
		}

		// Seteando "getValue"
		if column.IsSlice && column.FieldType == "string" {
			f := column.Field
			column.getValue = func(ptr unsafe.Pointer) any {
				fieldValue := f.Interface(ptr)
				if values, ok := fieldValue.([]string); ok {
					strValues := []string{}
					for _, v := range values {
						strValues = append(strValues, `'`+v+`'`)
					}
					return "{" + strings.Join(strValues, ",") + "}"
				} else {
					panic("It must be string but was not.")
				}
			}
		} else if column.IsSlice {
			f := column.Field
			column.getValue = func(ptr unsafe.Pointer) any {
				// reflectToSlice needs to be updated or replaced
				concatenatedValues := Concatx(",", reflectToSlicePtr(f, ptr))
				return "{" + concatenatedValues + "}"
			}
		} else if column.IsPointer {
			f := column.Field
			column.getValue = func(ptr unsafe.Pointer) any {
				if f.IsNil(ptr) {
					return nil
				}
				return f.Interface(ptr)
			}
		} else if column.IsComplexType {
			f := column.Field
			fName := column.FieldName
			column.getValue = func(ptr unsafe.Pointer) any {
				fieldValue := f.Interface(ptr)
				recordBytes, err := cbor.Marshal(fieldValue)
				if err != nil {
					fmt.Println("Error al encodeding .cbor:: ", fName, err)
					return ""
				}
				hexString := hex.EncodeToString(recordBytes)
				return "0x" + hexString
			}
		} else if column.FieldType == "string" {
			f := column.Field
			column.getValue = func(ptr unsafe.Pointer) any {
				fieldValue := f.Interface(ptr)
				return any(fmt.Sprintf("'%v'", fieldValue))
			}
		} else {
			f := column.Field
			column.getValue = func(ptr unsafe.Pointer) any {
				return f.Interface(ptr)
			}
		}

		if _, ok := dbTable.columnsMap[column.Name]; ok {
			panic("The following column name is repeated:" + column.Name)
		} else {
			column.Idx = int16(column.FieldIdx) + 1
			dbTable.columnsMap[column.Name] = column
			if DebugFull {
				fmt.Printf("Mapped Col: %s, Field: %s, Offset: %d\n", column.Name, column.FieldName, column.Field.Offset)
			}
		}
	}
	if schema.Partition != nil {
		dbTable.partKey = dbTable.columnsMap[schema.Partition.GetInfo().Name]
		dbTable.keysIdx = append(dbTable.keysIdx, dbTable.partKey.Idx)
	}

	for _, key := range schema.Keys {
		col := dbTable.columnsMap[key.GetInfo().Name]
		dbTable.keys = append(dbTable.keys, col)
		dbTable.keysIdx = append(dbTable.keysIdx, col.Idx)
	}

	idxCount := int8(1)
	for _, column := range schema.GlobalIndexes {
		colInfo := dbTable.columnsMap[column.GetInfo().Name]
		index := &viewInfo{
			Type:    1,
			name:    fmt.Sprintf(`%v__%v_index_0`, dbTable.name, colInfo.Name),
			idx:     idxCount,
			column:  colInfo,
			columns: []string{colInfo.Name},
		}
		c := column
		index.getCreateScript = func() string {
			colInfo := c.GetInfo()
			colName := colInfo.Name
			if colInfo.IsSlice {
				colName = fmt.Sprintf("VALUES(%v)", colInfo.Name)
			}
			return fmt.Sprintf(`CREATE INDEX %v ON %v (%v)`, index.name, dbTable.GetFullName(), colName)
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	for _, column := range schema.LocalIndexes {
		colInfo := column.GetInfo()
		index := &viewInfo{
			Type:    2,
			name:    fmt.Sprintf(`%v__%v_index_1`, dbTable.name, colInfo.Name),
			idx:     idxCount,
			column:  dbTable.columnsMap[colInfo.Name],
			columns: []string{dbTable.partKey.Name, colInfo.Name},
		}
		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v ((%v),%v)`,
				index.name, dbTable.GetFullName(), index.columns[0], index.columns[1])
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	for _, indexColumns := range schema.HashIndexes {
		columns := []*columnInfo{}
		names := []string{}
		columnsNormal := []*columnInfo{}
		var columnSlice *columnInfo

		for _, colInfo := range indexColumns {
			column := dbTable.columnsMap[colInfo.GetInfo().Name]
			if column.IsComplexType {
				panic("No puede ser un struct como columna de una view")
			}
			if column.IsSlice {
				if columnSlice != nil {
					panic(fmt.Sprintf(`Table "%v". Can't create view with slice columns "%v" and "%v"`, dbTable.name, columnSlice.Name, column.Name))
				}
				columnSlice = column
			} else {
				columnsNormal = append(columnsNormal, column)
			}
			names = append(names, column.Name)
			columns = append(columns, column)
		}

		colnames := strings.Join(names, "_")
		column := &columnInfo{
			Name:      fmt.Sprintf(`zz_%v`, colnames),
			FieldType: "int32",
			Type:      "int",
			IsVirtual: true,
			Idx:       dbTable._maxColIdx,
		}

		dbTable._maxColIdx++
		dbTable.columnsMap[column.Name] = column

		index := &viewInfo{
			Type:    3,
			name:    fmt.Sprintf(`%v__%v_index`, dbTable.name, colnames),
			idx:     idxCount,
			columns: names,
			column:  column,
		}

		if columnSlice != nil {
			column.FieldType = "[]int32"
			column.Type = "set<int>"
			column.IsSlice = true
			index.Type = 4

			colNormal := columnsNormal
			colSlice := columnSlice
			column.getValue = func(ptr unsafe.Pointer) any {
				values := []any{}
				for _, col := range colNormal {
					values = append(values, col.getValue(ptr))
				}
				hashValues := []int32{}
				hashValues = append(hashValues, HashInt(values...))

				if colSlice.IsPointer && colSlice.Field.IsNil(ptr) {
					// Skip if nil pointer
				} else {
					fieldValue := colSlice.Field.Interface(ptr)
					for _, vl := range reflectToSliceValue(fieldValue) {
						hashValues = append(hashValues, HashInt(vl))
						hashValues = append(hashValues, HashInt(append(values, vl)...))
					}
				}
				return "{" + Concatx(",", hashValues) + "}"
			}

			colName := column.Name
			index.getStatement = func(statements ...ColumnStatement) []string {
				values := []any{}
				for _, st := range statements {
					values = append(values, st.GetValue())
				}
				hashValue := HashInt(values...)
				return []string{fmt.Sprintf("%v CONTAINS %v", colName, hashValue)}
			}
		} else {
			cols := columns
			column.getValue = func(ptr unsafe.Pointer) any {
				values := []any{}
				for _, e := range cols {
					values = append(values, e.getValue(ptr))
				}
				return HashInt(values...)
			}
		}

		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v (%v)`,
				index.name, dbTable.GetFullName(), index.column.Name)
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	// VIEWS
	for _, viewConfig := range schema.Views {
		viewCfg := viewConfig
		colNames := []string{}
		columns := []*columnInfo{} // No incluye la particion
		isRangeView := len(viewCfg.ConcatI64) > 0 || len(viewCfg.ConcatI32) > 0
		if isRangeView {
			viewCfg.KeepPart = true
		}

		for _, colInfo := range viewCfg.Cols {
			column := dbTable.columnsMap[colInfo.GetInfo().Name]
			if column.IsComplexType {
				panic("No puede usar un struct como columna de una view.")
			}
			if column.IsSlice {
				panic("No puede usar un slice como columna de una view. Intente con un índice global.")
			}
			colNames = append(colNames, column.Name)
			columns = append(columns, column)
		}

		colNamesNoPart := colNames

		colNamesJoined := strings.Join(colNames, "_")
		if dbTable.partKey != nil {
			colNames = append([]string{dbTable.partKey.Name}, colNames...)
			if viewCfg.KeepPart {
				colNamesJoined = "pk_" + colNamesJoined
			} else {
				colNamesJoined = dbTable.partKey.Name + "_" + colNamesJoined
				columns = append([]*columnInfo{dbTable.partKey}, columns...)
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

		for _, e := range columns {
			fmt.Println("view:", view.name, "| ", e.Name)
		}

		if len(columns) > 1 {
			view.column = &columnInfo{
				Name: fmt.Sprintf(`zz_%v`, colNamesJoined), IsVirtual: true,
				FieldType: "int32", Type: "int",
				Idx: dbTable._maxColIdx,
			}
			dbTable._maxColIdx++
			dbTable.columnsMap[view.column.Name] = view.column
		}

		// Si sólo es una columna, no es necesario autogenerar
		if len(columns) == 1 {
			view.column = columns[0]
		} else if isRangeView {
			isInt64 := len(viewCfg.ConcatI64) > 0
			if isInt64 {
				view.column.FieldType = "int64"
				view.column.Type = "bigint"
			}
			view.Type = 8

			// Si ha especificado un int64 radix entonces se puede concatener, maximo 2 columnas
			if len(columns) < 2 {
				panic(fmt.Sprintf(`La view "%v" de la tabla "%v" posee menos de 2 columnas para usar el IntConcatRadix`, dbTable.name, view.name))
			}

			radixes := append(append(viewCfg.ConcatI64, viewCfg.ConcatI32...), 0)

			if len(radixes) != len(columns) {
				panic(fmt.Sprintf(`The view "%v" in "%v" need to have %v radix arguments for the %v columns provided`,
					view.name, dbTable.name, len(columns)-1, len(columns)))
			}

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

			radixesI64 := []int64{}
			for _, v := range radixes {
				radixesI64 = append(radixesI64, int64(v))
			}

			supportedTypes := []string{"int8", "int16", "int32", "int64", "int"}

			for _, col := range columns {
				if col.IsSlice || !slices.Contains(supportedTypes, col.FieldType) {
					panic(fmt.Sprintf(`For view "%v" in "%v" need the column %v need to be a int type for the radix value be computed.`,
						view.name, dbTable.name, col.Name))
				}
			}

			var makeValue = func(values []int64) int64 {
				sumValue := int64(0)
				for i, value := range values {
					valueI64 := value * Pow10Int64(radixesI64[i])
					sumValue += valueI64
				}
				return sumValue
			}

			viewCols := columns
			isI64 := isInt64
			view.column.getValue = func(ptr unsafe.Pointer) any {
				values := []int64{}
				for _, col := range viewCols {
					values = append(values, convertToInt64(col.getValue(ptr)))
				}
				sumValue := makeValue(values)
				// fmt.Printf("Radix Sum Calculado %v | %v | %v\n", sumValue, values, radixesI64)
				if isI64 {
					return any(sumValue)
				} else {
					return any(int32(sumValue))
				}
			}

			viewPtr := view
			viewCfgPtr := &viewCfg
			view.getStatement = func(statements ...ColumnStatement) []string {

				//Identify is a statement has a partition value
				statementsMap := map[string]statementRangeGroup{}
				useBeetween := false

				fmt.Println(statements)

				for i := range statements {
					st := &statements[i]
					fmt.Println("statement (2):", st.Col, "|", st.Value, "|", st.Operator)

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

				whereStatements := []string{}
				var partStatement *ColumnStatement

				if viewCfgPtr.KeepPart {
					partStatement = statementsMap[dbTable.partKey.Name].from
					if partStatement == nil {
						panic(fmt.Sprintf(`The partition "%v" for table "%v" wasn't found.`, dbTable.partKey.Name, dbTable.name))
					}
				}

				for _, col := range viewCols {
					if _, ok := statementsMap[col.Name]; !ok {
						statementsMap[col.Name] = statementRangeGroup{
							from: &ColumnStatement{Value: int64(0)},
						}
					}
				}

				getValuesGroups := func() (valuesGroups [][]int64, rangeColumns []*columnInfo) {
					for _, col := range viewCols {
						// fmt.Println("iterando columna::", col.Name)
						st := statementsMap[col.Name].from
						if len(rangeColumns) > 0 || slices.Contains(rangeOperators, st.Operator) {
							rangeColumns = append(rangeColumns, col)
							// fmt.Println("continue here::", col.Name)
							continue
						}

						if st == nil {
							// fmt.Println("iterando columna 1::", col.Name)
							for i := range valuesGroups {
								valuesGroups[i] = append(valuesGroups[i], 0)
							}
						} else {
							statementValues := st.Values
							if len(statementValues) == 0 {
								statementValues = append(statementValues, st.Value)
							}

							// fmt.Println("iterando columna 2::", col.Name, statementValues)

							if len(valuesGroups) > 0 {
								valuesGroupsCurrent := valuesGroups
								valuesGroups = [][]int64{}
								for _, value := range statementValues {
									valueInt64 := convertToInt64(value)
									for _, vg := range valuesGroupsCurrent {
										valuesGroups = append(valuesGroups, append(vg, valueInt64))
									}
								}
							} else {
								for _, v := range statementValues {
									valuesGroups = append(valuesGroups, []int64{convertToInt64(v)})
								}
							}
						}
					}
					return valuesGroups, rangeColumns
				}

				if useBeetween {
					valuesFrom, valuesTo := []int64{}, []int64{}

					for _, col := range viewCols {
						srg := statementsMap[col.Name]
						valuesFrom = append(valuesFrom, convertToInt64(srg.from.Value))
						if srg.betweenTo != nil {
							valuesTo = append(valuesTo, convertToInt64(srg.betweenTo.Value))
						} else {
							valuesTo = append(valuesTo, convertToInt64(srg.from.Value))
						}
					}

					whereSt := fmt.Sprintf("%v >= %v AND %v < %v",
						viewPtr.column.Name, makeValue(valuesFrom),
						viewPtr.column.Name, makeValue(valuesTo),
					)
					if partStatement != nil {
						whereSt = fmt.Sprintf("%v = %v AND % v",
							dbTable.partKey.Name, convertToInt64(partStatement.Value), whereSt)
					}

					fmt.Println("Is useBeetween::", whereSt)

					return []string{whereSt}
				} else if slices.Contains(rangeOperators, statements[len(statements)-1].Operator) {
					valuesGroups, rangeColumns := getValuesGroups()

					// Create the ranges
					for _, valuesFrom := range valuesGroups {
						idxEnd := len(valuesFrom) - 1
						valuesTo := slices.Clone(valuesFrom)
						valuesTo[idxEnd]++

						for _, col := range rangeColumns {
							st := statementsMap[col.Name]
							valuesFrom = append(valuesFrom, convertToInt64(st.from.Value))
							valuesTo = append(valuesTo, 0)
						}

						whereStatement := fmt.Sprintf("%v >= %v AND %v < %v",
							viewPtr.column.Name, makeValue(valuesFrom),
							viewPtr.column.Name, makeValue(valuesTo),
						)
						whereStatements = append(whereStatements, whereStatement)
					}
				} else {
					valuesGroups, _ := getValuesGroups()
					fmt.Println("values group::", valuesGroups)

					hashValues := []int64{}
					for _, values := range valuesGroups {
						hashValues = append(hashValues, makeValue(values))
					}
					whereStatements = append(whereStatements,
						fmt.Sprintf("%v IN (%v)", viewPtr.column.Name, Concatx(", ", hashValues)),
					)
				}

				if partStatement != nil {
					for i, ws := range whereStatements {
						whereStatements[i] = fmt.Sprintf("%v = %v AND % v",
							dbTable.partKey.Name, convertToInt64(partStatement.Value), ws)
					}
				}

				fmt.Println("where statements::", whereStatements)
				return whereStatements
			}
		} else {
			viewPtr := view
			viewCfgPtr := &viewCfg
			viewCols := columns
			view.Operators = []string{"=", "IN"}
			view.Type = 7
			// Sino crea un hash de las columnas
			view.column.getValue = func(ptr unsafe.Pointer) any {
				values := []any{}
				// Si una de las columnas es un slice puede iterar por el slice para obtener los values y gurdarla en una columa Set<any>
				for _, e := range viewCols {
					values = append(values, e.getValue(ptr))
				}
				return HashInt(values...)
			}

			view.getStatement = func(statements ...ColumnStatement) []string {
				for i, e := range statements {
					fmt.Println("Statement ", i, " | ", e)
				}

				valuesGroups := [][]any{{}}
				statement := ""
				// Si una de las columnas es un slice puede iterar por el slice para obtener los values y gurdarla en una columa Set<any>
				for _, e := range viewCols {
					for _, st := range statements {
						if st.Col == e.Name {
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
					statement = fmt.Sprintf("%v = %v", viewPtr.column.Name, hashValues[0])
				} else {
					values := strings.Join(hashValues, ", ")
					statement = fmt.Sprintf("%v IN (%v)", viewPtr.column.Name, values)
				}

				if viewCfgPtr.KeepPart {
					for _, st := range statements {
						if st.Col == dbTable.partKey.Name {
							statement = fmt.Sprintf("%v = %v AND ", st.Col, st.Value) + statement
						}
					}
				}
				return []string{statement}
			}
		}

		viewPtr := view
		view.getCreateScript = func() string {
			whereCols := append([]*columnInfo{viewPtr.column}, dbTable.keys...)
			var wherePartCol *columnInfo

			if dbTable.partKey != nil {
				if viewCfg.KeepPart {
					wherePartCol = dbTable.partKey
				} else {
					whereCols = append([]*columnInfo{viewPtr.column, dbTable.partKey}, dbTable.keys...)
				}
			}

			keyNames := []string{}
			for _, col := range whereCols {
				keyNames = append(keyNames, col.Name)
			}

			pk := strings.Join(keyNames, ",")
			if wherePartCol != nil {
				pk = fmt.Sprintf("(%v), %v", wherePartCol.Name, pk)
			}

			whereColumnsNotNull := []string{}
			for _, col := range whereCols {
				if col.Type == "text" {
					whereColumnsNotNull = append(whereColumnsNotNull, col.Name+" > ''")
					/*} else if col.IsSlice {
					whereColumnsNotNull = append(whereColumnsNotNull, col.Name+" IS NOT NULL")
					*/
				} else {
					whereColumnsNotNull = append(whereColumnsNotNull, col.Name+" > 0")
				}
			}

			colNames := []string{}
			for _, col := range dbTable.columns {
				if !col.IsVirtual || col.Name == viewPtr.column.Name {
					colNames = append(colNames, col.Name)
				}
			}

			query := fmt.Sprintf(`CREATE MATERIALIZED VIEW %v.%v AS
			SELECT * FROM %v
			WHERE %v
			PRIMARY KEY (%v)
			%v;`,
				dbTable.keyspace, viewPtr.name, dbTable.GetFullName(),
				strings.Join(whereColumnsNotNull, " AND "), pk, makeStatementWith)
			return query
		}

		dbTable.views[view.name] = view
	}

	for _, col := range dbTable.columnsMap {
		dbTable.columns = append(dbTable.columns, col)
		dbTable.columnsIdxMap[col.Idx] = col
	}

	for _, e := range dbTable.indexes {
		dbTable.indexViews = append(dbTable.indexViews, e)
	}
	for _, e := range dbTable.views {
		dbTable.indexViews = append(dbTable.indexViews, e)
	}

	for _, idxview := range dbTable.indexViews {
		for _, colname := range idxview.columns {
			col := dbTable.columnsMap[colname]
			idxview.columnsIdx = append(idxview.columnsIdx, col.Idx)
		}
	}

	return dbTable
}
