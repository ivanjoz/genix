package config

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"cmp"
	"fmt"
	"maps"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	companyCreditUsageDays = int32(30)
	// Bumping this re-sends the route catalog to every client. The names come from the generated
	// route table, so they only change when the backend is rebuilt with new routes.
	companyCreditRoutesVersion = int32(1)
	// Company IDs at or below zero are not tenants: zero is the platform-wide observability
	// aggregate written by the same daemon into the same table.
	companyCreditFirstTenantID = int32(1)
	// A UnixDay stays under 100_000 for another two centuries, so five decimal digits is enough to
	// carry it as the low half of a composite id. The high half holds a company or a user, which
	// caps either at 21_474 before the packed value leaves int32 — the width the client's compact
	// id encoder can carry.
	companyCreditDayIDFactor = int32(100_000)
	// Same packing for the user-label endpoint, where the low half is a user id instead of a day.
	companyUserLabelIDFactor = int32(100_000)
)

// companyCreditDay is one company's usage on one day. TimeFrame doubles as the delta watermark:
// the daily row is rewritten in place all day under a fixed key, so the client resends its highest
// frame and the handler answers with `>=` — today refreshed, plus any day that has since started.
type companyCreditDay struct {
	ID        int32
	CompanyID int32
	Day       int16
	CPU       uint64
	Inference uint64
	Updated   int32 `json:"upd"`
	Status    int8  `json:"ss"`
}

// creditRouteName is the id -> name catalog the cards need to label a route. It is an independent
// delta collection carrying its own version, exactly as observability's Routes does, so a client
// that already holds it never receives it again.
type creditRouteName struct {
	ID      int16
	Route   string
	Updated int32 `json:"upd"`
	Status  int8  `json:"ss"`
}

// companyCreditBudgetMeter is one company's entitlement and what is left of it, in both windows the
// limiter refuses a charge on. Remaining rather than used, because that is the question the card
// asks; the ceilings travel too, since a bar needs the total its fill is a fraction of.
//
// Sent on every response regardless of the client's watermark: the figures move with every charge,
// so a delta that withheld them would leave a stale meter on screen. It is one short row per
// company, which is the same trade `Days` already makes for today's row.
type companyCreditBudgetMeter struct {
	ID                      int32
	DailyCPU                int64
	DailyInference          int64
	DailyRemainingCPU       uint64
	DailyRemainingInference uint64
	MonthlyCPUCeiling       int64
	MonthlyInferenceCeiling int64
	RemainingCPU            uint64
	RemainingInference      uint64
	// El pool diario de lecturas y lo que queda de él. Una company con estas dos cifras separadas de
	// las de arriba es una que puede estar sirviendo GET con la cuota agotada, que es lo único que
	// explica tráfico después de un rechazo.
	ExtraCPU          int64
	DayExtraCPUUsed   uint64
	ExtraRemainingCPU uint64
	IsCurrentMonth    bool
	Updated           int32 `json:"upd"`
	Status            int8  `json:"ss"`
}

type companyCreditReport struct {
	Days    []companyCreditDay
	Budgets []companyCreditBudgetMeter
	Routes  []creditRouteName `json:",omitempty"`
}

// companyCreditCompanyDay is the company's own total for one day, with the per-route split the API
// breakdown reads. Only this table carries routes: the same breakdown per user would multiply the
// payload by the headcount for a view nothing renders.
type companyCreditCompanyDay struct {
	ID        int32
	Day       int16
	CPU       uint64
	Inference uint64
	Routes    []creditUsageRoute `json:",omitempty"`
	Updated   int32              `json:"upd"`
	Status    int8               `json:"ss"`
}

// companyCreditUserDay is one user's total for one day. Totals only, no route split.
type companyCreditUserDay struct {
	ID        int32
	UserID    int32
	Day       int16
	CPU       uint64
	Inference uint64
	Updated   int32 `json:"upd"`
	Status    int8  `json:"ss"`
}

// Company and Users are independent delta collections, each with its own watermark, so the client
// can hold one without implying anything about the other.
type companyCreditUsage struct {
	Company []companyCreditCompanyDay
	Users   []companyCreditUserDay
	Routes  []creditRouteName `json:",omitempty"`
}

