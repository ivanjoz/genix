package types

import (
	"strings"
	"testing"
)

const (
	leatherID   = int32(30) // m² with 100 sub-units (dm²)
	threadID    = int32(31) // whole units
	sandalID    = int32(40)
	bootID      = int32(41)
	shoemakerID = int32(1)
)

func shoeFixture() (map[int32]Product, map[int32]Product) {
	sizes := []ProductPresentation{{ID: 36, Status: 1}, {ID: 37, Status: 1}, {ID: 38, Status: 0}}
	productsByID := map[int32]Product{
		// 10 pairs: 2 m² of leather (10% waste) and 1 thread; 60 minutes of shoemaker.
		sandalID: {ID: sandalID, Name: "Sandalia", Status: ProductStatusActive, Origin: ProductOriginMade, Presentations: sizes,
			ProductionYieldPortions: 10, ProductionSupplies: []ProductionSupply{
				{SupplyID: leatherID, Quantity: 2000, SubDivisor: 100, WasteBasisPoints: 1000}, {SupplyID: threadID, Quantity: 1000, SubDivisor: 1},
			}, ProductionLabor: []ProductionLabor{{LaborTypeID: shoemakerID, Minutes: 60}}},
		// 1 pair: 50 dm² of leather; 30 minutes.
		bootID: {ID: bootID, Name: "Bota", Status: ProductStatusActive, Origin: ProductOriginMade, ProductionYieldPortions: 1,
			ProductionSupplies: []ProductionSupply{{SupplyID: leatherID, Quantity: 50, SubDivisor: 100}},
			ProductionLabor:    []ProductionLabor{{LaborTypeID: shoemakerID, Minutes: 30}}},
	}
	suppliesByID := map[int32]Product{
		leatherID: {ID: leatherID, Name: "Cuero", Status: ProductStatusSupply, SbuQuantity: 100},
		threadID:  {ID: threadID, Name: "Hilo", Status: ProductStatusSupply},
	}
	return productsByID, suppliesByID
}

func TestValidateProductionOrderLines(t *testing.T) {
	productsByID, _ := shoeFixture()
	productsByID[50] = Product{ID: 50, Name: "Comprado", Status: ProductStatusActive, Origin: ProductOriginBought}
	cases := []struct {
		name      string
		lines     []ProductionOrderLine
		wantError string
	}{
		{"sizes of a sandal and a boot", []ProductionOrderLine{
			{ProductID: sandalID, PresentationID: 36, PlannedUnits: 10}, {ProductID: bootID, PlannedUnits: 2}}, ""},
		{"no lines", nil, "al menos un producto"},
		{"bought product", []ProductionOrderLine{{ProductID: 50, PlannedUnits: 1}}, "no es fabricado"},
		{"inactive size", []ProductionOrderLine{{ProductID: sandalID, PresentationID: 38, PlannedUnits: 1}}, "no está activa"},
		{"presentation 0 on a product with sizes", []ProductionOrderLine{{ProductID: sandalID, PlannedUnits: 1}}, "no está activa"},
		{"repeated size", []ProductionOrderLine{
			{ProductID: sandalID, PresentationID: 36, PlannedUnits: 1}, {ProductID: sandalID, PresentationID: 36, PlannedUnits: 2}}, "repetida"},
		{"no units", []ProductionOrderLine{{ProductID: bootID, PlannedUnits: 0}}, "mayores a 0"},
	}
	for _, testCase := range cases {
		err := ValidateProductionOrderLines(testCase.lines, productsByID)
		if testCase.wantError == "" && err != nil {
			t.Errorf("%s: unexpected error %v", testCase.name, err)
		}
		if testCase.wantError != "" && (err == nil || !strings.Contains(err.Error(), testCase.wantError)) {
			t.Errorf("%s: want error containing %q, got %v", testCase.name, testCase.wantError, err)
		}
	}
}

func TestExplodeProductionOrder(t *testing.T) {
	productsByID, suppliesByID := shoeFixture()
	lines := []ProductionOrderLine{
		{ProductID: sandalID, PresentationID: 36, PlannedUnits: 12}, {ProductID: sandalID, PresentationID: 37, PlannedUnits: 8},
		{ProductID: bootID, PlannedUnits: 3},
	}
	// Leather: 20 sandals × 200 dm² ÷ 10 × 1.1 = 440, + 3 boots × 50 = 150 → 590, × 1.05 adjustment = 619.5 → 620.
	// Thread: 20 × 1 unit ÷ 10 = 2 × 1.05 = 2.1 → 3. Shoemaker: 20 × 60 ÷ 10 + 3 × 30 = 210 minutes at 1200/h = 4200.
	supplies, labor, err := ExplodeProductionOrder(lines, 500, productsByID, suppliesByID, map[int32]int32{shoemakerID: 1200})
	if err != nil {
		t.Fatal(err)
	}
	if len(supplies) != 2 || supplies[0] != (ProductionOrderSupply{SupplyID: leatherID, SubDivisor: 100, Planned: 620}) ||
		supplies[1] != (ProductionOrderSupply{SupplyID: threadID, SubDivisor: 1, Planned: 3}) {
		t.Fatalf("supplies: %+v", supplies)
	}
	if len(labor) != 1 || labor[0] != (ProductionOrderLabor{LaborTypeID: shoemakerID, PlannedMinutes: 210, PlannedCost: 4200}) {
		t.Fatalf("labor: %+v", labor)
	}

	staleRecipe := productsByID[bootID]
	staleRecipe.ProductionSupplies = []ProductionSupply{{SupplyID: leatherID, Quantity: 50, SubDivisor: 10}}
	productsByID[bootID] = staleRecipe
	if _, _, err := ExplodeProductionOrder(lines, 0, productsByID, suppliesByID, nil); err == nil || !strings.Contains(err.Error(), "sub-unidad anterior") {
		t.Fatalf("stale divisor: want an error, got %v", err)
	}
}

