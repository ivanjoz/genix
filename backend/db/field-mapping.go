package db

import (
	"fmt"
	"reflect"
)

type number1 interface {
	int | int32 | int8 | uint8 | int16 | uint16 | int64
}

/*
type numfloat interface {
	float32 | float64
}
*/

func setReflectInt[T number1, E number1](e *reflect.Value, vl *E) {
	(*e).SetInt(int64(*vl))
}

func setReflectIntPointer[T number1, E number1](e *reflect.Value, vl *E) {
	if vl != nil {
		pv := T(*vl)
		(*e).Set(reflect.ValueOf(&pv))
	}
}

func setReflectIntSlice[T number1, E number1](e *reflect.Value, vl *[]E) {
	newSlice := []T{}
	for _, v := range *vl {
		newSlice = append(newSlice, T(v))
	}
	(*e).Set(reflect.ValueOf(newSlice))
}

func setReflectIntSlicePointer[T number1, E number1](e *reflect.Value, vl *[]E) {
	newSlice := []T{}
	for _, v := range *vl {
		newSlice = append(newSlice, T(v))
	}
	(*e).Set(reflect.ValueOf(&newSlice))
}

type fieldMap struct {
	fieldType string
	isPointer int8
	isSlice   int8
}

func makeMappingKey(col *columnInfo) fieldMap {
	fm := fieldMap{fieldType: col.FieldType}
	if col.IsPointer {
		fm.isPointer = 1
	}
	if col.IsSlice {
		fm.isSlice = 1
	}
	return fm
}

func printError(valType string, v any) {
	fmt.Printf("Error: El valor %v no fue mapeado = %v\n", valType, reflect.Indirect(reflect.ValueOf(v)))
}

var fieldMapping = map[fieldMap]func(e *reflect.Value, value any){
	// NO SLICE
	{"string", 0, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*string); ok {
			// fmt.Println("seteando string::", *vl)
			(*e).SetString(*vl)
		}
	},
	{"string", 1, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*string); ok {
			(*e).Set(reflect.ValueOf(vl))
		}
	},
	{"int32", 0, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int); ok {
			setReflectInt[int32](e, vl)
		} else {
			printError("int32", v)
		}
	},
	{"int32", 1, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int); ok {
			setReflectIntPointer[int32](e, vl)
		} else {
			printError("int32", v)
		}
	},
	{"int", 0, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int); ok {
			setReflectInt[int32](e, vl)
		} else {
			printError("int", v)
		}
	},
	{"int", 1, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int); ok {
			setReflectIntPointer[int32](e, vl)
		} else {
			printError("int", v)
		}
	},
	{"int16", 0, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int16); ok {
			setReflectInt[int16](e, vl)
		} else if vl, ok := v.(*int); ok {
			setReflectInt[int](e, vl)
		} else {
			printError("int16", v)
		}
	},
	{"int16", 1, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int16); ok {
			setReflectIntPointer[int16](e, vl)
		} else if vl, ok := v.(*int); ok {
			setReflectIntPointer[int](e, vl)
		} else {
			printError("int16", v)
		}
	},
	{"int8", 0, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int8); ok {
			setReflectInt[int](e, vl)
		} else if vl, ok := v.(*int); ok {
			setReflectInt[int](e, vl)
		} else {
			printError("int8", v)
		}
	},
	{"int8", 1, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int8); ok {
			setReflectIntPointer[int](e, vl)
		} else if vl, ok := v.(*int); ok {
			setReflectIntPointer[int](e, vl)
		} else {
			printError("int8", v)
		}
	},
	{"int64", 0, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int64); ok {
			setReflectInt[int64](e, vl)
		} else if vl, ok := v.(*int); ok {
			setReflectInt[int64](e, vl)
		} else {
			printError("int64", v)
		}
	},
	{"int64", 1, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*int64); ok {
			setReflectIntPointer[int64](e, vl)
		} else if vl, ok := v.(*int); ok {
			setReflectIntPointer[int64](e, vl)
		} else {
			printError("int64", v)
		}
	},
	{"float32", 0, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*float32); ok {
			(*e).SetFloat(float64(*vl))
		} else if vl, ok := v.(*float64); ok {
			(*e).SetFloat(float64(*vl))
		} else {
			printError("float32", v)
		}
	},
	{"float32", 1, 0}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*float32); ok {
			(*e).Set(reflect.ValueOf(vl))
		} else if vl, ok := v.(*float64); ok {
			(*e).Set(reflect.ValueOf(float32(*vl)))
		} else {
			printError("float32", v)
		}
	},
	// SLICE
	{"int32", 0, 1}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int32](e, vl)
		} else {
			printError("[]int32", v)
		}
	},
	{"int32", 1, 1}: func(e *reflect.Value, v any) { // Pointer
		if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int32](e, vl)
		} else {
			printError("[]int32", v)
		}
	},
	{"int16", 0, 1}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*[]int16); ok {
			setReflectIntSlice[int16](e, vl)
		} else if vl, ok := v.(*[]int8); ok {
			setReflectIntSlice[int16](e, vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int16](e, vl)
		} else {
			printError("[]int16", v)
		}
	},
	{"int16", 1, 1}: func(e *reflect.Value, v any) { // Pointer
		if vl, ok := v.(*[]int16); ok {
			setReflectIntSlicePointer[int16](e, vl)
		} else if vl, ok := v.(*[]int8); ok {
			setReflectIntSlicePointer[int16](e, vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int16](e, vl)
		} else {
			printError("[]int16", v)
		}
	},
	{"int8", 0, 1}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*[]int8); ok {
			setReflectIntSlice[int8](e, vl)
		} else if vl, ok := v.(*[]int16); ok {
			setReflectIntSlice[int8](e, vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int8](e, vl)
		} else {
			printError("[]int8", v)
		}
	},
	{"int8", 1, 1}: func(e *reflect.Value, v any) { // Pointer
		if vl, ok := v.(*[]int8); ok {
			setReflectIntSlicePointer[int8](e, vl)
		} else if vl, ok := v.(*[]int16); ok {
			setReflectIntSlicePointer[int8](e, vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int8](e, vl)
		} else {
			printError("[]int8", v)
		}
	},
	{"int64", 0, 1}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*[]int64); ok {
			setReflectIntSlice[int64](e, vl)
		} else if vl, ok := v.(*[]int16); ok {
			setReflectIntSlice[int64](e, vl)
		} else {
			printError("[]int16", v)
		}
	},
	{"int64", 1, 1}: func(e *reflect.Value, v any) { // Pointer
		if vl, ok := v.(*[]int64); ok {
			setReflectIntSlicePointer[int64](e, vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int64](e, vl)
		} else {
			printError("[]int16", v)
		}
	},
	{"string", 0, 1}: func(e *reflect.Value, v any) {
		if vl, ok := v.(*[]string); ok {
			(*e).Set(reflect.ValueOf(*vl))
		}
	},
	{"string", 1, 1}: func(e *reflect.Value, v any) { // Pointer
		if vl, ok := v.(*[]string); ok {
			(*e).Set(reflect.ValueOf(vl))
		}
	},
}
