package index_builder

import "testing"

func TestRemoveAllTokenSequenceOccurrences_RemovesRepeatedBrandSequence(t *testing.T) {
	productTokens := splitNormalizedTokens("Coca Cola Zero Coca Cola 500ml zero")
	brandTokens := splitNormalizedTokens("COCA COLA")

	filteredTokens, removedOccurrences := removeAllTokenSequenceOccurrences(productTokens, brandTokens)
	if removedOccurrences != 2 {
		t.Fatalf("expected 2 brand occurrences removed, got=%d", removedOccurrences)
	}
	if len(filteredTokens) != 3 || filteredTokens[0] != "zero" || filteredTokens[1] != "500ml" || filteredTokens[2] != "zero" {
		t.Fatalf("unexpected filtered tokens: %#v", filteredTokens)
	}
}

func TestSplitNormalizedTokens_MatchesAccentsAndCase(t *testing.T) {
	productTokens := splitNormalizedTokens("Café Perú Tostado CAFE PERU")
	brandTokens := splitNormalizedTokens("cafe peru")

	filteredTokens, removedOccurrences := removeAllTokenSequenceOccurrences(productTokens, brandTokens)
	if removedOccurrences != 2 {
		t.Fatalf("expected 2 brand occurrences removed, got=%d", removedOccurrences)
	}
	if len(filteredTokens) != 1 || filteredTokens[0] != "tostado" {
		t.Fatalf("unexpected filtered tokens: %#v", filteredTokens)
	}
}

func TestRemoveAllTokenSequenceOccurrences_DoesNotRemoveSubstrings(t *testing.T) {
	productTokens := splitNormalizedTokens("detergente soluble 500ml")
	brandTokens := splitNormalizedTokens("sol")

	filteredTokens, removedOccurrences := removeAllTokenSequenceOccurrences(productTokens, brandTokens)
	if removedOccurrences != 0 {
		t.Fatalf("expected 0 removed occurrences, got=%d", removedOccurrences)
	}
	if len(filteredTokens) != 3 || filteredTokens[0] != "detergente" || filteredTokens[1] != "soluble" || filteredTokens[2] != "500ml" {
		t.Fatalf("unexpected filtered tokens: %#v", filteredTokens)
	}
}

func TestRemoveAllTokenSequenceOccurrences_AllowsEmptyResult(t *testing.T) {
	productTokens := splitNormalizedTokens("ab ab")
	brandTokens := splitNormalizedTokens("ab")

	filteredTokens, removedOccurrences := removeAllTokenSequenceOccurrences(productTokens, brandTokens)
	if removedOccurrences != 2 {
		t.Fatalf("expected 2 removed occurrences, got=%d", removedOccurrences)
	}
	if len(filteredTokens) != 0 {
		t.Fatalf("expected empty filtered tokens, got=%#v", filteredTokens)
	}
}
