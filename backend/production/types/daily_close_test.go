package types

import (
	"app/core"
	"strings"
	"testing"
)

const (
	riceID   = int32(10) // kg with 1000 g, stock account 252
	saltID   = int32(11) // whole units
	lomoID   = int32(20)
	arrozID  = int32(21)
	cafeID   = int32(22) // standard cost only
	sinCosto = int32(23) // neither recipe nor standard cost
)

func dailyCloseFixture() DailyCloseInput {
	return DailyCloseInput{
		SoldUnitsByProduct: map[int32]int32{lomoID: 10, arrozID: 5, cafeID: 3},
		ProductsByID: map[int32]Product{
			// 20 portions: 1 kg 500 g of rice and 2 units of salt.
			lomoID: {ID: lomoID, Name: "Lomo", ProductionYieldPortions: 20, ProductionExtraCostPerUnit: 100, ProductionSupplies: []ProductionSupply{
				{SupplyID: riceID, Quantity: 1500, SubDivisor: 1000}, {SupplyID: saltID, Quantity: 2000, SubDivisor: 1},
			}},
			// 1 portion: 300 g of rice.
			arrozID: {ID: arrozID, Name: "Arroz", ProductionYieldPortions: 1, ProductionSupplies: []ProductionSupply{
				{SupplyID: riceID, Quantity: 300, SubDivisor: 1000},
			}},
			cafeID:   {ID: cafeID, Name: "Café", ProductionStandardCost: 250},
			sinCosto: {ID: sinCosto, Name: "Sin costo"},
		},
		SuppliesByID: map[int32]Product{
			riceID: {ID: riceID, Name: "Arroz kg", Status: ProductStatusSupply, SbuQuantity: 1000, SbuUnit: "g", AccountCode: 6032},
			saltID: {ID: saltID, Name: "Sal", Status: ProductStatusSupply, AccountCode: 6032},
		},
		SupplyStockByID: map[int32]DailyCloseSupplyStock{
			// 1 kg free + 2 kg in lot 7, worth 3000 cents (1000 per kg).
			riceID: {SubDivisor: 1000, StockValue: 3000, Buckets: []DailyCloseStockBucket{
				{Quantity: core.Quantity{Units: 1}}, {LotID: 7, Quantity: core.Quantity{Units: 2}},
			}},
			saltID: {SubDivisor: 1, StockValue: 500, Buckets: []DailyCloseStockBucket{{Quantity: core.Quantity{Units: 5}}}},
		},
	}
}

func TestPlanDailyCloseConsumesTheRecipesFreeBucketFirst(t *testing.T) {
	plan := PlanDailyClose(dailyCloseFixture())
	if len(plan.BlockingReasons) > 0 {
		t.Fatalf("unexpected blocking reasons: %v", plan.BlockingReasons)
	}
	// Rice: lomo needs ceil(1500 × 10 / 20) = 750 g, arroz 300 × 5 = 1500 g; 2250 g of 3000 g.
	// Salt: ceil(2 × 10 / 20) = 1 unit.
	wantSupplies := []ProductionCloseSupply{
		{SupplyID: riceID, SubDivisor: 1000, Needed: 2250, Available: 3000, Consumed: 2250},
		{SupplyID: saltID, SubDivisor: 1, Needed: 1, Available: 5, Consumed: 1},
	}
	for index, want := range wantSupplies {
		if plan.Supplies[index] != want {
			t.Errorf("supply %v: got %+v, want %+v", index, plan.Supplies[index], want)
		}
	}
	// Lomo takes 750 g of the free kg; arroz the other 250 g, then 1250 g of lot 7.
	wantConsumptions := []DailyCloseConsumption{
		{ProductID: lomoID, SupplyID: riceID, SubUnits: 750, SubDivisor: 1000, EstimatedValue: 750},
		{ProductID: arrozID, SupplyID: riceID, SubUnits: 250, SubDivisor: 1000, EstimatedValue: 250},
		{ProductID: arrozID, SupplyID: riceID, LotID: 7, SubUnits: 1250, SubDivisor: 1000, EstimatedValue: 1250},
		{ProductID: lomoID, SupplyID: saltID, SubUnits: 1, SubDivisor: 1, EstimatedValue: 100},
	}
	if len(plan.Consumptions) != len(wantConsumptions) {
		t.Fatalf("consumptions: got %+v", plan.Consumptions)
	}
	for index, want := range wantConsumptions {
		if plan.Consumptions[index] != want {
			t.Errorf("consumption %v: got %+v, want %+v", index, plan.Consumptions[index], want)
		}
	}
	// Lomo: 750 + 100 of supplies + 100 × 10 extra. Arroz: 1500. Café: 250 × 3 standard.
	wantProducts := []ProductionCloseProduct{
		{ProductID: lomoID, Units: 10, CostSource: CloseCostSourceSupplies, Value: 1850},
		{ProductID: arrozID, Units: 5, CostSource: CloseCostSourceSupplies, Value: 1500},
		{ProductID: cafeID, Units: 3, CostSource: CloseCostSourceStandard, Value: 750},
	}
	for index, want := range wantProducts {
		if plan.Products[index] != want {
			t.Errorf("product %v: got %+v, want %+v", index, plan.Products[index], want)
		}
	}
}

