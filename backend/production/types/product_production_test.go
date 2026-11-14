package types

import (
	"strings"
	"testing"
)

func TestValidateProductProduction(t *testing.T) {
	// 6032 (suministros bought into stock) has stock account 252; 656 (suministros, expensed) has none.
	suppliesByID := map[int32]Product{
		10: {ID: 10, Name: "Arroz", Status: ProductStatusSupply, SbuQuantity: 1000, AccountCode: 6032},
		11: {ID: 11, Name: "Sal", Status: ProductStatusSupply, SbuQuantity: 0, AccountCode: 6032},
		12: {ID: 12, Name: "Limpieza", Status: ProductStatusSupply, AccountCode: 656},
		13: {ID: 13, Name: "Gaseosa", Status: ProductStatusActive, AccountCode: 6032},
	}
	recipe := func(supplies ...ProductionSupply) Product {
		return Product{ProductionYieldPortions: 20, ProductionSupplies: supplies}
	}

	cases := []struct {
		name      string
		origin    int8
		product   Product // only its Production* columns are read
		wantError string  // "" = accepted
	}{
		{"empty config on a bought product", ProductOriginBought, Product{}, ""},
		{"standard cost only", ProductOriginMade, Product{ProductionStandardCost: 500}, ""},
		{"1 kg 250 g of rice", ProductOriginMade, recipe(ProductionSupply{SupplyID: 10, Quantity: 1250, SubDivisor: 1000}), ""},
		{"whole units of a supply without sub-unit", ProductOriginMade, recipe(ProductionSupply{SupplyID: 11, Quantity: 2000, SubDivisor: 1}), ""},
		{"config on a bought product", ProductOriginBought, Product{ProductionStandardCost: 500}, "solo un producto fabricado"},
		{"negative cost", ProductOriginMade, Product{ProductionExtraCostPerUnit: -1}, "no pueden ser negativos"},
		{"supplies without yield", ProductOriginMade, Product{ProductionSupplies: []ProductionSupply{{SupplyID: 10, Quantity: 1000, SubDivisor: 1000}}}, "cuántas porciones"},
		{"unknown supply", ProductOriginMade, recipe(ProductionSupply{SupplyID: 99, Quantity: 1000, SubDivisor: 1}), "no existe"},
		{"a product, not a supply", ProductOriginMade, recipe(ProductionSupply{SupplyID: 13, Quantity: 1000, SubDivisor: 1}), "no existe"},
		{"repeated supply", ProductOriginMade, recipe(
			ProductionSupply{SupplyID: 10, Quantity: 1000, SubDivisor: 1000},
			ProductionSupply{SupplyID: 10, Quantity: 2000, SubDivisor: 1000}), "repetido"},
		{"stale divisor", ProductOriginMade, recipe(ProductionSupply{SupplyID: 10, Quantity: 1000, SubDivisor: 6}), "usa el divisor"},
		{"zero quantity", ProductOriginMade, recipe(ProductionSupply{SupplyID: 10, Quantity: 0, SubDivisor: 1000}), "inválida"},
		{"sub-units on a supply without sub-unit", ProductOriginMade, recipe(ProductionSupply{SupplyID: 11, Quantity: 1, SubDivisor: 1}), "inválida"},
		{"expense account supply", ProductOriginMade, recipe(ProductionSupply{SupplyID: 12, Quantity: 1000, SubDivisor: 1}), "no es de existencias"},
	}
	for _, testCase := range cases {
		product := testCase.product
		product.Name, product.Origin = "Lomo saltado", testCase.origin
		err := ValidateProductProduction(&product, suppliesByID)
		if testCase.wantError == "" && err != nil {
			t.Errorf("%s: unexpected error %v", testCase.name, err)
		}
		if testCase.wantError != "" && (err == nil || !strings.Contains(err.Error(), testCase.wantError)) {
			t.Errorf("%s: want error containing %q, got %v", testCase.name, testCase.wantError, err)
		}
	}
}
