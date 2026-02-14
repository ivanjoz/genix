package libs

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	defaultMinCategoryID = 1
	defaultMaxCategoryID = 40
	columnarMagicHeader  = "GIX2"
	columnarVersionV2    = uint8(2)
)

// BuildBinaryFromProductosFile reads product names and writes a columnar index optimized for zstd compression.
// Goal: prioritize compression ratio and scan locality by grouping homogeneous fields into columns.
// File layout:
// [magic:4][version:1][product_count:4]
// [len_ids:4][len_heads:4][len_brands:4][len_cat1:4][len_cat2:4][len_mask_lens:4][len_masks:4][len_text_lens:4][len_text:4]
// [ids_delta_varint_rle][heads][brands][cat1][cat2][mask_lens_varint][masks_payload][text_lens_varint][text_payload]
func BuildBinaryFromProductosFile(productosFilePath, outputBinaryPath string, seed int64) error {
	log.Printf("offline-search-v2: opening productos source file: %s", productosFilePath)

	productSourceFile, openSourceError := os.Open(productosFilePath)
	if openSourceError != nil {
		return fmt.Errorf("open productos file: %w", openSourceError)
	}
	defer productSourceFile.Close()

	productScanner := bufio.NewScanner(productSourceFile)
	productScanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	randomGenerator := rand.New(rand.NewSource(seed))

	// Columns are stored independently so zstd can exploit repetition within each field type.
	productIDs := make([]uint16, 0, 12000)
	headerColumn := make([]uint8, 0, 12000)
	brandColumn := make([]uint8, 0, 12000)
	categoryOneColumn := make([]uint8, 0, 12000)
	categoryTwoColumn := make([]uint8, 0, 12000)
	maskLengthColumn := make([]uint64, 0, 12000)
	maskPayloadColumn := make([]uint8, 0, 48000)
	textLengthColumn := make([]uint64, 0, 12000)
	textPayloadColumn := make([]uint8, 0, 256000)

	nextProductID := int32(1)
	processedProductsCount := 0

	for productScanner.Scan() {
		originalProductLine := strings.TrimSpace(productScanner.Text())
		if originalProductLine == "" {
			continue
		}

		// Random test metadata is useful to validate parser robustness with non-correlated fields.
		randomBrandID := int16(randomGenerator.Intn(defaultMaxCategoryID-defaultMinCategoryID+1) + defaultMinCategoryID)
		randomCategoryOneID := int16(randomGenerator.Intn(defaultMaxCategoryID-defaultMinCategoryID+1) + defaultMinCategoryID)
		randomCategoryTwoID := int16(randomGenerator.Intn(defaultMaxCategoryID-defaultMinCategoryID+1) + defaultMinCategoryID)

		encodedProductRecord := EncodeProduct(
			nextProductID,
			randomBrandID,
			[]int16{randomCategoryOneID, randomCategoryTwoID},
			originalProductLine,
		)

		// ProductID is read back from encoded bytes to keep source of truth aligned with encoder output.
		productIDValue := uint16(encodedProductRecord.ProductID[0]) | (uint16(encodedProductRecord.ProductID[1]) << 8)
		productIDs = append(productIDs, productIDValue)
		headerColumn = append(headerColumn, encodedProductRecord.Head)
		brandColumn = append(brandColumn, encodedProductRecord.BrandID)

		categoryOne := uint8(0)
		categoryTwo := uint8(0)
		if len(encodedProductRecord.CategoriesIDs) > 0 {
			categoryOne = encodedProductRecord.CategoriesIDs[0]
		}
		if len(encodedProductRecord.CategoriesIDs) > 1 {
			categoryTwo = encodedProductRecord.CategoriesIDs[1]
		}
		categoryOneColumn = append(categoryOneColumn, categoryOne)
		categoryTwoColumn = append(categoryTwoColumn, categoryTwo)

		maskLengthColumn = append(maskLengthColumn, uint64(len(encodedProductRecord.WordCount)))
		maskPayloadColumn = append(maskPayloadColumn, encodedProductRecord.WordCount...)

		textLengthColumn = append(textLengthColumn, uint64(len(encodedProductRecord.TextEncoded)))
		textPayloadColumn = append(textPayloadColumn, encodedProductRecord.TextEncoded...)

		processedProductsCount++
		nextProductID++

		if processedProductsCount%1000 == 0 {
			log.Printf(
				"offline-search-v2: encoded=%d ids=%d heads=%d brands=%d masks=%d text=%d",
				processedProductsCount,
				len(productIDs),
				len(headerColumn),
				len(brandColumn),
				len(maskPayloadColumn),
				len(textPayloadColumn),
			)
		}
	}

	if scannerError := productScanner.Err(); scannerError != nil {
		return fmt.Errorf("scan productos file: %w", scannerError)
	}

	// ID column gets a custom delta+RLE+varint representation because IDs are mostly incremental.
	productIDDeltaColumn := encodeProductIDDeltasWithRLEVarint(productIDs)
	maskLengthsVarintColumn := encodeUVarintColumn(maskLengthColumn)
	textLengthsVarintColumn := encodeUVarintColumn(textLengthColumn)

	columnSections := [][]uint8{
		productIDDeltaColumn,
		headerColumn,
		brandColumn,
		categoryOneColumn,
		categoryTwoColumn,
		maskLengthsVarintColumn,
		maskPayloadColumn,
		textLengthsVarintColumn,
		textPayloadColumn,
	}

	totalCapacity := 4 + 1 + 4 + (len(columnSections) * 4)
	for _, sectionBytes := range columnSections {
		totalCapacity += len(sectionBytes)
	}

	finalBinaryIndex := make([]uint8, 0, totalCapacity)
	finalBinaryIndex = append(finalBinaryIndex, []byte(columnarMagicHeader)...)
	finalBinaryIndex = append(finalBinaryIndex, columnarVersionV2)

	var productCountBytes [4]byte
	binary.LittleEndian.PutUint32(productCountBytes[:], uint32(processedProductsCount))
	finalBinaryIndex = append(finalBinaryIndex, productCountBytes[:]...)

	for _, sectionBytes := range columnSections {
		// Prefix each section with its byte length to enable O(1) section slicing on load.
		var sectionLengthBytes [4]byte
		binary.LittleEndian.PutUint32(sectionLengthBytes[:], uint32(len(sectionBytes)))
		finalBinaryIndex = append(finalBinaryIndex, sectionLengthBytes[:]...)
	}
	for _, sectionBytes := range columnSections {
		finalBinaryIndex = append(finalBinaryIndex, sectionBytes...)
	}

	log.Printf(
		"offline-search-v2: writing binary file: %s (products=%d bytes=%d)",
		outputBinaryPath,
		processedProductsCount,
		len(finalBinaryIndex),
	)
	if writeBinaryError := os.WriteFile(outputBinaryPath, finalBinaryIndex, 0o644); writeBinaryError != nil {
		return fmt.Errorf("write output binary: %w", writeBinaryError)
	}

	return nil
}

