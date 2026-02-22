package index_builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecodeBinary_Sample50PrintBrandAndCategories(t *testing.T) {
	indexPathsToTry := []string{
		filepath.Join("libs", "index_builder", "productos.idx"),
		"productos.idx",
	}
	var indexBytes []byte
	var readErr error
	indexPath := ""
	for _, candidatePath := range indexPathsToTry {
		indexBytes, readErr = os.ReadFile(candidatePath)
		if readErr == nil {
			indexPath = candidatePath
			break
		}
	}
	if readErr != nil {
		t.Fatalf("read productos.idx: %v", readErr)
	}
	t.Logf("input=%s", indexPath)

	decodedResult, decodeErr := DecodeBinary(indexBytes)
	if decodeErr != nil {
		// This fixture can be stale while format is evolving; strict decoder supports only the current format.
		if strings.Contains(decodeErr.Error(), "unsupported text version=") || strings.Contains(decodeErr.Error(), "invalid text section count=") {
			t.Skipf("fixture is not current format v%d: %v", BinaryVersion, decodeErr)
		}
		t.Fatalf("decode productos.idx: %v", decodeErr)
	}
	if decodedResult.Taxonomy == nil {
		t.Fatalf("expected taxonomy block in productos.idx")
	}

	seed := time.Now().UnixNano()
	sampledRecords := SampleDecodedRecords(decodedResult.Records, 50, seed)
	if len(sampledRecords) != 50 {
		t.Fatalf("expected 50 sampled records got=%d", len(sampledRecords))
	}

	t.Logf("sample_seed=%d", seed)
	for sampleIndex, sampledRecord := range sampledRecords {
		categoryPreview := "-"
		if len(sampledRecord.CategoryNames) > 0 {
			categoryPreview = strings.Join(sampledRecord.CategoryNames, ", ")
		}
		brandPreview := sampledRecord.BrandName
		if brandPreview == "" {
			brandPreview = "-"
		}
		// Print random sample rows for manual decoder verification against source data expectations.
		t.Logf("[%03d] ROW=%d PRODUCT_ID=%d BRAND=%s CATEGORIES=%s TEXT=%s", sampleIndex+1, sampledRecord.RecordIndex+1, sampledRecord.ProductID, brandPreview, categoryPreview, sampledRecord.Text)
	}
}

func TestDecodeBinary_CombinedRoundtrip(t *testing.T) {
	buildInput := BuildInput{
		Products: []RecordInput{
			{ID: 101, Text: "Leche Entera 1L", BrandID: 1, CategoriesIDs: []int32{10, 11}},
			{ID: 102, Text: "Pan Integral", BrandID: 2, CategoriesIDs: []int32{12}},
			{ID: 103, Text: "Tomate Frito", BrandID: 1, CategoriesIDs: []int32{13, 12}},
		},
		Brands: []RecordInput{
			{ID: 1, Text: "Marca Uno"},
			{ID: 2, Text: "Marca Dos"},
		},
		Categories: []RecordInput{
			{ID: 10, Text: "Lacteos"},
			{ID: 11, Text: "Bebidas"},
			{ID: 12, Text: "Panaderia"},
			{ID: 13, Text: "Despensa"},
		},
	}

	artifacts, buildErr := BuildProductosIndex(buildInput)
	if buildErr != nil {
		t.Fatalf("build productos index: %v", buildErr)
	}
	combinedPayload, toBytesErr := artifacts.ToBytes()
	if toBytesErr != nil {
		t.Fatalf("build combined bytes: %v", toBytesErr)
	}
	if len(combinedPayload) == 0 {
		t.Fatalf("combined payload should not be empty")
	}

	decodedResult, decodeErr := DecodeBinary(combinedPayload)
	if decodeErr != nil {
		t.Fatalf("decode combined payload: %v", decodeErr)
	}
	if decodedResult.Stats.RecordCount != int32(len(decodedResult.Records)) {
		t.Fatalf("record count mismatch stats=%d rows=%d", decodedResult.Stats.RecordCount, len(decodedResult.Records))
	}
	if decodedResult.Taxonomy == nil {
		t.Fatalf("taxonomy should exist in combined payload")
	}
	if decodedResult.Stats.DictionaryBytes <= 0 || decodedResult.Stats.ShapesBytes <= 0 || decodedResult.Stats.ContentBytes <= 0 {
		t.Fatalf("text section stats must be positive")
	}
	if decodedResult.Stats.TaxonomyBytes <= 0 {
		t.Fatalf("taxonomy bytes must be positive")
	}
	expectedProductIDSet := map[int32]struct{}{
		101: {},
		102: {},
		103: {},
	}

	for recordIndex, decodedRecord := range decodedResult.Records {
		// Every decoded row must be enriched by taxonomy mappings in combined payloads.
		if decodedRecord.BrandName == "" {
			t.Fatalf("record %d missing brand name", recordIndex)
		}
		if len(decodedRecord.CategoryNames) == 0 {
			t.Fatalf("record %d missing categories", recordIndex)
		}
		if _, exists := expectedProductIDSet[decodedRecord.ProductID]; !exists {
			t.Fatalf("record %d has unexpected product id=%d", recordIndex, decodedRecord.ProductID)
		}
	}
}
