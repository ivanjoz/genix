package types

import (
	"app/core"
	logistics "app/logistics/types"
	"slices"
	"testing"
)

func TestOrderSupplyStockBucketsWalksEarliestExpiryFirstAndExpiredLast(t *testing.T) {
	const today = int16(100)
	buckets := []SupplyStockBucket{
		{LotID: 3},                      // released, no expiry
		{LotID: 5, ExpirationDate: 200}, // released
		{LotID: 6, ExpirationDate: 90},  // expired: warned, taken last
		{LotID: 2, QualityStatus: logistics.LotQualityQuarantine},
		{LotID: 4, ExpirationDate: 150},     // released, expires first
		{LotID: 1, ExpirationDate: 150},     // same expiry, older lot
		{Quantity: core.Quantity{Units: 1}}, // the free bucket
	}
	OrderSupplyStockBuckets(buckets, today)
	gotLotIDs := []int32{}
	for _, bucket := range buckets {
		gotLotIDs = append(gotLotIDs, bucket.LotID)
	}
	if wantLotIDs := []int32{0, 1, 4, 5, 3, 6, 2}; !slices.Equal(gotLotIDs, wantLotIDs) {
		t.Fatalf("walk order: got %v, want %v", gotLotIDs, wantLotIDs)
	}
}

func TestBlockedLotsAreNeitherAvailableNorWalked(t *testing.T) {
	// 1 kg free, 2 kg in lot 7 (Cuarentena), 1 kg in lot 8 (Liberado): 4 kg worth 4000 cents.
	stock := SupplyStock{SubDivisor: 1000, StockValue: 4000, Buckets: []SupplyStockBucket{
		{Quantity: core.Quantity{Units: 1}},
		{LotID: 8, QualityStatus: logistics.LotQualityReleased, Quantity: core.Quantity{Units: 1}},
		{LotID: 7, QualityStatus: logistics.LotQualityQuarantine, Quantity: core.Quantity{Units: 2}},
	}}
	atDivisor, err := ReadSupplyStockAtDivisor(stock, 1000)
	if err != nil {
		t.Fatal(err)
	}
	// The balance values the exits, so it keeps the blocked lot; the availability does not.
	if atDivisor.BalanceSubUnits != 4000 || atDivisor.AvailableSubUnits != 2000 {
		t.Fatalf("balance %v available %v, want 4000 / 2000", atDivisor.BalanceSubUnits, atDivisor.AvailableSubUnits)
	}

	consumptions, err := WalkSupplyBuckets(riceID, 1000, stock, atDivisor, []ProductSupplyTake{{ProductID: lomoID, SubUnits: 1500}})
	if err != nil {
		t.Fatal(err)
	}
	wantConsumptions := []SupplyConsumption{
		{ProductID: lomoID, SupplyID: riceID, SubUnits: 1000, SubDivisor: 1000, EstimatedValue: 1000},
		{ProductID: lomoID, SupplyID: riceID, LotID: 8, SubUnits: 500, SubDivisor: 1000, EstimatedValue: 500},
	}
	if !slices.Equal(consumptions, wantConsumptions) {
		t.Fatalf("consumptions: got %+v, want %+v", consumptions, wantConsumptions)
	}

	if _, err := WalkSupplyBuckets(riceID, 1000, stock, atDivisor, []ProductSupplyTake{{ProductID: lomoID, SubUnits: 2500}}); err == nil {
		t.Fatal("a take that needs the blocked lot must be refused")
	}
}

func TestPlanDailyCloseSkipsALotInQuarantine(t *testing.T) {
	input := dailyCloseFixture()
	// Lot 7's 2 kg of rice are in Cuarentena: only the free kg is available for 2250 g needed.
	riceStock := input.SupplyStockByID[riceID]
	riceStock.Buckets = []SupplyStockBucket{
		{Quantity: core.Quantity{Units: 1}},
		{LotID: 7, QualityStatus: logistics.LotQualityQuarantine, Quantity: core.Quantity{Units: 2}},
	}
	input.SupplyStockByID[riceID] = riceStock
	plan := PlanDailyClose(input)
	if plan.Supplies[0].Available != 1000 || plan.Supplies[0].Consumed != 1000 {
		t.Fatalf("rice with a lot in quarantine: %+v", plan.Supplies[0])
	}
	for _, consumption := range plan.Consumptions {
		if consumption.LotID == 7 {
			t.Fatalf("the lot in quarantine was consumed: %+v", consumption)
		}
	}
}
