package core

import (
	"app/db2"
	"app/types"
	"fmt"
)

func GetCounter(name string, incrementCount int, empresaID ...int32) (int64, error) {
	if len(empresaID) == 1 {
		name = fmt.Sprintf("x%v_%v", empresaID[0], name)
	}

	result := []types.Increment{}
	query := db2.Query(&result)
	query.Select().Name.Equals(name)

	if err := query.Exec(); err != nil {
		return 0, Err("Error al obtener el counter: ", err)
	}

	currentValue := int64(1)
	if len(result) > 0 {
		currentValue = result[0].CurrentValue
	}

	queryUpdateStr := fmt.Sprintf(
		"UPDATE %v.sequences SET current_value = current_value + %v WHERE name = '%v'",
		Env.DB_NAME, incrementCount, name,
	)

	if err := db2.QueryExec(queryUpdateStr); err != nil {
		Log(queryUpdateStr)
		panic(err)
	}

	return currentValue, nil
}
