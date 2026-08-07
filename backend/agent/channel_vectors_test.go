package agent

import (
	"fmt"
	"testing"
)

// TestCrossLanguageVectors pins the wire format across the three
// implementations. The expected column was produced by the TypeScript codec
// (frontend/core/agent/channel.ts); if this test fails, a browser can no longer
// name a channel the server understands.
func TestCrossLanguageVectors(t *testing.T) {
	vectors := []struct {
		companyID, userID int32
		tabID             string
		expectedToken     string
	}{
		{1, 1, "N2xQaG8x", "AQE3bFBobzE"},
		{7, 42, "N2xQaG8x", "Byo3bFBobzE"},
		{127, 128, "N2xQaG8x", "f4ABN2xQaG8x"},
		{128, 127, "AAAAAAAA", "gAF_AAAAAAAA"},
		{999999, 1, "____buff", "v4Q9Af___27n3w"},
		{2147483647, 2147483647, "-_-_-_-_", "_____wf_____B_v_v_v_vw"},
		{16383, 16384, "N2xQaG8x", "_3-AgAE3bFBobzE"},
		{2097151, 2097152, "dGFyZGlv", "__9_gICAAXRhcmRpbw"},
	}
	for _, vector := range vectors {
		goToken := EncodeChannelToken(vector.companyID, vector.userID, vector.tabID)
		if goToken != vector.expectedToken {
			t.Fatalf("Go y TypeScript divergen para %d/%d/%s: go=%q ts=%q",
				vector.companyID, vector.userID, vector.tabID, goToken, vector.expectedToken)
		}
		companyID, userID, tabID, decodeError := DecodeChannelToken(vector.expectedToken)
		if decodeError != nil {
			t.Fatalf("Go no pudo decodificar el token de TypeScript %q: %v", vector.expectedToken, decodeError)
		}
		if companyID != vector.companyID || userID != vector.userID || tabID != vector.tabID {
			t.Fatalf("decodificación incorrecta de %q: %d/%d/%s", vector.expectedToken, companyID, userID, tabID)
		}
		fmt.Printf("  ok %d/%d/%s → %s (%d chars)\n", companyID, userID, tabID, goToken, len(goToken))
	}
}
