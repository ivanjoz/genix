package db

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/gocql/gocql"
	"github.com/viant/xunsafe"
)

type cacheVersionConfig struct {
	cacheVersionFieldIndex []int
	partitionColumn        IColInfo
	keyColumn              IColInfo
}

func shouldUseCacheVersionFeature(scyllaTable ScyllaTable[any]) bool {
	// Prevent recursive writes when the cache-version table itself is written.
	return scyllaTable.saveCacheVersion && scyllaTable.name != "cache_version"
}

func getJSONTagName(field reflect.StructField) string {
	tagValue := field.Tag.Get("json")
	if tagValue == "" {
		return ""
	}
	return strings.Split(tagValue, ",")[0]
}

func validateCacheVersionFeature(recordType reflect.Type, scyllaTable ScyllaTable[any]) cacheVersionConfig {
	if recordType.Kind() == reflect.Pointer {
		recordType = recordType.Elem()
	}
	if recordType.Kind() != reflect.Struct {
		panic(fmt.Sprintf(`Table "%v": cache-version feature requires a struct model.`, scyllaTable.name))
	}

	if len(scyllaTable.keys) != 1 {
		panic(fmt.Sprintf(`Table "%v": SaveCacheVersion requires exactly one key column.`, scyllaTable.name))
	}
	keyColumn := scyllaTable.keys[0]
	keyFieldType := keyColumn.GetType().FieldType
	if keyFieldType != "int16" && keyFieldType != "int32" && keyFieldType != "int64" {
		panic(fmt.Sprintf(`Table "%v": SaveCacheVersion key column "%v" must be int16/int32/int64. Found: %v`,
			scyllaTable.name, keyColumn.GetName(), keyFieldType))
	}

	partitionColumn := scyllaTable.GetPartKey()
	if partitionColumn == nil || partitionColumn.IsNil() {
		panic(fmt.Sprintf(`Table "%v": SaveCacheVersion requires a partition column.`, scyllaTable.name))
	}
	partitionFieldType := partitionColumn.GetType().FieldType
	if partitionFieldType != "int32" && partitionFieldType != "int64" {
		panic(fmt.Sprintf(`Table "%v": SaveCacheVersion partition column "%v" must be int32/int64. Found: %v`,
			scyllaTable.name, partitionColumn.GetName(), partitionFieldType))
	}

	cacheVersionFieldIndex := []int(nil)
	for i := 0; i < recordType.NumField(); i++ {
		field := recordType.Field(i)
		isCacheVersionField := field.Name == "CacheVersion" || getJSONTagName(field) == "ccv"
		if !isCacheVersionField {
			continue
		}
		if field.Type.Kind() != reflect.Uint8 {
			panic(fmt.Sprintf(`Table "%v": cache-version field "%v" must be uint8.`, scyllaTable.name, field.Name))
		}
		cacheVersionFieldIndex = field.Index
		break
	}

	if len(cacheVersionFieldIndex) == 0 {
		panic(fmt.Sprintf(`Table "%v": SaveCacheVersion requires a uint8 "CacheVersion" field or json tag "ccv".`, scyllaTable.name))
	}

	return cacheVersionConfig{
		cacheVersionFieldIndex: cacheVersionFieldIndex,
		partitionColumn:        partitionColumn,
		keyColumn:              keyColumn,
	}
}

func decodeCacheVersions(cachedValues []byte) map[uint8]uint8 {
	cacheVersionByGroup := map[uint8]uint8{}
	for i := 0; i+1 < len(cachedValues); i += 2 {
		cacheGroupID := cachedValues[i]
		cacheVersionByGroup[cacheGroupID] = cachedValues[i+1]
	}
	return cacheVersionByGroup
}

func normalizePartitionID(rawPartitionID int64, tableName string) int32 {
	// cache_version uses int32 partition IDs; source int64 partitions are normalized here.
	if rawPartitionID > math.MaxInt32 || rawPartitionID < math.MinInt32 {
		panic(fmt.Sprintf(`Table "%v": partition value %v overflows int32 for cache_version.`, tableName, rawPartitionID))
	}
	return int32(rawPartitionID)
}

func encodeCacheVersions(cacheVersionByGroup map[uint8]uint8) []byte {
	cacheGroupIDs := make([]int, 0, len(cacheVersionByGroup))
	for cacheGroupID := range cacheVersionByGroup {
		cacheGroupIDs = append(cacheGroupIDs, int(cacheGroupID))
	}
	sort.Ints(cacheGroupIDs)

	cachedValues := make([]byte, 0, len(cacheGroupIDs)*2)
	for _, cacheGroupID := range cacheGroupIDs {
		cacheGroupIDU8 := uint8(cacheGroupID)
		cachedValues = append(cachedValues, cacheGroupIDU8, cacheVersionByGroup[cacheGroupIDU8])
	}
	return cachedValues
}

func getCacheVersionsByPartition(keyspace string, partitionID int32, tableID int32) (map[uint8]uint8, error) {
	if keyspace == "" {
		keyspace = connParams.Keyspace
	}
	query := fmt.Sprintf("SELECT cached_values FROM %v.cache_version WHERE partition = ? AND table_id = ? LIMIT 1", keyspace)

	var cachedValues []byte
	err := getScyllaConnection().Query(query, partitionID, tableID).Scan(&cachedValues)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return map[uint8]uint8{}, nil
		}
		return nil, err
	}
	return decodeCacheVersions(cachedValues), nil
}

