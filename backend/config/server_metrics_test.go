package config

import (
	coreTypes "app/core/types"
	"testing"
)

// makeServerMetricRow builds one stored row from an absolute slot, which is how the tests below
// think about time: the table's (Date, Slot) pair is just that number split at the day boundary.
func makeServerMetricRow(absoluteSlot int64, cpuPercent int16) coreTypes.ServerMetric {
	return coreTypes.ServerMetric{
		Date:       int16(absoluteSlot / serverMetricsSlotsPerDay),
		Slot:       int16(absoluteSlot % serverMetricsSlotsPerDay),
		CpuPercent: cpuPercent,
	}
}

func TestGroupServerMetricsByHourSplitsOnTheHourBoundary(t *testing.T) {
	// Two slots in one hour and one in the next: the bucket is the unit the client caches, so a
	// sample landing a slot past :00 has to start a new record rather than extend the sealed one.
	firstBucketStart := int64(492_000) * serverMetricsSlotsPerHour
	rows := []coreTypes.ServerMetric{
		makeServerMetricRow(firstBucketStart, 100),
		makeServerMetricRow(firstBucketStart+serverMetricsSlotsPerHour-1, 200),
		makeServerMetricRow(firstBucketStart+serverMetricsSlotsPerHour, 300),
	}

	buckets := groupServerMetricsByHour(rows)

	if len(buckets) != 2 {
		t.Fatalf("expected 2 hour buckets, got %d", len(buckets))
	}
	if buckets[0].BucketID != 492_000 || buckets[1].BucketID != 492_001 {
		t.Fatalf("unexpected bucket ids: %d and %d", buckets[0].BucketID, buckets[1].BucketID)
	}
	// Offsets are within the hour, not within the day: 0 and 719 for a full first bucket.
	if got := buckets[0].SlotsInHour; len(got) != 2 || got[0] != 0 || got[1] != int16(serverMetricsSlotsPerHour-1) {
		t.Fatalf("unexpected slot offsets in the first bucket: %v", got)
	}
	if got := buckets[1].SlotsInHour; len(got) != 1 || got[0] != 0 {
		t.Fatalf("unexpected slot offsets in the second bucket: %v", got)
	}
	// Every parallel array must stay aligned with SlotsInHour, which is what the client merges on.
	if got := buckets[0].CpuPercent; len(got) != 2 || got[0] != 100 || got[1] != 200 {
		t.Fatalf("cpu values lost their alignment with the slots: %v", got)
	}
}

func TestBucketWatermarkIsItsNewestAbsoluteSlot(t *testing.T) {
	// The client keeps the maximum `upd` and sends it back, so a bucket reporting anything but its
	// newest slot would make the next delta re-send rows the client already has, or skip rows it
	// does not — the two failures this response shape exists to prevent.
	firstBucketStart := int64(492_100) * serverMetricsSlotsPerHour
	rows := []coreTypes.ServerMetric{
		makeServerMetricRow(firstBucketStart+5, 1),
		makeServerMetricRow(firstBucketStart+9, 2),
	}

	buckets := groupServerMetricsByHour(rows)

	if len(buckets) != 1 {
		t.Fatalf("expected 1 hour bucket, got %d", len(buckets))
	}
	if int64(buckets[0].Updated) != firstBucketStart+9 {
		t.Fatalf("expected watermark %d, got %d", firstBucketStart+9, buckets[0].Updated)
	}
	if buckets[0].Status != 1 {
		t.Fatalf("expected an active status, got %d", buckets[0].Status)
	}
}

