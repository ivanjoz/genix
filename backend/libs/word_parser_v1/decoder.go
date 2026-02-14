package libs

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

type DecodedProductRecord struct {
	Head             uint8
	ProductID        uint16
	HasBrand         bool
	BrandID          uint8
	CategoriesIDs    []uint8
	WordMaskBytes    []uint8
	WordSizes        []uint8
	TextEncoded      []uint8
	TextSyllables    []string
	WordSyllableRows [][]string
	WordsAsSyllables []string
	ApproximateText  string
}

// DecodeProductRecordBytes parses one encoded product record and reconstructs metadata + syllable text.
// Goal: provide a human-readable/debuggable view of a compact binary record.
func DecodeProductRecordBytes(recordBytes []uint8) (*DecodedProductRecord, error) {
	EnsureInitialized()

	if len(recordBytes) < 3 {
		return nil, fmt.Errorf("record too short: %d", len(recordBytes))
	}

	recordHead := recordBytes[0]
	productID := binary.LittleEndian.Uint16(recordBytes[1:3])
	currentOffset := 3

	hasBrand := (recordHead & (1 << 2)) != 0
	usesFourBitWordCounts := (recordHead & (1 << 3)) != 0
	realCategoryCount := int(recordHead&0x03) + 1
	wordMaskByteCount := int((recordHead>>4)&0x03) + 1

	brandID := uint8(0)
	if hasBrand {
		// Brand is physically present only when hasBrand=1 in the head flags.
		if currentOffset >= len(recordBytes) {
			return nil, fmt.Errorf("missing brand byte")
		}
		brandID = recordBytes[currentOffset]
		currentOffset++
	}

	if len(recordBytes[currentOffset:]) < realCategoryCount {
		return nil, fmt.Errorf("missing categories: need=%d have=%d", realCategoryCount, len(recordBytes[currentOffset:]))
	}
	categories := append([]uint8(nil), recordBytes[currentOffset:currentOffset+realCategoryCount]...)
	currentOffset += realCategoryCount

	if len(recordBytes[currentOffset:]) < wordMaskByteCount {
		return nil, fmt.Errorf("missing word mask bytes: need=%d have=%d", wordMaskByteCount, len(recordBytes[currentOffset:]))
	}
	wordMaskBytes := append([]uint8(nil), recordBytes[currentOffset:currentOffset+wordMaskByteCount]...)
	currentOffset += wordMaskByteCount

	// Remaining bytes are the phoneme stream for this record.
	textEncoded := append([]uint8(nil), recordBytes[currentOffset:]...)

	// Word sizes are reconstructed from bit-packed mask and bounded by real stream length.
	// This protects decoding when the mask contains implicit padding.
	decodedWordSizes := unpackWordSizes(wordMaskBytes, usesFourBitWordCounts, len(textEncoded))
	textSyllables := decodeSyllableIDsToStrings(textEncoded)
	wordSyllableRows, wordsAsSyllables := splitSyllablesByWordSizesDetailed(textSyllables, decodedWordSizes)

	return &DecodedProductRecord{
		Head:             recordHead,
		ProductID:        productID,
		HasBrand:         hasBrand,
		BrandID:          brandID,
		CategoriesIDs:    categories,
		WordMaskBytes:    wordMaskBytes,
		WordSizes:        decodedWordSizes,
		TextEncoded:      textEncoded,
		TextSyllables:    textSyllables,
		WordSyllableRows: wordSyllableRows,
		WordsAsSyllables: wordsAsSyllables,
		ApproximateText:  strings.Join(wordsAsSyllables, " "),
	}, nil
}

