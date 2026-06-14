package business

import (
	textsearch "app/libs/text-search"
	"reflect"
	"strings"
	"testing"
)

func TestParseImageAssetSummary(t *testing.T) {
	summaries, err := parseImageAssetSummary("electronics-tech:71\nhome-living:87\n")
	if err != nil {
		t.Fatalf("parseImageAssetSummary returned error: %v", err)
	}
	if len(summaries) != 2 || summaries[0].Name != "electronics-tech" || summaries[1].MaxID != 87 {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}
}

func TestParseImageAssetSummaryRejectsDuplicateCategory(t *testing.T) {
	_, err := parseImageAssetSummary("electronics-tech:71\nelectronics-tech:72\n")
	if err == nil || !strings.Contains(err.Error(), "duplicates category") {
		t.Fatalf("expected duplicate category error, got %v", err)
	}
}

func TestParseImageAssetListBuildsDelta(t *testing.T) {
	content := strings.Join([]string{
		"# Lista",
		"",
		"| Nombre | Descripción | Elementos | Colores Dominantes | Fondo | Relación de Aspecto | Iluminación |",
		"|--------|-------------|-----------|-------------------|-------|---------------------|-------------|",
		"| 28 | Registro anterior | teléfono | negro | clean | 3:2 | light |",
		"| 36 | Mano con teléfono \\| código visible | teléfono, código QR | negro | clean | 3:2 | light |",
	}, "\n")

	records, err := parseImageAssetList(
		imageAssetCategorySummary{Name: "electronics-tech", MaxID: 36},
		123,
		28,
		456,
		content,
	)
	if err != nil {
		t.Fatalf("parseImageAssetList returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one delta record, got %d", len(records))
	}
	record := records[0]
	if record.ID != 36 || record.CategoryID != 123 || record.Updated != 456 {
		t.Fatalf("unexpected record identity: %+v", record)
	}
	if !reflect.DeepEqual(record.Objects, []string{"teléfono", "código QR"}) {
		t.Fatalf("unexpected objects: %v", record.Objects)
	}
	if len(record.Bigrams) == 0 {
		t.Fatal("expected encoded bigrams")
	}
	expectedBigrams := imageAssetBigramsToInt8(textsearch.EncodeTextBigrams("teléfono código QR"))
	if !reflect.DeepEqual(record.Bigrams, expectedBigrams) {
		t.Fatalf("bigrams must be generated only from Elementos: got %v want %v", record.Bigrams, expectedBigrams)
	}
}

func TestParseImageAssetListRejectsSummaryMismatch(t *testing.T) {
	content := "| Nombre | Descripción | Elementos | Colores Dominantes | Fondo | Relación de Aspecto | Iluminación |\n" +
		"|--------|-------------|-----------|-------------------|-------|---------------------|-------------|\n" +
		"| 35 | Descripción | objeto | negro | clean | 3:2 | light |\n"

	_, err := parseImageAssetList(
		imageAssetCategorySummary{Name: "electronics-tech", MaxID: 36},
		123,
		0,
		456,
		content,
	)
	if err == nil || !strings.Contains(err.Error(), "maximum ID mismatch") {
		t.Fatalf("expected maximum ID mismatch, got %v", err)
	}
}

func TestImageAssetBigramsToInt8PreservesBytes(t *testing.T) {
	encoded := []uint8{0, 127, 128, 255}
	stored := imageAssetBigramsToInt8(encoded)
	for index, storedValue := range stored {
		if uint8(storedValue) != encoded[index] {
			t.Fatalf("byte %d changed: got %d want %d", index, uint8(storedValue), encoded[index])
		}
	}
}
