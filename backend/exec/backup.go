package exec

import (
	"app/aws"
	"app/core"
	"app/handlers"
	"archive/tar"
	"bytes"
	"fmt"
	"time"

	"github.com/klauspost/compress/zstd"
)

func MakeBackup(args *core.ExecArgs) core.FuncResponse {

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	encoder, _ := zstd.NewWriter(nil)

	addTarRecord := func(name string, records any) error {
		content, _ := core.GobEncode(records)
		compressed := encoder.EncodeAll(content, make([]byte, 0, len(content)))

		hdr := &tar.Header{
			Name: name, Mode: 0600, Size: int64(len(compressed)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return core.Err("Error al escribir TAR header:", name, "|", err)
		}
		if _, err := tw.Write(compressed); err != nil {
			return core.Err("Error al escribir TAR body:", name, "|", err)
		}
		return nil
	}

	// Empresa
	empresasTable := handlers.MakeEmpresaTable()
	empresa, err := empresasTable.GetItem(fmt.Sprintf("%v", args.Param2))

	if err != nil {
		return args.MakeErr("Error al obtener la empresa:", err)
	}

	if err := addTarRecord("empresa|1", *empresa); err != nil {
		return args.MakeErr(err)
	}

	// Usuarios
	usuariosTable := handlers.MakeUsuarioTable(args.Param2)
	usuarios, err := usuariosTable.QueryBatch([]aws.DynamoQueryParam{
		{Index: "sk", GreaterThan: "0"},
	})

	if err != nil {
		return args.MakeErr("Error al obtener los usuarios:", err)
	}

	if err := addTarRecord("usuarios|1", usuarios); err != nil {
		return args.MakeErr(err)
	}

	// Accesos
	accesosTable := handlers.MakeAccesoTable()
	accesos, err := accesosTable.QueryBatch([]aws.DynamoQueryParam{
		{Index: "sk", GreaterThan: "0"},
	})

	if err != nil {
		return args.MakeErr("Error al obtener los accesos:", err)
	}

	if err := addTarRecord("accesos|1", accesos); err != nil {
		return args.MakeErr(err)
	}

	// Scylla Tables / Controllers
	for _, controller := range MakeScyllaControllers() {
		name := fmt.Sprintf("%v|%v", controller.ScyllaTable.Name, 1)

		core.Log("Obteniendo registros de: ", name, "...")

		records, err := controller.GetRecordsGob(1, 100000, nil)
		if err != nil {
			return args.MakeErr(err)
		}

		core.Log("Registros obtenidos: ", len(records))
		compressed := encoder.EncodeAll(records, make([]byte, 0, len(records)))

		hdr := &tar.Header{
			Name: name, Mode: 0600, Size: int64(len(compressed)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return args.MakeErr("Error al escribir TAR header:", name, "|", err)
		}
		if _, err := tw.Write(compressed); err != nil {
			return args.MakeErr("Error al escribir TAR body:", name, "|", err)
		}
	}

	hash := core.MakeRandomBase36String(12)
	unixTime := time.Now().Unix()
	content := buf.Bytes()
	fileName := fmt.Sprintf("%v-%v-%v.%v", unixTime, len(content), hash, "tar")

	core.Log("Enviando archivo a S3:", fileName)

	aws.SendFileToS3(aws.FileToS3Args{
		Bucket:      core.Env.S3_BUCKET,
		Path:        "backups",
		FileContent: content,
		ContentType: "application/x-tar",
		Name:        fileName,
	})

	return core.FuncResponse{}
}
