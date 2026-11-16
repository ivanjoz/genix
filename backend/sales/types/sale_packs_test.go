package types

import (
	production "app/production/types"
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// salePackCase is one case of the shared file the Go and TypeScript engines both run
// (docs/PACKS_PLAN.md §4): if the two disagree, a case fails on one side.
type salePackCase struct {
	Name     string
	Products []production.Product
	Packs    []production.ProductPack
	Lines    []SaleOrderLine
	Expected []SaleOrderLine
}

func TestApplySalePacksSharedCases(t *testing.T) {
	caseBytes, err := os.ReadFile("testdata/sale_packs_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	cases := []salePackCase{}
	if err := json.Unmarshal(caseBytes, &cases); err != nil {
		t.Fatal(err)
	}
	for _, packCase := range cases {
		t.Run(packCase.Name, func(t *testing.T) {
			productByID := map[int32]production.Product{}
			for _, product := range packCase.Products {
				productByID[product.ID] = product
			}
			packedLines := ApplySalePacks(packCase.Lines, productByID, packCase.Packs)
			if !slices.Equal(packedLines, packCase.Expected) {
				t.Fatalf("got %+v\nwant %+v", packedLines, packCase.Expected)
			}
			// Splitting moves units between lines, never adds or drops one.
			sumUnits := func(lines []SaleOrderLine) (total int32) {
				for _, line := range lines {
					total += line.Quantity
				}
				return total
			}
			if sumUnits(packedLines) != sumUnits(packCase.Lines) {
				t.Fatalf("the packed lines hold %v units, the order %v", sumUnits(packedLines), sumUnits(packCase.Lines))
			}
		})
	}
}

func TestPackRelevance(t *testing.T) {
	pack := production.ProductPack{Components: []production.ProductPackComponent{
		{ProductID: 12, Quantity: 1}, {CategoryID: 10, Quantity: 2},
	}}
	if relevance := PackRelevance(pack); relevance != 7 {
		t.Fatalf("relevance = %v, want 7 (5 per product slot + 2 per category slot)", relevance)
	}
}
