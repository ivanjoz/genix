package db

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"golang.org/x/sync/errgroup"
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

func (e scyllaTable[T]) fullName() string {
	return fmt.Sprintf("%v.%v", e.keyspace, e.name)
}

type IColumnStatement interface {
	GetValue() any
	GetName() string
}
type ColumnStatement struct {
	Col         string
	Operator    string
	Value       any
	Values      []any
	BetweenFrom []any
	BetweenTo   []any
}
type TableSchema struct {
	Keyspace string
	// StructType    T
	Name          string
	Keys          []Coln
	Partition     Coln
	GlobalIndexes []Coln
	LocalIndexes  []Coln
	HashIndexes   [][]Coln
	Views         []View
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

func (q *Query[T]) Where(ce ColumnStatement) *Query[T] {
	q.init()
	q.statements = append(q.statements, statementGroup{
		group: []ColumnStatement{ce},
	})
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

func Select[T TableSchemaInterface](handler func(query *Query[T], schemaTable T)) QueryResult[T] {

	query := Query[T]{}
	baseType := *new(T)
	handler(&query, baseType)
	records := []T{}
	err := selectExec(&records, &query)
	return QueryResult[T]{records, err}
}

func SelectRef[T TableSchemaInterface](recordsGetted *[]T, handler func(query *Query[T], schemaTable T)) error {

	query := Query[T]{}
	baseType := *new(T)
	handler(&query, baseType)
	return selectExec(recordsGetted, &query)
}

type posibleIndex struct {
	indexView    *viewInfo
	colsIncluded []int16
	colsMissing  []int16
	priority     int8
}

func selectExec[T TableSchemaInterface](recordsGetted *[]T, query *Query[T]) error {

	scyllaTable := makeTable(*new(T))
	viewTableName := scyllaTable.name

	if len(scyllaTable.keyspace) == 0 {
		scyllaTable.keyspace = connParams.Keyspace
	}
	if len(scyllaTable.keyspace) == 0 {
		return errors.New("no se ha especificado un keyspace")
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

	queryTemplate := fmt.Sprintf("SELECT %v ", strings.Join(columnNames, ", ")) + "FROM %v.%v %v"
	hashOperators := []string{"=", "IN"}
	isHash := true

	statements := []ColumnStatement{}
	colsWhereIdx := []int16{}
	allAreKeys := true

	for _, st := range query.statements {
		col := scyllaTable.columnsMap[st.group[0].Col]
		if !slices.Contains(scyllaTable.keysIdx, col.Idx) {
			allAreKeys = false
		}

		colsWhereIdx = append(colsWhereIdx, col.Idx)
		statements = append(statements, st.group[0])
		if isHash && !slices.Contains(hashOperators, st.group[0].Operator) {
			isHash = false
		}
	}

	posibleViewsOrIndexes := []posibleIndex{}

	if len(statements) > 0 && !allAreKeys {
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
				hasOperators := true
				if len(indview.Operators) > 0 {
					for _, st := range statements {
						if !slices.Contains(indview.Operators, st.Operator) {
							hasOperators = false
							break
						}
					}
				}

				if !hasOperators {
					continue
				}

				if len(psb.colsIncluded) == len(colsWhereIdx) {
					psb.priority = 6 + int8(len(colsWhereIdx))*2
					if isHash && indview.Type == 7 {
						psb.priority += 2
					}
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
		if viewOrIndex.Type >= 6 {
			viewTableName = viewOrIndex.name
		}

		if viewOrIndex.getStatement != nil {
			fmt.Println("Creating statement Index:", viewOrIndex.name)
			// Revisa qué columnas satisfacen el índice
			statementsSelected := []ColumnStatement{}
			for _, st := range statements {
				if slices.Contains(viewOrIndex.columns, st.Col) {
					statementsSelected = append(statementsSelected, st)
				} else {
					statementsRemain = append(statementsRemain, st)
				}
			}
			whereStatements = viewOrIndex.getStatement(statementsSelected...)
		} else {
			// No es necesario cambiar nada del query
			statementsRemain = statements
		}
	} else {
		statementsRemain = statements
	}

	if len(whereStatements) == 0 {
		whereStatements = append(whereStatements, "")
	}

	// fmt.Println("where statements::", whereStatements)
	whereStatementsRemainSects := []string{}
	for _, st := range statementsRemain {
		/*
			fmt.Println("column 1::", st.Column)
			fmt.Println("column 2::", st.Operator)
			fmt.Println("column 3::", st.GetValue())
		*/
		where := fmt.Sprintf("%v %v %v", st.Col, st.Operator, st.GetValue())
		// fmt.Println("where::", where)
		whereStatementsRemainSects = append(whereStatementsRemainSects, where)
	}

	whereStatementsRemain := strings.Join(whereStatementsRemainSects, " AND ")
	queryWhereStatements := []string{}

	for _, whereStatement := range whereStatements {
		if whereStatementsRemain != "" {
			if whereStatement != "" {
				whereStatementsRemain = " AND " + whereStatementsRemain
			}
			whereStatement += whereStatementsRemain
		}
		if whereStatement != "" {
			whereStatement = " WHERE " + whereStatement
		}
		queryWhereStatements = append(queryWhereStatements, whereStatement)
	}

	recordsMap := map[int]*[]T{}
	for i := range queryWhereStatements {
		recordsMap[i] = &[]T{}
	}

	eg := errgroup.Group{}

	for i, whereStatement := range queryWhereStatements {
		queryStr := fmt.Sprintf(queryTemplate, scyllaTable.keyspace, viewTableName, whereStatement)
		records := recordsMap[i]

		fmt.Println("Query::", queryStr)
		eg.Go(func() error {

			iter := getScyllaConnection().Query(queryStr).Iter()
			rd, err := iter.RowData()
			if err != nil {
				fmt.Println("Error on RowData::", err)
				if strings.Contains(err.Error(), "use ALLOW FILTERING") {
					fmt.Println("Posible Indexes or Views::")
					for _, e := range posibleViewsOrIndexes {
						typeName := indexTypes[e.indexView.Type]
						colnames := strings.Join(e.indexView.columns, ", ")
						msg := fmt.Sprintf("%v (%v) %v", typeName, e.indexView.column.Type, colnames)
						fmt.Println(msg)
					}
				}
				return err
			}

			scanner := iter.Scanner()
			fmt.Println("starting iterator | columns::", len(scyllaTable.columns))

			for scanner.Next() {

				rowValues := rd.Values

				err := scanner.Scan(rowValues...)
				if err != nil {
					fmt.Println("Error on scan::", err)
					return err
				}

				rec := new(T)
				ref := reflect.ValueOf(rec).Elem()

				for idx, colname := range columnNames {
					column := scyllaTable.columnsMap[colname]
					value := rowValues[idx]
					if value == nil {
						continue
					}
					if column.setValue != nil {
						field := ref.Field(column.FieldIdx)
						column.setValue(&field, value)
						// Revisa si necesita parsearse un string a un struct como JSON
					} else if column.IsComplexType {
						fmt.Println("complex type::", column.FieldName, "|", column.FieldType)
						if vl, ok := value.(*[]uint8); ok {
							if len(*vl) <= 3 {
								continue
							}
							newElm := reflect.New(column.RefType).Elem()
							err = cbor.Unmarshal([]byte(*vl), newElm.Addr().Interface())
							if err != nil {
								fmt.Println("Error al convertir: ", newElm, "|", err.Error())
							}
							fmt.Println("col complex:", column.Name, " | ", newElm, " | L:", len(*vl))
							if column.IsPointer {
								ref.Field(column.FieldIdx).Set(reflect.ValueOf(newElm))
							} else {
								ref.Field(column.FieldIdx).Set(newElm)
							}
						} else {
							fmt.Print("Complex Type could not be parsed:", column.Name)
						}
					} else {
						fmt.Print("Column is not mapped:: ", column.Name)
					}
				}

				if len(queryWhereStatements) == 1 {
					(*recordsGetted) = append((*recordsGetted), *rec)
				} else {
					(*records) = append((*records), *rec)
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if len(queryWhereStatements) > 1 {
		for _, records := range recordsMap {
			(*recordsGetted) = append((*recordsGetted), *records...)
		}
	}

	return nil
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

func Insert[T TableSchemaInterface](records *[]T, columnsToExclude ...Coln) error {

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
	fmt.Println(queryInsert)

	if err := QueryExec(queryInsert); err != nil {
		fmt.Println("Error inserting records:", err)
		return err
	}

	return nil
}

func makeUpdateQuery[T TableSchemaInterface](records *[]T, columnsToInclude []Coln, columnsToExclude []Coln, onlyVirtual bool) []string {

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
			scyllaTable.fullName(), Concatx(", ", setStatements), Concatx(" and ", whereStatements),
		)

		queryStatements = append(queryStatements, queryStatement)
	}

	return queryStatements
}

func Update[T TableSchemaInterface](records *[]T, columnsToInclude ...Coln) error {

	if len(columnsToInclude) == 0 {
		panic("No se incluyeron columnas a actualizar.")
	}

	queryStatements := makeUpdateQuery(records, columnsToInclude, nil, false)
	queryInsert := makeQueryStatement(queryStatements)
	if err := QueryExec(queryInsert); err != nil {
		fmt.Println(queryInsert)
		fmt.Println("Error inserting records:", err)
		return err
	}
	return nil
}
