package db

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/gocql/gocql"
)

type scyllaTable[T any] struct {
	name          string
	keyspace      string
	keys          []*columnInfo
	partKey       *columnInfo
	keysIdx       []int16
	columns       []*columnInfo
	columnsMap    map[string]*columnInfo
	columnsIdxMap map[int16]*columnInfo
	indexes       map[string]*viewInfo
	views         map[string]*viewInfo
	indexViews    []*viewInfo
	ViewsExcluded []string
	_maxColIdx    int16
}

func (e scyllaTable[T]) GetFullName() string {
	return fmt.Sprintf("%v.%v", e.keyspace, e.name)
}
func (e scyllaTable[T]) GetColumns() map[string]*columnInfo {
	return e.columnsMap
}
func (e scyllaTable[T]) GetKeys() []*columnInfo {
	return e.keys
}
func (e scyllaTable[T]) GetPartKey() *columnInfo {
	return e.partKey
}

type IColumnStatement interface {
	GetValue() any
	GetName() string
}
type ColumnStatement struct {
	Col      string
	Operator string
	Value    any
	Values   []any
	From     []ColumnStatement
	To       []ColumnStatement
}

type TableSchema struct {
	Keyspace string
	// StructType    T
	Name           string
	Keys           []Coln
	Partition      Coln
	GlobalIndexes  []Coln
	LocalIndexes   []Coln
	HashIndexes    [][]Coln
	Views          []View
	SequenceColumn Coln
}

func (q ColumnStatement) GetValue() any {
	if len(q.Values) > 0 && q.Operator == "IN" {
		values := []string{}
		for _, v := range q.Values {
			if str, ok := v.(string); ok {
				values = append(values, `'`+str+`'`)
			} else {
				values = append(values, fmt.Sprintf("%v", v))
			}
		}
		return "(" + strings.Join(values, ", ") + ")"
	} else if str, ok := q.Value.(string); ok {
		return `'` + str + `'`
	} else {
		return q.Value
	}
}

type Coln interface {
	GetInfo() columnInfo
	GetName() string
}

type ColumnSetName interface {
	SetName(string)
}

type View struct {
	Cols []Coln
	// Para concatenar numeros como = int64(e.AlmacenID)*1e9 + int64(e.Updated)
	ConcatI64 []int8
	// Para concatenar numeros como = int32(e.AlmacenID)*1e5 + int64(e.Updated)
	ConcatI32 []int8
	// Keep the original table partition in the created view.
	// Example: key = (part_col) new_col, pk_col
	KeepPart bool
}

type Index struct {
	Cols []Coln
	// Crea un hash para todas las combinaciones
	HashAll bool
}

type Col[T any] struct {
	C string
}

func (q *Col[T]) SetName(name string) {
	q.C = name
}
func (q Col[T]) GetInfo() columnInfo {
	col := columnInfo{Name: q.C}
	if reflect.TypeFor[T]().Kind() == reflect.Interface {
		col.FieldType = "any"
	} else {
		col.FieldType = reflect.TypeFor[T]().Name()
	}
	return col
}

func (q Col[T]) GetName() string {
	return q.C
}

// Generic
func (e Col[T]) Equals(v T) ColumnStatement {
	return ColumnStatement{Col: e.C, Operator: "=", Value: any(v)}
}
func (e Col[T]) In(values_ ...T) ColumnStatement {
	values := []any{}
	for _, v := range values_ {
		values = append(values, any(v))
	}
	return ColumnStatement{Col: e.C, Operator: "IN", Values: values}
}
func (e Col[T]) ConcurrentIn(values_ ...T) ColumnStatement {
	values := []any{}
	for _, v := range values_ {
		values = append(values, any(v))
	}
	return ColumnStatement{Col: e.C, Operator: "CIN", Values: values}
}
func (e Col[T]) GreaterThan(v T) ColumnStatement {
	return ColumnStatement{Col: e.C, Operator: ">", Value: any(v)}
}
func (e Col[T]) GreaterEqual(v T) ColumnStatement {
	return ColumnStatement{Col: e.C, Operator: ">=", Value: any(v)}
}
func (e Col[T]) LessThan(v T) ColumnStatement {
	return ColumnStatement{Col: e.C, Operator: "<", Value: any(v)}
}
func (e Col[T]) LessEqual(v T) ColumnStatement {
	return ColumnStatement{Col: e.C, Operator: "<=", Value: any(v)}
}
func (e Col[T]) Between(v1 T, v2 T) ColumnStatement {
	return ColumnStatement{
		Col:      e.C,
		Operator: "BETWEEN",
		From:     []ColumnStatement{{Col: e.C, Value: v1}},
		To:       []ColumnStatement{{Col: e.C, Value: v2}},
	}
}

