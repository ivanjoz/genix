package exec

import (
	"app/core"
	"app/db2"
	s "app/types"
	"bytes"
	"encoding/gob"
	"fmt"
)

func MakeScyllaControllers2() []db2.ScyllaController2 {
	return []db2.ScyllaController2{
		//v2 controllers
		makeControllerDB2[s.Almacen](),
		makeControllerDB2[s.Sede](),
		makeControllerDB2[s.AlmacenProducto](),
		makeControllerDB2[s.PaisCiudad](),
		makeControllerDB2[s.GaleriaImagen](),
		makeControllerDB2[s.ListaCompartidaRegistro](),
		makeControllerDB2[s.AlmacenMovimiento](),
		makeControllerDB2[s.Usuario](),
		makeControllerDB2[s.Caja](),
		makeControllerDB2[s.CajaMovimiento](),
		makeControllerDB2[s.CajaCuadre](),
		makeControllerDB2[s.Producto](),
		makeControllerDB2[s.Increment](),
		makeControllerDB2[DemoStruct](),
	}
}

// makeControllerDB2 creates a ScyllaController for db2 package types using generics.
// This function automatically handles queries for any db2 table type using the
// TableQueryInterface for clean, simple query building.
func makeControllerDB2[T db2.TableBaseInterface[E, T], E db2.TableSchemaInterface[E]]() db2.ScyllaController2 {
	// Get the table struct instance
	schema := db2.MakeSchema[T]()
	scyllaTable := db2.MakeScyllaTable[T]()

	// Get table name and keyspace
	tableName := schema.Name
	keyspace := schema.Keyspace
	if keyspace == "" {
		keyspace = core.Env.DB_NAME
	}
	fullTableName := fmt.Sprintf("%s.%s", keyspace, tableName)

	// Get partition key name
	partKeyName := ""
	if schema.Partition != nil {
		partKeyName = schema.Partition.GetName()
	}

	// Get clustering key info
	firstKeyName := ""
	if len(schema.Keys) > 0 {
		firstKeyName = schema.Keys[0].GetName()
	}

	queryRecords := func(empresaID, limit int32, lastKey any) ([]T, error) {
		records := []T{}
		query := any(db2.Query(&records)).(db2.TableQueryInterface[E])

		// Add empresa_id filter if partition key is empresa_id
		if partKeyName == "empresa_id" {
			query.SetWhere("empresa_id", "=", empresaID)
		}

		// Add lastKey filter if provided (for pagination)
		if lastKey != nil && firstKeyName != "" {
			query.SetWhere(firstKeyName, ">=", lastKey)
		}

		// Add limit if provided
		if limit > 0 {
			query.Limit(limit)
		}

		// Execute the query
		fmt.Println("Obteniendo registros de::", tableName)
		if err := query.Exec(); err != nil {
			return nil, core.Err("Error al consultar", tableName, ":", err)
		}
		fmt.Println("reqgistros obtenidos (1)::", len(records))
		return records, nil
	}

	return db2.ScyllaController2{
		TableName: tableName,
		Schema:    schema,
		Table:     scyllaTable,
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
			fmt.Println("registros obtenidos (2)::", len(records))

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
				return core.Err("Error al decodificar registros de:", tableName, ".", err)
			}

			core.Log(core.ToJsonNoErr(records))

			core.Log("Tabla:", tableName, "| Guardando", len(records), "registros...")

			// Delete existing records for this empresa_id if partition key is empresa_id
			if partKeyName == "empresa_id" {
				statement := fmt.Sprintf(`DELETE FROM %v WHERE empresa_id = %v`, fullTableName, empresaID)
				if err := db2.QueryExec(statement); err != nil {
					core.Log("Error en statement: ", statement)
					return core.Err("Error al eliminar registros:", err)
				}
			}

			// Insert new records
			if err = db2.Insert(&records); err != nil {
				return core.Err("Error al insertar registros:", err)
			}

			return nil
		},
		InitTable: func(mode int8) {
			// TODO: Implement table initialization for db2
			core.Log("InitTable not yet implemented for db2:", tableName)
		},
		RecalcVirtualColumns: func() {
			// TODO: Implement virtual column recalculation for db2
			core.Log("RecalcVirtualColumns not yet implemented for db2:", tableName)
		},
	}
}
