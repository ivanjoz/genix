package config

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"cmp"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	creditUsageDaysCount = int32(15)
	dailyTimeFramePrefix = int32(200_000_000)
	secondsPerUTCDate    = int64(86_400)
	companyAggregateID   = int32(-1)
)

type creditUsageTotals struct {
	CPU       uint64
	Inference uint64
}

type creditUsageDay struct {
	Day       int16
	CPU       uint64
	Inference uint64
}

// creditUsageRoute is what one API route cost over the whole queried range. Range-aggregated and
// not per day: fifteen days times forty-odd routes is a nested payload nothing would plot, while
// "which endpoints cost this user the most" is the question the breakdown exists to answer.
//
// Route carries the name because the numbers alone are unreadable, and the generated table is the
// only authority on them — including for retired routes, which is what keeps an old row legible.
type creditUsageRoute struct {
	RouteID   int16
	Route     string
	CPU       uint64
	Inference uint64
}

type creditUsageScope struct {
	CPUDailyLimit       uint64
	InferenceDailyLimit uint64
	Days                []creditUsageDay
	Routes              []creditUsageRoute
}

type creditUsageResponse struct {
	User    creditUsageScope
	Company creditUsageScope
}

func GetCreditUsage(req *core.HandlerArgs) core.HandlerResponse {
	currentUnixDay := int32(time.Now().UTC().Unix() / secondsPerUTCDate)
	firstUnixDay := currentUnixDay - creditUsageDaysCount + 1
	firstTimeFrame := dailyTimeFramePrefix + firstUnixDay
	lastTimeFrame := dailyTimeFramePrefix + currentUnixDay
	userRows, companyRows := []coreTypes.CreditUsage{}, []coreTypes.CreditUsage{}
	budget := coreTypes.CompanyCreditBudget{CompanyID: req.User.CompanyID}

	core.Log("credit usage query started::", " company::", req.User.CompanyID,
		" user::", req.User.ID, " first_day::", firstUnixDay, " last_day::", currentUnixDay)
	queries := errgroup.Group{}
	queries.Go(func() error {
		query := db.Query(&userRows)
		query.CompanyID.Equals(req.User.CompanyID).UserID.Equals(req.User.ID).TimeFrame.Between(firstTimeFrame, lastTimeFrame)
		return query.Exec()
	})
	queries.Go(func() error {
		var budgetError error
		budget, budgetError = getCompanyCreditBudgetRecord(req.User.CompanyID)
		return budgetError
	})
	queries.Go(func() error {
		query := db.Query(&companyRows)
		query.CompanyID.Equals(req.User.CompanyID).UserID.Equals(companyAggregateID).TimeFrame.Between(firstTimeFrame, lastTimeFrame)
		return query.Exec()
	})
	if err := queries.Wait(); err != nil {
		core.Log("credit usage query failed::", " company::", req.User.CompanyID, " user::", req.User.ID, " err::", err)
		return req.MakeErr("No se pudo obtener el uso de créditos.", err)
	}

	companyCPUDailyLimit := nonNegativeBudget(budget.DailyCPU)
	companyInferenceDailyLimit := nonNegativeBudget(budget.DailyInference)
	userUsage, err := makeCreditUsageScope(userRows, firstUnixDay,
		companyCPUDailyLimit/2, companyInferenceDailyLimit/2)
	if err != nil {
		core.Log("credit usage user blob invalid::", " company::", req.User.CompanyID, " user::", req.User.ID, " err::", err)
		return req.MakeErr("No se pudo interpretar el uso de créditos del usuario.", err)
	}
	companyUsage, err := makeCreditUsageScope(companyRows, firstUnixDay,
		companyCPUDailyLimit, companyInferenceDailyLimit)
	if err != nil {
		core.Log("credit usage company blob invalid::", " company::", req.User.CompanyID, " err::", err)
		return req.MakeErr("No se pudo interpretar el uso de créditos de la empresa.", err)
	}

	core.Log("credit usage query completed::", " company::", req.User.CompanyID, " user::", req.User.ID,
		" user_rows::", len(userRows), " company_rows::", len(companyRows))
	return req.MakeResponse(creditUsageResponse{User: userUsage, Company: companyUsage})
}

func makeCreditUsageScope(rows []coreTypes.CreditUsage, firstUnixDay int32, cpuLimit, inferenceLimit uint64) (creditUsageScope, error) {
	// Zero-fill the fixed UTC range so chart columns never shift when a day has no usage.
	days := make([]creditUsageDay, creditUsageDaysCount)
	for dayOffset := range creditUsageDaysCount {
		days[dayOffset].Day = int16(firstUnixDay + dayOffset)
	}
	// One map across every row: the breakdown spans the range, and a per-row map would only be
	// merged back into this one anyway.
	routeTotals := map[int16]creditUsageTotals{}
	for _, row := range rows {
		dayOffset := row.TimeFrame - dailyTimeFramePrefix - firstUnixDay
		if dayOffset < 0 || dayOffset >= creditUsageDaysCount {
			return creditUsageScope{}, fmt.Errorf("daily time frame %d is outside the requested range", row.TimeFrame)
		}
		totals, err := decodeCreditUsage(row.UsedCredits, routeTotals)
		if err != nil {
			return creditUsageScope{}, fmt.Errorf("invalid credit blob in time frame %d: %w", row.TimeFrame, err)
		}
		days[dayOffset].CPU = totals.CPU
		days[dayOffset].Inference = totals.Inference
	}
	return creditUsageScope{
		CPUDailyLimit:       cpuLimit,
		InferenceDailyLimit: inferenceLimit,
		Days:                days,
		Routes:              makeCreditUsageRoutes(routeTotals),
	}, nil
}

