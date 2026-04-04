package db

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"github.com/fxamacker/cbor/v2"
	"github.com/viant/xunsafe"
)

type colInfo struct {
	Name         string
	FieldName    string
	NameAlias    string
	IsPrimaryKey int8
	FieldIdx     int
	IsVirtual    bool
	HasView      bool
	ViewIdx      int8
	Idx          int16
	RefType      reflect.Type
	Field        *xunsafe.Field
}

type columnInfo struct {
	colInfo
	colType
	hasCollectionTagOptions bool
	getValue                func(ptr unsafe.Pointer) any
	getRawValue             func(ptr unsafe.Pointer) any
	getStatementValue       func(ptr unsafe.Pointer) any
	setValue                func(ptr unsafe.Pointer, v any)
	decimalSize             int8
	autoincrementRandSize   int8
	compositeBucketing      []int8
	isWeek                  bool
	useInt32Packing         bool
	aggregateFn             string
}

func (c *columnInfo) GetValue(ptr unsafe.Pointer) any {
	if c.getValue != nil {
		return c.getValue(ptr)
	}
	return makeScyllaValue(c.Field, ptr, c.Type, c.ColType)
}

func (c *columnInfo) GetRawValue(ptr unsafe.Pointer) any {
	if c.getRawValue != nil {
		return c.getRawValue(ptr)
	}
	if c.Field == nil {
		return nil
	}
	if c.IsPointer {
		if c.Field.IsNil(ptr) {
			return nil
		}
	}
	return c.Field.Interface(ptr)
}

func (c *columnInfo) GetStatementValue(ptr unsafe.Pointer) any {
	if c.getStatementValue != nil {
		return c.getStatementValue(ptr)
	}
	if c.getRawValue != nil {
		return c.getRawValue(ptr)
	}
	if c.getValue != nil {
		return c.getValue(ptr)
	}
	if c.Field == nil {
		return nil
	}
	if c.IsPointer {
		if c.Field.IsNil(ptr) {
			return nil
		}
		return c.Field.Interface(ptr)
	}
	// Unsupported unsigned types are stored as raw blob bytes instead of numeric CQL types.
	if c.Type == 9 {
		if encodedBlob, encoded, err := encodeUnsignedValueToBlob(c.Field.Interface(ptr), c.RefType); encoded {
			if err != nil {
				fmt.Println("Error encoding unsigned blob:", c.FieldName, err)
				return nil
			}
			return encodedBlob
		}
	}
	if c.IsComplexType {
		fieldValue := c.Field.Interface(ptr)
		recordBytes, err := cbor.Marshal(fieldValue)
		if err != nil {
			fmt.Println("Error al encodeding .cbor:: ", c.FieldName, err)
			return ""
		}
		return recordBytes
	}
	return c.Field.Interface(ptr)
}

func (c *columnInfo) SetValue(ptr unsafe.Pointer, v any) {
	if c.setValue != nil {
		c.setValue(ptr, v)
		return
	}
	if c.Field == nil {
		return
	}
	if c.Type == 9 {
		if decodedValue, decoded, err := decodeUnsignedValueFromBlob(v, c.RefType); decoded {
			if err != nil {
				// Keep backward compatibility with legacy CBOR blobs when binary decode is not possible.
				fmt.Printf("Error decoding unsigned blob for Col %s, trying legacy CBOR: %v\n", c.Name, err)
			} else {
				// xunsafe generic Set does not reliably assign []uint16 slices; use direct typed memory assignment.
				destination := reflect.NewAt(c.RefType, c.Field.Pointer(ptr)).Elem()
				decodedReflectValue := reflect.ValueOf(decodedValue)
				if decodedReflectValue.IsValid() {
					if decodedReflectValue.Type().AssignableTo(c.RefType) {
						destination.Set(decodedReflectValue)
						return
					}
					if decodedReflectValue.Type().ConvertibleTo(c.RefType) {
						destination.Set(decodedReflectValue.Convert(c.RefType))
						return
					}
				}
			}
		}

		var vl []byte
		if b, ok := v.(*[]byte); ok {
			vl = *b
		} else if b, ok := v.([]byte); ok {
			vl = b
		}

		if len(vl) > 3 && c.Field != nil {
			// Direct unmarshal into the field memory using xunsafe pointer
			dest := reflect.NewAt(c.RefType, c.Field.Pointer(ptr)).Interface()
			err := cbor.Unmarshal(vl, dest)
			if err != nil {
				fmt.Printf("Error al convertir ComplexType for Col %s: %v\n", c.Name, err)
			}
		} else if ShouldLog() {
			fmt.Printf("Complex Type could not be parsed or empty: %s (Type: %T)\n", c.Name, v)
		}
	} else {
		assingValue(c.Field, ptr, c.Type, v)
	}
}

