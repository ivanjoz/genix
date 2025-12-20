package db2

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/xunsafe"
)

type ScyllaTable[T any] struct {
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

func (e ScyllaTable[T]) GetFullName() string {
	return fmt.Sprintf("%v.%v", e.keyspace, e.name)
}
func (e ScyllaTable[T]) GetColumns() map[string]*columnInfo {
	return e.columnsMap
}
func (e ScyllaTable[T]) GetKeys() []*columnInfo {
	return e.keys
}
func (e ScyllaTable[T]) GetPartKey() *columnInfo {
	return e.partKey
}

type viewInfo struct {
	/* 1 = Global index, 2 = Local index, 3 = Hash index, 4 = view*/
	Type            int8
	name            string
	idx             int8
	column          *columnInfo
	columns         []string
	columnsNoPart   []string
	columnsIdx      []int16
	Operators       []string
	getStatement    func(statements ...ColumnStatement) []string
	getCreateScript func() string
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
	statements     []ColumnStatement
	columnsInclude []columnInfo
	columnsExclude []columnInfo
	between        ColumnStatement
	orderBy        string
	limit          int32
	allowFilter    bool
	refSlice       any /* referencia al slice de resultados */
}

// Interfaces
type TableSchemaInterface[T any] interface {
	GetSchema() TableSchema
}

type TableBaseInterface[T any, E any] interface {
	GetBaseStruct() E
	GetTableStruct() T
}

type TableStructInterfaceQuery[T any, E any] interface {
	SetRefSlice(*[]E)
}

type TableInterface[T any] interface {
	GetSchema() TableSchema
	GetTableStruct() T
}

type ColGetInfoPointer interface {
	GetInfoPointer() *columnInfo
	SetSchemaStruct(any)
	SetTableInfo(*TableInfo)
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

type TableQueryInterface[T any] interface {
	GetSchema() TableSchema
	SetWhere(string, string, any)
	Limit(int32) *T
	AllowFilter() *T
	Exec() error
}

type TableDeployInterface interface {
	MakeTableSchema() TableSchema
	MakeScyllaTable() ScyllaTable[any]
}

// TableStruct
type TableStruct[T TableSchemaInterface[T], E TableBaseInterface[T, E]] struct {
	schemaStruct *T
	tableInfo    *TableInfo
}

// GobEncode implements gob.GobEncoder interface
// TableStruct is only used for queries and doesn't need to be encoded
func (e TableStruct[T, E]) GobEncode() ([]byte, error) {
	return []byte{}, nil
}

// GobDecode implements gob.GobDecoder interface
// TableStruct is only used for queries and doesn't need to be decoded
func (e *TableStruct[T, E]) GobDecode(data []byte) error {
	return nil
}

func (e *TableStruct[T, E]) MakeTableSchema() TableSchema {
	return MakeSchema[E]()
}
func (e *TableStruct[T, E]) MakeScyllaTable() ScyllaTable[any] {
	return makeTable(new(T))
}

func (e *TableStruct[T, E]) SetWhere(colname string, operator string, value any) {
	cs := ColumnStatement{Col: colname, Operator: operator, Value: value}
	e.tableInfo.statements = append(e.tableInfo.statements, cs)
}

func (e *TableStruct[T, E]) SetTableInfo(t *TableInfo) {
	e.tableInfo = t
}
func (e *TableStruct[T, E]) SetRefSlice(refSlice *[]E) {
	e.tableInfo.refSlice = refSlice
}
func (e *TableStruct[T, E]) GetTableInfo() *TableInfo {
	return e.tableInfo
}
func (e *TableStruct[T, E]) GetInfoPointer() *columnInfo { // Para compatibilidad
	return &columnInfo{}
}
func (e *TableStruct[T, E]) SetSchemaStruct(schemaStruct any) {
	if schema, ok := schemaStruct.(*T); ok {
		e.schemaStruct = schema
	}
}
func (e TableStruct[T, E]) GetBaseStruct() E {
	return *new(E)
}
func (e TableStruct[T, E]) GetTableStruct() T {
	return *new(T)
}
func (e *TableStruct[T, E]) Select(columns ...Coln) *T {
	for _, col := range columns {
		e.tableInfo.columnsInclude = append(e.tableInfo.columnsInclude, col.GetInfo())
	}
	return e.schemaStruct
}

func (e *TableStruct[T, E]) Exclude(columns ...Coln) *T {
	for _, col := range columns {
		e.tableInfo.columnsExclude = append(e.tableInfo.columnsExclude, col.GetInfo())
	}
	return e.schemaStruct
}

func (e *TableStruct[T, E]) GetRefSchema() *T {
	return e.schemaStruct
}

func (e *TableStruct[T, E]) Exec() error {
	return execQuery[T, E](e.schemaStruct, e.tableInfo)
}

func (e *TableStruct[T, E]) AllowFilter() *T {
	e.tableInfo.allowFilter = true
	return e.schemaStruct
}

func (e *TableStruct[T, E]) Limit(limit int32) *T {
	e.tableInfo.limit = limit
	return e.schemaStruct
}

func (e *TableStruct[T, E]) OrderDesc() *T {
	e.tableInfo.orderBy = "ORDER BY %v DESC"
	return e.schemaStruct
}

func (e *TableStruct[T, E]) Update(records *[]E, columnsToInclude ...Coln) error {
	return Update(records, columnsToInclude...)
}

func (e *TableStruct[T, E]) UpdateOne(record E, columnsToInclude ...Coln) error {
	return UpdateOne(record, columnsToInclude...)
}

func (e *TableStruct[T, E]) UpdateExclude(records *[]E, columnsToExclude ...Coln) error {
	return UpdateExclude(records, columnsToExclude...)
}

func (e *TableStruct[T, E]) Insert(records *[]E, columnsToExclude ...Coln) error {
	return Insert(records, columnsToExclude...)
}

func (e *TableStruct[T, E]) InsertOne(record E, columnsToExclude ...Coln) error {
	return InsertOne(record, columnsToExclude...)
}

// Col and ColSlice
type Col[T TableInterface[T], E any] struct {
	info         columnInfo
	schemaStruct *T
	tableInfo    *TableInfo
}

func (q Col[T, E]) GetInfo() columnInfo {
	return q.info
}

func (q *Col[T, E]) GetInfoPointer() *columnInfo {
	return &q.info
}

func (q Col[T, E]) GetName() string {
	return q.info.Name
}

func (c *Col[T, E]) SetTableInfo(tableInfo *TableInfo) {
	c.tableInfo = tableInfo
}

func (c *Col[T, E]) SetSchemaStruct(schemaStruct any) {
	// fmt.Println("seteando schemaStruct", c.info.Name)
	if schema, ok := schemaStruct.(*T); ok {
		c.schemaStruct = schema
	} else {
		fmt.Println("no seteado!!")
	}
}

func (e *Col[T, E]) Exclude(v E) *T {
	return e.schemaStruct
}

func (e *Col[T, E]) Equals(v E) *T {
	// fmt.Println("e.schemaStruct", e.schemaStruct)
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: "=", Value: any(v)})
	return e.schemaStruct
}

