package db

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/fatih/color"
	"github.com/kr/pretty"
	"github.com/viant/xunsafe"
)

var (
	DebugFull          bool
	logVariableCheched bool
	LogCount           uint32
)

func ShouldLog() bool {
	if !logVariableCheched {
		DebugFull = os.Getenv("LOGS_FULL") != ""
		logVariableCheched = true
	}
	if !DebugFull {
		return false
	}
	return atomic.LoadUint32(&LogCount) < 5
}

func IncrementLogCount() {
	atomic.AddUint32(&LogCount, 1)
}

func BasicHashInt(s string) int32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int32(h.Sum32())
}

const base64Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+/"

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

func HashInt(values ...any) int32 {
	buf := new(bytes.Buffer)

	for _, anyVal := range values {
		switch val := anyVal.(type) {
		case int:
			binary.Write(buf, binary.LittleEndian, int64(val))
		case int32:
			binary.Write(buf, binary.LittleEndian, val)
		case int64:
			binary.Write(buf, binary.LittleEndian, val)
		case int16:
			binary.Write(buf, binary.LittleEndian, val)
		case int8:
			binary.Write(buf, binary.LittleEndian, val)
		case float32:
			binary.Write(buf, binary.LittleEndian, val)
		case float64:
			binary.Write(buf, binary.LittleEndian, val)
		case string:
			buf.WriteString(val)
		default:
			buf.WriteString(fmt.Sprintf("%v", val))
		}
		buf.WriteByte(0)
	}

	h := fnv.New32a()
	h.Write(buf.Bytes())
	return int32(h.Sum32())
}

func Logx(style int8, messageInColor string, params ...any) {
	var c *color.Color

	switch style {
	case 1:
		c = color.New(color.FgCyan, color.Bold)
	case 2:
		c = color.New(color.FgGreen, color.Bold)
	case 3:
		c = color.New(color.FgYellow, color.Bold)
	case 4:
		c = color.New(color.FgBlue, color.Bold)
	case 5:
		c = color.New(color.FgRed, color.Bold)
	case 6:
		c = color.New(color.FgMagenta, color.Bold)
	}

	c.Print(messageInColor)
	if len(params) > 0 {
		fmt.Print(" | ")
		for _, e := range params {
			fmt.Print(e)
			fmt.Print(" ")
		}
		fmt.Println("")
	}
}

func convertToInt64(val any) int64 {
	// Use a type assertion to check if it's an int or other integer types
	switch v := val.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	default:
		// The value is not an integer
		fmt.Println("Error: Value is not an integer:", v)
		return 0
	}
}

func Pow10Int64(m int64) int64 {
	if m == 0 {
		return 1
	}

	if m == 1 {
		return 10
	}

	number := int64(10)
	for i := int64(2); i <= m; i++ {
		number *= 10
	}
	return number
}

func Concatx[T any](sep string, slice1 []T) string {
	sliceOfStrings := []string{}
	for _, value := range slice1 {
		sliceOfStrings = append(sliceOfStrings, fmt.Sprintf("%v", value))
	}
	return strings.Join(sliceOfStrings, sep)
}

func Err(content ...any) error {
	errMessage := Concatx(" ", content)
	return errors.New(errMessage)
}

func sliceToAny[T any](valuesGeneric *[]T) []any {
	values := []any{}
	for _, v := range *valuesGeneric {
		values = append(values, any(v))
	}
	return values
}

func reflectToSlicePtr(field *xunsafe.Field, ptr unsafe.Pointer) []any {
	return reflectToSliceValue(field.Interface(ptr))
}

func reflectToSliceValue(value any) []any {
	var values []any

	switch sl := value.(type) {
	case []int:
		values = sliceToAny(&sl)
	case []int8:
		values = sliceToAny(&sl)
	case []int16:
		values = sliceToAny(&sl)
	case []int32:
		values = sliceToAny(&sl)
	case []int64:
		values = sliceToAny(&sl)
	case []float32:
		values = sliceToAny(&sl)
	case []float64:
		values = sliceToAny(&sl)
	case []string:
		values = sliceToAny(&sl)
	default:
		// The value is not an integer
		panic("Value was not recognised of a slice: " + reflect.TypeOf(value).String())
	}
	return values
}

var (
	matchFirstCap  = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap    = regexp.MustCompile("([a-z0-9])([A-Z])")
	matchUnderline = regexp.MustCompile("_([a-z0-9])_")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	res := strings.ToLower(snake)

	for matchUnderline.MatchString(res) {
		res = matchUnderline.ReplaceAllString(res, "_$1")
	}

	return res
}

func Print(Struct any) {
	pretty.Println(Struct)
}

func Concat62(values ...any) string {
	valuesStrings := []string{}
	for _, va := range values {
		str := ""
		if v, ok := va.(string); ok {
			str = v
		} else if v, ok := va.(int32); ok {
			str = EncodeToBase62(int64(v))
		} else if v, ok := va.(int64); ok {
			str = EncodeToBase62(int64(v))
		} else if v, ok := va.(int); ok {
			str = EncodeToBase62(int64(v))
		} else if v, ok := va.(int16); ok {
			str = EncodeToBase62(int64(v))
		} else {
			str = fmt.Sprintf("%v", v)
		}
		valuesStrings = append(valuesStrings, str)
	}
	return strings.Join(valuesStrings, "_")
}

// characters used for conversion
const base32Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Encode encodes an int64 to a base62 encoded string.
func EncodeToBase62(number int64) string {
	return encodeToBase62(uint64(number))
}

// Decode decodes a base62 encoded string to an int64.
func DecodeFromBase62(token string) int64 {
	return int64(decodeFromBase62(token))
}

// EncodeUint64 encodes a uint64 to a base62 encoded string.
func encodeToBase62(number uint64) string {
	if number == 0 {
		return string(base32Alphabet[0])
	}

	chars := make([]byte, 0)

	length := uint64(len(base32Alphabet))

	for number > 0 {
		result := number / length
		remainder := number % length
		chars = append(chars, base32Alphabet[remainder])
		number = result
	}

	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}

	return string(chars)
}

// DecodeUint64 decodes a base62 encoded string to an uint64.
func decodeFromBase62(token string) uint64 {
	number := uint64(0)
	idx := float64(0.0)
	chars := []byte(base32Alphabet)

	charsLength := float64(len(chars))
	tokenLength := float64(len(token))

	for _, c := range []byte(token) {
		power := tokenLength - (idx + 1)
		index := uint64(bytes.IndexByte(chars, c))
		number += index * uint64(math.Pow(charsLength, power))
		idx++
	}

	return number
}

func GetRandomInt64(digits int8) int64 {
	if digits <= 0 {
		return 0
	}
	max := Pow10Int64(int64(digits))
	return rand.Int64N(max)
}
