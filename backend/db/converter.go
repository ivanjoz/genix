package db

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/fxamacker/cbor/v2"
	"github.com/viant/xunsafe"
)

type colType struct {
	Type          int8
	FieldType     string
	ColType       string
	IsSlice       bool
	IsPointer     bool
	IsComplexType bool
}

// It must me no "int", is int32 or int64
var colTypes = []colType{
	{1, "string", "text", false, false, false},
	{2, "int64", "bigint", false, false, false},
	{3, "int32", "int", false, false, false},
	{4, "int16", "smallint", false, false, false},
	{5, "int8", "tinyint", false, false, false},
	{6, "float32", "float", false, false, false},
	{7, "float64", "double", false, false, false},
	{8, "bool", "boolean", false, false, false},
	{9, "[]byte", "blob", false, false, true},
	{11, "[]string", "set<text>", true, false, false},
	{12, "[]int64", "set<bigint>", true, false, false},
	{13, "[]int32", "set<int>", true, false, false},
	{14, "[]int16", "set<smallint>", true, false, false},
	{15, "[]int8", "set<tinyint>", true, false, false},
	{16, "[]float32", "set<float>", true, false, false},
	{17, "[]float64", "set<double>", true, false, false},
	{21, "*string", "text", false, true, false},
	{22, "*int64", "bigint", false, true, false},
	{23, "*int32", "int", false, true, false},
	{24, "*int16", "smallint", false, true, false},
	{25, "*int8", "tinyint", false, true, false},
	{26, "*float32", "float", false, true, false},
	{27, "*float64", "double", false, true, false},
	{31, "*[]string", "set<text>", true, true, false},
	{32, "*[]int64", "set<bigint>", true, true, false},
	{33, "*[]int32", "set<int>", true, true, false},
	{34, "*[]int16", "set<smallint>", true, true, false},
	{35, "*[]int8", "set<tinyint>", true, true, false},
	{36, "*[]float32", "set<float>", true, true, false},
	{37, "*[]float64", "set<double>", true, true, false},
}

var colTypesMap = map[int8]colType{}
var colTypesFieldMap = map[string]colType{}
var colTypesDBMap = map[string]colType{}
var initColTypesOnce sync.Once

type number1 interface {
	int | int32 | int8 | uint8 | int16 | uint16 | int64 | float32 | float64
}

func setField[T any](f *xunsafe.Field, ptr unsafe.Pointer, val T) {
	*(*T)(f.Pointer(ptr)) = val
}

func setReflectIntSlice[T number1, E number1](f *xunsafe.Field, ptr unsafe.Pointer, vl *[]E) {
	newSlice := make([]T, len(*vl))
	for i, v := range *vl {
		newSlice[i] = T(v)
	}
	setField(f, ptr, newSlice)
}

func setReflectIntSlicePointer[T number1, E number1](f *xunsafe.Field, ptr unsafe.Pointer, vl *[]E) {
	newSlice := make([]T, len(*vl))
	for i, v := range *vl {
		newSlice[i] = T(v)
	}
	setField(f, ptr, &newSlice)
}

func printError(valType string, v any) {
	fmt.Printf("Error: El valor %v no fue mapeado = %v | %T\n", valType, v, v)
	panic("!")
}

func GetColTypeByID(typeID int8) colType {
	initColTypesOnce.Do(func() {
		for _, ct := range colTypes {
			colTypesMap[ct.Type] = ct
			colTypesFieldMap[ct.FieldType] = ct
			colTypesFieldMap["*"+ct.FieldType] = ct
			colTypesDBMap[ct.ColType] = ct
		}
	})
	return colTypesMap[typeID]
}

func GetColTypeByName(fieldName string, dbName string) colType {
	GetColTypeByID(0)
	if fieldName != "" {
		return colTypesFieldMap[fieldName]
	} else if dbName != "" {
		return colTypesDBMap[dbName]
	}
	return colType{}
}

// Number constraint to cover all numeric types requested
type Number interface {
	~int | ~int64 | ~int32 | ~int16 | ~int8 | ~float64 | ~float32
}

