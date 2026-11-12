package runner

import (
	"app/core"
	_ "embed"
	"encoding/csv"
	"math"
	"strconv"
	"strings"
)

// ParsePSV reads a pipe-separated file with a header row and returns the data rows.
func ParsePSV(content string, expectedColumns int) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = '|'
	reader.FieldsPerRecord = expectedColumns

	records, readError := reader.ReadAll()
	if readError != nil {
		return nil, readError
	}
	if len(records) < 2 {
		return nil, core.Err("el archivo .psv no tiene filas de datos")
	}
	return records[1:], nil
}

// ParsePriceToCents reads a decimal amount written in currency units and returns cents, which is
// how every monetary column is persisted.
func ParsePriceToCents(rawValue string) (int32, error) {
	parsedValue, parseError := strconv.ParseFloat(strings.TrimSpace(rawValue), 64)
	if parseError != nil {
		return 0, parseError
	}
	return int32(math.Round(parsedValue * 100)), nil
}
