package db

import (
	"fmt"
	"reflect"
	"slices"
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
type TableSchema struct {
	Name          string
	Partition     ColInfo
	GlobalIndexes []ColInfo
	LocalIndexes  []ColInfo
	HashIndexes   [][]ColInfo
	Views         []TableView
}

func (q ColumnStatement) GetValue() any {
	return any(q.Value)
}

type ColInfo interface {
	GetInfo() columnInfo
	GetValue() any
}

type TableView struct {
	Cols []ColInfo
	// Para concatenar numeros como = int64(e.AlmacenID)*1e9 + int64(e.Updated)
	Int64ConcatRadix int8
}

func (q Col[T]) GetInfo() columnInfo {
	typ := *new(T)
	fieldType := reflect.TypeOf(typ).String()
	col := columnInfo{Name: q.Name, IsPointer: true, FieldType: fieldType}
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

type TableSchemaInterface interface {
	GetTableSchema() TableSchema
}

type Query[T TableSchemaInterface] struct {
	T              T
	statements     []statementGroup
	columnsInclude []ColInfo
	columnsExclude []ColInfo
}

func (q *Query[T]) init() {
	q.T = *new(T)
}

func (q *Query[T]) Columns(columns ...ColInfo) *Query[T] {
	q.columnsInclude = append(q.columnsInclude, columns...)
	return q
}
func (q *Query[T]) Exclude(columns ...ColInfo) *Query[T] {
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

type viewInfo struct {
	iType         int8 /* 1 = Global index, 2 = Local index, 3 = Hash index, 4 = view*/
	name          string
	idx           int8
	virtualColumn string
	column        columnInfo
	columns       []columnInfo
	// Para concatenar numeros como = int64(e.AlmacenID)*1e9 + int64(e.Updated)
	int64ConcatRadix int8
	getValue         func(s *reflect.Value) any
	getStatement     func(statements ...ColumnStatement) string
}

type QueryResult struct {
	Records any
	Error   error
}

func QuerySelect[T TableSchemaInterface](handler func(query *Query[T], e *T)) QueryResult {

	query := Query[T]{}
	handler(&query, &query.T)
	records, err := query.Exec()

	return QueryResult{records, err}
}

func (q *Query[T]) Exec() ([]T, error) {

	scyllaTable := makeTable[T]()
	viewTableName := scyllaTable.name
	queryStr := "SELECT %v FROM %v"
	indexOperators := []string{"=", "IN"}

	statements := []ColumnStatement{}
	columnsWhere := []string{}
	for _, st := range q.statements {
		statements = append(statements, st.group[0])
		columnsWhere = append(columnsWhere, st.group[0].Column)
	}

	posibleViews := []viewInfo{}
	posibleIndexes := []viewInfo{}

	if len(statements) > 1 {
		// Revisa si puede usar una vista
		for _, view := range scyllaTable.views {
			isIncluded := true
			for _, col := range view.columns {
				if slices.Contains(columnsWhere, col.Name) {
					isIncluded = false
				}
			}
			if isIncluded {
				posibleViews = append(posibleViews, view)
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
					if slices.Contains(columnsWhere, col.Name) {
						isIncluded = false
					}
				}
				if isIncluded {
					posibleIndexes = append(posibleIndexes, index)
				}
			}
		}
	}

	// Revisa si hay un view que satisfaga este request
	if len(posibleViews) > 0 {
		view := posibleViews[0]
		viewTableName = view.name
	} else if len(posibleIndexes) > 0 {
		index := posibleIndexes[0]
		viewTableName = index.name
	}

	queryStr = fmt.Sprintf(queryStr, "", viewTableName)

	return []T{}, nil
}

type scyllaTable struct {
	name          string
	nameSingle    string
	primaryKey    columnInfo
	partitionKey  columnInfo
	columns       []columnInfo
	columnsMap    map[string]columnInfo
	indexes       map[string]viewInfo
	views         map[string]viewInfo
	ViewsExcluded []string
}
