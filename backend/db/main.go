package db

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/viant/xunsafe"
)

var indexTypes = map[int8]string{
	1: "Global Index",
	2: "Local Index",
	3: "Hash Index",
	4: "View",
}

type IColInfo interface {
	GetName() string
	GetValue(ptr unsafe.Pointer) any
	GetRawValue(ptr unsafe.Pointer) any
	GetStatementValue(ptr unsafe.Pointer) any
	SetValue(ptr unsafe.Pointer, v any)
	GetInfo() *colInfo
	GetType() *colType
	IsNil() bool
	// SetAutoincrementRandSize sets the random suffix size for autoincrement columns
	SetAutoincrementRandSize(size int8)
	// SetDecimalSize sets the decimal size for KeyIntPacking columns
	SetDecimalSize(size int8)
}

type ScyllaTable[T any] struct {
	name            string
	keyspace        string
	keys            []IColInfo
	partKey         IColInfo
	keysIdx         []int16
	columns         []IColInfo
	columnsMap      map[string]IColInfo
	columnsIdxMap   map[int16]IColInfo
	indexes         map[string]*viewInfo
	views           map[string]*viewInfo
	indexViews      []*viewInfo
	ViewsExcluded   []string
	useSequences    bool
	sequencePartCol IColInfo
	keyConcatenated []IColInfo
	keyIntPacking   []IColInfo
	// packedIndexes stores metadata for packed indexes declared in schema (local and global).
	packedIndexes     []*packedIndexInfo
	autoincrementPart IColInfo
	autoincrementCol  IColInfo
	capabilities      []QueryCapability
	// Composite bucket metadata is used to materialize virtual hash sets and plan range+contains reads.
	compositeBucketIndexes []compositeBucketIndex
	_maxColIdx             int16
}

// compositeBucketIndex stores the source columns and generated virtual bucket columns for one HashIndexes entry.
type compositeBucketIndex struct {
	name          string
	sourceColumns []IColInfo
	bucketColumn  IColInfo
	// bucketIsWeek keeps schema-level week semantics for custom week-code arithmetic in bucketing and range planning.
	bucketIsWeek         bool
	bucketSizes          []int8
	virtualColumnsBySize map[int8]IColInfo
}

func (e ScyllaTable[T]) GetFullName() string {
	return fmt.Sprintf("%v.%v", e.keyspace, e.name)
}
func (e ScyllaTable[T]) GetName() string {
	return e.name
}
func (e ScyllaTable[T]) GetColumns() map[string]IColInfo {
	return e.columnsMap
}
func (e ScyllaTable[T]) GetKeys() []IColInfo {
	return e.keys
}
func (e ScyllaTable[T]) GetPartKey() IColInfo {
	return e.partKey
}

type viewInfo struct {
	/* 1 = Global index, 2 = Local index, 3 = Hash index, 4 = view*/
	Type          int8
	name          string
	idx           int8
	column        IColInfo
	columns       []string
	columnsNoPart []string
	columnsIdx    []int16
	Operators     []string
	// RequiresPostFilter indicates the index/view can overfetch and should be exact-filtered in memory.
	// This is required for packed indexes when DecimalSize() truncation is applied.
	RequiresPostFilter bool
	getStatement       func(statements ...ColumnStatement) []string
	getCreateScript    func() string
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
	Name                    string
	Keys                    []Coln
	Partition               Coln
	GlobalIndexesDeprecated []Coln
	LocalIndexes            []Coln
	HashIndexes             [][]Coln
	Indexes                 [][]Coln //  new column
	GlobalIndexes           [][]Coln //  new column
	ViewsDeprecated         []View
	SequenceColumn          Coln
	CounterColumn           Coln
	UseSequences            bool
	SequencePartCol         Coln
	KeyConcatenated         []Coln
	KeyIntPacking           []Coln
	AutoincrementPart       Coln
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
	// Create a hash for use with IN operators
	UseHash bool
	// Columns to project, all by default
	Project []Coln
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
	GetTableStruct() T
}

type TableBaseInterface[T any, E any] interface {
	GetBaseStruct() E
	GetTableStruct() T
}

type TableBaseInterfaceWithCounter[T any, E any] interface {
	TableBaseInterface[T, E]
	GetCounter(increment int, partValue any, secondPartValue ...any) (int64, error)
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
	// field just for encoding purposes
	I__ bool `gob:"-" json:"-"`
}

func (e TableStruct[T, E]) GetSchema() TableSchema {
	return TableSchema{}
}

func (e *TableStruct[T, E]) MakeTableSchema() TableSchema {
	return MakeSchema[E]()
}

