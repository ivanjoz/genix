package word_parser_v2

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
)

// DecodedIndexRecord is one reconstructed product phrase from the binary index.
type DecodedIndexRecord struct {
	RecordIndex   int
	ShapeClass    uint8
	ShapeLocalID  int
	WordSizes     []uint8
	DecodedPhrase string
}

// DecodedIndexSummary exposes section sizes and counts parsed from the index header/body.
type DecodedIndexSummary struct {
	RecordCount           int
	DictionaryCount       int
	DictionaryBytes       int
	ShapeCountClass0      int
	ShapeCountClass1      int
	ShapeTableClass0Bytes int
	ShapeTableClass1Bytes int
	ShapeUsageBytes       int
	ShapeDictionaryBytes  int
	ContentBytes          int
	ShapeDeltaClass0      bool
	ShapeDeltaClass1      bool
	ShapeDelta8Count      int
	ShapeDelta16Count     int
	ShapeDelta24Count     int
	ShapeDelta8Bits       int
	ShapeDelta16Bits      int
	ShapeDelta24Bits      int
	ShapeProgress5Pct     string
}

// DecodeBinaryIndexFile loads productos.idx and reconstructs all records in stored order.
func DecodeBinaryIndexFile(indexPath string) ([]DecodedIndexRecord, DecodedIndexSummary, error) {
	indexBytes, readError := os.ReadFile(indexPath)
	if readError != nil {
		return nil, DecodedIndexSummary{}, fmt.Errorf("read index file: %w", readError)
	}
	return decodeBinaryIndexBytes(indexBytes)
}

// DecodeBinaryIndexDictionary returns dictionary tokens in final slot order as stored in the index.
func DecodeBinaryIndexDictionary(indexPath string) ([]string, error) {
	indexBytes, readError := os.ReadFile(indexPath)
	if readError != nil {
		return nil, fmt.Errorf("read index file: %w", readError)
	}
	return decodeBinaryIndexDictionaryBytes(indexBytes)
}

