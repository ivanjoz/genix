package db

import (
	"slices"
	"strings"
	"testing"
	"unsafe"
)

// containsAny reports whether values holds an entry equal to needle.
// slices.Contains can't be used on []any because any is not a comparable
// type parameter, so the prepared-statement assertions walk Args manually.
func containsAny(values []any, needle any) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

type updateCounterRecord struct {
	TableStruct[updateCounterSchema, updateCounterRecord]
	CompanyID int32  `db:"empresa_id"`
	ID        int64  `db:"id"`
	Nombre    string `db:"nombre"`
}

type updateCounterSchema struct {
	TableStruct[updateCounterSchema, updateCounterRecord]
	CompanyID Col[updateCounterSchema, int32]
	ID        Col[updateCounterSchema, int64]
	Nombre    Col[updateCounterSchema, string]
}

type updateCounterDisabledRecord struct {
	TableStruct[updateCounterDisabledSchema, updateCounterDisabledRecord]
	CompanyID int32  `db:"empresa_id"`
	ID        int64  `db:"id"`
	Nombre    string `db:"nombre"`
}

type updateCounterDisabledSchema struct {
	TableStruct[updateCounterDisabledSchema, updateCounterDisabledRecord]
	CompanyID Col[updateCounterDisabledSchema, int32]
	ID        Col[updateCounterDisabledSchema, int64]
	Nombre    Col[updateCounterDisabledSchema, string]
}

type indexGroupRecord struct {
	TableStruct[indexGroupSchema, indexGroupRecord]
	CompanyID  int32   `db:"empresa_id"`
	ID         int64   `db:"id"`
	Date      int16   `db:"date"`
	ClientID   int32   `db:"client_id"`
	ProductIDs []int32 `db:"product_ids,list"`
}

type indexGroupSchema struct {
	TableStruct[indexGroupSchema, indexGroupRecord]
	CompanyID  Col[indexGroupSchema, int32]
	ID         Col[indexGroupSchema, int64]
	Date      Col[indexGroupSchema, int16]
	ClientID   Col[indexGroupSchema, int32]
	ProductIDs Col[indexGroupSchema, []int32]
}

type weekIndexGroupRecord struct {
	TableStruct[weekIndexGroupSchema, weekIndexGroupRecord]
	CompanyID int32 `db:"empresa_id"`
	ID        int64 `db:"id"`
	Date     int16 `db:"date"`
	Status    int8  `db:"status"`
}

type weekIndexGroupSchema struct {
	TableStruct[weekIndexGroupSchema, weekIndexGroupRecord]
	CompanyID Col[weekIndexGroupSchema, int32]
	ID        Col[weekIndexGroupSchema, int64]
	Date     Col[weekIndexGroupSchema, int16]
	Status    Col[weekIndexGroupSchema, int8]
}

func (e updateCounterSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:  "test_keyspace",
		Name:      "update_counter_records",
		Partition: e.CompanyID,
		Keys:      []Coln{e.ID},
	}
}

func (e updateCounterDisabledSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:             "test_keyspace",
		Name:                 "update_counter_records_disabled",
		Partition:            e.CompanyID,
		Keys:                 []Coln{e.ID},
		DisableUpdateCounter: true,
	}
}

func (e indexGroupSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:  "test_keyspace",
		Name:      "index_group_records",
		Partition: e.CompanyID,
		Keys:      []Coln{e.ID},
		Indexes: []Index{
			{
				Keys:          []Coln{e.Date},
				UseIndexGroup: true,
			},
			{
				Keys:          []Coln{e.Date.StoreAsWeek(), e.ClientID, e.ProductIDs},
				UseIndexGroup: true,
			},
		},
	}
}

func (e weekIndexGroupSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:  "test_keyspace",
		Name:      "week_index_group_records",
		Partition: e.CompanyID,
		Keys:      []Coln{e.ID},
		Indexes: []Index{
			{
				Keys:          []Coln{e.Date.StoreAsWeek(), e.Status},
				UseIndexGroup: true,
			},
		},
	}
}

func TestMakeScyllaTableAddsManagedAuditColumns(t *testing.T) {
	scyllaTable := MakeScyllaTable[updateCounterRecord, updateCounterSchema]()

	for _, columnName := range []string{"created", "updated", "update_counter"} {
		if scyllaTable.columnsMap[columnName] == nil {
			t.Fatalf("expected managed column %q to exist in table metadata", columnName)
		}
	}
}

