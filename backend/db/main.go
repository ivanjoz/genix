package db

import (
	"app/core"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

type Col[T any] struct {
	Name  string
	Value T
}

type ColPoint[T any] struct {
	Name  string
	Value *T
}

type IColumnStatement interface {
	GetValue() any
	GetName() string
}
type ColumnStatement struct {
	Column   string
	Operator string
	Value    any
	Values   []any
}
type TableSchema[T any] struct {
	Keyspace      string
	StructType    T
	Name          string
	PrimaryKey    Column
	Partition     Column
	GlobalIndexes []Column
	LocalIndexes  []Column
	HashIndexes   [][]Column
	Views         []TableView
}

func (q ColumnStatement) GetValue() any {
	if len(q.Values) > 0 {
		return ""
	} else if str, ok := q.Value.(string); ok {
		return `'` + str + `'`
	} else {
		return q.Value
	}
}

type Column interface {
	GetInfo() columnInfo
	GetValue() any
}

type ColumnSetName interface {
	SetName(string)
}

type TableView struct {
	Cols []Column
	// Para concatenar numeros como = int64(e.AlmacenID)*1e9 + int64(e.Updated)
	Int64ConcatRadix int8
}

func (q *Col[T]) SetName(name string) {
	q.Name = name
}
func (q Col[T]) GetInfo() columnInfo {
	typ := *new(T)
	fieldType := reflect.TypeOf(typ).String()
	col := columnInfo{Name: q.Name, FieldType: fieldType}
	return col
}
func (q Col[T]) GetValue() any {
	return any(q.Value)
}
func (q ColPoint[T]) GetInfo() columnInfo {
	typ := *new(T)
	fieldType := reflect.TypeOf(typ).String()
	col := columnInfo{Name: q.Name, IsPointer: true, FieldType: fieldType}
	return col
}
func (q ColPoint[T]) GetValue() any {
	return any(*q.Value)
}

// Generic
func (e Col[T]) Equals(v T) ColumnStatement {
	return ColumnStatement{e.Name, "=", any(v), nil}
}
func (e Col[T]) In(values_ ...T) ColumnStatement {
	values := []any{}
	for _, v := range values_ {
		values = append(values, any(v))
	}
	return ColumnStatement{e.Name, "IN", any(*new(T)), values}
}
func (e Col[T]) GreaterThan(v T) ColumnStatement {
	return ColumnStatement{e.Name, ">", any(v), nil}
}
func (e Col[T]) GreaterEqual(v T) ColumnStatement {
	return ColumnStatement{e.Name, ">=", any(v), nil}
}
func (e Col[T]) LessThan(v T) ColumnStatement {
	return ColumnStatement{e.Name, "<", any(v), nil}
}
func (e Col[T]) LessEqual(v T) ColumnStatement {
	return ColumnStatement{e.Name, "<=", any(v), nil}
}

// Generic Array
type ColSlice[T any] struct {
	Name   string
	Values []T
}

func (q ColSlice[T]) GetValue() any {
	return any(q.Values)
}

func (q ColSlice[T]) GetInfo() columnInfo {
	typ := *new(T)
	return columnInfo{Name: q.Name, FieldType: reflect.TypeOf(typ).String(), IsSlice: true}
}

func (e ColSlice[T]) Contains(v T) ColumnStatement {
	return ColumnStatement{e.Name, "CONTAINS", any(v), nil}
}

type CoAny = Col[any]
type CoInt = Col[int]
type CoI32 = Col[int32]
type CoI16 = Col[int16]
type CoStr = Col[string]
type CoI64 = Col[int64]
type CoF32 = Col[float32]
type CoF64 = Col[float64]
type CsInt = ColSlice[int]
type CsI32 = ColSlice[int32]
type CsI16 = ColSlice[int16]
type CsStr = ColSlice[string]
type CpInt = ColPoint[int]
type CpI32 = ColPoint[int32]
type CpI16 = ColPoint[int16]
type CpStr = ColPoint[string]
type CpI64 = ColPoint[int64]
type CpF32 = ColPoint[float32]
type CpF64 = ColPoint[float64]

type statementGroup struct {
	group []ColumnStatement
}

type TableSchemaInterface[T any] interface {
	GetSchema() TableSchema[T]
}

type Query[T any] struct {
	T              T
	statements     []statementGroup
	columnsInclude []Column
	columnsExclude []Column
	recordsToJoin  []T
	columnsJoin    []Column
}

func (q *Query[T]) init() {
	q.T = *new(T)
}

func (q *Query[T]) Columns(columns ...Column) *Query[T] {
	q.columnsInclude = append(q.columnsInclude, columns...)
	return q
}
func (q *Query[T]) Exclude(columns ...Column) *Query[T] {
	q.columnsExclude = append(q.columnsExclude, columns...)
	return q
}

func (q *Query[T]) Where(ce ColumnStatement) *Query[T] {
	q.init()
	q.statements = append(q.statements, statementGroup{
		group: []ColumnStatement{ce},
	})
	return q
}

func (q *Query[T]) WhereOr(ce ...ColumnStatement) *Query[T] {
	q.init()
	q.statements = append(q.statements, statementGroup{group: ce})
	return q
}

type WithJoin[T any] struct {
	query         *Query[T]
	recordsToJoin []T
}

func (e *WithJoin[T]) Join(columns ...Column) *Query[T] {
	e.query.recordsToJoin = e.recordsToJoin
	e.query.columnsJoin = columns
	return e.query
}

func (q *Query[T]) With(records ...T) *WithJoin[T] {
	q.init()
	wj := WithJoin[T]{query: q, recordsToJoin: records}
	return &wj
}

type viewInfo struct {
	iType   int8 /* 1 = Global index, 2 = Local index, 3 = Hash index, 4 = view*/
	name    string
	idx     int8
	column  columnInfo
	columns []string
	// Para concatenar numeros como = int64(e.AlmacenID)*1e9 + int64(e.Updated)
	int64ConcatRadix int8
	getValue         func(s *reflect.Value) any
	getStatement     func(statements ...ColumnStatement) string
	getCreateScript  func() string
}

type QueryResult[T any] struct {
	Records []T
	Error   error
}

func QuerySelect[T TableSchemaInterface[E], E any](handler func(query *Query[E], schemaTable T)) QueryResult[E] {

	query := Query[E]{}
	scyllaTable := MakeTable[T]()
	handler(&query, scyllaTable.baseType)
	records, err := selectExec[E, T](&query)

	return QueryResult[E]{records, err}
}

func selectExec[T any, E TableSchemaInterface[T]](query *Query[T]) ([]T, error) {

	scyllaTable := MakeTable[E]()
	viewTableName := scyllaTable.name

	if len(scyllaTable.keyspace) == 0 {
		scyllaTable.keyspace = connParams.Keyspace
	}
	if len(scyllaTable.keyspace) == 0 {
		return nil, errors.New("no se ha especificado un keyspace")
	}

	queryStr := "SELECT * FROM %v.%v WHERE %v"
	indexOperators := []string{"=", "IN"}

	statements := []ColumnStatement{}
	columnsWhere := []string{}
	for _, st := range query.statements {
		statements = append(statements, st.group[0])
		columnsWhere = append(columnsWhere, st.group[0].Column)
	}

	posibleViewsOrIndexes := []viewInfo{}

	if len(statements) > 1 {
		// Revisa si puede usar una vista
		for _, view := range scyllaTable.views {
			isIncluded := true
			for _, col := range view.columns {
				if slices.Contains(columnsWhere, col) {
					isIncluded = false
				}
			}
			if isIncluded {
				posibleViewsOrIndexes = append(posibleViewsOrIndexes, view)
			}
		}

		// Revisa si puede user un índice compuesto (sólo para operadores IN y =)
		findComposeIndex := true
		for _, st := range statements {
			if !slices.Contains(indexOperators, st.Operator) {
				findComposeIndex = false
			}
		}

		if findComposeIndex {
			for _, index := range scyllaTable.indexes {
				isIncluded := true
				for _, col := range index.columns {
					if slices.Contains(columnsWhere, col) {
						isIncluded = false
					}
				}
				if isIncluded {
					posibleViewsOrIndexes = append(posibleViewsOrIndexes, index)
				}
			}
		}
	}

	var statementsRemain []ColumnStatement
	whereStatements := []string{}

	fmt.Println("posible index::", len(posibleViewsOrIndexes))

	// Revisa si hay un view que satisfaga este request
	if len(posibleViewsOrIndexes) > 0 {
		//TODO: aquí debe escoger el mejor índice o view en caso existan 2
		viewOrIndex := posibleViewsOrIndexes[0]

		// Revisa qué columnas satisfacen el índice
		statementsSelected := []ColumnStatement{}
		for _, st := range statements {
			if slices.Contains(viewOrIndex.columns, st.Column) {
				statementsSelected = append(statementsSelected, st)
			} else {
				statementsRemain = append(statementsRemain, st)
			}
		}
		fmt.Println("hola???")
		whereStatements = append(whereStatements, viewOrIndex.getStatement(statementsSelected...))
	} else {
		statementsRemain = statements
	}

	// fmt.Println("where statements::", whereStatements)

	for _, st := range statementsRemain {
		fmt.Println("column 1::", st.Column)
		fmt.Println("column 2::", st.Operator)
		fmt.Println("column 3::", st.GetValue())

		where := fmt.Sprintf("%v %v %v", st.Column, st.Operator, st.GetValue())
		// fmt.Println("where::", where)
		whereStatements = append(whereStatements, where)
	}

	whereStatementsConcat := strings.Join(whereStatements, " AND ")
	queryStr = fmt.Sprintf(queryStr, scyllaTable.keyspace, viewTableName, whereStatementsConcat)

	fmt.Println("query string::", queryStr)

	iter := getScyllaConnection().Query(queryStr).Iter()
	rd, err := iter.RowData()
	if err != nil {
		fmt.Println("Error on RowData::", err)
		return nil, err
	}

	scanner := iter.Scanner()
	records := []T{}

	fmt.Println("starting iterator")
	fmt.Println("columns::", len(scyllaTable.columns))

	for scanner.Next() {

		rowValues := rd.Values

		err := scanner.Scan(rowValues...)
		if err != nil {
			fmt.Println("Error on scan::", err)
			return nil, err
		}

		rec := new(T)
		ref := reflect.ValueOf(rec).Elem()

		for idx, column := range scyllaTable.columns {
			value := rowValues[idx]
			if value == nil {
				continue
			}
			if mapField, ok := fieldMapping[column.FieldType]; ok {
				field := ref.Field(column.FieldIdx)
				val := reflect.ValueOf(value)
				if val.Kind() == reflect.Ptr {
					// Dereference the pointer to get the underlying value
					val = val.Elem()
				}
				fmt.Printf("Mapeando valor | F: %v N: %v C: %v | %v\n", field, column.Name, column.FieldType, val)
				mapField(&field, value, column.IsPointer)
				// Revisa si necesita parsearse un string a un struct como JSON
			} else if column.IsComplexType {
				// Log("complex type::", column.FieldName)
				if vl, ok := value.(*string); ok {
					newStruct := column.RefType.Interface()
					err := json.Unmarshal([]byte(*vl), newStruct)
					if err != nil {
						fmt.Println("Error al convertir: ", newStruct, *vl, err.Error())
					}
					if column.IsPointer {
						ref.Field(column.FieldIdx).Set(reflect.ValueOf(newStruct))
					} else {
						ref.Field(column.FieldIdx).Set(reflect.ValueOf(newStruct).Elem())
					}
				} else if vl, ok := value.(*[]uint8); ok {
					if len(*vl) <= 2 {
						continue
					}
					newStruct := column.RefType.Interface()
					err = cbor.Unmarshal(*vl, &newStruct)
					if err != nil {
						fmt.Println("Error al convertir: ", newStruct, *vl, err.Error())
					}
					if column.IsPointer {
						ref.Field(column.FieldIdx).Set(reflect.ValueOf(newStruct))
					} else {
						ref.Field(column.FieldIdx).Set(reflect.ValueOf(newStruct).Elem())
					}
				}
			} else {
				fmt.Print("Column is not mapped:: ", column)
			}
		}
		//Print(rec)
		records = append(records, *rec)
	}

	core.Print(records)

	return records, nil
}

type scyllaTable[T any] struct {
	baseType      T
	name          string
	keyspace      string
	primaryKey    columnInfo
	partitionKey  columnInfo
	columns       []columnInfo
	columnsMap    map[string]columnInfo
	indexes       map[string]viewInfo
	views         map[string]viewInfo
	ViewsExcluded []string
}