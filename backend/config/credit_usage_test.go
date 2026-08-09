package config

import (
	coreTypes "app/core/types"
	"reflect"
	"testing"
)

func TestDecodeCreditUsageSumsCanonicalAPIGroups(t *testing.T) {
	// Group 1 uses one-byte values; group 3 uses the documented two-byte example.
	encoded := []byte{0x04, 0x05, 0x07, 0x0D, 0x01, 0x2C, 0x00, 0x19}
	got, err := decodeCreditUsage(encoded)
	if err != nil {
		t.Fatalf("decodeCreditUsage() error = %v", err)
	}
	want := creditUsageTotals{CPU: 305, Inference: 32}
	if got != want {
		t.Fatalf("decodeCreditUsage() = %+v, want %+v", got, want)
	}
}

func TestDecodeCreditUsageRejectsInvalidEncoding(t *testing.T) {
	invalidBlobs := [][]byte{
		{0x05, 0x00},                         // Truncated two-byte values.
		{0x04, 0x01, 0x01, 0x04, 0x02, 0x02}, // Duplicate API group.
		{0x00, 0x00, 0x00},                   // All-zero group.
		{0x01, 0x00, 0x01, 0x00, 0x01},       // Non-minimal width.
	}
	for _, invalidBlob := range invalidBlobs {
		if _, err := decodeCreditUsage(invalidBlob); err == nil {
			t.Fatalf("decodeCreditUsage(%x) unexpectedly succeeded", invalidBlob)
		}
	}
}

func TestMakeCreditUsageScopeZeroFillsFifteenUTCDays(t *testing.T) {
	rows := []coreTypes.CreditUsage{{
		TimeFrame:   dailyTimeFramePrefix + 103,
		UsedCredits: []byte{0x04, 0x05, 0x07},
	}}
	got, err := makeCreditUsageScope(rows, 100, 1_000, 500)
	if err != nil {
		t.Fatalf("makeCreditUsageScope() error = %v", err)
	}
	if len(got.Days) != int(creditUsageDaysCount) {
		t.Fatalf("len(Days) = %d, want %d", len(got.Days), creditUsageDaysCount)
	}
	if got.Days[0].Day != 100 || got.Days[14].Day != 114 {
		t.Fatalf("day range = %d..%d, want 100..114", got.Days[0].Day, got.Days[14].Day)
	}
	wantUsedDay := creditUsageDay{Day: 103, CPU: 5, Inference: 7}
	if !reflect.DeepEqual(got.Days[3], wantUsedDay) {
		t.Fatalf("Days[3] = %+v, want %+v", got.Days[3], wantUsedDay)
	}
}
