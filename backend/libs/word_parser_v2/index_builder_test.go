package word_parser_v2

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBuildNamesBinaryIndexFromFileWritesBinaryOutput(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "productos.txt")
	outputPath := filepath.Join(tempDir, "productos.idx")

	inputContent := "Queso de cabra\nGel de ducha para piel\nKGS pack en caja\n"
	if writeError := os.WriteFile(inputPath, []byte(inputContent), 0o644); writeError != nil {
		t.Fatalf("failed to create input file: %v", writeError)
	}

	fixedConfig := DefaultFixedSyllableGeneratorConfig()
	fixedConfig.MaxSlots = 120
	frequentConfig := DefaultFrequentSyllableGeneratorConfig()
	frequentConfig.TotalSlots = 180
	frequentConfig.TopFrequentCount = 80

	if buildError := BuildNamesBinaryIndexFromFile(inputPath, outputPath, fixedConfig, frequentConfig); buildError != nil {
		t.Fatalf("unexpected build error: %v", buildError)
	}

	binaryOutput, readError := os.ReadFile(outputPath)
	if readError != nil {
		t.Fatalf("failed to read output idx file: %v", readError)
	}
	if len(binaryOutput) < headerSizeV3 {
		t.Fatalf("binary output too small: %d", len(binaryOutput))
	}
	if string(binaryOutput[:len(binaryIndexMagicV3)]) != binaryIndexMagicV3 {
		t.Fatalf("unexpected binary magic header: %q", string(binaryOutput[:len(binaryIndexMagicV3)]))
	}
	if binaryOutput[len(binaryIndexMagicV3)] != binaryIndexVersion {
		t.Fatalf("unexpected binary version=%d", binaryOutput[len(binaryIndexMagicV3)])
	}

	recordCount := binary.LittleEndian.Uint32(binaryOutput[10:14])
	dictionaryCount := binaryOutput[14]
	shapeCountClass0 := binaryOutput[15]
	shapeCountClass1 := binaryOutput[16]
	shapeCountClass0Overflow := binary.LittleEndian.Uint16(binaryOutput[17:19])
	shapeCountClass1Overflow := binary.LittleEndian.Uint16(binaryOutput[19:21])
	dictionaryBytes := binary.LittleEndian.Uint32(binaryOutput[21:25])
	shapeTableClass0Bytes := binary.LittleEndian.Uint32(binaryOutput[25:29])
	shapeTableClass1Bytes := binary.LittleEndian.Uint32(binaryOutput[29:33])

	if recordCount == 0 {
		t.Fatalf("expected record_count > 0")
	}
	if dictionaryCount == 0 {
		t.Fatalf("expected dictionary_count > 0")
	}
	totalShapeCount := int(shapeCountClass0) + int(shapeCountClass1) + int(shapeCountClass0Overflow) + int(shapeCountClass1Overflow)
	if totalShapeCount != 0 {
		t.Fatalf("expected tableless shape header counters=0 got=%d", totalShapeCount)
	}
	if shapeTableClass0Bytes != 0 || shapeTableClass1Bytes != 0 {
		t.Fatalf("expected tableless shape table bytes=0 got class0=%d class1=%d", shapeTableClass0Bytes, shapeTableClass1Bytes)
	}

	sectionStart := uint32(headerSizeV3)
	sectionEnd := sectionStart + dictionaryBytes + shapeTableClass0Bytes + shapeTableClass1Bytes
	if sectionEnd > uint32(len(binaryOutput)) {
		t.Fatalf("invalid section boundaries start=%d end=%d total=%d", sectionStart, sectionEnd, len(binaryOutput))
	}
}

