package types

import "app/db"

// CompanyCreditBudget stores entitlement only; accepted consumption remains in credit_usage.
type CompanyCreditBudget struct {
	db.TableStruct[CompanyCreditBudgetTable, CompanyCreditBudget]
	CompanyID               int32
	DailyCPU                int64
	DailyInference          int64
	BudgetMonthStartDay     int16
	MonthlyCPUCeiling       int64
	MonthlyInferenceCeiling int64
	Updated                 int32
}

type CompanyCreditBudgetTable struct {
	db.TableStruct[CompanyCreditBudgetTable, CompanyCreditBudget]
	CompanyID               db.Col[CompanyCreditBudgetTable, int32]
	DailyCPU                db.Col[CompanyCreditBudgetTable, int64]
	DailyInference          db.Col[CompanyCreditBudgetTable, int64]
	BudgetMonthStartDay     db.Col[CompanyCreditBudgetTable, int16]
	MonthlyCPUCeiling       db.Col[CompanyCreditBudgetTable, int64]
	MonthlyInferenceCeiling db.Col[CompanyCreditBudgetTable, int64]
	Updated                 db.Col[CompanyCreditBudgetTable, int32]
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
