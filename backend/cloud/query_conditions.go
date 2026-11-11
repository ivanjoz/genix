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

// matchIndex picks the single-key index on the one column the query constrains (besides the
// partition). Every mirrored read is one lookup (a login handle, an email) or one range (companies
// by updated); composite indexes are not queried.
func matchIndex(indexes []IndexMeta, conditions []queryCondition) (IndexMeta, error) {
	if len(conditions) != 1 {
		return IndexMeta{}, fmt.Errorf("a mirror query constrains exactly one non-partition column, got %d", len(conditions))
	}
	for _, index := range indexes {
		if len(index.Keys) == 1 && index.Keys[0].ColumnName == conditions[0].ColumnName {
			return index, nil
		}
	}
	return IndexMeta{}, fmt.Errorf("no single-key index on %s; declare one in the table's GetSchema().Indexes",
		conditions[0].ColumnName)
}

// indexKeyRange is the pair of bounds a query maps onto an index value. An exact match sets both
// bounds to the same string.
type indexKeyRange struct {
	Lower   string
	Upper   string
	IsExact bool
}

// buildIndexKeyRange renders the bounds of the query's one condition, prefixed with the partition
// when the index carries it, so the scan stays inside the tenant.
func buildIndexKeyRange(index IndexMeta, partitionValue string, condition queryCondition) (indexKeyRange, error) {
	lowerValue, upperValue, err := boundsForOperator(index.Keys[0], condition)
	if err != nil {
		return indexKeyRange{}, err
	}
	if index.PrefixFieldName != "" {
		lowerValue, upperValue = partitionValue+"_"+lowerValue, partitionValue+"_"+upperValue
	}
	return indexKeyRange{Lower: lowerValue, Upper: upperValue, IsExact: lowerValue == upperValue}, nil
}

// boundsForOperator turns one operator on the trailing key into the lowest and highest
// values that satisfy it. Padded components compare in numeric order, so a strict
// inequality is just the adjacent value.
func boundsForOperator(key IndexKeyMeta, condition queryCondition) (string, string, error) {
	if key.Digits <= 0 {
		if condition.Operator != "=" {
			return "", "", fmt.Errorf(
				"index key %s has no declared digit width, so it only supports Equals(); add .Size(bits) to range over it",
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