func TestBuildNamesBinaryIndexFromRealProductosFileUsesAllRequestedSlots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real corpus integration test in short mode")
	}

	inputPath := "productos.txt"
	if _, statError := os.Stat(inputPath); statError != nil {
		fallbackPath := filepath.Join("libs", "word_parser_v2", "productos.txt")
		if _, fallbackStatError := os.Stat(fallbackPath); fallbackStatError != nil {
			t.Fatalf("missing real productos.txt fixture: %v / %v", statError, fallbackStatError)
		}
		inputPath = fallbackPath
	}

	productNames, loadError := LoadProductNamesFromFile(inputPath)
	if loadError != nil {
		t.Fatalf("failed to load real productos file: %v", loadError)
	}
	if len(productNames) != 10000 {
		t.Fatalf("expected 10000 product names, got %d", len(productNames))
	}

	fixedConfig := DefaultFixedSyllableGeneratorConfig()
	fixedConfig.MaxSlots = 120
	frequentConfig := DefaultFrequentSyllableGeneratorConfig()
	frequentConfig.TotalSlots = 254
	frequentConfig.TopFrequentCount = 200

	cleanedProductNames := make([]string, 0, len(productNames))
	connectorSet := buildConnectorSet(fixedConfig.ConnectorTokens)
	for _, rawProductName := range productNames {
		normalizedTokens := normalizeAndFilterTokens(rawProductName, connectorSet)
		if len(normalizedTokens) == 0 {
			continue
		}
		cleanedProductNames = append(cleanedProductNames, strings.Join(normalizedTokens, " "))
	}

	fixedSlots, reservedSyllables, _, fixedError := GenerateFixedSyllableSlotsDetailed(fixedConfig)
	if fixedError != nil {
		t.Fatalf("unexpected fixed generation error: %v", fixedError)
	}
	generatedDictionary, frequentError := GenerateFrequentSyllableSlotsWithReserved(cleanedProductNames, fixedSlots, reservedSyllables, frequentConfig)
	if frequentError != nil {
		t.Fatalf("unexpected frequent generation error: %v", frequentError)
	}

	totalSlots := len(generatedDictionary.FixedSlots) + len(generatedDictionary.FrequentSlots)
	if totalSlots != frequentConfig.TotalSlots {
		t.Fatalf("expected fixed+computed=%d, got fixed=%d computed=%d total=%d", frequentConfig.TotalSlots, len(generatedDictionary.FixedSlots), len(generatedDictionary.FrequentSlots), totalSlots)
	}
	if len(generatedDictionary.CombinedSlots) != frequentConfig.TotalSlots {
		t.Fatalf("expected combined slots=%d, got %d", frequentConfig.TotalSlots, len(generatedDictionary.CombinedSlots))
	}
}

func TestBuildBinaryIndexPayloadUsesDeltaDictionaryWhenSmaller(t *testing.T) {
	dictionarySlots := []string{"casaaaa", "casaaab", "casaaac", "casaaad", "casaaae", "casaaaf"}
	records := []EncodedNameRecord{
		{WordSizes: []uint8{2}, EncodedContent: []uint8{1, 2}},
		{WordSizes: []uint8{2}, EncodedContent: []uint8{3, 4}},
		{WordSizes: []uint8{2}, EncodedContent: []uint8{5, 6}},
	}

	binaryOutput, buildError := buildBinaryIndexPayload(dictionarySlots, records)
	if buildError != nil {
		t.Fatalf("unexpected payload build error: %v", buildError)
	}
	if len(binaryOutput) < headerSizeV3 {
		t.Fatalf("payload too small: %d", len(binaryOutput))
	}

	headerFlags := binaryOutput[9]
	if headerFlags&headerFlagDictionaryDelta == 0 {
		t.Fatalf("expected dictionary delta mode to be enabled flags=%08b", headerFlags)
	}
}

