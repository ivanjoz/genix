package routing

import "testing"

func TestLocalizedResponseFallsBackToSpanish(t *testing.T) {
	if got := LocalizedResponse(ResponseOutOfScope, LanguageMixed); got != localizedResponses[ResponseOutOfScope][LanguageSpanish] {
		t.Fatalf("unexpected fallback response: %q", got)
	}
}
