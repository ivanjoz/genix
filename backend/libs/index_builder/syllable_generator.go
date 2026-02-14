package index_builder

import "sort"

// SyllableFrequency describes one token and its usage count.
type SyllableFrequency struct {
	Syllable string
	Count    int32
}

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
	groups := [][]string{
		{"1"}, {"2"}, {"3", "y"}, {"4", "-"}, {"5"}, {"6", "z"}, {"7", "x"}, {"8", "w"}, {"9"},
		{"g", "gr", "grs"},
		{"ml", "mm", "mg", "mgs", "m3", "m2"},
		{"k", "kg", "kgs", "kilo", "kilos", "kilogramos"},
		{"ud", "un", "uns", "unidad", "unidades"},
		{"c", "cm", "cm2", "cm3", "cl"},
		{"l", "lt", "lts", "litro", "litos"},
		{"r", "rr"},
		{"q", "qu"},
		{"p", "pack", "paq", "paquete"},
		{"a"}, {"e"}, {"i"}, {"o"}, {"u"},
		{"b"}, {"d"}, {"f"}, {"h"}, {"j"}, {"m"}, {"n"}, {"s"}, {"t"}, {"v"},
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
