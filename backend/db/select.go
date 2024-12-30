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

	// Between Statement
	if len(query.between.From) > 0 {
		statements = append(statements, query.between)
		for _, st := range query.between.From {
			col := scyllaTable.columnsMap[st.Col]
			colsWhereIdx = append(colsWhereIdx, col.Idx)
			if !slices.Contains(scyllaTable.keysIdx, col.Idx) {
				allAreKeys = false
			}
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

	statementsSelected := []ColumnStatement{}
	statementsRemain := []ColumnStatement{}
	statementsLogs_ := []ColumnStatement{}
	whereStatements := []string{}
	orderColumn := scyllaTable.keys[0]

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
			for _, st := range statements {
				if len(st.From) > 0 {
					isIncluded := true
					for _, e := range st.From {
						if !slices.Contains(viewOrIndex.columns, e.Col) {
							isIncluded = false
							break
						}
						statementsLogs_ = append(statementsLogs_, e)
					}
					if isIncluded {
						statementsSelected = append(statementsSelected, st)
					}
				} else if slices.Contains(viewOrIndex.columns, st.Col) {
					statementsSelected = append(statementsSelected, st)
					statementsLogs_ = append(statementsLogs_, st)
				} else {
					statementsRemain = append(statementsRemain, st)
				}
			}
			orderColumn = viewOrIndex.column
			whereStatements = viewOrIndex.getStatement(statementsSelected...)
		} else {
			// No es necesario cambiar nada del query
			statementsRemain = statements
		}
	} else {
		statementsRemain = statements
	}

	statementsLogs := []string{}
	for _, st := range append(statementsLogs_, statementsRemain...) {
		statementsLogs = append(statementsLogs, fmt.Sprintf("%v %v %v", st.Col, st.Operator, st.Value))
	}

	fmt.Println("Statements:", strings.Join(statementsLogs, " | "))

	if len(whereStatements) == 0 {
		whereStatements = append(whereStatements, "")
	}

	// fmt.Println("where statements::", whereStatements)
	whereStatementsRemainSects := []string{}
	for _, st := range statementsRemain {
		if len(st.From) > 0 {
			for i := range st.From {
				colname := st.From[i].Col
				w1 := fmt.Sprintf("%v >= %v", colname, st.From[i].GetValue())
				whereStatementsRemainSects = append(whereStatementsRemainSects, w1)
				w2 := fmt.Sprintf("%v < %v", colname, st.To[i].GetValue())
				whereStatementsRemainSects = append(whereStatementsRemainSects, w2)
			}
		} else {
			whereStatementsRemainSects = append(whereStatementsRemainSects,
				fmt.Sprintf("%v %v %v", st.Col, st.Operator, st.GetValue()))
		}
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
		if len(query.orderBy) > 0 {
			whereStatement += " " + fmt.Sprintf(query.orderBy, orderColumn.Name)
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
		doScan := func() error {
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
						// fmt.Println("complex type::", column.FieldName, "|", column.FieldType)
						if vl, ok := value.(*[]uint8); ok {
							if len(*vl) <= 3 {
								continue
							}
							newElm := reflect.New(column.RefType).Elem()
							err = cbor.Unmarshal([]byte(*vl), newElm.Addr().Interface())
							if err != nil {
								fmt.Println("Error al convertir: ", newElm, "|", err.Error())
							}
							// fmt.Println("col complex:", column.Name, " | ", newElm, " | L:", len(*vl))
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
		}

		eg.Go(func() error {
			err := doScan()
			if err != nil {
				if strings.Contains(err.Error(), "no hosts available") {
					scyllaSession = nil
					fmt.Println("Reconectando con ScyllaDB...")
					getScyllaConnection()
					err = doScan()
				}
			}
			return err
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
