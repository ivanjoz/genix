package index_builder

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"
)

const (
	taxonomySectionBrandIDs       uint8 = 1
	taxonomySectionBrandNames     uint8 = 2
	taxonomySectionCategoryIDs    uint8 = 3
	taxonomySectionCategoryNames  uint8 = 4
	taxonomySectionBrandIndexes   uint8 = 5
	taxonomySectionCategoryCounts uint8 = 6
	taxonomySectionCategoryIndex  uint8 = 7
)

type DecodedRecord struct {
	RecordIndex int32
	ProductID   int32
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
	BuildSunixTime    int32
	DictionaryCount   int32
	DictionaryBytes   int32
	AliasBytes        int32
	ShapesBytes       int32
	ContentBytes      int32
	ProductIDsBytes   int32
	ShapeDelta8Count  int32
	ShapeDelta16Count int32
	ShapeDelta24Count int32

	TaxonomyBytes          int32
	TaxonomyBrandCount     int32
	TaxonomyCategoryCount  int32
	TaxonomyBrandIndexMode string
}

type DecodeResult struct {
	DictionaryTokens  []string
	DictionaryAliases map[string]uint8
	Records           []DecodedRecord
	Taxonomy          *DecodedTaxonomy
	Stats             DecodeStats
}

type binarySectionDescriptor struct {
	sectionID uint8
	offset    int
	length    int
	itemCount uint32
	checksum  uint32
}

func DecodeBinary(indexBytes []byte) (*DecodeResult, error) {
	textHeader, textHeaderErr := decodeTextHeader(indexBytes)
	if textHeaderErr != nil {
		return nil, textHeaderErr
	}

	dictionarySection := textHeader.requiredSection(indexBytes, textSectionDictionary)
	shapeSection := textHeader.requiredSection(indexBytes, textSectionShapes)
	contentSection := textHeader.requiredSection(indexBytes, textSectionContent)
	aliasSection := textHeader.requiredSection(indexBytes, textSectionAliases)
	productIDsSection := textHeader.requiredSection(indexBytes, textSectionProductIDs)

	dictionaryTokens, dictionaryErr := decodeDictionarySectionRaw(dictionarySection, int(textHeader.dictionaryCount))
	if dictionaryErr != nil {
		return nil, dictionaryErr
	}

	decodedProductIDs, productIDsErr := decodeProductIDsSection(productIDsSection, int(textHeader.recordCount))
	if productIDsErr != nil {
		return nil, productIDsErr
	}

	shapeValues, delta8Count, delta16Count, delta24Count, shapeErr := decodeShapeDeltaStream(shapeSection, int(textHeader.recordCount))
	if shapeErr != nil {
		return nil, shapeErr
	}
	decodedShapes := make([][]uint8, 0, len(shapeValues))
	for _, shapeValue := range shapeValues {
		decodedShapes = append(decodedShapes, decodeShapeValue(shapeValue))
	}

	decodedRecords, contentErr := decodeContentRecords(contentSection, decodedShapes, dictionaryTokens, decodedProductIDs)
	if contentErr != nil {
		return nil, contentErr
	}
	if len(decodedRecords) != int(textHeader.recordCount) {
		return nil, fmt.Errorf("decoded record count mismatch expected=%d got=%d", textHeader.recordCount, len(decodedRecords))
	}

	decodeResult := &DecodeResult{
		DictionaryTokens:  dictionaryTokens,
		DictionaryAliases: map[string]uint8{},
		Records:           decodedRecords,
		Stats: DecodeStats{
			RecordCount:       textHeader.recordCount,
			BuildSunixTime:    textHeader.buildSunixTime,
			DictionaryCount:   textHeader.dictionaryCount,
			DictionaryBytes:   int32(len(dictionarySection)),
			AliasBytes:        int32(len(aliasSection)),
			ShapesBytes:       int32(len(shapeSection)),
			ContentBytes:      int32(len(contentSection)),
			ProductIDsBytes:   int32(len(productIDsSection)),
			ShapeDelta8Count:  int32(delta8Count),
			ShapeDelta16Count: int32(delta16Count),
			ShapeDelta24Count: int32(delta24Count),
		},
	}
	decodedAliases, aliasErr := decodeAliasSection(aliasSection, dictionaryTokens, int(textHeader.sectionsByID[textSectionAliases].itemCount))
	if aliasErr != nil {
		return nil, aliasErr
	}
	decodeResult.DictionaryAliases = decodedAliases

	if textHeader.payloadEnd == len(indexBytes) {
		return decodeResult, nil
	}

	decodedTaxonomy, taxonomyBytes, taxonomyErr := decodeTaxonomyBlock(indexBytes[textHeader.payloadEnd:], int(textHeader.recordCount))
	if taxonomyErr != nil {
		return nil, taxonomyErr
	}

	attachErr := attachTaxonomyToRecords(decodeResult.Records, decodedTaxonomy)
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
	recordCount     int32
	buildSunixTime  int32
	dictionaryCount int32
	sectionsByID    map[uint8]binarySectionDescriptor
	payloadEnd      int
}

