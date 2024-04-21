package core

import (
	"app/types"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gocql/gocql"
)

var mu sync.Mutex
var scyllaSession *gocql.Session
var isConnectingTime int64 = 0

func ScyllaConnect(reconect_ ...bool) *gocql.Session {
	reconect := false
	if len(reconect_) == 1 {
		reconect = reconect_[0]
	}

	nowTime := time.Now().Unix()

	if (isConnectingTime + 12) > nowTime {
		for scyllaSession == nil {
			time.Sleep(2 * time.Millisecond)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if reconect {
		if scyllaSession != nil {
			scyllaSession.Close()
			scyllaSession = nil
		}
	} else if scyllaSession != nil {
		return scyllaSession
	}

	isConnectingTime = nowTime

	cluster := gocql.NewCluster(Env.DB_HOST)
	fallback := gocql.RoundRobinHostPolicy()
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(fallback)
	cluster.Port = int(Env.DB_PORT)
	cluster.Consistency = gocql.LocalOne
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = time.Second * 10
	cluster.Compressor = gocql.SnappyCompressor{}
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username:              Env.DB_USER,
		Password:              Env.DB_PASSWORD,
		AllowedAuthenticators: []string{"org.apache.cassandra.auth.PasswordAuthenticator"},
	}

	session, err := cluster.CreateSession() //  gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		log.Println(err)
		return nil
	}
	Log("Base de datos ScyllaDB Conectada!!")

	scyllaSession = session
	return scyllaSession
}

type BDColumn struct {
	Name           string
	FieldType      string
	FieldName      string
	NameAlias      string
	Type           string
	RefType        reflect.Value
	FieldIdx       int
	IsPrimaryKey   int8
	IsPointer      bool
	IsViewExcluded bool
	HasView        bool
	IsComplexType  bool
	ViewIdx        int8
}

type BDIndex struct {
	Name        string
	ColumnsPos  map[string]int
	MakeIntHash func(reflect.Value) int32

	MakeIntHashFomValues func(ColumnName, Values []string) int32
}

func (e *BDColumn) ParseValue(field reflect.Value) string {
	var v any
	if e.IsPointer {
		if field.IsNil() {
			return "NULL"
		} else {
			v = field.Elem().Interface()
		}
	} else if field.Kind() == reflect.Slice && field.Len() == 0 && e.IsComplexType {
		return "NULL"
	} else {
		v = field.Interface()
	}

	fType := strings.ReplaceAll(e.FieldType, "*", "")

	if e.IsComplexType {
		recordBytes, err := MsgPEncode(v)
		/*
			Log("Encoding type::", e.Name, " | Is Null:", v == nil, " | Kind:", field.Kind(), " | Len:", field.Len())
			Print(recordBytes)
			Print(v)
		*/
		if err != nil {
			Log("Error al encoded .gob:: ", e.FieldName, err.Error())
			return ""
		}
		hexString := hex.EncodeToString(recordBytes)
		Log("Hex String: ", "0x"+hexString)
		return "0x" + hexString
	} else if fType == "string" {
		return fmt.Sprintf(`'%v'`, v)
	} else if fType[0:2] == "[]" {
		concatenatedValues := ""
		if sl, ok := v.([]int32); ok {
			concatenatedValues = Concatx(",", sl)
		} else if sl, ok := v.([]int16); ok {
			concatenatedValues = Concatx(",", sl)
		} else if sl, ok := v.([]int8); ok {
			concatenatedValues = Concatx(",", sl)
		} else if sl, ok := v.([]int); ok {
			concatenatedValues = Concatx(",", sl)
		} else if sl, ok := v.([]int64); ok {
			concatenatedValues = Concatx(",", sl)
		} else if sl, ok := v.([]string); ok {
			strValues := []string{}
			for _, v := range sl {
				strValues = append(strValues, `'`+v+`'`)
			}
			concatenatedValues = strings.Join(strValues, ",")
		} else {
			errMsg := fmt.Sprintf("No se pudo identificar el Set<>: %v", v)
			panic(errMsg)
		}
		return "{" + concatenatedValues + "}"
	} else {
		return fmt.Sprintf(`%v`, v)
	}
}