func encodeUVarintColumn(values []uint64) []uint8 {
	encodedColumn := make([]uint8, 0, len(values)*2)
	var scratchBuffer [binary.MaxVarintLen64]byte
	for _, currentValue := range values {
		// Variable-length integer encoding keeps short values (common case) very compact.
		writtenBytes := binary.PutUvarint(scratchBuffer[:], currentValue)
		encodedColumn = append(encodedColumn, scratchBuffer[:writtenBytes]...)
	}
	return encodedColumn
}

// encodeProductIDDeltasWithRLEVarint stores ID deltas compactly.
// Control byte 0 => run of delta=1 with next uvarint(runLen).
// Control byte 1 => explicit delta with next uvarint(delta).
func encodeProductIDDeltasWithRLEVarint(productIDs []uint16) []uint8 {
	if len(productIDs) == 0 {
		return []uint8{}
	}

	encodedDeltaColumn := make([]uint8, 0, len(productIDs))
	var scratchBuffer [binary.MaxVarintLen64]byte

	appendExplicitDelta := func(deltaValue uint16) {
		// Control=1 means "single explicit delta value follows".
		encodedDeltaColumn = append(encodedDeltaColumn, 1)
		writtenBytes := binary.PutUvarint(scratchBuffer[:], uint64(deltaValue))
		encodedDeltaColumn = append(encodedDeltaColumn, scratchBuffer[:writtenBytes]...)
	}
	appendRunOfOnes := func(runLength uint64) {
		// Control=0 means "run of +1 deltas follows".
		encodedDeltaColumn = append(encodedDeltaColumn, 0)
		writtenBytes := binary.PutUvarint(scratchBuffer[:], runLength)
		encodedDeltaColumn = append(encodedDeltaColumn, scratchBuffer[:writtenBytes]...)
	}

	previousProductID := uint16(0)
	currentRunOfOneDeltas := uint64(0)

	for _, currentProductID := range productIDs {
		currentDelta := currentProductID - previousProductID
		previousProductID = currentProductID

		if currentDelta == 1 {
			currentRunOfOneDeltas++
			continue
		}

		if currentRunOfOneDeltas > 0 {
			appendRunOfOnes(currentRunOfOneDeltas)
			currentRunOfOneDeltas = 0
		}
		// Non-sequential jump is encoded explicitly to preserve exact ID sequence.
		appendExplicitDelta(currentDelta)
	}

	if currentRunOfOneDeltas > 0 {
		appendRunOfOnes(currentRunOfOneDeltas)
	}

	return encodedDeltaColumn
}

// BuildBinaryFromDefaultProductosFile is a convenience wrapper.
func BuildBinaryFromDefaultProductosFile() error {
	return BuildBinaryFromProductosFile("libs/productos.txt", "libs/products_search.idx", time.Now().UnixNano())
}
