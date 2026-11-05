package core

import "testing"

func TestPeriodOfDay(t *testing.T) {
	cases := []struct {
		name       string
		unixDay    int16
		wantPeriod int16
	}{
		{"first day of Oct 2026", 20727, 2610},
		{"last day of Oct 2026", 20757, 2610},
		{"first day of Nov 2026", 20758, 2611},
		{"no date", 0, 0},
	}
	for _, testCase := range cases {
		if got := PeriodOfDay(testCase.unixDay); got != testCase.wantPeriod {
			t.Errorf("%s: PeriodOfDay(%d) = %d, want %d", testCase.name, testCase.unixDay, got, testCase.wantPeriod)
		}
	}
}

func TestPeriodDays(t *testing.T) {
	if got := PeriodFirstDay(2610); got != 20727 {
		t.Errorf("PeriodFirstDay(2610) = %d, want 20727", got)
	}
	if got := PeriodLastDay(2610); got != 20757 {
		t.Errorf("PeriodLastDay(2610) = %d, want 20757", got)
	}
	if got := PeriodLastDay(2612); PeriodOfDay(got) != 2612 || PeriodOfDay(got+1) != 2701 {
		t.Errorf("PeriodLastDay(2612) = %d is not the last day of Dec 2026", got)
	}
}

func TestAddPeriods(t *testing.T) {
	for _, testCase := range []struct{ period, months, want int16 }{
		{2603, -12, 2503}, {2603, -3, 2512}, {2612, 1, 2701}, {2610, 0, 2610}, {2601, -1, 2512},
	} {
		if got := AddPeriods(testCase.period, int(testCase.months)); got != testCase.want {
			t.Errorf("AddPeriods(%d, %d) = %d, want %d", testCase.period, testCase.months, got, testCase.want)
		}
	}
}

func TestSUnixTimeToUnixDay(t *testing.T) {
	// 2026-10-01 00:00 UTC is unix 1790812800, so SUnixTime 395406400.
	if got := SUnixTimeToUnixDay(395406400); got != PeriodFirstDay(2610) {
		t.Errorf("SUnixTimeToUnixDay = %d, want %d", got, PeriodFirstDay(2610))
	}
}
