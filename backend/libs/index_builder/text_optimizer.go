package index_builder

import "strings"

type ProductTextOptimizationStats struct {
	Changed                  bool
	RemovedBrandOccurrences  int
	RemovedDuplicateTokens   int
	OriginalNormalizedTokens int
	FinalNormalizedTokens    int
}

// OptimizeProductTextAggressive removes all brand-token occurrences and duplicate tokens
// from normalized product text to reduce stage-1 content bytes.
func OptimizeProductTextAggressive(productName string, brandName string) (string, ProductTextOptimizationStats) {
	productTokens := splitNormalizedTokens(productName)
	brandTokens := splitNormalizedTokens(brandName)

	optimizationStats := ProductTextOptimizationStats{
		OriginalNormalizedTokens: len(productTokens),
		FinalNormalizedTokens:    len(productTokens),
	}
	if len(productTokens) == 0 {
		return "", optimizationStats
	}

	if len(brandTokens) > 0 {
		filteredTokens, removedOccurrences := removeAllTokenSequenceOccurrences(productTokens, brandTokens)
		productTokens = filteredTokens
		optimizationStats.RemovedBrandOccurrences = removedOccurrences
	}

	dedupedTokens := make([]string, 0, len(productTokens))
	seenTokens := make(map[string]struct{}, len(productTokens))
	for _, normalizedToken := range productTokens {
		if _, alreadySeen := seenTokens[normalizedToken]; alreadySeen {
			optimizationStats.RemovedDuplicateTokens++
			continue
		}
		seenTokens[normalizedToken] = struct{}{}
		dedupedTokens = append(dedupedTokens, normalizedToken)
	}
	productTokens = dedupedTokens

	optimizationStats.FinalNormalizedTokens = len(productTokens)
	optimizedText := strings.Join(productTokens, " ")
	optimizationStats.Changed = optimizedText != normalizeText(productName)
	return optimizedText, optimizationStats
}

func splitNormalizedTokens(rawText string) []string {
	normalizedText := normalizeText(rawText)
	if normalizedText == "" {
		return nil
	}
	return strings.Fields(normalizedText)
}

func removeAllTokenSequenceOccurrences(sourceTokens []string, sequenceTokens []string) ([]string, int) {
	if len(sourceTokens) == 0 || len(sequenceTokens) == 0 || len(sequenceTokens) > len(sourceTokens) {
		return append([]string(nil), sourceTokens...), 0
	}

	filteredTokens := make([]string, 0, len(sourceTokens))
	removedOccurrences := 0
	currentIndex := 0
	for currentIndex < len(sourceTokens) {
		if currentIndex+len(sequenceTokens) <= len(sourceTokens) &&
			tokenSequenceEquals(sourceTokens[currentIndex:currentIndex+len(sequenceTokens)], sequenceTokens) {
			removedOccurrences++
			currentIndex += len(sequenceTokens)
			continue
		}
		filteredTokens = append(filteredTokens, sourceTokens[currentIndex])
		currentIndex++
	}
	return filteredTokens, removedOccurrences
}

func tokenSequenceEquals(leftTokens []string, rightTokens []string) bool {
	if len(leftTokens) != len(rightTokens) {
		return false
	}
	for tokenIndex := 0; tokenIndex < len(leftTokens); tokenIndex++ {
		if leftTokens[tokenIndex] != rightTokens[tokenIndex] {
			return false
		}
	}
	return true
}
