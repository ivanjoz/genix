// Package accounting seeds fixed assets acquired from the supplies catalog — half bought, half
// donated, part of them paid — and posts their depreciation at every month end.
package accounting

import (
	"app/core"
	"app/seeds/runner"
	"time"

	accountingModule "app/accounting"
)

var Module = runner.Module{Name: "accounting", Setup: Setup}

// Setup acquires the static asset list unless the company already holds that many assets.
func Setup(run *runner.Run) ([]runner.DayStep, error) {
	if run.Config.Assets > 0 {
		if err := seedAssets(run); err != nil {
			return nil, core.Err("No se pudieron generar los activos:", err)
		}
	}
	return []runner.DayStep{{Name: "depreciation", Order: runner.OrderDepreciation,
		Run: func(dayStart time.Time) error { return runMonthEndDepreciation(run, dayStart) }}}, nil
}

// runMonthEndDepreciation posts the month's depreciation on its last day, after the tills are counted.
func runMonthEndDepreciation(run *runner.Run, dayStart time.Time) error {
	if !runner.IsMonthDay(dayStart, 31) {
		return nil
	}
	core.SetHistoricalUnix(dayStart.Add(22 * time.Hour).Unix())
	request := run.MakeRequest("POST.asset-depreciation-run", nil, "")
	if response := run.Send(request, accountingModule.PostAssetDepreciationRun); response.StatusCode != 200 {
		return core.Err("no se pudo correr la depreciación:", response.Error)
	}
	return nil
}
