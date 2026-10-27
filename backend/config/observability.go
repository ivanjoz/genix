package config

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	observabilityDefaultHours      = int32(4)
	observabilityMaxHours          = int32(24)
	observabilityFiveMinuteSeconds = int64(300)
	observabilityFiveMinutePrefix  = int32(100_000_000)
	observabilityPlatformCompanyID = int32(0)
	observabilityRoutesVersion     = int32(1)
)

type ObservabilityRouteDetail struct {
	RouteID           int16
	CPU               uint64
	Inference         uint64
	EstimatedRequests uint64
	FailedRequests    uint64
	ErrorOccurrences  uint64
	ErrorIDs          []int32
	ErrorIDCounts     []uint64
}

type ObservabilityFrame struct {
	ID        int32                      `json:"ID" cb:"ID,1"`
	TimeFrame int32                      `cb:"TimeFrame,2"`
	Details   []ObservabilityRouteDetail `cb:"Details,3"`
	Updated   int32                      `json:"upd" cb:"upd,4"`
	Status    int8                       `json:"ss" cb:"ss,5"`
}

type ObservabilityRoute struct {
	ID      int16  `json:"ID" cb:"ID,1"`
	Route   string `cb:"Route,2"`
	Updated int32  `json:"upd" cb:"upd,3"`
	Status  int8   `json:"ss" cb:"ss,4"`
}

type ObservabilityResponse struct {
	// The window's frames whole, ending with the replace Metadata (core.WithDeltaReplace): a frame
	// that fell out of the window leaves the client's cache.
	Frames []any                `cb:"Frames,1"`
	Routes []ObservabilityRoute `json:",omitempty" cb:"Routes,2"`
}

type observabilityRouteAccumulator struct {
	ObservabilityRouteDetail
	errorCounts map[int32]uint64
}

// GetObservability returns absolute five-minute records for the whole window on every sync.
func GetObservability(req *core.HandlerArgs) core.HandlerResponse {
	windowHours := observabilityDefaultHours
	if requestedHours := req.GetQueryInt("hours"); requestedHours > 0 {
		windowHours = min(requestedHours, observabilityMaxHours)
	}

	nowUnix := time.Now().UTC().Unix()
	currentTimeFrame := unixToObservabilityTimeFrame(nowUnix)
	firstQueryFrame := currentTimeFrame - windowHours*12 + 1

	creditRows, logRows := []coreTypes.CreditUsageCompany{}, []coreTypes.UserLog{}
	queries := errgroup.Group{}
	queries.Go(func() error {
		// The platform aggregate is a company row under the reserved company id zero.
		query := db.Query(&creditRows)
		query.CompanyID.Equals(observabilityPlatformCompanyID).
			TimeFrame.Between(firstQueryFrame, currentTimeFrame)
		return query.Exec()
	})
	queries.Go(func() error {
		var err error
		logRows, err = readObservabilityLogs(firstQueryFrame, currentTimeFrame)
		return err
	})
	if err := queries.Wait(); err != nil {
		core.Log("observability query failed::", " first_frame::", firstQueryFrame,
			" current_frame::", currentTimeFrame, " err::", err)
		return req.MakeErr("No se pudo obtener la información de observabilidad.", err)
	}

	frames, err := makeObservabilityFrames(firstQueryFrame, currentTimeFrame, creditRows, logRows)
	if err != nil {
		core.Log("observability aggregation failed::", " credit_rows::", len(creditRows),
			" log_rows::", len(logRows), " err::", err)
		return req.MakeErr("No se pudo interpretar la información de observabilidad.", err)
	}
	return req.MakeResponse(ObservabilityResponse{
		Frames: core.WithDeltaReplace(frames),
		Routes: makeObservabilityRoutes(),
	})
}