// Generic function to append any numeric slice in {val1, val2} format
func makeNumericSlice[T Number](slice []T) []byte {
	dst := []byte{}

	for i, v := range slice {
		if i > 0 {
			dst = append(dst, ',')
		}
		// Since we need to call specific strconv functions,
		// we switch on the type one more time inside the generic.
		switch val := any(v).(type) {
		case int:
			dst = append(dst, Int64ToBase64Bytes(int64(val))...)
		case int64:
			dst = append(dst, Int64ToBase64Bytes(val)...)
		case int32:
			dst = append(dst, Int64ToBase64Bytes(int64(val))...)
		case int16:
			dst = append(dst, Int64ToBase64Bytes(int64(val))...)
		case int8:
			dst = append(dst, Int64ToBase64Bytes(int64(val))...)
		case float64:
			dst = strconv.AppendFloat(dst, val, 'f', -1, 64)
		case float32:
			dst = strconv.AppendFloat(dst, float64(val), 'f', -1, 32)
		}
	}
	return dst
}

// Helper to convert any integer constraint to int64 for strconv
func reflectToInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case *int:
		return int64(*t)
	case int64:
		return t
	case *int64:
		return *t
	case int32:
		return int64(t)
	case *int32:
		return int64(*t)
	case int16:
		return int64(t)
	case *int16:
		return int64(*t)
	case int8:
		return int64(t)
	case *int8:
		return int64(*t)
	}
	return 0
}

func reflectToFloat64(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case *float64:
		return *t
	case float32:
		return float64(t)
	case *float32:
		return float64(*t)
	}
	return 0
}

func reflectToFloat32(v any) float32 {
	switch t := v.(type) {
	case float32:
		return t
	case *float32:
		return *t
	case float64:
		return float32(t)
	case *float64:
		return float32(*t)
	}
	return 0
}

const pipeByte byte = '|'
const specialByte byte = '`'
const backslashByte byte = '\\'

// | is replaced by '`'
// \ is replaced by '```'
// '``' is used to contatenate 2 strings

func sanitizeString(value string) []byte {
	dst := []byte{}

	for _, b := range []byte(value) {
		switch b {
		case pipeByte:
			dst = append(dst, specialByte)
		case backslashByte:
			dst = append(dst, specialByte, specialByte, specialByte)
		default:
			dst = append(dst, b)
		}
	}
	return dst
}

func unSanitizeString(value string) string {
	dst := []byte{}
	src := []byte(value)
	for i := 0; i < len(src); i++ {
		if src[i] == specialByte {
			if i+2 < len(src) && src[i+1] == specialByte && src[i+2] == specialByte {
				dst = append(dst, backslashByte)
				i += 2
			} else {
				dst = append(dst, pipeByte)
			}
		} else {
			dst = append(dst, src[i])
		}
	}
	return string(dst)
}

const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-/"

func Int64ToBase64Bytes(n int64) []byte {
	if n == 0 {
		return []byte{base64Chars[0]}
	}

	// 1. Determine if negative and use absolute value
	isNegative := false
	un := uint64(n)
	if n < 0 {
		isNegative = true
		// Handle math.MinInt64 edge case by converting to uint64 first
		un = uint64(-n)
	}

	// 2. Max length: 11 chars for magnitude + 1 for sign = 12
	var buf [12]byte
	i := 12

	// 3. Mathematical Base64 conversion
	for un > 0 {
		i--
		buf[i] = base64Chars[un%64]
		un /= 64
	}

	// 4. Add the minus sign if necessary
	if isNegative {
		i--
		buf[i] = '-'
	}

	// Return only the occupied part of the buffer
	// We create a copy to avoid the buffer escaping to heap if needed
	result := make([]byte, 12-i)
	copy(result, buf[i:])
	return result
}

func Base64BytesToInt64(b []byte) int64 {
	if len(b) == 0 {
		return 0
	}
	isNegative := false
	if b[0] == '-' {
		isNegative = true
		b = b[1:]
	}
	var res uint64
	for _, char := range b {
		idx := strings.IndexByte(base64Chars, char)
		if idx == -1 {
			continue
		}
		res = res*64 + uint64(idx)
	}
	if isNegative {
		return -int64(res)
	}
	return int64(res)
}

