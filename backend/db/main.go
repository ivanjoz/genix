package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

type scyllaTable[T any] struct {
	baseType      T
	name          string
	keyspace      string
	keys          []columnInfo
	partitionKey  columnInfo
	columns       []*columnInfo
	columnsMap    map[string]*columnInfo
	columnsIdxMap map[int16]*columnInfo
	indexes       map[string]*viewInfo
	views         map[string]*viewInfo
	indexViews    []*viewInfo
	ViewsExcluded []string
	_maxColIdx    int16
}

func (e scyllaTable[T]) fullName() string {
	return fmt.Sprintf("%v.%v", e.keyspace, e.name)
}

type Col[T any] struct {
	C string
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
	Keyspace string
	// StructType    T
	Name          string
	Keys          []Column
	Partition     Column
	GlobalIndexes []Column
	LocalIndexes  []Column
	HashIndexes   [][]Column
	Views         []View
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
}

type ColumnSetName interface {
	SetName(string)
}

type View struct {
	Cols []Column
	// Para concatenar numeros como = int64(e.AlmacenID)*1e9 + int64(e.Updated)
	IntConcatRadix []int8
}

type Index struct {
	Cols []Column
	// Crea un hash para todas las combinaciones
	HashAll bool
}

func (q *Col[T]) SetName(name string) {
	q.C = name
}
func (q Col[T]) GetInfo() columnInfo {
	typ := *new(T)
	fieldType := reflect.TypeOf(typ).String()
	col := columnInfo{Name: q.C, FieldType: fieldType}
	return col
}

// Generic
func (e Col[T]) Equals(v T) ColumnStatement {
	return ColumnStatement{e.C, "=", any(v), nil}
}
func (e Col[T]) In(values_ ...T) ColumnStatement {
	values := []any{}
	for _, v := range values_ {
		values = append(values, any(v))
	}
	return ColumnStatement{e.C, "IN", any(*new(T)), values}
}
func (e Col[T]) GreaterThan(v T) ColumnStatement {
	return ColumnStatement{e.C, ">", any(v), nil}
}
func (e Col[T]) GreaterEqual(v T) ColumnStatement {
	return ColumnStatement{e.C, ">=", any(v), nil}
}
func (e Col[T]) LessThan(v T) ColumnStatement {
	return ColumnStatement{e.C, "<", any(v), nil}
}
func (e Col[T]) LessEqual(v T) ColumnStatement {
	return ColumnStatement{e.C, "<=", any(v), nil}
}

