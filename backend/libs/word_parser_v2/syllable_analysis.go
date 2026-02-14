package word_parser_v2

import (
	"sort"
	"strings"
)

type SyllableWordMapEntry struct {
	Syllable string
	Length   int
	Count    int
	Words    []string
}

type SyllableCoverageMetrics struct {
	TotalWords                 int
	WordsWithSingleLetterParts int
	SingleLetterUsage          int
	TwoLetterUsage             int
	ThreeLetterUsage           int
	OtherLengthUsage           int
}

type segmentationCost struct {
	SingleLetterCount int
	TokenCount        int
}

func BuildTopSyllableWordMap(productNames []string, topN int, includeLengths map[int]bool) []SyllableWordMapEntry {
	if topN <= 0 {
		return nil
	}

	type syllableStats struct {
		Count   int
		WordSet map[string]struct{}
	}
	statsBySyllable := make(map[string]*syllableStats, 1024)

	for _, rawProductName := range productNames {
		normalizedProductName := normalizeText(rawProductName)
		if normalizedProductName == "" {
			continue
		}
		for _, wordToken := range strings.Fields(normalizedProductName) {
			normalizedWordToken := normalizeToken(wordToken)
			if normalizedWordToken == "" {
				continue
			}
			for _, extractedSyllable := range splitWordIntoSyllables(normalizedWordToken) {
				syllableLength := len([]rune(extractedSyllable))
				if includeLengths != nil {
					if _, shouldInclude := includeLengths[syllableLength]; !shouldInclude {
						continue
					}
				}
				currentStats, exists := statsBySyllable[extractedSyllable]
				if !exists {
					currentStats = &syllableStats{WordSet: make(map[string]struct{}, 8)}
					statsBySyllable[extractedSyllable] = currentStats
				}
				currentStats.Count++
				if len(currentStats.WordSet) < 8 {
					currentStats.WordSet[normalizedWordToken] = struct{}{}
				}
			}
		}
	}

	entries := make([]SyllableWordMapEntry, 0, len(statsBySyllable))
	for syllable, stats := range statsBySyllable {
		wordList := make([]string, 0, len(stats.WordSet))
		for word := range stats.WordSet {
			wordList = append(wordList, word)
		}
		sort.Strings(wordList)
		entries = append(entries, SyllableWordMapEntry{
			Syllable: syllable,
			Length:   len([]rune(syllable)),
			Count:    stats.Count,
			Words:    wordList,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		if entries[i].Length != entries[j].Length {
			return entries[i].Length < entries[j].Length
		}
		return entries[i].Syllable < entries[j].Syllable
	})

	if len(entries) > topN {
		entries = entries[:topN]
	}
	return entries
}

func BuildPrioritizedTopNSyllables(topEntries []SyllableWordMapEntry, targetCount int, twoLetterRatio float64) []string {
	if targetCount <= 0 {
		return nil
	}
	if twoLetterRatio < 0 {
		twoLetterRatio = 0
	}
	if twoLetterRatio > 1 {
		twoLetterRatio = 1
	}

	twoLetterTarget := int(float64(targetCount) * twoLetterRatio)
	threeLetterTarget := targetCount - twoLetterTarget
	selected := make([]string, 0, targetCount)
	selectedSet := make(map[string]struct{}, targetCount)

	addedTwoLetter := 0
	for _, entry := range topEntries {
		if len(selected) >= targetCount || addedTwoLetter >= twoLetterTarget {
			break
		}
		if entry.Length != 2 {
			continue
		}
		if _, exists := selectedSet[entry.Syllable]; exists {
			continue
		}
		selected = append(selected, entry.Syllable)
		selectedSet[entry.Syllable] = struct{}{}
		addedTwoLetter++
	}

	addedThreeLetter := 0
	for _, entry := range topEntries {
		if len(selected) >= targetCount || addedThreeLetter >= threeLetterTarget {
			break
		}
		if entry.Length != 3 {
			continue
		}
		if _, exists := selectedSet[entry.Syllable]; exists {
			continue
		}
		selected = append(selected, entry.Syllable)
		selectedSet[entry.Syllable] = struct{}{}
		addedThreeLetter++
	}

	for _, entry := range topEntries {
		if len(selected) >= targetCount {
			break
		}
		if _, exists := selectedSet[entry.Syllable]; exists {
			continue
		}
		selected = append(selected, entry.Syllable)
		selectedSet[entry.Syllable] = struct{}{}
	}

	return selected
}

// BuildCoverageGreedyTopNSyllables selects syllables by marginal reduction of 1-letter fallback usage.
func BuildCoverageGreedyTopNSyllables(
	productNames []string,
	fixedSlots []string,
	candidateEntries []SyllableWordMapEntry,
	targetCount int,
) []string {
	if targetCount <= 0 || len(candidateEntries) == 0 {
		return nil
	}

	type uniqueWordStats struct {
		Word                 string
		ExtractedSyllables   []string
		ExtractedSyllableSet map[string]struct{}
		Count                int
		CurrentCost          segmentationCost
	}

	// Aggregate repeated words once and weight gains by frequency.
	uniqueWordStatsByToken := make(map[string]*uniqueWordStats, 8192)
	for _, rawProductName := range productNames {
		normalizedProductName := normalizeText(rawProductName)
		if normalizedProductName == "" {
			continue
		}
		for _, rawWordToken := range strings.Fields(normalizedProductName) {
			normalizedWordToken := normalizeToken(rawWordToken)
			if normalizedWordToken == "" {
				continue
			}
			existingStats, exists := uniqueWordStatsByToken[normalizedWordToken]
			if !exists {
				extractedSyllables := splitWordIntoSyllables(normalizedWordToken)
				if len(extractedSyllables) == 0 {
					extractedSyllables = []string{normalizedWordToken}
				}
				extractedSyllableSet := make(map[string]struct{}, len(extractedSyllables))
				for _, extractedSyllable := range extractedSyllables {
					extractedSyllableSet[extractedSyllable] = struct{}{}
				}
				existingStats = &uniqueWordStats{
					Word:                 normalizedWordToken,
					ExtractedSyllables:   extractedSyllables,
					ExtractedSyllableSet: extractedSyllableSet,
				}
				uniqueWordStatsByToken[normalizedWordToken] = existingStats
			}
			existingStats.Count++
		}
	}

	uniqueWordStatsList := make([]*uniqueWordStats, 0, len(uniqueWordStatsByToken))
	for _, currentStats := range uniqueWordStatsByToken {
		uniqueWordStatsList = append(uniqueWordStatsList, currentStats)
	}
	sort.Slice(uniqueWordStatsList, func(i, j int) bool {
		return uniqueWordStatsList[i].Word < uniqueWordStatsList[j].Word
	})

	candidateSyllables := make([]string, 0, len(candidateEntries))
	candidateFrequencyBySyllable := make(map[string]int, len(candidateEntries))
	candidateWordIndexes := make(map[string][]int, len(candidateEntries))
	for _, candidateEntry := range candidateEntries {
		if candidateEntry.Length < 2 || candidateEntry.Length > 3 {
			continue
		}
		candidateSyllable := normalizeToken(candidateEntry.Syllable)
		if candidateSyllable == "" {
			continue
		}
		if _, exists := candidateFrequencyBySyllable[candidateSyllable]; exists {
			continue
		}
		candidateSyllables = append(candidateSyllables, candidateSyllable)
		candidateFrequencyBySyllable[candidateSyllable] = candidateEntry.Count
	}
	for wordIndex, currentWordStats := range uniqueWordStatsList {
		for _, candidateSyllable := range candidateSyllables {
			if _, exists := currentWordStats.ExtractedSyllableSet[candidateSyllable]; exists {
				candidateWordIndexes[candidateSyllable] = append(candidateWordIndexes[candidateSyllable], wordIndex)
			}
		}
	}

	selectedSyllables := make([]string, 0, targetCount)
	selectedSet := make(map[string]struct{}, len(fixedSlots)+targetCount)
	for _, fixedSyllable := range fixedSlots {
		normalizedFixedSyllable := normalizeToken(fixedSyllable)
		if normalizedFixedSyllable == "" {
			continue
		}
		selectedSet[normalizedFixedSyllable] = struct{}{}
	}

	for _, currentWordStats := range uniqueWordStatsList {
		currentWordStats.CurrentCost = extractedSyllableCost(currentWordStats.ExtractedSyllables, selectedSet)
	}

	for len(selectedSyllables) < targetCount {
		bestCandidate := ""
		bestGainSingleLetters := 0
		bestGainTokens := 0
		bestCandidateFrequency := 0

		for _, candidateSyllable := range candidateSyllables {
			if _, alreadySelected := selectedSet[candidateSyllable]; alreadySelected {
				continue
			}

			affectedWordIndexes := candidateWordIndexes[candidateSyllable]
			if len(affectedWordIndexes) == 0 {
				continue
			}

			selectedSet[candidateSyllable] = struct{}{}
			gainSingleLetters := 0
			gainTokens := 0
			for _, affectedWordIndex := range affectedWordIndexes {
				currentWordStats := uniqueWordStatsList[affectedWordIndex]
				nextCost := extractedSyllableCost(currentWordStats.ExtractedSyllables, selectedSet)
				if nextCost.SingleLetterCount > currentWordStats.CurrentCost.SingleLetterCount {
					continue
				}
				gainSingleLetters += (currentWordStats.CurrentCost.SingleLetterCount - nextCost.SingleLetterCount) * currentWordStats.Count
				if nextCost.SingleLetterCount == currentWordStats.CurrentCost.SingleLetterCount {
					gainTokens += (currentWordStats.CurrentCost.TokenCount - nextCost.TokenCount) * currentWordStats.Count
				}
			}
			delete(selectedSet, candidateSyllable)

			candidateFrequency := candidateFrequencyBySyllable[candidateSyllable]
			if gainSingleLetters > bestGainSingleLetters ||
				(gainSingleLetters == bestGainSingleLetters && gainTokens > bestGainTokens) ||
				(gainSingleLetters == bestGainSingleLetters && gainTokens == bestGainTokens && candidateFrequency > bestCandidateFrequency) ||
				(gainSingleLetters == bestGainSingleLetters && gainTokens == bestGainTokens && candidateFrequency == bestCandidateFrequency && (bestCandidate == "" || candidateSyllable < bestCandidate)) {
				bestCandidate = candidateSyllable
				bestGainSingleLetters = gainSingleLetters
				bestGainTokens = gainTokens
				bestCandidateFrequency = candidateFrequency
			}
		}

		if bestCandidate == "" {
			break
		}

		selectedSet[bestCandidate] = struct{}{}
		selectedSyllables = append(selectedSyllables, bestCandidate)
		for _, affectedWordIndex := range candidateWordIndexes[bestCandidate] {
			currentWordStats := uniqueWordStatsList[affectedWordIndex]
			currentWordStats.CurrentCost = extractedSyllableCost(currentWordStats.ExtractedSyllables, selectedSet)
		}
	}

	return selectedSyllables
}

func EvaluateSyllableSetCoverage(productNames []string, selectedSyllables []string) SyllableCoverageMetrics {
	selectedSet := make(map[string]struct{}, len(selectedSyllables)+64)
	for _, syllable := range selectedSyllables {
		normalizedSyllable := normalizeToken(syllable)
		if normalizedSyllable == "" {
			continue
		}
		selectedSet[normalizedSyllable] = struct{}{}
	}

	for runeValue := 'a'; runeValue <= 'z'; runeValue++ {
		selectedSet[string(runeValue)] = struct{}{}
	}
	for digitRune := '1'; digitRune <= '9'; digitRune++ {
		selectedSet[string(digitRune)] = struct{}{}
	}

	metrics := SyllableCoverageMetrics{}
	for _, rawProductName := range productNames {
		normalizedProductName := normalizeText(rawProductName)
		if normalizedProductName == "" {
			continue
		}
		for _, wordToken := range strings.Fields(normalizedProductName) {
			normalizedWordToken := normalizeToken(wordToken)
			if normalizedWordToken == "" {
				continue
			}
			metrics.TotalWords++
			wordHasSingleLetterParts := false

			extractedSyllables := splitWordIntoSyllables(normalizedWordToken)
			if len(extractedSyllables) == 0 {
				extractedSyllables = []string{normalizedWordToken}
			}

			for _, extractedSyllable := range extractedSyllables {
				if _, exists := selectedSet[extractedSyllable]; exists {
					syllableLength := len([]rune(extractedSyllable))
					switch syllableLength {
					case 1:
						metrics.SingleLetterUsage++
						wordHasSingleLetterParts = true
					case 2:
						metrics.TwoLetterUsage++
					case 3:
						metrics.ThreeLetterUsage++
					default:
						metrics.OtherLengthUsage++
					}
					continue
				}

				for _, singleRune := range extractedSyllable {
					if _, exists := selectedSet[string(singleRune)]; exists {
						metrics.SingleLetterUsage++
						wordHasSingleLetterParts = true
					}
				}
			}

			if wordHasSingleLetterParts {
				metrics.WordsWithSingleLetterParts++
			}
		}
	}

	return metrics
}

func extractedSyllableCost(extractedSyllables []string, selectedSet map[string]struct{}) segmentationCost {
	cost := segmentationCost{}
	for _, extractedSyllable := range extractedSyllables {
		if extractedSyllable == "" {
			continue
		}
		if _, exists := selectedSet[extractedSyllable]; exists {
			cost.TokenCount++
			continue
		}
		// Mirror encoder fallback: unknown extracted chunk is emitted as per-rune tokens.
		runeCount := len([]rune(extractedSyllable))
		cost.SingleLetterCount += runeCount
		cost.TokenCount += runeCount
	}
	return cost
}
