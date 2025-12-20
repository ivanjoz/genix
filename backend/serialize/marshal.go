package serialize

import (
	"encoding/json"
	"reflect"

	"github.com/viant/xunsafe"
)

type Encoder struct {
	lastType    reflect.Type
	registry    *FieldRegistry
	isAnalyzing bool // true during first pass (analysis), false during second pass (marshal)
}

func NewEncoder() *Encoder {
	return &Encoder{
		registry: globalRegistry,
	}
}

// Marshal converts an object to the custom array-based format using two-pass optimization
// Returns a single array with two elements: [keys, content]
// - keys: array of type definitions with field names (using JSON tags), ordered by usage
// - content: the serialized data with fields ordered by usage (most used first)
//
// Two-pass process:
// 1. First pass: Analyze all data to count field usage
// 2. Compute optimized field order (most used fields first)
// 3. Second pass: Marshal using the optimized order
func Marshal(v any) ([]byte, error) {
	e := NewEncoder()
	e.registry.ResetUsedFlags() // Reset for fresh tracking

	// === FIRST PASS: Analyze and count field usage ===
	e.isAnalyzing = true
	_, err := e.marshalContent(v)
	if err != nil {
		return nil, err
	}

	// Compute optimized order based on usage counts
	e.registry.ComputeOptimizedOrder()

	// === SECOND PASS: Marshal with optimized field order ===
	e.isAnalyzing = false
	e.lastType = nil // Reset for fresh type tracking
	content, err := e.marshalContent(v)
	if err != nil {
		return nil, err
	}

	keys := e.registry.GetKeysList()
	result := []any{keys, content}

	return json.Marshal(result)
}

// GetKeysList returns the list of keys for all registered types (only used fields)
func (e *Encoder) GetKeysList() [][]any {
	return e.registry.GetKeysList()
}

// GetKeysListAll returns the list of keys for all registered types (all fields)
func (e *Encoder) GetKeysListAll() [][]any {
	return e.registry.GetKeysListAll()
}

func (e *Encoder) marshalContent(v any) ([]any, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return e.marshalStruct(val)
	case reflect.Slice, reflect.Array:
		return e.marshalSlice(val)
	default:
		// Fallback for other types
		return []any{val.Interface()}, nil
	}
}

func (e *Encoder) marshalStruct(val reflect.Value) ([]any, error) {
	t := val.Type()
	id := e.registry.GetID(t)
	xStruct := e.registry.GetStruct(id)
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
	ptr := xunsafe.AsPointer(val.Addr().Interface())

	// Determine field iteration order
	var fieldOrder []int
	if typeInfo != nil && typeInfo.IsOptimized {
		fieldOrder = typeInfo.OptimizedOrder
	} else {
		// Default order
		fieldOrder = make([]int, len(xStruct.Fields))
		for i := range fieldOrder {
			fieldOrder[i] = i
		}
	}

	// First pass: collect field values and track which are zero
	type fieldData struct {
		orderIdx int
		isZero   bool
		value    any
	}
	fieldDataList := make([]fieldData, len(fieldOrder))
	lastNonZeroIdx := -1

	for orderIdx, fieldIdx := range fieldOrder {
		field := &xStruct.Fields[fieldIdx]
		fVal := field.Interface(ptr)

		if isZero(fVal) {
			fieldDataList[orderIdx] = fieldData{orderIdx: orderIdx, isZero: true}
		} else {
			// Mark this field as used in the registry (during first pass)
			if e.isAnalyzing {
				e.registry.MarkFieldUsed(id, fieldIdx)
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
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		return e.marshalStruct(val)
	}
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		return e.marshalSlice(val)
	}
	return v, nil
}