func decodeTextHeader(indexBytes []byte) (*decodedTextHeader, error) {
	// Header layout: magic + version + flags + record_count + build_sunix_time + dictionary_count + section_count + header_size.
	const baseHeaderSize = len(BinaryMagic) + 1 + 1 + 4 + 4 + 1 + 1 + 2
	if len(indexBytes) < baseHeaderSize {
		return nil, fmt.Errorf("index too small bytes=%d", len(indexBytes))
	}
	if string(indexBytes[:len(BinaryMagic)]) != BinaryMagic {
		return nil, fmt.Errorf("invalid text magic")
	}
	if indexBytes[len(BinaryMagic)] != BinaryVersion {
		return nil, fmt.Errorf("unsupported text version=%d (expected=%d)", indexBytes[len(BinaryMagic)], BinaryVersion)
	}

	cursor := len(BinaryMagic) + 1
	_ = indexBytes[cursor] // Reserved flags.
	cursor++

	recordCount := int32(binary.LittleEndian.Uint32(indexBytes[cursor : cursor+4]))
	cursor += 4
	buildSunixTime := int32(binary.LittleEndian.Uint32(indexBytes[cursor : cursor+4]))
	cursor += 4
	dictionaryCount := int32(indexBytes[cursor])
	cursor++
	sectionCount := int(indexBytes[cursor])
	cursor++
	headerSize := int(binary.LittleEndian.Uint16(indexBytes[cursor : cursor+2]))
	cursor += 2

	if recordCount <= 0 {
		return nil, fmt.Errorf("invalid record count=%d", recordCount)
	}
	if dictionaryCount <= 0 || dictionaryCount > 255 {
		return nil, fmt.Errorf("invalid dictionary count=%d", dictionaryCount)
	}
	if sectionCount != 5 {
		return nil, fmt.Errorf("invalid text section count=%d", sectionCount)
	}

	sectionsByID, payloadEnd, sectionErr := decodeSectionTable(indexBytes, cursor, headerSize, sectionCount)
	if sectionErr != nil {
		return nil, fmt.Errorf("decode text section table: %w", sectionErr)
	}

	requiredTextSections := []uint8{textSectionDictionary, textSectionShapes, textSectionContent, textSectionAliases, textSectionProductIDs}
	for _, sectionID := range requiredTextSections {
		if _, hasSection := sectionsByID[sectionID]; !hasSection {
			return nil, fmt.Errorf("missing text section=%d", sectionID)
		}
	}

	if sectionsByID[textSectionDictionary].itemCount != uint32(dictionaryCount) {
		return nil, fmt.Errorf("text dictionary item count mismatch expected=%d got=%d", dictionaryCount, sectionsByID[textSectionDictionary].itemCount)
	}
	if sectionsByID[textSectionShapes].itemCount != uint32(recordCount) {
		return nil, fmt.Errorf("text shapes item count mismatch expected=%d got=%d", recordCount, sectionsByID[textSectionShapes].itemCount)
	}
	if sectionsByID[textSectionContent].itemCount != uint32(sectionsByID[textSectionContent].length) {
		return nil, fmt.Errorf("text content item count mismatch expected=%d got=%d", sectionsByID[textSectionContent].length, sectionsByID[textSectionContent].itemCount)
	}
	if sectionsByID[textSectionProductIDs].itemCount != uint32(recordCount) {
		return nil, fmt.Errorf("text product_ids item count mismatch expected=%d got=%d", recordCount, sectionsByID[textSectionProductIDs].itemCount)
	}

	return &decodedTextHeader{
		recordCount:     recordCount,
		buildSunixTime:  buildSunixTime,
		dictionaryCount: dictionaryCount,
		sectionsByID:    sectionsByID,
		payloadEnd:      payloadEnd,
	}, nil
}

