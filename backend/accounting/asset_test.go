package accounting

import (
	accountingTypes "app/accounting/types"
	"testing"
	"time"
)

// A purchased asset owes money on its own account, not through the expense register.
func TestPendingAmountTracksPartialPayments(t *testing.T) {
	asset := &accountingTypes.Asset{
		AcquisitionValue: 400000,
		PurchaseAmount:   400000,
		PaymentStatus:    accountingTypes.AssetPaymentPending,
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
	donated := &accountingTypes.Asset{
		AcquisitionDate:    dayOf(2024, time.January, 1),
		AcquisitionValue:   1000000,
		PurchaseAmount:     0,
		PaymentStatus:      accountingTypes.AssetPaymentNone,
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
	asset := &accountingTypes.Asset{
		AcquisitionValue:        400000,
		PurchaseAmount:          400000,
		PaidAmount:              400000,
		PaymentStatus:           accountingTypes.AssetPaymentPaid,
		AccumulatedDepreciation: 100000,
		DepreciationMonths:      48,
		// Set explicitly: ResolveAssetStatus preserves an explicit removal, and status 0 is
		// both "removed" and the zero value, so a live asset always carries its own.
		Status: accountingTypes.AssetStatusActive,
	}

	if got := asset.PendingAmount(); got != 0 {
		t.Errorf("fully paid: pending = %d, want 0", got)
	}
	if got := asset.BookValue(); got != 300000 {
		t.Errorf("fully paid but only a quarter depreciated: book value = %d, want 300000", got)
	}
	if got := ResolveAssetStatus(asset); got != accountingTypes.AssetStatusActive {
		t.Errorf("paying an asset off does not retire it: status = %d, want active", got)
	}
}
