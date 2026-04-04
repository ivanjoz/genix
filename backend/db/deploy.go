package db

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/viant/xunsafe"
)

type ScyllaController[T TableBaseInterface[E, T], E TableSchemaInterface[E]] struct {
	TableName string
	Table     ScyllaTable[T]
	Schema    TableSchema
}

type CSVResult struct {
	Content   []byte
	RowsCount int32
}

type virtualColumnsRecalcUpdate[T any] struct {
	record                T
	changedVirtualColumns []IColInfo
}

type ScyllaControllerInterface interface {
	GetTable() ScyllaTable[any]
	GetTableName() string
	GetRecords(partValue, limit int32, lastKey any) []any
	GetRecordsGob(partValue, limit int32, lastKey any) ([]byte, error)
	RestoreCSVRecords(partValue int32, content *[]byte) error
	GetRecordsCSV(partValue int32) (CSVResult, error)
	ReloadRecords(partValue int32) error
	RecalcVirtualColumns(partValue int32) error
	ResetCounter(partValue any) error
}

func (e *ScyllaController[T, E]) GetTable() ScyllaTable[any] {
	return ScyllaTable[any](e.Table)
}

func (e *ScyllaController[T, E]) GetTableName() string {
	return e.Table.name
}

func (e *ScyllaController[T, E]) GetRecords(partValue, limit int32, lastKey any) []any {

	records := []T{}
	query := any(Query(&records)).(TableQueryInterface[E])

	pk := e.Table.GetPartKey()
	if partValue > 0 && pk != nil && !pk.IsNil() {
		query.SetWhere(pk.GetName(), "=", partValue)
	}

	// Add lastKey filter if provided (for pagination)
	if lastKey != nil && len(e.Table.keys) > 0 {
		query.SetWhere(e.Table.keys[0].GetName(), ">=", lastKey)
	}

	// Execute the query
	fmt.Println("Obteniendo registros de::", e.Table.name)
	if err := query.Exec(); err != nil {
		fmt.Println("Error al consultar", e.Table.name, ":", err)
		return nil
	}
	fmt.Println("reqgistros obtenidos (1)::", len(records))

	recordsAny := []any{}
	for _, e := range records {
		recordsAny = append(recordsAny, any(e))
	}

	return recordsAny
}

func (e *ScyllaController[T, E]) GetRecordsCSV(partValue int32) (CSVResult, error) {
	scyllaTable := &e.Table
	return exportToCSV(scyllaTable, partValue)
}

func (e *ScyllaController[T, E]) ReloadRecords(partValue int32) error {
	records := []T{}
	query := any(Query(&records)).(TableQueryInterface[E])

	pk := e.Table.GetPartKey()
	if partValue > 0 && pk != nil && !pk.IsNil() {
		query.SetWhere(pk.GetName(), "=", partValue)
	}

	if err := query.Exec(); err != nil {
		fmt.Println("Error al consultar", e.Table.name, ":", err)
		return err
	}

	if len(records) > 0 {
		if err := Insert(&records); err != nil {
			return Err("Error al re-insertar registros:", err)
		}
	}

	return nil
}