func (c *columnInfo) compileFastAccessors() {
	if c.Field == nil {
		return
	}
	// Do not override custom virtual-key/view accessors that already encode business-specific logic.
	if c.getRawValue != nil || c.getValue != nil || c.getStatementValue != nil || c.setValue != nil {
		return
	}

	switch c.Type {
	// Scalar fast paths use direct xunsafe typed accessors to avoid interface boxing
	// and repeated type-switch work in hot scan/write loops.
	case 1: // string
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.String(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetString(ptr, coerceString(v)) }
	case 2: // int64
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.Int64(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetInt64(ptr, reflectToInt64(v)) }
	case 3: // int32
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.Int32(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetInt32(ptr, int32(reflectToInt64(v))) }
	case 4: // int16
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.Int16(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetInt16(ptr, int16(reflectToInt64(v))) }
	case 5: // int8
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.Int8(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetInt8(ptr, int8(reflectToInt64(v))) }
	case 6: // float32
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.Float32(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetFloat32(ptr, reflectToFloat32(v)) }
	case 7: // float64
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.Float64(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetFloat64(ptr, reflectToFloat64(v)) }
	case 8: // bool
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.Bool(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetBool(ptr, coerceBool(v)) }
	case 10: // int
		c.getRawValue = func(ptr unsafe.Pointer) any { return c.Field.Int(ptr) }
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) { c.Field.SetInt(ptr, int(reflectToInt64(v))) }
	// Hot slice types get exact-type fast setters and SQL literal builders.
	// Any non-exact input type intentionally falls back to generic conversion.
	case 11: // []string
		c.getRawValue = func(ptr unsafe.Pointer) any { return *(*[]string)(c.Field.Pointer(ptr)) }
		c.getStatementValue = c.getRawValue
		// Keep direct SQL statement builders fast for []string-heavy write paths.
		c.getValue = func(ptr unsafe.Pointer) any {
			return makeStringCollectionLiteral(c.ColType, *(*[]string)(c.Field.Pointer(ptr)))
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []string:
				setField(c.Field, ptr, slices.Clone(typedValue))
				return
			case *[]string:
				if typedValue == nil {
					setField(c.Field, ptr, []string(nil))
					return
				}
				setField(c.Field, ptr, slices.Clone(*typedValue))
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 12: // []int64
		c.getRawValue = func(ptr unsafe.Pointer) any { return *(*[]int64)(c.Field.Pointer(ptr)) }
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			return makeSignedIntCollectionLiteral(c.ColType, *(*[]int64)(c.Field.Pointer(ptr)))
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []int64:
				setField(c.Field, ptr, slices.Clone(typedValue))
				return
			case *[]int64:
				if typedValue == nil {
					setField(c.Field, ptr, []int64(nil))
					return
				}
				setField(c.Field, ptr, slices.Clone(*typedValue))
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 13: // []int32
		c.getRawValue = func(ptr unsafe.Pointer) any { return *(*[]int32)(c.Field.Pointer(ptr)) }
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			return makeSignedIntCollectionLiteral(c.ColType, *(*[]int32)(c.Field.Pointer(ptr)))
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []int32:
				setField(c.Field, ptr, slices.Clone(typedValue))
				return
			case *[]int32:
				if typedValue == nil {
					setField(c.Field, ptr, []int32(nil))
					return
				}
				setField(c.Field, ptr, slices.Clone(*typedValue))
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 14: // []int16
		c.getRawValue = func(ptr unsafe.Pointer) any { return *(*[]int16)(c.Field.Pointer(ptr)) }
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			return makeSignedIntCollectionLiteral(c.ColType, *(*[]int16)(c.Field.Pointer(ptr)))
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []int16:
				setField(c.Field, ptr, slices.Clone(typedValue))
				return
			case *[]int16:
				if typedValue == nil {
					setField(c.Field, ptr, []int16(nil))
					return
				}
				setField(c.Field, ptr, slices.Clone(*typedValue))
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 15: // []int8
		c.getRawValue = func(ptr unsafe.Pointer) any { return *(*[]int8)(c.Field.Pointer(ptr)) }
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			return makeSignedIntCollectionLiteral(c.ColType, *(*[]int8)(c.Field.Pointer(ptr)))
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []int8:
				setField(c.Field, ptr, slices.Clone(typedValue))
				return
			case *[]int8:
				if typedValue == nil {
					setField(c.Field, ptr, []int8(nil))
					return
				}
				setField(c.Field, ptr, slices.Clone(*typedValue))
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	// Pointer scalar paths preserve nil semantics and avoid reflection.Interface calls.
	// Fast setters cover exact common assignments while keeping fallback compatibility.
	case 21: // *string
		c.getRawValue = func(ptr unsafe.Pointer) any {
			stringPointer := *(**string)(c.Field.Pointer(ptr))
			if stringPointer == nil {
				return nil
			}
			return stringPointer
		}
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case string:
				stringValue := typedValue
				setField(c.Field, ptr, &stringValue)
				return
			case *string:
				setField(c.Field, ptr, typedValue)
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 22: // *int64
		c.getRawValue = func(ptr unsafe.Pointer) any {
			int64Pointer := *(**int64)(c.Field.Pointer(ptr))
			if int64Pointer == nil {
				return nil
			}
			return int64Pointer
		}
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) {
			int64Value := reflectToInt64(v)
			setField(c.Field, ptr, &int64Value)
		}
	case 23: // *int32
		c.getRawValue = func(ptr unsafe.Pointer) any {
			int32Pointer := *(**int32)(c.Field.Pointer(ptr))
			if int32Pointer == nil {
				return nil
			}
			return int32Pointer
		}
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) {
			int32Value := int32(reflectToInt64(v))
			setField(c.Field, ptr, &int32Value)
		}
	case 24: // *int16
		c.getRawValue = func(ptr unsafe.Pointer) any {
			int16Pointer := *(**int16)(c.Field.Pointer(ptr))
			if int16Pointer == nil {
				return nil
			}
			return int16Pointer
		}
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) {
			int16Value := int16(reflectToInt64(v))
			setField(c.Field, ptr, &int16Value)
		}
	case 25: // *int8
		c.getRawValue = func(ptr unsafe.Pointer) any {
			int8Pointer := *(**int8)(c.Field.Pointer(ptr))
			if int8Pointer == nil {
				return nil
			}
			return int8Pointer
		}
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) {
			int8Value := int8(reflectToInt64(v))
			setField(c.Field, ptr, &int8Value)
		}
	case 26: // *float32
		c.getRawValue = func(ptr unsafe.Pointer) any {
			float32Pointer := *(**float32)(c.Field.Pointer(ptr))
			if float32Pointer == nil {
				return nil
			}
			return float32Pointer
		}
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) {
			float32Value := reflectToFloat32(v)
			setField(c.Field, ptr, &float32Value)
		}
	case 27: // *float64
		c.getRawValue = func(ptr unsafe.Pointer) any {
			float64Pointer := *(**float64)(c.Field.Pointer(ptr))
			if float64Pointer == nil {
				return nil
			}
			return float64Pointer
		}
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) {
			float64Value := reflectToFloat64(v)
			setField(c.Field, ptr, &float64Value)
		}
	case 28: // *int
		c.getRawValue = func(ptr unsafe.Pointer) any {
			intPointer := *(**int)(c.Field.Pointer(ptr))
			if intPointer == nil {
				return nil
			}
			return intPointer
		}
		c.getStatementValue = c.getRawValue
		c.setValue = func(ptr unsafe.Pointer, v any) {
			intValue := int(reflectToInt64(v))
			setField(c.Field, ptr, &intValue)
		}
	// Pointer-to-slice paths mirror slice optimizations but retain nil pointer identity.
	// We clone on write to avoid aliasing caller-owned backing arrays.
	case 31: // *[]string
		c.getRawValue = func(ptr unsafe.Pointer) any {
			stringSlicePointer := *(**[]string)(c.Field.Pointer(ptr))
			if stringSlicePointer == nil {
				return nil
			}
			return stringSlicePointer
		}
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			stringSlicePointer := *(**[]string)(c.Field.Pointer(ptr))
			if stringSlicePointer == nil {
				return nil
			}
			return makeStringCollectionLiteral(c.ColType, *stringSlicePointer)
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []string:
				stringSliceValue := slices.Clone(typedValue)
				setField(c.Field, ptr, &stringSliceValue)
				return
			case *[]string:
				if typedValue == nil {
					setField(c.Field, ptr, (*[]string)(nil))
					return
				}
				stringSliceValue := slices.Clone(*typedValue)
				setField(c.Field, ptr, &stringSliceValue)
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 32: // *[]int64
		c.getRawValue = func(ptr unsafe.Pointer) any {
			int64SlicePointer := *(**[]int64)(c.Field.Pointer(ptr))
			if int64SlicePointer == nil {
				return nil
			}
			return int64SlicePointer
		}
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			int64SlicePointer := *(**[]int64)(c.Field.Pointer(ptr))
			if int64SlicePointer == nil {
				return nil
			}
			return makeSignedIntCollectionLiteral(c.ColType, *int64SlicePointer)
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []int64:
				int64SliceValue := slices.Clone(typedValue)
				setField(c.Field, ptr, &int64SliceValue)
				return
			case *[]int64:
				if typedValue == nil {
					setField(c.Field, ptr, (*[]int64)(nil))
					return
				}
				int64SliceValue := slices.Clone(*typedValue)
				setField(c.Field, ptr, &int64SliceValue)
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 33: // *[]int32
		c.getRawValue = func(ptr unsafe.Pointer) any {
			int32SlicePointer := *(**[]int32)(c.Field.Pointer(ptr))
			if int32SlicePointer == nil {
				return nil
			}
			return int32SlicePointer
		}
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			int32SlicePointer := *(**[]int32)(c.Field.Pointer(ptr))
			if int32SlicePointer == nil {
				return nil
			}
			return makeSignedIntCollectionLiteral(c.ColType, *int32SlicePointer)
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []int32:
				int32SliceValue := slices.Clone(typedValue)
				setField(c.Field, ptr, &int32SliceValue)
				return
			case *[]int32:
				if typedValue == nil {
					setField(c.Field, ptr, (*[]int32)(nil))
					return
				}
				int32SliceValue := slices.Clone(*typedValue)
				setField(c.Field, ptr, &int32SliceValue)
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 34: // *[]int16
		c.getRawValue = func(ptr unsafe.Pointer) any {
			int16SlicePointer := *(**[]int16)(c.Field.Pointer(ptr))
			if int16SlicePointer == nil {
				return nil
			}
			return int16SlicePointer
		}
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			int16SlicePointer := *(**[]int16)(c.Field.Pointer(ptr))
			if int16SlicePointer == nil {
				return nil
			}
			return makeSignedIntCollectionLiteral(c.ColType, *int16SlicePointer)
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []int16:
				int16SliceValue := slices.Clone(typedValue)
				setField(c.Field, ptr, &int16SliceValue)
				return
			case *[]int16:
				if typedValue == nil {
					setField(c.Field, ptr, (*[]int16)(nil))
					return
				}
				int16SliceValue := slices.Clone(*typedValue)
				setField(c.Field, ptr, &int16SliceValue)
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 35: // *[]int8
		c.getRawValue = func(ptr unsafe.Pointer) any {
			int8SlicePointer := *(**[]int8)(c.Field.Pointer(ptr))
			if int8SlicePointer == nil {
				return nil
			}
			return int8SlicePointer
		}
		c.getStatementValue = c.getRawValue
		c.getValue = func(ptr unsafe.Pointer) any {
			int8SlicePointer := *(**[]int8)(c.Field.Pointer(ptr))
			if int8SlicePointer == nil {
				return nil
			}
			return makeSignedIntCollectionLiteral(c.ColType, *int8SlicePointer)
		}
		c.setValue = func(ptr unsafe.Pointer, v any) {
			switch typedValue := v.(type) {
			case []int8:
				int8SliceValue := slices.Clone(typedValue)
				setField(c.Field, ptr, &int8SliceValue)
				return
			case *[]int8:
				if typedValue == nil {
					setField(c.Field, ptr, (*[]int8)(nil))
					return
				}
				int8SliceValue := slices.Clone(*typedValue)
				setField(c.Field, ptr, &int8SliceValue)
				return
			}
			assingValue(c.Field, ptr, c.Type, v)
		}
	case 36, 37:
		// Less-used float pointer-slice types stay generic to keep fast-path code size controlled.
		// Pointer/slice pointer columns keep generic conversion to preserve legacy nil/value semantics.
		c.getRawValue = func(ptr unsafe.Pointer) any {
			if c.Field.IsNil(ptr) {
				return nil
			}
			return c.Field.Interface(ptr)
		}
		c.getStatementValue = c.getRawValue
	}
}

