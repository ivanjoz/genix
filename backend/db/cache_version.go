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

// Cache-version feature tracks invalidation counters per record-group with a compact persistence model.
// Each table with SaveCacheVersion enabled maps records into groups using uint8(record_id).
// Group versions are uint8 counters stored in cache_version per packed_id and wrap on overflow.
// On write (insert/update), the ORM loads packed table+partition state, increments touched groups, and saves it back.
// On read (select), the ORM loads the same state and assigns the current group version to each record `ccv`.
// Storage is encoded as [group,version,group,version,...] to minimize row size and serialization cost.
// Validation is strict: single numeric key, required partition, and required uint8 CacheVersion/json:"ccv" field.
// Table-level metadata is precomputed during ScyllaTable creation, so runtime hooks avoid repeated schema reflection.

type tableStructCacheMetaGetter interface {
	getCacheVersionFieldIndex() []int
}

// Feature is opt-in per schema and skipped for the cache-version table itself to avoid recursive writes.
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

// Finds the response field that will receive the group cache version for selected rows.
func findCacheVersionFieldIndexInRecordType(recordType reflect.Type) []int {
	if recordType.Kind() == reflect.Pointer {
		recordType = recordType.Elem()
	}
	if recordType.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < recordType.NumField(); i++ {
		field := recordType.Field(i)
		isCacheVersionField := field.Name == "CacheVersion" || getJSONTagName(field) == "ccv"
		if !isCacheVersionField {
			continue
		}
		if field.Type.Kind() != reflect.Uint8 {
			panic(fmt.Sprintf(`Record "%v": cache-version field "%v" must be uint8.`, recordType.Name(), field.Name))
		}
		return field.Index
	}

	return nil
}

// Precomputes and validates all table-level metadata needed by runtime cache-version updates/assignment.
func configureCacheVersionFields[T TableSchemaInterface[T]](schemaStruct *T, scyllaTable *ScyllaTable[any]) {
	if !shouldUseCacheVersionFeature(*scyllaTable) {
		return
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

	scyllaTable.cacheVersionPartitionCol = partitionColumn
	scyllaTable.cacheVersionKeyCol = keyColumn

	if schemaMeta, ok := any(schemaStruct).(tableStructCacheMetaGetter); ok {
		fieldIndex := schemaMeta.getCacheVersionFieldIndex()
		if len(fieldIndex) == 0 {
			panic(fmt.Sprintf(`Table "%v": SaveCacheVersion requires a uint8 "CacheVersion" field or json tag "ccv".`, scyllaTable.name))
		}
		scyllaTable.cacheVersionFieldIndex = append([]int(nil), fieldIndex...)
		return
	}

	panic(fmt.Sprintf(`Table "%v": could not resolve cache-version metadata from schema struct.`, scyllaTable.name))
}

// Decodes compact [group,version,...] bytes into an in-memory map for mutations/lookups.
func decodeCacheVersions(cachedValues []byte) map[uint8]uint8 {
	cacheVersionByGroup := map[uint8]uint8{}
	for i := 0; i+1 < len(cachedValues); i += 2 {
		cacheGroupID := cachedValues[i]
		cacheVersionByGroup[cacheGroupID] = cachedValues[i+1]
	}
	return cacheVersionByGroup
}

func nextCacheVersion(currentVersion uint8) uint8 {
	// Versions are 1..255; rollover keeps the sequence non-zero.
	if currentVersion == 0 || currentVersion == 255 {
		return 1
	}
	return currentVersion + 1
}

func normalizePartitionID(rawPartitionID int64, tableName string) int32 {
	// cache_version uses int32 partition IDs; source int64 partitions are normalized here.
	if rawPartitionID > math.MaxInt32 || rawPartitionID < math.MinInt32 {
		panic(fmt.Sprintf(`Table "%v": partition value %v overflows int32 for cache_version.`, tableName, rawPartitionID))
	}
	return int32(rawPartitionID)
}

func makeCacheVersionPackedID(partitionID int32, tableID int32) int64 {
	// Lossless packing: high 32 bits = partition, low 32 bits = table hash ID.
	return (int64(uint32(partitionID)) << 32) | int64(uint32(tableID))
}

// Encodes map state deterministically by sorted group IDs to avoid unstable write payloads.
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

// Reads one cache-version row by packed_id, defaulting to empty when it doesn't exist yet.
func getCacheVersionsByPackedID(keyspace string, packedID int64) (map[uint8]uint8, error) {
	if keyspace == "" {
		keyspace = connParams.Keyspace
	}
	query := fmt.Sprintf("SELECT cached_values FROM %v.cache_version WHERE packed_id = ? LIMIT 1", keyspace)

	var cachedValues []byte
	err := getScyllaConnection().Query(query, packedID).Scan(&cachedValues)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return map[uint8]uint8{}, nil
		}
		return nil, err
	}
	return decodeCacheVersions(cachedValues), nil
}

