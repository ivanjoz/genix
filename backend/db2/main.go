package db2

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/gocql/gocql"
	"github.com/kr/pretty"
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

type ColumnSetInfo interface {
	SetName(string)
	SetTableInfo(*TableInfo)
	SetSchemaStruct(any)
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

type TableInfo struct {
	statements       []ColumnStatement
	allowFilter      bool
	columnsToInclude []string
	isExcluding      bool
	columnsToExclude []string
}

type TableStructInterface[T any] interface {
	//Query(columns ...Coln) *T
	// AddStatement(statement ColumnStatement)
	Init() *T
	// GetRefSchema() T
	//GetTable() *TableInfo
	//	Exclude() *T
}

type TableStruct[T TableStructInterface[T], E any] struct {
	Ref          *TableStruct[T, E]
	schemaStruct *T
	tableInfo    *TableInfo
}

func (e *TableStruct[T, E]) SetName(t string) {}
func (e *TableStruct[T, E]) SetTableInfo(t *TableInfo) {
	e.tableInfo = t
}
func (e *TableStruct[T, E]) GetTableInfo() *TableInfo {
	return e.tableInfo
}
func (e *TableStruct[T, E]) SetSchemaStruct(schemaStruct any) {
	if schema, ok := schemaStruct.(*T); ok {
		e.schemaStruct = schema
	}
}

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func InitTable[T any](schemaStruct *T) *T {

	structValue := reflect.ValueOf(schemaStruct).Elem()
	structType := structValue.Type()
	refTableInfo := &TableInfo{}

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		// Check if field can be addressed and if it implements ColumnSetName interface
		if !field.CanAddr() || !field.Addr().CanInterface() {
			continue
		}

		fieldAddr := field.Addr()
		// Try to get the interface and check if it implements ColumnSetName
		colSetter, ok := fieldAddr.Interface().(ColumnSetInfo)
		if !ok {
			fmt.Println("El field", fieldType.Name, "no implementa ColumnSetInfo")
			continue
		} else {
			fmt.Println("Field seteado!", fieldType.Name, "|", fieldType.Type)
		}

		// Extract column name from db tag or convert field name to snake_case
		columnName := toSnakeCase(fieldType.Name)
		if tag := fieldType.Tag.Get("db"); tag != "" {
			columnName = tag
		}

		// Set the column name using the interface method
		colSetter.SetName(columnName)
		colSetter.SetTableInfo(refTableInfo)
		colSetter.SetSchemaStruct(schemaStruct)
	}
	fmt.Println("schemaStruct (1)", schemaStruct)
	return schemaStruct
}

func (e *TableStruct[T, E]) Query(columns ...Coln) *T {
	for _, col := range columns {
		e.tableInfo.columnsToInclude = append(e.tableInfo.columnsToInclude, col.GetName())
	}
	fmt.Println("Table info in query::", e.tableInfo)

	return e.schemaStruct
}

func (e *TableStruct[T, E]) GetRefSchema() *T {
	return e.schemaStruct
}

func (e *TableStruct[T, E]) QueryExclude(columns ...Coln) *T {
	e.Ref = &TableStruct[T, E]{}
	e.schemaStruct = new(T)
	// prepareTable(e.schemaStruct)
	// e.columnsToExclude = append(e.columnsToExclude, columns...)
	return e.schemaStruct
}

func Print(Struct any) {
	pretty.Println(Struct)
}

func (e *TableStruct[T, E]) Exec() {
	fmt.Println("Exec!!")
	fmt.Println(e.schemaStruct)
	for _, statement := range e.tableInfo.statements {
		fmt.Println("Statement:", statement)
	}
	Print(e.tableInfo)
}

func (e *TableStruct[T, E]) AllowFilter() *T {
	e.tableInfo.allowFilter = true
	return e.schemaStruct
}

func (e *TableStruct[T, E]) Exclude() *T {
	e.tableInfo.isExcluding = true
	return e.schemaStruct
}

