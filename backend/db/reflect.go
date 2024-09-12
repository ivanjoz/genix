package db

import (
	"fmt"
	"reflect"
	"strings"
)

type columnInfo struct {
	Name      string
	FieldType string
	//FieldName      string
	NameAlias string
	Type      string
	IsSlice   bool
	// RefType        reflect.Value
	MethodIdx      int
	IsPrimaryKey   int8
	IsPointer      bool
	IsViewExcluded bool
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

func makeTable[T TableSchemaInterface]() scyllaTable {
	newT := *new(T)
	schema := newT.GetTableSchema()

	dbTable := scyllaTable{
		name:         schema.Name,
		partitionKey: schema.Partition.GetInfo(),
		primaryKey:   schema.Partition.GetInfo(),
		columnsMap:   map[string]columnInfo{},
		indexes:      map[string]viewInfo{},
		views:        map[string]viewInfo{},
	}

	refTyp := reflect.ValueOf(newT)
	methodsPrefix := []string{"func() db.ColSlice[", "func() db.Col["}

	for i := 0; i < refTyp.NumMethod(); i++ {
		method := refTyp.Method(i)
		methodType := method.Type().String()
		founded := false
		for _, mp := range methodsPrefix {
			if strings.HasPrefix(methodType, mp) {
				founded = true
			}
		}
		if !method.IsValid() || !founded {
			continue
		}

		colBase := method.Call(nil)[0]
		if col, ok := colBase.Interface().(ColInfo); ok {
			columnInfo := col.GetInfo()
			fmt.Println("Column Name:", columnInfo.Name, "| Type:", columnInfo.FieldType, "| Is Slice:", columnInfo.IsSlice)

			columnInfo.getValue = func(rv *reflect.Value) any {
				if cb, ok := rv.Method(i).Call(nil)[0].Interface().(ColInfo); ok {
					return cb.GetValue()
				}
				return nil
			}
			columnInfo.Type = scyllaFieldToColumnTypesMap[columnInfo.FieldType]
			if columnInfo.Type == "" {
				columnInfo.IsComplexType = true
				columnInfo.Type = "blob"
			} else if columnInfo.IsSlice {
				columnInfo.Type = fmt.Sprintf("set<%v>", columnInfo.Type)
			}
			dbTable.columnsMap[columnInfo.Name] = columnInfo
			dbTable.columns = append(dbTable.columns, columnInfo)
		}
	}

	idxCount := int8(1)
	for _, column := range schema.GlobalIndexes {
		colInfo := dbTable.columnsMap[column.GetInfo().Name]
		index := viewInfo{
			iType:   1,
			name:    fmt.Sprintf(`%v__%v_index_g`, dbTable.name, colInfo.Name),
			idx:     idxCount,
			column:  colInfo,
			columns: []columnInfo{colInfo},
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
		idxCount++
		dbTable.indexes[index.name] = index
	}

	for _, column := range schema.LocalIndexes {
		colInfo := column.GetInfo()
		index := viewInfo{
			iType:   2,
			name:    fmt.Sprintf(`%v__%v_index_l`, dbTable.name, colInfo.Name),
			idx:     idxCount,
			columns: []columnInfo{dbTable.partitionKey, colInfo},
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
		idxCount++
		dbTable.indexes[index.name] = index
	}

	for _, indexColumns := range schema.HashIndexes {
		columns := []columnInfo{}
		names := []string{}

		for _, col := range indexColumns {
			name := col.GetInfo().Name
			names = append(names, name)
			columns = append(columns, dbTable.columnsMap[name])
		}
		colnames := strings.Join(names, "_")

		index := viewInfo{
			iType:         2,
			name:          fmt.Sprintf(`%v__%v_index`, dbTable.name, colnames),
			idx:           idxCount,
			columns:       columns,
			virtualColumn: fmt.Sprintf(`zz_%v`, colnames),
			getValue: func(s *reflect.Value) any {
				values := []string{}
				for _, e := range columns {
					values = append(values, fmt.Sprintf("%v", e.getValue(s)))
				}
				return BasicHashInt(strings.Join(values, "|"))
			},
		}
		index.getStatement = func(statements ...ColumnStatement) string {
			if len(statements) < 2 {
				panic(fmt.Sprintf("Error columna %v: El número de valores debe ser >= 2", index.name))
			}

			values := []string{}
			for _, e := range statements {
				values = append(values, fmt.Sprintf("%v", e.Value))
			}
			hashInt := BasicHashInt(strings.Join(values, "|"))
			// Revisar casuísica IN
			return fmt.Sprintf("%v %v %v", index.virtualColumn, statements[0].Operator, hashInt)
		}

		idxCount++
		dbTable.indexes[index.name] = index
	}

	for _, viewConfig := range schema.Views {
		columns := []columnInfo{}
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
			columns: columns,
		}

		dbTable.views[view.name] = view
	}

	return dbTable
}
