package db

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strings"
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

type ScyllaControllerInterface interface {
	GetTable() ScyllaTable[any]
	GetTableName() string
	GetRecords(partValue, limit int32, lastKey any) []any
	GetRecordsGob(partValue, limit int32, lastKey any) ([]byte, error)
	RestoreCSVRecords(partValue int32, content *[]byte) error
	GetRecordsCSV(partValue int32) (CSVResult, error)
	ReloadRecords(partValue int32) error
 //	ResetCounter(partValue any) error
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
