package word_parser_v2

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	binaryIndexMagicV3        = "WPV3IDX1"
	binaryIndexVersion        = uint8(2)
	headerFlagDictionaryDelta = uint8(1 << 0)
	headerFlagShapeDeltaC0    = uint8(1 << 1)
	headerFlagShapeDeltaC1    = uint8(1 << 2)
	headerFlagShapeInlineC0   = uint8(1 << 3)
	headerFlagShapeInlineC1   = uint8(1 << 4)
	headerFlagShapeCompactV2  = uint8(1 << 5)
	// Header layout V2 keeps compact shape counters in uint8 and stores overflow in uint16.
	headerSizeV3 = 33
)

const (
	shapeClassTwoBit  = uint8(0)
	shapeClassFourBit = uint8(1)
	shapeCountU8Limit = 255
)

const (
	shapeDeltaDecoratorRaw          = uint8(0)
	shapeDeltaDecoratorSmallMut2Bit = uint8(1)
	shapeDeltaDecoratorPrefixAppend = uint8(2)
	shapeDeltaDecoratorTrimAppend   = uint8(3)
	compactNumericTokenPrefix       = "n0"
)

// EncodedNameRecord stores one normalized product name as shape + syllable IDs.
type EncodedNameRecord struct {
	WordSizes      []uint8
	EncodedContent []uint8
}

type shapeBucket struct {
	WordSizes          []uint8
	RecordContents     [][]uint8
	TotalSyllableCount int
}

type dictionaryTokenEntry struct {
	Token string
	ID    uint8
}

type shapeDeltaStreamStats struct {
	NormalDeltaByteCount  int
	EscapedDeltaByteCount int
	EscapedDeltaCount     int
	FirstShapeValueByteCount int
	TotalDeltaStreamBytes int
}

type sortedRecordByShapeValue struct {
	ShapeValue     uint32
	WordSizes      []uint8
	EncodedContent []uint8
}

type topWordFamilyCandidate struct {
	FamilyKey      string
	CanonicalWord  string
	AliasWords     []string
	TotalFrequency int
}

// BuildNamesBinaryIndexFromFile reads product names, normalizes/removes connectors, encodes, and writes productos.idx.
func BuildNamesBinaryIndexFromFile(
	inputProductosPath string,
	outputIndexPath string,
	fixedConfig FixedSyllableGeneratorConfig,
	frequentConfig FrequentSyllableGeneratorConfig,
) error {
	inputFileInfo, inputStatError := os.Stat(inputProductosPath)
	if inputStatError != nil {
		return fmt.Errorf("stat input file: %w", inputStatError)
	}

	productNames, loadProductNamesError := LoadProductNamesFromFile(inputProductosPath)
	if loadProductNamesError != nil {
		return loadProductNamesError
	}
	if len(productNames) == 0 {
		return fmt.Errorf("no product names found in input file")
	}

	connectorSet := buildConnectorSet(fixedConfig.ConnectorTokens)
	cleanedProductNames := make([]string, 0, len(productNames))
	totalConnectorTokensRemoved := 0
	totalRecordsTruncatedToEightWords := 0
	totalWordsTruncated := 0
	for _, rawProductName := range productNames {
		normalizedTokens := normalizeAndFilterTokens(rawProductName, connectorSet)
		rawTokenCount := len(strings.Fields(normalizeText(rawProductName)))
		if rawTokenCount > len(normalizedTokens) {
			totalConnectorTokensRemoved += rawTokenCount - len(normalizedTokens)
		}
		if len(normalizedTokens) == 0 {
			continue
		}
		// Test strategy: cap each phrase to 8 words to enforce compact shapes.
		if len(normalizedTokens) > 8 {
			totalRecordsTruncatedToEightWords++
			totalWordsTruncated += len(normalizedTokens) - 8
			normalizedTokens = normalizedTokens[:8]
		}
		cleanedProductNames = append(cleanedProductNames, strings.Join(normalizedTokens, " "))
	}
	if len(cleanedProductNames) == 0 {
		return fmt.Errorf("all product names were empty after normalization and connector filtering")
	}
	remainingConnectorTokensAfterFilter := countConnectorTokensInNames(cleanedProductNames, connectorSet)
	log.Printf(
		"word_parser_v2: connector_and_word_cap_stats connectors_removed=%d remaining_connectors_after_filter=%d truncated_records_to_8_words=%d truncated_words_total=%d",
		totalConnectorTokensRemoved,
		remainingConnectorTokensAfterFilter,
		totalRecordsTruncatedToEightWords,
		totalWordsTruncated,
	)

	fixedSlots, reservedSyllables, fixedAliasToSlot, fixedGenerationError := GenerateFixedSyllableSlotsDetailed(fixedConfig)
	if fixedGenerationError != nil {
		return fixedGenerationError
	}
	extractedFrequencyBySyllable := extractSyllableFrequency(cleanedProductNames)
	effectiveFixedSlots, effectiveReservedSyllables, effectiveAliasToSlot, droppedFixedSlots := buildEffectiveFixedSet(
		fixedSlots,
		reservedSyllables,
		fixedAliasToSlot,
		extractedFrequencyBySyllable,
	)
	log.Printf(
		"word_parser_v2: effective_fixed_filter original=%d effective=%d dropped=%d",
		len(fixedSlots),
		len(effectiveFixedSlots),
		droppedFixedSlots,
	)
	if validateError := validateAndLogFixedSyllables(effectiveFixedSlots); validateError != nil {
		return validateError
	}
	generatedDictionary, frequentGenerationError := GenerateFrequentSyllableSlotsWithReserved(cleanedProductNames, effectiveFixedSlots, effectiveReservedSyllables, frequentConfig)
	if frequentGenerationError != nil {
		return frequentGenerationError
	}
	if len(generatedDictionary.CombinedSlots) > 255 {
		return fmt.Errorf("combined dictionary too large for uint8 ids: %d", len(generatedDictionary.CombinedSlots))
	}
	logSyllableSummaryLine("Fixed syllables", effectiveFixedSlots)
	logSyllableSummaryLine("Computed syllables", generatedDictionary.FrequentSlots)
	totalExtractedSyllables, uniqueExtractedSyllables := extractSyllableStats(cleanedProductNames)

	// This map is the encoding source of truth before dictionary compaction/remapping.
	syllableToID := make(map[string]uint8, len(generatedDictionary.CombinedSlots)*2)
	for alias, slotID := range effectiveAliasToSlot {
		if slotID == 0 || slotID > 255 {
			continue
		}
		syllableToID[alias] = uint8(slotID)
	}

	normalizedDictionarySlots := make([]string, len(generatedDictionary.CombinedSlots))
	for slotIndex, syllable := range generatedDictionary.CombinedSlots {
		normalizedSyllable := normalizeToken(syllable)
		if normalizedSyllable == "" {
			return fmt.Errorf("empty normalized syllable at dictionary slot=%d", slotIndex+1)
		}
		normalizedDictionarySlots[slotIndex] = normalizedSyllable
		syllableID := uint8(slotIndex + 1)
		if _, exists := syllableToID[normalizedSyllable]; !exists {
			syllableToID[normalizedSyllable] = syllableID
		}
	}
	if len(normalizedDictionarySlots) > 0 {
		// Numeric compaction tokens are aliases that always resolve inside current dictionary bounds.
		maxDictionarySlotID := uint8(len(normalizedDictionarySlots))
		for compactBucket := 1; compactBucket <= 244; compactBucket++ {
			remappedDictionarySlot := uint8(((compactBucket - 1) % int(maxDictionarySlotID)) + 1)
			syllableToID[buildCompactNumericToken(uint8(compactBucket))] = remappedDictionarySlot
		}
	}

	encodedRecords := make([]EncodedNameRecord, 0, len(cleanedProductNames))
	unknownSyllableCounts := make(map[string]int)
	for productIndex, cleanedProductName := range cleanedProductNames {
		encodedRecord, encodeError := encodeProductName(cleanedProductName, syllableToID)
		if encodeError != nil {
			return fmt.Errorf("encode product at index %d: %w", productIndex, encodeError)
		}
		for _, unknownSyllable := range encodedRecordUnknownSyllables(cleanedProductName, syllableToID) {
			unknownSyllableCounts[unknownSyllable]++
		}
		if len(encodedRecord.WordSizes) == 0 || len(encodedRecord.EncodedContent) == 0 {
			continue
		}
		encodedRecords = append(encodedRecords, encodedRecord)
	}
	if len(encodedRecords) == 0 {
		return fmt.Errorf("no records with encodable syllables after normalization")
	}

	optimizedDictionarySlots, optimizationStats := applyTopWordFamilySlotExpansion(
		cleanedProductNames,
		normalizedDictionarySlots,
		syllableToID,
	)
	normalizedDictionarySlots = optimizedDictionarySlots
	log.Printf(
		"word_parser_v2: top_word_family_fill scanned_families=%d selected_families=%d added_canonical_words=%d added_alias_mappings=%d dictionary_slots_after=%d",
		optimizationStats.ScannedFamilyCount,
		optimizationStats.SelectedFamilyCount,
		optimizationStats.AddedCanonicalWordCount,
		optimizationStats.AddedAliasMappingCount,
		len(normalizedDictionarySlots),
	)

	logDictionaryUsageCoverageStats(generatedDictionary, encodedRecords)
	logShapeDiversityStats(encodedRecords)
	logShapeCoverageStats(encodedRecords, shapeCountU8Limit)

	binaryOutput, buildPayloadError := buildBinaryIndexPayload(normalizedDictionarySlots, encodedRecords)
	if buildPayloadError != nil {
		return buildPayloadError
	}
	if writeBinaryError := os.WriteFile(outputIndexPath, binaryOutput, 0o644); writeBinaryError != nil {
		return fmt.Errorf("write binary index file: %w", writeBinaryError)
	}
	outputFileInfo, outputStatError := os.Stat(outputIndexPath)
	if outputStatError != nil {
		return fmt.Errorf("stat output file: %w", outputStatError)
	}

	log.Printf(
		"word_parser_v2: stats fixed_syllables=%d extracted_syllables_total=%d extracted_syllables_unique=%d input_kb=%.2f output_kb=%.2f",
		len(effectiveFixedSlots),
		totalExtractedSyllables,
		uniqueExtractedSyllables,
		float64(inputFileInfo.Size())/1024.0,
		float64(outputFileInfo.Size())/1024.0,
	)
	log.Printf(
		"word_parser_v2: wrote productos.idx input=%d cleaned=%d dictionary_slots=%d records=%d bytes=%d",
		len(productNames),
		len(cleanedProductNames),
		len(normalizedDictionarySlots),
		len(encodedRecords),
		len(binaryOutput),
	)
	logTopUnknownSyllables(unknownSyllableCounts, 20)
	if len(unknownSyllableCounts) > 0 {
		return fmt.Errorf("unknown syllables remain after single-letter fallback unique=%d", len(unknownSyllableCounts))
	}
	return nil
}

type topWordFamilyFillStats struct {
	ScannedFamilyCount       int
	SelectedFamilyCount      int
	AddedCanonicalWordCount  int
	AddedAliasMappingCount   int
}

