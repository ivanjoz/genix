package db

import (
	"fmt"
	"strings"
	"sync"
)

// Generic by-IDs reads let one endpoint resolve "give me the label for these IDs" for any table,
// instead of one handler + one full-record payload per table. A table opts in through
// TableSchema.GenericRecord, which maps its own columns onto the flat GenericRecord shape.
//
// Everything expensive is precompiled during table creation (configureGenericRecordFields):
// column lookup, the SELECT projection string, and one closure per column that owns a typed scan
// buffer and copies it into the result. The read loop therefore performs no reflection, no map
// lookups and no type switching per row — it just walks a slice of closures.
//
// Requirements are inherited from the cache-version feature: SaveCacheVersion enabled, exactly one
// integer key column, integer partition. That is what makes "ID is an integer" always true here.

// GenericRecordSchema declares which columns fill the generic shape. Name is required; the rest are
// optional. ID and Status are not listed: they are always the table's single key column and its
// "status" column, resolved automatically.
type GenericRecordSchema struct {
	Name Coln // string  — the display label
	S1   Coln // string  — optional secondary text (e.g. SKU, document number)
	S2   Coln // string  — optional
	N1   Coln // integer — optional numeric (e.g. price, foreign key)
	N2   Coln // integer — optional
}

func (schema GenericRecordSchema) isEmpty() bool {
	return schema.Name == nil && schema.S1 == nil && schema.S2 == nil && schema.N1 == nil && schema.N2 == nil
}

// GenericRecord is the flat, table-agnostic row returned to the client. Short json tags keep the
// payload small, which is the whole point of this endpoint. ID keeps its capitalised name and ccv/ss
// are required by the frontend by-ID cache contract (IMinimalRecord).
type GenericRecord struct {
	ID           int64  `json:"ID"`
	Name         string `json:"nm,omitempty"`
	S1           string `json:"s1,omitempty"`
	S2           string `json:"s2,omitempty"`
	N1           int64  `json:"n1,omitempty"`
	N2           int64  `json:"n2,omitempty"`
	Status       int8   `json:"ss,omitempty"`
	CacheVersion uint8  `json:"ccv,omitempty"`
}

// genericScanSlot binds a typed buffer gocql scans into to the assignment that moves it into the
// record. Allocated once per query and reused for every row of that query.
type genericScanSlot struct {
	scanTarget any
	assign     func(record *GenericRecord)
}

// genericColumnAccessor is the precompiled form of one mapped column: the type switch already
// happened at table-build time, so makeScanSlot only allocates the buffer and closes over it.
type genericColumnAccessor struct {
	columnName   string
	makeScanSlot func() genericScanSlot
}

type genericRecordPlan struct {
	// projection is the joined column list, built once so no per-query string work is needed.
	projection string
	accessors  []genericColumnAccessor
}

// newScanSlots materializes one buffer set for a query. Row scanning then reuses these buffers.
func (plan *genericRecordPlan) newScanSlots() ([]any, []genericScanSlot) {
	slots := make([]genericScanSlot, len(plan.accessors))
	scanTargets := make([]any, len(plan.accessors))
	for accessorIndex, accessor := range plan.accessors {
		slots[accessorIndex] = accessor.makeScanSlot()
		scanTargets[accessorIndex] = slots[accessorIndex].scanTarget
	}
	return scanTargets, slots
}

// Precompiles a string column: the buffer is a *string and the assignment is a direct field write.
func makeGenericStringAccessor(column IColInfo, assign func(*GenericRecord, string)) genericColumnAccessor {
	return genericColumnAccessor{
		columnName: column.GetName(),
		makeScanSlot: func() genericScanSlot {
			scannedValue := new(string)
			return genericScanSlot{
				scanTarget: scannedValue,
				assign:     func(record *GenericRecord) { assign(record, *scannedValue) },
			}
		},
	}
}

// Precompiles an integer column. The width switch happens here, once per table, so each row only
// runs a direct widening conversion instead of an interface type switch.
func makeGenericIntAccessor(tableName string, column IColInfo, assign func(*GenericRecord, int64)) genericColumnAccessor {
	accessor := genericColumnAccessor{columnName: column.GetName()}

	switch column.GetType().FieldType {
	case "int8":
		accessor.makeScanSlot = func() genericScanSlot {
			scannedValue := new(int8)
			return genericScanSlot{scanTarget: scannedValue,
				assign: func(record *GenericRecord) { assign(record, int64(*scannedValue)) }}
		}
	case "int16":
		accessor.makeScanSlot = func() genericScanSlot {
			scannedValue := new(int16)
			return genericScanSlot{scanTarget: scannedValue,
				assign: func(record *GenericRecord) { assign(record, int64(*scannedValue)) }}
		}
	case "int32":
		accessor.makeScanSlot = func() genericScanSlot {
			scannedValue := new(int32)
			return genericScanSlot{scanTarget: scannedValue,
				assign: func(record *GenericRecord) { assign(record, int64(*scannedValue)) }}
		}
	case "int64", "int":
		accessor.makeScanSlot = func() genericScanSlot {
			scannedValue := new(int64)
			return genericScanSlot{scanTarget: scannedValue,
				assign: func(record *GenericRecord) { assign(record, *scannedValue) }}
		}
	default:
		panic(fmt.Sprintf(`Table "%v": GenericRecord column "%v" must be an integer. Found: %v`,
			tableName, column.GetName(), column.GetType().FieldType))
	}

	return accessor
}

