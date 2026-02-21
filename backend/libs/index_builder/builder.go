package index_builder

import "fmt"

type ProductTextOptimizationAggregate struct {
	ChangedProducts          int
	FallbackOriginalProducts int
	RemovedBrandOccurrences  int
	RemovedDuplicateTokens   int
	NormalizedTokensBefore   int
	NormalizedTokensAfter    int
}

type ProductosIndexBuildArtifacts struct {
	TextIndexResult     *BuildResult
	TaxonomyIndexResult *TaxonomyBuildResult
	OptimizationStats   ProductTextOptimizationAggregate
}

// BuildProductosIndex runs a sequential end-to-end build:
// 1) optimize product text using brand-aware normalization
// 2) build text index
// 3) build taxonomy columns aligned to text sorted IDs
func BuildProductosIndex(buildInput BuildInput) (*ProductosIndexBuildArtifacts, error) {
	if len(buildInput.Products) == 0 {
		return nil, fmt.Errorf("build input requires products")
	}

	brandNameByID := buildBrandNameLookup(buildInput.Brands)
	optimizedProductRecords := make([]RecordInput, 0, len(buildInput.Products))
	optimizationStats := ProductTextOptimizationAggregate{}
	for _, currentProductRecord := range buildInput.Products {
		// Strip brand terms + duplicate tokens before stage-1 encoding to reduce content bytes.
		brandNameForProduct := brandNameByID[currentProductRecord.BrandID]
		optimizedProductText, currentProductStats := OptimizeProductTextAggressive(currentProductRecord.Text, brandNameForProduct)

		optimizationStats.NormalizedTokensBefore += currentProductStats.OriginalNormalizedTokens
		optimizationStats.NormalizedTokensAfter += currentProductStats.FinalNormalizedTokens
		optimizationStats.RemovedBrandOccurrences += currentProductStats.RemovedBrandOccurrences
		optimizationStats.RemovedDuplicateTokens += currentProductStats.RemovedDuplicateTokens
		if currentProductStats.Changed {
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
	textBuildResult, textBuildErr := Build(optimizedProductRecords, textBuildOptions)
	if textBuildErr != nil {
		return nil, fmt.Errorf("build text index: %w", textBuildErr)
	}

	taxonomyBuildResult, taxonomyBuildErr := BuildTaxonomySecondPass(textBuildResult.SortedIDs, BuildInput{
		Products:   optimizedProductRecords,
		Brands:     buildInput.Brands,
		Categories: buildInput.Categories,
	})
	if taxonomyBuildErr != nil {
		return nil, fmt.Errorf("build taxonomy index: %w", taxonomyBuildErr)
	}

	return &ProductosIndexBuildArtifacts{
		TextIndexResult:     textBuildResult,
		TaxonomyIndexResult: taxonomyBuildResult,
		OptimizationStats:   optimizationStats,
	}, nil
}

func buildBrandNameLookup(brandRecords []RecordInput) map[int32]string {
	brandNameByID := make(map[int32]string, len(brandRecords)+1)
	for _, currentBrandRecord := range brandRecords {
		brandNameByID[currentBrandRecord.ID] = currentBrandRecord.Text
	}
	// Keep a sentinel brand label for products without explicit brand assignment.
	if _, hasSentinelBrand := brandNameByID[0]; !hasSentinelBrand {
		brandNameByID[0] = "Sin marca"
	}
	return brandNameByID
}
