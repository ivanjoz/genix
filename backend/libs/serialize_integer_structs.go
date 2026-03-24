package libs

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/viant/xunsafe"
)

/*
 * First two bits represent the intsize
 * 0 = uint14, 1 = uint18, 2 = uint22, 3 = uint30
 */
func SerializeInt30Struct[T any](object T) []byte {
	objectType := reflect.TypeOf(object)
	if objectType.Kind() != reflect.Struct {
		panic("SerializeInt30Struct expects a struct value")
	}

	objectPointer := xunsafe.AsPointer(&object)
	structFields := xunsafe.NewStruct(objectType).Fields
	encodedBytes := make([]byte, 0, len(structFields)*4)
	bitOffset := uint8(0)

	for fieldIndex := range structFields {
		field := &structFields[fieldIndex]
		normalizedValue := readInt30FieldValue(field, objectPointer)
		sizeFlag, payloadBits := compactInt30Size(normalizedValue)

		// Each field stores a 2-bit size selector followed by the compact payload.
		for shift := uint8(2); shift > 0; shift-- {
			if bitOffset == 0 {
				encodedBytes = append(encodedBytes, 0)
			}
			if ((sizeFlag >> (shift - 1)) & 1) == 1 {
				lastByteIndex := len(encodedBytes) - 1
				encodedBytes[lastByteIndex] |= 1 << (7 - bitOffset)
			}
			bitOffset = (bitOffset + 1) % 8
		}

		// Payload bits are emitted MSB-first to match the decoder bit order.
		for shift := payloadBits; shift > 0; shift-- {
			if bitOffset == 0 {
				encodedBytes = append(encodedBytes, 0)
			}
			if ((normalizedValue >> (shift - 1)) & 1) == 1 {
				lastByteIndex := len(encodedBytes) - 1
				encodedBytes[lastByteIndex] |= 1 << (7 - bitOffset)
			}
			bitOffset = (bitOffset + 1) % 8
		}
	}

	return encodedBytes
}

func DeserializeInt30Struct[T any](encodedBytes []byte, target *T) error {
	if target == nil {
		return fmt.Errorf("DeserializeInt30Struct expects a non-nil target")
	}

	objectType := reflect.TypeOf(*target)
	if objectType.Kind() != reflect.Struct {
		return fmt.Errorf("DeserializeInt30Struct expects a struct target")
	}

	objectPointer := xunsafe.AsPointer(target)
	structFields := xunsafe.NewStruct(objectType).Fields
	bitOffset := 0

	for fieldIndex := range structFields {
		field := &structFields[fieldIndex]
		var sizeFlag uint32
		for bitIndex := 0; bitIndex < 2; bitIndex++ {
			byteIndex := bitOffset / 8
			if byteIndex >= len(encodedBytes) {
				return fmt.Errorf("read size flag for field %s: unexpected end of input", field.Name)
			}

			bitPosition := bitOffset % 8
			currentBit := (encodedBytes[byteIndex] >> (7 - bitPosition)) & 1
			sizeFlag = (sizeFlag << 1) | uint32(currentBit)
			bitOffset++
		}

		payloadBits, flagError := payloadBitsFromFlag(sizeFlag)
		if flagError != nil {
			return fmt.Errorf("invalid size flag for field %s: %w", field.Name, flagError)
		}

		var fieldValue uint32
		for bitIndex := uint8(0); bitIndex < payloadBits; bitIndex++ {
			byteIndex := bitOffset / 8
			if byteIndex >= len(encodedBytes) {
				return fmt.Errorf("read payload for field %s: unexpected end of input", field.Name)
			}

			bitPosition := bitOffset % 8
			currentBit := (encodedBytes[byteIndex] >> (7 - bitPosition)) & 1
			fieldValue = (fieldValue << 1) | uint32(currentBit)
			bitOffset++
		}

		assignInt30FieldValue(field, objectPointer, fieldValue)
	}

	return nil
}

const maxUint30Value = (1 << 30) - 1

