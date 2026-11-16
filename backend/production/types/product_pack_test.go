package types

import (
	"strings"
	"testing"
)

func TestValidateProductPack(t *testing.T) {
	menuPack := func() ProductPack {
		return ProductPack{
			Name: "Menú del día", PriceType: PackPriceFixed, PriceValue: 1800,
			Components: []ProductPackComponent{{CategoryID: 3, Quantity: 1}, {CategoryID: 4, Quantity: 1}},
		}
	}
	if err := ValidateProductPack(menuPack()); err != nil {
		t.Fatalf("a menu pack must be valid: %v", err)
	}

	cases := []struct {
		name      string
		edit      func(pack *ProductPack)
		wantError string
	}{
		{"no name", func(pack *ProductPack) { pack.Name = " " }, "nombre"},
		{"no components", func(pack *ProductPack) { pack.Components = nil }, "al menos un componente"},
		{"both product and category", func(pack *ProductPack) { pack.Components[0].ProductID = 7 }, "un producto o una categoría"},
		{"neither product nor category", func(pack *ProductPack) { pack.Components[0].CategoryID = 0 }, "un producto o una categoría"},
		{"zero quantity", func(pack *ProductPack) { pack.Components[0].Quantity = 0 }, "cantidad"},
		{"repeated category", func(pack *ProductPack) { pack.Components[1].CategoryID = 3 }, "repetida"},
		{"single unit", func(pack *ProductPack) { pack.Components = pack.Components[:1] }, "al menos 2 unidades"},
		{"unknown price type", func(pack *ProductPack) { pack.PriceType = 9 }, "tipo de precio"},
		{"zero amount", func(pack *ProductPack) { pack.PriceValue = 0 }, "mayor a cero"},
		{"full percentage", func(pack *ProductPack) { pack.PriceType, pack.PriceValue = PackPricePercentOff, 10000 }, "porcentaje"},
		{"unknown rounding", func(pack *ProductPack) { pack.RoundingMode = 4 }, "redondeo"},
	}
	for _, testCase := range cases {
		pack := menuPack()
		testCase.edit(&pack)
		err := ValidateProductPack(pack)
		if err == nil || !strings.Contains(err.Error(), testCase.wantError) {
			t.Errorf("%s: want an error containing %q, got %v", testCase.name, testCase.wantError, err)
		}
	}
}
