package libs

import (
	"fmt"
	"testing"
)

func TestSerializeInt30Struct_CompactsAndClampsValues(t *testing.T) {
	type compactRecord struct {
		Tiny     int16
		Small    uint32
		Medium   int32
		Large    int64
		Negative int32
		Overflow uint64
	}

	serializedBytes := SerializeInt30Struct(compactRecord{
		Tiny:     42,
		Small:    1 << 17,
		Medium:   1 << 20,
		Large:    1 << 25,
		Negative: -99,
		Overflow: 1 << 40,
	})

	bitOffset := 0
	readBits := func(bitCount int) (uint32, error) {
		// The test decoder mirrors the production bit order to assert exact stream contents.
		var value uint32
		for bitIndex := 0; bitIndex < bitCount; bitIndex++ {
			byteIndex := bitOffset / 8
			if byteIndex >= len(serializedBytes) {
				return 0, fmt.Errorf("unexpected end of input at bit %d", bitOffset)
			}

			bitPosition := bitOffset % 8
			currentBit := (serializedBytes[byteIndex] >> (7 - bitPosition)) & 1
			value = (value << 1) | uint32(currentBit)
			bitOffset++
		}
		return value, nil
	}

	testCases := []struct {
		expectedFlag  uint32
		expectedBits  int
		expectedValue uint32
	}{
		{expectedFlag: 0, expectedBits: 14, expectedValue: 42},
		{expectedFlag: 1, expectedBits: 18, expectedValue: 1 << 17},
		{expectedFlag: 2, expectedBits: 22, expectedValue: 1 << 20},
		{expectedFlag: 3, expectedBits: 30, expectedValue: 1 << 25},
		{expectedFlag: 0, expectedBits: 14, expectedValue: 0},
		{expectedFlag: 3, expectedBits: 30, expectedValue: maxUint30Value},
	}

	for caseIndex, testCase := range testCases {
		sizeFlag, readError := readBits(2)
		if readError != nil {
			t.Fatalf("case %d: unexpected flag read error: %v", caseIndex, readError)
		}
		if sizeFlag != testCase.expectedFlag {
			t.Fatalf("case %d: expected flag %d, got %d", caseIndex, testCase.expectedFlag, sizeFlag)
		}

		fieldValue, readError := readBits(testCase.expectedBits)
		if readError != nil {
			t.Fatalf("case %d: unexpected payload read error: %v", caseIndex, readError)
		}
		if fieldValue != testCase.expectedValue {
			t.Fatalf("case %d: expected value %d, got %d", caseIndex, testCase.expectedValue, fieldValue)
		}
	}
}

func TestDeserializeInt30Struct_RoundTrip(t *testing.T) {
	type compactRecord struct {
		Tiny     int16
		Small    uint32
		Medium   int32
		Large    int64
		Negative int32
		Overflow uint64
	}

	sourceRecord := compactRecord{
		Tiny:     42,
		Small:    1 << 17,
		Medium:   1 << 20,
		Large:    1 << 25,
		Negative: -99,
		Overflow: 1 << 40,
	}

	var decodedRecord compactRecord
	decodeError := DeserializeInt30Struct(SerializeInt30Struct(sourceRecord), &decodedRecord)
	if decodeError != nil {
		t.Fatalf("unexpected decode error: %v", decodeError)
	}

	if decodedRecord.Tiny != 42 {
		t.Fatalf("expected Tiny 42, got %d", decodedRecord.Tiny)
	}
	if decodedRecord.Small != 1<<17 {
		t.Fatalf("expected Small %d, got %d", 1<<17, decodedRecord.Small)
	}
	if decodedRecord.Medium != 1<<20 {
		t.Fatalf("expected Medium %d, got %d", 1<<20, decodedRecord.Medium)
	}
	if decodedRecord.Large != 1<<25 {
		t.Fatalf("expected Large %d, got %d", 1<<25, decodedRecord.Large)
	}
	if decodedRecord.Negative != 0 {
		t.Fatalf("expected Negative 0, got %d", decodedRecord.Negative)
	}
	if decodedRecord.Overflow != maxUint30Value {
		t.Fatalf("expected Overflow %d, got %d", maxUint30Value, decodedRecord.Overflow)
	}
}

func TestDeserializeInt30Struct_TruncatedInput(t *testing.T) {
	type compactRecord struct {
		Value int32
	}

	var decodedRecord compactRecord
	decodeError := DeserializeInt30Struct([]byte{}, &decodedRecord)
	if decodeError == nil {
		t.Fatal("expected decode error for truncated input")
	}
}