func applyTopWordFamilySlotExpansion(
	cleanedProductNames []string,
	inputDictionarySlots []string,
	syllableToID map[string]uint8,
) ([]string, topWordFamilyFillStats) {
	const (
		targetDictionarySlots      = 254
		topWordFamilyCandidatesCap = 80
	)
	fillStats := topWordFamilyFillStats{}
	if len(inputDictionarySlots) >= targetDictionarySlots {
		return inputDictionarySlots, fillStats
	}

	// Build normalized word frequencies from cleaned corpus.
	wordFrequencyByToken := make(map[string]int, 2048)
	for _, cleanedProductName := range cleanedProductNames {
		for _, wordToken := range strings.Fields(cleanedProductName) {
			normalizedWordToken := normalizeToken(wordToken)
			if normalizedWordToken == "" {
				continue
			}
			if !isAlphabeticToken(normalizedWordToken) || utf8.RuneCountInString(normalizedWordToken) < 3 {
				continue
			}
			wordFrequencyByToken[normalizedWordToken]++
		}
	}
	if len(wordFrequencyByToken) == 0 {
		return inputDictionarySlots, fillStats
	}

	// Group families with singular/plural normalization by suffix `s`/`es`.
	type familyAccumulator struct {
		TotalFrequency int
		WordFrequency  map[string]int
	}
	familyByKey := make(map[string]*familyAccumulator, len(wordFrequencyByToken))
	for currentWord, currentCount := range wordFrequencyByToken {
		familyKey := resolveWordFamilyKey(currentWord, wordFrequencyByToken)
		familyEntry := familyByKey[familyKey]
		if familyEntry == nil {
			familyEntry = &familyAccumulator{WordFrequency: make(map[string]int, 4)}
			familyByKey[familyKey] = familyEntry
		}
		familyEntry.TotalFrequency += currentCount
		familyEntry.WordFrequency[currentWord] += currentCount
	}

	familyCandidates := make([]topWordFamilyCandidate, 0, len(familyByKey))
	for familyKey, familyEntry := range familyByKey {
		canonicalWord := ""
		canonicalFrequency := -1
		aliasWords := make([]string, 0, len(familyEntry.WordFrequency))
		for aliasWord, aliasFrequency := range familyEntry.WordFrequency {
			aliasWords = append(aliasWords, aliasWord)
			if aliasFrequency > canonicalFrequency {
				canonicalWord = aliasWord
				canonicalFrequency = aliasFrequency
				continue
			}
			if aliasFrequency == canonicalFrequency {
				if utf8.RuneCountInString(aliasWord) < utf8.RuneCountInString(canonicalWord) {
					canonicalWord = aliasWord
				} else if utf8.RuneCountInString(aliasWord) == utf8.RuneCountInString(canonicalWord) && aliasWord < canonicalWord {
					canonicalWord = aliasWord
				}
			}
		}
		sort.Strings(aliasWords)
		familyCandidates = append(familyCandidates, topWordFamilyCandidate{
			FamilyKey:      familyKey,
			CanonicalWord:  canonicalWord,
			AliasWords:     aliasWords,
			TotalFrequency: familyEntry.TotalFrequency,
		})
	}
	fillStats.ScannedFamilyCount = len(familyCandidates)

	sort.SliceStable(familyCandidates, func(leftIndex, rightIndex int) bool {
		leftCandidate := familyCandidates[leftIndex]
		rightCandidate := familyCandidates[rightIndex]
		if leftCandidate.TotalFrequency != rightCandidate.TotalFrequency {
			return leftCandidate.TotalFrequency > rightCandidate.TotalFrequency
		}
		if leftCandidate.CanonicalWord != rightCandidate.CanonicalWord {
			return leftCandidate.CanonicalWord < rightCandidate.CanonicalWord
		}
		return leftCandidate.FamilyKey < rightCandidate.FamilyKey
	})

	selectedFamilyCandidates := familyCandidates
	if len(selectedFamilyCandidates) > topWordFamilyCandidatesCap {
		selectedFamilyCandidates = selectedFamilyCandidates[:topWordFamilyCandidatesCap]
	}
	fillStats.SelectedFamilyCount = len(selectedFamilyCandidates)

	workingDictionarySlots := append([]string(nil), inputDictionarySlots...)
	for _, familyCandidate := range selectedFamilyCandidates {
		canonicalWord := familyCandidate.CanonicalWord
		canonicalDictionaryID, canonicalAlreadyExists := syllableToID[canonicalWord]
		if !canonicalAlreadyExists {
			if len(workingDictionarySlots) >= targetDictionarySlots {
				break
			}
			workingDictionarySlots = append(workingDictionarySlots, canonicalWord)
			canonicalDictionaryID = uint8(len(workingDictionarySlots))
			syllableToID[canonicalWord] = canonicalDictionaryID
			fillStats.AddedCanonicalWordCount++
		}
		for _, aliasWord := range familyCandidate.AliasWords {
			if _, aliasAlreadyExists := syllableToID[aliasWord]; aliasAlreadyExists {
				continue
			}
			syllableToID[aliasWord] = canonicalDictionaryID
			fillStats.AddedAliasMappingCount++
		}
	}
	return workingDictionarySlots, fillStats
}

func resolveWordFamilyKey(wordToken string, wordFrequencyByToken map[string]int) string {
	if strings.HasSuffix(wordToken, "es") && utf8.RuneCountInString(wordToken) > 3 {
		singularWord := strings.TrimSuffix(wordToken, "es")
		if singularWord != "" {
			if _, exists := wordFrequencyByToken[singularWord]; exists {
				return singularWord
			}
		}
	}
	if strings.HasSuffix(wordToken, "s") && utf8.RuneCountInString(wordToken) > 2 {
		singularWord := strings.TrimSuffix(wordToken, "s")
		if singularWord != "" {
			if _, exists := wordFrequencyByToken[singularWord]; exists {
				return singularWord
			}
		}
	}
	return wordToken
}

func isAlphabeticToken(token string) bool {
	for _, currentRune := range token {
		if currentRune < 'a' || currentRune > 'z' {
			return false
		}
	}
	return true
}

func encodeProductName(cleanedProductName string, syllableToID map[string]uint8) (EncodedNameRecord, error) {
	wordTokens := strings.Fields(cleanedProductName)
	encodedContent := make([]uint8, 0, 32)
	wordSizes := make([]uint8, 0, len(wordTokens))

	for _, wordToken := range wordTokens {
		encodedWord := encodeWordTokenWithFallback(wordToken, syllableToID)
		if len(encodedWord) == 0 {
			continue
		}
		// Strategy clamp: keep at most 7 syllables per word to fit 3-bit shape slots.
		if len(encodedWord) > 7 {
			encodedWord = encodedWord[:7]
		}
		wordSizes = append(wordSizes, uint8(len(encodedWord)))
		encodedContent = append(encodedContent, encodedWord...)
	}

	return EncodedNameRecord{
		WordSizes:      wordSizes,
		EncodedContent: encodedContent,
	}, nil
}

func encodedRecordUnknownSyllables(cleanedProductName string, syllableToID map[string]uint8) []string {
	unknownSyllables := make([]string, 0, 8)
	for _, wordToken := range strings.Fields(cleanedProductName) {
		normalizedWordToken := normalizeToken(wordToken)
		if normalizedWordToken == "" {
			continue
		}
		if compactNumericID, compactSuffix, hasCompactNumericPrefix := splitCompactNumericTokenAndSuffix(normalizedWordToken); hasCompactNumericPrefix {
			if _, exists := syllableToID[buildCompactNumericToken(compactNumericID)]; !exists {
				unknownSyllables = append(unknownSyllables, buildCompactNumericToken(compactNumericID))
			}
			if compactSuffix == "" {
				continue
			}
			normalizedWordToken = compactSuffix
		}
		if _, exists := syllableToID[normalizedWordToken]; exists {
			continue
		}
		if compactNumericID, isCompactNumericToken := decodeCompactNumericToken(wordToken); isCompactNumericToken {
			if _, exists := syllableToID[buildCompactNumericToken(compactNumericID)]; !exists {
				unknownSyllables = append(unknownSyllables, buildCompactNumericToken(compactNumericID))
			}
			continue
		}
		extractedSyllables := splitWordIntoSyllables(wordToken)
		if len(extractedSyllables) == 0 {
			normalizedWordToken := normalizeToken(wordToken)
			if normalizedWordToken != "" {
				extractedSyllables = []string{normalizedWordToken}
			}
		}
		for _, extractedSyllable := range extractedSyllables {
			if _, exists := syllableToID[extractedSyllable]; exists {
				continue
			}
			missingAfterSingleLetterFallback := false
			for _, singleRune := range extractedSyllable {
				if _, exists := syllableToID[string(singleRune)]; exists {
					continue
				}
				missingAfterSingleLetterFallback = true
				break
			}
			if !missingAfterSingleLetterFallback {
				continue
			}
			unknownSyllables = append(unknownSyllables, extractedSyllable)
		}
	}
	return unknownSyllables
}

func encodeWordTokenWithFallback(wordToken string, syllableToID map[string]uint8) []uint8 {
	normalizedWordToken := normalizeToken(wordToken)
	if normalizedWordToken == "" {
		return nil
	}
	// Prefer full-token alias mapping (e.g. "unidades" -> slot of "ud") before syllable splitting.
	if fullTokenID, exists := syllableToID[normalizedWordToken]; exists {
		return []uint8{fullTokenID}
	}
	if compactNumericID, isCompactNumericToken := decodeCompactNumericToken(normalizedWordToken); isCompactNumericToken {
		compactNumericKey := buildCompactNumericToken(compactNumericID)
		if mappedID, exists := syllableToID[compactNumericKey]; exists {
			return []uint8{mappedID}
		}
		return nil
	}
	if compactNumericID, compactSuffix, hasCompactNumericPrefix := splitCompactNumericTokenAndSuffix(normalizedWordToken); hasCompactNumericPrefix {
		encodedWord := make([]uint8, 0, 4)
		compactNumericKey := buildCompactNumericToken(compactNumericID)
		if mappedID, exists := syllableToID[compactNumericKey]; exists {
			encodedWord = append(encodedWord, mappedID)
		}
		if compactSuffix == "" {
			return encodedWord
		}
		if suffixID, exists := syllableToID[compactSuffix]; exists {
			encodedWord = append(encodedWord, suffixID)
			return encodedWord
		}
		for _, suffixSyllable := range splitWordIntoSyllables(compactSuffix) {
			if suffixID, exists := syllableToID[suffixSyllable]; exists {
				encodedWord = append(encodedWord, suffixID)
				continue
			}
			for _, singleRune := range suffixSyllable {
				singleToken := string(singleRune)
				if singleTokenID, singleExists := syllableToID[singleToken]; singleExists {
					encodedWord = append(encodedWord, singleTokenID)
				}
			}
		}
		return encodedWord
	}

	extractedSyllables := splitWordIntoSyllables(normalizedWordToken)
	if len(extractedSyllables) == 0 {
		extractedSyllables = []string{normalizedWordToken}
	}

	encodedWord := make([]uint8, 0, len(extractedSyllables))
	for _, syllable := range extractedSyllables {
		syllableID, exists := syllableToID[syllable]
		if exists {
			encodedWord = append(encodedWord, syllableID)
			continue
		}

		// Fallback: encode unknown chunk as single-character tokens.
		for _, singleRune := range syllable {
			singleToken := string(singleRune)
			singleTokenID, singleTokenExists := syllableToID[singleToken]
			if !singleTokenExists {
				continue
			}
			encodedWord = append(encodedWord, singleTokenID)
		}
	}
	return encodedWord
}

func extractSyllableStats(cleanedProductNames []string) (int, int) {
	totalExtracted := 0
	uniqueExtractedSet := make(map[string]struct{}, 512)
	for _, cleanedProductName := range cleanedProductNames {
		for _, wordToken := range strings.Fields(cleanedProductName) {
			for _, extractedSyllable := range splitWordIntoSyllables(wordToken) {
				if extractedSyllable == "" {
					continue
				}
				totalExtracted++
				uniqueExtractedSet[extractedSyllable] = struct{}{}
			}
		}
	}
	return totalExtracted, len(uniqueExtractedSet)
}

func extractSyllableFrequency(cleanedProductNames []string) map[string]int {
	frequencyBySyllable := make(map[string]int, 1024)
	for _, cleanedProductName := range cleanedProductNames {
		for _, wordToken := range strings.Fields(cleanedProductName) {
			for _, extractedSyllable := range splitWordIntoSyllables(wordToken) {
				if extractedSyllable == "" {
					continue
				}
				frequencyBySyllable[extractedSyllable]++
			}
		}
	}
	return frequencyBySyllable
}

func buildEffectiveFixedSet(
	originalFixedSlots []string,
	originalReservedSyllables []string,
	originalAliasToSlot map[string]uint16,
	extractedFrequencyBySyllable map[string]int,
) ([]string, []string, map[string]uint16, int) {
	keptFixedSlots := make([]string, 0, len(originalFixedSlots))
	oldSlotToNewSlot := make(map[uint16]uint16, len(originalFixedSlots))
	for slotIndex, fixedSyllable := range originalFixedSlots {
		normalizedSyllable := normalizeToken(fixedSyllable)
		if normalizedSyllable == "" {
			continue
		}
		// Keep every fixed slot to preserve preassigned aliases regardless of corpus frequency.
		keptFixedSlots = append(keptFixedSlots, normalizedSyllable)
		oldSlotToNewSlot[uint16(slotIndex+1)] = uint16(len(keptFixedSlots))
	}

	keptAliasToSlot := make(map[string]uint16, len(originalAliasToSlot))
	for alias, oldSlotID := range originalAliasToSlot {
		newSlotID, exists := oldSlotToNewSlot[oldSlotID]
		if !exists {
			continue
		}
		normalizedAlias := normalizeToken(alias)
		if normalizedAlias == "" {
			continue
		}
		keptAliasToSlot[normalizedAlias] = newSlotID
	}

	keptReservedSet := make(map[string]struct{}, len(keptAliasToSlot))
	for _, reservedSyllable := range originalReservedSyllables {
		normalizedReserved := normalizeToken(reservedSyllable)
		if normalizedReserved == "" {
			continue
		}
		if _, exists := keptAliasToSlot[normalizedReserved]; !exists {
			continue
		}
		keptReservedSet[normalizedReserved] = struct{}{}
	}
	keptReservedSyllables := make([]string, 0, len(keptReservedSet))
	for reservedSyllable := range keptReservedSet {
		keptReservedSyllables = append(keptReservedSyllables, reservedSyllable)
	}
	sort.Strings(keptReservedSyllables)

	droppedFixedSlots := len(originalFixedSlots) - len(keptFixedSlots)
	return keptFixedSlots, keptReservedSyllables, keptAliasToSlot, droppedFixedSlots
}

func isNumericToken(token string) bool {
	if token == "" {
		return false
	}
	for _, currentRune := range token {
		if currentRune < '0' || currentRune > '9' {
			return false
		}
	}
	return true
}

func isSingleCharacterToken(token string) bool {
	return utf8.RuneCountInString(token) == 1
}

func validateAndLogFixedSyllables(fixedSlots []string) error {
	seenSyllables := make(map[string]int, len(fixedSlots))
	for slotIndex, syllable := range fixedSlots {
		normalizedSyllable := normalizeToken(syllable)
		if normalizedSyllable == "" {
			continue
		}
		if previousIndex, alreadySeen := seenSyllables[normalizedSyllable]; alreadySeen {
			return fmt.Errorf(
				"duplicated fixed syllable=%s current_slot=%d previous_slot=%d",
				normalizedSyllable,
				slotIndex+1,
				previousIndex+1,
			)
		}
		seenSyllables[normalizedSyllable] = slotIndex
	}
	return nil
}

