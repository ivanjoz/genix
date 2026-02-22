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
	ProductIDsSection []byte
	Shapes            []byte
	Content           []byte
	BuildSunixTime    int32
	HeaderFlags       uint8
	DictionaryTokens  []string
	DictionarySection []byte
	AliasSection      []byte
	AliasCount        int32
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

	brandNormalizedNameByID := make(map[int32]string, len(buildInput.Brands)+1)
	for _, currentBrandRecord := range buildInput.Brands {
		brandNormalizedNameByID[currentBrandRecord.ID] = normalizeText(currentBrandRecord.Text)
	}
	// Keep a normalized sentinel brand label for products without explicit brand assignment.
	if _, hasSentinelBrand := brandNormalizedNameByID[0]; !hasSentinelBrand {
		brandNormalizedNameByID[0] = normalizeText("Sin marca")
	}

	optimizedProductRecords := make([]RecordInput, 0, len(buildInput.Products))
	optimizationStats := ProductTextOptimizationAggregate{}

	for _, currentProductRecord := range buildInput.Products {
		// Apply centralized cleaning so all product-text optimization rules stay in one function.
		normalizedBrandNameForProduct := brandNormalizedNameByID[currentProductRecord.BrandID]
		optimizedProductText := CleanProductTextByBrand(currentProductRecord.Text, normalizedBrandNameForProduct)
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

// CleanProductTextByBrand normalizes product text and removes:
// 1) brand token sequences and standalone brand tokens
// 2) connector words from default options
// 3) single-letter tokens
// The brand input must already be normalized (normalizeText output).
func CleanProductTextByBrand(rawProductName string, normalizedBrandName string) string {
	productTokens := strings.Fields(normalizeText(rawProductName))
	if len(productTokens) == 0 {
		return ""
	}

	brandTokens := strings.Fields(normalizedBrandName)
	if len(brandTokens) > 0 {
		// Remove contiguous brand-name sequence occurrences across the product tokens.
		productTokensWithoutBrandSequence := make([]string, 0, len(productTokens))
		for productTokenIndex := 0; productTokenIndex < len(productTokens); {
			canCompareBrandSequence := productTokenIndex+len(brandTokens) <= len(productTokens)
			brandSequenceMatches := canCompareBrandSequence
			if canCompareBrandSequence {
				for brandTokenIndex := 0; brandTokenIndex < len(brandTokens); brandTokenIndex++ {
					if productTokens[productTokenIndex+brandTokenIndex] != brandTokens[brandTokenIndex] {
						brandSequenceMatches = false
						break
					}
				}
			}
			if brandSequenceMatches {
				productTokenIndex += len(brandTokens)
				continue
			}
			productTokensWithoutBrandSequence = append(productTokensWithoutBrandSequence, productTokens[productTokenIndex])
			productTokenIndex++
		}
		productTokens = productTokensWithoutBrandSequence

		brandTokenSet := make(map[string]struct{}, len(brandTokens))
		for _, brandToken := range brandTokens {
			brandTokenSet[brandToken] = struct{}{}
		}

		// Remove remaining standalone brand tokens to avoid leftover brand noise.
		filteredByBrandWords := make([]string, 0, len(productTokens))
		for _, productToken := range productTokens {
			if _, isBrandToken := brandTokenSet[productToken]; isBrandToken {
				continue
			}
			filteredByBrandWords = append(filteredByBrandWords, productToken)
		}
		productTokens = filteredByBrandWords
	}

	// Deduplicate while preserving order so repeated words do not bloat search payload.
	deduplicatedTokens := make([]string, 0, len(productTokens))
	seenTokenSet := make(map[string]struct{}, len(productTokens))
	for _, productToken := range productTokens {
		if _, alreadySeen := seenTokenSet[productToken]; alreadySeen {
			continue
		}
		seenTokenSet[productToken] = struct{}{}
		deduplicatedTokens = append(deduplicatedTokens, productToken)
	}

	// Reuse shared token normalization/filter logic to keep behavior aligned with BuildIndex.
	cleanedTokens := normalizeAndFilterTokens(strings.Join(deduplicatedTokens, " "), defaultConnectorWordSet)
	return strings.Join(cleanedTokens, " ")
}
