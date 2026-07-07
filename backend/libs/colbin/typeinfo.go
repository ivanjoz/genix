package colbin

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/viant/xunsafe"
)

// fieldMeta describes one struct field: its wire id, type class, how to read/write
// its scalar value via xunsafe, and (later phases) nested array/struct descriptors.
type fieldMeta struct {
	id       uint8          // wire field-id (FNV of name, linear-probed)
	name     string         // name used for the hash (cb tag override or Go name)
	fType    uint8          // ft* type class
	offset   uintptr        // byte offset of the field within the struct
	xf       *xunsafe.Field // xunsafe accessor for fast scalar get/set (struct fields only)
	goKind   reflect.Kind   // exact kind for int/uint/float dispatch
	intWidth uint8          // native bit width for ftInt fields (8/16/32/64)

	// Composite descriptors (Phase B). Elements of an array have no struct field,
	// so scalar elements are accessed by direct pointer cast (see value_elem.go).
	sub       *typeInfo    // ftStruct: nested sub-table (also for struct elements)
	elem      *fieldMeta   // ftArray: element descriptor; ptr: the pointee descriptor
	sliceType reflect.Type // ftArray: the slice type (to build slices on decode)
	elemSize  uintptr      // ftArray: size of one element

	// Nullability (Phase C). A pointer field is nullable: elem describes the pointee,
	// accessed like an array element, and pointeeType allocates backing on decode.
	nullable    bool
	pointeeType reflect.Type

	// Maps (Phase C). K restricted to scalar/string; V any supported type.
	mapKey, mapVal         *fieldMeta
	mapKeyType, mapValType reflect.Type
	mapType                reflect.Type
}

// typeInfo is the cached, ordered field layout for a struct type.
type typeInfo struct {
	rtype  reflect.Type
	size   uintptr
	fields []fieldMeta          // in declaration order
	byID   map[uint8]*fieldMeta // wire id -> field, for decode
}

var typeInfoCache sync.Map // reflect.Type -> *typeInfo

// getTypeInfo returns cached field layout for a struct type, building it once.
func getTypeInfo(t reflect.Type) (*typeInfo, error) {
	if cached, ok := typeInfoCache.Load(t); ok {
		return cached.(*typeInfo), nil
	}
	ti, err := buildTypeInfo(t)
	if err != nil {
		return nil, err
	}
	typeInfoCache.Store(t, ti)
	return ti, nil
}

// buildTypeInfo introspects a struct type and assigns each field a wire id.
// Explicit ids (`cb:"5"`) are reserved first; the remaining fields get an FNV hash
// of their name, linear-probed around the already-used slots.
func buildTypeInfo(t reflect.Type) (*typeInfo, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("colbin: expected struct, got %s", t.Kind())
	}
	xs := xunsafe.NewStruct(t)
	ti := &typeInfo{rtype: t, size: t.Size()}

	explicitIDs := make([]int, 0) // parallel to ti.fields: >=0 explicit, -1 hashed
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		name, explicitID, skip := parseCbTag(sf)
		if skip {
			continue
		}
		if len(ti.fields) >= 254 {
			return nil, fmt.Errorf("colbin: %s exceeds 254 encodable fields", t.Name())
		}
		if explicitID > 254 { // 255 is reserved
			return nil, fmt.Errorf("colbin: field %s.%s id %d out of range 0..254", t.Name(), sf.Name, explicitID)
		}
		fm, err := describeType(sf.Type)
		if err != nil {
			return nil, fmt.Errorf("colbin: field %s.%s: %w", t.Name(), sf.Name, err)
		}
		fm.name = name
		fm.offset = sf.Offset
		fm.xf = &xs.Fields[i]
		ti.fields = append(ti.fields, fm)
		explicitIDs = append(explicitIDs, explicitID)
	}

	used := map[uint8]bool{reservedFieldID: true} // 255 is reserved
	for k, id := range explicitIDs {              // pass 1: reserve explicit ids
		if id < 0 {
			continue
		}
		if used[uint8(id)] {
			return nil, fmt.Errorf("colbin: %s has duplicate field id %d", t.Name(), id)
		}
		used[uint8(id)] = true
		ti.fields[k].id = uint8(id)
	}
	for k, id := range explicitIDs { // pass 2: hash + probe the rest
		if id >= 0 {
			continue
		}
		hid := probeFieldID(fnv8(ti.fields[k].name), used)
		used[hid] = true
		ti.fields[k].id = hid
	}

	ti.byID = make(map[uint8]*fieldMeta, len(ti.fields))
	for i := range ti.fields {
		ti.byID[ti.fields[i].id] = &ti.fields[i]
	}
	return ti, nil
}

