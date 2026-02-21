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

	BrandDictionaryIndex int32
	BrandID              uint16
	BrandName            string

	CategoryDictionaryIndexes []uint8
	CategoryIDs               []uint16
	CategoryNames             []string
}

type DecodedTaxonomy struct {
	BrandIDs                    []uint16
	BrandNames                  []string
	CategoryIDs                 []uint16
	CategoryNames               []string
	ProductBrandDictionaryIndex []int
	ProductCategoryCounts       []int
	ProductCategoryIndexes      []uint8
	BrandIndexEncodingFlag      uint8
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

	TaxonomyBytes          int32
	TaxonomyBrandCount     int32
	TaxonomyCategoryCount  int32
	TaxonomyBrandIndexMode string
}

type DecodeResult struct {
	DictionaryTokens []string
	Records          []DecodedRecord
	Taxonomy         *DecodedTaxonomy
	Stats            DecodeStats
}

func DecodeBinary(indexBytes []byte) (*DecodeResult, error) {
	textHeader, textHeaderErr := decodeTextHeader(indexBytes)
	if textHeaderErr != nil {
		return nil, textHeaderErr
	}

	textPayloadEnd := textHeader.headerSize + textHeader.dictionaryBytes + textHeader.shapesBytes + textHeader.contentBytes
	if textPayloadEnd > len(indexBytes) {
		return nil, fmt.Errorf("text sections exceed file bounds")
	}

	cursor := textHeader.headerSize
	dictionarySection := indexBytes[cursor : cursor+textHeader.dictionaryBytes]
	cursor += textHeader.dictionaryBytes
	shapeSection := indexBytes[cursor : cursor+textHeader.shapesBytes]
	cursor += textHeader.shapesBytes
	contentSection := indexBytes[cursor : cursor+textHeader.contentBytes]

	dictionaryTokens, dictionaryErr := decodeDictionarySectionRaw(dictionarySection, int(textHeader.dictionaryCount))
	if dictionaryErr != nil {
		return nil, dictionaryErr
	}

	shapeValues, count8, count16, count24, shapeErr := decodeShapeDeltaStream(shapeSection, int(textHeader.recordCount))
	if shapeErr != nil {
		return nil, shapeErr
	}
	decodedShapes := make([][]uint8, 0, len(shapeValues))
	for _, shapeValue := range shapeValues {
		decodedShapes = append(decodedShapes, decodeShapeValue(shapeValue))
	}

	decodedRecords, contentErr := decodeContentRecords(contentSection, decodedShapes, dictionaryTokens)
	if contentErr != nil {
		return nil, contentErr
	}
	if len(decodedRecords) != int(textHeader.recordCount) {
		return nil, fmt.Errorf("decoded record count mismatch expected=%d got=%d", textHeader.recordCount, len(decodedRecords))
	}

	decodeResult := &DecodeResult{
		DictionaryTokens: dictionaryTokens,
		Records:          decodedRecords,
		Stats: DecodeStats{
			RecordCount:       textHeader.recordCount,
			DictionaryCount:   textHeader.dictionaryCount,
			DictionaryBytes:   int32(textHeader.dictionaryBytes),
			ShapesBytes:       int32(textHeader.shapesBytes),
			ContentBytes:      int32(textHeader.contentBytes),
			ShapeDelta8Count:  int32(count8),
			ShapeDelta16Count: int32(count16),
			ShapeDelta24Count: int32(count24),
		},
	}

	if textPayloadEnd == len(indexBytes) {
		return decodeResult, nil
	}

	decodedTaxonomy, taxonomyBytes, taxonomyErr := decodeTaxonomyBlock(indexBytes[textPayloadEnd:], int(textHeader.recordCount))
	if taxonomyErr != nil {
		return nil, taxonomyErr
	}

	attachErr := attachTaxonomyToRecords(decodedRecords, decodedTaxonomy)
	if attachErr != nil {
		return nil, attachErr
	}

	decodeResult.Taxonomy = decodedTaxonomy
	decodeResult.Stats.TaxonomyBytes = int32(taxonomyBytes)
	decodeResult.Stats.TaxonomyBrandCount = int32(len(decodedTaxonomy.BrandIDs))
	decodeResult.Stats.TaxonomyCategoryCount = int32(len(decodedTaxonomy.CategoryIDs))
	decodeResult.Stats.TaxonomyBrandIndexMode = brandEncodingName(decodedTaxonomy.BrandIndexEncodingFlag)
	return decodeResult, nil
}