// Generic Array
type ColSlice[T any] struct {
	Name string
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

type statementGroup struct {
	group []ColumnStatement
}

type TableSchemaInterface interface {
	GetSchema() TableSchema
}

type Query[T any] struct {
	T              T
	statements     []statementGroup
	columnsInclude []columnInfo
	columnsExclude []columnInfo
	recordsToJoin  []T
	columnsJoin    []columnInfo
}

func (q *Query[T]) init() {
	q.T = *new(T)
}

func (q *Query[T]) Columns(columns ...Column) *Query[T] {
	for _, col := range columns {
		q.columnsInclude = append(q.columnsInclude, col.GetInfo())
	}
	return q
}
func (q *Query[T]) Exclude(columns ...Column) *Query[T] {
	for _, col := range columns {
		q.columnsExclude = append(q.columnsExclude, col.GetInfo())
	}
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
	for _, col := range columns {
		e.query.columnsJoin = append(e.query.columnsJoin, col.GetInfo())
	}
	return e.query
}

func (q *Query[T]) With(records ...T) *WithJoin[T] {
	q.init()
	wj := WithJoin[T]{query: q, recordsToJoin: records}
	return &wj
}

type viewInfo struct {
	Type       int8 /* 1 = Global index, 2 = Local index, 3 = Hash index, 4 = view*/
	name       string
	idx        int8
	column     *columnInfo
	columns    []string
	columnsIdx []int16
	//	getValue        func(s *reflect.Value) any
	getStatement    func(statements ...ColumnStatement) string
	getCreateScript func() string
}

type QueryResult[T any] struct {
	Records []T
	Error   error
}

func Select[T TableSchemaInterface](handler func(query *Query[T], schemaTable T)) QueryResult[T] {

	query := Query[T]{}
	baseType := *new(T)
	handler(&query, baseType)
	records, err := selectExec(&query)

	return QueryResult[T]{records, err}
}

type posibleIndex struct {
	indexView    *viewInfo
	colsIncluded []int16
	colsMissing  []int16
	priority     int8
}

func selectExec[T TableSchemaInterface](query *Query[T]) ([]T, error) {

	scyllaTable := makeTable[T](*new(T))
	viewTableName := scyllaTable.name

	if len(scyllaTable.keyspace) == 0 {
		scyllaTable.keyspace = connParams.Keyspace
	}
	if len(scyllaTable.keyspace) == 0 {
		return nil, errors.New("no se ha especificado un keyspace")
	}

	columnNames := []string{}
	if len(query.columnsInclude) > 0 {
		for _, col := range query.columnsInclude {
			columnNames = append(columnNames, col.Name)
		}
	} else {
		columnsExclude := []string{}
		for _, col := range query.columnsExclude {
			columnsExclude = append(columnsExclude, col.Name)
		}
		for _, col := range scyllaTable.columns {
			if !slices.Contains(columnsExclude, col.Name) && !col.IsVirtual {
				columnNames = append(columnNames, col.Name)
			}
		}
	}

	queryStr := fmt.Sprintf("SELECT %v ", strings.Join(columnNames, ", ")) +
		"FROM %v.%v WHERE %v"
	// indexOperators := []string{"=", "IN"}

	statements := []ColumnStatement{}
	columnsWhere := []string{}
	colsWhereIdx := []int16{}

	for _, st := range query.statements {
		col := scyllaTable.columnsMap[st.group[0].Column]
		colsWhereIdx = append(colsWhereIdx, col.Idx)
		statements = append(statements, st.group[0])
		columnsWhere = append(columnsWhere, st.group[0].Column)
	}

	posibleViewsOrIndexes := []posibleIndex{}

	if len(statements) > 0 {
		// Revisa si puede usar una vista
		for _, indview := range scyllaTable.indexViews {
			psb := posibleIndex{indexView: indview}
			for _, idx := range indview.columnsIdx {
				if slices.Contains(colsWhereIdx, idx) {
					psb.colsIncluded = append(psb.colsIncluded, idx)
				} else {
					psb.colsMissing = append(psb.colsMissing, idx)
				}
			}
			if len(psb.colsIncluded) > 0 {
				if len(psb.colsIncluded) == len(colsWhereIdx) {
					psb.priority = 6 + int8(len(colsWhereIdx))*2
				}
				psb.priority -= int8(len(psb.colsMissing)) * 2

				posibleViewsOrIndexes = append(posibleViewsOrIndexes, psb)
			}
		}
	}

	var statementsRemain []ColumnStatement
	whereStatements := []string{}

	fmt.Println("posible indexes::", len(posibleViewsOrIndexes))

	// Revisa si hay un view que satisfaga este request
	if len(posibleViewsOrIndexes) > 0 {
		//TODO: aquí debe escoger el mejor índice o view en caso existan 2
		slices.SortFunc(posibleViewsOrIndexes, func(a, b posibleIndex) int {
			return int(b.priority - a.priority)
		})

		viewOrIndex := posibleViewsOrIndexes[0].indexView
		fmt.Println("Posible Index:", viewOrIndex.name)

		if viewOrIndex.getStatement != nil {
			fmt.Println("Creating statement Index:", viewOrIndex.name)
			// Revisa qué columnas satisfacen el índice
			statementsSelected := []ColumnStatement{}
			for _, st := range statements {
				if slices.Contains(viewOrIndex.columns, st.Column) {
					statementsSelected = append(statementsSelected, st)
				} else {
					statementsRemain = append(statementsRemain, st)
				}
			}
			whereStatements = append(whereStatements, viewOrIndex.getStatement(statementsSelected...))
		} else {
			// No es necesario cambiar nada del query
			statementsRemain = statements
		}
	} else {
		statementsRemain = statements
	}

	// fmt.Println("where statements::", whereStatements)

	for _, st := range statementsRemain {
		/*
			fmt.Println("column 1::", st.Column)
			fmt.Println("column 2::", st.Operator)
			fmt.Println("column 3::", st.GetValue())
		*/
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
		if strings.Contains(err.Error(), "use ALLOW FILTERING") {
			fmt.Println("Posible Indexes or Views::")
			for _, e := range posibleViewsOrIndexes {
				typeName := indexTypes[e.indexView.Type]
				colnames := strings.Join(e.indexView.columns, ", ")
				msg := fmt.Sprintf(`%v (%v) %v`+"\n", typeName, e.indexView.column.Type, colnames)
				fmt.Println(msg)
			}
		}
		return nil, err
	}

	scanner := iter.Scanner()
	records := []T{}

	fmt.Println("starting iterator | columns::", len(scyllaTable.columns))

	for scanner.Next() {

		rowValues := rd.Values

		err := scanner.Scan(rowValues...)
		if err != nil {
			fmt.Println("Error on scan::", err)
			return nil, err
		}

		rec := new(T)
		ref := reflect.ValueOf(rec).Elem()

		for idx, colname := range columnNames {
			column := scyllaTable.columnsMap[colname]
			value := rowValues[idx]
			if value == nil {
				continue
			}
			if mapField, ok := fieldMapping[column.FieldType]; ok {
				field := ref.Field(column.FieldIdx)
				/*
					val := reflect.ValueOf(value)
					if val.Kind() == reflect.Ptr {
						// Dereference the pointer to get the underlying value
						val = val.Elem()
					}
					fmt.Printf("Mapeando valor | F: %v N: %v C: %v | %v\n", column.FieldName, column.Name, column.FieldType, val)
				*/
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
		records = append(records, *rec)
	}

	// core.Print(records)

	return records, nil
}

func parseValueToString(v any) string {
	if str, ok := v.(string); ok {
		return `'` + str + `'`
	} else {
		return fmt.Sprintf("%v", v)
	}
}

func makeQueryStatement(statements []string) string {
	queryStr := ""
	if len(statements) == 1 {
		queryStr = statements[0]
	} else {
		statements := strings.Join(statements, ";\n") + ";"
		queryStr = fmt.Sprintf("BEGIN BATCH\n%v\nAPPLY BATCH;", statements)
	}
	return queryStr
}

func InsertExclude[T TableSchemaInterface](records *[]T, columnsToExclude ...Column) error {

	scyllaTable := makeTable(*new(T))

	columns := []*columnInfo{}
	if len(columnsToExclude) > 0 {
		columsToExcludeNames := []string{}
		for _, e := range columnsToExclude {
			columsToExcludeNames = append(columsToExcludeNames, e.GetInfo().Name)
		}
		for _, col := range scyllaTable.columns {
			if !slices.Contains(columsToExcludeNames, col.Name) {
				columns = append(columns, col)
			}
		}
	} else {
		columns = scyllaTable.columns
	}

	columnsNames := []string{}
	for _, col := range columns {
		columnsNames = append(columnsNames, col.Name)
	}
	/*
		virtualIndexes := []viewInfo{}
		for _, index := range scyllaTable.indexes {
			if index.column.IsVirtual {
				virtualIndexes = append(virtualIndexes, index)
			}
		}
	*/

	queryStrInsert := fmt.Sprintf(`INSERT INTO %v (%v) VALUES `,
		scyllaTable.fullName(), strings.Join(columnsNames, ", "))

	queryStatements := []string{}

	for _, rec := range *records {
		refValue := reflect.ValueOf(rec)
		fmt.Println("Type:", reflect.TypeOf(rec).String())

		recordInsertValues := []string{}

		for _, col := range columns {
			if col.getValue == nil {
				panic("is nil column: getValue() = " + col.Name + " | " + col.FieldName)
			}
			value := col.getValue(&refValue)
			recordInsertValues = append(recordInsertValues, fmt.Sprintf("%v", value))
		}

		statement := /*" " +*/ queryStrInsert + "(" + strings.Join(recordInsertValues, ", ") + ")"
		queryStatements = append(queryStatements, statement)
	}

	queryInsert := makeQueryStatement(queryStatements)
	if err := QueryExec(queryInsert); err != nil {
		fmt.Println(queryInsert)
		fmt.Println("Error inserting records:", err)
		return err
	}

	return nil
}
