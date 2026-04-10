package core

import "testing"

func TestMakeGroupIndexCacheValuesRestoresSignedHashAndAlignedCounter(t *testing.T) {
	records := makeGroupIndexCacheValues(
		[]int64{4294967295, 2147483648, 42},
		[]int64{7, 8, 9},
	)

	if len(records) != 3 {
		t.Fatalf("expected 3 cache records, got %+v", records)
	}
	if records[0].GroupHash != -1 || records[0].UpdateCounter != 7 {
		t.Fatalf("unexpected first record: %+v", records[0])
	}
	if records[1].GroupHash != -2147483648 || records[1].UpdateCounter != 8 {
		t.Fatalf("unexpected second record: %+v", records[1])
	}
	if records[2].GroupHash != 42 || records[2].UpdateCounter != 9 {
		t.Fatalf("unexpected third record: %+v", records[2])
	}
}

func TestMakeGroupIndexCacheValuesSkipsMissingCounters(t *testing.T) {
	records := makeGroupIndexCacheValues([]int64{10, 20}, []int64{5})

	if len(records) != 1 {
		t.Fatalf("expected one aligned cache record, got %+v", records)
	}
	if records[0].GroupHash != 10 || records[0].UpdateCounter != 5 {
		t.Fatalf("unexpected aligned record: %+v", records[0])
	}
}
