package index_builder

import (
	"fmt"
	"sort"
)

const (
	maxTopCategories              = 243
	othersCategoryIndex     uint8 = 243
	maxCategoriesPerProduct       = 4
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

	// ProductBrandIndexesU8 keeps the first product brand-index rows that fit into uint8.
	// ProductBrandIndexesU16 stores the remaining rows; decoder concatenates both arrays.
	ProductBrandIndexesU8  []uint8
	ProductBrandIndexesU16 []uint16

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

	productBrandIndexesU8, productBrandIndexesU16, splitErr := splitBrandIndexesColumns(productBrandIndexes)
	if splitErr != nil {
		return nil, splitErr
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
		SortedProductIDs:       append([]int32(nil), sortedProductIDs...),
		BrandIDs:               orderedBrandIDs,
		BrandNames:             orderedBrandNames,
		CategoryIDs:            categoryIDs,
		CategoryNames:          categoryNames,
		ProductBrandIndexesU8:  productBrandIndexesU8,
		ProductBrandIndexesU16: productBrandIndexesU16,
		ProductCategoryCount:   packedCategoryCount,
		ProductCategoryIndexes: productCategoryIndexes,
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

func splitBrandIndexesColumns(productBrandIndexes []int) ([]uint8, []uint16, error) {
	productBrandIndexesU8 := make([]uint8, 0, len(productBrandIndexes))
	productBrandIndexesU16 := make([]uint16, 0, len(productBrandIndexes))

	startedU16Segment := false
	for _, brandIndex := range productBrandIndexes {
		if brandIndex < 0 || brandIndex > 65535 {
			return nil, nil, fmt.Errorf("brand index=%d overflows uint16", brandIndex)
		}
		// Keep concat-friendly ordering: once u16 segment starts, all remaining rows go to u16.
		if !startedU16Segment && brandIndex <= 255 {
			productBrandIndexesU8 = append(productBrandIndexesU8, uint8(brandIndex))
			continue
		}
		startedU16Segment = true
		productBrandIndexesU16 = append(productBrandIndexesU16, uint16(brandIndex))
	}
	return productBrandIndexesU8, productBrandIndexesU16, nil
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