func TestBuildBinaryIndexPayloadStoresShapeClasses(t *testing.T) {
	dictionarySlots := []string{"ab", "ac", "ad", "ae", "af", "ag", "ah", "ai", "aj"}
	records := []EncodedNameRecord{
		{WordSizes: []uint8{2, 4}, EncodedContent: []uint8{1, 2, 3, 4, 5, 6}},
		{WordSizes: []uint8{5}, EncodedContent: []uint8{7, 8, 9, 1, 2}},
	}

	binaryOutput, buildError := buildBinaryIndexPayload(dictionarySlots, records)
	if buildError != nil {
		t.Fatalf("unexpected payload build error: %v", buildError)
	}

	shapeCountClass0 := binaryOutput[15]
	shapeCountClass1 := binaryOutput[16]
	if shapeCountClass0 != 0 {
		t.Fatalf("expected unified stream to keep class0 shape count=0 got=%d", shapeCountClass0)
	}
	if shapeCountClass1 != 0 {
		t.Fatalf("expected tableless shape stream class1 shape count=0 got=%d", shapeCountClass1)
	}
}

func TestClassifyShapeWordSizesRejectsWordLongerThan16(t *testing.T) {
	_, classifyError := classifyShapeWordSizes([]uint8{3, 17})
	if classifyError == nil {
		t.Fatalf("expected classify error for oversized word size")
	}
}

func TestNormalizeAndFilterTokensRemovesConnectors(t *testing.T) {
	connectors := buildConnectorSet([]string{"de", "la", "con", "para", "en"})
	tokens := normalizeAndFilterTokens("Queso de cabra en salsa", connectors)
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens after connector filtering, got %d (%v)", len(tokens), tokens)
	}
	if tokens[0] != "queso" || tokens[1] != "cabra" || tokens[2] != "salsa" {
		t.Fatalf("unexpected normalized tokens: %v", tokens)
	}
}

func TestBuildBinaryIndexPayloadStoresOverflowShapeCounters(t *testing.T) {
	dictionarySlots := []string{"aa"}
	records := make([]EncodedNameRecord, 0, 320)

	// Build many unique class-0 shapes so class0 count exceeds the uint8 compact range.
	targetShapeCount := 320
	for wordCount := 1; len(records) < targetShapeCount; wordCount++ {
		currentShape := make([]uint8, wordCount)
		for slotIndex := range currentShape {
			currentShape[slotIndex] = 1
		}
		for {
			totalSyllables := 0
			for _, wordSize := range currentShape {
				totalSyllables += int(wordSize)
			}
			encodedContent := make([]uint8, totalSyllables)
			for contentIndex := range encodedContent {
				encodedContent[contentIndex] = 1
			}
			records = append(records, EncodedNameRecord{
				WordSizes:      append([]uint8(nil), currentShape...),
				EncodedContent: encodedContent,
			})
			if len(records) >= targetShapeCount {
				break
			}

			cursorIndex := len(currentShape) - 1
			for ; cursorIndex >= 0; cursorIndex-- {
				if currentShape[cursorIndex] < 4 {
					currentShape[cursorIndex]++
					break
				}
				currentShape[cursorIndex] = 1
			}
			if cursorIndex < 0 {
				break
			}
		}
	}

	binaryOutput, buildError := buildBinaryIndexPayload(dictionarySlots, records)
	if buildError != nil {
		t.Fatalf("unexpected payload build error: %v", buildError)
	}
	if len(binaryOutput) < headerSizeV3 {
		t.Fatalf("payload too small: %d", len(binaryOutput))
	}
	shapeCountClass0Compact := binaryOutput[15]
	shapeCountClass0Overflow := binary.LittleEndian.Uint16(binaryOutput[17:19])
	shapeCountClass1Compact := binaryOutput[16]
	shapeCountClass1Overflow := binary.LittleEndian.Uint16(binaryOutput[19:21])
	if shapeCountClass0Compact != 0 || shapeCountClass0Overflow != 0 || shapeCountClass1Compact != 0 || shapeCountClass1Overflow != 0 {
		t.Fatalf("expected tableless shape counters all zero got c0=%d c0o=%d c1=%d c1o=%d", shapeCountClass0Compact, shapeCountClass0Overflow, shapeCountClass1Compact, shapeCountClass1Overflow)
	}
}