func valueToCSVBase64(val any) []byte {

	if val == nil {
		return []byte{}
	}

	switch v := val.(type) {
	// Individual Numbers
	case int:
		return Int64ToBase64Bytes(int64(v))
	case *int:
		return Int64ToBase64Bytes(int64(*v))
	case int64:
		return Int64ToBase64Bytes(v)
	case *int64:
		return Int64ToBase64Bytes(*v)
	case int32:
		return Int64ToBase64Bytes(int64(v))
	case *int32:
		return Int64ToBase64Bytes(int64(*v))
	case int16:
		return Int64ToBase64Bytes(int64(v))
	case *int16:
		return Int64ToBase64Bytes(int64(*v))
	case int8:
		return Int64ToBase64Bytes(int64(v))
	case *int8:
		return Int64ToBase64Bytes(int64(*v))
	case float64:
		return strconv.AppendFloat([]byte{}, v, 'f', -1, 64)
	case *float64:
		return strconv.AppendFloat([]byte{}, *v, 'f', -1, 64)
	case float32:
		return strconv.AppendFloat([]byte{}, float64(v), 'f', -1, 32)
	case *float32:
		return strconv.AppendFloat([]byte{}, float64(*v), 'f', -1, 32)

	// Slices using our Generic Function
	case []int:
		return makeNumericSlice(v)
	case *[]int:
		return makeNumericSlice(*v)
	case []int64:
		return makeNumericSlice(v)
	case *[]int64:
		return makeNumericSlice(*v)
	case []int32:
		return makeNumericSlice(v)
	case *[]int32:
		return makeNumericSlice(*v)
	case []float64:
		return makeNumericSlice(v)
	case *[]float64:
		return makeNumericSlice(*v)
	case []float32:
		return makeNumericSlice(v)
	case *[]float32:
		return makeNumericSlice(*v)

	// String & String Slices
	case string:
		return sanitizeString(v)
	case *string:
		return sanitizeString(*v)
	case []string:
		dst := []byte{}
		for i, s := range v {
			if i > 0 {
				dst = append(dst, '`', '`')
			}
			dst = append(dst, sanitizeString(s)...)
		}
		return dst
	case *[]string:
		dst := []byte{}
		for i, s := range *v {
			if i > 0 {
				dst = append(dst, '`', '`')
			}
			dst = append(dst, sanitizeString(s)...)
		}
		return dst
	case []byte:
		dst := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
		base64.StdEncoding.Encode(dst, v)
		return dst
	case *[]byte:
		dst := make([]byte, base64.StdEncoding.EncodedLen(len(*v)))
		base64.StdEncoding.Encode(dst, *v)
		return dst
	case bool:
		if v {
			return []byte{'1'}
		}
		return []byte{'0'}
	case *bool:
		if *v {
			return []byte{'1'}
		}
		return []byte{'0'}
	default:
		return []byte(fmt.Sprintf("%v", v))
	}
}

func base64CSVStringToValue(val string, valType int8) any {
	if val == "" {
		return nil
	}

	switch valType {
	case 1: // string
		return unSanitizeString(val)
	case 2, 3, 4, 5: // int64, int32, int16, int8
		return Base64BytesToInt64([]byte(val))
	case 6, 7: // float32, float64
		f, _ := strconv.ParseFloat(val, 64)
		if valType == 6 {
			return float32(f)
		}
		return f
	case 8: // bool
		return val == "1"
	case 9: // []byte
		recordBytes, _ := base64.StdEncoding.DecodeString(val)
		return recordBytes
	case 11: // []string
		parts := strings.Split(val, "``")
		res := make([]string, len(parts))
		for i, p := range parts {
			res[i] = unSanitizeString(p)
		}
		return res
	case 12, 13, 14, 15: // []int64, []int32, []int16, []int8
		fmt.Println("slice value:", val)
		parts := strings.Split(val, ",")
		switch valType {
		case 12:
			res := make([]int64, len(parts))
			for i, p := range parts {
				res[i] = Base64BytesToInt64([]byte(p))
			}
			return res
		case 13:
			res := make([]int32, len(parts))
			for i, p := range parts {
				res[i] = int32(Base64BytesToInt64([]byte(p)))
			}
			return res
		case 14:
			res := make([]int16, len(parts))
			for i, p := range parts {
				res[i] = int16(Base64BytesToInt64([]byte(p)))
			}
			return res
		default:
			res := make([]int8, len(parts))
			for i, p := range parts {
				res[i] = int8(Base64BytesToInt64([]byte(p)))
			}
			return res
		}
	case 16, 17: // []float32, []float64
		parts := strings.Split(val, ",")
		if valType == 16 {
			res := make([]float32, len(parts))
			for i, p := range parts {
				f, _ := strconv.ParseFloat(p, 32)
				res[i] = float32(f)
			}
			return res
		} else {
			res := make([]float64, len(parts))
			for i, p := range parts {
				f, _ := strconv.ParseFloat(p, 64)
				res[i] = f
			}
			return res
		}
	}

	return nil
}

