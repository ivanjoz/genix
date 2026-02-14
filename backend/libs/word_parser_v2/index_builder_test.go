package word_parser_v2

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildNamesBinaryIndexFromFileWritesBinaryOutput(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "productos.txt")
	outputPath := filepath.Join(tempDir, "productos.idx")

	inputContent := "Queso de cabra\nGel de ducha para piel\nKGS pack en caja\n"
	if writeError := os.WriteFile(inputPath, []byte(inputContent), 0o644); writeError != nil {
		t.Fatalf("failed to create input file: %v", writeError)
	}

	fixedConfig := DefaultFixedSyllableGeneratorConfig()
	fixedConfig.MaxSlots = 120
	frequentConfig := DefaultFrequentSyllableGeneratorConfig()
	frequentConfig.TotalSlots = 180
	frequentConfig.TopFrequentCount = 80

	if buildError := BuildNamesBinaryIndexFromFile(inputPath, outputPath, fixedConfig, frequentConfig); buildError != nil {
		t.Fatalf("unexpected build error: %v", buildError)
	}

	binaryOutput, readError := os.ReadFile(outputPath)
	if readError != nil {
		t.Fatalf("failed to read output idx file: %v", readError)
	}
	if len(binaryOutput) < headerSizeV3 {
		t.Fatalf("binary output too small: %d", len(binaryOutput))
	}
	if string(binaryOutput[:len(binaryIndexMagicV3)]) != binaryIndexMagicV3 {
		t.Fatalf("unexpected binary magic header: %q", string(binaryOutput[:len(binaryIndexMagicV3)]))
	}
	if binaryOutput[len(binaryIndexMagicV3)] != binaryIndexVersion {
		t.Fatalf("unexpected binary version=%d", binaryOutput[len(binaryIndexMagicV3)])
	}

	recordCount := binary.LittleEndian.Uint32(binaryOutput[10:14])
	dictionaryCount := binaryOutput[14]
	shapeCountClass0 := binaryOutput[15]
	shapeCountClass1 := binaryOutput[16]
	dictionaryBytes := binary.LittleEndian.Uint32(binaryOutput[17:21])
	shapeTableClass0Bytes := binary.LittleEndian.Uint32(binaryOutput[21:25])
	shapeTableClass1Bytes := binary.LittleEndian.Uint32(binaryOutput[25:29])

	if recordCount == 0 {
		t.Fatalf("expected record_count > 0")
	}
	if dictionaryCount == 0 {
		t.Fatalf("expected dictionary_count > 0")
	}
	if shapeCountClass0+shapeCountClass1 == 0 {
		t.Fatalf("expected at least one shape in header")
	}

	sectionStart := uint32(headerSizeV3)
	sectionEnd := sectionStart + dictionaryBytes + shapeTableClass0Bytes + shapeTableClass1Bytes
	if sectionEnd > uint32(len(binaryOutput)) {
		t.Fatalf("invalid section boundaries start=%d end=%d total=%d", sectionStart, sectionEnd, len(binaryOutput))
	}
}