func buildBinaryIndexPayload(dictionarySlots []string, encodedRecords []EncodedNameRecord) ([]uint8, error) {
	compactedDictionaryTokens, remappedRecords, compactError := compactDictionaryAndRemapRecords(dictionarySlots, encodedRecords)
	if compactError != nil {
		return nil, compactError
	}
	if len(compactedDictionaryTokens) == 0 {
		return nil, fmt.Errorf("dictionary is empty after compaction")
	}
	if len(compactedDictionaryTokens) > 255 {
		return nil, fmt.Errorf("dictionary too large after compaction count=%d", len(compactedDictionaryTokens))
	}

	dictionarySectionRaw, dictionaryRawError := buildRawDictionarySection(compactedDictionaryTokens)
	if dictionaryRawError != nil {
		return nil, dictionaryRawError
	}
	dictionarySectionDelta, remapFromRawToSorted, dictionaryDeltaError := buildDeltaDictionarySection(compactedDictionaryTokens)
	if dictionaryDeltaError != nil {
		return nil, dictionaryDeltaError
	}

	dictionarySection := dictionarySectionRaw
	activeDictionaryTokens := append([]string(nil), compactedDictionaryTokens...)
	headerFlags := uint8(0)
	if len(dictionarySectionDelta) < len(dictionarySectionRaw) {
		// Delta dictionary requires remapping content IDs into sorted dictionary order.
		for recordIndex := range remappedRecords {
			for syllableIndex, oldID := range remappedRecords[recordIndex].EncodedContent {
				newID := remapFromRawToSorted[oldID]
				if newID == 0 {
					return nil, fmt.Errorf("missing delta remap for record=%d syllable_index=%d old_id=%d", recordIndex, syllableIndex, oldID)
				}
				remappedRecords[recordIndex].EncodedContent[syllableIndex] = newID
			}
		}
		dictionarySection = dictionarySectionDelta
		sortedDictionaryTokens, buildSortedTokensError := rebuildSortedDictionaryTokensByID(compactedDictionaryTokens, remapFromRawToSorted)
		if buildSortedTokensError != nil {
			return nil, buildSortedTokensError
		}
		activeDictionaryTokens = sortedDictionaryTokens
		headerFlags |= headerFlagDictionaryDelta
	}

	sortedRecords, sortError := buildRecordsSortedByShapeValue(remappedRecords)
	if sortError != nil {
		return nil, sortError
	}
	if writeDebugError := writeSortedShapeDebugRows("libs/word_parser_v2/shape_debug_rows.log", sortedRecords, activeDictionaryTokens); writeDebugError != nil {
		return nil, writeDebugError
	}
	shapeUsageSection, shapeDeltaStreamUsesEscape, shapeDeltaStats, shapeStreamError := buildTablelessShapeDeltaUsageSection(sortedRecords)
	if shapeStreamError != nil {
		return nil, shapeStreamError
	}

	class0CompactCount, class0OverflowCount, class0CountError := splitShapeCountCompactAndOverflow(0)
	if class0CountError != nil {
		return nil, class0CountError
	}
	class1CompactCount, class1OverflowCount, class1CountError := splitShapeCountCompactAndOverflow(0)
	if class1CountError != nil {
		return nil, class1CountError
	}

	class0ShapeTable := []uint8(nil)
	class1ShapeTable := []uint8(nil)
	class0UsageSection := []uint8(nil)
	class1UsageSection := shapeUsageSection
	class0UseDelta := false
	// Shape delta stream is always active in unified inline mode.
	class1UseDelta := true
	headerFlags |= headerFlagShapeInlineC1
	headerFlags |= headerFlagShapeCompactV2
	if class1UseDelta {
		headerFlags |= headerFlagShapeDeltaC1
	}

	contentSection := buildContentSectionFromSortedShapeRecords(sortedRecords)
	totalStoredShapeBytes := len(class0ShapeTable) + len(class1ShapeTable) + len(class0UsageSection) + len(class1UsageSection)

	binaryPayload := make([]uint8, 0, headerSizeV3+len(dictionarySection)+len(class0ShapeTable)+len(class1ShapeTable)+len(class0UsageSection)+len(class1UsageSection)+len(contentSection))
	binaryPayload = append(binaryPayload, []byte(binaryIndexMagicV3)...)
	binaryPayload = append(binaryPayload, binaryIndexVersion)
	binaryPayload = append(binaryPayload, headerFlags)

	var recordCountBuffer [4]byte
	binary.LittleEndian.PutUint32(recordCountBuffer[:], uint32(len(remappedRecords)))
	binaryPayload = append(binaryPayload, recordCountBuffer[:]...)

	binaryPayload = append(binaryPayload, uint8(len(compactedDictionaryTokens)))
	binaryPayload = append(binaryPayload, class0CompactCount)
	binaryPayload = append(binaryPayload, class1CompactCount)

	var class0OverflowBuffer [2]byte
	binary.LittleEndian.PutUint16(class0OverflowBuffer[:], class0OverflowCount)
	binaryPayload = append(binaryPayload, class0OverflowBuffer[:]...)

	var class1OverflowBuffer [2]byte
	binary.LittleEndian.PutUint16(class1OverflowBuffer[:], class1OverflowCount)
	binaryPayload = append(binaryPayload, class1OverflowBuffer[:]...)

	var dictionarySectionSizeBuffer [4]byte
	binary.LittleEndian.PutUint32(dictionarySectionSizeBuffer[:], uint32(len(dictionarySection)))
	binaryPayload = append(binaryPayload, dictionarySectionSizeBuffer[:]...)

	var class0ShapeTableSizeBuffer [4]byte
	binary.LittleEndian.PutUint32(class0ShapeTableSizeBuffer[:], uint32(len(class0ShapeTable)))
	binaryPayload = append(binaryPayload, class0ShapeTableSizeBuffer[:]...)

	var class1ShapeTableSizeBuffer [4]byte
	binary.LittleEndian.PutUint32(class1ShapeTableSizeBuffer[:], uint32(len(class1ShapeTable)))
	binaryPayload = append(binaryPayload, class1ShapeTableSizeBuffer[:]...)

	binaryPayload = append(binaryPayload, dictionarySection...)
	binaryPayload = append(binaryPayload, class0ShapeTable...)
	binaryPayload = append(binaryPayload, class1ShapeTable...)
	binaryPayload = append(binaryPayload, class0UsageSection...)
	binaryPayload = append(binaryPayload, class1UsageSection...)
	binaryPayload = append(binaryPayload, contentSection...)

	log.Printf(
		"word_parser_v2: v3_index header_size=%d index_size=%d content_size=%d header_flags=%d dictionary_count=%d dictionary_bytes=%d shape_class0=%d shape_class1=%d shape_class0_overflow=%d shape_class1_overflow=%d shape_table_mode_c0=%t shape_table_mode_c1=%t shape_inline_c0=%t shape_inline_c1=%t unified_shape_stream=%t",
		headerSizeV3,
		len(binaryPayload),
		len(contentSection),
		headerFlags,
		len(compactedDictionaryTokens),
		len(dictionarySection),
		0,
		0,
		class0OverflowCount,
		class1OverflowCount,
		class0UseDelta,
		class1UseDelta,
		headerFlags&headerFlagShapeInlineC0 != 0,
		headerFlags&headerFlagShapeInlineC1 != 0,
		shapeDeltaStreamUsesEscape,
	)
	log.Printf(
		"word_parser_v2: shape_storage_stats normal_delta_u16_bytes=%d escaped_delta_bytes=%d escaped_delta_count=%d first_shape_value_bytes=%d shape_delta_stream_bytes=%d total_stored_shapes_bytes=%d actual_content_bytes=%d",
		shapeDeltaStats.NormalDeltaByteCount,
		shapeDeltaStats.EscapedDeltaByteCount,
		shapeDeltaStats.EscapedDeltaCount,
		shapeDeltaStats.FirstShapeValueByteCount,
		shapeDeltaStats.TotalDeltaStreamBytes,
		totalStoredShapeBytes,
		len(contentSection),
	)

	return binaryPayload, nil
}

func buildRecordsSortedByShapeValue(records []EncodedNameRecord) ([]sortedRecordByShapeValue, error) {
	sortedRecords := make([]sortedRecordByShapeValue, 0, len(records))
	for recordIndex, encodedRecord := range records {
		shapeValue, shapeValueError := encodeShapeWordSizesToFixed24BitValue(encodedRecord.WordSizes)
		if shapeValueError != nil {
			return nil, fmt.Errorf("encode shape value record=%d: %w", recordIndex, shapeValueError)
		}
		totalSyllableCount := 0
		for _, wordSize := range encodedRecord.WordSizes {
			totalSyllableCount += int(wordSize)
		}
		if totalSyllableCount != len(encodedRecord.EncodedContent) {
			return nil, fmt.Errorf("record content length mismatch record=%d expected=%d got=%d", recordIndex, totalSyllableCount, len(encodedRecord.EncodedContent))
		}
		sortedRecords = append(sortedRecords, sortedRecordByShapeValue{
			ShapeValue:     shapeValue,
			WordSizes:      append([]uint8(nil), encodedRecord.WordSizes...),
			EncodedContent: append([]uint8(nil), encodedRecord.EncodedContent...),
		})
	}

	sort.SliceStable(sortedRecords, func(leftIndex, rightIndex int) bool {
		if sortedRecords[leftIndex].ShapeValue != sortedRecords[rightIndex].ShapeValue {
			return sortedRecords[leftIndex].ShapeValue < sortedRecords[rightIndex].ShapeValue
		}
		return compareWordSizeSlices(sortedRecords[leftIndex].WordSizes, sortedRecords[rightIndex].WordSizes) < 0
	})
	return sortedRecords, nil
}

func rebuildSortedDictionaryTokensByID(rawDictionaryTokens []string, remapFromRawToSorted []uint8) ([]string, error) {
	sortedDictionaryTokens := make([]string, len(rawDictionaryTokens))
	for rawIndex, rawToken := range rawDictionaryTokens {
		rawID := uint8(rawIndex + 1)
		if int(rawID) >= len(remapFromRawToSorted) {
			return nil, fmt.Errorf("dictionary remap out of bounds raw_id=%d", rawID)
		}
		sortedID := remapFromRawToSorted[rawID]
		if sortedID == 0 || int(sortedID) > len(sortedDictionaryTokens) {
			return nil, fmt.Errorf("dictionary remap invalid raw_id=%d sorted_id=%d", rawID, sortedID)
		}
		sortedDictionaryTokens[int(sortedID)-1] = rawToken
	}
	for sortedIndex, sortedToken := range sortedDictionaryTokens {
		if sortedToken == "" {
			return nil, fmt.Errorf("dictionary remap left empty sorted slot=%d", sortedIndex+1)
		}
	}
	return sortedDictionaryTokens, nil
}

func writeSortedShapeDebugRows(outputPath string, sortedRecords []sortedRecordByShapeValue, dictionaryTokens []string) error {
	outputDirectoryPath := filepath.Dir(outputPath)
	if outputDirectoryPath != "." && outputDirectoryPath != "" {
		if mkdirError := os.MkdirAll(outputDirectoryPath, 0o755); mkdirError != nil {
			return fmt.Errorf("create shape debug directory: %w", mkdirError)
		}
	}
	debugLines := make([]string, 0, len(sortedRecords)+1)
	debugLines = append(debugLines, "record_index\tshape_value\tdelta_value\tshape\tshape_2bit_bytes\tphrase_plain")
	previousShapeValue := uint32(0)
	for recordIndex, sortedRecord := range sortedRecords {
		currentShapeValue := sortedRecord.ShapeValue
		deltaValue := uint32(0)
		if currentShapeValue >= previousShapeValue {
			deltaValue = currentShapeValue - previousShapeValue
		}
		shapeTwoBitBytesText := buildShapeTwoBitBytesDebugText(sortedRecord.WordSizes)
		decodedPhrase, decodePhraseError := decodePlainPhraseFromRecord(sortedRecord, dictionaryTokens)
		if decodePhraseError != nil {
			return fmt.Errorf("decode debug phrase record=%d: %w", recordIndex, decodePhraseError)
		}
		debugLines = append(
			debugLines,
			fmt.Sprintf(
				"%d\t%s\t%s\t%v\t%s\t%s",
				recordIndex,
				fmt.Sprintf("%d", currentShapeValue),
				fmt.Sprintf("%d", deltaValue),
				sortedRecord.WordSizes,
				shapeTwoBitBytesText,
				decodedPhrase,
			),
		)
		previousShapeValue = currentShapeValue
	}
	if writeError := os.WriteFile(outputPath, []byte(strings.Join(debugLines, "\n")+"\n"), 0o644); writeError != nil {
		return fmt.Errorf("write shape debug rows: %w", writeError)
	}
	log.Printf("word_parser_v2: wrote shape debug rows file=%s rows=%d", outputPath, len(sortedRecords))
	return nil
}

func buildShapeTwoBitBytesDebugText(wordSizes []uint8) string {
	for _, wordSize := range wordSizes {
		if wordSize < 1 || wordSize > 4 {
			return "overflow_word_gt4"
		}
	}
	twoBitPacked := packTwoBitValues(convertWordSizesToTwoBitDigits(wordSizes))
	if len(twoBitPacked) == 0 {
		return "empty"
	}
	formattedBytes := make([]string, 0, len(twoBitPacked))
	for _, currentByte := range twoBitPacked {
		formattedBytes = append(formattedBytes, fmt.Sprintf("%02x", currentByte))
	}
	return strings.Join(formattedBytes, "")
}

