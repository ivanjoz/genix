package word_parser_v2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateFixedSyllableSlotsRespectsMaxSlots(t *testing.T) {
	config := FixedSyllableGeneratorConfig{
		PreassignedSlots: map[uint16][]string{
			1: {"1"},
			2: {"2"},
			3: {"k", "kg", "kgs"},
		},
		Vowels:                        []string{"a", "e", "i", "o", "u"},
		Consonants:                    []string{"b", "c"},
		VowelCombinationPatterns:      [][]string{{"b*"}},
		EnableTwoSyllableCombinations: true,
		MaxSlots:                      8,
	}

	fixedSlots, generationError := GenerateFixedSyllableSlots(config)
	if generationError != nil {
		t.Fatalf("unexpected error generating fixed slots: %v", generationError)
	}
	if len(fixedSlots) != 8 {
		t.Fatalf("expected 8 fixed slots, got %d", len(fixedSlots))
	}
	if fixedSlots[0] != "1" || fixedSlots[1] != "2" || fixedSlots[2] != "k" {
		t.Fatalf("unexpected fixed prefix ordering: %v", fixedSlots[:3])
	}
}

func TestGenerateFrequentSyllableSlotsStartsAfterOccupiedSlots(t *testing.T) {
	fixedSlots := []string{"1", "2", "x", "ba"}
	productNames := []string{
		"Sion especial",
		"Sion natural",
		"Aceite vegetal",
	}

	dictionary, generationError := GenerateFrequentSyllableSlots(
		productNames,
		fixedSlots,
		FrequentSyllableGeneratorConfig{TopFrequentCount: 200, TotalSlots: 10},
	)
	if generationError != nil {
		t.Fatalf("unexpected error generating frequent slots: %v", generationError)
	}

	if dictionary.FirstFrequentSlot != len(fixedSlots) {
		t.Fatalf("expected first frequent slot to be %d, got %d", len(fixedSlots), dictionary.FirstFrequentSlot)
	}
	if len(dictionary.CombinedSlots) <= len(fixedSlots) {
		t.Fatalf("expected combined slots to include frequent section, got %v", dictionary.CombinedSlots)
	}
	for fixedIndex, expectedFixed := range fixedSlots {
		if dictionary.CombinedSlots[fixedIndex] != expectedFixed {
			t.Fatalf("fixed slot changed at %d expected=%s got=%s", fixedIndex, expectedFixed, dictionary.CombinedSlots[fixedIndex])
		}
	}

	foundSion := false
	for _, generated := range dictionary.FrequentSlots {
		if generated == "sion" {
			foundSion = true
			break
		}
	}
	if !foundSion {
		t.Fatalf("expected frequent slots to include sion exception, got %v", dictionary.FrequentSlots)
	}
}

func TestSplitWordIntoSyllablesPatterns(t *testing.T) {
	result := splitWordIntoSyllables("sionico")
	if len(result) == 0 {
		t.Fatalf("expected non-empty syllables for sionico")
	}

	containsSion := false
	for _, syllable := range result {
		if syllable == "sion" {
			containsSion = true
		}
	}
	if !containsSion {
		t.Fatalf("expected sion exception in split result: %v", result)
	}
}

func TestSplitWordIntoSyllablesSupportsCVCCVCAndCVC(t *testing.T) {
	resultCV := splitWordIntoSyllables("casa")
	if len(resultCV) < 2 || resultCV[0] != "ca" || resultCV[1] != "sa" {
		t.Fatalf("expected CV segmentation for casa, got %v", resultCV)
	}

	resultCCV := splitWordIntoSyllables("plato")
	foundPla := false
	for _, syllable := range resultCCV {
		if syllable == "pla" {
			foundPla = true
			break
		}
	}
	if !foundPla {
		t.Fatalf("expected CCV syllable pla in plato, got %v", resultCCV)
	}

	resultCVC := splitWordIntoSyllables("barco")
	foundBar := false
	for _, syllable := range resultCVC {
		if syllable == "bar" {
			foundBar = true
			break
		}
	}
	if !foundBar {
		t.Fatalf("expected CVC syllable bar in barco, got %v", resultCVC)
	}
}

func TestGenerateDictionaryFromProductNamesPipeline(t *testing.T) {
	fixedConfig := FixedSyllableGeneratorConfig{
		PreassignedSlots: map[uint16][]string{
			1: {"1"},
			2: {"2"},
			3: {"k", "kg", "kgs"},
		},
		Vowels:                        []string{"a", "e", "i", "o", "u"},
		VowelCombinationPatterns:      [][]string{{"b*"}},
		EnableTwoSyllableCombinations: false,
		MaxSlots:                      5,
	}
	frequentConfig := FrequentSyllableGeneratorConfig{
		TopFrequentCount: 10,
		TotalSlots:       8,
	}

	productNames := []string{
		"Sion activo",
		"Sion escolar",
		"Aceite especial",
	}

	generatedDictionary, generationError := GenerateDictionaryFromProductNames(productNames, fixedConfig, frequentConfig)
	if generationError != nil {
		t.Fatalf("unexpected pipeline error: %v", generationError)
	}
	if generatedDictionary.FirstFrequentSlot != len(generatedDictionary.FixedSlots) {
		t.Fatalf(
			"expected frequent start to match fixed length; fixed=%d start=%d",
			len(generatedDictionary.FixedSlots),
			generatedDictionary.FirstFrequentSlot,
		)
	}
	if len(generatedDictionary.CombinedSlots) == 0 {
		t.Fatalf("expected combined slots to be generated")
	}
}

func TestGenerateFrequentWithReservedAliasesSkipsExistingSlotAliases(t *testing.T) {
	productNames := []string{
		"kg kgs k",
		"kg aceite",
	}
	fixedSlots := []string{"k", "1"}
	reservedAliases := []string{"k", "kg", "kgs"}

	generatedDictionary, generationError := GenerateFrequentSyllableSlotsWithReserved(
		productNames,
		fixedSlots,
		reservedAliases,
		FrequentSyllableGeneratorConfig{TopFrequentCount: 20, TotalSlots: 10},
	)
	if generationError != nil {
		t.Fatalf("unexpected error generating with reserved aliases: %v", generationError)
	}

	for _, generatedSyllable := range generatedDictionary.FrequentSlots {
		if generatedSyllable == "k" || generatedSyllable == "kg" || generatedSyllable == "kgs" {
			t.Fatalf("reserved alias leaked into frequent slots: %v", generatedDictionary.FrequentSlots)
		}
	}
}

func TestLoadProductNamesFromFile(t *testing.T) {
	tempDir := t.TempDir()
	productNamesFilePath := filepath.Join(tempDir, "productos.txt")
	writeFileError := os.WriteFile(productNamesFilePath, []byte("  \nSion activo\n\nAceite vegetal\n"), 0o644)
	if writeFileError != nil {
		t.Fatalf("failed creating temp file: %v", writeFileError)
	}

	productNames, loadError := LoadProductNamesFromFile(productNamesFilePath)
	if loadError != nil {
		t.Fatalf("unexpected load error: %v", loadError)
	}
	if len(productNames) != 2 {
		t.Fatalf("expected 2 product names, got %d", len(productNames))
	}
	if productNames[0] != "Sion activo" || productNames[1] != "Aceite vegetal" {
		t.Fatalf("unexpected product names content: %v", productNames)
	}
}
