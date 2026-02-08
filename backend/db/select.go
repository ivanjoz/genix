package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/viant/xunsafe"
	"golang.org/x/sync/errgroup"
)

type smartRangeDef struct {
	from func(v, min, max string) string
	to   func(v, min, max string) string
}

var smartRangeMap = map[string]smartRangeDef{
	">=": {
		from: func(v, min, max string) string { return v },
		to:   func(v, min, max string) string { return max },
	},
	">": {
		from: func(v, min, max string) string { return v + "\uffff" },
		to:   func(v, min, max string) string { return max },
	},
	"<=": {
		from: func(v, min, max string) string { return min },
		to:   func(v, min, max string) string { return v + "\uffff" },
	},
	"<": {
		from: func(v, min, max string) string { return min },
		to:   func(v, min, max string) string { return v },
	},
}

type posibleIndex struct {
	indexView    *viewInfo
	colsIncluded []int16
	colsMissing  []int16
	priority     int8
}

// selectExec executes a query using TableSchema and TableInfo
func selectExec[E any](recordsGetted *[]E, tableInfo *TableInfo, scyllaTable ScyllaTable[any]) error {

	viewTableName := scyllaTable.name

	if len(scyllaTable.keyspace) == 0 {
		scyllaTable.keyspace = connParams.Keyspace
	}
	if len(scyllaTable.keyspace) == 0 {
		return errors.New("no se ha especificado un keyspace")
	}

	columnNames := []string{}
	if len(tableInfo.columnsInclude) > 0 {
		for _, col := range tableInfo.columnsInclude {
			columnNames = append(columnNames, col.GetName())
		}
	} else {
		columnsExclude := []string{}
		for _, col := range tableInfo.columnsExclude {
			columnsExclude = append(columnsExclude, col.GetName())
		}
		for _, col := range scyllaTable.columns {
			if !slices.Contains(columnsExclude, col.GetName()) && !col.GetInfo().IsVirtual {
				columnNames = append(columnNames, col.GetName())
			}
		}
	}

	queryTemplate := fmt.Sprintf("SELECT %v ", strings.Join(columnNames, ", ")) + "FROM %v.%v %v"
	hashOperators := []string{"=", "IN"}
	isHash := true

	statements := []ColumnStatement{}
	colsWhereIdx := []int16{}

	for _, st := range tableInfo.statements {
		col := scyllaTable.columnsMap[st.Col]

		colsWhereIdx = append(colsWhereIdx, col.GetInfo().Idx)
		statements = append(statements, st)
		if isHash && !slices.Contains(hashOperators, st.Operator) {
			isHash = false
		}
	}

	// Between Statement
	if len(tableInfo.between.From) > 0 {
		statements = append(statements, tableInfo.between)
	}

	bestCap := MatchQueryCapability(statements, scyllaTable.capabilities)

	statementsSelected := []ColumnStatement{}
	statementsRemain := []ColumnStatement{}
	whereStatements := []string{}
	orderColumn := scyllaTable.keys[0]

	if bestCap != nil {
		fmt.Println("Best Match Signature:", bestCap.Signature)
		
		handledCols := make(map[string]bool)
		parts := strings.Split(bestCap.Signature, "|")
		for i := 0; i < len(parts); i += 2 {
			handledCols[parts[i]] = true
		}

		if bestCap.Source != nil {
			viewOrIndex := bestCap.Source
			fmt.Println("Selected Index/View:", viewOrIndex.name)
			if viewOrIndex.Type >= 6 {
				viewTableName = viewOrIndex.name
			}

			if viewOrIndex.getStatement != nil {
				// Identify which statements match the view columns
				for _, st := range statements {
					if slices.Contains(viewOrIndex.columns, st.Col) {
						statementsSelected = append(statementsSelected, st)
					} else if len(st.From) > 0 { // BETWEEN
						isIncluded := true
						for _, e := range st.From {
							if !slices.Contains(viewOrIndex.columns, e.Col) {
								isIncluded = false
								break
							}
						}
						if isIncluded {
							statementsSelected = append(statementsSelected, st)
						} else {
							statementsRemain = append(statementsRemain, st)
						}
					} else {
						statementsRemain = append(statementsRemain, st)
					}
				}
				orderColumn = viewOrIndex.column
				whereStatements = viewOrIndex.getStatement(statementsSelected...)
			} else {
				// Direct index usage, no special statement generator
				statementsRemain = statements
			}
		} else if bestCap.IsKey {
			// Key or KeyConcatenated logic
			keyCol := scyllaTable.keys[0]
			hasKeyColQuery := false
			for _, st := range statements {
				if st.Col == keyCol.GetName() {
					hasKeyColQuery = true
					break
				}
			}

			if !hasKeyColQuery && len(scyllaTable.keyConcatenated) > 0 {
				// Apply KeyConcatenated smart logic
				prefixValues := []any{}
				var rangeSt *ColumnStatement
				handledInKeyConcat := make(map[string]bool)

				for _, concatCol := range scyllaTable.keyConcatenated {
					found := false
					for _, st := range statements {
						if st.Col == concatCol.GetName() {
							if st.Operator == "=" {
								prefixValues = append(prefixValues, st.Value)
								handledInKeyConcat[st.Col] = true
								found = true
								break
							} else if slices.Contains(rangeOperators, st.Operator) || st.Operator == "BETWEEN" {
								rangeSt = &st
								handledInKeyConcat[st.Col] = true
								found = true
								break
							}
						}
					}
					if !found || rangeSt != nil {
						break
					}
				}

				if len(prefixValues) > 0 || rangeSt != nil {
					prefixStr := ""
					if len(prefixValues) > 0 {
						prefixStr = Concat62(prefixValues...)
					}

					var newSt ColumnStatement
					if rangeSt == nil {
						val := prefixStr
						if len(prefixValues) == len(scyllaTable.keyConcatenated) {
							newSt = ColumnStatement{Col: keyCol.GetName(), Operator: "=", Value: val}
						} else {
							newSt = ColumnStatement{
								Col:      keyCol.GetName(),
								Operator: "BETWEEN",
								From:     []ColumnStatement{{Col: keyCol.GetName(), Value: val + "_"}},
								To:       []ColumnStatement{{Col: keyCol.GetName(), Value: val + "_\uffff"}},
							}
						}
					} else {
						if rangeSt.Operator == "BETWEEN" {
							valFrom := Concat62(append(prefixValues, rangeSt.From[0].Value)...)
							valTo := Concat62(append(prefixValues, rangeSt.To[0].Value)...)
							newSt = ColumnStatement{
								Col:      keyCol.GetName(),
								Operator: "BETWEEN",
								From:     []ColumnStatement{{Col: keyCol.GetName(), Value: valFrom}},
								To:       []ColumnStatement{{Col: keyCol.GetName(), Value: valTo + "\uffff"}},
							}
						} else if trans, ok := smartRangeMap[rangeSt.Operator]; ok {
							valWithRange := Concat62(append(prefixValues, rangeSt.Value)...)
							prefixMin, prefixMax := "", "\uffff"
							if prefixStr != "" {
								prefixMin = prefixStr + "_"
								prefixMax = prefixStr + "_\uffff"
							}
							fromVal := trans.from(valWithRange, prefixMin, prefixMax)
							toVal := trans.to(valWithRange, prefixMin, prefixMax)
							newSt = ColumnStatement{Col: keyCol.GetName(), Operator: "BETWEEN"}
							if fromVal != "" {
								newSt.From = append(newSt.From, ColumnStatement{Col: keyCol.GetName(), Operator: ">=", Value: fromVal})
							}
							if toVal != "" {
								newSt.To = append(newSt.To, ColumnStatement{Col: keyCol.GetName(), Operator: "<", Value: toVal})
							}
						}
					}
					
					// Add the new PK statement and skip handled concatenated columns
					statementsRemain = append(statementsRemain, newSt)
					for _, st := range statements {
						if !handledInKeyConcat[st.Col] {
							statementsRemain = append(statementsRemain, st)
						}
					}
				} else {
					statementsRemain = statements
				}
			} else if !hasKeyColQuery && len(scyllaTable.keyIntPacking) > 0 {
				// Apply KeyIntPacking smart logic
				prefixValues := []any{}
				var rangeSt *ColumnStatement
				handledInKeyPack := make(map[string]bool)

				for _, packedCol := range scyllaTable.keyIntPacking {
					colName := packedCol.GetName()
					if colName == "autoincrement_placeholder" {
						break
					}
					found := false
					for _, st := range statements {
						if st.Col == colName {
							if st.Operator == "=" {
								prefixValues = append(prefixValues, st.Value)
								handledInKeyPack[st.Col] = true
								found = true
								break
							} else if slices.Contains(rangeOperators, st.Operator) || st.Operator == "BETWEEN" {
								rangeSt = &st
								handledInKeyPack[st.Col] = true
								found = true
								break
							}
						}
					}
					if !found || rangeSt != nil {
						break
					}
				}

				if len(prefixValues) > 0 || rangeSt != nil {
					// Calculation formula: packedValue = (packedValue * 10^col.decimalSize) + val
					// We need to calculate the value and the remaining shift to place it in the correct slot
					
					makePacked := func(vals []any, rSt *ColumnStatement) (fromVal int64, toVal int64, isEquality bool) {
						remainingDigits := int64(19)
						var currentPacked int64
						
						// Process equality prefix
						for i, col := range scyllaTable.keyIntPacking {
							colInfo := col.(*columnInfo)
							decSize := int64(colInfo.decimalSize)
							if i == len(scyllaTable.keyIntPacking)-1 && decSize == 0 {
								decSize = remainingDigits
							}
							remainingDigits -= decSize
							
							if i < len(vals) {
								val := convertToInt64(vals[i])
								currentPacked += val * Pow10Int64(remainingDigits)
							} else if rSt != nil && col.GetName() == rSt.Col {
								// Handle range on current column
								if rSt.Operator == "BETWEEN" {
									fromVal = currentPacked + convertToInt64(rSt.From[0].Value)*Pow10Int64(remainingDigits)
									toVal = currentPacked + (convertToInt64(rSt.To[0].Value)+1)*Pow10Int64(remainingDigits)
								} else {
									// Handle other range operators if needed, or fallback to simple prefix range
									val := convertToInt64(rSt.Value)
									fromVal = currentPacked + val*Pow10Int64(remainingDigits)
									// For range operators like >, < we might need more complex logic, 
									// but for now let's focus on the common prefix range case.
									toVal = fromVal + Pow10Int64(remainingDigits)
								}
								return fromVal, toVal, false
							} else {
								// End of provided values
								fromVal = currentPacked
								toVal = currentPacked + Pow10Int64(remainingDigits+decSize)
								isEquality = (i == len(scyllaTable.keyIntPacking))
								return fromVal, toVal, isEquality
							}
						}
						return currentPacked, currentPacked, true
					}

					from, to, isEq := makePacked(prefixValues, rangeSt)
					
					var newSt ColumnStatement
					if isEq {
						newSt = ColumnStatement{Col: keyCol.GetName(), Operator: "=", Value: from}
					} else {
						newSt = ColumnStatement{
							Col:      keyCol.GetName(),
							Operator: "BETWEEN",
							From:     []ColumnStatement{{Col: keyCol.GetName(), Value: from}},
							To:       []ColumnStatement{{Col: keyCol.GetName(), Value: to}},
						}
					}
					
					// Add the new PK statement and skip handled packed columns
					statementsRemain = append(statementsRemain, newSt)
					for _, st := range statements {
						if !handledInKeyPack[st.Col] {
							statementsRemain = append(statementsRemain, st)
						}
					}
				} else {
					statementsRemain = statements
				}
			} else {
				statementsRemain = statements
			}
		}
	} else {
		// No match found, fallback to default behavior (all remain)
		statementsRemain = statements
	}

	if len(whereStatements) == 0 {
		whereStatements = append(whereStatements, "")
	}

	// fmt.Println("where statements::", whereStatements)
	whereStatementsRemainSects := []string{}
	for _, st := range statementsRemain {
		if len(st.From) > 0 {
			for i := range st.From {
				colname := st.From[i].Col
				w1 := fmt.Sprintf("%v >= %v", colname, st.From[i].GetValue())
				whereStatementsRemainSects = append(whereStatementsRemainSects, w1)
				w2 := fmt.Sprintf("%v < %v", colname, st.To[i].GetValue())
				whereStatementsRemainSects = append(whereStatementsRemainSects, w2)
			}
		} else {
			whereStatementsRemainSects = append(whereStatementsRemainSects,
				fmt.Sprintf("%v %v %v", st.Col, st.Operator, st.GetValue()))
		}
	}

	whereStatementsRemain := strings.Join(whereStatementsRemainSects, " AND ")
	queryWhereStatements := []string{}

	for _, whereStatement := range whereStatements {
		if whereStatementsRemain != "" {
			if whereStatement != "" {
				whereStatementsRemain = " AND " + whereStatementsRemain
			}
			whereStatement += whereStatementsRemain
		}
		if whereStatement != "" {
			whereStatement = " WHERE " + whereStatement
		}
		if len(tableInfo.orderBy) > 0 {
			whereStatement += " " + fmt.Sprintf(tableInfo.orderBy, orderColumn.GetName())
		}
		if tableInfo.limit > 0 {
			whereStatement += fmt.Sprintf(" LIMIT %v", tableInfo.limit)
		}
		queryWhereStatements = append(queryWhereStatements, whereStatement)
	}

	recordsMap := map[int]*[]E{}
	for i := range queryWhereStatements {
		recordsMap[i] = &[]E{}
	}

	eg := errgroup.Group{}

	for i, whereStatement := range queryWhereStatements {
		queryStr := fmt.Sprintf(queryTemplate, scyllaTable.keyspace, viewTableName, whereStatement)
		records := recordsMap[i]

		fmt.Println("Query::", queryStr)
		doScan := func() error {
			iter := getScyllaConnection().Query(queryStr).Iter()
			rd, err := iter.RowData()
			if err != nil {
				fmt.Println("Error on RowData::", err)
				if strings.Contains(err.Error(), "use ALLOW FILTERING") {
					fmt.Println("Posible Indexes or Views::")
					for _, e := range scyllaTable.capabilities {
						fmt.Println(e.Signature)
					}
				}
				return err
			}

			scanner := iter.Scanner()
			fmt.Println("starting iterator | columns::", len(scyllaTable.columns))

			rowCount := 0
			for scanner.Next() {
				rowCount++

				rowValues := rd.Values

				if err := scanner.Scan(rowValues...); err != nil {
					fmt.Println("Error on scan::", err)
					return err
				}

				rec := new(E)
				ptr := xunsafe.AsPointer(rec)
				shouldLog := ShouldLog()

				if shouldLog {
					fmt.Printf("\n--- Scanning record %d ---\n", atomic.LoadUint32(&LogCount)+1)
				}

				for idx, colname := range columnNames {
					column := scyllaTable.columnsMap[colname]
					value := rowValues[idx]

					if shouldLog {
						valStr := "nil"
						if value != nil {
							valStr = fmt.Sprintf("%v", reflect.Indirect(reflect.ValueOf(value)).Interface())
						}
						offset := uintptr(0)
						if column.GetInfo().Field != nil {
							offset = column.GetInfo().Field.Offset
						}
						fmt.Printf("Col: %-20s (Field: %-20s) | Offset: %-4d | DB Value: %s\n", colname, column.GetInfo().FieldName, offset, valStr)
					}

					if value == nil {
						continue
					}
					if shouldLog {
						fmt.Printf("Calling SetValue for Col: %s\n", colname)
					}
					column.SetValue(ptr, value)
				}

				if shouldLog {
					recJSON, _ := json.MarshalIndent(rec, "", "  ")
					fmt.Printf("Resulting Struct: %s\n", string(recJSON))
					IncrementLogCount()
				}

				if len(queryWhereStatements) == 1 {
					(*recordsGetted) = append((*recordsGetted), *rec)
				} else {
					(*records) = append((*records), *rec)
				}
			}
			return nil
		}

		eg.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("Panic in select goroutine: %v", r)
				}
			}()

			err = doScan()
			if err != nil {
				if strings.Contains(err.Error(), "no hosts available") {
					scyllaSession = nil
					fmt.Println("Reconectando con ScyllaDB...")
					getScyllaConnection()
					err = doScan()
				}
			}
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if len(queryWhereStatements) > 1 {
		for _, records := range recordsMap {
			(*recordsGetted) = append((*recordsGetted), *records...)
		}
	}

	return nil
}
