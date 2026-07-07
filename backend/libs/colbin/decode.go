package colbin

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

// decoder walks the byte stream with an explicit cursor; each column computes
// its own byte span so the cursor can advance to the next column.
type decoder struct {
	data []byte
	pos  int
}

// Unmarshal decodes a colbin message into dst, which must be a pointer to a slice
// of structs or a pointer to a struct (for an N==1 message).
func Unmarshal(data []byte, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("colbin: Unmarshal needs a non-nil pointer")
	}
	dec := &decoder{data: data}
	if dec.readByte() != formatVersion {
		return fmt.Errorf("colbin: bad version byte")
	}
	n64, m := binary.Uvarint(dec.data[dec.pos:])
	if m <= 0 {
		return fmt.Errorf("colbin: bad record count")
	}
	dec.pos += m
	n := int(n64)

	target := rv.Elem()
	var elemType reflect.Type
	var recordPtrs []unsafe.Pointer
	var backing reflect.Value // kept alive so the backing array survives

	switch target.Kind() {
	case reflect.Slice:
		elemType = target.Type().Elem()
		backing = reflect.MakeSlice(target.Type(), n, n)
		base := backing.UnsafePointer()
		size := elemType.Size()
		recordPtrs = make([]unsafe.Pointer, n)
		for i := 0; i < n; i++ {
			recordPtrs[i] = unsafe.Add(base, uintptr(i)*size)
		}
	case reflect.Struct:
		if n != 1 {
			return fmt.Errorf("colbin: message has %d records, cannot decode into a single struct", n)
		}
		elemType = target.Type()
		recordPtrs = []unsafe.Pointer{target.Addr().UnsafePointer()}
	default:
		return fmt.Errorf("colbin: Unmarshal target must be *slice or *struct, got %s", target.Kind())
	}

	ti, err := getTypeInfo(elemType)
	if err != nil {
		return err
	}
	if err := dec.decodeSubTable(ti, n, recordPtrs); err != nil {
		return err
	}
	if target.Kind() == reflect.Slice {
		target.Set(backing)
	}
	return nil
}

// decodeSubTable reads [colCount] then each [id][column] into the given records.
func (dec *decoder) decodeSubTable(ti *typeInfo, n int, ptrs []unsafe.Pointer) error {
	colCount := int(dec.readByte())
	for c := 0; c < colCount; c++ {
		id := dec.readByte()
		fm := ti.byID[id]
		if fm == nil {
			return fmt.Errorf("colbin: unknown field id %d (schema mismatch)", id)
		}
		if err := dec.decodeColumn(fm, n, ptrs); err != nil {
			return err
		}
	}
	return nil
}

func (dec *decoder) readByte() byte {
	b := dec.data[dec.pos]
	dec.pos++
	return b
}

// decodeColumn reads one column into STRUCT FIELDS (value at ptr+fm.offset).
func (dec *decoder) decodeColumn(fm *fieldMeta, n int, ptrs []unsafe.Pointer) error {
	if fm.nullable {
		return dec.decodeNullableColumn(fm.elem, fm.pointeeType, n, offsetPtrs(ptrs, fm.offset))
	}
	switch fm.fType {
	case ftInt:
		vals := dec.readIntColumn(n, fm.intWidth)
		for i, p := range ptrs {
			setInt64(fm, p, vals[i])
		}
	case ftFloat:
		vals := dec.readFloatColumn(n)
		for i, p := range ptrs {
			setFloat64(fm, p, vals[i])
		}
	case ftString, ftBytes:
		dec.readByte() // top flags byte (carries only the type)
		blobs := dec.readBlobs(n)
		for i, p := range ptrs {
			if fm.fType == ftString {
				fm.xf.SetString(p, string(blobs[i])) // copies out of the input buffer
			} else {
				fm.xf.SetBytes(p, cloneBytes(blobs[i]))
			}
		}
	case ftStruct:
		dec.readByte() // flags (ftStruct)
		childPtrs := make([]unsafe.Pointer, len(ptrs))
		for i, p := range ptrs {
			childPtrs[i] = unsafe.Add(p, fm.offset)
		}
		return dec.decodeSubTable(fm.sub, n, childPtrs)
	case ftArray:
		dec.readByte() // flags (ftArray)
		shPtrs := make([]unsafe.Pointer, len(ptrs))
		for i, p := range ptrs {
			shPtrs[i] = unsafe.Add(p, fm.offset)
		}
		return dec.decodeArrayBody(fm.elem, fm.sliceType, fm.elemSize, shPtrs)
	case ftMap:
		return dec.decodeMapColumn(fm, n, offsetPtrs(ptrs, fm.offset))
	}
	return nil
}

