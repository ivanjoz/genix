package index_builder

import (
	"fmt"
	"sort"
)

const (
	maxTopCategories              = 243
	othersCategoryIndex     uint8 = 243
	maxCategoriesPerProduct       = 4
	maxBrandIndexUint12           = 4095
	maxBrandIndexUint16           = 65535

	// BrandIndexEncodingUint12 means product brand-indexes are packed as 12-bit values.
	BrandIndexEncodingUint12 uint8 = 1
	// BrandIndexEncodingUint16 means product brand-indexes are encoded as uint16 values.
	BrandIndexEncodingUint16 uint8 = 2
)

// BuildInput is the stage-2 taxonomy input envelope.
// The same RecordInput type is reused for products, brands, and categories.
type BuildInput struct {
	Products   []RecordInput
	Brands     []RecordInput
	Categories []RecordInput
}

// TaxonomyBuildResult stores columnar taxonomy sections aligned to sorted products.
type TaxonomyBuildResult struct {
	SortedProductIDs []int32

	BrandIDs   []uint16
	BrandNames []string

	CategoryIDs   []uint16
	CategoryNames []string

	// BrandIndexEncodingFlag selects product brand-index column format.
	// 1 => ProductBrandIndexesUint12Packed
	// 2 => ProductBrandIndexesUint16
	BrandIndexEncodingFlag uint8
	// ProductBrandIndexesUint12Packed stores 2 indexes per 3 bytes.
	ProductBrandIndexesUint12Packed []uint8
	// ProductBrandIndexesUint16 stores one uint16 per product row.
	ProductBrandIndexesUint16 []uint16

	// ProductCategoryCount stores 2-bit counters (count-1) packed in groups of 4 products per byte.
	ProductCategoryCount []uint8
	// ProductCategoryIndexes stores flattened mapped category indexes, consumed using ProductCategoryCount.
	ProductCategoryIndexes []uint8
}

type rankedCategoryRow struct {
	categoryID int32
	usageCount int
}

