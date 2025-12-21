package db2

import (
	"fmt"
	"slices"
	"unsafe"

	"github.com/viant/xunsafe"
)

type number1 interface {
	int | int32 | int8 | uint8 | int16 | uint16 | int64
}

func setField[T any](f *xunsafe.Field, ptr unsafe.Pointer, val T) {
	*(*T)(f.Pointer(ptr)) = val
}

func setReflectIntSlice[T number1, E number1](f *xunsafe.Field, ptr unsafe.Pointer, vl *[]E) {
	newSlice := make([]T, len(*vl))
	for i, v := range *vl {
		newSlice[i] = T(v)
	}
	setField(f, ptr, newSlice)
}

func setReflectIntSlicePointer[T number1, E number1](f *xunsafe.Field, ptr unsafe.Pointer, vl *[]E) {
	newSlice := make([]T, len(*vl))
	for i, v := range *vl {
		newSlice[i] = T(v)
	}
	setField(f, ptr, &newSlice)
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
	fmt.Printf("Error: El valor %v no fue mapeado = %v\n", valType, v)
}

var fieldMapping = map[fieldMap]func(f *xunsafe.Field, ptr unsafe.Pointer, value any){
	// NO SLICE
	{"string", 0, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*string); ok {
			f.SetString(ptr, *vl)
		}
	},
	{"string", 1, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*string); ok {
			f.Set(ptr, vl)
		}
	},
	{"int32", 0, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int32); ok {
			f.SetInt32(ptr, *vl)
		} else if vl, ok := v.(*int); ok {
			f.SetInt32(ptr, int32(*vl))
		} else if vl, ok := v.(*int64); ok {
			f.SetInt32(ptr, int32(*vl))
		} else {
			printError("int32", v)
		}
	},
	{"int32", 1, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int32); ok {
			f.Set(ptr, vl)
		} else if vl, ok := v.(*int); ok {
			val := int32(*vl)
			f.Set(ptr, &val)
		} else if vl, ok := v.(*int64); ok {
			val := int32(*vl)
			f.Set(ptr, &val)
		} else {
			printError("int32", v)
		}
	},
	{"int", 0, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int); ok {
			f.SetInt(ptr, *vl)
		} else if vl, ok := v.(*int32); ok {
			f.SetInt(ptr, int(*vl))
		} else if vl, ok := v.(*int64); ok {
			f.SetInt(ptr, int(*vl))
		} else {
			printError("int", v)
		}
	},
	{"int", 1, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int); ok {
			f.Set(ptr, vl)
		} else if vl, ok := v.(*int32); ok {
			val := int(*vl)
			f.Set(ptr, &val)
		} else if vl, ok := v.(*int64); ok {
			val := int(*vl)
			f.Set(ptr, &val)
		} else {
			printError("int", v)
		}
	},
	{"int16", 0, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int16); ok {
			f.SetInt16(ptr, *vl)
		} else if vl, ok := v.(*int); ok {
			f.SetInt16(ptr, int16(*vl))
		} else if vl, ok := v.(*int32); ok {
			f.SetInt16(ptr, int16(*vl))
		} else if vl, ok := v.(*int64); ok {
			f.SetInt16(ptr, int16(*vl))
		} else {
			printError("int16", v)
		}
	},
	{"int16", 1, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int16); ok {
			f.Set(ptr, vl)
		} else if vl, ok := v.(*int); ok {
			val := int16(*vl)
			f.Set(ptr, &val)
		} else if vl, ok := v.(*int32); ok {
			val := int16(*vl)
			f.Set(ptr, &val)
		} else if vl, ok := v.(*int64); ok {
			val := int16(*vl)
			f.Set(ptr, &val)
		} else {
			printError("int16", v)
		}
	},
	{"int8", 0, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int8); ok {
			f.SetInt8(ptr, *vl)
		} else if vl, ok := v.(*int); ok {
			f.SetInt8(ptr, int8(*vl))
		} else if vl, ok := v.(*int32); ok {
			f.SetInt8(ptr, int8(*vl))
		} else if vl, ok := v.(*int64); ok {
			f.SetInt8(ptr, int8(*vl))
		} else {
			printError("int8", v)
		}
	},
	{"int8", 1, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int8); ok {
			f.Set(ptr, vl)
		} else if vl, ok := v.(*int); ok {
			val := int8(*vl)
			f.Set(ptr, &val)
		} else if vl, ok := v.(*int32); ok {
			val := int8(*vl)
			f.Set(ptr, &val)
		} else if vl, ok := v.(*int64); ok {
			val := int8(*vl)
			f.Set(ptr, &val)
		} else {
			printError("int8", v)
		}
	},
	{"int64", 0, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int64); ok {
			f.SetInt64(ptr, *vl)
		} else if vl, ok := v.(*int); ok {
			f.SetInt64(ptr, int64(*vl))
		} else if vl, ok := v.(*int32); ok {
			f.SetInt64(ptr, int64(*vl))
		} else {
			printError("int64", v)
		}
	},
	{"int64", 1, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*int64); ok {
			f.Set(ptr, vl)
		} else if vl, ok := v.(*int); ok {
			val := int64(*vl)
			f.Set(ptr, &val)
		} else if vl, ok := v.(*int32); ok {
			val := int64(*vl)
			f.Set(ptr, &val)
		} else {
			printError("int64", v)
		}
	},
	{"float32", 0, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*float32); ok {
			f.SetFloat32(ptr, *vl)
		} else if vl, ok := v.(*float64); ok {
			f.SetFloat32(ptr, float32(*vl))
		} else {
			printError("float32", v)
		}
	},
	{"float32", 1, 0}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*float32); ok {
			f.Set(ptr, vl)
		} else if vl, ok := v.(*float64); ok {
			val := new(float32)
			*val = float32(*vl)
			f.Set(ptr, val)
		} else {
			printError("float32", v)
		}
	},
	{"int", 0, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlice[int](f, ptr, &vl)
		} else if vl, ok := v.(*[]int32); ok {
			setReflectIntSlice[int](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlice[int](f, ptr, &vl)
		} else if vl, ok := v.(*[]int64); ok {
			setReflectIntSlice[int](f, ptr, vl)
		} else if vl, ok := v.([]int64); ok {
			setReflectIntSlice[int](f, ptr, &vl)
		} else {
			printError("[]int", v)
		}
	},
	{"int", 1, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) { // Pointer
		if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlicePointer[int](f, ptr, &vl)
		} else if vl, ok := v.(*[]int32); ok {
			setReflectIntSlicePointer[int](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlicePointer[int](f, ptr, &vl)
		} else if vl, ok := v.(*[]int64); ok {
			setReflectIntSlicePointer[int](f, ptr, vl)
		} else if vl, ok := v.([]int64); ok {
			setReflectIntSlicePointer[int](f, ptr, &vl)
		} else {
			printError("[]int", v)
		}
	},
	// SLICE
	{"int32", 0, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*[]int32); ok {
			setReflectIntSlice[int32](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlice[int32](f, ptr, &vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int32](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlice[int32](f, ptr, &vl)
		} else if vl, ok := v.(*[]int64); ok {
			setReflectIntSlice[int32](f, ptr, vl)
		} else if vl, ok := v.([]int64); ok {
			setReflectIntSlice[int32](f, ptr, &vl)
		} else {
			printError("[]int32", v)
		}
	},
	{"int32", 1, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) { // Pointer
		if vl, ok := v.(*[]int32); ok {
			setReflectIntSlicePointer[int32](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlicePointer[int32](f, ptr, &vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int32](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlicePointer[int32](f, ptr, &vl)
		} else if vl, ok := v.(*[]int64); ok {
			setReflectIntSlicePointer[int32](f, ptr, vl)
		} else if vl, ok := v.([]int64); ok {
			setReflectIntSlicePointer[int32](f, ptr, &vl)
		} else {
			printError("[]int32", v)
		}
	},
	{"int16", 0, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*[]int16); ok {
			setReflectIntSlice[int16](f, ptr, vl)
		} else if vl, ok := v.([]int16); ok {
			setReflectIntSlice[int16](f, ptr, &vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int16](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlice[int16](f, ptr, &vl)
		} else if vl, ok := v.(*[]int32); ok {
			setReflectIntSlice[int16](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlice[int16](f, ptr, &vl)
		} else {
			printError("[]int16", v)
		}
	},
	{"int16", 1, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) { // Pointer
		if vl, ok := v.(*[]int16); ok {
			setReflectIntSlicePointer[int16](f, ptr, vl)
		} else if vl, ok := v.([]int16); ok {
			setReflectIntSlicePointer[int16](f, ptr, &vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int16](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlicePointer[int16](f, ptr, &vl)
		} else if vl, ok := v.(*[]int32); ok {
			setReflectIntSlicePointer[int16](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlicePointer[int16](f, ptr, &vl)
		} else {
			printError("[]int16", v)
		}
	},
	{"int8", 0, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*[]int8); ok {
			setReflectIntSlice[int8](f, ptr, vl)
		} else if vl, ok := v.([]int8); ok {
			setReflectIntSlice[int8](f, ptr, &vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int8](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlice[int8](f, ptr, &vl)
		} else if vl, ok := v.(*[]int32); ok {
			setReflectIntSlice[int8](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlice[int8](f, ptr, &vl)
		} else {
			printError("[]int8", v)
		}
	},
	{"int8", 1, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) { // Pointer
		if vl, ok := v.(*[]int8); ok {
			setReflectIntSlicePointer[int8](f, ptr, vl)
		} else if vl, ok := v.([]int8); ok {
			setReflectIntSlicePointer[int8](f, ptr, &vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int8](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlicePointer[int8](f, ptr, &vl)
		} else if vl, ok := v.(*[]int32); ok {
			setReflectIntSlicePointer[int8](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlicePointer[int8](f, ptr, &vl)
		} else {
			printError("[]int8", v)
		}
	},
	{"int64", 0, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*[]int64); ok {
			setReflectIntSlice[int64](f, ptr, vl)
		} else if vl, ok := v.([]int64); ok {
			setReflectIntSlice[int64](f, ptr, &vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlice[int64](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlice[int64](f, ptr, &vl)
		} else if vl, ok := v.(*[]int32); ok {
			setReflectIntSlice[int64](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlice[int64](f, ptr, &vl)
		} else {
			printError("[]int64", v)
		}
	},
	{"int64", 1, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) { // Pointer
		if vl, ok := v.(*[]int64); ok {
			setReflectIntSlicePointer[int64](f, ptr, vl)
		} else if vl, ok := v.([]int64); ok {
			setReflectIntSlicePointer[int64](f, ptr, &vl)
		} else if vl, ok := v.(*[]int); ok {
			setReflectIntSlicePointer[int64](f, ptr, vl)
		} else if vl, ok := v.([]int); ok {
			setReflectIntSlicePointer[int64](f, ptr, &vl)
		} else if vl, ok := v.(*[]int32); ok {
			setReflectIntSlicePointer[int64](f, ptr, vl)
		} else if vl, ok := v.([]int32); ok {
			setReflectIntSlicePointer[int64](f, ptr, &vl)
		} else {
			printError("[]int64", v)
		}
	},
	{"string", 0, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) {
		if vl, ok := v.(*[]string); ok {
			setField(f, ptr, slices.Clone(*vl))
		} else if vl, ok := v.([]string); ok {
			setField(f, ptr, slices.Clone(vl))
		}
	},
	{"string", 1, 1}: func(f *xunsafe.Field, ptr unsafe.Pointer, v any) { // Pointer
		if vl, ok := v.(*[]string); ok {
			val := slices.Clone(*vl)
			setField(f, ptr, &val)
		} else if vl, ok := v.([]string); ok {
			val := slices.Clone(vl)
			setField(f, ptr, &val)
		}
	},
}