func convertWordSizesToTwoBitDigits(wordSizes []uint8) []uint8 {
	twoBitDigits := make([]uint8, 0, len(wordSizes))
	for _, wordSize := range wordSizes {
		twoBitDigits = append(twoBitDigits, wordSize-1)
	}
	return twoBitDigits
}

func decodePlainPhraseFromRecord(sortedRecord sortedRecordByShapeValue, dictionaryTokens []string) (string, error) {
	wordTexts := make([]string, 0, len(sortedRecord.WordSizes))
	contentCursor := 0
	for _, wordSize := range sortedRecord.WordSizes {
		syllableCount := int(wordSize)
		if contentCursor+syllableCount > len(sortedRecord.EncodedContent) {
			return "", fmt.Errorf("content overflow cursor=%d need=%d total=%d", contentCursor, syllableCount, len(sortedRecord.EncodedContent))
		}
		var wordBuilder strings.Builder
		for syllableIndex := 0; syllableIndex < syllableCount; syllableIndex++ {
			dictionaryID := int(sortedRecord.EncodedContent[contentCursor])
			contentCursor++
			if dictionaryID <= 0 || dictionaryID > len(dictionaryTokens) {
				return "", fmt.Errorf("invalid dictionary id=%d", dictionaryID)
			}
			wordBuilder.WriteString(dictionaryTokens[dictionaryID-1])
		}
		wordTexts = append(wordTexts, wordBuilder.String())
	}
	if contentCursor != len(sortedRecord.EncodedContent) {
		return "", fmt.Errorf("unused encoded content bytes=%d", len(sortedRecord.EncodedContent)-contentCursor)
	}
	return strings.Join(wordTexts, " "), nil
}

func buildTablelessShapeDeltaUsageSection(sortedRecords []sortedRecordByShapeValue) ([]uint8, bool, shapeDeltaStreamStats, error) {
	shapeDeltaStreamBytes, usesEscapeEncoding, streamStats, streamError := buildShapeValueDeltaStream(sortedRecords)
	if streamError != nil {
		return nil, false, shapeDeltaStreamStats{}, streamError
	}

	shapeUsageSection := make([]uint8, 0, len(shapeDeltaStreamBytes)+binary.MaxVarintLen64)
	shapeUsageSection = appendUvarint(shapeUsageSection, uint64(len(shapeDeltaStreamBytes)))
	shapeUsageSection = append(shapeUsageSection, shapeDeltaStreamBytes...)
	return shapeUsageSection, usesEscapeEncoding, streamStats, nil
}

func buildShapeValueDeltaStream(sortedRecords []sortedRecordByShapeValue) ([]uint8, bool, shapeDeltaStreamStats, error) {
	if len(sortedRecords) == 0 {
		return nil, false, shapeDeltaStreamStats{}, fmt.Errorf("shape value stream requires at least one record")
	}

	streamStats := shapeDeltaStreamStats{}
	usesEscapeEncoding := false
	shapeBitWriter := newBitWriter()
	previousShapeValue := uint32(0)
	for recordIndex, sortedRecord := range sortedRecords {
		currentShapeValue := sortedRecord.ShapeValue
		if currentShapeValue < previousShapeValue {
			return nil, false, shapeDeltaStreamStats{}, fmt.Errorf("shape values must be monotonic previous=%d current=%d", previousShapeValue, currentShapeValue)
		}
		shapeDeltaValue := currentShapeValue - previousShapeValue
		tokenKind, writtenBits, appendError := appendShapeDeltaTokenBits(shapeBitWriter, shapeDeltaValue)
		if appendError != nil {
			return nil, false, shapeDeltaStreamStats{}, fmt.Errorf("encode shape delta record=%d: %w", recordIndex, appendError)
		}
		if recordIndex == 0 {
			streamStats.FirstShapeValueByteCount = (writtenBits + 7) / 8
		}
		switch tokenKind {
		case shapeDeltaTokenKindSmall8:
			streamStats.NormalDeltaByteCount += writtenBits
		case shapeDeltaTokenKindMedium16:
			streamStats.EscapedDeltaByteCount += writtenBits
		case shapeDeltaTokenKindEscape24:
			usesEscapeEncoding = true
			streamStats.EscapedDeltaCount++
			streamStats.EscapedDeltaByteCount += writtenBits
		}
		previousShapeValue = currentShapeValue
	}

	shapeDeltaStreamBytes := shapeBitWriter.Bytes()
	streamStats.TotalDeltaStreamBytes = len(shapeDeltaStreamBytes)
	logShapeStreamProgressByFivePercent(len(sortedRecords), shapeDeltaStreamBytes)
	return shapeDeltaStreamBytes, usesEscapeEncoding, streamStats, nil
}

type shapeDeltaTokenKind uint8

const (
	shapeDeltaTokenKindSmall8 shapeDeltaTokenKind = iota
	shapeDeltaTokenKindMedium16
	shapeDeltaTokenKindEscape24
)

func appendShapeDeltaTokenBits(shapeBitWriter *bitWriter, shapeDeltaValue uint32) (shapeDeltaTokenKind, int, error) {
	if shapeDeltaValue <= 255 {
		shapeBitWriter.WriteBit(0)
		shapeBitWriter.WriteBits(shapeDeltaValue, 8)
		return shapeDeltaTokenKindSmall8, 9, nil
	}
	if shapeDeltaValue <= 65534 {
		shapeBitWriter.WriteBit(1)
		shapeBitWriter.WriteBits(shapeDeltaValue, 16)
		return shapeDeltaTokenKindMedium16, 17, nil
	}
	if shapeDeltaValue <= 0xFFFFFF {
		shapeBitWriter.WriteBit(1)
		shapeBitWriter.WriteBits(0xFFFF, 16)
		shapeBitWriter.WriteBits(shapeDeltaValue, 24)
		return shapeDeltaTokenKindEscape24, 41, nil
	}
	return shapeDeltaTokenKindEscape24, 0, fmt.Errorf("shape delta exceeds 24-bit max delta=%d", shapeDeltaValue)
}

func encodeShapeWordSizesToFixed24BitValue(wordSizes []uint8) (uint32, error) {
	if len(wordSizes) == 0 {
		return 0, fmt.Errorf("shape has zero words")
	}
	if len(wordSizes) > 8 {
		return 0, fmt.Errorf("shape has too many words count=%d max=8", len(wordSizes))
	}

	shapeValue := uint32(0)
	for wordIndex := 0; wordIndex < 8; wordIndex++ {
		wordCode := uint8(0)
		if wordIndex < len(wordSizes) {
			wordSize := wordSizes[wordIndex]
			if wordSize < 1 || wordSize > 7 {
				return 0, fmt.Errorf("invalid word size at index=%d value=%d allowed=1..7", wordIndex, wordSize)
			}
			wordCode = wordSize
		}
		shapeValue = (shapeValue << 3) | uint32(wordCode)
	}
	return shapeValue, nil
}

func countConnectorTokensInNames(cleanedProductNames []string, connectorSet map[string]struct{}) int {
	connectorTokenCount := 0
	for _, cleanedName := range cleanedProductNames {
		for _, token := range strings.Fields(cleanedName) {
			if _, isConnector := connectorSet[token]; isConnector {
				connectorTokenCount++
			}
		}
	}
	return connectorTokenCount
}

func logShapeStreamProgressByFivePercent(recordCount int, shapeDeltaStreamBytes []uint8) {
	if recordCount == 0 {
		return
	}

	thresholdRecordByPercent := make(map[int]int, 21)
	for percentStep := 0; percentStep <= 100; percentStep += 5 {
		thresholdRecord := int(math.Ceil(float64(recordCount) * float64(percentStep) / 100.0))
		if thresholdRecord < 1 {
			thresholdRecord = 1
		}
		if thresholdRecord > recordCount {
			thresholdRecord = recordCount
		}
		thresholdRecordByPercent[percentStep] = thresholdRecord
	}

	percentProgressBytes := make(map[int]int, 21)
	shapeBitReader := newBitReader(shapeDeltaStreamBytes)
	for recordIndex := 0; recordIndex < recordCount; recordIndex++ {
		_, readError := readShapeDeltaTokenFromBits(shapeBitReader)
		if readError != nil {
			return
		}
		processedRecords := recordIndex + 1
		for percentStep := 0; percentStep <= 100; percentStep += 5 {
			if thresholdRecordByPercent[percentStep] == processedRecords {
				percentProgressBytes[percentStep] = shapeBitReader.BytesConsumed()
			}
		}
	}

	progressEntries := make([]string, 0, 21)
	for percentStep := 0; percentStep <= 100; percentStep += 5 {
		progressEntries = append(progressEntries, fmt.Sprintf("p%d=%d", percentStep, percentProgressBytes[percentStep]))
	}
	log.Printf("word_parser_v2: shape_stream_progress_5pct %s", strings.Join(progressEntries, ", "))
}

type bitWriter struct {
	bytes         []uint8
	bitOffsetInByte uint8
}

func newBitWriter() *bitWriter {
	return &bitWriter{bytes: make([]uint8, 0, 128)}
}

func (bitStreamWriter *bitWriter) WriteBit(bitValue uint8) {
	if bitStreamWriter.bitOffsetInByte == 0 {
		bitStreamWriter.bytes = append(bitStreamWriter.bytes, 0)
	}
	if bitValue&1 == 1 {
		lastByteIndex := len(bitStreamWriter.bytes) - 1
		bitPosition := 7 - bitStreamWriter.bitOffsetInByte
		bitStreamWriter.bytes[lastByteIndex] |= 1 << bitPosition
	}
	bitStreamWriter.bitOffsetInByte++
	if bitStreamWriter.bitOffsetInByte == 8 {
		bitStreamWriter.bitOffsetInByte = 0
	}
}

func (bitStreamWriter *bitWriter) WriteBits(bitsValue uint32, bitCount int) {
	for bitIndex := bitCount - 1; bitIndex >= 0; bitIndex-- {
		nextBit := uint8((bitsValue >> bitIndex) & 1)
		bitStreamWriter.WriteBit(nextBit)
	}
}

func (bitStreamWriter *bitWriter) Bytes() []uint8 {
	return append([]uint8(nil), bitStreamWriter.bytes...)
}

type bitReader struct {
	bytes         []uint8
	totalBitOffset int
}

func newBitReader(sourceBytes []uint8) *bitReader {
	return &bitReader{bytes: sourceBytes}
}

func (bitStreamReader *bitReader) ReadBit() (uint8, error) {
	totalBitCount := len(bitStreamReader.bytes) * 8
	if bitStreamReader.totalBitOffset >= totalBitCount {
		return 0, fmt.Errorf("bitstream exhausted")
	}
	byteIndex := bitStreamReader.totalBitOffset / 8
	bitIndexInByte := uint8(bitStreamReader.totalBitOffset % 8)
	bitPosition := 7 - bitIndexInByte
	nextBit := (bitStreamReader.bytes[byteIndex] >> bitPosition) & 1
	bitStreamReader.totalBitOffset++
	return nextBit, nil
}

func (bitStreamReader *bitReader) ReadBits(bitCount int) (uint32, error) {
	readValue := uint32(0)
	for bitIndex := 0; bitIndex < bitCount; bitIndex++ {
		nextBit, readError := bitStreamReader.ReadBit()
		if readError != nil {
			return 0, readError
		}
		readValue = (readValue << 1) | uint32(nextBit)
	}
	return readValue, nil
}

func (bitStreamReader *bitReader) BytesConsumed() int {
	if bitStreamReader.totalBitOffset == 0 {
		return 0
	}
	return (bitStreamReader.totalBitOffset + 7) / 8
}

func readShapeDeltaTokenFromBits(shapeBitReader *bitReader) (uint32, error) {
	flagBit, readFlagError := shapeBitReader.ReadBit()
	if readFlagError != nil {
		return 0, readFlagError
	}
	if flagBit == 0 {
		smallDeltaValue, readDeltaError := shapeBitReader.ReadBits(8)
		if readDeltaError != nil {
			return 0, readDeltaError
		}
		return smallDeltaValue, nil
	}

	mediumDeltaValue, readMediumDeltaError := shapeBitReader.ReadBits(16)
	if readMediumDeltaError != nil {
		return 0, readMediumDeltaError
	}
	if mediumDeltaValue != 0xFFFF {
		return mediumDeltaValue, nil
	}

	escapeDeltaValue, readEscapeDeltaError := shapeBitReader.ReadBits(24)
	if readEscapeDeltaError != nil {
		return 0, readEscapeDeltaError
	}
	return escapeDeltaValue, nil
}

func buildContentSectionFromSortedShapeRecords(sortedRecords []sortedRecordByShapeValue) []uint8 {
	contentSection := make([]uint8, 0, 1024)
	for _, currentRecord := range sortedRecords {
		contentSection = append(contentSection, currentRecord.EncodedContent...)
	}
	return contentSection
}

