package accounting

import (
	accountingTypes "app/accounting/types"
	"testing"
	"time"
)

// dayOf builds a UnixDay from a calendar date, so the tests read as dates rather than
// as five-digit integers.
func dayOf(year int, month time.Month, day int) int16 {
	return utcToUnixDay(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

// A laptop bought for 4,000.00 (400000 cents) depreciating over 48 months.
func laptop() *accountingTypes.Asset {
	return &accountingTypes.Asset{
		AcquisitionDate:    dayOf(2024, time.January, 15),
		AcquisitionValue:   400000,
		DepreciationMonths: 48,
		Quantity:           1,
		Status:             accountingTypes.AssetStatusActive,
	}
}

func TestScheduleStartsTheMonthAfterAcquisition(t *testing.T) {
	asset := laptop()
	// Through the end of February: only one period can have elapsed, February's.
	periods := DepreciationSchedule(asset, dayOf(2024, time.February, 29))

	if len(periods) != 1 {
		t.Fatalf("expected 1 period through Feb, got %d", len(periods))
	}
	if got, want := periods[0].PeriodDate, dayOf(2024, time.February, 1); got != want {
		t.Errorf("first period date = %d, want %d (Feb 1st)", got, want)
	}
	if periods[0].PeriodIndex != 1 {
		t.Errorf("first period index = %d, want 1", periods[0].PeriodIndex)
	}

	// January, the acquisition month itself, must not be posted.
	januaryPeriods := DepreciationSchedule(asset, dayOf(2024, time.January, 31))
	if len(januaryPeriods) != 0 {
		t.Errorf("acquisition month must not depreciate, got %d periods", len(januaryPeriods))
	}
}

func TestFullScheduleSumsToAcquisitionValueExactly(t *testing.T) {
	asset := laptop()
	// Well past the end of the 48-month life.
	periods := DepreciationSchedule(asset, dayOf(2030, time.January, 1))

	if len(periods) != 48 {
		t.Fatalf("expected 48 periods, got %d", len(periods))
	}

	var total int32
	for _, period := range periods {
		total += period.Amount
	}
	if total != asset.AcquisitionValue {
		t.Errorf("periods sum to %d, want exactly %d", total, asset.AcquisitionValue)
	}
	if last := periods[47]; last.Accumulated != asset.AcquisitionValue {
		t.Errorf("final accumulated = %d, want %d", last.Accumulated, asset.AcquisitionValue)
	}
}

// 100000 over 3 months is 33333.33...; integer division loses a cent per period, and the
// final period has to absorb it or the book value never reaches zero.
func TestFinalPeriodAbsorbsTheRoundingRemainder(t *testing.T) {
	asset := &accountingTypes.Asset{
		AcquisitionDate:    dayOf(2024, time.January, 1),
		AcquisitionValue:   100000,
		DepreciationMonths: 3,
	}
	periods := DepreciationSchedule(asset, dayOf(2025, time.January, 1))

	if len(periods) != 3 {
		t.Fatalf("expected 3 periods, got %d", len(periods))
	}
	if periods[0].Amount != 33333 || periods[1].Amount != 33333 {
		t.Errorf("regular periods = %d/%d, want 33333 each", periods[0].Amount, periods[1].Amount)
	}
	if periods[2].Amount != 33334 {
		t.Errorf("final period = %d, want 33334 (absorbing the 1-cent remainder)", periods[2].Amount)
	}

	var total int32
	for _, period := range periods {
		total += period.Amount
	}
	if total != 100000 {
		t.Errorf("periods sum to %d, want 100000", total)
	}
}

func TestDisposalStopsTheSchedule(t *testing.T) {
	asset := laptop()
	asset.DisposalDate = dayOf(2024, time.June, 10)

	periods := DepreciationSchedule(asset, dayOf(2030, time.January, 1))

	// Feb, Mar, Apr, May, Jun all begin on or before the disposal date; July does not.
	if len(periods) != 5 {
		t.Fatalf("expected 5 periods up to disposal, got %d", len(periods))
	}
	if got, want := periods[4].PeriodDate, dayOf(2024, time.June, 1); got != want {
		t.Errorf("last period = %d, want %d (June 1st)", got, want)
	}
}

// The donated computer: it cost nothing and is still worth 10,000, so it still depreciates.
func TestDonatedAssetDepreciatesOnValueNotPrice(t *testing.T) {
	asset := &accountingTypes.Asset{
		AcquisitionDate:    dayOf(2024, time.January, 1),
		AcquisitionValue:   1000000, // 10,000.00 — the book value.
		DepreciationMonths: 48,
		// A donation owes nothing: PurchaseAmount stays 0 and the asset is never payable,
		// but it still carries a book value and still depreciates.
		PurchaseAmount: 0,
		PaymentStatus:  accountingTypes.AssetPaymentNone,
	}
	periods := DepreciationSchedule(asset, dayOf(2024, time.March, 31))

	if len(periods) != 2 {
		t.Fatalf("expected 2 periods, got %d", len(periods))
	}
	if periods[0].Amount != 20833 {
		t.Errorf("monthly amount = %d, want 20833 (1000000/48)", periods[0].Amount)
	}
}

func TestNonDepreciableSupplyYieldsNothing(t *testing.T) {
	asset := &accountingTypes.Asset{
		AcquisitionDate:    dayOf(2024, time.January, 1),
		AcquisitionValue:   50000,
		DepreciationMonths: 0, // A plain consumable.
	}
	if periods := DepreciationSchedule(asset, dayOf(2030, time.January, 1)); len(periods) != 0 {
		t.Errorf("a consumable must not depreciate, got %d periods", len(periods))
	}
}

func TestPendingPeriodsAreIdempotentAgainstTheWatermark(t *testing.T) {
	asset := laptop()
	through := dayOf(2024, time.May, 31)

	firstRun := PendingDepreciationPeriods(asset, through)
	if len(firstRun) != 4 {
		t.Fatalf("expected 4 pending periods (Feb-May), got %d", len(firstRun))
	}

	// Simulate posting them: the watermark advances to the last period written.
	asset.LastDepreciationDate = firstRun[len(firstRun)-1].PeriodDate
	asset.AccumulatedDepreciation = firstRun[len(firstRun)-1].Accumulated

	secondRun := PendingDepreciationPeriods(asset, through)
	if len(secondRun) != 0 {
		t.Errorf("re-running the same day must post nothing, got %d periods", len(secondRun))
	}

	// A month later, exactly one new period is due.
	thirdRun := PendingDepreciationPeriods(asset, dayOf(2024, time.June, 30))
	if len(thirdRun) != 1 {
		t.Fatalf("expected 1 new period in June, got %d", len(thirdRun))
	}
	if got, want := thirdRun[0].PeriodDate, dayOf(2024, time.June, 1); got != want {
		t.Errorf("new period = %d, want %d", got, want)
	}
}

func TestResolveAssetStatus(t *testing.T) {
	fullyDepreciated := laptop()
	fullyDepreciated.AccumulatedDepreciation = fullyDepreciated.AcquisitionValue
	if got := ResolveAssetStatus(fullyDepreciated); got != accountingTypes.AssetStatusFullyDepreciated {
		t.Errorf("fully depreciated status = %d, want %d", got, accountingTypes.AssetStatusFullyDepreciated)
	}

	// Disposal outranks full depreciation: an asset can be sold before its life ends.
	disposed := laptop()
	disposed.AccumulatedDepreciation = disposed.AcquisitionValue
	disposed.DisposalDate = dayOf(2025, time.January, 1)
	if got := ResolveAssetStatus(disposed); got != accountingTypes.AssetStatusDisposed {
		t.Errorf("disposed status = %d, want %d", got, accountingTypes.AssetStatusDisposed)
	}

	if got := ResolveAssetStatus(laptop()); got != accountingTypes.AssetStatusActive {
		t.Errorf("new asset status = %d, want %d", got, accountingTypes.AssetStatusActive)
	}
}