// GetCompanyCreditUsageReport streams the platform's daily company aggregates. It ranks nothing and
// names nothing: the client already holds the company catalog, and every total, zero-filled series
// and ordering it wants is derivable from these rows.
func GetCompanyCreditUsageReport(req *core.HandlerArgs) core.HandlerResponse {
	queryStartedAt := time.Now()
	lastFrame := currentDailyTimeFrame()
	firstFrame := lastFrame - companyCreditUsageDays + 1
	// The watermark is itself a time frame, so it narrows the read directly. `>=` rather than `>`
	// because the client's highest frame is today's, and today's row is still being written.
	if watermark := creditUsageWatermark(req, "Days"); watermark > firstFrame {
		firstFrame = min(watermark, lastFrame)
	}

	// The view is partitioned by frame, and a partition key takes IN rather than a range.
	frames := make([]int32, 0, lastFrame-firstFrame+1)
	for frame := firstFrame; frame <= lastFrame; frame++ {
		frames = append(frames, frame)
	}
	rows := []coreTypes.CreditUsageCompany{}
	query := db.Query(&rows)
	// One partition per frame on the day-partitioned view, and that partition holds nothing but
	// company rows: the cost of this report is the number of days requested, not the number of
	// tenants, and not the platform's total user count either.
	query.TimeFrame.In(frames...)
	if err := query.Exec(); err != nil {
		core.Log("company credit report failed::", " first_frame::", firstFrame,
			" last_frame::", lastFrame, " err::", err)
		return req.MakeErr("No se pudo obtener el uso de créditos por empresa.", err)
	}

	report := companyCreditReport{Days: make([]companyCreditDay, 0, len(rows))}
	for _, row := range rows {
		// Company id zero is the platform-wide observability aggregate, written into this table by
		// the same daemon. It is not a tenant and must never reach the company ranking.
		if row.CompanyID < companyCreditFirstTenantID {
			continue
		}
		totals, err := decodeCreditUsage(row.UsedCredits, nil)
		if err != nil {
			core.Log("company credit report blob invalid::", " company::", row.CompanyID,
				" frame::", row.TimeFrame, " err::", err)
			return req.MakeErr("No se pudo interpretar el uso de créditos por empresa.", err)
		}
		day := int16(row.TimeFrame - dailyTimeFramePrefix)
		report.Days = append(report.Days, companyCreditDay{
			ID:        row.CompanyID*companyCreditDayIDFactor + int32(day),
			CompanyID: row.CompanyID, Day: day, CPU: totals.CPU, Inference: totals.Inference,
			Updated: row.TimeFrame, Status: 1,
		})
	}
	budgets, budgetError := getCompanyCreditBudgets()
	if budgetError != nil {
		core.Log("company credit budgets read failed::", " err::", budgetError)
		return req.MakeErr("No se pudo obtener el presupuesto de créditos de las empresas.", budgetError)
	}
	report.Budgets = make([]companyCreditBudgetMeter, 0, len(budgets))
	for _, budget := range budgets {
		report.Budgets = append(report.Budgets, companyCreditBudgetMeter{
			ID:                      budget.CompanyID,
			DailyCPU:                budget.DailyCPU,
			DailyInference:          budget.DailyInference,
			DailyRemainingCPU:       budget.DailyRemainingCPU,
			DailyRemainingInference: budget.DailyRemainingInference,
			MonthlyCPUCeiling:       budget.MonthlyCPUCeiling,
			MonthlyInferenceCeiling: budget.MonthlyInferenceCeiling,
			RemainingCPU:            budget.CurrentCPU,
			RemainingInference:      budget.CurrentInference,
			ExtraCPU:                budget.ExtraCPU,
			DayExtraCPUUsed:         budget.DayExtraCPUUsed,
			ExtraRemainingCPU:       budget.ExtraRemainingCPU,
			IsCurrentMonth:          budget.IsCurrentMonth,
			Updated:                 lastFrame,
			Status:                  1,
		})
	}
	if req.GetQueryInt("Routes") < companyCreditRoutesVersion {
		report.Routes = makeCreditRouteNames()
	}
	core.Log("company credit report completed::", " first_frame::", firstFrame, " last_frame::", lastFrame,
		" rows::", len(rows), " days::", len(report.Days), " budgets::", len(report.Budgets),
		" elapsed::", time.Since(queryStartedAt))
	return req.MakeResponse(report)
}

