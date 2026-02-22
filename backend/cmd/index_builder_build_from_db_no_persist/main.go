package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"app/core"
	"app/db"
	"app/handlers"
)

func main() {
	empresaID := flag.Int("empresa-id", 1, "empresa ID used to fetch products/brands/categories from DB")
	outputPath := flag.String("output", "libs/index_builder/productos.idx", "path to write the generated combined index")
	flag.Parse()

	if *empresaID <= 0 {
		log.Fatalf("index_builder: empresa-id inválido=%d", *empresaID)
	}

	core.PopulateVariables()
	db.SetScyllaConnection(db.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	})

	buildOutput, buildErr := handlers.BuildProductosSearchIndexNoPersist(int32(*empresaID))
	if buildErr != nil {
		log.Fatalf("index_builder: no-persist build from db failed: %v", buildErr)
	}

	combinedIndexBytes, bytesErr := buildOutput.IndexBuild.ToBytes()
	if bytesErr != nil {
		log.Fatalf("index_builder: no-persist bytes build failed: %v", bytesErr)
	}
	// Persist a local idx artifact so local debug runs always produce a reusable binary file.
	if writeErr := os.WriteFile(*outputPath, combinedIndexBytes, 0o644); writeErr != nil {
		log.Fatalf("index_builder: write output idx failed (%s): %v", *outputPath, writeErr)
	}

	fmt.Printf("Index Summary (DB NoPersist)\n")
	fmt.Printf("empresa_id: %d\n", *empresaID)
	fmt.Printf("output: %s\n", *outputPath)
	fmt.Printf("text_records: %d\n", len(buildOutput.IndexBuild.SortedIDs))
	fmt.Printf("dictionary_size_bytes: %d\n", buildOutput.IndexBuild.Stats.DictionaryBytes)
	fmt.Printf("aliases_size_bytes: %d\n", buildOutput.IndexBuild.Stats.AliasBytes)
	fmt.Printf("shape_size_bytes: %d\n", buildOutput.IndexBuild.Stats.ShapesBytes)
	fmt.Printf("content_size_bytes: %d\n", buildOutput.IndexBuild.Stats.ContentBytes)
	fmt.Printf("product_ids_size_bytes: %d\n", buildOutput.IndexBuild.Stats.ProductIDsBytes)
	fmt.Printf("taxonomy_brands: %d\n", len(buildOutput.IndexBuild.BrandIDs))
	fmt.Printf("taxonomy_categories: %d\n", len(buildOutput.IndexBuild.CategoryIDs))
	fmt.Printf("taxonomy_brand_index_mode: %s\n", buildOutput.IndexBuild.BrandIndexEncodingName())
	fmt.Printf("total_size_bytes_stage1_plus_stage2: %d\n", len(combinedIndexBytes))
}
