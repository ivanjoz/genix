package config

import (
	"app/cloud"
	"app/config/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	companyCreditUsageDays  = int32(30)
	companyCreditQueryLimit = 8
	companyCreditUnknownAPI = "API.UNKNOWN"
)

type companyCreditUsageSummary struct {
	CompanyID      int32
	Company        string
	Status         int8
	AdminName      string
	AdminUser      string
	CPU            uint64
	Inference      uint64
	TodayCPU       uint64
	TodayInference uint64
	ActiveDays     int16
	Days           []creditUsageDay
}

type companyCreditUsageReport struct {
	FirstDay    int16
	LastDay     int16
	GeneratedAt int32
	Companies   []companyCreditUsageSummary
}

type companyCreditUsageDetail struct {
	CompanyID int32
	Day       int16
	CPU       uint64
	Inference uint64
	Routes    []creditUsageRoute
}

type companyCreditUsageUser struct {
	UserID    int32
	Name      string
	User      string
	CPU       uint64
	Inference uint64
	Days      []creditUsageDay
}

type companyCreditUsageUsersReport struct {
	CompanyID int32
	FirstDay  int16
	LastDay   int16
	Users     []companyCreditUsageUser
}

// GetCompanyCreditUsageReport ranks all catalog companies from partition-local daily aggregates.
func GetCompanyCreditUsageReport(req *core.HandlerArgs) core.HandlerResponse {
	queryStartedAt := time.Now()
	lastDay := int32(core.FechaUnix())
	firstDay := lastDay - companyCreditUsageDays + 1
	companies, err := getCompaniesUpdatedSince(0)
	if err != nil {
		core.Log("company credit report catalog failed::", " err::", err)
		return req.MakeErr("No se pudieron obtener las empresas para el reporte de créditos.", err)
	}
	companies = normalizeCompanyCreditCatalog(companies)
	core.Log("company credit report started::", " companies::", len(companies),
		" first_day::", firstDay, " last_day::", lastDay)

	summaries := make([]companyCreditUsageSummary, len(companies))
	rowCounts := make([]int, len(companies))
	missingAdministrators := make([]bool, len(companies))
	queries := errgroup.Group{}
	queries.SetLimit(companyCreditQueryLimit)
	for companyIndex := range companies {
		companyIndex := companyIndex
		queries.Go(func() error {
			company := companies[companyIndex]
			administrator, administratorError := getCompanyCreditAdministrator(company.ID)
			if administratorError != nil {
				return fmt.Errorf("company %d administrator query: %w", company.ID, administratorError)
			}
			rows := []coreTypes.CreditUsage{}
			query := db.Query(&rows)
			query.CompanyID.Equals(company.ID).
				UserID.Equals(companyAggregateID).
				TimeFrame.Between(dailyTimeFramePrefix+firstDay, dailyTimeFramePrefix+lastDay)
			if queryError := query.Exec(); queryError != nil {
				return fmt.Errorf("company %d daily credit query: %w", company.ID, queryError)
			}
			summary, summaryError := makeCompanyCreditUsageSummary(company, rows, firstDay)
			if summaryError != nil {
				return fmt.Errorf("company %d credit aggregation: %w", company.ID, summaryError)
			}
			summary.AdminName, summary.AdminUser = makeCompanyCreditAdministratorIdentity(administrator)
			missingAdministrators[companyIndex] = administrator == nil
			summaries[companyIndex] = summary
			rowCounts[companyIndex] = len(rows)
			return nil
		})
	}
	if err = queries.Wait(); err != nil {
		core.Log("company credit report failed::", " companies::", len(companies), " err::", err)
		return req.MakeErr("No se pudo obtener el uso de créditos por empresa.", err)
	}

	rowsRead := 0
	missingAdministratorCount := 0
	for _, rowCount := range rowCounts {
		rowsRead += rowCount
	}
	for _, isMissing := range missingAdministrators {
		if isMissing {
			missingAdministratorCount++
		}
	}
	sortCompanyCreditUsageSummaries(summaries)
	core.Log("company credit report completed::", " companies::", len(summaries),
		" rows::", rowsRead, " missing_administrators::", missingAdministratorCount,
		" elapsed::", time.Since(queryStartedAt))
	return req.MakeResponse(companyCreditUsageReport{
		FirstDay: int16(firstDay), LastDay: int16(lastDay), GeneratedAt: core.SUnixTime(), Companies: summaries,
	})
}

