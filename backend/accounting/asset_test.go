package accounting

import (
	"app/accounting/types"
	"testing"
	"time"
)

// A purchased asset owes money on its own account, not through the expense register.
func TestPendingAmountTracksPartialPayments(t *testing.T) {
	asset := &types.Asset{
		AcquisitionValue: 400000,
		PurchaseAmount:   400000,
		PaymentStatus:    types.AssetPaymentPending,
	}

	if got := asset.PendingAmount(); got != 400000 {
		t.Errorf("nothing paid yet: pending = %d, want 400000", got)
	}

	asset.PaidAmount = 150000
	if got := asset.PendingAmount(); got != 250000 {
		t.Errorf("after a partial payment: pending = %d, want 250000", got)
	}

	// An overpayment (a write-off settling more than the balance) must not read as negative.
	asset.PaidAmount = 450000
	if got := asset.PendingAmount(); got != 0 {
		t.Errorf("overpaid: pending = %d, want 0", got)
	}
}

// A donated asset owes nothing, so it is never payable — but it is still on the books and
// still depreciates. This is the case that made purchase and value separate numbers.
func TestDonatedAssetOwesNothingButStillHasValue(t *testing.T) {
	donated := &types.Asset{
		AcquisitionDate:    dayOf(2024, time.January, 1),
		AcquisitionValue:   1000000,
		PurchaseAmount:     0,
		PaymentStatus:      types.AssetPaymentNone,
		DepreciationMonths: 48,
	}

	if got := donated.PendingAmount(); got != 0 {
		t.Errorf("a donation owes nothing: pending = %d, want 0", got)
	}
	if got := donated.BookValue(); got != 1000000 {
		t.Errorf("book value = %d, want 1000000", got)
	}
	if periods := DepreciationSchedule(donated, dayOf(2024, time.March, 31)); len(periods) != 2 {
		t.Errorf("a donation still depreciates: got %d periods, want 2", len(periods))
	}
}

// Depreciation reduces book value; it has nothing to do with what was paid. An asset can be
// fully paid and barely depreciated, or fully depreciated and still unpaid.
func TestBookValueIsIndependentOfPayment(t *testing.T) {
	asset := &types.Asset{
		AcquisitionValue:        400000,
		PurchaseAmount:          400000,
		PaidAmount:              400000,
		PaymentStatus:           types.AssetPaymentPaid,
		AccumulatedDepreciation: 100000,
		DepreciationMonths:      48,
		// Set explicitly: ResolveAssetStatus preserves an explicit removal, and status 0 is
		// both "removed" and the zero value, so a live asset always carries its own.
		Status: types.AssetStatusActive,
	}

	if got := asset.PendingAmount(); got != 0 {
		t.Errorf("fully paid: pending = %d, want 0", got)
	}
	if got := asset.BookValue(); got != 300000 {
		t.Errorf("fully paid but only a quarter depreciated: book value = %d, want 300000", got)
	}
	if got := ResolveAssetStatus(asset); got != types.AssetStatusActive {
		t.Errorf("paying an asset off does not retire it: status = %d, want active", got)
	}
}

// The payment slot is derived from the two amounts everywhere, so acquisition, payment and edit
// cannot disagree about what "paid" means.
func TestResolveAssetPaymentStatus(t *testing.T) {
	cases := []struct {
		name           string
		purchaseAmount int32
		paidAmount     int32
		want           int8
	}{
		{"donated", 0, 0, types.AssetPaymentNone},
		{"nothing paid yet", 400000, 0, types.AssetPaymentPending},
		{"partially paid", 400000, 150000, types.AssetPaymentPending},
		{"exactly settled", 400000, 400000, types.AssetPaymentPaid},
		{"overpaid by a write-off", 400000, 450000, types.AssetPaymentPaid},
	}
	for _, testCase := range cases {
		if got := ResolveAssetPaymentStatus(
			testCase.purchaseAmount, testCase.paidAmount,
		); got != testCase.want {
			t.Errorf("%s: status = %d, want %d", testCase.name, got, testCase.want)
		}
	}
}

