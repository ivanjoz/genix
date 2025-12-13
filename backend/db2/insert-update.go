package db2

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/gocql/gocql"
)

func MakeInsertStatement[T TableSchemaInterface[T]](records *[]T, columnsToExclude ...Coln) []string {
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

func MakeInsertBatch[T TableSchemaInterface[T]](records *[]T, columnsToExclude ...Coln) *gocql.Batch {
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

func Insert[T TableSchemaInterface[T]](records *[]T, columnsToExclude ...Coln) error {

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

func makeUpdateStatementsBase[T TableSchemaInterface[T]](records *[]T, columnsToInclude []Coln, columnsToExclude []Coln, onlyVirtual bool) []string {

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

func MakeUpdateStatements[T TableSchemaInterface[T]](records *[]T, columnsToInclude ...Coln) []string {
	return makeUpdateStatementsBase(records, columnsToInclude, nil, false)
}

func Update[T TableSchemaInterface[T]](records *[]T, columnsToInclude ...Coln) error {

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

func UpdateExclude[T TableSchemaInterface[T]](records *[]T, columnsToExclude ...Coln) error {

	queryStatements := makeUpdateStatementsBase(records, nil, columnsToExclude, false)
	queryInsert := makeQueryStatement(queryStatements)
	if err := QueryExec(queryInsert); err != nil {
		fmt.Println(queryInsert)
		fmt.Println("Error inserting records:", err)
		return err
	}
	return nil
}

func InsertOrUpdate[T TableSchemaInterface[T]](
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