// Resolves a schema-declared column to the table's mapped column, failing loudly on typos.
func resolveGenericColumn(scyllaTable *ScyllaTable, declaredColumn Coln, fieldName string) IColInfo {
	if declaredColumn == nil {
		return nil
	}
	column := scyllaTable.columnsMap[declaredColumn.GetInfo().Name]
	if column == nil || column.IsNil() {
		panic(fmt.Sprintf(`Table "%v": GenericRecord.%v column "%v" is not mapped in the table.`,
			scyllaTable.name, fieldName, declaredColumn.GetInfo().Name))
	}
	return column
}

func mustBeStringColumn(tableName string, column IColInfo, fieldName string) {
	if column.GetType().FieldType != "string" {
		panic(fmt.Sprintf(`Table "%v": GenericRecord.%v column "%v" must be string. Found: %v`,
			tableName, fieldName, column.GetName(), column.GetType().FieldType))
	}
}

// configureGenericRecordFields validates the GenericRecord config and precompiles its access plan
// once per table build, so no query-time schema reflection is ever needed.
func configureGenericRecordFields(scyllaTable *ScyllaTable, schema TableSchema) {
	if schema.GenericRecord.isEmpty() {
		return
	}

	// ccv is what makes the generic read incremental, so the two features are inseparable.
	if !shouldUseCacheVersionFeature(*scyllaTable) {
		panic(fmt.Sprintf(`Table "%v": GenericRecord requires SaveCacheVersion enabled.`, scyllaTable.name))
	}
	if schema.GenericRecord.Name == nil {
		panic(fmt.Sprintf(`Table "%v": GenericRecord requires a Name column.`, scyllaTable.name))
	}

	plan := genericRecordPlan{}

	// The ID accessor comes first and is always the cache-version key column, which the
	// cache-version validation already guarantees to be a single integer key.
	plan.accessors = append(plan.accessors, makeGenericIntAccessor(scyllaTable.name, scyllaTable.cacheVersionKeyCol,
		func(record *GenericRecord, value int64) { record.ID = value }))

	nameColumn := resolveGenericColumn(scyllaTable, schema.GenericRecord.Name, "Name")
	mustBeStringColumn(scyllaTable.name, nameColumn, "Name")
	plan.accessors = append(plan.accessors, makeGenericStringAccessor(nameColumn,
		func(record *GenericRecord, value string) { record.Name = value }))

	if column := resolveGenericColumn(scyllaTable, schema.GenericRecord.S1, "S1"); column != nil {
		mustBeStringColumn(scyllaTable.name, column, "S1")
		plan.accessors = append(plan.accessors, makeGenericStringAccessor(column,
			func(record *GenericRecord, value string) { record.S1 = value }))
	}
	if column := resolveGenericColumn(scyllaTable, schema.GenericRecord.S2, "S2"); column != nil {
		mustBeStringColumn(scyllaTable.name, column, "S2")
		plan.accessors = append(plan.accessors, makeGenericStringAccessor(column,
			func(record *GenericRecord, value string) { record.S2 = value }))
	}
	if column := resolveGenericColumn(scyllaTable, schema.GenericRecord.N1, "N1"); column != nil {
		plan.accessors = append(plan.accessors, makeGenericIntAccessor(scyllaTable.name, column,
			func(record *GenericRecord, value int64) { record.N1 = value }))
	}
	if column := resolveGenericColumn(scyllaTable, schema.GenericRecord.N2, "N2"); column != nil {
		plan.accessors = append(plan.accessors, makeGenericIntAccessor(scyllaTable.name, column,
			func(record *GenericRecord, value int64) { record.N2 = value }))
	}

	// Status lets the client cache tombstones instead of re-requesting deleted rows forever. Every
	// table has one, but stay tolerant: a table without it simply reports Status 0.
	if statusColumn, exists := scyllaTable.columnsMap["status"]; exists && !statusColumn.IsNil() {
		plan.accessors = append(plan.accessors, makeGenericIntAccessor(scyllaTable.name, statusColumn,
			func(record *GenericRecord, value int64) { record.Status = int8(value) }))
	}

	columnNames := make([]string, len(plan.accessors))
	for accessorIndex, accessor := range plan.accessors {
		columnNames[accessorIndex] = accessor.columnName
	}
	plan.projection = strings.Join(columnNames, ", ")

	scyllaTable.genericRecordPlan = &plan
}

