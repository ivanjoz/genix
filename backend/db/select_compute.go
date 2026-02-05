package db

import (
	"fmt"
	"slices"
	"strings"
)

type QueryCapability struct {
	Signature string
	Source    *viewInfo
	Priority  int
	IsKey     bool // If it's the main table primary key
}

// GetQuerySignature generates a signature for a set of ColumnStatements
func GetQuerySignature(statements []ColumnStatement) string {
	// Sort statements by column name to ensure consistent signatures for hashing/matching
	// but for Scylla we usually care about the order of keys.
	// Actually, the matching logic should probably be smarter than just string matching
	// because the order in WHERE doesn't matter, but the order in the index DOES matter.
	
	// For now, let's just collect what we have.
	type colOp struct {
		col string
		op  string
	}
	ops := []colOp{}
	for _, st := range statements {
		op := "="
		if slices.Contains(rangeOperators, st.Operator) || st.Operator == "BETWEEN" {
			op = "~"
		}
		ops = append(ops, colOp{st.Col, op})
	}
	
	// We need to match these ops against the capabilities.
	return "" // Will be used differently
}

func (dbTable *ScyllaTable[T]) ComputeCapabilities() []QueryCapability {
	caps := []QueryCapability{}

	// 1. Main Table Primary Key
	pk := dbTable.partKey
	if pk != nil && !pk.IsNil() {
		// Just partition
		caps = append(caps, QueryCapability{
			Signature: fmt.Sprintf("%v|=", pk.GetName()),
			Priority:  10,
			IsKey:     true,
		})

		// Partition + Clustering Keys
		currentSig := fmt.Sprintf("%v|=", pk.GetName())
		for i, key := range dbTable.keys {
			colName := key.GetName()
			// Equality
			caps = append(caps, QueryCapability{
				Signature: currentSig + fmt.Sprintf("|%v|=", colName),
				Priority:  20 + i*2,
				IsKey:     true,
			})
			// Range (only supported on clustering keys)
			caps = append(caps, QueryCapability{
				Signature: currentSig + fmt.Sprintf("|%v|~", colName),
				Priority:  15 + i*2,
				IsKey:     true,
			})
			currentSig += fmt.Sprintf("|%v|=", colName)
		}
	}

	// 2. Local Indexes
	for _, idx := range dbTable.indexes {
		if idx.Type == 2 { // Local Index
			pkName := dbTable.GetPartKey().GetName()
			colName := idx.column.GetName()
			// Equality
			caps = append(caps, QueryCapability{
				Signature: fmt.Sprintf("%v|=|%v|=", pkName, colName),
				Source:    idx,
				Priority:  12,
			})
			// Range
			caps = append(caps, QueryCapability{
				Signature: fmt.Sprintf("%v|=|%v|~", pkName, colName),
				Source:    idx,
				Priority:  11,
			})
		} else if idx.Type == 1 { // Global Index
			colName := idx.column.GetName()
			caps = append(caps, QueryCapability{
				Signature: fmt.Sprintf("%v|=", colName),
				Source:    idx,
				Priority:  10,
			})
		}
	}

	// 3. Views
	for _, view := range dbTable.indexViews {
		if view.Type < 6 {
			continue
		}
		
		sigBase := ""
		if view.Type == 7 || view.Type == 3 { // Hash
			cols := []string{}
			for _, col := range view.columns {
				cols = append(cols, col+"|=")
			}
			sigBase = strings.Join(cols, "|")
			caps = append(caps, QueryCapability{
				Signature: sigBase,
				Source:    view,
				Priority:  30 + len(view.columns)*2,
			})
		} else if view.Type == 6 { // Simple View
			cols := []string{}
			for _, col := range view.columns {
				cols = append(cols, col+"|=")
			}
			sigBase = strings.Join(cols, "|")
			caps = append(caps, QueryCapability{
				Signature: sigBase,
				Source:    view,
				Priority:  25 + len(view.columns)*2,
			})
		} else if view.Type == 8 { // Range/Radix
			// Radix views always have equality on prefix, range on last.
			// They also usually keep part.
			cols := []string{}
			for i, col := range view.columns {
				if i < len(view.columns)-1 {
					cols = append(cols, col+"|=")
				} else {
					// Last one can be = or ~
					sigPrefix := strings.Join(cols, "|")
					if sigPrefix != "" {
						sigPrefix += "|"
					}
					
					caps = append(caps, QueryCapability{
						Signature: sigPrefix + col + "|=",
						Source:    view,
						Priority:  35 + len(view.columns)*2,
					})
					caps = append(caps, QueryCapability{
						Signature: sigPrefix + col + "|~",
						Source:    view,
						Priority:  30 + len(view.columns)*2,
					})
				}
			}
		}
	}

	// 4. KeyConcatenated
	if len(dbTable.keyConcatenated) > 0 {
		pk := dbTable.partKey
		if pk != nil && !pk.IsNil() {
			pkName := pk.GetName()
			currentSig := fmt.Sprintf("%v|=", pkName)
			
			for i, col := range dbTable.keyConcatenated {
				colName := col.GetName()
				// Equality on this prefix maps to a range or equality on the actual PK
				// If it's the last column of KeyConcatenated, it's equality on PK
				// Otherwise it's a range prefix search on PK.
				
				isLast := i == len(dbTable.keyConcatenated)-1
				
				// Equality
				caps = append(caps, QueryCapability{
					Signature: currentSig + fmt.Sprintf("|%v|=", colName),
					Priority:  25 + i*2,
					IsKey:     true, // It's handled by PK smart logic
				})
				
				// Range on this column
				caps = append(caps, QueryCapability{
					Signature: currentSig + fmt.Sprintf("|%v|~", colName),
					Priority:  20 + i*2,
					IsKey:     true,
				})
				
				if isLast {
					// All concatenated columns provided with equality = Equality on PK
				}

				currentSig += fmt.Sprintf("|%v|=", colName)
			}
		}
	}

	return caps
}

func MatchQueryCapability(statements []ColumnStatement, capabilities []QueryCapability) *QueryCapability {
	// Create a map of available columns and their operators in the query
	queryOps := make(map[string][]string)
	for _, st := range statements {
		op := "="
		if slices.Contains(rangeOperators, st.Operator) || st.Operator == "BETWEEN" {
			op = "~"
		}
		queryOps[st.Col] = append(queryOps[st.Col], op)
	}

	var bestMatch *QueryCapability
	
	for _, cap := range capabilities {
		parts := strings.Split(cap.Signature, "|")
		match := true
		
		// Every column in the signature MUST be in the query with the correct operator
		colMatches := make(map[string]int)
		for i := 0; i < len(parts); i += 2 {
			col := parts[i]
			op := parts[i+1]
			
			qOps, ok := queryOps[col]
			if !ok {
				match = false
				break
			}
			
			// Check if we have enough of this op in the query
			found := false
			startIdx := colMatches[col]
			for j := startIdx; j < len(qOps); j++ {
				if qOps[j] == op || (op == "~" && qOps[j] == "=") {
					found = true
					colMatches[col] = j + 1
					break
				}
			}
			
			if !found {
				match = false
				break
			}
		}
		
		if match {
			if bestMatch == nil || cap.Priority > bestMatch.Priority {
				bestMatch = &cap
			} else if cap.Priority == bestMatch.Priority {
				// Prefer signature with more columns if priorities are equal
				if len(parts) > len(strings.Split(bestMatch.Signature, "|")) {
					bestMatch = &cap
				}
			}
		}
	}
	
	return bestMatch
}
