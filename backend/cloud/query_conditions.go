package cloud

import "fmt"

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