func compactDictionaryAndRemapRecords(dictionarySlots []string, encodedRecords []EncodedNameRecord) ([]string, []EncodedNameRecord, error) {
	tokenToCanonicalID := make(map[string]uint8, len(dictionarySlots))
	oldToCanonicalID := make([]uint8, len(dictionarySlots)+1)
	compactedTokens := make([]string, 0, len(dictionarySlots))

	for slotIndex, slotToken := range dictionarySlots {
		normalizedToken := normalizeToken(slotToken)
		if normalizedToken == "" {
			continue
		}
		if existingID, exists := tokenToCanonicalID[normalizedToken]; exists {
			oldToCanonicalID[slotIndex+1] = existingID
			continue
		}
		newID := uint8(len(compactedTokens) + 1)
		tokenToCanonicalID[normalizedToken] = newID
		oldToCanonicalID[slotIndex+1] = newID
		compactedTokens = append(compactedTokens, normalizedToken)
	}

	remappedRecords := make([]EncodedNameRecord, 0, len(encodedRecords))
	for recordIndex, encodedRecord := range encodedRecords {
		remappedContent := make([]uint8, 0, len(encodedRecord.EncodedContent))
		for syllableIndex, oldID := range encodedRecord.EncodedContent {
			if int(oldID) >= len(oldToCanonicalID) {
				return nil, nil, fmt.Errorf("record=%d syllable_index=%d id_out_of_range=%d", recordIndex, syllableIndex, oldID)
			}
			newID := oldToCanonicalID[oldID]
			if newID == 0 {
				return nil, nil, fmt.Errorf("record=%d syllable_index=%d references_empty_dictionary_slot=%d", recordIndex, syllableIndex, oldID)
			}
			remappedContent = append(remappedContent, newID)
		}
		copiedWordSizes := append([]uint8(nil), encodedRecord.WordSizes...)
		remappedRecords = append(remappedRecords, EncodedNameRecord{WordSizes: copiedWordSizes, EncodedContent: remappedContent})
	}

	return compactedTokens, remappedRecords, nil
}

func buildRawDictionarySection(dictionaryTokens []string) ([]uint8, error) {
	section := make([]uint8, 0, len(dictionaryTokens)*4)
	for tokenIndex, dictionaryToken := range dictionaryTokens {
		tokenBytes := []byte(dictionaryToken)
		if len(tokenBytes) == 0 || len(tokenBytes) > 255 {
			return nil, fmt.Errorf("raw dictionary token length out of range slot=%d len=%d", tokenIndex+1, len(tokenBytes))
		}
		section = append(section, uint8(len(tokenBytes)))
		section = append(section, tokenBytes...)
	}
	return section, nil
}

func buildDeltaDictionarySection(dictionaryTokens []string) ([]uint8, []uint8, error) {
	sortedEntries := make([]dictionaryTokenEntry, 0, len(dictionaryTokens))
	for tokenIndex, dictionaryToken := range dictionaryTokens {
		sortedEntries = append(sortedEntries, dictionaryTokenEntry{Token: dictionaryToken, ID: uint8(tokenIndex + 1)})
	}
	sort.Slice(sortedEntries, func(leftIndex, rightIndex int) bool {
		left := sortedEntries[leftIndex]
		right := sortedEntries[rightIndex]
		if left.Token == right.Token {
			return left.ID < right.ID
		}
		return left.Token < right.Token
	})

	section := make([]uint8, 0, len(dictionaryTokens)*3)
	oldToSortedID := make([]uint8, len(dictionaryTokens)+1)
	previousTokenBytes := []byte{}
	for sortedIndex, sortedEntry := range sortedEntries {
		currentTokenBytes := []byte(sortedEntry.Token)
		if len(currentTokenBytes) == 0 || len(currentTokenBytes) > 255 {
			return nil, nil, fmt.Errorf("delta dictionary token length out of range slot=%d len=%d", sortedEntry.ID, len(currentTokenBytes))
		}
		prefixLength := sharedPrefixLengthBytes(previousTokenBytes, currentTokenBytes)
		if prefixLength > 255 {
			prefixLength = 255
		}
		suffixLength := len(currentTokenBytes) - prefixLength
		if suffixLength <= 0 || suffixLength > 255 {
			return nil, nil, fmt.Errorf("delta dictionary suffix length out of range sorted_index=%d suffix_len=%d", sortedIndex, suffixLength)
		}
		section = append(section, uint8(prefixLength), uint8(suffixLength))
		section = append(section, currentTokenBytes[prefixLength:]...)
		oldToSortedID[sortedEntry.ID] = uint8(sortedIndex + 1)
		previousTokenBytes = currentTokenBytes
	}
	return section, oldToSortedID, nil
}

func groupRecordsBySortedShapes(records []EncodedNameRecord) ([]shapeBucket, []shapeBucket, error) {
	classBucketsMap := map[uint8]map[string]*shapeBucket{
		shapeClassTwoBit:  {},
		shapeClassFourBit: {},
	}

	for recordIndex, encodedRecord := range records {
		shapeClass, classifyError := classifyShapeWordSizes(encodedRecord.WordSizes)
		if classifyError != nil {
			return nil, nil, fmt.Errorf("record=%d classify shape: %w", recordIndex, classifyError)
		}

		shapeKey := shapeWordSizesKey(encodedRecord.WordSizes)
		bucket, exists := classBucketsMap[shapeClass][shapeKey]
		if !exists {
			copiedWordSizes := append([]uint8(nil), encodedRecord.WordSizes...)
			totalSyllables := 0
			for _, wordSize := range copiedWordSizes {
				totalSyllables += int(wordSize)
			}
			bucket = &shapeBucket{WordSizes: copiedWordSizes, TotalSyllableCount: totalSyllables}
			classBucketsMap[shapeClass][shapeKey] = bucket
		}

		if len(encodedRecord.EncodedContent) != bucket.TotalSyllableCount {
			return nil, nil, fmt.Errorf(
				"record content length does not match shape sum record=%d got=%d expected=%d",
				recordIndex,
				len(encodedRecord.EncodedContent),
				bucket.TotalSyllableCount,
			)
		}
		copiedContent := append([]uint8(nil), encodedRecord.EncodedContent...)
		bucket.RecordContents = append(bucket.RecordContents, copiedContent)
	}

	class0Buckets := sortedBucketsFromMap(classBucketsMap[shapeClassTwoBit])
	class1Buckets := sortedBucketsFromMap(classBucketsMap[shapeClassFourBit])
	return class0Buckets, class1Buckets, nil
}

func sortedBucketsFromMap(bucketMap map[string]*shapeBucket) []shapeBucket {
	keys := make([]string, 0, len(bucketMap))
	for shapeKey := range bucketMap {
		keys = append(keys, shapeKey)
	}
	if len(keys) == 0 {
		return nil
	}

	// Deterministic seed order: highest frequency first, then lexicographic shape as tie-break.
	sort.Slice(keys, func(leftIndex, rightIndex int) bool {
		leftBucket := bucketMap[keys[leftIndex]]
		rightBucket := bucketMap[keys[rightIndex]]
		leftFrequency := len(leftBucket.RecordContents)
		rightFrequency := len(rightBucket.RecordContents)
		if leftFrequency != rightFrequency {
			return leftFrequency > rightFrequency
		}
		return compareWordSizeSlices(leftBucket.WordSizes, rightBucket.WordSizes) < 0
	})

	orderedBuckets := make([]shapeBucket, 0, len(keys))
	unusedKeys := make(map[string]struct{}, len(keys))
	for _, shapeKey := range keys {
		unusedKeys[shapeKey] = struct{}{}
	}

	currentKey := keys[0]
	for len(unusedKeys) > 0 {
		_, currentStillUnused := unusedKeys[currentKey]
		if !currentStillUnused {
			currentKey = nextBestSeedKey(keys, bucketMap, unusedKeys)
			if currentKey == "" {
				break
			}
		}

		currentBucket := bucketMap[currentKey]
		orderedBuckets = append(orderedBuckets, *currentBucket)
		delete(unusedKeys, currentKey)

		nextKey := nextNearestNeighborKey(currentBucket, keys, bucketMap, unusedKeys)
		if nextKey == "" {
			currentKey = nextBestSeedKey(keys, bucketMap, unusedKeys)
		} else {
			currentKey = nextKey
		}
	}

	return orderedBuckets
}

func buildUnifiedDeltaOrderedBuckets(class0Buckets []shapeBucket, class1Buckets []shapeBucket) []shapeBucket {
	mergedBuckets := make([]shapeBucket, 0, len(class0Buckets)+len(class1Buckets))
	mergedBuckets = append(mergedBuckets, class0Buckets...)
	mergedBuckets = append(mergedBuckets, class1Buckets...)
	if len(mergedBuckets) <= 1 {
		return mergedBuckets
	}

	seedIndexes := make([]int, len(mergedBuckets))
	for bucketIndex := range mergedBuckets {
		seedIndexes[bucketIndex] = bucketIndex
	}
	sort.Slice(seedIndexes, func(leftIndex, rightIndex int) bool {
		leftBucket := mergedBuckets[seedIndexes[leftIndex]]
		rightBucket := mergedBuckets[seedIndexes[rightIndex]]
		leftFrequency := len(leftBucket.RecordContents)
		rightFrequency := len(rightBucket.RecordContents)
		if leftFrequency != rightFrequency {
			return leftFrequency > rightFrequency
		}
		return compareWordSizeSlices(leftBucket.WordSizes, rightBucket.WordSizes) < 0
	})

	unusedByIndex := make(map[int]struct{}, len(mergedBuckets))
	for bucketIndex := range mergedBuckets {
		unusedByIndex[bucketIndex] = struct{}{}
	}

	pickNextSeed := func() int {
		for _, seedIndex := range seedIndexes {
			if _, exists := unusedByIndex[seedIndex]; exists {
				return seedIndex
			}
		}
		return -1
	}

	orderedBuckets := make([]shapeBucket, 0, len(mergedBuckets))
	currentIndex := pickNextSeed()
	for len(unusedByIndex) > 0 {
		if _, exists := unusedByIndex[currentIndex]; !exists {
			currentIndex = pickNextSeed()
			if currentIndex < 0 {
				break
			}
		}

		currentBucket := mergedBuckets[currentIndex]
		orderedBuckets = append(orderedBuckets, currentBucket)
		delete(unusedByIndex, currentIndex)
		if len(unusedByIndex) == 0 {
			break
		}

		nextIndex := -1
		bestTransitionCost := int(^uint(0) >> 1)
		bestFrequency := -1
		for candidateIndex := range unusedByIndex {
			candidateBucket := mergedBuckets[candidateIndex]
			transitionCost, transitionError := deltaTransitionCostBytesUnifiedCompact(currentBucket.WordSizes, candidateBucket.WordSizes)
			if transitionError != nil {
				continue
			}
			candidateFrequency := len(candidateBucket.RecordContents)
			if transitionCost < bestTransitionCost {
				bestTransitionCost = transitionCost
				nextIndex = candidateIndex
				bestFrequency = candidateFrequency
				continue
			}
			if transitionCost == bestTransitionCost {
				if candidateFrequency > bestFrequency {
					nextIndex = candidateIndex
					bestFrequency = candidateFrequency
					continue
				}
				if candidateFrequency == bestFrequency && nextIndex >= 0 {
					if compareWordSizeSlices(candidateBucket.WordSizes, mergedBuckets[nextIndex].WordSizes) < 0 {
						nextIndex = candidateIndex
						bestFrequency = candidateFrequency
					}
				}
			}
		}
		if nextIndex >= 0 {
			currentIndex = nextIndex
			continue
		}
		currentIndex = pickNextSeed()
	}

	return orderedBuckets
}

func nextBestSeedKey(sortedKeys []string, bucketMap map[string]*shapeBucket, unusedKeys map[string]struct{}) string {
	for _, shapeKey := range sortedKeys {
		if _, exists := unusedKeys[shapeKey]; exists {
			return shapeKey
		}
	}
	return ""
}

func nextNearestNeighborKey(
	currentBucket *shapeBucket,
	sortedKeys []string,
	bucketMap map[string]*shapeBucket,
	unusedKeys map[string]struct{},
) string {
	if len(unusedKeys) == 0 {
		return ""
	}

	bestKey := ""
	bestCost := int(^uint(0) >> 1)
	bestFrequency := -1
	for _, shapeKey := range sortedKeys {
		if _, exists := unusedKeys[shapeKey]; !exists {
			continue
		}

		candidateBucket := bucketMap[shapeKey]
		transitionCost, costError := deltaTransitionCostBytes(currentBucket.WordSizes, candidateBucket.WordSizes, classifyShapeBucket(currentBucket.WordSizes))
		if costError != nil {
			continue
		}
		candidateFrequency := len(candidateBucket.RecordContents)
		if transitionCost < bestCost {
			bestCost = transitionCost
			bestKey = shapeKey
			bestFrequency = candidateFrequency
			continue
		}
		if transitionCost == bestCost {
			if candidateFrequency > bestFrequency {
				bestKey = shapeKey
				bestFrequency = candidateFrequency
				continue
			}
			if candidateFrequency == bestFrequency && bestKey != "" {
				if compareWordSizeSlices(candidateBucket.WordSizes, bucketMap[bestKey].WordSizes) < 0 {
					bestKey = shapeKey
					bestFrequency = candidateFrequency
				}
			}
		}
	}
	return bestKey
}

func classifyShapeBucket(wordSizes []uint8) uint8 {
	shapeClass, classifyError := classifyShapeWordSizes(wordSizes)
	if classifyError != nil {
		return shapeClassFourBit
	}
	return shapeClass
}

func deltaTransitionCostBytes(previousWordSizes []uint8, currentWordSizes []uint8, shapeClass uint8) (int, error) {
	encodedRow, encodeError := encodeDeltaShapeRow(previousWordSizes, currentWordSizes, shapeClass)
	if encodeError != nil {
		return 0, encodeError
	}
	return len(encodedRow), nil
}

