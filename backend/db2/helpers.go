package db2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
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
