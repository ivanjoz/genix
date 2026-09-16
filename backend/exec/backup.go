package exec

import (
	"app/cloud"
	"app/core"
	"archive/tar"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

// backupBatchSize is how many records go into one TAR entry. A table is exported in
// batches so neither the backup nor the restore ever holds more than this many records
// in memory, whatever the table's size.
const backupBatchSize = 50_000

func SaveBackup(companyID int32) error {

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	encoder, _ := zstd.NewWriter(nil)
	defer encoder.Close()
	var err error

	// Scylla Tables - Controllers
	for _, controller := range MakeScyllaControllers() {
		tableName := controller.GetTableName()
		core.Log("Obteniendo registros de:", tableName, "...")

		batchIndex := 0
		rowsCount, err := controller.ExportRecordsColbin(companyID, backupBatchSize,
			func(encoded []byte, batchRowsCount int32) error {
				// The restore reads the table name off the first segment, so it stays first.
				name := fmt.Sprintf("%v.%v.%v.colbin.zstd", tableName, companyID, batchIndex)
				batchIndex++

				compressed := encoder.EncodeAll(encoded, make([]byte, 0, len(encoded)))
				core.Log(fmt.Sprintf("%v | %v registros | colbin: %.3f kb | zstd: %.3f kb",
					name, batchRowsCount, float64(len(encoded))/1000, float64(len(compressed))/1000))

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
				return nil
			})

		if err != nil {
			return core.Err("Error al exportar la tabla", tableName, ":", err)
		}

		core.Log("Registros obtenidos:", tableName, "|", rowsCount, "| batches:", batchIndex)
	}

	if err := tw.Close(); err != nil {
		return core.Err("Error al cerrar el TAR writer:", err)
	}

	hash := core.MakeRandomBase36String(12)
	unixTime := time.Now().Unix()
	unixTimeNegative := 9999999999 - unixTime
	fileName := fmt.Sprintf("%v-%v.%v", unixTimeNegative, hash, "tar")

	core.Log("Enviando archivo a S3:", fileName)

	err = cloud.SaveFile(cloud.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        fmt.Sprintf("backups/%v", companyID),
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

	prefix := fmt.Sprintf("backups/%v/", args.User.CompanyID)

	filesInStorage, err := cloud.ListFiles(cloud.SaveFileArgs{
		Bucket:  core.Env.S3_BUCKET,
		Prefix:  prefix,
		MaxKeys: 30,
	})

	if err != nil {
		return args.MakeErr("Error al listar los backups:", err)
	}

	files := []BackupFile{}
	for _, file := range filesInStorage {
		files = append(files, BackupFile{
			Name:    strings.ReplaceAll(file.Key, prefix, ""),
			Size:    int32(file.Size),
			Created: file.LastModified.Unix(),
		})
	}

	return args.MakeResponse(files)
}