func (e *ScyllaController[T, E]) RecalcVirtualColumns(partValue int32) error {
	scyllaTable := getOrCompileScyllaTable(initStructTable[E, T](new(E)))
	virtualColumns := []IColInfo{}
	selectedColumns := make([]IColInfo, 0, len(scyllaTable.columns))
	for _, column := range scyllaTable.columns {
		if column == nil || column.IsNil() {
			continue
		}
		selectedColumns = append(selectedColumns, column)
		// Recalc only persists generated ORM columns; physical source fields stay untouched.
		if column.GetInfo().IsVirtual {
			virtualColumns = append(virtualColumns, column)
		}
	}
	if len(virtualColumns) == 0 {
		return nil
	}

	// Keep query projection order stable so scanned values map back to ORM columns by position.
	slices.SortFunc(selectedColumns, func(leftColumn, rightColumn IColInfo) int {
		return int(leftColumn.GetInfo().Idx - rightColumn.GetInfo().Idx)
	})

	selectColumnNames := make([]string, 0, len(selectedColumns))
	for _, selectedColumn := range selectedColumns {
		selectColumnNames = append(selectColumnNames, selectedColumn.GetName())
	}

	queryStr := fmt.Sprintf("SELECT %v FROM %v", strings.Join(selectColumnNames, ", "), scyllaTable.GetFullName())
	partitionColumn := scyllaTable.GetPartKey()
	if partValue > 0 && partitionColumn != nil && !partitionColumn.IsNil() {
		queryStr = fmt.Sprintf("%v WHERE %v = %v", queryStr, partitionColumn.GetName(), partValue)
	}

	fmt.Printf("RecalcVirtualColumns | table=%s | query=%s\n", scyllaTable.name, queryStr)
	queryIterator := getScyllaConnection().Query(queryStr).Iter()
	rowData, err := queryIterator.RowData()
	if err != nil {
		return Err("RecalcVirtualColumns RowData failed for table", scyllaTable.name, ":", err)
	}

	updatesToApply := []virtualColumnsRecalcUpdate[T]{}
	rowsScanned := 0

	scanner := queryIterator.Scanner()
	for scanner.Next() {
		if err := scanner.Scan(rowData.Values...); err != nil {
			return Err("RecalcVirtualColumns scan failed for table", scyllaTable.name, ":", err)
		}
		rowsScanned++

		record := *new(T)
		recordPointer := xunsafe.AsPointer(&record)
		persistedVirtualValueByName := map[string]string{}

		for valueIndex, selectedColumn := range selectedColumns {
			rawValue := dereferenceScyllaValue(rowData.Values[valueIndex])
			if selectedColumn.GetInfo().IsVirtual {
				persistedVirtualValueByName[selectedColumn.GetName()] = makeVirtualValueSignature(rawValue)
				continue
			}
			// Rebuild the source record from raw DB values so virtual accessors recompute from persisted inputs.
			selectedColumn.SetValue(recordPointer, rawValue)
		}

		changedVirtualColumns := []IColInfo{}
		for _, virtualColumn := range virtualColumns {
			persistedValueSignature := persistedVirtualValueByName[virtualColumn.GetName()]
			recalculatedValueSignature := makeVirtualValueSignature(virtualColumn.GetStatementValue(recordPointer))
			if persistedValueSignature != recalculatedValueSignature {
				changedVirtualColumns = append(changedVirtualColumns, virtualColumn)
				fmt.Printf("RecalcVirtualColumns | changed virtual column=%s | persisted=%s | recalculated=%s\n",
					virtualColumn.GetName(), persistedValueSignature, recalculatedValueSignature)
			}
		}
		if len(changedVirtualColumns) == 0 {
			continue
		}

		updatesToApply = append(updatesToApply, virtualColumnsRecalcUpdate[T]{
			record:                record,
			changedVirtualColumns: changedVirtualColumns,
		})
	}

	if err := queryIterator.Close(); err != nil {
		return Err("RecalcVirtualColumns query close failed for table", scyllaTable.name, ":", err)
	}

	if len(updatesToApply) == 0 {
		fmt.Printf("RecalcVirtualColumns | table=%s | rows_scanned=%d | rows_updated=0\n", scyllaTable.name, rowsScanned)
		return nil
	}

	updateStatements := make([]string, 0, len(updatesToApply))
	for _, updateToApply := range updatesToApply {
		recordPointer := xunsafe.AsPointer(&updateToApply.record)
		setStatements := make([]string, 0, len(updateToApply.changedVirtualColumns))
		for _, column := range updateToApply.changedVirtualColumns {
			setStatements = append(setStatements, fmt.Sprintf(`%v = %v`, column.GetName(), column.GetValue(recordPointer)))
		}

		whereColumns := scyllaTable.keys
		if partitionColumn != nil && !partitionColumn.IsNil() {
			whereColumns = append([]IColInfo{partitionColumn}, whereColumns...)
		}

		whereStatements := make([]string, 0, len(whereColumns))
		for _, whereColumn := range whereColumns {
			whereStatements = append(whereStatements, fmt.Sprintf(`%v = %v`, whereColumn.GetName(), whereColumn.GetValue(recordPointer)))
		}

		updateStatements = append(updateStatements, fmt.Sprintf(
			"UPDATE %v SET %v WHERE %v",
			scyllaTable.GetFullName(),
			Concatx(", ", setStatements),
			Concatx(" and ", whereStatements),
		))
	}

	if len(updateStatements) == 0 {
		return nil
	}

	if err := QueryExecStatements(updateStatements); err != nil {
		return Err("RecalcVirtualColumns update failed for table", scyllaTable.name, ":", err)
	}

	fmt.Printf("RecalcVirtualColumns | table=%s | rows_scanned=%d | rows_updated=%d\n",
		scyllaTable.name, rowsScanned, len(updatesToApply))
	return nil
}

