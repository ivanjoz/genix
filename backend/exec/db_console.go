package exec

import (
	"app/core"
	"app/db"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DbConsole is the CLI door to the ORM's name-addressed data surface: list tables, describe one,
// query it with filters, count, insert and update — all without compiling a new Go file per
// question. It exists because the typed API needs the record type at compile time, so a human or
// an agent holding only a table name had no way in.
//
//	cd backend
//	go run . fn-db '{"op":"tables"}'
//	go run . fn-db tmp/db-request.json
//
// Reads land in a JSON file (default tmp/db-result.json) rather than on stdout, because startup
// and the ORM both log there and a result of thousands of rows would be unusable mixed in.

const (
	defaultRecordLimit = 50
	// maxRecordLimit caps one call: past this the answer belongs in a report handler, not here.
	maxRecordLimit    = 5000
	defaultOutputPath = "tmp/db-result.json"
	// previewRecordCount is how many rows are echoed to stdout, just to confirm the shape.
	previewRecordCount = 3
)

type dbConsoleRequest struct {
	Op    string `json:"op"`
	Table string `json:"table"`

	Filters     []db.FilterSpec `json:"filters"`
	Columns     []string        `json:"columns"`
	Limit       int32           `json:"limit"`
	AllowFilter bool            `json:"allow_filter"`
	OrderDesc   bool            `json:"order_desc"`

	Records        json.RawMessage `json:"records"`
	UpdateColumns  []string        `json:"update_columns"`
	ExcludeColumns []string        `json:"exclude_columns"`
	// Apply is the write gate. config.toml normally points at a shared database, so an insert or
	// an update is validated and reported but never executed until it is explicitly set.
	Apply bool `json:"apply"`

	Out string `json:"out"`
}

type dbConsoleResult struct {
	Op      string `json:"op"`
	Table   string `json:"table,omitempty"`
	Count   int    `json:"count"`
	Applied bool   `json:"applied,omitempty"`
	// LimitReached warns that the read stopped at the limit, so Count is a floor and not a total.
	LimitReached bool                 `json:"limit_reached,omitempty"`
	Tables       []dbTableSummary     `json:"tables,omitempty"`
	Schema       *db.TableDescription `json:"schema,omitempty"`
	Records      json.RawMessage      `json:"records,omitempty"`
}

type dbTableSummary struct {
	Name      string   `json:"name"`
	ID        int16    `json:"id"`
	Partition string   `json:"partition,omitempty"`
	Keys      []string `json:"keys,omitempty"`
}

// registerControllersOnce keeps the whole table set out of process startup: compiling 40-odd
// schemas is wasted work on every Lambda cold start, and only this console needs them all.
var registerControllersOnce sync.Once

func resolveController(tableName string) (db.Controller, error) {
	if tableName == "" {
		return nil, core.Err(`falta "table" en la petición`)
	}
	registerControllersOnce.Do(func() {
		for _, controller := range MakeScyllaControllers() {
			resolvedController := controller
			db.RegisterControllerFactory(resolvedController.GetTableName(),
				func() db.Controller { return resolvedController })
		}
	})
	return db.ResolveControllerByName(tableName)
}

func DbConsole(args *core.ExecArgs) core.FuncResponse {
	// The ORM reports a misuse by panicking (a view whose key columns were not updated together,
	// a value outside a packed range). Those messages name the fix, so they are worth far more
	// as an error line than as a stack trace on a console driven by hand.
	defer func() {
		if recoveredValue := recover(); recoveredValue != nil {
			core.Log("Error del ORM:: ", recoveredValue)
			os.Exit(1)
		}
	}()

	request, err := parseDbConsoleRequest(args.Message)
	if err != nil {
		return args.MakeErr(err.Error())
	}

	result, err := runDbConsoleOperation(request)
	if err != nil {
		return args.MakeErr(err.Error())
	}

	outputPath := request.Out
	if outputPath == "" {
		outputPath = defaultOutputPath
	}
	if err := writeDbConsoleResult(outputPath, result); err != nil {
		return args.MakeErr(err.Error())
	}

	printDbConsoleSummary(result, outputPath)
	return core.FuncResponse{Message: fmt.Sprintf("%v: %v -> %v", request.Op, result.Count, outputPath)}
}

// parseDbConsoleRequest takes either inline JSON or a path to a JSON file, so a one-line query
// stays a one-liner while a payload with records can come from a file. Numbers are kept as
// json.Number: an int64 id round-tripped through float64 would lose its last digits.
func parseDbConsoleRequest(message string) (*dbConsoleRequest, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, core.Err(`falta la petición. Ejemplo: go run . fn-db '{"op":"tables"}'`)
	}

	payload := []byte(message)
	if message[0] != '{' {
		fileContent, err := os.ReadFile(message)
		if err != nil {
			return nil, core.Err("no se pudo leer el archivo de petición:", err.Error())
		}
		payload = fileContent
	}

	request := dbConsoleRequest{}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil {
		return nil, core.Err("la petición no es JSON válido:", err.Error())
	}
	return &request, nil
}

