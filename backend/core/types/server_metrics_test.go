package types

import (
	"app/db"
	"testing"
)

// The column names ARE the contract with server_utils/src/sysmetrics/writer.rs, which names every
// one of them in its INSERT. Renaming a field here changes the derived column name and the daemon's
// prepare fails at startup — and since the collector fails open, the only symptom in production
// would be a table that quietly stops filling. This asserts the names at build time instead.
func TestServerMetricsColumnsMatchTheRustWriter(t *testing.T) {
	expectedColumns := []string{
		"date",
		"slot",
		"cpu_percent",
		"mem_percent",
		"disk_percent",
		"net_rx_rate",
		"net_tx_rate",
		"backend_mem_mb",
		"backend_cpu_percent",
		"server_utils_mem_mb",
		"server_utils_cpu_percent",
		"search_mem_mb",
		"search_cpu_percent",
		"scylla_mem_mb",
		"scylla_cpu_percent",
	}

	columns := db.MakeTable[ServerMetric]().GetColumns()
	for _, expected := range expectedColumns {
		if _, found := columns[expected]; !found {
			t.Fatalf("column %q is missing; the Rust INSERT names it", expected)
		}
	}
	// An extra column is as bad as a missing one: the daemon would never write it, leaving a
	// column that is null on every row of the table.
	if len(columns) != len(expectedColumns) {
		t.Fatalf("server_metrics compiled to %d columns, expected %d", len(columns), len(expectedColumns))
	}
}

// A slot has to land in exactly one place for a given second, and the day has to roll over cleanly:
// an off-by-one at midnight would write the last five seconds of a day into slot 0 of that same
// day, overwriting the first row of the morning.
func TestSlotOfDay(t *testing.T) {
	cases := []struct {
		unixSeconds int64
		expected    int16
	}{
		{0, 0},                        // midnight UTC
		{4, 0},                        // still the first slot
		{5, 1},                        // the boundary belongs to the next slot
		{86_400 - 1, SlotsPerDay - 1}, // the last second of the day
		{86_400, 0},                   // and the next day starts over
		{1_767_225_600 + 615*60, 7380},
	}
	for _, testCase := range cases {
		if slot := SlotOfDay(testCase.unixSeconds); slot != testCase.expected {
			t.Fatalf("unix %d gave slot %d, expected %d", testCase.unixSeconds, slot, testCase.expected)
		}
	}
}

// The whole point of the int16 key: a day's slots must fit it, with the sentinel below zero.
func TestSlotsPerDayFitsTheInt16Key(t *testing.T) {
	if SlotsPerDay > 32_767 {
		t.Fatalf("%d slots per day overflows the int16 clustering key", SlotsPerDay)
	}
	if ServerMetricNotMeasured >= 0 {
		t.Fatal("the not-measured sentinel must stay below every real reading")
	}
}
