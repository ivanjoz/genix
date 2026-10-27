package config

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	// One bucket is one wall-clock hour. Sealed buckets are immutable — a past hour can never gain a
	// sample — so a client fetches each one exactly once and only the live bucket produces deltas.
	// That immutability is the whole reason the bucket exists; a per-day record would keep growing
	// and a per-slot record would repeat fourteen JSON key names 2880 times.
	serverMetricsSlotsPerHour   = 3600 / coreTypes.ServerMetricSlotSeconds
	serverMetricsSecondsPerHour = int64(3600)
	// Widest window the route will serve.
	serverMetricsMaxWindowHours     = int64(24)
	serverMetricsDefaultWindowHours = int64(4)
	serverMetricsSlotsPerDay        = int64(coreTypes.SlotsPerDay)
)

// ServerMetricsHour is one hour of samples as parallel arrays, which is what makes the delta cheap:
// the client merges by SlotsInHour position, so a refresh carries only the slots recorded since its
// watermark instead of the whole hour.
type ServerMetricsHour struct {
	// Hours since the unix epoch. Monotonic, so it is both the cache key and the bucket's identity,
	// and the wall-clock hour is recoverable by multiplying back.
	BucketID int32 `json:"ID" cb:"ID,1"`
	// Offset of each sample inside the hour, 0..719, positioning every other array. Offsets rather
	// than slot-of-day so the value stays three digits across 2880 of them.
	SlotsInHour []int16 `cb:"SlotsInHour,2"`

	CpuPercent  []int16 `cb:"CpuPercent,3"`
	MemPercent  []int16 `cb:"MemPercent,4"`
	DiskPercent []int16 `cb:"DiskPercent,5"`
	NetRxRate   []int16 `cb:"NetRxRate,6"`
	NetTxRate   []int16 `cb:"NetTxRate,7"`

	BackendMemMb       []int16 `cb:"BackendMemMb,8"`
	BackendCpuPercent  []int16 `cb:"BackendCpuPercent,9"`
	FarewardMemMb      []int16 `cb:"FarewardMemMb,10"`
	FarewardCpuPercent []int16 `cb:"FarewardCpuPercent,11"`
	SearchMemMb        []int16 `cb:"SearchMemMb,12"`
	SearchCpuPercent   []int16 `cb:"SearchCpuPercent,13"`
	ScyllaMemMb        []int16 `cb:"ScyllaMemMb,14"`
	ScyllaCpuPercent   []int16 `cb:"ScyllaCpuPercent,15"`

	// Absolute slot (unix seconds / 5) of the newest sample here. This is the delta watermark: the
	// client keeps the maximum across records and sends it back, and the handler returns strictly
	// newer slots. Never omitempty — a zero watermark would silently mean "cold load" forever.
	Updated int32 `json:"upd" cb:"upd,16"`
	Status  int8  `json:"ss" cb:"ss,17"`
}

// ServerMetricsResponse carries the window's hour buckets whole, ending with the replace Metadata
// (core.WithDeltaReplace): a bucket that fell out of the window leaves the client's cache.
type ServerMetricsResponse struct {
	Hours []any
}

// GetServerMetrics serves the Server Panel charts from the rows fareward writes every five
// seconds. It reads no CompanyID: server_metrics describes the machine, not a tenant, and the route
// is restricted to the SaaS company in saasOnlyRoutes.
func GetServerMetrics(req *core.HandlerArgs) core.HandlerResponse {
	windowHours := serverMetricsDefaultWindowHours
	if requestedHours := int64(req.GetQueryInt("hours")); requestedHours > 0 {
		windowHours = min(requestedHours, serverMetricsMaxWindowHours)
	}

	// The real clock, not core.Now(): fareward writes these rows with the real clock and has no
	// historical-clock override, so a frozen backend clock would query a day nothing ever wrote.
	nowUnixSeconds := time.Now().UTC().Unix()
	newestSlot := nowUnixSeconds / coreTypes.ServerMetricSlotSeconds
	firstLiveBucket := nowUnixSeconds/serverMetricsSecondsPerHour - windowHours + 1

	rows, err := readServerMetricRows(firstLiveBucket*serverMetricsSlotsPerHour, newestSlot)
	if err != nil {
		return req.MakeErr("No se pudieron obtener las métricas del servidor.", err)
	}
	return req.MakeResponse(ServerMetricsResponse{Hours: core.WithDeltaReplace(groupServerMetricsByHour(rows))})
}