func trySetNumberSlice[T number1](f *xunsafe.Field, ptr unsafe.Pointer, colType int8, value any) {
	switch vl := value.(type) {
	case []int:
		setReflectIntSlice[T](f, ptr, &vl)
	case *[]int:
		setReflectIntSlice[T](f, ptr, vl)
	case []int64:
		setReflectIntSlice[T](f, ptr, &vl)
	case *[]int64:
		setReflectIntSlice[T](f, ptr, vl)
	case []int32:
		setReflectIntSlice[T](f, ptr, &vl)
	case *[]int32:
		setReflectIntSlice[T](f, ptr, vl)
	case []int16:
		setReflectIntSlice[T](f, ptr, &vl)
	case *[]int16:
		setReflectIntSlice[T](f, ptr, vl)
	case []int8:
		setReflectIntSlice[T](f, ptr, &vl)
	case *[]int8:
		setReflectIntSlice[T](f, ptr, vl)
	case []float64:
		setReflectIntSlice[T](f, ptr, &vl)
	case *[]float64:
		setReflectIntSlice[T](f, ptr, vl)
	case []float32:
		setReflectIntSlice[T](f, ptr, &vl)
	case *[]float32:
		setReflectIntSlice[T](f, ptr, vl)
	default:
		printError(GetColTypeByID(colType).FieldType, value)
	}
}

func trySetNumberSlicePointer[T number1](f *xunsafe.Field, ptr unsafe.Pointer, colType int8, value any) {
	switch vl := value.(type) {
	case []int:
		setReflectIntSlicePointer[T](f, ptr, &vl)
	case *[]int:
		setReflectIntSlicePointer[T](f, ptr, vl)
	case []int64:
		setReflectIntSlicePointer[T](f, ptr, &vl)
	case *[]int64:
		setReflectIntSlicePointer[T](f, ptr, vl)
	case []int32:
		setReflectIntSlicePointer[T](f, ptr, &vl)
	case *[]int32:
		setReflectIntSlicePointer[T](f, ptr, vl)
	case []int16:
		setReflectIntSlicePointer[T](f, ptr, &vl)
	case *[]int16:
		setReflectIntSlicePointer[T](f, ptr, vl)
	case []int8:
		setReflectIntSlicePointer[T](f, ptr, &vl)
	case *[]int8:
		setReflectIntSlicePointer[T](f, ptr, vl)
	case []float64:
		setReflectIntSlicePointer[T](f, ptr, &vl)
	case *[]float64:
		setReflectIntSlicePointer[T](f, ptr, vl)
	case []float32:
		setReflectIntSlicePointer[T](f, ptr, &vl)
	case *[]float32:
		setReflectIntSlicePointer[T](f, ptr, vl)
	default:
		printError(GetColTypeByID(colType).FieldType, value)
	}
}