// Persists the entire compact group-version state for a packed_id tuple.
func saveCacheVersionsByPackedID(keyspace string, packedID int64, cacheVersionByGroup map[uint8]uint8) error {
	if keyspace == "" {
		keyspace = connParams.Keyspace
	}
	query := fmt.Sprintf("UPDATE %v.cache_version SET cached_values = ? WHERE packed_id = ?", keyspace)
	cachedValues := encodeCacheVersions(cacheVersionByGroup)
	return getScyllaConnection().Query(query, cachedValues, packedID).Exec()
}

// InitCacheVersionTable ensures the cache_version table exists before cache-version reads/writes are executed.
func InitCacheVersionTable() error {
	keyspace := connParams.Keyspace
	if keyspace == "" {
		return errors.New("InitCacheVersionTable: no keyspace configured")
	}

	createTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %v.cache_version (
			packed_id bigint, cached_values blob,
			PRIMARY KEY (packed_id)
		)
		%v;`,
		keyspace, makeStatementWith)

	return QueryExec(createTableQuery)
}

func setRecordCacheVersion(recordPtr reflect.Value, cacheVersionFieldIndex []int, cacheVersion uint8) {
	// Set by pre-resolved field path to avoid repeated tag/name lookups while assigning many records.
	recordPtr.Elem().FieldByIndex(cacheVersionFieldIndex).SetUint(uint64(cacheVersion))
}

// Applies already-loaded versions to each record by reading partition and key directly from mapped columns.
func assignCacheVersionsToRecords[T any](
	records *[]T,
	scyllaTable ScyllaTable[any],
	cacheVersionByPackedID map[int64]map[uint8]uint8,
) {
	tableID := BasicHashInt(scyllaTable.name)
	for i := range *records {
		record := &(*records)[i]
		recordPtr := reflect.ValueOf(record)
		rawRecordPtr := xunsafe.AsPointer(record)

		partitionID := normalizePartitionID(convertToInt64(scyllaTable.cacheVersionPartitionCol.GetRawValue(rawRecordPtr)), scyllaTable.name)
		recordID := convertToInt64(scyllaTable.cacheVersionKeyCol.GetRawValue(rawRecordPtr))
		cacheGroupID := uint8(recordID)
		packedID := makeCacheVersionPackedID(partitionID, tableID)

		cacheVersion := uint8(1)
		if cacheVersionByGroup, exists := cacheVersionByPackedID[packedID]; exists {
			if currentVersion, hasGroup := cacheVersionByGroup[cacheGroupID]; hasGroup {
				cacheVersion = currentVersion
			}
		}

		setRecordCacheVersion(recordPtr, scyllaTable.cacheVersionFieldIndex, cacheVersion)
	}
}

// Write path: increments touched groups per partition and stores the updated compact state back to cache_version.
func updateCacheVersionsAfterWrite[T any](records *[]T, scyllaTable ScyllaTable[any]) error {
	if !shouldUseCacheVersionFeature(scyllaTable) || len(*records) == 0 {
		return nil
	}

	tableID := BasicHashInt(scyllaTable.name)
	cacheGroupsByPackedID := map[int64]map[uint8]struct{}{}

	// Collect unique touched groups, so repeated IDs in the same batch increment only once.
	for i := range *records {
		record := &(*records)[i]
		rawRecordPtr := xunsafe.AsPointer(record)

		partitionID := normalizePartitionID(convertToInt64(scyllaTable.cacheVersionPartitionCol.GetRawValue(rawRecordPtr)), scyllaTable.name)
		recordID := convertToInt64(scyllaTable.cacheVersionKeyCol.GetRawValue(rawRecordPtr))
		cacheGroupID := uint8(recordID)
		packedID := makeCacheVersionPackedID(partitionID, tableID)

		if _, exists := cacheGroupsByPackedID[packedID]; !exists {
			cacheGroupsByPackedID[packedID] = map[uint8]struct{}{}
		}
		cacheGroupsByPackedID[packedID][cacheGroupID] = struct{}{}
	}

	cacheVersionByPackedID := map[int64]map[uint8]uint8{}
	// Read-modify-write per packed key keeps each table+partition group state independent.
	for packedID, cacheGroupsToIncrement := range cacheGroupsByPackedID {
		cacheVersionByGroup, err := getCacheVersionsByPackedID(scyllaTable.keyspace, packedID)
		if err != nil {
			return err
		}

		for cacheGroupID := range cacheGroupsToIncrement {
			cacheVersionByGroup[cacheGroupID] = nextCacheVersion(cacheVersionByGroup[cacheGroupID])
		}

		if err := saveCacheVersionsByPackedID(scyllaTable.keyspace, packedID, cacheVersionByGroup); err != nil {
			return err
		}
		cacheVersionByPackedID[packedID] = cacheVersionByGroup
	}

	assignCacheVersionsToRecords(records, scyllaTable, cacheVersionByPackedID)
	return nil
}

// Read path: loads current group versions and assigns ccv to every selected record.
func assignCacheVersionsAfterSelect[T any](records *[]T, scyllaTable ScyllaTable[any]) error {
	if !shouldUseCacheVersionFeature(scyllaTable) || len(*records) == 0 {
		return nil
	}

	tableID := BasicHashInt(scyllaTable.name)
	cacheVersionByPackedID := map[int64]map[uint8]uint8{}

	// Fetch each packed table+partition state once, then reuse it for all matching records.
	for i := range *records {
		record := &(*records)[i]
		rawRecordPtr := xunsafe.AsPointer(record)
		partitionID := normalizePartitionID(convertToInt64(scyllaTable.cacheVersionPartitionCol.GetRawValue(rawRecordPtr)), scyllaTable.name)
		packedID := makeCacheVersionPackedID(partitionID, tableID)

		if _, exists := cacheVersionByPackedID[packedID]; exists {
			continue
		}

		cacheVersionByGroup, err := getCacheVersionsByPackedID(scyllaTable.keyspace, packedID)
		if err != nil {
			return err
		}
		cacheVersionByPackedID[packedID] = cacheVersionByGroup
	}

	assignCacheVersionsToRecords(records, scyllaTable, cacheVersionByPackedID)
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
	if scyllaTable.cacheVersionPartitionCol == nil || scyllaTable.cacheVersionPartitionCol.IsNil() {
		return columnNames
	}
	if scyllaTable.cacheVersionKeyCol == nil || scyllaTable.cacheVersionKeyCol.IsNil() {
		return columnNames
	}

	// Ensure partition+id are available in scanned records so cache-group assignment is always accurate.
	columnNames = appendColumnIfMissing(columnNames, scyllaTable.cacheVersionPartitionCol.GetName())
	columnNames = appendColumnIfMissing(columnNames, scyllaTable.cacheVersionKeyCol.GetName())
	return columnNames
}

/* Selecting The Version */
type IDCacheVersion struct {
	ID           int64
	CacheVersion uint8
}
