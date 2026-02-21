package index_builder

import "fmt"

func convertIntSliceToUint16(values []int, fieldLabel string, maxAllowedValue int) ([]uint16, error) {
	convertedValues := make([]uint16, 0, len(values))
	for valueIndex, value := range values {
		if value < 0 || value > maxAllowedValue {
			return nil, fmt.Errorf("%s value=%d overflows uint16 at index=%d", fieldLabel, value, valueIndex)
		}
		convertedValues = append(convertedValues, uint16(value))
	}
	return convertedValues, nil
}

func packIntSliceToUint12(values []int, fieldLabel string, maxAllowedValue int) ([]uint8, error) {
	packedBytes := make([]uint8, 0, expectedUint12PackedBytes(len(values)))
	for valueIndex := 0; valueIndex < len(values); valueIndex += 2 {
		leftValue := values[valueIndex]
		if leftValue < 0 || leftValue > maxAllowedValue {
			return nil, fmt.Errorf("%s value=%d overflows uint12 at index=%d", fieldLabel, leftValue, valueIndex)
		}

		rightValue := 0
		if valueIndex+1 < len(values) {
			rightValue = values[valueIndex+1]
			if rightValue < 0 || rightValue > maxAllowedValue {
				return nil, fmt.Errorf("%s value=%d overflows uint12 at index=%d", fieldLabel, rightValue, valueIndex+1)
			}
		}

		// Pack [LLLLLLLL|LLLLRRRR|RRRRRRRR] where each value is 12 bits.
		packedBytes = append(packedBytes, uint8(leftValue>>4))
		packedBytes = append(packedBytes, uint8(((leftValue&0x0F)<<4)|((rightValue>>8)&0x0F)))
		packedBytes = append(packedBytes, uint8(rightValue&0xFF))
	}
	return packedBytes, nil
}

func expectedUint12PackedBytes(valueCount int) int {
	return ((valueCount + 1) / 2) * 3
}
