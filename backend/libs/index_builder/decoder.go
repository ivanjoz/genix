package index_builder

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

type DecodedRecord struct {
	RecordIndex int32
	Shape       []uint8
	Text        string
}

type DecodeStats struct {
	RecordCount       int32
	DictionaryCount   int32
	DictionaryBytes   int32
	ShapesBytes       int32
	ContentBytes      int32
	ShapeDelta8Count  int32
	ShapeDelta16Count int32
	ShapeDelta24Count int32
}

type DecodeResult struct {
	DictionaryTokens []string
	Records          []DecodedRecord
	Stats            DecodeStats
}

func DecodeBinary(indexBytes []byte) (*DecodeResult, error) {
	const headerSize = 27
	if len(indexBytes) < headerSize {
		return nil, fmt.Errorf("index too small bytes=%d", len(indexBytes))
	}
	if string(indexBytes[:len(BinaryMagic)]) != BinaryMagic {
		return nil, fmt.Errorf("invalid magic")
	}
	if indexBytes[len(BinaryMagic)] != BinaryVersion {
		return nil, fmt.Errorf("unsupported version=%d", indexBytes[len(BinaryMagic)])
	}

	cursor := len(BinaryMagic) + 1
	_ = indexBytes[cursor] // flags reserved for future decode branches.
	cursor++

	recordCount := int32(binary.LittleEndian.Uint32(indexBytes[cursor : cursor+4]))
	cursor += 4
	dictionaryCount := int32(indexBytes[cursor])
	cursor++
	dictionaryBytes := int(binary.LittleEndian.Uint32(indexBytes[cursor : cursor+4]))
	cursor += 4
	shapesBytes := int(binary.LittleEndian.Uint32(indexBytes[cursor : cursor+4]))
	cursor += 4
	contentBytes := int(binary.LittleEndian.Uint32(indexBytes[cursor : cursor+4]))
	cursor += 4

	if recordCount <= 0 {
		return nil, fmt.Errorf("invalid record count=%d", recordCount)
	}
	if dictionaryCount <= 0 || dictionaryCount > 255 {
		return nil, fmt.Errorf("invalid dictionary count=%d", dictionaryCount)
	}
	if cursor+dictionaryBytes+shapesBytes+contentBytes > len(indexBytes) {
		return nil, fmt.Errorf("sections exceed file bounds")
	}

	dictionarySection := indexBytes[cursor : cursor+dictionaryBytes]
	cursor += dictionaryBytes
	shapeSection := indexBytes[cursor : cursor+shapesBytes]
	cursor += shapesBytes
	contentSection := indexBytes[cursor : cursor+contentBytes]

	dictionaryTokens, decodeDictionaryErr := decodeDictionarySectionRaw(dictionarySection, int(dictionaryCount))
	if decodeDictionaryErr != nil {
		return nil, decodeDictionaryErr
	}

	shapeValues, d8, d16, d24, decodeShapeErr := decodeShapeDeltaStream(shapeSection, int(recordCount))
	if decodeShapeErr != nil {
		return nil, decodeShapeErr
	}
	decodedShapes := make([][]uint8, 0, len(shapeValues))
	for _, shapeValue := range shapeValues {
		decodedShapes = append(decodedShapes, decodeShapeValue(shapeValue))
	}

	decodedRecords, decodeContentErr := decodeContentRecords(contentSection, decodedShapes, dictionaryTokens)
	if decodeContentErr != nil {
		return nil, decodeContentErr
	}
	if len(decodedRecords) != int(recordCount) {
		return nil, fmt.Errorf("decoded record count mismatch expected=%d got=%d", recordCount, len(decodedRecords))
	}

	return &DecodeResult{
		DictionaryTokens: dictionaryTokens,
		Records:          decodedRecords,
		Stats: DecodeStats{
			RecordCount:       recordCount,
			DictionaryCount:   dictionaryCount,
			DictionaryBytes:   int32(dictionaryBytes),
			ShapesBytes:       int32(shapesBytes),
			ContentBytes:      int32(contentBytes),
			ShapeDelta8Count:  int32(d8),
			ShapeDelta16Count: int32(d16),
			ShapeDelta24Count: int32(d24),
		},
	}, nil
}

func decodeDictionarySectionRaw(section []byte, dictionaryCount int) ([]string, error) {
	tokens := make([]string, 0, dictionaryCount)
	cursor := 0
	for tokenIndex := 0; tokenIndex < dictionaryCount; tokenIndex++ {
		if cursor >= len(section) {
			return nil, fmt.Errorf("dictionary truncated at token=%d", tokenIndex)
		}
		tokenLen := int(section[cursor])
		cursor++
		if tokenLen <= 0 || cursor+tokenLen > len(section) {
			return nil, fmt.Errorf("invalid dictionary token len at token=%d", tokenIndex)
		}
		tokens = append(tokens, string(section[cursor:cursor+tokenLen]))
		cursor += tokenLen
	}
	if cursor != len(section) {
		return nil, fmt.Errorf("dictionary trailing bytes=%d", len(section)-cursor)
	}
	return tokens, nil
}

