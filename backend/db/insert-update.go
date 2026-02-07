package db

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gocql/gocql"
	"github.com/viant/xunsafe"
)

func handlePreInsert[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any],
) error {
	if scyllaTable.autoincrementCol == nil && len(scyllaTable.keyIntPacking) == 0 {
		return nil
	}

	partitionColumn := scyllaTable.GetPartKey()	
	// Group records by composite key: partition value + autoincrementPart value
	groups := map[string][]*T{}
	
	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)
		
		// Get partition key value
		partitionValue := int32(0)
		if partitionColumn != nil {
			partitionValue = int32(convertToInt64(partitionColumn.GetRawValue(ptr)))
		}
		
		// Get autoincrement part value (0 if not defined)
		autoPartVal := int64(0)
		if scyllaTable.autoincrementPart != nil {
			autoPartVal = convertToInt64(scyllaTable.autoincrementPart.GetRawValue(ptr))
		}
		
		// Group key is concatenation of partition value + autoincrementPart value
		key := fmt.Sprintf("%d|%v", partitionValue, autoPartVal)
		groups[key] = append(groups[key], rec)
	}

	for partKey, group := range groups {
		partValues := strings.Split(partKey, "|")
		
		// Filter to only include records that need autoincrement (ID == 0)
		recordsNeedingAutoincrement := []*T{}
		recordNeedsAutoincrement := map[*T]bool{}
		
		for _, rec := range group {
			ptr := xunsafe.AsPointer(rec)
			// Check if the primary key has a value of 0
			var keyVal int64 = 0
			if len(scyllaTable.keys) > 0 {
				rawKeyVal := scyllaTable.keys[0].GetRawValue(ptr)
				keyVal = convertToInt64(rawKeyVal)
			}
			
			// Only apply autoincrement if key is exactly 0
			if keyVal <= 0 {
				recordsNeedingAutoincrement = append(recordsNeedingAutoincrement, rec)
				recordNeedsAutoincrement[rec] = true
			}
		}
		
		var counterVal int64
		var err error
		
		if scyllaTable.autoincrementCol != nil && len(recordsNeedingAutoincrement) > 0 {
			// Determine counter name
			counterName := fmt.Sprintf("x%v_%v_%v", partValues[0], scyllaTable.name, partValues[1])

			// Get range of IDs - only for records that need autoincrement
			keyspace := strings.Split(scyllaTable.GetFullName(), ".")[0]
			counterVal, err = GetCounter(keyspace, counterName, len(recordsNeedingAutoincrement))
			if err != nil {
				return err
			}
			// GetCounter returns the value after increment, so we need the first value of the range
			counterVal = counterVal - int64(len(recordsNeedingAutoincrement)) + 1
		}

		for _, rec := range group {
			ptr := xunsafe.AsPointer(rec)
			var currentAutoVal int64

			// Only apply autoincrement if schema has Autoincrement defined AND record ID is 0
			if scyllaTable.autoincrementCol != nil && recordNeedsAutoincrement[rec] {
				currentAutoVal = counterVal
				counterVal++

				colInfo := scyllaTable.autoincrementCol.(*columnInfo)
				if colInfo.autoincrementRandSize > 0 {
					suffix := GetRandomInt64(colInfo.autoincrementRandSize)
					currentAutoVal = currentAutoVal*Pow10Int64(int64(colInfo.autoincrementRandSize)) + suffix
				}

				// If not packing, set directly
				if len(scyllaTable.keyIntPacking) == 0 {
					scyllaTable.autoincrementCol.SetValue(ptr, currentAutoVal)
				}
			}

			if len(scyllaTable.keyIntPacking) > 0 {
				var packedValue int64
				remainingDigits := int64(19)
				for i, col := range scyllaTable.keyIntPacking {
					if col == nil {
						continue
					}
					var val int64
					if col == scyllaTable.autoincrementCol {
						val = currentAutoVal
					} else {
						val = convertToInt64(col.GetRawValue(ptr))
					}

					colPackingInfo := col.(*columnInfo)
					decSize := int64(colPackingInfo.decimalSize)
					// If it's the last one and size is 0, it takes all remaining space
					if i == len(scyllaTable.keyIntPacking)-1 && decSize == 0 {
						decSize = remainingDigits
					}

					remainingDigits -= decSize
					if remainingDigits < 0 {
						remainingDigits = 0
					}

					shift := Pow10Int64(remainingDigits)
					packedValue += val * shift
				}
				// Set into the only key
				scyllaTable.keys[0].SetValue(ptr, packedValue)
			}
		}
	}

	return nil
}