// LoadProductsFromColumnarIndex reads GIX2 index and reconstructs decoded records.
// Goal: decode columnar storage back into per-record structures used by debug/verification tools.
func LoadProductsFromColumnarIndex(indexPath string, maxRecords int) ([]DecodedProductRecord, error) {
	indexBytes, readFileError := os.ReadFile(indexPath)
	if readFileError != nil {
		return nil, fmt.Errorf("read index file: %w", readFileError)
	}

	const (
		sectionCount   = 9
		minHeaderBytes = 4 + 1 + 4 + (sectionCount * 4)
	)
	if len(indexBytes) < minHeaderBytes {
		return nil, fmt.Errorf("index too short: %d", len(indexBytes))
	}
	if string(indexBytes[0:4]) != columnarMagicHeader {
		return nil, fmt.Errorf("invalid index magic: %q", string(indexBytes[0:4]))
	}
	if indexBytes[4] != columnarVersionV2 {
		return nil, fmt.Errorf("unsupported index version: %d", indexBytes[4])
	}

	productCount := int(binary.LittleEndian.Uint32(indexBytes[5:9]))
	sectionLengths := make([]int, 0, sectionCount)
	lengthOffset := 9
	for currentSection := 0; currentSection < sectionCount; currentSection++ {
		// Section lengths are a compact TOC to allow sequential parsing without separators.
		sectionLength := int(binary.LittleEndian.Uint32(indexBytes[lengthOffset : lengthOffset+4]))
		sectionLengths = append(sectionLengths, sectionLength)
		lengthOffset += 4
	}

	sections := make([][]uint8, 0, sectionCount)
	sectionOffset := minHeaderBytes
	for _, currentSectionLength := range sectionLengths {
		if currentSectionLength < 0 || sectionOffset+currentSectionLength > len(indexBytes) {
			return nil, fmt.Errorf("invalid section length=%d at offset=%d", currentSectionLength, sectionOffset)
		}
		sections = append(sections, indexBytes[sectionOffset:sectionOffset+currentSectionLength])
		sectionOffset += currentSectionLength
	}

	productIDs, decodeIDsError := decodeProductIDDeltasWithRLEVarint(sections[0], productCount)
	if decodeIDsError != nil {
		return nil, decodeIDsError
	}
	maskLengths, decodeMaskLenError := decodeUVarintColumnFixedCount(sections[5], productCount)
	if decodeMaskLenError != nil {
		return nil, decodeMaskLenError
	}
	textLengths, decodeTextLenError := decodeUVarintColumnFixedCount(sections[7], productCount)
	if decodeTextLenError != nil {
		return nil, decodeTextLenError
	}

	if maxRecords <= 0 || maxRecords > productCount {
		maxRecords = productCount
	}

	decodedProducts := make([]DecodedProductRecord, 0, maxRecords)
	maskPayloadOffset := 0
	textPayloadOffset := 0

	for productIndex := 0; productIndex < maxRecords; productIndex++ {
		headByte := sections[1][productIndex]
		brandByte := sections[2][productIndex]
		categoryOneByte := sections[3][productIndex]
		categoryTwoByte := sections[4][productIndex]

		recordMaskLength := int(maskLengths[productIndex])
		recordTextLength := int(textLengths[productIndex])

		if maskPayloadOffset+recordMaskLength > len(sections[6]) {
			return nil, fmt.Errorf("mask payload overflow at record=%d", productIndex)
		}
		if textPayloadOffset+recordTextLength > len(sections[8]) {
			return nil, fmt.Errorf("text payload overflow at record=%d", productIndex)
		}

		recordMaskBytes := append([]uint8(nil), sections[6][maskPayloadOffset:maskPayloadOffset+recordMaskLength]...)
		recordTextBytes := append([]uint8(nil), sections[8][textPayloadOffset:textPayloadOffset+recordTextLength]...)
		maskPayloadOffset += recordMaskLength
		textPayloadOffset += recordTextLength

		// Category count is encoded in head; v2 stores two category columns, so we slice accordingly.
		// Rationale: we keep head as the source of truth and pad missing category slots with zero.
		realCategoryCount := int(headByte&0x03) + 1
		reconstructedCategories := make([]uint8, 0, realCategoryCount)
		if realCategoryCount >= 1 {
			reconstructedCategories = append(reconstructedCategories, categoryOneByte)
		}
		if realCategoryCount >= 2 {
			reconstructedCategories = append(reconstructedCategories, categoryTwoByte)
		}
		for len(reconstructedCategories) < realCategoryCount {
			reconstructedCategories = append(reconstructedCategories, 0)
		}

		reconstructedProductIDBytes := []uint8{
			uint8(productIDs[productIndex]),
			uint8(productIDs[productIndex] >> 8),
		}
		reconstructedRecordBytes := ProductEncoded{
			Head:          headByte,
			ProductID:     reconstructedProductIDBytes,
			BrandID:       brandByte,
			CategoriesIDs: reconstructedCategories,
			WordCount:     recordMaskBytes,
			TextEncoded:   recordTextBytes,
		}.Encode()

		decodedRecord, decodeRecordError := DecodeProductRecordBytes(reconstructedRecordBytes)
		if decodeRecordError != nil {
			return nil, fmt.Errorf("decode record %d: %w", productIndex, decodeRecordError)
		}
		decodedProducts = append(decodedProducts, *decodedRecord)
	}

	return decodedProducts, nil
}