func coerceString(value any) string {
	switch typedValue := value.(type) {
	case string:
		return typedValue
	case *string:
		if typedValue != nil {
			return *typedValue
		}
	}
	return ""
}

func coerceBool(value any) bool {
	switch typedValue := value.(type) {
	case bool:
		return typedValue
	case *bool:
		if typedValue != nil {
			return *typedValue
		}
	}
	return false
}

func makeStringCollectionLiteral(collectionColType string, values []string) string {
	openBracket, closeBracket := getCollectionLiteralBrackets(collectionColType)
	stringValuesQuoted := make([]string, len(values))
	for valueIndex, currentValue := range values {
		stringValuesQuoted[valueIndex] = "'" + currentValue + "'"
	}
	return openBracket + strings.Join(stringValuesQuoted, ",") + closeBracket
}

func makeSignedIntCollectionLiteral[T ~int64 | ~int32 | ~int16 | ~int8](collectionColType string, values []T) string {
	openBracket, closeBracket := getCollectionLiteralBrackets(collectionColType)
	if len(values) == 0 {
		return openBracket + closeBracket
	}
	var statementBuilder strings.Builder
	statementBuilder.Grow(len(values) * 4)
	statementBuilder.WriteString(openBracket)
	for valueIndex, currentValue := range values {
		if valueIndex > 0 {
			statementBuilder.WriteByte(',')
		}
		statementBuilder.WriteString(strconv.FormatInt(int64(currentValue), 10))
	}
	statementBuilder.WriteString(closeBracket)
	return statementBuilder.String()
}

func getCollectionLiteralBrackets(collectionColType string) (string, string) {
	normalizedCollectionType := strings.ToLower(unwrapFrozenCollectionType(collectionColType))
	if strings.HasPrefix(normalizedCollectionType, "list<") {
		return "[", "]"
	}
	return "{", "}"
}

func (c *columnInfo) GetType() *colType {
	return &c.colType
}

func (c *columnInfo) GetName() string {
	return c.Name
}

func (c *columnInfo) GetInfo() *colInfo {
	return &c.colInfo
}

func (c *columnInfo) IsNil() bool {
	return c == nil
}

func (c *columnInfo) SetAutoincrementRandSize(size int8) {
	c.autoincrementRandSize = size
}

func (c *columnInfo) SetDecimalSize(size int8) {
	c.decimalSize = size
}