type ScyllaTable struct {
	Name          string
	NameSingle    string
	PrimaryKey    string
	PartitionKey  string
	Columns       []BDColumn
	ColumnsMap    map[string]BDColumn
	Indexes       map[string]BDIndex
	Views         map[string]ScyllaView
	ViewsExcluded []string
}

var TypeToScyllaTableMap = map[string]ScyllaTable{}

func GetCounter(name string, incrementCount int, empresaID ...int32) (int64, error) {
	if len(empresaID) == 1 {
		name = fmt.Sprintf("x%v_%v", empresaID[0], name)
	}

	registros := []types.Increment{}
	err := DBSelect(&registros).Where("name").Equals(name).Exec()
	if err != nil {
		return 0, Err("Error al obtener el counter: ", err)
	}

	currentValue := int64(1)
	if len(registros) > 0 {
		currentValue = registros[0].CurrentValue
	}

	queryUpdateStr := fmt.Sprintf(
		"UPDATE %v.sequences SET current_value = current_value + %v WHERE name = '%v'",
		Env.DB_NAME, incrementCount, name,
	)

	queryUpdate := ScyllaConnect().Query(queryUpdateStr)
	if err := queryUpdate.Exec(); err != nil {
		Log(queryUpdate)
		panic(err)
	}

	return currentValue, nil
}

func MakeQueryStatement(statements []string) string {
	queryStr := ""
	if len(statements) == 1 {
		queryStr = statements[0]
	} else {
		statements := strings.Join(statements, "\n")
		queryStr = fmt.Sprintf("BEGIN BATCH\n%v\nAPPLY BATCH;", statements)
	}
	return queryStr
}

func MakeInsertQuery[T any](records *[]T, columnsToAvoid ...string) []string {

	var newType T
	scyllaTable := MakeScyllaTable(newType)

	columnsToSave := []BDColumn{}
	columnsNames := []string{}
	for _, e := range scyllaTable.Columns {
		columnsNames = append(columnsNames, e.Name)
		columnsToSave = append(columnsToSave, e)
	}

	indexes := []BDIndex{}
	for _, index := range scyllaTable.Indexes {
		columnsNames = append(columnsNames, index.Name)
		indexes = append(indexes, index)
	}

	queryStrInsert := fmt.Sprintf(`INSERT INTO %v (%v) VALUES `,
		scyllaTable.Name, strings.Join(columnsNames, ", "))

	queryStatements := []string{}

	for _, rec := range *records {

		refValue := reflect.ValueOf(rec)
		recordInsertValues := []string{}

		for _, col := range columnsToSave {
			v := ""
			if col.ViewIdx > 0 {
				if baseI, ok := any(&rec).(IGetView); ok {
					v = parseValueToString(baseI.GetView(col.ViewIdx))
				}
			} else {
				v = col.ParseValue(refValue.Field(col.FieldIdx))
			}
			recordInsertValues = append(recordInsertValues, v)
		}

		for _, index := range indexes {
			v := fmt.Sprintf("%v", index.MakeIntHash(refValue))
			recordInsertValues = append(recordInsertValues, v)
		}

		statement := /*" " +*/ queryStrInsert + "(" + strings.Join(recordInsertValues, ", ") + ")"
		queryStatements = append(queryStatements, statement)
	}

	return queryStatements
}