func TestMakeScyllaTableSkipsManagedUpdateCounterWhenDisabled(t *testing.T) {
	scyllaTable := MakeScyllaTable[updateCounterDisabledRecord, updateCounterDisabledSchema]()

	if scyllaTable.columnsMap["created"] == nil || scyllaTable.columnsMap["updated"] == nil {
		t.Fatalf("expected created and updated managed columns to remain enabled")
	}
	if scyllaTable.columnsMap["update_counter"] != nil {
		t.Fatalf("expected update_counter managed column to be disabled")
	}
	if scyllaTable.updateCounterCol != nil {
		t.Fatalf("expected updateCounterCol runtime metadata to be nil when disabled")
	}
}

func TestApplyWriteManagedColumnsUsesPartitionScopedUpdatedCounter(t *testing.T) {
	scyllaTable := MakeScyllaTable[updateCounterRecord, updateCounterSchema]()
	records := []updateCounterRecord{
		{CompanyID: 7, ID: 1, Nombre: "uno"},
		{CompanyID: 7, ID: 2, Nombre: "dos"},
		{CompanyID: 8, ID: 3, Nombre: "tres"},
	}

	originalCounterFetcher := getWriteCounterValue
	originalManagedUnixTime := getManagedUnixTime
	t.Cleanup(func() {
		getWriteCounterValue = originalCounterFetcher
		getManagedUnixTime = originalManagedUnixTime
	})

	getManagedUnixTime = func() int32 { return 77 }

	counterCalls := []string{}
	getWriteCounterValue = func(keyspace string, name string, increment int) (int64, error) {
		counterCalls = append(counterCalls, name)
		if keyspace != "test_keyspace" {
			t.Fatalf("unexpected keyspace: %s", keyspace)
		}
		if increment != 1 {
			t.Fatalf("unexpected counter increment: %d", increment)
		}

		switch name {
		case "x7_update_counter_records_updated":
			return 41, nil
		case "x8_update_counter_records_updated":
			return 52, nil
		default:
			t.Fatalf("unexpected counter name: %s", name)
			return 0, nil
		}
	}

	managedValues, err := applyWriteManagedColumns(&records, scyllaTable, true)
	if err != nil {
		t.Fatalf("applyWriteManagedColumns returned error: %v", err)
	}

	if len(counterCalls) != 2 {
		t.Fatalf("expected one counter fetch per partition, got %v", counterCalls)
	}
	slices.Sort(counterCalls)
	if counterCalls[0] != "x7_update_counter_records_updated" || counterCalls[1] != "x8_update_counter_records_updated" {
		t.Fatalf("unexpected counter names: %v", counterCalls)
	}

	for recordIndex := range records {
		if managedValues.createdValues[recordIndex] != int32(77) {
			t.Fatalf("expected managed created 77, got %v at index %d", managedValues.createdValues[recordIndex], recordIndex)
		}
		if managedValues.updatedValues[recordIndex] != int32(77) {
			t.Fatalf("expected managed updated 77, got %v at index %d", managedValues.updatedValues[recordIndex], recordIndex)
		}
	}
	if managedValues.updateCounterValues[0] != int32(41) || managedValues.updateCounterValues[1] != int32(41) || managedValues.updateCounterValues[2] != int32(52) {
		t.Fatalf("unexpected managed update counters: %v", managedValues.updateCounterValues)
	}
}

func TestMakeUpdateStatementsAlwaysPersistsManagedUpdatedColumns(t *testing.T) {
	records := []updateCounterRecord{
		{CompanyID: 7, ID: 1, Nombre: "nuevo"},
	}
	table := Table[updateCounterRecord, updateCounterSchema]()

	originalCounterFetcher := getWriteCounterValue
	originalManagedUnixTime := getManagedUnixTime
	t.Cleanup(func() {
		getWriteCounterValue = originalCounterFetcher
		getManagedUnixTime = originalManagedUnixTime
	})

	getManagedUnixTime = func() int32 { return 77 }
	getWriteCounterValue = func(keyspace string, name string, increment int) (int64, error) {
		return 41, nil
	}

	preparedStatements := MakeUpdateStatements(&records, table.Nombre)
	if len(preparedStatements) != 1 {
		t.Fatalf("expected one update statement, got %d", len(preparedStatements))
	}
	preparedStatement := preparedStatements[0]

	for _, expectedSetClause := range []string{"nombre = ?", "updated = ?", "update_counter = ?"} {
		if !strings.Contains(preparedStatement.Stmt, expectedSetClause) {
			t.Fatalf("expected SET clause %q in statement: %s", expectedSetClause, preparedStatement.Stmt)
		}
	}

	expectedBoundValues := []any{"nuevo", int32(77), int32(41)}
	for _, expectedValue := range expectedBoundValues {
		if !containsAny(preparedStatement.Args, expectedValue) {
			t.Fatalf("expected bound value %v in args %v", expectedValue, preparedStatement.Args)
		}
	}
}

