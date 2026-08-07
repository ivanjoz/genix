package cloud

import (
	"fmt"
	"strings"
)

type queryCondition struct {
	ColumnName string
	Operator   string
	Value      interface{}
	ValueEnd   interface{}
}

func appendCondition(existingConditions []queryCondition, pendingColumn string, operator string, value interface{}, valueEnd interface{}) []queryCondition {
	// Keep query building explicit so Exec can validate the logical partition before hitting the provider.
	return append(existingConditions, queryCondition{
		ColumnName: pendingColumn,
		Operator:   operator,
		Value:      value,
		ValueEnd:   valueEnd,
	})
}

func findLogicalPartitionColumn(columns []ColumnMeta) (string, bool) {
	for _, column := range columns {
		if column.IsPK {
			return column.ColumnName, true
		}
	}
	return "", false
}

func splitQueryConditions(conditions []queryCondition, partitionColumn string, requirePartition bool) (*queryCondition, []queryCondition, error) {
	var partitionCondition *queryCondition
	remainingConditions := make([]queryCondition, 0, len(conditions))

	for _, condition := range conditions {
		if condition.ColumnName == "" {
			return nil, nil, fmt.Errorf("must specify a column using Where() before the operator")
		}
		if requirePartition && condition.ColumnName == partitionColumn {
			if condition.Operator != "=" {
				return nil, nil, fmt.Errorf("partition column %s must use Equals()", partitionColumn)
			}
			if partitionCondition != nil {
				return nil, nil, fmt.Errorf("partition column %s can only be provided once", partitionColumn)
			}
			conditionCopy := condition
			partitionCondition = &conditionCopy
			continue
		}
		remainingConditions = append(remainingConditions, condition)
	}

	if requirePartition && partitionCondition == nil {
		return nil, nil, fmt.Errorf("missing required partition filter Where(%q).Equals(...)", partitionColumn)
	}

	return partitionCondition, remainingConditions, nil
}

// matchIndex picks the index whose leading key columns are exactly the columns the query
// constrains, in the same order. Only the last constrained key may use a range operator:
// the composite is a single string, so an unpinned prefix component would make the range
// meaningless.
func matchIndex(indexes []IndexMeta, conditions []queryCondition) (IndexMeta, error) {
	if len(conditions) == 0 {
		return IndexMeta{}, fmt.Errorf("a query needs at least one non-partition Where()")
	}

	for conditionIndex, condition := range conditions[:len(conditions)-1] {
		if condition.Operator != "=" {
			return IndexMeta{}, fmt.Errorf("only the last Where() may use a range operator, but %s (position %d) uses %s",
				condition.ColumnName, conditionIndex+1, condition.Operator)
		}
	}

	for _, index := range indexes {
		if len(conditions) > len(index.Keys) {
			continue
		}
		matches := true
		for conditionIndex, condition := range conditions {
			if index.Keys[conditionIndex].ColumnName != condition.ColumnName {
				matches = false
				break
			}
		}
		if matches {
			return index, nil
		}
	}

	constrainedColumns := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		constrainedColumns = append(constrainedColumns, condition.ColumnName)
	}
	return IndexMeta{}, fmt.Errorf("no index starts with the columns (%s); declare one in the table's GetSchema().Indexes",
		strings.Join(constrainedColumns, ", "))
}

// compositeRange is the pair of bounds a query maps onto a composite index value. An
// exact match sets both bounds to the same string.
type compositeRange struct {
	Lower   string
	Upper   string
	IsExact bool
}

// buildCompositeRange renders the bounds for one query against one index. Everything the
// query pins becomes a literal prefix; whatever the index still has room for is filled
// with the lowest and highest values its digit width can hold, which is what keeps the
// scan inside the pinned prefix instead of spilling into the next tenant or status.
func buildCompositeRange(index IndexMeta, partitionValue string, conditions []queryCondition) (compositeRange, error) {
	parts := make([]string, 0, len(index.Keys)+1)
	if index.PrefixFieldName != "" {
		parts = append(parts, partitionValue)
	}

	// Every condition but the last pins its component to a single value.
	for conditionIndex, condition := range conditions[:len(conditions)-1] {
		parts = append(parts, formatIndexKeyValue(index.Keys[conditionIndex], condition.Value))
	}

	trailingCondition := conditions[len(conditions)-1]
	trailingKey := index.Keys[len(conditions)-1]
	remainingKeys := index.Keys[len(conditions):]

	lowerValue, upperValue, err := boundsForOperator(trailingKey, trailingCondition)
	if err != nil {
		return compositeRange{}, err
	}

	lowerParts := append(append([]string{}, parts...), lowerValue)
	upperParts := append(append([]string{}, parts...), upperValue)

	for _, remainingKey := range remainingKeys {
		if remainingKey.Digits <= 0 {
			return compositeRange{}, fmt.Errorf(
				"index key %s has no declared digit width, so it cannot be left unconstrained in a range query; add .DecimalSize(n) or pin it with Equals()",
				remainingKey.ColumnName)
		}
		lowerParts = append(lowerParts, strings.Repeat("0", int(remainingKey.Digits)))
		upperParts = append(upperParts, strings.Repeat("9", int(remainingKey.Digits)))
	}

	lower := strings.Join(lowerParts, "_")
	upper := strings.Join(upperParts, "_")

	return compositeRange{Lower: lower, Upper: upper, IsExact: lower == upper}, nil
}

// boundsForOperator turns one operator on the trailing key into the lowest and highest
// values that satisfy it. Padded components compare in numeric order, so a strict
// inequality is just the adjacent value.
func boundsForOperator(key IndexKeyMeta, condition queryCondition) (string, string, error) {
	if key.Digits <= 0 {
		if condition.Operator != "=" {
			return "", "", fmt.Errorf(
				"index key %s has no declared digit width, so it only supports Equals(); add .DecimalSize(n) to range over it",
				key.ColumnName)
		}
		value := formatIndexKeyValue(key, condition.Value)
		return value, value, nil
	}

	lowestValue := strings.Repeat("0", int(key.Digits))
	highestValue := strings.Repeat("9", int(key.Digits))

	switch condition.Operator {
	case "=":
		value := formatIndexKeyValue(key, condition.Value)
		return value, value, nil
	case ">=":
		return formatIndexKeyValue(key, condition.Value), highestValue, nil
	case ">":
		return formatIndexKeyValue(key, toInt64(condition.Value)+1), highestValue, nil
	case "<=":
		return lowestValue, formatIndexKeyValue(key, condition.Value), nil
	case "<":
		return lowestValue, formatIndexKeyValue(key, toInt64(condition.Value)-1), nil
	case "BETWEEN":
		return formatIndexKeyValue(key, condition.Value), formatIndexKeyValue(key, condition.ValueEnd), nil
	}

	return "", "", fmt.Errorf("unsupported operator %s on index key %s", condition.Operator, key.ColumnName)
}
