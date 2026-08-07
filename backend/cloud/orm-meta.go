package cloud

import (
	"app/db"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"reflect"
	"strings"
	"sync"
)

// maxMirrorIndexes is the number of shared string GSIs declared on the single DynamoDB
// table (ix1..ix4 in cloud/template.yml). A table that declares more mirrorable indexes
// than this has nowhere to put them, so it is rejected at metadata-build time instead of
// silently losing a lookup path.
const maxMirrorIndexes = 4

// ColumnMeta is one mirrored column. It is resolved from the table struct, never from a
// struct tag: the mirror stores exactly the columns the primary database stores, which is
// also what keeps write-only record fields (a plaintext password) out of the mirror.
type ColumnMeta struct {
	// FieldName is the *record* struct field, which is how values are read by reflection.
	FieldName  string
	FieldType  reflect.Type
	ColumnName string
	IsPK       bool // partition column
	IsSK       bool // clustering key column
}

// IndexKeyMeta is one component of a composite index value.
type IndexKeyMeta struct {
	ColumnName string
	FieldName  string
	// Digits is the zero-padded width this component occupies, taken from the column's
	// .DecimalSize(n). Padding is what makes a range scan over the composite string
	// compare in numeric order; a zero width means the value is written as-is and the
	// component can only be matched for equality.
	Digits int8
}

// IndexMeta is one lookup path: a DynamoDB GSI slot, or a set of D1 indexed columns.
type IndexMeta struct {
	DynamoIndex string
	// PrefixFieldName is the record field whose value prefixes the composite, scoping the
	// index to one tenant. Empty for tables with no partition column.
	PrefixFieldName string
	Keys            []IndexKeyMeta
}

// TableMeta is everything the mirror needs to know about one table, derived from the
// table struct's db.TableSchema.
type TableMeta struct {
	TableName  string
	HashPrefix string
	Columns    []ColumnMeta
	Indexes    []IndexMeta
	// PartitionColumn is the logical tenant column, "" when the table has none.
	PartitionColumn string
}

// mirrorableIndexTypes are the index kinds that correspond to a real lookup structure in
// the mirror. TypeInheritFromKey is a virtual range over a packed key and TypeDelta is
// bound to the primary database's version counter; neither has a mirror equivalent.
var mirrorableIndexTypes = map[int8]bool{
	db.TypeLocalIndex:  true,
	db.TypeGlobalIndex: true,
	db.TypeView:        true,
}

var tableMetaCache sync.Map

// getStructHashPrefix computes a 6-character Base64 URL-encoded hash of the struct name.
// It stays keyed on the *record* struct name so existing DynamoDB pk values remain valid.
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

// buildTableMeta resolves a record type's mirror metadata from its table schema. The
// result is immutable and cached per record type, since compiling the table struct on
// every Insert/Select would repeat work the primary ORM already caches.
func buildTableMeta[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]]() (*TableMeta, error) {
	recordType := reflect.TypeOf(*new(RecordT))
	if cached, isCached := tableMetaCache.Load(recordType); isCached {
		return cached.(*TableMeta), nil
	}

	table := db.TableOf[RecordT, TableT, D]()
	schema := (*table).GetSchema()

	if schema.Name == "" {
		return nil, fmt.Errorf("table schema for %q declares no Name", recordType.Name())
	}

	meta := &TableMeta{
		TableName:  schema.Name,
		HashPrefix: getStructHashPrefix(recordType.Name()),
	}

	if schema.Partition != nil {
		meta.PartitionColumn = schema.Partition.GetName()
	}
	keyColumnNames := map[string]bool{}
	for _, keyColumn := range schema.Keys {
		keyColumnNames[keyColumn.GetName()] = true
	}

	columnsByName := map[string]ColumnMeta{}
	tableValue := reflect.ValueOf(table).Elem()
	tableType := tableValue.Type()

	for fieldIndex := 0; fieldIndex < tableType.NumField(); fieldIndex++ {
		// The embedded TableStruct also satisfies the column interfaces, so it is skipped
		// by position rather than by a failed type assertion.
		if tableType.Field(fieldIndex).Anonymous {
			continue
		}

		column, isColumn := tableValue.Field(fieldIndex).Addr().Interface().(db.Coln)
		if !isColumn {
			continue
		}

		columnInfo := column.GetInfo()
		columnMeta := ColumnMeta{
			FieldName:  columnInfo.FieldName,
			FieldType:  columnInfo.RefType,
			ColumnName: columnInfo.Name,
			IsPK:       columnInfo.Name == meta.PartitionColumn,
			IsSK:       keyColumnNames[columnInfo.Name],
		}
		meta.Columns = append(meta.Columns, columnMeta)
		columnsByName[columnMeta.ColumnName] = columnMeta
	}

	if len(meta.Columns) == 0 {
		return nil, fmt.Errorf("table schema for %q resolved no columns", recordType.Name())
	}

	partitionFieldName := ""
	if meta.PartitionColumn != "" {
		partitionColumn, isDeclared := columnsByName[meta.PartitionColumn]
		if !isDeclared {
			return nil, fmt.Errorf("partition column %q of %q is not a column of the table struct",
				meta.PartitionColumn, recordType.Name())
		}
		partitionFieldName = partitionColumn.FieldName
	}

	for _, schemaIndex := range schema.Indexes {
		if !mirrorableIndexTypes[schemaIndex.Type] || len(schemaIndex.Keys) == 0 {
			continue
		}

		indexMeta := IndexMeta{PrefixFieldName: partitionFieldName}
		if schemaIndex.Partition != nil {
			// An index that overrides the partition is scoped by that column instead.
			overrideColumn, isDeclared := columnsByName[schemaIndex.Partition.GetName()]
			if !isDeclared {
				return nil, fmt.Errorf("index partition override %q of %q is not a column of the table struct",
					schemaIndex.Partition.GetName(), recordType.Name())
			}
			indexMeta.PrefixFieldName = overrideColumn.FieldName
		}

		for _, indexKey := range schemaIndex.Keys {
			keyInfo := indexKey.GetInfo()
			keyColumn, isDeclared := columnsByName[keyInfo.Name]
			if !isDeclared {
				return nil, fmt.Errorf("index key %q of %q is not a column of the table struct",
					keyInfo.Name, recordType.Name())
			}
			// A view's leading column takes its width from the ones after it, so the schema
			// cannot declare one there. The mirror needs an explicit width for every
			// component regardless, and derives the missing ones from the Go type.
			digits := keyInfo.DecimalDigits
			if digits <= 0 {
				digits = inferKeyDigits(keyColumn.FieldType)
			}

			indexMeta.Keys = append(indexMeta.Keys, IndexKeyMeta{
				ColumnName: keyColumn.ColumnName,
				FieldName:  keyColumn.FieldName,
				Digits:     digits,
			})
		}

		indexMeta.DynamoIndex = fmt.Sprintf("ix%d", len(meta.Indexes)+1)
		meta.Indexes = append(meta.Indexes, indexMeta)
	}

	if len(meta.Indexes) > maxMirrorIndexes {
		return nil, fmt.Errorf("table %q declares %d mirrorable indexes but the mirror table only has %d index slots",
			meta.TableName, len(meta.Indexes), maxMirrorIndexes)
	}

	actualMeta, _ := tableMetaCache.LoadOrStore(recordType, meta)
	return actualMeta.(*TableMeta), nil
}

