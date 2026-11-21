package types

import (
	"strings"
	"testing"
)

func menuFixture() DiningMenu {
	return DiningMenu{
		Name: "Carta principal",
		Blocks: []DiningMenuBlock{
			{Kind: MenuBlockTitle, Text: " Entradas ", Align: MenuAlignCenter, CategoryID: 9, Variant: 3},
			{Kind: MenuBlockCategory, CategoryID: 3, Variant: MenuProductsCompact},
			{Kind: MenuBlockProducts, ProductIDs: []int32{7, 8}, Variant: MenuProductsCards},
			{Kind: MenuBlockAllProducts, Variant: MenuProductsWithPhoto, RestOnly: true},
			{Kind: MenuBlockDivider},
		},
	}
}

var (
	fixtureCategories = map[int32]bool{3: true}
	fixtureProducts   = map[int32]bool{7: true, 8: true}
)

func TestNormalizeClearsUnusedFieldsAndFillsTheme(t *testing.T) {
	menu := menuFixture()
	NormalizeDiningMenu(&menu)
	if menu.Theme != DefaultMenuTheme() {
		t.Fatalf("theme not defaulted: %+v", menu.Theme)
	}
	title := menu.Blocks[0]
	if title.Text != "Entradas" || title.CategoryID != 0 || title.Variant != 0 {
		t.Fatalf("title keeps unused fields: %+v", title)
	}
	if err := ValidateDiningMenu(menu, fixtureCategories, fixtureProducts); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRefusals(t *testing.T) {
	cases := map[string]func(menu *DiningMenu){
		"nombre":       func(menu *DiningMenu) { menu.Name = "" },
		"colores":      func(menu *DiningMenu) { menu.Theme.AccentColor = "red" },
		"categoría":    func(menu *DiningMenu) { menu.Blocks[1].CategoryID = 99 },
		"repetido":     func(menu *DiningMenu) { menu.Blocks[2].ProductIDs = []int32{7, 7} },
		"no existe":    func(menu *DiningMenu) { menu.Blocks[2].ProductIDs = []int32{7, 50} },
		"estilo":       func(menu *DiningMenu) { menu.Blocks[3].Variant = 0 },
		"texto":        func(menu *DiningMenu) { menu.Blocks[0].Text = "" },
		"imagen":       func(menu *DiningMenu) { menu.Blocks[4] = DiningMenuBlock{Kind: MenuBlockImage} },
		"tipo":         func(menu *DiningMenu) { menu.Blocks[4].Kind = 42 },
		"muchos":       func(menu *DiningMenu) { menu.Blocks = make([]DiningMenuBlock, MaxMenuBlocks+1) },
		"sin tamaño":   func(menu *DiningMenu) { menu.Blocks[2].ProductIDs = nil },
		"alineación":   func(menu *DiningMenu) { menu.Blocks[0].Align = 0 },
		"nombre largo": func(menu *DiningMenu) { menu.Name = strings.Repeat("a", MaxMenuNameLength+1) },
	}
	for name, mutate := range cases {
		menu := menuFixture()
		NormalizeDiningMenu(&menu)
		mutate(&menu)
		if err := ValidateDiningMenu(menu, fixtureCategories, fixtureProducts); err == nil {
			t.Errorf("%s: expected a refusal", name)
		}
	}
}

func TestMenuReferencedIDs(t *testing.T) {
	menu := menuFixture()
	NormalizeDiningMenu(&menu)
	categoryIDs, productIDs := MenuReferencedIDs(menu)
	if len(categoryIDs) != 1 || categoryIDs[0] != 3 || len(productIDs) != 2 {
		t.Fatalf("categories %v products %v", categoryIDs, productIDs)
	}
}
