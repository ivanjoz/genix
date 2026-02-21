package index_builder

import "testing"

func TestOptimizeProductTextAggressive_RemovesAllBrandOccurrencesAndDedupes(t *testing.T) {
	optimizedText, optimizationStats := OptimizeProductTextAggressive(
		"Coca Cola Zero Coca Cola 500ml zero",
		"COCA COLA",
	)

	if optimizedText != "zero 500ml" {
		t.Fatalf("unexpected optimized text: %q", optimizedText)
	}
	if optimizationStats.RemovedBrandOccurrences != 2 {
		t.Fatalf("expected 2 brand occurrences removed, got=%d", optimizationStats.RemovedBrandOccurrences)
	}
	if optimizationStats.RemovedDuplicateTokens != 1 {
		t.Fatalf("expected 1 duplicate token removed, got=%d", optimizationStats.RemovedDuplicateTokens)
	}
	if !optimizationStats.Changed {
		t.Fatalf("expected changed=true")
	}
}

func TestOptimizeProductTextAggressive_MatchesAccentsAndCase(t *testing.T) {
	optimizedText, optimizationStats := OptimizeProductTextAggressive(
		"Café Perú Tostado CAFE PERU",
		"cafe peru",
	)

	if optimizedText != "tostado" {
		t.Fatalf("unexpected optimized text: %q", optimizedText)
	}
	if optimizationStats.RemovedBrandOccurrences != 2 {
		t.Fatalf("expected 2 brand occurrences removed, got=%d", optimizationStats.RemovedBrandOccurrences)
	}
}

func TestOptimizeProductTextAggressive_DoesNotRemoveSubstrings(t *testing.T) {
	optimizedText, optimizationStats := OptimizeProductTextAggressive(
		"detergente soluble 500ml",
		"sol",
	)

	if optimizedText != "detergente soluble 500ml" {
		t.Fatalf("unexpected optimized text: %q", optimizedText)
	}
	if optimizationStats.RemovedBrandOccurrences != 0 {
		t.Fatalf("expected 0 removed brand occurrences, got=%d", optimizationStats.RemovedBrandOccurrences)
	}
	if optimizationStats.Changed {
		t.Fatalf("expected changed=false")
	}
}

func TestOptimizeProductTextAggressive_AllowsEmptyResult(t *testing.T) {
	optimizedText, optimizationStats := OptimizeProductTextAggressive(
		"ab ab",
		"ab",
	)

	if optimizedText != "" {
		t.Fatalf("expected empty optimized text, got=%q", optimizedText)
	}
	if optimizationStats.RemovedBrandOccurrences != 2 {
		t.Fatalf("expected 2 removed occurrences, got=%d", optimizationStats.RemovedBrandOccurrences)
	}
}