// Correcting the book value rewrites the whole elapsed schedule rather than spreading the
// difference over the months still to come: the register has to end up looking like the asset
// was entered correctly the first time.
func TestEditingTheValueRewritesEveryElapsedPeriod(t *testing.T) {
	asset := &types.Asset{
		AcquisitionDate:    dayOf(2024, time.January, 15),
		AcquisitionValue:   400000,
		DepreciationMonths: 48,
		Status:             types.AssetStatusActive,
	}
	throughDate := dayOf(2024, time.June, 20)

	postedSchedule := DepreciationSchedule(asset, throughDate)
	if len(postedSchedule) != 5 {
		t.Fatalf("Feb through Jun: got %d periods, want 5", len(postedSchedule))
	}

	// The typo: the value was half of what was entered.
	asset.AcquisitionValue = 200000
	rewrittenSchedule := DepreciationSchedule(asset, throughDate)

	if len(rewrittenSchedule) != len(postedSchedule) {
		t.Fatalf("a value change moves no period: got %d, want %d",
			len(rewrittenSchedule), len(postedSchedule))
	}
	for periodIndex := range rewrittenSchedule {
		if rewrittenSchedule[periodIndex].PeriodDate != postedSchedule[periodIndex].PeriodDate {
			t.Errorf("period %d moved date: the posted rows must pair up by index", periodIndex+1)
		}
		// Every posted amount is rewritten, not just the ones after the edit.
		if rewrittenSchedule[periodIndex].Amount != 4166 {
			t.Errorf("period %d amount = %d, want 4166",
				periodIndex+1, rewrittenSchedule[periodIndex].Amount)
		}
	}
	if got := rewrittenSchedule[len(rewrittenSchedule)-1].Accumulated; got != 20830 {
		t.Errorf("accumulated after the rewrite = %d, want 20830", got)
	}
}

// Moving the acquisition date forward means months that were posted never happened. The
// schedule shrinks, and rewriteAssetDepreciation retires the tail it no longer covers.
func TestMovingTheAcquisitionDateForwardShortensTheSchedule(t *testing.T) {
	asset := &types.Asset{
		AcquisitionDate:    dayOf(2024, time.January, 15),
		AcquisitionValue:   400000,
		DepreciationMonths: 48,
		Status:             types.AssetStatusActive,
	}
	throughDate := dayOf(2024, time.June, 20)

	postedCount := len(DepreciationSchedule(asset, throughDate))

	asset.AcquisitionDate = dayOf(2024, time.April, 15)
	rewrittenSchedule := DepreciationSchedule(asset, throughDate)

	if len(rewrittenSchedule) != 2 {
		t.Fatalf("May and Jun: got %d periods, want 2", len(rewrittenSchedule))
	}
	if len(rewrittenSchedule) >= postedCount {
		t.Errorf("the schedule must shrink: %d periods, was %d", len(rewrittenSchedule), postedCount)
	}
	// Nothing is lost: the periods that remain still start the schedule from the first month.
	if rewrittenSchedule[0].PeriodIndex != 1 {
		t.Errorf("first remaining period index = %d, want 1", rewrittenSchedule[0].PeriodIndex)
	}
}

// A date change so recent that no month has elapsed leaves nothing posted at all, which is what
// resets the asset's accumulated total and watermark to zero.
func TestMovingTheAcquisitionDateIntoTheCurrentMonthEmptiesTheSchedule(t *testing.T) {
	asset := &types.Asset{
		AcquisitionDate:    dayOf(2024, time.June, 5),
		AcquisitionValue:   400000,
		DepreciationMonths: 48,
		Status:             types.AssetStatusActive,
	}

	if periods := DepreciationSchedule(asset, dayOf(2024, time.June, 20)); len(periods) != 0 {
		t.Errorf("depreciation starts the month after acquisition: got %d periods, want 0", len(periods))
	}
}