func (e *Col[T, E]) In(values_ ...E) *T {
	values := []any{}
	for _, v := range values_ {
		values = append(values, any(v))
	}
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: "IN", Values: values})
	return e.schemaStruct
}

func (e *Col[T, E]) GreaterThan(v E) *T {
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: ">", Value: any(v)})
	return e.schemaStruct
}

func (e *Col[T, E]) GreaterEqual(v E) *T {
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: ">=", Value: any(v)})
	return e.schemaStruct
}

func (e *Col[T, E]) LessThan(v E) *T {
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: "<", Value: any(v)})
	return e.schemaStruct
}

func (e *Col[T, E]) LessEqual(v E) *T {
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: "<=", Value: any(v)})
	return e.schemaStruct
}

func (e *Col[T, E]) Between(v1 E, v2 E) *T {
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{
		Col:      e.info.Name,
		Operator: "BETWEEN",
		From:     []ColumnStatement{{Col: e.info.Name, Value: v1}},
		To:       []ColumnStatement{{Col: e.info.Name, Value: v2}},
	})
	return e.schemaStruct
}

type ColSlice[T TableInterface[T], E any] struct {
	info         columnInfo
	schemaStruct *T
	tableInfo    *TableInfo
}

func (q ColSlice[T, E]) GetInfo() columnInfo {
	q.info.IsSlice = true
	return q.info
}

func (q *ColSlice[T, E]) GetInfoPointer() *columnInfo {
	q.info.IsSlice = true
	return &q.info
}

func (q ColSlice[T, E]) GetName() string {
	return q.info.Name
}

func (c *ColSlice[T, E]) SetTableInfo(tableInfo *TableInfo) {
	c.tableInfo = tableInfo
}

func (c *ColSlice[T, E]) SetSchemaStruct(schemaStruct any) {
	if schema, ok := schemaStruct.(*T); ok {
		c.schemaStruct = schema
	}
}

func (e *ColSlice[T, E]) Contains(v E) *T {
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: "CONTAINS", Value: any(v)})
	return e.schemaStruct
}

