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
	Index   int    // Original field index in the struct
	Name    string // JSON tag name or field name
	RawName string // Original field name (without JSON tag)
	Ignored bool   // Whether the field is omitted (json:"-")
}

// TypeInfo stores metadata for a registered type.
//
// Everything here is derived from the Go type and is therefore immutable once built — it is
// shared across every concurrent Marshal. Per-response state (which fields a given payload
// actually used) lives on the Encoder instead; see encoderTypeUsage.
//
// OptimizedOrder is the one field that is learned rather than derived. It is frozen from the
// first payload that carries the type and reused verbatim afterwards, which is what lets
// Marshal run a single pass. Freezing is safe because the order is transmitted in the payload's
// keys header on every response, so the decoder never depends on the encoder's history.
type TypeInfo struct {
	ID      int
	Type    reflect.Type
	XStruct *xunsafe.Struct
	Fields  []FieldInfo
	// DefaultOrder lists the non-ignored field indices in declaration order. Precomputed so a
	// type that has not been optimized yet does not rebuild this slice for every record.
	DefaultOrder []int
	// OptimizedOrder / IsOptimized are written once under the registry lock and read-only after.
	OptimizedOrder []int
	IsOptimized    bool
}

// NOTE: read OptimizedOrder/IsOptimized only through FieldRegistry.orderFor. freezeOrder writes
// them under the registry's write lock, and one goroutine can be freezing a type while another
// is starting to encode it, so an unsynchronised read here is a genuine race. The slice contents
// are safe to read afterwards: the slice header is assigned once and never mutated.

// FieldRegistry manages the mapping between types and their IDs
type FieldRegistry struct {
	mu       sync.RWMutex
	typeToID map[reflect.Type]int
	idToInfo map[int]*TypeInfo
	nextID   int
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
	if jsonTag == "-" {
		return "-"
	}
	if jsonTag == "" {
		return field.Name
	}
	// Handle "name,omitempty" format
	if idx := strings.Index(jsonTag, ","); idx != -1 {
		name := jsonTag[:idx]
		// If name is empty (e.g., ",omitempty"), use the struct field name
		if name == "" {
			return field.Name
		}
		return name
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
	defaultOrder := make([]int, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		name := getFieldName(sf)
		ignored := name == "-"
		if ignored {
			name = sf.Name // Keep a name for debug, but mark ignored
		}
		fields[i] = FieldInfo{
			Index:   i,
			Name:    name,
			RawName: sf.Name,
			Ignored: ignored,
		}
		if !ignored {
			defaultOrder = append(defaultOrder, i)
		}
	}

	r.idToInfo[id] = &TypeInfo{
		ID:           id,
		Type:         t,
		XStruct:      xStruct,
		Fields:       fields,
		DefaultOrder: defaultOrder,
	}
	return id
}

// freezeOrder records the field order learned from the first payload that carried this type.
// Later payloads reuse it, which is what removes the per-response analysis pass. Once set it is
// never rewritten: a stable order keeps the wire format predictable, and the keys header makes
// any order self-describing to the decoder anyway.
func (r *FieldRegistry) freezeOrder(id int, usageCounts []int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, found := r.idToInfo[id]
	if !found || info.IsOptimized {
		return
	}

	// Sort non-ignored fields by how often they carried a non-zero value. Most-used first makes
	// the always-zero fields cluster at the tail, where marshalStruct truncates them outright
	// instead of spending a skip index on each one.
	order := append([]int(nil), info.DefaultOrder...)
	sort.SliceStable(order, func(i, j int) bool {
		return usageCounts[order[i]] > usageCounts[order[j]]
	})

	info.OptimizedOrder = order
	info.IsOptimized = true
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

// encoderTypeUsage is the per-Marshal record of what one type's fields actually carried in
// this payload. It lives on the Encoder rather than the registry so concurrent Marshal calls
// never touch shared mutable state — that was the data race behind the corrupted-payload risk
// documented in backend/docs/LAMBDA.md.
type encoderTypeUsage struct {
	// order is the field order this payload was emitted with. Captured here so the keys header
	// is built against the exact order the values were written in.
	order []int
	// mask is indexed by struct field index: true when some record carried a non-zero value.
	mask []bool
	// counts feeds freezeOrder the first time a type is seen. Nil once the order is frozen,
	// since there is nothing left to learn.
	counts []int
}

// orderFor returns the field iteration order to encode a type with, and whether that order has
// already been frozen. Taken under the read lock because freezeOrder can be running on another
// goroutine; see the note on TypeInfo.
func (r *FieldRegistry) orderFor(id int) ([]int, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, found := r.idToInfo[id]
	if !found {
		return nil, false
	}
	if info.IsOptimized {
		return info.OptimizedOrder, true
	}
	return info.DefaultOrder, false
}

// usageFor returns this Encoder's usage record for a type, creating it on first contact. The
// order is captured once per payload here, so a type that gets frozen mid-encode still uses one
// consistent order for every record it writes — which is what keeps the keys header aligned.
func (e *Encoder) usageFor(id int, info *TypeInfo) *encoderTypeUsage {
	if usage, found := e.typeUsage[id]; found {
		return usage
	}

	order, isOptimized := e.registry.orderFor(id)
	usage := &encoderTypeUsage{
		order: order,
		mask:  make([]bool, len(info.Fields)),
	}
	// Only bother counting while the order is still unknown.
	if !isOptimized {
		usage.counts = make([]int, len(info.Fields))
	}
	e.typeUsage[id] = usage
	return usage
}

// learnFieldOrders freezes the emit order for every type this payload saw for the first time.
// Called after the content is encoded, so the counts reflect the whole payload.
func (e *Encoder) learnFieldOrders() {
	for id, usage := range e.typeUsage {
		if usage.counts != nil {
			e.registry.freezeOrder(id, usage.counts)
		}
	}
}

var globalRegistry = NewFieldRegistry()

// isEncodedArray reports whether a marshaled value serializes to a JSON array.
// Structs, slices and maps all come back from marshalValue as []any; anything
// else is a scalar (or null) and can never be mistaken for a skip block.
func isEncodedArray(v any) bool {
	if v == nil {
		return false
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.Slice, reflect.Array:
		return true
	}
	return false
}

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
