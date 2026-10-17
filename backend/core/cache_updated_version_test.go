package core

import "testing"

// Pins the cc-ver wire format against the frontend's concatenateUint16s (see parsers.test.ts):
// one fixed-width little-endian u16 array, utils.Base64, no padding, no magnitude buckets.
func TestParseConcatenatedUint16sKeepsClientOrder(t *testing.T) {
	// concatenateUint16s([7, 40000, 3, 65535, 0, 255, 256])
	const encoded = "0k0-b-B-zzw--Ew---3"

	decoded := parseConcatenatedUint16s(encoded)
	expected := []uint16{7, 40000, 3, 65535, 0, 255, 256}

	if len(decoded) != len(expected) {
		t.Fatalf("expected %d values, got %d: %v", len(expected), len(decoded), decoded)
	}
	for i := range expected {
		if decoded[i] != expected[i] {
			t.Fatalf("value %d decoded as %d, want %d (full: %v)", i, decoded[i], expected[i], decoded)
		}
	}

	if len(parseConcatenatedUint16s("")) != 0 {
		t.Fatal("an empty cc-ver must decode to no values")
	}
}