// getCompanyCreditAdministrator resolves the canonical company administrator by its fixed user ID.
func getCompanyCreditAdministrator(companyID int32) (*coreTypes.User, error) {
	const administratorUserID = int32(1)
	if cloud.IsDataMirrorEnabled() {
		return cloud.GetByID(coreTypes.User{CompanyID: companyID, ID: administratorUserID})
	}

	administrators := []coreTypes.User{}
	administratorQuery := db.Query(&administrators)
	administratorQuery.CompanyID.Equals(companyID).ID.Equals(administratorUserID).Limit(1)
	if err := administratorQuery.Exec(); err != nil {
		return nil, err
	}
	if len(administrators) == 0 {
		return nil, nil
	}
	return &administrators[0], nil
}

// makeCompanyCreditAdministratorIdentity exposes only the display fields required by company cards.
func makeCompanyCreditAdministratorIdentity(administrator *coreTypes.User) (string, string) {
	if administrator == nil {
		return "", ""
	}
	administratorName := strings.TrimSpace(administrator.FirstName + " " + administrator.LastName)
	if administratorName == "" {
		administratorName = administrator.User
	}
	return administratorName, administrator.User
}

// GetCompanyCreditUsageDetail decodes one company's absolute daily row into API totals.
func GetCompanyCreditUsageDetail(req *core.HandlerArgs) core.HandlerResponse {
	// The transport reserves company-id for the authenticated tenant; this is the report target.
	companyID := req.GetQueryInt("target-company-id")
	requestedDay := req.GetQueryInt("day")
	lastDay := int32(core.FechaUnix())
	firstDay := lastDay - companyCreditUsageDays + 1
	if companyID <= 0 {
		return req.MakeErr("Debe enviar una empresa válida.")
	}
	if err := validateCompanyCreditUsageDay(requestedDay, firstDay, lastDay); err != nil {
		return req.MakeErr("El día solicitado no pertenece al reporte de los últimos 30 días.", err)
	}
	company, err := getCompanyByID(companyID)
	if err != nil {
		core.Log("company credit detail company lookup failed::", " company::", companyID, " err::", err)
		return req.MakeErr("No se pudo validar la empresa solicitada.", err)
	}
	if company == nil {
		return req.MakeErr("No se encontró la empresa solicitada.")
	}

	core.Log("company credit detail started::", " company::", companyID, " day::", requestedDay)
	rows := []coreTypes.CreditUsage{}
	query := db.Query(&rows)
	query.CompanyID.Equals(companyID).
		UserID.Equals(companyAggregateID).
		TimeFrame.Equals(dailyTimeFramePrefix + requestedDay).
		Limit(1)
	if err = query.Exec(); err != nil {
		core.Log("company credit detail query failed::", " company::", companyID,
			" day::", requestedDay, " err::", err)
		return req.MakeErr("No se pudo obtener el detalle diario de créditos.", err)
	}
	detail, err := makeCompanyCreditUsageDetail(companyID, requestedDay, rows)
	if err != nil {
		core.Log("company credit detail decode failed::", " company::", companyID,
			" day::", requestedDay, " err::", err)
		return req.MakeErr("No se pudo interpretar el detalle diario de créditos.", err)
	}
	core.Log("company credit detail completed::", " company::", companyID,
		" day::", requestedDay, " routes::", len(detail.Routes))
	return req.MakeResponse(detail)
}

