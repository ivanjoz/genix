package core

import (
	"app/types"
	"fmt"
	"strings"
)

type ScyllaColumns struct {
	types.TAGS `table:"system_schema.columns"`
	Table      string `db:"table_name"`
	Name       string `db:"column_name"`
	Type       string `db:"type"`
}

type ScyllaViews struct {
	types.TAGS `table:"system_schema.views"`
	ViewName   string `db:"view_name"`
	Table      string `db:"base_table_name"`
}

var ScyllaColumnsMap map[string][]ScyllaColumns
var ScyllaViewsMap map[string][]ScyllaViews

func InitTable[T any](mode int8) {
	if ScyllaColumnsMap == nil {
		Log("Consultando columns... ")
		scyllaColumns := []ScyllaColumns{}

		err := DBSelect(&scyllaColumns).
			Where("keyspace_name").Equals(Env.DB_NAME).Exec()

		if err != nil {
			panic("Error al obtener las columnsa: " + err.Error())
		}

		ScyllaColumnsMap = SliceToMapP(scyllaColumns,
			func(e ScyllaColumns) string { return e.Table })

		Log("Consultando views/indexes... ")
		scyllaViews := []ScyllaViews{}

		err = DBSelect(&scyllaViews).
			Where("keyspace_name").Equals(Env.DB_NAME).Exec()

		if err != nil {
			panic("Error al obtener las views: " + err.Error())
		}

		ScyllaViewsMap = SliceToMapP(scyllaViews,
			func(e ScyllaViews) string { return e.Table })
	}

	var newType T
	scyllaTable := MakeScyllaTable(newType)
	Log(fmt.Sprintf(`► Analizando Tabla: "%v"...`, scyllaTable.Name))

	columnNamesMap := map[string]BDColumn{}

	for _, e := range scyllaTable.ColumnsMap {
		columnNamesMap[e.Name] = e
	}

	dbcolumns := ScyllaColumnsMap[scyllaTable.NameSingle]
	// Si no existe la tabla entonces la crea...
	if len(dbcolumns) == 0 {
		Logx(6, "No se encontró la tabla: "+scyllaTable.Name+"\n")
		if mode > 2 {
			return
		}
		Logx(2, fmt.Sprintf(`Creando tabla "%v"...`+"\n", scyllaTable.Name))

		columnsTypes := []string{}
		for _, e := range scyllaTable.Columns {
			columnsTypes = append(columnsTypes, e.Name+" "+e.Type)
		}

		pk := scyllaTable.PrimaryKey
		if len(scyllaTable.PartitionKey) > 0 {
			pk = fmt.Sprintf("(%v), %v", scyllaTable.PartitionKey, pk)
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
		query = fmt.Sprintf(query, scyllaTable.Name, strings.Join(columnsTypes, ", "), pk)
		Log(query)

		err := DBExec(query)
		if err != nil {
			panic(fmt.Sprintf(`Error creando tabla "%v" | `, scyllaTable.Name) + err.Error())
		}

		Logx(2, fmt.Sprintf(`Tabla creada: "%v"`+"\n", scyllaTable.Name))
	}

	columnsScyllaMap := map[string]string{}

	for _, e := range dbcolumns {
		if column, ok := columnNamesMap[e.Name]; ok {
			if column.Type != e.Type {
				Logx(5, fmt.Sprintf(`La columna "%v" está definida con type "%v", pero en el Struct está con "%v" equivalente a "%v"`+"\n", e.Name, e.Type, column.FieldType, column.Type))
			}
			delete(columnNamesMap, e.Name)
		} else {
			columnsScyllaMap[e.Name] = e.Type
		}
	}

	if len(dbcolumns) > 0 {
		// Revisa las columnas existentes en la BD per no mapeadas
		for name, dbType := range columnsScyllaMap {
			Logx(5, fmt.Sprintf(`La columna "%v" con type "%v" existe en la BD origen no está mapeada en el Struct.`+"\n", name, dbType))
		}
		// Revisa las columnas que deben crearse en BD
		for _, column := range columnNamesMap {
			Logx(5, fmt.Sprintf(`La columna "%v" con struct type "%v" no existe en la BD de origen.`+"\n", column.Name, column.FieldType))
			if mode == 2 {
				query := fmt.Sprintf(`ALTER TABLE %v ADD %v %v`, scyllaTable.Name, column.Name, column.Type)
				Log(fmt.Sprintf(`EJecutando agregar columna: "%v"...`, query))

				if err := DBExec(query); err != nil {
					panic(fmt.Sprintf(`Error agregando columna "%v" | %v`, column.Name, err))
				}
				Logx(2, fmt.Sprintf(`Columna Agregada: "%v"`+"\n", column.Name))
			}
		}
	}

	tableViewsSlice := SliceInclude[string]{}
	for _, e := range ScyllaViewsMap[scyllaTable.NameSingle] {
		tableViewsSlice.Add(e.ViewName)
	}

	for _, view := range scyllaTable.Views {

		if tableViewsSlice.Include(strings.Split(view.Name, ".")[1]) {
			continue
		}

		Logx(5, fmt.Sprintf(`No se encontró la view "%v" en la tabla "%v". Preparando creación...`+"\n", view.Name, scyllaTable.NameSingle))

		whereColumns := []string{view.ColumnName, scyllaTable.PrimaryKey}
		pk := strings.Join(whereColumns, ",")

		if len(scyllaTable.PartitionKey) > 0 {
			whereColumns = append([]string{scyllaTable.PartitionKey}, whereColumns...)
			pk = fmt.Sprintf("(%v), %v", scyllaTable.PartitionKey, pk)
		}

		whereColumnsNotNull := []string{}
		for _, e := range whereColumns {
			if e == view.ColumnName {
				col := scyllaTable.ColumnsMap[e]
				if col.Type == "text" {
					whereColumnsNotNull = append(whereColumnsNotNull, e+" > ''")
				} else {
					whereColumnsNotNull = append(whereColumnsNotNull, e+" > 0")
				}
			} else {
				whereColumnsNotNull = append(whereColumnsNotNull, e+" IS NOT NULL")
			}
		}

		selectColumns := "*"

		if len(scyllaTable.ViewsExcluded) > 0 {
			selectColumnNames := []string{}
			for _, e := range scyllaTable.Columns {
				if e.Name == view.ColumnName || !Contains(scyllaTable.ViewsExcluded, e.Name) {
					selectColumnNames = append(selectColumnNames, e.Name)
				}
			}
			selectColumns = strings.Join(selectColumnNames, ", ")
		}

		query := fmt.Sprintf(`CREATE MATERIALIZED VIEW %v
			AS
			SELECT %v FROM %v
			WHERE %v
			PRIMARY KEY (%v)
			WITH caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
			and compaction = {'class': 'SizeTieredCompactionStrategy'}
			and compression = {'compression_level': '3', 'sstable_compression': 'org.apache.cassandra.io.compress.ZstdCompressor'}
			and dclocal_read_repair_chance = 0
			and speculative_retry = '99.0PERCENTILE';`,
			view.Name, selectColumns, scyllaTable.Name, strings.Join(whereColumnsNotNull, " AND "), pk)

		Log(query)

		if err := DBExec(query); err != nil {
			Log(err)
			panic(fmt.Sprintf(`Error creando view "%v"`, view.Name))
		}

		Logx(2, fmt.Sprintf(`View creada: "%v"`+"\n", view.Name))
	}
}