// Generic Array
type ColSlice[T any] struct {
	Name string
}

func (q ColSlice[T]) GetInfo() columnInfo {
	return columnInfo{Name: q.Name, FieldType: reflect.TypeFor[T]().String(), IsSlice: true}
}

func (q ColSlice[T]) GetName() string {
	return q.Name
}

func (e ColSlice[T]) Contains(v T) ColumnStatement {
	return ColumnStatement{Col: e.Name, Operator: "CONTAINS", Value: any(v)}
}

type CoAny = Col[any]
type CoInt = Col[int]
type CoI32 = Col[int32]
type CoI16 = Col[int16]
type CoI8 = Col[int8]
type CoStr = Col[string]
type CoI64 = Col[int64]
type CoF32 = Col[float32]
type CoF64 = Col[float64]
type CsInt = ColSlice[int]
type CsI32 = ColSlice[int32]
type CsI16 = ColSlice[int16]
type CsI8 = ColSlice[int8]
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
	between        ColumnStatement
	orderBy        string
	limit          int32
}

func (q *Query[T]) init() {
	q.T = *new(T)
}

func (q *Query[T]) Columns(columns ...Coln) *Query[T] {
	for _, col := range columns {
		q.columnsInclude = append(q.columnsInclude, col.GetInfo())
	}
	return q
}
func (q *Query[T]) Exclude(columns ...Coln) *Query[T] {
	for _, col := range columns {
		q.columnsExclude = append(q.columnsExclude, col.GetInfo())
	}
	return q
}

func (q *Query[T]) Where(statements ...ColumnStatement) *Query[T] {
	q.init()
	if statements[0].Operator == "BETWEEN" {
		st := statements[0]
		q.between.Operator = "BETWEEN"
		q.between.From = append(q.between.From, st.From...)
		q.between.To = append(q.between.To, st.To...)
	} else {
		q.statements = append(q.statements, statementGroup{
			group: statements,
		})
	}
	return q
}

func (q *Query[T]) OrderDescending() *Query[T] {
	q.orderBy = "ORDER BY %v DESC"
	return q
}

func (q *Query[T]) Limit(limit int32) *Query[T] {
	q.limit = limit
	return q
}

func (q *Query[T]) WhereIF(condition bool, ce ColumnStatement) *Query[T] {
	if condition {
		q.Where(ce)
	}
	return q
}

func (q *Query[T]) WhereOr(ce ...ColumnStatement) *Query[T] {
	q.init()
	q.statements = append(q.statements, statementGroup{group: ce})
	return q
}

type QueryBetweem[T any] struct {
	query *Query[T]
}

func (q QueryBetweem[T]) And(statements ...ColumnStatement) *Query[T] {
	fromColNames := ""
	for _, c := range q.query.between.From {
		fromColNames += (c.Col + "_")
	}
	toColNames := ""
	for _, c := range statements {
		toColNames += (c.Col + "_")
	}
	if fromColNames != toColNames {
		panic(fmt.Sprintf(`The "From" and "To" statements for the BETWEEN operators must contains the same columns in the same order. Getted "%v" vs "%v"`, fromColNames, toColNames))
	}
	q.query.between.To = statements
	return q.query
}

func (q *Query[T]) Between(statements ...ColumnStatement) QueryBetweem[T] {
	q.between.From = statements
	q.between.Operator = "BETWEEN"
	if len(statements) == 1 {
		q.between.Col = statements[0].Col
	}
	return QueryBetweem[T]{query: q}
}

type WithJoin[T any] struct {
	query         *Query[T]
	recordsToJoin []T
}

func (e *WithJoin[T]) Join(columns ...Coln) *Query[T] {
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
	Type          int8 /* 1 = Global index, 2 = Local index, 3 = Hash index, 4 = view*/
	name          string
	idx           int8
	column        *columnInfo
	columns       []string
	columnsNoPart []string
	columnsIdx    []int16
	Operators     []string
	//	getValue        func(s *reflect.Value) any
	getStatement    func(statements ...ColumnStatement) []string
	getCreateScript func() string
}

