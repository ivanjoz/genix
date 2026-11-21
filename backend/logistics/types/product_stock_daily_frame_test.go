package types

import "testing"

func TestToProductStockDailyFramesSplitsInflowAndOutflow(t *testing.T) {
	cases := []struct {
		name      string
		movement  WarehouseProductMovement
		wantLines []ProductStockDailyFrame
	}{
		{
			name:     "inflow",
			movement: WarehouseProductMovement{Date: 20000, SiteID: 1, ProductID: 7, Quantity: 5, SubQuantity: 2, SubDivisor: 6},
			wantLines: []ProductStockDailyFrame{
				{Date: 20000, SiteID: 1, ProductID: 7, SubDivisor: 6, InUnits: 5, InSub: 2},
			},
		},
		{
			name:     "outflow",
			movement: WarehouseProductMovement{Date: 20000, SiteID: 1, ProductID: 7, Quantity: -3, SubDivisor: 1},
			wantLines: []ProductStockDailyFrame{
				{Date: 20000, SiteID: 1, ProductID: 7, SubDivisor: 1, OutUnits: -3},
			},
		},
		{
			// A "set stock to X" delta may pair units and sub-units of opposite signs: 1 unit less 8
			// sixths is an outflow of 2 sixths.
			name:     "mixed signs follow the whole value",
			movement: WarehouseProductMovement{Date: 20000, SiteID: 1, ProductID: 7, Quantity: 1, SubQuantity: -8, SubDivisor: 6},
			wantLines: []ProductStockDailyFrame{
				{Date: 20000, SiteID: 1, ProductID: 7, SubDivisor: 6, OutUnits: 1, OutSub: -8},
			},
		},
		{
			name:     "no divisor reads as whole units",
			movement: WarehouseProductMovement{Date: 20000, SiteID: 1, ProductID: 7, Quantity: 4},
			wantLines: []ProductStockDailyFrame{
				{Date: 20000, SiteID: 1, ProductID: 7, SubDivisor: 1, InUnits: 4},
			},
		},
		{
			name:      "transfer inside one site adds nothing",
			movement:  WarehouseProductMovement{Date: 20000, SiteID: 1, WarehouseRefID: 9, RefSiteID: 1, ProductID: 7, Quantity: -4},
			wantLines: []ProductStockDailyFrame{},
		},
		{
			name:     "transfer between sites moves both",
			movement: WarehouseProductMovement{Date: 20000, SiteID: 1, WarehouseRefID: 9, RefSiteID: 2, ProductID: 7, Quantity: -4},
			wantLines: []ProductStockDailyFrame{
				{Date: 20000, SiteID: 1, ProductID: 7, SubDivisor: 1, OutUnits: -4},
			},
		},
	}
	for _, testCase := range cases {
		lines := testCase.movement.ToProductStockDailyFrames([]ProductStockDailyFrame{})
		if len(lines) != len(testCase.wantLines) {
			t.Fatalf("%s: got %d lines %+v, want %+v", testCase.name, len(lines), lines, testCase.wantLines)
		}
		for index := range lines {
			if lines[index] != testCase.wantLines[index] {
				t.Errorf("%s: got %+v, want %+v", testCase.name, lines[index], testCase.wantLines[index])
			}
		}
	}
}
