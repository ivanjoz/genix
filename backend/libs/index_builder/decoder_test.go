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
		t.Logf("[%03d] ROW=%d BRAND=%s CATEGORIES=%s TEXT=%s", sampleIndex+1, sampledRecord.RecordIndex+1, brandPreview, categoryPreview, sampledRecord.Text)
	}
}
