package config

import (
	"app/config/types"
	coreTypes "app/core/types"
	"testing"
)

func TestMakeCompanyCreditUsageSummaryZeroFillsAndTotalsThirtyDays(t *testing.T) {
	rows := []coreTypes.CreditUsage{
		{TimeFrame: dailyTimeFramePrefix + 100, UsedCredits: []byte{0, 4, 10, 2}},
		{TimeFrame: dailyTimeFramePrefix + 129, UsedCredits: []byte{0, 8, 20, 3}},
	}
	summary, err := makeCompanyCreditUsageSummary(types.Company{ID: 7, Name: "Acme", Status: 1}, rows, 100)
	if err != nil {
		t.Fatalf("makeCompanyCreditUsageSummary() error = %v", err)
	}
	if len(summary.Days) != 30 || summary.Days[0].Day != 100 || summary.Days[29].Day != 129 {
		t.Fatalf("unexpected day window: %#v", summary.Days)
	}
	if summary.CPU != 30 || summary.Inference != 5 || summary.TodayCPU != 20 || summary.ActiveDays != 2 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.Days[1].CPU != 0 || summary.Days[1].Inference != 0 {
		t.Fatalf("missing day was not zero-filled: %#v", summary.Days[1])
	}
}

func TestSortCompanyCreditUsageSummariesUsesStableCreditOrder(t *testing.T) {
	summaries := []companyCreditUsageSummary{
		{CompanyID: 3, CPU: 10, Inference: 9},
		{CompanyID: 1, CPU: 20},
		{CompanyID: 2, CPU: 10, Inference: 9},
	}
	sortCompanyCreditUsageSummaries(summaries)
	if summaries[0].CompanyID != 1 || summaries[1].CompanyID != 2 || summaries[2].CompanyID != 3 {
		t.Fatalf("unexpected ranking: %#v", summaries)
	}
}

func TestNormalizeCompanyCreditCatalogKeepsNewestRecord(t *testing.T) {
	companies := normalizeCompanyCreditCatalog([]types.Company{
		{ID: 2, Name: "Old", Updated: 2},
		{ID: 0, Name: "Reserved"},
		{ID: 2, Name: "New", Updated: 3},
		{ID: 1, Name: "First", Updated: 1},
	})
	if len(companies) != 2 || companies[0].ID != 1 || companies[1].Name != "New" {
		t.Fatalf("unexpected normalized catalog: %#v", companies)
	}
}

func TestMakeCompanyCreditUsageDetailReturnsRouteBreakdown(t *testing.T) {
	detail, err := makeCompanyCreditUsageDetail(4, 100, []coreTypes.CreditUsage{{
		TimeFrame:   dailyTimeFramePrefix + 100,
		UsedCredits: []byte{0, 4, 10, 2, 0, 8, 20, 3},
	}})
	if err != nil {
		t.Fatalf("makeCompanyCreditUsageDetail() error = %v", err)
	}
	if detail.CPU != 30 || detail.Inference != 5 || len(detail.Routes) != 2 {
		t.Fatalf("unexpected detail: %#v", detail)
	}
	if detail.Routes[0].RouteID != 2 || detail.Routes[1].RouteID != 1 {
		t.Fatalf("routes were not ranked by CPU: %#v", detail.Routes)
	}
}

func TestValidateCompanyCreditUsageDayRejectsOutsideWindow(t *testing.T) {
	if validateCompanyCreditUsageDay(100, 100, 129) != nil {
		t.Fatal("first day should be valid")
	}
	if validateCompanyCreditUsageDay(130, 100, 129) == nil {
		t.Fatal("future day should be rejected")
	}
}

func TestMakeCompanyCreditAdministratorIdentitySanitizesDisplayFields(t *testing.T) {
	administrator := coreTypes.User{
		CompanyID: 7, ID: 1, FirstName: "  Ada", LastName: "Lovelace  ", User: "admin",
		Email: "private@example.com", PasswordHash: "not-for-the-report",
	}
	administratorName, administratorUser := makeCompanyCreditAdministratorIdentity(&administrator)
	if administratorName != "Ada Lovelace" || administratorUser != "admin" {
		t.Fatalf("unexpected administrator identity: %q / %q", administratorName, administratorUser)
	}

	fallbackName, fallbackUser := makeCompanyCreditAdministratorIdentity(&coreTypes.User{User: "admin"})
	if fallbackName != "admin" || fallbackUser != "admin" {
		t.Fatalf("unexpected administrator fallback: %q / %q", fallbackName, fallbackUser)
	}

	missingName, missingUser := makeCompanyCreditAdministratorIdentity(nil)
	if missingName != "" || missingUser != "" {
		t.Fatalf("missing administrator should be empty: %q / %q", missingName, missingUser)
	}
}

func TestMakeCompanyCreditUsageUserBuildsThirtyDaySeries(t *testing.T) {
	user := coreTypes.User{ID: 7, FirstName: "  Ada", LastName: "Lovelace  ", User: "ada"}
	usage, err := makeCompanyCreditUsageUser(user, []coreTypes.CreditUsage{
		{TimeFrame: dailyTimeFramePrefix + 100, UsedCredits: []byte{0, 4, 10, 2}},
		{TimeFrame: dailyTimeFramePrefix + 129, UsedCredits: []byte{0, 8, 20, 3}},
	}, 100)
	if err != nil {
		t.Fatalf("makeCompanyCreditUsageUser() error = %v", err)
	}
	if usage.Name != "Ada Lovelace" || usage.User != "ada" || usage.CPU != 30 || usage.Inference != 5 {
		t.Fatalf("unexpected user usage: %#v", usage)
	}
	if len(usage.Days) != 30 || usage.Days[0].CPU != 10 || usage.Days[29].Inference != 3 {
		t.Fatalf("unexpected user days: %#v", usage.Days)
	}
}

func TestSortCompanyCreditUsageUsersRanksByCost(t *testing.T) {
	users := []companyCreditUsageUser{
		{UserID: 3, CPU: 10, Inference: 9},
		{UserID: 1, CPU: 20},
		{UserID: 2, CPU: 10, Inference: 9},
	}
	sortCompanyCreditUsageUsers(users)
	if users[0].UserID != 1 || users[1].UserID != 2 || users[2].UserID != 3 {
		t.Fatalf("unexpected ranking: %#v", users)
	}
}
