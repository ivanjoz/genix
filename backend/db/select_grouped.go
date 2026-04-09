package db

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

type RecordGroup[T any] struct {
	IndexID          int16   `json:"ig"`
	GroupHash        int32   `json:"id"`
	IndexGroupValues []int64 `json:"igVal"`
	Records          []T     `json:"records"`
	UpdateCounter    int32   `json:"upc"`
}

type queryIndexGroupHash struct {
	hashValue        int32
	indexGroupValues []int64
}

type indexGroupSelectPlan struct {
	partitionValue int32
	indexGroup     indexGroupInfo
	hashGroups     []queryIndexGroupHash
}

type indexGroupFetchState struct {
	hashValue        int32
	indexGroupValues []int64
	updateCounter    int32
}

// loadIndexGroupFreshnessRows is overridable in tests so grouped planning can be verified without Scylla.
var loadIndexGroupFreshnessRows = loadIndexGroupFreshnessRowsFromScylla

func execIndexGroupQuery[T TableSchemaInterface[T], E any](schemaStruct *T, tableInfo *TableInfo) error {
	if len(tableInfo.columnsInclude) > 0 || len(tableInfo.columnsExclude) > 0 {
		return fmt.Errorf("QueryIndexGroup does not support Select() or Exclude() yet")
	}
	if len(tableInfo.groupByColumns) > 0 {
		return fmt.Errorf("QueryIndexGroup does not support GroupBy()")
	}
	if tableInfo.orderBy != "" {
		return fmt.Errorf("QueryIndexGroup does not support OrderDesc() yet")
	}
	if tableInfo.limit > 0 {
		return fmt.Errorf("QueryIndexGroup does not support Limit() yet")
	}
	if tableInfo.allowFilter {
		return fmt.Errorf("QueryIndexGroup does not support AllowFilter()")
	}

	refGroups := tableInfo.refSlice.(*[]RecordGroup[E])
	*refGroups = (*refGroups)[:0]

	scyllaTable := getOrCompileScyllaTable(schemaStruct)
	if len(scyllaTable.keyspace) == 0 {
		scyllaTable.keyspace = connParams.Keyspace
	}
	if len(scyllaTable.keyspace) == 0 {
		return fmt.Errorf("no se ha especificado un keyspace")
	}

	queryPlan, err := buildIndexGroupSelectPlan(tableInfo, scyllaTable)
	if err != nil {
		return err
	}

	hashValues := extractQueryIndexGroupHashes(queryPlan.hashGroups)
	serverFreshnessByHash, err := loadIndexGroupFreshnessRows(scyllaTable, queryPlan.indexGroup, queryPlan.partitionValue, hashValues)
	if err != nil {
		return err
	}

	fetchStates := filterIndexGroupFetches(queryPlan.hashGroups, serverFreshnessByHash, tableInfo.cachedIndexGroups)
	if len(fetchStates) == 0 {
		fmt.Printf("QueryIndexGroup skipped all groups: table=%s partition=%d hashes=%d\n",
			scyllaTable.name, queryPlan.partitionValue, len(hashValues))
		return nil
	}

	fetchedGroups, err := fetchIndexGroupRecordGroups[E](scyllaTable, queryPlan, fetchStates, collectSelectStatements(tableInfo), time.Now())
	if err != nil {
		return err
	}

	*refGroups = append(*refGroups, fetchedGroups...)
	return nil
}

