package db

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type ScyllaColumns struct {
	Name     string
	Type     string
	Keyspace string
	Table    string
}

func (e ScyllaColumns) Table_() CoStr    { return CoStr{"table_name"} }
func (e ScyllaColumns) Keyspace_() CoStr { return CoStr{"keyspace_name"} }
func (e ScyllaColumns) Name_() CoStr     { return CoStr{"column_name"} }
func (e ScyllaColumns) Type_() CoStr     { return CoStr{"type"} }

func (e ScyllaColumns) GetSchema() TableSchema {
	return TableSchema{
		Keyspace: "system_schema",
		Name:     "columns",
		Keys:     []Coln{e.Keyspace_()},
	}
}

type ScyllaViews struct {
	ViewName string
	Table    string
}

func (e ScyllaViews) ViewName_() CoStr { return CoStr{"view_name"} }
func (e ScyllaViews) Table_() CoStr    { return CoStr{"base_table_name"} }

type ScyllaIndexes struct {
	Name     string
	Table    string
	Keyspace string
	Kind     string
}

func (e ScyllaIndexes) GetSchema() TableSchema {
	return TableSchema{
		Keyspace: "system_schema",
		Name:     "indexes",
		Keys:     []Coln{e.Keyspace_()},
	}
}

func (e ScyllaIndexes) Table_() CoStr    { return CoStr{"table_name"} }
func (e ScyllaIndexes) Keyspace_() CoStr { return CoStr{"keyspace_name"} }
func (e ScyllaIndexes) Name_() CoStr     { return CoStr{"index_name"} }
func (e ScyllaIndexes) Kind_() CoStr     { return CoStr{"kind"} }

func DeployScylla(structTables ...any) {

	fmt.Println("Obteniendo columnas...")
	scyllaColumns := Select(func(q *Query[ScyllaColumns], col ScyllaColumns) {
		q.Where(col.Keyspace_().Equals(connParams.Keyspace))
	})

	if scyllaColumns.Err != nil {
		panic("Error:" + scyllaColumns.Err.Error())
	}

	fmt.Println("Scylla columns obtenidas::", len(scyllaColumns.Records))

	tableColumnsMap := map[string][]ScyllaColumns{}

	for _, e := range scyllaColumns.Records {
		key := fmt.Sprintf("%v.%v", e.Keyspace, e.Table)
		tableColumnsMap[key] = append(tableColumnsMap[key], e)
	}

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

	fmt.Println("Obteniendo Indices...")
	scyllaIndexes := Select(func(q *Query[ScyllaIndexes], col ScyllaIndexes) {
		q.Where(col.Keyspace_().Equals(connParams.Keyspace))
	})

	if scyllaIndexes.Err != nil {
		panic("Error:" + scyllaIndexes.Err.Error())
	}

	tableIdexesMap := map[string][]string{}
	for _, e := range scyllaIndexes.Records {
		key := fmt.Sprintf("%v.%v", e.Keyspace, e.Table)
		tableIdexesMap[key] = append(tableIdexesMap[key], e.Name)
	}

	for _, st := range structTables {
		var table scyllaTable[any]
		fmt.Println("Struct Type:", reflect.TypeOf(st).Name())

		if ITableSchema, ok := st.(TableSchemaInterface); ok {
			table = MakeTable(ITableSchema.GetSchema(), st)
		} else {
			panic("El Type no implementa TableSchemaInterface")
		}

		tableName := table.fullName()

		originColumns := tableColumnsMap[tableName]
		fmt.Println("current columns::", len(originColumns))

		// Si no existe la tabla entonces la crea...
		if len(originColumns) == 0 {
			Logx(6, "No se encontró la tabla: "+tableName+"\n")
			Logx(2, fmt.Sprintf(`Creando tabla "%v"...`+"\n", tableName))

			columnsTypes := []string{}
			for _, e := range table.columns {
				columnsTypes = append(columnsTypes, e.Name+" "+e.Type)
			}

			keys := []string{}
			for _, key := range table.keys {
				keys = append(keys, key.Name)
			}

			pk := strings.Join(keys, ", ")
			if len(table.partKey.Name) > 0 {
				pk = fmt.Sprintf("(%v), %v", table.partKey.Name, pk)
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

		columnsSchemaMap := map[string]*columnInfo{}
		for name, col := range table.columnsMap {
			// core.Log("columnas 11:", name, "|", col.FieldName)
			columnsSchemaMap[name] = col
		}
		columnsNoMapeadas := map[string]string{}

		for _, originColumn := range originColumns {
			if column, ok := columnsSchemaMap[originColumn.Name]; ok {
				if column.Type != originColumn.Type {
					Logx(5, fmt.Sprintf(`La columna "%v" está definida con type "%v", pero en el Struct está con "%v" equivalente a "%v"\n`, originColumn.Name, originColumn.Type, column.FieldType, column.Type))
				}
				delete(columnsSchemaMap, originColumn.Name)
			} else {
				columnsNoMapeadas[originColumn.Name] = originColumn.Type
			}
		}

		if len(originColumns) > 0 {
			// Revisa las columnas existentes en la BD per no mapeadas
			for name, dbType := range columnsNoMapeadas {
				Logx(5, fmt.Sprintf(`La columna "%v" con type "%v" existe en la BD origen no está mapeada en el Struct.`+"\n", name, dbType))
			}
			// Revisa las columnas que deben crearse en BD
			for _, column := range columnsSchemaMap {
				Logx(5, fmt.Sprintf(`La columna "%v" con struct type "%v" no existe en la BD de origen.`+"\n", column.Name, column.FieldType))
				query := fmt.Sprintf(`ALTER TABLE %v ADD %v %v`, tableName, column.Name, column.Type)
				fmt.Printf(`Ejecutando agregar columna "%v"...`+"\n", query)

				if err := QueryExec(query); err != nil {
					panic(fmt.Sprintf(`Error agregando columna "%v" | %v`, column.Name, err))
				}
				Logx(2, fmt.Sprintf(`Columna Agregada: "%v"`+"\n", column.Name))
			}
		}

		//Revisa si posee índices, en su defecto los crea
		tableIndexes := tableIdexesMap[tableName]

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

		//Revisa si posee views, en su defecto las crea
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
				panic(fmt.Sprintf(`Error creando el índice "%v" en %v`, view.name, tableName))
			}

			Logx(2, fmt.Sprintf(`View created "%v"`+"\n", view.name))
		}
	}
}

func RecalcVirtualColumns[T TableSchemaInterface]() {
	scyllaTable := makeTable(*new(T))

	columnsToUpdateIdx := []int16{}
	for _, viewIndex := range scyllaTable.indexViews {
		if viewIndex.column.IsVirtual {
			for _, idx := range viewIndex.columnsIdx {
				if !slices.Contains(columnsToUpdateIdx, idx) {
					columnsToUpdateIdx = append(columnsToUpdateIdx, idx)
				}
			}
		}
	}

	if len(columnsToUpdateIdx) == 0 {
		panic("no hay columnas virtuales a actulizar")
	}

	query := Query[T]{}
	records := []T{}
	err := selectExec(&records, &query)
	if err != nil {
		fmt.Println("Error al obtener los registros::", err)
	}

	fmt.Println("registros obtenidos::", len(records))
	fmt.Print(records)

	queryStatements := makeUpdateStatementsBase(&records, nil, nil, true)
	fmt.Println("actualizando registros::", len(queryStatements))
	queryInsert := makeQueryStatement(queryStatements)
	if err := QueryExec(queryInsert); err != nil {
		fmt.Println(queryInsert)
		fmt.Println("Error updating records:", err)
	}
}
