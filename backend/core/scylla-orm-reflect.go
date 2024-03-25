package core

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

var ScyllaFieldToColumnTypesMap = map[string]string{
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

type ScyllaView struct {
	Name       string
	Idx        int8
	Type       string
	FieldType  string
	ColumnName string
}

type IGetView interface {
	GetView(view int8) any
}

func MakeScyllaTable[T any](newType T) ScyllaTable {

	typeName := fmt.Sprintf("%v", reflect.TypeOf(newType))

	if scyllaTable, ok := TypeToScyllaTableMap[typeName]; ok {
		return scyllaTable
	}

	table := ScyllaTable{
		Indexes:    map[string]BDIndex{},
		ColumnsMap: map[string]BDColumn{},
		Views:      map[string]ScyllaView{},
	}

	viewsCombinedMap := map[int8][]string{}

	schemaTypes := reflect.ValueOf(newType).Type()
	indexNameToColumns := map[string][]string{}
	primaryKeyCount := int8(1)

	for i := 0; i < schemaTypes.NumField(); i++ {
		f := schemaTypes.Field(i)

		if _, ok := f.Tag.Lookup("table"); ok {
			tagString := schemaTypes.Field(i).Tag.Get("table")
			values := strings.Split(tagString, ",")
			table.NameSingle = values[0]
			if strings.Contains(table.NameSingle, ".") {
				table.Name = table.NameSingle
			} else {
				table.Name = Env.DB_NAME + "." + table.NameSingle
			}
		}

		if _, ok := f.Tag.Lookup("db"); ok {
			tagString := schemaTypes.Field(i).Tag.Get("db")
			tagValues := strings.Split(tagString, ",")

			column := BDColumn{
				FieldIdx:  i,
				FieldType: f.Type.String(),
				FieldName: f.Name,
				Name:      tagValues[0],
			}

			if column.FieldType[:1] == "*" {
				column.FieldType = column.FieldType[1:]
				column.IsPointer = true
			}

			if len(tagValues) == 2 && tagValues[1] == "counter" {
				column.Type = "counter"
			} else if column.FieldType[0:2] == "[]" {
				ft := column.FieldType[2:]
				if dbType, ok := ScyllaFieldToColumnTypesMap[ft]; ok {
					Log("type encontrado::", ft, dbType)
					column.Type = fmt.Sprintf("set<%v>", dbType)
				} else {
					column.IsComplexType = true
				}
			} else if _, ok := ScyllaFieldToColumnTypesMap[column.FieldType]; ok {
				column.Type = ScyllaFieldToColumnTypesMap[column.FieldType]
			} else {
				column.IsComplexType = true
			}

			if column.IsPointer {
				column.RefType = reflect.New(f.Type.Elem())
			} else {
				column.RefType = reflect.New(f.Type) // .Elem()
			}

			if column.IsComplexType {
				column.Type = "blob"
			}

			for i, v := range tagValues {
				if i == 0 {
					continue
				}
				if v == "pk" {
					column.IsPrimaryKey = primaryKeyCount
					primaryKeyCount++
					if len(table.PrimaryKey) > 0 && len(table.PartitionKey) == 0 {
						table.PartitionKey = table.PrimaryKey
					}
					table.PrimaryKey = column.Name
				} else if v == "view" {
					table.Views[column.Name] = ScyllaView{
						Name:       fmt.Sprintf(`%v__%v_view`, table.Name, column.Name),
						ColumnName: column.Name,
					}
					column.HasView = true
				} else if len(v) > 5 && v[0:5] == "view." {
					viewIdx := int8(SrtToInt(v[5:]))
					if viewIdx == 0 {
						panic(fmt.Sprintf(`No se reconoció el número de la view "%v"`, v))
					}
					viewsCombinedMap[viewIdx] = append(viewsCombinedMap[viewIdx], column.Name)
				} else if v == "exclude" {
					table.ViewsExcluded = append(table.ViewsExcluded, column.Name)
					column.IsViewExcluded = true
				} else if len(v) == 3 && v[:2] == "zx" {
					sl := indexNameToColumns[v]
					indexNameToColumns[v] = append(sl, column.Name)
				}
			}

			table.Columns = append(table.Columns, column)
		}
	}

	for idx, columns := range viewsCombinedMap {
		if len(columns) == 1 {
			panic(fmt.Sprintf(`La view "view.%v" en la columna "%v" necesita otra columna para ser combinada. Si sólo es necesario una columna colocar sólo "view"`, idx, columns[0]))
		}
		cnames := strings.Join(columns, "_")
		view := ScyllaView{
			Idx:        idx,
			Name:       fmt.Sprintf(`%v__%v_view`, table.Name, cnames),
			ColumnName: fmt.Sprintf(`zv_%v`, cnames),
		}

		if baseI, ok := any(new(T)).(IGetView); ok {
			va := baseI.GetView(view.Idx)
			view.FieldType = fmt.Sprintf("%T", va)
			if scyllaType, ok := ScyllaFieldToColumnTypesMap[view.FieldType]; ok {
				view.Type = scyllaType
			} else {
				panic(fmt.Sprintf(`El type "%v" en "view.%v" no se pudo convertir al type de Scylla.`, view.FieldType, view.Idx))
			}
		} else {
			panic(fmt.Sprintf(`No se encontró el método "GetView(view int8) any" que debería ser implementado como el siguiente ejemplo: 
				func (e *Struct) GetView(view int8) any {
					if view == %v {
						return e.param1*100 + int32(e.param2)
					}
				}
			`, idx))
		}

		table.Views[cnames] = view
		table.ViewsExcluded = append(table.ViewsExcluded, view.ColumnName)

		table.Columns = append(table.Columns, BDColumn{
			Name:           view.ColumnName,
			FieldIdx:       -1,
			Type:           view.Type,
			FieldType:      view.FieldType,
			IsViewExcluded: true,
			ViewIdx:        view.Idx,
		})
	}

	for _, e := range table.Columns {
		table.ColumnsMap[e.Name] = e
	}

	for indexName, columnNames := range indexNameToColumns {
		sort.Strings(columnNames)
		columns := []BDColumn{}
		columnsPosition := map[string]int{}

		for i, cn := range columnNames {
			columns = append(columns, table.ColumnsMap[cn])
			columnsPosition[cn] = i
		}

		if len(columns) == 1 {
			Log("El índice compuesto:", indexName, "contiene una sola columna:", columns[0])
			continue
		}

		table.Indexes[strings.Join(columnNames, "+")] = BDIndex{
			Name:       indexName,
			ColumnsPos: columnsPosition,
			MakeIntHash: func(refValue reflect.Value) int32 {
				values := []string{}
				for _, col := range columns {
					var v any
					field := refValue.Field(col.FieldIdx)
					if col.IsPointer {
						if !field.IsNil() {
							v = field.Elem().Interface()
						}
					} else {
						v = field.Interface()
					}
					if v != nil {
						values = append(values, strings.ToLower(fmt.Sprintf(`%v`, v)))
					}
				}
				if len(values) == 0 {
					return 0
				}
				hashInt := BasicHashInt(strings.Join(values, "+"))
				Log("Creando Hash:: ", strings.Join(values, "+"), hashInt)
				return hashInt
			},
			MakeIntHashFomValues: func(columnNames, values []string) int32 {
				sortedValues := make([]string, len(columnNames))
				for i, columnName := range columnNames {
					pos := columnsPosition[columnName]
					sortedValues[pos] = values[i]
				}
				hashInt := BasicHashInt(strings.ToLower(strings.Join(sortedValues, "+")))
				Log("Creando Hash:: ", strings.Join(sortedValues, "+"), hashInt)
				return hashInt
			},
		}
	}

	TypeToScyllaTableMap[typeName] = table
	return table
}

func setReflectInt[T Number1, E Number1](e *reflect.Value, vl *E, isPointer bool) {
	if isPointer && vl != nil {
		pv := T(*vl)
		(*e).Set(reflect.ValueOf(&pv))
	} else if !isPointer {
		(*e).SetInt(int64(*vl))
	}
}

func setReflectIntSlice[T Number1, E Number1](e *reflect.Value, vl *[]E, isPointer bool) {
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

func setReflectFloat[T NumFloat](e *reflect.Value, vl *T, isPointer bool) {
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
