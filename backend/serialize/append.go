package serialize

// Direct-to-bytes encoder.
//
// The tree-based path built a []any mirror of the whole payload and handed it to sonic, which
// meant every scalar was boxed into an interface (one heap allocation each) and the object graph
// was walked twice — once to build the tree, once for sonic to render it. These append* helpers
// write the compact [keys, content] form straight into a byte slice instead, so nothing is
// boxed and nothing is walked twice.
//
// The wire format is unchanged. Because sonic used to render the final bytes, these helpers must
// reproduce sonic's scalar encoding exactly; appendJSONString in particular follows sonic's
// escaping rules rather than encoding/json's, and TestAppendJSONStringMatchesSonic pins that
// against the real thing.

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
)

// maxStackFields is the field count that fits in appendStruct's stack-allocated zero map. Wider
// structs fall back to a heap slice. Sized past the widest response struct in the codebase
// (business/types.Product, ~36 fields) so the common path never allocates.
const maxStackFields = 96

// appendValue writes val's compact representation into dst and returns the extended slice.
func (e *Encoder) appendValue(dst []byte, val reflect.Value) ([]byte, error) {
	if !val.IsValid() {
		return append(dst, "null"...), nil
	}

	for (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && !val.IsNil() {
		if val.Kind() == reflect.Ptr {
			pointer := val.Pointer()
			if e.seen[pointer] {
				return append(dst, "null"...), nil // Cycle detected, break recursion
			}
			e.seen[pointer] = true
			defer func(p uintptr) { delete(e.seen, p) }(pointer)
		}
		val = val.Elem()
	}

	if (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && val.IsNil() {
		return append(dst, "null"...), nil
	}

	switch val.Kind() {
	case reflect.Struct:
		return e.appendStruct(dst, val)
	case reflect.Slice, reflect.Array:
		return e.appendSlice(dst, val)
	case reflect.Map:
		return e.appendMap(dst, val)
	}

	return appendScalar(dst, val)
}

// encodesAsArray reports whether val will be written as a JSON array.
//
// It backs the empty-skip-block rule in appendStruct: a header-0 record drops the skip block
// when nothing is skipped, which is only decodable while the first value is a scalar. An array
// sitting at position 1 is indistinguishable from a skip list, so one has to be emitted.
func encodesAsArray(val reflect.Value) bool {
	if !val.IsValid() {
		return false
	}
	for (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && !val.IsNil() {
		val = val.Elem()
	}
	if (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && val.IsNil() {
		return false
	}
	switch val.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	}
	return false
}

func (e *Encoder) appendStruct(dst []byte, val reflect.Value) ([]byte, error) {
	structType := val.Type()
	typeID := e.registry.GetID(structType)
	typeInfo := e.registry.GetTypeInfo(typeID)

	isNewType := e.lastType != structType
	e.lastType = structType

	usage := e.usageFor(typeID, typeInfo)
	fieldOrder := usage.order

	// Zero-detection has to finish before any value is written, because the skip block precedes
	// the values on the wire. Backed by a stack array so nested structs each get their own —
	// a shared scratch buffer on the Encoder would be clobbered by recursion.
	var zeroFieldsArray [maxStackFields]bool
	var zeroFields []bool
	if len(fieldOrder) <= maxStackFields {
		zeroFields = zeroFieldsArray[:len(fieldOrder)]
	} else {
		zeroFields = make([]bool, len(fieldOrder))
	}

	lastNonZeroIdx := -1
	for orderIdx, fieldIdx := range fieldOrder {
		if val.Field(fieldIdx).IsZero() {
			zeroFields[orderIdx] = true
			continue
		}
		zeroFields[orderIdx] = false
		// This field carries a value in this payload, so it needs a name in the keys header.
		usage.mask[fieldIdx] = true
		if usage.counts != nil {
			usage.counts[fieldIdx]++
		}
		lastNonZeroIdx = orderIdx
	}

	dst = append(dst, '[')
	if isNewType {
		dst = append(dst, '1')
	} else {
		dst = append(dst, '0')
	}

	// Reference block: the type ID plus the skipped positions, or just the skipped positions on
	// a repeat of the same type. Only positions before the last value matter — everything after
	// it is truncated rather than skipped.
	hasSkippedFields := false
	for orderIdx := 0; orderIdx <= lastNonZeroIdx; orderIdx++ {
		if zeroFields[orderIdx] {
			hasSkippedFields = true
			break
		}
	}

	appendSkipIndices := func(dst []byte, needsLeadingComma bool) []byte {
		for orderIdx := 0; orderIdx <= lastNonZeroIdx; orderIdx++ {
			if !zeroFields[orderIdx] {
				continue
			}
			if needsLeadingComma {
				dst = append(dst, ',')
			}
			dst = strconv.AppendInt(dst, int64(orderIdx), 10)
			needsLeadingComma = true
		}
		return dst
	}

	if isNewType {
		dst = append(dst, ",["...)
		dst = strconv.AppendInt(dst, int64(typeID), 10)
		dst = appendSkipIndices(dst, true)
		dst = append(dst, ']')
	} else if hasSkippedFields {
		dst = append(dst, ",["...)
		dst = appendSkipIndices(dst, false)
		dst = append(dst, ']')
	} else if lastNonZeroIdx >= 0 && encodesAsArray(val.Field(fieldOrder[0])) {
		// No skips, and the first value is an array: emit an empty skip block so position 1 is
		// always the skip list when it is an array, which is the invariant both decoders rely on.
		dst = append(dst, ",[]"...)
	}

	for orderIdx := 0; orderIdx <= lastNonZeroIdx; orderIdx++ {
		if zeroFields[orderIdx] {
			continue
		}
		dst = append(dst, ',')

		var err error
		dst, err = e.appendValue(dst, val.Field(fieldOrder[orderIdx]))
		if err != nil {
			return nil, err
		}
	}

	return append(dst, ']'), nil
}

func (e *Encoder) appendSlice(dst []byte, val reflect.Value) ([]byte, error) {
	dst = append(dst, "[2"...)

	for i := range val.Len() {
		dst = append(dst, ',')

		var err error
		dst, err = e.appendValue(dst, val.Index(i))
		if err != nil {
			return nil, err
		}
	}

	return append(dst, ']'), nil
}

func (e *Encoder) appendMap(dst []byte, val reflect.Value) ([]byte, error) {
	dst = append(dst, "[3"...)

	// Keys are stringified and sorted so the same map always serializes identically.
	mapKeys := val.MapKeys()
	sortedKeys := make([]struct {
		value reflect.Value
		text  string
	}, len(mapKeys))

	for i, mapKey := range mapKeys {
		sortedKeys[i].value = mapKey
		sortedKeys[i].text = fmt.Sprintf("%v", mapKey.Interface())
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].text < sortedKeys[j].text
	})

	for _, sortedKey := range sortedKeys {
		dst = append(dst, ',')
		dst = appendJSONString(dst, sortedKey.text)
		dst = append(dst, ',')

		var err error
		dst, err = e.appendValue(dst, val.MapIndex(sortedKey.value))
		if err != nil {
			return nil, err
		}
	}

	return append(dst, ']'), nil
}

// appendKeysList renders the payload's keys header: for every type this payload touched, the
// positions in its emit order that carried a value, paired with the field name.
//
// Format: [[id, orderIdx1, "name1", orderIdx2, "name2", ...], ...]
//
// Positions whose field was zero in every record are simply absent, which is exactly what the
// per-record skip indices already assume — both index into the same emit-order space.
func (e *Encoder) appendKeysList(dst []byte) []byte {
	dst = append(dst, '[')
	if len(e.typeUsage) == 0 {
		return append(dst, ']')
	}

	// Emit types in ID order so the same payload always serializes identically.
	typeIDs := make([]int, 0, len(e.typeUsage))
	for typeID := range e.typeUsage {
		typeIDs = append(typeIDs, typeID)
	}
	sort.Ints(typeIDs)

	wroteEntry := false
	for _, typeID := range typeIDs {
		usage := e.typeUsage[typeID]
		typeInfo := e.registry.GetTypeInfo(typeID)
		if typeInfo == nil {
			continue
		}

		// A type with no used fields contributes nothing the decoder can act on.
		hasUsedField := false
		for _, fieldIdx := range usage.order {
			if usage.mask[fieldIdx] {
				hasUsedField = true
				break
			}
		}
		if !hasUsedField {
			continue
		}

		if wroteEntry {
			dst = append(dst, ',')
		}
		dst = append(dst, '[')
		dst = strconv.AppendInt(dst, int64(typeID), 10)

		for orderIdx, fieldIdx := range usage.order {
			if !usage.mask[fieldIdx] {
				continue
			}
			dst = append(dst, ',')
			dst = strconv.AppendInt(dst, int64(orderIdx), 10)
			dst = append(dst, ',')
			dst = appendJSONString(dst, typeInfo.Fields[fieldIdx].Name)
		}

		dst = append(dst, ']')
		wroteEntry = true
	}

	return append(dst, ']')
}

func appendScalar(dst []byte, val reflect.Value) ([]byte, error) {
	switch val.Kind() {
	case reflect.Bool:
		if val.Bool() {
			return append(dst, "true"...), nil
		}
		return append(dst, "false"...), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(dst, val.Int(), 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.AppendUint(dst, val.Uint(), 10), nil

	case reflect.Float32:
		return appendJSONFloat(dst, val.Float(), 32), nil

	case reflect.Float64:
		return appendJSONFloat(dst, val.Float(), 64), nil

	case reflect.String:
		return appendJSONString(dst, val.String()), nil
	}

	return nil, fmt.Errorf("serialize: unsupported value kind %s", val.Kind())
}

// appendJSONFloat mirrors encoding/json's float formatting, which is what sonic emits too:
// shortest round-tripping form, 'e' notation only outside [1e-6, 1e21), and the exponent
// trimmed from "e-09" to "e-9".
func appendJSONFloat(dst []byte, value float64, bitSize int) []byte {
	if math.IsInf(value, 0) || math.IsNaN(value) {
		// Not representable in JSON; null matches what a failed encode would degrade to.
		return append(dst, "null"...)
	}

	format := byte('f')
	absolute := math.Abs(value)
	if absolute != 0 {
		if (bitSize == 64 && (absolute < 1e-6 || absolute >= 1e21)) ||
			(bitSize == 32 && (float64(float32(absolute)) < 1e-6 || float64(float32(absolute)) >= 1e21)) {
			format = 'e'
		}
	}

	start := len(dst)
	dst = strconv.AppendFloat(dst, value, format, -1, bitSize)

	if format == 'e' {
		// Trim the leading zero of a two-digit exponent: 1e-09 -> 1e-9.
		formatted := dst[start:]
		if n := len(formatted); n >= 4 && formatted[n-4] == 'e' && formatted[n-3] == '-' && formatted[n-2] == '0' {
			formatted[n-2] = formatted[n-1]
			dst = dst[:len(dst)-1]
		}
	}

	return dst
}

const lowerHexDigits = "0123456789abcdef"

// appendJSONString writes a JSON string literal following sonic's rules, which are looser than
// encoding/json's in three ways that all matter here: sonic does not escape HTML characters
// (< > &), does not escape U+2028/U+2029, and copies bytes through verbatim without validating
// UTF-8 — invalid sequences are passed on untouched rather than replaced with U+FFFD. So this
// is a plain byte scan: only the quote, the backslash and control characters below 0x20 are
// escaped, the latter as \u00xx apart from the tab/newline/carriage-return short forms. DEL
// (0x7f) is not a control character by JSON's definition and passes through.
func appendJSONString(dst []byte, value string) []byte {
	dst = append(dst, '"')

	start := 0
	for i := 0; i < len(value); i++ {
		char := value[i]
		if char >= 0x20 && char != '"' && char != '\\' {
			continue
		}

		dst = append(dst, value[start:i]...)
		switch char {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			dst = append(dst, '\\', 'u', '0', '0', lowerHexDigits[char>>4], lowerHexDigits[char&0xf])
		}
		start = i + 1
	}

	dst = append(dst, value[start:]...)
	return append(dst, '"')
}
