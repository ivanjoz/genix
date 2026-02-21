package types

import (
	"reflect"
	"testing"
)

func TestEncodeDecodeIDs_RoundTripAcrossBitWidths(t *testing.T) {
	testCases := []struct {
		name             string
		inputIDs         []int32
		expectedBitWidth byte
	}{
		{
			name:             "uses 8 bits",
			inputIDs:         []int32{1, 2, 255},
			expectedBitWidth: 8,
		},
		{
			name:             "uses 16 bits",
			inputIDs:         []int32{1, 256, 65535},
			expectedBitWidth: 16,
		},
		{
			name:             "uses 24 bits",
			inputIDs:         []int32{1, 65536, 16777215},
			expectedBitWidth: 24,
		},
		{
			name:             "uses 32 bits",
			inputIDs:         []int32{1, 16777216, 2147483647},
			expectedBitWidth: 32,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// The first byte stores the selected bit width for all IDs in the payload.
			encodedIDs := EncodeIDs(testCase.inputIDs)
			if len(encodedIDs) == 0 {
				t.Fatalf("expected encoded payload, got empty bytes")
			}
			if encodedIDs[0] != testCase.expectedBitWidth {
				t.Fatalf("expected bit width %d, got %d", testCase.expectedBitWidth, encodedIDs[0])
			}

			decodedIDs := DecodeIDs(encodedIDs)
			if !reflect.DeepEqual(decodedIDs, testCase.inputIDs) {
				t.Fatalf("roundtrip mismatch, expected %v got %v", testCase.inputIDs, decodedIDs)
			}
		})
	}
}

func TestEncodeDecodeIDs_EmptyPayload(t *testing.T) {
	// Empty input is represented by a header byte with value 0.
	encodedIDs := EncodeIDs(nil)
	if !reflect.DeepEqual(encodedIDs, []byte{0}) {
		t.Fatalf("expected empty-header payload, got %v", encodedIDs)
	}

	decodedIDs := DecodeIDs(encodedIDs)
	if len(decodedIDs) != 0 {
		t.Fatalf("expected empty decoded IDs, got %v", decodedIDs)
	}
}

func TestDecodeIDs_InvalidPayloadReturnsNil(t *testing.T) {
	testCases := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "invalid bit width",
			payload: []byte{12, 1, 2},
		},
		{
			name:    "truncated 16-bit payload",
			payload: []byte{16, 1},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Invalid headers and truncated payloads are rejected defensively.
			decodedIDs := DecodeIDs(testCase.payload)
			if decodedIDs != nil {
				t.Fatalf("expected nil for invalid payload, got %v", decodedIDs)
			}
		})
	}
}