// BuildTaxonomySecondPass builds taxonomy columns aligned to stage-1 SortedIDs.
func BuildTaxonomySecondPass(sortedProductIDs []int32, input BuildInput) (*TaxonomyBuildResult, error) {
	if len(sortedProductIDs) == 0 {
		return nil, fmt.Errorf("taxonomy pass requires non-empty sorted product IDs")
	}
	if len(input.Products) == 0 {
		return nil, fmt.Errorf("taxonomy pass requires products")
	}

	productByID := make(map[int32]RecordInput, len(input.Products))
	for _, productRecord := range input.Products {
		if _, alreadyExists := productByID[productRecord.ID]; alreadyExists {
			return nil, fmt.Errorf("duplicated product ID=%d", productRecord.ID)
		}
		productByID[productRecord.ID] = productRecord
	}

	productsInSortedOrder := make([]RecordInput, 0, len(sortedProductIDs))
	for _, sortedProductID := range sortedProductIDs {
		productRecord, exists := productByID[sortedProductID]
		if !exists {
			return nil, fmt.Errorf("sorted product ID=%d not found in taxonomy products", sortedProductID)
		}
		productsInSortedOrder = append(productsInSortedOrder, productRecord)
	}

	brandNameByID, brandNameErr := buildNameLookupByID(input.Brands, "brand")
	if brandNameErr != nil {
		return nil, brandNameErr
	}
	categoryNameByID, categoryNameErr := buildNameLookupByID(input.Categories, "category")
	if categoryNameErr != nil {
		return nil, categoryNameErr
	}

	// Build brand dictionary by first appearance in the stage-1 sorted product sequence.
	brandIndexByOriginalID := make(map[int32]int, 256)
	orderedBrandIDs := make([]uint16, 0, 256)
	orderedBrandNames := make([]string, 0, 256)
	productBrandIndexes := make([]int, 0, len(productsInSortedOrder))
	for _, productRecord := range productsInSortedOrder {
		existingBrandIndex, exists := brandIndexByOriginalID[productRecord.BrandID]
		if exists {
			productBrandIndexes = append(productBrandIndexes, existingBrandIndex)
			continue
		}

		brandName, brandExists := brandNameByID[productRecord.BrandID]
		if !brandExists {
			return nil, fmt.Errorf("product ID=%d references unknown brand ID=%d", productRecord.ID, productRecord.BrandID)
		}
		if productRecord.BrandID < 0 || productRecord.BrandID > 65535 {
			return nil, fmt.Errorf("brand ID=%d overflows uint16", productRecord.BrandID)
		}

		newBrandIndex := len(orderedBrandIDs)
		brandIndexByOriginalID[productRecord.BrandID] = newBrandIndex
		orderedBrandIDs = append(orderedBrandIDs, uint16(productRecord.BrandID))
		orderedBrandNames = append(orderedBrandNames, brandName)
		productBrandIndexes = append(productBrandIndexes, newBrandIndex)
	}

	brandIndexEncodingFlag, packedBrandIndexesUint12, productBrandIndexesUint16, encodeBrandIndexesErr := encodeProductBrandIndexes(
		productBrandIndexes,
		len(orderedBrandIDs),
	)
	if encodeBrandIndexesErr != nil {
		return nil, encodeBrandIndexesErr
	}

	// Rank categories by usage frequency, then category ID for deterministic ties.
	categoryFrequencyByID := make(map[int32]int, 512)
	for _, productRecord := range productsInSortedOrder {
		for _, categoryID := range productRecord.CategoriesIDs {
			categoryFrequencyByID[categoryID]++
		}
	}

	rankedCategories := make([]rankedCategoryRow, 0, len(categoryFrequencyByID))
	for categoryID, usageCount := range categoryFrequencyByID {
		rankedCategories = append(rankedCategories, rankedCategoryRow{categoryID: categoryID, usageCount: usageCount})
	}
	sort.SliceStable(rankedCategories, func(leftIndex, rightIndex int) bool {
		leftRow := rankedCategories[leftIndex]
		rightRow := rankedCategories[rightIndex]
		if leftRow.usageCount != rightRow.usageCount {
			return leftRow.usageCount > rightRow.usageCount
		}
		return leftRow.categoryID < rightRow.categoryID
	})

	topCategoriesCount := len(rankedCategories)
	if topCategoriesCount > maxTopCategories {
		topCategoriesCount = maxTopCategories
	}

	// Keep a fixed 244-slot dictionary where slot 243 is always "Otros".
	categoryIDs := make([]uint16, maxTopCategories+1)
	categoryNames := make([]string, maxTopCategories+1)
	categoryIDs[othersCategoryIndex] = 0
	categoryNames[othersCategoryIndex] = "Otros"

	categoryIndexByOriginalID := make(map[int32]uint8, topCategoriesCount)
	for rowIndex := 0; rowIndex < topCategoriesCount; rowIndex++ {
		categoryID := rankedCategories[rowIndex].categoryID
		if categoryID < 0 || categoryID > 65535 {
			return nil, fmt.Errorf("category ID=%d overflows uint16", categoryID)
		}
		categoryName, hasCategoryName := categoryNameByID[categoryID]
		if !hasCategoryName {
			return nil, fmt.Errorf("missing category name for category ID=%d", categoryID)
		}
		categoryIDs[rowIndex] = uint16(categoryID)
		categoryNames[rowIndex] = categoryName
		categoryIndexByOriginalID[categoryID] = uint8(rowIndex)
	}

	productCategoryCountMinusOne := make([]uint8, 0, len(productsInSortedOrder))
	productCategoryIndexes := make([]uint8, 0, len(productsInSortedOrder)*2)
	for _, productRecord := range productsInSortedOrder {
		uniqueMappedCategories := make([]uint8, 0, maxCategoriesPerProduct)
		seenMappedCategory := make(map[uint8]struct{}, len(productRecord.CategoriesIDs))
		for _, categoryID := range productRecord.CategoriesIDs {
			mappedCategoryIndex, exists := categoryIndexByOriginalID[categoryID]
			if !exists {
				mappedCategoryIndex = othersCategoryIndex
			}
			if _, alreadyAdded := seenMappedCategory[mappedCategoryIndex]; alreadyAdded {
				continue
			}
			seenMappedCategory[mappedCategoryIndex] = struct{}{}
			uniqueMappedCategories = append(uniqueMappedCategories, mappedCategoryIndex)
			if len(uniqueMappedCategories) == maxCategoriesPerProduct {
				break
			}
		}

		// Ensure each product contributes at least one category slot to keep 2-bit count encoding valid.
		if len(uniqueMappedCategories) == 0 {
			uniqueMappedCategories = append(uniqueMappedCategories, othersCategoryIndex)
		}

		productCategoryCountMinusOne = append(productCategoryCountMinusOne, uint8(len(uniqueMappedCategories)-1))
		productCategoryIndexes = append(productCategoryIndexes, uniqueMappedCategories...)
	}

	packedCategoryCount, packErr := packTwoBitCategoryCounts(productCategoryCountMinusOne)
	if packErr != nil {
		return nil, packErr
	}

	return &TaxonomyBuildResult{
		SortedProductIDs:                append([]int32(nil), sortedProductIDs...),
		BrandIDs:                        orderedBrandIDs,
		BrandNames:                      orderedBrandNames,
		CategoryIDs:                     categoryIDs,
		CategoryNames:                   categoryNames,
		BrandIndexEncodingFlag:          brandIndexEncodingFlag,
		ProductBrandIndexesUint12Packed: packedBrandIndexesUint12,
		ProductBrandIndexesUint16:       productBrandIndexesUint16,
		ProductCategoryCount:            packedCategoryCount,
		ProductCategoryIndexes:          productCategoryIndexes,
	}, nil
}

