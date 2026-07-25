package cloud

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"reflect"
	"regexp"
	"strings"
)

// ColumnMeta holds the metadata for a struct field mapped to a database column.
type ColumnMeta struct {
	FieldName   string
	FieldType   reflect.Type
	ColumnName  string
	IsPK        bool   // Partition key (DynamoDB pk) / Primary Key D1
	IsSK        bool   // Sort key (DynamoDB sk)
	IsIndex     bool   // DynamoDB GSI / D1 Index
	DynamoIndex string // E.g., "ix1", "ix2" for DynamoDB indexes
}

// toSnakeCase converts a CamelCase string to snake_case.
func toSnakeCase(str string) string {
	matchFirstCap := regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// getStructHashPrefix computes a 6-character Base64 URL-encoded hash of the struct name.
func getStructHashPrefix(structName string) string {
	h := fnv.New32a()
	h.Write([]byte(structName))
	sum := h.Sum(nil) // 4 bytes

	// Base64 URL Encoding without padding
	b64 := base64.RawURLEncoding.EncodeToString(sum)

	// A 4-byte slice encoded to base64 takes ceil(4/3)*4 = 6 characters (padding stripped).
	if len(b64) > 6 {
		return b64[:6]
	}
	return b64
}

// parseColumns uses reflection to parse struct fields and their "col" tags.
// Returns:
// - []ColumnMeta: The parsed metadata for each mapped column.
// - string: The struct's snake_case table name.
// - string: The 6-character Base64 URL-encoded hash prefix of the struct name.
func parseColumns(model interface{}) ([]ColumnMeta, string, string) {
	var cols []ColumnMeta
	t := reflect.TypeOf(model)

	// If it's a pointer, get the underlying element
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	structName := t.Name()
	tableName := toSnakeCase(structName)
	hashPrefix := getStructHashPrefix(structName)

	ixCount := 1
	skCount := 0

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Embedded helper structs like db.TableStruct are not persisted by the cloud ORM.
		if field.Anonymous {
			continue
		}

		// Ignore unexported fields
		if field.PkgPath != "" {
			continue
		}

		colMeta := ColumnMeta{
			FieldName: field.Name,
			FieldType: field.Type,
		}

		colTag, hasTag := field.Tag.Lookup("col")
		if !hasTag {
			colMeta.ColumnName = toSnakeCase(field.Name)
		} else {
			parts := strings.Split(colTag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] == "" {
				colMeta.ColumnName = toSnakeCase(field.Name)
			} else {
				colMeta.ColumnName = parts[0]
			}
			for _, part := range parts[1:] {
				switch part {
				case "pk":
					colMeta.IsPK = true
				case "sk":
					colMeta.IsSK = true
					skCount++
					if skCount > 1 {
						panic("multiple fields tagged with 'sk' found; only one sk is allowed per struct")
					}
				case "index":
					colMeta.IsIndex = true
					colMeta.DynamoIndex = fmt.Sprintf("ix%d", ixCount)
					ixCount++
				}
			}
		}

		cols = append(cols, colMeta)
	}

	return cols, tableName, hashPrefix
}

func stringify(v reflect.Value) string {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	return fmt.Sprintf("%v", v.Interface())
}
