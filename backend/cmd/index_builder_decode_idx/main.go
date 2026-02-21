package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"app/libs/index_builder"
)

func main() {
	inputPath := flag.String("input", "libs/index_builder/productos.idx", "path to idx file")
	sampleCount := flag.Int("sample", 10, "number of decoded random samples")
	seed := flag.Int64("seed", 0, "random seed for stable sampling (0 uses current time)")
	flag.Parse()

	indexBytes, readErr := os.ReadFile(*inputPath)
	if readErr != nil {
		log.Fatalf("index_builder: read failed: %v", readErr)
	}

	decodedResult, decodeErr := index_builder.DecodeBinary(indexBytes)
	if decodeErr != nil {
		log.Fatalf("index_builder: decode failed: %v", decodeErr)
	}

	fmt.Printf("Decoder Summary\n")
	fmt.Printf("records: %d\n", decodedResult.Stats.RecordCount)
	fmt.Printf("dictionary_count: %d\n", decodedResult.Stats.DictionaryCount)
	fmt.Printf("dictionary_bytes: %d\n", decodedResult.Stats.DictionaryBytes)
	fmt.Printf("shape_size_bytes: %d\n", decodedResult.Stats.ShapesBytes)
	fmt.Printf("content_size_bytes: %d\n", decodedResult.Stats.ContentBytes)
	fmt.Printf("shape_delta_counts: d8=%d d16=%d d24=%d\n", decodedResult.Stats.ShapeDelta8Count, decodedResult.Stats.ShapeDelta16Count, decodedResult.Stats.ShapeDelta24Count)
	if decodedResult.Taxonomy != nil {
		fmt.Printf("taxonomy_bytes: %d\n", decodedResult.Stats.TaxonomyBytes)
		fmt.Printf("taxonomy_brands: %d\n", decodedResult.Stats.TaxonomyBrandCount)
		fmt.Printf("taxonomy_categories: %d\n", decodedResult.Stats.TaxonomyCategoryCount)
		fmt.Printf("taxonomy_brand_index_mode: %s\n", decodedResult.Stats.TaxonomyBrandIndexMode)
	}
	fmt.Printf("input: %s\n", *inputPath)

	requestedSampleCount := int32(*sampleCount)
	if requestedSampleCount > 0 {
		samples := index_builder.SampleDecodedRecords(decodedResult.Records, requestedSampleCount, *seed)
		fmt.Printf("samples: %d\n", len(samples))
		for sampleIndex, sample := range samples {
			categoryPreview := "-"
			if len(sample.CategoryNames) > 0 {
				categoryPreview = strings.Join(sample.CategoryNames, ", ")
			}
			brandPreview := sample.BrandName
			if brandPreview == "" {
				brandPreview = "-"
			}
			fmt.Printf("[%03d] ROW=%d BRAND=%s CATEGORIES=%s SHAPE=%v TEXT=%s\n", sampleIndex+1, sample.RecordIndex+1, brandPreview, categoryPreview, sample.Shape, sample.Text)
		}
	}
}