type bitReader struct {
	bytes  []byte
	offset int
}

func newBitReader(bytes []byte) *bitReader {
	return &bitReader{bytes: bytes, offset: 0}
}

func (reader *bitReader) readBit() (uint8, error) {
	if reader.offset >= len(reader.bytes)*8 {
		return 0, fmt.Errorf("bitstream exhausted")
	}
	byteIndex := reader.offset / 8
	bitIndex := reader.offset % 8
	bit := (reader.bytes[byteIndex] >> (7 - bitIndex)) & 1
	reader.offset++
	return bit, nil
}

func (reader *bitReader) readBits(count int) (uint32, error) {
	value := uint32(0)
	for i := 0; i < count; i++ {
		bit, bitErr := reader.readBit()
		if bitErr != nil {
			return 0, bitErr
		}
		value = (value << 1) | uint32(bit)
	}
	return value, nil
}

func decodeShapeDeltaStream(shapeBytes []byte, recordCount int) ([]uint32, int, int, int, error) {
	reader := newBitReader(shapeBytes)
	shapeValues := make([]uint32, 0, recordCount)
	previous := uint32(0)
	count8 := 0
	count16 := 0
	count24 := 0

	for recordIndex := 0; recordIndex < recordCount; recordIndex++ {
		flagBit, flagErr := reader.readBit()
		if flagErr != nil {
			return nil, 0, 0, 0, fmt.Errorf("read shape flag at record=%d: %w", recordIndex, flagErr)
		}
		delta := uint32(0)
		if flagBit == 0 {
			deltaValue, deltaErr := reader.readBits(8)
			if deltaErr != nil {
				return nil, 0, 0, 0, fmt.Errorf("read 8-bit delta at record=%d: %w", recordIndex, deltaErr)
			}
			delta = deltaValue
			count8++
		} else {
			mediumDelta, deltaErr := reader.readBits(16)
			if deltaErr != nil {
				return nil, 0, 0, 0, fmt.Errorf("read 16-bit delta at record=%d: %w", recordIndex, deltaErr)
			}
			if mediumDelta == 0xFFFF {
				escapedDelta, escapeErr := reader.readBits(24)
				if escapeErr != nil {
					return nil, 0, 0, 0, fmt.Errorf("read 24-bit delta at record=%d: %w", recordIndex, escapeErr)
				}
				delta = escapedDelta
				count24++
			} else {
				delta = mediumDelta
				count16++
			}
		}
		current := previous + delta
		shapeValues = append(shapeValues, current)
		previous = current
	}
	return shapeValues, count8, count16, count24, nil
}

func decodeShapeValue(shapeValue uint32) []uint8 {
	decoded := make([]uint8, 0, 8)
	for wordIndex := 0; wordIndex < 8; wordIndex++ {
		shift := uint(21 - (wordIndex * 3))
		wordCode := uint8((shapeValue >> shift) & 0x07)
		if wordCode == 0 {
			break
		}
		decoded = append(decoded, wordCode)
	}
	return decoded
}

func decodeContentRecords(content []byte, shapes [][]uint8, dictionaryTokens []string) ([]DecodedRecord, error) {
	cursor := 0
	records := make([]DecodedRecord, 0, len(shapes))
	for recordIndex, shape := range shapes {
		words := make([]string, 0, len(shape))
		for _, wordSize := range shape {
			syllableCount := int(wordSize)
			if cursor+syllableCount > len(content) {
				return nil, fmt.Errorf("content overflow at record=%d", recordIndex)
			}
			var wordBuilder strings.Builder
			for syllableIndex := 0; syllableIndex < syllableCount; syllableIndex++ {
				dictionaryID := int(content[cursor])
				cursor++
				if dictionaryID <= 0 || dictionaryID > len(dictionaryTokens) {
					return nil, fmt.Errorf("invalid dictionary id=%d at record=%d", dictionaryID, recordIndex)
				}
				wordBuilder.WriteString(dictionaryTokens[dictionaryID-1])
			}
			words = append(words, wordBuilder.String())
		}
		records = append(records, DecodedRecord{
			RecordIndex: int32(recordIndex),
			Shape:       append([]uint8(nil), shape...),
			Text:        strings.Join(words, " "),
		})
	}
	if cursor != len(content) {
		return nil, fmt.Errorf("unused content bytes=%d", len(content)-cursor)
	}
	return records, nil
}

func SampleDecodedRecords(records []DecodedRecord, sampleCount int32, seed int64) []DecodedRecord {
	if sampleCount <= 0 || len(records) == 0 {
		return nil
	}
	if int(sampleCount) > len(records) {
		sampleCount = int32(len(records))
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	randomGenerator := rand.New(rand.NewSource(seed))
	permutation := randomGenerator.Perm(len(records))
	selected := permutation[:sampleCount]
	sort.Ints(selected)

	sampled := make([]DecodedRecord, 0, sampleCount)
	for _, idx := range selected {
		sampled = append(sampled, records[idx])
	}
	return sampled
}