// SampleRandomDecodedRecords returns up to sampleCount random records. If seed is zero, current time is used.
func SampleRandomDecodedRecords(records []DecodedIndexRecord, sampleCount int, seed int64) []DecodedIndexRecord {
	if sampleCount <= 0 || len(records) == 0 {
		return nil
	}
	if sampleCount > len(records) {
		sampleCount = len(records)
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	randomGenerator := rand.New(rand.NewSource(seed))
	randomPermutation := randomGenerator.Perm(len(records))
	selectedIndexes := randomPermutation[:sampleCount]
	sort.Ints(selectedIndexes)

	sampledRecords := make([]DecodedIndexRecord, 0, sampleCount)
	for _, selectedIndex := range selectedIndexes {
		sampledRecords = append(sampledRecords, records[selectedIndex])
	}
	return sampledRecords
}

func decodeBinaryIndexBytes(indexBytes []byte) ([]DecodedIndexRecord, DecodedIndexSummary, error) {
	if len(indexBytes) < headerSizeV3 {
		return nil, DecodedIndexSummary{}, fmt.Errorf("index too small: %d", len(indexBytes))
	}
	if string(indexBytes[:len(binaryIndexMagicV3)]) != binaryIndexMagicV3 {
		return nil, DecodedIndexSummary{}, fmt.Errorf("invalid magic: %q", string(indexBytes[:len(binaryIndexMagicV3)]))
	}
	if indexBytes[len(binaryIndexMagicV3)] != binaryIndexVersion {
		return nil, DecodedIndexSummary{}, fmt.Errorf("unsupported version=%d expected=%d", indexBytes[len(binaryIndexMagicV3)], binaryIndexVersion)
	}

	headerFlags := indexBytes[9]
	recordCount := int(binary.LittleEndian.Uint32(indexBytes[10:14]))
	dictionaryCount := int(indexBytes[14])
	shapeCountClass0 := int(indexBytes[15]) + int(binary.LittleEndian.Uint16(indexBytes[17:19]))
	shapeCountClass1 := int(indexBytes[16]) + int(binary.LittleEndian.Uint16(indexBytes[19:21]))
	dictionaryBytes := int(binary.LittleEndian.Uint32(indexBytes[21:25]))
	shapeTableClass0Bytes := int(binary.LittleEndian.Uint32(indexBytes[25:29]))
	shapeTableClass1Bytes := int(binary.LittleEndian.Uint32(indexBytes[29:33]))

	cursorIndex := headerSizeV3
	if cursorIndex+dictionaryBytes+shapeTableClass0Bytes+shapeTableClass1Bytes > len(indexBytes) {
		return nil, DecodedIndexSummary{}, fmt.Errorf("section sizes exceed file bounds")
	}

	dictionarySection := indexBytes[cursorIndex : cursorIndex+dictionaryBytes]
	cursorIndex += dictionaryBytes
	shapeTableClass0Section := indexBytes[cursorIndex : cursorIndex+shapeTableClass0Bytes]
	cursorIndex += shapeTableClass0Bytes
	shapeTableClass1Section := indexBytes[cursorIndex : cursorIndex+shapeTableClass1Bytes]
	cursorIndex += shapeTableClass1Bytes

	dictionaryTokens, decodeDictionaryError := decodeDictionarySection(dictionarySection, dictionaryCount, headerFlags&headerFlagDictionaryDelta != 0)
	if decodeDictionaryError != nil {
		return nil, DecodedIndexSummary{}, decodeDictionaryError
	}

	class0Shapes, decodeClass0Error := decodeShapeTableSection(shapeTableClass0Section, shapeCountClass0, shapeClassTwoBit, headerFlags&headerFlagShapeDeltaC0 != 0)
	if decodeClass0Error != nil {
		return nil, DecodedIndexSummary{}, decodeClass0Error
	}
	class1Shapes, decodeClass1Error := decodeShapeTableSection(shapeTableClass1Section, shapeCountClass1, shapeClassFourBit, headerFlags&headerFlagShapeDeltaC1 != 0)
	if decodeClass1Error != nil {
		return nil, DecodedIndexSummary{}, decodeClass1Error
	}

	if headerFlags&headerFlagShapeInlineC1 != 0 {
		shapeStreamByteLength, nextCursorIndex, readStreamLengthError := readUvarintAt(indexBytes, cursorIndex)
		if readStreamLengthError != nil {
			return nil, DecodedIndexSummary{}, fmt.Errorf("read unified shape stream length: %w", readStreamLengthError)
		}
		cursorIndex = nextCursorIndex
		if shapeStreamByteLength > uint64(len(indexBytes)-cursorIndex) {
			return nil, DecodedIndexSummary{}, fmt.Errorf("unified shape stream length out of bounds stream_len=%d available=%d", shapeStreamByteLength, len(indexBytes)-cursorIndex)
		}
		shapeStreamSection := indexBytes[cursorIndex : cursorIndex+int(shapeStreamByteLength)]
		cursorIndex += int(shapeStreamByteLength)

		recordWordSizes, shapeDeltaStats, decodeShapeStreamError := decodeRecordShapeWordSizesFromDeltaStream(shapeStreamSection, recordCount)
		if decodeShapeStreamError != nil {
			return nil, DecodedIndexSummary{}, decodeShapeStreamError
		}

		shapeUsageBytes := cursorIndex - (headerSizeV3 + dictionaryBytes + shapeTableClass0Bytes + shapeTableClass1Bytes)
		contentSection := indexBytes[cursorIndex:]
		expectedContentBytes, expectedContentBytesError := computeExpectedContentBytesFromShapeStream(recordWordSizes)
		if expectedContentBytesError != nil {
			return nil, DecodedIndexSummary{}, expectedContentBytesError
		}
		if expectedContentBytes != len(contentSection) {
			return nil, DecodedIndexSummary{}, fmt.Errorf("content size mismatch expected=%d got=%d", expectedContentBytes, len(contentSection))
		}

		decodedRecords, decodeRecordsError := decodeContentRecordsFromShapeStream(contentSection, dictionaryTokens, recordWordSizes)
		if decodeRecordsError != nil {
			return nil, DecodedIndexSummary{}, decodeRecordsError
		}
		if len(decodedRecords) != recordCount {
			return nil, DecodedIndexSummary{}, fmt.Errorf("record count mismatch expected=%d got=%d", recordCount, len(decodedRecords))
		}

		summary := DecodedIndexSummary{
			RecordCount:           recordCount,
			DictionaryCount:       dictionaryCount,
			DictionaryBytes:       dictionaryBytes,
			ShapeCountClass0:      shapeCountClass0,
			ShapeCountClass1:      shapeCountClass1,
			ShapeTableClass0Bytes: shapeTableClass0Bytes,
			ShapeTableClass1Bytes: shapeTableClass1Bytes,
			ShapeUsageBytes:       shapeUsageBytes,
			ShapeDictionaryBytes:  shapeTableClass0Bytes + shapeTableClass1Bytes,
			ContentBytes:          len(contentSection),
			ShapeDeltaClass0:      headerFlags&headerFlagShapeDeltaC0 != 0,
			ShapeDeltaClass1:      headerFlags&headerFlagShapeDeltaC1 != 0,
			ShapeDelta8Count:      shapeDeltaStats.Delta8Count,
			ShapeDelta16Count:     shapeDeltaStats.Delta16Count,
			ShapeDelta24Count:     shapeDeltaStats.Delta24Count,
			ShapeDelta8Bits:       shapeDeltaStats.Delta8Bits,
			ShapeDelta16Bits:      shapeDeltaStats.Delta16Bits,
			ShapeDelta24Bits:      shapeDeltaStats.Delta24Bits,
			ShapeProgress5Pct:     shapeDeltaStats.Progress5PctLine,
		}
		return decodedRecords, summary, nil
	}

	class0UsageCounts := make([]uint64, shapeCountClass0)
	for usageIndex := range class0UsageCounts {
		shapeUsageCount, nextCursorIndex, readUsageError := readUvarintAt(indexBytes, cursorIndex)
		if readUsageError != nil {
			return nil, DecodedIndexSummary{}, fmt.Errorf("read class0 usage index=%d: %w", usageIndex, readUsageError)
		}
		class0UsageCounts[usageIndex] = shapeUsageCount
		cursorIndex = nextCursorIndex
	}

	class1UsageCounts := make([]uint64, shapeCountClass1)
	for usageIndex := range class1UsageCounts {
		shapeUsageCount, nextCursorIndex, readUsageError := readUvarintAt(indexBytes, cursorIndex)
		if readUsageError != nil {
			return nil, DecodedIndexSummary{}, fmt.Errorf("read class1 usage index=%d: %w", usageIndex, readUsageError)
		}
		class1UsageCounts[usageIndex] = shapeUsageCount
		cursorIndex = nextCursorIndex
	}

	shapeUsageBytes := cursorIndex - (headerSizeV3 + dictionaryBytes + shapeTableClass0Bytes + shapeTableClass1Bytes)

	contentSection := indexBytes[cursorIndex:]
	expectedContentBytes, computeContentBytesError := computeExpectedContentBytes(class0Shapes, class0UsageCounts, class1Shapes, class1UsageCounts)
	if computeContentBytesError != nil {
		return nil, DecodedIndexSummary{}, computeContentBytesError
	}
	if expectedContentBytes != len(contentSection) {
		return nil, DecodedIndexSummary{}, fmt.Errorf("content size mismatch expected=%d got=%d", expectedContentBytes, len(contentSection))
	}

	decodedRecords, decodeRecordsError := decodeContentRecords(contentSection, dictionaryTokens, class0Shapes, class0UsageCounts, class1Shapes, class1UsageCounts)
	if decodeRecordsError != nil {
		return nil, DecodedIndexSummary{}, decodeRecordsError
	}
	if len(decodedRecords) != recordCount {
		return nil, DecodedIndexSummary{}, fmt.Errorf("record count mismatch expected=%d got=%d", recordCount, len(decodedRecords))
	}

	summary := DecodedIndexSummary{
		RecordCount:           recordCount,
		DictionaryCount:       dictionaryCount,
		DictionaryBytes:       dictionaryBytes,
		ShapeCountClass0:      shapeCountClass0,
		ShapeCountClass1:      shapeCountClass1,
		ShapeTableClass0Bytes: shapeTableClass0Bytes,
		ShapeTableClass1Bytes: shapeTableClass1Bytes,
		ShapeUsageBytes:       shapeUsageBytes,
		ShapeDictionaryBytes:  shapeTableClass0Bytes + shapeTableClass1Bytes,
		ContentBytes:          len(contentSection),
		ShapeDeltaClass0:      headerFlags&headerFlagShapeDeltaC0 != 0,
		ShapeDeltaClass1:      headerFlags&headerFlagShapeDeltaC1 != 0,
	}
	return decodedRecords, summary, nil
}

func decodeBinaryIndexDictionaryBytes(indexBytes []byte) ([]string, error) {
	if len(indexBytes) < headerSizeV3 {
		return nil, fmt.Errorf("index too small: %d", len(indexBytes))
	}
	if string(indexBytes[:len(binaryIndexMagicV3)]) != binaryIndexMagicV3 {
		return nil, fmt.Errorf("invalid magic: %q", string(indexBytes[:len(binaryIndexMagicV3)]))
	}
	if indexBytes[len(binaryIndexMagicV3)] != binaryIndexVersion {
		return nil, fmt.Errorf("unsupported version=%d expected=%d", indexBytes[len(binaryIndexMagicV3)], binaryIndexVersion)
	}

	headerFlags := indexBytes[9]
	dictionaryCount := int(indexBytes[14])
	dictionaryBytes := int(binary.LittleEndian.Uint32(indexBytes[21:25]))
	if dictionaryCount <= 0 || dictionaryCount > 255 {
		return nil, fmt.Errorf("invalid dictionary_count=%d", dictionaryCount)
	}
	if headerSizeV3+dictionaryBytes > len(indexBytes) {
		return nil, fmt.Errorf("dictionary section exceeds file bounds")
	}

	dictionarySection := indexBytes[headerSizeV3 : headerSizeV3+dictionaryBytes]
	return decodeDictionarySection(dictionarySection, dictionaryCount, headerFlags&headerFlagDictionaryDelta != 0)
}

func decodeDictionarySection(section []byte, dictionaryCount int, isDeltaMode bool) ([]string, error) {
	tokens := make([]string, 0, dictionaryCount)
	cursorIndex := 0

	if !isDeltaMode {
		for tokenIndex := 0; tokenIndex < dictionaryCount; tokenIndex++ {
			if cursorIndex >= len(section) {
				return nil, fmt.Errorf("raw dictionary truncated at token=%d", tokenIndex)
			}
			tokenLength := int(section[cursorIndex])
			cursorIndex++
			if tokenLength <= 0 || cursorIndex+tokenLength > len(section) {
				return nil, fmt.Errorf("raw dictionary invalid token length at token=%d len=%d", tokenIndex, tokenLength)
			}
			tokens = append(tokens, string(section[cursorIndex:cursorIndex+tokenLength]))
			cursorIndex += tokenLength
		}
		if cursorIndex != len(section) {
			return nil, fmt.Errorf("raw dictionary trailing bytes=%d", len(section)-cursorIndex)
		}
		return tokens, nil
	}

	previousToken := []byte{}
	for tokenIndex := 0; tokenIndex < dictionaryCount; tokenIndex++ {
		if cursorIndex+2 > len(section) {
			return nil, fmt.Errorf("delta dictionary truncated at token=%d", tokenIndex)
		}
		prefixLength := int(section[cursorIndex])
		suffixLength := int(section[cursorIndex+1])
		cursorIndex += 2
		if prefixLength > len(previousToken) {
			return nil, fmt.Errorf("delta dictionary prefix overflow token=%d", tokenIndex)
		}
		if suffixLength <= 0 || cursorIndex+suffixLength > len(section) {
			return nil, fmt.Errorf("delta dictionary invalid suffix token=%d len=%d", tokenIndex, suffixLength)
		}
		currentTokenBytes := append([]byte{}, previousToken[:prefixLength]...)
		currentTokenBytes = append(currentTokenBytes, section[cursorIndex:cursorIndex+suffixLength]...)
		cursorIndex += suffixLength
		tokens = append(tokens, string(currentTokenBytes))
		previousToken = currentTokenBytes
	}
	if cursorIndex != len(section) {
		return nil, fmt.Errorf("delta dictionary trailing bytes=%d", len(section)-cursorIndex)
	}
	return tokens, nil
}

func decodeShapeTableSection(section []byte, shapeCount int, shapeClass uint8, isDeltaMode bool) ([][]uint8, error) {
	shapes := make([][]uint8, 0, shapeCount)
	cursorIndex := 0
	var previousShape []uint8

	for shapeIndex := 0; shapeIndex < shapeCount; shapeIndex++ {
		var currentShape []uint8
		if !isDeltaMode {
			wordSizes, nextCursorIndex, decodeError := decodeRawShapeRowAt(section, cursorIndex, shapeClass)
			if decodeError != nil {
				return nil, fmt.Errorf("decode raw shape row index=%d: %w", shapeIndex, decodeError)
			}
			currentShape = wordSizes
			cursorIndex = nextCursorIndex
		} else {
			if cursorIndex >= len(section) {
				return nil, fmt.Errorf("delta shape table truncated at index=%d", shapeIndex)
			}
			decorator := section[cursorIndex]
			cursorIndex++
			switch decorator {
			case shapeDeltaDecoratorRaw:
				wordSizes, nextCursorIndex, decodeError := decodeRawShapeRowAt(section, cursorIndex, shapeClass)
				if decodeError != nil {
					return nil, fmt.Errorf("decode delta/raw row index=%d: %w", shapeIndex, decodeError)
				}
				currentShape = wordSizes
				cursorIndex = nextCursorIndex
			case shapeDeltaDecoratorSmallMut2Bit:
				if len(previousShape) == 0 {
					return nil, fmt.Errorf("small mutation row without previous shape index=%d", shapeIndex)
				}
				mutationByteLen, nextCursorIndex, readLenError := readUvarintFromBytes(section, cursorIndex)
				if readLenError != nil {
					return nil, fmt.Errorf("read mutation byte len index=%d: %w", shapeIndex, readLenError)
				}
				cursorIndex = nextCursorIndex
				if cursorIndex+int(mutationByteLen) > len(section) {
					return nil, fmt.Errorf("mutation bytes out of bounds index=%d", shapeIndex)
				}
				mutationCodes := unpackTwoBitValuesAtCount(section[cursorIndex:cursorIndex+int(mutationByteLen)], len(previousShape))
				cursorIndex += int(mutationByteLen)
				currentShape = make([]uint8, len(previousShape))
				for wordIndex, mutationCode := range mutationCodes {
					baseValue := int(previousShape[wordIndex])
					var deltaValue int
					switch mutationCode {
					case 0:
						deltaValue = -1
					case 1:
						deltaValue = 0
					case 2:
						deltaValue = 1
					case 3:
						deltaValue = 2
					default:
						return nil, fmt.Errorf("invalid mutation code=%d index=%d", mutationCode, shapeIndex)
					}
					newValue := baseValue + deltaValue
					if !isValidShapeWordSize(uint8(newValue), shapeClass) {
						return nil, fmt.Errorf("invalid mutated word size=%d index=%d", newValue, shapeIndex)
					}
					currentShape[wordIndex] = uint8(newValue)
				}
			case shapeDeltaDecoratorPrefixAppend:
				if len(previousShape) == 0 {
					return nil, fmt.Errorf("prefix append row without previous shape index=%d", shapeIndex)
				}
				appendWordCount, nextCursorIndex, readAppendCountError := readUvarintFromBytes(section, cursorIndex)
				if readAppendCountError != nil {
					return nil, fmt.Errorf("read prefix append count index=%d: %w", shapeIndex, readAppendCountError)
				}
				cursorIndex = nextCursorIndex
				appendPackedLen, nextPackedCursorIndex, readPackedLenError := readUvarintFromBytes(section, cursorIndex)
				if readPackedLenError != nil {
					return nil, fmt.Errorf("read prefix append packed len index=%d: %w", shapeIndex, readPackedLenError)
				}
				cursorIndex = nextPackedCursorIndex
				if cursorIndex+int(appendPackedLen) > len(section) {
					return nil, fmt.Errorf("prefix append packed bytes out of bounds index=%d", shapeIndex)
				}
				appendWordSizes, unpackError := unpackShapeWordSizes(section[cursorIndex:cursorIndex+int(appendPackedLen)], int(appendWordCount), shapeClass)
				if unpackError != nil {
					return nil, fmt.Errorf("unpack prefix append index=%d: %w", shapeIndex, unpackError)
				}
				cursorIndex += int(appendPackedLen)
				currentShape = append(append([]uint8{}, previousShape...), appendWordSizes...)
			case shapeDeltaDecoratorTrimAppend:
				if len(previousShape) == 0 {
					return nil, fmt.Errorf("trim append row without previous shape index=%d", shapeIndex)
				}
				trimCount, nextCursorIndex, readTrimCountError := readUvarintFromBytes(section, cursorIndex)
				if readTrimCountError != nil {
					return nil, fmt.Errorf("read trim count index=%d: %w", shapeIndex, readTrimCountError)
				}
				cursorIndex = nextCursorIndex
				appendWordCount, nextAppendCountCursorIndex, readAppendCountError := readUvarintFromBytes(section, cursorIndex)
				if readAppendCountError != nil {
					return nil, fmt.Errorf("read trim append count index=%d: %w", shapeIndex, readAppendCountError)
				}
				cursorIndex = nextAppendCountCursorIndex
				appendPackedLen, nextPackedCursorIndex, readPackedLenError := readUvarintFromBytes(section, cursorIndex)
				if readPackedLenError != nil {
					return nil, fmt.Errorf("read trim append packed len index=%d: %w", shapeIndex, readPackedLenError)
				}
				cursorIndex = nextPackedCursorIndex
				if cursorIndex+int(appendPackedLen) > len(section) {
					return nil, fmt.Errorf("trim append packed bytes out of bounds index=%d", shapeIndex)
				}
				if int(trimCount) > len(previousShape) {
					return nil, fmt.Errorf("trim count too large index=%d trim=%d prev=%d", shapeIndex, trimCount, len(previousShape))
				}
				appendWordSizes, unpackError := unpackShapeWordSizes(section[cursorIndex:cursorIndex+int(appendPackedLen)], int(appendWordCount), shapeClass)
				if unpackError != nil {
					return nil, fmt.Errorf("unpack trim append index=%d: %w", shapeIndex, unpackError)
				}
				cursorIndex += int(appendPackedLen)
				currentShape = append(append([]uint8{}, previousShape[:len(previousShape)-int(trimCount)]...), appendWordSizes...)
			default:
				return nil, fmt.Errorf("unknown shape delta decorator=%d index=%d", decorator, shapeIndex)
			}
		}

		if len(currentShape) == 0 {
			return nil, fmt.Errorf("decoded empty shape at index=%d", shapeIndex)
		}
		for _, wordSize := range currentShape {
			if !isValidShapeWordSize(wordSize, shapeClass) {
				return nil, fmt.Errorf("decoded invalid word size=%d index=%d class=%d", wordSize, shapeIndex, shapeClass)
			}
		}
		shapes = append(shapes, currentShape)
		previousShape = currentShape
	}
	if cursorIndex != len(section) {
		return nil, fmt.Errorf("shape table trailing bytes=%d", len(section)-cursorIndex)
	}
	return shapes, nil
}

func decodeRawShapeRowAt(source []byte, offset int, shapeClass uint8) ([]uint8, int, error) {
	wordCount, nextCursorIndex, readWordCountError := readUvarintFromBytes(source, offset)
	if readWordCountError != nil {
		return nil, offset, readWordCountError
	}
	if wordCount == 0 {
		return nil, offset, fmt.Errorf("word count cannot be zero")
	}
	packedLength, nextPackedCursorIndex, readPackedLengthError := readUvarintFromBytes(source, nextCursorIndex)
	if readPackedLengthError != nil {
		return nil, offset, readPackedLengthError
	}
	cursorIndex := nextPackedCursorIndex
	if cursorIndex+int(packedLength) > len(source) {
		return nil, offset, fmt.Errorf("packed shape bytes out of bounds")
	}
	packedWordSizes := source[cursorIndex : cursorIndex+int(packedLength)]
	wordSizes, unpackError := unpackShapeWordSizes(packedWordSizes, int(wordCount), shapeClass)
	if unpackError != nil {
		return nil, offset, unpackError
	}
	return wordSizes, cursorIndex + int(packedLength), nil
}

func unpackShapeWordSizes(packedWordSizes []byte, wordCount int, shapeClass uint8) ([]uint8, error) {
	if wordCount <= 0 {
		return nil, fmt.Errorf("word count must be > 0")
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
	} else if shapeClass == shapeClassFourBit {
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
	} else {
		return nil, fmt.Errorf("unknown shape class=%d", shapeClass)
	}

	if len(wordSizes) != wordCount {
		return nil, fmt.Errorf("decoded word count mismatch expected=%d got=%d", wordCount, len(wordSizes))
	}
	return wordSizes, nil
}

func isValidShapeWordSize(wordSize uint8, shapeClass uint8) bool {
	if wordSize < 1 {
		return false
	}
	if shapeClass == shapeClassTwoBit {
		return wordSize <= 4
	}
	if shapeClass == shapeClassFourBit {
		return wordSize <= 16
	}
	return false
}

func unpackTwoBitValuesAtCount(packedValues []byte, expectedCount int) []uint8 {
	operationCodes := make([]uint8, 0, expectedCount)
	for _, packedByte := range packedValues {
		operationCodes = append(operationCodes, (packedByte>>6)&0x03)
		if len(operationCodes) >= expectedCount {
			break
		}
		operationCodes = append(operationCodes, (packedByte>>4)&0x03)
		if len(operationCodes) >= expectedCount {
			break
		}
		operationCodes = append(operationCodes, (packedByte>>2)&0x03)
		if len(operationCodes) >= expectedCount {
			break
		}
		operationCodes = append(operationCodes, packedByte&0x03)
		if len(operationCodes) >= expectedCount {
			break
		}
	}
	return operationCodes
}

func computeExpectedContentBytes(class0Shapes [][]uint8, class0Usage []uint64, class1Shapes [][]uint8, class1Usage []uint64) (int, error) {
	totalContentBytes := uint64(0)
	computeClass := func(shapes [][]uint8, usage []uint64) error {
		for shapeIndex := range shapes {
			totalSyllablesPerRecord := 0
			for _, wordSize := range shapes[shapeIndex] {
				totalSyllablesPerRecord += int(wordSize)
			}
			totalContentBytes += uint64(totalSyllablesPerRecord) * usage[shapeIndex]
		}
		return nil
	}
	if computeError := computeClass(class0Shapes, class0Usage); computeError != nil {
		return 0, computeError
	}
	if computeError := computeClass(class1Shapes, class1Usage); computeError != nil {
		return 0, computeError
	}
	if totalContentBytes > uint64(^uint(0)>>1) {
		return 0, fmt.Errorf("content bytes overflow")
	}
	return int(totalContentBytes), nil
}

func decodeContentRecords(contentSection []byte, dictionaryTokens []string, class0Shapes [][]uint8, class0Usage []uint64, class1Shapes [][]uint8, class1Usage []uint64) ([]DecodedIndexRecord, error) {
	records := make([]DecodedIndexRecord, 0)
	cursorIndex := 0
	recordIndex := 0

	decodeClass := func(shapeClass uint8, shapes [][]uint8, usage []uint64) error {
		for shapeIndex := range shapes {
			wordSizes := shapes[shapeIndex]
			totalSyllablesPerRecord := 0
			for _, wordSize := range wordSizes {
				totalSyllablesPerRecord += int(wordSize)
			}
			for runIndex := uint64(0); runIndex < usage[shapeIndex]; runIndex++ {
				if cursorIndex+totalSyllablesPerRecord > len(contentSection) {
					return fmt.Errorf("content truncated at record=%d", recordIndex)
				}
				recordSyllableIDs := contentSection[cursorIndex : cursorIndex+totalSyllablesPerRecord]
				cursorIndex += totalSyllablesPerRecord

				wordTexts := make([]string, 0, len(wordSizes))
				syllableCursor := 0
				for _, wordSize := range wordSizes {
					wordSyllableCount := int(wordSize)
					var wordBuilder strings.Builder
					for syllableIndex := 0; syllableIndex < wordSyllableCount; syllableIndex++ {
						syllableID := int(recordSyllableIDs[syllableCursor])
						syllableCursor++
						if syllableID <= 0 || syllableID > len(dictionaryTokens) {
							return fmt.Errorf("invalid syllable id=%d at record=%d", syllableID, recordIndex)
						}
						wordBuilder.WriteString(dictionaryTokens[syllableID-1])
					}
					wordTexts = append(wordTexts, wordBuilder.String())
				}

				records = append(records, DecodedIndexRecord{
					RecordIndex:   recordIndex,
					ShapeClass:    shapeClass,
					ShapeLocalID:  shapeIndex + 1,
					WordSizes:     append([]uint8{}, wordSizes...),
					DecodedPhrase: strings.Join(wordTexts, " "),
				})
				recordIndex++
			}
		}
		return nil
	}

	if decodeError := decodeClass(shapeClassTwoBit, class0Shapes, class0Usage); decodeError != nil {
		return nil, decodeError
	}
	if decodeError := decodeClass(shapeClassFourBit, class1Shapes, class1Usage); decodeError != nil {
		return nil, decodeError
	}
	if cursorIndex != len(contentSection) {
		return nil, fmt.Errorf("content trailing bytes=%d", len(contentSection)-cursorIndex)
	}
	return records, nil
}

func readUvarintAt(source []byte, offset int) (uint64, int, error) {
	if offset >= len(source) {
		return 0, offset, fmt.Errorf("offset out of range")
	}
	readValue, consumedBytes := binary.Uvarint(source[offset:])
	if consumedBytes <= 0 {
		return 0, offset, fmt.Errorf("invalid uvarint")
	}
	return readValue, offset + consumedBytes, nil
}

func readUvarintFromBytes(source []byte, offset int) (uint64, int, error) {
	if offset >= len(source) {
		return 0, offset, fmt.Errorf("offset out of range")
	}
	readValue, consumedBytes := binary.Uvarint(source[offset:])
	if consumedBytes <= 0 {
		return 0, offset, fmt.Errorf("invalid uvarint")
	}
	return readValue, offset + consumedBytes, nil
}

type decodedShapeDeltaStats struct {
	Delta8Count     int
	Delta16Count    int
	Delta24Count    int
	Delta8Bits      int
	Delta16Bits     int
	Delta24Bits     int
	Progress5PctLine string
}

func decodeRecordShapeWordSizesFromDeltaStream(shapeStreamSection []byte, expectedRecordCount int) ([][]uint8, decodedShapeDeltaStats, error) {
	if expectedRecordCount == 0 {
		if len(shapeStreamSection) != 0 {
			return nil, decodedShapeDeltaStats{}, fmt.Errorf("unexpected shape stream for empty records bytes=%d", len(shapeStreamSection))
		}
		return nil, decodedShapeDeltaStats{}, nil
	}
	recordWordSizes := make([][]uint8, expectedRecordCount)
	deltaStats := decodedShapeDeltaStats{}
	thresholdRecordByPercent := buildProgressThresholdByPercent(expectedRecordCount)
	progressBytesByPercent := make(map[int]int, 21)

	shapeBitReader := newDecodeBitReader(shapeStreamSection)
	previousShapeValue := uint32(0)
	for recordIndex := 0; recordIndex < expectedRecordCount; recordIndex++ {
		shapeDeltaValue, tokenKind, tokenBits, readShapeDeltaError := readShapeDeltaTokenFromDecodeBits(shapeBitReader)
		if readShapeDeltaError != nil {
			return nil, decodedShapeDeltaStats{}, fmt.Errorf("read shape delta record=%d: %w", recordIndex, readShapeDeltaError)
		}
		switch tokenKind {
		case 8:
			deltaStats.Delta8Count++
			deltaStats.Delta8Bits += tokenBits
		case 16:
			deltaStats.Delta16Count++
			deltaStats.Delta16Bits += tokenBits
		case 24:
			deltaStats.Delta24Count++
			deltaStats.Delta24Bits += tokenBits
		}
		currentShapeValue := previousShapeValue + shapeDeltaValue
		if currentShapeValue > 0xFFFFFF {
			return nil, decodedShapeDeltaStats{}, fmt.Errorf("shape value exceeds 24-bit range at record=%d value=%d", recordIndex, currentShapeValue)
		}
		currentWordSizes, decodeShapeValueError := decodeShapeWordSizesFromFixed24BitValue(currentShapeValue)
		if decodeShapeValueError != nil {
			return nil, decodedShapeDeltaStats{}, fmt.Errorf("decode shape value record=%d: %w", recordIndex, decodeShapeValueError)
		}
		recordWordSizes[recordIndex] = currentWordSizes
		previousShapeValue = currentShapeValue

		processedRecords := recordIndex + 1
		for percentStep := 0; percentStep <= 100; percentStep += 5 {
			if thresholdRecordByPercent[percentStep] == processedRecords {
				progressBytesByPercent[percentStep] = shapeBitReader.bytesConsumed()
			}
		}
	}
	progressEntries := make([]string, 0, 21)
	for percentStep := 0; percentStep <= 100; percentStep += 5 {
		progressEntries = append(progressEntries, fmt.Sprintf("p%d=%d", percentStep, progressBytesByPercent[percentStep]))
	}
	deltaStats.Progress5PctLine = strings.Join(progressEntries, ", ")
	return recordWordSizes, deltaStats, nil
}

func decodeShapeWordSizesFromFixed24BitValue(shapeValue uint32) ([]uint8, error) {
	wordSizes := make([]uint8, 0, 8)
	seenPadding := false
	for wordIndex := 0; wordIndex < 8; wordIndex++ {
		shiftBits := uint(21 - (wordIndex * 3))
		wordCode := uint8((shapeValue >> shiftBits) & 0x07)
		if wordCode == 0 {
			seenPadding = true
			continue
		}
		if seenPadding {
			return nil, fmt.Errorf("non-zero word code after padding at index=%d code=%d", wordIndex, wordCode)
		}
		if wordCode > 7 {
			return nil, fmt.Errorf("word code out of range at index=%d code=%d", wordIndex, wordCode)
		}
		wordSizes = append(wordSizes, wordCode)
	}
	if len(wordSizes) == 0 {
		return nil, fmt.Errorf("decoded empty shape")
	}
	return wordSizes, nil
}

type decodeBitReader struct {
	bytes          []uint8
	totalBitOffset int
}

func newDecodeBitReader(sourceBytes []uint8) *decodeBitReader {
	return &decodeBitReader{bytes: sourceBytes}
}

func (bitStreamReader *decodeBitReader) readBit() (uint8, error) {
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

func (bitStreamReader *decodeBitReader) readBits(bitCount int) (uint32, error) {
	readValue := uint32(0)
	for bitIndex := 0; bitIndex < bitCount; bitIndex++ {
		nextBit, readError := bitStreamReader.readBit()
		if readError != nil {
			return 0, readError
		}
		readValue = (readValue << 1) | uint32(nextBit)
	}
	return readValue, nil
}

func (bitStreamReader *decodeBitReader) bytesConsumed() int {
	if bitStreamReader.totalBitOffset == 0 {
		return 0
	}
	return (bitStreamReader.totalBitOffset + 7) / 8
}

func readShapeDeltaTokenFromDecodeBits(shapeBitReader *decodeBitReader) (uint32, int, int, error) {
	flagBit, readFlagError := shapeBitReader.readBit()
	if readFlagError != nil {
		return 0, 0, 0, readFlagError
	}
	if flagBit == 0 {
		shapeDeltaValue, readDeltaError := shapeBitReader.readBits(8)
		if readDeltaError != nil {
			return 0, 0, 0, readDeltaError
		}
		return shapeDeltaValue, 8, 9, nil
	}

	mediumDeltaValue, readMediumError := shapeBitReader.readBits(16)
	if readMediumError != nil {
		return 0, 0, 0, readMediumError
	}
	if mediumDeltaValue != 0xFFFF {
		return mediumDeltaValue, 16, 17, nil
	}
	escapeDeltaValue, readEscapeError := shapeBitReader.readBits(24)
	if readEscapeError != nil {
		return 0, 0, 0, readEscapeError
	}
	return escapeDeltaValue, 24, 41, nil
}

func buildProgressThresholdByPercent(recordCount int) map[int]int {
	thresholdRecordByPercent := make(map[int]int, 21)
	for percentStep := 0; percentStep <= 100; percentStep += 5 {
		thresholdRecord := int(float64(recordCount) * float64(percentStep) / 100.0)
		if thresholdRecord < 1 {
			thresholdRecord = 1
		}
		if thresholdRecord > recordCount {
			thresholdRecord = recordCount
		}
		thresholdRecordByPercent[percentStep] = thresholdRecord
	}
	return thresholdRecordByPercent
}

func computeExpectedContentBytesFromShapeStream(recordWordSizes [][]uint8) (int, error) {
	totalContentBytes := uint64(0)
	for recordIndex := range recordWordSizes {
		totalSyllablesPerRecord := 0
		for _, wordSize := range recordWordSizes[recordIndex] {
			totalSyllablesPerRecord += int(wordSize)
		}
		if totalSyllablesPerRecord <= 0 {
			return 0, fmt.Errorf("invalid zero-sized shape at record=%d", recordIndex)
		}
		totalContentBytes += uint64(totalSyllablesPerRecord)
	}
	if totalContentBytes > uint64(^uint(0)>>1) {
		return 0, fmt.Errorf("content bytes overflow")
	}
	return int(totalContentBytes), nil
}

func decodeContentRecordsFromShapeStream(contentSection []byte, dictionaryTokens []string, recordWordSizes [][]uint8) ([]DecodedIndexRecord, error) {
	decodedRecords := make([]DecodedIndexRecord, 0, len(recordWordSizes))
	contentCursorIndex := 0
	for recordIndex := range recordWordSizes {
		wordSizes := recordWordSizes[recordIndex]
		totalSyllablesPerRecord := 0
		for _, wordSize := range wordSizes {
			totalSyllablesPerRecord += int(wordSize)
		}
		if contentCursorIndex+totalSyllablesPerRecord > len(contentSection) {
			return nil, fmt.Errorf("content truncated at record=%d", recordIndex)
		}

		recordSyllableIDs := contentSection[contentCursorIndex : contentCursorIndex+totalSyllablesPerRecord]
		contentCursorIndex += totalSyllablesPerRecord

		wordTexts := make([]string, 0, len(wordSizes))
		syllableCursor := 0
		for _, wordSize := range wordSizes {
			wordSyllableCount := int(wordSize)
			var wordBuilder strings.Builder
			for syllableIndex := 0; syllableIndex < wordSyllableCount; syllableIndex++ {
				syllableID := int(recordSyllableIDs[syllableCursor])
				syllableCursor++
				if syllableID <= 0 || syllableID > len(dictionaryTokens) {
					return nil, fmt.Errorf("invalid syllable id=%d at record=%d", syllableID, recordIndex)
				}
				wordBuilder.WriteString(dictionaryTokens[syllableID-1])
			}
			wordTexts = append(wordTexts, wordBuilder.String())
		}

		recordShapeClass := shapeClassFourBit
		if shapeClass, classifyError := classifyShapeWordSizes(wordSizes); classifyError == nil {
			recordShapeClass = shapeClass
		}
		decodedRecords = append(decodedRecords, DecodedIndexRecord{
			RecordIndex:   recordIndex,
			ShapeClass:    recordShapeClass,
			ShapeLocalID:  0,
			WordSizes:     append([]uint8{}, wordSizes...),
			DecodedPhrase: strings.Join(wordTexts, " "),
		})
	}
	if contentCursorIndex != len(contentSection) {
		return nil, fmt.Errorf("content trailing bytes=%d", len(contentSection)-contentCursorIndex)
	}
	return decodedRecords, nil
}
