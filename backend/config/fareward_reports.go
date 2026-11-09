package config

import (
	"app/core"
	coreTypes "app/core/types"
	"context"
	"math"
)

// Usage, budgets and server metrics live in fareward, which is their source of truth (fareward
// plan D7): these helpers read them over its report opcode and convert them into the record types
// the panels already aggregate, so the handlers keep their logic and only change where they read.

func readCompanyUsage(companyID, firstFrame, lastFrame int32) ([]coreTypes.CreditUsageCompany, error) {
	rows, err := core.QueryCompanyUsage(context.Background(), companyID, firstFrame, lastFrame)
	if err != nil {
		return nil, err
	}
	return companyUsageRecords(rows), nil
}

func readEveryCompanyUsage(firstFrame, lastFrame int32) ([]coreTypes.CreditUsageCompany, error) {
	rows, err := core.QueryEveryCompanyUsage(context.Background(), firstFrame, lastFrame)
	if err != nil {
		return nil, err
	}
	return companyUsageRecords(rows), nil
}

// readUserUsage reads one user's rows, or every user's of the company when userID is 0.
func readUserUsage(companyID, userID, firstFrame, lastFrame int32) ([]coreTypes.CreditUsageUser, error) {
	rows, err := core.QueryUserUsage(context.Background(), companyID, userID, firstFrame, lastFrame)
	if err != nil {
		return nil, err
	}
	records := make([]coreTypes.CreditUsageUser, len(rows))
	for index, row := range rows {
		records[index] = coreTypes.CreditUsageUser{
			CompanyID: row.CompanyID, UserID: row.UserID, TimeFrame: row.TimeFrame,
			UsedCredits: row.UsedCredits,
		}
	}
	return records, nil
}

func companyUsageRecords(rows []core.CompanyUsageRow) []coreTypes.CreditUsageCompany {
	records := make([]coreTypes.CreditUsageCompany, len(rows))
	for index, row := range rows {
		records[index] = coreTypes.CreditUsageCompany{
			CompanyID: row.CompanyID, TimeFrame: row.TimeFrame, UsedCredits: row.UsedCredits,
		}
	}
	return records
}

// readCompanyBudgets reads one company's budget, or every company's when companyID is 0.
func readCompanyBudgets(companyID int32) ([]coreTypes.CompanyCreditBudget, error) {
	budgets, err := core.QueryCompanyBudgets(context.Background(), companyID)
	if err != nil {
		return nil, err
	}
	records := make([]coreTypes.CompanyCreditBudget, len(budgets))
	for index, budget := range budgets {
		records[index] = companyBudgetRecord(budget)
	}
	return records, nil
}

// companyBudgetRecord converts fareward's unsigned figures into the signed columns the panel's
// arithmetic was written against. A figure past int64 cannot be set (the mutation refuses it), so
// the clamp only guards against a corrupt value turning negative.
func companyBudgetRecord(budget core.CompanyBudget) coreTypes.CompanyCreditBudget {
	signed := func(value uint64) int64 {
		if value > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(value)
	}
	return coreTypes.CompanyCreditBudget{
		CompanyID:               budget.CompanyID,
		DailyCPU:                signed(budget.DailyCPU),
		DailyInference:          signed(budget.DailyInference),
		BudgetMonthStartDay:     budget.BudgetMonthStartDay,
		MonthlyCPUCeiling:       signed(budget.MonthlyCPUCeiling),
		MonthlyInferenceCeiling: signed(budget.MonthlyInferenceCeiling),
		LastSetCPU:              signed(budget.LastSetCPU),
		LastSetInference:        signed(budget.LastSetInference),
		Updated:                 budget.Updated,
		UsageDayPeriod:          budget.UsageDayPeriod,
		DayCPUUsed:              signed(budget.DayCPUUsed),
		DayInferenceUsed:        signed(budget.DayInferenceUsed),
		UsageMonthStartDay:      budget.UsageMonthStartDay,
		MonthCPUUsed:            signed(budget.MonthCPUUsed),
		MonthInferenceUsed:      signed(budget.MonthInferenceUsed),
		DayExtraCPUUsed:         signed(budget.DayExtraCPUUsed),
		MonthExtraCPUUsed:       signed(budget.MonthExtraCPUUsed),
		UsageUpdated:            budget.UsageUpdated,
	}
}

// readServerMetrics reads the windows between two (unix day, slot) positions, in time order.
func readServerMetrics(firstDate, firstSlot, lastDate, lastSlot int16) ([]serverMetricRow, error) {
	rows, err := core.QueryServerMetrics(context.Background(), firstDate, firstSlot, lastDate, lastSlot)
	if err != nil {
		return nil, err
	}
	records := make([]serverMetricRow, len(rows))
	for index, row := range rows {
		records[index] = serverMetricRow{
			ServerMetric: coreTypes.ServerMetric{
				Date: row.Date, Slot: row.Slot,
				CpuPercent: row.CPUPercent, MemPercent: row.MemPercent, DiskPercent: row.DiskPercent,
				NetRxRate: row.NetRxRate, NetTxRate: row.NetTxRate,
				BackendMemMb: row.BackendMemMB, BackendCpuPercent: row.BackendCPUPercent,
				FarewardMemMb: row.FarewardMemMB, FarewardCpuPercent: row.FarewardCPUPercent,
				ScyllaMemMb: row.DatabaseMemMB, ScyllaCpuPercent: row.DatabaseCPUPercent,
			},
			DbPoolMb:  row.DatabaseBufferPoolMB,
			DbDirtyMb: row.DatabaseDirtyMB,
			DbDiskGb:  row.DatabaseDiskGBTenths,
		}
	}
	return records, nil
}
