package colbin

import (
	"reflect"
	"unsafe"
)

// Phase C: nullability (pointers) and maps.
//
// Nullable column wire layout (in place of the usual [flags][payload]):
//   [nullFlags:1] [presence bitmap: ceil(N/8) bytes IF has_nulls] innerElementColumn
// The inner column holds only the non-null values (dense). The decoder knows a
// field is nullable from its Go type, so no extra type byte is needed.
//
// Map column wire layout:
//   [flags:1=ftMap] lengthColumn keysColumn valuesColumn
// where lengthColumn is the per-record entry count, and keys/values are flattened
// element columns over every entry across all records.

// --- encode ---

// appendNullableColumn writes the null wrapper then the dense value column.
// slotPtrs point at the *T pointer slots (struct field or array element).
func appendNullableColumn(out []byte, pointee *fieldMeta, slotPtrs []unsafe.Pointer) []byte {
	hasNulls := false
	for _, s := range slotPtrs {
		if *(*unsafe.Pointer)(s) == nil {
			hasNulls = true
			break
		}
	}
	present := make([]unsafe.Pointer, 0, len(slotPtrs)) // value pointers of non-nil entries
	if !hasNulls {
		out = append(out, 0) // nullFlags: no nulls, no bitmap
		for _, s := range slotPtrs {
			present = append(present, *(*unsafe.Pointer)(s))
		}
	} else {
		out = append(out, 1) // nullFlags: bitmap follows
		bitmap := make([]byte, (len(slotPtrs)+7)/8)
		for i, s := range slotPtrs {
			if pv := *(*unsafe.Pointer)(s); pv != nil {
				bitmap[i>>3] |= 1 << (uint(i) & 7) // 1 == present
				present = append(present, pv)
			}
		}
		out = append(out, bitmap...)
	}
	return encodeElemColumn(out, pointee, present)
}

// encodeMapColumn flattens all records' maps into a length column + keys + values.
// slotPtrs point at the map header of each record.
func encodeMapColumn(out []byte, fm *fieldMeta, slotPtrs []unsafe.Pointer) []byte {
	lenBuf := getI64(len(slotPtrs))
	total := 0
	maps := make([]reflect.Value, len(slotPtrs))
	for i, sp := range slotPtrs {
		mv := reflect.NewAt(fm.mapType, sp).Elem()
		maps[i] = mv
		(*lenBuf)[i] = int64(mv.Len())
		total += mv.Len()
	}
	out = appendIntColumn(out, *lenBuf, 32)
	putI64(lenBuf)

	// Copy entries into addressable backing slices (map keys/values aren't
	// addressable, so the pointer-based element encoders can't read them directly).
	keys := reflect.MakeSlice(reflect.SliceOf(fm.mapKeyType), total, total)
	vals := reflect.MakeSlice(reflect.SliceOf(fm.mapValType), total, total)
	idx := 0
	for _, mv := range maps {
		it := mv.MapRange()
		for it.Next() {
			keys.Index(idx).Set(it.Key())
			vals.Index(idx).Set(it.Value())
			idx++
		}
	}
	out = encodeElemColumn(out, fm.mapKey, elemPointers(keys, fm.mapKeyType, total))
	out = encodeElemColumn(out, fm.mapVal, elemPointers(vals, fm.mapValType, total))
	return out
}

// elemPointers returns a value pointer to each element of a backing slice.
func elemPointers(backing reflect.Value, elemType reflect.Type, n int) []unsafe.Pointer {
	base := backing.UnsafePointer()
	size := elemType.Size()
	ptrs := make([]unsafe.Pointer, n)
	for i := 0; i < n; i++ {
		ptrs[i] = unsafe.Add(base, uintptr(i)*size)
	}
	return ptrs
}

// --- decode ---

// decodeNullableColumn reverses appendNullableColumn, allocating one backing slice
// of the pointee type for the present slots (kept alive by the typed *T slots).
func (dec *decoder) decodeNullableColumn(pointee *fieldMeta, pointeeType reflect.Type, n int, slotPtrs []unsafe.Pointer) error {
	hasNulls := dec.readByte()&1 == 1
	var present []bool
	numPresent := n
	if hasNulls {
		bmBytes := (n + 7) / 8
		bitmap := dec.data[dec.pos : dec.pos+bmBytes]
		dec.pos += bmBytes
		present = make([]bool, n)
		numPresent = 0
		for i := 0; i < n; i++ {
			if bitmap[i>>3]>>(uint(i)&7)&1 == 1 {
				present[i] = true
				numPresent++
			}
		}
	}
	backing := reflect.MakeSlice(reflect.SliceOf(pointeeType), numPresent, numPresent)
	base := backing.UnsafePointer()
	size := pointeeType.Size()
	presentPtrs := make([]unsafe.Pointer, numPresent)
	k := 0
	for i := 0; i < n; i++ {
		if hasNulls && !present[i] {
			continue // leave the slot nil
		}
		vp := unsafe.Add(base, uintptr(k)*size)
		*(*unsafe.Pointer)(slotPtrs[i]) = vp // point the field/element at the value
		presentPtrs[k] = vp
		k++
	}
	return dec.decodeElemColumn(pointee, pointeeType, size, numPresent, presentPtrs)
}

// decodeMapColumn reverses encodeMapColumn.
func (dec *decoder) decodeMapColumn(fm *fieldMeta, n int, slotPtrs []unsafe.Pointer) error {
	dec.readByte() // ftMap flags
	lengths := dec.readIntColumn(n, 32)
	total := 0
	for _, l := range lengths {
		total += int(l)
	}
	keys := reflect.MakeSlice(reflect.SliceOf(fm.mapKeyType), total, total)
	if err := dec.decodeElemColumn(fm.mapKey, fm.mapKeyType, fm.mapKeyType.Size(), total,
		elemPointers(keys, fm.mapKeyType, total)); err != nil {
		return err
	}
	vals := reflect.MakeSlice(reflect.SliceOf(fm.mapValType), total, total)
	if err := dec.decodeElemColumn(fm.mapVal, fm.mapValType, fm.mapValType.Size(), total,
		elemPointers(vals, fm.mapValType, total)); err != nil {
		return err
	}
	idx := 0
	for i := 0; i < n; i++ {
		l := int(lengths[i])
		if l == 0 {
			continue // leave the map field nil (matches Go zero value)
		}
		m := reflect.MakeMapWithSize(fm.mapType, l)
		for j := 0; j < l; j++ {
			m.SetMapIndex(keys.Index(idx), vals.Index(idx))
			idx++
		}
		reflect.NewAt(fm.mapType, slotPtrs[i]).Elem().Set(m)
	}
	return nil
}
