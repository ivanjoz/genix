package core

import (
	"app/types"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

var scyllaSession *gocql.Session

func ScyllaConnect() *gocql.Session {
	if scyllaSession != nil {
		return scyllaSession
	}

	cluster := gocql.NewCluster(Env.DB_HOST)
	fallback := gocql.RoundRobinHostPolicy()
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(fallback)
	cluster.Port = int(Env.DB_PORT)
	cluster.Consistency = gocql.One
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
	Name          string
	FieldType     string
	FieldName     string
	NameAlias     string
	Type          string
	RefType       reflect.Value
	FieldIdx      int
	IsPointer     bool
	IsPrimaryKey  int8
	IsComplexType bool
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
	Name         string
	NameSingle   string
	PrimaryKey   string
	PartitionKey string
	Columns      []BDColumn
	ColumnsMap   map[string]BDColumn
	Indexes      map[string]BDIndex
	Views        map[string]string
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

func DBInsert[T any](records *[]T, columnsToAvoid ...string) error {
	conn := ScyllaConnect()

	var newType T
	scyllaTable := MakeScyllaTable(newType)

	columnsToSave := scyllaTable.Columns
	columnsNames := []string{}
	for _, e := range columnsToSave {
		columnsNames = append(columnsNames, e.Name)
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
			v := col.ParseValue(refValue.Field(col.FieldIdx))
			recordInsertValues = append(recordInsertValues, v)
		}

		for _, index := range indexes {
			v := fmt.Sprintf("%v", index.MakeIntHash(refValue))
			recordInsertValues = append(recordInsertValues, v)
		}

		statement := " " + queryStrInsert + "(" + strings.Join(recordInsertValues, ", ") + ")"
		queryStatements = append(queryStatements, statement)
	}

	queryStr := MakeQueryStatement(queryStatements)
	if err := conn.Query(queryStr).Exec(); err != nil {
		Log(queryStr)
		return err
	}
	return nil
}

func DBUpdate[T any](records *[]T, columnsToInclude ...string) error {
	conn := ScyllaConnect()

	var newType T
	scyllaTable := MakeScyllaTable(newType)

	columnsToUpdate := []BDColumn{}
	columnsWhere := []BDColumn{}
	includeAll := len(columnsToInclude) == 0 || Contains(columnsToInclude, "*")

	for _, e := range scyllaTable.Columns {
		if e.IsPrimaryKey > 0 {
			columnsWhere = append(columnsWhere, e)
			continue
		}
		if includeAll || Contains(columnsToInclude, e.Name) {
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

	queryStr := MakeQueryStatement(queryStatements)
	Log(queryStr)
	if err := conn.Query(queryStr).Exec(); err != nil {
		// Log(queryStr)
		return err
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

type QueryParams struct {
	Type      int8
	Columns   []string
	Value     any
	Values    []any
	Operator  string
	Connector string
	Group     int32
}

type QuerySelect[T any] struct {
	Records        *[]T
	ColumnsToAvoid []string
	ComandsWhere   []QueryParams
	Limit          int32
	GroupCount     int32
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

func (e *QuerySelect[T]) addOperator(name string, value any) *QuerySelect[T] {
	errMsg, current := e.checkError(name)
	if len(errMsg) > 0 {
		panic(errMsg)
	}
	current.Operator = name
	current.Value = value
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
		return e.addOperator("=", values[0])
	} else {
		return e.addOperatorValues("=", values)
	}
}
func (e *QuerySelect[T]) GreatThan(value any) *QuerySelect[T] {
	return e.addOperator(">", value)
}
func (e *QuerySelect[T]) GreatEq(value any) *QuerySelect[T] {
	return e.addOperator(">=", value)
}
func (e *QuerySelect[T]) LessThan(value any) *QuerySelect[T] {
	return e.addOperator("<", value)
}
func (e *QuerySelect[T]) LessEq(value any) *QuerySelect[T] {
	return e.addOperator("<=", value)
}
func (e *QuerySelect[T]) In(values []any) *QuerySelect[T] {
	return e.addOperatorValues("IN", values)
}
func (e *QuerySelect[T]) IN(values ...any) *QuerySelect[T] {
	return e.addOperatorValues("IN", values)
}
func (e *QuerySelect[T]) Contains(value any) *QuerySelect[T] {
	return e.addOperator("CONTAINS", value)
}
func (e *QuerySelect[T]) Like(value any) *QuerySelect[T] {
	return e.addOperator("LIKE", value)
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
	Log("Views::", scyllaTable.Name, scyllaTable.Views)

	columnNames := []string{}
	columnsIdxMap := map[int]BDColumn{}

	for _, co := range scyllaTable.Columns {
		columnsIdxMap[len(columnNames)] = co
		columnNames = append(columnNames, co.Name)
	}

	tableName := scyllaTable.Name
	queryStr := "SELECT %v FROM %v"
	indexOperators := MakeSliceInclude([]string{"=", "IN"})

	whereGroups := SliceToMap(e.ComandsWhere, func(e QueryParams) int32 { return e.Group })
	for _, whereGroup := range whereGroups {
		wheres := []string{}
		for _, qp := range whereGroup {
			var wh string

			// Revisa si hay un view para esa columna
			if len(qp.Columns) == 1 {
				if view, ok := scyllaTable.Views[qp.Columns[0]]; ok {
					tableName = view
				}
			}

			// Revisa si se puede utilizar un indice compuesto
			indexCols := ""
			if len(qp.Columns) > 0 && indexOperators.Include(qp.Operator) {
				cols := qp.Columns
				sort.Strings(cols)
				indexCols = strings.Join(cols, "+")
			}

			if index, ok := scyllaTable.Indexes[indexCols]; ok {
				if qp.Operator == "=" {
					equalValues := []string{}
					if len(qp.Values) > 0 {
						for _, v := range qp.Values {
							equalValues = append(equalValues, fmt.Sprintf("%v", v))
						}
					} else {
						vs := strings.Split(fmt.Sprintf("%v", qp.Value), "+")
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
			} else if len(qp.Values) > 0 {
				inValues := []string{}

				for _, v := range qp.Values {
					inValues = append(inValues, parseValueToString(v))
				}
				if qp.Operator == "IN" {
					wh = fmt.Sprintf(`%v IN (%v)`, qp.Columns[0], strings.Join(inValues, ", "))
				} else {
					if len(qp.Values) != len(qp.Columns) {
						return Err("El numero de valores no coincide con el número de columnas.")
					}
					wheresToJoin := []string{}
					for _, col := range qp.Columns {
						wheresToJoin = append(wheresToJoin, fmt.Sprintf(`%v %v %v`, col, qp.Operator, qp.Values[0]))
					}
					wh = strings.Join(wheresToJoin, " AND ")
				}
			} else {
				wh = fmt.Sprintf(`%v %v %v`,
					qp.Columns[0], qp.Operator, parseValueToString(qp.Value))
			}
			wheres = append(wheres, wh)
		}
		queryStr += " WHERE " + strings.Join(wheres, " AND ")
	}

	if len(allowFiltering) > 0 && allowFiltering[0] {
		queryStr += " ALLOW FILTERING"
	}

	queryStr = fmt.Sprintf(queryStr, strings.Join(columnNames, ", "), tableName)
	Log(queryStr)

	iter := conn.Query(queryStr).Iter()
	rd, _ := iter.RowData()
	scanner := iter.Scanner()

	for scanner.Next() {

		rowValues := rd.Values
		err := scanner.Scan(rowValues...)
		if err != nil {
			Log(queryStr)
			return err
		}

		rec := new(T)
		ref := reflect.ValueOf(rec).Elem()

		for idx, column := range columnsIdxMap {
			value := rowValues[idx]
			if value == nil {
				continue
			}
			if mapField, ok := fieldMapping[column.FieldType]; ok {
				field := ref.Field(column.FieldIdx)
				mapField(&field, value, column.IsPointer)
				// Revisa si necesita parsearse un string a un struct como JSON
			} else if column.IsComplexType {
				// Log("complex type::", column.FieldName)
				if vl, ok := value.(*string); ok {
					newStruct := column.RefType.Interface()
					// fmt.Printf("Type: %T \n", newStruct)
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
					// Log("Valor Columna:", column.Name, " | ", strings.TrimSpace(string(*vl)), "Len:", len(*vl), "")
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
		(*e.Records) = append((*e.Records), *rec)
	}

	err := scanner.Err()
	if err != nil {
		Log(queryStr)
		return Err(err)
	}

	return nil
}