// parseCbTag reads the `cb` struct tag. Comma-separated tokens: an integer token
// sets the explicit field id (no hashing); the first non-integer token overrides
// the hashed name. `cb:"-"` skips the field; empty/absent falls back to the Go
// field name with a hashed id. Examples: `cb:"5"` (id 5), `cb:"id,5"` (name "id",
// id 5), `cb:"id"` (hashed id from "id").
func parseCbTag(sf reflect.StructField) (name string, explicitID int, skip bool) {
	name, explicitID = sf.Name, -1
	tag := sf.Tag.Get("cb")
	if tag == "-" {
		return "", -1, true
	}
	if tag == "" {
		return sf.Name, -1, false
	}
	nameSet := false
	for _, tok := range strings.Split(tag, ",") {
		if tok == "" {
			continue
		}
		if id, err := strconv.Atoi(tok); err == nil {
			explicitID = id // integer token -> explicit id
		} else if !nameSet {
			name, nameSet = tok, true // first non-integer token -> name
		}
	}
	return name, explicitID, false
}

// describeType maps a Go type to its wire type class, recursively resolving
// array element and nested struct descriptors. Used for both struct fields and
// array elements (the caller fills id/name/offset/xf for actual struct fields).
func describeType(t reflect.Type) (fieldMeta, error) {
	switch k := t.Kind(); k {
	case reflect.Int8, reflect.Uint8:
		return fieldMeta{fType: ftInt, goKind: k, intWidth: 8}, nil
	case reflect.Int16, reflect.Uint16:
		return fieldMeta{fType: ftInt, goKind: k, intWidth: 16}, nil
	case reflect.Int32, reflect.Uint32:
		return fieldMeta{fType: ftInt, goKind: k, intWidth: 32}, nil
	case reflect.Int64, reflect.Uint64, reflect.Int, reflect.Uint:
		return fieldMeta{fType: ftInt, goKind: k, intWidth: 64}, nil
	case reflect.Bool:
		return fieldMeta{fType: ftInt, goKind: k, intWidth: 8}, nil
	case reflect.Float32:
		return fieldMeta{fType: ftFloat, goKind: k, intWidth: 32}, nil
	case reflect.Float64:
		return fieldMeta{fType: ftFloat, goKind: k, intWidth: 64}, nil
	case reflect.String:
		return fieldMeta{fType: ftString, goKind: k}, nil
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return fieldMeta{fType: ftBytes, goKind: k}, nil // []byte
		}
		et := t.Elem()
		em, err := describeType(et)
		if err != nil {
			return fieldMeta{}, err
		}
		return fieldMeta{fType: ftArray, goKind: k, sliceType: t, elemSize: et.Size(), elem: &em}, nil
	case reflect.Struct:
		sub, err := getTypeInfo(t) // recurse (nested struct becomes a sub-table)
		if err != nil {
			return fieldMeta{}, err
		}
		return fieldMeta{fType: ftStruct, goKind: k, sub: sub}, nil
	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Ptr {
			return fieldMeta{}, fmt.Errorf("pointer-to-pointer %s not supported", t)
		}
		pd, err := describeType(t.Elem()) // pointee accessed like an array element
		if err != nil {
			return fieldMeta{}, err
		}
		return fieldMeta{fType: pd.fType, goKind: pd.goKind, intWidth: pd.intWidth,
			sub: pd.sub, nullable: true, pointeeType: t.Elem(), elem: &pd}, nil
	case reflect.Map:
		kd, err := describeType(t.Key())
		if err != nil {
			return fieldMeta{}, err
		}
		if kd.fType != ftInt && kd.fType != ftFloat && kd.fType != ftString {
			return fieldMeta{}, fmt.Errorf("map key %s must be scalar or string", t.Key())
		}
		vd, err := describeType(t.Elem())
		if err != nil {
			return fieldMeta{}, err
		}
		return fieldMeta{fType: ftMap, mapType: t, mapKeyType: t.Key(), mapValType: t.Elem(),
			mapKey: &kd, mapVal: &vd}, nil
	default:
		return fieldMeta{}, fmt.Errorf("unsupported type %s", t)
	}
}

// fnv8 computes FNV-1a 32-bit over s, xor-folded down to 8 bits.
func fnv8(s string) uint8 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return uint8(h ^ (h >> 8) ^ (h >> 16) ^ (h >> 24))
}

// probeFieldID returns start, or the next free id (wrapping, skipping used slots).
func probeFieldID(start uint8, used map[uint8]bool) uint8 {
	id := start
	for used[id] {
		id++ // wraps at 256; reservedFieldID(255) is pre-marked so it's skipped
	}
	return id
}
