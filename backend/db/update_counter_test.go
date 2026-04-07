package db

import (
	"slices"
	"strings"
	"testing"
	"unsafe"
)

type updateCounterRecord struct {
	TableStruct[updateCounterSchema, updateCounterRecord]
	EmpresaID int32  `db:"empresa_id"`
	ID        int64  `db:"id"`
	Nombre    string `db:"nombre"`
}

type updateCounterSchema struct {
	TableStruct[updateCounterSchema, updateCounterRecord]
	EmpresaID Col[updateCounterSchema, int32]
	ID        Col[updateCounterSchema, int64]
	Nombre    Col[updateCounterSchema, string]
}

type updateCounterDisabledRecord struct {
	TableStruct[updateCounterDisabledSchema, updateCounterDisabledRecord]
	EmpresaID int32  `db:"empresa_id"`
	ID        int64  `db:"id"`
	Nombre    string `db:"nombre"`
}

type updateCounterDisabledSchema struct {
	TableStruct[updateCounterDisabledSchema, updateCounterDisabledRecord]
	EmpresaID Col[updateCounterDisabledSchema, int32]
	ID        Col[updateCounterDisabledSchema, int64]
	Nombre    Col[updateCounterDisabledSchema, string]
}

type indexGroupRecord struct {
	TableStruct[indexGroupSchema, indexGroupRecord]
	EmpresaID  int32   `db:"empresa_id"`
	ID         int64   `db:"id"`
	Fecha      int16   `db:"fecha"`
	ClientID   int32   `db:"client_id"`
	ProductIDs []int32 `db:"product_ids,list"`
}

type indexGroupSchema struct {
	TableStruct[indexGroupSchema, indexGroupRecord]
	EmpresaID  Col[indexGroupSchema, int32]
	ID         Col[indexGroupSchema, int64]
	Fecha      Col[indexGroupSchema, int16]
	ClientID   Col[indexGroupSchema, int32]
	ProductIDs Col[indexGroupSchema, []int32]
}

type weekIndexGroupRecord struct {
	TableStruct[weekIndexGroupSchema, weekIndexGroupRecord]
	EmpresaID int32 `db:"empresa_id"`
	ID        int64 `db:"id"`
	Fecha     int16 `db:"fecha"`
	Status    int8  `db:"status"`
}

type weekIndexGroupSchema struct {
	TableStruct[weekIndexGroupSchema, weekIndexGroupRecord]
	EmpresaID Col[weekIndexGroupSchema, int32]
	ID        Col[weekIndexGroupSchema, int64]
	Fecha     Col[weekIndexGroupSchema, int16]
	Status    Col[weekIndexGroupSchema, int8]
}

func (e updateCounterSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:  "test_keyspace",
		Name:      "update_counter_records",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
	}
}

func (e updateCounterDisabledSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:             "test_keyspace",
		Name:                 "update_counter_records_disabled",
		Partition:            e.EmpresaID,
		Keys:                 []Coln{e.ID},
		DisableUpdateCounter: true,
	}
}

func (e indexGroupSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:  "test_keyspace",
		Name:      "index_group_records",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
		Indexes: []Index{
			{
				Keys:          []Coln{e.Fecha},
				UseIndexGroup: true,
			},
			{
				Keys:          []Coln{e.Fecha.StoreAsWeek(), e.ClientID, e.ProductIDs},
				UseIndexGroup: true,
			},
		},
	}
}