func buildIndexGroupSelectPlan(tableInfo *TableInfo, scyllaTable ScyllaTable[any]) (*indexGroupSelectPlan, error) {
	statements := collectSelectStatements(tableInfo)
	partitionColumn := scyllaTable.GetPartKey()
	if partitionColumn == nil || partitionColumn.IsNil() {
		return nil, fmt.Errorf(`QueryIndexGroup requires a partition column on table "%v"`, scyllaTable.name)
	}

	statementByColumn := map[string][]ColumnStatement{}
	betweenStatementsCount := 0
	for _, statement := range statements {
		statementByColumn[statement.Col] = append(statementByColumn[statement.Col], statement)
		if statement.Operator == "BETWEEN" {
			betweenStatementsCount++
		}
	}

	partitionStatements := statementByColumn[partitionColumn.GetName()]
	if len(partitionStatements) == 0 || partitionStatements[0].Operator != "=" {
		return nil, fmt.Errorf(`QueryIndexGroup requires partition equality on "%v"`, partitionColumn.GetName())
	}
	if betweenStatementsCount != 1 {
		return nil, fmt.Errorf("QueryIndexGroup requires exactly one BETWEEN statement")
	}

	var bestIndexGroup *indexGroupInfo
	bestScore := -1
	for indexGroupIndex := range scyllaTable.indexGroups {
		indexGroup := &scyllaTable.indexGroups[indexGroupIndex]
		score, isValid := scoreIndexGroupCandidate(*indexGroup, statementByColumn)
		if !isValid || score <= bestScore {
			continue
		}
		bestScore = score
		bestIndexGroup = indexGroup
	}

	if bestIndexGroup == nil {
		return nil, fmt.Errorf(`QueryIndexGroup did not find a compatible UseIndexGroup for table "%v"`, scyllaTable.name)
	}

	hashGroups, err := buildQueryIndexGroupHashes(*bestIndexGroup, statements)
	if err != nil {
		return nil, err
	}
	if len(hashGroups) == 0 {
		return nil, fmt.Errorf("QueryIndexGroup did not produce candidate hashes")
	}

	fmt.Printf("QueryIndexGroup plan selected: table=%s group=%s candidate_hashes=%d\n",
		scyllaTable.name, bestIndexGroup.name, len(hashGroups))

	return &indexGroupSelectPlan{
		partitionValue: convertToInt32(partitionStatements[0].Value),
		indexGroup:     *bestIndexGroup,
		hashGroups:     hashGroups,
	}, nil
}

func scoreIndexGroupCandidate(indexGroup indexGroupInfo, statementByColumn map[string][]ColumnStatement) (int, bool) {
	betweenColumns := 0
	fanoutSize := 1

	for _, sourceColumn := range indexGroup.sourceColumns {
		if sourceColumn.weekOnly {
			// TODO: Future version should probe fecha hashes first and only then decide whether week hashes help fetch records.
			return 0, false
		}

		columnStatements := statementByColumn[sourceColumn.column.GetName()]
		if len(columnStatements) != 1 {
			return 0, false
		}

		statement := columnStatements[0]
		switch statement.Operator {
		case "=":
			fanoutSize *= 1
		case "CONTAINS":
			fanoutSize *= 1
		case "IN":
			if len(statement.Values) == 0 {
				return 0, false
			}
			fanoutSize *= len(statement.Values)
		case "BETWEEN":
			if len(statement.From) == 0 || len(statement.To) == 0 {
				return 0, false
			}
			betweenColumns++
			fromValue := convertToInt64(statement.From[0].Value)
			toValue := convertToInt64(statement.To[0].Value)
			if toValue < fromValue {
				fromValue, toValue = toValue, fromValue
			}
			fanoutSize *= int(toValue-fromValue) + 1
		default:
			return 0, false
		}
	}

	if betweenColumns != 1 {
		return 0, false
	}

	// Prefer the most specific group and break ties by lower hash fanout.
	return len(indexGroup.sourceColumns)*100000 - fanoutSize, true
}

func buildQueryIndexGroupHashes(indexGroup indexGroupInfo, statements []ColumnStatement) ([]queryIndexGroupHash, error) {
	valuesGroups := [][]int64{{}}

	for _, sourceColumn := range indexGroup.sourceColumns {
		statement, err := findIndexGroupStatement(sourceColumn.column.GetName(), statements)
		if err != nil {
			return nil, err
		}

		columnValues, err := resolveIndexGroupQueryValues(sourceColumn, statement)
		if err != nil {
			return nil, err
		}
		if len(columnValues) == 0 {
			return nil, nil
		}

		nextGroups := make([][]int64, 0, len(valuesGroups)*len(columnValues))
		for _, valuesGroup := range valuesGroups {
			for _, value := range columnValues {
				nextGroups = append(nextGroups, append(append([]int64{}, valuesGroup...), value))
			}
		}
		valuesGroups = nextGroups
	}

	hashGroupsByValue := map[int32]queryIndexGroupHash{}
	for _, valuesGroup := range valuesGroups {
		hashValue := HashInt64(valuesGroup...)
		if _, exists := hashGroupsByValue[hashValue]; exists {
			continue
		}
		hashGroupsByValue[hashValue] = queryIndexGroupHash{
			hashValue:        hashValue,
			indexGroupValues: append([]int64(nil), valuesGroup...),
		}
	}

	hashValues := make([]int32, 0, len(hashGroupsByValue))
	for hashValue := range hashGroupsByValue {
		hashValues = append(hashValues, hashValue)
	}
	slices.Sort(hashValues)

	hashGroups := make([]queryIndexGroupHash, 0, len(hashValues))
	for _, hashValue := range hashValues {
		hashGroups = append(hashGroups, hashGroupsByValue[hashValue])
	}
	return hashGroups, nil
}

