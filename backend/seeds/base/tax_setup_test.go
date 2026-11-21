package base

import (
	accountingTypes "app/accounting/types"
	"testing"
	"time"
)

// Catches transcription typos in the SUNAT schedule: every deadline is a business day, digits share
// a day in pairs (2–3, 4–5, 6–7, 8–9), the days never go backwards across the columns, and every row
// passes the handler's validation.
func TestSunatDueDays2026AreConsistent(t *testing.T) {
	if len(sunatDueDays2026) != 12 {
		t.Fatalf("expected the 12 periods of 2026, got %d", len(sunatDueDays2026))
	}
	for period, daysOfMonth := range sunatDueDays2026 {
		dueYear, dueMonth := 2000+int(period/100), time.Month(period%100)+1
		if dueMonth > time.December {
			dueYear, dueMonth = dueYear+1, time.January
		}
		calendar := accountingTypes.TaxDueCalendar{Period: period}
		for column, dayOfMonth := range daysOfMonth {
			dueDate := time.Date(dueYear, dueMonth, dayOfMonth, 0, 0, 0, 0, time.UTC)
			if weekday := dueDate.Weekday(); weekday == time.Saturday || weekday == time.Sunday {
				t.Errorf("period %d column %d falls on a %s", period, column, weekday)
			}
			if column > 0 && dayOfMonth < daysOfMonth[column-1] {
				t.Errorf("period %d column %d is earlier than column %d", period, column, column-1)
			}
			calendar.DueDays = append(calendar.DueDays, int16(dueDate.Unix()/86400))
		}
		for _, pairedDigit := range []int{2, 4, 6, 8} {
			if daysOfMonth[pairedDigit] != daysOfMonth[pairedDigit+1] {
				t.Errorf("period %d: digits %d and %d should share a day", period, pairedDigit, pairedDigit+1)
			}
		}
		if err := accountingTypes.ValidateTaxDueCalendar(calendar); err != nil {
			t.Errorf("period %d: %v", period, err)
		}
	}
}