func Query[T TableBaseInterface[E, T], E TableSchemaInterface[E]](refSlice *[]T) *E {
	refTable := initStructTable[E, T](new(E))
	any(refTable).(TableStructInterfaceQuery[E, T]).SetRefSlice(refSlice)
	return refTable
}

func MakeScyllaTable[T TableBaseInterface[E, T], E TableSchemaInterface[E]]() ScyllaTable[any] {
	refTable := initStructTable[E, T](new(E))
	return makeTable(refTable)
}

func MakeSchema[T TableBaseInterface[E, T], E TableSchemaInterface[E]]() TableSchema {
	refTable := initStructTable[E, T](new(E))
	return (*refTable).GetSchema()
}

func initStructTable[T any, E any](schemaStruct *T) *T {
	fmt.Println("making table...")
	structRefValue := reflect.ValueOf(*new(E))
	structRefType := structRefValue.Type()

	fieldNameIdxMap := map[string]*columnInfo{}
	refTableInfo := &TableInfo{}

	for i := 0; i < structRefType.NumField(); i++ {
		field := structRefType.Field(i)
		xfield := xunsafe.FieldByName(structRefType, field.Name)
		col := &columnInfo{
			FieldIdx:  i,
			FieldType: field.Type.String(),
			FieldName: field.Name,
			RefType:   field.Type,
			Field:     xfield,
		}
		if DebugFull {
			offset := uintptr(0)
			if xfield != nil {
				offset = xfield.Offset
			}
			fmt.Printf("Base Struct Field: %-20s | Type: %-15s | Offset: %d\n", field.Name, field.Type.String(), offset)
		}

		if col.FieldType[0:1] == "*" {
			col.IsPointer = true
			col.FieldType = col.FieldType[1:]
		}
		if col.FieldType[0:2] == "[]" {
			col.IsSlice = true
			col.FieldType = col.FieldType[2:]
		}

		// fmt.Println("fieldtype obtenido::", col.FieldName, col.FieldType)
		// fmt.Println("Fieldname::", col.FieldName, "| Type:", col.FieldType)
		fieldNameIdxMap[col.FieldName] = col
	}

	structValue := reflect.ValueOf(schemaStruct).Elem()
	structType := structValue.Type()

	// fmt.Println("fieldNameIdxMap...", len(fieldNameIdxMap), "| nf:", structValue.NumField())

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		// Check if field can be addressed and if it implements ColGetInfoPointer interface
		if !field.CanAddr() || !field.Addr().CanInterface() {
			fmt.Println("no es::", fieldType.Name)
			continue
		}

		fieldAddr := field.Addr()
		// Try to get the interface and check if it implements ColGetInfoPointer
		column, ok := fieldAddr.Interface().(ColGetInfoPointer)
		if !ok {
			fmt.Println("El field", fieldType.Name, "no implementa ColumnSetInfo")
			continue
		} else {
			// fmt.Println("Field seteado!", fieldType.Name, "|", fieldType.Type)
		}

		// Extract column name from db tag or convert field name to snake_case
		columnName := toSnakeCase(fieldType.Name)
		if tag := fieldType.Tag.Get("db"); tag != "" {
			columnName = tag
		}

		if colBase, ok := fieldNameIdxMap[fieldType.Name]; ok {
			colInfo := column.GetInfoPointer()
			colInfo.Name = columnName
			colInfo.FieldIdx = colBase.FieldIdx
			colInfo.FieldType = colBase.FieldType
			colInfo.FieldName = colBase.FieldName
			colInfo.RefType = colBase.RefType
			colInfo.Field = colBase.Field
			if DebugFull {
				fmt.Printf("Init Col: %s, Field: %s, Offset: %d\n", colInfo.Name, colInfo.FieldName, colInfo.Field.Offset)
			}
		} else if fieldType.Name != "TableStruct" {
			err := fmt.Sprintf(`No se encontró el field "%v" en el struct "%v"`, fieldType.Name, structRefType.Name())
			panic(err)
		}

		// Set the column name using the interface method
		column.SetSchemaStruct(schemaStruct)
		column.SetTableInfo(refTableInfo)
	}
	// fmt.Println("schemaStruct (1)", schemaStruct)
	return schemaStruct
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

// execQuery executes a query based on TableInfo and returns records
func execQuery[T TableSchemaInterface[T], E any](schemaStruct *T, tableInfo *TableInfo) error {
	records := (tableInfo.refSlice).(*[]E)
	scyllaTable := makeTable(schemaStruct)
	return selectExec(records, tableInfo, scyllaTable)
}
