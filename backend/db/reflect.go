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
	NameAlias      string
	Type           string
	IsSlice        bool
	RefType        reflect.Value
	MethodIdx      int
	IsPrimaryKey   int8
	IsPointer      bool
	IsViewExcluded bool
	HasView        bool
	IsComplexType  bool
	ViewIdx        int8
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

func makeTable[T any]() scyllaTable {
	dbTable := scyllaTable{
		// Indexes:    map[string]BDIndex{},
		columnsMap: map[string]columnInfo{},
		// Views:      map[string]ScyllaView{},
	}
	refTyp := reflect.ValueOf(*new(T))
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
			columnInfo.Type = scyllaFieldToColumnTypesMap[columnInfo.FieldType]
			if columnInfo.Type == "" {
				columnInfo.IsComplexType = true
			} else if columnInfo.IsSlice {
				columnInfo.Type = fmt.Sprintf("set<%v>", columnInfo.Type)
			}
			dbTable.columnsMap[columnInfo.Name] = columnInfo
		}
	}

	if e, ok := refTyp.Interface().(TableSchemaInterface); ok {
		schema := e.GetTableSchema()
		dbTable.Name = schema.Name
		// dbTable.Name = e.GetTable()
	}

	return dbTable
}