func TestShareReceiptSupplyAndLabor(t *testing.T) {
	productsByID, _ := shoeFixture()
	// 10 sandals (+2 scrapped) weigh 12 × 220 ÷ 10 = 264 dm² of leather; 4 boots weigh 200. 500 consumed.
	received := []ReceiptProductUnits{{ProductID: sandalID, Good: 10, Scrapped: 2}, {ProductID: bootID, Good: 4}}
	takes := ShareReceiptSupply(leatherID, 500, received, productsByID)
	if len(takes) != 2 || takes[0].SubUnits != 285 || takes[1].SubUnits != 215 {
		t.Fatalf("leather takes: %+v", takes)
	}
	// Only sandals use thread: all of it goes to them.
	if takes := ShareReceiptSupply(threadID, 2, received, productsByID); len(takes) != 1 || takes[0] != (ProductSupplyTake{ProductID: sandalID, SubUnits: 2}) {
		t.Fatalf("thread takes: %+v", takes)
	}
	// A product with only scrapped units carries no value: the good units absorb its share.
	scrapOnly := []ReceiptProductUnits{{ProductID: sandalID, Scrapped: 5}, {ProductID: bootID, Good: 1}}
	if takes := ShareReceiptSupply(leatherID, 100, scrapOnly, productsByID); len(takes) != 1 || takes[0].ProductID != bootID {
		t.Fatalf("scrap only: %+v", takes)
	}

	// Labor 1000: sandals' recipe labor for 10 good = 60 min, boots' for 4 = 120 min.
	shares := ShareReceiptLabor(1000, received, productsByID, map[int32]int32{shoemakerID: 600})
	if shares[0] != 333 || shares[1] != 667 {
		t.Fatalf("labor shares: %v", shares)
	}
}

func TestIsProductionOrderFullyReceived(t *testing.T) {
	lines := []ProductionOrderLine{{PlannedUnits: 10, ProducedUnits: 12}, {PlannedUnits: 5, ProducedUnits: 4, ScrappedUnits: 1}}
	if IsProductionOrderFullyReceived(lines) {
		t.Fatal("a scrapped unit does not fulfil a planned one")
	}
	lines[1].ProducedUnits = 5
	if !IsProductionOrderFullyReceived(lines) {
		t.Fatal("every line reached its plan (one over-produced)")
	}
}

func TestCappedLaborAmountAndOrderLaborRecorded(t *testing.T) {
	// 90 minutes at 1000/h = 1500; the month's cap leaves 1200 after 8800 booked of 10000.
	if got := CappedLaborAmount(90, 1000, 10000, 8800); got != 1200 {
		t.Fatalf("capped: got %v, want 1200", got)
	}
	if got := CappedLaborAmount(90, 1000, 10000, 12000); got != 0 {
		t.Fatalf("over the cap: got %v, want 0", got)
	}
	if got := CappedLaborAmount(90, 1000, -1, 50000); got != 1500 {
		t.Fatalf("off payroll, no cap: got %v, want 1500", got)
	}

	planned := []ProductionOrderLabor{{LaborTypeID: 1, PlannedMinutes: 600, PlannedCost: 10000, RecordedMinutes: 99}}
	entries := []ProductionLaborEntry{
		{LaborTypeID: 1, Minutes: 120, Amount: 2000, Status: 1},
		{LaborTypeID: 2, Minutes: 30, Amount: 400, Status: 1}, // logged but not planned
		{LaborTypeID: 1, Minutes: 60, Amount: 900, Status: 0}, // deleted
	}
	labor, recordedAmount := ProductionOrderLaborRecorded(planned, entries)
	if recordedAmount != 2400 || len(labor) != 2 || labor[0].RecordedMinutes != 120 || labor[0].PlannedMinutes != 600 ||
		labor[1] != (ProductionOrderLabor{LaborTypeID: 2, RecordedMinutes: 30, RecordedCost: 400}) {
		t.Fatalf("recorded: %v %+v", recordedAmount, labor)
	}
	if planned[0].RecordedMinutes != 99 {
		t.Fatal("the planned slice must not be modified")
	}
}
