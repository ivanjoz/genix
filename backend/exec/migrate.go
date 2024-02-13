package exec

import (
	"app/core"
	s "app/types"
	"archive/tar"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

func MakeScyllaControllers() []ScyllaController {
	return []ScyllaController{
		makeController[s.Almacen](),
		makeController[s.Sede](),
		makeController[s.Producto](),
		makeController[s.ProductoStock](),
		makeController[s.ListaCompartidaRegistro](),
		makeController[s.Increment](),
	}
}

type ScyllaController struct {
	ScyllaTable   core.ScyllaTable
	GetRecords    func(partitionID, limit int32, lastKey any) ([]any, error)
	GetRecordsGob func(partitionID, limit int32, lastKey any) ([]byte, error)
	InitTable     func(mode int8)
}

func makeController[T any]() ScyllaController {
	var newType T
	scyllaTable := core.MakeScyllaTable(newType)

	GetRecords := func(partitionID, limit int32, lastKey any) ([]T, error) {
		records := []T{}

		query := core.DBSelect(&records)
		if len(scyllaTable.PartitionKey) > 0 {
			query = query.Where(scyllaTable.PartitionKey).Equals(partitionID)
		}

		if lastKey != nil {
			query = query.Where(scyllaTable.PrimaryKey).GreatThan(lastKey)
		}

		if limit > 0 {
			query.Limit = limit
		}

		err := query.Exec()
		if err != nil {
			return records, core.Err("Error al consultar", scyllaTable.Name, ":", err)
		}

		return records, nil
	}

	return ScyllaController{
		ScyllaTable: scyllaTable,
		GetRecords: func(partitionID, limit int32, lastKey any) ([]any, error) {
			records, err := GetRecords(partitionID, limit, lastKey)

			if err != nil {
				return []any{}, err
			}

			recordsInterface := []any{}
			for _, e := range records {
				recordsInterface = append(recordsInterface, e)
			}

			return recordsInterface, nil
		},
		GetRecordsGob: func(partitionID, limit int32, lastKey any) ([]byte, error) {
			records, err := GetRecords(partitionID, limit, lastKey)

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
		InitTable: func(mode int8) {
			core.InitTable[T](mode)
		},
	}
}

func CreateBackupFile(args *core.ExecArgs) core.FuncResponse {

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, controller := range MakeScyllaControllers() {
		name := fmt.Sprintf("%v|%v", controller.ScyllaTable.Name, 1)

		core.Log("Obteniendo registros de: ", name, "...")

		records, err := controller.GetRecordsGob(1, 100000, nil)
		if err != nil {
			panic(err)
		}

		core.Log("Registros obtenidos: ", len(records))

		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(records)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			panic("Error al escribir TAR header: " + name + "| " + err.Error())
		}
		if _, err := tw.Write([]byte(records)); err != nil {
			panic("Error al escribir TAR body: " + name + "| " + err.Error())
		}
	}

	if err := tw.Close(); err != nil {
		panic("Error al cerrar TAR: " + err.Error())
	}

	tarPath := core.Env.TMP_DIR + "backup.tar"

	err := os.WriteFile(tarPath, buf.Bytes(), 0644)
	if err != nil {
		log.Fatal("error writing to file:", err)
	}

	core.Log("Backup TAR generado en: ", tarPath)

	return core.FuncResponse{}
}
