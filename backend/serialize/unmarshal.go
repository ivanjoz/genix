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

	// arr[0] contains the keys (type definitions) - can be used for self-describing data
	// arr[1] contains the actual content
	content, ok := arr[1].([]any)
	if !ok {
		return fmt.Errorf("invalid content format: expected array")
	}

	d := NewDecoder()
	return d.Unmarshal(content, v)
}

func (d *Decoder) Unmarshal(data []any, v any) error {
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
	default:
		return setPrimitive(val, data)
	}
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

		// Check for optional skip block
		if len(arr) > 1 {
			if subArr, ok := arr[1].([]any); ok {
				isSkipBlock := false
				firstField := &xStruct.Fields[0]
				firstFieldKind := firstField.Type.Kind()
				if firstFieldKind == reflect.Ptr {
					firstFieldKind = firstField.Type.Elem().Kind()
				}

				if firstFieldKind != reflect.Slice && firstFieldKind != reflect.Array && firstFieldKind != reflect.Struct {
					isSkipBlock = true
				} else if len(subArr) > 0 {
					if h, ok := subArr[0].(float64); ok {
						if h != 0 && h != 1 && h != 2 {
							isSkipBlock = true
						}
					} else {
						isSkipBlock = true
					}
				}

				if isSkipBlock {
					for _, s := range subArr {
						skipIndices = append(skipIndices, int(s.(float64)))
					}
					valueStartIdx = 2
				} else {
					valueStartIdx = 1
				}
			} else {
				valueStartIdx = 1
			}
		} else {
			valueStartIdx = 1
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

	// Get the optimized order if available
	typeInfo := d.registry.GetTypeInfo(typeID)
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

	valPtr := xunsafe.AsPointer(val.Addr().Interface())
	valueIdx := 0
	for orderIdx, fieldIdx := range fieldOrder {
		if skipMap[orderIdx] { // Skip indices are in order space, not field space
			continue
		}
		if valueIdx >= len(values) {
			break
		}

		field := &xStruct.Fields[fieldIdx]
		fVal := reflect.New(field.Type).Elem()
		err := d.unmarshalValue(values[valueIdx], fVal)
		if err != nil {
			return err
		}

		field.SetValue(valPtr, fVal.Interface())
		valueIdx++
	}
	return nil
}

func (d *Decoder) unmarshalSlice(arr []any, val reflect.Value) error {
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return fmt.Errorf("cannot unmarshal slice into %v", val.Type())
	}

	elemType := val.Type().Elem()
	slice := reflect.MakeSlice(val.Type(), 0, len(arr)-1)

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