// readServerMetricRows slices the requested absolute-slot range out of the table. The range is
// expressed per unix day because that is the partition: one Equals on the partition plus a range on
// the clustering key is the only access pattern server_metrics was keyed for, and the reason it
// carries no index. A window of at most 24 hours touches at most two days, so this is two queries.
func readServerMetricRows(firstSlot, lastSlot int64) ([]coreTypes.ServerMetric, error) {
	firstDay, lastDay := firstSlot/serverMetricsSlotsPerDay, lastSlot/serverMetricsSlotsPerDay
	rowsPerDay := make([][]coreTypes.ServerMetric, lastDay-firstDay+1)

	queries := errgroup.Group{}
	for dayOffset := range rowsPerDay {
		unixDay := firstDay + int64(dayOffset)
		firstSlotOfDay := unixDay * serverMetricsSlotsPerDay
		fromSlot := int16(max(firstSlot, firstSlotOfDay) - firstSlotOfDay)
		toSlot := int16(min(lastSlot, firstSlotOfDay+serverMetricsSlotsPerDay-1) - firstSlotOfDay)

		queries.Go(func() error {
			query := db.Query(&rowsPerDay[dayOffset])
			query.Date.Equals(int16(unixDay)).Slot.Between(fromSlot, toSlot)
			return query.Exec()
		})
	}
	if err := queries.Wait(); err != nil {
		return nil, err
	}

	rows := []coreTypes.ServerMetric{}
	for _, dayRows := range rowsPerDay {
		rows = append(rows, dayRows...)
	}
	return rows, nil
}

// groupServerMetricsByHour turns flat rows into one record per hour. Rows arrive in clustering order
// within a day and the days are appended oldest first, so appending in sequence keeps every parallel
// array sorted by time without a second pass.
func groupServerMetricsByHour(rows []coreTypes.ServerMetric) []ServerMetricsHour {
	buckets := []ServerMetricsHour{}
	bucketIndexByID := map[int32]int{}

	for _, row := range rows {
		absoluteSlot := int64(row.Date)*serverMetricsSlotsPerDay + int64(row.Slot)
		bucketID := int32(absoluteSlot / serverMetricsSlotsPerHour)

		bucketIndex, bucketExists := bucketIndexByID[bucketID]
		if !bucketExists {
			bucketIndex = len(buckets)
			bucketIndexByID[bucketID] = bucketIndex
			buckets = append(buckets, ServerMetricsHour{BucketID: bucketID, Status: 1})
		}

		bucket := &buckets[bucketIndex]
		bucket.SlotsInHour = append(bucket.SlotsInHour, int16(absoluteSlot%serverMetricsSlotsPerHour))
		bucket.CpuPercent = append(bucket.CpuPercent, row.CpuPercent)
		bucket.MemPercent = append(bucket.MemPercent, row.MemPercent)
		bucket.DiskPercent = append(bucket.DiskPercent, row.DiskPercent)
		bucket.NetRxRate = append(bucket.NetRxRate, row.NetRxRate)
		bucket.NetTxRate = append(bucket.NetTxRate, row.NetTxRate)
		bucket.BackendMemMb = append(bucket.BackendMemMb, row.BackendMemMb)
		bucket.BackendCpuPercent = append(bucket.BackendCpuPercent, row.BackendCpuPercent)
		bucket.FarewardMemMb = append(bucket.FarewardMemMb, row.FarewardMemMb)
		bucket.FarewardCpuPercent = append(bucket.FarewardCpuPercent, row.FarewardCpuPercent)
		bucket.SearchMemMb = append(bucket.SearchMemMb, row.SearchMemMb)
		bucket.SearchCpuPercent = append(bucket.SearchCpuPercent, row.SearchCpuPercent)
		bucket.ScyllaMemMb = append(bucket.ScyllaMemMb, row.ScyllaMemMb)
		bucket.ScyllaCpuPercent = append(bucket.ScyllaCpuPercent, row.ScyllaCpuPercent)

		if newestSlot := int32(absoluteSlot); newestSlot > bucket.Updated {
			bucket.Updated = newestSlot
		}
	}

	return buckets
}
