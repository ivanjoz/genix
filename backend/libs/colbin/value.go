package colbin

import (
	"reflect"
	"unsafe"
)

// Scalar get/set helpers routed through xunsafe typed accessors (no interface
// boxing on the hot numeric path). readInt64/setInt64 normalize every integer
// and bool kind to int64, the encoder's internal column representation.

func readInt64(fm *fieldMeta, p unsafe.Pointer) int64 {
	switch fm.goKind {
	case reflect.Int8:
		return int64(fm.xf.Int8(p))
	case reflect.Int16:
		return int64(fm.xf.Int16(p))
	case reflect.Int32:
		return int64(fm.xf.Int32(p))
	case reflect.Int64:
		return fm.xf.Int64(p)
	case reflect.Int:
		return int64(fm.xf.Int(p))
	case reflect.Uint8:
		return int64(fm.xf.Uint8(p))
	case reflect.Uint16:
		return int64(fm.xf.Uint16(p))
	case reflect.Uint32:
		return int64(fm.xf.Uint32(p))
	case reflect.Uint64:
		return int64(fm.xf.Uint64(p))
	case reflect.Uint:
		return int64(fm.xf.Uint(p))
	case reflect.Bool:
		if fm.xf.Bool(p) {
			return 1
		}
	}
	return 0
}

func setInt64(fm *fieldMeta, p unsafe.Pointer, v int64) {
	switch fm.goKind {
	case reflect.Int8:
		fm.xf.SetInt8(p, int8(v))
	case reflect.Int16:
		fm.xf.SetInt16(p, int16(v))
	case reflect.Int32:
		fm.xf.SetInt32(p, int32(v))
	case reflect.Int64:
		fm.xf.SetInt64(p, v)
	case reflect.Int:
		fm.xf.SetInt(p, int(v))
	case reflect.Uint8:
		fm.xf.SetUint8(p, uint8(v))
	case reflect.Uint16:
		fm.xf.SetUint16(p, uint16(v))
	case reflect.Uint32:
		fm.xf.SetUint32(p, uint32(v))
	case reflect.Uint64:
		fm.xf.SetUint64(p, uint64(v))
	case reflect.Uint:
		fm.xf.SetUint(p, uint(v))
	case reflect.Bool:
		fm.xf.SetBool(p, v != 0)
	}
}

func readFloat64(fm *fieldMeta, p unsafe.Pointer) float64 {
	if fm.goKind == reflect.Float32 {
		return float64(fm.xf.Float32(p))
	}
	return fm.xf.Float64(p)
}

func setFloat64(fm *fieldMeta, p unsafe.Pointer, v float64) {
	if fm.goKind == reflect.Float32 {
		fm.xf.SetFloat32(p, float32(v))
		return
	}
	fm.xf.SetFloat64(p, v)
}
