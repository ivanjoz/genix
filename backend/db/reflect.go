package db

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unsafe"

	"github.com/fxamacker/cbor/v2"
	"github.com/viant/xunsafe"
)

type colInfo struct {
	Name         string
	FieldName    string
	NameAlias    string
	IsPrimaryKey int8
	FieldIdx     int
	IsVirtual    bool
	HasView      bool
	ViewIdx      int8
	Idx          int16
	RefType      reflect.Type
	Field        *xunsafe.Field
}

type columnInfo struct {
	colInfo
	colType
	getValue    func(ptr unsafe.Pointer) any
	getRawValue func(ptr unsafe.Pointer) any
}

func (c *columnInfo) GetValue(ptr unsafe.Pointer) any {
	if c.getValue != nil {
		return c.getValue(ptr)
	}
	return makeScyllaValue(c.Field, ptr, c.Type)
}

func (c *columnInfo) GetRawValue(ptr unsafe.Pointer) any {
	if c.getRawValue != nil {
		return c.getRawValue(ptr)
	}
	if c.Field == nil {
		return nil
	}
	if c.IsPointer {
		if c.Field.IsNil(ptr) {
			return nil
		}
	}
	return c.Field.Interface(ptr)
}

func (c *columnInfo) GetStatementValue(ptr unsafe.Pointer) any {
	if c.getRawValue != nil {
		return c.getRawValue(ptr)
	}
	if c.getValue != nil {
		return c.getValue(ptr)
	}
	if c.Field == nil {
		return nil
	}
	if c.IsPointer {
		if c.Field.IsNil(ptr) {
			return nil
		}
		return c.Field.Interface(ptr)
	}
	if c.IsComplexType {
		fieldValue := c.Field.Interface(ptr)
		recordBytes, err := cbor.Marshal(fieldValue)
		if err != nil {
			fmt.Println("Error al encodeding .cbor:: ", c.FieldName, err)
			return ""
		}
		return recordBytes
	}
	return c.Field.Interface(ptr)
}

func (c *columnInfo) SetValue(ptr unsafe.Pointer, v any) {
	if c.Type == 9 {
		var vl []byte
		if b, ok := v.(*[]byte); ok {
			vl = *b
		} else if b, ok := v.([]byte); ok {
			vl = b
		}

		if len(vl) > 3 && c.Field != nil {
			// Direct unmarshal into the field memory using xunsafe pointer
			dest := reflect.NewAt(c.RefType, c.Field.Pointer(ptr)).Interface()
			err := cbor.Unmarshal(vl, dest)
			if err != nil {
				fmt.Printf("Error al convertir ComplexType for Col %s: %v\n", c.Name, err)
			}
		} else if ShouldLog() {
			fmt.Printf("Complex Type could not be parsed or empty: %s (Type: %T)\n", c.Name, v)
		}
	} else {
		assingValue(c.Field, ptr, c.Type, v)
	}
}

func (c *columnInfo) GetType() *colType {
	return &c.colType
}

func (c *columnInfo) GetName() string {
	return c.Name
}

func (c *columnInfo) GetInfo() *colInfo {
	return &c.colInfo
}

