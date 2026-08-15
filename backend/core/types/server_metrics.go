package types

import "app/db"

// One row every five seconds with what the machine was doing, written by server_utils and never by
// this backend: the ORM owns the schema, the daemon owns the writes. That is the same split
// user_logs uses, and here it is not a preference — server_utils is the only Genix process
// guaranteed to be on the box, since the backend may be a Lambda, so it is the only one that can
// promise a continuous series.
//
// Every column below the key is the PEAK of its five-second window, not an average and not a point
// sample: the daemon samples once a second and keeps the highest of the five. That is what makes a
// one-second spike visible in a five-second row, and it is also the reason these rows CANNOT BE
// SUMMED — adding NetRxRate across a day overstates the bytes actually transferred, because every
// value is a peak standing in for five seconds. This table answers "how bad did it get, and when";
// for "how much in total" the counters in /proc are the source.
//
// -1 in any column means "not measured": the service has no cgroup on this box, or every
// sub-sample of that window failed to read. It is a sentinel rather than a null because every
// column is an int16 in which 0 is a perfectly legitimate reading, so a missing backend and an idle
// backend would otherwise be the same row.
type ServerMetric struct {
	db.TableStruct[ServerMetricTable, ServerMetric]
	// Unix day, and the partition: one day of samples is one partition, so the TTL expires it
	// whole.
	Date int16 `json:",omitempty"`
	// secondsIntoDay / 5, so 0..17279. Zero-based like FrameOfDay in user_logs.go.
	Slot int16 `json:",omitempty"`
	// Host-wide. Percentages are hundredths: 23.45% is 2345.
	CpuPercent  int16 `json:",omitempty"`
	MemPercent  int16 `json:",omitempty"`
	DiskPercent int16 `json:",omitempty"`
	// Network rates in 5 KB/s units, so an int16 reaches 163 MB/s while still resolving the
	// single-digit KB/s an idle box shows.
	NetRxRate int16 `json:",omitempty"`
	NetTxRate int16 `json:",omitempty"`
	// Per-service memory in megabytes (saturating at the int16 ceiling, 32 GB) and CPU as a
	// percentage of the WHOLE machine, so a Scylla pinning eight of eight cores reads 100.00% and
	// not the top-style 800% that would not fit here.
	BackendMemMb          int16 `json:",omitempty"`
	BackendCpuPercent     int16 `json:",omitempty"`
	ServerUtilsMemMb      int16 `json:",omitempty"`
	ServerUtilsCpuPercent int16 `json:",omitempty"`
	SearchMemMb           int16 `json:",omitempty"`
	SearchCpuPercent      int16 `json:",omitempty"`
	ScyllaMemMb           int16 `json:",omitempty"`
	ScyllaCpuPercent      int16 `json:",omitempty"`
}

type ServerMetricTable struct {
	db.TableStruct[ServerMetricTable, ServerMetric]
	Date                  db.Col[ServerMetricTable, int16]
	Slot                  db.Col[ServerMetricTable, int16]
	CpuPercent            db.Col[ServerMetricTable, int16]
	MemPercent            db.Col[ServerMetricTable, int16]
	DiskPercent           db.Col[ServerMetricTable, int16]
	NetRxRate             db.Col[ServerMetricTable, int16]
	NetTxRate             db.Col[ServerMetricTable, int16]
	BackendMemMb          db.Col[ServerMetricTable, int16]
	BackendCpuPercent     db.Col[ServerMetricTable, int16]
	ServerUtilsMemMb      db.Col[ServerMetricTable, int16]
	ServerUtilsCpuPercent db.Col[ServerMetricTable, int16]
	SearchMemMb           db.Col[ServerMetricTable, int16]
	SearchCpuPercent      db.Col[ServerMetricTable, int16]
	ScyllaMemMb           db.Col[ServerMetricTable, int16]
	ScyllaCpuPercent      db.Col[ServerMetricTable, int16]
}

func (e ServerMetricTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		// Written externally in whole rows, like user_logs and credit_usage: no ORM-managed
		// created/updated columns and no update counter. A sample is written once and never
		// touched again, so there is nothing for either to describe.
		ID:                    47,
		Name:                  "server_metrics",
		Partition:             e.Date,
		Keys:                  db.Cols(e.Slot),
		DisableDefaultColumns: true,
		// No indexes on purpose. The only question this table answers is "give me a day's slots",
		// which the partition and the clustering order already answer, and every index would be a
		// second write on a path that writes 17280 times a day.
	}
}

// SlotsPerDay is how many rows one day's partition holds at the five-second cadence, and the
// reason the key fits an int16 with room to spare.
const SlotsPerDay = 86_400 / ServerMetricSlotSeconds

// ServerMetricSlotSeconds is the width of one slot. Mirrored by server_metrics.row_seconds in
// config.toml, which server_utils validates against this same arithmetic: change one and the
// clustering key stops meaning what every stored row meant.
const ServerMetricSlotSeconds = 5

// SlotOfDay is the slot a UTC timestamp falls in, 0..17279. UTC and not local time for the same
// reason FrameOfDay is: the key has to mean the same thing to the Rust writer and to every reader,
// whatever timezone either happens to run in.
func SlotOfDay(unixSeconds int64) int16 {
	secondsIntoDay := unixSeconds % 86_400
	if secondsIntoDay < 0 {
		secondsIntoDay += 86_400
	}
	return int16(secondsIntoDay / ServerMetricSlotSeconds)
}

// ServerMetricNotMeasured is the value a column carries when nothing could be read for it. Readers
// must filter it out before charting or averaging; treating it as a number plots a -0.01% dip.
const ServerMetricNotMeasured int16 = -1