func extractQueryIndexGroupHashes(hashGroups []queryIndexGroupHash) []int32 {
	hashValues := make([]int32, 0, len(hashGroups))
	for _, hashGroup := range hashGroups {
		hashValues = append(hashValues, hashGroup.hashValue)
	}
	return hashValues
}

func findIndexGroupStatement(columnName string, statements []ColumnStatement) (ColumnStatement, error) {
	for _, statement := range statements {
		if statement.Col == columnName {
			return statement, nil
		}
	}
	return ColumnStatement{}, fmt.Errorf(`QueryIndexGroup missing statement for column "%v"`, columnName)
}

func resolveIndexGroupQueryValues(sourceColumn indexGroupSourceColumn, statement ColumnStatement) ([]int64, error) {
	switch statement.Operator {
	case "=":
		return resolveIndexGroupValues(sourceColumn, statement.Value), nil
	case "CONTAINS":
		return resolveIndexGroupValues(sourceColumn, statement.Value), nil
	case "IN":
		columnValues := make([]int64, 0, len(statement.Values))
		for _, rawValue := range statement.Values {
			columnValues = append(columnValues, resolveIndexGroupValues(sourceColumn, rawValue)...)
		}
		return columnValues, nil
	case "BETWEEN":
		if len(statement.From) == 0 || len(statement.To) == 0 {
			return nil, fmt.Errorf(`QueryIndexGroup received an invalid BETWEEN for "%v"`, statement.Col)
		}
		if sourceColumn.weekOnly {
			return nil, fmt.Errorf(`QueryIndexGroup TODO: week-only probing is not implemented for "%v"`, statement.Col)
		}

		fromValue := convertToInt64(statement.From[0].Value)
		toValue := convertToInt64(statement.To[0].Value)
		if toValue < fromValue {
			fromValue, toValue = toValue, fromValue
		}
		columnValues := make([]int64, 0, int(toValue-fromValue)+1)
		for currentValue := fromValue; currentValue <= toValue; currentValue++ {
			columnValues = append(columnValues, currentValue)
		}
		return columnValues, nil
	default:
		return nil, fmt.Errorf(`QueryIndexGroup does not support operator "%v" on "%v"`, statement.Operator, statement.Col)
	}
}

func loadIndexGroupFreshnessRowsFromScylla(
	scyllaTable ScyllaTable[any],
	indexGroup indexGroupInfo,
	partitionValue int32,
	hashValues []int32,
) (map[int32]int32, error) {
	if len(hashValues) == 0 {
		return map[int32]int32{}, nil
	}
	if scyllaTable.indexUpdatedTable == nil {
		return nil, fmt.Errorf(`QueryIndexGroup requires "__index_updated" metadata for table "%v"`, scyllaTable.name)
	}

	freshnessByHash := map[int32]int32{}
	maxHashesPerQuery := getMaxClusteringKeyRestrictionsPerQuery()
	if maxHashesPerQuery <= 0 {
		maxHashesPerQuery = 100
	}

	for startIndex := 0; startIndex < len(hashValues); startIndex += maxHashesPerQuery {
		endIndex := startIndex + maxHashesPerQuery
		if endIndex > len(hashValues) {
			endIndex = len(hashValues)
		}

		hashChunk := hashValues[startIndex:endIndex]
		placeholders := make([]string, 0, len(hashChunk))
		queryValues := make([]any, 0, len(hashChunk)+2)
		queryValues = append(queryValues, partitionValue)
		queryValues = append(queryValues, indexGroup.indexID)
		for _, hashValue := range hashChunk {
			placeholders = append(placeholders, "?")
			queryValues = append(queryValues, hashValue)
		}

		queryStr := fmt.Sprintf(
			`SELECT index_hash, update_counter FROM %v.%v WHERE partition_id = ? AND index_id = ? AND index_hash IN (%v)`,
			scyllaTable.keyspace,
			scyllaTable.indexUpdatedTable.name,
			strings.Join(placeholders, ", "),
		)

		fmt.Printf("QueryIndexGroup freshness probe: table=%s partition=%d index_id=%d hashes=%d\n",
			scyllaTable.name, partitionValue, indexGroup.indexID, len(hashChunk))

		iter := getScyllaConnection().Query(queryStr, queryValues...).Iter()
		var hashValue int32
		var updateCounter int32
		for iter.Scan(&hashValue, &updateCounter) {
			freshnessByHash[hashValue] = updateCounter
		}
		if err := iter.Close(); err != nil {
			return nil, err
		}
	}

	return freshnessByHash, nil
}