func deltaTransitionCostBytesUnifiedCompact(previousWordSizes []uint8, currentWordSizes []uint8) (int, error) {
	encodedRow, encodeError := encodeUnifiedCompactDeltaShapeRow(previousWordSizes, currentWordSizes)
	if encodeError != nil {
		return 0, encodeError
	}
	return len(encodedRow), nil
}

func buildShapeTableSection(buckets []shapeBucket, shapeClass uint8) ([]uint8, bool, error) {
	rawSection, rawBuildError := buildRawShapeTableSection(buckets, shapeClass)
	if rawBuildError != nil {
		return nil, false, rawBuildError
	}
	deltaSection, deltaBuildError := buildDeltaShapeTableSection(buckets, shapeClass)
	if deltaBuildError != nil {
		return nil, false, deltaBuildError
	}
	if len(deltaSection) < len(rawSection) {
		return deltaSection, true, nil
	}
	return rawSection, false, nil
}

func buildRawShapeTableSection(buckets []shapeBucket, shapeClass uint8) ([]uint8, error) {
	section := make([]uint8, 0, len(buckets)*6)
	for shapeIndex, bucket := range buckets {
		packedWordSizes, packError := packShapeWordSizes(bucket.WordSizes, shapeClass)
		if packError != nil {
			return nil, fmt.Errorf("pack shape sizes shape_index=%d: %w", shapeIndex, packError)
		}
		section = appendUvarint(section, uint64(len(bucket.WordSizes)))
		section = appendUvarint(section, uint64(len(packedWordSizes)))
		section = append(section, packedWordSizes...)
	}
	return section, nil
}

func buildDeltaShapeTableSection(buckets []shapeBucket, shapeClass uint8) ([]uint8, error) {
	section := make([]uint8, 0, len(buckets)*6)
	var previousWordSizes []uint8
	for shapeIndex, bucket := range buckets {
		bestEncodedRow, encodeError := encodeDeltaShapeRow(previousWordSizes, bucket.WordSizes, shapeClass)
		if encodeError != nil {
			return nil, fmt.Errorf("encode delta shape row shape_index=%d: %w", shapeIndex, encodeError)
		}
		section = append(section, bestEncodedRow...)
		previousWordSizes = bucket.WordSizes
	}
	return section, nil
}

func buildInlineShapeStreamSection(buckets []shapeBucket, shapeClass uint8) ([]uint8, bool, error) {
	section := make([]uint8, 0, len(buckets)*8)
	var previousWordSizes []uint8
	usesDeltaRows := false
	for shapeIndex, bucket := range buckets {
		section = appendUvarint(section, uint64(len(bucket.RecordContents)))
		encodedShapeRow, encodeError := encodeDeltaShapeRow(previousWordSizes, bucket.WordSizes, shapeClass)
		if encodeError != nil {
			return nil, false, fmt.Errorf("encode inline shape row shape_index=%d: %w", shapeIndex, encodeError)
		}
		if len(encodedShapeRow) > 0 && encodedShapeRow[0] != shapeDeltaDecoratorRaw {
			usesDeltaRows = true
		}
		section = append(section, encodedShapeRow...)
		previousWordSizes = bucket.WordSizes
	}
	return section, usesDeltaRows, nil
}

// buildUnifiedInlineShapeStreamSection writes one run-length + delta-shape stream.
// Compact V2 removes redundant packed-length varints in delta rows.
func buildUnifiedInlineShapeStreamSection(buckets []shapeBucket) ([]uint8, bool, error) {
	section := make([]uint8, 0, len(buckets)*6)
	var previousWordSizes []uint8
	usesDeltaRows := false
	for shapeIndex, currentBucket := range buckets {
		section = appendUvarint(section, uint64(len(currentBucket.RecordContents)))
		encodedShapeRow, encodeError := encodeUnifiedCompactDeltaShapeRow(previousWordSizes, currentBucket.WordSizes)
		if encodeError != nil {
			return nil, false, fmt.Errorf("encode unified compact shape row shape_index=%d: %w", shapeIndex, encodeError)
		}
		if len(encodedShapeRow) > 0 && encodedShapeRow[0] != shapeDeltaDecoratorRaw {
			usesDeltaRows = true
		}
		section = append(section, encodedShapeRow...)
		previousWordSizes = currentBucket.WordSizes
	}
	return section, usesDeltaRows, nil
}

func encodeUnifiedCompactDeltaShapeRow(previousWordSizes []uint8, currentWordSizes []uint8) ([]uint8, error) {
	rawRow, rawError := encodeUnifiedCompactRawShapeRow(currentWordSizes)
	if rawError != nil {
		return nil, rawError
	}
	bestRow := rawRow

	smallMutationRow, smallMutationSupported := encodeUnifiedCompactSmallMutationRow(previousWordSizes, currentWordSizes)
	if smallMutationSupported && len(smallMutationRow) < len(bestRow) {
		bestRow = smallMutationRow
	}

	prefixAppendRow, prefixAppendSupported, prefixAppendError := encodeUnifiedCompactPrefixAppendRow(previousWordSizes, currentWordSizes)
	if prefixAppendError != nil {
		return nil, prefixAppendError
	}
	if prefixAppendSupported && len(prefixAppendRow) < len(bestRow) {
		bestRow = prefixAppendRow
	}

	trimAppendRow, trimAppendSupported, trimAppendError := encodeUnifiedCompactTrimAppendRow(previousWordSizes, currentWordSizes)
	if trimAppendError != nil {
		return nil, trimAppendError
	}
	if trimAppendSupported && len(trimAppendRow) < len(bestRow) {
		bestRow = trimAppendRow
	}

	return bestRow, nil
}

func encodeUnifiedCompactRawShapeRow(wordSizes []uint8) ([]uint8, error) {
	packedWordSizes, packError := packShapeWordSizes(wordSizes, shapeClassFourBit)
	if packError != nil {
		return nil, packError
	}
	encodedRow := make([]uint8, 0, 1+1+len(packedWordSizes))
	encodedRow = append(encodedRow, shapeDeltaDecoratorRaw)
	encodedRow = appendUvarint(encodedRow, uint64(len(wordSizes)))
	encodedRow = append(encodedRow, packedWordSizes...)
	return encodedRow, nil
}

func encodeUnifiedCompactSmallMutationRow(previousWordSizes []uint8, currentWordSizes []uint8) ([]uint8, bool) {
	if len(previousWordSizes) == 0 || len(previousWordSizes) != len(currentWordSizes) {
		return nil, false
	}

	mutationOps := make([]uint8, len(currentWordSizes))
	for wordIndex := range currentWordSizes {
		deltaValue := int(currentWordSizes[wordIndex]) - int(previousWordSizes[wordIndex])
		switch deltaValue {
		case -1:
			mutationOps[wordIndex] = 0
		case 0:
			mutationOps[wordIndex] = 1
		case 1:
			mutationOps[wordIndex] = 2
		case 2:
			mutationOps[wordIndex] = 3
		default:
			return nil, false
		}
	}

	mutationBytes := packTwoBitValues(mutationOps)
	encodedRow := make([]uint8, 0, 1+len(mutationBytes))
	encodedRow = append(encodedRow, shapeDeltaDecoratorSmallMut2Bit)
	encodedRow = append(encodedRow, mutationBytes...)
	return encodedRow, true
}

func encodeUnifiedCompactPrefixAppendRow(previousWordSizes []uint8, currentWordSizes []uint8) ([]uint8, bool, error) {
	if len(previousWordSizes) == 0 || len(currentWordSizes) <= len(previousWordSizes) {
		return nil, false, nil
	}
	sharedPrefixLength := sharedPrefixLengthWordSizes(previousWordSizes, currentWordSizes)
	if sharedPrefixLength != len(previousWordSizes) {
		return nil, false, nil
	}

	appendWordSizes := currentWordSizes[len(previousWordSizes):]
	packedAppendWordSizes, packError := packShapeWordSizes(appendWordSizes, shapeClassFourBit)
	if packError != nil {
		return nil, false, packError
	}
	encodedRow := make([]uint8, 0, 1+1+len(packedAppendWordSizes))
	encodedRow = append(encodedRow, shapeDeltaDecoratorPrefixAppend)
	encodedRow = appendUvarint(encodedRow, uint64(len(appendWordSizes)))
	encodedRow = append(encodedRow, packedAppendWordSizes...)
	return encodedRow, true, nil
}

func encodeUnifiedCompactTrimAppendRow(previousWordSizes []uint8, currentWordSizes []uint8) ([]uint8, bool, error) {
	if len(previousWordSizes) == 0 {
		return nil, false, nil
	}
	sharedPrefixLength := sharedPrefixLengthWordSizes(previousWordSizes, currentWordSizes)
	trimCount := len(previousWordSizes) - sharedPrefixLength
	if trimCount <= 0 {
		return nil, false, nil
	}
	appendWordSizes := currentWordSizes[sharedPrefixLength:]
	if len(appendWordSizes) == 0 {
		return nil, false, nil
	}
	packedAppendWordSizes, packError := packShapeWordSizes(appendWordSizes, shapeClassFourBit)
	if packError != nil {
		return nil, false, packError
	}
	encodedRow := make([]uint8, 0, 1+2+len(packedAppendWordSizes))
	encodedRow = append(encodedRow, shapeDeltaDecoratorTrimAppend)
	encodedRow = appendUvarint(encodedRow, uint64(trimCount))
	encodedRow = appendUvarint(encodedRow, uint64(len(appendWordSizes)))
	encodedRow = append(encodedRow, packedAppendWordSizes...)
	return encodedRow, true, nil
}

func encodeDeltaShapeRow(previousWordSizes []uint8, currentWordSizes []uint8, shapeClass uint8) ([]uint8, error) {
	rawRow, rawError := encodeRawShapeRow(currentWordSizes, shapeClass)
	if rawError != nil {
		return nil, rawError
	}
	bestRow := rawRow

	smallMutationRow, smallMutationSupported, smallMutationError := encodeSmallMutationShapeRow(previousWordSizes, currentWordSizes)
	if smallMutationError != nil {
		return nil, smallMutationError
	}
	if smallMutationSupported && len(smallMutationRow) < len(bestRow) {
		bestRow = smallMutationRow
	}

	prefixAppendRow, prefixAppendSupported, prefixAppendError := encodePrefixAppendShapeRow(previousWordSizes, currentWordSizes, shapeClass)
	if prefixAppendError != nil {
		return nil, prefixAppendError
	}
	if prefixAppendSupported && len(prefixAppendRow) < len(bestRow) {
		bestRow = prefixAppendRow
	}

	trimAppendRow, trimAppendSupported, trimAppendError := encodeTrimAppendShapeRow(previousWordSizes, currentWordSizes, shapeClass)
	if trimAppendError != nil {
		return nil, trimAppendError
	}
	if trimAppendSupported && len(trimAppendRow) < len(bestRow) {
		bestRow = trimAppendRow
	}

	return bestRow, nil
}

func encodeRawShapeRow(wordSizes []uint8, shapeClass uint8) ([]uint8, error) {
	packedWordSizes, packError := packShapeWordSizes(wordSizes, shapeClass)
	if packError != nil {
		return nil, packError
	}
	encodedRow := make([]uint8, 0, 1+2+len(packedWordSizes))
	encodedRow = append(encodedRow, shapeDeltaDecoratorRaw)
	encodedRow = appendUvarint(encodedRow, uint64(len(wordSizes)))
	encodedRow = appendUvarint(encodedRow, uint64(len(packedWordSizes)))
	encodedRow = append(encodedRow, packedWordSizes...)
	return encodedRow, nil
}

func encodeSmallMutationShapeRow(previousWordSizes []uint8, currentWordSizes []uint8) ([]uint8, bool, error) {
	if len(previousWordSizes) == 0 || len(previousWordSizes) != len(currentWordSizes) {
		return nil, false, nil
	}

	mutationOps := make([]uint8, len(currentWordSizes))
	for wordIndex := range currentWordSizes {
		deltaValue := int(currentWordSizes[wordIndex]) - int(previousWordSizes[wordIndex])
		switch deltaValue {
		case -1:
			mutationOps[wordIndex] = 0
		case 0:
			mutationOps[wordIndex] = 1
		case 1:
			mutationOps[wordIndex] = 2
		case 2:
			mutationOps[wordIndex] = 3
		default:
			return nil, false, nil
		}
	}

	mutationBytes := packTwoBitValues(mutationOps)
	encodedRow := make([]uint8, 0, 1+1+len(mutationBytes))
	encodedRow = append(encodedRow, shapeDeltaDecoratorSmallMut2Bit)
	encodedRow = appendUvarint(encodedRow, uint64(len(mutationBytes)))
	encodedRow = append(encodedRow, mutationBytes...)
	return encodedRow, true, nil
}

func encodePrefixAppendShapeRow(previousWordSizes []uint8, currentWordSizes []uint8, shapeClass uint8) ([]uint8, bool, error) {
	if len(previousWordSizes) == 0 {
		return nil, false, nil
	}
	if len(currentWordSizes) <= len(previousWordSizes) {
		return nil, false, nil
	}
	sharedPrefixLength := sharedPrefixLengthWordSizes(previousWordSizes, currentWordSizes)
	if sharedPrefixLength != len(previousWordSizes) {
		return nil, false, nil
	}

	appendWordSizes := currentWordSizes[len(previousWordSizes):]
	packedAppendWordSizes, packError := packShapeWordSizes(appendWordSizes, shapeClass)
	if packError != nil {
		return nil, false, packError
	}

	encodedRow := make([]uint8, 0, 1+2+len(packedAppendWordSizes))
	encodedRow = append(encodedRow, shapeDeltaDecoratorPrefixAppend)
	encodedRow = appendUvarint(encodedRow, uint64(len(appendWordSizes)))
	encodedRow = appendUvarint(encodedRow, uint64(len(packedAppendWordSizes)))
	encodedRow = append(encodedRow, packedAppendWordSizes...)
	return encodedRow, true, nil
}

