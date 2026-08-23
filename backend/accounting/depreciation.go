package accounting

import (
	accountingTypes "app/accounting/types"
	"time"
)

// unixDayToUTC / utcToUnixDay round-trip a UnixDay (int16) through a UTC date so the
// month walker can reason about calendar months deterministically. Same convention as
// the expense cadence walker in finance/expenses.go.
func unixDayToUTC(day int16) time.Time { return time.Unix(int64(day)*86400, 0).UTC() }
func utcToUnixDay(t time.Time) int16   { return int16(t.UTC().Unix() / 86400) }

// DepreciationPeriod is one month of an asset's straight-line schedule.
type DepreciationPeriod struct {
	// PeriodDate is the UnixDay of the first day of the month being posted. It is the
	// dedupe key for lazy generation: a period whose date is already covered by
	// Asset.LastDepreciationDate is never written twice.
	PeriodDate int16
	// PeriodIndex counts from 1, so the caller can name the entry ("month 7 of 48")
	// without recomputing it from the dates.
	PeriodIndex int16
	Amount      int32
	// Accumulated is the running total *including* this period, so the last period's
	// value always equals the asset's acquisition value exactly.
	Accumulated int32
}

// DepreciationSchedule returns every period of an asset's straight-line life that has
// already elapsed as of throughDate.
//
// Three rules decide the shape, and each of them is a decision rather than an obvious
// default (see RATIONALE.md):
//
//   - Depreciation starts the month *after* acquisition. An asset bought on the 28th does
//     not earn a full month of depreciation for those three days.
//   - The final period absorbs the rounding remainder, so the periods sum to exactly
//     AcquisitionValue and the book value lands on 0 rather than a few cents either side.
//   - Disposal cuts the schedule off. A month is posted only if the asset was still held
//     at the start of it.
//
// A zero DepreciationMonths (a plain consumable) or a non-positive value yields nothing.
func DepreciationSchedule(asset *accountingTypes.Asset, throughDate int16) []DepreciationPeriod {
	if asset.DepreciationMonths <= 0 || asset.AcquisitionValue <= 0 || asset.AcquisitionDate <= 0 {
		return nil
	}

	// Straight line: every month carries the same amount except the last, which takes
	// whatever integer division left behind.
	monthlyAmount := asset.AcquisitionValue / int32(asset.DepreciationMonths)

	acquisitionTime := unixDayToUTC(asset.AcquisitionDate)
	// Anchor on the first of the acquisition month, then step forward: period N is the
	// month starting N months after that, which is the month *after* acquisition for N=1.
	monthAnchor := time.Date(
		acquisitionTime.Year(), acquisitionTime.Month(), 1, 0, 0, 0, 0, time.UTC,
	)

	periods := make([]DepreciationPeriod, 0, asset.DepreciationMonths)
	accumulated := int32(0)

	for periodIndex := int16(1); periodIndex <= asset.DepreciationMonths; periodIndex++ {
		periodDate := utcToUnixDay(monthAnchor.AddDate(0, int(periodIndex), 0))

		// The period has not happened yet, and nothing after it has either.
		if periodDate > throughDate {
			break
		}
		// Disposed before this month began: the asset stops depreciating from then on.
		if asset.DisposalDate > 0 && periodDate > asset.DisposalDate {
			break
		}

		periodAmount := monthlyAmount
		if periodIndex == asset.DepreciationMonths {
			// Last period: whatever is left, so the schedule sums to the acquisition value.
			periodAmount = asset.AcquisitionValue - accumulated
		}
		accumulated += periodAmount

		periods = append(periods, DepreciationPeriod{
			PeriodDate:  periodDate,
			PeriodIndex: periodIndex,
			Amount:      periodAmount,
			Accumulated: accumulated,
		})
	}

	return periods
}

// PendingDepreciationPeriods narrows a schedule to the periods not yet posted, which is
// what lazy generation writes. Filtering on the stored watermark rather than on a count
// keeps generation idempotent: running it twice in one day produces nothing the second time.
func PendingDepreciationPeriods(asset *accountingTypes.Asset, throughDate int16) []DepreciationPeriod {
	schedule := DepreciationSchedule(asset, throughDate)
	pending := make([]DepreciationPeriod, 0, len(schedule))
	for _, period := range schedule {
		if period.PeriodDate > asset.LastDepreciationDate {
			pending = append(pending, period)
		}
	}
	return pending
}

// ResolveAssetStatus derives the lifecycle slot from the asset's own numbers, so the
// status can never disagree with the ledger that produced it.
func ResolveAssetStatus(asset *accountingTypes.Asset) int8 {
	if asset.Status == accountingTypes.AssetStatusRemoved {
		return accountingTypes.AssetStatusRemoved
	}
	if asset.DisposalDate > 0 {
		return accountingTypes.AssetStatusDisposed
	}
	if asset.DepreciationMonths > 0 && asset.AccumulatedDepreciation >= asset.AcquisitionValue {
		return accountingTypes.AssetStatusFullyDepreciated
	}
	return accountingTypes.AssetStatusActive
}