func TestBuildBinaryIndexPayloadEnablesShapeDeltaModeWhenSmaller(t *testing.T) {
	dictionarySlots := []string{"aa"}
	records := make([]EncodedNameRecord, 0, 120)

	// Use long similar class-0 shapes to make prefix/suffix delta metadata smaller than raw shape rows.
	basePrefixLength := 18
	for prefixVariant := 1; prefixVariant <= 3; prefixVariant++ {
		for firstTail := 1; firstTail <= 4; firstTail++ {
			for secondTail := 1; secondTail <= 4; secondTail++ {
				wordSizes := make([]uint8, 0, basePrefixLength+2)
				for prefixIndex := 0; prefixIndex < basePrefixLength; prefixIndex++ {
					wordSizes = append(wordSizes, uint8(prefixVariant))
				}
				wordSizes = append(wordSizes, uint8(firstTail), uint8(secondTail))

				totalSyllables := 0
				for _, wordSize := range wordSizes {
					totalSyllables += int(wordSize)
				}
				encodedContent := make([]uint8, totalSyllables)
				for contentIndex := range encodedContent {
					encodedContent[contentIndex] = 1
				}
				records = append(records, EncodedNameRecord{
					WordSizes:      wordSizes,
					EncodedContent: encodedContent,
				})
				if len(records) >= 120 {
					break
				}
			}
			if len(records) >= 120 {
				break
			}
		}
		if len(records) >= 120 {
			break
		}
	}
	if len(records) < 40 {
		t.Fatalf("insufficient generated records for delta-mode test: %d", len(records))
	}

	_, buildError := buildBinaryIndexPayload(dictionarySlots, records)
	if buildError == nil {
		t.Fatalf("expected payload build error for shape with >8 words")
	}
	if !strings.Contains(buildError.Error(), "too many words") {
		t.Fatalf("expected max-8-words error, got: %v", buildError)
	}
}

func TestBuildDeltaShapeTableSectionRoundTripDecorators(t *testing.T) {
	buckets := []shapeBucket{
		{WordSizes: []uint8{2, 2, 2, 2, 2, 2}},
		{WordSizes: []uint8{2, 2, 2, 2, 2, 3}},    // small mutation (+1 on last)
		{WordSizes: []uint8{2, 2, 2, 2, 2, 3, 4}}, // prefix append
		{WordSizes: []uint8{2, 2, 2, 2, 1}},       // trim append
	}

	encodedSection, buildError := buildDeltaShapeTableSection(buckets, shapeClassTwoBit)
	if buildError != nil {
		t.Fatalf("unexpected delta section build error: %v", buildError)
	}
	decodedShapes, decodeError := decodeDeltaShapeTableSectionForTest(encodedSection, len(buckets), shapeClassTwoBit)
	if decodeError != nil {
		t.Fatalf("unexpected delta section decode error: %v", decodeError)
	}
	if len(decodedShapes) != len(buckets) {
		t.Fatalf("expected decoded shape count=%d got=%d", len(buckets), len(decodedShapes))
	}

	for shapeIndex := range buckets {
		expected := buckets[shapeIndex].WordSizes
		got := decodedShapes[shapeIndex]
		if compareWordSizeSlices(expected, got) != 0 {
			t.Fatalf("shape mismatch index=%d expected=%v got=%v", shapeIndex, expected, got)
		}
	}
}

func TestEncodeProductNameFallsBackToSingleLettersForUnknownSyllable(t *testing.T) {
	syllableToID := map[string]uint8{
		"c": 1,
		"h": 2,
		"o": 3,
	}

	encodedRecord, encodeError := encodeProductName("cho", syllableToID)
	if encodeError != nil {
		t.Fatalf("unexpected encode error: %v", encodeError)
	}
	if len(encodedRecord.WordSizes) != 1 || encodedRecord.WordSizes[0] != 3 {
		t.Fatalf("unexpected word sizes: %v", encodedRecord.WordSizes)
	}
	if len(encodedRecord.EncodedContent) != 3 {
		t.Fatalf("unexpected encoded content length: %v", encodedRecord.EncodedContent)
	}
	if encodedRecord.EncodedContent[0] != 1 || encodedRecord.EncodedContent[1] != 2 || encodedRecord.EncodedContent[2] != 3 {
		t.Fatalf("unexpected encoded content values: %v", encodedRecord.EncodedContent)
	}
}

