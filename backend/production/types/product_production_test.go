package types

import (
	"app/core"
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
		// A semi-finished supply has no purchase account: its stock account is 231 by its origin.
		14: {ID: 14, Name: "Capellada aparada", Status: ProductStatusSupply, Origin: ProductOriginMade},
	}
	laborTypesByID := map[int32]ProductionLaborType{
		1: {ID: 1, Name: "Zapatero", Status: 1},
		2: {ID: 2, Name: "Retirado", Status: 0},
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
		{"supplies without yield", ProductOriginMade, Product{ProductionSupplies: []ProductionSupply{{SupplyID: 10, Quantity: 1000, SubDivisor: 1000}}}, "cuántas unidades"},
		{"unknown supply", ProductOriginMade, recipe(ProductionSupply{SupplyID: 99, Quantity: 1000, SubDivisor: 1}), "no existe"},
		{"a product, not a supply", ProductOriginMade, recipe(ProductionSupply{SupplyID: 13, Quantity: 1000, SubDivisor: 1}), "no existe"},
		{"repeated supply", ProductOriginMade, recipe(
			ProductionSupply{SupplyID: 10, Quantity: 1000, SubDivisor: 1000},
			ProductionSupply{SupplyID: 10, Quantity: 2000, SubDivisor: 1000}), "repetido"},
		{"stale divisor", ProductOriginMade, recipe(ProductionSupply{SupplyID: 10, Quantity: 1000, SubDivisor: 6}), "usa el divisor"},
		{"zero quantity", ProductOriginMade, recipe(ProductionSupply{SupplyID: 10, Quantity: 0, SubDivisor: 1000}), "inválida"},
		{"sub-units on a supply without sub-unit", ProductOriginMade, recipe(ProductionSupply{SupplyID: 11, Quantity: 1, SubDivisor: 1}), "inválida"},
		{"expense account supply", ProductOriginMade, recipe(ProductionSupply{SupplyID: 12, Quantity: 1000, SubDivisor: 1}), "no es de existencias"},
		{"15% waste", ProductOriginMade, recipe(ProductionSupply{SupplyID: 10, Quantity: 1000, SubDivisor: 1000, WasteBasisPoints: 1500}), ""},
		{"negative waste", ProductOriginMade, recipe(ProductionSupply{SupplyID: 10, Quantity: 1000, SubDivisor: 1000, WasteBasisPoints: -1}), "merma"},
		{"waste over 100%", ProductOriginMade, recipe(ProductionSupply{SupplyID: 10, Quantity: 1000, SubDivisor: 1000, WasteBasisPoints: 10001}), "merma"},
		{"semi-finished supply", ProductOriginMade, recipe(ProductionSupply{SupplyID: 14, Quantity: 1000, SubDivisor: 1}), ""},
		{"labor only (pure assembly)", ProductOriginMade, Product{ProductionYieldPortions: 1, ProductionLabor: []ProductionLabor{{LaborTypeID: 1, Minutes: 30}}}, ""},
		{"labor without yield", ProductOriginMade, Product{ProductionLabor: []ProductionLabor{{LaborTypeID: 1, Minutes: 30}}}, "cuántas unidades"},
		{"inactive labor type", ProductOriginMade, Product{ProductionYieldPortions: 1, ProductionLabor: []ProductionLabor{{LaborTypeID: 2, Minutes: 30}}}, "no existe"},
		{"repeated labor type", ProductOriginMade, Product{ProductionYieldPortions: 1, ProductionLabor: []ProductionLabor{
			{LaborTypeID: 1, Minutes: 30}, {LaborTypeID: 1, Minutes: 10}}}, "repetido"},
		{"labor without minutes", ProductOriginMade, Product{ProductionYieldPortions: 1, ProductionLabor: []ProductionLabor{{LaborTypeID: 1}}}, "minutos"},
	}
	for _, testCase := range cases {
		product := testCase.product
		product.Name, product.Origin = "Lomo saltado", testCase.origin
		err := ValidateProductProduction(&product, suppliesByID, laborTypesByID)
		if testCase.wantError == "" && err != nil {
			t.Errorf("%s: unexpected error %v", testCase.name, err)
		}
		if testCase.wantError != "" && (err == nil || !strings.Contains(err.Error(), testCase.wantError)) {
			t.Errorf("%s: want error containing %q, got %v", testCase.name, testCase.wantError, err)
		}
	}
}

