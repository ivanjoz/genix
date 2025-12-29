package db

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

func exportToCSV[T any](scyllaTable *ScyllaTable[T], partValue int32) (CSVResult, error) {

	columnsOrder := map[string]int{}

	for _, col := range scyllaTable.keys {
		columnsOrder[col.GetName()] = len(columnsOrder) + 1
	}

	for _, col := range scyllaTable.columns {
		pk := scyllaTable.GetPartKey()
		if pk != nil && !pk.IsNil() && pk.GetInfo().Idx == col.GetInfo().Idx {
			continue
		}
		if col.GetInfo().IsVirtual {
			continue
		}
		if _, ok := columnsOrder[col.GetName()]; !ok {
			columnsOrder[col.GetName()] = len(columnsOrder) + 1
		}
	}

	columnsNames := make([]string, len(columnsOrder))
	columnsNamesWType := make([]string, len(columnsOrder))
	csv := CSVResult{}

	for name, i := range columnsOrder {
		col := scyllaTable.columnsMap[name]
		columnsNames[i-1] = name
		columnsNamesWType[i-1] = fmt.Sprintf("%v:%v", name, col.GetType().Type)
	}

	for i, name := range columnsNamesWType {
		// Write Headers
		csv.Content = append(csv.Content, []byte(name)...)
		if i < len(columnsOrder)-1 {
			csv.Content = append(csv.Content, '|')
		}
	}

	csv.Content = append(csv.Content, '\n')

	fmt.Println("Table:", scyllaTable.name, "| Cols:", strings.Join(columnsNames, ", "))

	whereClause := ""
	pk := scyllaTable.GetPartKey()
	if pk != nil && !pk.IsNil() {
		whereClause = fmt.Sprintf(" WHERE %v = %v", pk.GetName(), partValue)
	}

	queryStr := fmt.Sprintf("SELECT %v FROM %v%v",
		strings.Join(columnsNames, ", "), scyllaTable.GetFullName(), whereClause)

	fmt.Println(queryStr)

	iter := getScyllaConnection().Query(queryStr).Iter()
	rd, err := iter.RowData()
	if err != nil {
		fmt.Println("Error on RowData::", err)
		return csv, err
	}

	// Stream rows directly to csvByte buffer
	scanner := iter.Scanner()
	for scanner.Next() {
		// Scan directly into rd.Values (reused memory)
		if err := scanner.Scan(rd.Values...); err != nil {
			fmt.Println("Error on scan::", err)
			return csv, err
		}

		// Append each value to the buffer immediately
		for i, value := range rd.Values {
			csv.Content = append(csv.Content, valueToCSVBase64(value)...)
			if i < (len(rd.Values) - 1) {
				csv.Content = append(csv.Content, '|')
			}
		}
		csv.Content = append(csv.Content, '\n')
		csv.RowsCount++
	}

	if err := iter.Close(); err != nil {
		return csv, err
	}

	return csv, nil
}

func isPointer(v any) bool {
	return reflect.ValueOf(v).Kind() == reflect.Ptr
}

func CsvToRecords[T any](
	scyllaTable *ScyllaTable[T], content *[]byte, partValue any,
) ([]T, error) {

	if content == nil {
		return nil, nil
	}

	// Extract the header
	isHeader := true
	columnNames := []string{}
	currentValue := []byte{}
	columns := []IColInfo{}

	parseColumns := func() {
		for _, colname := range columnNames {
			name := strings.Split(colname, ":")[0]
			columns = append(columns, scyllaTable.columnsMap[name])
		}
	}

	var currentRecord T
	parsedRecords := []T{}
	colIdx := 0

	for i := 0; i < len(*content); i++ {
		b := (*content)[i]

		if isHeader {
			if b == '\n' {
				columnNames = append(columnNames, string(currentValue))
				parseColumns()
				currentValue = []byte{}
				isHeader = false
				continue
			}
			if b == pipeByte {
				columnNames = append(columnNames, string(currentValue))
				currentValue = []byte{}
				continue
			}
			currentValue = append(currentValue, b)
			continue
		}

		if b == pipeByte || b == '\n' {
			// Set the value to the current record
			if colIdx < len(columns) {
				colInfo := columns[colIdx]
				if colInfo != nil {
					value := base64CSVStringToValue(string(currentValue), colInfo.GetType().Type)
					if colInfo.GetType().Type == 9 {
						fmt.Println("Blob:", colInfo.GetName(), "|", value, "|", string(currentValue), "|", colInfo.GetType().Type)
					}
					colInfo.SetValue(unsafe.Pointer(&currentRecord), value)
				}
			}

			currentValue = []byte{}
			colIdx++

			if b == '\n' {
				if scyllaTable.partKey != nil {
					scyllaTable.partKey.SetValue(unsafe.Pointer(&currentRecord), partValue)
				}
				parsedRecords = append(parsedRecords, currentRecord)
				currentRecord = *new(T)
				colIdx = 0
			}
		} else {
			currentValue = append(currentValue, b)
		}
	}

	return parsedRecords, nil
}
