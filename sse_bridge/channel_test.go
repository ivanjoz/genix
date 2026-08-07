package main

import (
	"encoding/base64"
	"encoding/binary"
	"testing"
)

const testTabID = "N2xQaG8x" // 8 chars = the 6 random bytes of a tab

func TestChannelTokenRoundTrip(t *testing.T) {
	for _, testCase := range []struct{ companyID, userID int32 }{
		{1, 1},
		{7, 42},
		{127, 128},               // the varint boundary: 1 byte → 2 bytes
		{999999, 1},              // 3-byte varint
		{2147483647, 2147483647}, // int32 ceiling
	} {
		channelToken := EncodeChannelToken(testCase.companyID, testCase.userID, testTabID)
		if len(channelToken) == 0 {
			t.Fatalf("no se pudo codificar company=%d user=%d", testCase.companyID, testCase.userID)
		}

		companyID, userID, tabID, decodeError := DecodeChannelToken(channelToken)
		if decodeError != nil {
			t.Fatalf("no se pudo decodificar %q: %v", channelToken, decodeError)
		}
		if companyID != testCase.companyID || userID != testCase.userID || tabID != testTabID {
			t.Fatalf("round-trip incorrecto: %d/%d/%s", companyID, userID, tabID)
		}
	}
}

// The token's whole point is being short: a decimal "7:42:<uuid>" is 41 chars.
func TestChannelTokenIsCompact(t *testing.T) {
	channelToken := EncodeChannelToken(7, 42, testTabID)
	if len(channelToken) != 11 {
		t.Fatalf("se esperaban 11 caracteres para ids pequeños, se obtuvieron %d (%q)", len(channelToken), channelToken)
	}
	if len(testTabID) != 8 {
		t.Fatalf("el tab id debe ocupar 8 caracteres, ocupa %d", len(testTabID))
	}
}

func TestChannelTokenRejectsMalformedInput(t *testing.T) {
	invalidTokens := map[string]string{
		"vacío":              "",
		"no base64url":       "!!!!!!!!!!!",
		"tab de menos bytes": base64.RawURLEncoding.EncodeToString([]byte{7, 42, 1, 2, 3}),
		"company cero":       base64.RawURLEncoding.EncodeToString(append([]byte{0, 42}, make([]byte, tabRandomBytes)...)),
		"sobran bytes":       base64.RawURLEncoding.EncodeToString(append([]byte{7, 42}, make([]byte, tabRandomBytes+1)...)),
	}
	for caseName, invalidToken := range invalidTokens {
		if _, _, _, decodeError := DecodeChannelToken(invalidToken); decodeError == nil {
			t.Fatalf("se aceptó un token inválido (%s): %q", caseName, invalidToken)
		}
	}
}

// Two strings must never name the same channel: otherwise a message published
// with one spelling would miss the stream registered under the other.
func TestChannelTokenRejectsNonCanonicalVarints(t *testing.T) {
	canonicalToken := EncodeChannelToken(7, 42, testTabID)

	// 7 written as an overlong 2-byte varint decodes to the same number.
	overlongBytes := []byte{0x87, 0x00}
	overlongBytes = binary.AppendUvarint(overlongBytes, 42)
	tabBytes, _ := base64.RawURLEncoding.DecodeString(testTabID)
	overlongToken := base64.RawURLEncoding.EncodeToString(append(overlongBytes, tabBytes...))

	if overlongToken == canonicalToken {
		t.Fatal("el caso de prueba no construyó una codificación alternativa")
	}
	if _, _, _, decodeError := DecodeChannelToken(overlongToken); decodeError == nil {
		t.Fatalf("se aceptó una codificación no canónica: %q", overlongToken)
	}
}