// A semi-finished supply's recipe may not reach the supply itself, and the chain of semi-finished
// supplies may not be deeper than MaxRecipeDepth.
func TestValidateRecipeGraph(t *testing.T) {
	semiFinished := func(id int32, recipeSupplyID int32) Product {
		supply := Product{ID: id, Name: "Semi", Status: ProductStatusSupply, Origin: ProductOriginMade, ProductionYieldPortions: 1}
		if recipeSupplyID != 0 {
			supply.ProductionSupplies = []ProductionSupply{{SupplyID: recipeSupplyID, Quantity: 1000, SubDivisor: 1}}
		}
		return supply
	}
	// 21 → 22 → 23 → 21 is a cycle; 31 → 32 → … → 37 is a chain of 7 levels.
	suppliesByID := map[int32]Product{21: semiFinished(21, 22), 22: semiFinished(22, 23), 23: semiFinished(23, 21)}
	for id := int32(31); id <= 37; id++ {
		suppliesByID[id] = semiFinished(id, core.If(id < 37, id+1, int32(0)))
	}

	cycle := suppliesByID[21]
	if err := ValidateProductProduction(&cycle, suppliesByID, nil); err == nil || !strings.Contains(err.Error(), "sí misma") {
		t.Errorf("cycle: want a self-reference error, got %v", err)
	}
	shallow := suppliesByID[33] // 33 → 34 → 35 → 36 → 37: 3 semi-finished levels below its recipe
	if err := ValidateProductProduction(&shallow, suppliesByID, nil); err != nil {
		t.Errorf("shallow chain: unexpected error %v", err)
	}
	deep := Product{ID: 1, Name: "Zapato", Origin: ProductOriginMade, ProductionYieldPortions: 1,
		ProductionSupplies: []ProductionSupply{{SupplyID: 31, Quantity: 1000, SubDivisor: 1}}}
	if err := ValidateProductProduction(&deep, suppliesByID, nil); err == nil || !strings.Contains(err.Error(), "niveles") {
		t.Errorf("deep chain: want a depth error, got %v", err)
	}
}

func TestSubUnitsWithWaste(t *testing.T) {
	// 1 kg of leather for 20 pairs, 15% waste: 40 pairs need 2300 g.
	leather := ProductionSupply{SupplyID: 1, Quantity: 1000, SubDivisor: 1000, WasteBasisPoints: 1500}
	if got := leather.SubUnitsWithWaste(40, 20); got != 2300 {
		t.Errorf("leather: want 2300, got %v", got)
	}
	// Rounded up: 1 g for 3 portions, 1 portion sold, still takes 1 g.
	salt := ProductionSupply{SupplyID: 2, Quantity: 1, SubDivisor: 1000}
	if got := salt.SubUnitsWithWaste(1, 3); got != 1 {
		t.Errorf("salt: want 1, got %v", got)
	}
}

func TestRecipeLaborStandardCost(t *testing.T) {
	// 20 portions need 30 min of a 1200-cent hour and 10 min of a 600-cent one: 700 cents per 20
	// portions, so 7 portions cost 245.
	product := Product{ProductionYieldPortions: 20, ProductionLabor: []ProductionLabor{
		{LaborTypeID: 1, Minutes: 30}, {LaborTypeID: 2, Minutes: 10},
	}}
	if got := RecipeLaborStandardCost(product, 7, map[int32]int32{1: 1200, 2: 600}); got != 245 {
		t.Errorf("want 245, got %v", got)
	}
	// A deleted type with no known rate adds nothing.
	if got := RecipeLaborStandardCost(product, 20, map[int32]int32{1: 1200}); got != 600 {
		t.Errorf("want 600, got %v", got)
	}
}
