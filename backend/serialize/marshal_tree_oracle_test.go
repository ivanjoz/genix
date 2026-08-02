package serialize

// Reference (oracle) implementation: the original tree-based encoder.
//
// It builds a []any mirror of the payload and lets sonic render it. That is exactly what the
// production path did before the direct-to-bytes encoder in append.go replaced it, so it is kept
// here — compiled only into the test binary, never into the app — as the oracle for
// TestDirectEncoderMatchesTreeEncoder. If a change to append.go alters what goes on the wire,
// that test fails against this.
//
// Do not "fix" anything here: its value is being the old behaviour, byte for byte.

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/bytedance/sonic"
)

// marshalTree renders v the old way: build the tree, then hand it to sonic. The keys header is
// produced by the same buildKeysList the production path uses, so a diff between this and
// Marshal isolates the content rendering.
func marshalTree(v any) ([]byte, error) {
	e := NewEncoder()

	content, err := e.marshalContent(v)
	if err != nil {
		return nil, err
	}

	keys := e.buildKeysList()
	e.learnFieldOrders()

	return sonic.Marshal([]any{keys, content})
}

func (e *Encoder) marshalContent(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	val := reflect.ValueOf(v)
	for (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && !val.IsNil() {
		if val.Kind() == reflect.Ptr {
			ptr := val.Pointer()
			if e.seen[ptr] {
				return nil, nil // Cycle detected, break recursion
			}
			e.seen[ptr] = true
			defer func(p uintptr) { delete(e.seen, p) }(ptr)
		}
		val = val.Elem()
	}

	if (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && val.IsNil() {
		return nil, nil
	}

	switch val.Kind() {
	case reflect.Struct:
		return e.marshalStruct(val)
	case reflect.Slice, reflect.Array:
		return e.marshalSlice(val)
	case reflect.Map:
		return e.marshalMap(val)
	default:
		// Fallback for other types
		if !val.IsValid() {
			return nil, nil
		}
		return val.Interface(), nil
	}
}

func (e *Encoder) marshalMap(val reflect.Value) ([]any, error) {
	result := []any{3}

	// Get all keys and sort them for consistent two-pass ordering
	mapKeys := val.MapKeys()
	keysWithStr := make([]struct {
		val reflect.Value
		str string
	}, len(mapKeys))

	for i, k := range mapKeys {
		keysWithStr[i] = struct {
			val reflect.Value
			str string
		}{k, fmt.Sprintf("%v", k.Interface())}
	}

	sort.Slice(keysWithStr, func(i, j int) bool {
		return keysWithStr[i].str < keysWithStr[j].str
	})

	for _, ks := range keysWithStr {
		v := val.MapIndex(ks.val)
		marshaledVal, err := e.marshalValue(v.Interface())
		if err != nil {
			return nil, err
		}
		result = append(result, ks.str, marshaledVal)
	}
	return result, nil
}

func (e *Encoder) marshalStruct(val reflect.Value) ([]any, error) {
	t := val.Type()
	id := e.registry.GetID(t)
	typeInfo := e.registry.GetTypeInfo(id)

	isNewType := e.lastType != t
	e.lastType = t

	var result []any
	if isNewType {
		result = append(result, 1)
	} else {
		result = append(result, 0)
	}

	// Ensure we have an addressable value
	if !val.CanAddr() {
		newVal := reflect.New(t).Elem()
		newVal.Set(val)
		val = newVal
	}

	// Emit order is shared and immutable; usage is tracked per Marshal call.
	usage := e.usageFor(id, typeInfo)
	fieldOrder := usage.order

	// Collect field values and track which are zero
	type fieldData struct {
		orderIdx int
		isZero   bool
		value    any
	}
	fieldDataList := make([]fieldData, len(fieldOrder))
	lastNonZeroIdx := -1

	for orderIdx, fieldIdx := range fieldOrder {
		fVal := val.Field(fieldIdx).Interface()

		if isZero(fVal) {
			fieldDataList[orderIdx] = fieldData{orderIdx: orderIdx, isZero: true}
		} else {
			// This field carries a value in this payload, so it needs a name in the keys header.
			usage.mask[fieldIdx] = true
			if usage.counts != nil {
				usage.counts[fieldIdx]++
			}

			// Recursively marshal if it's a struct or slice
			marshaledVal, err := e.marshalValue(fVal)
			if err != nil {
				return nil, err
			}
			fieldDataList[orderIdx] = fieldData{orderIdx: orderIdx, isZero: false, value: marshaledVal}
			lastNonZeroIdx = orderIdx
		}
	}

	// Build skip indices and values - only include skip indices before the last non-zero value
	var skipIndices []int
	var values []any
	for orderIdx := 0; orderIdx <= lastNonZeroIdx; orderIdx++ {
		fd := fieldDataList[orderIdx]
		if fd.isZero {
			skipIndices = append(skipIndices, orderIdx)
		} else {
			values = append(values, fd.value)
		}
	}

	// Reference block
	if isNewType {
		refBlock := []any{id}
		for _, skip := range skipIndices {
			refBlock = append(refBlock, skip)
		}
		result = append(result, refBlock)
	} else if len(skipIndices) > 0 {
		result = append(result, skipIndices)
	} else if len(values) > 0 && isEncodedArray(values[0]) {
		// A header-0 record normally drops the skip block when nothing is skipped, so
		// the first value sits right after the header. That is only decodable while
		// the first value is a scalar: an encoded array (nested struct/slice/map) is
		// indistinguishable from a skip list — `[2]` is both "empty slice" and "skip
		// index 2". Emit an empty skip block so position 1 is always the skip list
		// when it is an array, which is the invariant both decoders rely on.
		result = append(result, []int{})
	}

	result = append(result, values...)
	return result, nil
}

func (e *Encoder) marshalSlice(val reflect.Value) ([]any, error) {
	result := []any{2}
	for i := 0; i < val.Len(); i++ {
		item, err := e.marshalValue(val.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (e *Encoder) marshalValue(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	val := reflect.ValueOf(v)
	for (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && !val.IsNil() {
		if val.Kind() == reflect.Ptr {
			ptr := val.Pointer()
			if e.seen[ptr] {
				return nil, nil // Cycle detected, break recursion
			}
			e.seen[ptr] = true
			defer func(p uintptr) { delete(e.seen, p) }(ptr)
		}
		val = val.Elem()
	}

	if (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && val.IsNil() {
		return nil, nil
	}

	if val.Kind() == reflect.Struct {
		return e.marshalStruct(val)
	}
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		return e.marshalSlice(val)
	}
	if val.Kind() == reflect.Map {
		return e.marshalMap(val)
	}

	if !val.IsValid() {
		return nil, nil
	}
	return val.Interface(), nil
}

// buildKeysList is the []any form of the keys header, used only by marshalTree so the oracle
// can hand a complete tree to sonic. Production renders the same thing via appendKeysList.
// (renders the payload's keys header: for every type this payload touched, the
// positions in its emit order that carried a value, paired with the field name.
//
// Format: [[id, orderIdx1, "name1", orderIdx2, "name2", ...], ...]
//
// Positions whose field was zero in every record are simply absent, which is exactly what the
// per-record skip indices already assume — both index into the same emit-order space.
func (e *Encoder) buildKeysList() [][]any {
	result := [][]any{}
	if len(e.typeUsage) == 0 {
		return result
	}

	// Emit types in ID order so the same payload always serializes identically.
	typeIDs := make([]int, 0, len(e.typeUsage))
	for id := range e.typeUsage {
		typeIDs = append(typeIDs, id)
	}
	sort.Ints(typeIDs)

	for _, id := range typeIDs {
		usage := e.typeUsage[id]
		info := e.registry.GetTypeInfo(id)
		if info == nil {
			continue
		}

		keyEntry := []any{id}
		for orderIdx, fieldIdx := range usage.order {
			if usage.mask[fieldIdx] {
				keyEntry = append(keyEntry, orderIdx, info.Fields[fieldIdx].Name)
			}
		}
		// A type with no used fields contributes nothing the decoder can act on.
		if len(keyEntry) > 1 {
			result = append(result, keyEntry)
		}
	}
	return result
}
