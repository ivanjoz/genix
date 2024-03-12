package types

import (
	"bytes"
	"fmt"
	"math"
	mrand "math/rand"
	"strings"
)

const base36Chars = "0123456789abcdefghijklmnopqrstuvwxyz"

func MakeRandomBase36String(length int) string {
	bytes := make([]byte, length)
	const ln = len(base36Chars)
	for i := 0; i < length; i++ {
		bytes[i] = base36Chars[mrand.Intn(ln)]
	}
	return string(bytes)
}

func ConcatInt64(num1, num2 int64) int64 {
	if num1 == 0 {
		return 0
	}
	return num1*10_000_000_000 + num2
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