func buildNameLookupByID(records []RecordInput, entityLabel string) (map[int32]string, error) {
	nameByID := make(map[int32]string, len(records))
	for _, record := range records {
		if _, exists := nameByID[record.ID]; exists {
			return nil, fmt.Errorf("duplicated %s ID=%d", entityLabel, record.ID)
		}
		nameByID[record.ID] = record.Text
	}
	return nameByID, nil
}

func encodeProductBrandIndexes(productBrandIndexes []int, uniqueBrandCount int) (uint8, []uint8, []uint16, error) {
	if uniqueBrandCount <= 0 {
		return 0, nil, nil, fmt.Errorf("invalid unique brand count=%d", uniqueBrandCount)
	}
	if uniqueBrandCount <= maxBrandIndexUint12 {
		// Use 12-bit mode when total dictionary cardinality fits 0..4094 index range.
		packedIndexes, packErr := packIntSliceToUint12(
			productBrandIndexes,
			"brand index",
			maxBrandIndexUint12,
		)
		if packErr != nil {
			return 0, nil, nil, packErr
		}
		return BrandIndexEncodingUint12, packedIndexes, nil, nil
	}

	// Fallback to uint16 mode for large brand dictionaries.
	encodedIndexes, encodeErr := convertIntSliceToUint16(
		productBrandIndexes,
		"brand index",
		maxBrandIndexUint16,
	)
	if encodeErr != nil {
		return 0, nil, nil, encodeErr
	}
	return BrandIndexEncodingUint16, nil, encodedIndexes, nil
}

func (taxonomyBuildResult *TaxonomyBuildResult) ProductBrandIndexesCount() int {
	// Count is inferred from encoded column payload and selected format.
	if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint12 {
		if len(taxonomyBuildResult.SortedProductIDs) > 0 {
			return len(taxonomyBuildResult.SortedProductIDs)
		}
		return (len(taxonomyBuildResult.ProductBrandIndexesUint12Packed) / 3) * 2
	}
	if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint16 {
		return len(taxonomyBuildResult.ProductBrandIndexesUint16)
	}
	return 0
}

func (taxonomyBuildResult *TaxonomyBuildResult) ProductBrandIndexesBytes() int {
	// Byte accounting depends on the active encoding flag.
	if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint12 {
		return len(taxonomyBuildResult.ProductBrandIndexesUint12Packed)
	}
	if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint16 {
		return len(taxonomyBuildResult.ProductBrandIndexesUint16) * 2
	}
	return 0
}

func (taxonomyBuildResult *TaxonomyBuildResult) BrandIndexEncodingName() string {
	// Human-readable mode name for logs/stats payloads.
	if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint12 {
		return "uint12"
	}
	if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint16 {
		return "uint16"
	}
	return "unknown"
}