func TestMakeInsertStatementIncludesManagedAuditColumnsWithoutStructFields(t *testing.T) {
	records := []updateCounterRecord{
		{CompanyID: 7, ID: 1, Nombre: "nuevo"},
	}

	originalCounterFetcher := getWriteCounterValue
	originalManagedUnixTime := getManagedUnixTime
	t.Cleanup(func() {
		getWriteCounterValue = originalCounterFetcher
		getManagedUnixTime = originalManagedUnixTime
	})

	getManagedUnixTime = func() int32 { return 77 }
	getWriteCounterValue = func(keyspace string, name string, increment int) (int64, error) {
		return 41, nil
	}

	insertStatements := MakeInsertStatement(&records)
	if len(insertStatements) != 1 {
		t.Fatalf("expected one insert statement, got %d", len(insertStatements))
	}
	preparedStatement := insertStatements[0]

	for _, expectedColumn := range []string{"created", "updated", "update_counter"} {
		if !strings.Contains(preparedStatement.Stmt, expectedColumn) {
			t.Fatalf("expected managed column %q in insert statement: %s", expectedColumn, preparedStatement.Stmt)
		}
	}

	for _, expectedValue := range []any{int32(77), int32(41)} {
		if !containsAny(preparedStatement.Args, expectedValue) {
			t.Fatalf("expected managed value %v in args %v", expectedValue, preparedStatement.Args)
		}
	}
}

func TestGetIndexUpdatedTableCreateScriptUsesExpectedPrimaryKey(t *testing.T) {
	createScript := getIndexUpdatedTableCreateScript("test_keyspace", &indexUpdatedTableInfo{name: "index_group_records__index_updated"})

	if !strings.Contains(createScript, "CREATE TABLE test_keyspace.index_group_records__index_updated") {
		t.Fatalf("unexpected index-updated create script: %s", createScript)
	}
	if !strings.Contains(createScript, "partition_id int") {
		t.Fatalf("expected int partition_id column: %s", createScript)
	}
	if !strings.Contains(createScript, "index_id smallint") {
		t.Fatalf("expected smallint index_id column: %s", createScript)
	}
	if !strings.Contains(createScript, "PRIMARY KEY ((partition_id), index_id, index_hash)") {
		t.Fatalf("unexpected index-updated primary key: %s", createScript)
	}
}

func TestMakeUpdateStatementsAllowsIndexGroupUpdateWhenOmittedValuesExistInStruct(t *testing.T) {
	records := []indexGroupRecord{
		{CompanyID: 7, ID: 1, Date: 18754, ClientID: 5, ProductIDs: []int32{11, 17}},
	}
	table := Table[indexGroupRecord, indexGroupSchema]()

	originalCounterFetcher := getWriteCounterValue
	originalManagedUnixTime := getManagedUnixTime
	t.Cleanup(func() {
		getWriteCounterValue = originalCounterFetcher
		getManagedUnixTime = originalManagedUnixTime
	})

	getManagedUnixTime = func() int32 { return 77 }
	getWriteCounterValue = func(keyspace string, name string, increment int) (int64, error) {
		return 41, nil
	}

	preparedStatements := MakeUpdateStatements(&records, table.ClientID)
	if len(preparedStatements) != 1 {
		t.Fatalf("expected one update statement, got %d", len(preparedStatements))
	}
	preparedStatement := preparedStatements[0]

	if !strings.Contains(preparedStatement.Stmt, "client_id = ?") {
		t.Fatalf("expected client_id SET clause in statement: %s", preparedStatement.Stmt)
	}
	if !strings.Contains(preparedStatement.Stmt, "zz_igs_date_client_id_product_ids = ?") {
		t.Fatalf("expected raw+slice index-group virtual column SET in statement: %s", preparedStatement.Stmt)
	}
	if !strings.Contains(preparedStatement.Stmt, "zz_iwks_date_client_id_product_ids = ?") {
		t.Fatalf("expected week+slice index-group virtual column SET in statement: %s", preparedStatement.Stmt)
	}
	if !containsAny(preparedStatement.Args, int32(5)) {
		t.Fatalf("expected client_id bound value 5 in args %v", preparedStatement.Args)
	}
}

