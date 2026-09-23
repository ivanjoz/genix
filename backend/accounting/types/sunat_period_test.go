package types

import "testing"

func TestPeriodBoundsCoverTheWholeMonth(t *testing.T) {
	firstDay, lastDay, err := PeriodBounds("202609")
	if err != nil {
		t.Fatalf("periodo válido rechazado: %v", err)
	}
	if got := FormatBookDate(firstDay); got != "01/09/2026" {
		t.Errorf("esperaba que septiembre empiece el 01/09/2026, obtuve %s", got)
	}
	if got := FormatBookDate(lastDay); got != "30/09/2026" {
		t.Errorf("esperaba que septiembre termine el 30/09/2026, obtuve %s", got)
	}
}

// The month's own length, not a fixed offset — the bug every "+30 días" makes.
func TestPeriodBoundsHandleFebruaryAndLeapYears(t *testing.T) {
	_, lastDay, _ := PeriodBounds("202502")
	if got := FormatBookDate(lastDay); got != "28/02/2025" {
		t.Errorf("esperaba 28/02/2025, obtuve %s", got)
	}
	_, leapLastDay, _ := PeriodBounds("202402")
	if got := FormatBookDate(leapLastDay); got != "29/02/2024" {
		t.Errorf("esperaba 29/02/2024, obtuve %s", got)
	}
}

func TestPeriodBoundsRejectsMalformedPeriods(t *testing.T) {
	for _, period := range []string{"20260900", "2026-09", "202613", ""} {
		if _, _, err := PeriodBounds(period); err == nil {
			t.Errorf("el periodo %q debió ser rechazado", period)
		}
	}
}

func TestPeriodOfMatchesItsOwnBounds(t *testing.T) {
	firstDay, lastDay, _ := PeriodBounds("202609")
	if got := PeriodOf(firstDay); got != "202609" {
		t.Errorf("esperaba 202609 para el primer día, obtuve %s", got)
	}
	if got := PeriodOf(lastDay); got != "202609" {
		t.Errorf("esperaba 202609 para el último día, obtuve %s", got)
	}
	if got := PeriodOf(lastDay + 1); got != "202610" {
		t.Errorf("el día siguiente ya es otro periodo, obtuve %s", got)
	}
}