func encodeTrimAppendShapeRow(previousWordSizes []uint8, currentWordSizes []uint8, shapeClass uint8) ([]uint8, bool, error) {
	if len(previousWordSizes) == 0 {
		return nil, false, nil
	}

	sharedPrefixLength := sharedPrefixLengthWordSizes(previousWordSizes, currentWordSizes)
	trimCount := len(previousWordSizes) - sharedPrefixLength
	if trimCount <= 0 {
		return nil, false, nil
	}

	appendWordSizes := currentWordSizes[sharedPrefixLength:]
	if len(appendWordSizes) == 0 {
		return nil, false, nil
	}
	packedAppendWordSizes, packError := packShapeWordSizes(appendWordSizes, shapeClass)
	if packError != nil {
		return nil, false, packError
	}

	encodedRow := make([]uint8, 0, 1+3+len(packedAppendWordSizes))
	encodedRow = append(encodedRow, shapeDeltaDecoratorTrimAppend)
	encodedRow = appendUvarint(encodedRow, uint64(trimCount))
	encodedRow = appendUvarint(encodedRow, uint64(len(appendWordSizes)))
	encodedRow = appendUvarint(encodedRow, uint64(len(packedAppendWordSizes)))
	encodedRow = append(encodedRow, packedAppendWordSizes...)
	return encodedRow, true, nil
}

func packTwoBitValues(values []uint8) []uint8 {
	packedValues := make([]uint8, 0, (len(values)+3)/4)
	for valueIndex := 0; valueIndex < len(values); valueIndex += 4 {
		firstValue := uint8(0)
		secondValue := uint8(0)
		thirdValue := uint8(0)
		fourthValue := uint8(0)
		if valueIndex < len(values) {
			firstValue = values[valueIndex] & 0x03
		}
		if valueIndex+1 < len(values) {
			secondValue = values[valueIndex+1] & 0x03
		}
		if valueIndex+2 < len(values) {
			thirdValue = values[valueIndex+2] & 0x03
		}
		if valueIndex+3 < len(values) {
			fourthValue = values[valueIndex+3] & 0x03
		}
		packedByte := (firstValue << 6) | (secondValue << 4) | (thirdValue << 2) | fourthValue
		packedValues = append(packedValues, packedByte)
	}
	return packedValues
}

func buildShapeUsageSection(buckets []shapeBucket) []uint8 {
	section := make([]uint8, 0, len(buckets)*2)
	for _, bucket := range buckets {
		section = appendUvarint(section, uint64(len(bucket.RecordContents)))
	}
	return section
}

func buildContentSection(class0Buckets []shapeBucket, class1Buckets []shapeBucket) []uint8 {
	section := make([]uint8, 0, 1024)
	appendBuckets := func(buckets []shapeBucket) {
		for _, bucket := range buckets {
			for _, recordContent := range bucket.RecordContents {
				section = append(section, recordContent...)
			}
		}
	}
	appendBuckets(class0Buckets)
	appendBuckets(class1Buckets)
	return section
}

func buildContentSectionFromOrderedBuckets(orderedBuckets []shapeBucket) []uint8 {
	section := make([]uint8, 0, 1024)
	for _, currentBucket := range orderedBuckets {
		for _, recordContent := range currentBucket.RecordContents {
			section = append(section, recordContent...)
		}
	}
	return section
}

func classifyShapeWordSizes(wordSizes []uint8) (uint8, error) {
	if len(wordSizes) == 0 {
		return 0, fmt.Errorf("shape has zero words")
	}
	maxWordSize := uint8(0)
	for wordIndex, wordSize := range wordSizes {
		if wordSize < 1 {
			return 0, fmt.Errorf("word_size must be >=1 word_index=%d", wordIndex)
		}
		if wordSize > 16 {
			return 0, fmt.Errorf("word_size exceeds hard limit=16 word_index=%d word_size=%d", wordIndex, wordSize)
		}
		if wordSize > maxWordSize {
			maxWordSize = wordSize
		}
	}
	if maxWordSize <= 4 {
		return shapeClassTwoBit, nil
	}
	return shapeClassFourBit, nil
}

func packShapeWordSizes(wordSizes []uint8, shapeClass uint8) ([]uint8, error) {
	if len(wordSizes) == 0 {
		return nil, fmt.Errorf("cannot pack empty word sizes")
	}
	if shapeClass == shapeClassTwoBit {
		packed := make([]uint8, 0, (len(wordSizes)+3)/4)
		for index := 0; index < len(wordSizes); index += 4 {
			chunk := [4]uint8{1, 1, 1, 1}
			for chunkIndex := 0; chunkIndex < 4; chunkIndex++ {
				wordIndex := index + chunkIndex
				if wordIndex >= len(wordSizes) {
					break
				}
				if wordSizes[wordIndex] < 1 || wordSizes[wordIndex] > 4 {
					return nil, fmt.Errorf("word size out of range for 2-bit mode: %d", wordSizes[wordIndex])
				}
				chunk[chunkIndex] = wordSizes[wordIndex]
			}
			packedByte := ((chunk[0] - 1) << 6) | ((chunk[1] - 1) << 4) | ((chunk[2] - 1) << 2) | (chunk[3] - 1)
			packed = append(packed, packedByte)
		}
		return packed, nil
	}
	if shapeClass == shapeClassFourBit {
		packed := make([]uint8, 0, (len(wordSizes)+1)/2)
		for index := 0; index < len(wordSizes); index += 2 {
			firstSize := wordSizes[index]
			if firstSize < 1 || firstSize > 16 {
				return nil, fmt.Errorf("word size out of range for 4-bit mode: %d", firstSize)
			}
			secondSize := uint8(1)
			if index+1 < len(wordSizes) {
				secondSize = wordSizes[index+1]
				if secondSize < 1 || secondSize > 16 {
					return nil, fmt.Errorf("word size out of range for 4-bit mode: %d", secondSize)
				}
			}
			packedByte := ((firstSize - 1) << 4) | (secondSize - 1)
			packed = append(packed, packedByte)
		}
		return packed, nil
	}
	return nil, fmt.Errorf("unknown shape class: %d", shapeClass)
}

func shapeWordSizesKey(wordSizes []uint8) string {
	// The byte-level key preserves exact shape and supports deterministic comparisons.
	return string(wordSizes)
}