func TestMakeUpdateStatementsRejectsIndexGroupUpdateWhenOmittedValuesMissingInStruct(t *testing.T) {
	records := []indexGroupRecord{
		{CompanyID: 7, ID: 1, Date: 0, ClientID: 5, ProductIDs: nil},
	}
	table := Table[indexGroupRecord, indexGroupSchema]()

	originalCounterFetcher := getWriteCounterValue
	originalManagedUnixTime := getManagedUnixTime
	t.Cleanup(func() {
		getWriteCounterValue = originalCounterFetcher
		getManagedUnixTime = originalManagedUnixTime
	})

	getManagedUnixTime = func() int32 { return 77 }
	getWriteCounterValue = func(keyspace string, name string, increment int) (int64, error) {
		return 41, nil
	}

	defer func() {
		recoveredValue := recover()
		if recoveredValue == nil {
			t.Fatal("expected partial IndexGroup update to panic")
		}
		recoveredMessage := recoveredValue.(string)
		if !strings.Contains(recoveredMessage, `IndexGroup "date_client_id_product_ids" needs struct values for omitted source columns`) {
			t.Fatalf("unexpected panic message: %s", recoveredMessage)
		}
	}()

	MakeUpdateStatements(&records, table.ClientID)
}

func TestSyncIndexGroupsAfterWritePersistsDedupedRows(t *testing.T) {
	scyllaTable := MakeScyllaTable[indexGroupRecord, indexGroupSchema]()
	records := []indexGroupRecord{
		{CompanyID: 7, ID: 1, Date: 18754, ClientID: 5, ProductIDs: []int32{11, 11, 17}},
	}

	originalPersistIndexUpdatedRows := persistIndexUpdatedRows
	originalPersistIndexUpdatedRowsAsync := persistIndexUpdatedRowsAsync
	t.Cleanup(func() {
		persistIndexUpdatedRows = originalPersistIndexUpdatedRows
		persistIndexUpdatedRowsAsync = originalPersistIndexUpdatedRowsAsync
	})
	persistIndexUpdatedRowsAsync = false

	var persistedKeyspace string
	var persistedTable string
	var persistedRows []indexUpdatedRow
	persistIndexUpdatedRows = func(keyspace string, tableName string, rows []indexUpdatedRow) error {
		persistedKeyspace = keyspace
		persistedTable = tableName
		persistedRows = append([]indexUpdatedRow{}, rows...)
		return nil
	}

	managedValues := managedWriteValues{
		updateCounterValues: []any{int32(41)},
	}

	if err := syncIndexGroupsAfterWrite(&records, &scyllaTable, managedValues); err != nil {
		t.Fatalf("syncIndexGroupsAfterWrite returned error: %v", err)
	}

	if persistedKeyspace != "test_keyspace" {
		t.Fatalf("unexpected persisted keyspace: %q", persistedKeyspace)
	}
	if persistedTable != "index_group_records__index_updated" {
		t.Fatalf("unexpected persisted table: %q", persistedTable)
	}

	expectedHashes := map[int32]struct{}{}
	for _, indexGroup := range scyllaTable.indexGroups {
		if !shouldPersistIndexUpdatedGroup(indexGroup) {
			continue
		}
		for _, hashValue := range computeIndexGroupHashes(unsafe.Pointer(&records[0]), indexGroup.sourceColumns) {
			expectedHashes[hashValue] = struct{}{}
		}
	}

	if len(persistedRows) != len(expectedHashes) {
		t.Fatalf("expected %d persisted rows, got %d", len(expectedHashes), len(persistedRows))
	}

	seenHashes := map[int32]struct{}{}
	for _, row := range persistedRows {
		if row.partitionID != 7 {
			t.Fatalf("unexpected partition id: %+v", row)
		}
		if row.indexID == 0 {
			t.Fatalf("expected persisted index_id: %+v", row)
		}
		if row.updateCounter != 41 {
			t.Fatalf("unexpected update counter: %+v", row)
		}
		if _, ok := expectedHashes[row.indexHash]; !ok {
			t.Fatalf("unexpected index hash persisted: %+v", row)
		}
		if _, duplicated := seenHashes[row.indexHash]; duplicated {
			t.Fatalf("duplicate index hash persisted: %+v", row)
		}
		seenHashes[row.indexHash] = struct{}{}
	}
}

