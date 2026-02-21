package main

import (
	"flag"
	"fmt"
	"log"

	"app/core"
	"app/db"
	"app/handlers"
)

func main() {
	empresaID := flag.Int("empresa-id", 1, "empresa ID used to fetch products/brands/categories from DB")
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

	buildOutput, buildErr := handlers.BuildProductosSearchIndex(int32(*empresaID))
	if buildErr != nil {
		log.Fatalf("index_builder: build from db failed: %v", buildErr)
	}

	fmt.Printf("Index Summary (DB)\n")
	fmt.Printf("empresa_id: %d\n", *empresaID)
	fmt.Printf("output: libs/index_builder/productos.idx\n")
	fmt.Printf("text_records: %d\n", len(buildOutput.IndexBuild.SortedIDs))
	fmt.Printf("dictionary_count: %d\n", buildOutput.IndexBuild.Stats.DictionaryCount)
	fmt.Printf("shape_size_bytes: %d\n", buildOutput.IndexBuild.Stats.ShapesBytes)
	fmt.Printf("content_size_bytes: %d\n", buildOutput.IndexBuild.Stats.ContentBytes)
	fmt.Printf("taxonomy_brands: %d\n", len(buildOutput.IndexBuild.BrandIDs))
	fmt.Printf("taxonomy_categories: %d\n", len(buildOutput.IndexBuild.CategoryIDs))
	fmt.Printf("taxonomy_brand_index_mode: %s\n", buildOutput.IndexBuild.BrandIndexEncodingName())
}