func decodeDeltaShapeTableSectionForTest(encodedSection []uint8, expectedShapeCount int, shapeClass uint8) ([][]uint8, error) {
	decodedShapes := make([][]uint8, 0, expectedShapeCount)
	cursorIndex := 0
	var previousWordSizes []uint8
	for len(decodedShapes) < expectedShapeCount {
		if cursorIndex >= len(encodedSection) {
			return nil, os.ErrInvalid
		}
		decorator := encodedSection[cursorIndex]
		cursorIndex++

		switch decorator {
		case shapeDeltaDecoratorRaw:
			wordCount, nextCursorIndex, readWordCountError := readUvarintForTest(encodedSection, cursorIndex)
			if readWordCountError != nil {
				return nil, readWordCountError
			}
			cursorIndex = nextCursorIndex
			packedLength, nextPackedCursorIndex, readPackedLengthError := readUvarintForTest(encodedSection, cursorIndex)
			if readPackedLengthError != nil {
				return nil, readPackedLengthError
			}
			cursorIndex = nextPackedCursorIndex
			if cursorIndex+int(packedLength) > len(encodedSection) {
				return nil, os.ErrInvalid
			}
			packedWordSizes := encodedSection[cursorIndex : cursorIndex+int(packedLength)]
			cursorIndex += int(packedLength)
			wordSizes, unpackError := unpackShapeWordSizesForTest(packedWordSizes, int(wordCount), shapeClass)
			if unpackError != nil {
				return nil, unpackError
			}
			decodedShapes = append(decodedShapes, wordSizes)
			previousWordSizes = wordSizes

		case shapeDeltaDecoratorSmallMut2Bit:
			if len(previousWordSizes) == 0 {
				return nil, os.ErrInvalid
			}
			mutationLength, nextCursorIndex, readMutationLengthError := readUvarintForTest(encodedSection, cursorIndex)
			if readMutationLengthError != nil {
				return nil, readMutationLengthError
			}
			cursorIndex = nextCursorIndex
			if cursorIndex+int(mutationLength) > len(encodedSection) {
				return nil, os.ErrInvalid
			}
			mutationBytes := encodedSection[cursorIndex : cursorIndex+int(mutationLength)]
			cursorIndex += int(mutationLength)
			operationCodes := unpackTwoBitValuesForTest(mutationBytes, len(previousWordSizes))
			currentWordSizes := make([]uint8, len(previousWordSizes))
			for wordIndex, operationCode := range operationCodes {
				baseWordSize := previousWordSizes[wordIndex]
				var deltaValue int
				switch operationCode {
				case 0:
					deltaValue = -1
				case 1:
					deltaValue = 0
				case 2:
					deltaValue = 1
				case 3:
					deltaValue = 2
				default:
					return nil, os.ErrInvalid
				}
				newWordSize := int(baseWordSize) + deltaValue
				if newWordSize < 1 || newWordSize > 16 {
					return nil, os.ErrInvalid
				}
				currentWordSizes[wordIndex] = uint8(newWordSize)
			}
			decodedShapes = append(decodedShapes, currentWordSizes)
			previousWordSizes = currentWordSizes

		case shapeDeltaDecoratorPrefixAppend:
			if len(previousWordSizes) == 0 {
				return nil, os.ErrInvalid
			}
			appendWordCount, nextCursorIndex, readAppendWordCountError := readUvarintForTest(encodedSection, cursorIndex)
			if readAppendWordCountError != nil {
				return nil, readAppendWordCountError
			}
			cursorIndex = nextCursorIndex
			packedLength, nextPackedCursorIndex, readPackedLengthError := readUvarintForTest(encodedSection, cursorIndex)
			if readPackedLengthError != nil {
				return nil, readPackedLengthError
			}
			cursorIndex = nextPackedCursorIndex
			if cursorIndex+int(packedLength) > len(encodedSection) {
				return nil, os.ErrInvalid
			}
			packedAppend := encodedSection[cursorIndex : cursorIndex+int(packedLength)]
			cursorIndex += int(packedLength)
			appendWordSizes, unpackError := unpackShapeWordSizesForTest(packedAppend, int(appendWordCount), shapeClass)
			if unpackError != nil {
				return nil, unpackError
			}
			currentWordSizes := append(append([]uint8(nil), previousWordSizes...), appendWordSizes...)
			decodedShapes = append(decodedShapes, currentWordSizes)
			previousWordSizes = currentWordSizes

		case shapeDeltaDecoratorTrimAppend:
			if len(previousWordSizes) == 0 {
				return nil, os.ErrInvalid
			}
			trimCount, nextCursorIndex, readTrimCountError := readUvarintForTest(encodedSection, cursorIndex)
			if readTrimCountError != nil {
				return nil, readTrimCountError
			}
			cursorIndex = nextCursorIndex
			appendWordCount, nextAppendCursorIndex, readAppendWordCountError := readUvarintForTest(encodedSection, cursorIndex)
			if readAppendWordCountError != nil {
				return nil, readAppendWordCountError
			}
			cursorIndex = nextAppendCursorIndex
			packedLength, nextPackedCursorIndex, readPackedLengthError := readUvarintForTest(encodedSection, cursorIndex)
			if readPackedLengthError != nil {
				return nil, readPackedLengthError
			}
			cursorIndex = nextPackedCursorIndex
			if cursorIndex+int(packedLength) > len(encodedSection) {
				return nil, os.ErrInvalid
			}
			packedAppend := encodedSection[cursorIndex : cursorIndex+int(packedLength)]
			cursorIndex += int(packedLength)
			appendWordSizes, unpackError := unpackShapeWordSizesForTest(packedAppend, int(appendWordCount), shapeClass)
			if unpackError != nil {
				return nil, unpackError
			}
			if int(trimCount) > len(previousWordSizes) {
				return nil, os.ErrInvalid
			}
			currentWordSizes := append(append([]uint8(nil), previousWordSizes[:len(previousWordSizes)-int(trimCount)]...), appendWordSizes...)
			decodedShapes = append(decodedShapes, currentWordSizes)
			previousWordSizes = currentWordSizes

		default:
			return nil, os.ErrInvalid
		}
	}
	return decodedShapes, nil
}

