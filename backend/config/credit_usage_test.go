package config

import (
	coreTypes "app/core/types"
	"reflect"
	"testing"
)

// The vectors here are the ones the Rust encoder produces (credits_blob.rs). Two independent
// implementations of one format only stay in agreement if both sides are pinned to the same bytes.
func TestDecodeCreditUsageSumsCanonicalRoutes(t *testing.T) {
	// Route 1 with one-byte values, then route 3 as the documented two-byte example:
	// header (1<<2)|0 = 0x0004, then (3<<2)|1 = 0x000D.
	encoded := []byte{0x00, 0x04, 0x05, 0x07, 0x00, 0x0D, 0x01, 0x2C, 0x00, 0x19}
	routeTotals := map[int16]creditUsageTotals{}

	got, err := decodeCreditUsage(encoded, routeTotals)
	if err != nil {
		t.Fatalf("decodeCreditUsage() error = %v", err)
	}

	want := creditUsageTotals{CPU: 305, Inference: 32}
	if got != want {
		t.Fatalf("decodeCreditUsage() = %+v, want %+v", got, want)
	}
	wantRoutes := map[int16]creditUsageTotals{
		1: {CPU: 5, Inference: 7},
		3: {CPU: 300, Inference: 25},
	}
	if !reflect.DeepEqual(routeTotals, wantRoutes) {
		t.Fatalf("per-route split = %+v, want %+v", routeTotals, wantRoutes)
	}
}

// The route field is fourteen bits wide now. A number past the old six-bit ceiling has to survive
// the widened header, and the top of the range is where a bad shift would show.
func TestDecodeCreditUsageReadsRoutesPastTheOldSixBitCeiling(t *testing.T) {
	// Routes 103 and 16383, both with one-byte values.
	encoded := []byte{0x01, 0x9C, 0x02, 0x03, 0xFF, 0xFC, 0x01, 0x01}
	routeTotals := map[int16]creditUsageTotals{}

	if _, err := decodeCreditUsage(encoded, routeTotals); err != nil {
		t.Fatalf("decodeCreditUsage() error = %v", err)
	}

	wantRoutes := map[int16]creditUsageTotals{
		103:    {CPU: 2, Inference: 3},
		16_383: {CPU: 1, Inference: 1},
	}
	if !reflect.DeepEqual(routeTotals, wantRoutes) {
		t.Fatalf("per-route split = %+v, want %+v", routeTotals, wantRoutes)
	}
}

// The same totals must accumulate across rows, because the breakdown spans the whole range while
// each blob only covers one day.
func TestDecodeCreditUsageAccumulatesOneRouteAcrossRows(t *testing.T) {
	routeTotals := map[int16]creditUsageTotals{}
	dayBlob := []byte{0x00, 0x04, 0x05, 0x07}

	for range 3 {
		if _, err := decodeCreditUsage(dayBlob, routeTotals); err != nil {
			t.Fatalf("decodeCreditUsage() error = %v", err)
		}
	}

	if got := routeTotals[1]; got != (creditUsageTotals{CPU: 15, Inference: 21}) {
		t.Fatalf("route 1 accumulated %+v across three rows", got)
	}
}

func TestDecodeCreditUsageRejectsInvalidEncoding(t *testing.T) {
	invalidBlobs := [][]byte{
		{0x00, 0x05, 0x00},                               // Truncated two-byte values.
		{0x00},                                           // A trailing byte that is half a header.
		{0x00, 0x04, 0x01, 0x01, 0x00, 0x04, 0x02, 0x02}, // Duplicate route.
		{0x00, 0x08, 0x01, 0x01, 0x00, 0x04, 0x02, 0x02}, // Routes going backwards.
		{0x00, 0x00, 0x00, 0x00},                         // All-zero route.
		{0x00, 0x01, 0x00, 0x01, 0x00, 0x01},             // Non-minimal width.
	}
	for _, invalidBlob := range invalidBlobs {
		if _, err := decodeCreditUsage(invalidBlob, nil); err == nil {
			t.Fatalf("decodeCreditUsage(%x) unexpectedly succeeded", invalidBlob)
		}
	}
}

func TestMakeCreditUsageScopeZeroFillsFifteenUTCDays(t *testing.T) {
	rows := []coreTypes.CreditUsage{{
		TimeFrame:   dailyTimeFramePrefix + 103,
		UsedCredits: []byte{0x00, 0x04, 0x05, 0x07},
	}}
	got, err := makeCreditUsageScope(rows, 100, 1_000, 500)
	if err != nil {
		t.Fatalf("makeCreditUsageScope() error = %v", err)
	}
	if len(got.Days) != int(creditUsageDaysCount) {
		t.Fatalf("len(Days) = %d, want %d", len(got.Days), creditUsageDaysCount)
	}
	if got.Days[0].Day != 100 || got.Days[14].Day != 114 {
		t.Fatalf("day range = %d..%d, want 100..114", got.Days[0].Day, got.Days[14].Day)
	}
	wantUsedDay := creditUsageDay{Day: 103, CPU: 5, Inference: 7}
	if !reflect.DeepEqual(got.Days[3], wantUsedDay) {
		t.Fatalf("Days[3] = %+v, want %+v", got.Days[3], wantUsedDay)
	}
}

// The breakdown is ordered by cost so a client can render its head and stop, and it names each
// route: the numbers alone say nothing, and the generated table is the only authority on them.
func TestMakeCreditUsageScopeRanksRoutesByCost(t *testing.T) {
	rows := []coreTypes.CreditUsage{{
		TimeFrame: dailyTimeFramePrefix + 103,
		// Route 1 with cpu 5, then route 34 with cpu 300.
		UsedCredits: []byte{0x00, 0x04, 0x05, 0x07, 0x00, 0x89, 0x01, 0x2C, 0x00, 0x19},
	}}

	got, err := makeCreditUsageScope(rows, 100, 1_000, 500)
	if err != nil {
		t.Fatalf("makeCreditUsageScope() error = %v", err)
	}

	if len(got.Routes) != 2 {
		t.Fatalf("expected two routes in the breakdown, got %d: %+v", len(got.Routes), got.Routes)
	}
	if got.Routes[0].RouteID != 34 || got.Routes[0].CPU != 300 {
		t.Fatalf("the most expensive route did not sort first: %+v", got.Routes)
	}
	if got.Routes[0].Route != "GET.products" {
		t.Fatalf("route 34 was named %q; the generated table calls it GET.products", got.Routes[0].Route)
	}
}