func (e *TableStruct[T, E]) AddStatement(statement ColumnStatement) {
	e.tableInfo.statements = append(e.tableInfo.statements, statement)
}

type TableInterface[T any] interface {
	//Query(columns ...Coln) *T
	// AddStatement(statement ColumnStatement)
	GetSchema() TableSchema
	// GetRefSchema() T
	//GetTable() *TableInfo
	//	Exclude() *T
}

type Col[T TableInterface[T], E any] struct {
	Name         string
	schemaStruct *T
	tableInfo    *TableInfo
}

func (q Col[T, E]) GetInfo() columnInfo {
	col := columnInfo{Name: q.Name}
	if reflect.TypeFor[T]().Kind() == reflect.Interface {
		col.FieldType = "any"
	} else {
		col.FieldType = reflect.TypeFor[T]().Name()
	}
	return col
}

func (q Col[T, E]) GetName() string {
	return q.Name
}

func (c *Col[T, E]) SetName(name string) {
	c.Name = name
}

func (c *Col[T, E]) SetTableInfo(tableInfo *TableInfo) {
	c.tableInfo = tableInfo
}

func (c *Col[T, E]) SetSchemaStruct(schemaStruct any) {
	if schema, ok := schemaStruct.(*T); ok {
		c.schemaStruct = schema
	}
}

func (e *Col[T, E]) Exclude(v E) *T {
	return e.schemaStruct
}

func (e *Col[T, E]) Equals(v E) *T {
	fmt.Println("Equals", e)
	fmt.Println("schemaStruct", e.schemaStruct)
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.Name, Operator: "=", Value: any(v)})
	// (*e.schemaStruct).AddStatement(ColumnStatement{Col: e.Name, Operator: "=", Value: any(v)})
	return e.schemaStruct
}

func (e Col[T, E]) In(values_ ...E) *T {
	values := []any{}
	for _, v := range values_ {
		values = append(values, any(v))
	}
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.Name, Operator: "IN", Values: values})
	// (*e.schemaStruct).AddStatement(ColumnStatement{Col: e.Name, Operator: "IN", Values: values})
	return e.schemaStruct
}
func (e Col[T, E]) ConcurrentIn(values_ ...E) ColumnStatement {
	values := []any{}
	for _, v := range values_ {
		values = append(values, any(v))
	}
	return ColumnStatement{Col: e.Name, Operator: "CIN", Values: values}
}
func (e Col[T, E]) GreaterThan(v E) ColumnStatement {
	return ColumnStatement{Col: e.Name, Operator: ">", Value: any(v)}
}
func (e Col[T, E]) GreaterEqual(v E) ColumnStatement {
	return ColumnStatement{Col: e.Name, Operator: ">=", Value: any(v)}
}
func (e Col[T, E]) LessThan(v E) ColumnStatement {
	return ColumnStatement{Col: e.Name, Operator: "<", Value: any(v)}
}
func (e Col[T, E]) LessEqual(v E) ColumnStatement {
	return ColumnStatement{Col: e.Name, Operator: "<=", Value: any(v)}
}
func (e Col[T, E]) Between(v1 E, v2 E) ColumnStatement {
	return ColumnStatement{
		Col:      e.Name,
		Operator: "BETWEEN",
		From:     []ColumnStatement{{Col: e.Name, Value: v1}},
		To:       []ColumnStatement{{Col: e.Name, Value: v2}},
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

func (c *ColSlice[T]) SetName(name string) {
	c.Name = name
}

func (e ColSlice[T]) Contains(v T) ColumnStatement {
	return ColumnStatement{Col: e.Name, Operator: "CONTAINS", Value: any(v)}
}

type statementGroup struct {
	group []ColumnStatement
}

type TableSchemaInterface interface {
	GetSchema() TableSchema
}

type GetSchemaTest1[T TableSchemaInterface] struct {
}

func (e GetSchemaTest1[T]) GetSchema() TableSchema {
	h := any(new(T)).(TableSchemaInterface)
	return h.GetSchema()
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
