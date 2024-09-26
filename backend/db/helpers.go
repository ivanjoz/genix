package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
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
			binary.Write(buf, binary.LittleEndian, val)
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
