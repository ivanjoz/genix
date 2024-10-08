package db

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

type columnInfo struct {
	Name      string
	FieldType string
	FieldName string
	//FieldName      string
	NameAlias string
	Type      string
	// RefType        reflect.Value
	FieldIdx      int
	Idx           int16
	RefType       reflect.Value
	IsPrimaryKey  int8
	IsSlice       bool
	IsPointer     bool
	IsVirtual     bool
	HasView       bool
	IsComplexType bool
	ViewIdx       int8
	getValue      func(s *reflect.Value) any
	setValue      func(s *reflect.Value, v any)
}

var scyllaFieldToColumnTypesMap = map[string]string{
	"string":  "text",
	"int":     "int",
	"int32":   "int",
	"int64":   "bigint",
	"int64.1": "counter",
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

var makeStatementWith string = `	WITH caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
	and compaction = {'class': 'SizeTieredCompactionStrategy'}
	and compression = {'compression_level': '3', 'sstable_compression': 'org.apache.cassandra.io.compress.ZstdCompressor'}
	and dclocal_read_repair_chance = 0
	and speculative_retry = '99.0PERCENTILE'`

// https://forum.scylladb.com/t/what-is-the-difference-between-clustering-primary-partition-and-composite-or-compound-keys-in-scylladb/41
func makeTable[T TableSchemaInterface](structType T) scyllaTable[any] {
	return MakeTable(structType.GetSchema(), structType)
}

func MakeTable[T any](schema TableSchema, structType T) scyllaTable[any] {

	structRefValue := reflect.ValueOf(structType)
	structRefType := structRefValue.Type()

	if len(schema.Keys) == 0 {
		panic("No se ha especificado una PrimaryKey")
	}

	dbTable := scyllaTable[any]{
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

	fieldNameIdxMap := map[string]columnInfo{}
	colRegex, _ := regexp.Compile(`^Col\[.*\]$`)
	colSliceRegex, _ := regexp.Compile(`^ColSlice\[.*\]$`)

	for i := 0; i < structRefValue.NumField(); i++ {
		col := columnInfo{
			FieldIdx:  i,
			FieldType: structRefType.Field(i).Type.String(),
			FieldName: structRefType.Field(i).Name,
		}
		if col.FieldType[0:1] == "*" {
			col.IsPointer = true
			col.FieldType = col.FieldType[1:]
		}
		if col.FieldType[0:2] == "[]" {
			col.IsSlice = true
			col.FieldType = col.FieldType[2:]
		}

		// fmt.Println("Fieldname::", col.FieldName, "| Type:", col.FieldType)
		fieldNameIdxMap[col.FieldName] = col
	}

	for i := 0; i < structRefType.NumMethod(); i++ {
		method := structRefType.Method(i)
		if method.Type.NumOut() != 1 {
			continue
		}
		methodOutName := method.Type.Out(0).Name()
		isCol := colRegex.MatchString(methodOutName)
		isColSlice := colSliceRegex.MatchString(methodOutName)

		if !isCol && !isColSlice {
			continue
		}

		// fmt.Printf("Method:: %v | %v \n", method.Name, methodOutName)
		fieldName := method.Name
		if fieldName[len(fieldName)-1:] == "_" {
			fieldName = fieldName[0 : len(fieldName)-1]
		}

		columnFromField := fieldNameIdxMap[fieldName]
		if columnFromField.FieldType == "" {
			panic(fmt.Sprintf(`No se encontró la columna "%v" en el struct "%v"`, fieldName, structRefType.Name()))
		}

		mathodValue := structRefValue.Method(i).Call([]reflect.Value{})
		var column columnInfo
		if col, ok := mathodValue[0].Interface().(Coln); ok {
			column = col.GetInfo()
		} else {
			panic(fmt.Sprintf("La columna %v está mal configurada.", fieldName))
		}

		column.FieldIdx = columnFromField.FieldIdx
		column.FieldType = columnFromField.FieldType
		column.FieldName = columnFromField.FieldName
		column.IsPointer = columnFromField.IsPointer
		column.IsSlice = columnFromField.IsSlice
		column.setValue = fieldMapping[makeMappingKey(&column)]

		if column.setValue == nil {
			fmt.Println("Unrecognized type for column:", column.FieldName, "|", column.FieldType)
		}

		// Seteando "getValue"
		if column.IsSlice && column.FieldType == "string" {
			column.getValue = func(s *reflect.Value) any {
				field := s.Field(column.FieldIdx)
				if column.IsPointer {
					field = field.Elem()
				}

				if values, ok := field.Interface().([]string); ok {
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
			column.getValue = func(s *reflect.Value) any {
				field := s.Field(column.FieldIdx)
				if column.IsPointer {
					field = field.Elem()
				}
				concatenatedValues := Concatx(",", reflectToSlice(&field))
				return "{" + concatenatedValues + "}"
			}
		} else if column.IsPointer {
			column.getValue = func(s *reflect.Value) any {
				field := s.Field(column.FieldIdx)
				if field.IsNil() {
					return nil
				} else {
					return field.Elem().Interface()
				}
			}
		} else if column.IsComplexType {
			column.getValue = func(s *reflect.Value) any {
				field := s.Field(column.FieldIdx)
				if column.IsPointer {
					field = field.Elem()
				}
				recordBytes, err := cbor.Marshal(field)
				if err != nil {
					fmt.Println("Error al encodeding .cbor:: ", column.FieldName, err)
					return ""
				}
				hexString := hex.EncodeToString(recordBytes)
				return "0x" + hexString
			}
		} else if column.FieldType == "string" {
			column.getValue = func(s *reflect.Value) any {
				field := s.Field(column.FieldIdx)
				return any(fmt.Sprintf("'%v'", field))
			}
		} else {
			column.getValue = func(s *reflect.Value) any {
				field := s.Field(column.FieldIdx)
				return field.Interface()
			}
		}

		// Seteando "setValue"
		column.Type = scyllaFieldToColumnTypesMap[column.FieldType]
		if column.Type == "" {
			column.IsComplexType = true
			column.Type = "blob"
		} else if column.IsSlice {
			column.Type = fmt.Sprintf("set<%v>", column.Type)
		}

		if _, ok := dbTable.columnsMap[column.Name]; ok {
			panic("The following column name is repeated:" + column.Name)
		} else {
			column.Idx = int16(column.FieldIdx) + 1
			dbTable.columnsMap[column.Name] = &column
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
		index := viewInfo{
			Type:    1,
			name:    fmt.Sprintf(`%v__%v_index_0`, dbTable.name, colInfo.Name),
			idx:     idxCount,
			column:  colInfo,
			columns: []string{colInfo.Name},
		}
		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v (%v)`,
				index.name, dbTable.fullName(), column.GetInfo().Name)
		}

		idxCount++
		dbTable.indexes[index.name] = &index
	}

	for _, column := range schema.LocalIndexes {
		colInfo := column.GetInfo()
		index := viewInfo{
			Type:    2,
			name:    fmt.Sprintf(`%v__%v_index_1`, dbTable.name, colInfo.Name),
			idx:     idxCount,
			column:  dbTable.columnsMap[column.GetInfo().Name],
			columns: []string{dbTable.partKey.Name, colInfo.Name},
		}
		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v ((%v),%v)`,
				index.name, dbTable.fullName(), index.columns[0], index.columns[1])
		}

		idxCount++
		dbTable.indexes[index.name] = &index
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
		column := columnInfo{
			Name:      fmt.Sprintf(`zz_%v`, colnames),
			FieldType: "int32",
			Type:      "int",
			IsVirtual: true,
			Idx:       dbTable._maxColIdx,
		}

		dbTable._maxColIdx++
		dbTable.columnsMap[column.Name] = &column

		index := viewInfo{
			Type:    3,
			name:    fmt.Sprintf(`%v__%v_index`, dbTable.name, colnames),
			idx:     idxCount,
			columns: names,
			column:  &column,
		}

		if columnSlice != nil {
			column.FieldType = "[]int32"
			column.Type = "set<int>"
			column.IsSlice = true
			index.Type = 4

			column.getValue = func(s *reflect.Value) any {
				values := []any{}
				for _, col := range columnsNormal {
					values = append(values, col.getValue(s))
				}
				hashValues := []int32{}
				hashValues = append(hashValues, HashInt(values...))

				reflectSlice := s.Field(columnSlice.FieldIdx)
				if columnSlice.IsPointer {
					reflectSlice = reflectSlice.Elem()
				}
				for _, vl := range reflectToSlice(&reflectSlice) {
					hashValues = append(hashValues, HashInt(vl))
					hashValues = append(hashValues, HashInt(append(values, vl)...))
				}
				return "{" + Concatx(",", hashValues) + "}"
			}

			index.getStatement = func(statements ...ColumnStatement) []string {
				values := []any{}
				for _, st := range statements {
					values = append(values, st.GetValue())
				}
				hashValue := HashInt(values...)
				return []string{fmt.Sprintf("%v CONTAINS %v", column.Name, hashValue)}
			}
		} else {
			column.getValue = func(s *reflect.Value) any {
				values := []any{}
				for _, e := range columns {
					values = append(values, e.getValue(s))
				}
				return HashInt(values...)
			}
		}

		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v (%v)`,
				index.name, dbTable.fullName(), index.column.Name)
		}

		idxCount++
		dbTable.indexes[index.name] = &index
	}

	// VIEWS
	for _, viewConfig := range schema.Views {
		colNames := []string{}
		columns := []*columnInfo{} // No incluye la particion
		isRangeView := len(viewConfig.ConcatI64) > 0 || len(viewConfig.ConcatI32) > 0
		if isRangeView {
			viewConfig.KeepPart = true
		}

		for _, colInfo := range viewConfig.Cols {
			column := dbTable.columnsMap[colInfo.GetInfo().Name]
			if column.IsComplexType {
				panic("No puede ser un struct como columna de una view")
			}
			if column.IsSlice {
				panic("No puede ser un slice como columna de una view")
			}
			colNames = append(colNames, column.Name)
			columns = append(columns, column)
		}

		colNamesNoPart := colNames

		colNamesJoined := strings.Join(colNames, "_")
		if dbTable.partKey != nil {
			colNames = append([]string{dbTable.partKey.Name}, colNames...)
			if viewConfig.KeepPart {
				colNamesJoined = "pk_" + colNamesJoined
			} else {
				colNamesJoined = dbTable.partKey.Name + "_" + colNamesJoined
				columns = append([]*columnInfo{dbTable.partKey}, columns...)
			}
		}
		if isRangeView {
			colNamesJoined = colNamesJoined + "_rng"
		}

		view := viewInfo{
			Type:          6,
			name:          fmt.Sprintf(`%v__%v_view`, dbTable.name, colNamesJoined),
			columns:       colNames,
			columnsNoPart: colNamesNoPart,
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
			isInt64 := len(viewConfig.ConcatI64) > 0
			if isInt64 {
				view.column.FieldType = "int64"
				view.column.Type = "bigint"
			}
			view.Type = 8

			// Si ha especificado un int64 radix entonces se puede concatener, maximo 2 columnas
			if len(columns) < 2 {
				panic(fmt.Sprintf(`La view "%v" de la tabla "%v" posee menos de 2 columnas para usar el IntConcatRadix`, dbTable.name, view.name))
			}

			radixes := append(append(viewConfig.ConcatI64, viewConfig.ConcatI32...), 0)

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

			view.column.getValue = func(s *reflect.Value) any {
				sumValue := int64(0)
				values := []any{}
				for i, col := range columns {
					value := col.getValue(s)
					fmt.Println("Value Getted", col.Name, "|", value)
					valueI64 := convertToInt64(value) * Pow10Int64(radixesI64[i])
					values = append(values, value)
					sumValue += valueI64
				}
				fmt.Printf("Radix Sum Calculado %v | %v | %v\n", sumValue, values, radixesI64)
				if isInt64 {
					return any(sumValue)
				} else {
					return any(int32(sumValue))
				}
			}
		} else {
			view.Type = 7
			// Sino crea un hash de las columnas
			view.column.getValue = func(s *reflect.Value) any {
				values := []any{}
				// Si una de las columnas es un slice puede iterar por el slice para obtener los values y gurdarla en una columa Set<any>
				for _, e := range columns {
					values = append(values, e.getValue(s))
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
				for _, e := range columns {
					for _, st := range statements {
						if st.Column == e.Name {
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
					statement = fmt.Sprintf("%v = %v", view.column.Name, hashValues[0])
				} else {
					values := strings.Join(hashValues, ", ")
					statement = fmt.Sprintf("%v IN (%v)", view.column.Name, values)
				}

				if viewConfig.KeepPart {
					for _, st := range statements {
						if st.Column == dbTable.partKey.Name {
							statement = fmt.Sprintf("%v = %v AND ", st.Column, st.Value) + statement
						}
					}
				}
				return []string{statement}
			}
		}

		view.getCreateScript = func() string {
			whereCols := append([]*columnInfo{view.column}, dbTable.keys...)
			var wherePartCol *columnInfo

			if dbTable.partKey != nil {
				if viewConfig.KeepPart {
					wherePartCol = dbTable.partKey
				} else {
					whereCols = append([]*columnInfo{view.column, dbTable.partKey}, dbTable.keys...)
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
				if !col.IsVirtual || col.Name == view.column.Name {
					colNames = append(colNames, col.Name)
				}
			}

			query := fmt.Sprintf(`CREATE MATERIALIZED VIEW %v.%v AS
			SELECT %v FROM %v
			WHERE %v
			PRIMARY KEY (%v)
			%v;`,
				dbTable.keyspace, view.name, strings.Join(colNames, ", "), dbTable.fullName(),
				strings.Join(whereColumnsNotNull, " AND "), pk, makeStatementWith)
			return query
		}

		dbTable.views[view.name] = &view
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
