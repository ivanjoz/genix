package exec

import (
	"app/cloud"
	"app/core"
	"app/db"
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

	fileBytes, err := cloud.GetFileFromS3(cloud.SaveFileArgs{
		Bucket: core.Env.S3_BUCKET,
		Path:   fmt.Sprintf("backups/%v", req.Usuario.EmpresaID),
		Name:   body.Name,
	})

	if err != nil {
		return req.MakeErr("Error al obtener el backup desde el S3", err)
	}

	// Build the table registry once so restore logs can distinguish a bad TAR from a missing controller.
	controllersMap := map[string]db.ScyllaControllerInterface{}
	registeredTableNames := []string{}
	for _, controller := range MakeScyllaControllers() {
		tableName := controller.GetTableName()
		controllersMap[tableName] = controller
		registeredTableNames = append(registeredTableNames, tableName)
	}

	core.Log(
		"RestoreBackup start | company:", req.Usuario.EmpresaID,
		"| backup:", body.Name,
		"| tar_bytes:", len(fileBytes),
		"| registered_tables:", len(registeredTableNames),
		"| tables:", strings.Join(registeredTableNames, ","),
	)

	reader := tar.NewReader(bytes.NewReader(fileBytes))
	decoder, decoderErr := zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	if decoderErr != nil {
		return req.MakeErr("Error al crear el decoder zstd", decoderErr)
	}
	defer decoder.Close()

	processedEntriesCount := 0
	restoredEntriesCount := 0
	skippedEntriesCount := 0

	for {
		header, err := reader.Next()

		// If no more files are found, we break the loop
		if err != nil {
			if err == tar.ErrHeader || err == io.EOF {
				break // end of tar archive
			}
			log.Fatalf("Error reading tar file: %s", err)
		}

		processedEntriesCount++
		core.Log(
			"RestoreBackup entry | index:", processedEntriesCount,
			"| header_name:", header.Name,
			"| header_size:", header.Size,
			"| header_type:", header.Typeflag,
		)

		nameSlice := strings.Split(header.Name, ".")
		if len(nameSlice) == 0 || len(nameSlice[0]) == 0 {
			skippedEntriesCount++
			core.Log("RestoreBackup skip | invalid header name:", header.Name)
			continue
		}
		tableName := nameSlice[0]

		if tableName == "empresa" || tableName == "accesos" || tableName == "perfiles" {
			skippedEntriesCount++
			core.Log(`RestoreBackup skip | dynamodb table not configured | table:`, tableName)
			continue
		}

		controller, ok := controllersMap[tableName]
		if !ok {
			skippedEntriesCount++
			core.Log(
				"RestoreBackup skip | controller not found | table:", tableName,
				"| header_name:", header.Name,
				"| available_tables:", strings.Join(registeredTableNames, ","),
			)
			continue
		}

		// Copy the raw TAR entry first so we can compare compressed and decompressed sizes in logs.
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, reader)
		if err != nil {
			log.Fatalf("Error reading file content: %s", err)
		}

		contentCompressed := buf.Bytes()
		content, err := decoder.DecodeAll(contentCompressed, nil)

		if err != nil {
			log.Fatalf("Error descomprimiendo registro TAR: %v | %v", header.Name, err)
		}

		csvPreview := string(content)
		if len(csvPreview) > 180 {
			csvPreview = csvPreview[:180]
		}
		csvPreview = strings.ReplaceAll(csvPreview, "\n", "\\n")

		core.Log(
			"RestoreBackup payload | table:", tableName,
			"| compressed_bytes:", len(contentCompressed),
			"| decompressed_bytes:", len(content),
			"| preview:", csvPreview,
		)

		core.Log("Restaurando registros:", header.Name, "| table:", tableName)
		if err = controller.RestoreCSVRecords(req.Usuario.EmpresaID, &content); err != nil {
			core.Log(err)
			continue
		}

		restoredEntriesCount++
		if err = controller.ResetCounter(req.Usuario.EmpresaID); err != nil {
			core.Log("RestoreBackup reset counter error | table:", tableName, "| err:", err)
			continue
		}

		core.Log("RestoreBackup entry restored | table:", tableName, "| company:", req.Usuario.EmpresaID)
	}

	core.Log(
		"RestoreBackup summary | backup:", body.Name,
		"| processed_entries:", processedEntriesCount,
		"| restored_entries:", restoredEntriesCount,
		"| skipped_entries:", skippedEntriesCount,
	)

	return req.MakeResponse(map[string]int{"ok": 1})
}

func CreateBackup(req *core.HandlerArgs) core.HandlerResponse {

	err := SaveBackup(req.Usuario.EmpresaID)
	if err != nil {
		req.MakeErr("Error al crear el backup:", err)
	}

	return req.MakeResponse(map[string]int{"ok": 1})
}

func ResetCounters(partValue any) {
	fmt.Println("Recalculando Counter de Tablas...")
	resetAppliedCount := 0
	resetErrorCount := 0
	for _, sc := range MakeScyllaControllers() {
		if err := sc.ResetCounter(partValue); err != nil {
			resetErrorCount++
			fmt.Printf("ResetCounter error | table=%s | partition=%v | err=%v\n", sc.GetTableName(), partValue, err)
			continue
		}
		resetAppliedCount++
	}
	fmt.Printf("ResetCounter summary | partition=%v | processed=%d | errors=%d\n", partValue, resetAppliedCount, resetErrorCount)
}

func ResetCounterPart(args *core.ExecArgs) core.FuncResponse {
	// Reuse the connected reset flow so exec mode behaves the same as the shared helper.
	ResetCounters(1)
	return core.FuncResponse{}
}