// GetCompanyCreditUsage returns one company's thirty days for every user in a single read. It
// replaces the per-day detail and per-user report handlers: both were views of these same rows.
func GetCompanyCreditUsage(req *core.HandlerArgs) core.HandlerResponse {
	queryStartedAt := time.Now()
	// The transport reserves company-id for the authenticated tenant; this is the report target.
	companyID := req.GetQueryInt("target-company-id")
	if companyID < companyCreditFirstTenantID {
		return req.MakeErr("Debe enviar una empresa válida.")
	}
	lastFrame := currentDailyTimeFrame()
	firstFrame := lastFrame - companyCreditUsageDays + 1
	if watermark := creditUsageWatermark(req, "Days"); watermark > firstFrame {
		firstFrame = min(watermark, lastFrame)
	}

	// Two reads, both native to their table and run together: the company series is the base key of
	// credit_usage_company, and the user series is one clustering range on credit_usage_user. The
	// company does not need validating first: an id that does not exist simply has no rows.
	companyRows := []coreTypes.CreditUsageCompany{}
	userRows := []coreTypes.CreditUsageUser{}
	queries := errgroup.Group{}
	queries.Go(func() error {
		query := db.Query(&companyRows)
		return query.CompanyID.Equals(companyID).TimeFrame.Between(firstFrame, lastFrame).Exec()
	})
	queries.Go(func() error {
		query := db.Query(&userRows)
		return query.CompanyID.Equals(companyID).TimeFrame.Between(firstFrame, lastFrame).Exec()
	})
	if err := queries.Wait(); err != nil {
		core.Log("company credit usage failed::", " company::", companyID,
			" first_frame::", firstFrame, " last_frame::", lastFrame, " err::", err)
		return req.MakeErr("No se pudo obtener el uso de créditos de la empresa.", err)
	}

	usage := companyCreditUsage{
		Company: make([]companyCreditCompanyDay, 0, len(companyRows)),
		Users:   make([]companyCreditUserDay, 0, len(userRows)),
	}
	for _, row := range companyRows {
		// Only this table carries the per-route split, which is what the API breakdown reads.
		routeTotals := map[int16]creditUsageTotals{}
		totals, err := decodeCreditUsage(row.UsedCredits, routeTotals)
		if err != nil {
			core.Log("company credit usage company blob invalid::", " company::", companyID,
				" frame::", row.TimeFrame, " err::", err)
			return req.MakeErr("No se pudo interpretar el uso de créditos de la empresa.", err)
		}
		// The day alone identifies a row here, and a UnixDay is always positive.
		day := int16(row.TimeFrame - dailyTimeFramePrefix)
		usage.Company = append(usage.Company, companyCreditCompanyDay{
			ID: int32(day), Day: day, CPU: totals.CPU, Inference: totals.Inference,
			Routes: makeCreditUsageRoutes(routeTotals), Updated: row.TimeFrame, Status: 1,
		})
	}
	for _, row := range userRows {
		totals, err := decodeCreditUsage(row.UsedCredits, nil)
		if err != nil {
			core.Log("company credit usage user blob invalid::", " company::", companyID,
				" user::", row.UserID, " frame::", row.TimeFrame, " err::", err)
			return req.MakeErr("No se pudo interpretar el uso de créditos por usuario.", err)
		}
		day := int16(row.TimeFrame - dailyTimeFramePrefix)
		usage.Users = append(usage.Users, companyCreditUserDay{
			ID:     row.UserID*companyCreditDayIDFactor + int32(day),
			UserID: row.UserID, Day: day, CPU: totals.CPU, Inference: totals.Inference,
			Updated: row.TimeFrame, Status: 1,
		})
	}
	if req.GetQueryInt("Routes") < companyCreditRoutesVersion {
		usage.Routes = makeCreditRouteNames()
	}

	core.Log("company credit usage completed::", " company::", companyID, " first_frame::", firstFrame,
		" last_frame::", lastFrame, " company_days::", len(usage.Company), " user_days::", len(usage.Users),
		" elapsed::", time.Since(queryStartedAt))
	return req.MakeResponse(usage)
}