func TestShouldPersistIndexUpdatedGroupSkipsWeekOnlyHashes(t *testing.T) {
	scyllaTable := MakeScyllaTable[weekIndexGroupRecord, weekIndexGroupSchema]()
	if len(scyllaTable.indexGroups) != 2 {
		t.Fatalf("expected raw and week variants, got %d", len(scyllaTable.indexGroups))
	}

	if !shouldPersistIndexUpdatedGroup(scyllaTable.indexGroups[0]) {
		t.Fatal("expected raw index-group hashes to be persisted")
	}
	if shouldPersistIndexUpdatedGroup(scyllaTable.indexGroups[1]) {
		t.Fatal("expected week-only index-group hashes to be excluded from __index_updated")
	}
	if scyllaTable.indexGroups[0].indexID != scyllaTable.indexGroups[1].indexID {
		t.Fatalf("expected raw and week index groups to reuse the same index_id, got %d and %d",
			scyllaTable.indexGroups[0].indexID, scyllaTable.indexGroups[1].indexID)
	}
}

func TestAllocateIndexGroupIDAvoidsHashCollisions(t *testing.T) {
	scyllaTable := &ScyllaTable{}

	firstID := allocateIndexGroupID(scyllaTable, []string{"date", "client_id"})
	reusedID := allocateIndexGroupID(scyllaTable, []string{"date", "client_id"})
	if firstID != reusedID {
		t.Fatalf("expected the same logical index group to reuse its id, got %d and %d", firstID, reusedID)
	}

	collisionStart := int16(BasicHashInt("new_group"))
	if collisionStart == 0 {
		collisionStart = 1
	}
	scyllaTable.indexGroupIDs = map[int16]string{
		collisionStart:   "occupied_a",
		collisionStart+1: "occupied_b",
	}

	nextID := allocateIndexGroupID(scyllaTable, []string{"new_group"})
	if nextID != collisionStart+2 {
		t.Fatalf("expected collision handling to advance to %d, got %d", collisionStart+2, nextID)
	}
	if scyllaTable.indexGroupIDs[nextID] != "new_group" {
		t.Fatalf("expected allocated id %d to be assigned to new_group", nextID)
	}
}

func TestAppendIndexUpdatedRowsForRecordKeepsMaxUpdateCounterPerHash(t *testing.T) {
	scyllaTable := MakeScyllaTable[indexGroupRecord, indexGroupSchema]()
	record := indexGroupRecord{
		CompanyID:  7,
		ID:         1,
		Date:      18754,
		ClientID:   5,
		ProductIDs: []int32{11, 17},
	}

	rowsByPartitionAndHash := map[string]indexUpdatedRow{}
	rowsToPersist := []indexUpdatedRow{}
	recordPointer := unsafe.Pointer(&record)

	appendIndexUpdatedRowsForRecord(recordPointer, &scyllaTable, 7, 41, rowsByPartitionAndHash, &rowsToPersist)
	appendIndexUpdatedRowsForRecord(recordPointer, &scyllaTable, 7, 55, rowsByPartitionAndHash, &rowsToPersist)
	appendIndexUpdatedRowsForRecord(recordPointer, &scyllaTable, 7, 12, rowsByPartitionAndHash, &rowsToPersist)

	if len(rowsToPersist) == 0 {
		t.Fatal("expected persisted rows")
	}

	for _, row := range rowsToPersist {
		if row.partitionID != 7 {
			t.Fatalf("unexpected partition id: %+v", row)
		}
		if row.updateCounter != 55 {
			t.Fatalf("expected deduped row to keep max update counter 55, got %+v", row)
		}
	}
}

func TestBuildSelectProjectionSkipsWriteOnlyManagedColumns(t *testing.T) {
	scyllaTable := MakeScyllaTable[Increment, IncrementTable]()

	columnNames, _, _ := buildSelectProjection(&TableInfo{}, scyllaTable)
	if slices.Contains(columnNames, managedCreatedColumnName) {
		t.Fatalf("did not expect %q in default projection: %v", managedCreatedColumnName, columnNames)
	}
	if slices.Contains(columnNames, managedUpdatedColumnName) {
		t.Fatalf("did not expect %q in default projection: %v", managedUpdatedColumnName, columnNames)
	}
	if slices.Contains(columnNames, managedUpdateCounterColumnName) {
		t.Fatalf("did not expect %q in default projection: %v", managedUpdateCounterColumnName, columnNames)
	}
	if !slices.Contains(columnNames, "name") || !slices.Contains(columnNames, "current_value") {
		t.Fatalf("expected real sequence columns in projection, got %v", columnNames)
	}
}