func readObservabilityLogs(firstTimeFrame, lastTimeFrame int32) ([]coreTypes.UserLog, error) {
	firstUnix := observabilityTimeFrameToUnix(firstTimeFrame)
	lastUnix := observabilityTimeFrameToUnix(lastTimeFrame) + observabilityFiveMinuteSeconds - 1
	firstDay, lastDay := firstUnix/secondsPerUTCDate, lastUnix/secondsPerUTCDate
	rowsPerDay := make([][]coreTypes.UserLog, lastDay-firstDay+1)

	queries := errgroup.Group{}
	for dayOffset := range rowsPerDay {
		unixDay := firstDay + int64(dayOffset)
		dayStart := unixDay * secondsPerUTCDate
		fromUnix := max(firstUnix, dayStart)
		toUnix := min(lastUnix, dayStart+secondsPerUTCDate-1)
		fromFrame := coreTypes.FrameOfDay(fromUnix)
		toFrame := coreTypes.FrameOfDay(toUnix)
		fromKey, _ := coreTypes.FrameRange(fromFrame)
		_, toKey := coreTypes.FrameRange(toFrame)

		queries.Go(func() error {
			return makeObservabilityLogQuery(
				&rowsPerDay[dayOffset], int16(unixDay), fromKey, toKey,
			).Exec()
		})
	}
	if err := queries.Wait(); err != nil {
		return nil, err
	}

	rows := []coreTypes.UserLog{}
	for _, dayRows := range rowsPerDay {
		rows = append(rows, dayRows...)
	}
	return rows, nil
}

func makeObservabilityLogQuery(
	rows *[]coreTypes.UserLog,
	unixDay int16,
	fromKey, toKey int64,
) *coreTypes.UserLogTable {
	query := db.Query(rows)
	// Select only columns projected by the frame view; selecting the whole base record would make
	// the ORM fall back to user_logs and Scylla would require ALLOW FILTERING.
	query.Select(
		query.Date,
		query.RequestID,
		query.FrameRouteCompanyAgg,
		query.RouteID,
		query.ErrorCount,
		query.ErrorIDs,
	).Date.Equals(unixDay).FrameRouteCompanyAgg.Between(fromKey, toKey)
	return query
}

func makeObservabilityFrames(
	firstTimeFrame, lastTimeFrame int32,
	creditRows []coreTypes.CreditUsageCompany,
	logRows []coreTypes.UserLog,
) ([]ObservabilityFrame, error) {
	if firstTimeFrame > lastTimeFrame {
		return nil, fmt.Errorf("invalid frame range %d..%d", firstTimeFrame, lastTimeFrame)
	}

	frames := make([]ObservabilityFrame, lastTimeFrame-firstTimeFrame+1)
	routesByFrame := make([]map[int16]*observabilityRouteAccumulator, len(frames))
	for frameOffset := range frames {
		timeFrame := firstTimeFrame + int32(frameOffset)
		frameID := core.UnixToSunix(observabilityTimeFrameToUnix(timeFrame))
		frames[frameOffset] = ObservabilityFrame{
			ID: frameID, TimeFrame: timeFrame, Details: []ObservabilityRouteDetail{},
			Updated: frameID, Status: 1,
		}
		routesByFrame[frameOffset] = map[int16]*observabilityRouteAccumulator{}
	}

	for _, row := range creditRows {
		frameOffset := row.TimeFrame - firstTimeFrame
		if frameOffset < 0 || frameOffset >= int32(len(frames)) {
			return nil, fmt.Errorf("credit time frame %d is outside the requested range", row.TimeFrame)
		}
		routeTotals := map[int16]creditUsageTotals{}
		if _, err := decodeCreditUsage(row.UsedCredits, routeTotals); err != nil {
			return nil, fmt.Errorf("invalid credit blob in time frame %d: %w", row.TimeFrame, err)
		}
		for routeID, totals := range routeTotals {
			detail := getObservabilityRouteAccumulator(routesByFrame[frameOffset], routeID)
			detail.CPU = totals.CPU
			detail.Inference = totals.Inference
			detail.EstimatedRequests = estimateObservabilityRequests(core.APIRouteNames[routeID], totals.CPU)
		}
	}

	for _, row := range logRows {
		if row.ErrorCount <= 0 {
			continue
		}
		requestUnix, err := decodeObservabilityRequestUnix(row.RequestID, row.Date)
		if err != nil {
			return nil, err
		}
		timeFrame := unixToObservabilityTimeFrame(requestUnix)
		frameOffset := timeFrame - firstTimeFrame
		if frameOffset < 0 || frameOffset >= int32(len(frames)) {
			continue
		}
		detail := getObservabilityRouteAccumulator(routesByFrame[frameOffset], row.RouteID)
		detail.FailedRequests++
		detail.ErrorOccurrences += uint64(row.ErrorCount)
		for _, errorID := range row.ErrorIDs {
			if errorID > 0 {
				detail.errorCounts[errorID]++
			}
		}
	}

	for frameOffset, routes := range routesByFrame {
		routeIDs := make([]int16, 0, len(routes))
		for routeID := range routes {
			routeIDs = append(routeIDs, routeID)
		}
		slices.Sort(routeIDs)
		for _, routeID := range routeIDs {
			accumulator := routes[routeID]
			errorIDs := make([]int32, 0, len(accumulator.errorCounts))
			for errorID := range accumulator.errorCounts {
				errorIDs = append(errorIDs, errorID)
			}
			slices.Sort(errorIDs)
			for _, errorID := range errorIDs {
				accumulator.ErrorIDs = append(accumulator.ErrorIDs, errorID)
				accumulator.ErrorIDCounts = append(accumulator.ErrorIDCounts, accumulator.errorCounts[errorID])
			}
			frames[frameOffset].Details = append(frames[frameOffset].Details, accumulator.ObservabilityRouteDetail)
		}
	}
	return frames, nil
}