func decodeAliasSection(section []byte, dictionaryTokens []string, expectedAliases int) (map[string]uint8, error) {
	decodedAliases := make(map[string]uint8, expectedAliases)
	cursor := 0
	for aliasIndex := 0; aliasIndex < expectedAliases; aliasIndex++ {
		if cursor >= len(section) {
			return nil, fmt.Errorf("alias section truncated at alias=%d", aliasIndex)
		}
		aliasLen := int(section[cursor])
		cursor++
		if aliasLen <= 0 || cursor+aliasLen > len(section) {
			return nil, fmt.Errorf("invalid alias length at alias=%d", aliasIndex)
		}
		aliasToken := string(section[cursor : cursor+aliasLen])
		cursor += aliasLen
		if cursor >= len(section) {
			return nil, fmt.Errorf("alias section missing dictionary id at alias=%d", aliasIndex)
		}
		dictionaryID := section[cursor]
		cursor++
		if dictionaryID == 0 || int(dictionaryID) > len(dictionaryTokens) {
			return nil, fmt.Errorf("alias dictionary id out of range alias=%s id=%d dict=%d", aliasToken, dictionaryID, len(dictionaryTokens))
		}
		decodedAliases[aliasToken] = dictionaryID
	}
	if cursor != len(section) {
		return nil, fmt.Errorf("alias section trailing bytes=%d", len(section)-cursor)
	}
	return decodedAliases, nil
}

func (decodedHeader *decodedTextHeader) requiredSection(indexBytes []byte, sectionID uint8) []byte {
	sectionDescriptor, exists := decodedHeader.sectionsByID[sectionID]
	if !exists {
		return nil
	}
	return indexBytes[sectionDescriptor.offset : sectionDescriptor.offset+sectionDescriptor.length]
}

