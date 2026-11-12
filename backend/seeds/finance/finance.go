// Package finance seeds a two-store minimarket's operating expenses and their payments, the
// tills' daily deposits into the bank and the month-end till counts.
package finance

import (
	"app/core"
	"app/seeds/runner"
	"time"
)

var Module = runner.Module{Name: "finance", Setup: Setup}

// Setup seeds the expense suppliers, loads the consumable supplies the inventory purchases buy, and
// makes sure every monthly bill has its expense schedule.
func Setup(run *runner.Run) ([]runner.DayStep, error) {
	if err := prepareExpenses(run); err != nil {
		return nil, err
	}
	return []runner.DayStep{
		{Name: "expenses", Order: runner.OrderExpenses, Run: func(dayStart time.Time) error { return runExpensesStep(run, dayStart) }},
		{Name: "till-transfers", Order: runner.OrderTillTransfers, Run: func(dayStart time.Time) error { return runTillTransfersStep(run, dayStart) }},
		{Name: "till-counts", Order: runner.OrderTillCounts, Run: func(dayStart time.Time) error { return runTillCountsStep(run, dayStart) }},
	}, nil
}

// BookAnyDayExpense books one expense of a template a business pays any day (not the monthly
// schedules nor the taxes, which belong to a calendar), paying it at once from the bank. The load
// test's back office books these.
func BookAnyDayExpense(run *runner.Run, now time.Time) error {
	candidateTemplates := []expenseTemplate{}
	for _, template := range expenseTemplates {
		if template.cadence != cadenceScheduledMonthly && template.cadence != cadenceMonthlyTax {
			candidateTemplates = append(candidateTemplates, template)
		}
	}
	template := candidateTemplates[run.Random.Intn(len(candidateTemplates))]
	planned := runner.PlannedExpense{TemplateKey: template.key, PayDay: core.TimeToFechaUnix(now)}
	if template.isInventory {
		supply := run.ExpenseSupplies[run.Random.Intn(len(run.ExpenseSupplies))]
		planned.ProductID = supply.ID
		planned.WarehouseID = run.Config.WarehouseIDs[run.Random.Intn(len(run.Config.WarehouseIDs))]
		planned.Quantity = int32(5 + run.Random.Intn(26))
		planned.UnitPrice = supply.Price
		planned.Amount = planned.UnitPrice * planned.Quantity
	} else {
		planned.Amount = int32(run.RandomInRange(int(template.minAmount), int(template.maxAmount)))
	}
	planned.PayAmount = planned.Amount
	return bookPlannedExpense(run, planned, now)
}