type decodedTextHeader struct {
	headerSize      int
	recordCount     int32
	dictionaryCount int32
	dictionaryBytes int
	shapesBytes     int
	contentBytes    int
}

func decodeTextHeader(indexBytes []byte) (*decodedTextHeader, error) {
	const textHeaderSize = len(BinaryMagic) + 1 + 1 + 4 + 1 + 4 + 4 + 4
	if len(indexBytes) < textHeaderSize {
		return nil, fmt.Errorf("index too small bytes=%d", len(indexBytes))
	}
	if string(indexBytes[:len(BinaryMagic)]) != BinaryMagic {
		return nil, fmt.Errorf("invalid text magic")
	}
	if indexBytes[len(BinaryMagic)] != BinaryVersion {
		return nil, fmt.Errorf("unsupported text version=%d", indexBytes[len(BinaryMagic)])
	}

	cursor := len(BinaryMagic) + 1
	_ = indexBytes[cursor] // Flags are currently not needed for decode branching.
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
	if dictionaryBytes < 0 || shapesBytes < 0 || contentBytes < 0 {
		return nil, fmt.Errorf("invalid negative section length")
	}

	return &decodedTextHeader{
		headerSize:      textHeaderSize,
		recordCount:     recordCount,
		dictionaryCount: dictionaryCount,
		dictionaryBytes: dictionaryBytes,
		shapesBytes:     shapesBytes,
		contentBytes:    contentBytes,
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

func decodeTaxonomyBlock(taxonomyBytes []byte, expectedProductCount int) (*DecodedTaxonomy, int, error) {
	if len(taxonomyBytes) < taxonomyHeaderSize {
		return nil, 0, fmt.Errorf("taxonomy section too small bytes=%d", len(taxonomyBytes))
	}
	if string(taxonomyBytes[:len(taxonomyBinaryMagic)]) != taxonomyBinaryMagic {
		return nil, 0, fmt.Errorf("invalid taxonomy magic")
	}
	if taxonomyBytes[len(taxonomyBinaryMagic)] != taxonomyBinaryVersion {
		return nil, 0, fmt.Errorf("unsupported taxonomy version=%d", taxonomyBytes[len(taxonomyBinaryMagic)])
	}

	cursor := len(taxonomyBinaryMagic) + 1
	brandEncodingFlag := taxonomyBytes[cursor]
	cursor++
	sortedProductCount := int(binary.LittleEndian.Uint32(taxonomyBytes[cursor : cursor+4]))
	cursor += 4
	if sortedProductCount != expectedProductCount {
		return nil, 0, fmt.Errorf("taxonomy/text row mismatch taxonomy=%d text=%d", sortedProductCount, expectedProductCount)
	}

	sectionLengths := make([]int, 7)
	for sectionIndex := 0; sectionIndex < 7; sectionIndex++ {
		sectionLengths[sectionIndex] = int(binary.LittleEndian.Uint32(taxonomyBytes[cursor : cursor+4]))
		cursor += 4
	}

	sectionsTotal := 0
	for _, sectionLength := range sectionLengths {
		sectionsTotal += sectionLength
	}
	if cursor+sectionsTotal != len(taxonomyBytes) {
		return nil, 0, fmt.Errorf("taxonomy size mismatch header=%d actual=%d", cursor+sectionsTotal, len(taxonomyBytes))
	}

	sections := make([][]byte, 7)
	for sectionIndex, sectionLength := range sectionLengths {
		sections[sectionIndex] = taxonomyBytes[cursor : cursor+sectionLength]
		cursor += sectionLength
	}

	brandIDs, brandIDsErr := decodeUint16Section(sections[0], "brand_ids")
	if brandIDsErr != nil {
		return nil, 0, brandIDsErr
	}
	brandNames, brandNamesErr := decodeStringColumnSection(sections[1], "brand_names")
	if brandNamesErr != nil {
		return nil, 0, brandNamesErr
	}
	if len(brandIDs) != len(brandNames) {
		return nil, 0, fmt.Errorf("brand dictionary mismatch ids=%d names=%d", len(brandIDs), len(brandNames))
	}

	categoryIDs, categoryIDsErr := decodeUint16Section(sections[2], "category_ids")
	if categoryIDsErr != nil {
		return nil, 0, categoryIDsErr
	}
	categoryNames, categoryNamesErr := decodeStringColumnSection(sections[3], "category_names")
	if categoryNamesErr != nil {
		return nil, 0, categoryNamesErr
	}
	if len(categoryIDs) != len(categoryNames) {
		return nil, 0, fmt.Errorf("category dictionary mismatch ids=%d names=%d", len(categoryIDs), len(categoryNames))
	}

	productBrandIndexes, brandIndexesErr := decodeBrandIndexes(sections[4], brandEncodingFlag, sortedProductCount)
	if brandIndexesErr != nil {
		return nil, 0, brandIndexesErr
	}
	for rowIndex, brandIndex := range productBrandIndexes {
		if brandIndex < 0 || brandIndex >= len(brandIDs) {
			return nil, 0, fmt.Errorf("brand index out of range row=%d index=%d dict=%d", rowIndex, brandIndex, len(brandIDs))
		}
	}

	productCategoryCounts, categoryCountsErr := decodePackedCategoryCounts(sections[5], sortedProductCount)
	if categoryCountsErr != nil {
		return nil, 0, categoryCountsErr
	}
	totalCategoryRefs := 0
	for _, categoryCount := range productCategoryCounts {
		totalCategoryRefs += categoryCount
	}
	if totalCategoryRefs != len(sections[6]) {
		return nil, 0, fmt.Errorf("category index payload mismatch expected=%d got=%d", totalCategoryRefs, len(sections[6]))
	}
	for payloadIndex, categoryIndex := range sections[6] {
		if int(categoryIndex) >= len(categoryIDs) {
			return nil, 0, fmt.Errorf("category index out of range at payload index=%d index=%d dict=%d", payloadIndex, categoryIndex, len(categoryIDs))
		}
	}

	decodedTaxonomy := &DecodedTaxonomy{
		BrandIDs:                    brandIDs,
		BrandNames:                  brandNames,
		CategoryIDs:                 categoryIDs,
		CategoryNames:               categoryNames,
		ProductBrandDictionaryIndex: productBrandIndexes,
		ProductCategoryCounts:       productCategoryCounts,
		ProductCategoryIndexes:      append([]uint8(nil), sections[6]...),
		BrandIndexEncodingFlag:      brandEncodingFlag,
	}
	return decodedTaxonomy, len(taxonomyBytes), nil
}

func decodeUint16Section(section []byte, sectionName string) ([]uint16, error) {
	if len(section)%2 != 0 {
		return nil, fmt.Errorf("%s bytes must be even got=%d", sectionName, len(section))
	}
	values := make([]uint16, 0, len(section)/2)
	for cursor := 0; cursor < len(section); cursor += 2 {
		values = append(values, binary.LittleEndian.Uint16(section[cursor:cursor+2]))
	}
	return values, nil
}

func decodeStringColumnSection(section []byte, sectionName string) ([]string, error) {
	values := make([]string, 0, 256)
	cursor := 0
	for cursor < len(section) {
		valueLen := int(section[cursor])
		cursor++
		if cursor+valueLen > len(section) {
			return nil, fmt.Errorf("%s truncated string at offset=%d", sectionName, cursor-1)
		}
		values = append(values, string(section[cursor:cursor+valueLen]))
		cursor += valueLen
	}
	return values, nil
}

func decodeBrandIndexes(brandIndexesSection []byte, brandEncodingFlag uint8, sortedProductCount int) ([]int, error) {
	switch brandEncodingFlag {
	case BrandIndexEncodingUint12:
		return unpackUint12Values(brandIndexesSection, sortedProductCount)
	case BrandIndexEncodingUint16:
		if len(brandIndexesSection) != sortedProductCount*2 {
			return nil, fmt.Errorf("uint16 brand index payload mismatch expected=%d got=%d", sortedProductCount*2, len(brandIndexesSection))
		}
		indexes := make([]int, 0, sortedProductCount)
		for cursor := 0; cursor < len(brandIndexesSection); cursor += 2 {
			indexes = append(indexes, int(binary.LittleEndian.Uint16(brandIndexesSection[cursor:cursor+2])))
		}
		return indexes, nil
	default:
		return nil, fmt.Errorf("unsupported brand index encoding flag=%d", brandEncodingFlag)
	}
}

func unpackUint12Values(packed []byte, expectedValues int) ([]int, error) {
	expectedBytes := expectedUint12PackedBytes(expectedValues)
	if len(packed) != expectedBytes {
		return nil, fmt.Errorf("uint12 payload mismatch expected=%d got=%d", expectedBytes, len(packed))
	}

	values := make([]int, 0, expectedValues)
	for cursor := 0; cursor < len(packed); cursor += 3 {
		leftValue := (int(packed[cursor]) << 4) | int((packed[cursor+1]>>4)&0x0F)
		values = append(values, leftValue)
		if len(values) == expectedValues {
			break
		}
		rightValue := (int(packed[cursor+1]&0x0F) << 8) | int(packed[cursor+2])
		values = append(values, rightValue)
	}
	return values, nil
}

func decodePackedCategoryCounts(packedCounts []byte, sortedProductCount int) ([]int, error) {
	expectedCountBytes := (sortedProductCount + 3) / 4
	if len(packedCounts) != expectedCountBytes {
		return nil, fmt.Errorf("category count payload mismatch expected=%d got=%d", expectedCountBytes, len(packedCounts))
	}

	counts := make([]int, 0, sortedProductCount)
	for productIndex := 0; productIndex < sortedProductCount; productIndex++ {
		packedByte := packedCounts[productIndex/4]
		shift := uint(6 - (productIndex%4)*2)
		countMinusOne := (packedByte >> shift) & 0x03
		counts = append(counts, int(countMinusOne)+1)
	}
	return counts, nil
}

func attachTaxonomyToRecords(records []DecodedRecord, taxonomy *DecodedTaxonomy) error {
	if taxonomy == nil {
		return nil
	}
	if len(records) != len(taxonomy.ProductBrandDictionaryIndex) {
		return fmt.Errorf("record/brand rows mismatch records=%d brands=%d", len(records), len(taxonomy.ProductBrandDictionaryIndex))
	}
	if len(records) != len(taxonomy.ProductCategoryCounts) {
		return fmt.Errorf("record/category rows mismatch records=%d counts=%d", len(records), len(taxonomy.ProductCategoryCounts))
	}

	categoryPayloadCursor := 0
	for recordIndex := range records {
		brandDictionaryIndex := taxonomy.ProductBrandDictionaryIndex[recordIndex]
		records[recordIndex].BrandDictionaryIndex = int32(brandDictionaryIndex)
		records[recordIndex].BrandID = taxonomy.BrandIDs[brandDictionaryIndex]
		records[recordIndex].BrandName = taxonomy.BrandNames[brandDictionaryIndex]

		categoryCount := taxonomy.ProductCategoryCounts[recordIndex]
		if categoryPayloadCursor+categoryCount > len(taxonomy.ProductCategoryIndexes) {
			return fmt.Errorf("category payload overflow at record=%d", recordIndex)
		}

		categoryIndexes := make([]uint8, 0, categoryCount)
		categoryIDs := make([]uint16, 0, categoryCount)
		categoryNames := make([]string, 0, categoryCount)
		for categoryOffset := 0; categoryOffset < categoryCount; categoryOffset++ {
			categoryDictionaryIndex := taxonomy.ProductCategoryIndexes[categoryPayloadCursor]
			categoryPayloadCursor++
			categoryIndexes = append(categoryIndexes, categoryDictionaryIndex)
			categoryIDs = append(categoryIDs, taxonomy.CategoryIDs[categoryDictionaryIndex])
			categoryNames = append(categoryNames, taxonomy.CategoryNames[categoryDictionaryIndex])
		}
		records[recordIndex].CategoryDictionaryIndexes = categoryIndexes
		records[recordIndex].CategoryIDs = categoryIDs
		records[recordIndex].CategoryNames = categoryNames
	}

	if categoryPayloadCursor != len(taxonomy.ProductCategoryIndexes) {
		return fmt.Errorf("unused category payload bytes=%d", len(taxonomy.ProductCategoryIndexes)-categoryPayloadCursor)
	}
	return nil
}

func brandEncodingName(flag uint8) string {
	switch flag {
	case BrandIndexEncodingUint12:
		return "uint12"
	case BrandIndexEncodingUint16:
		return "uint16"
	default:
		return "unknown"
	}
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
