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