// GetCompanyCreditUsageUsers builds the 30-day usage series only when the SaaS operator opens the users tab.
func GetCompanyCreditUsageUsers(req *core.HandlerArgs) core.HandlerResponse {
	queryStartedAt := time.Now()
	companyID := req.GetQueryInt("target-company-id")
	if companyID <= 0 {
		return req.MakeErr("Debe enviar una empresa válida.")
	}
	company, err := getCompanyByID(companyID)
	if err != nil {
		core.Log("company credit users company lookup failed::", " company::", companyID, " err::", err)
		return req.MakeErr("No se pudo validar la empresa solicitada.", err)
	}
	if company == nil {
		return req.MakeErr("No se encontró la empresa solicitada.")
	}

	users, err := getCompanyCreditUsers(companyID)
	if err != nil {
		core.Log("company credit users catalog failed::", " company::", companyID, " err::", err)
		return req.MakeErr("No se pudieron obtener los usuarios de la empresa.", err)
	}
	lastDay := int32(core.FechaUnix())
	firstDay := lastDay - companyCreditUsageDays + 1
	userUsage := make([]companyCreditUsageUser, len(users))
	rowCounts := make([]int, len(users))
	queries := errgroup.Group{}
	queries.SetLimit(companyCreditQueryLimit)
	for userIndex := range users {
		userIndex := userIndex
		queries.Go(func() error {
			user := users[userIndex]
			rows := []coreTypes.CreditUsage{}
			query := db.Query(&rows)
			query.CompanyID.Equals(companyID).UserID.Equals(user.ID).
				TimeFrame.Between(dailyTimeFramePrefix+firstDay, dailyTimeFramePrefix+lastDay)
			if queryError := query.Exec(); queryError != nil {
				return fmt.Errorf("user %d daily credit query: %w", user.ID, queryError)
			}
			usage, usageError := makeCompanyCreditUsageUser(user, rows, firstDay)
			if usageError != nil {
				return fmt.Errorf("user %d credit aggregation: %w", user.ID, usageError)
			}
			userUsage[userIndex] = usage
			rowCounts[userIndex] = len(rows)
			return nil
		})
	}
	if err = queries.Wait(); err != nil {
		core.Log("company credit users failed::", " company::", companyID, " users::", len(users), " err::", err)
		return req.MakeErr("No se pudo obtener el uso de créditos por usuario.", err)
	}
	sortCompanyCreditUsageUsers(userUsage)
	rowsRead := 0
	for _, rowCount := range rowCounts {
		rowsRead += rowCount
	}
	core.Log("company credit users completed::", " company::", companyID, " users::", len(userUsage),
		" rows::", rowsRead, " elapsed::", time.Since(queryStartedAt))
	return req.MakeResponse(companyCreditUsageUsersReport{
		CompanyID: companyID, FirstDay: int16(firstDay), LastDay: int16(lastDay), Users: userUsage,
	})
}

// getCompanyCreditUsers returns user identities without exposing private account fields. Legacy
// administrators may have status zero, so status must not decide whether historical usage exists.
func getCompanyCreditUsers(companyID int32) ([]coreTypes.User, error) {
	users := []coreTypes.User{}
	if cloud.IsDataMirrorEnabled() {
		userGroups := make([][]coreTypes.User, 3)
		queries := errgroup.Group{}
		for status := range userGroups {
			status := status
			queries.Go(func() error {
				return cloud.Select(&userGroups[status]).Where("company_id").Equals(companyID).
					Where("status").Equals(status).Where("updated").GreaterEqual(0).Exec()
			})
		}
		if err := queries.Wait(); err != nil {
			return nil, err
		}
		for _, userGroup := range userGroups {
			users = append(users, userGroup...)
		}
		return users, nil
	}
	query := db.Query(&users)
	query.Select(query.ID, query.User, query.FirstName, query.LastName).
		CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, err
	}
	return users, nil
}

// normalizeCompanyCreditCatalog removes reserved IDs and stabilizes duplicate catalog records.
func normalizeCompanyCreditCatalog(companies []types.Company) []types.Company {
	companiesByID := map[int32]types.Company{}
	for _, company := range companies {
		if company.ID <= 0 {
			continue
		}
		current, exists := companiesByID[company.ID]
		if !exists || company.Updated >= current.Updated {
			companiesByID[company.ID] = company
		}
	}
	normalized := make([]types.Company, 0, len(companiesByID))
	for _, company := range companiesByID {
		normalized = append(normalized, company)
	}
	slices.SortFunc(normalized, func(left, right types.Company) int {
		return cmp.Compare(left.ID, right.ID)
	})
	return normalized
}

func makeCompanyCreditUsageSummary(
	company types.Company,
	rows []coreTypes.CreditUsage,
	firstDay int32,
) (companyCreditUsageSummary, error) {
	days, totalCPU, totalInference, activeDays, err := makeCompanyCreditUsageDays(rows, firstDay)
	if err != nil {
		return companyCreditUsageSummary{}, err
	}

	summary := companyCreditUsageSummary{
		CompanyID: company.ID, Company: company.Name, Status: company.Status, Days: days,
		CPU: totalCPU, Inference: totalInference, ActiveDays: activeDays,
	}
	if summary.Company == "" {
		summary.Company = fmt.Sprintf("Company #%d", company.ID)
	}
	summary.TodayCPU = days[len(days)-1].CPU
	summary.TodayInference = days[len(days)-1].Inference
	return summary, nil
}

