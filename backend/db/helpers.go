package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/fatih/color"
)

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

	if style == 1 {
		c = color.New(color.FgCyan, color.Bold)
	} else if style == 2 {
		c = color.New(color.FgGreen, color.Bold)
	} else if style == 3 {
		c = color.New(color.FgYellow, color.Bold)
	} else if style == 4 {
		c = color.New(color.FgBlue, color.Bold)
	} else if style == 5 {
		c = color.New(color.FgRed, color.Bold)
	} else if style == 6 {
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
