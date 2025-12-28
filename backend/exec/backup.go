package exec

import (
	"app/aws"
	"app/core"
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
	defer encoder.Close()
	var err error

	// Scylla Tables - Controllers
	for _, controller := range MakeScyllaControllers() {
		scyllaTable := controller.GetTable()
		name := fmt.Sprintf("%v.%v.csv.zstd", scyllaTable.GetName(), empresaID)

		core.Log("Obteniendo registros de: ", name, "...")

		csv, err := controller.GetRecordsCSV(empresaID)
		if err != nil {
			return core.Err(err)
		}

		core.Log("Registros obtenidos: ", csv.RowsCount)

		compressed := encoder.EncodeAll(csv.Content, make([]byte, 0, len(csv.Content)))
		core.Log(fmt.Sprintf("%v registros comprimidos: %.3f kb", csv.RowsCount, float64(len(compressed))/1000))

		hdr := &tar.Header{
			Name: name, Mode: 0600, Size: int64(len(compressed)),
			ModTime: time.Now(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return core.Err("Error al escribir TAR header:", name, "|", err)
		}
		if _, err := tw.Write(compressed); err != nil {
			return core.Err("Error al escribir TAR body:", name, "|", err)
		}
	}

	if err := tw.Close(); err != nil {
		return core.Err("Error al cerrar el TAR writer:", err)
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