func decodeSectionTable(payload []byte, tableStart int, headerSize int, sectionCount int) (map[uint8]binarySectionDescriptor, int, error) {
	const sectionEntrySize = 1 + 4 + 4 + 4 + 4
	if sectionCount <= 0 {
		return nil, 0, fmt.Errorf("invalid section count=%d", sectionCount)
	}
	if headerSize < tableStart {
		return nil, 0, fmt.Errorf("invalid header size=%d table_start=%d", headerSize, tableStart)
	}
	if tableStart+sectionCount*sectionEntrySize > len(payload) {
		return nil, 0, fmt.Errorf("truncated section table")
	}

	sectionsByID := make(map[uint8]binarySectionDescriptor, sectionCount)
	cursor := tableStart
	maxPayloadEnd := headerSize
	for sectionIndex := 0; sectionIndex < sectionCount; sectionIndex++ {
		sectionID := payload[cursor]
		cursor++
		offset := int(binary.LittleEndian.Uint32(payload[cursor : cursor+4]))
		cursor += 4
		length := int(binary.LittleEndian.Uint32(payload[cursor : cursor+4]))
		cursor += 4
		itemCount := binary.LittleEndian.Uint32(payload[cursor : cursor+4])
		cursor += 4
		checksum := binary.LittleEndian.Uint32(payload[cursor : cursor+4])
		cursor += 4

		if _, duplicatedSectionID := sectionsByID[sectionID]; duplicatedSectionID {
			return nil, 0, fmt.Errorf("duplicated section id=%d", sectionID)
		}
		if length < 0 || offset < headerSize || offset+length > len(payload) {
			return nil, 0, fmt.Errorf("invalid section bounds id=%d offset=%d length=%d", sectionID, offset, length)
		}

		sectionPayload := payload[offset : offset+length]
		if crc32.ChecksumIEEE(sectionPayload) != checksum {
			return nil, 0, fmt.Errorf("checksum mismatch id=%d", sectionID)
		}

		sectionsByID[sectionID] = binarySectionDescriptor{
			sectionID: sectionID,
			offset:    offset,
			length:    length,
			itemCount: itemCount,
			checksum:  checksum,
		}
		if offset+length > maxPayloadEnd {
			maxPayloadEnd = offset + length
		}
	}
	if cursor != headerSize {
		return nil, 0, fmt.Errorf("header size mismatch expected=%d got=%d", headerSize, cursor)
	}
	return sectionsByID, maxPayloadEnd, nil
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

func decodeProductIDsSection(section []byte, expectedRecordCount int) ([]int32, error) {
	decodedProductIDs := make([]int32, 0, expectedRecordCount)
	cursor := 0

	readWord := func() (uint16, error) {
		if cursor+2 > len(section) {
			return 0, fmt.Errorf("product_ids section truncated at byte=%d", cursor)
		}
		wordValue := binary.LittleEndian.Uint16(section[cursor : cursor+2])
		cursor += 2
		return wordValue, nil
	}

	for recordIndex := 0; recordIndex < expectedRecordCount; recordIndex++ {
		firstWord, readErr := readWord()
		if readErr != nil {
			return nil, fmt.Errorf("read product id word at record=%d: %w", recordIndex, readErr)
		}

		var decodedProductID uint32
		if firstWord != productIDUint32Sentinel {
			decodedProductID = uint32(firstWord)
		} else {
			lowWord, lowErr := readWord()
			if lowErr != nil {
				return nil, fmt.Errorf("read product id low word at record=%d: %w", recordIndex, lowErr)
			}
			highWord, highErr := readWord()
			if highErr != nil {
				return nil, fmt.Errorf("read product id high word at record=%d: %w", recordIndex, highErr)
			}
			decodedProductID = uint32(lowWord) | (uint32(highWord) << 16)
		}

		if decodedProductID == 0 {
			return nil, fmt.Errorf("invalid decoded product id=0 at record=%d", recordIndex)
		}
		if decodedProductID > math.MaxInt32 {
			return nil, fmt.Errorf("decoded product id overflows int32 at record=%d id=%d", recordIndex, decodedProductID)
		}
		decodedProductIDs = append(decodedProductIDs, int32(decodedProductID))
	}

	if cursor != len(section) {
		return nil, fmt.Errorf("product_ids section trailing bytes=%d", len(section)-cursor)
	}
	return decodedProductIDs, nil
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
	previousShapeValue := uint32(0)
	delta8Count := 0
	delta16Count := 0
	delta24Count := 0

	for recordIndex := 0; recordIndex < recordCount; recordIndex++ {
		flagBit, flagErr := reader.readBit()
		if flagErr != nil {
			return nil, 0, 0, 0, fmt.Errorf("read shape flag at record=%d: %w", recordIndex, flagErr)
		}
		deltaValue := uint32(0)
		if flagBit == 0 {
			decodedDelta8, deltaErr := reader.readBits(8)
			if deltaErr != nil {
				return nil, 0, 0, 0, fmt.Errorf("read 8-bit delta at record=%d: %w", recordIndex, deltaErr)
			}
			deltaValue = decodedDelta8
			delta8Count++
		} else {
			decodedDelta16, deltaErr := reader.readBits(16)
			if deltaErr != nil {
				return nil, 0, 0, 0, fmt.Errorf("read 16-bit delta at record=%d: %w", recordIndex, deltaErr)
			}
			if decodedDelta16 == 0xFFFF {
				decodedDelta24, escapeErr := reader.readBits(24)
				if escapeErr != nil {
					return nil, 0, 0, 0, fmt.Errorf("read 24-bit delta at record=%d: %w", recordIndex, escapeErr)
				}
				deltaValue = decodedDelta24
				delta24Count++
			} else {
				deltaValue = decodedDelta16
				delta16Count++
			}
		}
		currentShapeValue := previousShapeValue + deltaValue
		shapeValues = append(shapeValues, currentShapeValue)
		previousShapeValue = currentShapeValue
	}
	return shapeValues, delta8Count, delta16Count, delta24Count, nil
}

func decodeShapeValue(shapeValue uint32) []uint8 {
	decodedWordSizes := make([]uint8, 0, 8)
	for wordIndex := 0; wordIndex < 8; wordIndex++ {
		shift := uint(21 - (wordIndex * 3))
		wordSyllableCount := uint8((shapeValue >> shift) & 0x07)
		if wordSyllableCount == 0 {
			break
		}
		decodedWordSizes = append(decodedWordSizes, wordSyllableCount)
	}
	return decodedWordSizes
}

func decodeContentRecords(content []byte, shapes [][]uint8, dictionaryTokens []string, productIDs []int32) ([]DecodedRecord, error) {
	if len(shapes) != len(productIDs) {
		return nil, fmt.Errorf("shape/product id count mismatch shapes=%d ids=%d", len(shapes), len(productIDs))
	}

	cursor := 0
	decodedRecords := make([]DecodedRecord, 0, len(shapes))
	for recordIndex, shape := range shapes {
		decodedWords := make([]string, 0, len(shape))
		for _, wordSyllableCount := range shape {
			syllableCount := int(wordSyllableCount)
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
			decodedWords = append(decodedWords, wordBuilder.String())
		}
		decodedRecords = append(decodedRecords, DecodedRecord{
			RecordIndex: int32(recordIndex),
			ProductID:   productIDs[recordIndex],
			Shape:       append([]uint8(nil), shape...),
			Text:        strings.Join(decodedWords, " "),
		})
	}
	if cursor != len(content) {
		return nil, fmt.Errorf("unused content bytes=%d", len(content)-cursor)
	}
	return decodedRecords, nil
}

func decodeTaxonomyBlock(taxonomyBytes []byte, expectedProductCount int) (*DecodedTaxonomy, int, error) {
	const baseTaxonomyHeaderSize = len(taxonomyBinaryMagic) + 1 + 1 + 4 + 1 + 2
	if len(taxonomyBytes) < baseTaxonomyHeaderSize {
		return nil, 0, fmt.Errorf("taxonomy section too small bytes=%d", len(taxonomyBytes))
	}
	if string(taxonomyBytes[:len(taxonomyBinaryMagic)]) != taxonomyBinaryMagic {
		return nil, 0, fmt.Errorf("invalid taxonomy magic")
	}
	if taxonomyBytes[len(taxonomyBinaryMagic)] != taxonomyBinaryVersion {
		return nil, 0, fmt.Errorf("unsupported taxonomy version=%d (expected=%d)", taxonomyBytes[len(taxonomyBinaryMagic)], taxonomyBinaryVersion)
	}

	cursor := len(taxonomyBinaryMagic) + 1
	brandEncodingFlag := taxonomyBytes[cursor]
	cursor++
	sortedProductCount := int(binary.LittleEndian.Uint32(taxonomyBytes[cursor : cursor+4]))
	cursor += 4
	sectionCount := int(taxonomyBytes[cursor])
	cursor++
	headerSize := int(binary.LittleEndian.Uint16(taxonomyBytes[cursor : cursor+2]))
	cursor += 2

	if sortedProductCount != expectedProductCount {
		return nil, 0, fmt.Errorf("taxonomy/text row mismatch taxonomy=%d text=%d", sortedProductCount, expectedProductCount)
	}
	if sectionCount != 7 {
		return nil, 0, fmt.Errorf("invalid taxonomy section count=%d", sectionCount)
	}

	sectionsByID, taxonomyPayloadEnd, tableErr := decodeSectionTable(taxonomyBytes, cursor, headerSize, sectionCount)
	if tableErr != nil {
		return nil, 0, fmt.Errorf("decode taxonomy section table: %w", tableErr)
	}

	requiredTaxonomySections := []uint8{
		taxonomySectionBrandIDs,
		taxonomySectionBrandNames,
		taxonomySectionCategoryIDs,
		taxonomySectionCategoryNames,
		taxonomySectionBrandIndexes,
		taxonomySectionCategoryCounts,
		taxonomySectionCategoryIndex,
	}
	for _, sectionID := range requiredTaxonomySections {
		if _, hasSection := sectionsByID[sectionID]; !hasSection {
			return nil, 0, fmt.Errorf("missing taxonomy section=%d", sectionID)
		}
	}

	brandIDsSection := sectionBytesByID(taxonomyBytes, sectionsByID, taxonomySectionBrandIDs)
	brandNamesSection := sectionBytesByID(taxonomyBytes, sectionsByID, taxonomySectionBrandNames)
	categoryIDsSection := sectionBytesByID(taxonomyBytes, sectionsByID, taxonomySectionCategoryIDs)
	categoryNamesSection := sectionBytesByID(taxonomyBytes, sectionsByID, taxonomySectionCategoryNames)
	brandIndexesSection := sectionBytesByID(taxonomyBytes, sectionsByID, taxonomySectionBrandIndexes)
	categoryCountsSection := sectionBytesByID(taxonomyBytes, sectionsByID, taxonomySectionCategoryCounts)
	categoryIndexesSection := sectionBytesByID(taxonomyBytes, sectionsByID, taxonomySectionCategoryIndex)

	brandIDs, brandIDsErr := decodeUint16Section(brandIDsSection, "brand_ids")
	if brandIDsErr != nil {
		return nil, 0, brandIDsErr
	}
	brandNames, brandNamesErr := decodeStringColumnSection(brandNamesSection, "brand_names")
	if brandNamesErr != nil {
		return nil, 0, brandNamesErr
	}
	if len(brandIDs) != len(brandNames) {
		return nil, 0, fmt.Errorf("brand dictionary mismatch ids=%d names=%d", len(brandIDs), len(brandNames))
	}
	if int(sectionsByID[taxonomySectionBrandIDs].itemCount) != len(brandIDs) {
		return nil, 0, fmt.Errorf("brand_ids item count mismatch expected=%d got=%d", len(brandIDs), sectionsByID[taxonomySectionBrandIDs].itemCount)
	}
	if int(sectionsByID[taxonomySectionBrandNames].itemCount) != len(brandNames) {
		return nil, 0, fmt.Errorf("brand_names item count mismatch expected=%d got=%d", len(brandNames), sectionsByID[taxonomySectionBrandNames].itemCount)
	}

	categoryIDs, categoryIDsErr := decodeUint16Section(categoryIDsSection, "category_ids")
	if categoryIDsErr != nil {
		return nil, 0, categoryIDsErr
	}
	categoryNames, categoryNamesErr := decodeStringColumnSection(categoryNamesSection, "category_names")
	if categoryNamesErr != nil {
		return nil, 0, categoryNamesErr
	}
	if len(categoryIDs) != len(categoryNames) {
		return nil, 0, fmt.Errorf("category dictionary mismatch ids=%d names=%d", len(categoryIDs), len(categoryNames))
	}
	if int(sectionsByID[taxonomySectionCategoryIDs].itemCount) != len(categoryIDs) {
		return nil, 0, fmt.Errorf("category_ids item count mismatch expected=%d got=%d", len(categoryIDs), sectionsByID[taxonomySectionCategoryIDs].itemCount)
	}
	if int(sectionsByID[taxonomySectionCategoryNames].itemCount) != len(categoryNames) {
		return nil, 0, fmt.Errorf("category_names item count mismatch expected=%d got=%d", len(categoryNames), sectionsByID[taxonomySectionCategoryNames].itemCount)
	}

	if int(sectionsByID[taxonomySectionBrandIndexes].itemCount) != sortedProductCount {
		return nil, 0, fmt.Errorf("brand_indexes item count mismatch expected=%d got=%d", sortedProductCount, sectionsByID[taxonomySectionBrandIndexes].itemCount)
	}
	productBrandIndexes, decodeBrandIndexesErr := decodeBrandIndexes(brandIndexesSection, brandEncodingFlag, sortedProductCount)
	if decodeBrandIndexesErr != nil {
		return nil, 0, decodeBrandIndexesErr
	}
	for rowIndex, brandDictionaryIndex := range productBrandIndexes {
		if brandDictionaryIndex < 0 || brandDictionaryIndex >= len(brandIDs) {
			return nil, 0, fmt.Errorf("brand index out of range row=%d index=%d dict=%d", rowIndex, brandDictionaryIndex, len(brandIDs))
		}
	}

	if int(sectionsByID[taxonomySectionCategoryCounts].itemCount) != sortedProductCount {
		return nil, 0, fmt.Errorf("category_count item count mismatch expected=%d got=%d", sortedProductCount, sectionsByID[taxonomySectionCategoryCounts].itemCount)
	}
	productCategoryCounts, categoryCountsErr := decodePackedCategoryCounts(categoryCountsSection, sortedProductCount)
	if categoryCountsErr != nil {
		return nil, 0, categoryCountsErr
	}

	totalCategoryReferences := 0
	for _, categoryCount := range productCategoryCounts {
		totalCategoryReferences += categoryCount
	}
	if totalCategoryReferences != len(categoryIndexesSection) {
		return nil, 0, fmt.Errorf("category indexes payload mismatch expected=%d got=%d", totalCategoryReferences, len(categoryIndexesSection))
	}
	if int(sectionsByID[taxonomySectionCategoryIndex].itemCount) != len(categoryIndexesSection) {
		return nil, 0, fmt.Errorf("category_indexes item count mismatch expected=%d got=%d", len(categoryIndexesSection), sectionsByID[taxonomySectionCategoryIndex].itemCount)
	}
	for payloadIndex, categoryDictionaryIndex := range categoryIndexesSection {
		if int(categoryDictionaryIndex) >= len(categoryIDs) {
			return nil, 0, fmt.Errorf("category index out of range at payload index=%d index=%d dict=%d", payloadIndex, categoryDictionaryIndex, len(categoryIDs))
		}
	}

	decodedTaxonomy := &DecodedTaxonomy{
		BrandIDs:                    brandIDs,
		BrandNames:                  brandNames,
		CategoryIDs:                 categoryIDs,
		CategoryNames:               categoryNames,
		ProductBrandDictionaryIndex: productBrandIndexes,
		ProductCategoryCounts:       productCategoryCounts,
		ProductCategoryIndexes:      append([]uint8(nil), categoryIndexesSection...),
		BrandIndexEncodingFlag:      brandEncodingFlag,
	}
	return decodedTaxonomy, taxonomyPayloadEnd, nil
}

func sectionBytesByID(payload []byte, sectionsByID map[uint8]binarySectionDescriptor, sectionID uint8) []byte {
	sectionDescriptor := sectionsByID[sectionID]
	return payload[sectionDescriptor.offset : sectionDescriptor.offset+sectionDescriptor.length]
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
	selectedIndexes := randomGenerator.Perm(len(records))[:sampleCount]
	sort.Ints(selectedIndexes)

	sampledRecords := make([]DecodedRecord, 0, sampleCount)
	for _, selectedRecordIndex := range selectedIndexes {
		sampledRecords = append(sampledRecords, records[selectedRecordIndex])
	}
	return sampledRecords
}