func saveCacheVersionsByPartition(keyspace string, partitionID int32, tableID int32, cacheVersionByGroup map[uint8]uint8) error {
	if keyspace == "" {
		keyspace = connParams.Keyspace
	}
	query := fmt.Sprintf("UPDATE %v.cache_version SET cached_values = ? WHERE partition = ? AND table_id = ?", keyspace)
	cachedValues := encodeCacheVersions(cacheVersionByGroup)
	return getScyllaConnection().Query(query, cachedValues, partitionID, tableID).Exec()
}

func setRecordCacheVersion(recordPtr reflect.Value, cacheVersionFieldIndex []int, cacheVersion uint8) {
	// Set by pre-resolved field path to avoid repeated tag/name lookups while assigning many records.
	recordPtr.Elem().FieldByIndex(cacheVersionFieldIndex).SetUint(uint64(cacheVersion))
}

func assignCacheVersionsToRecords[T any](
	records *[]T,
	config cacheVersionConfig,
	cacheVersionByPartition map[int32]map[uint8]uint8,
) {
	for i := range *records {
		record := &(*records)[i]
		recordPtr := reflect.ValueOf(record)
		rawRecordPtr := xunsafe.AsPointer(record)

		partitionID := normalizePartitionID(convertToInt64(config.partitionColumn.GetRawValue(rawRecordPtr)), config.keyColumn.GetInfo().Name)
		recordID := convertToInt64(config.keyColumn.GetRawValue(rawRecordPtr))
		cacheGroupID := uint8(recordID)

		cacheVersion := uint8(0)
		if cacheVersionByGroup, exists := cacheVersionByPartition[partitionID]; exists {
			cacheVersion = cacheVersionByGroup[cacheGroupID]
		}

		setRecordCacheVersion(recordPtr, config.cacheVersionFieldIndex, cacheVersion)
	}
}

func updateCacheVersionsAfterWrite[T any](records *[]T, scyllaTable ScyllaTable[any]) error {
	if !shouldUseCacheVersionFeature(scyllaTable) || len(*records) == 0 {
		return nil
	}

	config := validateCacheVersionFeature(reflect.TypeOf(*new(T)), scyllaTable)
	tableID := BasicHashInt(scyllaTable.name)

	cacheGroupsByPartition := map[int32]map[uint8]struct{}{}
	for i := range *records {
		record := &(*records)[i]
		rawRecordPtr := xunsafe.AsPointer(record)

		partitionID := normalizePartitionID(convertToInt64(config.partitionColumn.GetRawValue(rawRecordPtr)), scyllaTable.name)
		recordID := convertToInt64(config.keyColumn.GetRawValue(rawRecordPtr))
		cacheGroupID := uint8(recordID)

		if _, exists := cacheGroupsByPartition[partitionID]; !exists {
			cacheGroupsByPartition[partitionID] = map[uint8]struct{}{}
		}
		cacheGroupsByPartition[partitionID][cacheGroupID] = struct{}{}
	}

	cacheVersionByPartition := map[int32]map[uint8]uint8{}
	for partitionID, cacheGroupsToIncrement := range cacheGroupsByPartition {
		cacheVersionByGroup, err := getCacheVersionsByPartition(scyllaTable.keyspace, partitionID, tableID)
		if err != nil {
			return err
		}

		for cacheGroupID := range cacheGroupsToIncrement {
			// uint8 overflow is intentional: 255 + 1 wraps to 0.
			cacheVersionByGroup[cacheGroupID]++
		}

		if err := saveCacheVersionsByPartition(scyllaTable.keyspace, partitionID, tableID, cacheVersionByGroup); err != nil {
			return err
		}
		cacheVersionByPartition[partitionID] = cacheVersionByGroup
	}

	assignCacheVersionsToRecords(records, config, cacheVersionByPartition)
	return nil
}

func assignCacheVersionsAfterSelect[T any](records *[]T, scyllaTable ScyllaTable[any]) error {
	if !shouldUseCacheVersionFeature(scyllaTable) || len(*records) == 0 {
		return nil
	}

	config := validateCacheVersionFeature(reflect.TypeOf(*new(T)), scyllaTable)
	tableID := BasicHashInt(scyllaTable.name)

	cacheVersionByPartition := map[int32]map[uint8]uint8{}
	for i := range *records {
		record := &(*records)[i]
		rawRecordPtr := xunsafe.AsPointer(record)
		partitionID := normalizePartitionID(convertToInt64(config.partitionColumn.GetRawValue(rawRecordPtr)), scyllaTable.name)

		if _, exists := cacheVersionByPartition[partitionID]; exists {
			continue
		}
		cacheVersionByGroup, err := getCacheVersionsByPartition(scyllaTable.keyspace, partitionID, tableID)
		if err != nil {
			return err
		}
		cacheVersionByPartition[partitionID] = cacheVersionByGroup
	}

	assignCacheVersionsToRecords(records, config, cacheVersionByPartition)
	return nil
}

func appendColumnIfMissing(columnNames []string, columnName string) []string {
	for _, currentColumnName := range columnNames {
		if currentColumnName == columnName {
			return columnNames
		}
	}
	return append(columnNames, columnName)
}

func ensureCacheVersionColumnsForSelect(columnNames []string, scyllaTable ScyllaTable[any]) []string {
	if !shouldUseCacheVersionFeature(scyllaTable) {
		return columnNames
	}
	partitionColumn := scyllaTable.GetPartKey()
	if partitionColumn == nil || partitionColumn.IsNil() || len(scyllaTable.keys) == 0 {
		return columnNames
	}

	// Ensure partition+id are available in scanned records so cache-group assignment is always accurate.
	columnNames = appendColumnIfMissing(columnNames, partitionColumn.GetName())
	columnNames = appendColumnIfMissing(columnNames, scyllaTable.keys[0].GetName())
	return columnNames
}