func dereferenceScyllaValue(value any) any {
	if value == nil {
		return nil
	}

	valueRef := reflect.ValueOf(value)
	for valueRef.Kind() == reflect.Pointer {
		if valueRef.IsNil() {
			return nil
		}
		valueRef = valueRef.Elem()
	}

	return valueRef.Interface()
}

func makeVirtualValueSignature(value any) string {
	if value == nil {
		return "<nil>"
	}

	value = dereferenceScyllaValue(value)
	if value == nil {
		return "<nil>"
	}

	valueRef := reflect.ValueOf(value)
	switch valueRef.Kind() {
	case reflect.Slice, reflect.Array:
		parts := make([]string, 0, valueRef.Len())
		for valueIndex := 0; valueIndex < valueRef.Len(); valueIndex++ {
			parts = append(parts, makeVirtualValueSignature(valueRef.Index(valueIndex).Interface()))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", valueRef.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", valueRef.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", valueRef.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", valueRef.Bool())
	case reflect.String:
		return valueRef.String()
	default:
		if byteSlice, ok := value.([]byte); ok {
			return fmt.Sprintf("%v", byteSlice)
		}
		return fmt.Sprintf("%v", value)
	}
}

func (e *ScyllaController[T, E]) ResetCounter(partValue any) error {
	scyllaTable := &e.Table

	// Sequence reset only applies to partitioned tables with explicit autoincrement usage.
	partitionColumn := scyllaTable.GetPartKey()
	if partitionColumn == nil || partitionColumn.IsNil() {
		return nil
	}
	if !scyllaTable.useSequences || scyllaTable.autoincrementCol == nil {
		return nil
	}
	if len(scyllaTable.keys) == 0 {
		return Err("ResetCounter requires at least one key column for table:", scyllaTable.name)
	}
	if partValue == nil {
		return Err("ResetCounter requires a non-nil partition value for table:", scyllaTable.name)
	}

	// Read max persisted key in the target partition to align sequence with current data.
	keyColumn := scyllaTable.keys[0]
	// Use column metadata to validate the key once, then reuse the shared numeric converter.
	switch keyColumn.GetType().Type {
	case 2, 3, 4, 5:
	default:
		return Err("ResetCounter only supports numeric key types. table:", scyllaTable.name, "key:", keyColumn.GetName())
	}

	maxValueQuery := fmt.Sprintf(
		"SELECT max(%v) FROM %v WHERE %v = ?",
		keyColumn.GetName(), scyllaTable.GetFullName(), partitionColumn.GetName(),
	)

	// Let gocql allocate the aggregate destination types, then normalize the first value.
	queryIterator := getScyllaConnection().Query(maxValueQuery, partValue).Iter()
	rowData, err := queryIterator.RowData()
	if err != nil {
		return Err("ResetCounter max-value query failed for table", scyllaTable.name, ":", err)
	}

	maxKeyValue := int64(0)
	rowScanner := queryIterator.Scanner()
	if rowScanner.Next() {
		if err := rowScanner.Scan(rowData.Values...); err != nil {
			return Err("ResetCounter max-value query failed for table", scyllaTable.name, ":", err)
		}
		if len(rowData.Values) > 0 && rowData.Values[0] != nil {
			maxKeyValue = reflectToInt64(rowData.Values[0])
		}
	}
	if err := queryIterator.Close(); err != nil {
		return Err("ResetCounter max-value query failed for table", scyllaTable.name, ":", err)
	}

	// Counter naming must match the insert path (x{partition}_{table}_{autoincrementPart}).
	counterName := fmt.Sprintf("x%v_%v_%v", partValue, scyllaTable.name, 0)
	currentCounterValue, err := getSequenceCurrentValue(counterName)
	if err != nil {
		return Err("ResetCounter sequence read failed for", counterName, ":", err)
	}

	// Counters are increment-only, so we apply the delta to move to the target absolute value.
	delta := maxKeyValue - currentCounterValue
	if delta == 0 {
		return nil
	}

	updateStatement := fmt.Sprintf("UPDATE %v.sequences SET current_value = current_value + ? WHERE name = ?", scyllaTable.keyspace)
	if err := getScyllaConnection().Query(updateStatement, delta, counterName).Exec(); err != nil {
		return Err("ResetCounter sequence update failed for", counterName, ":", err)
	}

	fmt.Printf("ResetCounter | table=%s | partition=%v | counter=%s | previous=%d | maxKey=%d | delta=%d\n",
		scyllaTable.name, partValue, counterName, currentCounterValue, maxKeyValue, delta)

	return nil
}