func MakeInsertStatement[T TableBaseInterface[E, T], E TableSchemaInterface[E]](records *[]T, columnsToExclude ...Coln) []string {
	refTable := initStructTable[E, T](new(E))
	scyllaTable := makeTable(refTable)

	columns := []IColInfo{}
	if len(columnsToExclude) > 0 {
		columsToExcludeNames := []string{}
		for _, e := range columnsToExclude {
			columsToExcludeNames = append(columsToExcludeNames, e.GetInfo().Name)
		}
		for _, col := range scyllaTable.columns {
			if !slices.Contains(columsToExcludeNames, col.GetName()) {
				columns = append(columns, col)
			}
		}
	} else {
		columns = scyllaTable.columns
	}

	columnsNames := []string{}
	for _, col := range columns {
		columnsNames = append(columnsNames, col.GetName())
	}

	queryStrInsert := fmt.Sprintf(`INSERT INTO %v (%v) VALUES `,
		scyllaTable.GetFullName(), strings.Join(columnsNames, ", "))

	queryStatements := []string{}

	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)
		// fmt.Println("Type:", reflect.TypeOf(rec).String())

		recordInsertValues := []string{}

		for _, col := range columns {
			value := col.GetValue(ptr)
			recordInsertValues = append(recordInsertValues, fmt.Sprintf("%v", value))
		}

		statement := /*" " +*/ queryStrInsert + "(" + strings.Join(recordInsertValues, ", ") + ")"
		queryStatements = append(queryStatements, statement)
	}
	return queryStatements
}

func Table[T TableBaseInterface[E, T], E TableSchemaInterface[E]]() *E {
	return initStructTable[E, T](new(E))
}

func makeInsertBatch[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, scyllaTable ScyllaTable[any], columnsToExclude ...Coln,
) *gocql.Batch {

	columns := []IColInfo{}
	if len(columnsToExclude) > 0 {
		columsToExcludeNames := []string{}
		for _, e := range columnsToExclude {
			columsToExcludeNames = append(columsToExcludeNames, e.GetInfo().Name)
		}
		for _, col := range scyllaTable.columns {
			if !slices.Contains(columsToExcludeNames, col.GetName()) {
				columns = append(columns, col)
			}
		}
	} else {
		columns = scyllaTable.columns
	}

	columnsNames := []string{}
	columnPlaceholders := []string{}
	for _, col := range columns {
		columnsNames = append(columnsNames, col.GetName())
		columnPlaceholders = append(columnPlaceholders, "?")
	}

	session := getScyllaConnection()
	batch := session.NewBatch(gocql.UnloggedBatch)

	queryStrInsert := fmt.Sprintf(`INSERT INTO %v (%v) VALUES (%v)`,
		scyllaTable.GetFullName(), strings.Join(columnsNames, ", "), strings.Join(columnPlaceholders, ", "))

	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)
		values := []any{}

		for _, col := range columns {
			var value any
			value = col.GetStatementValue(ptr)
			if value == nil {
				value = col.GetValue(ptr)
			}
			values = append(values, value)
		}

		// fmt.Println("VALUES::")
		// fmt.Println(values)
		batch.Query(queryStrInsert, values...)
	}
	return batch
}

func Insert[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToExclude ...Coln,
) error {
	refTable := initStructTable[E, T](new(E))
	scyllaTable := makeTable(refTable)

	if err := handlePreInsert(records, scyllaTable); err != nil {
		return err
	}

	session := getScyllaConnection()
	//fmt.Println("BATCH (1)::")
	queryBatch := makeInsertBatch(records, scyllaTable, columnsToExclude...)

	//fmt.Println("BATCH (2)::")
	//fmt.Println(queryBatch.Entries)

	if err := session.ExecuteBatch(queryBatch); err != nil {
		fmt.Println("Error inserting records:", err)
		return err
	}

	return nil
}

func InsertOne[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	record T, columnsToExclude ...Coln,
) error {
	return Insert(&[]T{record}, columnsToExclude...)
}