func DBExec(query *gocql.Query) error {
	if err := query.Exec(); err != nil {
		if strings.Contains(err.Error(), "no hosts available") {
			Log(`Error en conexión db: "no hosts available", reconectando...`)
			ScyllaConnect(true)
			Log(`Ejecutando query luego de reconexión...`)
			err = query.Exec()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func DBInsert[T any](records *[]T, columnsToAvoid ...string) error {
	queryStatements := MakeInsertQuery(records, columnsToAvoid...)

	queryStr := MakeQueryStatement(queryStatements)
	LogDebug(queryStr)
	if err := ScyllaConnect().Query(queryStr).Exec(); err != nil {
		Log("Error al insertar registros: ", err)
		Log(queryStr)
		return err
	}
	return nil
}

func MakeUpdateQuery[T any](records *[]T, columnsToInclude ...string) []string {
	var newType T
	scyllaTable := MakeScyllaTable(newType)

	columnsToUpdate := []BDColumn{}
	columnsWhere := []BDColumn{}
	includeAll := len(columnsToInclude) == 0 || Contains(columnsToInclude, "*")
	useExclude := len(columnsToInclude) > 0 && columnsToInclude[0] == "-"

	for _, e := range scyllaTable.Columns {
		if e.IsPrimaryKey > 0 {
			columnsWhere = append(columnsWhere, e)
			continue
		}
		if useExclude {
			if !Contains(columnsToInclude, e.Name) {
				columnsToUpdate = append(columnsToUpdate, e)
			}
		} else if includeAll || Contains(columnsToInclude, e.Name) {
			columnsToUpdate = append(columnsToUpdate, e)
		}
	}

	queryStatements := []string{}

	for _, rec := range *records {
		refValue := reflect.ValueOf(rec)

		setStatements := []string{}
		for _, col := range columnsToUpdate {
			v := col.ParseValue(refValue.Field(col.FieldIdx))
			setStatements = append(setStatements, fmt.Sprintf(`%v = %v`, col.Name, v))
		}

		whereStatements := []string{}
		for _, col := range columnsWhere {
			v := col.ParseValue(refValue.Field(col.FieldIdx))
			whereStatements = append(whereStatements, fmt.Sprintf(`%v = %v`, col.Name, v))
		}

		queryStatement := fmt.Sprintf(
			"UPDATE %v SET %v WHERE %v",
			scyllaTable.Name, Concatx(", ", setStatements), Concatx(" and ", whereStatements),
		)

		queryStatements = append(queryStatements, queryStatement)
	}
	return queryStatements
}

func DBUpdate[T any](records *[]T, columnsToInclude ...string) error {
	queryStatements := MakeUpdateQuery(records, columnsToInclude...)

	queryStr := MakeQueryStatement(queryStatements)
	LogDebug(queryStr)
	if err := ScyllaConnect().Query(queryStr).Exec(); err != nil {
		Log("Error al actualizar registros: ", err)
		Log(queryStr)
		return err
	}
	return nil
}

func DBUpdateInsert[T any](
	records *[]T,
	isRecordForInsert func(e T) bool,
	columnsToAvoidUpdate []string,
	columnsToAvoidInsert ...string,
) error {

	recordsToInsert := []T{}
	recordsToUpdate := []T{}

	for _, e := range *records {
		if isRecordForInsert(e) {
			recordsToInsert = append(recordsToInsert, e)
		} else {
			recordsToUpdate = append(recordsToUpdate, e)
		}
	}

	if len(recordsToUpdate) > 0 {
		LogDebug("Registros a actualizar:", len(recordsToUpdate))

		columnsToAvoidUpdate = append([]string{"-"}, columnsToAvoidUpdate...)
		err := DBUpdate(&recordsToUpdate, columnsToAvoidUpdate...)
		if err != nil {
			return err
		}
	}

	if len(recordsToInsert) > 0 {
		LogDebug("Registros a insertar:", len(recordsToInsert))

		err := DBInsert(&recordsToInsert, columnsToAvoidInsert...)
		if err != nil {
			return err
		}
	}

	return nil
}

func DBSelect[T any](records *[]T, columnsToAvoid ...string) *QuerySelect[T] {
	return &QuerySelect[T]{
		Records:        records,
		ColumnsToAvoid: columnsToAvoid,
	}
}

func DBSelectReflect(records *[]reflect.Type, columnsToAvoid ...string) *QuerySelect[reflect.Type] {
	return &QuerySelect[reflect.Type]{
		Records:        records,
		ColumnsToAvoid: columnsToAvoid,
	}
}

func ExecuteStatements(statements []string) error {
	batchStatement := MakeQueryStatement(statements)
	return ScyllaConnect().Query(batchStatement).Exec()
}

type QueryParams struct {
	Type      int8
	Columns   []string
	Values    []any
	Operator  string
	Connector string
	Group     int32
}

type QuerySelect[T any] struct {
	Records          *[]T
	ColumnsToAvoid   []string
	ColumnsToInclude []string
	ComandsWhere     []QueryParams
	GroupCount       int32
	OrderBy          string
	limit            int32
}

func (e *QuerySelect[T]) Where(names ...string) *QuerySelect[T] {
	if len(e.ComandsWhere) > 0 {
		cur := &e.ComandsWhere[len(e.ComandsWhere)-1]
		if len(cur.Columns) == 0 || len(cur.Operator) == 0 {
			panic("No se puede agregar un WHERE en este lugar.")
		}
	}
	e.ComandsWhere = append(e.ComandsWhere,
		QueryParams{Columns: names, Group: e.GroupCount})
	return e
}

func (e *QuerySelect[T]) Columns(names ...string) *QuerySelect[T] {
	e.ColumnsToInclude = append(e.ColumnsToInclude, names...)
	return e
}

func (e *QuerySelect[T]) checkError(name string) (string, *QueryParams) {
	ln := len(e.ComandsWhere)
	if ln == 0 {
		return "No se puede agregar el operador:: " + name + " en este lugar.", nil
	} else {
		current := &e.ComandsWhere[len(e.ComandsWhere)-1]
		if len(current.Columns) == 0 {
			return "No se puede agregar el operador:: " + name + " sin una columna", nil
		}
		return "", current
	}
}

func (e *QuerySelect[T]) addOperator(name string, values []any) *QuerySelect[T] {
	errMsg, current := e.checkError(name)
	if len(errMsg) > 0 {
		panic(errMsg)
	}
	if len(current.Columns) != len(values) {
		panic(fmt.Sprintf(`El número de columnas no corresponde con los valores: "%v" != "%v"`, current.Columns, values))
	}

	current.Operator = name
	current.Values = values
	return e
}

func (e *QuerySelect[T]) addOperatorValues(name string, values []any) *QuerySelect[T] {
	errMsg, current := e.checkError(name)
	if len(errMsg) > 0 {
		panic(errMsg)
	}
	current.Operator = name
	current.Values = values
	return e
}

func (e *QuerySelect[T]) Equals(values ...any) *QuerySelect[T] {
	if len(values) == 1 {
		return e.addOperator("=", values)
	} else {
		return e.addOperatorValues("=", values)
	}
}
func (e *QuerySelect[T]) GreatThan(values ...any) *QuerySelect[T] {
	return e.addOperator(">", values)
}
func (e *QuerySelect[T]) GreatEq(values ...any) *QuerySelect[T] {
	return e.addOperator(">=", values)
}
func (e *QuerySelect[T]) LessThan(values ...any) *QuerySelect[T] {
	return e.addOperator("<", values)
}
func (e *QuerySelect[T]) LessEq(values ...any) *QuerySelect[T] {
	return e.addOperator("<=", values)
}
func (e *QuerySelect[T]) In(values []any) *QuerySelect[T] {
	return e.addOperatorValues("IN", values)
}
func (e *QuerySelect[T]) IN(values ...any) *QuerySelect[T] {
	return e.addOperatorValues("IN", values)
}
func (e *QuerySelect[T]) Contains(values ...any) *QuerySelect[T] {
	return e.addOperator("CONTAINS", values)
}
func (e *QuerySelect[T]) Like(values ...any) *QuerySelect[T] {
	return e.addOperator("LIKE", values)
}
func (e *QuerySelect[T]) OrderAscending() *QuerySelect[T] {
	e.OrderBy = "ORDER BY %v ASC"
	return e
}
func (e *QuerySelect[T]) OrderDescending() *QuerySelect[T] {
	e.OrderBy = "ORDER BY %v DESC"
	return e
}
func (e *QuerySelect[T]) Limit(limit int32) *QuerySelect[T] {
	e.limit = limit
	return e
}

func parseValueToString(v any) string {
	if str, ok := v.(string); ok {
		return `'` + str + `'`
	} else {
		return fmt.Sprintf("%v", v)
	}
}

func (e *QuerySelect[T]) Exec(allowFiltering ...bool) error {
	conn := ScyllaConnect()

	var newType T
	scyllaTable := MakeScyllaTable(newType)

	viewTableName := scyllaTable.Name
	queryStr := "SELECT %v FROM %v"
	indexOperators := MakeSliceInclude([]string{"=", "IN"})
	columnsWhere := SliceInclude[string]{}

	whereGroups := SliceToMap(e.ComandsWhere, func(e QueryParams) int32 { return e.Group })
	for _, whereGroup := range whereGroups {
		// Print(whereGroup)

		wheres := []string{}
		for _, qp := range whereGroup {
			var wh string

			// Revisa si hay un view para esa columna
			if len(qp.Columns) >= 1 {
				colnames := strings.Join(qp.Columns, "_")
				if view, ok := scyllaTable.Views[colnames]; ok {
					viewTableName = view.Name
					// Columnas combinadas
					if view.Idx > 0 {
						base := new(T)
						ref := reflect.ValueOf(base).Elem()

						for i, colname := range qp.Columns {
							va := qp.Values[i]
							if column, ok := scyllaTable.ColumnsMap[colname]; ok {
								rType := reflect.TypeOf(va).Name()

								if rType != column.FieldType {
									if _va, ok := va.(int64); ok {
										if column.FieldType == "int32" {
											va = int32(_va)
										} else if column.FieldType == "int" {
											va = int(_va)
										}
									} else if _va, ok := va.(int32); ok {
										if column.FieldType == "int64" {
											va = int64(_va)
										} else if column.FieldType == "int" {
											va = int(_va)
										}
									} else if _va, ok := va.(int); ok {
										if column.FieldType == "int64" {
											va = int64(_va)
										} else if column.FieldType == "int32" {
											va = int32(_va)
										}
									}
								}

								rType = reflect.TypeOf(va).Name()
								if rType != column.FieldType {
									panic(fmt.Sprintf(`%v | Los tipos de datos de la columna "%v" no coinciden ("%v" != "%v") | Valor: %v`, scyllaTable.Name, view.ColumnName, rType, column.FieldType, va))
								}
								fmt.Println("columna tipo::", view.ColumnName, rType, column.FieldType, va)
								ref.Field(column.FieldIdx).Set(reflect.ValueOf(va))
							}
						}

						// Convierte los valores multiples en un sólo valor y una cola columna que representa el sk de la vista creada
						if baseI, ok := any(base).(IGetView); ok {
							// qp.Value = baseI.GetView(view.Idx)
							qp.Values = []any{baseI.GetView(view.Idx)}
							qp.Columns = []string{view.ColumnName}
						}
					}
				}
			}

			// Revisa si se puede utilizar un indice compuesto
			indexCols := ""
			if len(qp.Columns) > 1 && indexOperators.Include(qp.Operator) {
				indexCols = strings.Join(qp.Columns, "+")
			}

			if index, ok := scyllaTable.Indexes[indexCols]; ok {
				if qp.Operator == "=" {
					equalValues := []string{}
					if len(qp.Values) > 0 {
						for _, v := range qp.Values {
							equalValues = append(equalValues, fmt.Sprintf("%v", v))
						}
					} else {
						vs := strings.Split(fmt.Sprintf("%v", qp.Values[0]), "+")
						equalValues = append(equalValues, vs...)
					}
					if len(equalValues) != len(qp.Columns) {
						msg := fmt.Sprintf(
							"Los valores: %v y las columnas: %v no coinciden.",
							Concatx(", ", qp.Columns),
							Concatx(", ", qp.Values))

						return Err(msg)
					}
					hashInt := index.MakeIntHashFomValues(qp.Columns, equalValues)
					wh = fmt.Sprintf(`%v %v %v`, index.Name, qp.Operator, hashInt)
				} else if qp.Operator == "IN" {
					hashSlice := []string{}
					for _, inValues := range qp.Values {
						vs := strings.Split(fmt.Sprintf("%v", inValues), "+")
						if len(vs) != len(qp.Columns) {
							vs_ := strings.Join(vs, ", ")
							return Err("Los valores no coincide con el número de columnas: " + vs_)
						}
						hashInt := index.MakeIntHashFomValues(qp.Columns, vs)
						hashSlice = append(hashSlice, fmt.Sprintf("%v", hashInt))
					}
					wh = fmt.Sprintf(`%v IN (%v)`, index.Name, strings.Join(hashSlice, ", "))
				}
				// Revisa si hay más de 1 valor en values
			} else if qp.Operator == "IN" {
				inValues := []string{}
				for _, v := range qp.Values {
					inValues = append(inValues, parseValueToString(v))
				}
				wh = fmt.Sprintf(`%v IN (%v)`, qp.Columns[0], strings.Join(inValues, ", "))
			} else {
				if len(qp.Values) != len(qp.Columns) {
					return Err("El numero de valores no coincide con el número de columnas.")
				}
				wheresToJoin := []string{}
				for i, col := range qp.Columns {
					v := parseValueToString(qp.Values[i])
					wheresToJoin = append(wheresToJoin, fmt.Sprintf(`%v %v %v`, col, qp.Operator, v))
				}
				wh = strings.Join(wheresToJoin, " AND ")
			}
			wheres = append(wheres, wh)
			for _, col := range qp.Columns {
				columnsWhere.Add(col)
			}
		}
		queryStr += " WHERE " + strings.Join(wheres, " AND ")
	}

	if len(allowFiltering) > 0 && allowFiltering[0] {
		queryStr += " ALLOW FILTERING"
	}

	//Revisa que las columnas existan
	for _, col := range e.ColumnsToInclude {
		if _, ok := scyllaTable.ColumnsMap[col]; !ok {
			return Err("La columna: ", col, " no existe en la tabla: ", scyllaTable.Name)
		}
	}

	isUsingView := scyllaTable.Name != viewTableName
	columnNames := []string{}
	columnsToInclude := MakeSliceInclude(e.ColumnsToInclude)
	columnsIdxMap := map[int]BDColumn{}

	for _, co := range scyllaTable.Columns {
		if !columnsToInclude.IncludeN(co.Name) {
			continue
		}
		if co.IsViewExcluded {
			if isUsingView && !columnsWhere.Include(co.Name) {
				continue
			}
		}
		columnsIdxMap[len(columnNames)] = co
		columnNames = append(columnNames, co.Name)
	}

	queryStr = fmt.Sprintf(queryStr, strings.Join(columnNames, ", "), viewTableName)
	if len(e.OrderBy) > 0 {
		if strings.Contains(e.OrderBy, "%v") {
			e.OrderBy = fmt.Sprintf(e.OrderBy, columnsWhere.Values[len(columnsWhere.Values)-1])
		}
		queryStr += " " + e.OrderBy
	}

	Log("query string::", scyllaTable.Name)
	Log(queryStr)

	iter := conn.Query(queryStr).Iter()
	rd, _ := iter.RowData()
	scanner := iter.Scanner()

	for scanner.Next() {

		rowValues := rd.Values

		err := scanner.Scan(rowValues...)
		if err != nil {
			return err
		}

		rec := new(T)
		ref := reflect.ValueOf(rec).Elem()

		for idx, column := range columnsIdxMap {
			value := rowValues[idx]
			if value == nil || column.FieldIdx < 0 {
				continue
			}
			if mapField, ok := fieldMapping[column.FieldType]; ok {
				field := ref.Field(column.FieldIdx)
				//Log("Mapeando valor::", field, column.Name, column.FieldType, values)
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
					err = MsgPDecode(*vl, &newStruct)
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
				Log("Columna no-mapeada:: ", column)
			}
		}
		//Print(rec)
		(*e.Records) = append((*e.Records), *rec)
	}

	err := scanner.Err()
	if err != nil {
		Log(queryStr)
		return Err(err)
	}

	return nil
}