func TestGroupServerMetricsByHourCrossesMidnight(t *testing.T) {
	// The last slot of a unix day and the first of the next are consecutive in absolute terms but
	// live in different partitions, read by different queries. They must still land in adjacent
	// buckets rather than restarting the numbering.
	lastSlotOfDay := int64(20_680)*serverMetricsSlotsPerDay + serverMetricsSlotsPerDay - 1
	rows := []coreTypes.ServerMetric{
		makeServerMetricRow(lastSlotOfDay, 10),
		makeServerMetricRow(lastSlotOfDay+1, 20),
	}

	buckets := groupServerMetricsByHour(rows)

	if len(buckets) != 2 {
		t.Fatalf("expected 2 hour buckets across midnight, got %d", len(buckets))
	}
	if buckets[1].BucketID-buckets[0].BucketID != 1 {
		t.Fatalf("buckets are not adjacent: %d then %d", buckets[0].BucketID, buckets[1].BucketID)
	}
	if buckets[1].SlotsInHour[0] != 0 {
		t.Fatalf("the first slot of a new day should open a bucket at offset 0, got %d", buckets[1].SlotsInHour[0])
	}
}

func TestEvictionIsSilentWithinTheSameHour(t *testing.T) {
	// The common case by far: a client refreshing every fifteen seconds. Re-sending the same dead
	// bucket ids on every one of those would undo the point of the delta.
	nowUnixSeconds := int64(492_200)*serverMetricsSecondsPerHour + 900
	watermarkSlot := nowUnixSeconds / coreTypes.ServerMetricSlotSeconds
	firstLiveBucket := int64(492_200) - serverMetricsDefaultWindowHours + 1

	if evicted := makeEvictedBucketIDs(watermarkSlot, firstLiveBucket, nowUnixSeconds); len(evicted) != 0 {
		t.Fatalf("expected no evictions inside the current hour, got %v", evicted)
	}
}

func TestEvictionCoversTheClientsWholeReachAfterAnHourRolls(t *testing.T) {
	// The client's watermark sits in the previous hour, so the window has moved on. Everything it
	// could still hold — a full max-window back from its newest bucket — up to the new window start
	// has to be named, or those buckets stay in IndexedDB forever.
	currentBucket := int64(492_300)
	nowUnixSeconds := currentBucket * serverMetricsSecondsPerHour
	clientNewestBucket := currentBucket - 1
	watermarkSlot := clientNewestBucket*serverMetricsSlotsPerHour + 10
	firstLiveBucket := currentBucket - serverMetricsDefaultWindowHours + 1

	evicted := makeEvictedBucketIDs(watermarkSlot, firstLiveBucket, nowUnixSeconds)

	if len(evicted) == 0 {
		t.Fatal("expected evictions once the hour rolled over")
	}
	if int64(evicted[0]) != clientNewestBucket-serverMetricsMaxWindowHours {
		t.Fatalf("eviction should start a max-window before the client's newest bucket, got %d", evicted[0])
	}
	if int64(evicted[len(evicted)-1]) != firstLiveBucket-1 {
		t.Fatalf("eviction should stop just before the window, got %d", evicted[len(evicted)-1])
	}
	// A live bucket must never be evicted: that would delete rows the same response is delivering.
	for _, bucketID := range evicted {
		if int64(bucketID) >= firstLiveBucket {
			t.Fatalf("bucket %d is inside the window and must not be evicted", bucketID)
		}
	}
}

func TestEvictionIsCappedForALongAbsence(t *testing.T) {
	// A client returning after months would otherwise be handed one id per hour of absence.
	currentBucket := int64(500_000)
	nowUnixSeconds := currentBucket * serverMetricsSecondsPerHour
	watermarkSlot := (currentBucket - 5_000) * serverMetricsSlotsPerHour
	firstLiveBucket := currentBucket - serverMetricsDefaultWindowHours + 1

	evicted := makeEvictedBucketIDs(watermarkSlot, firstLiveBucket, nowUnixSeconds)

	if len(evicted) != serverMetricsMaxEvictedBuckets {
		t.Fatalf("expected the eviction list capped at %d, got %d", serverMetricsMaxEvictedBuckets, len(evicted))
	}
}

func TestColdClientIsSentNoEvictions(t *testing.T) {
	// No watermark means an empty cache, so there is nothing of the client's own to delete.
	if evicted := makeEvictedBucketIDs(0, 492_400, 492_400*serverMetricsSecondsPerHour); evicted != nil {
		t.Fatalf("a cold client should be sent no evictions, got %v", evicted)
	}
}
