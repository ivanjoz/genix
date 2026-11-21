package types

import (
	"testing"
	"time"
)

func unixDayOf(year int, month time.Month, day int) int16 {
	return int16(time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// December 2026 falls due in January 2027 (SUNAT's 2026 page).
func december2026Calendar() TaxDueCalendar {
	days := []int{18, 19, 20, 20, 21, 21, 22, 22, 25, 25, 26}
	calendar := TaxDueCalendar{Period: 2612}
	for _, day := range days {
		calendar.DueDays = append(calendar.DueDays, unixDayOf(2027, time.January, day))
	}
	return calendar
}

func TestDueDayForPicksTheRUCColumnOrTheGoodTaxpayerOne(t *testing.T) {
	calendar := december2026Calendar()
	cases := []struct {
		ruc            string
		isGoodTaxpayer bool
		want           int16
	}{
		{"20605712000", false, unixDayOf(2027, time.January, 18)},
		{"20605712009", false, unixDayOf(2027, time.January, 25)},
		{"20605712009", true, unixDayOf(2027, time.January, 26)},
		{"", false, 0},
		{"2060571200X", false, 0},
	}
	for _, testCase := range cases {
		if got := calendar.DueDayFor(testCase.ruc, testCase.isGoodTaxpayer); got != testCase.want {
			t.Errorf("DueDayFor(%q, %v) = %d, want %d", testCase.ruc, testCase.isGoodTaxpayer, got, testCase.want)
		}
	}
}

func TestValidateTaxDueCalendar(t *testing.T) {
	if err := ValidateTaxDueCalendar(december2026Calendar()); err != nil {
		t.Fatalf("the December 2026 schedule should be valid: %v", err)
	}

	short := december2026Calendar()
	short.DueDays = short.DueDays[:10]
	if ValidateTaxDueCalendar(short) == nil {
		t.Error("a schedule without the buenos contribuyentes column was accepted")
	}

	insidePeriod := december2026Calendar()
	insidePeriod.DueDays[3] = unixDayOf(2026, time.December, 31)
	if ValidateTaxDueCalendar(insidePeriod) == nil {
		t.Error("a deadline inside the period it declares was accepted")
	}

	badPeriod := december2026Calendar()
	badPeriod.Period = 2613
	if ValidateTaxDueCalendar(badPeriod) == nil {
		t.Error("period 2613 was accepted")
	}
}
