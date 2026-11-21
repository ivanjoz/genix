package sales

import (
	"app/core"
	"app/db"
	salesModule "app/sales"
	salesTypes "app/sales/types"
	"app/seeds/runner"
	"slices"
	"time"
)

// peruFixedHolidays are Peru's national holidays on a fixed date, as of 2026.
var peruFixedHolidays = []struct {
	Month time.Month
	Day   int
	Name  string
}{
	{time.January, 1, "Año Nuevo"},
	{time.May, 1, "Día del Trabajo"},
	{time.June, 7, "Batalla de Arica y Día de la Bandera"},
	{time.June, 29, "San Pedro y San Pablo"},
	{time.July, 23, "Día de la Fuerza Aérea del Perú"},
	{time.July, 28, "Fiestas Patrias"},
	{time.July, 29, "Fiestas Patrias"},
	{time.August, 6, "Batalla de Junín"},
	{time.August, 30, "Santa Rosa de Lima"},
	{time.October, 8, "Combate de Angamos"},
	{time.November, 1, "Día de Todos los Santos"},
	{time.December, 8, "Inmaculada Concepción"},
	{time.December, 9, "Batalla de Ayacucho"},
	{time.December, 25, "Navidad"},
}

// peruHolyThursdays is Jueves Santo per seeded year; Viernes Santo is the next day.
var peruHolyThursdays = map[int]time.Time{
	2024: time.Date(2024, time.March, 28, 0, 0, 0, 0, time.UTC),
	2025: time.Date(2025, time.April, 17, 0, 0, 0, 0, time.UTC),
	2026: time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC),
	2027: time.Date(2027, time.March, 25, 0, 0, 0, 0, time.UTC),
}

func unixDayOf(date time.Time) int16 { return int16(date.Unix() / 86400) }

// ensurePeruHolidays writes Peru's national holidays of 2024–2027 into the global calendar
// (partition 0) through POST.calendar-events as the super-admin. A holiday already stored with the
// same date and name is skipped.
func ensurePeruHolidays(run *runner.Run) error {
	storedHolidays := []salesTypes.CalendarEvent{}
	query := db.Query(&storedHolidays)
	query.Select(query.Date, query.Name, query.Status).CompanyID.Equals(salesTypes.CalendarGlobalCompanyID)
	if err := query.Exec(); err != nil {
		return err
	}

	holidays := []salesTypes.CalendarEvent{}
	for year, holyThursday := range peruHolyThursdays {
		holidays = append(holidays,
			salesTypes.CalendarEvent{Date: unixDayOf(holyThursday), Name: "Jueves Santo"},
			salesTypes.CalendarEvent{Date: unixDayOf(holyThursday.AddDate(0, 0, 1)), Name: "Viernes Santo"})
		for _, holiday := range peruFixedHolidays {
			holidays = append(holidays, salesTypes.CalendarEvent{
				Date: unixDayOf(time.Date(year, holiday.Month, holiday.Day, 0, 0, 0, 0, time.UTC)), Name: holiday.Name})
		}
	}

	savedCount := 0
	for _, holiday := range holidays {
		if slices.ContainsFunc(storedHolidays, func(stored salesTypes.CalendarEvent) bool {
			return stored.Status > 0 && stored.Date == holiday.Date && stored.Name == holiday.Name
		}) {
			continue
		}
		holiday.EventType, holiday.Status, holiday.IsGlobal = salesTypes.CalendarEventHoliday, 1, true
		if err := runner.CallAs(run.UserToken, "POST.calendar-events", salesModule.PostCalendarEvents, nil, holiday, nil); err != nil {
			return err
		}
		savedCount++
	}
	core.Log("Seed:: feriados del Perú", "nuevos", savedCount, "de", len(holidays))
	return nil
}
