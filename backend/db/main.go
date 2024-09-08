package db

import (
	"reflect"
)

type Col[T any] struct {
	Name  string
	Value T
}

type IColumnStatement interface {
	GetValue() any
	GetName() string
}
type ColumnStatement struct {
	Column   string
	Operator string
	Value    any
}
type TableSchema struct {
	Name        string
	Indexes     []ColInfo
	HashIndexes [][]ColInfo
	Views       []TableView
}

func (q ColumnStatement) GetValue() any {
	return any(q.Value)
}

type ColInfo interface {
	GetInfo() columnInfo
}

type TableView struct {
	Cols []ColInfo
	// Para concatenar numeros como = int64(e.AlmacenID)*1e9 + int64(e.Updated)
	Int64ConcatRadix int8
}

func (q Col[T]) GetInfo() columnInfo {
	typ := *new(T)
	fieldType := reflect.TypeOf(typ).String()
	col := columnInfo{Name: q.Name}
	if fieldType[0:1] == "*" {
		col.IsPointer = true
		fieldType = fieldType[1:]
	}
	col.FieldType = fieldType
	return col
}

// Generic
func (e Col[T]) Equals(v T) ColumnStatement {
	return ColumnStatement{e.Name, "=", any(e.Value)}
}
func (e Col[T]) In(v ...T) ColumnStatement {
	return ColumnStatement{e.Name, "IN", any(e.Value)}
}
func (e Col[T]) GreaterThan(v T) ColumnStatement {
	return ColumnStatement{e.Name, ">", any(e.Value)}
}
func (e Col[T]) GreaterEqual(v T) ColumnStatement {
	return ColumnStatement{e.Name, ">=", any(e.Value)}
}
func (e Col[T]) LessThan(v T) ColumnStatement {
	return ColumnStatement{e.Name, "<", any(e.Value)}
}
func (e Col[T]) LessEqual(v T) ColumnStatement {
	return ColumnStatement{e.Name, "<=", any(e.Value)}
}

// Generic Array
type ColSlice[T any] struct {
	Name   string
	Values []T
}

func (q ColSlice[T]) GetInfo() columnInfo {
	typ := *new(T)
	return columnInfo{Name: q.Name, FieldType: reflect.TypeOf(typ).String(), IsSlice: true}
}

func (e ColSlice[T]) Contains(v T) ColumnStatement {
	return ColumnStatement{e.Name, "CONTAINS", any(e.Values)}
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
type CpInt = Col[*int]
type CpI32 = Col[*int32]
type CpI16 = Col[*int16]
type CpStr = Col[*string]
type CpI64 = Col[*int64]
type CpF32 = Col[*float32]
type CpF64 = Col[*float64]

type statementGroup struct {
	group []ColumnStatement
}

type TableSchemaInterface interface {
	GetTableSchema() TableSchema
}

type Query[T TableSchemaInterface] struct {
	T          T
	statements []statementGroup
}

func (q *Query[T]) init() {
	q.T = *new(T)
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
	Name      string
	Idx       int8
	Type      string
	KeyColumn string
	Columns   []ColInfo
	// Para concatenar numeros como = int64(e.AlmacenID)*1e9 + int64(e.Updated)
	Int64ConcatRadix int8
}

func (q *Query[T]) Exec() ([]T, error) {
	/*
		scyllaTable := makeTable[T]()
			viewTableName := scyllaTable.Name
			queryStr := "SELECT %v FROM %v"
			indexOperators := []string{"=", "IN"}

			statements := []ColumnStatement{}
			columnsWhere := []string{}
			for _, st := range q.statements {
				statements = append(statements, st.group[0])
				columnsWhere = append(columnsWhere, st.group[0].Column)
			}

			posibleViews := []viewInfo{}
			posibleHashIndexes := [][]ColInfo{}

			if len(statements) > 1 {
				// Revisa si puede usar una vista
				for _, view := range scyllaTable.views {
					isIncluded := true
					for _, col := range view.Columns {
						if slices.Contains(columnsWhere, col.GetInfo().Name) {
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
						for _, col := range index.Columns {
							if slices.Contains(columnsWhere, col.GetInfo().Name) {
								isIncluded = false
							}
						}
					}
				}
			}

			// Revisa si hay un view que satisfaga este request
			if len(posibleViews) > 0 {
				view := posibleViews[0]
				viewTableName = view.Name
			}
	*/

	return []T{}, nil
}

type scyllaTable struct {
	Name          string
	NameSingle    string
	PrimaryKey    string
	PartitionKey  string
	columns       []columnInfo
	columnsMap    map[string]columnInfo
	indexes       map[string]viewInfo
	views         map[string]viewInfo
	ViewsExcluded []string
}