type QueryResult[T any] struct {
	Records []T
	Err     error
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

func MakeInsertStatement[T TableSchemaInterface](records *[]T, columnsToExclude ...Coln) []string {
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

	queryStrInsert := fmt.Sprintf(`INSERT INTO %v (%v) VALUES `,
		scyllaTable.GetFullName(), strings.Join(columnsNames, ", "))

	queryStatements := []string{}

	for _, rec := range *records {
		refValue := reflect.ValueOf(rec)
		// fmt.Println("Type:", reflect.TypeOf(rec).String())

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
	return queryStatements
}

func MakeInsertBatch[T TableSchemaInterface](records *[]T, columnsToExclude ...Coln) *gocql.Batch {
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
	columnPlaceholders := []string{}
	for _, col := range columns {
		columnsNames = append(columnsNames, col.Name)
		columnPlaceholders = append(columnPlaceholders, "?")
	}

	session := getScyllaConnection()
	batch := session.NewBatch(gocql.UnloggedBatch)

	queryStrInsert := fmt.Sprintf(`INSERT INTO %v (%v) VALUES (%v)`,
		scyllaTable.GetFullName(), strings.Join(columnsNames, ", "), strings.Join(columnPlaceholders, ", "))

	for _, rec := range *records {
		refValue := reflect.ValueOf(rec)
		values := []any{}

		for _, col := range columns {
			if col.getValue == nil {
				panic("is nil column: getValue() = " + col.Name + " | " + col.FieldName)
			}
			var value any
			if col.getStatementValue != nil {
				value = col.getStatementValue(&refValue)
			} else {
				value = col.getValue(&refValue)
			}
			values = append(values, value)
		}

		fmt.Println("VALUES::")
		fmt.Println(values)
		batch.Query(queryStrInsert, values...)
	}
	return batch
}

func Insert[T TableSchemaInterface](records *[]T, columnsToExclude ...Coln) error {

	session := getScyllaConnection()
	fmt.Println("BATCH (1)::")
	queryBatch := MakeInsertBatch(records, columnsToExclude...)

	fmt.Println("BATCH (2)::")
	fmt.Println(queryBatch.Entries)

	if err := session.ExecuteBatch(queryBatch); err != nil {
		fmt.Println("Error inserting records:", err)
		return err
	}

	return nil
}

func makeUpdateStatementsBase[T TableSchemaInterface](records *[]T, columnsToInclude []Coln, columnsToExclude []Coln, onlyVirtual bool) []string {

	scyllaTable := makeTable(*new(T))
	columnsToUpdate := []*columnInfo{}

	if len(columnsToInclude) > 0 {
		for _, col_ := range columnsToInclude {
			col := scyllaTable.columnsMap[col_.GetName()]
			if slices.Contains(scyllaTable.keysIdx, col.Idx) {
				msg := fmt.Sprintf(`Table "%v": The column "%v" can't be updated because is part of primary key.`, scyllaTable.name, col.Name)
				panic(msg)
			}
			columnsToUpdate = append(columnsToUpdate, col)
		}
	} else {
		columnsToExcludeNames := []string{}
		for _, c := range columnsToExclude {
			columnsToExcludeNames = append(columnsToExcludeNames, c.GetName())
		}
		for _, col := range scyllaTable.columns {
			isExcluded := slices.Contains(columnsToExcludeNames, col.Name)
			if !col.IsVirtual && !isExcluded && !slices.Contains(scyllaTable.keysIdx, col.Idx) {
				columnsToUpdate = append(columnsToUpdate, col)
			}
		}
	}

	columnsIdx := []int16{}
	for _, col := range columnsToUpdate {
		columnsIdx = append(columnsIdx, col.Idx)
	}

	//Revisa si hay columnas que deben actualizarse juntas para los índices calculados
	for _, indexViews := range scyllaTable.indexViews {
		if indexViews.column.IsVirtual {
			includedCols := []int16{}
			notIncludedCols := []int16{}
			for _, colIdx := range indexViews.columnsIdx {
				if slices.Contains(columnsIdx, colIdx) || slices.Contains(scyllaTable.keysIdx, colIdx) {
					includedCols = append(includedCols, colIdx)
				} else {
					notIncludedCols = append(notIncludedCols, colIdx)
				}
			}
			if len(includedCols) > 0 && len(notIncludedCols) > 0 {
				colNames := strings.Join(indexViews.columns, `, `)
				includedColsNames := []string{}
				for _, idx := range notIncludedCols {
					includedColsNames = append(includedColsNames, scyllaTable.columnsIdxMap[idx].Name)
				}

				msg := fmt.Sprintf(`Table "%v": A composit index/view requires the columns %v are updated together. Included: %v`, scyllaTable.name, colNames, strings.Join(includedColsNames, ", "))
				panic(msg)
			} else if len(includedCols) > 0 {
				columnsToUpdate = append(columnsToUpdate, indexViews.column)
			}
		}
	}

	if onlyVirtual {
		cols := columnsToUpdate
		columnsToUpdate = nil
		for _, col := range cols {
			if col.IsVirtual {
				columnsToUpdate = append(columnsToUpdate, col)
			}
		}
	}

	columnsWhere := scyllaTable.keys

	if scyllaTable.partKey != nil {
		columnsWhere = append([]*columnInfo{scyllaTable.partKey}, columnsWhere...)
	}

	queryStatements := []string{}

	for _, rec := range *records {
		refValue := reflect.ValueOf(rec)

		setStatements := []string{}
		for _, col := range columnsToUpdate {
			v := col.getValue(&refValue)
			setStatements = append(setStatements, fmt.Sprintf(`%v = %v`, col.Name, v))
		}

		whereStatements := []string{}
		for _, col := range columnsWhere {
			v := col.getValue(&refValue)
			whereStatements = append(whereStatements, fmt.Sprintf(`%v = %v`, col.Name, v))
		}

		queryStatement := fmt.Sprintf(
			"UPDATE %v SET %v WHERE %v",
			scyllaTable.GetFullName(), Concatx(", ", setStatements), Concatx(" and ", whereStatements),
		)

		queryStatements = append(queryStatements, queryStatement)
	}

	return queryStatements
}

func MakeUpdateStatements[T TableSchemaInterface](records *[]T, columnsToInclude ...Coln) []string {
	return makeUpdateStatementsBase(records, columnsToInclude, nil, false)
}

func Update[T TableSchemaInterface](records *[]T, columnsToInclude ...Coln) error {

	if len(columnsToInclude) == 0 {
		panic("No se incluyeron columnas a actualizar.")
	}

	queryStatements := makeUpdateStatementsBase(records, columnsToInclude, nil, false)
	queryUpdate := makeQueryStatement(queryStatements)
	fmt.Println(queryUpdate)

	if err := QueryExec(queryUpdate); err != nil {
		fmt.Println("Error updating records:", err)
		return err
	}
	return nil
}

func UpdateExclude[T TableSchemaInterface](records *[]T, columnsToExclude ...Coln) error {

	queryStatements := makeUpdateStatementsBase(records, nil, columnsToExclude, false)
	queryInsert := makeQueryStatement(queryStatements)
	if err := QueryExec(queryInsert); err != nil {
		fmt.Println(queryInsert)
		fmt.Println("Error inserting records:", err)
		return err
	}
	return nil
}

func InsertOrUpdate[T TableSchemaInterface](
	records *[]T,
	isRecordForInsert func(e *T) bool,
	columnsToExcludeUpdate []Coln,
	columnsToExcludeInsert ...Coln,
) error {

	recordsToInsert := []T{}
	recordsToUpdate := []T{}

	for _, e := range *records {
		if isRecordForInsert(&e) {
			recordsToInsert = append(recordsToInsert, e)
		} else {
			recordsToUpdate = append(recordsToUpdate, e)
		}
	}

	if len(recordsToUpdate) > 0 {
		fmt.Println("Registros a actualizar:", len(recordsToUpdate))
		err := UpdateExclude(&recordsToUpdate, columnsToExcludeUpdate...)
		if err != nil {
			return err
		}
	}

	if len(recordsToInsert) > 0 {
		fmt.Println("Registros a insertar:", len(recordsToInsert))

		err := Insert(&recordsToInsert, columnsToExcludeInsert...)
		if err != nil {
			return err
		}
	}

	return nil
}