func readInt30FieldValue(field *xunsafe.Field, objectPointer unsafe.Pointer) uint32 {
	switch field.Kind() {
	case reflect.Int:
		return normalizeInt30Signed(int64(field.Int(objectPointer)))
	case reflect.Int8:
		return normalizeInt30Signed(int64(field.Int8(objectPointer)))
	case reflect.Int16:
		return normalizeInt30Signed(int64(field.Int16(objectPointer)))
	case reflect.Int32:
		return normalizeInt30Signed(int64(field.Int32(objectPointer)))
	case reflect.Int64:
		return normalizeInt30Signed(field.Int64(objectPointer))
	case reflect.Uint:
		return normalizeInt30Unsigned(uint64(field.Uint(objectPointer)))
	case reflect.Uint8:
		return normalizeInt30Unsigned(uint64(field.Uint8(objectPointer)))
	case reflect.Uint16:
		return normalizeInt30Unsigned(uint64(field.Uint16(objectPointer)))
	case reflect.Uint32:
		return normalizeInt30Unsigned(uint64(field.Uint32(objectPointer)))
	case reflect.Uint64:
		return normalizeInt30Unsigned(field.Uint64(objectPointer))
	default:
		panic("SerializeInt30Struct only supports integer fields")
	}
}

func normalizeInt30Signed(fieldValue int64) uint32 {
	// Negative values are saturated to zero to preserve the unsigned packing contract.
	if fieldValue <= 0 {
		return 0
	}
	return normalizeInt30Unsigned(uint64(fieldValue))
}

func normalizeInt30Unsigned(fieldValue uint64) uint32 {
	// Values above 30 bits are saturated because the format has no larger payload class.
	if fieldValue > maxUint30Value {
		return maxUint30Value
	}
	return uint32(fieldValue)
}

func compactInt30Size(fieldValue uint32) (uint8, uint8) {
	switch {
	case fieldValue <= (1<<14)-1:
		return 0, 14
	case fieldValue <= (1<<18)-1:
		return 1, 18
	case fieldValue <= (1<<22)-1:
		return 2, 22
	default:
		return 3, 30
	}
}

func payloadBitsFromFlag(sizeFlag uint32) (uint8, error) {
	switch sizeFlag {
	case 0:
		return 14, nil
	case 1:
		return 18, nil
	case 2:
		return 22, nil
	case 3:
		return 30, nil
	default:
		return 0, fmt.Errorf("unsupported flag %d", sizeFlag)
	}
}

func assignInt30FieldValue(field *xunsafe.Field, objectPointer unsafe.Pointer, fieldValue uint32) {
	switch field.Kind() {
	case reflect.Int:
		field.SetInt(objectPointer, int(clampUint32ToUint64(fieldValue, uint64(maxIntValue))))
	case reflect.Int8:
		field.SetInt8(objectPointer, int8(clampUint32ToUint64(fieldValue, 1<<7-1)))
	case reflect.Int16:
		field.SetInt16(objectPointer, int16(clampUint32ToUint64(fieldValue, 1<<15-1)))
	case reflect.Int32:
		field.SetInt32(objectPointer, int32(clampUint32ToUint64(fieldValue, 1<<31-1)))
	case reflect.Int64:
		field.SetInt64(objectPointer, int64(fieldValue))
	case reflect.Uint:
		field.SetUint(objectPointer, uint(clampUint32ToUint64(fieldValue, maxUintValue)))
	case reflect.Uint8:
		field.SetUint8(objectPointer, uint8(clampUint32ToUint64(fieldValue, 1<<8-1)))
	case reflect.Uint16:
		field.SetUint16(objectPointer, uint16(clampUint32ToUint64(fieldValue, 1<<16-1)))
	case reflect.Uint32:
		field.SetUint32(objectPointer, fieldValue)
	case reflect.Uint64:
		field.SetUint64(objectPointer, uint64(fieldValue))
	default:
		panic("DeserializeInt30Struct only supports integer fields")
	}
}

func clampUint32ToUint64(fieldValue uint32, maxValue uint64) uint64 {
	if uint64(fieldValue) > maxValue {
		return maxValue
	}
	return uint64(fieldValue)
}

const (
	maxUintValue = ^uint64(0)
	maxIntValue  = int(^uint(0) >> 1)
)