func readUvarintForTest(source []uint8, offset int) (uint64, int, error) {
	if offset >= len(source) {
		return 0, offset, os.ErrInvalid
	}
	readValue, consumedBytes := binary.Uvarint(source[offset:])
	if consumedBytes <= 0 {
		return 0, offset, os.ErrInvalid
	}
	return readValue, offset + consumedBytes, nil
}

func unpackTwoBitValuesForTest(packedValues []uint8, expectedCount int) []uint8 {
	unpackedValues := make([]uint8, 0, expectedCount)
	for _, packedByte := range packedValues {
		unpackedValues = append(unpackedValues, (packedByte>>6)&0x03)
		if len(unpackedValues) >= expectedCount {
			break
		}
		unpackedValues = append(unpackedValues, (packedByte>>4)&0x03)
		if len(unpackedValues) >= expectedCount {
			break
		}
		unpackedValues = append(unpackedValues, (packedByte>>2)&0x03)
		if len(unpackedValues) >= expectedCount {
			break
		}
		unpackedValues = append(unpackedValues, packedByte&0x03)
		if len(unpackedValues) >= expectedCount {
			break
		}
	}
	return unpackedValues
}

func unpackShapeWordSizesForTest(packedWordSizes []uint8, wordCount int, shapeClass uint8) ([]uint8, error) {
	if wordCount <= 0 {
		return nil, os.ErrInvalid
	}
	wordSizes := make([]uint8, 0, wordCount)
	if shapeClass == shapeClassTwoBit {
		for _, packedByte := range packedWordSizes {
			wordSizes = append(wordSizes, ((packedByte>>6)&0x03)+1)
			if len(wordSizes) >= wordCount {
				break
			}
			wordSizes = append(wordSizes, ((packedByte>>4)&0x03)+1)
			if len(wordSizes) >= wordCount {
				break
			}
			wordSizes = append(wordSizes, ((packedByte>>2)&0x03)+1)
			if len(wordSizes) >= wordCount {
				break
			}
			wordSizes = append(wordSizes, (packedByte&0x03)+1)
			if len(wordSizes) >= wordCount {
				break
			}
		}
		if len(wordSizes) != wordCount {
			return nil, os.ErrInvalid
		}
		return wordSizes, nil
	}
	if shapeClass == shapeClassFourBit {
		for _, packedByte := range packedWordSizes {
			wordSizes = append(wordSizes, ((packedByte>>4)&0x0F)+1)
			if len(wordSizes) >= wordCount {
				break
			}
			wordSizes = append(wordSizes, (packedByte&0x0F)+1)
			if len(wordSizes) >= wordCount {
				break
			}
		}
		if len(wordSizes) != wordCount {
			return nil, os.ErrInvalid
		}
		return wordSizes, nil
	}
	return nil, os.ErrInvalid
}

