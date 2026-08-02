package serialize

import (
	"fmt"
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/viant/xunsafe"
)

type Decoder struct {
	lastType   reflect.Type
	lastTypeID int
	registry   *FieldRegistry
	// fieldOrderByType maps a type ID to the field indices the payload emitted, positioned by
	// emit order. Decoded from the payload's own keys header, so a payload is self-describing
	// and decoding never depends on the encoder's registry state. Nil when Unmarshal is called
	// on bare content with no header.
	fieldOrderByType map[int][]int
}

func NewDecoder() *Decoder {
	return &Decoder{
		registry: globalRegistry,
	}
}

// Unmarshal converts the custom format back to an object
// Expects format: [keys, content] where keys is the type definitions and content is the data
func Unmarshal(data []byte, v any) error {
	var arr []any
	if err := sonic.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) != 2 {
		return fmt.Errorf("invalid format: expected [keys, content], got array of length %d", len(arr))
	}

	d := NewDecoder()

	// arr[0] is the keys header; it carries the emit order this payload was written with.
	fieldOrder, err := d.parseKeysHeader(arr[0])
	if err != nil {
		return err
	}
	d.fieldOrderByType = fieldOrder

	return d.Unmarshal(arr[1], v)
}

// parseKeysHeader turns [[id, orderIdx, "name", ...], ...] into typeID -> emit-ordered field
// indices, resolving each name against the registered struct.
//
// The header only names positions that carried a value somewhere in the payload. Positions it
// omits were zero in every record, so they are either listed in a record's skip block or fall
// past the last value and get truncated — either way the decoder never needs a field for them,
// and -1 is stored as a placeholder to keep positions aligned.
func (d *Decoder) parseKeysHeader(rawKeys any) (map[int][]int, error) {
	keyEntries, isArray := rawKeys.([]any)
	if !isArray {
		return nil, nil
	}

	fieldOrderByType := make(map[int][]int, len(keyEntries))

	for _, rawEntry := range keyEntries {
		entry, isEntryArray := rawEntry.([]any)
		if !isEntryArray || len(entry) < 3 {
			continue
		}

		typeIDFloat, isNumber := entry[0].(float64)
		if !isNumber {
			continue
		}
		typeID := int(typeIDFloat)

		typeInfo := d.registry.GetTypeInfo(typeID)
		if typeInfo == nil {
			return nil, fmt.Errorf("keys header references unknown type ID: %d", typeID)
		}

		// Resolve names against this type once per header entry rather than per record.
		fieldIndexByName := make(map[string]int, len(typeInfo.Fields))
		for i, field := range typeInfo.Fields {
			if !field.Ignored {
				fieldIndexByName[field.Name] = i
			}
		}

		fieldOrder := []int{}
		for i := 1; i+1 < len(entry); i += 2 {
			orderIdxFloat, isOrderNumber := entry[i].(float64)
			fieldName, isName := entry[i+1].(string)
			if !isOrderNumber || !isName {
				continue
			}
			orderIdx := int(orderIdxFloat)

			// Grow to the position this name occupies, padding gaps with -1.
			for len(fieldOrder) <= orderIdx {
				fieldOrder = append(fieldOrder, -1)
			}
			if fieldIndex, found := fieldIndexByName[fieldName]; found {
				fieldOrder[orderIdx] = fieldIndex
			}
		}

		fieldOrderByType[typeID] = fieldOrder
	}

	return fieldOrderByType, nil
}

func (d *Decoder) Unmarshal(data any, v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer")
	}
	return d.unmarshalValue(data, val.Elem())
}

func (d *Decoder) unmarshalValue(data any, val reflect.Value) error {
	if data == nil {
		val.Set(reflect.Zero(val.Type()))
		return nil
	}

	arr, ok := data.([]any)
	if !ok {
		// Primitive value
		return setPrimitive(val, data)
	}

	if len(arr) == 0 {
		return nil
	}

	header, ok := arr[0].(float64)
	if !ok {
		return fmt.Errorf("invalid header: %v", arr[0])
	}

	switch int(header) {
	case 1, 0:
		return d.unmarshalStruct(arr, val)
	case 2:
		return d.unmarshalSlice(arr, val)
	case 3:
		return d.unmarshalMap(arr, val)
	default:
		return setPrimitive(val, data)
	}
}

func (d *Decoder) unmarshalMap(arr []any, val reflect.Value) error {
	mapType := val.Type()
	if val.Kind() == reflect.Interface {
		mapType = reflect.TypeOf(map[string]any{})
	} else if val.Kind() != reflect.Map {
		return fmt.Errorf("cannot unmarshal map into %v", val.Type())
	}

	m := reflect.MakeMap(mapType)

	keyType := mapType.Key()
	elemType := mapType.Elem()

	for i := 1; i < len(arr); i += 2 {
		if i+1 >= len(arr) {
			break
		}
		key := reflect.New(keyType).Elem()
		err := d.unmarshalValue(arr[i], key)
		if err != nil {
			return err
		}

		elem := reflect.New(elemType).Elem()
		err = d.unmarshalValue(arr[i+1], elem)
		if err != nil {
			return err
		}

		m.SetMapIndex(key, elem)
	}

	val.Set(m)
	return nil
}