func TestPlanDailyCloseSharesAShortSupply(t *testing.T) {
	input := dailyCloseFixture()
	// Only 1 kg of rice for 2250 g needed: lomo gets floor(1000 × 750 / 2250) = 333, arroz the rest.
	input.SupplyStockByID[riceID] = DailyCloseSupplyStock{SubDivisor: 1000, StockValue: 1000,
		Buckets: []DailyCloseStockBucket{{Quantity: core.Quantity{Units: 1}}}}
	plan := PlanDailyClose(input)
	if len(plan.BlockingReasons) > 0 {
		t.Fatalf("a short supply does not block without the strict flag: %v", plan.BlockingReasons)
	}
	if plan.Supplies[0].Consumed != 1000 || plan.Consumptions[0].SubUnits != 333 || plan.Consumptions[1].SubUnits != 667 {
		t.Fatalf("shares: %+v / %+v", plan.Supplies[0], plan.Consumptions)
	}
	if !plan.Products[0].ShortSupplies || !plan.Products[1].ShortSupplies || plan.Products[2].ShortSupplies {
		t.Fatalf("short flags: %+v", plan.Products)
	}

	input.IsStrict = true
	strictPlan := PlanDailyClose(input)
	if len(strictPlan.BlockingReasons) != 1 || !strings.Contains(strictPlan.BlockingReasons[0], "se necesitan 2 + 250 g y hay 1") {
		t.Fatalf("strict: %v", strictPlan.BlockingReasons)
	}
}

func TestPlanDailyCloseExtraCostAloneIsValueEnough(t *testing.T) {
	input := dailyCloseFixture()
	// Rice and salt worth nothing: lomo still has its extra cost (100 × 10), arroz has no cost at all.
	input.SupplyStockByID[riceID] = DailyCloseSupplyStock{SubDivisor: 1000, Buckets: []DailyCloseStockBucket{{Quantity: core.Quantity{Units: 3}}}}
	input.SupplyStockByID[saltID] = DailyCloseSupplyStock{SubDivisor: 1, Buckets: []DailyCloseStockBucket{{Quantity: core.Quantity{Units: 5}}}}
	plan := PlanDailyClose(input)
	if plan.Products[0].Value != 1000 {
		t.Fatalf("lomo value: got %v, want 1000", plan.Products[0].Value)
	}
	reasons := strings.Join(plan.BlockingReasons, " | ")
	if strings.Contains(reasons, `"Lomo"`) || !strings.Contains(reasons, `"Arroz" no tiene costo`) {
		t.Fatalf("blocking reasons: %v", plan.BlockingReasons)
	}
}

func TestPlanDailyCloseBlockingReasons(t *testing.T) {
	cases := []struct {
		name       string
		edit       func(input *DailyCloseInput)
		wantReason string
	}{
		{"no cost source", func(input *DailyCloseInput) { input.SoldUnitsByProduct[sinCosto] = 1 }, "ni costo estándar"},
		{"supply without value", func(input *DailyCloseInput) {
			input.SupplyStockByID[riceID] = DailyCloseSupplyStock{SubDivisor: 1000, Buckets: []DailyCloseStockBucket{{Quantity: core.Quantity{Units: 3}}}}
			delete(input.SoldUnitsByProduct, lomoID)
		}, "no tiene costo"},
		{"stale recipe divisor", func(input *DailyCloseInput) {
			rice := input.SuppliesByID[riceID]
			rice.SbuQuantity = 2000
			input.SuppliesByID[riceID] = rice
		}, "sub-unidad anterior"},
		{"supply on an expense account", func(input *DailyCloseInput) {
			salt := input.SuppliesByID[saltID]
			salt.AccountCode = 656
			input.SuppliesByID[saltID] = salt
		}, "no es de existencias"},
	}
	for _, testCase := range cases {
		input := dailyCloseFixture()
		testCase.edit(&input)
		plan := PlanDailyClose(input)
		if !strings.Contains(strings.Join(plan.BlockingReasons, " | "), testCase.wantReason) {
			t.Errorf("%s: want a reason containing %q, got %v", testCase.name, testCase.wantReason, plan.BlockingReasons)
		}
	}
}