func TestAtomicFirstFrequentSelectionPrioritizesTwoLetterSyllables(t *testing.T) {
	productNames := []string{
		"casa mesa taza",
		"barco masa pasa",
		"queso bote lata",
	}

	generatedDictionary, generationError := GenerateFrequentSyllableSlotsWithReserved(
		productNames,
		nil,
		nil,
		FrequentSyllableGeneratorConfig{
			TopFrequentCount: 40,
			TotalSlots:       40,
			Strategy:         "atomic_first",
		},
	)
	if generationError != nil {
		t.Fatalf("unexpected generation error: %v", generationError)
	}

	foundShortSyllable := false
	foundLongSyllable := false
	seenLongBeforeShort := false
	for _, currentSyllable := range generatedDictionary.FrequentSlots {
		syllableLength := len([]rune(currentSyllable))
		if syllableLength <= 2 {
			foundShortSyllable = true
			if foundLongSyllable {
				seenLongBeforeShort = true
				break
			}
			continue
		}
		foundLongSyllable = true
	}

	if !foundShortSyllable || !foundLongSyllable {
		t.Fatalf("test corpus must produce both <=2 and >2 syllables, got frequent=%v", generatedDictionary.FrequentSlots)
	}
	if seenLongBeforeShort {
		t.Fatalf("atomic_first should prioritize <=2 syllables before >2 syllables, got frequent=%v", generatedDictionary.FrequentSlots)
	}
}