func nonNegativeBudget(value int64) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value)
}

// makeCreditUsageRoutes orders the breakdown by what it costs, most expensive first, so a client
// can render the head of the list and stop. The route number breaks ties only to keep the order
// stable — map iteration is not, and a list that reshuffles between two identical requests looks
// like data changing when nothing did.
func makeCreditUsageRoutes(routeTotals map[int16]creditUsageTotals) []creditUsageRoute {
	routes := make([]creditUsageRoute, 0, len(routeTotals))
	for routeID, totals := range routeTotals {
		routes = append(routes, creditUsageRoute{
			RouteID:   routeID,
			Route:     core.APIRouteNames[routeID],
			CPU:       totals.CPU,
			Inference: totals.Inference,
		})
	}
	slices.SortFunc(routes, func(left, right creditUsageRoute) int {
		if left.CPU != right.CPU {
			return cmp.Compare(right.CPU, left.CPU)
		}
		if left.Inference != right.Inference {
			return cmp.Compare(right.Inference, left.Inference)
		}
		return cmp.Compare(left.RouteID, right.RouteID)
	})
	return routes
}

// decodeCreditUsage reads one row's blob, returning its totals and accumulating the per-route
// split into routeTotals, which may be nil when only the totals are wanted.
//
// This is the second implementation of the format; the writer is Rust
// (server_utils/src/limiter/credits_blob.rs). It is deliberately strict about the three rules that
// make the encoding canonical — ascending routes, no all-zero entry, narrowest width — because a
// blob that breaks one of them was not written by that encoder, and the alternative to refusing it
// is charting a number nobody produced.
func decodeCreditUsage(encoded []byte, routeTotals map[int16]creditUsageTotals) (creditUsageTotals, error) {
	totals := creditUsageTotals{}
	previousRouteID := -1
	for offset := 0; offset < len(encoded); {
		// The header is two bytes; a lone trailing byte would otherwise be read as a whole one and
		// invent a route out of the padding.
		if len(encoded)-offset < 2 {
			return creditUsageTotals{}, fmt.Errorf("credit blob ends inside an entry header")
		}
		header := binary.BigEndian.Uint16(encoded[offset : offset+2])
		offset += 2
		routeID := int(header >> 2)
		valueWidth := int(header&0b11) + 1
		if routeID <= previousRouteID {
			return creditUsageTotals{}, fmt.Errorf("routes are not strictly ascending")
		}
		if len(encoded)-offset < valueWidth*2 {
			return creditUsageTotals{}, fmt.Errorf("credit blob ends inside route %d", routeID)
		}

		cpuCredits := readBigEndianCredit(encoded[offset : offset+valueWidth])
		offset += valueWidth
		inferenceCredits := readBigEndianCredit(encoded[offset : offset+valueWidth])
		offset += valueWidth
		if cpuCredits == 0 && inferenceCredits == 0 {
			return creditUsageTotals{}, fmt.Errorf("route %d contains only zero values", routeID)
		}
		if smallestCreditWidth(max(cpuCredits, inferenceCredits)) != valueWidth {
			return creditUsageTotals{}, fmt.Errorf("route %d is not canonically encoded", routeID)
		}
		if math.MaxUint64-totals.CPU < cpuCredits || math.MaxUint64-totals.Inference < inferenceCredits {
			return creditUsageTotals{}, fmt.Errorf("summed credits overflow uint64")
		}
		totals.CPU += cpuCredits
		totals.Inference += inferenceCredits
		if routeTotals != nil {
			accumulated := routeTotals[int16(routeID)]
			if math.MaxUint64-accumulated.CPU < cpuCredits ||
				math.MaxUint64-accumulated.Inference < inferenceCredits {
				return creditUsageTotals{}, fmt.Errorf("route %d credits overflow uint64", routeID)
			}
			accumulated.CPU += cpuCredits
			accumulated.Inference += inferenceCredits
			routeTotals[int16(routeID)] = accumulated
		}
		previousRouteID = routeID
	}
	return totals, nil
}

func readBigEndianCredit(encoded []byte) uint64 {
	var value uint64
	for _, encodedByte := range encoded {
		value = value<<8 | uint64(encodedByte)
	}
	return value
}

func smallestCreditWidth(value uint64) int {
	switch {
	case value <= math.MaxUint8:
		return 1
	case value <= math.MaxUint16:
		return 2
	case value <= 0xFF_FFFF:
		return 3
	default:
		return 4
	}
}