// GetCompanyUsersByIDs resolves user display labels across companies for the operator panel.
//
// The versioned by-IDs path cannot serve this: mainHandler strips a client-sent "cmp" on private
// routes, so its partition always resolves to the caller's own company. Packing the company into
// the requested id is what lets one route answer for every tenant, and the ids stay inside int32 so
// the client's compact id encoder can carry them.
func GetCompanyUsersByIDs(req *core.HandlerArgs) core.HandlerResponse {
	packedIDs := req.ExtractIDs()
	if len(packedIDs) == 0 {
		return req.MakeErr("No se enviaron ids de usuarios.")
	}

	// Split the packed ids back into one user list per company, which is what turns an arbitrary
	// client batch into one bounded query per tenant it actually asked about.
	userIDsByCompany := map[int32][]int32{}
	for _, packedID := range packedIDs {
		companyID := int32(packedID) / companyUserLabelIDFactor
		userID := int32(packedID) % companyUserLabelIDFactor
		if companyID < companyCreditFirstTenantID || userID <= 0 {
			continue
		}
		userIDsByCompany[companyID] = append(userIDsByCompany[companyID], userID)
	}
	if len(userIDsByCompany) == 0 {
		return req.MakeErr("No se enviaron ids de usuarios válidos.")
	}

	companyIDs := slices.Sorted(maps.Keys(userIDsByCompany))
	labelsByCompany := make([][]companyUserLabel, len(companyIDs))
	queries := errgroup.Group{}
	queries.SetLimit(8)
	for companyIndex, companyID := range companyIDs {
		queries.Go(func() error {
			users := []coreTypes.User{}
			query := db.Query(&users)
			// Only the label columns: this endpoint must never widen into a user record.
			query.Select(query.ID, query.User, query.FirstName, query.LastName).
				CompanyID.Equals(companyID).ID.In(userIDsByCompany[companyID]...)
			if err := query.Exec(); err != nil {
				return fmt.Errorf("company %d user labels: %w", companyID, err)
			}
			labels := make([]companyUserLabel, 0, len(users))
			for _, user := range users {
				labels = append(labels, companyUserLabel{
					ID:   companyID*companyUserLabelIDFactor + user.ID,
					User: user.User, FirstName: user.FirstName, LastName: user.LastName,
				})
			}
			labelsByCompany[companyIndex] = labels
			return nil
		})
	}
	if err := queries.Wait(); err != nil {
		core.Log("company user labels failed::", " companies::", len(companyIDs), " err::", err)
		return req.MakeErr("No se pudieron obtener los usuarios solicitados.", err)
	}

	labels := []companyUserLabel{}
	for _, companyLabels := range labelsByCompany {
		labels = append(labels, companyLabels...)
	}
	core.Log("company user labels completed::", " companies::", len(companyIDs), " labels::", len(labels))
	return req.MakeResponse(labels)
}

// companyUserLabel is what a card renders and nothing more. ID is the packed (company, user) pair
// the client asked for, so it can key its cache by the same value it sent.
type companyUserLabel struct {
	ID        int32
	User      string
	FirstName string
	LastName  string
}

// creditUsageWatermark reads the delta bound, accepting either the collection-named parameter the
// multi-collection cache sends or the plain one a single-collection client would.
func creditUsageWatermark(req *core.HandlerArgs, collection string) int32 {
	return core.Coalesce(req.GetQueryInt(collection), req.GetQueryInt("upd"))
}

func makeCreditRouteNames() []creditRouteName {
	// Route zero means an unmatched API. It has no valid delta-cache identity, so the client falls
	// back to its own label rather than receiving an ID=0 record that IndexedDB rejects.
	routes := make([]creditRouteName, 0, len(core.APIRouteNames))
	for routeID, route := range core.APIRouteNames {
		if routeID <= 0 {
			continue
		}
		routes = append(routes, creditRouteName{
			ID: routeID, Route: route, Updated: companyCreditRoutesVersion, Status: 1,
		})
	}
	slices.SortFunc(routes, func(left, right creditRouteName) int {
		return cmp.Compare(left.ID, right.ID)
	})
	return routes
}
