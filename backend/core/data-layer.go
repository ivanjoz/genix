package core

import (
	"app/db"
	"app/types"
	"fmt"
)

func GetCounter(name string, incrementCount int, empresaID ...int32) (int64, error) {
	if len(empresaID) == 1 {
		name = fmt.Sprintf("x%v_%v", empresaID[0], name)
	}

	result := db.Select(func(q *db.Query[types.Increment], col types.Increment) {
		q.Where(col.TableName_().Equals(name))
	})

	if result.Err != nil {
		return 0, Err("Error al obtener el counter: ", result.Err)
	}

	currentValue := int64(1)
	if len(result.Records) > 0 {
		currentValue = result.Records[0].CurrentValue
	}

	queryUpdateStr := fmt.Sprintf(
		"UPDATE %v.sequences SET current_value = current_value + %v WHERE name = '%v'",
		Env.DB_NAME, incrementCount, name,
	)

	if err := db.QueryExec(queryUpdateStr); err != nil {
		Log(queryUpdateStr)
		panic(err)
	}

	return currentValue, nil
}
