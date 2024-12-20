package exec

import (
	"app/core"
	"app/db"
	s "app/types"
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
)

func MakeScyllaControllers() []ScyllaController {
	return []ScyllaController{
		makeController[s.Almacen](),
		makeController[s.Sede](),
		makeController[s.Producto](),
		// makeController[s.ProductoStock](),
		makeController[s.ListaCompartidaRegistro](),
		makeController[s.PaisCiudad](),
		makeController[s.AlmacenProducto](),
		makeController[s.AlmacenMovimiento](),
		makeController[s.Increment](),
		makeController[s.Usuario](),
		makeController[s.Caja](),
		makeController[s.CajaMovimiento](),
		makeController[s.CajaCuadre](),
	}
}

type ScyllaController struct {
	TableName            string
	StructType           db.TableSchemaInterface
	GetRecords           func(empresaID, limit int32, lastKey any) ([]any, error)
	GetRecordsGob        func(empresaID, limit int32, lastKey any) ([]byte, error)
	RestoreGobRecords    func(empresaID int32, content []byte) error
	InitTable            func(mode int8)
	RecalcVirtualColumns func()
}

func makeController[T db.TableSchemaInterface]() ScyllaController {
	newType := *new(T)
	scyllaTable := db.MakeTable(newType.GetSchema(), newType)
	columnsMap := scyllaTable.GetColumns()
	tableName := scyllaTable.GetFullName()
	tableNameSingle := strings.Split(tableName, ".")[1]
	partKey := scyllaTable.GetPartKey()

	queryRecords := func(empresaID, limit int32, lastKey any) ([]T, error) {

		query := db.Select(func(q *db.Query[T], col T) {
			if _, ok := columnsMap["empresa_id"]; ok {
				q.Where(db.ColumnStatement{
					Col: "empresa_id", Operator: "=", Value: empresaID,
				})
			} else {
				core.Log("No hay columna de particionado (empresa_id): ", tableName)
			}

			if lastKey != nil {
				q.Where(db.ColumnStatement{
					Col: scyllaTable.GetKeys()[0].Name, Operator: ">=", Value: lastKey,
				})
			}

			if limit > 0 {
				q.Limit(limit)
			}
		})

		if query.Err != nil {
			return query.Records, core.Err("Error al consultar", tableName, ":", query.Err)
		}

		return query.Records, nil
	}

	return ScyllaController{
		StructType: newType,
		GetRecords: func(empresaID, limit int32, lastKey any) ([]any, error) {
			records, err := queryRecords(empresaID, limit, lastKey)

			if err != nil {
				return []any{}, err
			}

			recordsInterface := []any{}
			for _, e := range records {
				recordsInterface = append(recordsInterface, e)
			}

			return recordsInterface, nil
		},
		GetRecordsGob: func(empresaID, limit int32, lastKey any) ([]byte, error) {
			records, err := queryRecords(empresaID, limit, lastKey)

			if err != nil {
				return []byte{}, err
			}

			var buffer bytes.Buffer
			encoder := gob.NewEncoder(&buffer)

			err = encoder.Encode(records)
			if err != nil {
				return []byte{}, err
			}

			return buffer.Bytes(), nil
		},
		RestoreGobRecords: func(empresaID int32, content []byte) error {
			reader := bytes.NewReader(content)
			dec := gob.NewDecoder(reader)

			records := []T{}
			err := dec.Decode(&records)
			if err != nil {
				return core.Err("Error al decodificar registros de:", tableNameSingle, ".", err)
			}

			// Las secuencias no se insertan sino se actualizan
			core.Log("Tabla:", tableNameSingle, "| Guardando", len(records), "registros...")

			// Lógica específica para secuencias
			if tableNameSingle == "sequences" {
				keys := []string{}
				updateStatements := []string{}
				recordsParsed := []s.Increment{}

				for _, e := range records {
					rec := any(e).(s.Increment)
					recordsParsed = append(recordsParsed, rec)
					keys = append(keys, rec.TableName)
				}

				query := db.Select(func(q *db.Query[s.Increment], col s.Increment) {
					q.Where(col.TableName_().In(keys...))
				})

				if query.Err != nil {
					return core.Err("Error al seleccionar registros:", err)
				}

				core.Log("Registros obtenidos:", len(query.Records))
				currentRecordsMap := core.SliceToMapK(query.Records,
					func(e s.Increment) string { return e.TableName })

				for _, e := range recordsParsed {
					currentValue := int64(0)
					if current, ok := currentRecordsMap[e.TableName]; ok {
						currentValue = current.CurrentValue
					}
					core.Log("Current Value:", currentValue)
					if e.CurrentValue == currentValue {
						continue
					}
					increment := ""
					if e.CurrentValue > currentValue {
						increment = fmt.Sprintf("+ %v", e.CurrentValue-currentValue)
					} else {
						increment = fmt.Sprintf("- %v", currentValue-e.CurrentValue)
					}

					statement := fmt.Sprintf(
						"UPDATE %v.sequences SET current_value = current_value %v WHERE name = '%v'",
						core.Env.DB_NAME, increment, e.TableName)
					updateStatements = append(updateStatements, statement)
				}

				core.Log("Registros a actualizar:", len(updateStatements))

				for _, statement := range updateStatements {
					core.Log("Enviando Statement:", statement)
					if err := db.QueryExec(statement); err != nil {
						core.Log("Error en statement: ", statement)
						return core.Err("Error al actualizar registros:", err)
					}
				}
			} else {
				if partKey != nil && partKey.Name == "empresa_id" {
					statement := fmt.Sprintf(`DELETE FROM %v WHERE empresa_id = %v`, tableName, empresaID)
					if err := db.QueryExec(statement); err != nil {
						core.Log("Error en statement: ", statement)
						return core.Err("Error al eliminar registros:", err)
					}
				}
				if err = db.Insert(&records); err != nil {
					return core.Err("Error al insertar registros:", err)
				}
			}
			return nil
		},
		InitTable: func(mode int8) {
			db.DeployScylla(1, *new(T))
		},
		RecalcVirtualColumns: func() {
			db.RecalcVirtualColumns[T]()
		},
	}
}
