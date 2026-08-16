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
	// Widest window the route will serve, and therefore the furthest back a client's cache can
	// reach. The eviction range is measured from it, so it must not be raised without raising that.
	serverMetricsMaxWindowHours     = int64(24)
	serverMetricsDefaultWindowHours = int64(4)
	// A client returning after a very long absence would otherwise be handed an unbounded list of
	// dead bucket IDs. Beyond the cap a few orphan buckets survive in IndexedDB until the next
	// `ver` bump — bounded garbage, which is better than an unbounded response.
	serverMetricsMaxEvictedBuckets = 400
	serverMetricsSlotsPerDay       = int64(coreTypes.SlotsPerDay)
)

// ServerMetricsHour is one hour of samples as parallel arrays, which is what makes the delta cheap:
// the client merges by SlotsInHour position, so a refresh carries only the slots recorded since its
// watermark instead of the whole hour.
type ServerMetricsHour struct {
	// Hours since the unix epoch. Monotonic, so it is both the cache key and the bucket's identity,
	// and the wall-clock hour is recoverable by multiplying back.
	BucketID int32 `json:"ID"`
	// Offset of each sample inside the hour, 0..719, positioning every other array. Offsets rather
	// than slot-of-day so the value stays three digits across 2880 of them.
	SlotsInHour []int16

	CpuPercent  []int16
	MemPercent  []int16
	DiskPercent []int16
	NetRxRate   []int16
	NetTxRate   []int16

	BackendMemMb          []int16
	BackendCpuPercent     []int16
	ServerUtilsMemMb      []int16
	ServerUtilsCpuPercent []int16
	SearchMemMb           []int16
	SearchCpuPercent      []int16
	ScyllaMemMb           []int16
	ScyllaCpuPercent      []int16

	// Absolute slot (unix seconds / 5) of the newest sample here. This is the delta watermark: the
	// client keeps the maximum across records and sends it back, and the handler returns strictly
	// newer slots. Never omitempty — a zero watermark would silently mean "cold load" forever.
	Updated int32 `json:"upd"`
	Status  int8  `json:"ss"`
}

// ServerMetricsResponse is an object rather than the bare array the records alone would form, purely
// so Hours_IDsToRemove has somewhere to live: a bare array normalizes to the cache's `_default` key,
// which has no room for the eviction channel. The cost is that the watermark comes back as `?Hours=`,
// named after this field, instead of `?upd=`.
type ServerMetricsResponse struct {
	Hours []ServerMetricsHour
	// Buckets that have fallen out of the window. The cache deletes these persisted rows before
	// applying the delta, so a bare list of ints evicts a whole day for a couple hundred bytes.
	HoursIDsToRemove []int32 `json:"Hours_IDsToRemove,omitempty"`
}

// GetServerMetrics serves the Server Panel charts from the rows server_utils writes every five
// seconds. It reads no CompanyID: server_metrics describes the machine, not a tenant, and the route
// is restricted to the SaaS company in saasOnlyRoutes.
func GetServerMetrics(req *core.HandlerArgs) core.HandlerResponse {
	windowHours := serverMetricsDefaultWindowHours
	if requestedHours := int64(req.GetQueryInt("hours")); requestedHours > 0 {
		windowHours = min(requestedHours, serverMetricsMaxWindowHours)
	}

	// `Hours` is what the delta cache names the watermark after this response's field; `upd` is the
	// trailing single-watermark param it also sends. Either one answers the same question.
	watermarkSlot := int64(core.Coalesce(req.GetQueryInt("Hours"), req.GetQueryInt("upd")))

	// The real clock, not core.Now(): server_utils writes these rows with the real clock and has no
	// historical-clock override, so a frozen backend clock would query a day nothing ever wrote.
	nowUnixSeconds := time.Now().UTC().Unix()
	newestSlot := nowUnixSeconds / coreTypes.ServerMetricSlotSeconds
	firstLiveBucket := nowUnixSeconds/serverMetricsSecondsPerHour - windowHours + 1
	// The watermark wins whenever it is inside the window: that is what turns a refresh into three
	// slots instead of the whole four hours.
	firstSlotWanted := max(firstLiveBucket*serverMetricsSlotsPerHour, watermarkSlot+1)

	response := ServerMetricsResponse{
		Hours:            []ServerMetricsHour{},
		HoursIDsToRemove: makeEvictedBucketIDs(watermarkSlot, firstLiveBucket, nowUnixSeconds),
	}
	if firstSlotWanted > newestSlot {
		// Nothing new since the client's watermark. Evictions may still be pending, so the response
		// is returned rather than short-circuited to empty.
		return req.MakeResponse(response)
	}

	rows, err := readServerMetricRows(firstSlotWanted, newestSlot)
	if err != nil {
		core.Log("server metrics query failed::", " first_slot::", firstSlotWanted,
			" newest_slot::", newestSlot, " err::", err)
		return req.MakeErr("No se pudieron obtener las métricas del servidor.", err)
	}

	response.Hours = groupServerMetricsByHour(rows)
	core.Log("server metrics query completed::", " window_hours::", windowHours,
		" watermark::", watermarkSlot, " rows::", len(rows), " buckets::", len(response.Hours),
		" evicted::", len(response.HoursIDsToRemove))
	return req.MakeResponse(response)
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
		bucket.ServerUtilsMemMb = append(bucket.ServerUtilsMemMb, row.ServerUtilsMemMb)
		bucket.ServerUtilsCpuPercent = append(bucket.ServerUtilsCpuPercent, row.ServerUtilsCpuPercent)
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

// makeEvictedBucketIDs lists the buckets the client still holds that have fallen out of the window.
//
// It fires only when the client's watermark sits in an earlier hour than the current one, so a
// refresh inside the same hour evicts nothing and the steady-state response stays at a few slots.
// The range starts a full max-window before the client's newest bucket because that is the furthest
// back its cache can reach — anything older than that it never had.
func makeEvictedBucketIDs(watermarkSlot, firstLiveBucket, nowUnixSeconds int64) []int32 {
	if watermarkSlot <= 0 {
		// A cold client has an empty cache: there is nothing of its own to evict.
		return nil
	}

	clientNewestBucket := watermarkSlot / serverMetricsSlotsPerHour
	if clientNewestBucket >= nowUnixSeconds/serverMetricsSecondsPerHour {
		return nil
	}

	firstEvictedBucket := clientNewestBucket - serverMetricsMaxWindowHours
	if firstLiveBucket-firstEvictedBucket > serverMetricsMaxEvictedBuckets {
		firstEvictedBucket = firstLiveBucket - serverMetricsMaxEvictedBuckets
	}

	evictedBucketIDs := []int32{}
	for bucketID := firstEvictedBucket; bucketID < firstLiveBucket; bucketID++ {
		evictedBucketIDs = append(evictedBucketIDs, int32(bucketID))
	}
	return evictedBucketIDs
}
