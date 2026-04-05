package db

import "fmt"

var selectOperators = map[int16]string{
	30000: "SELECT ",
	30001: ",",
	31001: ", ",
	30002: ")",
	31002: ") ",
	30003: "(",
	31003: "( ",
}

type SelectStatement struct {
	hash uint64
	syllaTable *ScyllaTable[any]
	view *View
	columns []int16
	statement []int16
	variables map[int16]any
}

func (e SelectStatement) Compute(variables []any){
	
	statement := ""
	
	for _, op := range e.statement {
		if op < 1000 {
			statement += e.syllaTable.columnsIdxMap[op].GetName()
		} else if operator, ok := selectOperators[op]; ok {
			statement += operator
		} else if variable, ok := e.variables[op]; ok {
			fmt.Println(variable)
			// Do something
		} else {
			panic("statement malformed:")
		}
	}	
}