// decodeArrayBody reads the length sub-column, allocates each record's slice,
// then decodes the flattened element column into the slices' backing arrays.
// shPtrs point at the slice headers to populate (one per record).
func (dec *decoder) decodeArrayBody(elem *fieldMeta, sliceType reflect.Type, elemSize uintptr, shPtrs []unsafe.Pointer) error {
	lengths := dec.readIntColumn(len(shPtrs), 32)
	total := 0
	elemPtrs := make([]unsafe.Pointer, 0)
	for i, sp := range shPtrs {
		l := int(lengths[i])
		total += l
		if l == 0 {
			continue // leave the slice field as nil (matches Go zero value)
		}
		sv := reflect.MakeSlice(sliceType, l, l)
		// Store the slice header into the field; GC keeps the backing array alive
		// because the field is typed as a slice.
		*(*sliceHeader)(sp) = sliceHeader{data: sv.UnsafePointer(), len: l, cap: l}
		for j := 0; j < l; j++ {
			elemPtrs = append(elemPtrs, unsafe.Add(sv.UnsafePointer(), uintptr(j)*elemSize))
		}
	}
	return dec.decodeElemColumn(elem, sliceType.Elem(), elemSize, total, elemPtrs)
}

// decodeElemColumn reads a column into ARRAY ELEMENTS (ptrs point at values).
func (dec *decoder) decodeElemColumn(elem *fieldMeta, elemType reflect.Type, elemSize uintptr, n int, ptrs []unsafe.Pointer) error {
	if elem.nullable {
		return dec.decodeNullableColumn(elem.elem, elem.pointeeType, n, ptrs)
	}
	switch elem.fType {
	case ftInt:
		vals := dec.readIntColumn(n, elem.intWidth)
		for i, p := range ptrs {
			setInt64At(elem.goKind, p, vals[i])
		}
	case ftFloat:
		vals := dec.readFloatColumn(n)
		for i, p := range ptrs {
			setFloat64At(elem.goKind, p, vals[i])
		}
	case ftString, ftBytes:
		dec.readByte()
		blobs := dec.readBlobs(n)
		for i, p := range ptrs {
			if elem.fType == ftString {
				*(*string)(p) = string(blobs[i])
			} else {
				*(*[]byte)(p) = cloneBytes(blobs[i])
			}
		}
	case ftStruct:
		dec.readByte()
		return dec.decodeSubTable(elem.sub, n, ptrs)
	case ftArray:
		dec.readByte()
		return dec.decodeArrayBody(elem.elem, elemType, elem.elemSize, ptrs)
	case ftMap:
		return dec.decodeMapColumn(elem, n, ptrs)
	}
	return nil
}

// readFloatColumn reads a flags byte + raw IEEE-754 payload; returns n values.
func (dec *decoder) readFloatColumn(n int) []float64 {
	flags := dec.readByte()
	width := uint8(32)
	if flags>>4&7 == 1 {
		width = 64
	}
	out := make([]float64, n)
	if flags>>7&1 == 1 { // empty column
		return out
	}
	span := n * int(width) / 8
	br := bitReader{buf: dec.data[dec.pos : dec.pos+span]}
	dec.pos += span
	for i := 0; i < n; i++ {
		if width == 64 {
			out[i] = math.Float64frombits(br.readBits(64))
		} else {
			out[i] = float64(math.Float32frombits(uint32(br.readBits(32))))
		}
	}
	return out
}

// readBlobs reads a length sub-column + concatenated bytes, returning n raw
// slices that alias the input buffer (callers copy as needed).
func (dec *decoder) readBlobs(n int) [][]byte {
	lengths := dec.readIntColumn(n, 32)
	out := make([][]byte, n)
	for i := 0; i < n; i++ {
		l := int(lengths[i])
		out[i] = dec.data[dec.pos : dec.pos+l]
		dec.pos += l
	}
	return out
}

// cloneBytes copies a slice so decoded []byte fields don't alias the input.
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// readIntColumn reads a flags byte + packed int payload and returns n decoded values.
func (dec *decoder) readIntColumn(n int, nativeWidth uint8) []int64 {
	flags := dec.readByte()
	isSigned := flags>>3&1 == 1
	prec := flags >> 4 & 7
	empty := flags>>7&1 == 1
	span := intColumnBytes(n, nativeWidth, prec, empty)
	br := bitReader{buf: dec.data[dec.pos : dec.pos+span]}
	dec.pos += span
	out := make([]int64, n)
	decodeIntColumn(&br, n, nativeWidth, isSigned, empty, prec, out)
	return out
}

// intColumnBytes returns the byte span of a packed int column (base + n deltas).
func intColumnBytes(n int, nativeWidth, prec uint8, empty bool) int {
	if empty {
		return 0
	}
	totalBits := int(nativeWidth) + n*int(intWidths[prec])
	return (totalBits + 7) / 8
}