// DebugPrintFirstProductsFromColumnarIndex prints human-readable info for first N records.
// Output contract: exactly 2 lines per record (metadata line + approx text line).
func DebugPrintFirstProductsFromColumnarIndex(indexPath string, recordsToPrint int) error {
	decodedRecords, loadError := LoadProductsFromColumnarIndex(indexPath, recordsToPrint)
	if loadError != nil {
		return loadError
	}

	for index, currentRecord := range decodedRecords {
		fmt.Printf(
			"record=%d product_id=%d brand=%d categories=%v word_sizes=%v word_syllables=%v\n",
			index+1,
			currentRecord.ProductID,
			currentRecord.BrandID,
			currentRecord.CategoriesIDs,
			currentRecord.WordSizes,
			currentRecord.WordSyllableRows,
		)
		fmt.Printf("approx_text=%q\n", currentRecord.ApproximateText)
	}
	return nil
}

func decodeSyllableIDsToStrings(syllableIDs []uint8) []string {
	decodedSyllables := make([]string, 0, len(syllableIDs))
	for _, currentSyllableID := range syllableIDs {
		syllableOptions, exists := silabas[currentSyllableID]
		if !exists || len(syllableOptions) == 0 {
			// Keep unknown IDs visible to ease dictionary/version mismatch debugging.
			decodedSyllables = append(decodedSyllables, fmt.Sprintf("[%d]", currentSyllableID))
			continue
		}
		decodedSyllables = append(decodedSyllables, syllableOptions[0])
	}
	return decodedSyllables
}

func splitSyllablesByWordSizes(syllables []string, wordSizes []uint8) []string {
	_, wordsAsSyllables := splitSyllablesByWordSizesDetailed(syllables, wordSizes)
	return wordsAsSyllables
}

func splitSyllablesByWordSizesDetailed(syllables []string, wordSizes []uint8) ([][]string, []string) {
	wordSyllableRows := make([][]string, 0, len(wordSizes))
	words := make([]string, 0, len(wordSizes))
	currentOffset := 0
	for _, currentWordSize := range wordSizes {
		nextOffset := currentOffset + int(currentWordSize)
		if nextOffset > len(syllables) {
			nextOffset = len(syllables)
		}
		currentWordSyllables := append([]string(nil), syllables[currentOffset:nextOffset]...)
		wordSyllableRows = append(wordSyllableRows, currentWordSyllables)
		words = append(words, strings.Join(currentWordSyllables, ""))
		currentOffset = nextOffset
		if currentOffset >= len(syllables) {
			break
		}
	}

	// If word mask under-reports, keep remaining syllables as a trailing word for debugging.
	// Goal: never silently lose phonemes while auditing encoding correctness.
	if currentOffset < len(syllables) {
		trailingSyllables := append([]string(nil), syllables[currentOffset:]...)
		wordSyllableRows = append(wordSyllableRows, trailingSyllables)
		words = append(words, strings.Join(trailingSyllables, ""))
	}
	return wordSyllableRows, words
}