func (c *columnInfo) IsNil() bool {
	return c == nil
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
		columnsMap:    map[string]IColInfo{},
		columnsIdxMap: map[int16]IColInfo{},
		indexes:       map[string]*viewInfo{},
		views:         map[string]*viewInfo{},
		useSequences:  schema.UseSequences,
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
		fieldName := field.Type().Name()

		if !ok {
			if !(len(fieldName) > 12 && fieldName[0:12] == "TableStruct[") {
				fmt.Println("No es una columna:", fieldName)
			}
			continue
		}

		// Get column info from the field
		column := colInterface.GetInfo()
		// fmt.Printf("DEBUG: makeTable field=%s, GetInfo().Name=%s, Type=%T\n", fieldName, column.Name, colInterface)
		if column.Name == "" {
			fmt.Printf("DEBUG: makeTable EMPTY NAME for field=%s! columnInfo=%+v\n", fieldName, column)
			panic("El nombre de la columna no está seteada.")
		}

		if DebugFull {
			offset := uintptr(0)
			if column.GetInfo().Field != nil {
				offset = column.GetInfo().Field.Offset
			}
			fmt.Printf("Mapped Table Column: %-20s | Field: %-20s | Type: %-10s | Offset: %d\n", column.GetName(), column.GetInfo().FieldName, column.FieldType, offset)
		}

		if sequenceColumn == column.GetName() {
			column.ColType = "counter"
		} /* else if column.ColType == "" {
			column.GetType().IsComplexType = true
			column.Type = 9
			column.GetType().IsSlice = false
			column.ColType = "blob"
		} else if column.GetType().IsSlice && column.ColType[0:3] != "set" {
			column.ColType = fmt.Sprintf("set<%v>", column.ColType)
		} */

		if _, ok := dbTable.columnsMap[column.GetName()]; ok {
			panic("The following column name is repeated:" + column.GetName())
		} else {
			column.GetInfo().Idx = int16(column.GetInfo().FieldIdx) + 1
			dbTable.columnsMap[column.GetName()] = &column
			if DebugFull {
				fmt.Printf("Mapped Col: %s, Field: %s, Offset: %d\n", column.GetName(), column.GetInfo().FieldName, column.GetInfo().Field.Offset)
			}
		}
	}

	if schema.Partition != nil {
		dbTable.partKey = dbTable.columnsMap[schema.Partition.GetInfo().Name]
		if dbTable.partKey != nil && !dbTable.partKey.IsNil() {
			dbTable.keysIdx = append(dbTable.keysIdx, dbTable.partKey.GetInfo().Idx)
		}
	}

	for _, key := range schema.Keys {
		col := dbTable.columnsMap[key.GetInfo().Name]
		dbTable.keys = append(dbTable.keys, col)
		dbTable.keysIdx = append(dbTable.keysIdx, col.GetInfo().Idx)
	}

	if len(schema.KeyConcatenated) > 0 {
		if len(dbTable.keys) != 1 {
			panic(fmt.Sprintf(`Table "%v": KeyConcatenated requires exactly one column in Keys. Found: %v`, dbTable.name, len(dbTable.keys)))
		}
		keyCol := dbTable.keys[0].(*columnInfo)
		if keyCol.Type != 1 { // 1 = string
			panic(fmt.Sprintf(`Table "%v": KeyConcatenated requires the key column to be a string. Found: %v`, dbTable.name, keyCol.FieldType))
		}

		concatCols := []IColInfo{}
		for _, col := range schema.KeyConcatenated {
			concatCol := dbTable.columnsMap[col.GetName()]
			concatCols = append(concatCols, concatCol)
			dbTable.keyConcatenated = append(dbTable.keyConcatenated, concatCol)
		}

		keyCol.getRawValue = func(ptr unsafe.Pointer) any {
			values := []any{}
			for _, col := range concatCols {
				values = append(values, col.GetRawValue(ptr))
			}
			return Concat62(values...)
		}

		keyCol.getValue = func(ptr unsafe.Pointer) any {
			return "'" + keyCol.getRawValue(ptr).(string) + "'"
		}
	}

	idxCount := int8(1)
	for _, column := range schema.GlobalIndexes {
		colInfo := dbTable.columnsMap[column.GetInfo().Name]
		index := &viewInfo{
			Type:    1,
			name:    fmt.Sprintf(`%v__%v_index_0`, dbTable.name, colInfo.GetName()),
			idx:     idxCount,
			column:  colInfo,
			columns: []string{colInfo.GetName()},
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

	if schema.SequencePartCol != nil {
		gi := schema.SequencePartCol.GetInfo()
		dbTable.sequencePartCol = &gi
	}

	for _, column := range schema.LocalIndexes {
		colInfo := column.GetInfo()
		index := &viewInfo{
			Type:    2,
			name:    fmt.Sprintf(`%v__%v_index_1`, dbTable.name, colInfo.GetName()),
			idx:     idxCount,
			column:  dbTable.columnsMap[colInfo.GetName()],
			columns: []string{dbTable.GetPartKey().GetName(), colInfo.GetName()},
		}
		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v ((%v),%v)`,
				index.name, dbTable.GetFullName(), index.columns[0], index.columns[1])
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	for _, indexColumns := range schema.HashIndexes {
		columns := []IColInfo{}
		names := []string{}
		columnsNormal := []IColInfo{}
		var columnSlice IColInfo

		for _, colInfo := range indexColumns {
			column := dbTable.columnsMap[colInfo.GetInfo().Name]
			if column.GetType().IsComplexType {
				panic("No puede ser un struct como columna de una view")
			}
			if column.GetType().IsSlice {
				if columnSlice != nil {
					panic(fmt.Sprintf(`Table "%v". Can't create view with slice columns "%v" and "%v"`, dbTable.name, columnSlice.GetName(), column.GetName()))
				}
				columnSlice = column
			} else {
				columnsNormal = append(columnsNormal, column)
			}
			names = append(names, column.GetName())
			columns = append(columns, column)
		}

		colnames := strings.Join(names, "_")
		column := &columnInfo{
			colInfo: colInfo{
				IsVirtual: true,
				Idx:       dbTable._maxColIdx,
			},
			colType: colType{
				FieldType: "int32",
				ColType:   "int",
			},
		}

		column.GetInfo().Name = fmt.Sprintf(`zz_%v`, colnames)

		dbTable._maxColIdx++
		dbTable.columnsMap[column.GetName()] = column

		index := &viewInfo{
			Type:    3,
			name:    fmt.Sprintf(`%v__%v_index`, dbTable.name, colnames),
			idx:     idxCount,
			columns: names,
			column:  column,
		}

		if columnSlice != nil {
			column.GetType().FieldType = "[]int32"
			column.GetType().ColType = "set<int>"
			column.GetType().IsSlice = true
			index.Type = 4

			colNormal := columnsNormal
			colSlice := columnSlice
			column.getValue = func(ptr unsafe.Pointer) any {
				values := []any{}
				for _, col := range colNormal {
					values = append(values, col.GetValue(ptr))
				}
				hashValues := []int32{}
				hashValues = append(hashValues, HashInt(values...))

				colSliceInfo := colSlice.GetInfo()
				if column.GetType().IsPointer && colSliceInfo.Field.IsNil(ptr) {
					// Skip if nil pointer
				} else {
					fieldValue := colSliceInfo.Field.Interface(ptr)
					for _, vl := range reflectToSliceValue(fieldValue) {
						hashValues = append(hashValues, HashInt(vl))
						hashValues = append(hashValues, HashInt(append(values, vl)...))
					}
				}
				return "{" + Concatx(",", hashValues) + "}"
			}

			colName := column.GetName()
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
					values = append(values, e.GetValue(ptr))
				}
				return HashInt(values...)
			}
		}

		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v (%v)`,
				index.name, dbTable.GetFullName(), index.column.GetName())
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	// VIEWS
	for _, viewConfig := range schema.Views {
		viewCfg := viewConfig
		colNames := []string{}
		columns := []IColInfo{} // No incluye la particion
		isRangeView := len(viewCfg.ConcatI64) > 0 || len(viewCfg.ConcatI32) > 0
		if isRangeView {
			viewCfg.KeepPart = true
		}

		for _, colInfo := range viewCfg.Cols {
			column := dbTable.columnsMap[colInfo.GetInfo().Name]
			if column.GetType().IsComplexType {
				panic("No puede usar un struct como columna de una view.")
			}
			if column.GetType().IsSlice {
				panic("No puede usar un slice como columna de una view. Intente con un índice global.")
			}
			colNames = append(colNames, column.GetName())
			columns = append(columns, column)
		}

		colNamesNoPart := colNames

		colNamesJoined := strings.Join(colNames, "_")
		pk := dbTable.GetPartKey()
		if pk != nil && !pk.IsNil() {
			colNames = append([]string{pk.GetName()}, colNames...)
			if viewCfg.KeepPart {
				colNamesJoined = "pk_" + colNamesJoined
			} else {
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

		for _, e := range columns {
			fmt.Println("view:", view.name, "| ", e.GetName())
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

		// Si sólo es una columna, no es necesario autogenerar
		if len(columns) == 1 {
			view.column = columns[0]
		} else if isRangeView {
			isInt64 := len(viewCfg.ConcatI64) > 0
			if isInt64 {
				view.column.GetType().FieldType = "int64"
				view.column.GetType().ColType = "bigint"
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
				if col.GetType().IsSlice || !slices.Contains(supportedTypes, col.GetType().FieldType) {
					panic(fmt.Sprintf(`For view "%v" in "%v" need the column %v need to be a int type for the radix value be computed.`,
						view.name, dbTable.name, col.GetName()))
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
			view.column.(*columnInfo).getValue = func(ptr unsafe.Pointer) any {
				values := []int64{}
				for _, col := range viewCols {
					values = append(values, convertToInt64(col.GetValue(ptr)))
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

				// fmt.Println(statements)

				for i := range statements {
					st := &statements[i]
					// fmt.Println("statement (2):", st.Col, "|", st.Value, "|", st.Operator)

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
					pk := dbTable.GetPartKey()
					if pk == nil || pk.IsNil() {
						panic(fmt.Sprintf(`The partition for table "%v" wasn't found.`, dbTable.name))
					}
					partStatement = statementsMap[pk.GetName()].from
					if partStatement == nil {
						panic(fmt.Sprintf(`The partition "%v" for table "%v" wasn't found.`, pk.GetName(), dbTable.name))
					}
				}

				for _, col := range viewCols {
					if _, ok := statementsMap[col.GetName()]; !ok {
						statementsMap[col.GetName()] = statementRangeGroup{
							from: &ColumnStatement{Value: int64(0)},
						}
					}
				}

				getValuesGroups := func() (valuesGroups [][]int64, rangeColumns []IColInfo) {
					for _, col := range viewCols {
						// fmt.Println("iterando columna::", col.GetName())
						st := statementsMap[col.GetName()].from
						if len(rangeColumns) > 0 || slices.Contains(rangeOperators, st.Operator) {
							rangeColumns = append(rangeColumns, col)
							// fmt.Println("continue here::", col.GetName())
							continue
						}

						if st == nil {
							// fmt.Println("iterando columna 1::", col.GetName())
							for i := range valuesGroups {
								valuesGroups[i] = append(valuesGroups[i], 0)
							}
						} else {
							statementValues := st.Values
							if len(statementValues) == 0 {
								statementValues = append(statementValues, st.Value)
							}

							// fmt.Println("iterando columna 2::", col.GetName(), statementValues)

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
						srg := statementsMap[col.GetName()]
						valuesFrom = append(valuesFrom, convertToInt64(srg.from.Value))
						if srg.betweenTo != nil {
							valuesTo = append(valuesTo, convertToInt64(srg.betweenTo.Value))
						} else {
							valuesTo = append(valuesTo, convertToInt64(srg.from.Value))
						}
					}

					whereSt := fmt.Sprintf("%v >= %v AND %v < %v",
						viewPtr.column.GetName(), makeValue(valuesFrom),
						viewPtr.column.GetName(), makeValue(valuesTo),
					)
					if partStatement != nil {
						pk := dbTable.GetPartKey()
						if pk != nil && !pk.IsNil() {
							whereSt = fmt.Sprintf("%v = %v AND %v",
								pk.GetName(), convertToInt64(partStatement.Value), whereSt)
						}
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
							st := statementsMap[col.GetName()]
							valuesFrom = append(valuesFrom, convertToInt64(st.from.Value))
							valuesTo = append(valuesTo, 0)
						}

						whereStatement := fmt.Sprintf("%v >= %v AND %v < %v",
							viewPtr.column.GetName(), makeValue(valuesFrom),
							viewPtr.column.GetName(), makeValue(valuesTo),
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
			view.column.(*columnInfo).getValue = func(ptr unsafe.Pointer) any {
				values := []any{}
				// Si una de las columnas es un slice puede iterar por el slice para obtener los values y gurdarla en una columa Set<any>
				for _, e := range viewCols {
					values = append(values, e.GetValue(ptr))
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

		viewPtr := view
		view.getCreateScript = func() string {
			whereCols := append([]IColInfo{viewPtr.column}, dbTable.keys...)
			var wherePartCol IColInfo

			pk_ := dbTable.GetPartKey()
			if pk_ != nil && !pk_.IsNil() {
				if viewCfg.KeepPart {
					wherePartCol = pk_
				} else {
					whereCols = append([]IColInfo{viewPtr.column, pk_}, dbTable.keys...)
				}
			}

			keyNames := []string{}
			for _, col := range whereCols {
				keyNames = append(keyNames, col.GetName())
			}

			pk := strings.Join(keyNames, ",")
			if wherePartCol != nil {
				pk = fmt.Sprintf("(%v), %v", wherePartCol.GetName(), pk)
			}

			whereColumnsNotNull := []string{}
			for _, col := range whereCols {
				if col.GetType().ColType == "text" {
					whereColumnsNotNull = append(whereColumnsNotNull, col.GetName()+" > ''")
					/*} else if col.IsSlice {
					whereColumnsNotNull = append(whereColumnsNotNull, col.Name+" IS NOT NULL")
					*/
				} else {
					whereColumnsNotNull = append(whereColumnsNotNull, col.GetName()+" > 0")
				}
			}

			colNames := []string{}
			for _, col := range dbTable.columns {
				if !col.GetInfo().IsVirtual || col.GetName() == viewPtr.column.GetName() {
					colNames = append(colNames, col.GetName())
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
		dbTable.columnsIdxMap[col.GetInfo().Idx] = col
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
			idxview.columnsIdx = append(idxview.columnsIdx, col.GetInfo().Idx)
		}
	}

	dbTable.capabilities = dbTable.ComputeCapabilities()

	return dbTable
}
