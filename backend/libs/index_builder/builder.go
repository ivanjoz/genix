package index_builder

import (
	"fmt"
	"strings"
)

type ProductTextOptimizationAggregate struct {
	ChangedProducts          int
	FallbackOriginalProducts int
}

type ProductosIndexBuild struct {
	// Stage-1 text index fields.
	SortedIDs         []int32
	Shapes            []byte
	Content           []byte
	BuildSunixTime    int32
	HeaderFlags       uint8
	DictionaryTokens  []string
	DictionarySection []byte
	Stats             BuildStats

	// Stage-2 taxonomy fields.
	BrandIDs                        []uint16
	BrandNames                      []string
	CategoryIDs                     []uint16
	CategoryNames                   []string
	BrandIndexEncodingFlag          uint8
	ProductBrandIndexesUint12Packed []uint8
	ProductBrandIndexesUint16       []uint16
	ProductCategoryCount            []uint8
	ProductCategoryIndexes          []uint8

	// Shared build telemetry.
	OptimizationStats ProductTextOptimizationAggregate
}

// BuildProductosIndex runs a sequential end-to-end build:
// 1) optimize product text using brand-aware normalization
// 2) build text index
// 3) build taxonomy columns aligned to text sorted IDs
func BuildProductosIndex(buildInput BuildInput) (*ProductosIndexBuild, error) {
	if len(buildInput.Products) == 0 {
		return nil, fmt.Errorf("build input requires products")
	}

	brandNameByID := make(map[int32]string, len(buildInput.Brands)+1)
	for _, currentBrandRecord := range buildInput.Brands {
		brandNameByID[currentBrandRecord.ID] = currentBrandRecord.Text
	}
	// Keep a sentinel brand label for products without explicit brand assignment.
	if _, hasSentinelBrand := brandNameByID[0]; !hasSentinelBrand {
		brandNameByID[0] = "Sin marca"
	}

	optimizedProductRecords := make([]RecordInput, 0, len(buildInput.Products))
	optimizationStats := ProductTextOptimizationAggregate{}

	for _, currentProductRecord := range buildInput.Products {
		// Strip brand terms + duplicate tokens before stage-1 encoding to reduce content bytes.
		brandNameForProduct := brandNameByID[currentProductRecord.BrandID]
		productTokens := splitNormalizedTokens(currentProductRecord.Text)
		brandTokens := splitNormalizedTokens(brandNameForProduct)
		if len(productTokens) > 0 && len(brandTokens) > 0 {
			filteredTokens, _ := removeAllTokenSequenceOccurrences(productTokens, brandTokens)
			productTokens = filteredTokens
		}

		dedupedTokens := make([]string, 0, len(productTokens))
		seenNormalizedTokens := make(map[string]struct{}, len(productTokens))
		for _, normalizedToken := range productTokens {
			if _, alreadySeen := seenNormalizedTokens[normalizedToken]; alreadySeen {
				continue
			}
			seenNormalizedTokens[normalizedToken] = struct{}{}
			dedupedTokens = append(dedupedTokens, normalizedToken)
		}

		optimizedProductText := strings.Join(dedupedTokens, " ")
		if optimizedProductText != normalizeText(currentProductRecord.Text) {
			optimizationStats.ChangedProducts++
		}

		// Keep original text when aggressive optimization removes all searchable tokens.
		if optimizedProductText == "" {
			optimizationStats.FallbackOriginalProducts++
			optimizedProductText = currentProductRecord.Text
		}

		optimizedProductRecords = append(optimizedProductRecords, RecordInput{
			ID:            currentProductRecord.ID,
			Text:          optimizedProductText,
			BrandID:       currentProductRecord.BrandID,
			CategoriesIDs: append([]int32(nil), currentProductRecord.CategoriesIDs...),
		})
	}

	textBuildOptions := DefaultOptions()
	textBuildResult, textBuildErr := BuildIndex(optimizedProductRecords, textBuildOptions)
	if textBuildErr != nil {
		return nil, fmt.Errorf("build text index: %w", textBuildErr)
	}

	taxonomyBuildErr := BuildTaxonomySecondPass(textBuildResult, BuildInput{
		Products:   optimizedProductRecords,
		Brands:     buildInput.Brands,
		Categories: buildInput.Categories,
	})
	if taxonomyBuildErr != nil {
		return nil, fmt.Errorf("build taxonomy index: %w", taxonomyBuildErr)
	}

	// Stage-2 enriches the same struct instance produced by stage-1.
	textBuildResult.OptimizationStats = optimizationStats

	return textBuildResult, nil
}

func splitNormalizedTokens(rawText string) []string {
	normalizedText := normalizeText(rawText)
	if normalizedText == "" {
		return nil
	}
	return strings.Fields(normalizedText)
}

func removeAllTokenSequenceOccurrences(sourceTokens []string, sequenceTokens []string) ([]string, int) {
	if len(sourceTokens) == 0 || len(sequenceTokens) == 0 || len(sequenceTokens) > len(sourceTokens) {
		return append([]string(nil), sourceTokens...), 0
	}

	filteredTokens := make([]string, 0, len(sourceTokens))
	removedOccurrences := 0
	currentIndex := 0
	for currentIndex < len(sourceTokens) {
		if currentIndex+len(sequenceTokens) <= len(sourceTokens) &&
			tokenSequenceEquals(sourceTokens[currentIndex:currentIndex+len(sequenceTokens)], sequenceTokens) {
			removedOccurrences++
			currentIndex += len(sequenceTokens)
			continue
		}
		filteredTokens = append(filteredTokens, sourceTokens[currentIndex])
		currentIndex++
	}
	return filteredTokens, removedOccurrences
}

func tokenSequenceEquals(leftTokens []string, rightTokens []string) bool {
	if len(leftTokens) != len(rightTokens) {
		return false
	}
	for tokenIndex := 0; tokenIndex < len(leftTokens); tokenIndex++ {
		if leftTokens[tokenIndex] != rightTokens[tokenIndex] {
			return false
		}
	}
	return true
}
