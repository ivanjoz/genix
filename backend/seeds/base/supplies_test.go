package base

import "testing"

// The supply catalog must load whole and be accepted by POST.supply-material: a bad row would only
// surface against a real database, halfway through a run.
func TestSupplyMaterialSeedsAreValid(t *testing.T) {
	supplySeeds, supplyError := readSupplyMaterialSeeds()
	if supplyError != nil {
		t.Fatalf("readSupplyMaterialSeeds: %v", supplyError)
	}
	if len(supplySeeds) != 100 {
		t.Fatalf("expected 100 supply materials, got %d", len(supplySeeds))
	}

	seenSKUs := map[string]bool{}
	for _, seed := range supplySeeds {
		if seed.sku == "" {
			t.Fatalf("supply material %q has no SKU", seed.name)
		}
		if seenSKUs[seed.sku] {
			t.Fatalf("duplicated SKU %q", seed.sku)
		}
		seenSKUs[seed.sku] = true
		if len(seed.name) < 2 {
			t.Fatalf("SKU %q has a name the handler would reject: %q", seed.sku, seed.name)
		}
		if seed.price < 0 {
			t.Fatalf("SKU %q has a negative price", seed.sku)
		}
	}
}
