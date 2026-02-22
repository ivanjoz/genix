package index_builder

import "testing"

func TestCleanProductTextByBrand_RemovesRepeatedBrandSequence(t *testing.T) {
	cleanedText := CleanProductTextByBrand("Coca Cola Zero Coca Cola 500ml zero", "coca cola")
	if cleanedText != "zero 012ml" {
		t.Fatalf("expected cleaned text 'zero 012ml', got=%q", cleanedText)
	}
}

func TestCleanProductTextByBrand_NormalizesAccentsAndCase(t *testing.T) {
	cleanedText := CleanProductTextByBrand("Café Perú Tostado CAFE PERU", "cafe peru")
	if cleanedText != "tostado" {
		t.Fatalf("expected cleaned text 'tostado', got=%q", cleanedText)
	}
}

func TestCleanProductTextByBrand_DoesNotRemoveSubstrings(t *testing.T) {
	cleanedText := CleanProductTextByBrand("detergente soluble 500ml", "sol")
	if cleanedText != "detergente soluble 012ml" {
		t.Fatalf("expected cleaned text 'detergente soluble 012ml', got=%q", cleanedText)
	}
}

func TestCleanProductTextByBrand_AllowsEmptyResult(t *testing.T) {
	cleanedText := CleanProductTextByBrand("ab ab", "ab")
	if cleanedText != "" {
		t.Fatalf("expected empty cleaned text, got=%q", cleanedText)
	}
}

func TestCleanProductTextByBrand_RemovesBrandConnectorsAndSingleLetters(t *testing.T) {
	cleanedText := CleanProductTextByBrand("Coca Cola de la bebida x con sin azucar", "coca cola")
	if cleanedText != "bebida azucar" {
		t.Fatalf("expected cleaned text 'bebida azucar', got=%q", cleanedText)
	}
}

func TestCleanProductTextByBrand_RemovesStandaloneBrandWord(t *testing.T) {
	cleanedText := CleanProductTextByBrand("Café molido peruano con Cafe", "cafe")
	if cleanedText != "molido peruano" {
		t.Fatalf("expected cleaned text 'molido peruano', got=%q", cleanedText)
	}
}
