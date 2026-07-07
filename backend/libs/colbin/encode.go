package colbin

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

// Marshal encodes a slice of structs (or a single struct, treated as N=1) into
// the columnar delta format. Pointers are dereferenced. Decode with Unmarshal
// into the same Go type.
func Marshal(v any) (out []byte, err error) {
	// The any encoder panics with encodeError for dynamic types it can't represent
	// (interface{}-held struct/chan/func, non-string map keys); recover into an error.
	defer func() {
		if r := recover(); r != nil {
			if ce, ok := r.(encodeError); ok {
				out, err = nil, ce.err
			} else {
				panic(r)
			}
		}
	}()
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("colbin: Marshal nil pointer")
		}
		rv = rv.Elem()
	}

	// Non-record top-level types (maps, []*struct, scalars, …) use value mode: a
	// single N=1 element column reusing the element machinery. struct / []struct
	// keep the columnar records layout below.
	if !topLevelIsRecords(rv.Type()) {
		out = append(out, formatVersion)
		return encodeValueMode(out, rv), nil
	}

	// Resolve record element type and a pointer to each record.
	var elemType reflect.Type
	var recordPtrs []unsafe.Pointer
	switch rv.Kind() {
	case reflect.Slice:
		elemType = rv.Type().Elem()
		n := rv.Len()
		recordPtrs = make([]unsafe.Pointer, n)
		base := rv.UnsafePointer() // &elem[0]
		size := elemType.Size()
		for i := 0; i < n; i++ {
			recordPtrs[i] = unsafe.Add(base, uintptr(i)*size)
		}
	case reflect.Struct:
		elemType = rv.Type()
		// Non-addressable value: copy into an addressable location to take its pointer.
		cp := reflect.New(elemType)
		cp.Elem().Set(rv)
		recordPtrs = []unsafe.Pointer{cp.UnsafePointer()}
	default:
		return nil, fmt.Errorf("colbin: Marshal expects struct or slice of structs, got %s", rv.Kind())
	}

	ti, err := getTypeInfo(elemType)
	if err != nil {
		return nil, err
	}

	out = make([]byte, 0, 16+len(recordPtrs)*len(ti.fields))
	out = append(out, formatVersion)
	out = binary.AppendUvarint(out, uint64(len(recordPtrs)))
	out = encodeSubTable(out, ti, recordPtrs)
	return out, nil
}

// encodeSubTable writes [colCount] then every field as [id][column]. Used at the
// top level and recursively for nested structs (struct fields / struct elements).
func encodeSubTable(out []byte, ti *typeInfo, ptrs []unsafe.Pointer) []byte {
	out = append(out, byte(len(ti.fields)))
	for i := range ti.fields {
		fm := &ti.fields[i]
		out = append(out, fm.id)
		out = encodeColumn(out, fm, ptrs)
	}
	return out
}

// encodeColumn appends a column for a STRUCT FIELD: a flags byte + payload.
// ptrs point at the containing struct; the value sits at ptr+fm.offset.
func encodeColumn(out []byte, fm *fieldMeta, ptrs []unsafe.Pointer) []byte {
	if fm.nullable { // pointer field: null wrapper + dense value column
		return appendNullableColumn(out, fm.elem, offsetPtrs(ptrs, fm.offset))
	}
	switch fm.fType {
	case ftInt:
		buf := getI64(len(ptrs))
		for i, p := range ptrs {
			(*buf)[i] = readInt64(fm, p)
		}
		out = appendIntColumn(out, *buf, fm.intWidth)
		putI64(buf)
		return out
	case ftFloat:
		buf := getF64(len(ptrs))
		for i, p := range ptrs {
			(*buf)[i] = readFloat64(fm, p)
		}
		out = appendFloatColumn(out, *buf, fm.intWidth) // intWidth holds 32/64 here
		putF64(buf)
		return out
	case ftString:
		out = append(out, ftString)
		buf := getBlobs(len(ptrs))
		for i, p := range ptrs {
			s := fm.xf.String(p)
			(*buf)[i] = unsafe.Slice(unsafe.StringData(s), len(s))
		}
		out = appendBlobColumn(out, *buf)
		putBlobs(buf)
		return out
	case ftBytes:
		out = append(out, ftBytes)
		buf := getBlobs(len(ptrs))
		for i, p := range ptrs {
			(*buf)[i] = fm.xf.Bytes(p)
		}
		out = appendBlobColumn(out, *buf)
		putBlobs(buf)
		return out
	case ftStruct:
		out = append(out, ftStruct)
		childPtrs := make([]unsafe.Pointer, len(ptrs))
		for i, p := range ptrs {
			childPtrs[i] = unsafe.Add(p, fm.offset) // nested struct sits inline
		}
		return encodeSubTable(out, fm.sub, childPtrs)
	case ftArray:
		out = append(out, ftArray)
		shPtrs := make([]unsafe.Pointer, len(ptrs))
		for i, p := range ptrs {
			shPtrs[i] = unsafe.Add(p, fm.offset) // -> slice header of this record
		}
		return encodeArrayBody(out, fm.elem, fm.elemSize, shPtrs)
	case ftMap:
		out = append(out, ftMap)
		return encodeMapColumn(out, fm, offsetPtrs(ptrs, fm.offset))
	case ftAny:
		out = append(out, ftAny)
		return encodeAnyColumn(out, fm, offsetPtrs(ptrs, fm.offset))
	}
	return out
}