func (taxonomyBuildResult *TaxonomyBuildResult) ValidateForBinary() error {
	if taxonomyBuildResult == nil {
		return fmt.Errorf("nil taxonomy result")
	}
	if len(taxonomyBuildResult.SortedProductIDs) == 0 {
		return fmt.Errorf("taxonomy payload requires sorted product IDs")
	}
	if len(taxonomyBuildResult.BrandIDs) != len(taxonomyBuildResult.BrandNames) {
		return fmt.Errorf("brand dictionary columns length mismatch")
	}
	if len(taxonomyBuildResult.CategoryIDs) != len(taxonomyBuildResult.CategoryNames) {
		return fmt.Errorf("category dictionary columns length mismatch")
	}
	if taxonomyBuildResult.ProductBrandIndexesCount() != len(taxonomyBuildResult.SortedProductIDs) {
		return fmt.Errorf("brand indexes count mismatch sorted products")
	}

	expectedPackedCategoryCountBytes := (len(taxonomyBuildResult.SortedProductIDs) + 3) / 4
	if len(taxonomyBuildResult.ProductCategoryCount) != expectedPackedCategoryCountBytes {
		return fmt.Errorf(
			"category count bytes mismatch expected=%d got=%d",
			expectedPackedCategoryCountBytes,
			len(taxonomyBuildResult.ProductCategoryCount),
		)
	}

	if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint12 {
		if len(taxonomyBuildResult.ProductBrandIndexesUint16) > 0 {
			return fmt.Errorf("invalid uint12 mode with uint16 brand indexes")
		}
		expectedPackedBrandBytes := expectedUint12PackedBytes(len(taxonomyBuildResult.SortedProductIDs))
		if len(taxonomyBuildResult.ProductBrandIndexesUint12Packed) != expectedPackedBrandBytes {
			return fmt.Errorf(
				"uint12 brand bytes mismatch expected=%d got=%d",
				expectedPackedBrandBytes,
				len(taxonomyBuildResult.ProductBrandIndexesUint12Packed),
			)
		}
	} else if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint16 {
		if len(taxonomyBuildResult.ProductBrandIndexesUint12Packed) > 0 {
			return fmt.Errorf("invalid uint16 mode with uint12 brand indexes")
		}
		if len(taxonomyBuildResult.ProductBrandIndexesUint16) != len(taxonomyBuildResult.SortedProductIDs) {
			return fmt.Errorf(
				"uint16 brand rows mismatch expected=%d got=%d",
				len(taxonomyBuildResult.SortedProductIDs),
				len(taxonomyBuildResult.ProductBrandIndexesUint16),
			)
		}
	} else {
		return fmt.Errorf("unsupported brand index encoding flag=%d", taxonomyBuildResult.BrandIndexEncodingFlag)
	}

	totalMappedCategoryIndexes := decodeTotalCategoryIndexes(
		taxonomyBuildResult.ProductCategoryCount,
		len(taxonomyBuildResult.SortedProductIDs),
	)
	if len(taxonomyBuildResult.ProductCategoryIndexes) != totalMappedCategoryIndexes {
		return fmt.Errorf(
			"category indexes length mismatch expected=%d got=%d",
			totalMappedCategoryIndexes,
			len(taxonomyBuildResult.ProductCategoryIndexes),
		)
	}

	categoryDictionarySize := len(taxonomyBuildResult.CategoryIDs)
	for categoryIndexPosition, mappedCategoryIndex := range taxonomyBuildResult.ProductCategoryIndexes {
		if int(mappedCategoryIndex) >= categoryDictionarySize {
			return fmt.Errorf(
				"mapped category index=%d out of bounds at position=%d dictionary=%d",
				mappedCategoryIndex,
				categoryIndexPosition,
				categoryDictionarySize,
			)
		}
	}
	return nil
}

func decodeTotalCategoryIndexes(packedCategoryCount []uint8, productCount int) int {
	totalIndexes := 0
	for productIndex := 0; productIndex < productCount; productIndex++ {
		shift := uint(6 - (productIndex%4)*2)
		countMinusOne := (packedCategoryCount[productIndex/4] >> shift) & 0x03
		totalIndexes += int(countMinusOne) + 1
	}
	return totalIndexes
}

func packTwoBitCategoryCounts(productCategoryCountMinusOne []uint8) ([]uint8, error) {
	packedBytes := make([]uint8, (len(productCategoryCountMinusOne)+3)/4)
	for productIndex, countMinusOne := range productCategoryCountMinusOne {
		if countMinusOne > 3 {
			return nil, fmt.Errorf("invalid category count encoding at product index=%d", productIndex)
		}
		shift := uint(6 - (productIndex%4)*2)
		packedBytes[productIndex/4] |= (countMinusOne & 0x03) << shift
	}
	return packedBytes, nil
}