func filterIndexGroupFetches(
	hashGroups []queryIndexGroupHash,
	serverFreshnessByHash map[int32]int32,
	cachedIndexGroups map[int32]int32,
) []indexGroupFetchState {
	fetchStates := make([]indexGroupFetchState, 0, len(hashGroups))
	for _, hashGroup := range hashGroups {
		updateCounter, existsOnServer := serverFreshnessByHash[hashGroup.hashValue]
		if !existsOnServer {
			continue
		}
		if cachedIndexGroups != nil {
			if cachedCounter, existsInCache := cachedIndexGroups[hashGroup.hashValue]; existsInCache && cachedCounter == updateCounter {
				continue
			}
		}
		fetchStates = append(fetchStates, indexGroupFetchState{
			hashValue:        hashGroup.hashValue,
			indexGroupValues: append([]int64(nil), hashGroup.indexGroupValues...),
			updateCounter:    updateCounter,
		})
	}
	return fetchStates
}

func fetchIndexGroupRecordGroups[E any](
	scyllaTable ScyllaTable[any],
	queryPlan *indexGroupSelectPlan,
	fetchStates []indexGroupFetchState,
	postFilterStatements []ColumnStatement,
	queryNoticeTime time.Time,
) ([]RecordGroup[E], error) {
	_, scanColumns, selectExpressions := buildSelectProjection(&TableInfo{}, scyllaTable)
	groupsByIndex := make([]RecordGroup[E], len(fetchStates))

	eg := errgroup.Group{}
	for fetchIndex, fetchState := range fetchStates {
		fetchIndex := fetchIndex
		fetchState := fetchState

		eg.Go(func() error {
			queryStr, queryValues := buildIndexGroupFetchQuery(scyllaTable, queryPlan, fetchState, selectExpressions)
			records := []E{}
			if err := scanSelectQueryRows(
				queryStr,
				queryValues,
				scanColumns,
				scyllaTable,
				&records,
				postFilterStatements,
				nil,
				queryNoticeTime,
			); err != nil {
				return err
			}

			groupsByIndex[fetchIndex] = RecordGroup[E]{
				IndexID:          queryPlan.indexGroup.indexID,
				GroupHash:        fetchState.hashValue,
				IndexGroupValues: append([]int64(nil), fetchState.indexGroupValues...),
				Records:          records,
				UpdateCounter:    fetchState.updateCounter,
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return groupsByIndex, nil
}

func buildIndexGroupFetchQuery(
	scyllaTable ScyllaTable[any],
	queryPlan *indexGroupSelectPlan,
	fetchState indexGroupFetchState,
	selectExpressions []string,
) (string, []any) {
	whereStatements := []string{
		fmt.Sprintf("%v = ?", scyllaTable.GetPartKey().GetName()),
	}
	queryValues := []any{queryPlan.partitionValue}

	// Single-column groups already have a real secondary index on the source column,
	// so grouped fetches can probe that column directly without a virtual hash column.
	if queryPlan.indexGroup.virtualColumn == nil || queryPlan.indexGroup.virtualColumn.IsNil() {
		sourceColumn := queryPlan.indexGroup.sourceColumns[0].column
		whereStatements = append(whereStatements, fmt.Sprintf("%v = ?", sourceColumn.GetName()))
		queryValues = append(queryValues, fetchState.indexGroupValues[0])
	} else if queryPlan.indexGroup.usesCollectionValues {
		whereStatements = append(whereStatements, fmt.Sprintf("%v CONTAINS ?", queryPlan.indexGroup.virtualColumn.GetName()))
		queryValues = append(queryValues, fetchState.hashValue)
	} else {
		whereStatements = append(whereStatements, fmt.Sprintf("%v = ?", queryPlan.indexGroup.virtualColumn.GetName()))
		queryValues = append(queryValues, fetchState.hashValue)
	}

	queryStr := fmt.Sprintf(
		"SELECT %v FROM %v.%v WHERE %v",
		strings.Join(selectExpressions, ", "),
		scyllaTable.keyspace,
		scyllaTable.name,
		strings.Join(whereStatements, " AND "),
	)
	return queryStr, queryValues
}