func (e *TableStruct[T, E]) MakeScyllaTable() ScyllaTable[any] {
	return makeTable(initStructTable[T, E](new(T)))
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

// The randDecimalSize is a random value appended to the autoincrement to avoid take the same value in high concurrent scenarios. If = 3, means ID = 100 now is 100567
func (e *TableStruct[T, E]) Autoincrement(randDecimalSize int8) Col[T, E] {
	if randDecimalSize > 8 {
		panic("randDecimalSize TOO BIG.")
	}
	return Col[T, E]{info: columnInfo{autoincrementRandSize: randDecimalSize}}
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
	// fmt.Printf("DEBUG: Col.GetInfo called for field=%s, Name=%s\n", q.info.FieldName, q.info.Name)
	if q.info.Type == 0 {
		typeOf := reflect.TypeOf((*E)(nil)).Elem().String()
		q.info.colType = GetColTypeByName(typeOf, "")
		if q.info.Type == 0 {
			q.info.colType = GetColTypeByID(9)
		}
		// fmt.Println("typeOf", q.info.Name, "|", typeOf, "|", q.info.Type, "|", q.info.IsSlice)
	}
	return q.info
}

func (q *Col[T, E]) GetInfoPointer() *columnInfo {
	if q.info.Type == 0 {
		q.info = q.GetInfo()
	}
	return &q.info
}

func (q Col[T, E]) DecimalSize(size int8) Col[T, E] {
	if size > 15 {
		panic("Decimal size TOO BIG in:" + q.GetName())
	}
	q.info.decimalSize = size
	return q
}

func (q Col[T, E]) Int32() Col[T, E] {
	q.info.useInt32Packing = true
	return q
}

func (q Col[T, E]) CompositeBucketing(buketsSize ...int8) Col[T, E] {
	q.info.compositeBucketing = buketsSize
	return q
}

func (q Col[T, E]) IsWeek() Col[T, E] {
	q.info.isWeek = true
	return q
}

func (q Col[T, E]) Autoincrement(randSufixSize int8) Col[T, E] {
	if randSufixSize > 15 {
		panic("Rand sufix size TOO BIG in:" + q.GetName())
	}

	if randSufixSize == 0 {
		randSufixSize = -1
	}
	q.info.autoincrementRandSize = randSufixSize
	return q
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

func (e *Col[T, E]) Contains(v int64) *T {
	// fmt.Println("e.schemaStruct", e.schemaStruct)
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: "CONTAINS", Value: any(v)})
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
	if q.info.Type == 0 {
		typeOf := reflect.TypeOf((*E)(nil)).Elem().String()
		if typeOf[0] == '*' {
			typeOf = "*[]" + typeOf[1:]
		} else {
			typeOf = "[]" + typeOf
		}
		q.info.colType = GetColTypeByName(typeOf, "")
		if q.info.colType.Type == 0 {
			panic("No se reconoió el slice type:" + typeOf)
		}
	}
	return q.info
}

