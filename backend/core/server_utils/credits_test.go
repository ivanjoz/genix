package server_utils

import (
	"bytes"
	"testing"
)

// These twelve bytes are the contract with the Rust decoder, which reads them by offset. The
// vectors here and the ones in parses_the_exact_wire_offsets (limiter/protocol.rs) are the same
// charge written from both ends.
func TestChargePayloadMatchesTheWireOffsets(t *testing.T) {
	payload, err := encodeCharge(0x123456, 42, 103, 300, 25)
	if err != nil {
		t.Fatalf("encodeCharge refused a valid charge: %v", err)
	}

	want := []byte{
		0x12, 0x34, 0x56, // company
		0x00, 0x00, 0x2A, // user
		0x00, 0x67, // route 103
		0x01, 0x2C, // cpu 300
		0x00, 0x19, // inference 25
	}
	if !bytes.Equal(payload, want) {
		t.Fatalf("charge payload = % X; want % X", payload, want)
	}
}

// Route zero is a request that matched no generated route. Its credits are real, so refusing it
// would make an unnumbered handler free; the ceiling it is checked against is the blob's, not the
// route table's, because this side must not go stale when a handler is added.
func TestChargeAcceptsUnknownRoutesAndRefusesUnencodableOnes(t *testing.T) {
	for _, routeID := range []int16{0, 1, maxChargeRouteID} {
		if _, err := encodeCharge(1, 1, routeID, 1, 0); err != nil {
			t.Fatalf("route %d was refused: %v", routeID, err)
		}
	}
	if _, err := encodeCharge(1, 1, maxChargeRouteID+1, 1, 0); err == nil {
		t.Fatal("a route past the encoding ceiling was accepted")
	}
}

func TestCreditFormulasRoundPartialBlocksUp(t *testing.T) {
	checks := []struct {
		method string
		bytes  int
		want   uint16
	}{
		{"GET", 0, 2}, {"GET", 8 * 1024, 2}, {"GET", 8*1024 + 1, 3},
		{"GET", 24 * 1024, 3}, {"GET", 24*1024 + 1, 4},
		{"POST", 0, 5}, {"POST", 8 * 1024, 5}, {"POST", 8*1024 + 1, 6},
		{"POST", 16 * 1024, 6}, {"POST", 16*1024 + 1, 7},
	}
	for _, check := range checks {
		got, err := APICPUCredits(check.method, check.bytes)
		if err != nil || got != check.want {
			t.Fatalf("APICPUCredits(%q, %d) = %d, %v; want %d", check.method, check.bytes, got, err, check.want)
		}
	}
	inference, err := InferenceCredits(8*1024+1, 8*1024+1)
	if err != nil || inference != 6 {
		t.Fatalf("InferenceCredits() = %d, %v; want 6", inference, err)
	}
}

func TestMonthlyCreditLimitResponseUsesTheReservedWindow(t *testing.T) {
	err := decodeCreditLimitResponse(0b1_1110)
	limit, ok := err.(*CreditLimitExceeded)
	if !ok {
		t.Fatalf("monthly response decoded as %T: %v", err, err)
	}
	if !limit.Company || limit.Window != "month" || !limit.CPU || !limit.Inference {
		t.Fatalf("monthly response decoded incorrectly: %+v", limit)
	}
}
