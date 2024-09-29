package db

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

type columnInfo struct {
	Name      string
	FieldType string
	FieldName string
	//FieldName      string
	NameAlias string
	Type      string
	// RefType        reflect.Value
	FieldIdx       int
	RefType        reflect.Value
	IsPrimaryKey   int8
	IsSlice        bool
	IsPointer      bool
	IsViewExcluded bool
	IsVirtual      bool
	HasView        bool
	IsComplexType  bool
	ViewIdx        int8
	getValue       func(s *reflect.Value) any
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
	// fmt.Println("struct type:", structRefType, "| numfield:", structRefType.NumField())

	if len(schema.Keys) == 0 {
		panic("No se ha especificado una PrimaryKey")
	}

	dbTable := scyllaTable[any]{
		keyspace:   schema.Keyspace,
		name:       schema.Name,
		columnsMap: map[string]*columnInfo{},
		indexes:    map[string]viewInfo{},
		views:      map[string]viewInfo{},
	}

	if dbTable.keyspace == "" {
		dbTable.keyspace = connParams.Keyspace
	}

	if schema.Partition != nil {
		dbTable.partitionKey = schema.Partition.GetInfo()
	}

	for _, key := range schema.Keys {
		dbTable.keys = append(dbTable.keys, key.GetInfo())
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
		if col, ok := mathodValue[0].Interface().(Column); ok {
			column = col.GetInfo()
		} else {
			panic(fmt.Sprintf("La columna %v está mal configurada.", fieldName))
		}

		column.FieldIdx = columnFromField.FieldIdx
		column.FieldType = columnFromField.FieldType
		column.FieldName = columnFromField.FieldName
		column.IsPointer = columnFromField.IsPointer
		column.IsSlice = columnFromField.IsSlice

		// fmt.Println("Column Name:", column.Name, "| Type:", column.FieldType, "| Is Slice:", column.IsSlice)

		column.getValue = func(s *reflect.Value) any {
			field := s.Field(column.FieldIdx)
			// fmt.Println("Col: ", column.Name, " | Type::", field.Type().String())
			if column.IsPointer {
				if field.IsNil() {
					return nil
				} else {
					return field.Elem().Interface()
				}
			} else {
				return field.Interface()
			}
		}

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
			dbTable.columnsMap[column.Name] = &column
		}
	}

	idxCount := int8(1)
	for _, column := range schema.GlobalIndexes {
		colInfo := dbTable.columnsMap[column.GetInfo().Name]
		index := viewInfo{
			iType:   1,
			name:    fmt.Sprintf(`%v__%v_index_0`, dbTable.name, colInfo.Name),
			idx:     idxCount,
			column:  colInfo,
			columns: []string{colInfo.Name},
			getValue: func(s *reflect.Value) any {
				return colInfo.getValue(s)
			},
			getStatement: func(statements ...ColumnStatement) string {
				if len(statements) != 1 {
					panic(fmt.Sprintf("Error columna %v: El número de valores debe ser = 1", colInfo.Name))
				}
				st := statements[0]
				return fmt.Sprintf("%v %v %v", colInfo.Name, st.Operator, st.Value)
			},
		}
		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v (%v)`,
				index.name, dbTable.fullName(), column.GetInfo().Name)
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	for _, column := range schema.LocalIndexes {
		colInfo := column.GetInfo()
		index := viewInfo{
			iType:   2,
			name:    fmt.Sprintf(`%v__%v_index_1`, dbTable.name, colInfo.Name),
			idx:     idxCount,
			columns: []string{dbTable.partitionKey.Name, colInfo.Name},
			getValue: func(s *reflect.Value) any {
				return colInfo.getValue(s)
			},
			getStatement: func(sts ...ColumnStatement) string {
				if len(sts) != 2 {
					panic(fmt.Sprintf("Error columna %v: El número de valores debe ser = 2", colInfo.Name))
				}
				return fmt.Sprintf("%v = %v AND %v %v %v", dbTable.partitionKey.Name, sts[0].Value, colInfo.Name, sts[1].Operator, sts[1].Value)
			},
		}
		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v ((%v),%v)`,
				index.name, dbTable.fullName(), index.columns[0], index.columns[1])
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	for _, indexColumns := range schema.HashIndexes {
		columns := []*columnInfo{}
		names := []string{}

		for _, col := range indexColumns {
			name := col.GetInfo().Name
			names = append(names, name)
			columns = append(columns, dbTable.columnsMap[name])
		}

		colnames := strings.Join(names, "_")
		column := columnInfo{
			Name:      fmt.Sprintf(`zz_%v`, colnames),
			FieldType: "int32", Type: "int", IsVirtual: true,
			getValue: func(s *reflect.Value) any {
				values := []any{}
				for _, e := range columns {
					values = append(values, e.getValue(s))
				}
				return HashInt(values...)
			},
		}

		dbTable.columnsMap[column.Name] = &column

		index := viewInfo{
			iType:    3,
			name:     fmt.Sprintf(`%v__%v_index`, dbTable.name, colnames),
			idx:      idxCount,
			columns:  names,
			column:   &column,
			getValue: column.getValue,
		}

		index.getStatement = func(statements ...ColumnStatement) string {
			if len(statements) < 2 {
				panic(fmt.Sprintf("Error columna %v: El número de valores debe ser >= 2", index.name))
			}

			values := []any{}
			for _, e := range statements {
				values = append(values, e.Value)
			}
			hashInt := HashInt(values...)
			// Revisar casuísica IN
			return fmt.Sprintf("%v %v %v", index.column.Name, statements[0].Operator, hashInt)
		}

		index.getCreateScript = func() string {
			return fmt.Sprintf(`CREATE INDEX %v ON %v (%v)`,
				index.name, dbTable.fullName(), index.column.Name)
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	// VIEWS
	for _, viewConfig := range schema.Views {
		columns := []*columnInfo{}
		names := []string{}

		for _, col := range viewConfig.Cols {
			name := col.GetInfo().Name
			names = append(names, name)
			columns = append(columns, dbTable.columnsMap[name])
		}

		colnames := strings.Join(names, "_")
		view := viewInfo{
			iType:   4,
			name:    fmt.Sprintf(`%v__%v_view`, dbTable.name, colnames),
			columns: names,
		}

		if len(columns) > 1 {
			view.column = &columnInfo{
				Name: fmt.Sprintf(`zz_%v`, colnames), IsVirtual: true,
				FieldType: "int32", Type: "int",
			}
			dbTable.columnsMap[view.column.Name] = view.column
		}

		// Si sólo es una columna, no es necesario autogenerar
		if len(columns) == 1 {
			view.column = columns[0]
			view.getValue = view.column.getValue
		} else if len(viewConfig.IntConcatRadix) > 0 {
			view.column.FieldType = "int64"
			view.column.Type = "bigint"
			// Si ha especificado un int64 radix entonces se puede concatener, maximo 2 columnas
			if len(columns) < 2 {
				panic(fmt.Sprintf(`La view "%v" de la tabla "%v" posee menos de 2 columnas para usar el IntConcatRadix`, dbTable.name, view.name))
			}

			radixes := append(viewConfig.IntConcatRadix, 0)

			if len(radixes) != len(view.columns) {
				panic(fmt.Sprintf(`The view "%v" in "%v" need to have %v radix arguments for the %v columns provided`,
					view.name, dbTable.name, len(view.columns)-1, len(view.columns)))
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
					valueI64 := convertToInt64(value) * Pow10Int64(radixesI64[i])
					values = append(values, value)
					sumValue += valueI64
				}
				fmt.Printf("Radix Sum Calculado %v | %v | %v\n", sumValue, values, radixesI64)
				return any(sumValue)
			}
			view.getValue = view.column.getValue
		} else {
			// Sino crea un hash de las columnas
			view.column.getValue = func(s *reflect.Value) any {
				values := []any{}
				for _, e := range columns {
					values = append(values, e.getValue(s))
				}
				return HashInt(values...)
			}
		}

		whereCols := append([]columnInfo{*view.column}, dbTable.keys...)
		pk := whereCols[0].Name + ", " + whereCols[1].Name

		if len(dbTable.partitionKey.Name) > 0 {
			whereCols = append([]columnInfo{dbTable.partitionKey}, whereCols...)
			pk = fmt.Sprintf("(%v), %v", whereCols[0].Name, pk)
		}

		whereColumnsNotNull := []string{}
		for _, col := range whereCols {
			if col.Type == "text" {
				whereColumnsNotNull = append(whereColumnsNotNull, col.Name+" > ''")
			} else if col.IsSlice {
				whereColumnsNotNull = append(whereColumnsNotNull, col.Name+" IS NOT NULL")
			} else {
				whereColumnsNotNull = append(whereColumnsNotNull, col.Name+" > 0")
			}
		}

		view.getCreateScript = func() string {
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

		dbTable.views[view.name] = view
	}

	for _, col := range dbTable.columnsMap {
		dbTable.columns = append(dbTable.columns, col)
	}

	return dbTable
}

type number1 interface {
	int | int32 | int8 | uint8 | int16 | uint16 | int64
}
type numfloat interface {
	float32 | float64
}

func setReflectInt[T number1, E number1](e *reflect.Value, vl *E, isPointer bool) {
	if isPointer && vl != nil {
		pv := T(*vl)
		(*e).Set(reflect.ValueOf(&pv))
	} else if !isPointer {
		(*e).SetInt(int64(*vl))
	}
}

func setReflectIntSlice[T number1, E number1](e *reflect.Value, vl *[]E, isPointer bool) {
	newSlice := []T{}
	for _, v := range *vl {
		newSlice = append(newSlice, T(v))
	}
	if isPointer {
		(*e).Set(reflect.ValueOf(&newSlice))
	} else {
		(*e).Set(reflect.ValueOf(newSlice))
	}
}

func setReflectFloat[T numfloat](e *reflect.Value, vl *T, isPointer bool) {
	if isPointer {
		pv := T(*vl)
		(*e).Set(reflect.ValueOf(&pv))
	} else {
		(*e).SetFloat(float64(*vl))
	}
}

var fieldMapping = map[string]func(e *reflect.Value, value any, isPointer bool){
	"string": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*string); ok {
			if ip {
				(*e).Set(reflect.ValueOf(vl))
			} else {
				(*e).SetString(*vl)
			}
		}
	},
	"int32": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*int); ok {
			setReflectInt[int32](e, vl, ip)
		}
	},
	"int": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*int); ok {
			setReflectInt[int](e, vl, ip)
		}
	},
	"int16": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*int16); ok {
			setReflectInt[int](e, vl, ip)
		} else if vl, ok := v.(*int); ok {
			setReflectInt[int](e, vl, ip)
		}
	},
	"int8": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*int8); ok {
			setReflectInt[int](e, vl, ip)
		} else if vl, ok := v.(*int); ok {
			setReflectInt[int](e, vl, ip)
		}
	},
	"int64": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*int64); ok {
			setReflectInt[int](e, vl, ip)
		}
	},
	"[]int32": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int32](e, vl, ip)
		}
	},
	"[]int16": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*[]int16); ok {
			setReflectIntSlice[int16](e, vl, ip)
		} else if vl, ok := v.(*[]int8); ok {
			setReflectIntSlice[int8](e, vl, ip)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int](e, vl, ip)
		}
	},
	"[]int8": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*[]int8); ok {
			setReflectIntSlice[int8](e, vl, ip)
		} else if vl, ok := v.(*[]int16); ok {
			setReflectIntSlice[int16](e, vl, ip)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int](e, vl, ip)
		}
	},
	"[]int64": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*[]int64); ok {
			setReflectIntSlice[int64](e, vl, ip)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int](e, vl, ip)
		}
	},
	"[]string": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*[]string); ok {
			if ip {
				(*e).Set(reflect.ValueOf(vl))
			} else {
				(*e).Set(reflect.ValueOf(*vl))
			}
		}
	},
	"float32": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*float32); ok {
			setReflectFloat(e, vl, ip)
		}
	},
	"float64": func(e *reflect.Value, v any, ip bool) {
		if vl, ok := v.(*float64); ok {
			setReflectFloat(e, vl, ip)
		}
	},
}
