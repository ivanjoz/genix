package config

import (
	"app/core"
	"slices"
	"testing"
)

// The row-mapping loops these handlers run are inline now, so what stays unit-testable here is the
// route catalog. The blob decoding underneath them, including every rejection path, is covered by
// TestDecodeCreditUsage* in credit_usage_test.go.
func TestMakeCreditRouteNamesIsSortedAndSkipsTheUnmatchedRoute(t *testing.T) {
	routes := makeCreditRouteNames()
	publishableRoutes := 0
	for routeID := range core.APIRouteNames {
		if routeID > 0 {
			publishableRoutes++
		}
	}
	if len(routes) != publishableRoutes {
		t.Fatalf("expected the whole catalog, got %d of %d", len(routes), publishableRoutes)
	}
	if !slices.IsSortedFunc(routes, func(left, right creditRouteName) int { return int(left.ID - right.ID) }) {
		t.Fatal("the catalog must be sorted so a client can merge it without re-sorting")
	}
	for _, route := range routes {
		// Route zero means an unmatched API and has no valid cache identity.
		if route.ID <= 0 {
			t.Fatalf("route id %d must not be published", route.ID)
		}
		if route.Updated != companyCreditRoutesVersion || route.Status != 1 {
			t.Fatalf("catalog rows carry the version as their watermark: %#v", route)
		}
	}
}

// The frames are written by the Rust credit daemon and read here, so the two have to agree on where
// a day starts. DAY_ZONE_OFFSET_SECONDS in server_utils/src/limiter/time_frame.rs is the other half
// of this constant; if one moves without the other, the reader queries frames the writer never
// wrote and every report silently comes back empty.
func TestCurrentDailyTimeFrameFollowsTheLimaDayNotTheHostZone(t *testing.T) {
	const limaDay = int64(20_683)
	limaMidnight := limaDay*int64(secondsPerUTCDate) - creditDayZoneOffsetSeconds
	defer core.SetHistoricalUnix(0)

	for _, testCase := range []struct {
		name        string
		unixSeconds int64
		expectedDay int64
	}{
		{"local midnight", limaMidnight, limaDay},
		{"local midday", limaMidnight + 12*3_600, limaDay},
		// 19:00 local is already tomorrow in UTC. This is the window that emptied the reports.
		{"local evening, past UTC midnight", limaMidnight + 19*3_600, limaDay},
		{"last second of the local day", limaMidnight + int64(secondsPerUTCDate) - 1, limaDay},
		{"first second of the next local day", limaMidnight + int64(secondsPerUTCDate), limaDay + 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			core.SetHistoricalUnix(testCase.unixSeconds)
			expectedFrame := dailyTimeFramePrefix + int32(testCase.expectedDay)
			if frame := currentDailyTimeFrame(); frame != expectedFrame {
				t.Fatalf("frame = %d, want %d", frame, expectedFrame)
			}
		})
	}
}

// The month boundary is compared against the daemon's own month_start_day, so it has to be read in
// the same business day as the frames.
func TestCurrentMonthStartDayIsReadInTheLimaDay(t *testing.T) {
	defer core.SetHistoricalUnix(0)
	// 2026-08-01 20:00 Lima is 2026-08-02 01:00 UTC: still August either way, but the day index
	// must be August's first day, which is the value the daemon stores.
	core.SetHistoricalUnix(int64(20_666)*int64(secondsPerUTCDate) - creditDayZoneOffsetSeconds + 20*3_600)
	if monthStart := currentMonthStartDay(); monthStart != 20_666 {
		t.Fatalf("month start = %d, want 20666", monthStart)
	}
	// 2026-07-31 21:00 Lima is 2026-08-01 02:00 UTC: a UTC reading would already say August.
	core.SetHistoricalUnix(int64(20_665)*int64(secondsPerUTCDate) - creditDayZoneOffsetSeconds + 21*3_600)
	if monthStart := currentMonthStartDay(); monthStart != 20_635 {
		t.Fatalf("month start = %d, want 20635 (July)", monthStart)
	}
}
