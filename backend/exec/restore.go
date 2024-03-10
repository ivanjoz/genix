package exec

import (
	"app/aws"
	"app/core"
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/klauspost/compress/zstd"
)

func RestoreBackup(req *core.HandlerArgs) core.HandlerResponse {
	type Body struct {
		Name string
		Mode int8
	}

	body := Body{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Name) == 0 {
		return req.MakeErr("No se envió el nombre del archivo a buscar.")
	}

	fileBytes, err := aws.GetFileFromS3(aws.FileToS3Args{
		Bucket: core.Env.S3_BUCKET,
		Path:   fmt.Sprintf("backups/%v", req.Usuario.EmpresaID),
		Name:   body.Name,
	})

	if err != nil {
		return req.MakeErr("Error al obtener el backup desde el S3", err)
	}

	controllersMap := map[string]ScyllaController{}

	for _, e := range MakeScyllaControllers() {
		controllersMap[e.ScyllaTable.NameSingle] = e
	}

	reader := tar.NewReader(bytes.NewReader(fileBytes))

	for {
		header, err := reader.Next()

		// If no more files are found, we break the loop
		if err != nil {
			if err == tar.ErrHeader || err == io.EOF {
				break // end of tar archive
			}
			log.Fatalf("Error reading tar file: %s", err)
		}

		nameSlice := strings.Split(header.Name, "|")
		tableName := nameSlice[0]

		if _, ok := controllersMap[tableName]; !ok {
			core.Log("No se encontró la tabla:", tableName)
			continue
		}

		controller := controllersMap[tableName]

		// Create a buffer to store the content of the file
		buf := new(bytes.Buffer)

		// Copy the content of the file to the buffer
		_, err = io.Copy(buf, reader)
		if err != nil {
			log.Fatalf("Error reading file content: %s", err)
		}

		contentCompressed := buf.Bytes()
		decoder, _ := zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
		content, err := decoder.DecodeAll(contentCompressed, nil)

		if err != nil {
			log.Fatalf("Error descomprimiendo registro TAR: %v | %v", header.Name, err)
		}

		fmt.Printf("Restaurando registros: %v\n", header.Name)
		controller.RestoreGobRecords(req.Usuario.EmpresaID, content)
	}

	return req.MakeResponse(map[string]int{"ok": 1})
}
