package db

import (
	"slices"
	"strings"
	"unicode"
)

const (
	maxTextSearchWords         = 12
	maxTextSearchCombination   = 3
	textSearchHashWordBoundary = "\x1f"
)

type textSearchIndexRow struct {
	partitionID int64
	id          int64
	hash        int32
	bigrams     []int8
	status      int8
}

func makeCommonSpanishWordSet() map[string]struct{} {
	commonWords := make(map[string]struct{}, len(CommonSpanishWords))
	for _, commonWord := range CommonSpanishWords {
		commonWords[normalizeTextSearchToken(commonWord)] = struct{}{}
	}
	return commonWords
}

var commonSpanishWordSet = makeCommonSpanishWordSet()

func parseTextSearchWords(rawText string) []string {
	normalizedWords := make([]string, 0, maxTextSearchWords)
	for _, rawToken := range strings.Fields(rawText) {
		normalizedToken := normalizeTextSearchToken(rawToken)
		if len(normalizedToken) <= 1 {
			continue
		}
		if _, isCommonWord := commonSpanishWordSet[normalizedToken]; isCommonWord {
			continue
		}

		normalizedWords = append(normalizedWords, normalizedToken)
		if len(normalizedWords) == maxTextSearchWords {
			break
		}
	}
	return normalizedWords
}

func normalizeTextSearchToken(rawToken string) string {
	var normalizedToken strings.Builder
	previousRune := rune(0)

	for _, currentRune := range strings.ToLower(rawToken) {
		currentRune = normalizeTextSearchRune(currentRune)
		if currentRune == 0 {
			previousRune = 0
			continue
		}
		if currentRune == previousRune && currentRune >= 'a' && currentRune <= 'z' {
			continue
		}

		normalizedToken.WriteRune(currentRune)
		previousRune = currentRune
	}
	return normalizedToken.String()
}

func normalizeTextSearchRune(rawRune rune) rune {
	switch rawRune {
	case 'á', 'à', 'ä', 'â':
		return 'a'
	case 'é', 'è', 'ë', 'ê':
		return 'e'
	case 'í', 'ì', 'ï', 'î':
		return 'i'
	case 'ó', 'ò', 'ö', 'ô':
		return 'o'
	case 'ú', 'ù', 'ü', 'û':
		return 'u'
	case 'ñ':
		return 'n'
	}

	if rawRune >= 'a' && rawRune <= 'z' {
		return rawRune
	}
	if unicode.IsDigit(rawRune) {
		return rawRune
	}
	return 0
}

func textSearchWordHasNumber(word string) bool {
	for _, currentRune := range word {
		if unicode.IsDigit(currentRune) {
			return true
		}
	}
	return false
}

func makeTextSearchBigrams(words []string) []uint8 {
	bigrams := make([]uint8, 0, len(words))
	for _, word := range words {
		if textSearchWordHasNumber(word) {
			continue
		}

		for currentIndex := 0; currentIndex < len(word)-1; currentIndex++ {
			bigramValue, exists := BigramMap[word[currentIndex:currentIndex+2]]
			if !exists {
				continue
			}
			bigrams = append(bigrams, bigramValue)
			break
		}
	}
	return bigrams
}

func convertTextSearchBigramsToInt8(bigrams []uint8) []int8 {
	convertedBigrams := make([]int8, 0, len(bigrams))
	for _, bigram := range bigrams {
		convertedBigrams = append(convertedBigrams, int8(bigram))
	}
	return convertedBigrams
}

func buildTextSearchRows(partitionID int64, baseID int64, status int8, rawText string) []textSearchIndexRow {
	words := parseTextSearchWords(rawText)
	if len(words) == 0 {
		return nil
	}

	bigrams := convertTextSearchBigramsToInt8(makeTextSearchBigrams(words))
	uniqueHashes := map[int32]struct{}{}

	for combinationSize := 1; combinationSize <= maxTextSearchCombination && combinationSize <= len(words); combinationSize++ {
		appendTextSearchCombinationHashes(words, combinationSize, 0, nil, uniqueHashes)
	}

	hashes := make([]int32, 0, len(uniqueHashes))
	for hashValue := range uniqueHashes {
		hashes = append(hashes, hashValue)
	}
	slices.Sort(hashes)

	rows := make([]textSearchIndexRow, 0, len(hashes))
	for _, hashValue := range hashes {
		rows = append(rows, textSearchIndexRow{
			partitionID: partitionID,
			id:          baseID,
			hash:        hashValue,
			bigrams:     bigrams,
			status:      status,
		})
	}
	return rows
}

func appendTextSearchCombinationHashes(
	words []string,
	targetSize int,
	startIndex int,
	currentCombination []string,
	uniqueHashes map[int32]struct{},
) {
	if len(currentCombination) == targetSize {
		// The delimiter prevents ambiguous joins such as ["ab", "c"] vs ["a", "bc"].
		uniqueHashes[BasicHashInt(strings.Join(currentCombination, textSearchHashWordBoundary))] = struct{}{}
		return
	}

	remainingSlots := targetSize - len(currentCombination)
	for wordIndex := startIndex; wordIndex <= len(words)-remainingSlots; wordIndex++ {
		nextCombination := append(append([]string{}, currentCombination...), words[wordIndex])
		appendTextSearchCombinationHashes(words, targetSize, wordIndex+1, nextCombination, uniqueHashes)
	}
}