func (d *Decoder) unmarshalStruct(arr []any, val reflect.Value) error {
	header := int(arr[0].(float64))
	var xStruct *xunsafe.Struct
	var skipIndices []int
	var valueStartIdx int
	var typeID int

	if header == 1 {
		if len(arr) < 2 {
			return fmt.Errorf("missing reference block for header 1")
		}
		refBlock, ok := arr[1].([]any)
		if !ok || len(refBlock) == 0 {
			return fmt.Errorf("invalid reference block")
		}
		typeID = int(refBlock[0].(float64))
		xStruct = d.registry.GetStruct(typeID)
		if xStruct == nil {
			return fmt.Errorf("unknown type ID: %d", typeID)
		}
		d.lastType = d.registry.GetType(typeID)
		d.lastTypeID = typeID
		for i := 1; i < len(refBlock); i++ {
			skipIndices = append(skipIndices, int(refBlock[i].(float64)))
		}
		valueStartIdx = 2
	} else {
		// header 0
		if d.lastType == nil {
			return fmt.Errorf("header 0 but no previous type")
		}
		typeID = d.lastTypeID
		xStruct = xunsafe.NewStruct(d.lastType)

		// The encoder guarantees that position 1 of a header-0 record is the skip
		// block whenever it is an array (emitting an empty one when the first value
		// would itself be an array), so no type-based guessing is needed here.
		valueStartIdx = 1
		if len(arr) > 1 {
			if subArr, ok := arr[1].([]any); ok {
				for _, s := range subArr {
					skipIndices = append(skipIndices, int(s.(float64)))
				}
				valueStartIdx = 2
			}
		}
	}

	if val.Kind() != reflect.Struct {
		if val.Kind() == reflect.Interface {
			newVal := reflect.New(d.lastType).Elem()
			err := d.populateStruct(xStruct, typeID, arr[valueStartIdx:], skipIndices, newVal)
			if err != nil {
				return err
			}
			val.Set(newVal)
			return nil
		}
		if val.Kind() == reflect.Ptr {
			// Handle pointer types
			if val.IsNil() {
				val.Set(reflect.New(d.lastType))
			}
			return d.populateStruct(xStruct, typeID, arr[valueStartIdx:], skipIndices, val.Elem())
		}
		return fmt.Errorf("cannot unmarshal struct into %v", val.Type())
	}

	return d.populateStruct(xStruct, typeID, arr[valueStartIdx:], skipIndices, val)
}

func (d *Decoder) populateStruct(xStruct *xunsafe.Struct, typeID int, values []any, skipIndices []int, val reflect.Value) error {
	skipMap := make(map[int]bool)
	for _, idx := range skipIndices {
		skipMap[idx] = true
	}

	// Prefer the order carried by the payload's keys header. Fall back to the registry only
	// when decoding bare content with no header (Decoder.Unmarshal called directly).
	fieldOrder, fromHeader := d.fieldOrderByType[typeID]
	if !fromHeader {
		registryOrder, _ := d.registry.orderFor(typeID)
		if registryOrder != nil {
			fieldOrder = registryOrder
		} else {
			for i := range xStruct.Fields {
				fieldOrder = append(fieldOrder, i)
			}
		}
	}

	valueIdx := 0
	for orderIdx, fieldIdx := range fieldOrder {
		if skipMap[orderIdx] { // Skip indices are in order space, not field space
			continue
		}
		if valueIdx >= len(values) {
			break
		}
		// A header position with no resolvable field still consumes its value slot.
		if fieldIdx < 0 {
			valueIdx++
			continue
		}

		field := &xStruct.Fields[fieldIdx]
		fVal := reflect.New(field.Type).Elem()
		err := d.unmarshalValue(values[valueIdx], fVal)
		if err != nil {
			return err
		}

		val.Field(fieldIdx).Set(fVal)
		valueIdx++
	}
	return nil
}

func (d *Decoder) unmarshalSlice(arr []any, val reflect.Value) error {
	sliceType := val.Type()
	if val.Kind() == reflect.Interface {
		// Default to []any for interface{}
		sliceType = reflect.TypeOf([]any{})
	} else if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return fmt.Errorf("cannot unmarshal slice into %v", val.Type())
	}

	elemType := sliceType.Elem()
	slice := reflect.MakeSlice(sliceType, 0, len(arr)-1)

	for i := 1; i < len(arr); i++ {
		elem := reflect.New(elemType).Elem()
		err := d.unmarshalValue(arr[i], elem)
		if err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}
	val.Set(slice)
	return nil
}
