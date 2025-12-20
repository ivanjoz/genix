package serialize

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/viant/xunsafe"
)

// FieldInfo stores metadata for a struct field
type FieldInfo struct {
	Index      int    // Original field index in the struct
	Name       string // JSON tag name or field name
	Used       bool   // Whether this field was used in any marshaled record
	RawName    string // Original field name (without JSON tag)
	UsageCount int    // How many times this field was used (for optimization)
}

// TypeInfo stores metadata for a registered type
type TypeInfo struct {
	ID             int
	Type           reflect.Type
	XStruct        *xunsafe.Struct
	Fields         []FieldInfo
	UsedMask       []bool // Tracks which fields have been used
	OptimizedOrder []int  // Field indices ordered by usage (most used first)
	IsOptimized    bool   // Whether the optimized order has been computed
}

// FieldRegistry manages the mapping between types and their IDs
type FieldRegistry struct {
	mu         sync.RWMutex
	typeToID   map[reflect.Type]int
	idToInfo   map[int]*TypeInfo
	nextID     int
}

func NewFieldRegistry() *FieldRegistry {
	return &FieldRegistry{
		typeToID: make(map[reflect.Type]int),
		idToInfo: make(map[int]*TypeInfo),
		nextID:   1,
	}
}

// getFieldName extracts the JSON tag name or falls back to the field name
func getFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return field.Name
	}
	// Handle "name,omitempty" format
	if idx := strings.Index(jsonTag, ","); idx != -1 {
		return jsonTag[:idx]
	}
	return jsonTag
}

func (r *FieldRegistry) GetID(t reflect.Type) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.typeToID[t]; ok {
		return id
	}
	id := r.nextID
	r.nextID++
	r.typeToID[t] = id

	xStruct := xunsafe.NewStruct(t)
	fields := make([]FieldInfo, t.NumField())
	usedMask := make([]bool, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fields[i] = FieldInfo{
			Index:   i,
			Name:    getFieldName(sf),
			RawName: sf.Name,
			Used:    false,
		}
	}

	r.idToInfo[id] = &TypeInfo{
		ID:       id,
		Type:     t,
		XStruct:  xStruct,
		Fields:   fields,
		UsedMask: usedMask,
	}
	return id
}

func (r *FieldRegistry) GetStruct(id int) *xunsafe.Struct {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, ok := r.idToInfo[id]; ok {
		return info.XStruct
	}
	return nil
}

func (r *FieldRegistry) GetType(id int) reflect.Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, ok := r.idToInfo[id]; ok {
		return info.Type
	}
	return nil
}

func (r *FieldRegistry) GetTypeInfo(id int) *TypeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.idToInfo[id]
}

// MarkFieldUsed marks a field as used for a given type ID and increments usage count
func (r *FieldRegistry) MarkFieldUsed(id int, fieldIndex int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.idToInfo[id]; ok {
		if fieldIndex < len(info.UsedMask) {
			info.UsedMask[fieldIndex] = true
			info.Fields[fieldIndex].Used = true
			info.Fields[fieldIndex].UsageCount++
		}
	}
}

// ComputeOptimizedOrder sorts fields by usage count (most used first)
// This should be called after the first pass (analysis) and before the second pass (marshal)
func (r *FieldRegistry) ComputeOptimizedOrder() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, info := range r.idToInfo {
		// Create a slice of field indices
		order := make([]int, len(info.Fields))
		for i := range order {
			order[i] = i
		}

		// Sort by usage count (descending) - most used first
		sort.Slice(order, func(i, j int) bool {
			return info.Fields[order[i]].UsageCount > info.Fields[order[j]].UsageCount
		})

		info.OptimizedOrder = order
		info.IsOptimized = true
	}
}

// GetOptimizedOrder returns the optimized field order for a type
func (r *FieldRegistry) GetOptimizedOrder(id int) []int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, ok := r.idToInfo[id]; ok {
		return info.OptimizedOrder
	}
	return nil
}

// IsOptimized returns whether the optimized order has been computed for a type
func (r *FieldRegistry) IsOptimized(id int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, ok := r.idToInfo[id]; ok {
		return info.IsOptimized
	}
	return false
}

// GetKeysList returns the list of keys for all registered types
// Format: [[id, idx1, "name1", idx2, "name2", ...], ...]
// Only includes fields that were actually used during marshaling
// Fields are ordered by usage (most used first) if optimization was computed
func (r *FieldRegistry) GetKeysList() [][]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result [][]any
	for id, info := range r.idToInfo {
		keyEntry := []any{id}

		if info.IsOptimized {
			// Use optimized order (most used fields first)
			for orderIdx, fieldIdx := range info.OptimizedOrder {
				if info.UsedMask[fieldIdx] {
					keyEntry = append(keyEntry, orderIdx+1, info.Fields[fieldIdx].Name) // 1-based optimized index
				}
			}
		} else {
			// Use original order
			for i, field := range info.Fields {
				if info.UsedMask[i] {
					keyEntry = append(keyEntry, i+1, field.Name) // 1-based index
				}
			}
		}
		// Only add if there are used fields
		if len(keyEntry) > 1 {
			result = append(result, keyEntry)
		}
	}
	return result
}

// GetKeysListAll returns the list of keys for all registered types
// Includes all fields, not just used ones
func (r *FieldRegistry) GetKeysListAll() [][]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result [][]any
	for id, info := range r.idToInfo {
		keyEntry := []any{id}
		for i, field := range info.Fields {
			keyEntry = append(keyEntry, i+1, field.Name) // 1-based index as per README
		}
		result = append(result, keyEntry)
	}
	return result
}

// Reset clears the used flags and optimization for all fields
func (r *FieldRegistry) ResetUsedFlags() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, info := range r.idToInfo {
		for i := range info.UsedMask {
			info.UsedMask[i] = false
			info.Fields[i].Used = false
			info.Fields[i].UsageCount = 0
		}
		info.OptimizedOrder = nil
		info.IsOptimized = false
	}
}

var globalRegistry = NewFieldRegistry()

func isZero(v any) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	return val.IsZero()
}

func setPrimitive(val reflect.Value, data any) error {
	if data == nil {
		val.Set(reflect.Zero(val.Type()))
		return nil
	}

	v := reflect.ValueOf(data)
	if v.Type().AssignableTo(val.Type()) {
		val.Set(v)
		return nil
	}

	// Handle numeric conversions from float64 (JSON default)
	if f, ok := data.(float64); ok {
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val.SetInt(int64(f))
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val.SetUint(uint64(f))
			return nil
		case reflect.Float32, reflect.Float64:
			val.SetFloat(f)
			return nil
		}
	}

	return fmt.Errorf("cannot set %T to %v", data, val.Type())
}
