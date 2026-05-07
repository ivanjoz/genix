package db

import "testing"

func TestNextCounterRangeRepairsNegativeStoredCounter(t *testing.T) {
	startValue, counterIncrement := nextCounterRange(-5, 6)

	if startValue != 1 {
		t.Fatalf("expected repaired range to start at 1, got %d", startValue)
	}
	if counterIncrement != 11 {
		t.Fatalf("expected corrective increment 11 to reserve IDs 1..6, got %d", counterIncrement)
	}
}

func TestNextCounterRangeKeepsPositiveStoredCounter(t *testing.T) {
	startValue, counterIncrement := nextCounterRange(41, 3)

	if startValue != 42 {
		t.Fatalf("expected range to continue at 42, got %d", startValue)
	}
	if counterIncrement != 3 {
		t.Fatalf("expected normal increment 3, got %d", counterIncrement)
	}
}