func getSequenceCurrentValue(counterName string) (int64, error) {
	result := []Increment{}
	if err := Query(&result).Name.Equals(counterName).Exec(); err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil
	}
	return result[0].CurrentValue, nil
}

func (e *ScyllaController[T, E]) GetRecordsGob(partValue, limit int32, lastKey any) ([]byte, error) {

	gob.Register(*new(T))
	records := e.GetRecords(partValue, limit, lastKey)
	if len(records) == 0 {
		return nil, nil
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(records)
	if err != nil {
		return []byte{}, err
	}

	return buffer.Bytes(), nil
}

func (e *ScyllaController[T, E]) RestoreCSVRecords(partValue int32, content *[]byte) error {
	scyllaTable := &e.Table
	records, err := CsvToRecords(scyllaTable, content, partValue)

	if err != nil {
		return err
	}

	pk := e.Table.GetPartKey()
	if partValue > 0 && pk != nil && !pk.IsNil() {
		statement := fmt.Sprintf(`DELETE FROM %v WHERE %v = %v`, e.Table.GetFullName(), pk.GetName(), partValue)
		if err := QueryExec(statement); err != nil {
			fmt.Println("Error en statement: ", statement)
			return Err("Error al eliminar registros:", err)
		}
	}

	// Insert new records
	fmt.Println("Registros a insertar:", len(records))

	if len(records) > 0 {
		Print(records[0])
	}

	if err := Insert(&records); err != nil {
		return Err("Error al insertar registros:", err)
	}
	return nil
}

type ScyllaColumns struct {
	Name     string
	Type     string
	Keyspace string
	Table    string
}

type ScyllaIndexes struct {
	Name     string
	Table    string
	Keyspace string
	Kind     string
}

var cacheCodePrev int32
var scyllaColumnsSaved []ScyllaColumns
var scyllaIndexesSaved []ScyllaIndexes

func DeployScylla(cacheCode int32, controllers ...ScyllaControllerInterface) {
	var scyllaColumns []ScyllaColumns
	var scyllaIndexes []ScyllaIndexes
	isFetched := false

	if cacheCode > 0 && cacheCode == cacheCodePrev {
		scyllaColumns = scyllaColumnsSaved
		scyllaIndexes = scyllaIndexesSaved
	} else {
		fmt.Println("Obteniendo columnas...")
		// Query system_schema.columns
		query := fmt.Sprintf("SELECT keyspace_name, table_name, column_name, type FROM system_schema.columns WHERE keyspace_name = '%s'", connParams.Keyspace)

		session := getScyllaConnection()
		iter := session.Query(query).Iter()
		var col ScyllaColumns
		for iter.Scan(&col.Keyspace, &col.Table, &col.Name, &col.Type) {
			scyllaColumns = append(scyllaColumns, col)
		}
		if err := iter.Close(); err != nil {
			panic("Error al obtener columnas:" + err.Error())
		}
		fmt.Println("Scylla columns obtenidas::", len(scyllaColumns))

		fmt.Println("Obteniendo Indices...")
		// Query system_schema.indexes
		indexQuery := fmt.Sprintf("SELECT keyspace_name, table_name, index_name, kind FROM system_schema.indexes WHERE keyspace_name = '%s'", connParams.Keyspace)

		iter = session.Query(indexQuery).Iter()
		var idx ScyllaIndexes
		for iter.Scan(&idx.Keyspace, &idx.Table, &idx.Name, &idx.Kind) {
			scyllaIndexes = append(scyllaIndexes, idx)
		}
		if err := iter.Close(); err != nil {
			panic("Error al obtener índices:" + err.Error())
		}
		fmt.Println("Índices obtenidos:", len(scyllaIndexes))
		isFetched = true
	}

	tableColumnsMap := map[string][]ScyllaColumns{}

	for _, e := range scyllaColumns {
		key := fmt.Sprintf("%v.%v", e.Keyspace, e.Table)
		tableColumnsMap[key] = append(tableColumnsMap[key], e)
	}

	if isFetched {
		tablesNames := []string{}

		for tableName, columns := range tableColumnsMap {
			tablesNames = append(tablesNames, tableName)

			s1 := strings.Split(tableName, "_")
			if s1[len(s1)-1] == "view" {
				continue
			}
			fmt.Println("✔ Table =", tableName)
			columnsNames := []string{}
			for _, c := range columns {
				columnsNames = append(columnsNames, fmt.Sprintf("%v(%v)", c.Name, c.Type))
			}
			fmt.Println("  Columns =", strings.Join(columnsNames, ", "))
		}

		fmt.Println("Tables::", tablesNames)
	}

	tableIndexesMap := map[string][]string{}
	for _, e := range scyllaIndexes {
		key := fmt.Sprintf("%v.%v", e.Keyspace, e.Table)
		tableIndexesMap[key] = append(tableIndexesMap[key], e.Name)
	}

	for _, controller := range controllers {
		table := controller.GetTable()
		tableName := table.GetFullName()

		fmt.Println("Struct Type:", controller.GetTableName())

		originColumns := tableColumnsMap[tableName]
		fmt.Println("current columns::", len(originColumns))

		// Si no existe la tabla entonces la crea...
		if len(originColumns) == 0 {
			Logx(6, "No se encontró la tabla: "+tableName+"\n")
			Logx(2, fmt.Sprintf(`Creando tabla "%v"...`+"\n", tableName))

			columnsTypes := []string{}
			for _, e := range table.columns {
				columnsTypes = append(columnsTypes, e.GetName()+" "+e.GetType().ColType)
			}

			keys := []string{}
			for _, key := range table.keys {
				keys = append(keys, key.GetName())
			}

			pk := strings.Join(keys, ", ")
			partKey := table.GetPartKey()
			if partKey != nil && !partKey.IsNil() && len(partKey.GetName()) > 0 {
				pk = fmt.Sprintf("(%v), %v", partKey.GetName(), pk)
			}

			query := `
		CREATE TABLE %v (
			%v,
			PRIMARY KEY (%v)
		)
			WITH caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
			and compaction = {'class': 'SizeTieredCompactionStrategy'}
			and compression = {'compression_level': '3', 'sstable_compression': 'org.apache.cassandra.io.compress.ZstdCompressor'}
			and dclocal_read_repair_chance = 0
			and speculative_retry = '99.0PERCENTILE';
		`
			query = fmt.Sprintf(query, tableName, strings.Join(columnsTypes, ", "), pk)
			fmt.Println(query)

			err := QueryExec(query)
			if err != nil {
				panic(fmt.Sprintf(`Error creando tabla "%v" | `, tableName) + err.Error())
			}

			Logx(2, fmt.Sprintf(`Tabla creada: "%v"`+"\n", tableName))
		}

		columnsSchemaMap := map[string]IColInfo{}
		for name, col := range table.columnsMap {
			columnsSchemaMap[name] = col
		}
		columnsNoMapeadas := map[string]string{}

		for _, originColumn := range originColumns {
			if column, ok := columnsSchemaMap[originColumn.Name]; ok {
				if column.GetType().ColType != originColumn.Type {
					Logx(5, fmt.Sprintf(`La columna "%v" está definida con type "%v", pero en el Struct está con "%v" equivalente a "%v"`+"\n", originColumn.Name, originColumn.Type, column.GetType().FieldType, column.GetType().ColType))
				}
				delete(columnsSchemaMap, originColumn.Name)
			} else {
				columnsNoMapeadas[originColumn.Name] = originColumn.Type
			}
		}

		if len(originColumns) > 0 {
			// Revisa las columnas existentes en la BD pero no mapeadas
			for name, dbType := range columnsNoMapeadas {
				Logx(5, fmt.Sprintf(`La columna "%v" con type "%v" existe en la BD origen no está mapeada en el Struct.`+"\n", name, dbType))
			}
			// Revisa las columnas que deben crearse en BD
			for _, column := range columnsSchemaMap {
				Logx(5, fmt.Sprintf(`La columna "%v" con struct type "%v" no existe en la BD de origen.`+"\n", column.GetName(), column.GetType().FieldType))
				query := fmt.Sprintf(`ALTER TABLE %v ADD %v %v`, tableName, column.GetName(), column.GetType().ColType)
				fmt.Printf(`Ejecutando agregar columna "%v"...`+"\n", query)

				if err := QueryExec(query); err != nil {
					panic(fmt.Sprintf(`Error agregando columna "%v" | %v`, column.GetName(), err))
				}
				Logx(2, fmt.Sprintf(`Columna Agregada: "%v"`+"\n", column.GetName()))
			}
		}

		// Revisa si posee índices, en su defecto los crea
		tableIndexes := tableIndexesMap[tableName]

		for _, index := range table.indexes {
			if slices.Contains(tableIndexes, index.name) {
				continue
			}
			Logx(5, fmt.Sprintf(`No se encontró el índice "%v" en "%v". Creando...`+"\n", index.name, tableName))

			createScript := index.getCreateScript()
			fmt.Println(createScript)
			if err := QueryExec(createScript); err != nil {
				fmt.Println(err)
				panic(fmt.Sprintf(`Error creando el índice "%v" en %v`, index.name, tableName))
			}
			Logx(2, fmt.Sprintf(`Index created "%v"`+"\n", index.name))
		}

		// Revisa si posee views, en su defecto las crea
		for _, view := range table.views {
			name := table.keyspace + "." + view.name
			if _, ok := tableColumnsMap[name]; ok {
				continue
			}
			Logx(5, fmt.Sprintf(`No se encontró la view "%v" en la tabla "%v". Preparando creación...`+"\n", view.name, table.name))

			createScript := view.getCreateScript()
			fmt.Println(createScript)
			if err := QueryExec(createScript); err != nil {
				fmt.Println(err)
				panic(fmt.Sprintf(`Error creando la view "%v" en %v`, view.name, tableName))
			}

			Logx(2, fmt.Sprintf(`View created "%v"`+"\n", view.name))
		}
	}

	// Cache the results if cacheCode is provided
	if cacheCode > 0 {
		cacheCodePrev = cacheCode
		scyllaColumnsSaved = scyllaColumns
		scyllaIndexesSaved = scyllaIndexes
	}
}
