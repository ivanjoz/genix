package db

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/viant/xunsafe"
)

type structFieldMetadataCacheEntry struct {
	recordType             reflect.Type
	fieldMetadataByName    map[string]columnInfo
	cacheVersionFieldIndex []int
}

type scyllaTableCacheEntry struct {
	once  sync.Once
	table ScyllaTable[any]
}

var (
	structFieldMetadataCache sync.Map
	scyllaTableCache         sync.Map
)

func getOrBuildStructFieldMetadata(recordType reflect.Type) *structFieldMetadataCacheEntry {
	if cachedEntry, cacheHit := structFieldMetadataCache.Load(recordType); cacheHit {
		return cachedEntry.(*structFieldMetadataCacheEntry)
	}

	metadataByFieldName := map[string]columnInfo{}
	for fieldIndex := 0; fieldIndex < recordType.NumField(); fieldIndex++ {
		recordField := recordType.Field(fieldIndex)
		if recordField.Name == "TableStruct" {
			continue
		}

		unsafeField := xunsafe.FieldByName(recordType, recordField.Name)
		columnType := GetColTypeByName(recordField.Type.String(), "")
		if columnType.Type == 0 {
			columnType = GetColTypeByID(9)
		}

		columnName := ""
		if dbTag := recordField.Tag.Get("db"); dbTag != "" {
			columnName = strings.Split(dbTag, ",")[0]
		}

		metadataByFieldName[recordField.Name] = columnInfo{
			colInfo: colInfo{
				Name:      columnName,
				FieldIdx:  fieldIndex,
				FieldName: recordField.Name,
				RefType:   recordField.Type,
				Field:     unsafeField,
			},
			colType: columnType,
		}
	}

	metadataEntry := &structFieldMetadataCacheEntry{
		recordType:             recordType,
		fieldMetadataByName:    metadataByFieldName,
		cacheVersionFieldIndex: findCacheVersionFieldIndexInRecordType(recordType),
	}

	actualEntry, _ := structFieldMetadataCache.LoadOrStore(recordType, metadataEntry)
	return actualEntry.(*structFieldMetadataCacheEntry)
}

func getOrCompileScyllaTable[T TableInterface[T]](schemaStruct *T) ScyllaTable[any] {
	cacheKey := reflect.TypeOf(schemaStruct).Elem().PkgPath() + "." + reflect.TypeOf(schemaStruct).Elem().Name()

	cacheEntryAny, _ := scyllaTableCache.LoadOrStore(cacheKey, &scyllaTableCacheEntry{})
	cacheEntry := cacheEntryAny.(*scyllaTableCacheEntry)
	cacheEntry.once.Do(func() {
		if ShouldLog() {
			fmt.Printf("Compiling ScyllaTable metadata once for %s\n", cacheKey)
		}
		cacheEntry.table = makeTable(schemaStruct)
	})
	return cacheEntry.table
}

// resetORMTableCachesForTesting clears ORM metadata caches for deterministic benchmarks/tests.
func resetORMTableCachesForTesting() {
	structFieldMetadataCache = sync.Map{}
	scyllaTableCache = sync.Map{}
}