func (q *ColSlice[T, E]) GetInfoPointer() *columnInfo {
	if q.info.Type == 0 {
		q.info = q.GetInfo()
	}
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

func initStructTable[T TableInterface[T], E any](schemaStruct *T) *T {
	// fmt.Println("making table...")
	structRefValue := reflect.ValueOf(*new(E))
	structRefType := structRefValue.Type()

	fieldNameIdxMap := map[string]*columnInfo{}
	refTableInfo := &TableInfo{}

	for i := 0; i < structRefType.NumField(); i++ {
		field := structRefType.Field(i)
		if field.Name == "TableStruct" {
			continue
		}
		xfield := xunsafe.FieldByName(structRefType, field.Name)
		col := &columnInfo{
			colInfo: colInfo{
				FieldIdx:  i,
				FieldName: field.Name,
				RefType:   field.Type,
				Field:     xfield,
			},
			colType: GetColTypeByName(field.Type.String(), ""),
		}

		if col.colType.Type == 0 {
			col.colType = GetColTypeByID(9)
		}

		if tag := field.Tag.Get("db"); tag != "" {
			col.Name = strings.Split(tag, ",")[0]
		}

		if DebugFull {
			offset := uintptr(0)
			if xfield != nil {
				offset = xfield.Offset
			}
			fmt.Printf("Base Struct Field: %-20s | Type: %-15s | Offset: %d\n", field.Name, field.Type.String(), offset)
		}

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
			fmt.Println("El field", fieldType.Name, "no implementa ColGetInfoPointer")
			continue
		}

		// Extract column name from db tag or convert field name to snake_case
		columnName := ""
		if tag := fieldType.Tag.Get("db"); tag != "" {
			columnName = strings.Split(tag, ",")[0]
		}

		if colBase, ok := fieldNameIdxMap[fieldType.Name]; ok {
			if columnName == "" {
				columnName = colBase.Name
			}

			if columnName == "" {
				columnName = toSnakeCase(fieldType.Name)
			}

			// fmt.Println("seteando nombre:", columnName)

			colInfo := column.GetInfoPointer()
			*colInfo = *colBase
			colInfo.Name = columnName

			column1, ok1 := fieldAddr.Interface().(Coln)
			if ok1 {
				// Transfer properties from Col if they were set
				if c, ok := any(column1).(*Col[T, E]); ok {
					colInfo.decimalSize = c.info.decimalSize
					colInfo.autoincrementRandSize = c.info.autoincrementRandSize
					colInfo.useInt32Packing = c.info.useInt32Packing
				}

				infoCheck := column1.GetInfo()
				if infoCheck.Name == "" {
					panic("No se seteo el nombre: " + columnName)
				}
			}

			// fmt.Printf("DEBUG: initStructTable: field=%s, colName=%s, colInfo.Name=%s, ptr=%p\n", fieldType.Name, columnName, colInfo.Name, colInfo)
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

/* Increment Table */
type Increment struct {
	TableStruct[IncrementTable, Increment]
	Name         string
	CurrentValue int64
}

type IncrementTable struct {
	TableStruct[IncrementTable, Increment]
	Name         Col[IncrementTable, string] // `db:"name,pk"`
	CurrentValue Col[IncrementTable, int64]  // `db:"current_value,counter"`
}

func (e IncrementTable) GetSchema() TableSchema {
	return TableSchema{
		Name:           "sequences",
		Keys:           []Coln{e.Name},
		SequenceColumn: &e.CurrentValue,
	}
}

// Deprecated: use GetCounter standalone function
func (e *TableStruct[T, E]) GetCounter(
	increment int, partValue any, secondPartValue ...any,
) (int64, error) {

	secondPartValue_ := any(0)
	if len(secondPartValue) > 0 {
		secondPartValue_ = secondPartValue[0]
	}

	scyllaTable := e.MakeScyllaTable()
	name := fmt.Sprintf("x%v_%v_%v", partValue, scyllaTable.name, secondPartValue_)

	return GetCounter(strings.Split(scyllaTable.GetFullName(), ".")[0], name, increment)
}

func GetCounter(keyspace string, name string, increment int) (int64, error) {
	result := []Increment{}

	if err := Query(&result).Name.Equals(name).Exec(); err != nil {
		return 0, Err("Error al obtener el counter: ", err)
	}

	currentValue := int64(1)
	if len(result) > 0 {
		currentValue = result[0].CurrentValue + 1
	}

	queryUpdateStr := fmt.Sprintf(
		"UPDATE %v.sequences SET current_value = current_value + %v WHERE name = '%v'",
		keyspace, increment, name,
	)

	if err := QueryExec(queryUpdateStr); err != nil {
		fmt.Println(queryUpdateStr)
		panic(err)
	}

	return currentValue, nil
}

type SeqValue struct {
	ID      int64 `db:"id"`
	SeqPart int64 `db:"seq_part"`
}

func (e ScyllaController[T, E]) ResetCounter(partValue any) error {

	scyllaTable := e.GetTable()
	if !scyllaTable.useSequences {
		return nil
	}

	seqValues := []SeqValue{}

	if scyllaTable.sequencePartCol == nil {

		maxValue := int64(0)

		queryMax := fmt.Sprintf(
			`SELECT max(%v) as id FROM %v WHERE %v = %v`,
			scyllaTable.keys[0].GetName(),
			scyllaTable.GetFullName(),
			scyllaTable.partKey.GetName(),
			partValue)

		if err := getScyllaConnection().Query(queryMax).Scan(&maxValue); err != nil {
			fmt.Println("Error al obtener el valor máximo (posiblemente tabla vacía): ", err)
		}

		seqValues = append(seqValues, SeqValue{ID: maxValue})

	} else {

		queryMax := fmt.Sprintf(
			`SELECT max(%v) as id, %v as seq_part FROM %v WHERE %v = %v GROUP BY %v ALLOW FILTERING`,
			scyllaTable.keys[0].GetName(),
			scyllaTable.sequencePartCol.GetName(),
			scyllaTable.GetFullName(),
			scyllaTable.partKey.GetName(),
			partValue,
			scyllaTable.sequencePartCol.GetName())

		iter := getScyllaConnection().Query(queryMax).Iter()
		var id int64
		var seqPart int64
		for iter.Scan(&id, &seqPart) {
			seqValues = append(seqValues, SeqValue{ID: id, SeqPart: seqPart})
		}
		if err := iter.Close(); err != nil {
			fmt.Println("Error al obtener los valores máximos agrupados: ", err)
		}
	}

	for _, seqValue := range seqValues {
		name := fmt.Sprintf("x%v_%v_%v", partValue, scyllaTable.name, seqValue.SeqPart)
		keyspace := strings.Split(scyllaTable.GetFullName(), ".")[0]

		queryUpdateStr := fmt.Sprintf(
			"UPDATE %v.sequences SET current_value = current_value + %v WHERE name = '%v'",
			keyspace, seqValue.ID, name,
		)

		fmt.Println(queryUpdateStr)

		if err := QueryExec(queryUpdateStr); err != nil {
			fmt.Println(queryUpdateStr)
			panic(err)
		}
	}

	return nil
}
