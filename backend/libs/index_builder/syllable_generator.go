package index_builder

import "sort"

// SyllableFrequency describes one token and its usage count.
type SyllableFrequency struct {
	Syllable string
	Count    int32
}

var spanishVowels = []string{"a", "e", "i", "o", "u"}
var spanishConsonants = []string{"b", "c", "d", "f", "g", "h", "j", "k", "l", "m", "n", "p", "q", "r", "s", "t", "v", "w", "x", "y", "z"}

// SplitTokenIntoSyllables exposes the package syllable splitter for tests/tools.
func SplitTokenIntoSyllables(token string) []string {
	return splitTokenIntoSyllables(token)
}

func splitTokenIntoSyllables(token string) []string {
	if token == "" {
		return nil
	}
	if len(token) <= 2 {
		return []string{token}
	}
	sy := make([]string, 0, (len(token)+1)/2)
	for index := 0; index < len(token); index += 2 {
		end := index + 2
		if end > len(token) {
			end = len(token)
		}
		sy = append(sy, token[index:end])
	}
	return sy
}

func defaultFixedAliases() (map[string]string, []string) {
	baseGroups := [][]string{
		{"1"}, {"2"}, {"y", "3"}, {"4"}, {"5"}, {"z", "6"}, {"x", "7"}, {"w", "8"}, {"9"},
		{"g", "gr", "grs"},
		{"ml", "mm", "mg", "mgs", "m3", "m2"},
		{"k", "kg", "kgs", "kilo", "kilos", "kilogramos"},
		{"ud", "un", "uns", "unidad", "unidades"},
		{"c", "cm", "cm2", "cm3", "cl"},
		{"l", "lt", "lts", "litro", "litos"},
		{"r", "rr"},
		{"q", "qu"},
		{"p", "pack", "paq", "paquete"},
	}
	groups := make([][]string, 0, len(baseGroups)+len(spanishVowels)+len(spanishConsonants))
	groups = append(groups, baseGroups...)
	canonicalTokenSet := make(map[string]struct{}, len(baseGroups)+len(spanishVowels)+len(spanishConsonants))
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		normalizedCanonical := normalizeToken(group[0])
		if normalizedCanonical == "" {
			continue
		}
		canonicalTokenSet[normalizedCanonical] = struct{}{}
	}
	// Keep frontend/backend parity by computing missing single-letter canonicals dynamically.
	for _, singleLetterToken := range append(append([]string(nil), spanishVowels...), spanishConsonants...) {
		normalizedSingle := normalizeToken(singleLetterToken)
		if normalizedSingle == "" {
			continue
		}
		if _, exists := canonicalTokenSet[normalizedSingle]; exists {
			continue
		}
		groups = append(groups, []string{normalizedSingle})
		canonicalTokenSet[normalizedSingle] = struct{}{}
	}

	aliasToCanonical := make(map[string]string, 128)
	canonicalTokens := make([]string, 0, len(groups))
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		canonical := normalizeToken(group[0])
		if canonical == "" {
			continue
		}
		canonicalTokens = append(canonicalTokens, canonical)
		for _, alias := range group {
			normalizedAlias := normalizeToken(alias)
			if normalizedAlias == "" {
				continue
			}
			aliasToCanonical[normalizedAlias] = canonical
		}
	}
	return aliasToCanonical, canonicalTokens
}

// ExtractTopSyllableFrequencies computes sorted syllable usage rows from input records.
func ExtractTopSyllableFrequencies(records []RecordInput, limit int32) []SyllableFrequency {
	if len(records) == 0 || limit <= 0 {
		return nil
	}
	connectorSet := make(map[string]struct{}, len(DefaultOptions().ConnectorWords))
	for _, connectorWord := range DefaultOptions().ConnectorWords {
		connectorSet[normalizeToken(connectorWord)] = struct{}{}
	}

	normalizedRecords := make([]normalizedRecord, 0, len(records))
	for _, inputRecord := range records {
		tokens := normalizeAndFilterTokens(inputRecord.Text, connectorSet)
		if len(tokens) == 0 {
			continue
		}
		normalizedRecords = append(normalizedRecords, normalizedRecord{id: inputRecord.ID, tokens: tokens})
	}
	aliasToCanonical, _ := defaultFixedAliases()
	frequencyBySyllable := extractSyllableFrequency(normalizedRecords, aliasToCanonical)
	rows := make([]SyllableFrequency, 0, len(frequencyBySyllable))
	for syllable, count := range frequencyBySyllable {
		rows = append(rows, SyllableFrequency{Syllable: syllable, Count: int32(count)})
	}
	sort.SliceStable(rows, func(leftIndex, rightIndex int) bool {
		if rows[leftIndex].Count != rows[rightIndex].Count {
			return rows[leftIndex].Count > rows[rightIndex].Count
		}
		return rows[leftIndex].Syllable < rows[rightIndex].Syllable
	})
	if int(limit) < len(rows) {
		rows = rows[:limit]
	}
	return rows
}