func (e weekIndexGroupSchema) GetSchema() TableSchema {
	return TableSchema{
		Keyspace:  "test_keyspace",
		Name:      "week_index_group_records",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
		Indexes: []Index{
			{
				Keys:          []Coln{e.Fecha.StoreAsWeek(), e.Status},
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
		{EmpresaID: 7, ID: 1, Nombre: "uno"},
		{EmpresaID: 7, ID: 2, Nombre: "dos"},
		{EmpresaID: 8, ID: 3, Nombre: "tres"},
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
		{EmpresaID: 7, ID: 1, Nombre: "nuevo"},
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

	updateStatements := MakeUpdateStatements(&records, table.Nombre)
	if len(updateStatements) != 1 {
		t.Fatalf("expected one update statement, got %d", len(updateStatements))
	}

	if !strings.Contains(updateStatements[0], "nombre = 'nuevo'") {
		t.Fatalf("expected explicit column update in statement: %s", updateStatements[0])
	}
	if !strings.Contains(updateStatements[0], "updated = 77") {
		t.Fatalf("expected managed updated column in statement: %s", updateStatements[0])
	}
	if !strings.Contains(updateStatements[0], "update_counter = 41") {
		t.Fatalf("expected managed update counter in statement: %s", updateStatements[0])
	}
}

func TestMakeInsertStatementIncludesManagedAuditColumnsWithoutStructFields(t *testing.T) {
	records := []updateCounterRecord{
		{EmpresaID: 7, ID: 1, Nombre: "nuevo"},
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

	for _, expectedColumn := range []string{"created", "updated", "update_counter"} {
		if !strings.Contains(insertStatements[0], expectedColumn) {
			t.Fatalf("expected managed column %q in insert statement: %s", expectedColumn, insertStatements[0])
		}
	}

	for _, expectedValue := range []string{"77", "41"} {
		if !strings.Contains(insertStatements[0], expectedValue) {
			t.Fatalf("expected managed value %q in insert statement: %s", expectedValue, insertStatements[0])
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
	if !strings.Contains(createScript, "PRIMARY KEY ((partition_id), update_counter, index_hash)") {
		t.Fatalf("unexpected index-updated primary key: %s", createScript)
	}
}

func TestMakeUpdateStatementsAllowsIndexGroupUpdateWhenOmittedValuesExistInStruct(t *testing.T) {
	records := []indexGroupRecord{
		{EmpresaID: 7, ID: 1, Fecha: 18754, ClientID: 5, ProductIDs: []int32{11, 17}},
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

	updateStatements := MakeUpdateStatements(&records, table.ClientID)
	if len(updateStatements) != 1 {
		t.Fatalf("expected one update statement, got %d", len(updateStatements))
	}
	if !strings.Contains(updateStatements[0], "client_id = 5") {
		t.Fatalf("expected client_id update in statement: %s", updateStatements[0])
	}
	if !strings.Contains(updateStatements[0], "zz_igs_fecha_client_id_product_ids") {
		t.Fatalf("expected raw+slice index-group virtual column update in statement: %s", updateStatements[0])
	}
	if !strings.Contains(updateStatements[0], "zz_iwks_fecha_client_id_product_ids") {
		t.Fatalf("expected week+slice index-group virtual column update in statement: %s", updateStatements[0])
	}
}

func TestMakeUpdateStatementsRejectsIndexGroupUpdateWhenOmittedValuesMissingInStruct(t *testing.T) {
	records := []indexGroupRecord{
		{EmpresaID: 7, ID: 1, Fecha: 0, ClientID: 5, ProductIDs: nil},
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
		if !strings.Contains(recoveredMessage, `IndexGroup "fecha_client_id_product_ids" needs struct values for omitted source columns`) {
			t.Fatalf("unexpected panic message: %s", recoveredMessage)
		}
	}()

	MakeUpdateStatements(&records, table.ClientID)
}

func TestSyncIndexGroupsAfterWritePersistsDedupedRows(t *testing.T) {
	scyllaTable := MakeScyllaTable[indexGroupRecord, indexGroupSchema]()
	records := []indexGroupRecord{
		{EmpresaID: 7, ID: 1, Fecha: 18754, ClientID: 5, ProductIDs: []int32{11, 11, 17}},
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

func TestStoreAsWeekIndexGroupUsesCollectionHashes(t *testing.T) {
	scyllaTable := MakeScyllaTable[weekIndexGroupRecord, weekIndexGroupSchema]()
	rawVirtualColumn := scyllaTable.columnsMap["zz_ig_fecha_status"]
	if rawVirtualColumn == nil {
		t.Fatal("expected raw-date index-group virtual column to exist")
	}
	if rawVirtualColumn.GetType().ColType != "int" {
		t.Fatalf("expected raw-date index-group virtual column to be int, got %s", rawVirtualColumn.GetType().ColType)
	}

	weekVirtualColumn := scyllaTable.columnsMap["zz_iwk_fecha_status"]
	if weekVirtualColumn == nil {
		t.Fatal("expected week-backed index-group virtual column to exist")
	}
	if weekVirtualColumn.GetType().ColType != "int" {
		t.Fatalf("expected week-backed index-group virtual column to be int, got %s", weekVirtualColumn.GetType().ColType)
	}

	record := weekIndexGroupRecord{
		EmpresaID: 7,
		ID:        1,
		Fecha:     18754,
		Status:    3,
	}

	rawHashes := computeIndexGroupHashes(unsafe.Pointer(&record), scyllaTable.indexGroups[0].sourceColumns)
	weekHashes := computeIndexGroupHashes(unsafe.Pointer(&record), scyllaTable.indexGroups[1].sourceColumns)
	if len(rawHashes) != 1 {
		t.Fatalf("expected one raw-date hash, got %v", rawHashes)
	}
	if len(weekHashes) != 1 {
		t.Fatalf("expected one week-code hash, got %v", weekHashes)
	}
	if rawHashes[0] == weekHashes[0] {
		t.Fatalf("expected distinct raw-date and week-code hashes, got raw=%v week=%v", rawHashes, weekHashes)
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
