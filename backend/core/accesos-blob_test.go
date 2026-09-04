package core

import (
	"bytes"
	"testing"
)

// The byte layout is the contract with fareward/src/limiter/access.rs and with
// genix-ui/security/accesos.ts, so it is asserted against bytes written by hand rather than by
// round-tripping the encoder against itself. Endianness in particular cannot fail loudly: read the
// wrong way round these bytes still decode, into a different access.
func TestTheGrantWordIsBigEndian(t *testing.T) {
	// acceso 34 nivel 4 = 34<<2 | 3 = 139 = 0x008B.
	accesosBlob, accesosSubBlob, err := EncodeAccesosGrants([]AccesoGrant{{AccesoID: 34, Nivel: 4}})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if want := []byte{0x00, 0x8B}; !bytes.Equal(accesosBlob, want) {
		t.Fatalf("accesos_computed = %v; want %v (big-endian 0x008B)", accesosBlob, want)
	}
	if len(accesosSubBlob) != 0 {
		t.Fatalf("an access with no sub-accesses must not reach accesos_sub_computed: %v", accesosSubBlob)
	}
}

// Which column an access lands in is the format's "has sub-accesses" bit, so the split is the one
// thing a reader depends on that no byte states.
func TestGrantsSplitByWhetherTheyCarrySubAccesos(t *testing.T) {
	accesosBlob, accesosSubBlob, err := EncodeAccesosGrants([]AccesoGrant{
		{AccesoID: 10, Nivel: 2, SubMask: 0b101}, // subs 1 and 3
		{AccesoID: 3, Nivel: 1},
		{AccesoID: 16, Nivel: 4},
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	// Sorted ascending regardless of input order: 3 then 16.
	if want := []byte{0x00, 0x0C, 0x00, 0x43}; !bytes.Equal(accesosBlob, want) {
		t.Fatalf("accesos_computed = %v; want %v", accesosBlob, want)
	}
	// acceso 10 nivel 2 = 10<<2|1 = 41 = 0x0029, then one sub byte 0b0000101 with MORE clear.
	if want := []byte{0x00, 0x29, 0x05}; !bytes.Equal(accesosSubBlob, want) {
		t.Fatalf("accesos_sub_computed = %v; want %v", accesosSubBlob, want)
	}
}

// A mask that needs sub 8+ spills into a second byte, and only then.
func TestSubAccesoBytesUseMoreOnlyWhenNeeded(t *testing.T) {
	for _, check := range []struct {
		subMask   uint16
		wantBytes []byte
	}{
		{0b1, []byte{0x01}},              // only "Todos"
		{0b1111111, []byte{0x7F}},        // subs 1..7 fill the first byte exactly
		{0b10000000, []byte{0x80, 0x01}}, // sub 8 is the first that needs a second byte
		// Subs 1 and 13: bit 0 in the first byte, and bit 12 lands at bit 5 of the second.
		{0b1000000000001, []byte{0x81, 0x20}},
	} {
		_, accesosSubBlob, err := EncodeAccesosGrants([]AccesoGrant{{AccesoID: 5, Nivel: 1, SubMask: check.subMask}})
		if err != nil {
			t.Fatalf("mask %b: encode failed: %v", check.subMask, err)
		}
		if got := accesosSubBlob[2:]; !bytes.Equal(got, check.wantBytes) {
			t.Errorf("mask %b encoded to %v; want %v", check.subMask, got, check.wantBytes)
		}

		decoded, err := DecodeAccesosGrants(nil, accesosSubBlob)
		if err != nil {
			t.Fatalf("mask %b: decode failed: %v", check.subMask, err)
		}
		if len(decoded) != 1 || decoded[0].SubMask != check.subMask {
			t.Errorf("mask %b round-tripped to %+v", check.subMask, decoded)
		}
	}
}

func TestGrantsRoundTripThroughBothBlobs(t *testing.T) {
	grants := []AccesoGrant{
		{AccesoID: 1, Nivel: 1},
		{AccesoID: 7, Nivel: 4, SubMask: 0b1},
		{AccesoID: 10, Nivel: 2, SubMask: 0b1000000000000},
		{AccesoID: 36, Nivel: 3},
		{AccesoID: MaxAccesoID, Nivel: 4},
	}
	accesosBlob, accesosSubBlob, err := EncodeAccesosGrants(grants)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	decoded, err := DecodeAccesosGrants(accesosBlob, accesosSubBlob)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(decoded) != len(grants) {
		t.Fatalf("decoded %d grants; want %d", len(decoded), len(grants))
	}
	for index, grant := range grants {
		if decoded[index] != grant {
			t.Errorf("grant %d round-tripped to %+v; want %+v", index, decoded[index], grant)
		}
	}
}

// The encoder refuses rather than repairs: every one of these is a bug upstream, and a blob written
// anyway would hide it behind a user who quietly holds the wrong thing.
func TestTheEncoderRefusesMalformedGrants(t *testing.T) {
	for name, grants := range map[string][]AccesoGrant{
		"acceso id zero":     {{AccesoID: 0, Nivel: 1}},
		"acceso id past max": {{AccesoID: MaxAccesoID + 1, Nivel: 1}},
		"nivel zero":         {{AccesoID: 1, Nivel: 0}},
		"nivel past four":    {{AccesoID: 1, Nivel: 5}},
		"duplicate acceso":   {{AccesoID: 4, Nivel: 1}, {AccesoID: 4, Nivel: 2}},
		"sub past ceiling":   {{AccesoID: 4, Nivel: 1, SubMask: 1 << MaxSubAccesoID}},
	} {
		if _, _, err := EncodeAccesosGrants(grants); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// Ordering is load-bearing now that entries are variable width, so a reader validates as it walks
// instead of trusting the writer. These are the corruptions that would otherwise answer the wrong
// question silently rather than fail.
func TestTheDecoderRefusesCorruptBlobs(t *testing.T) {
	for name, check := range map[string]struct{ accesos, sub []byte }{
		"odd length accesos":        {accesos: []byte{0x00, 0x0C, 0x00}},
		"accesos out of order":      {accesos: []byte{0x00, 0x43, 0x00, 0x0C}},
		"accesos duplicated":        {accesos: []byte{0x00, 0x0C, 0x00, 0x0C}},
		"sub truncated mid grant":   {sub: []byte{0x00, 0x29}},
		"sub dangling more bit":     {sub: []byte{0x00, 0x29, 0x81}},
		"sub out of order":          {sub: []byte{0x00, 0x43, 0x01, 0x00, 0x29, 0x01}},
		"sub entry with empty mask": {sub: []byte{0x00, 0x29, 0x00}},
	} {
		if _, err := DecodeAccesosGrants(check.accesos, check.sub); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// "Todos" is the one piece of catalog meaning that never reaches the daemon: it returns the raw
// mask and Go expands bit 0.
func TestHasSubAccesoExpandsTodos(t *testing.T) {
	todosOnly := uint16(1 << (SubAccesoTodosID - 1))
	for subAccesoID := int32(1); subAccesoID <= MaxSubAccesoID; subAccesoID++ {
		if !HasSubAcceso(todosOnly, subAccesoID) {
			t.Errorf("Todos must satisfy sub-acceso %d", subAccesoID)
		}
	}

	onlySubFive := uint16(1 << 4)
	if !HasSubAcceso(onlySubFive, 5) {
		t.Error("sub-acceso 5 must be held")
	}
	for _, subAccesoID := range []int32{1, 4, 6, MaxSubAccesoID} {
		if HasSubAcceso(onlySubFive, subAccesoID) {
			t.Errorf("sub-acceso %d must not be held", subAccesoID)
		}
	}
	if HasSubAcceso(0, 1) || HasSubAcceso(onlySubFive, 0) || HasSubAcceso(onlySubFive, MaxSubAccesoID+1) {
		t.Error("an empty mask or an out-of-range id must never be held")
	}
}

// DecodeSubAccesoBytes is the seam where the daemon's opacity ends: fareward copies these bytes out
// of accesos_sub_computed without knowing what a bit means, and this is the only thing that turns
// them back into a mask the catalog can name. Everything below is a shape the daemon can hand over.
func TestDecodeSubAccesoBytes(t *testing.T) {
	for name, check := range map[string]struct {
		subAccesoBytes []byte
		wantMask       uint16
	}{
		// No sub bytes is a legitimate answer, not a corruption: it is an access the gate
		// authorized whose slot contributed nothing to the reply's tail.
		"no bytes at all":  {subAccesoBytes: nil, wantMask: 0},
		"empty slice":      {subAccesoBytes: []byte{}, wantMask: 0},
		"todos only":       {subAccesoBytes: []byte{0x01}, wantMask: 0b1},
		"first byte full":  {subAccesoBytes: []byte{0x7F}, wantMask: 0b1111111},
		"spills to second": {subAccesoBytes: []byte{0x80, 0x01}, wantMask: 0b10000000},
		// Subs 1 and 13. The near-miss worth pinning: sub 13 is bit 12, which lands at bit 5 of
		// the second byte (0x20), not bit 6 (0x40). Reading 0x40 here would grant sub 14.
		"lowest and highest": {subAccesoBytes: []byte{0x81, 0x20}, wantMask: 0b1000000000001},
		// A tail longer than the run is not an error here: the run self-terminates on MORE and the
		// reply packs one run per slot, so the caller slices per slot and never sees the rest.
		"stops at the terminator": {subAccesoBytes: []byte{0x01, 0x7F}, wantMask: 0b1},
	} {
		gotMask, err := DecodeSubAccesoBytes(check.subAccesoBytes)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if gotMask != check.wantMask {
			t.Errorf("%s: decoded %013b; want %013b", name, gotMask, check.wantMask)
		}
	}

	for name, subAccesoBytes := range map[string][]byte{
		"dangling more bit":  {0x81},
		"never terminates":   {0x80, 0x80, 0x80},
		"mask past the ceil": {0xC0, 0xC0, 0x7F},
	} {
		if _, err := DecodeSubAccesoBytes(subAccesoBytes); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
