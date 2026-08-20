package types

import "app/db"

// CompanyCreditBudget stores entitlement only; accepted consumption remains in credit_usage_company.
type CompanyCreditBudget struct {
	db.TableStruct[CompanyCreditBudgetTable, CompanyCreditBudget]
	CompanyID               int32
	DailyCPU                int64
	DailyInference          int64
	BudgetMonthStartDay     int16
	MonthlyCPUCeiling       int64
	MonthlyInferenceCeiling int64
	// LastSet* is the figure the last "set current" wrote. The ceiling alone cannot answer "how much
	// of the granted credit is gone", because it folds in the usage that existed when it was set;
	// with this, consumed-since-the-grant is LastSet minus what remains.
	LastSetCPU       int64
	LastSetInference int64
	Updated          int32
	// Usage counters the credit daemon flushes here every flush interval. They are the very numbers
	// the daemon enforces on (quota.rs: day_used and month_used), so a reader that subtracts them
	// from the ceilings above shows what the limiter would actually allow, instead of re-deriving it
	// from the usage rows and drifting. Written only by the daemon; the backend never writes them.
	// The period columns say which window each counter belongs to: a reader whose current window
	// differs reads the counters as zero, because a window the daemon has not touched yet is unused.
	UsageDayPeriod     int16
	DayCPUUsed         int64
	DayInferenceUsed   int64
	UsageMonthStartDay int16
	MonthCPUUsed       int64
	MonthInferenceUsed int64
	// Zero means the daemon has never flushed for this company, which is the only case where the
	// counters cannot be trusted and month usage has to be summed from the usage rows instead.
	UsageUpdated int32
}

type CompanyCreditBudgetTable struct {
	db.TableStruct[CompanyCreditBudgetTable, CompanyCreditBudget]
	CompanyID               db.Col[CompanyCreditBudgetTable, int32]
	DailyCPU                db.Col[CompanyCreditBudgetTable, int64]
	DailyInference          db.Col[CompanyCreditBudgetTable, int64]
	BudgetMonthStartDay     db.Col[CompanyCreditBudgetTable, int16]
	MonthlyCPUCeiling       db.Col[CompanyCreditBudgetTable, int64]
	MonthlyInferenceCeiling db.Col[CompanyCreditBudgetTable, int64]
	LastSetCPU              db.Col[CompanyCreditBudgetTable, int64]
	LastSetInference        db.Col[CompanyCreditBudgetTable, int64]
	Updated                 db.Col[CompanyCreditBudgetTable, int32]
	UsageDayPeriod          db.Col[CompanyCreditBudgetTable, int16]
	DayCPUUsed              db.Col[CompanyCreditBudgetTable, int64]
	DayInferenceUsed        db.Col[CompanyCreditBudgetTable, int64]
	UsageMonthStartDay      db.Col[CompanyCreditBudgetTable, int16]
	MonthCPUUsed            db.Col[CompanyCreditBudgetTable, int64]
	MonthInferenceUsed      db.Col[CompanyCreditBudgetTable, int64]
	UsageUpdated            db.Col[CompanyCreditBudgetTable, int32]
}

func (e CompanyCreditBudgetTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		// One company owns exactly one entitlement row, so company_id is the complete primary key.
		ID:                    48,
		Name:                  "company_credit_budget",
		Keys:                  db.Cols(e.CompanyID),
		DisableDefaultColumns: true,
	}
}