// PartitionField returns the record field backing the partition column, and whether the
// table has one at all.
func (meta *TableMeta) PartitionField() (string, bool) {
	for _, column := range meta.Columns {
		if column.IsPK {
			return column.FieldName, true
		}
	}
	return "", false
}

// KeyField returns the record field backing the single clustering key column. The mirror
// stores one item per record under a single sort key, so a table with several key columns
// has no unambiguous mirror identity.
func (meta *TableMeta) KeyField() (string, error) {
	keyFieldName := ""
	for _, column := range meta.Columns {
		if !column.IsSK {
			continue
		}
		if keyFieldName != "" {
			return "", fmt.Errorf("table %q declares more than one key column; the cloud mirror supports exactly one", meta.TableName)
		}
		keyFieldName = column.FieldName
	}
	if keyFieldName == "" {
		return "", fmt.Errorf("table %q declares no key column", meta.TableName)
	}
	return keyFieldName, nil
}

// buildIndexValue renders one record's value for an index: the partition prefix followed
// by every key component, underscore-separated. This is the string a Dynamo GSI is
// queried against, so writes and reads must build it identically.
func (index IndexMeta) buildIndexValue(record reflect.Value) string {
	parts := make([]string, 0, len(index.Keys)+1)
	if index.PrefixFieldName != "" {
		parts = append(parts, stringify(record.FieldByName(index.PrefixFieldName)))
	}
	for _, key := range index.Keys {
		parts = append(parts, formatIndexKeyValue(key, record.FieldByName(key.FieldName).Interface()))
	}
	return strings.Join(parts, "_")
}

// formatIndexKeyValue renders one component. A declared digit width is zero-padded so
// that lexicographic comparison over the composite matches numeric order.
func formatIndexKeyValue(key IndexKeyMeta, value any) string {
	if key.Digits <= 0 {
		return fmt.Sprintf("%v", value)
	}
	return fmt.Sprintf("%0*d", int(key.Digits), toInt64(value))
}

// inferKeyDigits is the widest decimal representation a Go integer type can take, which
// is the padding an index component needs to compare in numeric order. A non-integer
// component gets no width and can therefore only be matched for equality.
func inferKeyDigits(fieldType reflect.Type) int8 {
	if fieldType == nil {
		return 0
	}
	switch fieldType.Kind() {
	case reflect.Int8, reflect.Uint8:
		return 3
	case reflect.Int16, reflect.Uint16:
		return 5
	case reflect.Int32, reflect.Uint32:
		return 10
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
		return 19
	case reflect.Bool:
		return 1
	}
	return 0
}

func toInt64(value any) int64 {
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflectValue.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(reflectValue.Uint())
	case reflect.Float32, reflect.Float64:
		return int64(reflectValue.Float())
	case reflect.Bool:
		if reflectValue.Bool() {
			return 1
		}
		return 0
	}
	return 0
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
