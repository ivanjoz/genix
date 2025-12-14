package exec

import (
	"app/aws"
	"app/core"
	"app/handlers"
	"archive/tar"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

func SaveBackup(empresaID int32) error {

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	encoder, _ := zstd.NewWriter(nil)

	addTarRecord := func(name string, records any) error {
		core.Log("Agrigando registro desde DynamoDB:", name)
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
	empresa, err := empresasTable.GetItem(fmt.Sprintf("%v", empresaID))

	if err != nil {
		return core.Err("Error al obtener la empresa:", err)
	}

	if err := addTarRecord("empresa|1", *empresa); err != nil {
		return core.Err(err)
	}

	// Accesos
	if empresaID == 1 {
		accesosTable := handlers.MakeAccesoTable()
		accesos, err := accesosTable.QueryBatch([]aws.DynamoQueryParam{
			{Index: "sk", GreaterThan: "0"},
		})

		if err != nil {
			return core.Err("Error al obtener los accesos:", err)
		}

		if err := addTarRecord("accesos|1", accesos); err != nil {
			return core.Err(err)
		}
	}

	// Usuarios
	usuariosTable := handlers.MakeUsuarioTable(empresaID)
	usuarios, err := usuariosTable.QueryBatch([]aws.DynamoQueryParam{
		{Index: "sk", GreaterThan: "0"},
	})

	if err != nil {
		return core.Err("Error al obtener los usuarios:", err)
	}

	if err := addTarRecord("usuarios|1", usuarios); err != nil {
		return core.Err(err)
	}

	// Perfiles
	perfilesTable := handlers.MakePerfilTable(empresaID)
	perfiles, err := perfilesTable.QueryBatch([]aws.DynamoQueryParam{
		{Index: "sk", GreaterThan: "0"},
	})

	if err != nil {
		return core.Err("Error al obtener los perfiles:", err)
	}

	if err := addTarRecord("perfiles|1", perfiles); err != nil {
		return core.Err(err)
	}

	// Scylla Tables - Controllers
	for _, controller := range MakeScyllaControllers2() {
		name := fmt.Sprintf("%v|%v", controller.TableName, 1)

		core.Log("Obteniendo registros de: ", name, "...")

		records, err := controller.GetRecordsGob(1, 100000, nil)
		if err != nil {
			return core.Err(err)
		}

		core.Log("Registros obtenidos: ", len(records))
		compressed := encoder.EncodeAll(records, make([]byte, 0, len(records)))
		core.Log("Comprimido::", len(compressed), " | ", len(records))

		hdr := &tar.Header{
			Name: name, Mode: 0600, Size: int64(len(compressed)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return core.Err("Error al escribir TAR header:", name, "|", err)
		}
		if _, err := tw.Write(compressed); err != nil {
			return core.Err("Error al escribir TAR body:", name, "|", err)
		}
	}

	hash := core.MakeRandomBase36String(12)
	unixTime := time.Now().Unix()
	unixTimeNegative := 9999999999 - unixTime
	fileName := fmt.Sprintf("%v-%v.%v", unixTimeNegative, hash, "tar")

	core.Log("Enviando archivo a S3:", fileName)

	err = aws.SendFileToS3(aws.FileToS3Args{
		Bucket:      core.Env.S3_BUCKET,
		Path:        fmt.Sprintf("backups/%v", empresaID),
		FileContent: buf.Bytes(),
		ContentType: "application/x-tar",
		Name:        fileName,
	})

	if err != nil {
		return core.Err("Error al guardar el backup.tar en S3:", err)
	}

	return nil
}

func DoSaveBackup(args *core.ExecArgs) core.FuncResponse {

	err := SaveBackup(1)
	if err != nil {
		panic(err)
	}

	return core.FuncResponse{}
}

type BackupFile struct {
	Name    string
	Size    int32
	Created int64 `json:"upd"`
}

func GetBackups(args *core.HandlerArgs) core.HandlerResponse {

	prefix := fmt.Sprintf("backups/%v/", args.Usuario.EmpresaID)

	s3Files, err := aws.S3ListFiles(aws.FileToS3Args{
		Bucket:  core.Env.S3_BUCKET,
		Prefix:  prefix,
		MaxKeys: 30,
	})

	if err != nil {
		return args.MakeErr("Error al listar los backups:", err)
	}

	files := []BackupFile{}
	for _, e := range s3Files {
		files = append(files, BackupFile{
			Name:    strings.ReplaceAll(*e.Key, prefix, ""),
			Size:    int32(*e.Size),
			Created: e.LastModified.Unix(),
		})
	}

	return args.MakeResponse(files)
}
