package colbin

import (
	"reflect"
	"unsafe"
)

// Array elements are not struct fields, so xunsafe.Field cannot address them.
// These helpers read/write a scalar directly at its value pointer, by kind —
// the same operation xunsafe performs internally, just on a bare element.

// sliceHeader mirrors reflect's slice layout so we can read/write a slice field
// via its value pointer. Writing a header whose Data comes from reflect.MakeSlice
// is GC-safe: the field is typed as a slice, so the collector keeps the backing
// array reachable once the header is stored.
type sliceHeader struct {
	data unsafe.Pointer
	len  int
	cap  int
}

func readInt64At(goKind reflect.Kind, p unsafe.Pointer) int64 {
	switch goKind {
	case reflect.Int8:
		return int64(*(*int8)(p))
	case reflect.Int16:
		return int64(*(*int16)(p))
	case reflect.Int32:
		return int64(*(*int32)(p))
	case reflect.Int64:
		return *(*int64)(p)
	case reflect.Int:
		return int64(*(*int)(p))
	case reflect.Uint8:
		return int64(*(*uint8)(p))
	case reflect.Uint16:
		return int64(*(*uint16)(p))
	case reflect.Uint32:
		return int64(*(*uint32)(p))
	case reflect.Uint64:
		return int64(*(*uint64)(p))
	case reflect.Uint:
		return int64(*(*uint)(p))
	case reflect.Bool:
		if *(*bool)(p) {
			return 1
		}
	}
	return 0
}

func setInt64At(goKind reflect.Kind, p unsafe.Pointer, v int64) {
	switch goKind {
	case reflect.Int8:
		*(*int8)(p) = int8(v)
	case reflect.Int16:
		*(*int16)(p) = int16(v)
	case reflect.Int32:
		*(*int32)(p) = int32(v)
	case reflect.Int64:
		*(*int64)(p) = v
	case reflect.Int:
		*(*int)(p) = int(v)
	case reflect.Uint8:
		*(*uint8)(p) = uint8(v)
	case reflect.Uint16:
		*(*uint16)(p) = uint16(v)
	case reflect.Uint32:
		*(*uint32)(p) = uint32(v)
	case reflect.Uint64:
		*(*uint64)(p) = uint64(v)
	case reflect.Uint:
		*(*uint)(p) = uint(v)
	case reflect.Bool:
		*(*bool)(p) = v != 0
	}
}

func readFloat64At(goKind reflect.Kind, p unsafe.Pointer) float64 {
	if goKind == reflect.Float32 {
		return float64(*(*float32)(p))
	}
	return *(*float64)(p)
}

func setFloat64At(goKind reflect.Kind, p unsafe.Pointer, v float64) {
	if goKind == reflect.Float32 {
		*(*float32)(p) = float32(v)
		return
	}
	*(*float64)(p) = v
}
