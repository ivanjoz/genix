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
