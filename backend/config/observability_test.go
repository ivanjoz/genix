package config

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"slices"
	"testing"
)

func TestObservabilityLogQuerySelectsOnlyFrameViewColumns(t *testing.T) {
	rows := []coreTypes.UserLog{}
	query := makeObservabilityLogQuery(&rows, 20_680, 10, 20)
	selectedColumns := make([]string, 0, len(query.GetTableInfo().ColumnsInclude))
	for _, column := range query.GetTableInfo().ColumnsInclude {
		selectedColumns = append(selectedColumns, column.Name)
	}

	expectedColumns := []string{
		"date", "request_id", "frame_route_company_agg", "route_id", "error_count", "error_ids",
	}
	if !slices.Equal(selectedColumns, expectedColumns) {
		t.Fatalf("observability log projection = %v, expected %v", selectedColumns, expectedColumns)
	}
	if slices.Contains(selectedColumns, "user_id") || slices.Contains(selectedColumns, "elapsed_ms") {
		t.Fatalf("base-only columns force an ALLOW FILTERING query: %v", selectedColumns)
	}
	hasFrameRange := slices.ContainsFunc(query.GetTableInfo().Statements, func(statement db.ColumnStatement) bool {
		return statement.Col == "frame_route_company_agg" && statement.Operator == "BETWEEN"
	})
	if !hasFrameRange {
		t.Fatalf("range predicate does not target the frame view: %#v", query.GetTableInfo().Statements)
	}
}

func TestDecodeObservabilityRequestUnixSupportsBothLayouts(t *testing.T) {
	requestUnix := int64(1_800_000_060)
	expectedDay := int16(requestUnix / secondsPerUTCDate)
	vpsID := int64(core.UnixToSunix(requestUnix))*10_000_000 + 17
	sunixMilliseconds := (requestUnix*1_000 - 1_000_000_000_000) / 2
	serverlessID := sunixMilliseconds*1_000_000 + 456_017

	for name, requestID := range map[string]int64{"vps": vpsID, "serverless": serverlessID} {
		t.Run(name, func(t *testing.T) {
			decodedUnix, err := decodeObservabilityRequestUnix(requestID, expectedDay)
			if err != nil {
				t.Fatal(err)
			}
			if decodedUnix != requestUnix {
				t.Fatalf("decoded %d, expected %d", decodedUnix, requestUnix)
			}
		})
	}

	if _, err := decodeObservabilityRequestUnix(vpsID, expectedDay+1); err == nil {
		t.Fatal("request ID with the wrong Unix day was accepted")
	}
}

func TestMakeObservabilityFramesCombinesCreditsAndExactErrors(t *testing.T) {
	firstUnix := int64(1_800_000_000)
	firstFrame := unixToObservabilityTimeFrame(firstUnix)
	requestUnix := firstUnix + 60
	requestID := int64(core.UnixToSunix(requestUnix))*10_000_000 + 1
	creditRows := []coreTypes.CreditUsageCompany{{
		TimeFrame: firstFrame,
		// Route 34, one-byte values, 10 CPU and 3 inference credits.
		UsedCredits: []byte{0x00, 0x88, 0x0A, 0x03},
	}}
	logRows := []coreTypes.UserLog{{
		Date:       int16(requestUnix / secondsPerUTCDate),
		RequestID:  requestID,
		RouteID:    34,
		ErrorCount: 2,
		ErrorIDs:   []int32{11, 12},
	}}

	frames, err := makeObservabilityFrames(firstFrame, firstFrame+1, creditRows, logRows)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || len(frames[0].Details) != 1 || len(frames[1].Details) != 0 {
		t.Fatalf("unexpected frame shape: %#v", frames)
	}
	detail := frames[0].Details[0]
	if detail.CPU != 10 || detail.Inference != 3 || detail.EstimatedRequests != 5 {
		t.Fatalf("unexpected credit totals: %#v", detail)
	}
	if detail.FailedRequests != 1 || detail.ErrorOccurrences != 2 {
		t.Fatalf("failed requests and occurrences were conflated: %#v", detail)
	}
	if !slices.Equal(detail.ErrorIDs, []int32{11, 12}) ||
		!slices.Equal(detail.ErrorIDCounts, []uint64{1, 1}) {
		t.Fatalf("unexpected error counts: %#v", detail)
	}
	if frames[0].ID != core.UnixToSunix(firstUnix) || frames[0].Updated != frames[0].ID {
		t.Fatalf("frame identity is not its frame-start SUnix: %#v", frames[0])
	}
}

func TestObservabilityEstimatesOnlyMeteredMethods(t *testing.T) {
	if estimated := estimateObservabilityRequests("GET.products", 11); estimated != 5 {
		t.Fatalf("GET estimate = %d", estimated)
	}
	if estimated := estimateObservabilityRequests("POST.products", 11); estimated != 2 {
		t.Fatalf("POST estimate = %d", estimated)
	}
	if estimated := estimateObservabilityRequests("PUT.purchase-orders", 11); estimated != 0 {
		t.Fatalf("PUT estimate should be unavailable, got %d", estimated)
	}
}

func TestObservabilityOverlapIDsForceAbsoluteReplacement(t *testing.T) {
	firstFrame := unixToObservabilityTimeFrame(1_800_000_000)
	frames, err := makeObservabilityFrames(firstFrame, firstFrame+1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	watermark := frames[1].ID
	idsToRemove := makeObservabilityFrameIDsToRemove(watermark, firstFrame, frames)
	for _, frame := range frames {
		if !slices.Contains(idsToRemove, frame.ID) {
			t.Fatalf("resent frame %d was not marked for replacement: %v", frame.ID, idsToRemove)
		}
	}
}

func TestObservabilityRoutesHaveValidCacheIDs(t *testing.T) {
	routes := makeObservabilityRoutes()
	if len(routes) != len(core.APIRouteNames) {
		t.Fatalf("route metadata count = %d, expected %d", len(routes), len(core.APIRouteNames))
	}
	for _, route := range routes {
		if route.ID <= 0 || route.Route == "" || route.Updated != observabilityRoutesVersion || route.Status != 1 {
			t.Fatalf("invalid cached route metadata: %#v", route)
		}
	}
}

func TestMakeRequestErrorsByIDPreservesHashCollisions(t *testing.T) {
	requestedIDs := []int32{7, 9}
	rows := []coreTypes.RequestError{
		{ID: 7, CodeLine: "z.go:9", Text: "second", Updated: 2},
		{ID: 7, CodeLine: "a.go:1", Text: "first", Updated: 1},
	}
	grouped := makeRequestErrorsByID(requestedIDs, rows)
	if len(grouped) != 2 || len(grouped[0].Entries) != 2 || len(grouped[1].Entries) != 0 {
		t.Fatalf("unexpected grouped records: %#v", grouped)
	}
	if grouped[0].Entries[0].CodeLine != "a.go:1" || grouped[0].Entries[1].CodeLine != "z.go:9" {
		t.Fatalf("collision entries are not sorted: %#v", grouped[0].Entries)
	}
}
