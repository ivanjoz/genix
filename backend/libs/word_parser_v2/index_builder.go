package word_parser_v2

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
)

const (
	binaryIndexMagicV3        = "WPV3IDX1"
	binaryIndexVersion        = uint8(1)
	headerFlagDictionaryDelta = uint8(1 << 0)
	headerSizeV3              = 29
)

const (
	shapeClassTwoBit  = uint8(0)
	shapeClassFourBit = uint8(1)
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
	for _, rawProductName := range productNames {
		normalizedTokens := normalizeAndFilterTokens(rawProductName, connectorSet)
		if len(normalizedTokens) == 0 {
			continue
		}
		cleanedProductNames = append(cleanedProductNames, strings.Join(normalizedTokens, " "))
	}
	if len(cleanedProductNames) == 0 {
		return fmt.Errorf("all product names were empty after normalization and connector filtering")
	}

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
	logDictionaryUsageCoverageStats(generatedDictionary, encodedRecords)
	logShapeDiversityStats(encodedRecords)

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
	return nil
}

func encodeProductName(cleanedProductName string, syllableToID map[string]uint8) (EncodedNameRecord, error) {
	wordTokens := strings.Fields(cleanedProductName)
	encodedContent := make([]uint8, 0, 32)
	wordSizes := make([]uint8, 0, len(wordTokens))

	for _, wordToken := range wordTokens {
		extractedSyllables := splitWordIntoSyllables(wordToken)
		if len(extractedSyllables) == 0 {
			continue
		}
		encodedWord := make([]uint8, 0, len(extractedSyllables))
		for _, syllable := range extractedSyllables {
			syllableID, exists := syllableToID[syllable]
			if !exists {
				continue
			}
			encodedWord = append(encodedWord, syllableID)
		}
		if len(encodedWord) == 0 {
			continue
		}
		if len(encodedWord) > 16 {
			// V3 enforces the hard bound used by shape class packing.
			return EncodedNameRecord{}, fmt.Errorf("word exceeds max syllables=16 token=%q syllables=%d", wordToken, len(encodedWord))
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
		for _, extractedSyllable := range splitWordIntoSyllables(wordToken) {
			if _, exists := syllableToID[extractedSyllable]; exists {
				continue
			}
			unknownSyllables = append(unknownSyllables, extractedSyllable)
		}
	}
	return unknownSyllables
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
		shouldKeep := extractedFrequencyBySyllable[normalizedSyllable] > 0 || isNumericToken(normalizedSyllable)
		if !shouldKeep {
			continue
		}
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
		headerFlags |= headerFlagDictionaryDelta
	}

	class0Buckets, class1Buckets, splitError := groupRecordsBySortedShapes(remappedRecords)
	if splitError != nil {
		return nil, splitError
	}
	if len(class0Buckets)+len(class1Buckets) > 255 {
		return nil, fmt.Errorf("shape count exceeds uint8 limit class0=%d class1=%d", len(class0Buckets), len(class1Buckets))
	}

	class0ShapeTable, class0ShapeTableError := buildShapeTableSection(class0Buckets, shapeClassTwoBit)
	if class0ShapeTableError != nil {
		return nil, class0ShapeTableError
	}
	class1ShapeTable, class1ShapeTableError := buildShapeTableSection(class1Buckets, shapeClassFourBit)
	if class1ShapeTableError != nil {
		return nil, class1ShapeTableError
	}
	class0UsageSection := buildShapeUsageSection(class0Buckets)
	class1UsageSection := buildShapeUsageSection(class1Buckets)
	contentSection := buildContentSection(class0Buckets, class1Buckets)

	binaryPayload := make([]uint8, 0, headerSizeV3+len(dictionarySection)+len(class0ShapeTable)+len(class1ShapeTable)+len(class0UsageSection)+len(class1UsageSection)+len(contentSection))
	binaryPayload = append(binaryPayload, []byte(binaryIndexMagicV3)...)
	binaryPayload = append(binaryPayload, binaryIndexVersion)
	binaryPayload = append(binaryPayload, headerFlags)

	var recordCountBuffer [4]byte
	binary.LittleEndian.PutUint32(recordCountBuffer[:], uint32(len(remappedRecords)))
	binaryPayload = append(binaryPayload, recordCountBuffer[:]...)

	binaryPayload = append(binaryPayload, uint8(len(compactedDictionaryTokens)))
	binaryPayload = append(binaryPayload, uint8(len(class0Buckets)))
	binaryPayload = append(binaryPayload, uint8(len(class1Buckets)))

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
		"word_parser_v2: v3_index header_size=%d index_size=%d content_size=%d header_flags=%d dictionary_count=%d dictionary_bytes=%d shape_class0=%d shape_class1=%d",
		headerSizeV3,
		len(binaryPayload),
		len(contentSection),
		headerFlags,
		len(compactedDictionaryTokens),
		len(dictionarySection),
		len(class0Buckets),
		len(class1Buckets),
	)

	return binaryPayload, nil
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
	sort.Slice(keys, func(leftIndex, rightIndex int) bool {
		leftWordSizes := bucketMap[keys[leftIndex]].WordSizes
		rightWordSizes := bucketMap[keys[rightIndex]].WordSizes
		return compareWordSizeSlices(leftWordSizes, rightWordSizes) < 0
	})

	buckets := make([]shapeBucket, 0, len(keys))
	for _, shapeKey := range keys {
		bucket := bucketMap[shapeKey]
		buckets = append(buckets, *bucket)
	}
	return buckets
}

func buildShapeTableSection(buckets []shapeBucket, shapeClass uint8) ([]uint8, error) {
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

func appendUvarint(target []uint8, value uint64) []uint8 {
	var buffer [binary.MaxVarintLen64]byte
	writtenBytes := binary.PutUvarint(buffer[:], value)
	return append(target, buffer[:writtenBytes]...)
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
	for _, token := range allTokens {
		if _, isConnector := connectorSet[token]; isConnector {
			continue
		}
		filteredTokens = append(filteredTokens, token)
	}
	return filteredTokens
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