func assingValue(f *xunsafe.Field, ptr unsafe.Pointer, colType int8, value any) {
	if value == nil {
		return
	}

	switch colType {
	case 1: // string
		if vl, ok := value.(string); ok {
			f.SetString(ptr, vl)
		} else if vl, ok := value.(*string); ok {
			f.SetString(ptr, *vl)
		} else {
			printError(GetColTypeByID(colType).FieldType, value)
		}
	case 2: // int64
		f.SetInt64(ptr, reflectToInt64(value))
	case 3: // int32
		f.SetInt32(ptr, int32(reflectToInt64(value)))
	case 4: // int16
		f.SetInt16(ptr, int16(reflectToInt64(value)))
	case 5: // int8
		f.SetInt8(ptr, int8(reflectToInt64(value)))
	case 6: // float32
		f.SetFloat32(ptr, reflectToFloat32(value))
	case 7: // float64
		f.SetFloat64(ptr, reflectToFloat64(value))
	case 8: // bool
		if vl, ok := value.(bool); ok {
			f.SetBool(ptr, vl)
		} else if vl, ok := value.(*bool); ok {
			f.SetBool(ptr, *vl)
		} else {
			printError(GetColTypeByID(colType).FieldType, value)
		}
	case 9: // IsComplexType = true | []byte as cbor
		if vl, ok := value.([]byte); ok {
			f.Set(ptr, vl)
		} else if vl, ok := value.(*[]byte); ok {
			f.Set(ptr, *vl)
		} else {
			printError(GetColTypeByID(colType).FieldType, value)
		}
	case 10: // int
		f.SetInt(ptr, int(reflectToInt64(value)))

	case 11: // []string
		if vl, ok := value.([]string); ok {
			f.Set(ptr, slices.Clone(vl))
		} else if vl, ok := value.(*[]string); ok {
			f.Set(ptr, slices.Clone(*vl))
		} else {
			printError(GetColTypeByID(colType).FieldType, value)
		}
	case 12: // []int64
		trySetNumberSlice[int64](f, ptr, colType, value)
	case 13: // []int32
		trySetNumberSlice[int32](f, ptr, colType, value)
	case 14: // []int16
		trySetNumberSlice[int16](f, ptr, colType, value)
	case 15: // []int8
		trySetNumberSlice[int8](f, ptr, colType, value)
	case 16: // []float32
		trySetNumberSlice[float32](f, ptr, colType, value)
	case 17: // []float64
		trySetNumberSlice[float64](f, ptr, colType, value)

	case 21: // *string
		if vl, ok := value.(string); ok {
			f.Set(ptr, &vl)
		} else if vl, ok := value.(*string); ok {
			f.Set(ptr, vl)
		} else {
			printError(GetColTypeByID(colType).FieldType, value)
		}
	case 22: // *int64
		val := reflectToInt64(value)
		f.Set(ptr, &val)
	case 23: // *int32
		val := int32(reflectToInt64(value))
		f.Set(ptr, &val)
	case 24: // *int16
		val := int16(reflectToInt64(value))
		f.Set(ptr, &val)
	case 25: // *int8
		val := int8(reflectToInt64(value))
		f.Set(ptr, &val)
	case 26: // *float32
		val := reflectToFloat32(value)
		f.Set(ptr, &val)
	case 27: // *float64
		val := reflectToFloat64(value)
		f.Set(ptr, &val)
	case 28: // *int
		val := int(reflectToInt64(value))
		f.Set(ptr, &val)

	case 31: // *[]string
		if vl, ok := value.([]string); ok {
			val := slices.Clone(vl)
			f.Set(ptr, &val)
		} else if vl, ok := value.(*[]string); ok {
			val := slices.Clone(*vl)
			f.Set(ptr, &val)
		} else {
			printError(GetColTypeByID(colType).FieldType, value)
		}
	case 32: // *[]int64
		trySetNumberSlicePointer[int64](f, ptr, colType, value)
	case 33: // *[]int32
		trySetNumberSlicePointer[int32](f, ptr, colType, value)
	case 34: // *[]int16
		trySetNumberSlicePointer[int16](f, ptr, colType, value)
	case 35: // *[]int8
		trySetNumberSlicePointer[int8](f, ptr, colType, value)
	case 36: // *[]float32
		trySetNumberSlicePointer[float32](f, ptr, colType, value)
	case 37: // *[]float64
		trySetNumberSlicePointer[float64](f, ptr, colType, value)
	}
}

func makeScyllaValue(f *xunsafe.Field, ptr unsafe.Pointer, colType int8) any {
	if f == nil || ptr == nil {
		return nil
	}

	// Handle pointer types first
	if (colType >= 21 && colType <= 28) || (colType >= 31 && colType <= 37) {
		if f.IsNil(ptr) {
			return nil
		}
	}

	switch colType {
	case 1, 21: // string, *string
		return "'" + f.String(ptr) + "'"
	case 11, 31: // []string, *[]string
		var values []string
		if colType == 11 {
			values = f.Interface(ptr).([]string)
		} else {
			pValues := f.Interface(ptr).(*[]string)
			if pValues == nil {
				return nil
			}
			values = *pValues
		}
		strValues := make([]string, len(values))
		for i, v := range values {
			strValues[i] = "'" + v + "'"
		}
		return "{" + strings.Join(strValues, ",") + "}"
	case 12, 13, 14, 15, 16, 17, 32, 33, 34, 35, 36, 37: // other slices/pointers to slices
		concatenatedValues := Concatx(",", reflectToSlicePtr(f, ptr))
		return "{" + concatenatedValues + "}"
	case 9: // []byte / blob (could be complex type)
		// Check if it's a complex type or just []byte
		if f.Type.Kind() != reflect.Slice || f.Type.Elem().Kind() != reflect.Uint8 {
			// Complex type
			fieldValue := f.Interface(ptr)
			recordBytes, err := cbor.Marshal(fieldValue)
			if err != nil {
				fmt.Println("Error al encodeding .cbor:: ", f.Name, err)
				return ""
			}
			hexString := hex.EncodeToString(recordBytes)
			return "0x" + hexString
		}
		// Plain []byte
		return f.Interface(ptr)
	default:
		return f.Interface(ptr)
	}
}
