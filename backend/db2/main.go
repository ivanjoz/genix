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
	statements     []ColumnStatement
	columnsInclude []columnInfo
	columnsExclude []columnInfo
	between        ColumnStatement
	orderBy        string
	limit          int32
	allowFilter    bool
}

type TableStructInterface[T any] interface {
	Query() *T
	GetSchema() TableSchema
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
func (e *TableStruct[T, E]) GetInfoPointer() *columnInfo { // Para compatibilidad
	return &columnInfo{}
}
func (e *TableStruct[T, E]) SetSchemaStruct(schemaStruct any) {
	if schema, ok := schemaStruct.(*T); ok {
		e.schemaStruct = schema
	}
}

type TableBaseInterface[T any, E any] interface {
	GetTable() E
	GetBaseStruct() T
}

func (e TableStruct[T, E]) GetTable() E {
	return *new(E)
}
func (e TableStruct[T, E]) GetBaseStruct() T {
	return *new(T)
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

func MakeTable[T TableBaseInterface[T, E], E any](schemaStruct *T) *T {
	fmt.Println("making table...")
	structRefValue := reflect.ValueOf(*new(E))
	structRefType := structRefValue.Type()

	fieldNameIdxMap := map[string]*columnInfo{}
	refTableInfo := &TableInfo{}

	for i := 0; i < structRefValue.NumField(); i++ {
		col := columnInfo{
			FieldIdx:  i,
			FieldType: structRefType.Field(i).Type.String(),
			FieldName: structRefType.Field(i).Name,
			RefType:   structRefType.Field(i).Type,
		}

		if col.FieldType[0:1] == "*" {
			col.IsPointer = true
			col.FieldType = col.FieldType[1:]
		}
		if col.FieldType[0:2] == "[]" {
			col.IsSlice = true
			col.FieldType = col.FieldType[2:]
		}

		fmt.Println("fieldtype obtenido::", col.FieldName, col.FieldType)
		// fmt.Println("Fieldname::", col.FieldName, "| Type:", col.FieldType)
		fieldNameIdxMap[col.FieldName] = &col
	}

	structValue := reflect.ValueOf(schemaStruct).Elem()
	structType := structValue.Type()

	fmt.Println("fieldNameIdxMap...", len(fieldNameIdxMap), "| nf:", structValue.NumField())

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		// Check if field can be addressed and if it implements ColumnSetName interface
		if !field.CanAddr() || !field.Addr().CanInterface() {
			fmt.Println("no es::", fieldType.Name)
			continue
		}

		fieldAddr := field.Addr()
		// Try to get the interface and check if it implements ColumnSetName
		column, ok := fieldAddr.Interface().(ColGetInfoPointer)
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

		if colBase, ok := fieldNameIdxMap[fieldType.Name]; ok {
			colInfo := column.GetInfoPointer()
			colInfo.Name = columnName
			colInfo.FieldIdx = colBase.FieldIdx
			colInfo.FieldType = colBase.FieldType
			colInfo.FieldName = colBase.FieldName
			colInfo.RefType = colBase.RefType
		} else if fieldType.Name != "TableStruct" {
			err := fmt.Sprintf(`No se encontró el field "%v" en el struct "%v"`, fieldType.Name, structRefType.Name())
			panic(err)
		}

		// Set the column name using the interface method
		column.SetSchemaStruct(schemaStruct)
		column.SetTableInfo(refTableInfo)
	}
	fmt.Println("schemaStruct (1)", schemaStruct)
	return schemaStruct
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

func Print(Struct any) {
	pretty.Println(Struct)
}

func (e *TableStruct[T, E]) Exec() ([]E, error) {
	return execQuery[T, E](e.schemaStruct, e.tableInfo)
}

func (e *TableStruct[T, E]) All() ([]E, error) {
	return e.Exec()
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

func (e *TableStruct[T, E]) Between(statements ...ColumnStatement) *T {
	e.tableInfo.between.From = statements
	e.tableInfo.between.Operator = "BETWEEN"
	if len(statements) == 1 {
		e.tableInfo.between.Col = statements[0].Col
	}
	return e.schemaStruct
}

func (e *TableStruct[T, E]) AddStatement(statement ColumnStatement) {
	e.tableInfo.statements = append(e.tableInfo.statements, statement)
}

type TableInterface[T any] interface {
	GetSchema() TableSchema
}

type Col[T TableInterface[T], E any] struct {
	info         columnInfo
	schemaStruct *T
	tableInfo    *TableInfo
}

func (q Col[T, E]) GetInfo() columnInfo {
	return q.info
}

type ColGetInfoPointer interface {
	GetInfoPointer() *columnInfo
	SetSchemaStruct(any)
	SetTableInfo(*TableInfo)
}

func (q *Col[T, E]) GetInfoPointer() *columnInfo {
	return &q.info
}

func (q Col[T, E]) GetName() string {
	return q.info.Name
}

func (c *Col[T, E]) SetName(name string) {
	c.info.Name = name
}

func (c *Col[T, E]) SetTableInfo(tableInfo *TableInfo) {
	c.tableInfo = tableInfo
}

func (c *Col[T, E]) SetSchemaStruct(schemaStruct any) {
	fmt.Println("seteando schemaStruct", c.info.Name)
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
	fmt.Println("e.schemaStruct", e.schemaStruct)
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

// Generic Array
type ColSlice[T any] struct {
	info         columnInfo
	schemaStruct any
	tableInfo    *TableInfo
}

func (q ColSlice[T]) GetInfo() columnInfo {
	return columnInfo{Name: q.info.Name, FieldType: reflect.TypeFor[T]().String(), IsSlice: true}
}

func (q ColSlice[T]) GetName() string {
	return q.info.Name
}

func (c *ColSlice[T]) SetName(name string) {
	c.info.Name = name
}

func (c *ColSlice[T]) SetTableInfo(tableInfo *TableInfo) {
	c.tableInfo = tableInfo
}

func (c *ColSlice[T]) SetSchemaStruct(schemaStruct any) {
	c.schemaStruct = schemaStruct
}

func (e *ColSlice[T]) Contains(v T) any {
	e.tableInfo.statements = append(e.tableInfo.statements, ColumnStatement{Col: e.info.Name, Operator: "CONTAINS", Value: any(v)})
	return e.schemaStruct
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

// execQuery executes a query based on TableInfo and returns records
func execQuery[T TableStructInterface[T], E any](schemaStruct *T, tableInfo *TableInfo) ([]E, error) {
	// Get the schema from the struct
	schema := (*schemaStruct).GetSchema()
	scyllaTable := MakeTableSchema(schema, schemaStruct)
	// Use the existing selectExec logic but adapted for our new structure
	records := []E{}
	err := selectExecNew(&records, tableInfo, scyllaTable)
	return records, err
}
