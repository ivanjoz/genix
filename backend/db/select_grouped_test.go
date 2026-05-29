package db

import "testing"

func TestQueryIndexGroupStoresCachedGroupsInTableInfo(t *testing.T) {
	groupedRecords := []RecordGroup[indexGroupRecord]{}
	query := QueryIndexGroup[indexGroupRecord, indexGroupSchema](&groupedRecords)

	query.IncludeCachedGroup(101, 33)

	tableInfo := any(query).(interface{ GetTableInfo() *TableInfo }).GetTableInfo()
	if !tableInfo.useIndexGroupSelect {
		t.Fatal("expected QueryIndexGroup to enable grouped execution")
	}
	if cachedCounter := tableInfo.cachedIndexGroups[101]; cachedCounter != 33 {
		t.Fatalf("expected cached group counter 33, got %d", cachedCounter)
	}
}

func TestBuildIndexGroupSelectPlanAllowsSingleColumnRawGroup(t *testing.T) {
	scyllaTable := MakeScyllaTable[indexGroupRecord, indexGroupSchema]()
	tableInfo := &TableInfo{
		statements: []ColumnStatement{
			{Col: "empresa_id", Operator: "=", Value: int32(7)},
			{Col: "date", Operator: "BETWEEN", From: []ColumnStatement{{Col: "date", Value: int16(18754)}}, To: []ColumnStatement{{Col: "date", Value: int16(18756)}}},
		},
	}

	queryPlan, err := buildIndexGroupSelectPlan(tableInfo, scyllaTable)
	if err != nil {
		t.Fatalf("buildIndexGroupSelectPlan returned error: %v", err)
	}

	if queryPlan.indexGroup.name != "date" {
		t.Fatalf("expected single-column index group, got %q", queryPlan.indexGroup.name)
	}
	if queryPlan.indexGroup.virtualColumn != nil && !queryPlan.indexGroup.virtualColumn.IsNil() {
		t.Fatalf("single-column index group should not require a virtual column, got %s", queryPlan.indexGroup.virtualColumn.GetName())
	}
	if len(queryPlan.hashGroups) != 3 {
		t.Fatalf("expected 3 hash groups, got %+v", queryPlan.hashGroups)
	}
	expectedFechas := map[int64]bool{
		18754: false,
		18755: false,
		18756: false,
	}
	for _, hashGroup := range queryPlan.hashGroups {
		if len(hashGroup.indexGroupValues) != 1 {
			t.Fatalf("unexpected single-column values: %+v", hashGroup)
		}
		dateValue := hashGroup.indexGroupValues[0]
		if _, exists := expectedFechas[dateValue]; !exists {
			t.Fatalf("unexpected single-column value: %+v", hashGroup)
		}
		expectedFechas[dateValue] = true
	}
	for dateValue, wasSeen := range expectedFechas {
		if !wasSeen {
			t.Fatalf("missing expected date value %d in %+v", dateValue, queryPlan.hashGroups)
		}
	}
}

func TestBuildIndexGroupFetchQueryUsesSourceColumnForSingleColumnGroup(t *testing.T) {
	scyllaTable := MakeScyllaTable[indexGroupRecord, indexGroupSchema]()
	queryPlan := &indexGroupSelectPlan{
		partitionValue: 7,
		indexGroup:     scyllaTable.indexGroups[0],
	}
	fetchState := indexGroupFetchState{
		hashValue:        HashInt64(18754),
		indexGroupValues: []int64{18754},
	}

	queryStr, queryValues := buildIndexGroupFetchQuery(scyllaTable, queryPlan, fetchState, []string{"*"})
	expectedQuery := "SELECT * FROM test_keyspace.index_group_records WHERE empresa_id = ? AND date = ?"
	if queryStr != expectedQuery {
		t.Fatalf("unexpected query: %s", queryStr)
	}
	if len(queryValues) != 2 || queryValues[0] != int32(7) || queryValues[1] != int64(18754) {
		t.Fatalf("unexpected query values: %+v", queryValues)
	}
}

func TestFilterIndexGroupFetchesSkipsMissingAndFreshHashes(t *testing.T) {
	hashGroups := []queryIndexGroupHash{
		{hashValue: 11, indexGroupValues: []int64{1, 11}},
		{hashValue: 12, indexGroupValues: []int64{2, 12}},
		{hashValue: 13, indexGroupValues: []int64{3, 13}},
		{hashValue: 14, indexGroupValues: []int64{4, 14}},
	}
	serverFreshnessByHash := map[int32]int32{
		11: 100,
		12: 200,
		14: 400,
	}
	cachedIndexGroups := map[int32]int32{
		11: 100,
		12: 150,
		13: 999,
	}

	fetchStates := filterIndexGroupFetches(hashGroups, serverFreshnessByHash, cachedIndexGroups)
	if len(fetchStates) != 2 {
		t.Fatalf("expected two changed hashes, got %+v", fetchStates)
	}
	if fetchStates[0].hashValue != 12 || fetchStates[0].updateCounter != 200 {
		t.Fatalf("unexpected first fetch state: %+v", fetchStates[0])
	}
	if len(fetchStates[0].indexGroupValues) != 2 || fetchStates[0].indexGroupValues[0] != 2 || fetchStates[0].indexGroupValues[1] != 12 {
		t.Fatalf("unexpected first fetch state values: %+v", fetchStates[0])
	}
	if fetchStates[1].hashValue != 14 || fetchStates[1].updateCounter != 400 {
		t.Fatalf("unexpected second fetch state: %+v", fetchStates[1])
	}
	if len(fetchStates[1].indexGroupValues) != 2 || fetchStates[1].indexGroupValues[0] != 4 || fetchStates[1].indexGroupValues[1] != 14 {
		t.Fatalf("unexpected second fetch state values: %+v", fetchStates[1])
	}
}

func TestSplitIndexGroupFetchesReturnsCachedMetadataStates(t *testing.T) {
	hashGroups := []queryIndexGroupHash{
		{hashValue: 11, indexGroupValues: []int64{1, 11}},
		{hashValue: 12, indexGroupValues: []int64{2, 12}},
	}
	serverFreshnessByHash := map[int32]int32{
		11: 100,
		12: 200,
	}
	cachedIndexGroups := map[int32]int32{
		11: 100,
		12: 150,
	}

	fetchStates, cachedStates := splitIndexGroupFetches(hashGroups, serverFreshnessByHash, cachedIndexGroups)
	if len(fetchStates) != 1 || fetchStates[0].hashValue != 12 || fetchStates[0].updateCounter != 200 {
		t.Fatalf("unexpected fetch states: %+v", fetchStates)
	}
	if len(cachedStates) != 1 || cachedStates[0].hashValue != 11 || cachedStates[0].updateCounter != 100 {
		t.Fatalf("unexpected cached states: %+v", cachedStates)
	}
	if len(cachedStates[0].indexGroupValues) != 2 || cachedStates[0].indexGroupValues[0] != 1 || cachedStates[0].indexGroupValues[1] != 11 {
		t.Fatalf("unexpected cached state values: %+v", cachedStates[0])
	}
}
