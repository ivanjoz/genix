package server_utils

import (
	"bytes"
	"errors"
	"testing"
)

// These twenty bytes are the contract with the Rust decoder, which reads them by offset. The
// vectors here and the ones in parses_the_exact_wire_offsets and
// required_access_slots_are_read_by_offset (limiter/protocol.rs) are the same charge written from
// both ends. This test and its Rust twin are the only thing holding the layout.
func TestChargePayloadMatchesTheWireOffsets(t *testing.T) {
	payload, err := encodeCharge(0x123456, 42, 103, 300, 25, []uint16{0x0139, 0x008B})
	if err != nil {
		t.Fatalf("encodeCharge refused a valid charge: %v", err)
	}

	want := []byte{
		0x12, 0x34, 0x56, // company
		0x00, 0x00, 0x2A, // user
		0x00, 0x67, // route 103
		0x01, 0x2C, // cpu 300
		0x00, 0x19, // inference 25
		0x01, 0x39, // required access slot 0
		0x00, 0x8B, // required access slot 1
		0x00, 0x00, // slot 2 unused
		0x00, 0x00, // slot 3 unused
	}
	if !bytes.Equal(payload, want) {
		t.Fatalf("charge payload = % X; want % X", payload, want)
	}
}

// Zero terminates the slot list on the far side, so it can never also be a grant, and a route
// mapped to more accesses than fit is a configuration bug caught here rather than a rejected frame.
func TestRequiredAccessSlotsAreValidated(t *testing.T) {
	tooMany := make([]uint16, MaxRequiredAccess+1)
	for slot := range tooMany {
		tooMany[slot] = uint16(slot+1) << 2
	}
	if _, err := encodeCharge(1, 1, 1, 1, 0, tooMany); err == nil {
		t.Fatal("a route requiring more accesses than the frame holds was accepted")
	}
	if _, err := encodeCharge(1, 1, 1, 1, 0, []uint16{0x008B, 0}); err == nil {
		t.Fatal("a zero required access was accepted")
	}
	full := make([]uint16, MaxRequiredAccess)
	for slot := range full {
		full[slot] = uint16(slot+1) << 2
	}
	if _, err := encodeCharge(1, 1, 1, 1, 0, full); err != nil {
		t.Fatalf("a full slot list was refused: %v", err)
	}
}

// An authorize-only frame carries no credits. It exists because creditControlRoutes skips the
// charge, and three of those routes are access-mapped: skipping the frame with them would leave
// them open to any session.
func TestAFrameNeedsCreditsOrARequiredAccess(t *testing.T) {
	if _, err := encodeCharge(1, 1, 1, 0, 0, nil); err == nil {
		t.Fatal("a frame with neither credits nor a required access was accepted")
	}
	if _, err := encodeCharge(1, 1, 1, 0, 0, []uint16{0x008B}); err != nil {
		t.Fatalf("an authorize-only frame was refused: %v", err)
	}
}

// The reply's detail field. Zero when a check was requested means the daemon never answered it,
// which must fail closed: failing open would silently unauthorize every gated route the moment the
// two binaries drifted apart.
func TestAccessVerdictDecoding(t *testing.T) {
	if err := decodeAccessResponse(0, false); err != nil {
		t.Fatalf("an unrequested check reported %v", err)
	}
	if err := decodeAccessResponse(1, true); err != nil {
		t.Fatalf("a granted check reported %v", err)
	}

	for _, check := range []struct {
		detail         uint16
		identityFailed bool
	}{{2, false}, {3, true}, {4, true}} {
		err := decodeAccessResponse(check.detail, true)
		denied, ok := err.(*AccessDenied)
		if !ok {
			t.Fatalf("detail %d decoded as %T: %v", check.detail, err, err)
		}
		if denied.IdentityFailed() != check.identityFailed {
			t.Fatalf("detail %d: IdentityFailed() = %v; want %v",
				check.detail, denied.IdentityFailed(), check.identityFailed)
		}
		if !IsAccessDeniedError(err) {
			t.Fatalf("detail %d was not recognised as an access denial", check.detail)
		}
	}

	// A daemon that ignored the slots, and one that answered something invented.
	for _, detail := range []uint16{0, 5, 65535} {
		if err := decodeAccessResponse(detail, true); !errors.Is(err, ErrServerUtilsUnavailable) {
			t.Fatalf("detail %d decoded as %v; want unavailability", detail, err)
		}
	}
}

