package accounting

import (
	"app/seeds/runner"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The asset seeds and base's supply catalog are coupled: every asset names its material by SKU, and
// that material has to be depreciable or PostAsset rejects the acquisition ("El insumo no tiene un
// esquema de depreciación configurado"). Nothing at compile time enforces that, and against a real
// database the mistake only surfaces halfway through a run, after assets have already been written.
func TestAssetSeedsPointAtDepreciableMaterials(t *testing.T) {
	supplyMaterials, err := os.ReadFile("../base/supply_materials.psv")
	if err != nil {
		t.Fatal(err)
	}
	// Columns: name | sku | price | unit | currency | depreciation months.
	supplyRows, err := runner.ParsePSV(string(supplyMaterials), 6)
	if err != nil {
		t.Fatal(err)
	}
	depreciationMonthsBySKU := map[string]int{}
	for _, row := range supplyRows {
		depreciationMonthsBySKU[strings.TrimSpace(row[1])], _ = strconv.Atoi(strings.TrimSpace(row[5]))
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