func makeUpdateStatementsBase[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToInclude []Coln, columnsToExclude []Coln, onlyVirtual bool,
) []string {

	refTable := initStructTable[E, T](new(E))
	scyllaTable := makeTable(refTable)
	columnsToUpdate := []IColInfo{}

	if len(columnsToInclude) > 0 {
		for _, col_ := range columnsToInclude {
			col := scyllaTable.columnsMap[col_.GetName()]
			if col == nil {
				Print(col)
				panic("No se encontró la columna (update):" + col_.GetName())
			}
			if slices.Contains(scyllaTable.keysIdx, col.GetInfo().Idx) {
				msg := fmt.Sprintf(`Table "%v": The column "%v" can't be updated because is part of primary key.`, scyllaTable.name, col.GetName())
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
			isExcluded := slices.Contains(columnsToExcludeNames, col.GetName())
			if !col.GetInfo().IsVirtual && !isExcluded && !slices.Contains(scyllaTable.keysIdx, col.GetInfo().Idx) {
				columnsToUpdate = append(columnsToUpdate, col)
			}
		}
	}

	columnsIdx := []int16{}
	for _, col := range columnsToUpdate {
		columnsIdx = append(columnsIdx, col.GetInfo().Idx)
	}
	columnsIncluded := slices.Concat(scyllaTable.keysIdx, columnsIdx)
	pk := scyllaTable.GetPartKey()
	if pk != nil && !pk.IsNil() {
		columnsIncluded = append(columnsIncluded, pk.GetInfo().Idx)
	}

	//Revisa si hay columnas que deben actualizarse juntas para los índices calculados
	for _, indexViews := range scyllaTable.indexViews {
		if indexViews.column.GetInfo().IsVirtual {
			includedCols := []int16{}
			notIncludedCols := []int16{}

			for _, colIdx := range indexViews.columnsIdx {
				if slices.Contains(columnsIncluded, colIdx) {
					includedCols = append(includedCols, colIdx)
				} else {
					notIncludedCols = append(notIncludedCols, colIdx)
				}
			}

			if len(includedCols) > 0 && len(notIncludedCols) > 0 {
				colnames := []string{}
				for _, colname := range indexViews.columns {
					if pk != nil && !pk.IsNil() && pk.GetName() == colname {
						continue
					}
					colnames = append(colnames, fmt.Sprintf(`"%v"`, colname))
				}

				includedColsNames := []string{}
				for _, idx := range notIncludedCols {
					includedColsNames = append(includedColsNames, scyllaTable.columnsIdxMap[idx].GetName())
				}

				msg := fmt.Sprintf(`Table "%v": A composit index/view requires the columns %v be updated together. Not Included: %v`, scyllaTable.name, strings.Join(colnames, ", "), strings.Join(includedColsNames, ", "))
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
			if col.GetInfo().IsVirtual {
				columnsToUpdate = append(columnsToUpdate, col)
			}
		}
	}

	columnsWhere := scyllaTable.keys

	pk = scyllaTable.GetPartKey()
	if pk != nil && !pk.IsNil() {
		columnsWhere = append([]IColInfo{pk}, columnsWhere...)
	}

	queryStatements := []string{}

	for i := range *records {
		rec := &(*records)[i]
		ptr := xunsafe.AsPointer(rec)

		setStatements := []string{}
		for _, col := range columnsToUpdate {
			v := col.GetValue(ptr)
			setStatements = append(setStatements, fmt.Sprintf(`%v = %v`, col.GetName(), v))
		}

		whereStatements := []string{}
		for _, col := range columnsWhere {
			v := col.GetValue(ptr)
			whereStatements = append(whereStatements, fmt.Sprintf(`%v = %v`, col.GetName(), v))
		}

		queryStatement := fmt.Sprintf(
			"UPDATE %v SET %v WHERE %v",
			scyllaTable.GetFullName(), Concatx(", ", setStatements), Concatx(" and ", whereStatements),
		)

		queryStatements = append(queryStatements, queryStatement)
	}

	return queryStatements
}

func MakeUpdateStatements[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToInclude ...Coln,
) []string {
	return makeUpdateStatementsBase(records, columnsToInclude, nil, false)
}

func Update[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToInclude ...Coln,
) error {

	if len(columnsToInclude) == 0 {
		panic("No se incluyeron columnas a actualizar.")
	}

	queryStatements := makeUpdateStatementsBase(records, columnsToInclude, nil, false)
	queryUpdate := makeQueryStatement(queryStatements)

	if err := QueryExec(queryUpdate); err != nil {
		fmt.Println(queryUpdate)
		fmt.Println("Error updating records:", err)
		return err
	}
	return nil
}

func UpdateOne[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	record T, columnsToInclude ...Coln,
) error {
	return Update(&[]T{record}, columnsToInclude...)
}

func UpdateExclude[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
	records *[]T, columnsToExclude ...Coln,
) error {

	queryStatements := makeUpdateStatementsBase(records, nil, columnsToExclude, false)
	queryInsert := makeQueryStatement(queryStatements)

	if err := QueryExec(queryInsert); err != nil {
		fmt.Println(queryInsert)
		fmt.Println("Error inserting records:", err)
		return err
	}
	return nil
}

func InsertOrUpdate[T TableBaseInterface[E, T], E TableSchemaInterface[E]](
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
		if err := UpdateExclude(&recordsToUpdate, columnsToExcludeUpdate...); err != nil {
			return err
		}
	}

	if len(recordsToInsert) > 0 {
		fmt.Println("Registros a insertar:", len(recordsToInsert))
		if err := Insert(&recordsToInsert, columnsToExcludeInsert...); err != nil {
			return err
		}
	}

	return nil
}