func makeCompanyCreditUsageDays(rows []coreTypes.CreditUsage, firstDay int32) ([]creditUsageDay, uint64, uint64, int16, error) {
	days := make([]creditUsageDay, companyCreditUsageDays)
	for dayOffset := range companyCreditUsageDays {
		days[dayOffset].Day = int16(firstDay + dayOffset)
	}
	var totalCPU, totalInference uint64
	var activeDays int16
	for _, row := range rows {
		dayOffset := row.TimeFrame - dailyTimeFramePrefix - firstDay
		if dayOffset < 0 || dayOffset >= companyCreditUsageDays {
			return nil, 0, 0, 0, fmt.Errorf("daily time frame %d is outside the report", row.TimeFrame)
		}
		totals, err := decodeCreditUsage(row.UsedCredits, nil)
		if err != nil {
			return nil, 0, 0, 0, fmt.Errorf("invalid credit blob in frame %d: %w", row.TimeFrame, err)
		}
		if math.MaxUint64-totalCPU < totals.CPU || math.MaxUint64-totalInference < totals.Inference {
			return nil, 0, 0, 0, fmt.Errorf("thirty-day credits overflow uint64")
		}
		days[dayOffset].CPU = totals.CPU
		days[dayOffset].Inference = totals.Inference
		totalCPU += totals.CPU
		totalInference += totals.Inference
		if totals.CPU > 0 || totals.Inference > 0 {
			activeDays++
		}
	}
	return days, totalCPU, totalInference, activeDays, nil
}

func makeCompanyCreditUsageUser(
	user coreTypes.User,
	rows []coreTypes.CreditUsage,
	firstDay int32,
) (companyCreditUsageUser, error) {
	days, totalCPU, totalInference, _, err := makeCompanyCreditUsageDays(rows, firstDay)
	if err != nil {
		return companyCreditUsageUser{}, err
	}
	displayName := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if displayName == "" {
		displayName = user.User
	}
	if displayName == "" {
		displayName = fmt.Sprintf("User #%d", user.ID)
	}
	return companyCreditUsageUser{
		UserID: user.ID, Name: displayName, User: user.User,
		CPU: totalCPU, Inference: totalInference, Days: days,
	}, nil
}

func sortCompanyCreditUsageUsers(users []companyCreditUsageUser) {
	slices.SortFunc(users, func(left, right companyCreditUsageUser) int {
		if left.CPU != right.CPU {
			return cmp.Compare(right.CPU, left.CPU)
		}
		if left.Inference != right.Inference {
			return cmp.Compare(right.Inference, left.Inference)
		}
		return cmp.Compare(left.UserID, right.UserID)
	})
}

func sortCompanyCreditUsageSummaries(summaries []companyCreditUsageSummary) {
	slices.SortFunc(summaries, func(left, right companyCreditUsageSummary) int {
		if left.CPU != right.CPU {
			return cmp.Compare(right.CPU, left.CPU)
		}
		if left.Inference != right.Inference {
			return cmp.Compare(right.Inference, left.Inference)
		}
		return cmp.Compare(left.CompanyID, right.CompanyID)
	})
}

func validateCompanyCreditUsageDay(day, firstDay, lastDay int32) error {
	if day < firstDay || day > lastDay {
		return fmt.Errorf("day %d is outside %d..%d", day, firstDay, lastDay)
	}
	return nil
}

func makeCompanyCreditUsageDetail(
	companyID int32,
	day int32,
	rows []coreTypes.CreditUsage,
) (companyCreditUsageDetail, error) {
	detail := companyCreditUsageDetail{CompanyID: companyID, Day: int16(day), Routes: []creditUsageRoute{}}
	if len(rows) == 0 {
		return detail, nil
	}
	routeTotals := map[int16]creditUsageTotals{}
	totals, err := decodeCreditUsage(rows[0].UsedCredits, routeTotals)
	if err != nil {
		return companyCreditUsageDetail{}, fmt.Errorf("invalid credit blob in frame %d: %w", rows[0].TimeFrame, err)
	}
	detail.CPU = totals.CPU
	detail.Inference = totals.Inference
	detail.Routes = makeCreditUsageRoutes(routeTotals)
	for routeIndex := range detail.Routes {
		if detail.Routes[routeIndex].Route == "" {
			detail.Routes[routeIndex].Route = companyCreditUnknownAPI
		}
	}
	return detail, nil
}