/* Table name registry */

// Resolving a table from a name string cannot use generics, so tables register a lazy factory. The
// factory is cheap (a closure); the ScyllaTable is only compiled when a name is actually requested,
// and MakeScyllaTable then memoizes it in scyllaTableCache.
var (
	tableFactoriesMutex  sync.RWMutex
	tableFactoriesByName = map[string]func() ScyllaTable{}
)

// RegisterTableFactory is called from generated code (see scripts/controllers/controllers_generator.go)
// for every table in the project.
func RegisterTableFactory(tableName string, makeTableFunc func() ScyllaTable) {
	tableFactoriesMutex.Lock()
	defer tableFactoriesMutex.Unlock()
	tableFactoriesByName[tableName] = makeTableFunc
}

// GenericRecordProjection returns the precompiled column list used for this table's generic by-IDs
// reads, or "" when the table does not expose them. Kept exported for diagnostics and for tests that
// verify a real schema's GenericRecord mapping without needing a live database.
func (e ScyllaTable) GenericRecordProjection() string {
	if e.genericRecordPlan == nil {
		return ""
	}
	return e.genericRecordPlan.projection
}

// ResolveTableByName exposes the name registry so callers outside db (tests, diagnostics) can force a
// table's metadata to compile and surface any schema validation panic early.
func ResolveTableByName(tableName string) (ScyllaTable, error) {
	return resolveTableByName(tableName)
}

func resolveTableByName(tableName string) (ScyllaTable, error) {
	tableFactoriesMutex.RLock()
	makeTableFunc, exists := tableFactoriesByName[tableName]
	tableFactoriesMutex.RUnlock()

	if !exists {
		return ScyllaTable{}, fmt.Errorf("la tabla %q no está registrada", tableName)
	}
	return makeTableFunc(), nil
}

/* Query */

// QueryCachedGenericByIDs resolves IDs to the flat GenericRecord shape for any table that opts in
// through TableSchema.GenericRecord, using the same cache-version filtering as QueryCachedIDs:
// records whose client ccv still matches the server are not read from the table at all.
func QueryCachedGenericByIDs(tableName string, cachedIDs []IDCacheVersion) ([]GenericRecord, error) {
	if len(cachedIDs) == 0 {
		fmt.Println("QueryCachedGenericByIDs: empty request, nothing to process")
		return nil, nil
	}

	scyllaTable, err := resolveTableByName(tableName)
	if err != nil {
		return nil, err
	}

	// A nil plan is the opt-in allowlist: tables that did not declare GenericRecord are never
	// readable through the shared generic endpoint.
	plan := scyllaTable.genericRecordPlan
	if plan == nil {
		return nil, fmt.Errorf("la tabla %q no expone registros genéricos", tableName)
	}
	if err := prepareCachedIDsTable(&scyllaTable, "QueryCachedGenericByIDs"); err != nil {
		return nil, err
	}

	fetchPlan, err := planCachedIDsFetch(scyllaTable, cachedIDs)
	if err != nil {
		return nil, err
	}
	if !fetchPlan.hasRecordsToFetch() {
		fmt.Println("QueryCachedGenericByIDs: all IDs resolved from cache_version, skipping table select")
		return nil, nil
	}

	tableID := BasicHashInt(scyllaTable.name)
	records := []GenericRecord{}

	err = forEachCachedIDsBatch(fetchPlan, scyllaTable, plan.projection,
		func(queryString string, queryValues []any, partitionID int32) error {
			// Buffers are allocated once per batch and overwritten per row by gocql.
			scanTargets, scanSlots := plan.newScanSlots()
			packedID := makeCacheVersionPackedID(partitionID, tableID)
			cacheVersionByGroup := fetchPlan.cacheVersionByPackedID[packedID]

			iterator := getScyllaConnection().Query(queryString, queryValues...).Iter()
			scanner := iterator.Scanner()
			for scanner.Next() {
				if err := scanner.Scan(scanTargets...); err != nil {
					iterator.Close()
					return err
				}
				record := GenericRecord{}
				for _, scanSlot := range scanSlots {
					scanSlot.assign(&record)
				}
				// ccv comes from the group state already loaded in phase 1 — no extra read.
				record.CacheVersion = resolveCacheVersionForID(cacheVersionByGroup, record.ID)
				records = append(records, record)
			}
			return iterator.Close()
		})
	if err != nil {
		return nil, err
	}

	return records, nil
}
