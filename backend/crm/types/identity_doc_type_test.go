package types

import "testing"

func TestDeriveIdentityDocTypeFallsBackToTheShape(t *testing.T) {
	cases := map[string]int8{
		"20000000002": IdentityDocRUC,
		"12345678":    IdentityDocDNI,
		"":            0,
		"X1234":       IdentityDocForeign,
	}
	for registryNumber, want := range cases {
		if got := DeriveIdentityDocType(registryNumber); got != want {
			t.Errorf("DeriveIdentityDocType(%q) = %v, want %v", registryNumber, got, want)
		}
	}
}

// The catalog is the only place the ids and their SUNAT codes are written down, and the
// frontend picker is generated from it — a duplicate id would silently give two document
// types the same option, and a duplicate code would print the wrong character on a book.
func TestIdentityDocOptionsAreUnique(t *testing.T) {
	seenIDs, seenCodes := map[int8]bool{}, map[string]bool{}
	for _, option := range IdentityDocOptions {
		if option.ID == 0 {
			t.Error("0 is reserved for 'not declared' and cannot name a catalog entry")
		}
		if seenIDs[option.ID] {
			t.Errorf("id %d appears twice in the catalog", option.ID)
		}
		if seenCodes[option.Code] {
			t.Errorf("SUNAT code %q appears twice in the catalog", option.Code)
		}
		seenIDs[option.ID], seenCodes[option.Code] = true, true
	}
}

// 0 is not a catalog entry, and an id nobody assigned must not resolve to a code: the save
// path uses this to tell "nothing declared" from "declared something invalid".
func TestSunatIdentityDocCodeRejectsUnknownIDs(t *testing.T) {
	for _, docType := range []int8{0, 2, 3, 5, 8, 16, -1} {
		if code := SunatIdentityDocCode(docType); code != "" {
			t.Errorf("SunatIdentityDocCode(%d) = %q, want the empty string", docType, code)
		}
		if IsIdentityDocType(docType) {
			t.Errorf("IsIdentityDocType(%d) must be false", docType)
		}
	}
}

// The one id that does not read as its own SUNAT code. It exists because 0 is the zero
// value — "nothing declared" — so SUNAT's own "0" had to go somewhere else.
func TestSunatCodeZeroIsIdentityDocNone(t *testing.T) {
	if SunatIdentityDocCode(IdentityDocNone) != "0" {
		t.Errorf("IdentityDocNone must print SUNAT's \"0\", got %q",
			SunatIdentityDocCode(IdentityDocNone))
	}
	if IdentityDocNone == 0 {
		t.Error("IdentityDocNone must not be 0: that value means the client declared nothing")
	}
}