func getObservabilityRouteAccumulator(
	routes map[int16]*observabilityRouteAccumulator,
	routeID int16,
) *observabilityRouteAccumulator {
	if detail := routes[routeID]; detail != nil {
		return detail
	}
	detail := &observabilityRouteAccumulator{
		ObservabilityRouteDetail: ObservabilityRouteDetail{RouteID: routeID},
		errorCounts:              map[int32]uint64{},
	}
	routes[routeID] = detail
	return detail
}

func estimateObservabilityRequests(route string, cpuCredits uint64) uint64 {
	switch {
	case strings.HasPrefix(route, "GET."):
		return cpuCredits / 2
	case strings.HasPrefix(route, "POST."):
		return cpuCredits / 5
	default:
		return 0
	}
}

func decodeObservabilityRequestUnix(requestID int64, expectedDay int16) (int64, error) {
	if requestID <= 0 {
		return 0, fmt.Errorf("request ID %d is not positive", requestID)
	}

	var unixSeconds int64
	if requestID >= 100_000_000_000_000_000 {
		sunixMilliseconds := requestID / 1_000_000
		unixSeconds = (sunixMilliseconds*2 + 1_000_000_000_000) / 1_000
	} else {
		sunixTime := requestID / 10_000_000
		unixSeconds = sunixTime*2 + 1_000_000_000
	}
	if unixSeconds < 0 || unixSeconds/secondsPerUTCDate != int64(expectedDay) {
		return 0, fmt.Errorf("request ID %d does not belong to Unix day %d", requestID, expectedDay)
	}
	return unixSeconds, nil
}

func unixToObservabilityTimeFrame(unixSeconds int64) int32 {
	return observabilityFiveMinutePrefix + int32(unixSeconds/observabilityFiveMinuteSeconds)
}

func observabilityTimeFrameToUnix(timeFrame int32) int64 {
	return int64(timeFrame-observabilityFiveMinutePrefix) * observabilityFiveMinuteSeconds
}

func makeObservabilityRoutes() []ObservabilityRoute {
	// Route zero means an unmatched API. It has no valid delta-cache identity, so cards use their
	// ROUTE.0 fallback instead of sending an ID=0 metadata record that IndexedDB rejects.
	routes := make([]ObservabilityRoute, 0, len(core.APIRouteNames))
	for routeID, route := range core.APIRouteNames {
		routes = append(routes, ObservabilityRoute{
			ID: routeID, Route: route, Updated: observabilityRoutesVersion, Status: 1,
		})
	}
	slices.SortFunc(routes, func(left, right ObservabilityRoute) int {
		return cmp.Compare(left.ID, right.ID)
	})
	return routes
}
