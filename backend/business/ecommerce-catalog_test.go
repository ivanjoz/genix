package business

import (
	logistics "app/logistics/types"
	production "app/production/types"
	"strings"
	"testing"

	"github.com/ivanjoz/colbin"
)

// The sellable rule as the catalog applies it: only sellable presentations listed, removal otherwise.
func TestMakeEcommerceCatalogProduct(t *testing.T) {
	product := production.Product{ID: 9, Name: "Polo", Status: production.ProductStatusActive, ImageMain: 4}
	stockRows := []logistics.ProductStock{{WarehouseID: 1, ProductID: 9, Quantity: 3, PendingDeliveryOnline: 1}}

	listed := makeEcommerceCatalogProduct(&product, stockRows)
	if listed.Status != 1 || len(listed.Stock) != 1 || listed.Stock[0] != (ecommerceCatalogStock{PresentationID: 0, Units: 2}) {
		t.Fatalf("listed product: got %+v, want status 1 and 2 units of presentation 0", listed)
	}

	product.WebHidden = true
	if hidden := makeEcommerceCatalogProduct(&product, stockRows); hidden.Status != 0 || hidden.Name != "" {
		t.Fatalf("hidden product: got %+v, want a bare removal", hidden)
	}

	product.WebHidden = false
	stockRows[0].PendingDeliveryOnline = 3
	if soldOut := makeEcommerceCatalogProduct(&product, stockRows); soldOut.Status != 0 {
		t.Fatalf("sold out product: got %+v, want a removal", soldOut)
	}
}

// The file is a self-describing colbin message the storefront decodes with no Go schema: the
// embedded meta and the nested stock records must come out as plain fields.
func TestEcommerceCatalogFileColbin(t *testing.T) {
	fileContent := ecommerceCatalogFileContent{
		Meta: ecommerceCatalogFile{Created: 100, Updated: 96, Generation: 7},
		Productos: []ecommerceCatalogProduct{{ID: 9, Name: "Polo", CategoryIDs: []int32{2}, Image: 4,
			Stock: []ecommerceCatalogStock{{PresentationID: 3, Units: 5}}, Status: 1}},
		Marcas: []ecommerceCatalogListRecord{{ID: 1, Name: "Marca", Status: 1}},
	}
	fileBytes, err := colbin.MarshalSelfDescribing(fileContent)
	if err != nil {
		t.Fatalf("MarshalSelfDescribing: %v", err)
	}
	text, err := colbin.ToJSON(nil, fileBytes)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	for _, expected := range []string{`"Created":100`, `"Updated":96`, `"Generation":7`, `"PresentationID":3`, `"Units":5`, `"ss":1`} {
		if !strings.Contains(string(text), expected) {
			t.Fatalf("decoded file lacks %s: %s", expected, text)
		}
	}
}
