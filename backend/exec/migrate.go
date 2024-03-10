package exec

import (
	"app/core"
	s "app/types"
	"bytes"
	"encoding/gob"
)

func MakeScyllaControllers() []ScyllaController {
	return []ScyllaController{
		makeController[s.Almacen](),
		makeController[s.Sede](),
		makeController[s.Producto](),
		makeController[s.ProductoStock](),
		makeController[s.ListaCompartidaRegistro](),
		makeController[s.PaisCiudad](),
		makeController[s.Increment](),
	}
}

type ScyllaController struct {
	ScyllaTable       core.ScyllaTable
	GetRecords        func(empresaID, limit int32, lastKey any) ([]any, error)
	GetRecordsGob     func(empresaID, limit int32, lastKey any) ([]byte, error)
	RestoreGobRecords func(empresaID int32, content []byte) error
	InitTable         func(mode int8)
}

func makeController[T any]() ScyllaController {
	var newType T
	scyllaTable := core.MakeScyllaTable(newType)

	GetRecords := func(empresaID, limit int32, lastKey any) ([]T, error) {
		records := []T{}

		query := core.DBSelect(&records)
		query.Limit = core.If(limit > 0, limit, 0)

		if _, ok := scyllaTable.ColumnsMap["empresa_id"]; ok {
			query = query.Where("empresa_id").Equals(empresaID)
		} else {
			core.Log("No hay key de particionado (empresa_id): ", scyllaTable.Name)
		}

		if lastKey != nil {
			query = query.Where(scyllaTable.PrimaryKey).GreatThan(lastKey)
		}

		err := query.Exec()
		if err != nil {
			return records, core.Err("Error al consultar", scyllaTable.Name, ":", err)
		}

		return records, nil
	}

	return ScyllaController{
		ScyllaTable: scyllaTable,
		GetRecords: func(empresaID, limit int32, lastKey any) ([]any, error) {
			records, err := GetRecords(empresaID, limit, lastKey)

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
			records, err := GetRecords(empresaID, limit, lastKey)

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
				return core.Err("Error al decodificar registros de:", scyllaTable.Name, ".", err)
			}

			// Las secuencias no se insertan sino se actualizan
			core.Log("Tabla:", scyllaTable.Name, "| Guardando", len(records), "registros...")

			if scyllaTable.NameSingle == "sequences" {
				err = core.DBUpdate(&records)
			} else {
				err = core.DBInsert(&records)
			}

			if err != nil {
				return core.Err("Error al insertar registros:", err)
			}
			return nil
		},
		InitTable: func(mode int8) {
			core.InitTable[T](mode)
		},
	}
}
