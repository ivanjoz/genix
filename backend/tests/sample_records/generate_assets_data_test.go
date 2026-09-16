package sample_records

import "testing"

// The two .psv files are coupled: every asset names its material by SKU, and that material has to
// be depreciable or PostAsset rejects the acquisition ("El insumo no tiene un esquema de
// depreciación configurado"). Nothing at compile time enforces that, and against a real database
// the mistake only surfaces halfway through a run, after assets have already been written.
func TestAssetSeedsPointAtDepreciableMaterials(t *testing.T) {
	supplySeeds, supplyError := readSupplyMaterialSeeds()
	if supplyError != nil {
		t.Fatalf("readSupplyMaterialSeeds: %v", supplyError)
	}
	if len(supplySeeds) != 100 {
		t.Fatalf("expected 100 supply materials, got %d", len(supplySeeds))
	}

	depreciationMonthsBySKU := map[string]int16{}
	for _, seed := range supplySeeds {
		if seed.sku == "" {
			t.Fatalf("supply material %q has no SKU", seed.name)
		}
		if _, isDuplicate := depreciationMonthsBySKU[seed.sku]; isDuplicate {
			t.Fatalf("duplicated SKU %q", seed.sku)
		}
		if len(seed.name) < 2 {
			t.Fatalf("SKU %q has a name the handler would reject: %q", seed.sku, seed.name)
		}
		if seed.price < 0 {
			t.Fatalf("SKU %q has a negative price", seed.sku)
		}
		depreciationMonthsBySKU[seed.sku] = seed.depreciationMonths
	}

	assetSeeds, assetError := readAssetSeeds()
	if assetError != nil {
		t.Fatalf("readAssetSeeds: %v", assetError)
	}
	if len(assetSeeds) != 100 {
		t.Fatalf("expected 100 assets, got %d", len(assetSeeds))
	}

	usedSerials := map[string]bool{}
	for _, seed := range assetSeeds {
		depreciationMonths, materialExists := depreciationMonthsBySKU[seed.materialSKU]
		if !materialExists {
			t.Fatalf("asset %q references unknown material %q", seed.name, seed.materialSKU)
		}
		if depreciationMonths <= 0 {
			t.Fatalf("asset %q references non-depreciable material %q", seed.name, seed.materialSKU)
		}
		// PostAsset refuses a value of 0, and a serial-tracked acquisition is one unit by
		// definition — Quantity is only read when no serial was given.
		if seed.acquisitionValue <= 0 {
			t.Fatalf("asset %q has a non-positive acquisition value", seed.name)
		}
		if seed.serialNumber != "" && seed.quantity != 1 {
			t.Fatalf("serial-tracked asset %q declares quantity %d", seed.name, seed.quantity)
		}
		if seed.serialNumber == "" && seed.quantity < 1 {
			t.Fatalf("grouped asset %q declares quantity %d", seed.name, seed.quantity)
		}
		if seed.serialNumber != "" {
			if usedSerials[seed.serialNumber] {
				t.Fatalf("serial %q is used twice", seed.serialNumber)
			}
			usedSerials[seed.serialNumber] = true
		}
	}
}

// Both granularities must be present, or the run would only ever exercise one of the two paths
// PostAsset has (one asset per serial, or a single grouped row).
func TestAssetSeedsCoverBothGranularities(t *testing.T) {
	assetSeeds, assetError := readAssetSeeds()
	if assetError != nil {
		t.Fatalf("readAssetSeeds: %v", assetError)
	}

	serialTracked, grouped := 0, 0
	for _, seed := range assetSeeds {
		if seed.serialNumber != "" {
			serialTracked++
			continue
		}
		grouped++
	}

	if serialTracked == 0 || grouped == 0 {
		t.Fatalf("expected both granularities, got %d serial-tracked and %d grouped", serialTracked, grouped)
	}
}

func TestParsePriceToCents(t *testing.T) {
	testCases := []struct {
		rawValue      string
		expectedCents int32
	}{
		{"4500.00", 450000},
		{"689.66", 68966},
		{"0.05", 5},
		{" 12.5 ", 1250},
		// Binary floating point turns this into 19.899999…; without rounding it truncates to 1989.
		{"19.90", 1990},
	}

	for _, testCase := range testCases {
		parsedCents, parseError := parsePriceToCents(testCase.rawValue)
		if parseError != nil {
			t.Fatalf("parsePriceToCents(%q): %v", testCase.rawValue, parseError)
		}
		if parsedCents != testCase.expectedCents {
			t.Errorf("parsePriceToCents(%q) = %d, want %d", testCase.rawValue, parsedCents, testCase.expectedCents)
		}
	}

	if _, parseError := parsePriceToCents("no-es-un-monto"); parseError == nil {
		t.Error("expected an error for a non-numeric amount")
	}
}