// offsetPtrs returns ptr+offset for each pointer (locate a field within its struct).
func offsetPtrs(ptrs []unsafe.Pointer, offset uintptr) []unsafe.Pointer {
	out := make([]unsafe.Pointer, len(ptrs))
	for i, p := range ptrs {
		out[i] = unsafe.Add(p, offset)
	}
	return out
}

// encodeArrayBody writes an array column: a per-record length sub-column, then
// the flattened element values as one nested element column. shPtrs point at the
// slice headers (one per record).
func encodeArrayBody(out []byte, elem *fieldMeta, elemSize uintptr, shPtrs []unsafe.Pointer) []byte {
	lenBuf := getI64(len(shPtrs))
	total := 0
	for i, sp := range shPtrs {
		sh := (*sliceHeader)(sp)
		(*lenBuf)[i] = int64(sh.len)
		total += sh.len
	}
	out = appendIntColumn(out, *lenBuf, 32)
	putI64(lenBuf)

	elemPtrs := make([]unsafe.Pointer, 0, total) // value pointer of every element, flattened
	for _, sp := range shPtrs {
		sh := (*sliceHeader)(sp)
		for j := 0; j < sh.len; j++ {
			elemPtrs = append(elemPtrs, unsafe.Add(sh.data, uintptr(j)*elemSize))
		}
	}
	return encodeElemColumn(out, elem, elemPtrs)
}

// encodeElemColumn appends a column for ARRAY ELEMENTS: ptrs point directly at
// each value (no struct offset). Scalars use direct casts; struct/array elements
// recurse.
func encodeElemColumn(out []byte, elem *fieldMeta, ptrs []unsafe.Pointer) []byte {
	if elem.nullable { // ptrs point at *T slots (e.g. []*T or map[K]*V)
		return appendNullableColumn(out, elem.elem, ptrs)
	}
	switch elem.fType {
	case ftInt:
		buf := getI64(len(ptrs))
		for i, p := range ptrs {
			(*buf)[i] = readInt64At(elem.goKind, p)
		}
		out = appendIntColumn(out, *buf, elem.intWidth)
		putI64(buf)
		return out
	case ftFloat:
		buf := getF64(len(ptrs))
		for i, p := range ptrs {
			(*buf)[i] = readFloat64At(elem.goKind, p)
		}
		out = appendFloatColumn(out, *buf, elem.intWidth)
		putF64(buf)
		return out
	case ftString:
		out = append(out, ftString)
		buf := getBlobs(len(ptrs))
		for i, p := range ptrs {
			s := *(*string)(p)
			(*buf)[i] = unsafe.Slice(unsafe.StringData(s), len(s))
		}
		out = appendBlobColumn(out, *buf)
		putBlobs(buf)
		return out
	case ftBytes:
		out = append(out, ftBytes)
		buf := getBlobs(len(ptrs))
		for i, p := range ptrs {
			(*buf)[i] = *(*[]byte)(p)
		}
		out = appendBlobColumn(out, *buf)
		putBlobs(buf)
		return out
	case ftStruct:
		out = append(out, ftStruct)
		return encodeSubTable(out, elem.sub, ptrs)
	case ftArray:
		out = append(out, ftArray)
		return encodeArrayBody(out, elem.elem, elem.elemSize, ptrs) // ptrs already at slice headers
	case ftMap:
		out = append(out, ftMap)
		return encodeMapColumn(out, elem, ptrs) // ptrs already at map slots
	case ftAny:
		out = append(out, ftAny)
		return encodeAnyColumn(out, elem, ptrs) // ptrs already at interface slots
	}
	return out
}

// appendFloatColumn stores raw IEEE-754 values (no delta) straight onto out.
// Empty (all-zero) columns carry only the flags byte. The precision bit records
// 32 vs 64 (the decoder also knows from the Go type).
func appendFloatColumn(out []byte, vals []float64, width uint8) []byte {
	empty := true
	for _, v := range vals {
		if v != 0 {
			empty = false
			break
		}
	}
	var prec uint8
	if width == 64 {
		prec = 1
	}
	out = append(out, ftFloat|prec<<4|boolBit(empty, 7))
	if empty {
		return out
	}
	bw := bitWriter{buf: out}
	for _, v := range vals {
		if width == 64 {
			bw.writeBits(math.Float64bits(v), 64)
		} else {
			bw.writeBits(uint64(math.Float32bits(float32(v))), 32)
		}
	}
	return bw.flush()
}

// appendBlobColumn writes a 32-bit-base length sub-column then concatenated bytes.
func appendBlobColumn(out []byte, blobs [][]byte) []byte {
	lenBuf := getI64(len(blobs))
	for i, b := range blobs {
		(*lenBuf)[i] = int64(len(b))
	}
	out = appendIntColumn(out, *lenBuf, 32)
	putI64(lenBuf)
	for _, b := range blobs {
		out = append(out, b...)
	}
	return out
}

// encodeValueMode encodes a single top-level value (map, []*struct, scalar, …) as
// one element column with N=1. The value is copied into an addressable cell so the
// element encoders can take its pointer. describeType errors surface as encodeError
// (recovered by Marshal), matching the any encoder's panic contract.
func encodeValueMode(out []byte, rv reflect.Value) []byte {
	fm, err := describeType(rv.Type())
	if err != nil {
		panic(encodeError{err})
	}
	cell := reflect.New(rv.Type()) // addressable copy to take &value
	cell.Elem().Set(rv)
	return encodeElemColumn(out, &fm, []unsafe.Pointer{cell.UnsafePointer()})
}

// boolBit returns 1<<shift if set, else 0 — for packing flag bits.
func boolBit(set bool, shift uint8) byte {
	if set {
		return 1 << shift
	}
	return 0
}