func TestExtractedSyllablesReduceSingleLetterUsageVsSingleLetterOnlyDictionary(t *testing.T) {
	cleanedProductNames := []string{
		"batido chocolate puleva botella 1 l",
		"zumo naranja natural botella 1 l",
		"leche desnatada botella 1 l",
		"queso semicurado pieza",
	}

	singleLetterDictionary := buildSingleLetterDictionaryForTest(cleanedProductNames)
	extractedDictionary := buildExtractedDictionaryWithSingleLetterFallbackForTest(cleanedProductNames)

	baselineSingleLetterUsage, baselineTotalUsage, baselineEncodeError := countSingleLetterUsageForDictionary(cleanedProductNames, singleLetterDictionary)
	if baselineEncodeError != nil {
		t.Fatalf("unexpected baseline encode error: %v", baselineEncodeError)
	}
	extractedSingleLetterUsage, extractedTotalUsage, extractedEncodeError := countSingleLetterUsageForDictionary(cleanedProductNames, extractedDictionary)
	if extractedEncodeError != nil {
		t.Fatalf("unexpected extracted encode error: %v", extractedEncodeError)
	}

	if extractedSingleLetterUsage >= baselineSingleLetterUsage {
		t.Fatalf(
			"expected extracted syllables to reduce 1-letter usage baseline=%d extracted=%d baseline_total=%d extracted_total=%d",
			baselineSingleLetterUsage,
			extractedSingleLetterUsage,
			baselineTotalUsage,
			extractedTotalUsage,
		)
	}
	t.Logf(
		"single_letter_usage baseline=%d extracted=%d baseline_total=%d extracted_total=%d reduction=%d",
		baselineSingleLetterUsage,
		extractedSingleLetterUsage,
		baselineTotalUsage,
		extractedTotalUsage,
		baselineSingleLetterUsage-extractedSingleLetterUsage,
	)
}

func buildSingleLetterDictionaryForTest(cleanedProductNames []string) map[string]uint8 {
	singleLetterSet := make(map[string]struct{}, 64)
	for _, cleanedProductName := range cleanedProductNames {
		for _, wordToken := range strings.Fields(cleanedProductName) {
			normalizedWordToken := normalizeToken(wordToken)
			for _, currentRune := range normalizedWordToken {
				singleLetterSet[string(currentRune)] = struct{}{}
			}
		}
	}
	sortedSingleLetters := make([]string, 0, len(singleLetterSet))
	for singleLetter := range singleLetterSet {
		sortedSingleLetters = append(sortedSingleLetters, singleLetter)
	}
	sort.Strings(sortedSingleLetters)

	dictionary := make(map[string]uint8, len(sortedSingleLetters))
	for letterIndex, singleLetter := range sortedSingleLetters {
		dictionary[singleLetter] = uint8(letterIndex + 1)
	}
	return dictionary
}

func buildExtractedDictionaryWithSingleLetterFallbackForTest(cleanedProductNames []string) map[string]uint8 {
	dictionary := buildSingleLetterDictionaryForTest(cleanedProductNames)
	nextID := len(dictionary) + 1

	extractedSet := make(map[string]struct{}, 256)
	for _, cleanedProductName := range cleanedProductNames {
		for _, wordToken := range strings.Fields(cleanedProductName) {
			for _, extractedSyllable := range splitWordIntoSyllables(wordToken) {
				if len([]rune(extractedSyllable)) <= 1 {
					continue
				}
				extractedSet[extractedSyllable] = struct{}{}
			}
		}
	}
	sortedExtracted := make([]string, 0, len(extractedSet))
	for extractedSyllable := range extractedSet {
		sortedExtracted = append(sortedExtracted, extractedSyllable)
	}
	sort.Strings(sortedExtracted)

	for _, extractedSyllable := range sortedExtracted {
		if _, exists := dictionary[extractedSyllable]; exists {
			continue
		}
		dictionary[extractedSyllable] = uint8(nextID)
		nextID++
	}
	return dictionary
}

func countSingleLetterUsageForDictionary(cleanedProductNames []string, dictionary map[string]uint8) (int, int, error) {
	idToToken := make(map[uint8]string, len(dictionary))
	for token, tokenID := range dictionary {
		idToToken[tokenID] = token
	}

	singleLetterUsage := 0
	totalUsage := 0
	for _, cleanedProductName := range cleanedProductNames {
		encodedRecord, encodeError := encodeProductName(cleanedProductName, dictionary)
		if encodeError != nil {
			return 0, 0, encodeError
		}
		for _, tokenID := range encodedRecord.EncodedContent {
			totalUsage++
			token := idToToken[tokenID]
			if len([]rune(token)) == 1 {
				singleLetterUsage++
			}
		}
	}
	return singleLetterUsage, totalUsage, nil
}