func compareWordSizeSlices(left []uint8, right []uint8) int {
	minimumLength := len(left)
	if len(right) < minimumLength {
		minimumLength = len(right)
	}
	for index := 0; index < minimumLength; index++ {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

func sharedPrefixLengthBytes(left []byte, right []byte) int {
	maximumPrefix := len(left)
	if len(right) < maximumPrefix {
		maximumPrefix = len(right)
	}
	for index := 0; index < maximumPrefix; index++ {
		if left[index] != right[index] {
			return index
		}
	}
	return maximumPrefix
}

func sharedPrefixLengthWordSizes(left []uint8, right []uint8) int {
	maximumPrefix := len(left)
	if len(right) < maximumPrefix {
		maximumPrefix = len(right)
	}
	for index := 0; index < maximumPrefix; index++ {
		if left[index] != right[index] {
			return index
		}
	}
	return maximumPrefix
}

func appendUvarint(target []uint8, value uint64) []uint8 {
	var buffer [binary.MaxVarintLen64]byte
	writtenBytes := binary.PutUvarint(buffer[:], value)
	return append(target, buffer[:writtenBytes]...)
}

func splitShapeCountCompactAndOverflow(shapeCount int) (uint8, uint16, error) {
	if shapeCount < 0 {
		return 0, 0, fmt.Errorf("shape count cannot be negative count=%d", shapeCount)
	}

	compactCount := shapeCount
	if compactCount > shapeCountU8Limit {
		compactCount = shapeCountU8Limit
	}
	overflowCount := shapeCount - compactCount
	if overflowCount > int(^uint16(0)) {
		return 0, 0, fmt.Errorf("shape count overflow exceeds uint16 capacity count=%d", shapeCount)
	}
	return uint8(compactCount), uint16(overflowCount), nil
}

func logSyllableSummaryLine(label string, syllables []string) {
	normalizedSyllables := make([]string, 0, len(syllables))
	for _, syllable := range syllables {
		normalizedSyllable := normalizeToken(syllable)
		if normalizedSyllable == "" {
			continue
		}
		normalizedSyllables = append(normalizedSyllables, normalizedSyllable)
	}
	log.Printf("word_parser_v2: %s (%d): %s", label, len(normalizedSyllables), strings.Join(normalizedSyllables, ", "))
}

func normalizeAndFilterTokens(rawText string, connectorSet map[string]struct{}) []string {
	normalizedText := normalizeText(rawText)
	if normalizedText == "" {
		return nil
	}

	allTokens := strings.Fields(normalizedText)
	filteredTokens := make([]string, 0, len(allTokens))
	appendFilteredToken := func(token string, preserveSingleChar bool) {
		if token == "" {
			return
		}
		if _, isConnector := connectorSet[token]; isConnector {
			return
		}
		if !preserveSingleChar && isSingleCharacterToken(token) {
			return
		}
		filteredTokens = append(filteredTokens, token)
	}
	for tokenIndex := 0; tokenIndex < len(allTokens); tokenIndex++ {
		token := allTokens[tokenIndex]
		if leadingNumericToken, trailingWordToken, hasAttachedNumericPrefix := splitNumericPrefixToken(token); hasAttachedNumericPrefix {
			// Always compact numeric prefix to one uint8-equivalent token.
			compactNumericToken := convertNumericTokenToCompactToken(leadingNumericToken)
			// Keep compact numeric + short unit in the same word, e.g. "olg".
			if len(trailingWordToken) > 0 && len(trailingWordToken) <= 3 {
				appendFilteredToken(compactNumericToken+trailingWordToken, true)
			} else {
				filteredTokens = append(filteredTokens, compactNumericToken)
				// Long suffix tokens stay separate.
				appendFilteredToken(trailingWordToken, false)
			}
			continue
		}
		// Compact standalone numbers to deterministic 1..244 buckets.
		if isNumericToken(token) {
			compactNumericToken := convertNumericTokenToCompactToken(token)
			if tokenIndex+1 < len(allTokens) {
				nextToken := allTokens[tokenIndex+1]
				if _, isNextConnector := connectorSet[nextToken]; !isNextConnector && len(nextToken) <= 3 && !isNumericToken(nextToken) {
					// Keep compact numeric + short unit in the same word, e.g. "olg".
					appendFilteredToken(compactNumericToken+nextToken, true)
					tokenIndex++
					continue
				}
			}
			filteredTokens = append(filteredTokens, compactNumericToken)
			continue
		}
		appendFilteredToken(token, false)
	}
	return filteredTokens
}

func convertNumericTokenToCompactToken(rawNumericToken string) string {
	parsedNumber, parseError := strconv.Atoi(rawNumericToken)
	if parseError != nil || parsedNumber <= 0 {
		return buildCompactNumericToken(1)
	}
	compactBucket := uint8(((parsedNumber - 1) % 244) + 1)
	return buildCompactNumericToken(compactBucket)
}

func buildCompactNumericToken(compactBucket uint8) string {
	return fmt.Sprintf("%s%03d", compactNumericTokenPrefix, compactBucket)
}

func decodeCompactNumericToken(normalizedWordToken string) (uint8, bool) {
	if !strings.HasPrefix(normalizedWordToken, compactNumericTokenPrefix) {
		return 0, false
	}
	if len(normalizedWordToken) != len(compactNumericTokenPrefix)+3 {
		return 0, false
	}
	rawBucketDigits := normalizedWordToken[len(compactNumericTokenPrefix):]
	parsedBucket, parseError := strconv.Atoi(rawBucketDigits)
	if parseError != nil || parsedBucket < 1 || parsedBucket > 244 {
		return 0, false
	}
	return uint8(parsedBucket), true
}

func splitNumericPrefixToken(rawToken string) (string, string, bool) {
	if rawToken == "" {
		return "", "", false
	}
	splitIndex := 0
	for splitIndex < len(rawToken) {
		currentByte := rawToken[splitIndex]
		if currentByte < '0' || currentByte > '9' {
			break
		}
		splitIndex++
	}
	if splitIndex == 0 || splitIndex >= len(rawToken) {
		return "", "", false
	}
	for trailingIndex := splitIndex; trailingIndex < len(rawToken); trailingIndex++ {
		currentByte := rawToken[trailingIndex]
		if currentByte < 'a' || currentByte > 'z' {
			return "", "", false
		}
	}
	return rawToken[:splitIndex], rawToken[splitIndex:], true
}

func splitCompactNumericTokenAndSuffix(normalizedWordToken string) (uint8, string, bool) {
	const compactNumericTokenLength = 5 // "n0" + 3 digits
	if len(normalizedWordToken) <= compactNumericTokenLength {
		return 0, "", false
	}
	compactNumericTokenCandidate := normalizedWordToken[:compactNumericTokenLength]
	compactNumericID, isCompactNumericToken := decodeCompactNumericToken(compactNumericTokenCandidate)
	if !isCompactNumericToken {
		return 0, "", false
	}
	compactSuffix := normalizedWordToken[compactNumericTokenLength:]
	return compactNumericID, compactSuffix, true
}

func buildConnectorSet(connectorTokens []string) map[string]struct{} {
	connectorSet := make(map[string]struct{}, len(connectorTokens))
	for _, connectorToken := range connectorTokens {
		normalizedConnector := normalizeToken(connectorToken)
		if normalizedConnector == "" {
			continue
		}
		connectorSet[normalizedConnector] = struct{}{}
	}
	return connectorSet
}

func logTopUnknownSyllables(unknownSyllableCounts map[string]int, limit int) {
	if len(unknownSyllableCounts) == 0 {
		log.Printf("word_parser_v2: unknown_syllables=0")
		return
	}
	type unknownSyllable struct {
		Syllable string
		Count    int
	}
	sortedUnknown := make([]unknownSyllable, 0, len(unknownSyllableCounts))
	for syllable, count := range unknownSyllableCounts {
		sortedUnknown = append(sortedUnknown, unknownSyllable{Syllable: syllable, Count: count})
	}
	sort.Slice(sortedUnknown, func(leftIndex, rightIndex int) bool {
		left := sortedUnknown[leftIndex]
		right := sortedUnknown[rightIndex]
		if left.Count == right.Count {
			return left.Syllable < right.Syllable
		}
		return left.Count > right.Count
	})
	if limit > len(sortedUnknown) {
		limit = len(sortedUnknown)
	}
	for index := 0; index < limit; index++ {
		log.Printf("word_parser_v2: unknown_syllable rank=%d value=%s count=%d", index+1, sortedUnknown[index].Syllable, sortedUnknown[index].Count)
	}
}

func logDictionaryUsageCoverageStats(generatedDictionary *GeneratedDictionary, encodedRecords []EncodedNameRecord) {
	if generatedDictionary == nil {
		return
	}
	if len(generatedDictionary.CombinedSlots) == 0 {
		log.Printf("word_parser_v2: usage_stats skipped empty dictionary")
		return
	}

	slotUsageCounts := make([]int, len(generatedDictionary.CombinedSlots)+1)
	totalEncodedSyllableUsages := 0
	for _, encodedRecord := range encodedRecords {
		for _, syllableID := range encodedRecord.EncodedContent {
			if int(syllableID) >= len(slotUsageCounts) {
				continue
			}
			slotUsageCounts[syllableID]++
			totalEncodedSyllableUsages++
		}
	}

	// Length buckets show how much traffic goes through 1/2/3-letter syllables.
	usageByLength := map[int]int{1: 0, 2: 0, 3: 0}
	distinctUsedByLength := map[int]int{1: 0, 2: 0, 3: 0}
	for slotIndex, token := range generatedDictionary.CombinedSlots {
		slotID := slotIndex + 1
		if slotID >= len(slotUsageCounts) {
			continue
		}
		usageCount := slotUsageCounts[slotID]
		if usageCount <= 0 {
			continue
		}
		tokenLength := len([]rune(normalizeToken(token)))
		if tokenLength < 1 || tokenLength > 3 {
			continue
		}
		usageByLength[tokenLength] += usageCount
		distinctUsedByLength[tokenLength]++
	}
	log.Printf(
		"word_parser_v2: syllable_length_usage usage_1=%d usage_2=%d usage_3=%d distinct_used_1=%d distinct_used_2=%d distinct_used_3=%d",
		usageByLength[1],
		usageByLength[2],
		usageByLength[3],
		distinctUsedByLength[1],
		distinctUsedByLength[2],
		distinctUsedByLength[3],
	)

	fixedSlotCount := len(generatedDictionary.FixedSlots)
	computedSlotCount := len(generatedDictionary.FrequentSlots)
	if computedSlotCount == 0 {
		log.Printf("word_parser_v2: computed_usage_stats computed_syllables=0")
		return
	}

	type computedSyllableUsage struct {
		Syllable string
		Count    int
	}
	computedUsageRows := make([]computedSyllableUsage, 0, computedSlotCount)
	totalComputedUsages := 0
	usedComputedSyllables := 0
	for computedIndex, computedSyllable := range generatedDictionary.FrequentSlots {
		slotID := fixedSlotCount + computedIndex + 1
		if slotID >= len(slotUsageCounts) {
			continue
		}
		usageCount := slotUsageCounts[slotID]
		totalComputedUsages += usageCount
		if usageCount > 0 {
			usedComputedSyllables++
		}
		computedUsageRows = append(computedUsageRows, computedSyllableUsage{
			Syllable: normalizeToken(computedSyllable),
			Count:    usageCount,
		})
	}

	sort.Slice(computedUsageRows, func(leftIndex, rightIndex int) bool {
		left := computedUsageRows[leftIndex]
		right := computedUsageRows[rightIndex]
		if left.Count == right.Count {
			return left.Syllable < right.Syllable
		}
		return left.Count > right.Count
	})

	averageUsageAllComputed := float64(totalComputedUsages) / float64(computedSlotCount)
	averageUsageUsedComputed := 0.0
	if usedComputedSyllables > 0 {
		averageUsageUsedComputed = float64(totalComputedUsages) / float64(usedComputedSyllables)
	}
	log.Printf(
		"word_parser_v2: computed_usage_stats total_usages=%d total_encoded_usages=%d computed_syllables=%d used_computed_syllables=%d avg_per_computed=%.2f avg_per_used_computed=%.2f",
		totalComputedUsages,
		totalEncodedSyllableUsages,
		computedSlotCount,
		usedComputedSyllables,
		averageUsageAllComputed,
		averageUsageUsedComputed,
	)

	coveragePercents := []int{20, 40, 60, 80, 100}
	for _, coveragePercent := range coveragePercents {
		syllableCountInGroup := int(math.Ceil(float64(computedSlotCount) * float64(coveragePercent) / 100.0))
		if syllableCountInGroup < 1 {
			syllableCountInGroup = 1
		}
		if syllableCountInGroup > computedSlotCount {
			syllableCountInGroup = computedSlotCount
		}
		coverageUsageCount := 0
		for usageIndex := 0; usageIndex < syllableCountInGroup; usageIndex++ {
			coverageUsageCount += computedUsageRows[usageIndex].Count
		}
		coverageOfComputedPct := 0.0
		if totalComputedUsages > 0 {
			coverageOfComputedPct = 100.0 * float64(coverageUsageCount) / float64(totalComputedUsages)
		}
		coverageOfTotalPct := 0.0
		if totalEncodedSyllableUsages > 0 {
			coverageOfTotalPct = 100.0 * float64(coverageUsageCount) / float64(totalEncodedSyllableUsages)
		}
		log.Printf(
			"word_parser_v2: computed_usage_coverage top_percent=%d syllables=%d usages=%d coverage_of_computed_pct=%.2f coverage_of_total_pct=%.2f",
			coveragePercent,
			syllableCountInGroup,
			coverageUsageCount,
			coverageOfComputedPct,
			coverageOfTotalPct,
		)
	}

	previewLimit := 20
	if previewLimit > len(computedUsageRows) {
		previewLimit = len(computedUsageRows)
	}
	previewRows := make([]string, 0, previewLimit)
	for previewIndex := 0; previewIndex < previewLimit; previewIndex++ {
		previewRows = append(previewRows, fmt.Sprintf("%s:%d", computedUsageRows[previewIndex].Syllable, computedUsageRows[previewIndex].Count))
	}
	log.Printf("word_parser_v2: computed_usage_top20=%s", strings.Join(previewRows, ", "))
}

func logShapeDiversityStats(encodedRecords []EncodedNameRecord) {
	if len(encodedRecords) == 0 {
		return
	}

	type shapeCounter struct {
		Key   string
		Count int
		Class uint8
		Shape []uint8
	}
	shapeByKey := make(map[string]*shapeCounter, len(encodedRecords))
	classRecordCounts := map[uint8]int{
		shapeClassTwoBit:  0,
		shapeClassFourBit: 0,
	}
	classUniqueCounts := map[uint8]int{
		shapeClassTwoBit:  0,
		shapeClassFourBit: 0,
	}

	for _, encodedRecord := range encodedRecords {
		shapeClass, classifyError := classifyShapeWordSizes(encodedRecord.WordSizes)
		if classifyError != nil {
			continue
		}
		classRecordCounts[shapeClass]++
		shapeKey := fmt.Sprintf("%d:%v", shapeClass, encodedRecord.WordSizes)
		shapeRow, exists := shapeByKey[shapeKey]
		if !exists {
			copiedWordSizes := append([]uint8(nil), encodedRecord.WordSizes...)
			shapeRow = &shapeCounter{
				Key:   shapeKey,
				Count: 0,
				Class: shapeClass,
				Shape: copiedWordSizes,
			}
			shapeByKey[shapeKey] = shapeRow
			classUniqueCounts[shapeClass]++
		}
		shapeRow.Count++
	}

	shapeRows := make([]shapeCounter, 0, len(shapeByKey))
	for _, shapeRow := range shapeByKey {
		shapeRows = append(shapeRows, *shapeRow)
	}
	sort.Slice(shapeRows, func(leftIndex, rightIndex int) bool {
		left := shapeRows[leftIndex]
		right := shapeRows[rightIndex]
		if left.Count == right.Count {
			if left.Class == right.Class {
				return compareWordSizeSlices(left.Shape, right.Shape) < 0
			}
			return left.Class < right.Class
		}
		return left.Count > right.Count
	})

	log.Printf(
		"word_parser_v2: shape_diversity total_records=%d unique_shapes=%d class0_records=%d class0_unique=%d class1_records=%d class1_unique=%d",
		len(encodedRecords),
		len(shapeRows),
		classRecordCounts[shapeClassTwoBit],
		classUniqueCounts[shapeClassTwoBit],
		classRecordCounts[shapeClassFourBit],
		classUniqueCounts[shapeClassFourBit],
	)

	previewLimit := 20
	if previewLimit > len(shapeRows) {
		previewLimit = len(shapeRows)
	}
	previewRows := make([]string, 0, previewLimit)
	for previewIndex := 0; previewIndex < previewLimit; previewIndex++ {
		previewRows = append(
			previewRows,
			fmt.Sprintf("class=%d shape=%v count=%d", shapeRows[previewIndex].Class, shapeRows[previewIndex].Shape, shapeRows[previewIndex].Count),
		)
	}
	log.Printf("word_parser_v2: shape_top20=%s", strings.Join(previewRows, " | "))
}

func logShapeCoverageStats(encodedRecords []EncodedNameRecord, compactShapeLimit int) {
	if len(encodedRecords) == 0 {
		return
	}
	if compactShapeLimit <= 0 {
		return
	}

	// Aggregate frequency by exact shape to rank the most common patterns.
	type shapeCounter struct {
		Class uint8
		Shape []uint8
		Count int
	}
	shapeCountersByKey := make(map[string]*shapeCounter, len(encodedRecords))
	for _, encodedRecord := range encodedRecords {
		shapeClass, classifyError := classifyShapeWordSizes(encodedRecord.WordSizes)
		if classifyError != nil {
			continue
		}
		shapeKey := fmt.Sprintf("%d:%v", shapeClass, encodedRecord.WordSizes)
		shapeCounterRow, exists := shapeCountersByKey[shapeKey]
		if !exists {
			shapeCounterRow = &shapeCounter{
				Class: shapeClass,
				Shape: append([]uint8(nil), encodedRecord.WordSizes...),
			}
			shapeCountersByKey[shapeKey] = shapeCounterRow
		}
		shapeCounterRow.Count++
	}

	sortedShapeCounters := make([]shapeCounter, 0, len(shapeCountersByKey))
	for _, shapeCounterRow := range shapeCountersByKey {
		sortedShapeCounters = append(sortedShapeCounters, *shapeCounterRow)
	}
	sort.Slice(sortedShapeCounters, func(leftIndex, rightIndex int) bool {
		leftShapeCounter := sortedShapeCounters[leftIndex]
		rightShapeCounter := sortedShapeCounters[rightIndex]
		if leftShapeCounter.Count != rightShapeCounter.Count {
			return leftShapeCounter.Count > rightShapeCounter.Count
		}
		if leftShapeCounter.Class != rightShapeCounter.Class {
			return leftShapeCounter.Class < rightShapeCounter.Class
		}
		return compareWordSizeSlices(leftShapeCounter.Shape, rightShapeCounter.Shape) < 0
	})

	if compactShapeLimit > len(sortedShapeCounters) {
		compactShapeLimit = len(sortedShapeCounters)
	}

	recordsInTopCompactShapes := 0
	for shapeIndex := 0; shapeIndex < compactShapeLimit; shapeIndex++ {
		recordsInTopCompactShapes += sortedShapeCounters[shapeIndex].Count
	}

	recordsEscapingCompactShapes := len(encodedRecords) - recordsInTopCompactShapes
	nonCommonShapesCount := len(sortedShapeCounters) - compactShapeLimit

	coveragePercent := 0.0
	if len(encodedRecords) > 0 {
		coveragePercent = 100.0 * float64(recordsInTopCompactShapes) / float64(len(encodedRecords))
	}
	log.Printf(
		"word_parser_v2: shape_coverage_top%d records_in_compact=%d records_escaping_compact=%d non_common_shapes=%d total_shapes=%d coverage_pct=%.2f",
		compactShapeLimit,
		recordsInTopCompactShapes,
		recordsEscapingCompactShapes,
		nonCommonShapesCount,
		len(sortedShapeCounters),
		coveragePercent,
	)
}