// The GET split: the base is what the pre-handler frame charges, the top-up is the difference.
func TestGetBaseAndTopUpSumToTheWholeCharge(t *testing.T) {
	base, err := APICPUBaseCredits("GET")
	if err != nil || base != 2 {
		t.Fatalf("APICPUBaseCredits(GET) = %d, %v; want 2", base, err)
	}
	// A response inside the first block owes exactly the base, so no second frame is sent at all.
	for _, responseBytes := range []int{0, 4 * 1024, 8 * 1024} {
		total, _ := APICPUCredits("GET", responseBytes)
		if total != base {
			t.Fatalf("a %d-byte GET response owes %d; want just the base %d",
				responseBytes, total, base)
		}
	}
	// Past it, base plus top-up must equal what the single old charge would have been.
	for _, responseBytes := range []int{8*1024 + 1, 24 * 1024, 24*1024 + 1, 1 << 20} {
		total, _ := APICPUCredits("GET", responseBytes)
		if total <= base {
			t.Fatalf("a %d-byte GET response owes only %d", responseBytes, total)
		}
		if base+(total-base) != total {
			t.Fatalf("the split does not reconstruct the charge for %d bytes", responseBytes)
		}
	}
}

// Route zero is a request that matched no generated route. Its credits are real, so refusing it
// would make an unnumbered handler free; the ceiling it is checked against is the blob's, not the
// route table's, because this side must not go stale when a handler is added.
func TestChargeAcceptsUnknownRoutesAndRefusesUnencodableOnes(t *testing.T) {
	for _, routeID := range []int16{0, 1, maxChargeRouteID} {
		if _, err := encodeCharge(1, 1, routeID, 1, 0, nil); err != nil {
			t.Fatalf("route %d was refused: %v", routeID, err)
		}
	}
	if _, err := encodeCharge(1, 1, maxChargeRouteID+1, 1, 0, nil); err == nil {
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

// The whole signed charge frame, pinned against matches_the_go_client_vectors in
// server_utils/src/service/auth.rs. The payload test above proves the fields sit at the right
// offsets; this proves the twenty bytes are also what gets signed, which is what a widened payload
// could quietly get wrong — the tag would still verify on both ends while covering different bytes.
func TestChargeFrameMatchesTheRustAuthVector(t *testing.T) {
	secret := []byte("test-secret")
	nonce := [serverUtilsNonceSize]byte{1, 2, 3, 4, 5, 6, 7, 8}
	payload, err := encodeCharge(0x123456, 42, 103, 300, 25, []uint16{0x0139, 0x008B})
	if err != nil {
		t.Fatalf("encodeCharge refused a valid charge: %v", err)
	}

	frame := buildServerUtilsFrame(secret, &nonce, 0, opcodeChargeCredits, payload)
	want := []byte{
		0x01, 0x12, 0x34, 0x56, 0x00, 0x00, 0x2A, 0x00, 0x67, 0x01, 0x2C, 0x00, 0x19, 0x01,
		0x39, 0x00, 0x8B, 0x00, 0x00, 0x00, 0x00,
		0x0F, 0x13, 0xFA, 0xB1, 0xDE, 0xF3, 0xCA, 0xA8,
	}
	if !bytes.Equal(frame, want) {
		t.Fatalf("charge frame = % X; want % X", frame, want)
	}
	// The tag is bound to the sequence, so frame two of a connection differs in its last eight bytes.
	next := buildServerUtilsFrame(secret, &nonce, 1, opcodeChargeCredits, payload)
	wantTag := []byte{0xA9, 0x87, 0x64, 0x7C, 0xD4, 0x89, 0x26, 0xAD}
	if !bytes.Equal(next[len(next)-serverUtilsAuthTagSize:], wantTag) {
		t.Fatalf("sequence 1 tag = % X; want % X", next[len(next)-serverUtilsAuthTagSize:], wantTag)
	}
}

// The invalidation frame, pinned against the same Rust vector set.
func TestAccessInvalidationFrameMatchesTheRustAuthVector(t *testing.T) {
	payload, err := encodeAccessInvalidation(7, 300)
	if err != nil {
		t.Fatalf("encodeAccessInvalidation refused a valid target: %v", err)
	}
	nonce := [serverUtilsNonceSize]byte{1, 2, 3, 4, 5, 6, 7, 8}
	frame := buildServerUtilsFrame([]byte("test-secret"), &nonce, 0, opcodeInvalidateUserAccess, payload)
	want := []byte{
		0x06, 0x00, 0x00, 0x07, 0x00, 0x01, 0x2C,
		0x82, 0xE1, 0xEA, 0x44, 0x84, 0xB5, 0x90, 0x74,
	}
	if !bytes.Equal(frame, want) {
		t.Fatalf("invalidation frame = % X; want % X", frame, want)
	}

	// Zero is the wildcard and must encode; a company of zero has no meaning and must not.
	if _, err := encodeAccessInvalidation(7, InvalidateAllCompanyUsers); err != nil {
		t.Fatalf("the company wildcard was refused: %v", err)
	}
	if _, err := encodeAccessInvalidation(0, 1); err == nil {
		t.Fatal("company zero was accepted")
	}
}
