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

type rankedCategoryRow struct {
	categoryID int32
	usageCount int
}

// BuildTaxonomySecondPass builds taxonomy columns aligned to stage-1 SortedIDs and
// fills the same ProductosIndexBuild instance in-place.
func BuildTaxonomySecondPass(indexBuild *ProductosIndexBuild, input BuildInput) error {
	if indexBuild == nil {
		return fmt.Errorf("taxonomy pass requires non-nil index build")
	}
	sortedProductIDs := indexBuild.SortedIDs
	if len(sortedProductIDs) == 0 {
		return fmt.Errorf("taxonomy pass requires non-empty sorted product IDs")
	}
	if len(input.Products) == 0 {
		return fmt.Errorf("taxonomy pass requires products")
	}

	productByID := make(map[int32]RecordInput, len(input.Products))
	for _, productRecord := range input.Products {
		if _, alreadyExists := productByID[productRecord.ID]; alreadyExists {
			return fmt.Errorf("duplicated product ID=%d", productRecord.ID)
		}
		productByID[productRecord.ID] = productRecord
	}

	productsInSortedOrder := make([]RecordInput, 0, len(sortedProductIDs))
	for _, sortedProductID := range sortedProductIDs {
		productRecord, exists := productByID[sortedProductID]
		if !exists {
			return fmt.Errorf("sorted product ID=%d not found in taxonomy products", sortedProductID)
		}
		productsInSortedOrder = append(productsInSortedOrder, productRecord)
	}

	brandNameByID, brandNameErr := buildNameLookupByID(input.Brands, "brand")
	if brandNameErr != nil {
		return brandNameErr
	}
	categoryNameByID, categoryNameErr := buildNameLookupByID(input.Categories, "category")
	if categoryNameErr != nil {
		return categoryNameErr
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
			return fmt.Errorf("product ID=%d references unknown brand ID=%d", productRecord.ID, productRecord.BrandID)
		}
		if productRecord.BrandID < 0 || productRecord.BrandID > 65535 {
			return fmt.Errorf("brand ID=%d overflows uint16", productRecord.BrandID)
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
		return encodeBrandIndexesErr
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
			return fmt.Errorf("category ID=%d overflows uint16", categoryID)
		}
		categoryName, hasCategoryName := categoryNameByID[categoryID]
		if !hasCategoryName {
			return fmt.Errorf("missing category name for category ID=%d", categoryID)
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
		return packErr
	}
	// Enrich the same build struct created by stage-1 with taxonomy columns.
	indexBuild.BrandIDs = orderedBrandIDs
	indexBuild.BrandNames = orderedBrandNames
	indexBuild.CategoryIDs = categoryIDs
	indexBuild.CategoryNames = categoryNames
	indexBuild.BrandIndexEncodingFlag = brandIndexEncodingFlag
	indexBuild.ProductBrandIndexesUint12Packed = packedBrandIndexesUint12
	indexBuild.ProductBrandIndexesUint16 = productBrandIndexesUint16
	indexBuild.ProductCategoryCount = packedCategoryCount
	indexBuild.ProductCategoryIndexes = productCategoryIndexes
	return nil
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

func (taxonomyBuildResult *ProductosIndexBuild) ProductBrandIndexesCount() int {
	if taxonomyBuildResult == nil {
		return 0
	}
	// Product brand indexes are aligned 1:1 with sorted products.
	return len(taxonomyBuildResult.SortedIDs)
}

func (taxonomyBuildResult *ProductosIndexBuild) ProductBrandIndexesBytes() int {
	if taxonomyBuildResult == nil {
		return 0
	}
	switch taxonomyBuildResult.BrandIndexEncodingFlag {
	case BrandIndexEncodingUint12:
		return len(taxonomyBuildResult.ProductBrandIndexesUint12Packed)
	case BrandIndexEncodingUint16:
		return len(taxonomyBuildResult.ProductBrandIndexesUint16) * 2
	default:
		return 0
	}
}

func (taxonomyBuildResult *ProductosIndexBuild) BrandIndexEncodingName() string {
	if taxonomyBuildResult == nil {
		return "unknown"
	}
	switch taxonomyBuildResult.BrandIndexEncodingFlag {
	case BrandIndexEncodingUint12:
		return "uint12"
	case BrandIndexEncodingUint16:
		return "uint16"
	default:
		return "unknown"
	}
}

func (taxonomyBuildResult *ProductosIndexBuild) ValidateForBinary() error {
	if taxonomyBuildResult == nil {
		return fmt.Errorf("nil taxonomy result")
	}
	if len(taxonomyBuildResult.SortedIDs) == 0 {
		return fmt.Errorf("taxonomy payload requires sorted product IDs")
	}
	if len(taxonomyBuildResult.BrandIDs) != len(taxonomyBuildResult.BrandNames) {
		return fmt.Errorf("brand dictionary columns length mismatch")
	}
	if len(taxonomyBuildResult.CategoryIDs) != len(taxonomyBuildResult.CategoryNames) {
		return fmt.Errorf("category dictionary columns length mismatch")
	}

	productCount := len(taxonomyBuildResult.SortedIDs)
	switch taxonomyBuildResult.BrandIndexEncodingFlag {
	case BrandIndexEncodingUint12:
		if len(taxonomyBuildResult.ProductBrandIndexesUint16) > 0 {
			return fmt.Errorf("invalid uint12 mode with uint16 brand indexes")
		}
		expectedPackedBrandBytes := expectedUint12PackedBytes(productCount)
		if len(taxonomyBuildResult.ProductBrandIndexesUint12Packed) != expectedPackedBrandBytes {
			return fmt.Errorf(
				"uint12 brand bytes mismatch expected=%d got=%d",
				expectedPackedBrandBytes,
				len(taxonomyBuildResult.ProductBrandIndexesUint12Packed),
			)
		}
	case BrandIndexEncodingUint16:
		if len(taxonomyBuildResult.ProductBrandIndexesUint12Packed) > 0 {
			return fmt.Errorf("invalid uint16 mode with uint12 brand indexes")
		}
		if len(taxonomyBuildResult.ProductBrandIndexesUint16) != productCount {
			return fmt.Errorf(
				"uint16 brand rows mismatch expected=%d got=%d",
				productCount,
				len(taxonomyBuildResult.ProductBrandIndexesUint16),
			)
		}
	default:
		return fmt.Errorf("unsupported brand index encoding flag=%d", taxonomyBuildResult.BrandIndexEncodingFlag)
	}

	expectedPackedCategoryCountBytes := (productCount + 3) / 4
	if len(taxonomyBuildResult.ProductCategoryCount) != expectedPackedCategoryCountBytes {
		return fmt.Errorf(
			"category count bytes mismatch expected=%d got=%d",
			expectedPackedCategoryCountBytes,
			len(taxonomyBuildResult.ProductCategoryCount),
		)
	}
	if len(taxonomyBuildResult.ProductCategoryIndexes) < productCount {
		return fmt.Errorf(
			"category indexes payload too small productCount=%d indexes=%d",
			productCount,
			len(taxonomyBuildResult.ProductCategoryIndexes),
		)
	}
	return nil
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