func runDbConsoleOperation(request *dbConsoleRequest) (*dbConsoleResult, error) {
	result := dbConsoleResult{Op: request.Op, Table: request.Table}

	switch request.Op {
	case "tables":
		result.Tables = listDbTables()
		result.Count = len(result.Tables)
		return &result, nil

	case "describe":
		controller, err := resolveController(request.Table)
		if err != nil {
			return nil, err
		}
		schema := controller.DescribeTable()
		result.Schema = &schema
		result.Count = len(schema.Columns)
		return &result, nil

	case "query", "count":
		controller, err := resolveController(request.Table)
		if err != nil {
			return nil, err
		}
		spec := db.QuerySpec{
			Filters:     request.Filters,
			Columns:     request.Columns,
			Limit:       resolveRecordLimit(request.Limit),
			AllowFilter: request.AllowFilter,
			OrderDesc:   request.OrderDesc,
		}
		// A count only needs the rows to exist, so it reads the keys and drops the payload.
		if request.Op == "count" {
			spec.Columns = controller.DescribeTable().Keys
		}
		records, count, err := controller.QueryRecordsJSON(spec)
		if err != nil {
			return nil, err
		}
		result.Count = count
		result.LimitReached = int32(count) >= spec.Limit
		if request.Op == "query" {
			result.Records = records
		}
		return &result, nil

	case "insert", "update":
		return runDbConsoleWrite(request)
	}

	return nil, core.Err(`op desconocida:`, request.Op,
		`- válidas: tables, describe, query, count, insert, update`)
}

func runDbConsoleWrite(request *dbConsoleRequest) (*dbConsoleResult, error) {
	controller, err := resolveController(request.Table)
	if err != nil {
		return nil, err
	}
	if len(request.Records) == 0 {
		return nil, core.Err(`falta "records" para el`, request.Op)
	}

	// Column names are validated before the gate, so a dry run still catches a typo instead of
	// letting it surface only once someone sets apply.
	schema := controller.DescribeTable()
	columnsToCheck := append(append([]string{}, request.UpdateColumns...), request.ExcludeColumns...)
	if err := validateColumnNames(&schema, columnsToCheck); err != nil {
		return nil, err
	}

	pendingRecords := []json.RawMessage{}
	if err := json.Unmarshal(request.Records, &pendingRecords); err != nil {
		return nil, core.Err(`"records" debe ser un array de objetos:`, err.Error())
	}

	result := dbConsoleResult{Op: request.Op, Table: request.Table, Count: len(pendingRecords)}
	if !request.Apply {
		core.Log(fmt.Sprintf(
			"DRY RUN: %v de %v registros en %v. Añada \"apply\": true para ejecutarlo.",
			request.Op, len(pendingRecords), request.Table))
		return &result, nil
	}

	writtenCount := 0
	if request.Op == "insert" {
		writtenCount, err = controller.InsertRecordsJSON(request.Records, request.ExcludeColumns)
	} else {
		writtenCount, err = controller.UpdateRecordsJSON(request.Records, request.UpdateColumns)
	}
	if err != nil {
		return nil, err
	}

	result.Count = writtenCount
	result.Applied = true
	return &result, nil
}

func listDbTables() []dbTableSummary {
	tables := []dbTableSummary{}
	for _, controller := range MakeScyllaControllers() {
		schema := controller.DescribeTable()
		tables = append(tables, dbTableSummary{
			Name: schema.Name, ID: schema.ID, Partition: schema.Partition, Keys: schema.Keys,
		})
	}
	return tables
}

func validateColumnNames(schema *db.TableDescription, columnNames []string) error {
	for _, columnName := range columnNames {
		found := false
		for _, column := range schema.Columns {
			if column.Name == columnName || column.FieldName == columnName {
				found = true
				break
			}
		}
		if !found {
			return core.Err("la tabla", schema.Name, "no tiene la columna", columnName)
		}
	}
	return nil
}

func resolveRecordLimit(requestedLimit int32) int32 {
	if requestedLimit <= 0 {
		return defaultRecordLimit
	}
	if requestedLimit > maxRecordLimit {
		return maxRecordLimit
	}
	return requestedLimit
}

func writeDbConsoleResult(outputPath string, result *dbConsoleResult) error {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return core.Err("no se pudo serializar el resultado:", err.Error())
	}
	if directory := filepath.Dir(outputPath); directory != "." {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return core.Err("no se pudo crear el directorio de salida:", err.Error())
		}
	}
	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		return core.Err("no se pudo escribir el resultado:", err.Error())
	}
	return nil
}

// printDbConsoleSummary echoes just enough to confirm the shape; the full answer is in the file.
func printDbConsoleSummary(result *dbConsoleResult, outputPath string) {
	limitNote := ""
	if result.LimitReached {
		limitNote = " (tope alcanzado: hay más, suba \"limit\")"
	}
	fmt.Printf("\n== db %v %v | %v registros%v -> %v\n",
		result.Op, result.Table, result.Count, limitNote, outputPath)

	if len(result.Records) == 0 {
		return
	}
	records := []json.RawMessage{}
	if err := json.Unmarshal(result.Records, &records); err != nil {
		return
	}
	for recordIndex, record := range records {
		if recordIndex >= previewRecordCount {
			fmt.Printf("   … %v más en %v\n", len(records)-previewRecordCount, outputPath)
			break
		}
		fmt.Printf("   %v\n", core.StrCut(string(record), 300))
	}
}
