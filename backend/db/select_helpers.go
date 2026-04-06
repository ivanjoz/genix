package db

import (
	"fmt"
	"hash/fnv"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type selectExecutionRoute int8

const (
	selectRouteAllStatements selectExecutionRoute = iota
	selectRouteViewStatements
	selectRouteKeyConcatenated
	selectRouteKeyIntPacking
	selectRouteCompositeBucket
	selectRouteNativeGroupBy
)

type selectPlanCache struct {
	mutex sync.RWMutex
	plans map[uint64]*SelectStatement
}

type BoundSelectStatement struct {
	QueryStr             string
	QueryValues          []any
	PostFilterStatements []ColumnStatement
}

type BoundSelectPlan struct {
	Statements            []BoundSelectStatement
	ScanColumns           []selectScanColumn
	RequiresDeduplication bool
	AssignCacheVersions   bool
}

// SelectStatement caches one compiled select shape so repeated queries skip capability matching and source selection.
type SelectStatement struct {
	hash uint64
	// queryTemplate already includes SELECT + FROM and expects the final WHERE/GROUP/ORDER/LIMIT suffix.
	queryTemplate string
	scanColumns   []selectScanColumn
	route         selectExecutionRoute
	sourceView    *viewInfo
	// selectedStatementIndexes tracks the predicates consumed by a source view so bind-time can reuse current values.
	selectedStatementIndexes   []int
	remainingStatementIndexes  []int
	postFilterStatementIndexes []int
	requiresDeduplication      bool
	assignCacheVersions        bool
	orderBy                    string
	orderColumnName            string
	limit                      int32
	allowFilter                bool
	groupByColumns             []string
}

func newSelectPlanCache() *selectPlanCache {
	return &selectPlanCache{plans: map[uint64]*SelectStatement{}}
}

func (e *selectPlanCache) Load(hash uint64) (*SelectStatement, bool) {
	if e == nil {
		return nil, false
	}

	e.mutex.RLock()
	defer e.mutex.RUnlock()

	plan, cacheHit := e.plans[hash]
	return plan, cacheHit
}

func (e *selectPlanCache) Store(hash uint64, plan *SelectStatement) {
	if e == nil || plan == nil {
		return
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.plans[hash] = plan
}

func collectSelectStatements(tableInfo *TableInfo) []ColumnStatement {
	// Keep statement extraction centralized so compile and bind share the exact same logical input order.
	statements := make([]ColumnStatement, 0, len(tableInfo.statements)+1)
	statements = append(statements, tableInfo.statements...)
	if len(tableInfo.between.From) > 0 {
		statements = append(statements, tableInfo.between)
	}
	return statements
}

func buildSelectProjection(tableInfo *TableInfo, scyllaTable ScyllaTable[any]) ([]string, []selectScanColumn, []string) {
	columnNames := []string{}
	scanColumns := []selectScanColumn{}
	selectExpressions := []string{}

	if len(tableInfo.columnsInclude) > 0 {
		for _, col := range tableInfo.columnsInclude {
			columnNames = append(columnNames, col.GetName())
		}
		scanColumns = buildDefaultScanColumns(columnNames)
		selectExpressions = append(selectExpressions, columnNames...)
		return columnNames, scanColumns, selectExpressions
	}

	if len(tableInfo.columnsExclude) > 0 {
		excludedColumns := make([]string, 0, len(tableInfo.columnsExclude))
		for _, col := range tableInfo.columnsExclude {
			excludedColumns = append(excludedColumns, col.GetName())
		}
		for _, col := range scyllaTable.columns {
			if slices.Contains(excludedColumns, col.GetName()) || col.GetInfo().IsVirtual {
				continue
			}
			columnNames = append(columnNames, col.GetName())
		}
	} else {
		for _, col := range scyllaTable.columns {
			if col.GetInfo().IsVirtual {
				continue
			}
			columnNames = append(columnNames, col.GetName())
		}
	}

	columnNames = ensureCacheVersionColumnsForSelect(columnNames, scyllaTable)
	scanColumns = buildDefaultScanColumns(columnNames)
	selectExpressions = append(selectExpressions, columnNames...)
	return columnNames, scanColumns, selectExpressions
}

func computeSelectShapeHash(tableInfo *TableInfo, scyllaTable ScyllaTable[any]) uint64 {
	// Hash only the query shape so the same logical select can reuse the compiled plan for different values.
	hashBuilder := fnv.New64a()

	writeText := func(value string) {
		_, _ = hashBuilder.Write([]byte(value))
		_, _ = hashBuilder.Write([]byte{0})
	}

	writeText(scyllaTable.name)

	switch {
	case len(tableInfo.columnsInclude) > 0:
		writeText("projection:include")
		for _, col := range tableInfo.columnsInclude {
			writeText(col.GetName())
		}
	case len(tableInfo.columnsExclude) > 0:
		writeText("projection:exclude")
		excludedNames := make([]string, 0, len(tableInfo.columnsExclude))
		for _, col := range tableInfo.columnsExclude {
			excludedNames = append(excludedNames, col.GetName())
		}
		slices.Sort(excludedNames)
		for _, colName := range excludedNames {
			writeText(colName)
		}
	default:
		writeText("projection:all")
	}

	if len(tableInfo.groupByColumns) > 0 {
		writeText("group-by")
		for _, col := range tableInfo.groupByColumns {
			writeText(col.GetName())
			writeText(col.aggregateFn)
		}
	}

	for _, statement := range collectSelectStatements(tableInfo) {
		writeText(statement.Col)
		writeText(capabilityOpForStatement(statement))

		switch statement.Operator {
		case "IN":
			writeText(fmt.Sprintf("values:%d", len(statement.Values)))
		case "BETWEEN":
			writeText(fmt.Sprintf("between:%d", len(statement.From)))
		default:
			writeText("values:1")
		}
	}

	writeText(tableInfo.orderBy)
	writeText(fmt.Sprintf("limit:%d", tableInfo.limit))
	if tableInfo.allowFilter {
		writeText("allow-filter")
	}

	return hashBuilder.Sum64()
}

func pickStatementsByIndexes(statements []ColumnStatement, indexes []int) []ColumnStatement {
	pickedStatements := make([]ColumnStatement, 0, len(indexes))
	for _, statementIndex := range indexes {
		if statementIndex < 0 || statementIndex >= len(statements) {
			continue
		}
		pickedStatements = append(pickedStatements, statements[statementIndex])
	}
	return pickedStatements
}

func makeSelectQueryTemplate(selectExpressions []string, keyspace, sourceTableName string) string {
	return fmt.Sprintf("SELECT %v FROM %v.%v %%v", strings.Join(selectExpressions, ", "), keyspace, sourceTableName)
}

func getMaxClusteringKeyRestrictionsPerQuery() int {
	// Keep the Scylla clustering-key fanout limit configurable while staying safe by default.
	rawMaxClusteringKeys := strings.TrimSpace(os.Getenv("MAX_CLUSTERING_KEY"))
	if rawMaxClusteringKeys == "" {
		return 100
	}

	maxClusteringKeys, parseError := strconv.Atoi(rawMaxClusteringKeys)
	if parseError != nil || maxClusteringKeys <= 0 {
		fmt.Printf("Invalid MAX_CLUSTERING_KEY=%q. Using default 100.\n", rawMaxClusteringKeys)
		return 100
	}
	return maxClusteringKeys
}

func chunkStatementInValues(values []any, chunkSize int) [][]any {
	if len(values) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 1
	}

	valueChunks := make([][]any, 0, (len(values)+chunkSize-1)/chunkSize)
	for startIndex := 0; startIndex < len(values); startIndex += chunkSize {
		endIndex := min(startIndex+chunkSize, len(values))
		valueChunks = append(valueChunks, slices.Clone(values[startIndex:endIndex]))
	}
	return valueChunks
}

func buildRemainingWhereClauseBatches(remainingStatements []ColumnStatement) []string {
	if len(remainingStatements) == 0 {
		return []string{""}
	}

	maxClusteringKeys := getMaxClusteringKeyRestrictionsPerQuery()
	statementBatches := [][]ColumnStatement{{}}
	currentCartesianProductSize := 1

	for _, statement := range remainingStatements {
		if statement.Operator != "IN" || len(statement.Values) == 0 {
			for batchIndex := range statementBatches {
				statementBatches[batchIndex] = append(statementBatches[batchIndex], statement)
			}
			continue
		}

		maxValuesPerBatch := maxClusteringKeys / currentCartesianProductSize
		if maxValuesPerBatch < 1 {
			maxValuesPerBatch = 1
		}

		valueChunks := chunkStatementInValues(statement.Values, maxValuesPerBatch)
		nextStatementBatches := make([][]ColumnStatement, 0, len(statementBatches)*len(valueChunks))
		maxChunkSize := 0

		for _, valueChunk := range valueChunks {
			if len(valueChunk) > maxChunkSize {
				maxChunkSize = len(valueChunk)
			}
			for _, currentBatchStatements := range statementBatches {
				statementBatch := slices.Clone(currentBatchStatements)
				statementCopy := statement
				statementCopy.Values = slices.Clone(valueChunk)
				statementBatch = append(statementBatch, statementCopy)
				nextStatementBatches = append(nextStatementBatches, statementBatch)
			}
		}

		statementBatches = nextStatementBatches
		currentCartesianProductSize *= max(1, maxChunkSize)
	}

	whereClauseBatches := make([]string, 0, len(statementBatches))
	for _, statementBatch := range statementBatches {
		whereClauseBatches = append(whereClauseBatches, strings.Join(buildRemainingWhereClauses(statementBatch), " AND "))
	}

	if len(whereClauseBatches) > 1 {
		fmt.Printf("Select batching IN query: statements=%d max_clustering_key=%d batches=%d\n",
			len(remainingStatements), maxClusteringKeys, len(whereClauseBatches))
	}

	return whereClauseBatches
}

func buildBoundSelectPlan(
	queryTemplate string,
	scanColumns []selectScanColumn,
	requiresDeduplication bool,
	assignCacheVersions bool,
	whereStatements []string,
	remainingStatements []ColumnStatement,
	postFilterStatements []ColumnStatement,
	groupByColumns []string,
	orderBy string,
	orderColumnName string,
	limit int32,
	allowFilter bool,
) *BoundSelectPlan {
	if len(whereStatements) == 0 {
		whereStatements = []string{""}
	}

	remainingWhereClauseBatches := buildRemainingWhereClauseBatches(remainingStatements)
	boundStatements := make([]BoundSelectStatement, 0, max(1, len(whereStatements)*len(remainingWhereClauseBatches)))

	for _, whereStatement := range whereStatements {
		for _, remainingWhereClause := range remainingWhereClauseBatches {
			whereStatementCombined := whereStatement
			whereRemainClause := remainingWhereClause
			if remainingWhereClause != "" {
				if whereStatementCombined != "" {
					whereRemainClause = " AND " + whereRemainClause
				}
				whereStatementCombined += whereRemainClause
			}
			if whereStatementCombined != "" {
				whereStatementCombined = " WHERE " + whereStatementCombined
			}
			if len(groupByColumns) > 0 {
				whereStatementCombined += " GROUP BY " + strings.Join(groupByColumns, ", ")
			}
			if orderBy != "" {
				whereStatementCombined += " " + fmt.Sprintf(orderBy, orderColumnName)
			}
			if limit > 0 {
				whereStatementCombined += fmt.Sprintf(" LIMIT %v", limit)
			}
			if allowFilter {
				whereStatementCombined += " ALLOW FILTERING"
			}

			boundStatements = append(boundStatements, BoundSelectStatement{
				QueryStr:             fmt.Sprintf(queryTemplate, whereStatementCombined),
				QueryValues:          nil,
				PostFilterStatements: slices.Clone(postFilterStatements),
			})
		}
	}

	return &BoundSelectPlan{
		Statements:            boundStatements,
		ScanColumns:           slices.Clone(scanColumns),
		RequiresDeduplication: requiresDeduplication,
		AssignCacheVersions:   assignCacheVersions,
	}
}

func buildRemainingStatementsForCompositePlan(statements []ColumnStatement, compositePlan *compositeBucketQueryPlan) []ColumnStatement {
	remainingStatements := []ColumnStatement{}
	for _, statement := range statements {
		if !compositePlan.handledColumns[statement.Col] {
			remainingStatements = append(remainingStatements, statement)
		}
	}
	return remainingStatements
}

func canRewriteKeyConcatenated(statements []ColumnStatement, scyllaTable ScyllaTable[any]) bool {
	_, canRewrite := buildKeyConcatenatedStatements(statements, scyllaTable)
	return canRewrite
}

func buildKeyConcatenatedStatements(statements []ColumnStatement, scyllaTable ScyllaTable[any]) ([]ColumnStatement, bool) {
	// Convert equality/range prefixes on smart concatenated keys into one physical PK predicate plus residual filters.
	keyCol := scyllaTable.keys[0]
	hasKeyColQuery := false
	for _, st := range statements {
		if st.Col == keyCol.GetName() {
			hasKeyColQuery = true
			break
		}
	}
	if hasKeyColQuery || len(scyllaTable.keyConcatenated) == 0 {
		return nil, false
	}

	prefixValues := []any{}
	var rangeStatement *ColumnStatement
	handledColumns := map[string]bool{}

	for _, concatCol := range scyllaTable.keyConcatenated {
		found := false
		for statementIndex := range statements {
			statement := statements[statementIndex]
			if statement.Col != concatCol.GetName() {
				continue
			}
			if statement.Operator == "=" {
				prefixValues = append(prefixValues, statement.Value)
				handledColumns[statement.Col] = true
				found = true
				break
			}
			if slices.Contains(rangeOperators, statement.Operator) || statement.Operator == "BETWEEN" {
				rangeStatement = &statements[statementIndex]
				handledColumns[statement.Col] = true
				found = true
				break
			}
		}
		if !found || rangeStatement != nil {
			break
		}
	}

	if len(prefixValues) == 0 && rangeStatement == nil {
		return nil, false
	}

	prefixValueText := ""
	if len(prefixValues) > 0 {
		prefixValueText = MakeKeyConcat(prefixValues...)
	}

	var rewrittenStatement ColumnStatement
	if rangeStatement == nil {
		if len(prefixValues) == len(scyllaTable.keyConcatenated) {
			rewrittenStatement = ColumnStatement{Col: keyCol.GetName(), Operator: "=", Value: prefixValueText}
		} else {
			rewrittenStatement = ColumnStatement{
				Col:      keyCol.GetName(),
				Operator: "BETWEEN",
				From:     []ColumnStatement{{Col: keyCol.GetName(), Value: prefixValueText + "_"}},
				To:       []ColumnStatement{{Col: keyCol.GetName(), Value: prefixValueText + "_\uffff"}},
			}
		}
	} else {
		if rangeStatement.Operator == "BETWEEN" {
			valueFrom := MakeKeyConcat(append(prefixValues, rangeStatement.From[0].Value)...)
			valueTo := MakeKeyConcat(append(prefixValues, rangeStatement.To[0].Value)...)
			rewrittenStatement = ColumnStatement{
				Col:      keyCol.GetName(),
				Operator: "BETWEEN",
				From:     []ColumnStatement{{Col: keyCol.GetName(), Value: valueFrom}},
				To:       []ColumnStatement{{Col: keyCol.GetName(), Value: valueTo + "\uffff"}},
			}
		} else if rangeTransform, ok := smartRangeMap[rangeStatement.Operator]; ok {
			rangeValue := MakeKeyConcat(append(prefixValues, rangeStatement.Value)...)
			prefixMin, prefixMax := "", "\uffff"
			if prefixValueText != "" {
				prefixMin = prefixValueText + "_"
				prefixMax = prefixValueText + "_\uffff"
			}
			fromValue := rangeTransform.from(rangeValue, prefixMin, prefixMax)
			toValue := rangeTransform.to(rangeValue, prefixMin, prefixMax)
			rewrittenStatement = ColumnStatement{Col: keyCol.GetName(), Operator: "BETWEEN"}
			if fromValue != "" {
				rewrittenStatement.From = append(rewrittenStatement.From, ColumnStatement{Col: keyCol.GetName(), Operator: ">=", Value: fromValue})
			}
			if toValue != "" {
				rewrittenStatement.To = append(rewrittenStatement.To, ColumnStatement{Col: keyCol.GetName(), Operator: "<", Value: toValue})
			}
		}
	}

	rewrittenStatements := []ColumnStatement{rewrittenStatement}
	for _, statement := range statements {
		if !handledColumns[statement.Col] {
			rewrittenStatements = append(rewrittenStatements, statement)
		}
	}

	return rewrittenStatements, true
}

func canRewriteKeyIntPacking(statements []ColumnStatement, scyllaTable ScyllaTable[any]) bool {
	_, canRewrite := buildKeyIntPackingStatements(statements, scyllaTable)
	return canRewrite
}

func buildKeyIntPackingStatements(statements []ColumnStatement, scyllaTable ScyllaTable[any]) ([]ColumnStatement, bool) {
	// Convert equality/range prefixes on packed numeric keys into one physical PK predicate plus residual filters.
	keyCol := scyllaTable.keys[0]
	hasKeyColQuery := false
	for _, st := range statements {
		if st.Col == keyCol.GetName() {
			hasKeyColQuery = true
			break
		}
	}
	if hasKeyColQuery || len(scyllaTable.keyIntPacking) == 0 {
		return nil, false
	}

	prefixValues := []any{}
	var rangeStatement *ColumnStatement
	handledColumns := map[string]bool{}

	for _, packedCol := range scyllaTable.keyIntPacking {
		colName := packedCol.GetName()
		if colName == "autoincrement_placeholder" {
			break
		}

		found := false
		for statementIndex := range statements {
			statement := statements[statementIndex]
			if statement.Col != colName {
				continue
			}
			if statement.Operator == "=" {
				prefixValues = append(prefixValues, statement.Value)
				handledColumns[statement.Col] = true
				found = true
				break
			}
			if slices.Contains(rangeOperators, statement.Operator) || statement.Operator == "BETWEEN" {
				rangeStatement = &statements[statementIndex]
				handledColumns[statement.Col] = true
				found = true
				break
			}
		}

		if !found || rangeStatement != nil {
			break
		}
	}

	if len(prefixValues) == 0 && rangeStatement == nil {
		return nil, false
	}

	makePackedRange := func(values []any, rangeStatement *ColumnStatement) (int64, int64, bool) {
		// Keep the exact packing math in one helper so compile and bind follow the same physical-key semantics.
		remainingDigits := int64(19)
		var packedValue int64

		for columnIndex, column := range scyllaTable.keyIntPacking {
			columnInfo := column.(*columnInfo)
			decimalSize := int64(columnInfo.decimalSize)
			if columnIndex == len(scyllaTable.keyIntPacking)-1 && decimalSize == 0 {
				decimalSize = remainingDigits
			}
			remainingDigits -= decimalSize

			if columnIndex < len(values) {
				packedValue += convertToInt64(values[columnIndex]) * Pow10Int64(remainingDigits)
				continue
			}

			if rangeStatement != nil && column.GetName() == rangeStatement.Col {
				if rangeStatement.Operator == "BETWEEN" {
					fromValue := packedValue + convertToInt64(rangeStatement.From[0].Value)*Pow10Int64(remainingDigits)
					toValue := packedValue + (convertToInt64(rangeStatement.To[0].Value)+1)*Pow10Int64(remainingDigits)
					return fromValue, toValue, false
				}

				rangeValue := convertToInt64(rangeStatement.Value)
				fromValue := packedValue + rangeValue*Pow10Int64(remainingDigits)
				return fromValue, fromValue + Pow10Int64(remainingDigits), false
			}

			fromValue := packedValue
			toValue := packedValue + Pow10Int64(remainingDigits+decimalSize)
			isEquality := columnIndex == len(scyllaTable.keyIntPacking)
			return fromValue, toValue, isEquality
		}

		return packedValue, packedValue, true
	}

	fromValue, toValue, isEquality := makePackedRange(prefixValues, rangeStatement)

	rewrittenStatement := ColumnStatement{
		Col:      keyCol.GetName(),
		Operator: "=",
		Value:    fromValue,
	}
	if !isEquality {
		rewrittenStatement = ColumnStatement{
			Col:      keyCol.GetName(),
			Operator: "BETWEEN",
			From:     []ColumnStatement{{Col: keyCol.GetName(), Value: fromValue}},
			To:       []ColumnStatement{{Col: keyCol.GetName(), Value: toValue}},
		}
	}

	rewrittenStatements := []ColumnStatement{rewrittenStatement}
	for _, statement := range statements {
		if !handledColumns[statement.Col] {
			rewrittenStatements = append(rewrittenStatements, statement)
		}
	}

	return rewrittenStatements, true
}

func compileSelectStatement(tableInfo *TableInfo, scyllaTable ScyllaTable[any]) (*SelectStatement, error) {
	statements := collectSelectStatements(tableInfo)
	selectShapeHash := computeSelectShapeHash(tableInfo, scyllaTable)

	if len(tableInfo.groupByColumns) > 0 {
		groupByPlan, err := buildNativeGroupByPlan(tableInfo, statements, scyllaTable)
		if err != nil {
			return nil, err
		}
		if groupByPlan == nil {
			return nil, fmt.Errorf("group by select shape did not produce a native plan")
		}

		sourceTableName := scyllaTable.name
		if groupByPlan.ViewTableName != "" {
			sourceTableName = groupByPlan.ViewTableName
		}

		orderColumnName := ""
		if groupByPlan.OrderColumn != nil && !groupByPlan.OrderColumn.IsNil() {
			orderColumnName = groupByPlan.OrderColumn.GetName()
		}

		compiledStatement := &SelectStatement{
			hash:                  selectShapeHash,
			queryTemplate:         makeSelectQueryTemplate(groupByPlan.SelectExpressions, scyllaTable.keyspace, sourceTableName),
			scanColumns:           slices.Clone(groupByPlan.ScanColumns),
			route:                 selectRouteNativeGroupBy,
			orderBy:               tableInfo.orderBy,
			orderColumnName:       orderColumnName,
			limit:                 tableInfo.limit,
			allowFilter:           tableInfo.allowFilter,
			groupByColumns:        slices.Clone(groupByPlan.GroupByColumns),
			assignCacheVersions:   false,
			requiresDeduplication: false,
		}

		fmt.Printf("Select plan compiled: table=%s hash=%d route=%d source=%s post_filter=false dedup=false\n",
			scyllaTable.name, compiledStatement.hash, compiledStatement.route, sourceTableName)
		return compiledStatement, nil
	}

	columnNames, scanColumns, selectExpressions := buildSelectProjection(tableInfo, scyllaTable)
	viewTableName := scyllaTable.name
	orderColumnName := ""
	if len(scyllaTable.keys) > 0 {
		orderColumnName = scyllaTable.keys[0].GetName()
	}

	compiledStatement := &SelectStatement{
		hash:                computeSelectShapeHash(tableInfo, scyllaTable),
		scanColumns:         slices.Clone(scanColumns),
		orderBy:             tableInfo.orderBy,
		orderColumnName:     orderColumnName,
		limit:               tableInfo.limit,
		allowFilter:         tableInfo.allowFilter,
		route:               selectRouteAllStatements,
		assignCacheVersions: true,
	}

	if compositePlan := tryBuildCompositeBucketPlan(statements, scyllaTable); compositePlan != nil {
		compiledStatement.route = selectRouteCompositeBucket
		compiledStatement.requiresDeduplication = true
		compiledStatement.queryTemplate = makeSelectQueryTemplate(selectExpressions, scyllaTable.keyspace, viewTableName)
		fmt.Printf("Select plan compiled: table=%s hash=%d route=%d source=%s post_filter=true dedup=true\n",
			scyllaTable.name, compiledStatement.hash, compiledStatement.route, viewTableName)
		return compiledStatement, nil
	}

	bestCapability := MatchQueryCapability(statements, scyllaTable.capabilities)
	if bestCapability != nil {
		if bestCapability.Source != nil {
			selectedView := bestCapability.Source
			if selectedView.Type >= 6 && canUseProjectedView(columnNames, selectedView) {
				viewTableName = selectedView.name
				if selectedView.Type == 9 {
					compiledStatement.requiresDeduplication = true
				}

				if selectedView.getStatement != nil {
					selectedStatementIndexes := []int{}
					remainingStatementIndexes := []int{}

					for statementIndex, statement := range statements {
						if slices.Contains(selectedView.columns, statement.Col) {
							selectedStatementIndexes = append(selectedStatementIndexes, statementIndex)
							continue
						}
						if len(statement.From) > 0 {
							isIncluded := true
							for _, betweenStatement := range statement.From {
								if !slices.Contains(selectedView.columns, betweenStatement.Col) {
									isIncluded = false
									break
								}
							}
							if isIncluded {
								selectedStatementIndexes = append(selectedStatementIndexes, statementIndex)
							} else {
								remainingStatementIndexes = append(remainingStatementIndexes, statementIndex)
							}
							continue
						}
						remainingStatementIndexes = append(remainingStatementIndexes, statementIndex)
					}

					compiledStatement.route = selectRouteViewStatements
					compiledStatement.sourceView = selectedView
					compiledStatement.selectedStatementIndexes = selectedStatementIndexes
					compiledStatement.remainingStatementIndexes = remainingStatementIndexes
					if selectedView.column != nil && !selectedView.column.IsNil() {
						compiledStatement.orderColumnName = selectedView.column.GetName()
					}
					if selectedView.RequiresPostFilter {
						// Post-filter must use current runtime values, so only the statement indexes are cached.
						compiledStatement.postFilterStatementIndexes = slices.Clone(selectedStatementIndexes)
						compiledStatement.requiresDeduplication = true
					}
				}
			}
		} else if bestCapability.IsKey {
			if canRewriteKeyConcatenated(statements, scyllaTable) {
				compiledStatement.route = selectRouteKeyConcatenated
			} else if canRewriteKeyIntPacking(statements, scyllaTable) {
				compiledStatement.route = selectRouteKeyIntPacking
			}
		}
	}

	compiledStatement.queryTemplate = makeSelectQueryTemplate(selectExpressions, scyllaTable.keyspace, viewTableName)

	fmt.Printf("Select plan compiled: table=%s hash=%d route=%d source=%s post_filter=%v dedup=%v\n",
		scyllaTable.name, compiledStatement.hash, compiledStatement.route, viewTableName,
		len(compiledStatement.postFilterStatementIndexes) > 0, compiledStatement.requiresDeduplication)

	return compiledStatement, nil
}

func (e *SelectStatement) Compute(tableInfo *TableInfo, scyllaTable ScyllaTable[any]) (*BoundSelectPlan, error) {
	// Bind current values into the cached query shape without rerunning planner selection in selectExec.
	statements := collectSelectStatements(tableInfo)
	whereStatements := []string{""}
	remainingStatements := statements
	postFilterStatements := pickStatementsByIndexes(statements, e.postFilterStatementIndexes)
	scanColumns := slices.Clone(e.scanColumns)
	queryTemplate := e.queryTemplate
	groupByColumns := slices.Clone(e.groupByColumns)
	orderColumnName := e.orderColumnName

	switch e.route {
	case selectRouteViewStatements:
		selectedStatements := pickStatementsByIndexes(statements, e.selectedStatementIndexes)
		remainingStatements = pickStatementsByIndexes(statements, e.remainingStatementIndexes)
		whereStatements = e.sourceView.getStatement(selectedStatements...)
	case selectRouteKeyConcatenated:
		rewrittenStatements, canRewrite := buildKeyConcatenatedStatements(statements, scyllaTable)
		if !canRewrite {
			return nil, fmt.Errorf("key concatenated select shape no longer matches the cached route")
		}
		remainingStatements = rewrittenStatements
	case selectRouteKeyIntPacking:
		rewrittenStatements, canRewrite := buildKeyIntPackingStatements(statements, scyllaTable)
		if !canRewrite {
			return nil, fmt.Errorf("key int packing select shape no longer matches the cached route")
		}
		remainingStatements = rewrittenStatements
	case selectRouteCompositeBucket:
		compositePlan := tryBuildCompositeBucketPlan(statements, scyllaTable)
		if compositePlan == nil {
			return nil, fmt.Errorf("composite bucket select shape no longer matches the cached route")
		}
		whereStatements = compositePlan.whereStatements
		remainingStatements = buildRemainingStatementsForCompositePlan(statements, compositePlan)
		postFilterStatements = slices.Clone(compositePlan.filterStatements)
	case selectRouteNativeGroupBy:
		groupByPlan, err := buildNativeGroupByPlan(tableInfo, statements, scyllaTable)
		if err != nil {
			return nil, err
		}
		if groupByPlan == nil {
			return nil, fmt.Errorf("group by select shape no longer matches the cached route")
		}

		sourceTableName := scyllaTable.name
		if groupByPlan.ViewTableName != "" {
			sourceTableName = groupByPlan.ViewTableName
		}
		queryTemplate = makeSelectQueryTemplate(groupByPlan.SelectExpressions, scyllaTable.keyspace, sourceTableName)
		scanColumns = slices.Clone(groupByPlan.ScanColumns)
		groupByColumns = slices.Clone(groupByPlan.GroupByColumns)
		remainingStatements = nil
		whereStatements = slices.Clone(groupByPlan.WhereStatements)
		if groupByPlan.OrderColumn != nil && !groupByPlan.OrderColumn.IsNil() {
			orderColumnName = groupByPlan.OrderColumn.GetName()
		}
	}

	return buildBoundSelectPlan(
		queryTemplate,
		scanColumns,
		e.requiresDeduplication,
		e.assignCacheVersions,
		whereStatements,
		remainingStatements,
		postFilterStatements,
		groupByColumns,
		e.orderBy,
		orderColumnName,
		e.limit,
		e.allowFilter,
	), nil
}

func tryGetOrCompileSelectStatement(tableInfo *TableInfo, scyllaTable ScyllaTable[any]) (*SelectStatement, error) {
	selectShapeHash := computeSelectShapeHash(tableInfo, scyllaTable)
	fmt.Printf("Select cache lookup: table=%s hash=%d\n", scyllaTable.name, selectShapeHash)

	if cachedPlan, cacheHit := scyllaTable.selectStatementCache.Load(selectShapeHash); cacheHit {
		fmt.Printf("Select cache hit: table=%s hash=%d\n", scyllaTable.name, selectShapeHash)
		return cachedPlan, nil
	}

	fmt.Printf("Select cache miss: table=%s hash=%d\n", scyllaTable.name, selectShapeHash)
	compiledStatement, err := compileSelectStatement(tableInfo, scyllaTable)
	if err != nil {
		return nil, err
	}

	scyllaTable.selectStatementCache.Store(selectShapeHash, compiledStatement)
	return compiledStatement, nil
}