func unpackWordSizes(wordMaskBytes []uint8, usesFourBitWordCounts bool, totalTextLength int) []uint8 {
	candidateSizes := make([]uint8, 0, len(wordMaskBytes)*4)
	if usesFourBitWordCounts {
		for _, currentMaskByte := range wordMaskBytes {
			highWordSize := ((currentMaskByte >> 4) & 0x0F) + 1
			lowWordSize := (currentMaskByte & 0x0F) + 1
			candidateSizes = append(candidateSizes, highWordSize, lowWordSize)
		}
	} else {
		for _, currentMaskByte := range wordMaskBytes {
			candidateSizes = append(
				candidateSizes,
				((currentMaskByte>>6)&0x03)+1,
				((currentMaskByte>>4)&0x03)+1,
				((currentMaskByte>>2)&0x03)+1,
				(currentMaskByte&0x03)+1,
			)
		}
	}

	// Because size=1 and padding share encoded 0, consume only sizes that fit exact payload bounds.
	// Rationale: in this format, trailing padded slots are indistinguishable from real one-syllable words.
	realWordSizes := make([]uint8, 0, len(candidateSizes))
	consumedSyllables := 0
	for _, candidateSize := range candidateSizes {
		if consumedSyllables+int(candidateSize) > totalTextLength {
			break
		}
		realWordSizes = append(realWordSizes, candidateSize)
		consumedSyllables += int(candidateSize)
		if consumedSyllables == totalTextLength {
			break
		}
	}
	return realWordSizes
}

func decodeUVarintColumnFixedCount(encodedColumn []uint8, expectedValues int) ([]uint64, error) {
	decodedValues := make([]uint64, 0, expectedValues)
	currentOffset := 0
	for len(decodedValues) < expectedValues {
		if currentOffset >= len(encodedColumn) {
			return nil, fmt.Errorf("unexpected end of uvarint column: got=%d want=%d", len(decodedValues), expectedValues)
		}
		currentValue, consumedBytes := binary.Uvarint(encodedColumn[currentOffset:])
		if consumedBytes <= 0 {
			return nil, fmt.Errorf("invalid uvarint at offset=%d", currentOffset)
		}
		decodedValues = append(decodedValues, currentValue)
		currentOffset += consumedBytes
	}
	return decodedValues, nil
}

func decodeProductIDDeltasWithRLEVarint(encodedColumn []uint8, expectedIDs int) ([]uint16, error) {
	decodedIDs := make([]uint16, 0, expectedIDs)
	currentOffset := 0
	previousID := uint16(0)

	for len(decodedIDs) < expectedIDs {
		if currentOffset >= len(encodedColumn) {
			return nil, fmt.Errorf("unexpected end of product id delta column: got=%d want=%d", len(decodedIDs), expectedIDs)
		}
		controlByte := encodedColumn[currentOffset]
		currentOffset++

		currentValue, consumedBytes := binary.Uvarint(encodedColumn[currentOffset:])
		if consumedBytes <= 0 {
			return nil, fmt.Errorf("invalid product id delta varint at offset=%d", currentOffset)
		}
		currentOffset += consumedBytes

		switch controlByte {
		case 0:
			// Compact path: repeat "+1" deltas runLength times.
			runLength := int(currentValue)
			for step := 0; step < runLength && len(decodedIDs) < expectedIDs; step++ {
				previousID++
				decodedIDs = append(decodedIDs, previousID)
			}
		case 1:
			// Fallback path: apply explicit delta value.
			if currentValue > 65535 {
				return nil, fmt.Errorf("delta out of uint16 range: %d", currentValue)
			}
			previousID = previousID + uint16(currentValue)
			decodedIDs = append(decodedIDs, previousID)
		default:
			return nil, fmt.Errorf("unknown product id delta control byte: %d", controlByte)
		}
	}

	return decodedIDs, nil
}