func TestBuildNamesBinaryIndexFromRealProductosFileUsesAllRequestedSlots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real corpus integration test in short mode")
	}

	inputPath := "productos.txt"
	if _, statError := os.Stat(inputPath); statError != nil {
		fallbackPath := filepath.Join("libs", "word_parser_v2", "productos.txt")
		if _, fallbackStatError := os.Stat(fallbackPath); fallbackStatError != nil {
			t.Fatalf("missing real productos.txt fixture: %v / %v", statError, fallbackStatError)
		}
		inputPath = fallbackPath
	}

	productNames, loadError := LoadProductNamesFromFile(inputPath)
	if loadError != nil {
		t.Fatalf("failed to load real productos file: %v", loadError)
	}
	if len(productNames) != 10000 {
		t.Fatalf("expected 10000 product names, got %d", len(productNames))
	}

	fixedConfig := DefaultFixedSyllableGeneratorConfig()
	fixedConfig.MaxSlots = 120
	frequentConfig := DefaultFrequentSyllableGeneratorConfig()
	frequentConfig.TotalSlots = 254
	frequentConfig.TopFrequentCount = 200

	cleanedProductNames := make([]string, 0, len(productNames))
	connectorSet := buildConnectorSet(fixedConfig.ConnectorTokens)
	for _, rawProductName := range productNames {
		normalizedTokens := normalizeAndFilterTokens(rawProductName, connectorSet)
		if len(normalizedTokens) == 0 {
			continue
		}
		cleanedProductNames = append(cleanedProductNames, strings.Join(normalizedTokens, " "))
	}

	fixedSlots, reservedSyllables, _, fixedError := GenerateFixedSyllableSlotsDetailed(fixedConfig)
	if fixedError != nil {
		t.Fatalf("unexpected fixed generation error: %v", fixedError)
	}
	generatedDictionary, frequentError := GenerateFrequentSyllableSlotsWithReserved(cleanedProductNames, fixedSlots, reservedSyllables, frequentConfig)
	if frequentError != nil {
		t.Fatalf("unexpected frequent generation error: %v", frequentError)
	}

	totalSlots := len(generatedDictionary.FixedSlots) + len(generatedDictionary.FrequentSlots)
	if totalSlots != frequentConfig.TotalSlots {
		t.Fatalf("expected fixed+computed=%d, got fixed=%d computed=%d total=%d", frequentConfig.TotalSlots, len(generatedDictionary.FixedSlots), len(generatedDictionary.FrequentSlots), totalSlots)
	}
	if len(generatedDictionary.CombinedSlots) != frequentConfig.TotalSlots {
		t.Fatalf("expected combined slots=%d, got %d", frequentConfig.TotalSlots, len(generatedDictionary.CombinedSlots))
	}
}

func TestBuildBinaryIndexPayloadUsesDeltaDictionaryWhenSmaller(t *testing.T) {
	dictionarySlots := []string{"casaaaa", "casaaab", "casaaac", "casaaad", "casaaae", "casaaaf"}
	records := []EncodedNameRecord{
		{WordSizes: []uint8{2}, EncodedContent: []uint8{1, 2}},
		{WordSizes: []uint8{2}, EncodedContent: []uint8{3, 4}},
		{WordSizes: []uint8{2}, EncodedContent: []uint8{5, 6}},
	}

	binaryOutput, buildError := buildBinaryIndexPayload(dictionarySlots, records)
	if buildError != nil {
		t.Fatalf("unexpected payload build error: %v", buildError)
	}
	if len(binaryOutput) < headerSizeV3 {
		t.Fatalf("payload too small: %d", len(binaryOutput))
	}

	headerFlags := binaryOutput[9]
	if headerFlags&headerFlagDictionaryDelta == 0 {
		t.Fatalf("expected dictionary delta mode to be enabled flags=%08b", headerFlags)
	}
}

func TestBuildBinaryIndexPayloadStoresShapeClasses(t *testing.T) {
	dictionarySlots := []string{"ab", "ac", "ad", "ae", "af", "ag", "ah", "ai", "aj"}
	records := []EncodedNameRecord{
		{WordSizes: []uint8{2, 4}, EncodedContent: []uint8{1, 2, 3, 4, 5, 6}},
		{WordSizes: []uint8{5}, EncodedContent: []uint8{7, 8, 9, 1, 2}},
	}

	binaryOutput, buildError := buildBinaryIndexPayload(dictionarySlots, records)
	if buildError != nil {
		t.Fatalf("unexpected payload build error: %v", buildError)
	}

	shapeCountClass0 := binaryOutput[15]
	shapeCountClass1 := binaryOutput[16]
	if shapeCountClass0 != 1 {
		t.Fatalf("expected class0 shape count=1 got=%d", shapeCountClass0)
	}
	if shapeCountClass1 != 1 {
		t.Fatalf("expected class1 shape count=1 got=%d", shapeCountClass1)
	}
}

func TestClassifyShapeWordSizesRejectsWordLongerThan16(t *testing.T) {
	_, classifyError := classifyShapeWordSizes([]uint8{3, 17})
	if classifyError == nil {
		t.Fatalf("expected classify error for oversized word size")
	}
}

func TestNormalizeAndFilterTokensRemovesConnectors(t *testing.T) {
	connectors := buildConnectorSet([]string{"de", "la", "con", "para", "en"})
	tokens := normalizeAndFilterTokens("Queso de cabra en salsa", connectors)
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens after connector filtering, got %d (%v)", len(tokens), tokens)
	}
	if tokens[0] != "queso" || tokens[1] != "cabra" || tokens[2] != "salsa" {
		t.Fatalf("unexpected normalized tokens: %v", tokens)
	}
}
