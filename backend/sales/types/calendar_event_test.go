package types

import (
	"strings"
	"testing"
)

func TestValidateCalendarEvent(t *testing.T) {
	validEvent := CalendarEvent{Date: 20000, EventType: CalendarEventPayday, Name: "  Quincena  "}
	if err := ValidateCalendarEvent(&validEvent); err != nil {
		t.Fatalf("valid event refused: %v", err)
	}
	if validEvent.Name != "Quincena" {
		t.Fatalf("name not trimmed: %q", validEvent.Name)
	}

	invalidEvents := map[string]CalendarEvent{
		"no date":       {EventType: CalendarEventHoliday, Name: "Navidad"},
		"type too low":  {Date: 20000, Name: "Navidad"},
		"type too high": {Date: 20000, EventType: CalendarEventCommercialDate + 1, Name: "Navidad"},
		"blank name":    {Date: 20000, EventType: CalendarEventHoliday, Name: "   "},
		"name too long": {Date: 20000, EventType: CalendarEventHoliday, Name: strings.Repeat("a", calendarEventNameMaxLength+1)},
	}
	for caseName, event := range invalidEvents {
		if err := ValidateCalendarEvent(&event); err == nil {
			t.Errorf("%s: expected an error", caseName)
		}
	}
}
